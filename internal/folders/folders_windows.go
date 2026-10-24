package folders

import (
	"syscall"
	"unsafe"
)

// Known Folder IDs aus der Windows-API (KnownFolders.h).
var knownFolderIDs = map[Kind]syscall.GUID{
	Desktop:   {Data1: 0xB4BFCC3A, Data2: 0xDB2C, Data3: 0x424C, Data4: [8]byte{0xB0, 0x29, 0x7F, 0xE9, 0x9A, 0x87, 0xC6, 0x41}},
	Documents: {Data1: 0xFDD39AD0, Data2: 0x238F, Data3: 0x46AF, Data4: [8]byte{0xAD, 0xB4, 0x6C, 0x85, 0x48, 0x03, 0x69, 0xC7}},
	Downloads: {Data1: 0x374DE290, Data2: 0x123F, Data3: 0x4565, Data4: [8]byte{0x91, 0x64, 0x39, 0xC4, 0x92, 0x5E, 0x46, 0x7B}},
	Music:     {Data1: 0x4BD8D571, Data2: 0x6D19, Data3: 0x48D3, Data4: [8]byte{0xBE, 0x97, 0x42, 0x22, 0x20, 0x08, 0x0E, 0x43}},
	Pictures:  {Data1: 0x33E28130, Data2: 0x4E1E, Data3: 0x4676, Data4: [8]byte{0x83, 0x5A, 0x98, 0x39, 0x5C, 0x3B, 0xC3, 0xBB}},
	Videos:    {Data1: 0x18989B1D, Data2: 0x99B5, Data3: 0x455B, Data4: [8]byte{0x84, 0x1C, 0xAB, 0x7C, 0x74, 0xE4, 0xDD, 0xFC}},
}

var (
	shGetKnownFolderPath = syscall.NewLazyDLL("shell32.dll").NewProc("SHGetKnownFolderPath")
	coTaskMemFree        = syscall.NewLazyDLL("ole32.dll").NewProc("CoTaskMemFree")
)

// Path liefert den Ordner über SHGetKnownFolderPath. Das berücksichtigt auch
// verschobene Ordner, z. B. „Dokumente“ in OneDrive oder auf Laufwerk D:.
func Path(k Kind) string {
	id, ok := knownFolderIDs[k]
	if !ok {
		return ""
	}
	var p *uint16
	hr, _, _ := shGetKnownFolderPath.Call(uintptr(unsafe.Pointer(&id)), 0, 0, uintptr(unsafe.Pointer(&p)))
	if p != nil {
		defer coTaskMemFree.Call(uintptr(unsafe.Pointer(p)))
	}
	if hr != 0 || p == nil {
		return ""
	}
	n := 0
	for *(*uint16)(unsafe.Add(unsafe.Pointer(p), n*2)) != 0 {
		n++
	}
	return syscall.UTF16ToString(unsafe.Slice(p, n))
}
