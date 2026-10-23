//go:build !windows

package stick

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOnRemovable(t *testing.T) {
	cases := map[string]bool{
		"/media/nicole/STICK/Flohh-Linux": true,
		"/run/media/nicole/STICK/flohh":   true,
		"/Volumes/STICK/Flohh-Mac-Intel":  true,
		"/home/nicole/Schreibtisch/flohh": false,
		"/home/nicole/.local/bin/flohh":   false,
		"/tmp/flohh-run-123/Flohh-Linux":  false,
		"/mediathek/Flohh-Linux":          false,
	}
	for path, want := range cases {
		if got := onRemovable(path); got != want {
			t.Errorf("onRemovable(%q) = %v, erwartet %v", path, got, want)
		}
	}
}

func TestCleanupKeepsOwnCopy(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	own := filepath.Join(tmp, prefix+"eigene")
	old := filepath.Join(tmp, prefix+"alt")
	other := filepath.Join(tmp, "fremd")
	for _, d := range []string{own, old, other} {
		os.Mkdir(d, 0o755)
	}
	t.Setenv(envCopy, own)

	Cleanup()

	for d, want := range map[string]bool{own: true, old: false, other: true} {
		_, err := os.Stat(d)
		if exists := err == nil; exists != want {
			t.Errorf("%s vorhanden = %v, erwartet %v", filepath.Base(d), exists, want)
		}
	}
}
