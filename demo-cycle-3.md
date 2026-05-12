# NetFlow Cycle 3 Demo Runbook

## Goal

Demo Cycle 3 as the Quant Engine layered on top of Cycles 1 and 2:

- Debt payoff optimization
- Monte Carlo wealth simulation
- Portfolio optimization with efficient-frontier output
- Deterministic local demo seed data
- Persisted assumptions for every run
- RabbitMQ async jobs and WebSocket completion alerts

All outputs are educational projections only and are not financial advice.

## 1. Prep Environment

From the repo root:

```powershell
cd C:\Users\fredy\Projects\NetFlow
```

Copy `.env.example` if needed:

```powershell
Copy-Item .env.example .env
```

For Plaid Sandbox, set:

```text
PLAID_ENV=sandbox
PLAID_CLIENT_ID=<your-plaid-sandbox-client-id>
PLAID_SECRET=<your-plaid-sandbox-secret>
NETFLOW_DEMO_SEED_ENABLED=true
```

`NETFLOW_DEMO_SEED_ENABLED=true` is local/demo-only. Keep it disabled outside sandbox/demo environments.

## 2. Build And Start

```powershell
docker compose up --build -d
docker compose ps
```

Expected services:

- `api`
- `client`
- `rabbitmq`
- `postgres-primary`
- `postgres-replica`
- `categorizer-worker`
- `arbitrage-worker`
- `quant-worker`

Check the new worker:

```powershell
docker compose logs quant-worker --tail=80
```

Expected: Celery ready plus `quant.worker :: listening on quant request queues`.

## 3. Register Or Login

Open:

```text
http://localhost:8081
```

Use the Session panel to register or login. Auth uses the existing HTTP-only `auth_token` cookie flow; JWTs are not returned in JSON bodies.

Browser auth and PowerShell auth are separate. If you want to use the CLI commands below, create a PowerShell web session first:

```powershell
$demoEmail = "quant@netflow.test"
$body = "{`"email`":`"$demoEmail`",`"password`":`"correcthorse`"}"
$response = Invoke-WebRequest -Uri http://localhost:8080/auth/register -Method POST -Body $body -ContentType "application/json" -SessionVariable nf
$response.Headers["Set-Cookie"]
$response.Content
```

Expected body contains user metadata only, not a token. Expected headers include `auth_token=...`.

If that user already exists, login instead and recreate `$nf`:

```powershell
$demoEmail = "quant@netflow.test"
$body = "{`"email`":`"$demoEmail`",`"password`":`"correcthorse`"}"
$response = Invoke-WebRequest -Uri http://localhost:8080/auth/login -Method POST -Body $body -ContentType "application/json" -SessionVariable nf
$response.Headers["Set-Cookie"]
```

Confirm the session is usable before running quant jobs:

```powershell
Invoke-WebRequest -Uri http://localhost:8080/quant/assumptions -Method GET -WebSession $nf
```

If you see `authentication cookie is required`, `$nf` was not created in the current PowerShell session or the register/login request failed.

## 4. Seed Cycle 3 Demo Data

In the client, click `Seed Cycle 3`.

CLI equivalent:

```powershell
Invoke-WebRequest -Uri http://localhost:8080/demo/seed-cycle-3 -Method POST -WebSession $nf -Body '{}' -ContentType 'application/json'
```

Verify debts:

```powershell
docker compose exec -T -e PGPASSWORD=netflow postgres-primary psql -U netflow -d netflow -c "SELECT d.name,d.balance,d.apr,d.minimum_payment FROM debts d JOIN users u ON u.id=d.user_id WHERE u.email='$demoEmail' ORDER BY d.name;"
```

Verify profile, holdings, assumptions, and benchmark fixtures:

```powershell
docker compose exec -T -e PGPASSWORD=netflow postgres-primary psql -U netflow -d netflow -c "SELECT p.monthly_surplus,p.current_investment_balance,p.monthly_contribution,p.target_wealth,p.risk_tolerance FROM quant_profiles p JOIN users u ON u.id=p.user_id WHERE u.email='$demoEmail';"
docker compose exec -T -e PGPASSWORD=netflow postgres-primary psql -U netflow -d netflow -c "SELECT h.symbol,h.asset_class,h.current_value FROM investment_holdings h JOIN users u ON u.id=h.user_id WHERE u.email='$demoEmail' ORDER BY h.symbol;"
docker compose exec -T -e PGPASSWORD=netflow postgres-primary psql -U netflow -d netflow -c "SELECT method,asset_class,expected_return,volatility,source FROM asset_class_assumptions ORDER BY asset_class;"
docker compose exec -T -e PGPASSWORD=netflow postgres-primary psql -U netflow -d netflow -c "SELECT asset_class,COUNT(*) FROM benchmark_return_series WHERE source='demo_fixture' GROUP BY asset_class ORDER BY asset_class;"
```

What this verifies: the demo seed wrote deterministic source data for the authenticated demo user. Direct `psql` queries see every user in the database, so filter by `$demoEmail` when checking user-owned tables. The debt, wealth, and portfolio jobs read from these rows later; they do not depend on Plaid Sandbox data.

## 5. Debt Solver

Client: click `Debt`.

CLI:

```powershell
Invoke-WebRequest -Uri http://localhost:8080/quant/debt/optimize -Method POST -WebSession $nf -Body '{"preference":"lowest_interest"}' -ContentType 'application/json'
```

Watch the worker and RabbitMQ:

```powershell
docker compose logs quant-worker --tail=120
```

Verify results:

```powershell
docker compose exec -T -e PGPASSWORD=netflow postgres-primary psql -U netflow -d netflow -c "SELECT r.id,r.status,r.recommended_strategy,r.total_interest,r.payoff_months,r.payoff_date FROM debt_optimization_runs r JOIN users u ON u.id=r.user_id WHERE u.email='$demoEmail' ORDER BY r.created_at DESC LIMIT 5;"
docker compose exec -T -e PGPASSWORD=netflow postgres-primary psql -U netflow -d netflow -c "WITH latest AS (SELECT r.id FROM debt_optimization_runs r JOIN users u ON u.id=r.user_id WHERE u.email='$demoEmail' ORDER BY r.created_at DESC LIMIT 1) SELECT s.strategy,s.feasible,s.total_interest,s.payoff_months FROM debt_optimization_strategy_summaries s JOIN latest ON latest.id=s.run_id ORDER BY s.strategy;"
docker compose exec -T -e PGPASSWORD=netflow postgres-primary psql -U netflow -d netflow -c "WITH latest AS (SELECT r.id FROM debt_optimization_runs r JOIN users u ON u.id=r.user_id WHERE u.email='$demoEmail' ORDER BY r.created_at DESC LIMIT 1) SELECT s.strategy,s.month_index,s.debt_name,s.payment,s.interest,s.ending_balance FROM debt_optimization_steps s JOIN latest ON latest.id=s.run_id ORDER BY s.strategy,s.month_index LIMIT 20;"
```

What this verifies: the API created a run, RabbitMQ delivered it, the quant worker completed it, and PostgreSQL now contains both the summary and month-by-month payoff schedule. `status` should become `completed`, and the strategy rows should include baseline comparisons such as minimum-only, avalanche, and snowball.

API:

```powershell
Invoke-WebRequest -Uri http://localhost:8080/quant/debt/runs/latest -Method GET -WebSession $nf
```

## 6. Monte Carlo Wealth Simulator

Client: click `Wealth`.

CLI:

```powershell
Invoke-WebRequest -Uri http://localhost:8080/quant/wealth/simulate -Method POST -WebSession $nf -Body '{"assumption_method":"demo_static"}' -ContentType 'application/json'
```

Verify P10/P50/P90 and target probabilities:

```powershell
docker compose exec -T -e PGPASSWORD=netflow postgres-primary psql -U netflow -d netflow -c "SELECT r.id,r.status,r.seed,r.scenario_count,r.horizons,r.assumption_method FROM wealth_simulation_runs r JOIN users u ON u.id=r.user_id WHERE u.email='$demoEmail' ORDER BY r.created_at DESC LIMIT 5;"
docker compose exec -T -e PGPASSWORD=netflow postgres-primary psql -U netflow -d netflow -c "WITH latest AS (SELECT r.id FROM wealth_simulation_runs r JOIN users u ON u.id=r.user_id WHERE u.email='$demoEmail' ORDER BY r.created_at DESC LIMIT 1) SELECT p.horizon_years,p.percentile,p.nominal_value,p.real_value FROM wealth_simulation_percentiles p JOIN latest ON latest.id=p.run_id ORDER BY p.horizon_years,p.percentile;"
docker compose exec -T -e PGPASSWORD=netflow postgres-primary psql -U netflow -d netflow -c "WITH latest AS (SELECT r.id FROM wealth_simulation_runs r JOIN users u ON u.id=r.user_id WHERE u.email='$demoEmail' ORDER BY r.created_at DESC LIMIT 1) SELECT g.horizon_years,g.target_wealth,g.probability FROM wealth_simulation_goal_probabilities g JOIN latest ON latest.id=g.run_id ORDER BY g.horizon_years;"
```

What this verifies: the Monte Carlo task stored reproducible run metadata, percentile bands for future fan charts, and target-wealth probabilities without storing all raw simulation paths.

API:

```powershell
Invoke-WebRequest -Uri http://localhost:8080/quant/wealth/runs/latest -Method GET -WebSession $nf
```

## 7. Portfolio Optimization

Client: click `Portfolio`.

CLI:

```powershell
Invoke-WebRequest -Uri http://localhost:8080/quant/portfolio/optimize -Method POST -WebSession $nf -Body '{"assumption_method":"demo_static","risk_tolerance":"moderate"}' -ContentType 'application/json'
```

Verify frontier/current/recommendation output:

```powershell
docker compose exec -T -e PGPASSWORD=netflow postgres-primary psql -U netflow -d netflow -c "SELECT r.id,r.status,r.portfolio_value,r.current_return,r.current_volatility,r.current_sharpe FROM portfolio_optimization_runs r JOIN users u ON u.id=r.user_id WHERE u.email='$demoEmail' ORDER BY r.created_at DESC LIMIT 5;"
docker compose exec -T -e PGPASSWORD=netflow postgres-primary psql -U netflow -d netflow -c "WITH latest AS (SELECT r.id FROM portfolio_optimization_runs r JOIN users u ON u.id=r.user_id WHERE u.email='$demoEmail' ORDER BY r.created_at DESC LIMIT 1) SELECT f.point_index,f.expected_return,f.volatility,f.sharpe FROM portfolio_frontier_points f JOIN latest ON latest.id=f.run_id ORDER BY f.point_index LIMIT 15;"
docker compose exec -T -e PGPASSWORD=netflow postgres-primary psql -U netflow -d netflow -c "WITH latest AS (SELECT r.id FROM portfolio_optimization_runs r JOIN users u ON u.id=r.user_id WHERE u.email='$demoEmail' ORDER BY r.created_at DESC LIMIT 1) SELECT p.asset_class,p.current_weight,p.target_weight,p.dollar_delta FROM portfolio_recommendations p JOIN latest ON latest.id=p.run_id ORDER BY p.asset_class;"
```

What this verifies: the optimizer persisted the current portfolio metrics, efficient-frontier points for Cycle 4 visualization, and recommended allocation deltas.

API:

```powershell
Invoke-WebRequest -Uri http://localhost:8080/quant/portfolio/runs/latest -Method GET -WebSession $nf
```

## 8. Assumptions And Estimators

Inspect active demo assumptions:

```powershell
Invoke-WebRequest -Uri http://localhost:8080/quant/assumptions -Method GET -WebSession $nf
```

Inspect run snapshots:

```powershell
docker compose exec -T -e PGPASSWORD=netflow postgres-primary psql -U netflow -d netflow -c "SELECT r.assumptions_snapshot FROM wealth_simulation_runs r JOIN users u ON u.id=r.user_id WHERE u.email='$demoEmail' ORDER BY r.created_at DESC LIMIT 1;"
docker compose exec -T -e PGPASSWORD=netflow postgres-primary psql -U netflow -d netflow -c "SELECT r.assumptions_snapshot,r.covariance_snapshot FROM portfolio_optimization_runs r JOIN users u ON u.id=r.user_id WHERE u.email='$demoEmail' ORDER BY r.created_at DESC LIMIT 1;"
```

What this verifies: each run is explainable later even if default assumptions change. The snapshots are the assumptions and covariance inputs actually used for that run.

Supported assumption methods:

- `demo_static`: fixed deterministic educational defaults.
- `historical_sample`: local synthetic monthly return fixture, no internet.
- `capm`: configurable demo-safe risk-free rate and market risk premium.
- `historical_with_shrinkage`: historical fixture with Ledoit-Wolf shrinkage or diagonal fallback.
- `black_litterman`: simplified equilibrium/no-view implementation for the first Cycle 3 cut.

Run alternative methods:

```powershell
Invoke-WebRequest -Uri http://localhost:8080/quant/wealth/simulate -Method POST -WebSession $nf -Body '{"assumption_method":"historical_with_shrinkage"}' -ContentType 'application/json'
Invoke-WebRequest -Uri http://localhost:8080/quant/portfolio/optimize -Method POST -WebSession $nf -Body '{"assumption_method":"capm"}' -ContentType 'application/json'
Invoke-WebRequest -Uri http://localhost:8080/quant/portfolio/optimize -Method POST -WebSession $nf -Body '{"assumption_method":"black_litterman"}' -ContentType 'application/json'
```

## 9. RabbitMQ And WebSockets

RabbitMQ management:

```text
http://localhost:15672
username: netflow
password: netflow
```

Routing keys:

| Routing key | Producer | Consumer |
|---|---|---|
| `quant.debt.requested` | Go API | `quant-worker` |
| `quant.wealth.requested` | Go API | `quant-worker` |
| `quant.portfolio.requested` | Go API | `quant-worker` |
| `quant.debt.completed` | `quant-worker` | Go API WebSocket fan-out |
| `quant.wealth.completed` | `quant-worker` | Go API WebSocket fan-out |
| `quant.portfolio.completed` | `quant-worker` | Go API WebSocket fan-out |
| `quant.error` | `quant-worker` | Go API WebSocket fan-out |

Queues:

- `quant.debt`
- `quant.wealth`
- `quant.portfolio`
- `netflow.api.quant.completed`

Browser WebSocket demo:

```javascript
const ws = new WebSocket("ws://localhost:8081/api/ws");
ws.onmessage = (e) => console.log(JSON.parse(e.data));
```

Then run any quant job. Completion events should include `run_id`, `run_kind`, and `status`.

## 10. Fallback Without Plaid Credentials

Cycle 3 does not rely on Plaid data. If Plaid credentials are unavailable:

1. Start the stack with `.env.example` values.
2. Register/login through the client or API.
3. Run `POST /demo/seed-cycle-3`.
4. Run the debt, wealth, and portfolio jobs.

The existing Plaid path still proves Cycle 1 and Cycle 2 intake when Sandbox credentials are present:

```text
Plaid -> Go API -> encrypted DB records -> RabbitMQ -> workers -> WebSocket alerts
```

## 11. Tests

Run Go tests through Docker if Go is not installed locally:

```powershell
docker run --rm -v "C:\Users\fredy\Projects\NetFlow:/src" -w /src golang:1.25-alpine go test ./...
```

Run quant worker tests:

```powershell
docker compose run --rm quant-worker pytest -q
```

## 12. Cleanup

```powershell
docker compose down
```

To remove local database and RabbitMQ data:

```powershell
docker compose down -v
```

Only use `-v` when you are okay deleting local demo data.
