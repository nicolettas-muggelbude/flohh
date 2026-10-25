# Änderungen

Versionen nach [Semantic Versioning](https://semver.org/lang/de/): `v0.x.y`
ist eine Vorabversion – bis `v1.0.0` kann sich noch einiges ändern.

## v0.1.0 – 2026-10-25

Erste Vorabversion.

### Verbinden

- Zwei Rechner finden sich von selbst: per LAN-Kabel direkt, über den Router
  (LAN oder WLAN), per Thunderbolt/USB4 oder mit USB-LAN-Adaptern.
- Direktes Kabel funktioniert auch, wenn die Rechner verschiedene
  IPv4-Adressbereiche haben (Verbindung dann über IPv6 link-local).
- Kopplung per 6-stelligem Code, verschlüsselt (TLS 1.3 + CPace). Nach drei
  falschen Codes gibt es einen neuen.
- Windows öffnet keinen Port, sondern verbindet sich selbst. Die
  Windows-Firewall fragt nur einmal nach.

### Umziehen

- Der alte Rechner gibt Benutzerordner frei (Dokumente, Bilder, Musik,
  Videos, Schreibtisch, Downloads) – auch eigene Ordner und versteckte wie
  AppData.
- Der neue Rechner holt sie in seine eigenen Benutzerordner („Bilder“ landet
  in „Pictures“) oder in selbst gewählte Ziele.
- Nichts wird überschrieben (sonst „Name (2).ext“); ein erneuter Start setzt
  fort und überspringt Vorhandenes.
- Änderungsdatum bleibt erhalten, Umlaute werden einheitlich geschrieben,
  unter Windows verbotene Zeichen ersetzt.
- Fortschritt mit Tempo und Restzeit; Zusammenfassung mit Fehlern.
- Schnell auch bei vielen kleinen Dateien (Anfragen vorausschicken,
  paralleles Vorlesen). Im Test: 34 GB in 5 Minuten über 1 Gbit.
- Gesperrte Dateien werden übersprungen und mit dem sperrenden Programm
  genannt (z. B. Microsoft Edge), samt Tipp.
- Bewusst weggelassen und gezählt: Systemdateien (desktop.ini …),
  Windows-interne Bereiche in AppData, Dateien, die nur in der Cloud liegen.

### Sonstiges

- Eine Datei pro System, ohne Installation; vom USB-Stick gestartet, gibt
  flohh den Stick sofort wieder frei.
- Linux: Aufnahme ins Anwendungsmenü.
- Oberfläche hell/dunkel nach System, vorbereitet für Übersetzungen.

### Bekannte Einschränkungen

- Windows ↔ Windows geht noch nicht (Firewall-Freigabe folgt).
- macOS ist gebaut, aber ungetestet und nicht von Apple signiert.
- Dateien, die nur in der Cloud liegen (OneDrive), werden nicht übertragen.
