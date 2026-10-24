//go:build linux

package folders

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUserDirsGerman(t *testing.T) {
	home := t.TempDir()
	cfg := filepath.Join(home, ".config")
	os.MkdirAll(cfg, 0o755)
	os.WriteFile(filepath.Join(cfg, "user-dirs.dirs"), []byte(`# von xdg-user-dirs-update
XDG_DESKTOP_DIR="$HOME/Schreibtisch"
XDG_DOWNLOAD_DIR="$HOME/Downloads"
XDG_PICTURES_DIR="$HOME/Bilder"
XDG_MUSIC_DIR="$HOME/Musik"
XDG_VIDEOS_DIR="$HOME/"
`), 0o644)
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")

	want := map[Kind]string{
		Desktop:   filepath.Join(home, "Schreibtisch"),
		Pictures:  filepath.Join(home, "Bilder"),
		Music:     filepath.Join(home, "Musik"),
		Documents: filepath.Join(home, "Documents"), // fehlt → englischer Standard
		Videos:    filepath.Join(home, "Videos"),    // zeigt auf $HOME → Standard
	}
	for k, w := range want {
		if got := Path(k); got != w {
			t.Errorf("Path(%s) = %q, erwartet %q", k, got, w)
		}
	}

	// Known liefert nur, was es wirklich gibt.
	os.Mkdir(filepath.Join(home, "Bilder"), 0o755)
	got := Known()
	if len(got) != 1 || got[0].Kind != Pictures {
		t.Errorf("Known() = %+v, erwartet nur Bilder", got)
	}
}
