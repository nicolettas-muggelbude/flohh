# 🎉 flohh v0.1.0 Release Notes

## 💾 Downloads

| Plattform | Download |
|-----------|----------|
| 📦 **Alle Systeme** (empfohlen – ZIP mit allen Programmen und Anleitung, auch für den USB-Stick) | [⬇ flohh-v0.1.0.zip](https://github.com/nicolettas-muggelbude/flohh/releases/download/v0.1.0/flohh-v0.1.0.zip) |
| 🪟 **Windows** (x64) | [⬇ flohh-v0.1.0-windows-x64.exe](https://github.com/nicolettas-muggelbude/flohh/releases/download/v0.1.0/flohh-v0.1.0-windows-x64.exe) |
| 🪟 **Windows** (ARM, z. B. Surface Pro X) | [⬇ flohh-v0.1.0-windows-arm64.exe](https://github.com/nicolettas-muggelbude/flohh/releases/download/v0.1.0/flohh-v0.1.0-windows-arm64.exe) |
| 🐧 **Linux** (x86_64) | [⬇ flohh-v0.1.0-linux-x86_64](https://github.com/nicolettas-muggelbude/flohh/releases/download/v0.1.0/flohh-v0.1.0-linux-x86_64) |
| 🐧 **Linux** (ARM64, z. B. Raspberry Pi) | [⬇ flohh-v0.1.0-linux-arm64](https://github.com/nicolettas-muggelbude/flohh/releases/download/v0.1.0/flohh-v0.1.0-linux-arm64) |
| 🍎 **macOS** (Apple Silicon, ungetestet) | [⬇ flohh-v0.1.0-macos-applesilicon](https://github.com/nicolettas-muggelbude/flohh/releases/download/v0.1.0/flohh-v0.1.0-macos-applesilicon) |
| 🍎 **macOS** (Intel, ungetestet) | [⬇ flohh-v0.1.0-macos-intel](https://github.com/nicolettas-muggelbude/flohh/releases/download/v0.1.0/flohh-v0.1.0-macos-intel) |
| 🔐 Prüfsummen | [⬇ SHA256SUMS.txt](https://github.com/nicolettas-muggelbude/flohh/releases/download/v0.1.0/SHA256SUMS.txt) |

---

## 🚀 Erste Vorabversion: Daten springen von Rechner zu Rechner!

**flohh v0.1.0** zieht deine Daten vom alten auf den neuen Rechner um – ohne
Installation, ohne Cloud, ohne Konto, ohne externes Laufwerk (HDD/SSD). Ein
Programm auf beiden Rechnern, ein Kabel dazwischen (oder dasselbe Netz), ein
Code – fertig.

**Wie ein Floh mit einem Satz von Hund zu Hund springt, springen deine Daten
jetzt von Rechner zu Rechner.** 🐾

---

## ✨ Highlights dieser Version

### 🔌 Verbinden ohne Fummelei

- Die Rechner **finden sich von selbst** – per LAN-Kabel direkt, über den
  Router (LAN oder WLAN), per Thunderbolt/USB4 oder mit USB-LAN-Adaptern
- **Direktes Kabel klappt immer**, auch wenn die Rechner verschiedene
  Adressbereiche haben (flohh weicht auf IPv6 link-local aus)
- **Windows öffnet keinen Port** – die Windows-Firewall fragt höchstens
  einmal nach

### 🔒 Sicher gekoppelt

- Der neue Rechner zeigt einen **6-stelligen Code**, der alte gibt ihn ein
- Verschlüsselt mit **TLS 1.3 + CPace** – der Code wird nie übertragen,
  Raten ist aussichtslos (nach drei Fehlversuchen gibt es einen neuen Code)
- Der alte Rechner gibt **nur frei, was angehakt ist** – nur lesend

### 📦 Umziehen, wie es sein soll

- **Benutzerordner** (Dokumente, Bilder, Musik, Videos, Schreibtisch,
  Downloads), **eigene Ordner** und **versteckte** wie AppData
- „Bilder“ landet im Bilder-Ordner des neuen Rechners – **auch wenn er dort
  „Pictures“ heißt**; eigene Ordner bekommen ein Ziel, das du ändern kannst
- **Nichts wird überschrieben** – abweichende Dateien kommen als
  „Name (2).ext“ dazu
- **Abgebrochen? Einfach neu starten** – schon Kopiertes wird übersprungen
- Änderungsdatum bleibt, Umlaute werden einheitlich geschrieben

### ⚡ Schnell

- Volle Leitung bei großen Dateien – **im Test 34 GB in 5 Minuten** über
  1 Gbit (ISO mit 116,8 MB/s)
- Auch viele kleine Dateien sind flott: flohh fragt voraus und liest parallel

### 🧹 Ehrlich bei dem, was nicht mitkommt

- **Gesperrte Dateien** werden mit dem Programm genannt, das sie offen hält
  („Gesperrt von Microsoft Edge: 31 Dateien“), samt Tipp
- **Bewusst weggelassen** und gezählt: Systemdateien wie desktop.ini,
  Windows-interne Bereiche in AppData und Dateien, die nur in der Cloud liegen

### 🧳 Klein und mobil

- **Eine Datei pro System**, keine Installation
- Vom **USB-Stick** gestartet, ist der Stick sofort wieder frei – ein Stick
  reicht für beide Rechner
- Linux: auf Wunsch **Eintrag im Anwendungsmenü**
- Oberfläche **hell/dunkel** nach Systemeinstellung

---

## 🚀 Starten

Am einfachsten: **flohh-v0.1.0.zip** laden und entpacken – im Ordner
`flohh-v0.1.0` liegen die Programme für alle Systeme und eine Anleitung
(`LIESMICH.txt`). Den Ordner kannst du auch auf einen USB-Stick kopieren.

### 🪟 Windows

Doppelklick auf `Flohh-Windows.exe`. Bei „Der Computer wurde durch Windows
geschützt“ auf „Weitere Informationen“ → „Trotzdem ausführen“. Fragt die
Firewall nach: „Zugriff zulassen“ (nur beim ersten Mal).

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
4. flohh bietet dann an, sich ins **Anwendungsmenü** aufzunehmen. Danach
   startest du es mit einem Klick über „Anwendungen anzeigen“.

### 🔁 Auf beiden Rechnern

Rolle wählen (alter Rechner „Senden“, neuer „Empfangen“), Code eingeben,
Ordner freigeben, Übertragung starten.

---

## ⚠️ Bekannte Einschränkungen

- **Windows ↔ Windows** geht noch nicht – die Firewall-Freigabe folgt
- **macOS** ist gebaut, aber ungetestet und nicht von Apple signiert
- Dateien, die **nur in der Cloud** liegen (OneDrive), werden nicht übertragen

**Getestet:** Windows 10 ↔ Ubuntu 24.04 per Direktkabel und USB-LAN-Adapter.

---

📋 Alle Änderungen: [CHANGELOG.md](https://github.com/nicolettas-muggelbude/flohh/blob/main/CHANGELOG.md)
