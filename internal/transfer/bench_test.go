package transfer

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// BenchmarkSmallFiles misst viele kleine Dateien – einmal ohne, einmal mit
// 2 ms Verzögerung beim Öffnen (so ähnlich bremst ein Virenscanner unter
// Windows). Aufruf: go test -run x -bench SmallFiles ./internal/transfer
func BenchmarkSmallFiles(b *testing.B) {
	src := b.TempDir()
	data := make([]byte, 8<<10)
	rand.Read(data)
	const n = 3000
	for i := 0; i < n; i++ {
		p := filepath.Join(src, fmt.Sprintf("ordner%02d", i%30), fmt.Sprintf("datei%04d.txt", i))
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, data, 0o644)
	}

	for _, delay := range []time.Duration{0, 2 * time.Millisecond} {
		b.Run(fmt.Sprintf("Öffnen+%v", delay), func(b *testing.B) {
			openFile = func(name string) (*os.File, error) {
				time.Sleep(delay)
				return os.Open(name)
			}
			defer func() { openFile = os.Open }()
			for i := 0; i < b.N; i++ {
				dst := b.TempDir()
				s := NewServer()
				s.Share([]Share{{ID: "documents", Path: src}})
				e := NewEngine(connect(b, s))
				start := time.Now()
				if err := e.Run(context.Background(), []Plan{{Root: "documents", Target: dst}}); err != nil {
					b.Fatal(err)
				}
				el := time.Since(start)
				b.ReportMetric(float64(n)/el.Seconds(), "Dateien/s")
				if p := e.Progress(); p.FilesDone != n {
					b.Fatalf("nur %d von %d kopiert", p.FilesDone, n)
				}
			}
		})
	}
}
