// Package browser öffnet eine Adresse im Standardbrowser.
//
// Die eigentliche Arbeit steckt in open_windows.go, open_darwin.go (macOS) und
// open_unix.go (Linux und andere). Go wählt
// beim Bauen anhand des Dateinamens bzw. der //go:build-Zeile automatisch die
// passende Datei für das Zielsystem aus.
package browser

// Open öffnet url im Standardbrowser und wartet nicht, bis er gestartet ist.
func Open(url string) error {
	return open(url)
}
