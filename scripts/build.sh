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
  darwin/amd64
  darwin/arm64
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

# USB-Stick-Ordner: verständliche Dateinamen plus Anleitung. Den Inhalt von
# dist/usb/ einfach auf einen (am besten exFAT-formatierten) Stick kopieren.
declare -A USB_NAMES=(
  [windows-amd64.exe]=Umzugskarton-Windows.exe
  [windows-arm64.exe]=Umzugskarton-Windows-ARM.exe
  [darwin-amd64]=Umzugskarton-Mac-Intel
  [darwin-arm64]=Umzugskarton-Mac-AppleSilicon
  [linux-amd64]=Umzugskarton-Linux
  [linux-arm64]=Umzugskarton-Linux-ARM
)
rm -rf dist/usb
mkdir -p dist/usb
for suffix in "${!USB_NAMES[@]}"; do
  cp "dist/umzugskarton-$suffix" "dist/usb/${USB_NAMES[$suffix]}"
done
cp packaging/usb/LIESMICH.txt dist/usb/

# Zwei Varianten zum Herunterladen:
#  - ZIP: auf einen beliebigen Stick entpacken
#  - IMG: mit Rufus/Etcher/GNOME "Laufwerke" auf einen Stick schreiben
ZIP="dist/umzugskarton-$VERSION-usb.zip"
IMG="dist/umzugskarton-$VERSION-usb.img"
rm -f "$ZIP"
(cd dist/usb && zip -q -X -r "../$(basename "$ZIP")" .)
echo "→ $ZIP"
(cd tools && go run ./mkusbimg -in ../dist/usb -out "../$IMG")

ls -lh dist dist/usb
