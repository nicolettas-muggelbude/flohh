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
	files int
	bytes int64
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
	var files int
	var bytes int64
	walkFiles(sh.path, func(_ string, fi fs.FileInfo) error {
		files++
		bytes += fi.Size()
		return nil
	})
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.gen == gen {
		sh.files, sh.bytes, sh.ready = files, bytes, true
	}
}

// Roots liefert die Freigaben mit ihrem Zählstand.
func (s *Server) Roots() []Root {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Root{}
	for _, id := range s.order {
		sh := s.shares[id]
		out = append(out, Root{ID: id, Name: sh.name, Source: sh.path, Files: sh.files, Bytes: sh.bytes, Ready: sh.ready})
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

// Serve beantwortet Anfragen, bis die Verbindung endet.
func (s *Server) Serve(rw io.ReadWriter) error {
	w := newWire(rw)
	for {
		var req request
		if err := w.readJSON(&req); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		if req.Progress != nil {
			s.mu.Lock()
			s.progress = *req.Progress
			s.finished = s.finished || req.Op == "done"
			s.mu.Unlock()
		}
		var err error
		switch req.Op {
		case "roots":
			err = w.writeJSON(response{Roots: s.Roots()})
		case "walk":
			err = s.walk(w, req.Root)
		case "read":
			err = s.read(w, req.Root, req.Path, req.Offset)
		case "done":
			err = w.writeJSON(response{})
		default:
			err = w.writeJSON(response{Error: "unknown_op"})
		}
		if err == nil {
			err = w.flush()
		}
		if err != nil {
			return err
		}
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
	})
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
	f, err := os.Open(full)
	if err != nil {
		return w.writeJSON(response{Error: readErrCode(err)})
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
			return w.writeJSON(response{Error: readErrCode(rerr)})
		}
	}
	return w.writeFrame(frameEnd, nil)
}

// Count zählt Dateien und Bytes unter dir – so, wie sie übertragen würden.
func Count(dir string) (files int, bytes int64) {
	walkFiles(dir, func(_ string, fi fs.FileInfo) error {
		files++
		bytes += fi.Size()
		return nil
	})
	return files, bytes
}

// walkFiles ruft fn für jede normale Datei unter root auf (Pfad relativ, mit
// „/“). Unlesbare Ordner, Verknüpfungen, Systemdateien und Dateien, die nur
// in der Cloud liegen, werden übersprungen. Fehler von fn brechen ab.
func walkFiles(root string, fn func(rel string, fi fs.FileInfo) error) error {
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil // unlesbarer Ordner: Inhalt überspringen
		}
		if !d.Type().IsRegular() || skipName(d.Name()) {
			return nil
		}
		fi, err := d.Info()
		if err != nil || cloudOnly(fi) {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return nil
		}
		return fn(filepath.ToSlash(rel), fi)
	})
}
