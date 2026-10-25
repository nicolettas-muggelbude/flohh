package transfer

import "strings"

// Windows-interne Bereiche, die beim Umzug nichts nützen: temporäre
// Dateien, Zwischenspeicher, Vorschaubilder, Registry-Teile, Listen
// „zuletzt verwendet“. Viele davon hält Windows ständig gesperrt. Sie
// werden übersprungen und als Systemdateien gezählt.
//
// Pfade in Kleinbuchstaben mit „/“; gesucht wird jeweils ab „/appdata/“, so
// greift der Filter egal, ob AppData selbst oder der ganze Benutzerordner
// freigegeben ist.
var windowsInternalDirs = []string{
	"/appdata/local/temp/",
	"/appdata/local/microsoft/windows/", // UsrClass.dat, WebCache, Explorer-Cache …
	"/appdata/local/microsoft/windowsapps/",
	"/appdata/local/connecteddevicesplatform/",
	"/appdata/local/d3dscache/",
	"/appdata/local/crashdumps/",
	"/appdata/local/microsoft/internet explorer/",
	"/appdata/locallow/microsoft/cryptneturlcache/",
	"/appdata/roaming/microsoft/windows/recent/",
}

// Ausnahmen innerhalb der Bereiche oben, die mitkommen sollen.
var windowsInternalKeep = []string{
	"/appdata/local/microsoft/windows/fonts/", // selbst installierte Schriften
}

// In Store-Apps (AppData\Local\Packages\<App>\…) nur die Zwischenspeicher.
var packageCacheDirs = []string{"/ac/", "/tempstate/", "/localcache/"}

// skipInternal: path liegt in einem Windows-internen Bereich unterhalb des
// freigegebenen Ordners root. Liegt root selbst schon in so einem Bereich,
// hat jemand ihn bewusst gewählt – dann wird nichts gefiltert.
func skipInternal(root, path string) bool {
	return !windowsInternal(root+"/") && windowsInternal(path)
}

// windowsInternal meldet, ob path in einem Windows-internen Bereich liegt.
func windowsInternal(path string) bool {
	p := strings.ToLower(strings.ReplaceAll(path, `\`, "/"))
	base := p[strings.LastIndex(p, "/")+1:]
	if strings.HasPrefix(base, "ntuser.") { // Registry des Benutzers
		return true
	}
	i := strings.Index(p, "/appdata/")
	if i < 0 {
		return false
	}
	p = p[i:]
	for _, k := range windowsInternalKeep {
		if strings.HasPrefix(p, k) {
			return false
		}
	}
	for _, d := range windowsInternalDirs {
		if strings.HasPrefix(p, d) {
			return true
		}
	}
	if rest, ok := strings.CutPrefix(p, "/appdata/local/packages/"); ok {
		if j := strings.Index(rest, "/"); j >= 0 {
			for _, c := range packageCacheDirs {
				if strings.HasPrefix(rest[j:], c) {
					return true
				}
			}
		}
	}
	return false
}
