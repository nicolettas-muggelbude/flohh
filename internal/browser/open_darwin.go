package browser

import "os/exec"

// open nutzt den macOS-Befehl "open", der Links an den Standardbrowser gibt.
func open(url string) error {
	return exec.Command("open", url).Start()
}
