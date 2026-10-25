package transfer

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Server läuft auf dem alten Rechner und liefert Dateien aus den
// freigegebenen Ordnern – nur lesend und nur von dort.
type Server struct {
	mu       sync.Mutex
	order    []string          // Reihenfolge der Freigaben
	shares   map[string]*share // ID (Ordnerart) → Freigabe
	gen      int               // erhöht sich bei jeder neuen Freigabe
	progress Progress          // letzter Stand vom Empfänger
	finished bool              // Empfänger hat „fertig“ gemeldet
}

type share struct {
	name  string
	path  string
	stats Stats
	ready bool
}

func NewServer() *Server { return &Server{shares: map[string]*share{}} }

// Share beschreibt einen Ordner, der freigegeben werden soll.
type Share struct {
	ID   string
	Name string // Anzeigename (eigene Ordner)
	Path string
}

// Share gibt Ordner frei und ersetzt frühere Freigaben. Dateien und Größe
// werden im Hintergrund gezählt.
func (s *Server) Share(list []Share) {
	s.mu.Lock()
	s.gen++
	gen := s.gen
	s.order = nil
	s.shares = map[string]*share{}
	for _, sh := range list {
		x := &share{name: sh.Name, path: sh.Path}
		s.order = append(s.order, sh.ID)
		s.shares[sh.ID] = x
		go s.count(gen, x)
	}
	s.mu.Unlock()
}

func (s *Server) count(gen int, sh *share) {
	st := Count(sh.path)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.gen == gen {
		sh.stats, sh.ready = st, true
	}
}

// Roots liefert die Freigaben mit ihrem Zählstand.
func (s *Server) Roots() []Root {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Root{}
	for _, id := range s.order {
		sh := s.shares[id]
		out = append(out, Root{ID: id, Name: sh.name, Source: sh.path, Stats: sh.stats, Ready: sh.ready})
	}
	return out
}

// Status liefert den letzten Stand, den der Empfänger gemeldet hat.
func (s *Server) Status() (p Progress, finished bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p = s.progress
	p.Failures = append([]Failure(nil), p.Failures...)
	return p, s.finished
}

// Tempo bei vielen kleinen Dateien: Der Empfänger schickt Anfragen voraus.
// Kleine Dateien liest der Sender dann schon parallel ein (das Öffnen kostet
// unter Windows wegen des Virenscanners spürbar Zeit), beantwortet wird
// trotzdem streng der Reihe nach.
const (
	smallFile = 1 << 20 // bis 1 MiB: vorab in den Speicher lesen
	readAhead = 8       // so viele kleine Dateien gleichzeitig öffnen
	maxQueued = 64      // so viele Anfragen höchstens in Arbeit (≤ 64 MiB)
)

// task ist eine Anfrage in Arbeit.
type task struct {
	req   request
	ready chan struct{} // nil: wird beim Antworten erledigt (nicht vorab gelesen)
	data  []byte        // vorab gelesene kleine Datei
	fail  *response     // Fehlerantwort statt Inhalt
}

// Serve beantwortet Anfragen, bis die Verbindung endet.
func (s *Server) Serve(rw io.ReadWriter) error {
	w := newWire(rw)
	tasks := make(chan *task, maxQueued)
	done := make(chan struct{})
	defer close(done)
	readErr := make(chan error, 1)
	sem := make(chan struct{}, readAhead)

	// Anfragen lesen und kleine Dateien schon einmal anfangen zu lesen.
	go func() {
		defer close(tasks)
		for {
			var req request
			if err := w.readJSON(&req); err != nil {
				readErr <- err
				return
			}
			if req.Progress != nil {
				s.mu.Lock()
				s.progress = *req.Progress
				s.finished = s.finished || req.Op == "done"
				s.mu.Unlock()
			}
			t := &task{req: req}
			if req.Op == "read" {
				s.prefetch(t, sem)
			}
			select {
			case tasks <- t:
			case <-done:
				return
			}
		}
	}()

	// Antworten der Reihe nach schreiben.
	for t := range tasks {
		if err := s.answer(w, t); err != nil {
			return err
		}
		if len(tasks) == 0 { // nichts mehr in der Schlange: abschicken
			if err := w.flush(); err != nil {
				return err
			}
		}
	}
	if err := <-readErr; !errors.Is(err, io.EOF) {
		return err
	}
	return w.flush()
}

// prefetch liest eine kleine Datei im Hintergrund in den Speicher.
func (s *Server) prefetch(t *task, sem chan struct{}) {
	full, err := s.resolve(t.req.Root, t.req.Path)
	if err != nil {
		t.fail = &response{Error: "not_found"}
		return
	}
	fi, err := os.Stat(full)
	if err != nil || !fi.Mode().IsRegular() || fi.Size() > smallFile || t.req.Offset != 0 {
		return // groß oder ungewöhnlich: beim Antworten direkt streamen
	}
	t.ready = make(chan struct{})
	go func() {
		defer close(t.ready)
		sem <- struct{}{}
		defer func() { <-sem }()
		f, err := openFile(full)
		if err != nil {
			r := readError(err, full)
			t.fail = &r
			return
		}
		defer f.Close()
		data, err := io.ReadAll(io.LimitReader(f, smallFile+1))
		if err != nil {
			r := readError(err, full)
			t.fail = &r
			return
		}
		t.data = data
	}()
}

func (s *Server) answer(w *wire, t *task) error {
	switch t.req.Op {
	case "roots":
		return w.writeJSON(response{Roots: s.Roots()})
	case "walk":
		return s.walk(w, t.req.Root)
	case "read":
		if t.ready != nil {
			<-t.ready
		}
		if t.fail != nil {
			return w.writeJSON(*t.fail)
		}
		if t.ready == nil {
			return s.read(w, t.req.Root, t.req.Path, t.req.Offset)
		}
		if err := w.writeJSON(response{Size: int64(len(t.data))}); err != nil {
			return err
		}
		for p := t.data; len(p) > 0; {
			n := min(len(p), chunkSize)
			if err := w.writeFrame(frameData, p[:n]); err != nil {
				return err
			}
			p = p[n:]
		}
		t.data = nil
		return w.writeFrame(frameEnd, nil)
	case "done":
		return w.writeJSON(response{})
	default:
		return w.writeJSON(response{Error: "unknown_op"})
	}
}

func (s *Server) sharePath(id string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sh, ok := s.shares[id]
	if !ok {
		return "", false
	}
	return sh.path, true
}

func (s *Server) walk(w *wire, id string) error {
	root, ok := s.sharePath(id)
	if !ok {
		return w.writeJSON(response{Error: "not_shared"})
	}
	var batch []Entry
	send := func() error {
		if len(batch) == 0 {
			return nil
		}
		err := w.writeJSON(response{Entries: batch})
		batch = batch[:0]
		return err
	}
	err := walkFiles(root, func(rel string, fi fs.FileInfo) error {
		batch = append(batch, Entry{Path: rel, Size: fi.Size(), MTime: fi.ModTime().Unix()})
		if len(batch) >= 500 {
			return send()
		}
		return nil
	}, nil)
	if err == nil {
		err = send()
	}
	if err != nil {
		return err
	}
	return w.writeFrame(frameEnd, nil)
}

// resolve macht aus Freigabe und relativem Pfad einen echten Pfad und
// stellt sicher, dass er – auch über Verknüpfungen – in der Freigabe bleibt.
func (s *Server) resolve(id, rel string) (string, error) {
	root, ok := s.sharePath(id)
	if !ok {
		return "", errors.New("nicht freigegeben")
	}
	rel, err := cleanRel(rel)
	if err != nil {
		return "", err
	}
	full := filepath.Join(root, filepath.FromSlash(rel))
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	real, err := filepath.EvalSymlinks(full)
	if err != nil {
		return "", err
	}
	r, err := filepath.Rel(realRoot, real)
	if err != nil || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) {
		return "", errors.New("außerhalb der Freigabe")
	}
	return real, nil
}

func (s *Server) read(w *wire, id, rel string, off int64) error {
	full, err := s.resolve(id, rel)
	if err != nil {
		return w.writeJSON(response{Error: "not_found"})
	}
	f, err := openFile(full)
	if err != nil {
		return w.writeJSON(readError(err, full))
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil || !fi.Mode().IsRegular() || off < 0 || off > fi.Size() {
		return w.writeJSON(response{Error: "read_failed"})
	}
	if off > 0 {
		if _, err := f.Seek(off, io.SeekStart); err != nil {
			return w.writeJSON(response{Error: "read_failed"})
		}
	}
	if err := w.writeJSON(response{Size: fi.Size() - off}); err != nil {
		return err
	}
	buf := make([]byte, chunkSize)
	for {
		n, rerr := f.Read(buf)
		if n > 0 {
			if err := w.writeFrame(frameData, buf[:n]); err != nil {
				return err
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			// Lesefehler mitten in der Datei (z. B. Bereich gesperrt):
			// statt „Ende“ ein Fehler.
			return w.writeJSON(readError(rerr, full))
		}
	}
	return w.writeFrame(frameEnd, nil)
}

// openFile öffnet eine Datei zum Lesen. Als Variable, damit Messläufe die
// Verzögerung eines Virenscanners nachstellen können.
var openFile = os.Open

// Stats zählt einen Ordner so, wie er übertragen würde – und was dabei
// bewusst wegbleibt.
type Stats struct {
	Files  int   `json:"files"`
	Bytes  int64 `json:"bytes"`
	System int   `json:"system"` // Systemdateien (desktop.ini, Thumbs.db, .DS_Store)
	Cloud  int   `json:"cloud"`  // nur in der Cloud (OneDrive „Nur online verfügbar“)
}

// Count zählt Dateien und Bytes unter dir.
func Count(dir string) Stats {
	var st Stats
	walkFiles(dir, func(_ string, fi fs.FileInfo) error {
		st.Files++
		st.Bytes += fi.Size()
		return nil
	}, &st)
	return st
}

// isCloudOnly ist eine Variable, damit Tests Cloud-Dateien nachstellen können.
var isCloudOnly = cloudOnly

// walkFiles ruft fn für jede normale Datei unter root auf (Pfad relativ, mit
// „/“). Unlesbare Ordner, Verknüpfungen, Systemdateien und Dateien, die nur
// in der Cloud liegen, werden übersprungen (gezählt in skipped, falls nicht
// nil). Fehler von fn brechen ab.
func walkFiles(root string, fn func(rel string, fi fs.FileInfo) error, skipped *Stats) error {
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil // unlesbarer Ordner: Inhalt überspringen
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if skipName(d.Name()) || skipInternal(root, p) {
			if skipped != nil {
				skipped.System++
			}
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return nil
		}
		if isCloudOnly(fi) {
			if skipped != nil {
				skipped.Cloud++
			}
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return nil
		}
		return fn(filepath.ToSlash(rel), fi)
	})
}
