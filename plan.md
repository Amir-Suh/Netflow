# Cycle 1 Execution Plan: Native Go Secure Intake & Distributed Core

## Summary

Build NetFlow as a greenfield Go backend with a Dockerized Go client interface, Dockerized infrastructure, JWT authentication, Plaid Sandbox banking intake, RabbitMQ event routing, PostgreSQL persistence, and native Go AES-256-GCM encryption at rest.

Do not use Stripe, production Plaid credentials, C++, pybind11, cgo, Python sidecars, or non-Go crypto modules.

## Key Changes

- Scaffold a single statically compiled Go API under `cmd/api` with internal packages for `auth`, `config`, `db`, `plaid`, `crypto`, `queue`, and HTTP routing.
- Add a containerized Go client interface that runs through Docker Compose so users do not download or install the client directly on their device.
- Add `docker-compose.yml` for the Go API, Go client interface, RabbitMQ, PostgreSQL primary, and PostgreSQL replica.
- Add `.env.example` using only Plaid Sandbox configuration: `PLAID_ENV=sandbox`, `PLAID_CLIENT_ID`, `PLAID_SECRET`, and documented sandbox test credentials.
- Add migrations for users, Plaid sandbox items, linked accounts, transactions, webhook events, and encryption metadata.

## Implementation Plan

### Task 1.1: Containerization & Broker Setup

- Create Dockerfile and Compose services for the Go API, Go client interface, PostgreSQL, and RabbitMQ.
- Add health checks for PostgreSQL, RabbitMQ, and API readiness so the client container waits for a usable backend.
- Configure Go builds to produce static Linux binaries suitable for container deployment.
- Add migration tooling and document local startup in `README.md`.

### Task 1.1a: Dockerized Go Client Interface

- Package the Go client interface as its own Docker image and Compose service, using a dedicated Dockerfile or multi-stage target.
- Configure the client container to call the API through Docker networking, for example `NETFLOW_API_BASE_URL=http://api:8080`.
- Expose only the client interface port needed for local use through Compose.
- Document client usage through `docker compose up --build` or `docker compose run client`, not through a host-installed binary.
- Do not add instructions that require downloading the Go client directly onto the user's device.

### Task 1.2: Go API Gateway & Auth

- Use `chi` for routing and middleware.
- Implement `/healthz`, `/readyz`, `/auth/register`, `/auth/login`, and JWT-protected route groups.
- Hash passwords using a vetted Go package and issue signed JWTs with expiration.
- Add request IDs, structured logs, panic recovery, and strict config validation at startup.

### Task 1.3: Plaid Sandbox Integration (Zero-Cost Intake)

- Do not use Stripe.
- Implement the Plaid API in Go using specifically the Plaid Sandbox environment.
- The Go API must complete the Plaid Link flow using sandbox test credentials.
- The Go API must ingest mock Plaid transaction webhooks and route resulting transaction events to RabbitMQ.
- Do not configure this cycle for live production data.
- Reject or fail startup if `PLAID_ENV` is not exactly `sandbox`.
- Do not add production Plaid environment names, live key examples, Stripe variables, or live-data setup instructions.
- Add routes:
  - `POST /plaid/link-token`
  - `POST /plaid/exchange-public-token`
  - `POST /plaid/sync`
  - `POST /plaid/webhook`
- Store Plaid Sandbox access tokens encrypted and never include them in logs or queue messages.
- Make webhook persistence idempotent to avoid duplicate transaction writes.

### Task 1.4: Native Go Encryption at Rest

- Implement `internal/crypto` using only Go standard library packages `crypto/aes`, `crypto/cipher`, `crypto/rand`, and `encoding/base64`.
- Use AES-256-GCM with a 32-byte base64 key loaded from environment configuration.
- Generate a fresh random nonce for every encrypted transaction description.
- Encrypt sensitive transaction descriptions immediately after Plaid webhook parsing and before PostgreSQL writes or RabbitMQ publishing.
- Store ciphertext, nonce, key ID, and algorithm metadata in PostgreSQL.
- Publish RabbitMQ transaction events without plaintext descriptions.

## Public Interfaces

- HTTP APIs use JSON and bearer JWT auth, except Plaid Sandbox webhooks.
- The Go client interface is accessed through its Docker Compose service and communicates with the API over the Compose network.
- Transaction reads from authenticated API endpoints may decrypt descriptions server-side before returning authorized responses.
- RabbitMQ event payloads include transaction ID, user ID, account ID, event type, sandbox source marker, and timestamp only.
- No host-installed Go client binary, Stripe route, Stripe config, production Plaid config, service boundary, FFI layer, cgo binding, Python extension, or C++ artifact exists in Cycle 1.

## Test Plan

- Unit test AES-256-GCM encryption/decryption, invalid keys, random nonce behavior, tampered ciphertext failure, and base64 config parsing.
- Unit test startup failure when `PLAID_ENV` is anything other than `sandbox`.
- Unit test JWT auth, Plaid Sandbox normalization, config validation, and RabbitMQ payload generation.
- Database tests verify migrations, encrypted description storage, nonce/key metadata, and idempotent webhook handling.
- Integration tests run the Go client interface, API, PostgreSQL, and RabbitMQ through Docker Compose using Plaid Sandbox credentials only.
- Compose verification builds the client image and confirms the client can reach API readiness through the internal service URL.
- Manual verification confirms PostgreSQL and RabbitMQ never contain plaintext transaction descriptions.

## Assumptions

- Cycle 1 is Plaid Sandbox only and zero-cost; Stripe and live production banking data are explicitly out of scope.
- AWS deployment favors one statically compiled Go API binary.
- The Go client interface can be packaged as a statically compiled Go Linux binary inside a container, avoiding direct host installation.
- AES key management starts with environment-provided base64 key material; later cycles may replace this with AWS KMS envelope encryption.
- PostgreSQL replica is included for infrastructure readiness, but production failover and read-routing are deferred.
