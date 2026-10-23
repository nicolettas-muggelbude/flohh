// Package stick sorgt dafür, dass flohh den USB-Stick nicht blockiert.
//
// Läuft flohh von einem Wechseldatenträger, kopiert es sich in einen
// temporären Ordner, startet die Kopie und beendet sich selbst. Der Stick ist
// damit sofort frei und kann abgezogen und zum zweiten Rechner getragen werden.
package stick

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	envCopy = "FLOHH_TEMP_COPY" // gesetzt in der Kopie: ihr temporärer Ordner
	prefix  = "flohh-run-"
)

// Relaunch startet flohh aus einer temporären Kopie neu, wenn es von einem
// Wechseldatenträger läuft. Rückgabe true: Die Kopie läuft, der Aufrufer soll
// sich beenden. Bei einem Fehler läuft flohh einfach direkt vom Stick weiter.
func Relaunch(args []string) (bool, error) {
	if os.Getenv(envCopy) != "" {
		return false, nil // wir sind schon die Kopie
	}
	exe, err := os.Executable()
	if err != nil {
		return false, nil
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	if !onRemovable(exe) {
		return false, nil
	}

	dir, err := os.MkdirTemp("", prefix)
	if err != nil {
		return false, err
	}
	dst := filepath.Join(dir, filepath.Base(exe))
	if err := copyFile(exe, dst); err != nil {
		os.RemoveAll(dir)
		return false, err
	}
	cmd := exec.Command(dst, args...)
	cmd.Dir = dir // Arbeitsordner nicht auf dem Stick, sonst bleibt er belegt
	cmd.Env = append(os.Environ(), envCopy+"="+dir)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		os.RemoveAll(dir)
		return false, err
	}
	return true, nil
}

// Cleanup löscht Kopien früherer Läufe. Unter Windows lässt sich eine
// laufende Datei nicht löschen – die eigene Kopie verschwindet deshalb erst
// beim nächsten Start.
func Cleanup() {
	own := os.Getenv(envCopy)
	matches, _ := filepath.Glob(filepath.Join(os.TempDir(), prefix+"*"))
	for _, m := range matches {
		if m != own {
			os.RemoveAll(m) // klappt nicht, wenn diese Kopie gerade läuft – dann später
		}
	}
}

// Done räumt beim Beenden die eigene Kopie weg (Linux, macOS).
func Done() {
	if dir := os.Getenv(envCopy); dir != "" {
		os.RemoveAll(dir)
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
