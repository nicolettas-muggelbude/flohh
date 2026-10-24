// Package move verbindet Kopplung und Übertragung zu einem Umzug: Der alte
// Rechner gibt Benutzerordner und eigene Ordner frei, der neue holt sie in
// seine Benutzerordner bzw. in selbst gewählte Ziele ab. Die Oberfläche
// fragt den Stand über State ab.
package move

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/nicolettas-muggelbude/flohh/internal/folders"
	"github.com/nicolettas-muggelbude/flohh/internal/peer"
	"github.com/nicolettas-muggelbude/flohh/internal/transfer"
)

// Fehlercodes für die Oberfläche.
const (
	ErrNotConnected = "not_connected"
	ErrNothing      = "nothing_selected"
	ErrBusy         = "busy"
	ErrBadFolder    = "bad_folder" // kein Ordner / nicht lesbar
	ErrDuplicate    = "duplicate"  // schon in der Liste
	ErrBadTarget    = "bad_target" // Ziel ungültig
)

// Error ist ein Fehler mit Code für die Oberfläche.
type Error struct{ Code string }

func (e *Error) Error() string { return e.Code }

// Folder ist ein Ordner in der Anzeige.
type Folder struct {
	ID      string       `json:"id"`             // Art (documents …) oder custom-N
	Kind    folders.Kind `json:"kind,omitempty"` // leer bei eigenen Ordnern
	Name    string       `json:"name,omitempty"` // Ordnername bei eigenen Ordnern
	Source  string       `json:"source"`         // Pfad auf dem alten Rechner
	Target  string       `json:"target,omitempty"`
	Files   int          `json:"files"`
	Bytes   int64        `json:"bytes"`
	Ready   bool         `json:"ready"`             // Zählung fertig
	Shared  bool         `json:"shared,omitempty"`  // Sender: freigegeben
	Custom  bool         `json:"custom,omitempty"`  // eigener Ordner
	AppData bool         `json:"appdata,omitempty"` // Programmeinstellungen von Windows
}

// State ist der Stand für die Oberfläche.
type State struct {
	Connected bool               `json:"connected"`
	Role      peer.Role          `json:"role,omitempty"`
	Folders   []Folder           `json:"folders"`            // Sender: eigene Ordner; Empfänger: Angebot
	Shared    bool               `json:"shared"`             // Sender hat freigegeben
	Running   bool               `json:"running"`            // Übertragung läuft
	Stopping  bool               `json:"stopping,omitempty"` // hält nach der aktuellen Datei an
	Progress  *transfer.Progress `json:"progress,omitempty"`
}

// Coordinator hält den Umzug für genau eine Verbindung zusammen.
type Coordinator struct {
	mu       sync.Mutex
	conn     net.Conn
	role     peer.Role
	local    []Folder // Sender: Benutzerordner und eigene Ordner
	nextID   int      // Sender: Zähler für custom-N
	server   *transfer.Server
	client   *transfer.Client
	offer    []transfer.Root   // Empfänger: zuletzt gesehenes Angebot
	targets  map[string]string // Empfänger: selbst gewählte Ziele
	engine   *transfer.Engine
	running  bool
	stop     context.CancelFunc
	stopping bool
	lastProg *transfer.Progress // Empfänger: Stand des letzten Laufs
}

func New() *Coordinator { return &Coordinator{} }

// Connected ist peer.Config.OnConnected: Es arbeitet mit der Verbindung,
// bis sie endet.
func (c *Coordinator) Connected(conn net.Conn, _ peer.Peer, role peer.Role) {
	c.mu.Lock()
	c.conn, c.role = conn, role
	c.local, c.offer, c.lastProg, c.stopping = nil, nil, nil, false
	c.targets, c.nextID = map[string]string{}, 0
	c.mu.Unlock()

	if role == peer.RoleSend {
		srv := transfer.NewServer()
		c.mu.Lock()
		c.server = srv
		c.mu.Unlock()
		c.addKnown()
		srv.Serve(conn) // bis der Empfänger auflegt
	} else {
		cl := transfer.NewClient(conn)
		c.mu.Lock()
		c.client = cl
		c.mu.Unlock()
		c.pollOffer(cl) // bis die Verbindung weg ist
	}

	c.mu.Lock()
	if c.stop != nil {
		c.stop()
	}
	c.conn, c.role, c.server, c.client, c.engine = nil, "", nil, nil, nil
	c.running, c.stop = false, nil
	c.mu.Unlock()
}

// addKnown nimmt beim Sender die Benutzerordner in die Liste auf.
func (c *Coordinator) addKnown() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, f := range folders.Known() {
		c.local = append(c.local, Folder{ID: string(f.Kind), Kind: f.Kind, Source: f.Path})
		go c.count(string(f.Kind), f.Path)
	}
}

// count zählt einen Ordner des Senders im Hintergrund.
func (c *Coordinator) count(id, path string) {
	files, bytes := transfer.Count(path)
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := range c.local {
		if c.local[i].ID == id {
			c.local[i].Files, c.local[i].Bytes, c.local[i].Ready = files, bytes, true
		}
	}
}

// AddFolder nimmt beim Sender einen eigenen Ordner in die Liste auf.
func (c *Coordinator) AddFolder(path string) (string, error) {
	path = filepath.Clean(path)
	if fi, err := os.Stat(path); err != nil || !fi.IsDir() || !filepath.IsAbs(path) {
		return "", &Error{ErrBadFolder}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.server == nil {
		return "", &Error{ErrNotConnected}
	}
	for _, f := range c.local {
		if f.Source == path {
			return "", &Error{ErrDuplicate}
		}
	}
	c.nextID++
	id := fmt.Sprintf("custom-%d", c.nextID)
	c.local = append(c.local, Folder{ID: id, Name: folderName(path), Source: path, Custom: true})
	go c.count(id, path)
	return id, nil
}

// RemoveFolder nimmt einen eigenen Ordner wieder aus der Liste.
func (c *Coordinator) RemoveFolder(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, f := range c.local {
		if f.ID == id && f.Custom {
			c.local = append(c.local[:i], c.local[i+1:]...)
			return nil
		}
	}
	return &Error{ErrBadFolder}
}

// folderName: „C:\Projekte“ → „Projekte“, „D:\“ → „D“, „/“ → „root“.
// (Daten, kein Anzeigetext – der Name wird auch Ordnername beim Ziel.)
func folderName(path string) string {
	name := filepath.Base(path)
	if name == string(filepath.Separator) || name == "." || name == "" {
		if vol := strings.TrimSuffix(filepath.VolumeName(path), ":"); vol != "" {
			return vol
		}
		return "root"
	}
	return name
}

// isAppData erkennt Programmeinstellungen von Windows (…\AppData\…).
func isAppData(path string) bool {
	p := strings.ToLower(strings.ReplaceAll(path, `\`, "/"))
	return strings.HasSuffix(p, "/appdata") || strings.Contains(p, "/appdata/")
}

// pollOffer fragt beim Empfänger alle 2 Sekunden das Angebot des Senders ab.
// Scheitert das, ist die Verbindung weg.
func (c *Coordinator) pollOffer(cl *transfer.Client) {
	for {
		roots, err := cl.Roots()
		if err != nil {
			return
		}
		c.mu.Lock()
		c.offer = roots
		c.mu.Unlock()
		time.Sleep(2 * time.Second)
	}
}

// defaultTarget: Benutzerordner landen im eigenen Benutzerordner gleicher
// Art, eigene Ordner in einem gleichnamigen Ordner im persönlichen Ordner.
func defaultTarget(r transfer.Root) string {
	if k := folders.Kind(r.ID); folders.Valid(k) {
		return folders.Path(k)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, transfer.SafeName(r.Name))
}

func (c *Coordinator) target(r transfer.Root) string {
	if t, ok := c.targets[r.ID]; ok {
		return t
	}
	return defaultTarget(r)
}

// State liefert den Stand für die Oberfläche.
func (c *Coordinator) State() State {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := State{Connected: c.conn != nil, Role: c.role, Folders: []Folder{}, Running: c.running, Stopping: c.stopping}
	switch {
	case c.role == peer.RoleSend && c.server != nil:
		shared := map[string]bool{}
		for _, r := range c.server.Roots() {
			shared[r.ID] = true
		}
		for _, f := range c.local {
			f.Shared = shared[f.ID]
			f.AppData = isAppData(f.Source)
			s.Folders = append(s.Folders, f)
		}
		s.Shared = len(shared) > 0
		if p, _ := c.server.Status(); p.Phase != "" {
			s.Progress = &p
		}
	case c.role == peer.RoleReceive:
		for _, r := range c.offer {
			f := Folder{ID: r.ID, Name: r.Name, Source: r.Source, Target: c.target(r),
				Files: r.Files, Bytes: r.Bytes, Ready: r.Ready, AppData: isAppData(r.Source)}
			if k := folders.Kind(r.ID); folders.Valid(k) {
				f.Kind = k
			} else {
				f.Custom = true
			}
			s.Folders = append(s.Folders, f)
		}
		if c.engine != nil {
			p := c.engine.Progress()
			s.Progress = &p
		} else if c.lastProg != nil {
			s.Progress = c.lastProg
		}
	}
	return s
}

// Share gibt beim Sender die gewählten Ordner frei.
func (c *Coordinator) Share(ids []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.server == nil {
		return &Error{ErrNotConnected}
	}
	var list []transfer.Share
	for _, id := range ids {
		for _, f := range c.local {
			if f.ID == id {
				list = append(list, transfer.Share{ID: f.ID, Name: f.Name, Path: f.Source})
			}
		}
	}
	if len(list) == 0 {
		return &Error{ErrNothing}
	}
	c.server.Share(list)
	return nil
}

// SetTarget legt beim Empfänger das Ziel eines Ordners fest.
func (c *Coordinator) SetTarget(id, path string) error {
	if path == "" || !filepath.IsAbs(path) {
		return &Error{ErrBadTarget}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.client == nil {
		return &Error{ErrNotConnected}
	}
	c.targets[id] = filepath.Clean(path)
	return nil
}

// Start beginnt beim Empfänger die Übertragung der gewählten Ordner.
func (c *Coordinator) Start(ids []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.client == nil {
		return &Error{ErrNotConnected}
	}
	if c.running {
		return &Error{ErrBusy}
	}
	var plans []transfer.Plan
	for _, id := range ids {
		for _, r := range c.offer {
			if r.ID != id {
				continue
			}
			target := c.target(r)
			if target == "" || os.MkdirAll(target, 0o755) != nil {
				return &Error{ErrBadTarget}
			}
			plans = append(plans, transfer.Plan{Root: r.ID, Target: target})
		}
	}
	if len(plans) == 0 {
		return &Error{ErrNothing}
	}

	ctx, cancel := context.WithCancel(context.Background())
	eng := transfer.NewEngine(c.client)
	c.engine, c.running, c.stop, c.stopping = eng, true, cancel, false
	go func() {
		eng.Run(ctx, plans)
		p := eng.Progress()
		c.mu.Lock()
		c.lastProg, c.engine, c.running, c.stopping = &p, nil, false, false
		cancel()
		c.stop = nil
		c.mu.Unlock()
	}()
	return nil
}

// Stop hält die Übertragung nach der aktuellen Datei an.
func (c *Coordinator) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.running || c.stop == nil {
		return errors.New("läuft nicht")
	}
	c.stop()
	c.stopping = true
	return nil
}
