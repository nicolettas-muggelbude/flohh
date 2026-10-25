"use strict";

// Mehrsprachigkeit. Alle Texte der Oberfläche stehen hier – nirgends sonst.
// Die Go-Seite schickt nur Daten und Fehlercodes, nie fertige Sätze.
//
// Schlüssel sind flach ("bereich.name"). Platzhalter in geschweiften Klammern
// werden von t() ersetzt: t("info.line", { version: "1.0" }).
// Für Mehrzahl ist der Wert ein Objekt mit den Formen aus Intl.PluralRules
// ("one", "other", in anderen Sprachen auch "few", "many" …); die Zahl kommt
// als Platzhalter {count}: t("files", { count: 3 }) → "3 Dateien".
//
// Neue Sprache: einen Block wie "de" ergänzen. Fehlende Schlüssel fallen auf
// Deutsch zurück.
const messages = {
  de: {
    "app.tagline": "Daten springen von Rechner zu Rechner",
    "roles.label": "Rolle wählen",
    "role.send.name": "Senden",
    "role.send.note": "Dieser Rechner ist der alte. Er gibt seine Daten frei.",
    "role.receive.name": "Empfangen",
    "role.receive.note": "Dieser Rechner ist der neue. Er holt die Daten ab.",
    "home.hint": "Starte flohh auf beiden Rechnern und wähle jeweils die passende Rolle.",
    "home.firewallWindows": "Windows fragt beim ersten Mal, ob flohh ins Netzwerk darf. Bitte „Zugriff zulassen“ klicken – mit Haken bei „Öffentliche Netzwerke“, denn ein direktes Kabel gilt für Windows als öffentliches Netz.",
    "common.cancel": "Abbrechen",

    "receive.title": "Empfangen",
    "receive.instruction": "Gib diesen Code am alten Rechner ein:",
    "receive.waiting": "Warte auf den alten Rechner …",
    "receive.seen": "Gefunden: {names}. Warte auf den Code …",
    "receive.addrs": "Findet dich der alte Rechner nicht? Dort unter „Adresse eingeben“: {addrs}",

    "send.title": "Senden",
    "send.instruction": "Wähle den neuen Rechner aus und gib den Code ein, den er anzeigt.",
    "send.peersLabel": "Neue Rechner im Netz",
    "send.searching": "Suche neue Rechner …",
    "send.searchingLong": "Noch kein Rechner gefunden. Läuft flohh auf dem neuen Rechner, ist dort „Empfangen“ gewählt und sind beide im selben Netz oder per Kabel verbunden? Ist der neue Rechner ein Windows-PC, muss dort im Firewall-Fenster „Zugriff zulassen“ gewählt sein. Sonst unten die Adresse eingeben.",
    "send.manualToggle": "Rechner nicht dabei? Adresse eingeben",
    "send.manualHint": "Der neue Rechner zeigt seine Adresse unter dem Code an. Das hilft, wenn der Router die Geräte trennt – z. B. LAN und WLAN in getrennten Netzen.",
    "send.addrLabel": "Adresse des neuen Rechners",
    "send.codeLabel": "Code vom neuen Rechner",
    "send.connect": "Verbinden",
    "send.pairing": "Prüfe Code …",
    "send.choosePeer": "Bitte zuerst einen Rechner auswählen.",

    "connected.title": "Verbunden",
    "connected.text": "Verbunden mit {name} ({os}). Die Verbindung ist verschlüsselt.",
    "connected.title.send": "Verbunden – alter Rechner",
    "connected.title.receive": "Verbunden – neuer Rechner",
    "connected.disconnect": "Trennen",

    "folder.documents": "Dokumente",
    "folder.pictures": "Bilder",
    "folder.music": "Musik",
    "folder.videos": "Videos",
    "folder.desktop": "Schreibtisch",
    "folder.downloads": "Downloads",
    "folder.files": { one: "{count} Datei", other: "{count} Dateien" },
    "folder.counting": "wird gezählt …",
    "folder.cloud": { one: "{count} nur in der Cloud", other: "{count} nur in der Cloud" },
    "folder.target": "kommt nach {path}",
    "folder.from": "vom alten Rechner: {path}",
    "folder.remove": "Entfernen",
    "folder.change": "Ändern …",
    "folder.appdataHint": "Programmeinstellungen von Windows – unter Linux und macOS meist nicht direkt nutzbar.",

    "share.addFolder": "Weiteren Ordner hinzufügen …",

    "browse.titleAdd": "Ordner hinzufügen",
    "browse.titleTarget": "Ziel für „{name}“",
    "browse.hintAdd": "Wähle einen Ordner, der zusätzlich auf den neuen Rechner soll.",
    "browse.hintTarget": "Wähle, wohin der Ordner auf diesem Rechner kommt. Vorhandene Dateien werden nicht überschrieben.",
    "browse.home": "Persönlicher Ordner",
    "browse.up": "Eine Ebene höher",
    "browse.showHidden": "Versteckte Ordner zeigen (z. B. AppData)",
    "browse.empty": "Keine Unterordner.",
    "browse.add": "Diesen Ordner hinzufügen",
    "browse.here": "Hierhin",
    "browse.sub": "Unterordner „{name}“ hier anlegen",

    "share.title": "Was soll auf den neuen Rechner?",
    "share.button": "Freigeben",
    "share.waiting": "Freigegeben: {names}. Warte, bis der neue Rechner die Übertragung startet …",
    "share.receiving": "Der neue Rechner holt die Daten ab:",
    "share.finished": "Ergebnis beim neuen Rechner:",

    "offer.title": "Der alte Rechner gibt diese Ordner frei:",
    "offer.wait": "Warte, bis am alten Rechner Ordner freigegeben werden …",
    "offer.start": "Übertragung starten",

    "progress.listing": "Dateien werden aufgelistet …",
    "progress.bytes": "{percent} · {done} von {total}",
    "progress.files": "{done} von {total} Dateien",
    "progress.eta": "Noch etwa {time} · {speed}/s",
    "progress.stop": "Anhalten",
    "progress.stopping": "Hält nach der aktuellen Datei an …",
    "progress.again": "Erneut starten",
    "progress.done": "Fertig! Alle Daten sind umgezogen.",
    "progress.doneWithErrors": "Fertig – aber nicht alle Dateien ließen sich übertragen.",
    "progress.summary": "{copied} kopiert · {skipped} schon vorhanden · {failed} Fehler",
    "progress.cancelled": "Angehalten. Ein neuer Start setzt fort – was schon da ist, wird übersprungen.",
    "progress.failed": "Abgebrochen: Die Verbindung zum anderen Rechner ist weg.",
    "progress.failures": "Diese Dateien konnten nicht übertragen werden:",
    "progress.skippedLabel": "Bewusst weggelassen:",
    "progress.skippedSystem": { one: "{count} Systemdatei (z. B. desktop.ini)", other: "{count} Systemdateien (z. B. desktop.ini)" },
    "progress.skippedCloud": { one: "{count} Datei nur in der Cloud", other: "{count} Dateien nur in der Cloud" },
    "progress.cloudHint": "Dateien, die nur in der Cloud liegen (z. B. OneDrive „Nur online verfügbar“), holst du am besten auf dem neuen Rechner direkt aus der Cloud.",

    "failure.read_failed": "auf dem alten Rechner nicht lesbar",
    "failure.locked": "von einem Programm geöffnet",
    "failure.lockedBy": "von {app} geöffnet",
    "progress.lockedByApp": {
      one: "Gesperrt von {app}: {count} Datei",
      other: "Gesperrt von {app}: {count} Dateien",
    },
    "progress.lockedHintApps": "Beende diese Programme ganz – auch im Hintergrund, z. B. über den Task-Manager – und klicke „Erneut starten“. Bereits Kopiertes wird übersprungen.",
    "progress.windowsItself": "Windows selbst",
    "progress.windowsTip": "Dateien, die Windows selbst offen hält (Dienste, Explorer), braucht der neue Rechner nicht – sie lassen sich ignorieren.",
    "progress.edgeTip": "Microsoft Edge läuft nach dem Schließen im Hintergrund weiter. Beende ihn im Task-Manager oder schalte in Edge unter Einstellungen → System und Leistung „Startboost“ und „Erweiterungen und Apps im Hintergrund weiter ausführen“ aus.",
    "failure.access_denied": "keine Berechtigung zum Lesen",
    "progress.lockedHint": {
      one: "{count} Datei war von einem Programm geöffnet (z. B. Browser oder Mailprogramm). Schließe die Programme auf dem alten Rechner und klicke „Erneut starten“ – bereits Kopiertes wird übersprungen.",
      other: "{count} Dateien waren von Programmen geöffnet (z. B. Browser oder Mailprogramm). Schließe die Programme auf dem alten Rechner und klicke „Erneut starten“ – bereits Kopiertes wird übersprungen.",
    },
    "failure.not_found": "auf dem alten Rechner nicht mehr vorhanden",
    "failure.file_changed": "hat sich während der Übertragung geändert",
    "failure.write_failed": "ließ sich hier nicht speichern",
    "failure.invalid_path": "ungültiger Name",

    "time.lessMinute": "unter 1 Minute",
    "time.minutes": { one: "{count} Minute", other: "{count} Minuten" },
    "time.hours": "{hours} Std. {minutes} Min.",

    "error.not_connected": "Keine Verbindung zum anderen Rechner.",
    "error.nothing_selected": "Bitte mindestens einen Ordner auswählen.",
    "error.busy": "Die Übertragung läuft schon.",
    "error.bad_folder": "Das ist kein lesbarer Ordner.",
    "error.duplicate": "Dieser Ordner ist schon in der Liste.",
    "error.bad_target": "In diesen Zielordner lässt sich nicht schreiben.",
    "error.browse_failed": "Dieser Ordner lässt sich nicht öffnen.",

    "error.bad_code": "Der Code hat 6 Ziffern.",
    "error.wrong_code": "Der Code stimmt nicht. Bitte noch einmal vom neuen Rechner ablesen.",
    "error.peer_gone": "Die Verbindung zum anderen Rechner ist weg.",
    "error.connect_failed": "Der andere Rechner ist nicht erreichbar. Läuft flohh dort noch? Unter Linux kann auch eine Firewall die Verbindung blocken – Hinweise stehen im Konsolenfenster.",
    "error.menu_failed": "flohh ließ sich nicht ins Anwendungsmenü aufnehmen.",

    "menu.offer": "flohh ins Anwendungsmenü aufnehmen? Dann startest du es künftig mit einem Klick über „Anwendungen anzeigen“.",
    "menu.install": "Aufnehmen",
    "menu.decline": "Nein danke",
    "menu.done": "flohh ist jetzt im Anwendungsmenü.",
    "error.wrong_role": "Das passt gerade nicht zur gewählten Rolle.",
    "error.network": "Das Netzwerk lässt sich nicht nutzen. Läuft flohh vielleicht schon einmal?",
    "error.firewall_peer": "Der neue Rechner (Windows) lässt keine Verbindung zu. Dort im Firewall-Fenster „Zugriff zulassen“ wählen – kam keins, flohh dort beenden, neu starten und „Empfangen“ wählen.",
    "error.bad_address": "Das ist keine gültige Adresse (z. B. 192.168.178.23).",
    "error.not_receiver": "Unter dieser Adresse läuft flohh, aber nicht mit „Empfangen“.",
    "error.bad_request": "Ungültige Anfrage.",
    "error.internal": "Unerwarteter Fehler.",

    "info.connecting": "Verbinde …",
    "info.line": "{host} · {os} ({arch}) · Version {version}",
    "info.unknownHost": "unbekannt",
    "info.noConnection": "Keine Verbindung zur App",
    "error.generic": "Fehler: {detail}",
    "quit.button": "Beenden",
    "quit.failed": "Beenden fehlgeschlagen: {detail}",
    "quit.done.title": "Flohh wurde beendet.",
    "quit.done.text": "Dieses Fenster kann geschlossen werden.",
  },
};

const fallbackLang = "de";

// Erste Sprache aus den Browser-Einstellungen, für die es ein Wörterbuch gibt.
const lang = (() => {
  for (const tag of navigator.languages || [navigator.language]) {
    const base = String(tag).toLowerCase().split("-")[0];
    if (messages[base]) return base;
  }
  return fallbackLang;
})();

const pluralRules = new Intl.PluralRules(lang);

function t(key, params = {}) {
  let msg = messages[lang][key] ?? messages[fallbackLang][key];
  if (msg === undefined) {
    console.warn(`i18n: Schlüssel fehlt: ${key}`);
    return key;
  }
  if (typeof msg === "object") {
    msg = msg[pluralRules.select(params.count)] ?? msg.other;
  }
  // Zahlen im Format der Sprache („1.234“), alles andere als Text.
  return msg.replace(/\{(\w+)\}/g, (m, name) => {
    if (!(name in params)) return m;
    const v = params[name];
    return typeof v === "number" ? formatNumber(v) : String(v);
  });
}

// Übersetzt alle markierten Elemente unterhalb von root:
//   data-i18n="schlüssel"            → Textinhalt
//   data-i18n-aria-label="schlüssel" → Attribut aria-label
function applyI18n(root = document) {
  document.documentElement.lang = lang;
  for (const el of root.querySelectorAll("[data-i18n]")) {
    el.textContent = t(el.dataset.i18n);
  }
  for (const el of root.querySelectorAll("[data-i18n-aria-label]")) {
    el.setAttribute("aria-label", t(el.dataset.i18nAriaLabel));
  }
}

// Zahl im Format der Sprache: 1234,5 / 1,234.5
function formatNumber(n, options) {
  return new Intl.NumberFormat(lang, options).format(n);
}

// Bytes lesbar, Zehnerpotenzen wie bei Datenträgern üblich: "1,5 GB".
function formatBytes(bytes) {
  const units = ["byte", "kilobyte", "megabyte", "gigabyte", "terabyte"];
  let i = 0;
  let v = bytes;
  while (v >= 1000 && i < units.length - 1) {
    v /= 1000;
    i++;
  }
  return formatNumber(v, {
    style: "unit",
    unit: units[i],
    maximumFractionDigits: i === 0 ? 0 : 1,
  });
}
