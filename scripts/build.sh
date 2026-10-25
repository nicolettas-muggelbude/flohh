#!/usr/bin/env bash
# Baut Flohh für alle Zielsysteme nach dist/.
# Aufruf aus dem Projektordner: ./scripts/build.sh
set -euo pipefail
cd "$(dirname "$0")/.."

# Version aus git: Tag (z. B. v0.1.0), sonst Commit-Kürzel; "-dirty" bei
# ungespeicherten Änderungen, "v0.1.0-3-gabc1234" für Stände nach einem Tag.
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo dev)

# Windows will als Dateiversion nur Zahlen: aus "v0.1.0…" wird "0.1.0".
FILEVER=0.0.0
if [[ $VERSION =~ ^v([0-9]+\.[0-9]+\.[0-9]+) ]]; then
  FILEVER=${BASH_REMATCH[1]}
fi

TARGETS=(
  windows/amd64
  windows/arm64
  linux/amd64
  linux/arm64
  darwin/amd64
  darwin/arm64
)

# Windows: Icon und Versionsinfo als Ressourcen-Datei (.syso). go build nimmt
# Dateien rsrc_windows_<arch>.syso im Paketordner automatisch mit. Sie werden
# nur für diesen Lauf erzeugt und danach wieder gelöscht.
trap 'rm -f cmd/flohh/rsrc_windows_*.syso' EXIT
go run github.com/tc-hib/go-winres@v0.3.3 simply \
  --arch amd64,arm64 --out cmd/flohh/rsrc \
  --icon assets/logo/flohh.ico --manifest cli \
  --product-name flohh --file-description "flohh – Daten springen von Rechner zu Rechner" \
  --original-filename flohh.exe --copyright "© 2026 Nicoletta · GPLv3" \
  --file-version "$FILEVER" --product-version "$VERSION"

mkdir -p dist
for target in "${TARGETS[@]}"; do
  os=${target%/*}
  arch=${target#*/}
  ext=""
  [[ $os == windows ]] && ext=".exe"
  out="dist/flohh-$os-$arch$ext"
  echo "→ $out"
  # CGO_ENABLED=0: rein in Go, keine C-Bibliotheken → statische Datei, die
  # auf jedem Linux läuft und sich ohne C-Compiler für Windows bauen lässt.
  # -trimpath: keine lokalen Pfade in der Datei. -s -w: ohne Debug-Infos (kleiner).
  CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -trimpath \
    -ldflags "-s -w -X main.version=$VERSION" \
    -o "$out" ./cmd/flohh
done

# USB-Stick-Ordner: verständliche Dateinamen plus Anleitung. Den Inhalt von
# dist/usb/ einfach auf einen (am besten exFAT-formatierten) Stick kopieren.
declare -A USB_NAMES=(
  [windows-amd64.exe]=Flohh-Windows.exe
  [windows-arm64.exe]=Flohh-Windows-ARM.exe
  [darwin-amd64]=Flohh-Mac-Intel
  [darwin-arm64]=Flohh-Mac-AppleSilicon
  [linux-amd64]=Flohh-Linux
  [linux-arm64]=Flohh-Linux-ARM
)
rm -rf dist/usb
mkdir -p dist/usb
for suffix in "${!USB_NAMES[@]}"; do
  cp "dist/flohh-$suffix" "dist/usb/${USB_NAMES[$suffix]}"
done
cp packaging/usb/LIESMICH.txt dist/usb/
# Lizenzen müssen mit den Programmen weitergegeben werden (GPL, BSD).
cp LICENSE dist/usb/LICENSE.txt
cp THIRD-PARTY-LICENSES.txt dist/usb/

# Zum Herunterladen: ein ZIP mit allem. Entpackt ergibt es einen Ordner
# „flohh-<version>“ (auch direkt auf einen USB-Stick).
ZIP="dist/flohh-$VERSION.zip"
rm -f "$ZIP"
rm -rf "dist/flohh-$VERSION"
cp -r dist/usb "dist/flohh-$VERSION"
(cd dist && zip -q -X -r "$(basename "$ZIP")" "flohh-$VERSION")
rm -rf "dist/flohh-$VERSION"
echo "→ $ZIP"

ls -lh dist dist/usb
