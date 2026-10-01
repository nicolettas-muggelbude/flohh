# Umzugskarton

Portable App, die beim PC-Umzug Bibliotheken und Ordner per LAN-Kabel von einem
Rechner auf einen anderen kopiert – Windows, Linux und macOS in allen Richtungen.

**Status:** Meilenstein 1 (Gerüst). Die Oberfläche startet, Senden und
Empfangen folgen.

## Bauen

Voraussetzung: Go 1.27 oder neuer.

```bash
./scripts/build.sh
```

Die Dateien landen in `dist/`:

| Datei | System |
| --- | --- |
| `umzugskarton-windows-amd64.exe` | Windows, Intel/AMD |
| `umzugskarton-windows-arm64.exe` | Windows auf ARM |
| `umzugskarton-linux-amd64` | Linux, Intel/AMD |
| `umzugskarton-linux-arm64` | Linux auf ARM (z. B. Raspberry Pi) |
| `umzugskarton-darwin-amd64` | macOS, Intel |
| `umzugskarton-darwin-arm64` | macOS, Apple Silicon (M1 und neuer) |

### macOS ohne Apple-Signatur

Die Mac-Dateien sind (noch) nicht von Apple notarisiert. macOS blockiert sie
nur, wenn sie per Browser, Mail, Nachrichten oder AirDrop auf den Mac kamen.

- **Vom USB-Stick** (unter Windows/Linux vorbereitet) oder per `curl` im
  Terminal geladen: startet direkt.
- **Im Browser heruntergeladen:** einmal starten, dann in
  *Systemeinstellungen → Datenschutz & Sicherheit* auf „Dennoch öffnen“
  klicken (seit macOS 15 hilft Rechtsklick → „Öffnen“ nicht mehr). Oder im
  Terminal: `xattr -d com.apple.quarantine umzugskarton-darwin-*`

## Starten

Datei ausführen – die Oberfläche öffnet sich im Standardbrowser.

```bash
./dist/umzugskarton-linux-amd64              # Oberfläche starten
./dist/umzugskarton-linux-amd64 version      # Version anzeigen
./dist/umzugskarton-linux-amd64 gui -no-browser -port 8080
```

## Entwickeln

```bash
go run ./cmd/umzugskarton    # direkt starten, ohne zu bauen
go test ./...                # Tests
go vet ./...                 # typische Fehler finden
gofmt -l .                   # nicht formatierte Dateien anzeigen
```

## Aufbau

```
cmd/umzugskarton/   Einstieg, Befehle
internal/ui/        HTTP-Server für Oberfläche und Steuer-Schnittstelle
internal/browser/   Standardbrowser öffnen (Windows / macOS / Linux)
web/                HTML, CSS, JS – per go:embed in die Datei gepackt
scripts/            Build-Skript
```
