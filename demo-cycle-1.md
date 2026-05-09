# NetFlow Cycle 1 Demo Runbook

## Goal

Demo Cycle 1 as a fully Dockerized secure intake stack:

- Go API container
- Go client interface container
- PostgreSQL primary and replica containers
- RabbitMQ container
- Plaid Sandbox-only configuration
- JWT auth
- AES-256-GCM encryption at rest
- RabbitMQ transaction event publishing without plaintext descriptions

## 1. Prep Environment

From the repo root:

```powershell
cd C:\Users\fredy\Projects\NetFlow
```

If you want the Plaid Sandbox flow to work live, create `.env` from `.env.example` and fill in real Sandbox credentials:

```powershell
Copy-Item .env.example .env
```

Set these in `.env`:

```text
PLAID_ENV=sandbox
PLAID_CLIENT_ID=<your-plaid-sandbox-client-id>
PLAID_SECRET=<your-plaid-sandbox-secret>
```

Keep `PLAID_ENV=sandbox`. The API intentionally refuses to start for non-sandbox Plaid environments.

## 2. Start Containers

```powershell
docker compose up --build -d
```

Check everything is healthy:

```powershell
docker compose ps
```

Expected healthy services:

- `api`
- `client`
- `rabbitmq`
- `postgres-primary`
- `postgres-replica`

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

## 4. Demo Script

### Show Dockerized Client

Open:

```text
http://localhost:8081
```

Point out:

- The client interface is served from the `client` container.
- The client calls the API through Docker networking at `http://api:8080`.
- No Go client binary is installed or downloaded directly onto the host device.
- Status should show `API online` and `Backend ready`.

### Register Or Login

In the client:

1. Enter an email and password.
2. Click `Register`.
3. Confirm a bearer token appears.

If the user already exists:

1. Click `Login`.
2. Confirm a bearer token appears.

### Plaid Sandbox Flow

In the client:

1. Click `Link token`.
2. Click `Open Link`.
3. Use Plaid Sandbox credentials:

```text
username: user_good
password: pass_good
```

4. Complete Sandbox Link.
5. Confirm a public token appears.
6. Click `Exchange token`.
7. Confirm an Item ID appears.
8. Click `Sync`.

For dynamic transaction testing, use:

```text
username: user_transactions_dynamic
password: any non-blank password
institution: First Platypus Bank / ins_109508
```

### Mock Webhook

After an Item ID exists, click:

```text
Mock webhook
```

Expected result:

- Webhook is accepted.
- Transactions are synced from Plaid Sandbox.
- Events are published to RabbitMQ.
- Plaintext transaction descriptions are not published to RabbitMQ.

## 5. Quick Terminal Checks

Client can reach API internally:

```powershell
docker compose exec client wget -qO- http://api:8080/readyz
```

Expected:

```json
{"status":"ready"}
```

API is ready from host:

```powershell
Invoke-RestMethod http://localhost:8080/readyz
```

Client is ready from host:

```powershell
Invoke-RestMethod http://localhost:8081/readyz
```

Run tests:

```powershell
docker run --rm -v "C:\Users\fredy\Projects\NetFlow:/src" -w /src golang:1.25-alpine go test ./...
```

## 6. RabbitMQ Demo

Open:

```text
http://localhost:15672
```

Show:

- Exchange: `netflow.events`
- Queue: `transactions.ingested`
- Routing key: `transactions.ingested`

Message payloads should include IDs and metadata only, not plaintext transaction descriptions.

## 7. PostgreSQL Encryption Check

Connect to Postgres:

```powershell
docker compose exec postgres-primary psql -U netflow -d netflow
```

Check transaction encryption columns:

```sql
SELECT
  id,
  description_ciphertext,
  description_nonce,
  description_key_id,
  description_algorithm,
  source_environment
FROM transactions
LIMIT 5;
```

Expected:

- `description_ciphertext` is encrypted text.
- `description_nonce` is present.
- `description_algorithm` is `AES-256-GCM`.
- `source_environment` is `sandbox`.
- No plaintext transaction names appear in stored descriptions.

Exit Postgres:

```sql
\q
```

## 8. Fallback Demo Without Plaid Credentials

If real Plaid Sandbox credentials are not available:

1. Start the stack with `.env.example` defaults.
2. Show all containers are healthy.
3. Open `http://localhost:8081`.
4. Show `API online` and `Backend ready`.
5. Register a user and show JWT creation.
6. Explain that Plaid API calls require real Sandbox credentials, while the API still enforces `PLAID_ENV=sandbox`.
7. Show the code/config rule that rejects non-sandbox Plaid environments.

Useful command:

```powershell
docker compose logs api --tail=80
```

## 9. Stop Containers

```powershell
docker compose down
```

To remove volumes and reset local data:

```powershell
docker compose down -v
```

Only use `-v` when you are okay deleting local Postgres and RabbitMQ data.
