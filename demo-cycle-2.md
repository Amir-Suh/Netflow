# NetFlow Cycle 2 Demo Runbook

## Goal

Demo Cycle 2 as the Intelligence Pipeline layered on top of the Cycle 1 stack:

- HTTP-only cookie authentication (Cycle 1 fix — replaces Bearer tokens in body)
- Python Celery NLP categorization worker (TF-IDF + rapidfuzz)
- Python Celery arbitrage scraper worker (Playwright headless Chromium)
- Real-time WebSocket alerts pushed from Go API to the browser
- New PostgreSQL columns and `arbitrage_opportunities` table

The Cycle 1 services keep running unchanged: Go API, Go client, PostgreSQL primary/replica, RabbitMQ.

## 1. Prep Environment

From the repo root:

```powershell
cd C:\Users\amirs\Desktop\Netflow\Netflow
```

If you already have a `.env` from Cycle 1, no changes are required — Cycle 2 reuses the same Plaid Sandbox credentials, AES key, and JWT secret. Otherwise:

```powershell
Copy-Item .env.example .env
```

Set Plaid Sandbox credentials in `.env`:

```text
PLAID_ENV=sandbox
PLAID_CLIENT_ID=<your-plaid-sandbox-client-id>
PLAID_SECRET=<your-plaid-sandbox-secret>
```

The categorizer worker reuses the same `AES_256_GCM_KEY_BASE64` so it can decrypt transaction descriptions written by the Go API.

## 2. Build And Start Containers

The first build pulls the Playwright base image and compiles the Python wheels — expect the initial run to take several minutes.

```powershell
docker compose up --build -d
```

Check everything is healthy:

```powershell
docker compose ps
```

Expected services (Cycle 1 + Cycle 2):

- `api`
- `client`
- `rabbitmq`
- `postgres-primary`
- `postgres-replica`
- `categorizer-worker`
- `arbitrage-worker`

The two worker services do not expose ports; verify they are running with logs:

```powershell
docker compose logs categorizer-worker --tail=30
docker compose logs arbitrage-worker --tail=30
```

Expected log lines:

```text
categorizer.worker :: listening on transactions.ingested
arbitrage.worker   :: listening on transactions.arbitrage
celery@... ready.
```

## 3. Open Demo URLs

Client interface:

```text
http://localhost:8081
```

API readiness:

```text
http://localhost:8080/readyz
```

RabbitMQ management:

```text
http://localhost:15672
```

RabbitMQ login:

```text
username: netflow
password: netflow
```

## 4. Verify The Cycle 1 Cookie Fix

Cycle 2 begins with a hardening fix: JWTs are now delivered as `HttpOnly` cookies instead of being returned in the JSON response body.

### Cookie is set on register/login

```powershell
$body = '{"email":"demo@netflow.test","password":"correcthorse"}'
$response = Invoke-WebRequest -Uri http://localhost:8080/auth/register -Method POST -Body $body -ContentType "application/json" -SessionVariable nf
$response.Headers["Set-Cookie"]
```

Expected `Set-Cookie` header:

```text
auth_token=eyJhbGciOi...; Path=/; Max-Age=86400; HttpOnly; SameSite=Lax
```

Confirm the response body no longer leaks the token:

```powershell
$response.Content
```

Expected JSON:

```json
{"user_id":1,"email":"demo@netflow.test"}
```

### Cookie unlocks protected routes

The `Invoke-WebRequest` session variable `$nf` automatically forwards the cookie:

```powershell
Invoke-WebRequest -Uri http://localhost:8080/plaid/link-token -Method POST -WebSession $nf
```

Expected: `200 OK` with a Plaid link token. Without the cookie:

```powershell
Invoke-WebRequest -Uri http://localhost:8080/plaid/link-token -Method POST
```

Expected: `401 Unauthorized` with body `{"error":"authentication cookie is required"}`.

### Logout clears the cookie

```powershell
Invoke-WebRequest -Uri http://localhost:8080/auth/logout -Method POST -WebSession $nf
Invoke-WebRequest -Uri http://localhost:8080/plaid/link-token -Method POST -WebSession $nf
```

Expected after logout: `401 Unauthorized`.

## 5. Demo Script — Intelligence Pipeline

### Trigger A Plaid Sandbox Sync

In the client (`http://localhost:8081`):

1. Register or login (now uses cookies, no visible token in the UI).
2. Click `Link token`.
3. Click `Open Link`.
4. Use Plaid Sandbox dynamic credentials so transactions stream:

```text
username: user_transactions_dynamic
password: any non-blank password
institution: First Platypus Bank / ins_109508
```

5. Click `Exchange token`.
6. Click `Sync`.
7. Click `Mock webhook`.

The Go API now publishes each transaction to `transactions.ingested`, where the categorizer worker picks it up.

### Watch The Categorizer Run

In a fresh terminal, tail the worker logs:

```powershell
docker compose logs -f categorizer-worker
```

Expected output for each ingested transaction:

```text
dispatched categorize_transaction tx=42 user=1
categorizing transaction id=42 user=1
Task categorize_transaction[...] succeeded: {'status': 'ok', 'transaction_id': 42, 'category': 'Dining', 'confidence': 1.0}
```

If the merchant matches `Utilities` or `Subscriptions`, the worker also fans out to the arbitrage queue:

```powershell
docker compose logs -f arbitrage-worker
```

Expected output:

```text
dispatched check_arbitrage for tx=42
arbitrage check tx=42 user=1 merchant=Netflix category=Subscriptions amount=15.99
scraping netflix at https://www.netflix.com/signup/planform
```

### Confirm Categories In PostgreSQL

```powershell
docker compose exec postgres-primary psql -U netflow -d netflow
```

```sql
SELECT id, merchant_name, category, ml_confidence, categorized_at
  FROM transactions
 WHERE category IS NOT NULL
 ORDER BY categorized_at DESC
 LIMIT 10;
```

Expected:

- `category` is one of `Dining`, `Groceries`, `Subscriptions`, `Utilities`, `Transportation`, etc.
- `ml_confidence` is between `0.0` and `1.0`.
- `categorized_at` is recent.

### Confirm Arbitrage Opportunities

```sql
SELECT id, merchant_name, category, current_amount, market_rate, savings_estimate, provider_url, found_at
  FROM arbitrage_opportunities
 ORDER BY found_at DESC
 LIMIT 5;
```

Expected: rows for any sandbox subscription/utility transaction whose scraped market rate beat the user's current amount. Rows where the scrape failed or no provider was matched will be skipped (logged only).

Exit Postgres:

```sql
\q
```

## 6. Real-Time WebSocket Alerts

The Go API exposes `GET /ws` (cookie-authenticated). The categorizer and arbitrage workers each emit a RabbitMQ message that the API forwards to the right user's open browser sessions.

### Quick Browser Console Demo

After logging in at `http://localhost:8081`, open DevTools (F12) → Console:

```javascript
const ws = new WebSocket("ws://localhost:8081/api/ws");
ws.onopen = () => console.log("ws connected");
ws.onmessage = (e) => console.log("alert", JSON.parse(e.data));
```

Then trigger another sync or mock webhook in the client. Each categorized transaction prints an alert in the console:

```text
alert {event_type: "transactions.categorized", transaction_id: 42, payload: {...}}
```

If a Subscriptions/Utilities transaction is processed, an arbitrage alert follows:

```text
alert {event_type: "transactions.arbitrage.results", transaction_id: 42, payload: {... market_rate: 11.99, savings_estimate: 4.00 ...}}
```

### CLI WebSocket Demo (Optional)

```powershell
# Login first to capture the cookie
Invoke-WebRequest -Uri http://localhost:8080/auth/login -Method POST `
  -Body '{"email":"demo@netflow.test","password":"correcthorse"}' `
  -ContentType "application/json" -SessionVariable nf | Out-Null

# Use websocat or wscat from inside a container
docker run --rm -it --network netflow_default ghcr.io/vi/websocat ws://api:8080/ws `
  -H "Cookie: auth_token=$($nf.Cookies.GetCookies('http://localhost:8080')['auth_token'].Value)"
```

## 7. RabbitMQ Routing Inspection

Open `http://localhost:15672` and look at the `netflow.events` topic exchange. You should now see four routing keys in use:

| Routing key | Producer | Consumer |
|---|---|---|
| `transactions.ingested` | Go API | categorizer-worker |
| `transactions.categorized` | categorizer-worker | Go API (WebSocket fan-out) |
| `transactions.arbitrage` | categorizer-worker | arbitrage-worker |
| `transactions.arbitrage.results` | arbitrage-worker | Go API (WebSocket fan-out) |

Bindings panel shows queues:

- `transactions.ingested`
- `transactions.arbitrage`
- `netflow.api.categorized`
- `netflow.api.arbitrage.results`
- `netflow.categorizer` (Celery internal)
- `netflow.arbitrage` (Celery internal)

## 8. Quick Terminal Checks

API ready:

```powershell
Invoke-RestMethod http://localhost:8080/readyz
```

Worker → API → DB sanity:

```powershell
docker compose exec postgres-primary psql -U netflow -d netflow -c "SELECT COUNT(*) FROM transactions WHERE category IS NOT NULL;"
docker compose exec postgres-primary psql -U netflow -d netflow -c "SELECT COUNT(*) FROM arbitrage_opportunities;"
```

Run Go tests:

```powershell
docker run --rm -v "C:\Users\amirs\Desktop\Netflow\Netflow:/src" -w /src golang:1.25-alpine go test ./...
```

Restart a single worker after editing it:

```powershell
docker compose up -d --build categorizer-worker
docker compose up -d --build arbitrage-worker
```

## 9. Fallback Demo Without Plaid Credentials

If you cannot reach Plaid Sandbox during the demo:

1. Start the stack with `.env.example` defaults.
2. Show all seven services healthy via `docker compose ps`.
3. Run the Cycle 1 cookie-fix walkthrough in section 4 (works without Plaid).
4. Manually insert a test transaction to drive the pipeline:

```powershell
docker compose exec postgres-primary psql -U netflow -d netflow
```

```sql
-- Build a fake encrypted payload via the Go API by registering and
-- calling /plaid/sync with a stub item; OR insert a row that the
-- categorizer cannot decrypt and watch the worker retry/skip path.
```

5. Open the browser console WebSocket demo to show the channel is up even before any events flow.

## 10. Stop Containers

```powershell
docker compose down
```

To remove volumes and reset local data (Postgres rows, RabbitMQ queues, categorized transactions, arbitrage history):

```powershell
docker compose down -v
```

Only use `-v` when you are okay deleting local Postgres and RabbitMQ data.
