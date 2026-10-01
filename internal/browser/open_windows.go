package browser

import "os/exec"

// open nutzt den Windows-Mechanismus für "Datei/Link öffnen". Anders als
// "cmd /c start" kommt er mit & und anderen Sonderzeichen in der URL klar.
func open(url string) error {
	return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}
