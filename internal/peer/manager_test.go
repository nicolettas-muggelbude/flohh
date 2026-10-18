package peer

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"testing"
	"time"
)

// loopback liefert die Rundruf-Adresse von m auf 127.0.0.1, sobald m sucht.
func loopback(m **Manager) func() []*net.UDPAddr {
	return func() []*net.UDPAddr {
		(*m).mu.Lock()
		defer (*m).mu.Unlock()
		if (*m).udpAddr == nil {
			return nil
		}
		return []*net.UDPAddr{{IP: net.IPv4(127, 0, 0, 1), Port: (*m).udpAddr.Port}}
	}
}

// setup baut Sender ("alt") und Empfänger ("neu"), die sich nur gegenseitig
// rufen. Wie im echten Betrieb nimmt die Windows-Seite keine Verbindungen an.
func setup(t *testing.T, sendOS, recvOS string) (s, r *Manager) {
	t.Helper()
	var err error
	s, err = New(Config{Name: "alt", OS: sendOS, Listen: sendOS != "windows", Targets: loopback(&r)})
	if err != nil {
		t.Fatal(err)
	}
	r, err = New(Config{Name: "neu", OS: recvOS, Listen: recvOS != "windows", Targets: loopback(&s)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Reset(); r.Reset() })
	if err := s.SetRole(RoleSend); err != nil {
		t.Fatal(err)
	}
	if err := r.SetRole(RoleReceive); err != nil {
		t.Fatal(err)
	}
	return s, r
}

func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("Zeit abgelaufen: %s", what)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func waitPeer(t *testing.T, s *Manager) Peer {
	t.Helper()
	var p Peer
	eventually(t, "Sender findet Empfänger", func() bool {
		st := s.State()
		if len(st.Peers) == 0 {
			return false
		}
		p = st.Peers[0]
		return true
	})
	return p
}

func otherCode(code string) string {
	n, _ := strconv.Atoi(code)
	return fmt.Sprintf("%06d", (n+1)%1_000_000)
}

func errCode(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return fmt.Sprint(err)
}

func TestPairing(t *testing.T) {
	cases := []struct{ name, sendOS, recvOS string }{
		{"Linux zu Linux, Sender ruft an", "linux", "linux"},
		{"Windows zu Linux, Windows ruft an", "windows", "linux"},
		{"Linux zu Windows, Windows-Empfänger ruft an", "linux", "windows"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, r := setup(t, c.sendOS, c.recvOS)
			p := waitPeer(t, s)
			if p.Name != "neu" || p.OS != c.recvOS {
				t.Fatalf("gefunden: %+v", p)
			}
			code := r.State().Code
			ctx := context.Background()

			// Falscher Code wird abgelehnt, die Suche läuft weiter.
			if err := s.Pair(ctx, p.ID, otherCode(code)); errCode(err) != ErrWrongCode {
				t.Fatalf("falscher Code: Fehler %v, erwartet %s", err, ErrWrongCode)
			}
			if st := s.State(); st.Phase != PhaseSearching || st.Error != ErrWrongCode {
				t.Fatalf("nach falschem Code: %+v", st)
			}

			// Richtiger Code koppelt beide Seiten.
			if err := s.Pair(ctx, p.ID, code[:3]+" "+code[3:]); err != nil {
				t.Fatalf("richtiger Code: %v", err)
			}
			eventually(t, "Empfänger verbunden", func() bool { return r.State().Phase == PhaseConnected })
			if st := s.State(); st.Phase != PhaseConnected || st.Partner == nil || st.Partner.Name != "neu" {
				t.Fatalf("Sender: %+v", st)
			}
			if st := r.State(); st.Partner == nil || st.Partner.Name != "alt" || st.Partner.OS != c.sendOS {
				t.Fatalf("Empfänger: %+v", st)
			}

			// Trennt eine Seite, merkt es die andere.
			s.Reset()
			eventually(t, "Empfänger merkt Trennung", func() bool {
				st := r.State()
				return st.Phase == PhaseIdle && st.Error == ErrPeerGone
			})
		})
	}
}

func TestNewCodeAfterThreeFailures(t *testing.T) {
	s, r := setup(t, "linux", "linux")
	p := waitPeer(t, s)
	first := r.State().Code
	for i := 0; i < 3; i++ {
		if err := s.Pair(context.Background(), p.ID, otherCode(first)); errCode(err) != ErrWrongCode {
			t.Fatalf("Versuch %d: %v", i+1, err)
		}
	}
	eventually(t, "neuer Code", func() bool { return r.State().Code != first })
}

func TestBadCode(t *testing.T) {
	s, _ := setup(t, "linux", "linux")
	p := waitPeer(t, s)
	for _, code := range []string{"", "12345", "1234567", "abcdef"} {
		if err := s.Pair(context.Background(), p.ID, code); errCode(err) != ErrBadCode {
			t.Errorf("Code %q: %v, erwartet %s", code, err, ErrBadCode)
		}
	}
}

func TestSenderDials(t *testing.T) {
	cases := []struct {
		send, recv string
		want       bool
	}{
		{"linux", "linux", true},
		{"windows", "linux", true},
		{"linux", "windows", false},
		{"darwin", "windows", false},
		{"windows", "windows", true},
	}
	for _, c := range cases {
		if got := senderDials(c.send, c.recv); got != c.want {
			t.Errorf("senderDials(%s, %s) = %v, erwartet %v", c.send, c.recv, got, c.want)
		}
	}
}
