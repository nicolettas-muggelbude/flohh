// Package web enthält die Dateien der Oberfläche. Die //go:embed-Zeile packt
// sie beim Bauen in die ausführbare Datei – zur Laufzeit wird nichts von der
// Festplatte gelesen.
package web

import "embed"

//go:embed index.html style.css i18n.js app.js img fonts
var Files embed.FS
