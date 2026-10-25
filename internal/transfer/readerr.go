package transfer

import (
	"errors"
	"io/fs"
)

// Fehlercodes beim Lesen auf dem alten Rechner (für die Oberfläche).
const (
	errLocked       = "locked"        // von einem Programm geöffnet (Windows)
	errAccessDenied = "access_denied" // keine Berechtigung
	errReadFailed   = "read_failed"   // sonstiger Lesefehler
)

// readErrCode ordnet einen Lesefehler einem Code zu. Gesperrte Dateien sind
// der häufigste Fall in AppData: Firefox, Thunderbird, Outlook halten ihre
// Datenbanken offen, solange sie laufen.
func readErrCode(err error) string {
	switch {
	case isLocked(err):
		return errLocked
	case errors.Is(err, fs.ErrPermission):
		return errAccessDenied
	default:
		return errReadFailed
	}
}
