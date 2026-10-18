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

	"github.com/nicolettas-muggelbude/flohh/internal/peer"
	"github.com/nicolettas-muggelbude/flohh/web"
)

// Session ist die Kopplung mit dem anderen Rechner (peer.Manager). Als
// Schnittstelle, damit Tests eine Attrappe einsetzen können.
type Session interface {
	State() peer.State
	SetRole(peer.Role) error
	Pair(ctx context.Context, id, code string) error
	Reset()
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
}

// New baut den HTTP-Handler der Oberfläche.
//
// host ist die Adresse, unter der die Oberfläche läuft (z. B.
// "127.0.0.1:53817"). Anfragen mit einem anderen Host-Header werden
// abgelehnt; das schützt vor DNS-Rebinding-Angriffen.
func New(version, host string, quit func(), session Session) http.Handler {
	s := &server{version: version, quit: quit, session: session}

	mux := http.NewServeMux()
	mux.Handle("GET /", http.FileServerFS(web.Files))
	mux.HandleFunc("GET /api/info", s.handleInfo)
	mux.HandleFunc("POST /api/quit", s.handleQuit)
	mux.HandleFunc("GET /api/state", s.handleState)
	mux.HandleFunc("POST /api/role", s.handleRole)
	mux.HandleFunc("POST /api/pair", s.handlePair)
	mux.HandleFunc("POST /api/reset", s.handleReset)

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
	Version  string `json:"version"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Hostname string `json:"hostname"`
}

func (s *server) handleInfo(w http.ResponseWriter, r *http.Request) {
	// Leer, wenn unbekannt – den Text dafür wählt die Oberfläche in ihrer Sprache.
	hostname, _ := os.Hostname()
	writeJSON(w, info{
		Version:  s.version,
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		Hostname: hostname,
	})
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
		Peer string `json:"peer"`
		Code string `json:"code"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if err := s.session.Pair(r.Context(), req.Peer, req.Code); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, s.session.State())
}

func (s *server) handleReset(w http.ResponseWriter, r *http.Request) {
	s.session.Reset()
	writeJSON(w, s.session.State())
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
	if errors.As(err, &pe) {
		code = pe.Code
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
