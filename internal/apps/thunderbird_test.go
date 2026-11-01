package apps

import (
	"os"
	"path/filepath"
	"testing"
)

// makeTB legt einen Thunderbird-Datenordner mit profiles.ini an.
func makeTB(t *testing.T, dir, ini string, profiles ...string) {
	t.Helper()
	for _, p := range profiles {
		os.MkdirAll(filepath.Join(dir, filepath.FromSlash(p)), 0o755)
	}
	os.MkdirAll(dir, 0o755)
	if err := os.WriteFile(filepath.Join(dir, "profiles.ini"), []byte(ini), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Heutiges Format: [Install…]-Abschnitt nennt das aktive Profil.
const iniInstall = `[Install4F96D1932A9F858E]
Default=Profiles/abc.default-release
Locked=1

[Profile1]
Name=default
IsRelative=1
Path=Profiles/xyz.default
Default=1

[Profile0]
Name=default-release
IsRelative=1
Path=Profiles/abc.default-release

[General]
StartWithLastProfile=1
Version=2
`

func TestLinuxVariants(t *testing.T) {
	home := t.TempDir()
	makeTB(t, filepath.Join(home, ".thunderbird"), iniInstall, "Profiles/abc.default-release", "Profiles/xyz.default")
	makeTB(t, filepath.Join(home, "snap/thunderbird/common/.thunderbird"), iniInstall, "Profiles/abc.default-release")
	makeTB(t, filepath.Join(home, ".var/app/org.mozilla.Thunderbird/.thunderbird"), iniInstall, "Profiles/abc.default-release")
	// Flatpak-Ordner ohne Profile: nicht anbieten
	makeTB(t, filepath.Join(home, ".var/app/net.thunderbird.Thunderbird/.thunderbird"), iniInstall)

	got := findThunderbird("linux", home, "")
	want := map[string]string{
		"thunderbird-system":  "Thunderbird",
		"thunderbird-snap":    "Thunderbird-Snap",
		"thunderbird-flatpak": "Thunderbird-Flatpak",
	}
	if len(got) != len(want) {
		t.Fatalf("gefunden: %+v", got)
	}
	for _, a := range got {
		if want[a.ID] != a.Name {
			t.Errorf("%s: Name %q", a.ID, a.Name)
		}
		if a.Active != "default-release" {
			t.Errorf("%s: aktives Profil %q, erwartet default-release (aus [Install…])", a.ID, a.Active)
		}
	}
	// Im Systempaket-Ordner gibt es zwei Profile, nur eins aktiv.
	for _, a := range got {
		if a.Variant == "system" && len(a.Profiles) != 2 {
			t.Errorf("Systempaket: %d Profile", len(a.Profiles))
		}
	}
}

func TestWindowsAndMac(t *testing.T) {
	home, appData := t.TempDir(), t.TempDir()
	makeTB(t, filepath.Join(appData, "Thunderbird"), iniInstall, "Profiles/abc.default-release")
	if got := findThunderbird("windows", home, appData); len(got) != 1 || got[0].ID != "thunderbird-windows" || got[0].Name != "Thunderbird" {
		t.Errorf("Windows: %+v", got)
	}
	makeTB(t, filepath.Join(home, "Library/Thunderbird"), iniInstall, "Profiles/abc.default-release")
	if got := findThunderbird("darwin", home, ""); len(got) != 1 || got[0].Variant != "macos" {
		t.Errorf("macOS: %+v", got)
	}
}

func TestActiveProfileFallbacks(t *testing.T) {
	cases := []struct {
		name, ini, want string
		profiles        []string
	}{
		{"nur Default=1 (älteres Format)", `[Profile0]
Name=alt
IsRelative=1
Path=a.alt

[Profile1]
Name=haupt
IsRelative=1
Path=b.haupt
Default=1
`, "haupt", []string{"a.alt", "b.haupt"}},
		{"einziges Profil ohne Kennzeichen", `[Profile0]
Name=einzig
IsRelative=1
Path=Profiles/e.einzig
`, "einzig", []string{"Profiles/e.einzig"}},
		{"Install zeigt auf gelöschtes Profil → Default=1", `[InstallX]
Default=Profiles/weg

[Profile0]
Name=da
IsRelative=1
Path=Profiles/da
Default=1
`, "da", []string{"Profiles/da"}},
		{"Windows-Schreibweise mit Backslash", `[Profile0]
Name=win
IsRelative=1
Path=Profiles\w.win
Default=1
`, "win", []string{"Profiles/w.win"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			makeTB(t, dir, c.ini, c.profiles...)
			profiles, err := readProfiles(dir)
			if err != nil {
				t.Fatal(err)
			}
			active := ""
			for _, p := range profiles {
				if p.Active {
					active = p.Name
				}
			}
			if active != c.want {
				t.Errorf("aktiv %q, erwartet %q (%+v)", active, c.want, profiles)
			}
		})
	}
}

func TestAbsoluteProfilePath(t *testing.T) {
	dir, elsewhere := t.TempDir(), t.TempDir()
	makeTB(t, dir, "[Profile0]\nName=extern\nIsRelative=0\nPath="+elsewhere+"\nDefault=1\n")
	profiles, err := readProfiles(dir)
	if err != nil || len(profiles) != 1 || profiles[0].Path != elsewhere || !profiles[0].Active {
		t.Errorf("Profil außerhalb: %+v, %v", profiles, err)
	}
}

func TestNothingInstalled(t *testing.T) {
	if got := findThunderbird("linux", t.TempDir(), ""); len(got) != 0 {
		t.Errorf("nichts installiert, gefunden: %+v", got)
	}
}
