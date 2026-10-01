# Umzugskarton

Portable App, die beim PC-Umzug Bibliotheken und Ordner per LAN-Kabel von einem
Rechner auf einen anderen kopiert – Windows und Linux in allen Richtungen.

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
internal/browser/   Standardbrowser öffnen (Windows / Linux)
web/                HTML, CSS, JS – per go:embed in die Datei gepackt
scripts/            Build-Skript
```
