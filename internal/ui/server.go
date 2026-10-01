// Package ui liefert die Oberfläche (HTML/CSS/JS) und die Steuer-Schnittstelle
// (/api/...) aus, über die die Oberfläche mit der App spricht.
package ui

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"runtime"

	"github.com/nicolettas-muggelbude/umzugskarton/web"
)

// apiHeader muss jede ändernde Anfrage (POST usw.) mitschicken. Eine fremde
// Webseite im Browser kann diesen Header nicht setzen, ohne dass der Browser
// vorher nachfragt (CORS-Preflight) – und diese Nachfrage beantworten wir nie.
// So kann keine Webseite heimlich Aktionen in Umzugskarton auslösen.
const apiHeader = "X-Umzugskarton"

// server hält alles, was die Handler brauchen. Kleingeschrieben = nur in
// diesem Paket sichtbar; von außen gibt es nur New.
type server struct {
	version string
	quit    func() // beendet die App
}

// New baut den HTTP-Handler der Oberfläche.
//
// host ist die Adresse, unter der die Oberfläche läuft (z. B.
// "127.0.0.1:53817"). Anfragen mit einem anderen Host-Header werden
// abgelehnt; das schützt vor DNS-Rebinding-Angriffen.
func New(version, host string, quit func()) http.Handler {
	s := &server{version: version, quit: quit}

	mux := http.NewServeMux()
	mux.Handle("GET /", http.FileServerFS(web.Files))
	mux.HandleFunc("GET /api/info", s.handleInfo)
	mux.HandleFunc("POST /api/quit", s.handleQuit)

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
type info struct {
	Version  string `json:"version"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Hostname string `json:"hostname"`
}

func (s *server) handleInfo(w http.ResponseWriter, r *http.Request) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unbekannt"
	}
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

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("JSON-Antwort fehlgeschlagen: %v", err)
	}
}
