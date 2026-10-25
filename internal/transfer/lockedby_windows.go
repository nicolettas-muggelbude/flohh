package transfer

import (
	"syscall"
	"unsafe"
)

// Restart Manager (rstrtmgr.dll): Dieselbe Schnittstelle, mit der
// Installationsprogramme melden „Diese Programme müssen geschlossen werden“.
// Braucht keine Administratorrechte.
var (
	rstrtmgr            = syscall.NewLazyDLL("rstrtmgr.dll")
	rmStartSession      = rstrtmgr.NewProc("RmStartSession")
	rmRegisterResources = rstrtmgr.NewProc("RmRegisterResources")
	rmGetList           = rstrtmgr.NewProc("RmGetList")
	rmEndSession        = rstrtmgr.NewProc("RmEndSession")
)

const errorMoreData = 234 // ERROR_MORE_DATA

// rmProcessInfo entspricht RM_PROCESS_INFO aus RestartManager.h.
type rmProcessInfo struct {
	PID              uint32
	StartTime        syscall.Filetime
	AppName          [256]uint16 // CCH_RM_MAX_APP_NAME + 1
	ServiceShortName [64]uint16  // CCH_RM_MAX_SVC_NAME + 1
	AppType          uint32
	AppStatus        uint32
	TSSessionID      uint32
	Restartable      int32
}

// lockedBy nennt die Programme, die path geöffnet haben (Anzeigenamen wie
// „Microsoft Edge“). Leer, wenn Windows es nicht sagen kann.
func lockedBy(path string) []string {
	if rstrtmgr.Load() != nil {
		return nil
	}
	var session uint32
	var key [33]uint16 // CCH_RM_SESSION_KEY + 1
	if r, _, _ := rmStartSession.Call(uintptr(unsafe.Pointer(&session)), 0, uintptr(unsafe.Pointer(&key[0]))); r != 0 {
		return nil
	}
	defer rmEndSession.Call(uintptr(session))

	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil
	}
	files := []*uint16{p}
	if r, _, _ := rmRegisterResources.Call(uintptr(session), 1, uintptr(unsafe.Pointer(&files[0])), 0, 0, 0, 0); r != 0 {
		return nil
	}

	infos := make([]rmProcessInfo, 8)
	for {
		needed, n, reasons := uint32(0), uint32(len(infos)), uint32(0)
		r, _, _ := rmGetList.Call(uintptr(session), uintptr(unsafe.Pointer(&needed)), uintptr(unsafe.Pointer(&n)),
			uintptr(unsafe.Pointer(&infos[0])), uintptr(unsafe.Pointer(&reasons)))
		if r == errorMoreData && int(needed) > len(infos) {
			infos = make([]rmProcessInfo, needed)
			continue
		}
		if r != 0 {
			return nil
		}
		var names []string
		seen := map[string]bool{}
		for _, info := range infos[:n] {
			name := syscall.UTF16ToString(info.AppName[:])
			if name == "" {
				name = syscall.UTF16ToString(info.ServiceShortName[:])
			}
			if name != "" && !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
		return names
	}
}
