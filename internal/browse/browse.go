// Package browse liefert die Ordnerauswahl der Oberfläche: Orte (Laufwerke,
// persönlicher Ordner) und die Unterordner eines Ordners. Nur für die
// eigene Oberfläche auf diesem Rechner – der andere Rechner sieht davon nichts.
package browse

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Dir ist ein Ordner in der Auswahl.
type Dir struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Hidden bool   `json:"hidden,omitempty"`
}

// Listing ist der Inhalt eines Ordners.
type Listing struct {
	Path   string `json:"path"`
	Parent string `json:"parent,omitempty"` // leer = oberste Ebene
	Dirs   []Dir  `json:"dirs"`
	Places []Dir  `json:"places"` // Sprungziele: persönlicher Ordner, Laufwerke
}

// List liefert die Unterordner von path; leer = persönlicher Ordner.
func List(path string) (Listing, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return Listing{}, err
		}
		path = home
	}
	if !filepath.IsAbs(path) {
		return Listing{}, errors.New("Pfad muss absolut sein")
	}
	path = filepath.Clean(path)
	entries, err := os.ReadDir(path)
	if err != nil {
		return Listing{}, err
	}
	l := Listing{Path: path, Dirs: []Dir{}, Places: places()}
	if parent := filepath.Dir(path); parent != path {
		l.Parent = parent
	}
	for _, e := range entries {
		full := filepath.Join(path, e.Name())
		if !isDir(e, full) {
			continue
		}
		l.Dirs = append(l.Dirs, Dir{Name: e.Name(), Path: full, Hidden: hidden(e, full)})
	}
	sort.Slice(l.Dirs, func(i, j int) bool {
		return strings.ToLower(l.Dirs[i].Name) < strings.ToLower(l.Dirs[j].Name)
	})
	return l, nil
}

// isDir: echte Ordner und Verknüpfungen auf Ordner (die Auswahl darf ihnen
// folgen; übertragen wird später trotzdem nur, was wirklich darin liegt).
func isDir(e os.DirEntry, full string) bool {
	if e.IsDir() {
		return true
	}
	if e.Type()&os.ModeSymlink != 0 {
		fi, err := os.Stat(full)
		return err == nil && fi.IsDir()
	}
	return false
}
