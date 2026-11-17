package apps

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// Schritt 2: Versionen. Thunderbird öffnet kein Profil, mit dem zuletzt eine
// neuere Version lief (Schutz vor Datenverlust). Darum vergleichen wir die
// Version des Profils (compatibility.ini) mit dem, was auf dem neuen Rechner
// installiert ist.

// Install ist ein installiertes Thunderbird.
type Install struct {
	Variant string `json:"variant"` // windows | macos | system | snap | flatpak
	Version string `json:"version"` // z. B. "128.5.0esr", leer = unbekannt
	Dir     string `json:"dir"`     // Programmordner (mit application.ini)
}

// Wie passt ein installiertes Thunderbird zum Profil vom alten Rechner?
const (
	FitNewer   = "newer"   // installiert ist neuer: Profil wird beim Start aktualisiert
	FitSame    = "same"    // gleiche Version
	FitOlder   = "older"   // installiert ist älter: Profil wird nicht geöffnet
	FitUnknown = "unknown" // eine der Versionen ist unbekannt
)

// Fit vergleicht die Version des Profils (profile) mit der installierten
// (installed).
func Fit(profile, installed string) string {
	switch c, ok := CompareVersions(installed, profile); {
	case !ok:
		return FitUnknown
	case c > 0:
		return FitNewer
	case c < 0:
		return FitOlder
	default:
		return FitSame
	}
}

// CompareVersions vergleicht Thunderbird-Versionen wie „128.5.0esr“ und
// „145.0“: -1, 0, 1. Verglichen werden nur die Zahlen; „esr“ ist kein
// Unterschied (ESR 128.5.0 und normale 128.5.0 sind derselbe Stand).
// ok ist false, wenn eine Version fehlt oder nicht mit einer Zahl beginnt.
func CompareVersions(a, b string) (int, bool) {
	x, okA := versionNumbers(a)
	y, okB := versionNumbers(b)
	if !okA || !okB {
		return 0, false
	}
	for i := 0; i < max(len(x), len(y)); i++ {
		var p, q int
		if i < len(x) {
			p = x[i]
		}
		if i < len(y) {
			q = y[i]
		}
		if p != q {
			if p < q {
				return -1, true
			}
			return 1, true
		}
	}
	return 0, true
}

// versionNumbers: „128.5.0esr“ → [128 5 0], „115.0b3“ → [115 0].
func versionNumbers(v string) ([]int, bool) {
	var out []int
	for _, part := range strings.Split(strings.TrimSpace(v), ".") {
		end := 0
		for end < len(part) && part[end] >= '0' && part[end] <= '9' {
			end++
		}
		if end == 0 {
			break
		}
		n, err := strconv.Atoi(part[:end])
		if err != nil {
			break
		}
		out = append(out, n)
		if end < len(part) {
			break // „0esr“, „0b3“: danach nichts mehr vergleichen
		}
	}
	return out, len(out) > 0
}

// profileVersion liest, mit welcher Version ein Profil zuletzt lief:
// compatibility.ini, LastVersion=128.5.0_20241119152138/20241119152138.
func profileVersion(profileDir string) string {
	sections, err := readINI(filepath.Join(profileDir, "compatibility.ini"))
	if err != nil {
		return ""
	}
	for _, s := range sections {
		if v := s.keys["LastVersion"]; v != "" {
			v, _, _ = strings.Cut(v, "_")
			return v
		}
	}
	return ""
}

// ThunderbirdInstalls findet installierte Thunderbirds auf diesem Rechner.
func ThunderbirdInstalls() []Install {
	home, _ := os.UserHomeDir()
	env := map[string]string{}
	for _, k := range []string{"ProgramFiles", "ProgramFiles(x86)", "ProgramW6432", "LOCALAPPDATA"} {
		env[k] = os.Getenv(k)
	}
	return findInstalls(runtime.GOOS, home, env)
}

func installCandidates(goos, home string, env map[string]string) []candidate {
	switch goos {
	case "windows":
		var out []candidate
		for _, k := range []string{"ProgramFiles", "ProgramW6432", "ProgramFiles(x86)", "LOCALAPPDATA"} {
			if env[k] != "" {
				out = append(out, candidate{"windows", filepath.Join(env[k], "Mozilla Thunderbird")})
			}
		}
		return out
	case "darwin":
		return []candidate{
			{"macos", "/Applications/Thunderbird.app/Contents/Resources"},
			{"macos", filepath.Join(home, "Applications", "Thunderbird.app", "Contents", "Resources")},
		}
	default:
		out := []candidate{
			{"system", "/usr/lib/thunderbird"},
			{"system", "/usr/lib64/thunderbird"},
			{"system", "/usr/lib/thunderbird-esr"},
			{"system", "/opt/thunderbird"},
			{"snap", "/snap/thunderbird/current/usr/lib/thunderbird"},
		}
		for _, base := range []string{"/var/lib/flatpak/app", filepath.Join(home, ".local", "share", "flatpak", "app")} {
			for _, id := range []string{"org.mozilla.Thunderbird", "net.thunderbird.Thunderbird"} {
				out = append(out, candidate{"flatpak", filepath.Join(base, id, "current", "active", "files", "lib", "thunderbird")})
			}
		}
		return out
	}
}

func findInstalls(goos, home string, env map[string]string) []Install {
	var out []Install
	seen := map[string]bool{}
	for _, c := range installCandidates(goos, home, env) {
		sections, err := readINI(filepath.Join(c.dir, "application.ini"))
		if err != nil {
			continue
		}
		real, err := filepath.EvalSymlinks(c.dir)
		if err != nil || seen[real] {
			continue // z. B. ProgramFiles und ProgramW6432 zeigen auf denselben Ordner
		}
		seen[real] = true
		in := Install{Variant: c.variant, Dir: c.dir}
		for _, s := range sections {
			if s.name == "App" {
				in.Version = s.keys["Version"]
			}
		}
		out = append(out, in)
	}
	return out
}
