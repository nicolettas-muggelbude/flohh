package browse

import (
	"os"
	"strings"
	"syscall"
)

// places: persönlicher Ordner und alle vorhandenen Laufwerke (C:\, D:\ …).
func places() []Dir {
	var out []Dir
	if home, err := os.UserHomeDir(); err == nil {
		out = append(out, Dir{Name: "home", Path: home})
	}
	for c := 'A'; c <= 'Z'; c++ {
		root := string(c) + `:\`
		if fi, err := os.Stat(root); err == nil && fi.IsDir() {
			out = append(out, Dir{Name: string(c) + ":", Path: root})
		}
	}
	return out
}

// hidden: Windows-Attribut „versteckt“ (z. B. AppData) oder Punkt am Anfang.
func hidden(e os.DirEntry, _ string) bool {
	if strings.HasPrefix(e.Name(), ".") {
		return true
	}
	fi, err := e.Info()
	if err != nil {
		return false
	}
	a, ok := fi.Sys().(*syscall.Win32FileAttributeData)
	return ok && a.FileAttributes&syscall.FILE_ATTRIBUTE_HIDDEN != 0
}
