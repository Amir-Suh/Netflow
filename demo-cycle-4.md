# NetFlow Cycle 4 Demo Runbook

## Goal

Demo Cycle 4 as the Visualization layer on top of Cycles 1, 2, and 3:

- Portfolio efficient-frontier scatter chart
- Wealth Monte Carlo fan chart (P10 / P50 / P90 bands + target line)
- Goal probability bar chart
- Debt payoff timeline (one line per strategy)
- WebSocket-driven auto-refresh: charts redraw the moment a quant job finishes

No backend, schema, or worker changes. The visualizations consume the existing `GET /quant/{kind}/runs/latest` payloads served by Cycle 3.

All outputs are educational projections only and are not financial advice.

## 1. Prep Environment

From the repo root:

```powershell
cd C:\Users\amirs\Desktop\Netflow\Netflow
```

Reuse the existing `.env` from Cycle 3 if it exists. Otherwise:

```powershell
Copy-Item .env.example .env
```

Required values:

```text
PLAID_ENV=sandbox
PLAID_CLIENT_ID=<your-plaid-sandbox-client-id>
PLAID_SECRET=<your-plaid-sandbox-secret>
NETFLOW_DEMO_SEED_ENABLED=true
```

`NETFLOW_DEMO_SEED_ENABLED=true` is local/demo-only.

## 2. Build And Start

```powershell
docker compose up --build -d
docker compose ps
```

Expected services (unchanged from Cycle 3):

- `api`
- `client`
- `rabbitmq`
- `postgres-primary`
- `postgres-replica`
- `categorizer-worker`
- `arbitrage-worker`
- `quant-worker`

Only the `client` image needs to rebuild for Cycle 4; the embedded HTML, CSS, and JS are compiled into the Go client binary.

## 3. Open The Client

```text
http://localhost:8081
```

What changed visually from Cycle 3:

- The right-hand column no longer leads with a dummy bar canvas + JSON `Latest Quant Results` block.
- Four new chart cards appear (Frontier, Wealth Fan, Goal Probability, Debt Timeline), each showing an empty-state message until a quant run completes.
- The Chart.js library loads from `cdn.jsdelivr.net` — confirm in DevTools → Network that `chart.umd.min.js` returns 200.

## 4. Register Or Login

In the Session panel: enter an email + password, click `Register` (or `Login` if the user already exists). The HTTP-only `auth_token` cookie carries over from Cycle 2; nothing here changed.

CLI parity for headless verification:

```powershell
$demoEmail = "viz@netflow.test"
$body = "{`"email`":`"$demoEmail`",`"password`":`"correcthorse`"}"
$response = Invoke-WebRequest -Uri http://localhost:8080/auth/register -Method POST -Body $body -ContentType "application/json" -SessionVariable nf
$response.Headers["Set-Cookie"]
```

If the user exists, swap `register` for `login` and recreate `$nf` so subsequent CLI calls share a session.

## 5. Seed Cycle 3 Demo Data

In the client, click `Seed Cycle 3`. This populates the debts, holdings, assumptions, and benchmark fixtures the quant worker reads. CLI:

```powershell
Invoke-WebRequest -Uri http://localhost:8080/demo/seed-cycle-3 -Method POST -WebSession $nf -Body '{}' -ContentType 'application/json'
```

Expected response body:

```json
{"status":"seeded","summary":{"debts":4,"holdings":4,"assumptions":4,"benchmark_return_rows":240}}
```

## 6. Portfolio Efficient Frontier

In the client, click `Portfolio`. Within ~2 seconds the **Portfolio Efficient Frontier** card should populate.

Expected:

- A green line of frontier points sweeping from low-volatility/low-return to high-volatility/high-return.
- A red diamond marker plotted at `(current_volatility, current_return)` for the user's current portfolio.
- Both axes labeled in percent.
- The chart-meta line below the canvas shows portfolio value, current return, current volatility, and Sharpe ratio.
- Hovering a point reveals `vol / ret / sharpe` in the tooltip.

CLI verification (no chart, but proves the data shape Chart.js consumes):

```powershell
Invoke-WebRequest -Uri http://localhost:8080/quant/portfolio/optimize -Method POST -WebSession $nf -Body '{"assumption_method":"demo_static","risk_tolerance":"moderate"}' -ContentType 'application/json'
Invoke-WebRequest -Uri http://localhost:8080/quant/portfolio/runs/latest -Method GET -WebSession $nf | Select-Object -ExpandProperty Content
```

Look for `frontier_points` (15 entries by default), `current_return`, and `current_volatility`.

## 7. Wealth Fan Chart + Goal Probability

Click `Wealth`. Within ~2 seconds two cards populate:

**Wealth Fan Chart**:

- Three lines spanning the 10y / 20y / 30y horizons.
- P10 (dark red) and P90 (green) lines flank a shaded band; P50 (dark teal) median sits inside.
- A grey dashed `Target` line marks the user's target wealth (from the quant profile).
- Y-axis formatted as currency. Tooltip shows full dollar amount per percentile.
- Meta line shows seed, scenario count, assumption method, and target.

**Goal Probability**:

- One bar per horizon, height = probability of reaching the target.
- Y-axis fixed to 0–100%.
- Bar color: green ≥ 70%, teal ≥ 40%, red below.

CLI:

```powershell
Invoke-WebRequest -Uri http://localhost:8080/quant/wealth/simulate -Method POST -WebSession $nf -Body '{"assumption_method":"demo_static"}' -ContentType 'application/json'
Invoke-WebRequest -Uri http://localhost:8080/quant/wealth/runs/latest -Method GET -WebSession $nf | Select-Object -ExpandProperty Content
```

Confirm `percentiles` contains rows for percentile 10, 50, 90 at each horizon, and `goal_probabilities` has one row per horizon.

## 8. Debt Payoff Timeline

Click `Debt`. Expected:

- One line per strategy (typically `minimum_only`, `avalanche`, `snowball`, plus the run's `recommended_strategy`).
- The recommended strategy renders with a thicker stroke (`borderWidth: 3`).
- All lines descend monotonically to zero by their respective payoff months.
- Tooltip shows the remaining balance per `(strategy, month)`.
- Meta line lists the recommended strategy, total interest, payoff months, plus a `· strategy: months / interest` summary per strategy.

CLI:

```powershell
Invoke-WebRequest -Uri http://localhost:8080/quant/debt/optimize -Method POST -WebSession $nf -Body '{"preference":"lowest_interest"}' -ContentType 'application/json'
Invoke-WebRequest -Uri http://localhost:8080/quant/debt/runs/latest -Method GET -WebSession $nf | Select-Object -ExpandProperty Content
```

## 9. Auto-Refresh Via WebSocket

The client opens a single WebSocket to `/api/ws` after auth and listens for `quant.{kind}.completed` events. Each event triggers a fetch of only the matching `/runs/latest` endpoint and a re-render of just that chart.

Demo the path without clicking the chart's button:

1. Open DevTools → Console.
2. From a PowerShell terminal that already has `$nf`:

```powershell
Invoke-WebRequest -Uri http://localhost:8080/quant/wealth/simulate -Method POST -WebSession $nf -Body '{"assumption_method":"capm"}' -ContentType 'application/json'
```

3. Watch the `Quant Events` panel append the `quant.wealth.completed` payload, and the Wealth Fan Chart re-render within ~2s.

Confirm in DevTools → Memory that triggering the same job multiple times does not grow the `Chart` retain count — each renderer calls `state.charts[kind].destroy()` before mounting a new instance.

## 10. Try Alternate Assumption Methods

The four charts re-render with any supported assumption method. Example:

```powershell
Invoke-WebRequest -Uri http://localhost:8080/quant/wealth/simulate -Method POST -WebSession $nf -Body '{"assumption_method":"historical_with_shrinkage"}' -ContentType 'application/json'
Invoke-WebRequest -Uri http://localhost:8080/quant/portfolio/optimize -Method POST -WebSession $nf -Body '{"assumption_method":"capm"}' -ContentType 'application/json'
Invoke-WebRequest -Uri http://localhost:8080/quant/portfolio/optimize -Method POST -WebSession $nf -Body '{"assumption_method":"black_litterman"}' -ContentType 'application/json'
```

Each completion triggers a chart refresh.

## 11. Empty-State Behavior

After `docker compose down -v` (wiping all data) and a fresh login but BEFORE seeding/running any quant job:

- Each chart card shows its empty-state message instead of rendering a chart.
- `Quant Events` is `[]`.
- `Response` panel shows the last raw API response (e.g., the auth payload).

This confirms the empty state is wired correctly and the charts gracefully ignore 404 responses from `/runs/latest`.

## 12. Tests

```powershell
docker run --rm -v "C:\Users\amirs\Desktop\Netflow\Netflow:/src" -w /src golang:1.25-alpine go test ./...
```

The embedded HTML/JS is a compile-time constant; the build proves the file is syntactically valid Go. The existing client and httpapi tests continue to pass — Cycle 4 does not touch handler logic.

## 13. Cleanup

```powershell
docker compose down
```

To remove local data:

```powershell
docker compose down -v
```

Only use `-v` when you are okay deleting local Postgres and RabbitMQ data.
