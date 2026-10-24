package web

import (
	"regexp"
	"testing"
)

// TestUniqueIDs: Jede id darf in index.html nur einmal vorkommen – sonst
// findet getElementById das falsche Element (schon einmal passiert).
func TestUniqueIDs(t *testing.T) {
	html, err := Files.ReadFile("index.html")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, m := range regexp.MustCompile(`\sid="([^"]+)"`).FindAllSubmatch(html, -1) {
		id := string(m[1])
		if seen[id] {
			t.Errorf("id %q kommt mehrfach vor", id)
		}
		seen[id] = true
	}
}
