"use strict";

// Texte immer über t("schlüssel") aus i18n.js holen, nie direkt hinschreiben.

// Ruft die Steuer-Schnittstelle der App auf. Ändernde Aufrufe (POST) brauchen
// den Header X-Umzugskarton, sonst lehnt der Server sie ab.
async function api(path, options = {}) {
  const headers = { "X-Umzugskarton": "1", ...(options.headers || {}) };
  const res = await fetch(path, { ...options, headers });
  if (!res.ok) throw new Error(`${res.status} ${await res.text()}`);
  return res.json();
}

let toastTimer;
function toast(text) {
  const el = document.getElementById("toast");
  el.textContent = text;
  el.hidden = false;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => { el.hidden = true; }, 4000);
}

// Eigennamen – werden nicht übersetzt.
const osNames = { windows: "Windows", linux: "Linux", darwin: "macOS" };

async function loadInfo() {
  const el = document.getElementById("info");
  try {
    const i = await api("/api/info");
    el.textContent = t("info.line", {
      host: i.hostname || t("info.unknownHost"),
      os: osNames[i.os] || i.os,
      arch: i.arch,
      version: i.version,
    });
  } catch (err) {
    el.textContent = t("info.noConnection");
    toast(t("error.generic", { detail: err.message }));
  }
}

document.getElementById("quit").addEventListener("click", async () => {
  try {
    await api("/api/quit", { method: "POST" });
    document.body.innerHTML =
      '<main style="padding-top:48px"><h1 data-i18n="quit.done.title"></h1>' +
      '<p data-i18n="quit.done.text"></p></main>';
    applyI18n();
  } catch (err) {
    toast(t("quit.failed", { detail: err.message }));
  }
});

applyI18n();
loadInfo();
