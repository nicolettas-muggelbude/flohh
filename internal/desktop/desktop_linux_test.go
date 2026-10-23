package desktop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeHome lenkt alle Pfade in einen Testordner um.
func fakeHome(t *testing.T) string {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	return home
}

func TestInstall(t *testing.T) {
	home := fakeHome(t)
	if st := State(); !st.Offer || st.Installed {
		t.Fatalf("vorher: %+v", st)
	}
	if err := Install([]byte("<svg/>")); err != nil {
		t.Fatal(err)
	}

	bin := filepath.Join(home, ".local/bin/flohh")
	if fi, err := os.Stat(bin); err != nil || fi.Mode().Perm()&0o100 == 0 {
		t.Fatalf("Programmdatei fehlt oder nicht ausführbar: %v", err)
	}
	entry, err := os.ReadFile(filepath.Join(home, ".local/share/applications/flohh.desktop"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(entry), `Exec="`+bin+`"`) || !strings.Contains(string(entry), "Icon=") {
		t.Errorf("Menü-Eintrag:\n%s", entry)
	}
	if _, err := os.Stat(filepath.Join(home, ".local/share/icons/hicolor/scalable/apps/flohh.svg")); err != nil {
		t.Error("Icon fehlt")
	}
	if st := State(); !st.Installed {
		t.Errorf("nachher: %+v", st)
	}
}

func TestDecline(t *testing.T) {
	fakeHome(t)
	if err := Decline(); err != nil {
		t.Fatal(err)
	}
	if st := State(); st.Offer {
		t.Errorf("nach „Nein danke“ wird weiter angeboten: %+v", st)
	}
}
