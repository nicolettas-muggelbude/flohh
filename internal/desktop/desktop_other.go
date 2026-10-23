//go:build !linux

package desktop

import "errors"

// State: Unter Windows und macOS gibt es nichts anzubieten – dort startet
// flohh per Doppelklick.
func State() Status { return Status{} }

// Install wird außerhalb von Linux nicht gebraucht.
func Install([]byte) error { return errors.New("nur unter Linux") }
