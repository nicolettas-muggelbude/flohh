#!/usr/bin/env bash
# Baut Umzugskarton für alle Zielsysteme nach dist/.
# Aufruf aus dem Projektordner: ./scripts/build.sh
set -euo pipefail
cd "$(dirname "$0")/.."

# Version aus git: Tag, sonst Commit-Kürzel; "-dirty" bei ungespeicherten Änderungen
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo dev)

TARGETS=(
  windows/amd64
  windows/arm64
  linux/amd64
  linux/arm64
)

mkdir -p dist
for target in "${TARGETS[@]}"; do
  os=${target%/*}
  arch=${target#*/}
  ext=""
  [[ $os == windows ]] && ext=".exe"
  out="dist/umzugskarton-$os-$arch$ext"
  echo "→ $out"
  # CGO_ENABLED=0: rein in Go, keine C-Bibliotheken → statische Datei, die
  # auf jedem Linux läuft und sich ohne C-Compiler für Windows bauen lässt.
  # -trimpath: keine lokalen Pfade in der Datei. -s -w: ohne Debug-Infos (kleiner).
  CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -trimpath \
    -ldflags "-s -w -X main.version=$VERSION" \
    -o "$out" ./cmd/umzugskarton
done

ls -lh dist
