//go:build !windows

package stick

import "strings"

// onRemovable meldet, ob path auf einem eingehängten Wechseldatenträger
// liegt. Linux hängt USB-Sticks unter /media oder /run/media ein, macOS
// unter /Volumes.
func onRemovable(path string) bool {
	for _, p := range []string{"/media/", "/run/media/", "/Volumes/"} {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}
