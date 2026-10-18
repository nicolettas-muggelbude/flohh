<h1>
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/logo/flohh-logo-dark.svg">
    <img src="assets/logo/flohh-logo.svg" alt="flohh" height="96">
  </picture>
</h1>

*Daten springen von Rechner zu Rechner.*

Portable App, die beim PC-Umzug Bibliotheken und Ordner von einem Rechner auf
einen anderen kopiert – Windows, Linux und macOS in allen Richtungen. Heute per
LAN-Kabel, später auch über WLAN und Thunderbolt/USB4.

**Status:** Meilenstein 2 läuft. Zwei Rechner finden sich im Netz (auch per
direktem Kabel) und koppeln sich verschlüsselt per 6-stelligem Code. Die
Übertragung der Daten folgt.

## Bauen

Voraussetzung: Go 1.27 oder neuer.

```bash
./scripts/build.sh
```

Die Dateien landen in `dist/`:

| Datei | System |
| --- | --- |
| `flohh-windows-amd64.exe` | Windows, Intel/AMD |
| `flohh-windows-arm64.exe` | Windows auf ARM |
| `flohh-linux-amd64` | Linux, Intel/AMD |
| `flohh-linux-arm64` | Linux auf ARM (z. B. Raspberry Pi) |
| `flohh-darwin-amd64` | macOS, Intel |
| `flohh-darwin-arm64` | macOS, Apple Silicon (M1 und neuer) |

### macOS ohne Apple-Signatur

Die Mac-Dateien sind (noch) nicht von Apple notarisiert. macOS blockiert sie
nur, wenn sie per Browser, Mail, Nachrichten oder AirDrop auf den Mac kamen.

- **Vom USB-Stick** (unter Windows/Linux vorbereitet) oder per `curl` im
  Terminal geladen: startet direkt.
- **Im Browser heruntergeladen:** einmal starten, dann in
  *Systemeinstellungen → Datenschutz & Sicherheit* auf „Dennoch öffnen“
  klicken (seit macOS 15 hilft Rechtsklick → „Öffnen“ nicht mehr). Oder im
  Terminal: `xattr -d com.apple.quarantine flohh-darwin-*`

### USB-Stick (bevorzugter Weg)

`build.sh` legt zusätzlich `dist/usb/` an: alle sechs Dateien mit
verständlichen Namen (`Flohh-Windows.exe`, `Flohh-Mac-Intel` …)
und eine `LIESMICH.txt` (Vorlage in `packaging/usb/`). Daraus entsteht
`flohh-<version>-usb.zip`: auf einen beliebigen Stick entpacken, vorhandene
Daten bleiben erhalten.

## Starten

Datei ausführen – die Oberfläche öffnet sich im Standardbrowser.

```bash
./dist/flohh-linux-amd64              # Oberfläche starten
./dist/flohh-linux-amd64 version      # Version anzeigen
./dist/flohh-linux-amd64 gui -no-browser -port 8080
```

## Entwickeln

```bash
go run ./cmd/flohh    # direkt starten, ohne zu bauen
go test ./...         # Tests
go vet ./...          # typische Fehler finden
gofmt -l .            # nicht formatierte Dateien anzeigen
```

## Aufbau

```
cmd/flohh/          Einstieg, Befehle
internal/ui/        HTTP-Server für Oberfläche und Steuer-Schnittstelle
internal/browser/   Standardbrowser öffnen (Windows / macOS / Linux)
web/                HTML, CSS, JS – per go:embed in die Datei gepackt
packaging/usb/      Anleitung für den Stick
scripts/            Build-Skript
```
