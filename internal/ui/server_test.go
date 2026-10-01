package ui

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Tests laufen mit "go test ./...". Jede Funktion, die mit Test beginnt und
// *testing.T bekommt, ist ein Test.

const testHost = "127.0.0.1:12345"

func TestGuard(t *testing.T) {
	h := New("test", testHost, func() {})

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
	h := New("test", testHost, func() { close(called) })

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
