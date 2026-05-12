package client

const indexHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>NetFlow Client</title>
  <script src="https://cdn.plaid.com/link/v2/stable/link-initialize.js" async></script>
  <style>
    :root {
      color-scheme: light;
      --ink: #18212f;
      --muted: #64748b;
      --line: #d8dee8;
      --surface: #ffffff;
      --panel: #f7f9fc;
      --accent: #126b60;
      --accent-strong: #0d4f47;
      --warn: #9a3412;
      --bad: #b91c1c;
      --ok: #15803d;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      color: var(--ink);
      background: var(--panel);
    }
    header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 16px;
      padding: 18px 24px;
      background: var(--surface);
      border-bottom: 1px solid var(--line);
    }
    h1 {
      margin: 0;
      font-size: 20px;
      line-height: 1.2;
      font-weight: 700;
    }
    main {
      display: grid;
      grid-template-columns: minmax(280px, 380px) minmax(0, 1fr);
      gap: 18px;
      padding: 18px;
      max-width: 1240px;
      margin: 0 auto;
    }
    section {
      background: var(--surface);
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 16px;
    }
    .stack { display: grid; gap: 14px; }
    .toolbar {
      display: flex;
      align-items: center;
      gap: 10px;
      flex-wrap: wrap;
    }
    h2 {
      margin: 0 0 12px;
      font-size: 15px;
      line-height: 1.3;
    }
    label {
      display: grid;
      gap: 6px;
      color: var(--muted);
      font-size: 12px;
      font-weight: 650;
    }
    input, textarea {
      width: 100%;
      border: 1px solid var(--line);
      border-radius: 6px;
      padding: 10px 11px;
      font: inherit;
      color: var(--ink);
      background: #ffffff;
    }
    textarea {
      min-height: 92px;
      resize: vertical;
      word-break: break-word;
    }
    button {
      min-height: 38px;
      border: 1px solid transparent;
      border-radius: 6px;
      padding: 8px 12px;
      font: inherit;
      font-weight: 700;
      cursor: pointer;
      color: #ffffff;
      background: var(--accent);
    }
    button:hover { background: var(--accent-strong); }
    button.secondary {
      color: var(--ink);
      background: #ffffff;
      border-color: var(--line);
    }
    button.secondary:hover { background: #eef4f2; }
    button:disabled {
      cursor: not-allowed;
      color: #94a3b8;
      background: #e8edf4;
      border-color: #e8edf4;
    }
    .status-row {
      display: flex;
      align-items: center;
      gap: 8px;
      color: var(--muted);
      font-size: 13px;
      font-weight: 650;
    }
    .dot {
      width: 10px;
      height: 10px;
      border-radius: 999px;
      background: var(--warn);
      flex: 0 0 auto;
    }
    .dot.ok { background: var(--ok); }
    .dot.bad { background: var(--bad); }
    .note {
      margin: 0;
      color: var(--muted);
      font-size: 12px;
      line-height: 1.45;
      font-weight: 650;
    }
    pre {
      margin: 0;
      min-height: 220px;
      max-height: 520px;
      overflow: auto;
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 14px;
      background: #101923;
      color: #dceaf7;
      font-size: 13px;
      line-height: 1.45;
      white-space: pre-wrap;
      word-break: break-word;
    }
    canvas {
      width: 100%;
      height: 86px;
      display: block;
      border: 1px solid var(--line);
      border-radius: 8px;
      background: #ffffff;
    }
    @media (max-width: 760px) {
      header { align-items: flex-start; flex-direction: column; }
      main { grid-template-columns: 1fr; padding: 12px; }
      .toolbar { align-items: stretch; }
      button { width: 100%; }
    }
  </style>
</head>
<body>
  <header>
    <h1>NetFlow Client</h1>
    <div class="toolbar">
      <div class="status-row"><span id="api-health-dot" class="dot"></span><span id="api-health">API</span></div>
      <div class="status-row"><span id="api-ready-dot" class="dot"></span><span id="api-ready">Ready</span></div>
      <button class="secondary" id="refresh-status">Refresh</button>
    </div>
  </header>
  <main>
    <div class="stack">
      <section>
        <h2>Session</h2>
        <div class="stack">
          <label>Email
            <input id="email" type="email" autocomplete="username" value="sandbox@example.com">
          </label>
          <label>Password
            <input id="password" type="password" autocomplete="current-password" value="correct horse battery staple">
          </label>
          <div class="toolbar">
            <button id="register">Register</button>
            <button class="secondary" id="login">Login</button>
          </div>
        </div>
      </section>
      <section>
        <h2>Plaid Sandbox</h2>
        <div class="stack">
          <div class="toolbar">
            <button id="create-link-token">Link token</button>
            <button class="secondary" id="open-link" disabled>Open Link</button>
          </div>
          <label>Public token
            <input id="public-token" spellcheck="false">
          </label>
          <button id="exchange-token">Exchange token</button>
          <label>Item ID
            <input id="item-id" spellcheck="false">
          </label>
          <div class="toolbar">
            <button id="sync-item">Sync</button>
            <button class="secondary" id="mock-webhook">Mock webhook</button>
          </div>
        </div>
      </section>
      <section>
        <h2>Quant Engine</h2>
        <div class="stack">
          <p class="note">Educational projections only. Not financial advice.</p>
          <button id="seed-cycle-3">Seed Cycle 3</button>
          <div class="toolbar">
            <button id="run-debt">Debt</button>
            <button id="run-wealth">Wealth</button>
            <button id="run-portfolio">Portfolio</button>
          </div>
          <button class="secondary" id="refresh-quant">Refresh Results</button>
        </div>
      </section>
    </div>
    <div class="stack">
      <section>
        <h2>Activity</h2>
        <canvas id="activity-canvas" width="900" height="172"></canvas>
      </section>
      <section>
        <h2>Response</h2>
        <pre id="output">{}</pre>
      </section>
      <section>
        <h2>Latest Quant Results</h2>
        <pre id="quant-summary">{}</pre>
      </section>
      <section>
        <h2>Quant Events</h2>
        <pre id="quant-events">[]</pre>
      </section>
    </div>
  </main>
  <script>
    const state = {
      linkToken: "",
      linkHandler: null,
      ws: null,
      quantEvents: [],
      samples: [12, 28, 18, 45, 32, 66, 42, 74, 52, 88, 64, 96]
    };
    const els = {
      health: document.getElementById("api-health"),
      healthDot: document.getElementById("api-health-dot"),
      ready: document.getElementById("api-ready"),
      readyDot: document.getElementById("api-ready-dot"),
      email: document.getElementById("email"),
      password: document.getElementById("password"),
      publicToken: document.getElementById("public-token"),
      itemID: document.getElementById("item-id"),
      output: document.getElementById("output"),
      quantSummary: document.getElementById("quant-summary"),
      quantEvents: document.getElementById("quant-events"),
      openLink: document.getElementById("open-link"),
      canvas: document.getElementById("activity-canvas")
    };

    function setOutput(value) {
      els.output.textContent = JSON.stringify(value, null, 2);
    }

    function setQuantSummary(value) {
      els.quantSummary.textContent = JSON.stringify(value, null, 2);
    }

    function setIndicator(dot, label, ok, text) {
      dot.className = ok ? "dot ok" : "dot bad";
      label.textContent = text;
    }

    async function api(path, options = {}) {
      const headers = Object.assign({"Accept": "application/json"}, options.headers || {});
      if (options.body && !headers["Content-Type"]) {
        headers["Content-Type"] = "application/json";
      }
      const response = await fetch("/api" + path, Object.assign({}, options, {headers}));
      const text = await response.text();
      let data = {};
      if (text) {
        try { data = JSON.parse(text); } catch (_) { data = {body: text}; }
      }
      if (!response.ok) {
        throw Object.assign(new Error(data.error || response.statusText), {response: data, status: response.status});
      }
      return data;
    }

    async function refreshStatus() {
      try {
        await api("/healthz", {method: "GET", headers: {}});
        setIndicator(els.healthDot, els.health, true, "API online");
      } catch (err) {
        setIndicator(els.healthDot, els.health, false, "API offline");
      }
      try {
        await api("/readyz", {method: "GET", headers: {}});
        setIndicator(els.readyDot, els.ready, true, "Backend ready");
      } catch (err) {
        setIndicator(els.readyDot, els.ready, false, "Backend waiting");
      }
    }

    async function auth(path) {
      const data = await api(path, {
        method: "POST",
        body: JSON.stringify({email: els.email.value.trim(), password: els.password.value})
      });
      setOutput(data);
      connectWebSocket();
    }

    async function createLinkToken() {
      const data = await api("/plaid/link-token", {method: "POST", body: "{}"});
      state.linkToken = data.link_token || "";
      setOutput(data);
      updateLinkButton();
    }

    function updateLinkButton() {
      els.openLink.disabled = !(state.linkToken && window.Plaid);
    }

    function openLink() {
      if (!state.linkToken || !window.Plaid) return;
      state.linkHandler = window.Plaid.create({
        token: state.linkToken,
        onSuccess: async function(publicToken, metadata) {
          els.publicToken.value = publicToken;
          setOutput({public_token: publicToken, metadata});
        },
        onExit: function(error, metadata) {
          setOutput({error, metadata});
        }
      });
      state.linkHandler.open();
    }

    async function exchangeToken() {
      const data = await api("/plaid/exchange-public-token", {
        method: "POST",
        body: JSON.stringify({public_token: els.publicToken.value.trim()})
      });
      els.itemID.value = data.item_id || "";
      setOutput(data);
    }

    async function syncItem() {
      const data = await api("/plaid/sync", {
        method: "POST",
        body: JSON.stringify({item_id: els.itemID.value.trim()})
      });
      setOutput(data);
    }

    async function mockWebhook() {
      const data = await api("/plaid/webhook", {
        method: "POST",
        body: JSON.stringify({
          webhook_type: "TRANSACTIONS",
          webhook_code: "SYNC_UPDATES_AVAILABLE",
          item_id: els.itemID.value.trim(),
          environment: "sandbox"
        })
      });
      setOutput(data);
    }

    async function seedCycle3() {
      const data = await api("/demo/seed-cycle-3", {method: "POST", body: "{}"});
      setOutput(data);
      await refreshQuant();
    }

    async function runDebt() {
      const data = await api("/quant/debt/optimize", {method: "POST", body: JSON.stringify({preference: "lowest_interest"})});
      setOutput(data);
    }

    async function runWealth() {
      const data = await api("/quant/wealth/simulate", {method: "POST", body: JSON.stringify({assumption_method: "demo_static"})});
      setOutput(data);
    }

    async function runPortfolio() {
      const data = await api("/quant/portfolio/optimize", {method: "POST", body: JSON.stringify({assumption_method: "demo_static"})});
      setOutput(data);
    }

    async function refreshQuant() {
      const result = {};
      for (const item of [
        ["debt", "/quant/debt/runs/latest"],
        ["wealth", "/quant/wealth/runs/latest"],
        ["portfolio", "/quant/portfolio/runs/latest"]
      ]) {
        try {
          result[item[0]] = await api(item[1], {method: "GET"});
        } catch (err) {
          result[item[0]] = err.response || {error: err.message};
        }
      }
      setQuantSummary(result);
    }

    function connectWebSocket() {
      if (state.ws && state.ws.readyState < 2) return;
      const scheme = window.location.protocol === "https:" ? "wss://" : "ws://";
      state.ws = new WebSocket(scheme + window.location.host + "/api/ws");
      state.ws.onmessage = (event) => {
        const data = JSON.parse(event.data);
        if ((data.event_type || "").startsWith("quant.")) {
          state.quantEvents.unshift(data);
          state.quantEvents = state.quantEvents.slice(0, 12);
          els.quantEvents.textContent = JSON.stringify(state.quantEvents, null, 2);
          refreshQuant().catch(() => {});
        }
      };
    }

    function drawActivity() {
      const ctx = els.canvas.getContext("2d");
      const w = els.canvas.width;
      const h = els.canvas.height;
      ctx.clearRect(0, 0, w, h);
      ctx.fillStyle = "#f8fafc";
      ctx.fillRect(0, 0, w, h);
      ctx.strokeStyle = "#d8dee8";
      ctx.lineWidth = 1;
      for (let i = 1; i < 4; i++) {
        const y = Math.round((h / 4) * i);
        ctx.beginPath();
        ctx.moveTo(0, y);
        ctx.lineTo(w, y);
        ctx.stroke();
      }
      const gap = 16;
      const barWidth = Math.max(18, (w - gap * (state.samples.length + 1)) / state.samples.length);
      state.samples.forEach((value, index) => {
        const x = gap + index * (barWidth + gap);
        const barHeight = Math.round((h - 36) * (value / 100));
        const y = h - barHeight - 18;
        ctx.fillStyle = index % 3 === 0 ? "#126b60" : index % 3 === 1 ? "#2563eb" : "#7c2d12";
        ctx.fillRect(x, y, barWidth, barHeight);
      });
    }

    document.getElementById("refresh-status").addEventListener("click", refreshStatus);
    document.getElementById("register").addEventListener("click", () => auth("/auth/register").catch(err => setOutput(err.response || {error: err.message})));
    document.getElementById("login").addEventListener("click", () => auth("/auth/login").catch(err => setOutput(err.response || {error: err.message})));
    document.getElementById("create-link-token").addEventListener("click", () => createLinkToken().catch(err => setOutput(err.response || {error: err.message})));
    document.getElementById("open-link").addEventListener("click", openLink);
    document.getElementById("exchange-token").addEventListener("click", () => exchangeToken().catch(err => setOutput(err.response || {error: err.message})));
    document.getElementById("sync-item").addEventListener("click", () => syncItem().catch(err => setOutput(err.response || {error: err.message})));
    document.getElementById("mock-webhook").addEventListener("click", () => mockWebhook().catch(err => setOutput(err.response || {error: err.message})));
    document.getElementById("seed-cycle-3").addEventListener("click", () => seedCycle3().catch(err => setOutput(err.response || {error: err.message})));
    document.getElementById("run-debt").addEventListener("click", () => runDebt().catch(err => setOutput(err.response || {error: err.message})));
    document.getElementById("run-wealth").addEventListener("click", () => runWealth().catch(err => setOutput(err.response || {error: err.message})));
    document.getElementById("run-portfolio").addEventListener("click", () => runPortfolio().catch(err => setOutput(err.response || {error: err.message})));
    document.getElementById("refresh-quant").addEventListener("click", () => refreshQuant().catch(err => setOutput(err.response || {error: err.message})));
    window.addEventListener("load", () => { updateLinkButton(); refreshStatus(); refreshQuant(); drawActivity(); connectWebSocket(); });
    window.addEventListener("resize", drawActivity);
    setInterval(updateLinkButton, 800);
  </script>
</body>
</html>`
