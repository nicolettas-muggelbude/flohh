package transfer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

	mu sync.Mutex
	p  Progress
}

func NewEngine(c *Client) *Engine { return &Engine{c: c, windows: runtime.GOOS == "windows"} }

// Progress liefert eine Kopie des aktuellen Stands.
func (e *Engine) Progress() Progress {
	e.mu.Lock()
	defer e.mu.Unlock()
	p := e.p
	p.Failures = append([]Failure(nil), e.p.Failures...)
	return p
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
	for _, j := range jobs {
		if ctx.Err() != nil {
			e.update(func(p *Progress) { p.Phase, p.Current = PhaseCancelled, "" })
			return ctx.Err()
		}
		e.update(func(p *Progress) { p.Current = j.entry.Path })
		if err := e.copyOne(j); err != nil {
			return e.fail(ctx, err)
		}
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

// copyOne überträgt eine Datei. Ein Fehler als Rückgabe heißt: Verbindung
// weg, alles abbrechen. Probleme mit der einzelnen Datei landen in Failures.
func (e *Engine) copyOne(j job) error {
	rel, err := cleanRel(j.entry.Path)
	if err != nil {
		e.failed(j, "invalid_path")
		return nil
	}
	dst := filepath.Join(j.target, targetRel(rel, e.windows))

	if fi, err := os.Stat(dst); err == nil {
		if sameFile(fi, j.entry) {
			// Schon da (z. B. vom abgebrochenen Lauf davor): überspringen.
			e.update(func(p *Progress) {
				p.FilesSkipped++
				p.BytesDone += j.entry.Size
			})
			return nil
		}
		dst = freeName(dst) // nie überschreiben
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		e.failed(j, "write_failed")
		return nil
	}
	// Erst unter Hilfsnamen schreiben: Ein Abbruch hinterlässt so keine
	// halbe Datei unter dem richtigen Namen.
	tmp := dst + ".flohh-teil"
	f, err := os.Create(tmp)
	if err != nil {
		e.failed(j, "write_failed")
		return nil
	}
	pr := e.Progress()
	n, err := e.c.Read(j.root, j.entry.Path, 0, &pr, progressWriter{f, e})
	closeErr := f.Close()
	if err == nil && closeErr != nil {
		err = localError{closeErr}
	}
	if err != nil {
		os.Remove(tmp)
		var re RemoteError
		var le localError
		switch {
		case errors.As(err, &re):
			e.failed(j, string(re))
		case errors.As(err, &le):
			e.failed(j, "write_failed")
		default:
			return err // Verbindung weg
		}
		// Rest der Datei als „erledigt“ zählen, damit der Balken voll wird.
		e.update(func(p *Progress) { p.BytesDone += max(0, j.entry.Size-n) })
		return nil
	}

	mtime := time.Unix(j.entry.MTime, 0)
	os.Chtimes(tmp, mtime, mtime)
	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		e.failed(j, "write_failed")
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
	f *os.File
	e *Engine
}

func (w progressWriter) Write(p []byte) (int, error) {
	n, err := w.f.Write(p)
	w.e.update(func(pr *Progress) {
		pr.BytesDone += int64(n)
		pr.BytesCopied += int64(n)
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
func freeName(p string) string {
	dir, name := filepath.Split(p)
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	if base == "" { // „.bashrc“
		base, ext = name, ""
	}
	for i := 2; ; i++ {
		cand := filepath.Join(dir, fmt.Sprintf("%s (%d)%s", base, i, ext))
		if _, err := os.Lstat(cand); errors.Is(err, os.ErrNotExist) {
			return cand
		}
	}
}
