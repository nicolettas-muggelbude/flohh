<h1>
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/logo/flohh-logo-dark.svg">
    <img src="assets/logo/flohh-logo.svg" alt="flohh" height="96">
  </picture>
</h1>

*Daten springen von Rechner zu Rechner.*

Portable App, die beim PC-Umzug Bibliotheken und Ordner von einem Rechner auf
einen anderen kopiert – Windows, Linux und macOS in allen Richtungen. Per
LAN-Kabel, über den Router (LAN oder WLAN) oder per Thunderbolt/USB4.

**Status:** Meilenstein 2 läuft. Zwei Rechner finden sich im Netz (auch per
direktem Kabel) und koppeln sich verschlüsselt per 6-stelligem Code. Der alte
Rechner gibt Benutzerordner frei (Dokumente, Bilder, Musik, Videos,
Schreibtisch, Downloads), der neue holt sie in seine eigenen Benutzerordner –
mit Fortschritt, ohne etwas zu überschreiben, und fortsetzbar.

## So lassen sich die Rechner verbinden

| Verbindung | Tempo (ca.) | Hinweis |
| --- | --- | --- |
| **LAN-Kabel direkt** von Rechner zu Rechner | 1 Gbit: bis 110 MB/s, 2,5 Gbit: bis 280 MB/s | kein Router nötig |
| **Über den Router** per LAN-Kabel | wie oben, je nach Router | beide am selben Router |
| **Über den Router** per WLAN | 5–60 MB/s | beide im selben WLAN, nicht im Gast-WLAN |
| **Gemischt**: einer per LAN, einer per WLAN | wie WLAN | beide am selben Router |
| **Thunderbolt 3/4/5 oder USB4** (USB-C) direkt | 300 MB/s und mehr | beide Anschlüsse müssen Thunderbolt/USB4 können |
| **USB-LAN-Adapter** (USB-A oder USB-C) plus LAN-Kabel | 2,5-Gbit-Adapter an USB 3: bis 280 MB/s, 1 Gbit: bis 110 MB/s, an USB 2: ca. 35 MB/s | für Laptops ohne LAN-Anschluss |

Die Werte sind Richtwerte: Es zählt die langsamere Seite – der langsamere
Anschluss, Router oder Festplatte. Im Test erreichte ein USB-LAN-Adapter
(Linux) direkt an Windows bis 160 MB/s. Gast-WLANs
und manche Mesh-Systeme lassen Geräte nicht miteinander sprechen
(„Client-Isolation“) – dann per Kabel verbinden.

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
internal/peer/      Andere Rechner finden, verschlüsselt per Code koppeln
internal/folders/   Benutzerordner je System (Dokumente, Bilder …)
internal/transfer/  Übertragungsprotokoll: freigeben, auflisten, abholen
internal/move/      Umzug: verbindet Kopplung, Ordner und Übertragung
internal/stick/     Start vom USB-Stick: Kopie, damit der Stick frei bleibt
internal/desktop/   Linux: Eintrag im Anwendungsmenü
internal/ui/        HTTP-Server für Oberfläche und Steuer-Schnittstelle
internal/browser/   Standardbrowser öffnen (Windows / macOS / Linux)
web/                HTML, CSS, JS – per go:embed in die Datei gepackt
assets/logo/        Logo, Icons (Quellen und fertige Dateien)
packaging/usb/      Anleitung für den Stick
scripts/            Build-Skripte
```

## Lizenz

- **Programmcode:** [GNU General Public License v3.0](LICENSE) (GPLv3)
- **Logo und Floh-Figur:** [CC BY-SA 4.0](assets/logo/LICENSE),
  © 2026 Nicoletta
- **Enthaltene Bibliotheken und Schriften** (BSD-3-Clause, SIL Open Font
  License): siehe [THIRD-PARTY-LICENSES.txt](THIRD-PARTY-LICENSES.txt)
