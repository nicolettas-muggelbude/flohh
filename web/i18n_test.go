package web

import (
	"regexp"
	"testing"
)

// TestI18nKeys prüft, dass jeder Text-Schlüssel, den HTML oder JavaScript
// benutzen, im deutschen Wörterbuch in i18n.js steht. Andere Sprachen dürfen
// lückenhaft sein – dort springt Deutsch ein.
func TestI18nKeys(t *testing.T) {
	read := func(name string) string {
		b, err := Files.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}

	// Schlüssel im Wörterbuch: Zeilen der Form   "bereich.name": ...
	defined := map[string]bool{}
	for _, m := range regexp.MustCompile(`(?m)^\s*"([\w.]+)":`).FindAllStringSubmatch(read("i18n.js"), -1) {
		defined[m[1]] = true
	}
	if len(defined) == 0 {
		t.Fatal("keine Schlüssel in i18n.js gefunden – Format geändert?")
	}

	used := regexp.MustCompile(`data-i18n(?:-[\w-]+)?="([\w.]+)"|\bt\("([\w.]+)"`)
	for _, file := range []string{"index.html", "app.js"} {
		for _, m := range used.FindAllStringSubmatch(read(file), -1) {
			key := m[1] + m[2]
			if !defined[key] {
				t.Errorf("%s: Schlüssel %q fehlt in i18n.js", file, key)
			}
		}
	}
}
