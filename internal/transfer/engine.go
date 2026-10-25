package transfer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

func jsonUnmarshal(p []byte, v any) error { return json.Unmarshal(p, v) }

// Phasen einer Übertragung.
const (
	PhaseListing   = "listing"   // Dateien werden aufgelistet
	PhaseCopying   = "copying"   // Dateien werden kopiert
	PhaseDone      = "done"      // fertig
	PhaseFailed    = "failed"    // abgebrochen wegen Fehler (Error = Code)
	PhaseCancelled = "cancelled" // abgebrochen auf Wunsch
)

const maxFailures = 20 // so viele Fehler werden einzeln aufgeführt

// Progress ist der Stand einer Übertragung – für die Anzeige auf beiden Seiten.
type Progress struct {
	Phase        string    `json:"phase"`
	FilesTotal   int       `json:"filesTotal"`
	FilesDone    int       `json:"filesDone"` // kopiert
	FilesSkipped int       `json:"filesSkipped"`
	FilesFailed  int       `json:"filesFailed"`
	FilesLocked  int       `json:"filesLocked"` // davon: von Programmen geöffnet
	BytesTotal   int64     `json:"bytesTotal"`
	BytesDone    int64     `json:"bytesDone"`   // inkl. übersprungener
	BytesCopied  int64     `json:"bytesCopied"` // nur wirklich übertragene (für das Tempo)
	Current      string    `json:"current,omitempty"`
	Error        string    `json:"error,omitempty"`
	Failures     []Failure `json:"failures,omitempty"`
	StartedAt    int64     `json:"startedAt"` // Unix-Millisekunden
	Speed        int64     `json:"speed"`     // Bytes/s der letzten Sekunden, gemessen beim Empfänger
}

// Failure ist eine Datei, die nicht übertragen werden konnte.
type Failure struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

// Plan sagt, welche Freigabe in welchen Zielordner kommt.
type Plan struct {
	Root   string
	Target string
}

// Engine holt auf dem neuen Rechner die Dateien ab.
type Engine struct {
	c       *Client
	windows bool

	mu      sync.Mutex
	p       Progress
	samples []sample // Messpunkte für das Tempo
}

func NewEngine(c *Client) *Engine { return &Engine{c: c, windows: runtime.GOOS == "windows"} }

// Progress liefert eine Kopie des aktuellen Stands.
func (e *Engine) Progress() Progress {
	e.mu.Lock()
	defer e.mu.Unlock()
	p := e.p
	p.Failures = append([]Failure(nil), e.p.Failures...)
	p.Speed = speed(e.samples, p.BytesCopied, time.Now())
	return p
}

// Tempo: Messpunkte (Zeit, übertragene Bytes) höchstens alle 200 ms, die
// letzten speedWindow lang. Gemessen wird hier, beim Speichern, mit der
// eigenen Uhr – so stimmt die Anzeige auf beiden Rechnern.
type sample struct {
	t time.Time
	b int64
}

const speedWindow = 5 * time.Second

func (e *Engine) addSample(b int64) {
	now := time.Now()
	if n := len(e.samples); n > 0 && now.Sub(e.samples[n-1].t) < 200*time.Millisecond {
		return
	}
	e.samples = append(e.samples, sample{now, b})
	// Alte Punkte weg, aber einen vor dem Fenster behalten (Startpunkt).
	for len(e.samples) > 2 && now.Sub(e.samples[1].t) > speedWindow {
		e.samples = e.samples[1:]
	}
}

// speed: übertragene Bytes seit dem ältesten Punkt im Fenster, geteilt
// durch die Zeit bis jetzt. Stockt die Übertragung, sinkt der Wert.
func speed(samples []sample, copied int64, at time.Time) int64 {
	if len(samples) == 0 {
		return 0
	}
	first := samples[0]
	dt := at.Sub(first.t)
	if dt < time.Second {
		return 0 // zu kurz für eine Aussage
	}
	return int64(float64(copied-first.b) / dt.Seconds())
}

func (e *Engine) update(f func(p *Progress)) {
	e.mu.Lock()
	f(&e.p)
	e.mu.Unlock()
}

type job struct {
	root, target string
	entry        Entry
}

// Run überträgt alle Pläne. Einzelne Dateien dürfen scheitern (siehe
// Progress.Failures); bricht die Verbindung ab, endet Run mit Fehler.
func (e *Engine) Run(ctx context.Context, plans []Plan) error {
	e.update(func(p *Progress) {
		*p = Progress{Phase: PhaseListing, StartedAt: time.Now().UnixMilli()}
		e.samples = nil
	})

	var jobs []job
	for _, pl := range plans {
		err := e.c.Walk(pl.Root, func(en Entry) {
			jobs = append(jobs, job{root: pl.Root, target: pl.Target, entry: en})
			e.update(func(p *Progress) {
				p.FilesTotal++
				p.BytesTotal += en.Size
			})
		})
		if err != nil {
			return e.fail(ctx, err)
		}
	}

	e.update(func(p *Progress) { p.Phase = PhaseCopying })
	if err := e.copyAll(ctx, jobs); err != nil {
		return e.fail(ctx, err)
	}
	if ctx.Err() != nil {
		e.update(func(p *Progress) { p.Phase, p.Current = PhaseCancelled, "" })
		return ctx.Err()
	}

	e.update(func(p *Progress) { p.Phase, p.Current = PhaseDone, "" })
	e.c.Done(e.Progress()) // Ergebnis an den Sender; Fehler hier sind egal
	return nil
}

// fail beendet den Lauf. Wer abbrechen will, schließt die Verbindung (nur so
// endet ein laufendes Read) und beendet ctx – das zählt als „abgebrochen“.
func (e *Engine) fail(ctx context.Context, err error) error {
	e.update(func(p *Progress) {
		p.Current = ""
		if ctx.Err() != nil {
			p.Phase = PhaseCancelled
		} else {
			p.Phase, p.Error = PhaseFailed, "connection_lost"
		}
	})
	return err
}

// pipeWindow: so viele Dateien fragt der Empfänger höchstens voraus an.
const pipeWindow = 32

// item ist eine geplante Datei: überspringen, als Fehler melden oder abholen.
type item struct {
	j     job
	dst   string // Zielpfad (beim Abholen)
	skip  bool   // gleich schon vorhanden
	fail  string // Fehlercode, ohne Anfrage
	fetch bool   // Anfrage ist unterwegs
}

// copyAll überträgt alle Dateien. Ein Fehler als Rückgabe heißt: Verbindung
// weg. Probleme einzelner Dateien landen in Failures.
//
// Ein Helfer plant die Dateien und schickt die Anfragen voraus (höchstens
// pipeWindow offen); hier werden die Antworten der Reihe nach gespeichert.
// So wartet keine Seite auf die andere.
func (e *Engine) copyAll(ctx context.Context, jobs []job) error {
	e.c.mu.Lock()
	defer e.c.mu.Unlock()

	items := make(chan *item, pipeWindow)
	slots := make(chan struct{}, pipeWindow)
	stop := make(chan struct{})
	defer close(stop)
	sendErr := make(chan error, 1)

	go func() {
		defer close(items)
		reserved := map[string]bool{} // Ziele, die schon vergeben sind
		for _, j := range jobs {
			if ctx.Err() != nil {
				return // anhalten: keine neuen Anfragen mehr
			}
			it := e.plan(j, reserved)
			if it.fetch {
				select {
				case slots <- struct{}{}:
				case <-stop:
					return
				}
				pr := e.Progress()
				if err := e.c.sendRead(j.root, j.entry.Path, 0, &pr); err != nil {
					sendErr <- err
					return
				}
			}
			select {
			case items <- it:
			case <-stop:
				return
			}
		}
	}()

	for it := range items {
		switch {
		case it.skip:
			// Schon da (z. B. vom abgebrochenen Lauf davor): überspringen.
			e.update(func(p *Progress) {
				p.FilesSkipped++
				p.BytesDone += it.j.entry.Size
			})
		case it.fail != "":
			e.failed(it.j, it.fail)
		default:
			e.update(func(p *Progress) { p.Current = it.j.entry.Path })
			err := e.receive(it)
			<-slots
			if err != nil {
				return err
			}
		}
	}
	select {
	case err := <-sendErr:
		return err
	default:
		return nil
	}
}

// plan entscheidet für eine Datei: überspringen, Fehler oder abholen – und
// unter welchem Namen. reserved verhindert, dass zwei Dateien desselben Laufs
// denselben Namen bekommen (z. B. „Ü“ zerlegt und zusammengesetzt).
func (e *Engine) plan(j job, reserved map[string]bool) *item {
	rel, err := cleanRel(j.entry.Path)
	if err != nil {
		return &item{j: j, fail: "invalid_path"}
	}
	dst := filepath.Join(j.target, targetRel(rel, e.windows))
	fi, statErr := os.Stat(dst)
	if statErr == nil && !reserved[dst] && sameFile(fi, j.entry) {
		return &item{j: j, skip: true}
	}
	if statErr == nil || reserved[dst] {
		dst = freeNameExcept(dst, reserved) // nie überschreiben
	}
	reserved[dst] = true
	return &item{j: j, dst: dst, fetch: true}
}

// receive speichert die Antwort auf eine vorausgeschickte Anfrage. Auch wenn
// hier nicht geschrieben werden kann, wird die Antwort ganz gelesen – sonst
// geriete die Verbindung aus dem Takt.
func (e *Engine) receive(it *item) error {
	// Erst unter Hilfsnamen schreiben: Ein Abbruch hinterlässt so keine
	// halbe Datei unter dem richtigen Namen.
	tmp := it.dst + ".flohh-teil"
	var f *os.File
	var w io.Writer = io.Discard
	localFail := os.MkdirAll(filepath.Dir(it.dst), 0o755)
	if localFail == nil {
		if f, localFail = os.Create(tmp); localFail == nil {
			w = f
		}
	}
	n, err := e.c.recvRead(progressWriter{w, e})
	if f != nil {
		if closeErr := f.Close(); err == nil && closeErr != nil {
			err = localError{closeErr}
		}
	}
	if err == nil && localFail != nil {
		err = localError{localFail}
	}
	if err != nil {
		if f != nil {
			os.Remove(tmp)
		}
		var re RemoteError
		var le localError
		switch {
		case errors.As(err, &re):
			e.failed(it.j, string(re))
		case errors.As(err, &le):
			e.failed(it.j, "write_failed")
		default:
			return err // Verbindung weg
		}
		// Rest der Datei als „erledigt“ zählen, damit der Balken voll wird.
		e.update(func(p *Progress) { p.BytesDone += max(0, it.j.entry.Size-n) })
		return nil
	}

	mtime := time.Unix(it.j.entry.MTime, 0)
	os.Chtimes(tmp, mtime, mtime)
	if err := os.Rename(tmp, it.dst); err != nil {
		os.Remove(tmp)
		e.failed(it.j, "write_failed")
		return nil
	}
	e.update(func(p *Progress) { p.FilesDone++ })
	return nil
}

func (e *Engine) failed(j job, code string) {
	e.update(func(p *Progress) {
		p.FilesFailed++
		if code == errLocked {
			p.FilesLocked++
		}
		if len(p.Failures) < maxFailures {
			p.Failures = append(p.Failures, Failure{Path: j.entry.Path, Error: code})
		}
	})
}

// progressWriter zählt geschriebene Bytes in den Fortschritt.
type progressWriter struct {
	w io.Writer
	e *Engine
}

func (w progressWriter) Write(p []byte) (int, error) {
	n, err := w.w.Write(p)
	w.e.update(func(pr *Progress) {
		pr.BytesDone += int64(n)
		pr.BytesCopied += int64(n)
		w.e.addSample(pr.BytesCopied) // e.mu ist hier gehalten
	})
	return n, err
}

// sameFile: gleiche Größe und gleiches Änderungsdatum (±2 s – FAT und exFAT
// speichern nur grob).
func sameFile(fi os.FileInfo, en Entry) bool {
	d := fi.ModTime().Unix() - en.MTime
	return fi.Mode().IsRegular() && fi.Size() == en.Size && d >= -2 && d <= 2
}

// freeName liefert einen freien Namen: „Urlaub.jpg“ → „Urlaub (2).jpg“ …
func freeName(p string) string { return freeNameExcept(p, nil) }

// freeNameExcept wie freeName, meidet aber auch Namen aus reserved (in
// diesem Lauf schon vergeben, aber noch nicht geschrieben).
func freeNameExcept(p string, reserved map[string]bool) string {
	dir, name := filepath.Split(p)
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	if base == "" { // „.bashrc“
		base, ext = name, ""
	}
	for i := 2; ; i++ {
		cand := filepath.Join(dir, fmt.Sprintf("%s (%d)%s", base, i, ext))
		if _, err := os.Lstat(cand); errors.Is(err, os.ErrNotExist) && !reserved[cand] {
			return cand
		}
	}
}
