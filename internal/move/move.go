// Package move verbindet Kopplung und Übertragung zu einem Umzug: Der alte
// Rechner gibt Benutzerordner frei, der neue holt sie in seine eigenen
// Benutzerordner ab. Die Oberfläche fragt den Stand über State ab.
package move

import (
	"context"
	"errors"
	"net"
	"os"
	"sync"
	"time"

	"github.com/nicolettas-muggelbude/flohh/internal/folders"
	"github.com/nicolettas-muggelbude/flohh/internal/peer"
	"github.com/nicolettas-muggelbude/flohh/internal/transfer"
)

// Fehlercodes für die Oberfläche.
const (
	ErrNotConnected = "not_connected"
	ErrWrongRole    = "wrong_role"
	ErrNothing      = "nothing_selected"
	ErrBusy         = "busy"
)

// Error ist ein Fehler mit Code für die Oberfläche.
type Error struct{ Code string }

func (e *Error) Error() string { return e.Code }

// Folder ist ein Benutzerordner in der Anzeige.
type Folder struct {
	Kind   folders.Kind `json:"kind"`
	Path   string       `json:"path,omitempty"`   // Sender: Quelle; Empfänger: Ziel
	Files  int          `json:"files"`            //
	Bytes  int64        `json:"bytes"`            //
	Ready  bool         `json:"ready"`            // Zählung fertig
	Shared bool         `json:"shared,omitempty"` // Sender: freigegeben
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
	local    []Folder // Sender: gezählte eigene Ordner
	server   *transfer.Server
	client   *transfer.Client
	offer    []transfer.Root // Empfänger: zuletzt gesehenes Angebot
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
	c.mu.Unlock()

	if role == peer.RoleSend {
		srv := transfer.NewServer()
		c.mu.Lock()
		c.server = srv
		c.mu.Unlock()
		go c.countLocal()
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

// countLocal zählt beim Sender die eigenen Benutzerordner für die Anzeige.
func (c *Coordinator) countLocal() {
	known := folders.Known()
	c.mu.Lock()
	c.local = make([]Folder, len(known))
	for i, f := range known {
		c.local[i] = Folder{Kind: f.Kind, Path: f.Path}
	}
	c.mu.Unlock()
	for i, f := range known {
		files, bytes := transfer.Count(f.Path)
		c.mu.Lock()
		if i < len(c.local) && c.local[i].Kind == f.Kind {
			c.local[i].Files, c.local[i].Bytes, c.local[i].Ready = files, bytes, true
		}
		c.mu.Unlock()
	}
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
			f.Shared = shared[string(f.Kind)]
			s.Folders = append(s.Folders, f)
		}
		s.Shared = len(shared) > 0
		if p, _ := c.server.Status(); p.Phase != "" {
			s.Progress = &p
		}
	case c.role == peer.RoleReceive:
		for _, r := range c.offer {
			k := folders.Kind(r.ID)
			s.Folders = append(s.Folders, Folder{Kind: k, Path: folders.Path(k), Files: r.Files, Bytes: r.Bytes, Ready: r.Ready})
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
func (c *Coordinator) Share(kinds []folders.Kind) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.server == nil {
		return &Error{ErrNotConnected}
	}
	ids, paths := []string{}, map[string]string{}
	for _, k := range kinds {
		for _, f := range c.local {
			if f.Kind == k {
				ids = append(ids, string(k))
				paths[string(k)] = f.Path
			}
		}
	}
	if len(ids) == 0 {
		return &Error{ErrNothing}
	}
	c.server.Share(ids, paths)
	return nil
}

// Start beginnt beim Empfänger die Übertragung der gewählten Ordner.
func (c *Coordinator) Start(kinds []folders.Kind) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.client == nil {
		return &Error{ErrNotConnected}
	}
	if c.running {
		return &Error{ErrBusy}
	}
	var plans []transfer.Plan
	for _, k := range kinds {
		offered := false
		for _, r := range c.offer {
			offered = offered || r.ID == string(k)
		}
		target := folders.Path(k)
		if !offered || target == "" {
			continue
		}
		if err := os.MkdirAll(target, 0o755); err != nil {
			continue
		}
		plans = append(plans, transfer.Plan{Root: string(k), Target: target})
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
