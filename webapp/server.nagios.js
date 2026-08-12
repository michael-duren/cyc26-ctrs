#!/usr/bin/env node
//
// nagios isolation monitor  --  presentation web app
//
// Same app as server.cyc.js, themed after nagios.org: dark console chrome,
// Nagios orange, and the classic OK / WARNING / CRITICAL service-status
// vocabulary. It reads the log written by scripts/evilnode.sh and maps each
// probe verdict onto a Nagios state:
//
//   ESCAPED   -> CRITICAL
//   LEAK      -> WARNING
//   CONTAINED -> OK
//
// Every boundary OK means the host is fully contained. Re-run the probe,
// refresh, done.
//
// Zero dependencies -- Node's built-in http only.  Node >= 18.

"use strict";

const http = require("http");
const fs = require("fs");

const PORT = process.env.PORT || 3000;

// If NODE_LOG is set we read EXACTLY that file (no fallback) so a stale
// log can never sneak in during the demo. Otherwise we try the probe's own
// default targets, most-privileged first.
const DEFAULT_LOGS = ["/var/log/node.log", "/tmp/node.log"];

// Nagios-style mark drawn inline so this theme needs no asset files.
const LOGO = `<svg viewBox="0 0 100 100" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="Nagios">
<rect x="2" y="2" width="96" height="96" rx="20" fill="#F58025"/>
<path d="M20 78 V22 h14 l32 40 V22 h14 v56 h-14 l-32-40 v40 z" fill="#ffffff"/>
</svg>`;

// ---------------------------------------------------------------------------
// log parsing
// ---------------------------------------------------------------------------

function pickLog() {
  const candidates = process.env.NODE_LOG
    ? [process.env.NODE_LOG]
    : DEFAULT_LOGS;
  for (const p of candidates) {
    try {
      fs.accessSync(p, fs.constants.R_OK);
      return p;
    } catch {
      /* try next */
    }
  }
  return null;
}

// Parse ONLY the most recent run. The probe uses `>>`, so the file accumulates
// runs; each run starts with a "# evilnode escape probe @ <stamp>" header.
function parseLog(text) {
  const lines = text.split(/\r?\n/);

  let start = 0;
  for (let i = 0; i < lines.length; i++) {
    if (lines[i].includes("evilnode escape probe @")) start = i;
  }
  const run = lines.slice(start);

  const stampRe = /evilnode escape probe @ (.+?)\s*$/;
  const secRe = /^==\s*(.+?)\s*==\s*$/;
  const verdictRe = /\[(ESCAPED|LEAK|CONTAINED|INFO)\]\s+(.*\S)\s*$/;

  let stamp = null;
  let section = null;
  let flag = null;
  const items = [];

  for (const line of run) {
    const sm = line.match(stampRe);
    if (sm) {
      stamp = sm[1].trim();
      continue;
    }

    const secm = line.match(secRe);
    if (secm) {
      const title = secm[1];
      const pm = title.match(/\(([^)]*)\)/);
      flag = pm ? pm[1].trim() : null;
      section = title.replace(/\s*\([^)]*\)\s*/, "").trim();
      continue;
    }

    const vm = line.match(verdictRe);
    if (vm && section && section.toLowerCase() !== "summary") {
      items.push({ section, flag, status: vm[1], message: vm[2].trim() });
    }
  }
  return { stamp, items };
}

function buildView() {
  const logPath = pickLog();
  if (!logPath) {
    return {
      state: "nodata",
      reason: "no-log",
      logPath: null,
      stamp: null,
      items: [],
    };
  }
  let text;
  try {
    text = fs.readFileSync(logPath, "utf8");
  } catch {
    return {
      state: "nodata",
      reason: "unreadable",
      logPath,
      stamp: null,
      items: [],
    };
  }

  const { stamp, items } = parseLog(text);
  if (items.length === 0) {
    return { state: "nodata", reason: "empty", logPath, stamp, items: [] };
  }

  const critical = items.filter((i) => i.status === "ESCAPED").length;
  const warning = items.filter((i) => i.status === "LEAK").length;
  const vulns = critical + warning;
  const state = vulns > 0 ? "vuln" : "secure";
  return {
    state,
    logPath,
    stamp,
    items,
    vulnCount: vulns,
    critical,
    warning,
    ok: items.length - vulns,
    total: items.length,
  };
}

// ---------------------------------------------------------------------------
// rendering
// ---------------------------------------------------------------------------

const esc = (s) =>
  String(s).replace(
    /[&<>"']/g,
    (c) =>
      ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[
        c
      ],
  );

// probe verdict -> nagios service state
const NAGIOS_STATE = {
  ESCAPED: "CRITICAL",
  LEAK: "WARNING",
  CONTAINED: "OK",
  INFO: "UNKNOWN",
};

const PAGE = (resultsHTML) => `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Nagios &middot; Container Isolation Monitor</title>
<style>
  :root{
    --bg:#12161a; --bg2:#0b0e11; --panel:#1b2127; --panel2:#232b33;
    --orange:#F58025; --orange2:#ffa04d; --line:#2c353f;
    --text:#eef2f6; --muted:#94a3b1;
    --crit:#d9382c; --warn:#f2c200; --ok:#3fa34d;
  }
  *{box-sizing:border-box}
  html,body{margin:0}
  body{
    font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,Inter,Helvetica,Arial,sans-serif;
    color:var(--text);
    background:linear-gradient(180deg, var(--bg) 0%, var(--bg2) 100%);
    background-attachment:fixed;
    min-height:100vh;
    -webkit-font-smoothing:antialiased;
  }
  .topbar{background:#0a0d10;border-bottom:3px solid var(--orange);
    padding:14px 24px;display:flex;align-items:center;gap:12px}
  .topbar svg{width:30px;height:30px}
  .topbar .word{font-weight:800;font-size:20px;letter-spacing:-.01em}
  .topbar .word span{color:var(--orange)}
  .topbar .nav{margin-left:auto;color:var(--muted);font-size:13px;
    letter-spacing:.12em;text-transform:uppercase}

  .wrap{max-width:1000px;margin:0 auto;padding:44px 24px 80px}
  .kicker{text-align:center;letter-spacing:.3em;text-transform:uppercase;
    font-size:12px;color:var(--orange);margin:0 0 18px;font-weight:700}
  .banner{text-align:center;font-weight:800;line-height:1.05;
    font-size:clamp(38px,8vw,86px);letter-spacing:-.02em;margin:0 0 10px;
    color:var(--text)}
  .banner .hl{color:var(--orange)}
  .sub{text-align:center;color:var(--muted);font-size:16px;margin:0 auto 34px;
    max-width:620px;line-height:1.5}

  .tally{display:flex;justify-content:center;gap:10px;margin:0 0 26px;flex-wrap:wrap}
  .tally .t{border:1px solid var(--line);background:var(--panel);border-radius:6px;
    padding:8px 14px;font:700 12px/1 ui-monospace,SFMono-Regular,Menlo,monospace;
    letter-spacing:.08em;display:flex;gap:8px;align-items:center}
  .tally .t b{font-size:14px}
  .tally .t.crit{color:var(--crit);border-color:rgba(217,56,44,.5)}
  .tally .t.warn{color:var(--warn);border-color:rgba(242,194,0,.5)}
  .tally .t.ok{color:var(--ok);border-color:rgba(63,163,77,.5)}

  .card{border:1px solid var(--line);border-radius:8px;overflow:hidden;
    background:var(--panel);box-shadow:0 20px 50px -30px rgba(0,0,0,.9)}
  .card-head{padding:22px 26px;border-bottom:1px solid var(--line);
    display:flex;align-items:center;gap:14px;background:var(--panel2)}
  .card-head .dot{width:12px;height:12px;border-radius:2px;flex:0 0 auto}
  .card-head h2{margin:0;font-size:clamp(20px,3vw,30px);font-weight:800;letter-spacing:-.01em}
  .card-head .sub2{margin:3px 0 0;font-size:14px;color:var(--muted)}

  .head-vuln{border-left:6px solid var(--crit)}
  .head-vuln .dot{background:var(--crit)}
  .head-secure{border-left:6px solid var(--ok)}
  .head-secure .dot{background:var(--ok)}
  .head-nodata{border-left:6px solid var(--muted)}
  .head-nodata .dot{background:var(--muted)}

  ul.vulns{list-style:none;margin:0;padding:0}
  ul.vulns li{display:flex;gap:14px;align-items:flex-start;padding:14px 20px}
  ul.vulns li:nth-child(even){background:rgba(255,255,255,.02)}
  ul.vulns li + li{border-top:1px solid rgba(255,255,255,.05)}
  .pill{flex:0 0 auto;min-width:82px;text-align:center;
    font:700 11px/1 ui-monospace,SFMono-Regular,Menlo,monospace;
    letter-spacing:.06em;padding:7px 8px;border-radius:3px;color:#0b0e11}
  .pill.ESCAPED{background:var(--crit);color:#fff}
  .pill.LEAK{background:var(--warn)}
  .pill.CONTAINED{background:var(--ok);color:#fff}
  .vbody{min-width:0}
  .vtitle{font-weight:700;font-size:15px;display:flex;gap:10px;align-items:baseline;flex-wrap:wrap}
  .flag{font:600 12px/1 ui-monospace,SFMono-Regular,Menlo,monospace;color:var(--orange2);
    background:rgba(245,128,37,.12);border:1px solid rgba(245,128,37,.35);
    padding:4px 7px;border-radius:3px}
  .vmsg{margin:6px 0 0;color:var(--muted);font-size:14px;line-height:1.5}

  .secure-body,.nodata-body{padding:26px;color:var(--muted);font-size:15px;line-height:1.6}
  .secure-body code,.nodata-body code{background:rgba(245,128,37,.12);
    border:1px solid rgba(245,128,37,.3);padding:2px 6px;border-radius:3px;
    font-family:ui-monospace,Menlo,monospace;color:var(--orange2)}
  .checks{display:grid;grid-template-columns:1fr 1fr;gap:8px;margin:18px 0 0}
  @media(max-width:560px){.checks{grid-template-columns:1fr}}
  .checks .ok{display:flex;gap:10px;align-items:center;font-size:14px;color:var(--text)}
  .checks .ok .tick{color:var(--ok);font-weight:800}

  .foot{margin-top:22px;text-align:center;color:var(--muted);font-size:12.5px}
  .foot code{font-family:ui-monospace,Menlo,monospace}
  .foot a{color:var(--orange2);text-decoration:none}
  .live{display:inline-flex;align-items:center;gap:7px;margin-left:8px}
  .live .beat{width:8px;height:8px;border-radius:50%;background:var(--ok);
    animation:beat 1.6s ease-in-out infinite}
  @keyframes beat{0%,100%{opacity:.35;transform:scale(.8)}50%{opacity:1;transform:scale(1.15)}}
</style>
</head>
<body>
  <div class="topbar">
    ${LOGO}
    <div class="word">Nag<span>ios</span></div>
    <div class="nav">Service Status Details</div>
  </div>
  <div class="wrap">
    <p class="kicker">Container Isolation Monitor</p>
    <h1 class="banner">hello, <span class="hl">Nagios</span>!</h1>
    <p class="sub">Free, powerful monitoring and alerting for servers, networks, applications and services.</p>

    <div id="tally"></div>
    <div id="results">${resultsHTML}</div>

    <div class="foot" id="foot"><span id="footmeta"></span><span class="live"><span class="beat"></span>live</span></div>
    <div class="foot"><a href="https://www.nagios.org/">nagios.org</a></div>
    <noscript><div class="foot">(live auto-refresh needs JS; reload the page after re-running the probe.)</div></noscript>
  </div>

<script>
const STATE = {ESCAPED:'CRITICAL', LEAK:'WARNING', CONTAINED:'OK', INFO:'UNKNOWN'};
let lastSig = null;
function render(v){
  // Only touch the DOM when the scan actually changed. Re-rendering every poll
  // would recreate the .beat node and restart its animation, making the live
  // dot snap back to small every few seconds.
  const sig = JSON.stringify(v);
  if(sig === lastSig) return;
  lastSig = sig;

  const tally = document.getElementById('tally');
  if(v.state === 'vuln' || v.state === 'secure'){
    tally.innerHTML = '<div class="tally">'
      + '<div class="t crit">CRITICAL <b>'+(v.critical||0)+'</b></div>'
      + '<div class="t warn">WARNING <b>'+(v.warning||0)+'</b></div>'
      + '<div class="t ok">OK <b>'+(v.ok||0)+'</b></div>'
      + '</div>';
  } else {
    tally.innerHTML = '';
  }

  const results = document.getElementById('results');
  if(v.state === 'vuln'){
    const items = v.items.filter(i => i.status==='ESCAPED' || i.status==='LEAK')
      .sort((a,b)=> (a.status==='ESCAPED'?0:1)-(b.status==='ESCAPED'?0:1));
    results.innerHTML =
      '<div class="card">'
      + '<div class="card-head head-vuln"><span class="dot"></span><div>'
      + '<h2>CRITICAL &mdash; YOU HAVE VULNERABILITIES</h2>'
      + '<p class="sub2">'+v.vulnCount+' of '+v.total+' isolation boundaries would let an attacker escape or leak host info.</p>'
      + '</div></div>'
      + '<ul class="vulns">'
      + items.map(function(i){
          return '<li><span class="pill '+i.status+'">'+STATE[i.status]+'</span>'
            + '<div class="vbody"><div class="vtitle">'+escapeHtml(i.section)
            + (i.flag ? ' <span class="flag">'+escapeHtml(i.flag)+'</span>' : '')
            + '</div><p class="vmsg">'+escapeHtml(i.message)+'</p></div></li>';
        }).join('')
      + '</ul></div>';
  } else if(v.state === 'secure'){
    results.innerHTML =
      '<div class="card">'
      + '<div class="card-head head-secure"><span class="dot"></span><div>'
      + '<h2>ALL SERVICES OK &#10003;</h2>'
      + '<p class="sub2">All '+v.total+' isolation boundaries are CONTAINED. Nice and locked down.</p>'
      + '</div></div>'
      + '<div class="secure-body">Every namespace held and the fork bomb is capped by the cgroup. '
      + 'Nothing in this container can see, touch, or take down the host.'
      + '<div class="checks">'
      + v.items.map(function(i){
          return '<div class="ok"><span class="tick">&#10003;</span>'+escapeHtml(i.section)
            + (i.flag ? ' <span class="flag">'+escapeHtml(i.flag)+'</span>' : '')+'</div>';
        }).join('')
      + '</div></div></div>';
  } else {
    results.innerHTML =
      '<div class="card"><div class="card-head head-nodata"><span class="dot"></span><div>'
      + '<h2>PENDING &mdash; no scan yet</h2>'
      + '</div></div><div class="nodata-body">'
      + '<code>'+escapeHtml(v.logPath || '/var/log/node.log')+'</code>'
      + '</div></div>';
  }
  // Update only the meta text; the persistent .beat node is left alone so its
  // animation keeps running smoothly.
  document.getElementById('footmeta').innerHTML =
    'log: <code>'+escapeHtml(v.logPath || '—')+'</code>'
    + (v.stamp ? ' &middot; scanned '+escapeHtml(v.stamp) : '') + ' ';
}
function escapeHtml(s){return String(s).replace(/[&<>"']/g,function(c){
  return ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'})[c];});}
async function poll(){
  try{ const r = await fetch('/api/status',{cache:'no-store'}); render(await r.json()); }
  catch(e){ /* keep last view */ }
}
poll();
setInterval(poll, 2500);
</script>
</body>
</html>`;

// server-side results (so the page is correct before JS runs)
function resultsHTML(v) {
  if (v.state === "vuln") {
    const items = v.items
      .filter((i) => i.status === "ESCAPED" || i.status === "LEAK")
      .sort(
        (a, b) =>
          (a.status === "ESCAPED" ? 0 : 1) - (b.status === "ESCAPED" ? 0 : 1),
      );
    return `<div class="card">
      <div class="card-head head-vuln"><span class="dot"></span><div>
        <h2>CRITICAL &mdash; YOU HAVE VULNERABILITIES</h2>
        <p class="sub2">${v.vulnCount} of ${v.total} isolation boundaries would let an attacker escape or leak host info.</p>
      </div></div>
      <ul class="vulns">${items
        .map(
          (i) => `<li>
        <span class="pill ${i.status}">${NAGIOS_STATE[i.status]}</span>
        <div class="vbody"><div class="vtitle">${esc(i.section)}${
          i.flag ? ` <span class="flag">${esc(i.flag)}</span>` : ""
        }</div><p class="vmsg">${esc(i.message)}</p></div></li>`,
        )
        .join("")}</ul></div>`;
  }
  if (v.state === "secure") {
    return `<div class="card">
      <div class="card-head head-secure"><span class="dot"></span><div>
        <h2>ALL SERVICES OK &#10003;</h2>
        <p class="sub2">All ${v.total} isolation boundaries are CONTAINED. Nice and locked down.</p>
      </div></div>
      <div class="secure-body">Every namespace held and the fork bomb is capped by the cgroup.
      Nothing in this container can see, touch, or take down the host.</div></div>`;
  }
  return `<div class="card"><div class="card-head head-nodata"><span class="dot"></span><div>
      <h2>PENDING &mdash; no scan yet</h2>
    </div></div><div class="nodata-body">
    <code>${esc(v.logPath || "/var/log/node.log")}</code>
    </div></div>`;
}

// ---------------------------------------------------------------------------
// server
// ---------------------------------------------------------------------------

const server = http.createServer((req, res) => {
  const url = req.url.split("?")[0];

  if (url === "/api/status") {
    res.writeHead(200, {
      "content-type": "application/json",
      "cache-control": "no-store",
    });
    res.end(JSON.stringify(buildView()));
    return;
  }
  if (url === "/logo.svg") {
    res.writeHead(200, { "content-type": "image/svg+xml" });
    res.end(LOGO);
    return;
  }
  if (url === "/" || url === "/index.html") {
    res.writeHead(200, {
      "content-type": "text/html; charset=utf-8",
      "cache-control": "no-store",
    });
    res.end(PAGE(resultsHTML(buildView())));
    return;
  }
  res.writeHead(404, { "content-type": "text/plain" });
  res.end("not found");
});

server.listen(PORT, () => {
  const v = buildView();
  console.log(`nagios isolation monitor  ->  http://localhost:${PORT}`);
  console.log(`  log source : ${v.logPath || "(none found yet)"}`);
  console.log(
    `  scan state : ${v.state}${v.state === "vuln" ? ` (${v.critical} CRITICAL / ${v.warning} WARNING of ${v.total})` : ""}`,
  );
  if (v.stamp) console.log(`  scanned at : ${v.stamp}`);
});
