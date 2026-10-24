// Package folders kennt die Benutzerordner (Dokumente, Bilder …) auf jedem
// System. Übertragen wird immer die Art des Ordners („pictures“), nie sein
// Pfad – so landen Bilder vom deutschen Windows im „Pictures“-Ordner eines
// englischen Linux und umgekehrt.
package folders

import "os"

// Kind ist die Art eines Benutzerordners, gleich auf allen Systemen.
type Kind string

const (
	Documents Kind = "documents"
	Pictures  Kind = "pictures"
	Music     Kind = "music"
	Videos    Kind = "videos"
	Desktop   Kind = "desktop"
	Downloads Kind = "downloads"
)

// All in der Reihenfolge, in der die Oberfläche sie zeigt.
var All = []Kind{Documents, Pictures, Music, Videos, Desktop, Downloads}

// Valid meldet, ob k eine bekannte Art ist.
func Valid(k Kind) bool {
	for _, a := range All {
		if a == k {
			return true
		}
	}
	return false
}

// Folder ist ein vorhandener Benutzerordner.
type Folder struct {
	Kind Kind   `json:"kind"`
	Path string `json:"path"`
}

// Known liefert die Benutzerordner, die es auf diesem Rechner gibt.
func Known() []Folder {
	var out []Folder
	for _, k := range All {
		if p := Path(k); p != "" {
			if fi, err := os.Stat(p); err == nil && fi.IsDir() {
				out = append(out, Folder{Kind: k, Path: p})
			}
		}
	}
	return out
}
