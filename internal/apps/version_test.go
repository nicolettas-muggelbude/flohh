package apps

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
		ok   bool
	}{
		{"145.0", "128.5.0esr", 1, true},
		{"128.5.0esr", "145.0", -1, true},
		{"128.5.0esr", "128.5.0", 0, true}, // ESR und normal: gleicher Stand
		{"128.5", "128.5.0", 0, true},
		{"128.10.0", "128.9.1", 1, true}, // Zahlen, nicht Text vergleichen
		{"115.0b3", "115.0", 0, true},
		{"", "128.0", 0, false},
		{"esr", "128.0", 0, false},
	}
	for _, tt := range tests {
		got, ok := CompareVersions(tt.a, tt.b)
		if got != tt.want || ok != tt.ok {
			t.Errorf("CompareVersions(%q, %q) = %d, %v; erwartet %d, %v", tt.a, tt.b, got, ok, tt.want, tt.ok)
		}
	}
}

func TestFit(t *testing.T) {
	tests := []struct{ profile, installed, want string }{
		{"128.5.0", "145.0", FitNewer},
		{"145.0", "128.5.0esr", FitOlder}, // normale Version → ESR: geht so nicht
		{"128.5.0", "128.5.0esr", FitSame},
		{"", "145.0", FitUnknown},
		{"128.5.0", "", FitUnknown},
	}
	for _, tt := range tests {
		if got := Fit(tt.profile, tt.installed); got != tt.want {
			t.Errorf("Fit(%q, %q) = %s, erwartet %s", tt.profile, tt.installed, got, tt.want)
		}
	}
}

func TestProfileVersion(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".thunderbird")
	makeTB(t, dir, iniInstall, "Profiles/abc.default-release", "Profiles/xyz.default")
	os.WriteFile(filepath.Join(dir, "Profiles/abc.default-release/compatibility.ini"),
		[]byte("[Compatibility]\r\nLastVersion=128.5.0_20241119152138/20241119152138\r\nLastOSABI=WINNT_x86_64-msvc\r\n"), 0o644)

	got := findThunderbird("linux", home, "")
	if len(got) != 1 {
		t.Fatalf("gefunden: %+v", got)
	}
	if got[0].Version != "128.5.0" {
		t.Errorf("Version des aktiven Profils %q, erwartet 128.5.0", got[0].Version)
	}
	for _, p := range got[0].Profiles {
		if p.Name == "default" && p.Version != "" {
			t.Errorf("Profil ohne compatibility.ini: Version %q", p.Version)
		}
	}
}

func writeAppINI(t *testing.T, dir, version string) {
	t.Helper()
	os.MkdirAll(dir, 0o755)
	ini := "[App]\nVendor=Mozilla\nName=Thunderbird\nVersion=" + version + "\n\n[Gecko]\nMinVersion=" + version + "\n"
	if err := os.WriteFile(filepath.Join(dir, "application.ini"), []byte(ini), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFindInstallsWindows(t *testing.T) {
	pf, pf86 := t.TempDir(), t.TempDir()
	writeAppINI(t, filepath.Join(pf, "Mozilla Thunderbird"), "128.5.0esr")
	env := map[string]string{"ProgramFiles": pf, "ProgramW6432": pf, "ProgramFiles(x86)": pf86}
	got := findInstalls("windows", "", env)
	if len(got) != 1 || got[0].Version != "128.5.0esr" || got[0].Variant != "windows" {
		t.Errorf("Windows: %+v (ProgramFiles und ProgramW6432 gleich → einmal)", got)
	}
}

func TestFindInstallsLinuxFlatpakUser(t *testing.T) {
	home := t.TempDir()
	writeAppINI(t, filepath.Join(home, ".local/share/flatpak/app/org.mozilla.Thunderbird/current/active/files/lib/thunderbird"), "145.0")
	var flatpak []Install
	for _, in := range findInstalls("linux", home, nil) {
		if in.Variant == "flatpak" {
			flatpak = append(flatpak, in)
		}
	}
	if len(flatpak) != 1 || flatpak[0].Version != "145.0" {
		t.Errorf("Flatpak im Benutzerordner: %+v", flatpak)
	}
}
