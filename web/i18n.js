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
    "app.tagline": "Daten per LAN-Kabel von Rechner zu Rechner umziehen",
    "roles.label": "Rolle wählen",
    "role.send.name": "Senden",
    "role.send.note": "Dieser Rechner ist der alte. Er gibt seine Daten frei.",
    "role.receive.name": "Empfangen",
    "role.receive.note": "Dieser Rechner ist der neue. Er holt die Daten ab.",
    "hint.milestone": "Senden und Empfangen folgen in den nächsten Meilensteinen.",
    "info.connecting": "Verbinde …",
    "info.line": "{host} · {os} ({arch}) · Version {version}",
    "info.unknownHost": "unbekannt",
    "info.noConnection": "Keine Verbindung zur App",
    "error.generic": "Fehler: {detail}",
    "quit.button": "Beenden",
    "quit.failed": "Beenden fehlgeschlagen: {detail}",
    "quit.done.title": "Umzugskarton wurde beendet.",
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
  return msg.replace(/\{(\w+)\}/g, (m, name) =>
    name in params ? String(params[name]) : m);
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
