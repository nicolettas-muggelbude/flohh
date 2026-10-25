package transfer

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func write(t *testing.T, path string, data []byte, mtime time.Time) {
	t.Helper()
	os.MkdirAll(filepath.Dir(path), 0o755)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	os.Chtimes(path, mtime, mtime)
}

// connect verbindet Server und Client über eine Leitung im Speicher.
func connect(t *testing.T, s *Server) *Client {
	a, b := net.Pipe()
	go s.Serve(a)
	t.Cleanup(func() { a.Close(); b.Close() })
	return NewClient(b)
}

func waitReady(t *testing.T, s *Server) []Root {
	t.Helper()
	for i := 0; i < 200; i++ {
		roots := s.Roots()
		if len(roots) > 0 && roots[0].Ready {
			return roots
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("Zählung wird nicht fertig")
	return nil
}

func TestTransfer(t *testing.T) {
	src, dst, outside := t.TempDir(), t.TempDir(), t.TempDir()
	old := time.Date(2020, 5, 17, 10, 30, 0, 0, time.UTC)

	big := make([]byte, 200_000) // mehrere Datenstücke
	rand.Read(big)
	write(t, filepath.Join(src, "a.txt"), []byte("hallo"), old)
	write(t, filepath.Join(src, "Unterordner", "b.bin"), big, old)
	write(t, filepath.Join(src, "Übersicht.txt"), []byte("macOS-Schreibweise"), old) // NFD
	write(t, filepath.Join(src, "konflikt.txt"), []byte("neu vom alten Rechner"), old)
	write(t, filepath.Join(src, "desktop.ini"), []byte("[.ShellClassInfo]"), old) // wird übersprungen
	write(t, filepath.Join(outside, "geheim.txt"), []byte("geheim"), old)
	os.Symlink(outside, filepath.Join(src, "raus")) // Verknüpfung nach draußen

	// Auf dem neuen Rechner liegt a.txt schon (gleich) und konflikt.txt (anders).
	write(t, filepath.Join(dst, "a.txt"), []byte("hallo"), old)
	write(t, filepath.Join(dst, "konflikt.txt"), []byte("eigene Datei, nicht überschreiben"), time.Now())

	s := NewServer()
	s.Share([]Share{{ID: "documents", Path: src}})
	roots := waitReady(t, s)
	if roots[0].Files != 4 {
		t.Errorf("Freigabe zählt %d Dateien, erwartet 4", roots[0].Files)
	}

	c := connect(t, s)
	e := NewEngine(c)
	if err := e.Run(context.Background(), []Plan{{Root: "documents", Target: dst}}); err != nil {
		t.Fatal(err)
	}

	p := e.Progress()
	if p.Phase != PhaseDone || p.FilesTotal != 4 || p.FilesDone != 3 || p.FilesSkipped != 1 || p.FilesFailed != 0 {
		t.Errorf("Fortschritt: %+v", p)
	}
	if p.BytesDone != p.BytesTotal {
		t.Errorf("Bytes %d von %d", p.BytesDone, p.BytesTotal)
	}

	check := func(rel string, want []byte) {
		t.Helper()
		got, err := os.ReadFile(filepath.Join(dst, rel))
		if err != nil {
			t.Errorf("%s fehlt: %v", rel, err)
			return
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s: falscher Inhalt", rel)
		}
	}
	check(filepath.Join("Unterordner", "b.bin"), big)
	check("Übersicht.txt", []byte("macOS-Schreibweise")) // jetzt NFC
	check("konflikt.txt", []byte("eigene Datei, nicht überschreiben"))
	check("konflikt (2).txt", []byte("neu vom alten Rechner"))

	if fi, err := os.Stat(filepath.Join(dst, "Unterordner", "b.bin")); err == nil && !fi.ModTime().Equal(old) {
		t.Errorf("Änderungsdatum %v, erwartet %v", fi.ModTime(), old)
	}
	for _, rel := range []string{"desktop.ini", "raus", filepath.Join("Unterordner", "b.bin.flohh-teil")} {
		if _, err := os.Lstat(filepath.Join(dst, rel)); err == nil {
			t.Errorf("%s hätte nicht ankommen dürfen", rel)
		}
	}

	if sp, finished := s.Status(); !finished || sp.FilesDone != 3 {
		t.Errorf("Sender kennt das Ergebnis nicht: %+v, fertig=%v", sp, finished)
	}
}

// Der Empfänger darf nichts außerhalb der Freigabe lesen.
func TestNoEscape(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	parent := filepath.Dir(root)
	write(t, filepath.Join(outside, "geheim.txt"), []byte("geheim"), time.Now())
	write(t, filepath.Join(parent, "nachbar.txt"), []byte("nachbar"), time.Now())
	os.Symlink(outside, filepath.Join(root, "raus"))
	t.Cleanup(func() { os.Remove(filepath.Join(parent, "nachbar.txt")) })

	s := NewServer()
	s.Share([]Share{{ID: "documents", Path: root}})
	c := connect(t, s)

	for _, p := range []string{"../nachbar.txt", "/etc/passwd", "raus/geheim.txt", `..\nachbar.txt`, ""} {
		var buf bytes.Buffer
		_, err := c.Read("documents", p, 0, nil, &buf)
		var re RemoteError
		if !errors.As(err, &re) || buf.Len() > 0 {
			t.Errorf("Read(%q): Fehler %v, %d Bytes – erwartet Ablehnung", p, err, buf.Len())
		}
	}
	if _, err := c.Read("pictures", "x.jpg", 0, nil, &bytes.Buffer{}); err == nil {
		t.Error("nicht freigegebener Ordner lesbar")
	}
	// Die Verbindung ist danach weiter benutzbar.
	if _, err := c.Roots(); err != nil {
		t.Errorf("Verbindung kaputt: %v", err)
	}
}

// Eine Datei ohne Leserecht wird als „keine Berechtigung“ gemeldet, die
// Übertragung läuft für die übrigen Dateien weiter.
func TestAccessDenied(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("braucht Unix-Rechte und keinen Administrator")
	}
	src, dst := t.TempDir(), t.TempDir()
	write(t, filepath.Join(src, "ok.txt"), []byte("ok"), time.Now())
	write(t, filepath.Join(src, "geheim.txt"), []byte("geheim"), time.Now())
	os.Chmod(filepath.Join(src, "geheim.txt"), 0)

	s := NewServer()
	s.Share([]Share{{ID: "documents", Path: src}})
	e := NewEngine(connect(t, s))
	if err := e.Run(context.Background(), []Plan{{Root: "documents", Target: dst}}); err != nil {
		t.Fatal(err)
	}
	p := e.Progress()
	if p.FilesDone != 1 || p.FilesFailed != 1 || len(p.Failures) != 1 || p.Failures[0].Error != errAccessDenied {
		t.Errorf("Fortschritt: %+v", p)
	}
}

func TestWindowsName(t *testing.T) {
	cases := map[string]string{
		"Notiz: wichtig?.txt": "Notiz_ wichtig_.txt",
		"con.txt":             "_con.txt",
		"NUL":                 "_NUL",
		"Ende. ":              "Ende",
		"a<b>c|d*e\"f":        "a_b_c_d_e_f",
		"normal.jpg":          "normal.jpg",
		"...":                 "_",
	}
	for in, want := range cases {
		if got := windowsName(in); got != want {
			t.Errorf("windowsName(%q) = %q, erwartet %q", in, got, want)
		}
	}
}

func TestFreeName(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"Urlaub.jpg", "Urlaub (2).jpg", ".bashrc"} {
		write(t, filepath.Join(dir, n), nil, time.Now())
	}
	if got := filepath.Base(freeName(filepath.Join(dir, "Urlaub.jpg"))); got != "Urlaub (3).jpg" {
		t.Errorf("freeName = %q", got)
	}
	if got := filepath.Base(freeName(filepath.Join(dir, ".bashrc"))); got != ".bashrc (2)" {
		t.Errorf("freeName = %q", got)
	}
}

func TestTargetRel(t *testing.T) {
	got := targetRel("Fotos/Übersicht: 2020.txt", true)
	want := filepath.Join("Fotos", "Übersicht_ 2020.txt")
	if got != want {
		t.Errorf("targetRel = %q, erwartet %q", got, want)
	}
	if runtime.GOOS != "windows" {
		if got := targetRel("a:b", false); got != "a:b" {
			t.Errorf("ohne Windows: %q", got)
		}
	}
}
