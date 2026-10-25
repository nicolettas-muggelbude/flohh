package transfer

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// Eine Datei, die (wie bei Edge) ohne Mitbenutzung geöffnet ist, wird als
// „gesperrt“ erkannt – samt dem Programm, das sie hält (hier: dieser Test).
func TestLockedByRestartManager(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gesperrt.db")
	os.WriteFile(path, []byte("x"), 0o644)
	p, _ := syscall.UTF16PtrFromString(path)
	h, err := syscall.CreateFile(p, syscall.GENERIC_READ, 0 /* keine Mitbenutzung */, nil,
		syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer syscall.CloseHandle(h)

	_, err = os.Open(path)
	if code := readErrCode(err); code != errLocked {
		t.Fatalf("Code %q (Fehler %v), erwartet %q", code, err, errLocked)
	}
	names := lockedBy(path)
	t.Logf("gesperrt von: %q", names)
	if len(names) == 0 {
		t.Error("Restart Manager nennt kein Programm")
	}
	if r := readError(err, path); r.Error != errLocked || r.By == "" {
		t.Errorf("Antwort: %+v", r)
	}
}
