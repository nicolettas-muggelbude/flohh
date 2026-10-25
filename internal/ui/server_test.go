package ui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nicolettas-muggelbude/flohh/internal/move"
	"github.com/nicolettas-muggelbude/flohh/internal/peer"
)

// Tests laufen mit "go test ./...". Jede Funktion, die mit Test beginnt und
// *testing.T bekommt, ist ein Test.

const testHost = "127.0.0.1:12345"

// fakeSession ist eine Attrappe für peer.Manager.
type fakeSession struct {
	role    peer.Role
	pairErr error
}

func (f *fakeSession) State() peer.State { return peer.State{Role: f.role, Phase: peer.PhaseIdle} }
func (f *fakeSession) SetRole(r peer.Role) error {
	f.role = r
	return nil
}
func (f *fakeSession) Pair(context.Context, string, string) error     { return f.pairErr }
func (f *fakeSession) PairAddr(context.Context, string, string) error { return f.pairErr }
func (f *fakeSession) Reset()                                         { f.role = "" }

func TestGuard(t *testing.T) {
	h := New("test", testHost, func() {}, &fakeSession{}, move.New())

	// Tabellengetriebener Test: jede Zeile ist ein Fall.
	cases := []struct {
		name   string
		method string
		path   string
		host   string
		header bool
		want   int
	}{
		{"Info vom richtigen Host", "GET", "/api/info", testHost, false, http.StatusOK},
		{"Startseite", "GET", "/", testHost, false, http.StatusOK},
		{"fremder Host (DNS-Rebinding)", "GET", "/api/info", "boese.example:12345", false, http.StatusForbidden},
		{"POST ohne Header", "POST", "/api/quit", testHost, false, http.StatusForbidden},
		// "GET /" passt auf alle Pfade; ein GET auf /api/quit landet also beim
		// Datei-Server und findet dort keine Datei.
		{"GET auf POST-Route", "GET", "/api/quit", testHost, false, http.StatusNotFound},
		{"POST mit Header, unbekannte Route", "POST", "/api/gibtsnicht", testHost, true, http.StatusMethodNotAllowed},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest(c.method, c.path, nil)
			req.Host = c.host
			if c.header {
				req.Header.Set(apiHeader, "1")
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != c.want {
				t.Errorf("Status %d, erwartet %d", rec.Code, c.want)
			}
		})
	}
}

func TestQuit(t *testing.T) {
	called := make(chan struct{})
	h := New("test", testHost, func() { close(called) }, &fakeSession{}, move.New())

	req := httptest.NewRequest("POST", "/api/quit", nil)
	req.Host = testHost
	req.Header.Set(apiHeader, "1")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d, erwartet 200", rec.Code)
	}
	<-called // wartet, bis quit tatsächlich aufgerufen wurde
}

func post(h http.Handler, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", path, strings.NewReader(body))
	req.Host = testHost
	req.Header.Set(apiHeader, "1")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestRoleAndPair(t *testing.T) {
	f := &fakeSession{pairErr: &peer.Error{Code: peer.ErrWrongCode}}
	h := New("test", testHost, func() {}, f, move.New())

	if rec := post(h, "/api/role", `{"role":"send"}`); rec.Code != http.StatusOK || f.role != peer.RoleSend {
		t.Fatalf("Rolle: Status %d, Rolle %q", rec.Code, f.role)
	}

	// Fehler kommen als Code, nicht als Satz – die Oberfläche übersetzt.
	rec := post(h, "/api/pair", `{"peer":"x","code":"123456"}`)
	var got struct{ Error string }
	json.NewDecoder(rec.Body).Decode(&got)
	if rec.Code != http.StatusConflict || got.Error != peer.ErrWrongCode {
		t.Fatalf("Pair: Status %d, Fehler %q", rec.Code, got.Error)
	}

	if rec := post(h, "/api/role", `kein json`); rec.Code != http.StatusBadRequest {
		t.Fatalf("kaputtes JSON: Status %d", rec.Code)
	}
}
