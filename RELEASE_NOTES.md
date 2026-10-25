# 🎉 flohh v0.2.0 Release Notes

## 💾 Downloads

| Plattform | Download |
|-----------|----------|
| 📦 **Alle Systeme** (empfohlen – ZIP mit allen Programmen, Anleitung und Prüfsummen, auch für den USB-Stick) | [⬇ flohh-v0.2.0.zip](https://github.com/nicolettas-muggelbude/flohh/releases/download/v0.2.0/flohh-v0.2.0.zip) |
| 🪟 **Windows** (x64) | [⬇ flohh-v0.2.0-windows-x64.exe](https://github.com/nicolettas-muggelbude/flohh/releases/download/v0.2.0/flohh-v0.2.0-windows-x64.exe) |
| 🪟 **Windows** (ARM, z. B. Surface Pro X) | [⬇ flohh-v0.2.0-windows-arm64.exe](https://github.com/nicolettas-muggelbude/flohh/releases/download/v0.2.0/flohh-v0.2.0-windows-arm64.exe) |
| 🐧 **Linux** (x86_64) | [⬇ flohh-v0.2.0-linux-x86_64](https://github.com/nicolettas-muggelbude/flohh/releases/download/v0.2.0/flohh-v0.2.0-linux-x86_64) |
| 🐧 **Linux** (ARM64, z. B. Raspberry Pi) | [⬇ flohh-v0.2.0-linux-arm64](https://github.com/nicolettas-muggelbude/flohh/releases/download/v0.2.0/flohh-v0.2.0-linux-arm64) |
| 🍎 **macOS** (Apple Silicon, ungetestet) | [⬇ flohh-v0.2.0-macos-applesilicon](https://github.com/nicolettas-muggelbude/flohh/releases/download/v0.2.0/flohh-v0.2.0-macos-applesilicon) |
| 🍎 **macOS** (Intel, ungetestet) | [⬇ flohh-v0.2.0-macos-intel](https://github.com/nicolettas-muggelbude/flohh/releases/download/v0.2.0/flohh-v0.2.0-macos-intel) |
| 🔐 Prüfsummen | [⬇ SHA256SUMS.txt](https://github.com/nicolettas-muggelbude/flohh/releases/download/v0.2.0/SHA256SUMS.txt) |

---

## 🪟↔🪟 Jetzt auch von Windows zu Windows!

**flohh v0.2.0** schafft den häufigsten Umzug überhaupt: vom alten auf den
neuen Windows-PC – per LAN, per WLAN oder per Kabel direkt. Wie gehabt ohne
Installation, ohne Cloud, ohne Konto, ohne externes Laufwerk (HDD/SSD).

**Ein Floh springt nicht nur von Hund zu Hund – er findet auch über den
Gartenzaun hinweg den Weg.** 🐾

---

## ✨ Highlights dieser Version

### 🪟 Windows ↔ Windows

- Ein Windows-Rechner, der **„Empfangen“** wählt, nimmt jetzt Verbindungen an
- Beim ersten Mal fragt die **Windows-Firewall** einmal nach – „Zugriff
  zulassen“ klicken, danach nie wieder
- Blockt die Firewall doch, sagt flohh das **klar und mit Lösung** statt nur
  „nicht erreichbar“
- Im Test: **Windows ↔ Windows über LAN und WLAN** ✅

### 📍 Verbinden per Adresse

- Finden sich die Rechner nicht von selbst – etwa weil der Router **LAN und
  WLAN trennt** oder Rundrufe filtert –, zeigt der neue Rechner seine
  **Adresse** unter dem Code an
- Am alten Rechner unter **„Rechner nicht dabei? Adresse eingeben“**
  eintippen, Code dazu – fertig

### 🐧 Linux

- Steckt im **Anwendungsmenü** noch eine ältere flohh-Version, bietet flohh
  **„Aktualisieren“** an

### 📦 Download und Anleitung

- **Ein ZIP mit allem** steht ganz oben – entpackt ergibt es einen Ordner mit
  allen Programmen, der Anleitung und **Prüfsummen** (`SHA256SUMS.txt`)
- Neuer Abschnitt **„Damit nichts kaputtgeht“**: erst auf dem Zielrechner
  entpacken, Stick immer auswerfen, Dateien bei Bedarf prüfen

---

## 🚀 Starten

Am einfachsten: **flohh-v0.2.0.zip** laden und **auf dem Rechner** entpacken,
auf dem flohh laufen soll – im Ordner `flohh-v0.2.0` liegen die Programme
für alle Systeme und die Anleitung (`LIESMICH.txt`).

### 🪟 Windows

Doppelklick auf `Flohh-Windows.exe`. Bei „Der Computer wurde durch Windows
geschützt“ auf „Weitere Informationen“ → „Trotzdem ausführen“. Fragt die
Firewall nach: **„Zugriff zulassen“** (nur beim ersten Mal).

### 🐧 Linux – bitte genau so, ein Doppelklick startet flohh nicht

1. **Kopieren:** `Flohh-Linux` (bzw. `Flohh-Linux-ARM`) auf den
   **Schreibtisch** oder in den persönlichen Ordner kopieren. Direkt vom
   USB-Stick oder aus dem ZIP lässt es sich nicht starten.
2. **Ausführbar machen** – ein Weg genügt:
   - **Rechtsklick** auf die Kopie → Eigenschaften → Schalter
     „Als Programm ausführen“ bzw. „Ausführbar“ einschalten
   - **Dateimanager:** Eigenschaften → Zugriffsrechte → Haken bei
     „Datei als Programm ausführen“
   - **Konsole:** `chmod +x ~/Schreibtisch/Flohh-Linux`
3. **Starten – nicht per Doppelklick:**
   - **Rechtsklick** auf die Datei → „Als Programm ausführen“
   - oder in der **Konsole:** `~/Schreibtisch/Flohh-Linux`
4. flohh bietet dann an, sich ins **Anwendungsmenü** aufzunehmen (bzw. eine
   ältere Version dort zu aktualisieren).

Stürzt flohh sofort mit „Speicherzugriffsfehler“ ab, ist die Datei beim
Kopieren beschädigt worden – prüfen mit `sha256sum -c --ignore-missing
SHA256SUMS.txt` im entpackten Ordner, dann neu kopieren.

### 🔁 Auf beiden Rechnern

Rolle wählen (alter Rechner „Senden“, neuer „Empfangen“), Code eingeben,
Ordner freigeben, Übertragung starten.

---

## ⚠️ Bekannte Einschränkungen

- **v0.2.0 und v0.1.0 verstehen sich nicht** – bitte auf beiden Rechnern
  v0.2.0 verwenden
- **macOS** ist gebaut, aber ungetestet und nicht von Apple signiert
- Dateien, die **nur in der Cloud** liegen (OneDrive), werden nicht übertragen

**Getestet:** Windows ↔ Windows über LAN und WLAN, Windows 10 ↔ Ubuntu 24.04
per Direktkabel und USB-LAN-Adapter.

---

📋 Alle Änderungen: [CHANGELOG.md](https://github.com/nicolettas-muggelbude/flohh/blob/main/CHANGELOG.md)
