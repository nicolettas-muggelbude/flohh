//go:build !windows

package transfer

import "io/fs"

// cloudOnly: Nur-Cloud-Dateien gibt es so nur unter Windows (OneDrive).
func cloudOnly(fs.FileInfo) bool { return false }
