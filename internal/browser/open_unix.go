//go:build !windows

package browser

import (
	"errors"
	"os/exec"
)

// Kandidaten in dieser Reihenfolge: xdg-open gibt es auf jedem Linux-Desktop,
// wslview und explorer.exe helfen beim Testen unter WSL.
var openers = []string{"xdg-open", "wslview", "explorer.exe"}

func open(url string) error {
	for _, name := range openers {
		path, err := exec.LookPath(name)
		if err != nil {
			continue // Programm nicht vorhanden, nächstes probieren
		}
		return exec.Command(path, url).Start()
	}
	return errors.New("kein Programm zum Öffnen gefunden (xdg-open fehlt)")
}
