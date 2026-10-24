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
    document.getElementById("firewall-hint").hidden = i.os !== "windows";
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
    if (state.phase === "connected") move = await api("/api/move");
    render();
  } catch {
    // App beendet oder kurz nicht erreichbar – beim nächsten Mal wieder.
  }
}

// ---- Umzug: freigeben, abholen, Fortschritt ----

const el = (id) => document.getElementById(id);
let move = null;
const shareSel = new Set();   // Sender: gewählte Ordnerarten
const offerSel = new Set();   // Empfänger: gewählte Ordnerarten
const offerSeen = new Set();  // Empfänger: schon einmal angebotene Arten
let shareKey = "", offerKey = "";

function resetMoveUI() {
  move = null;
  shareSel.clear(); offerSel.clear(); offerSeen.clear();
  shareKey = offerKey = "";
}

function renderMove() {
  if (!move || !move.connected) return;
  const sender = move.role === "send";
  document.querySelector("#view-connected h2").textContent = t(`connected.title.${move.role}`);
  el("send-share").hidden = !sender || move.shared;
  el("share-status").hidden = !sender || !move.shared;
  el("recv-offer").hidden = sender || move.running || !!move.progress;
  el("recv-progress").hidden = sender || (!move.running && !move.progress);
  if (sender) renderSender(); else renderReceiver();
}

function renderSender() {
  if (!move.shared) {
    const key = JSON.stringify(move.folders);
    if (key !== shareKey) {
      if (!shareKey) move.folders.forEach((f) => shareSel.add(f.kind)); // anfangs alle
      shareKey = key;
      folderRows(el("share-list"), move.folders, shareSel, false);
    }
    el("share-button").disabled = !move.folders.length;
    return;
  }
  const names = move.folders.filter((f) => f.shared).map((f) => t(`folder.${f.kind}`)).join(", ");
  const p = move.progress;
  const busy = p && (p.phase === "listing" || p.phase === "copying");
  el("share-status-text").textContent =
    !p ? t("share.waiting", { names }) : busy ? t("share.receiving") : t("share.finished");
  renderProgress(el("send-progress"), move.progress, false);
}

function renderReceiver() {
  for (const f of move.folders) {
    if (!offerSeen.has(f.kind)) { offerSeen.add(f.kind); offerSel.add(f.kind); } // neu angeboten: gewählt
  }
  const key = JSON.stringify(move.folders);
  if (key !== offerKey) {
    offerKey = key;
    folderRows(el("offer-list"), move.folders, offerSel, true);
  }
  el("offer-wait").textContent = move.folders.length ? "" : t("offer.wait");
  el("start-button").disabled = !move.folders.length;
  renderProgress(el("recv-progress-box"), move.progress, move.stopping);
  el("stop-button").hidden = !move.running;
  el("again-button").hidden = move.running;
}

function span(cls, text) {
  const s = document.createElement("span");
  s.className = cls;
  s.textContent = text;
  return s;
}

// folderRows zeichnet Ordner mit Häkchen; sel merkt sich die Auswahl.
function folderRows(container, list, sel, showTarget) {
  container.replaceChildren(...list.map((f) => {
    const label = document.createElement("label");
    label.className = "folder";
    const cb = document.createElement("input");
    cb.type = "checkbox";
    cb.checked = sel.has(f.kind);
    cb.addEventListener("change", () => (cb.checked ? sel.add(f.kind) : sel.delete(f.kind)));
    const text = document.createElement("span");
    text.className = "folder-text";
    text.append(
      span("folder-name", t(`folder.${f.kind}`)),
      span("folder-path", showTarget ? t("folder.target", { path: f.path }) : f.path),
    );
    const size = f.ready
      ? `${t("folder.files", { count: f.files })} · ${formatBytes(f.bytes)}`
      : t("folder.counting");
    label.append(cb, text, span("folder-size", size));
    return label;
  }));
}

function formatDuration(sec) {
  if (sec < 60) return t("time.lessMinute");
  const m = Math.round(sec / 60);
  if (m < 60) return t("time.minutes", { count: m });
  return t("time.hours", { hours: Math.floor(m / 60), minutes: m % 60 });
}

function failureText(code) {
  const key = `failure.${code}`;
  return (messages[lang][key] ?? messages[fallbackLang][key]) ? t(key) : code;
}

// currentSpeed: Tempo der letzten Sekunden (Bytes/s), nur wirklich
// übertragene Bytes. Ein Durchschnitt seit Beginn wäre irreführend – viele
// kleine Dateien am Anfang drücken ihn lange nach unten. Messpunkte hängen
// am Anzeigekasten, jede Seite misst für sich.
const speedSamples = new WeakMap();
function currentSpeed(box, p) {
  let s = speedSamples.get(box);
  if (!s || s.run !== p.startedAt) {
    s = { run: p.startedAt, points: [] };
    speedSamples.set(box, s);
  }
  const now = Date.now();
  s.points.push({ t: now, bytes: p.bytesCopied });
  while (s.points.length > 2 && now - s.points[0].t > 8000) s.points.shift();
  const first = s.points[0];
  const dt = (now - first.t) / 1000;
  return dt >= 2 ? (p.bytesCopied - first.bytes) / dt : 0;
}

// renderProgress zeigt Balken, Zahlen, aktuelle Datei bzw. Ergebnis.
function renderProgress(box, p, stopping) {
  box.replaceChildren();
  if (!p) return;
  const add = (cls, text) => {
    const e = document.createElement("p");
    e.className = cls;
    e.textContent = text;
    box.append(e);
  };
  if (p.phase === "listing") {
    add("status", t("progress.listing"));
    return;
  }
  const frac = p.bytesTotal ? Math.min(1, p.bytesDone / p.bytesTotal) : (p.phase === "done" ? 1 : 0);
  const bar = document.createElement("div");
  bar.className = "bar";
  bar.setAttribute("role", "progressbar");
  bar.setAttribute("aria-valuemin", "0");
  bar.setAttribute("aria-valuemax", "100");
  bar.setAttribute("aria-valuenow", String(Math.round(frac * 100)));
  const fill = document.createElement("div");
  fill.className = "bar-fill";
  fill.style.width = `${(frac * 100).toFixed(1)}%`;
  bar.append(fill);
  box.append(bar);

  add("progress-line", t("progress.bytes", {
    percent: formatNumber(frac, { style: "percent", maximumFractionDigits: 0 }),
    done: formatBytes(p.bytesDone),
    total: formatBytes(p.bytesTotal),
  }));
  add("progress-line", t("progress.files", {
    done: p.filesDone + p.filesSkipped + p.filesFailed,
    total: p.filesTotal,
  }));

  if (p.phase === "copying") {
    const speed = currentSpeed(box, p);
    if (speed > 0) {
      add("progress-line muted", t("progress.eta", {
        time: formatDuration((p.bytesTotal - p.bytesDone) / speed),
        speed: formatBytes(speed),
      }));
    }
    if (stopping) add("status", t("progress.stopping"));
    if (p.current) add("current", p.current);
    return;
  }

  const msg = {
    done: p.filesFailed ? "progress.doneWithErrors" : "progress.done",
    cancelled: "progress.cancelled",
    failed: "progress.failed",
  }[p.phase];
  if (msg) add("result", t(msg));
  add("progress-line", t("progress.summary", {
    copied: p.filesDone, skipped: p.filesSkipped, failed: p.filesFailed,
  }));
  if (p.failures && p.failures.length) {
    add("progress-line", t("progress.failures"));
    const ul = document.createElement("ul");
    ul.className = "failures";
    for (const f of p.failures) {
      const li = document.createElement("li");
      li.textContent = `${f.path} – ${failureText(f.error)}`;
      ul.append(li);
    }
    box.append(ul);
  }
}

async function moveAction(path, data) {
  try {
    move = await post(path, data);
    render();
  } catch (err) {
    toast(errorText(err.code));
  }
}

el("share-button").addEventListener("click", () => moveAction("/api/move/share", { kinds: [...shareSel] }));
el("start-button").addEventListener("click", () => moveAction("/api/move/start", { kinds: [...offerSel] }));
el("again-button").addEventListener("click", () => moveAction("/api/move/start", { kinds: [...offerSel] }));
el("stop-button").addEventListener("click", () => moveAction("/api/move/stop"));

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
    renderMove();
    return;
  }
  resetMoveUI();
  if (state.role === "receive") {
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
