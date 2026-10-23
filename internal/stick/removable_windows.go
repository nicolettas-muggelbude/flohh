package stick

import (
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

const driveRemovable = 2 // DRIVE_REMOVABLE aus der Windows-API

var getDriveType = syscall.NewLazyDLL("kernel32.dll").NewProc("GetDriveTypeW")

// onRemovable meldet, ob path auf einem Wechseldatenträger (USB-Stick,
// Speicherkarte) liegt. USB-Festplatten meldet Windows als feste Laufwerke.
func onRemovable(path string) bool {
	vol := filepath.VolumeName(path) // z. B. "E:"
	if vol == "" || strings.HasPrefix(vol, `\\`) {
		return false // Netzwerkpfad
	}
	root, err := syscall.UTF16PtrFromString(vol + `\`)
	if err != nil {
		return false
	}
	r, _, _ := getDriveType.Call(uintptr(unsafe.Pointer(root)))
	return r == driveRemovable
}
