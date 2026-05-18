package client

const loginHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>NetFlow &mdash; Sign In</title>
  <style>
    :root {
      color-scheme: light;
      --ink: #18212f;
      --muted: #64748b;
      --line: #d8dee8;
      --surface: #ffffff;
      --accent: #126b60;
      --accent-strong: #0d4f47;
      --bad: #b91c1c;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      color: var(--ink);
      background: linear-gradient(135deg, #126b60 0%, #0d4f47 55%, #082e29 100%);
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 24px;
      visibility: hidden;
    }
    body.ready { visibility: visible; }
    .card {
      width: 100%;
      max-width: 420px;
      background: var(--surface);
      border-radius: 14px;
      padding: 36px;
      box-shadow: 0 24px 60px rgba(0, 0, 0, 0.28);
    }
    .brand { text-align: center; margin-bottom: 28px; }
    .brand h1 {
      margin: 0;
      font-size: 30px;
      font-weight: 800;
      letter-spacing: -0.02em;
      color: var(--accent);
    }
    .brand p {
      margin: 8px 0 0;
      font-size: 13px;
      color: var(--muted);
      font-weight: 650;
    }
    label {
      display: block;
      font-size: 12px;
      font-weight: 650;
      color: var(--muted);
      margin-bottom: 6px;
    }
    .field { margin-bottom: 16px; }
    input {
      width: 100%;
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 12px 14px;
      font: inherit;
      color: var(--ink);
      background: #ffffff;
    }
    input:focus {
      outline: 2px solid var(--accent);
      outline-offset: 1px;
      border-color: var(--accent);
    }
    .actions { display: grid; gap: 10px; margin-top: 22px; }
    button {
      min-height: 44px;
      border: 1px solid transparent;
      border-radius: 8px;
      padding: 10px 14px;
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
    button:disabled { cursor: not-allowed; opacity: 0.7; }
    .error {
      margin-top: 14px;
      padding: 10px 12px;
      border-radius: 8px;
      background: #fef2f2;
      border: 1px solid #fecaca;
      color: var(--bad);
      font-size: 13px;
      font-weight: 650;
      display: none;
    }
    .error.visible { display: block; }
    .note {
      margin: 18px 0 0;
      text-align: center;
      color: var(--muted);
      font-size: 12px;
      font-weight: 650;
    }
  </style>
</head>
<body>
  <div class="card">
    <div class="brand">
      <h1>NetFlow</h1>
      <p>Personal finance intelligence</p>
    </div>
    <div class="field">
      <label for="email">Email</label>
      <input id="email" type="email" autocomplete="username" value="sandbox@example.com">
    </div>
    <div class="field">
      <label for="password">Password</label>
      <input id="password" type="password" autocomplete="current-password" value="correct horse battery staple">
    </div>
    <div id="error" class="error"></div>
    <div class="actions">
      <button id="login">Sign In</button>
      <button class="secondary" id="register">Create Account</button>
    </div>
    <p class="note">Educational sandbox. Not for production use.</p>
  </div>
  <script>
    const els = {
      email: document.getElementById("email"),
      password: document.getElementById("password"),
      error: document.getElementById("error"),
      login: document.getElementById("login"),
      register: document.getElementById("register")
    };
    function showError(msg) {
      els.error.textContent = msg;
      els.error.classList.add("visible");
    }
    function clearError() {
      els.error.textContent = "";
      els.error.classList.remove("visible");
    }
    function setBusy(busy) {
      els.login.disabled = busy;
      els.register.disabled = busy;
    }
    async function submit(path) {
      const email = els.email.value.trim();
      const password = els.password.value;
      if (!email || !password) {
        showError("Email and password are required.");
        return;
      }
      clearError();
      setBusy(true);
      try {
        const response = await fetch("/api" + path, {
          method: "POST",
          headers: {"Content-Type": "application/json", "Accept": "application/json"},
          body: JSON.stringify({email: email, password: password})
        });
        const text = await response.text();
        let data = {};
        try { data = text ? JSON.parse(text) : {}; } catch (_) {}
        if (!response.ok) {
          showError(data.error || "Unable to sign in. Please try again.");
          return;
        }
        try { localStorage.setItem("netflow.email", email); } catch (_) {}
        window.location.href = "/";
      } catch (err) {
        showError("Network error. Please try again.");
      } finally {
        setBusy(false);
      }
    }
    els.login.addEventListener("click", () => submit("/auth/login"));
    els.register.addEventListener("click", () => submit("/auth/register"));
    [els.email, els.password].forEach(input => {
      input.addEventListener("keydown", e => {
        if (e.key === "Enter") submit("/auth/login");
      });
    });
    fetch("/api/quant/assumptions", {method: "GET"})
      .then(r => {
        if (r.ok) {
          window.location.href = "/";
        } else {
          document.body.classList.add("ready");
        }
      })
      .catch(() => document.body.classList.add("ready"));
  </script>
</body>
</html>`

const indexHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>NetFlow Client</title>
  <script src="https://cdn.plaid.com/link/v2/stable/link-initialize.js" async></script>
  <script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.6/dist/chart.umd.min.js" defer></script>
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
      --chart-a: #126b60;
      --chart-b: #2563eb;
      --chart-c: #9a3412;
      --chart-d: #15803d;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      color: var(--ink);
      background: var(--panel);
      visibility: hidden;
    }
    body.ready { visibility: visible; }
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
      color: var(--accent);
      letter-spacing: -0.01em;
    }
    .user-chip {
      display: inline-flex;
      align-items: center;
      padding: 6px 10px;
      border-radius: 999px;
      background: #eef4f2;
      color: var(--accent-strong);
      font-size: 12px;
      font-weight: 700;
      max-width: 220px;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
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
      min-height: 140px;
      max-height: 320px;
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
    .chart-card {
      position: relative;
      height: 280px;
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 8px;
      background: #ffffff;
    }
    .chart-card canvas { display: block; width: 100% !important; height: 100% !important; }
    .chart-empty {
      position: absolute;
      inset: 0;
      display: flex;
      align-items: center;
      justify-content: center;
      color: var(--muted);
      font-size: 13px;
      font-weight: 650;
      text-align: center;
      padding: 16px;
    }
    .chart-meta {
      margin: 6px 0 0;
      color: var(--muted);
      font-size: 12px;
      font-weight: 650;
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
    <h1>NetFlow</h1>
    <div class="toolbar">
      <div class="status-row"><span id="api-health-dot" class="dot"></span><span id="api-health">API</span></div>
      <div class="status-row"><span id="api-ready-dot" class="dot"></span><span id="api-ready">Ready</span></div>
      <span id="user-chip" class="user-chip" style="display:none"></span>
      <button class="secondary" id="refresh-status">Refresh</button>
      <button class="secondary" id="logout">Sign Out</button>
    </div>
  </header>
  <main>
    <div class="stack">
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
        <h2>Portfolio Efficient Frontier</h2>
        <div class="chart-card">
          <canvas id="chart-frontier"></canvas>
          <div id="chart-frontier-empty" class="chart-empty">Run a portfolio optimization to populate the frontier.</div>
        </div>
        <p id="chart-frontier-meta" class="chart-meta"></p>
      </section>
      <section>
        <h2>Wealth Fan Chart</h2>
        <div class="chart-card">
          <canvas id="chart-wealth"></canvas>
          <div id="chart-wealth-empty" class="chart-empty">Run a wealth simulation to populate percentile bands.</div>
        </div>
        <p id="chart-wealth-meta" class="chart-meta"></p>
      </section>
      <section>
        <h2>Goal Probability</h2>
        <div class="chart-card">
          <canvas id="chart-goal"></canvas>
          <div id="chart-goal-empty" class="chart-empty">Probability of reaching the target wealth per horizon.</div>
        </div>
      </section>
      <section>
        <h2>Debt Payoff Timeline</h2>
        <div class="chart-card">
          <canvas id="chart-debt"></canvas>
          <div id="chart-debt-empty" class="chart-empty">Run a debt optimization to compare payoff strategies.</div>
        </div>
        <p id="chart-debt-meta" class="chart-meta"></p>
      </section>
      <section>
        <h2>Response</h2>
        <pre id="output">{}</pre>
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
      charts: { frontier: null, wealth: null, goal: null, debt: null }
    };
    const els = {
      health: document.getElementById("api-health"),
      healthDot: document.getElementById("api-health-dot"),
      ready: document.getElementById("api-ready"),
      readyDot: document.getElementById("api-ready-dot"),
      publicToken: document.getElementById("public-token"),
      itemID: document.getElementById("item-id"),
      output: document.getElementById("output"),
      quantEvents: document.getElementById("quant-events"),
      openLink: document.getElementById("open-link"),
      userChip: document.getElementById("user-chip")
    };

    function setOutput(value) {
      els.output.textContent = JSON.stringify(value, null, 2);
    }

    function setIndicator(dot, label, ok, text) {
      dot.className = ok ? "dot ok" : "dot bad";
      label.textContent = text;
    }

    function setEmpty(key, visible, message) {
      const el = document.getElementById("chart-" + key + "-empty");
      if (!el) return;
      el.style.display = visible ? "flex" : "none";
      if (visible && message) el.textContent = message;
    }

    function setMeta(key, text) {
      const el = document.getElementById("chart-" + key + "-meta");
      if (el) el.textContent = text || "";
    }

    function destroyChart(key) {
      if (state.charts[key]) {
        state.charts[key].destroy();
        state.charts[key] = null;
      }
    }

    function formatCurrency(value) {
      if (value == null || isNaN(value)) return "$0";
      return "$" + Math.round(value).toLocaleString();
    }

    function formatPercent(value, digits) {
      if (value == null || isNaN(value)) return "0%";
      const d = digits == null ? 1 : digits;
      return (value * 100).toFixed(d) + "%";
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

    async function logout() {
      try { await api("/auth/logout", {method: "POST"}); } catch (_) {}
      try { localStorage.removeItem("netflow.email"); } catch (_) {}
      window.location.href = "/login";
    }

    function renderUserChip() {
      try {
        const email = localStorage.getItem("netflow.email");
        if (email) {
          els.userChip.textContent = email;
          els.userChip.style.display = "inline-flex";
        }
      } catch (_) {}
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
      setTimeout(() => refreshQuantKind("debt").catch(() => {}), 3500);
    }

    async function runWealth() {
      const data = await api("/quant/wealth/simulate", {method: "POST", body: JSON.stringify({assumption_method: "demo_static"})});
      setOutput(data);
      setTimeout(() => refreshQuantKind("wealth").catch(() => {}), 3500);
    }

    async function runPortfolio() {
      const data = await api("/quant/portfolio/optimize", {method: "POST", body: JSON.stringify({assumption_method: "demo_static", risk_tolerance: "moderate"})});
      setOutput(data);
      setTimeout(() => refreshQuantKind("portfolio").catch(() => {}), 3500);
    }

    function renderFrontierChart(details) {
      if (typeof Chart === "undefined") return;
      if (!details || !details.run || !Array.isArray(details.frontier_points) || details.frontier_points.length === 0) {
        destroyChart("frontier");
        setEmpty("frontier", true);
        setMeta("frontier", "");
        return;
      }
      const points = details.frontier_points
        .map(p => ({x: p.volatility, y: p.expected_return, sharpe: p.sharpe}))
        .sort((a, b) => a.x - b.x);
      const current = {
        x: details.run.current_volatility,
        y: details.run.current_return
      };
      destroyChart("frontier");
      setEmpty("frontier", false);
      const ctx = document.getElementById("chart-frontier").getContext("2d");
      state.charts.frontier = new Chart(ctx, {
        type: "line",
        data: {
          datasets: [
            {
              label: "Efficient Frontier",
              data: points,
              showLine: true,
              borderColor: "#126b60",
              backgroundColor: "#126b60",
              pointRadius: 4,
              tension: 0.2
            },
            {
              label: "Current Portfolio",
              data: [current],
              showLine: false,
              borderColor: "#b91c1c",
              backgroundColor: "#b91c1c",
              pointRadius: 9,
              pointStyle: "rectRot"
            }
          ]
        },
        options: {
          responsive: true,
          maintainAspectRatio: false,
          parsing: false,
          scales: {
            x: {
              type: "linear",
              title: { display: true, text: "Volatility (annualized)" },
              ticks: { callback: v => formatPercent(v) }
            },
            y: {
              title: { display: true, text: "Expected Return (annualized)" },
              ticks: { callback: v => formatPercent(v) }
            }
          },
          plugins: {
            legend: { position: "bottom" },
            tooltip: {
              callbacks: {
                label: ctx => {
                  const pt = ctx.raw;
                  const base = ctx.dataset.label + ": vol " + formatPercent(pt.x) + ", ret " + formatPercent(pt.y);
                  if (pt.sharpe != null) return base + ", sharpe " + pt.sharpe.toFixed(2);
                  return base;
                }
              }
            }
          }
        }
      });
      setMeta("frontier",
        "Portfolio value " + formatCurrency(details.run.portfolio_value) +
        " | current return " + formatPercent(details.run.current_return) +
        " | current vol " + formatPercent(details.run.current_volatility) +
        " | sharpe " + (details.run.current_sharpe || 0).toFixed(2));
    }

    function renderWealthFanChart(details) {
      if (typeof Chart === "undefined") return;
      if (!details || !Array.isArray(details.percentiles) || details.percentiles.length === 0) {
        destroyChart("wealth");
        setEmpty("wealth", true);
        setMeta("wealth", "");
        return;
      }
      const byPercentile = {};
      details.percentiles.forEach(p => {
        if (!byPercentile[p.percentile]) byPercentile[p.percentile] = [];
        byPercentile[p.percentile].push({horizon: p.horizon_years, value: p.nominal_value});
      });
      Object.values(byPercentile).forEach(arr => arr.sort((a, b) => a.horizon - b.horizon));
      const horizons = (byPercentile[50] || byPercentile[10] || byPercentile[90] || []).map(p => p.horizon);
      const target = details.run && details.run.target_wealth;
      const datasets = [];
      if (byPercentile[10]) datasets.push({
        label: "P10",
        data: byPercentile[10].map(p => p.value),
        borderColor: "#9a3412",
        backgroundColor: "rgba(154, 52, 18, 0.12)",
        fill: "+2",
        tension: 0.2,
        pointRadius: 2
      });
      if (byPercentile[50]) datasets.push({
        label: "P50 (median)",
        data: byPercentile[50].map(p => p.value),
        borderColor: "#0d4f47",
        backgroundColor: "#0d4f47",
        borderWidth: 2,
        tension: 0.2,
        pointRadius: 3
      });
      if (byPercentile[90]) datasets.push({
        label: "P90",
        data: byPercentile[90].map(p => p.value),
        borderColor: "#15803d",
        backgroundColor: "#15803d",
        tension: 0.2,
        pointRadius: 2
      });
      if (target && target > 0 && horizons.length > 0) {
        datasets.push({
          label: "Target",
          data: horizons.map(() => target),
          borderColor: "#64748b",
          borderDash: [6, 4],
          pointRadius: 0,
          fill: false
        });
      }
      destroyChart("wealth");
      setEmpty("wealth", false);
      const ctx = document.getElementById("chart-wealth").getContext("2d");
      state.charts.wealth = new Chart(ctx, {
        type: "line",
        data: { labels: horizons.map(h => h + "y"), datasets },
        options: {
          responsive: true,
          maintainAspectRatio: false,
          scales: {
            x: { title: { display: true, text: "Horizon (years)" } },
            y: {
              title: { display: true, text: "Projected Wealth ($)" },
              ticks: { callback: v => formatCurrency(v) }
            }
          },
          plugins: {
            legend: { position: "bottom" },
            tooltip: {
              callbacks: { label: ctx => ctx.dataset.label + ": " + formatCurrency(ctx.raw) }
            }
          }
        }
      });
      setMeta("wealth",
        "Seed " + (details.run && details.run.seed) +
        " | scenarios " + (details.run && details.run.scenario_count) +
        " | method " + (details.run && details.run.assumption_method) +
        " | target " + formatCurrency(target));
    }

    function renderGoalChart(details) {
      if (typeof Chart === "undefined") return;
      if (!details || !Array.isArray(details.goal_probabilities) || details.goal_probabilities.length === 0) {
        destroyChart("goal");
        setEmpty("goal", true);
        return;
      }
      const data = [...details.goal_probabilities].sort((a, b) => a.horizon_years - b.horizon_years);
      destroyChart("goal");
      setEmpty("goal", false);
      const ctx = document.getElementById("chart-goal").getContext("2d");
      state.charts.goal = new Chart(ctx, {
        type: "bar",
        data: {
          labels: data.map(d => d.horizon_years + "y"),
          datasets: [{
            label: "Probability of reaching target",
            data: data.map(d => d.probability),
            backgroundColor: data.map(d =>
              d.probability >= 0.7 ? "#15803d" :
              d.probability >= 0.4 ? "#126b60" : "#9a3412")
          }]
        },
        options: {
          responsive: true,
          maintainAspectRatio: false,
          scales: {
            y: {
              min: 0,
              max: 1,
              ticks: { callback: v => Math.round(v * 100) + "%" }
            }
          },
          plugins: {
            legend: { display: false },
            tooltip: {
              callbacks: {
                label: ctx => "Probability: " + formatPercent(ctx.raw, 1) +
                  " | target " + formatCurrency(data[ctx.dataIndex].target_wealth)
              }
            }
          }
        }
      });
    }

    function renderDebtTimelineChart(details) {
      if (typeof Chart === "undefined") return;
      if (!details || !Array.isArray(details.steps) || details.steps.length === 0) {
        destroyChart("debt");
        setEmpty("debt", true);
        setMeta("debt", "");
        return;
      }
      const byStrategy = {};
      details.steps.forEach(s => {
        if (!byStrategy[s.strategy]) byStrategy[s.strategy] = {};
        byStrategy[s.strategy][s.month_index] = (byStrategy[s.strategy][s.month_index] || 0) + (s.ending_balance || 0);
      });
      const monthSet = new Set();
      Object.values(byStrategy).forEach(mp => Object.keys(mp).forEach(k => monthSet.add(parseInt(k, 10))));
      const months = [...monthSet].sort((a, b) => a - b);
      const palette = ["#126b60", "#2563eb", "#9a3412", "#15803d", "#7c2d12"];
      const strategies = Object.keys(byStrategy).sort();
      const datasets = strategies.map((strategy, i) => ({
        label: strategy,
        data: months.map(m => byStrategy[strategy][m] != null ? byStrategy[strategy][m] : null),
        borderColor: palette[i % palette.length],
        backgroundColor: palette[i % palette.length],
        tension: 0.1,
        pointRadius: 0,
        spanGaps: true,
        borderWidth: strategy === details.run.recommended_strategy ? 3 : 2
      }));
      destroyChart("debt");
      setEmpty("debt", false);
      const ctx = document.getElementById("chart-debt").getContext("2d");
      state.charts.debt = new Chart(ctx, {
        type: "line",
        data: { labels: months.map(m => "M" + m), datasets },
        options: {
          responsive: true,
          maintainAspectRatio: false,
          scales: {
            x: { title: { display: true, text: "Month" } },
            y: {
              title: { display: true, text: "Remaining Balance ($)" },
              ticks: { callback: v => formatCurrency(v) }
            }
          },
          plugins: {
            legend: { position: "bottom" },
            tooltip: {
              callbacks: { label: ctx => ctx.dataset.label + ": " + formatCurrency(ctx.raw) }
            }
          }
        }
      });
      const summaryParts = (details.strategy_summaries || []).map(s =>
        s.strategy + ": " + s.payoff_months + "mo / " + formatCurrency(s.total_interest));
      setMeta("debt",
        "Recommended: " + (details.run.recommended_strategy || "n/a") +
        " | total interest " + formatCurrency(details.run.total_interest) +
        " | payoff " + (details.run.payoff_months || 0) + "mo" +
        (summaryParts.length ? " || " + summaryParts.join(" · ") : ""));
    }

    const quantRefreshMap = {
      debt: { path: "/quant/debt/runs/latest", render: renderDebtTimelineChart },
      portfolio: { path: "/quant/portfolio/runs/latest", render: renderFrontierChart },
      wealth: {
        path: "/quant/wealth/runs/latest",
        render: details => { renderWealthFanChart(details); renderGoalChart(details); }
      }
    };

    async function refreshQuantKind(kind) {
      const entry = quantRefreshMap[kind];
      if (!entry) return;
      try {
        const data = await api(entry.path, {method: "GET"});
        entry.render(data);
      } catch (err) {
        if (err.status === 404) {
          if (kind === "wealth") { renderWealthFanChart(null); renderGoalChart(null); }
          else if (kind === "portfolio") renderFrontierChart(null);
          else if (kind === "debt") renderDebtTimelineChart(null);
          return;
        }
        setOutput(err.response || {error: err.message});
      }
    }

    async function refreshQuant() {
      await Promise.all(Object.keys(quantRefreshMap).map(kind => refreshQuantKind(kind)));
    }

    function connectWebSocket() {
      if (state.ws && state.ws.readyState < 2) return;
      const scheme = window.location.protocol === "https:" ? "wss://" : "ws://";
      state.ws = new WebSocket(scheme + window.location.host + "/api/ws");
      state.ws.onmessage = (event) => {
        const data = JSON.parse(event.data);
        const evt = data.event_type || "";
        if (!evt.startsWith("quant.")) return;
        state.quantEvents.unshift(data);
        state.quantEvents = state.quantEvents.slice(0, 12);
        els.quantEvents.textContent = JSON.stringify(state.quantEvents, null, 2);
        const parts = evt.split(".");
        if (parts.length >= 2) refreshQuantKind(parts[1]).catch(() => {});
      };
    }

    document.getElementById("refresh-status").addEventListener("click", refreshStatus);
    document.getElementById("logout").addEventListener("click", () => logout());
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
    async function initDashboard() {
      try {
        const probe = await fetch("/api/quant/assumptions", {method: "GET"});
        if (probe.status === 401 || probe.status === 403) {
          window.location.href = "/login";
          return;
        }
      } catch (_) {
        // network error — let user retry; don't redirect
      }
      document.body.classList.add("ready");
      renderUserChip();
      updateLinkButton();
      refreshStatus();
      refreshQuant().catch(() => {});
      connectWebSocket();
    }
    window.addEventListener("load", initDashboard);
    setInterval(updateLinkButton, 800);
  </script>
</body>
</html>`
