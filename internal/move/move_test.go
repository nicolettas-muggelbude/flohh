package move

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nicolettas-muggelbude/flohh/internal/transfer"
)

func TestIsAppData(t *testing.T) {
	cases := map[string]bool{
		`C:\Users\Nicole\AppData`:                 true,
		`C:\Users\Nicole\AppData\Roaming\Mozilla`: true,
		`C:\Users\Nicole\Documents`:               false,
		`C:\Projekte\AppDataTool`:                 false,
		"/home/nicole/.config":                    false,
	}
	for p, want := range cases {
		if got := isAppData(p); got != want {
			t.Errorf("isAppData(%q) = %v, erwartet %v", p, got, want)
		}
	}
}

func TestDefaultTarget(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	got := defaultTarget(transfer.Root{ID: "custom-1", Name: "Projekte"})
	if want := filepath.Join(home, "Projekte"); got != want {
		t.Errorf("Ziel %q, erwartet %q", got, want)
	}
}

func TestAddRemoveFolder(t *testing.T) {
	c := New()
	c.server = transfer.NewServer() // wie nach dem Koppeln als Sender
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("abc"), 0o644)

	id, err := c.AddFolder(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.AddFolder(dir); err == nil {
		t.Error("doppelter Ordner angenommen")
	}
	if _, err := c.AddFolder(filepath.Join(dir, "a.txt")); err == nil {
		t.Error("Datei als Ordner angenommen")
	}
	if err := c.Share([]string{id}); err != nil {
		t.Fatal(err)
	}
	roots := c.server.Roots()
	if len(roots) != 1 || roots[0].Name != filepath.Base(dir) || roots[0].Source != dir {
		t.Errorf("Freigabe: %+v", roots)
	}
	if err := c.RemoveFolder(id); err != nil {
		t.Fatal(err)
	}
	if len(c.local) != 0 {
		t.Errorf("Ordner nicht entfernt: %+v", c.local)
	}
}
