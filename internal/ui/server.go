// Package ui liefert die Oberfläche (HTML/CSS/JS) und die Steuer-Schnittstelle
// (/api/...) aus, über die die Oberfläche mit der App spricht.
package ui

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"runtime"

	"github.com/nicolettas-muggelbude/flohh/internal/browse"
	"github.com/nicolettas-muggelbude/flohh/internal/desktop"
	"github.com/nicolettas-muggelbude/flohh/internal/move"
	"github.com/nicolettas-muggelbude/flohh/internal/peer"
	"github.com/nicolettas-muggelbude/flohh/web"
)

// Session ist die Kopplung mit dem anderen Rechner (peer.Manager). Als
// Schnittstelle, damit Tests eine Attrappe einsetzen können.
type Session interface {
	State() peer.State
	SetRole(peer.Role) error
	Pair(ctx context.Context, id, code string) error
	PairAddr(ctx context.Context, addr, code string) error
	Reset()
}

// Move ist der Umzug über die gekoppelte Verbindung (move.Coordinator).
type Move interface {
	State() move.State
	Share(ids []string) error
	Start(ids []string) error
	AddFolder(path string) (string, error)
	RemoveFolder(id string) error
	SetTarget(id, path string) error
	Stop() error
}

// apiHeader muss jede ändernde Anfrage (POST usw.) mitschicken. Eine fremde
// Webseite im Browser kann diesen Header nicht setzen, ohne dass der Browser
// vorher nachfragt (CORS-Preflight) – und diese Nachfrage beantworten wir nie.
// So kann keine Webseite heimlich Aktionen in Flohh auslösen.
const apiHeader = "X-Flohh"

// server hält alles, was die Handler brauchen. Kleingeschrieben = nur in
// diesem Paket sichtbar; von außen gibt es nur New.
type server struct {
	version string
	quit    func() // beendet die App
	session Session
	move    Move
}

// New baut den HTTP-Handler der Oberfläche.
//
// host ist die Adresse, unter der die Oberfläche läuft (z. B.
// "127.0.0.1:53817"). Anfragen mit einem anderen Host-Header werden
// abgelehnt; das schützt vor DNS-Rebinding-Angriffen.
func New(version, host string, quit func(), session Session, mv Move) http.Handler {
	s := &server{version: version, quit: quit, session: session, move: mv}

	mux := http.NewServeMux()
	mux.Handle("GET /", http.FileServerFS(web.Files))
	mux.HandleFunc("GET /api/info", s.handleInfo)
	mux.HandleFunc("POST /api/quit", s.handleQuit)
	mux.HandleFunc("GET /api/state", s.handleState)
	mux.HandleFunc("POST /api/role", s.handleRole)
	mux.HandleFunc("POST /api/pair", s.handlePair)
	mux.HandleFunc("POST /api/reset", s.handleReset)
	mux.HandleFunc("POST /api/menu", s.handleMenu)
	mux.HandleFunc("GET /api/move", s.handleMoveState)
	mux.HandleFunc("POST /api/move/share", s.handleMoveIDs(mv.Share))
	mux.HandleFunc("POST /api/move/start", s.handleMoveIDs(mv.Start))
	mux.HandleFunc("POST /api/move/stop", s.handleMoveStop)
	mux.HandleFunc("POST /api/move/add", s.handleMoveAdd)
	mux.HandleFunc("POST /api/move/remove", s.handleMoveRemove)
	mux.HandleFunc("POST /api/move/target", s.handleMoveTarget)
	mux.HandleFunc("GET /api/browse", s.handleBrowse)

	return guard(host, mux)
}

// guard lässt nur Anfragen durch, die wirklich von der eigenen Oberfläche
// kommen können, und reicht sie dann an next weiter.
func guard(host string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != host {
			http.Error(w, "falscher Host", http.StatusForbidden)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Header.Get(apiHeader) == "" {
			http.Error(w, "Header "+apiHeader+" fehlt", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// info ist die Antwort von GET /api/info. Die `json:"..."`-Angaben legen
// fest, wie die Felder im JSON heißen.
//
// Grundsatz für alle /api-Antworten: nur Daten und Codes, keine Sätze. Texte
// für Menschen entstehen in der Oberfläche (web/i18n.js), damit sie übersetzt
// werden können. Fehler später z. B. als {"error": "disk_full"}.
type info struct {
	Version  string         `json:"version"`
	OS       string         `json:"os"`
	Arch     string         `json:"arch"`
	Hostname string         `json:"hostname"`
	Menu     desktop.Status `json:"menu"` // Linux: Aufnahme ins Anwendungsmenü
}

func (s *server) handleInfo(w http.ResponseWriter, r *http.Request) {
	// Leer, wenn unbekannt – den Text dafür wählt die Oberfläche in ihrer Sprache.
	hostname, _ := os.Hostname()
	writeJSON(w, info{
		Version:  s.version,
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		Hostname: hostname,
		Menu:     desktop.State(),
	})
}

// handleMenu nimmt flohh ins Anwendungsmenü auf („install“) oder merkt sich
// „Nein danke“ („decline“).
func (s *server) handleMenu(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action string `json:"action"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	var err error
	switch req.Action {
	case "install":
		var icon []byte
		if icon, err = web.Files.ReadFile("img/icon.svg"); err == nil {
			err = desktop.Install(icon)
		}
	case "decline":
		err = desktop.Decline()
	default:
		writeErrorCode(w, http.StatusBadRequest, "bad_request")
		return
	}
	if err != nil {
		log.Printf("Anwendungsmenü: %v", err)
		writeErrorCode(w, http.StatusConflict, "menu_failed")
		return
	}
	writeJSON(w, desktop.State())
}

func (s *server) handleQuit(w http.ResponseWriter, r *http.Request) {
	// Erst antworten, dann beenden – sonst bekommt der Browser keine Antwort.
	writeJSON(w, map[string]bool{"ok": true})
	go s.quit()
}

func (s *server) handleState(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.session.State())
}

func (s *server) handleRole(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Role peer.Role `json:"role"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if err := s.session.SetRole(req.Role); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, s.session.State())
}

func (s *server) handlePair(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Peer string `json:"peer"` // gefundener Rechner …
		Addr string `json:"addr"` // … oder eingetippte Adresse
		Code string `json:"code"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	var err error
	if req.Addr != "" {
		err = s.session.PairAddr(r.Context(), req.Addr, req.Code)
	} else {
		err = s.session.Pair(r.Context(), req.Peer, req.Code)
	}
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, s.session.State())
}

func (s *server) handleReset(w http.ResponseWriter, r *http.Request) {
	s.session.Reset()
	writeJSON(w, s.session.State())
}

func (s *server) handleMoveState(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.move.State())
}

// handleMoveIDs nimmt {"ids": ["documents", "custom-1", …]} für
// Freigeben bzw. Starten.
func (s *server) handleMoveIDs(action func([]string) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			IDs []string `json:"ids"`
		}
		if !readJSON(w, r, &req) {
			return
		}
		if err := action(req.IDs); err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, s.move.State())
	}
}

// handleMoveAdd: eigener Ordner beim Sender; Antwort mit seiner ID.
func (s *server) handleMoveAdd(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	id, err := s.move.AddFolder(req.Path)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, map[string]any{"id": id, "move": s.move.State()})
}

func (s *server) handleMoveRemove(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if err := s.move.RemoveFolder(req.ID); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, s.move.State())
}

func (s *server) handleMoveTarget(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID   string `json:"id"`
		Path string `json:"path"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if err := s.move.SetTarget(req.ID, req.Path); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, s.move.State())
}

// handleBrowse: Ordnerauswahl auf diesem Rechner (?path=…, leer = home).
func (s *server) handleBrowse(w http.ResponseWriter, r *http.Request) {
	l, err := browse.List(r.URL.Query().Get("path"))
	if err != nil {
		writeErrorCode(w, http.StatusConflict, "browse_failed")
		return
	}
	writeJSON(w, l)
}

func (s *server) handleMoveStop(w http.ResponseWriter, r *http.Request) {
	s.move.Stop() // läuft nichts, ist auch gut
	writeJSON(w, s.move.State())
}

func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeErrorCode(w, http.StatusBadRequest, "bad_request")
		return false
	}
	return true
}

// writeError antwortet mit {"error": "<code>"}; die Oberfläche übersetzt ihn.
func writeError(w http.ResponseWriter, err error) {
	code := "internal"
	var pe *peer.Error
	var me *move.Error
	switch {
	case errors.As(err, &pe):
		code = pe.Code
	case errors.As(err, &me):
		code = me.Code
	}
	writeErrorCode(w, http.StatusConflict, code)
}

func writeErrorCode(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": code})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("JSON-Antwort fehlgeschlagen: %v", err)
	}
}
