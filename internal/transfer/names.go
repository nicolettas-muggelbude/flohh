package transfer

import (
	"errors"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/text/unicode/norm"
)

// cleanRel prüft einen relativen Pfad vom anderen Rechner und liefert ihn
// bereinigt. Absolute Pfade und „..“ werden abgelehnt – niemand soll aus
// einem freigegebenen Ordner herausgreifen oder außerhalb des Zielordners
// schreiben können.
func cleanRel(p string) (string, error) {
	if p == "" || strings.ContainsRune(p, 0) || strings.HasPrefix(p, "/") || strings.Contains(p, `\`) {
		return "", errors.New("ungültiger Pfad")
	}
	c := path.Clean(p)
	if c == "." || c == ".." || strings.HasPrefix(c, "../") {
		return "", errors.New("ungültiger Pfad")
	}
	return c, nil
}

// targetRel macht aus einem Pfad des Senders einen Pfad für dieses System:
// Umlaute einheitlich (NFC – macOS liefert sie oft zerlegt) und unter
// Windows unerlaubte Zeichen und Namen ersetzt.
func targetRel(p string, windows bool) string {
	parts := strings.Split(p, "/")
	for i, part := range parts {
		part = norm.NFC.String(part)
		if windows {
			part = windowsName(part)
		}
		parts[i] = part
	}
	return filepath.Join(parts...)
}

// SafeName macht aus einem Ordnernamen des anderen Rechners einen Namen, den
// dieses System anlegen kann (z. B. für das vorgeschlagene Ziel).
func SafeName(name string) string {
	name = norm.NFC.String(strings.Trim(name, `/\`))
	if runtime.GOOS == "windows" {
		return windowsName(name)
	}
	if name == "" || name == "." || name == ".." {
		return "_"
	}
	return strings.ReplaceAll(name, "/", "_")
}

var reservedWindows = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true,
	"COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true,
	"LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

// windowsName ersetzt, was Windows in Dateinamen nicht erlaubt.
func windowsName(s string) string {
	s = strings.Map(func(r rune) rune {
		if r < 32 || strings.ContainsRune(`<>:"/\|?*`, r) {
			return '_'
		}
		return r
	}, s)
	s = strings.TrimRight(s, ". ") // Windows verwirft Punkte und Leerzeichen am Ende
	if s == "" {
		return "_"
	}
	base, _, _ := strings.Cut(s, ".")
	if reservedWindows[strings.ToUpper(base)] {
		s = "_" + s
	}
	return s
}

// skipName sind Dateien, die nur das jeweilige System für sich anlegt.
func skipName(name string) bool {
	switch strings.ToLower(name) {
	case "desktop.ini", "thumbs.db", ".ds_store":
		return true
	}
	return false
}
