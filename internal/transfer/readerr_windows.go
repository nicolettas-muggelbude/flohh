package transfer

import (
	"errors"
	"syscall"
)

const (
	errorSharingViolation syscall.Errno = 32 // ERROR_SHARING_VIOLATION
	errorLockViolation    syscall.Errno = 33 // ERROR_LOCK_VIOLATION
)

// isLocked: Die Datei ist von einem anderen Programm geöffnet bzw. gesperrt.
func isLocked(err error) bool {
	return errors.Is(err, errorSharingViolation) || errors.Is(err, errorLockViolation)
}
