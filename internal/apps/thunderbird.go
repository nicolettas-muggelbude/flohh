// Package apps erkennt Programme, deren Daten beim Umzug mitkommen sollen.
// Schritt 1: Thunderbird – erkennen, welche Installationen Daten haben und
// welches Profil aktiv ist. Übertragen wird der ganze Thunderbird-Ordner.
// Schritt 2 (version.go): Versionen von Profil und Installation vergleichen.
package apps

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// App ist ein gefundener Datenordner eines Programms.
type App struct {
	ID       string    `json:"id"`      // eindeutig, z. B. "thunderbird-snap"
	App      string    `json:"app"`     // "thunderbird"
	Variant  string    `json:"variant"` // windows | macos | system | snap | flatpak
	Name     string    `json:"name"`    // Ordnername beim Ziel, z. B. "Thunderbird-Snap"
	Dir      string    `json:"dir"`     // Datenordner mit profiles.ini
	Profiles []Profile `json:"profiles"`
	Active   string    `json:"active"`  // Name des aktiven Profils
	Version  string    `json:"version"` // Version, mit der das aktive Profil zuletzt lief
}

// Profile ist ein Thunderbird-Profil aus profiles.ini.
type Profile struct {
	Name    string `json:"name"`
	Path    string `json:"path"` // absoluter Pfad
	Active  bool   `json:"active"`
	Version string `json:"version"` // aus compatibility.ini, leer = unbekannt
}

// candidate ist ein möglicher Datenordner.
type candidate struct {
	variant, dir string
}

// Thunderbird findet alle Thunderbird-Installationen mit Profilen.
func Thunderbird() []App {
	home, _ := os.UserHomeDir()
	appData, _ := os.UserConfigDir() // Windows: %APPDATA%
	return findThunderbird(runtime.GOOS, home, appData)
}

func thunderbirdCandidates(goos, home, appData string) []candidate {
	switch goos {
	case "windows":
		return []candidate{{"windows", filepath.Join(appData, "Thunderbird")}}
	case "darwin":
		return []candidate{{"macos", filepath.Join(home, "Library", "Thunderbird")}}
	default:
		return []candidate{
			{"system", filepath.Join(home, ".thunderbird")},
			{"system", filepath.Join(home, ".mozilla-thunderbird")}, // ältere Debian-Fassungen
			{"snap", filepath.Join(home, "snap", "thunderbird", "common", ".thunderbird")},
			{"flatpak", filepath.Join(home, ".var", "app", "org.mozilla.Thunderbird", ".thunderbird")},
			{"flatpak", filepath.Join(home, ".var", "app", "net.thunderbird.Thunderbird", ".thunderbird")},
		}
	}
}

func findThunderbird(goos, home, appData string) []App {
	var out []App
	seen := map[string]bool{} // gleiche Variante zweimal (z. B. zwei Flatpak-Namen) → einmal
	for _, c := range thunderbirdCandidates(goos, home, appData) {
		profiles, err := readProfiles(c.dir)
		if err != nil || len(profiles) == 0 {
			continue
		}
		real, err := filepath.EvalSymlinks(c.dir)
		if err != nil || seen[real] {
			continue // z. B. ~/.mozilla-thunderbird als Verknüpfung auf ~/.thunderbird
		}
		seen[real] = true
		a := App{
			ID:       "thunderbird-" + c.variant,
			App:      "thunderbird",
			Variant:  c.variant,
			Name:     folderName(c.variant),
			Dir:      c.dir,
			Profiles: profiles,
		}
		for _, p := range profiles {
			if p.Active {
				a.Active, a.Version = p.Name, p.Version
			}
		}
		// Zwei Systempaket-Ordner (alt und neu) bekommen verschiedene IDs.
		if n := countVariant(out, c.variant); n > 0 {
			a.ID = a.ID + "-" + strings.TrimPrefix(filepath.Base(c.dir), ".")
			a.Name = a.Name + "-" + strings.TrimPrefix(filepath.Base(c.dir), ".")
		}
		out = append(out, a)
	}
	return out
}

func countVariant(apps []App, variant string) int {
	n := 0
	for _, a := range apps {
		if a.Variant == variant {
			n++
		}
	}
	return n
}

// folderName ist der Name beim Ziel – ohne Punkt, damit der Ordner im
// persönlichen Ordner sichtbar ist.
func folderName(variant string) string {
	switch variant {
	case "snap":
		return "Thunderbird-Snap"
	case "flatpak":
		return "Thunderbird-Flatpak"
	default:
		return "Thunderbird"
	}
}

// readProfiles liest profiles.ini (und installs.ini) in dir und liefert die
// Profile, deren Ordner es wirklich gibt. Aktiv ist:
//  1. das Profil, das ein [Install…]-Abschnitt als Default nennt,
//  2. sonst das mit Default=1,
//  3. sonst das einzige vorhandene.
func readProfiles(dir string) ([]Profile, error) {
	sections, err := readINI(filepath.Join(dir, "profiles.ini"))
	if err != nil {
		return nil, err
	}
	if extra, err := readINI(filepath.Join(dir, "installs.ini")); err == nil {
		sections = append(sections, extra...)
	}

	var profiles []Profile
	var defaultFlag string
	for _, s := range sections {
		if !strings.HasPrefix(s.name, "Profile") {
			continue
		}
		path := profilePath(dir, s.keys["Path"], s.keys["IsRelative"] != "0")
		if path == "" {
			continue
		}
		if fi, err := os.Stat(path); err != nil || !fi.IsDir() {
			continue // Eintrag ohne Ordner: nicht anbieten
		}
		name := s.keys["Name"]
		if name == "" {
			name = filepath.Base(path)
		}
		profiles = append(profiles, Profile{Name: name, Path: path, Version: profileVersion(path)})
		if s.keys["Default"] == "1" && defaultFlag == "" {
			defaultFlag = path
		}
	}

	active := ""
	for _, s := range sections {
		if strings.HasPrefix(s.name, "Install") && s.keys["Default"] != "" {
			p := profilePath(dir, s.keys["Default"], !filepath.IsAbs(s.keys["Default"]))
			if hasProfile(profiles, p) {
				active = p
				break
			}
		}
	}
	if active == "" {
		active = defaultFlag
	}
	if active == "" && len(profiles) == 1 {
		active = profiles[0].Path
	}
	for i := range profiles {
		profiles[i].Active = samePath(profiles[i].Path, active)
	}
	return profiles, nil
}

func profilePath(dir, p string, relative bool) string {
	if p == "" {
		return ""
	}
	p = filepath.FromSlash(strings.ReplaceAll(p, `\`, "/"))
	if relative {
		return filepath.Join(dir, p)
	}
	return filepath.Clean(p)
}

func hasProfile(profiles []Profile, path string) bool {
	for _, p := range profiles {
		if samePath(p.Path, path) {
			return true
		}
	}
	return false
}

func samePath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	if runtime.GOOS == "windows" {
		return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

type iniSection struct {
	name string
	keys map[string]string
}

// readINI liest eine einfache INI-Datei (Abschnitte, Schlüssel=Wert).
func readINI(path string) ([]iniSection, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []iniSection
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		// "\xef\xbb\xbf": Byte-Order-Mark, die manche Editoren an den Anfang setzen
		line := strings.TrimSpace(strings.TrimPrefix(sc.Text(), "\xef\xbb\xbf"))
		switch {
		case line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#"):
		case strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]"):
			out = append(out, iniSection{name: line[1 : len(line)-1], keys: map[string]string{}})
		default:
			if k, v, ok := strings.Cut(line, "="); ok && len(out) > 0 {
				out[len(out)-1].keys[strings.TrimSpace(k)] = strings.TrimSpace(v)
			}
		}
	}
	return out, sc.Err()
}
