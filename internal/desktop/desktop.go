// Package desktop nimmt flohh unter Linux ins Anwendungsmenü auf.
//
// Hintergrund: Die Dateiverwaltung von Ubuntu (GNOME) startet Programme nicht
// per Doppelklick. Mit einem Eintrag im Anwendungsmenü startet man flohh
// dagegen ganz normal mit einem Klick – samt Floh-Icon.
package desktop

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Status sagt der Oberfläche, ob sie die Aufnahme ins Menü anbieten soll.
type Status struct {
	Offer     bool `json:"offer"`     // Hinweis anzeigen
	Installed bool `json:"installed"` // es gibt schon einen Menü-Eintrag
	Outdated  bool `json:"outdated"`  // … aber mit einer anderen Fassung
}

type settings struct {
	MenuDeclined bool `json:"menuDeclined"`
}

func settingsPath() (string, error) {
	dir, err := os.UserConfigDir() // ~/.config
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "flohh", "settings.json"), nil
}

func loadSettings() settings {
	var s settings
	if p, err := settingsPath(); err == nil {
		if data, err := os.ReadFile(p); err == nil {
			json.Unmarshal(data, &s)
		}
	}
	return s
}

func saveSettings(s settings) error {
	p, err := settingsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

// Decline merkt sich, dass der Hinweis nicht mehr kommen soll.
func Decline() error {
	s := loadSettings()
	s.MenuDeclined = true
	return saveSettings(s)
}
