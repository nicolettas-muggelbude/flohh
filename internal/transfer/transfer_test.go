package transfer

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
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
func connect(t testing.TB, s *Server) *Client {
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

// Systemdateien und Cloud-Dateien werden nicht übertragen, aber gezählt.
// Cloud-Dateien gibt es nur unter Windows; hier stellen wir sie nach.
func TestSkippedCounted(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	write(t, filepath.Join(src, "normal.txt"), []byte("da"), time.Now())
	write(t, filepath.Join(src, "desktop.ini"), []byte("[x]"), time.Now())
	write(t, filepath.Join(src, "Fotos", "Thumbs.db"), []byte("x"), time.Now())
	write(t, filepath.Join(src, "Fotos", "urlaub.jpg.cloud"), []byte("nur online"), time.Now())
	isCloudOnly = func(fi os.FileInfo) bool { return filepath.Ext(fi.Name()) == ".cloud" }
	defer func() { isCloudOnly = cloudOnly }()

	st := Count(src)
	if st.Files != 1 || st.System != 2 || st.Cloud != 1 {
		t.Errorf("Zählung: %+v", st)
	}
	s := NewServer()
	s.Share([]Share{{ID: "documents", Path: src}})
	if r := waitReady(t, s); r[0].Cloud != 1 || r[0].System != 2 {
		t.Errorf("Freigabe meldet: %+v", r[0])
	}
	e := NewEngine(connect(t, s))
	if err := e.Run(context.Background(), []Plan{{Root: "documents", Target: dst}}); err != nil {
		t.Fatal(err)
	}
	if p := e.Progress(); p.FilesTotal != 1 || p.FilesDone != 1 {
		t.Errorf("übertragen: %+v", p)
	}
}

// Zwei Dateien desselben Laufs mit gleichem Zielnamen (macOS-Schreibweise
// und normale Schreibweise von „Ü“): keine darf die andere überschreiben.
func TestSameTargetTwice(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("macOS behandelt beide Schreibweisen als denselben Namen")
	}
	src, dst := t.TempDir(), t.TempDir()
	write(t, filepath.Join(src, "Übung.txt"), []byte("zerlegt"), time.Now())
	write(t, filepath.Join(src, "Übung.txt"), []byte("zusammengesetzt"), time.Now())

	s := NewServer()
	s.Share([]Share{{ID: "documents", Path: src}})
	e := NewEngine(connect(t, s))
	if err := e.Run(context.Background(), []Plan{{Root: "documents", Target: dst}}); err != nil {
		t.Fatal(err)
	}
	a, _ := os.ReadFile(filepath.Join(dst, "Übung.txt"))
	b, _ := os.ReadFile(filepath.Join(dst, "Übung (2).txt"))
	if got := string(a) + "+" + string(b); got != "zerlegt+zusammengesetzt" && got != "zusammengesetzt+zerlegt" {
		t.Errorf("Inhalte: %q", got)
	}
}

// Anhalten mitten im Lauf: Status „abgebrochen“, keine halben Dateien.
func TestCancelMidway(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	for i := 0; i < 200; i++ {
		write(t, filepath.Join(src, fmt.Sprintf("d%03d.txt", i)), []byte("inhalt"), time.Now())
	}
	openFile = func(name string) (*os.File, error) {
		time.Sleep(5 * time.Millisecond)
		return os.Open(name)
	}
	defer func() { openFile = os.Open }()

	s := NewServer()
	s.Share([]Share{{ID: "documents", Path: src}})
	e := NewEngine(connect(t, s))
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(100*time.Millisecond, cancel)
	e.Run(ctx, []Plan{{Root: "documents", Target: dst}})

	p := e.Progress()
	if p.Phase != PhaseCancelled || p.FilesDone == 0 || p.FilesDone >= 200 {
		t.Errorf("Fortschritt nach Anhalten: %+v", p)
	}
	entries, _ := os.ReadDir(dst)
	for _, en := range entries {
		if filepath.Ext(en.Name()) == ".flohh-teil" {
			t.Errorf("halbe Datei übrig: %s", en.Name())
		}
	}
	if len(entries) != p.FilesDone {
		t.Errorf("%d Dateien im Ziel, %d gemeldet", len(entries), p.FilesDone)
	}
}

func TestSpeed(t *testing.T) {
	t0 := time.Unix(1000, 0)
	at := func(s float64) time.Time { return t0.Add(time.Duration(s * float64(time.Second))) }
	samples := []sample{{at(0), 0}, {at(1), 50_000_000}, {at(2), 100_000_000}}

	// 100 MB in 2 s gemessen, jetzt bei 2 s: 50 MB/s.
	if got := speed(samples, 100_000_000, at(2)); got != 50_000_000 {
		t.Errorf("Tempo %d, erwartet 50 MB/s", got)
	}
	// Stockt es 2 s lang, sinkt das Tempo: 100 MB in 4 s.
	if got := speed(samples, 100_000_000, at(4)); got != 25_000_000 {
		t.Errorf("nach Stocken %d, erwartet 25 MB/s", got)
	}
	// Unter einer Sekunde Messzeit: keine Aussage.
	if got := speed(samples[:1], 10, at(0.5)); got != 0 {
		t.Errorf("zu früh: %d", got)
	}
}

// Das gemeldete Tempo passt zum tatsächlichen Tempo, auch wenn der Sender
// gebremst ist (hier: 50 ms je Datei beim Öffnen, 8 parallel → ca. 4 s).
func TestSpeedMatchesReality(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	data := make([]byte, 20_000)
	for i := 0; i < 640; i++ {
		write(t, filepath.Join(src, fmt.Sprintf("d%03d", i)), data, time.Now())
	}
	openFile = func(name string) (*os.File, error) {
		time.Sleep(50 * time.Millisecond)
		return os.Open(name)
	}
	defer func() { openFile = os.Open }()

	s := NewServer()
	s.Share([]Share{{ID: "documents", Path: src}})
	e := NewEngine(connect(t, s))
	done := make(chan struct{})
	go func() {
		e.Run(context.Background(), []Plan{{Root: "documents", Target: dst}})
		close(done)
	}()
	time.Sleep(1500 * time.Millisecond)
	p1, t1 := e.Progress(), time.Now()
	time.Sleep(1 * time.Second)
	p2, t2 := e.Progress(), time.Now()
	<-done

	real := float64(p2.BytesCopied-p1.BytesCopied) / t2.Sub(t1).Seconds()
	shown := float64(p2.Speed)
	if real == 0 || shown < real*0.6 || shown > real*1.4 {
		t.Errorf("angezeigt %.1f MB/s, tatsächlich %.1f MB/s", shown/1e6, real/1e6)
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
