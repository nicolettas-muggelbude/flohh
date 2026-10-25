package transfer

import (
	"io/fs"
	"syscall"
)

const (
	attrOffline            = 0x00001000 // FILE_ATTRIBUTE_OFFLINE
	attrRecallOnOpen       = 0x00040000 // FILE_ATTRIBUTE_RECALL_ON_OPEN
	attrRecallOnDataAccess = 0x00400000 // FILE_ATTRIBUTE_RECALL_ON_DATA_ACCESS
)

// cloudOnly erkennt Dateien, die nur in der Cloud liegen (OneDrive
// „Nur online verfügbar“, auch andere Cloud-Programme mit Platzhaltern).
// Sie zu lesen würde sie erst herunterladen – das wäre langsam und braucht
// Internet. Gezählt und angezeigt werden sie trotzdem (Stats.Cloud).
func cloudOnly(fi fs.FileInfo) bool {
	a, ok := fi.Sys().(*syscall.Win32FileAttributeData)
	return ok && a.FileAttributes&(attrOffline|attrRecallOnOpen|attrRecallOnDataAccess) != 0
}
