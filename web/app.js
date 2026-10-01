"use strict";

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

const osNames = { windows: "Windows", linux: "Linux", darwin: "macOS" };

async function loadInfo() {
  const el = document.getElementById("info");
  try {
    const i = await api("/api/info");
    el.textContent = `${i.hostname} · ${osNames[i.os] || i.os} (${i.arch}) · Version ${i.version}`;
  } catch (err) {
    el.textContent = "Keine Verbindung zur App";
    toast(`Fehler: ${err.message}`);
  }
}

document.getElementById("quit").addEventListener("click", async () => {
  try {
    await api("/api/quit", { method: "POST" });
    document.body.innerHTML =
      '<main style="padding-top:48px"><h1>Umzugskarton wurde beendet.</h1>' +
      "<p>Dieses Fenster kann geschlossen werden.</p></main>";
  } catch (err) {
    toast(`Beenden fehlgeschlagen: ${err.message}`);
  }
});

loadInfo();
