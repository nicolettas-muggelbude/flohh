//go:build !windows

package transfer

// isLocked: Linux und macOS sperren Dateien beim Lesen nicht.
func isLocked(error) bool { return false }
