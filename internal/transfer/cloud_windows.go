package transfer

import (
	"io/fs"
	"syscall"
)

const (
	attrRecallOnOpen       = 0x00040000 // FILE_ATTRIBUTE_RECALL_ON_OPEN
	attrRecallOnDataAccess = 0x00400000 // FILE_ATTRIBUTE_RECALL_ON_DATA_ACCESS
)

// cloudOnly erkennt Dateien, die nur in der Cloud liegen (OneDrive
// „Nur online verfügbar“). Sie zu lesen würde sie erst herunterladen –
// das wäre langsam und braucht Internet.
// TODO: Anzahl übersprungener Cloud-Dateien in der Oberfläche anzeigen.
func cloudOnly(fi fs.FileInfo) bool {
	a, ok := fi.Sys().(*syscall.Win32FileAttributeData)
	return ok && a.FileAttributes&(attrRecallOnOpen|attrRecallOnDataAccess) != 0
}
