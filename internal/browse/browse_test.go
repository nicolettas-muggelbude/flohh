package browse

import (
	"os"
	"path/filepath"
	"testing"
)

func TestList(t *testing.T) {
	dir := t.TempDir()
	for _, d := range []string{"Zebra", "apfel", ".versteckt"} {
		os.Mkdir(filepath.Join(dir, d), 0o755)
	}
	os.WriteFile(filepath.Join(dir, "datei.txt"), nil, 0o644) // keine Ordner → nicht gelistet

	l, err := List(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, d := range l.Dirs {
		names = append(names, d.Name)
	}
	if len(names) != 3 || names[0] != ".versteckt" || names[1] != "apfel" || names[2] != "Zebra" {
		t.Errorf("Ordner: %v", names)
	}
	if !l.Dirs[0].Hidden || l.Dirs[1].Hidden {
		t.Errorf("versteckt falsch erkannt: %+v", l.Dirs)
	}
	if l.Parent != filepath.Dir(dir) {
		t.Errorf("Elternordner %q", l.Parent)
	}
	if len(l.Places) == 0 || l.Places[0].Name != "home" {
		t.Errorf("Orte: %+v", l.Places)
	}

	if _, err := List("relativ/pfad"); err == nil {
		t.Error("relativer Pfad angenommen")
	}
}
