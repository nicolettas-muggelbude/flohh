package transfer

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Eine Datei, die ein anderes Programm (wie Edge) ohne Mitbenutzung geöffnet
// hat, wird als „gesperrt“ erkannt – samt Programmname. Hier sperrt
// PowerShell die Datei. (Den eigenen Prozess meldet Windows immer als
// „kritisch“, darum ein zweites Programm.)
func TestLockedByRestartManager(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gesperrt.db")
	os.WriteFile(path, []byte("x"), 0o644)
	ps := exec.Command("powershell", "-NoProfile", "-Command",
		"$f=[IO.File]::Open('"+path+"','Open','Read','None'); Write-Output bereit; Start-Sleep 20")
	out, _ := ps.StdoutPipe()
	if err := ps.Start(); err != nil {
		t.Skip("PowerShell nicht verfügbar")
	}
	defer ps.Process.Kill()
	buf := make([]byte, 64)
	out.Read(buf) // wartet auf „bereit“
	time.Sleep(200 * time.Millisecond)

	_, err := os.Open(path)
	if code := readErrCode(err); code != errLocked {
		t.Fatalf("Code %q (Fehler %v), erwartet %q", code, err, errLocked)
	}
	names := lockedBy(path)
	t.Logf("gesperrt von: %q", names)
	if len(names) == 0 || names[0] == ByWindows || !strings.Contains(strings.ToLower(names[0]), "powershell") {
		t.Errorf("erwartet PowerShell, bekommen %q", names)
	}
	if r := readError(err, path); r.Error != errLocked || r.By != names[0] {
		t.Errorf("Antwort: %+v", r)
	}
}
