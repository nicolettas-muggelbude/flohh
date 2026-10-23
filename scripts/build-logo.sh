#!/usr/bin/env bash
# Erzeugt aus den SVG-Quellen in assets/logo/ die Wortmarken (Schrift als
# Pfade), PNGs und das Windows-Icon. Nur nötig, wenn sich das Logo ändert.
#
# Braucht: inkscape, rsvg-convert (librsvg2-bin), convert (imagemagick),
# python3, curl. Die Schrift Fredoka (SIL Open Font License) wird nur für
# diesen Lauf heruntergeladen und nicht im System installiert.
#
# Aufruf aus dem Projektordner: ./scripts/build-logo.sh
set -euo pipefail
cd "$(dirname "$0")/.."
OUT="$PWD/assets/logo"
W=$(mktemp -d)
trap 'rm -rf "$W"' EXIT
mkdir -p "$W/fonts"

curl -fsSL -o "$W/fonts/Fredoka.ttf" \
  "https://github.com/google/fonts/raw/main/ofl/fredoka/Fredoka%5Bwdth%2Cwght%5D.ttf"
cat > "$W/fonts.conf" <<EOF
<?xml version="1.0"?>
<!DOCTYPE fontconfig SYSTEM "fonts.dtd">
<fontconfig>
  <include ignore_missing="yes">/etc/fonts/fonts.conf</include>
  <dir>$W/fonts</dir>
  <cachedir>$W/cache</cachedir>
</fontconfig>
EOF
export FONTCONFIG_FILE="$W/fonts.conf"
fc-cache -f "$W/fonts" >/dev/null

# Wortmarke = Bildzeichen + Text „flohh“, Grundlinie auf Höhe der Füße
python3 - "$OUT" "$W" <<'PY'
import re, sys
out, w = sys.argv[1], sys.argv[2]
def inner(name):
    s = open(f"{out}/{name}", encoding="utf-8").read()
    s = re.sub(r"^.*?<svg[^>]*>", "", s, flags=re.S)
    s = re.sub(r"<title>.*?</title>", "", s)
    return s.replace("</svg>", "")
for src, dst, color in [("flohh-mark.svg", "logo.svg", "#1F1D1A"),
                        ("flohh-mark-dark.svg", "logo-dark.svg", "#F6F3EC")]:
    open(f"{w}/{dst}", "w", encoding="utf-8").write(
        '<svg xmlns="http://www.w3.org/2000/svg" width="1040" height="372" viewBox="36 14 520 186">'
        '<title>flohh</title>' + inner(src) +
        f'<text x="262" y="188" font-family="Fredoka" font-weight="600" font-size="97" '
        f'letter-spacing="-2" fill="{color}">flohh</text></svg>')
PY

for n in logo logo-dark; do
  inkscape "$W/$n.svg" --export-text-to-path --export-plain-svg \
    --export-area-drawing --export-margin=4 --export-filename="$OUT/flohh-$n.svg" 2>/dev/null
  echo "→ assets/logo/flohh-$n.svg"
done

mkdir -p "$OUT/png"
png() { rsvg-convert -w "$2" "$OUT/$1.svg" -o "$OUT/png/$1-$2.png"; echo "→ assets/logo/png/$1-$2.png"; }
png flohh-icon 512
png flohh-icon 256
png flohh-icon 128
png flohh-mark 512
png flohh-logo 1200
png flohh-logo-dark 1200

# Windows-Icon: kleine Stufen vereinfacht, große mit Beinen und Haaren
for s in 16 24 32; do rsvg-convert -w $s "$OUT/flohh-icon-small.svg" -o "$W/i$s.png"; done
for s in 48 64 128 256; do rsvg-convert -w $s "$OUT/flohh-icon.svg" -o "$W/i$s.png"; done
convert "$W"/i{16,24,32,48,64,128,256}.png "$OUT/flohh.ico"
echo "→ assets/logo/flohh.ico"

# Kopien für die Oberfläche (web/ wird per go:embed in die App gepackt)
cp "$OUT/flohh-logo.svg" "$OUT/flohh-logo-dark.svg" web/img/
cp "$OUT/flohh-icon-small.svg" web/img/favicon.svg
cp "$OUT/flohh-icon.svg" web/img/icon.svg   # Icon für den Linux-Menü-Eintrag
echo "→ web/img/"
