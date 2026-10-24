//go:build !windows

package folders

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Englische Standardnamen; macOS benutzt sie immer, Linux, wenn
// user-dirs.dirs fehlt.
var defaultNames = map[Kind]string{
	Desktop:   "Desktop",
	Documents: "Documents",
	Downloads: "Downloads",
	Music:     "Music",
	Pictures:  "Pictures",
	Videos:    "Videos",
}

// Schlüssel in ~/.config/user-dirs.dirs (freedesktop xdg-user-dirs).
var xdgKeys = map[Kind]string{
	Desktop:   "XDG_DESKTOP_DIR",
	Documents: "XDG_DOCUMENTS_DIR",
	Downloads: "XDG_DOWNLOAD_DIR",
	Music:     "XDG_MUSIC_DIR",
	Pictures:  "XDG_PICTURES_DIR",
	Videos:    "XDG_VIDEOS_DIR",
}

// Path liefert den Benutzerordner der Art k, "" wenn unbekannt.
func Path(k Kind) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	if runtime.GOOS == "darwin" {
		if k == Videos {
			return filepath.Join(home, "Movies")
		}
		return filepath.Join(home, defaultNames[k])
	}
	cfg := os.Getenv("XDG_CONFIG_HOME")
	if cfg == "" {
		cfg = filepath.Join(home, ".config")
	}
	if p, ok := readUserDirs(filepath.Join(cfg, "user-dirs.dirs"), home)[k]; ok {
		return p
	}
	return filepath.Join(home, defaultNames[k])
}

// readUserDirs liest Zeilen wie XDG_PICTURES_DIR="$HOME/Bilder". Ein Ordner,
// der auf $HOME selbst zeigt, gilt als „nicht eingerichtet“.
func readUserDirs(file, home string) map[Kind]string {
	out := map[Kind]string{}
	f, err := os.Open(file)
	if err != nil {
		return out
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		key, val, ok := strings.Cut(line, "=")
		if !ok || strings.HasPrefix(line, "#") {
			continue
		}
		val = strings.Trim(val, `"`)
		val = strings.Replace(val, "$HOME", home, 1)
		if !filepath.IsAbs(val) || filepath.Clean(val) == filepath.Clean(home) {
			continue
		}
		for k, xk := range xdgKeys {
			if xk == key {
				out[k] = filepath.Clean(val)
			}
		}
	}
	return out
}
