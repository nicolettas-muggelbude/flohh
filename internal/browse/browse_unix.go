//go:build !windows

package browse

import (
	"os"
	"runtime"
	"strings"
)

// places: persönlicher Ordner, das Wurzelverzeichnis und eingehängte
// Laufwerke.
func places() []Dir {
	var out []Dir
	if home, err := os.UserHomeDir(); err == nil {
		out = append(out, Dir{Name: "home", Path: home})
	}
	out = append(out, Dir{Name: "/", Path: "/"})
	mounts := "/media/" + os.Getenv("USER")
	if runtime.GOOS == "darwin" {
		mounts = "/Volumes"
	}
	if entries, err := os.ReadDir(mounts); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				out = append(out, Dir{Name: e.Name(), Path: mounts + "/" + e.Name()})
			}
		}
	}
	return out
}

func hidden(e os.DirEntry, _ string) bool { return strings.HasPrefix(e.Name(), ".") }
