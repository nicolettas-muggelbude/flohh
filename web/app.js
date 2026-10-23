"use strict";

// Texte immer über t("schlüssel") aus i18n.js holen, nie direkt hinschreiben.

// Ruft die Steuer-Schnittstelle der App auf. Ändernde Aufrufe (POST) brauchen
// den Header X-Flohh, sonst lehnt der Server sie ab. Fehler kommen als
// {"error": "code"} und landen in err.code.
async function api(path, options = {}) {
  const headers = { "X-Flohh": "1", ...(options.headers || {}) };
  if (options.body) headers["Content-Type"] = "application/json";
  const res = await fetch(path, { ...options, headers });
  if (!res.ok) {
    let code = "internal";
    try { code = (await res.json()).error || code; } catch { /* kein JSON */ }
    const err = new Error(code);
    err.code = code;
    throw err;
  }
  return res.json();
}

function post(path, data) {
  return api(path, { method: "POST", body: data ? JSON.stringify(data) : undefined });
}

// Fehlercode → Text; unbekannte Codes bekommen einen allgemeinen Text.
function errorText(code) {
  const key = `error.${code}`;
  return (messages[lang][key] ?? messages[fallbackLang][key]) ? t(key) : t("error.internal");
}

let toastTimer;
function toast(text) {
  const el = document.getElementById("toast");
  el.textContent = text;
  el.hidden = false;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => { el.hidden = true; }, 5000);
}

// Eigennamen – werden nicht übersetzt.
const osNames = { windows: "Windows", linux: "Linux", darwin: "macOS" };
const osName = (os) => osNames[os] || os;

async function loadInfo() {
  const el = document.getElementById("info");
  try {
    const i = await api("/api/info");
    el.textContent = t("info.line", {
      host: i.hostname || t("info.unknownHost"),
      os: osName(i.os),
      arch: i.arch,
      version: i.version,
    });
    document.getElementById("menu-offer").hidden = !i.menu.offer;
  } catch (err) {
    el.textContent = t("info.noConnection");
  }
}

// Linux: ins Anwendungsmenü aufnehmen oder „Nein danke“
async function menu(action) {
  try {
    await post("/api/menu", { action });
    document.getElementById("menu-offer").hidden = true;
    if (action === "install") toast(t("menu.done"));
  } catch (err) {
    toast(errorText(err.code));
  }
}
document.getElementById("menu-install").addEventListener("click", () => menu("install"));
document.getElementById("menu-decline").addEventListener("click", () => menu("decline"));

// ---- Zustand vom Server holen und anzeigen ----

let state = null;
let selectedPeer = null;   // ID des gewählten Empfängers (Sender)
let searchingSince = 0;    // für den Hinweis, wenn lange nichts gefunden wird
let pairing = false;       // Code wird gerade geprüft

async function refresh() {
  try {
    state = await api("/api/state");
    render();
  } catch {
    // App beendet oder kurz nicht erreichbar – beim nächsten Mal wieder.
  }
}

function show(view) {
  for (const v of ["home", "receive", "send", "connected"]) {
    document.getElementById(`view-${v}`).hidden = v !== view;
  }
}

function render() {
  if (state.phase === "connected") {
    show("connected");
    const p = state.partner;
    document.getElementById("connected-text").textContent =
      t("connected.text", { name: p.name, os: osName(p.os) });
  } else if (state.role === "receive") {
    show("receive");
    renderReceive();
  } else if (state.role === "send") {
    show("send");
    renderSend();
  } else {
    show("home");
    searchingSince = 0;
    // z. B. „Verbindung weg“, wenn die Gegenseite getrennt hat
    if (state.error && state.error !== lastError) toast(errorText(state.error));
  }
  lastError = state.error || "";
}
let lastError = "";

function renderReceive() {
  const code = state.code || "";
  document.getElementById("code").textContent = `${code.slice(0, 3)} ${code.slice(3)}`;
  const status = document.getElementById("receive-status");
  if (state.error) {
    status.textContent = errorText(state.error);
  } else if (state.peers.length) {
    status.textContent = t("receive.seen", { names: state.peers.map((p) => p.name).join(", ") });
  } else {
    status.textContent = t("receive.waiting");
  }
}

let renderedPeers = "";
function renderSend() {
  if (!searchingSince) searchingSince = Date.now();
  const list = document.getElementById("peer-list");
  const ids = state.peers.map((p) => `${p.id}:${p.name}`).join("|");

  // Nur neu zeichnen, wenn sich die Liste ändert – sonst springt der Fokus.
  if (ids !== renderedPeers) {
    renderedPeers = ids;
    if (!state.peers.some((p) => p.id === selectedPeer)) {
      selectedPeer = state.peers.length === 1 ? state.peers[0].id : null;
    }
    list.replaceChildren(...state.peers.map((p) => {
      const label = document.createElement("label");
      label.className = "peer";
      const input = document.createElement("input");
      input.type = "radio";
      input.name = "peer";
      input.value = p.id;
      input.checked = p.id === selectedPeer;
      input.addEventListener("change", () => { selectedPeer = p.id; });
      const name = document.createElement("span");
      name.className = "peer-name";
      name.textContent = p.name;
      const os = document.createElement("span");
      os.className = "peer-os";
      os.textContent = osName(p.os);
      label.append(input, name, os);
      return label;
    }));
  }

  const status = document.getElementById("send-status");
  if (pairing) {
    status.textContent = t("send.pairing");
  } else if (state.peers.length) {
    status.textContent = "";
  } else if (Date.now() - searchingSince > 8000) {
    status.textContent = t("send.searchingLong");
  } else {
    status.textContent = t("send.searching");
  }
  document.getElementById("pair-button").disabled = pairing || !state.peers.length;
}

// ---- Aktionen ----

for (const btn of document.querySelectorAll("[data-role]")) {
  btn.addEventListener("click", async () => {
    try {
      state = await post("/api/role", { role: btn.dataset.role });
      render();
    } catch (err) {
      toast(errorText(err.code));
    }
  });
}

for (const btn of document.querySelectorAll("[data-action=reset]")) {
  btn.addEventListener("click", async () => {
    try {
      state = await post("/api/reset");
      renderedPeers = "";
      document.getElementById("code-input").value = "";
      render();
    } catch (err) {
      toast(errorText(err.code));
    }
  });
}

document.getElementById("pair-form").addEventListener("submit", async (ev) => {
  ev.preventDefault();
  if (!selectedPeer) {
    toast(t("send.choosePeer"));
    return;
  }
  const input = document.getElementById("code-input");
  pairing = true;
  render();
  try {
    state = await post("/api/pair", { peer: selectedPeer, code: input.value });
    input.value = "";
  } catch (err) {
    toast(errorText(err.code));
    input.select();
  } finally {
    pairing = false;
    await refresh();
  }
});

document.getElementById("quit").addEventListener("click", async () => {
  try {
    await post("/api/quit");
    document.body.innerHTML =
      '<main style="padding-top:48px"><h1 data-i18n="quit.done.title"></h1>' +
      '<p data-i18n="quit.done.text"></p></main>';
    applyI18n();
  } catch (err) {
    toast(t("quit.failed", { detail: errorText(err.code) }));
  }
});

applyI18n();
loadInfo();
refresh();
setInterval(refresh, 1000);
