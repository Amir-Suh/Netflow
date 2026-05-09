# NetFlow

NetFlow is a Go API gateway and Dockerized Go client interface for secure financial transaction intake. Cycle 1 is intentionally limited to Plaid Sandbox data, PostgreSQL, RabbitMQ, JWT authentication, and native Go AES-256-GCM encryption.

## Hard Scope Rules

- Do not use Stripe in Cycle 1.
- Do not configure Plaid development or production environments.
- Do not use C++, pybind11, cgo, Python sidecars, or non-Go crypto modules.
- Sensitive transaction descriptions must be encrypted before PostgreSQL writes and must never be published to RabbitMQ.

## Local Setup

1. Copy `.env.example` to `.env` when you are ready to use real Plaid Sandbox credentials.
2. Fill in Plaid Sandbox values for `PLAID_CLIENT_ID` and `PLAID_SECRET`.
3. Keep `PLAID_ENV=sandbox`; the API fails startup for any other value.
4. Start the stack:

```powershell
docker compose up --build
```

The Dockerized client interface listens on `http://localhost:8081` and reaches the API through the Compose network at `http://api:8080`. The API also listens on `http://localhost:8080`. RabbitMQ management is available at `http://localhost:15672` with `netflow` / `netflow`.

The Go client is built into the `client` container from `cmd/client`; do not install or download a separate client binary on the host.
Compose loads `.env.example` for local defaults and then applies `.env` overrides when that file exists.

## API Flow

- `POST /auth/register` creates a local user and returns a bearer JWT.
- `POST /auth/login` returns a bearer JWT for an existing user.
- `POST /plaid/link-token` creates a Plaid Sandbox Link token.
- `POST /plaid/exchange-public-token` stores the encrypted Sandbox access token.
- `POST /plaid/sync` ingests Sandbox transactions for a linked Item.
- `POST /plaid/webhook` accepts mock Plaid Sandbox transaction webhooks and routes ingested transaction events to RabbitMQ.

Use Plaid Sandbox Link test credentials:

- username: `user_good`
- password: `pass_good`

For transaction webhook testing, use `user_transactions_dynamic` with any non-blank password and a non-OAuth Sandbox institution such as First Platypus Bank (`ins_109508`). Plaid-delivered webhooks require `PLAID_WEBHOOK_URL` to be publicly reachable; for local testing, post a mock Sandbox webhook directly to `/plaid/webhook`.

## Verification

Run tests when Go is available:

```powershell
go test ./...
```

Verify the Dockerized client and API build:

```powershell
docker compose build api client
```

After the stack is running, confirm the client can reach API readiness inside Docker:

```powershell
docker compose exec client wget -qO- http://api:8080/readyz
```

Manual checks:

- PostgreSQL `transactions.description_ciphertext` should contain ciphertext, not Plaid transaction names.
- RabbitMQ `transactions.ingested` messages should contain IDs and metadata only, never plaintext descriptions.
