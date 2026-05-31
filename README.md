# QRIS Latency Optimizer

Full-stack QRIS payment simulation focused on comparing normal and optimized
payment-confirmation latency.

The project includes:

- Go backend API with Gin and clean repository/usecase layers
- Merchant dashboard built with React + Vite
- Customer scanner/payment app built with React + Vite
- PostgreSQL as the source of truth
- Redis cache for merchant lookup and transaction-status polling
- RabbitMQ queue for asynchronous payment confirmation
- Prometheus + Grafana monitoring
- Toxiproxy rural-network simulation
- K6 load-test scripts

## Project Structure

```text
backend/          Go API, domain/usecase/repository code, QRIS payload logic
frontend/         Merchant dashboard, QRIS generation UI
customer-app/     Customer QR scanner and payment confirmation UI
k6/               Load-test scripts and rural proxy setup
grafana/          Provisioned dashboard and datasource config
report-purpose/   Architecture, flow, and report notes
scripts/          Helper scripts for switching customer app network mode
docker-compose.yml
prometheus.yml
```

## Stack

- Go, Gin, GORM
- PostgreSQL 15
- Redis 7
- RabbitMQ management image
- React 19 + Vite
- Prometheus
- Grafana
- Toxiproxy
- K6

## Architecture Summary

- PostgreSQL is the source of truth for merchants and transactions.
- The backend creates the `pgcrypto` extension, runs GORM `AutoMigrate`, and
  seeds default merchants at startup.
- Redis is an optional acceleration layer. If Redis is unavailable, the backend
  continues through PostgreSQL.
- Merchant data is warmed into Redis at startup and also cached when QRIS
  payloads are generated or QRID lookups happen.
- Transaction status uses cache-aside:
  Redis first, PostgreSQL fallback, then Redis repopulation.
- Optimized payment confirmation publishes work to RabbitMQ and returns
  `PROCESSING` quickly. A background worker updates PostgreSQL to `SUCCESS` and
  invalidates the transaction cache.
- Synchronous confirmation is kept as a baseline comparison endpoint.
- Prometheus records server latency, confirmation metrics, worker metrics,
  cache metrics, and client-reported request latency.
- Toxiproxy exposes port `8081` to simulate rural 3G-like latency and bandwidth.

## Environment

Create a repo-root `.env` file before running Docker Compose. The checked-in
`.gitignore` intentionally ignores `.env`.

Example:

```env
DB_USER=user
DB_PASSWORD=user
DB_HOST=localhost
DB_PORT=5432
DB_NAME=qrisdatabase

REDIS_HOST=127.0.0.1
REDIS_PORT=6379

RABBITMQ_USER=guest
RABBITMQ_PASSWORD=guest
RABBITMQ_HOST=localhost
RABBITMQ_PORT=5672

CORS_ALLOWED_ORIGINS=http://localhost:5173,http://127.0.0.1:5173,http://localhost:5174,http://127.0.0.1:5174

CUSTOMER_APP_API_PORT=8080

PGADMIN_DEFAULT_EMAIL=admin@admin.com
PGADMIN_DEFAULT_PASSWORD=admin

GF_AUTH_ANONYMOUS_ENABLED=true
GF_AUTH_ANONYMOUS_ORG_ROLE=Admin
GF_SECURITY_ADMIN_USER=admin
GF_SECURITY_ADMIN_PASSWORD=12345
```

Docker Compose overrides service hostnames internally. For example, the backend
container receives `DB_HOST=db`, `REDIS_HOST=redis`, and
`RABBITMQ_HOST=rabbitmq`.

## Run With Docker Compose

From the repo root:

```bash
docker compose up -d
```

This starts:

- Backend API on `http://localhost:8080`
- Merchant dashboard on `http://localhost:5173`
- Customer app on `http://localhost:5174`
- PostgreSQL on `localhost:5432`
- Redis on `localhost:6379`
- RedisInsight on `http://localhost:5540`
- RabbitMQ management on `http://localhost:15672`
- pgAdmin on `http://localhost:5050`
- Prometheus on `http://localhost:9090`
- Grafana on `http://localhost:3000`
- Toxiproxy management API on `http://localhost:8474`

Useful checks:

```bash
curl http://localhost:8080/api/ping
curl http://localhost:8080/api/merchants
curl http://localhost:8080/metrics
```

## Run Apps Manually

For local development loops, start only the dependency containers you need, then
run the backend and apps on the host:

```bash
docker compose up -d db redis rabbitmq redisinsight pgadmin toxiproxy
```

If the full Compose stack is already running, stop the app container you want to
replace locally first, for example:

```bash
docker compose stop golang
```

Backend:

```bash
cd backend
go run ./cmd
```

Merchant dashboard:

```bash
cd frontend
npm install
npm run dev
```

Customer app:

```bash
cd customer-app
npm install
npm run dev
```

Default local app URLs:

```text
Backend:            http://localhost:8080
Merchant dashboard: http://localhost:5173
Customer app:       http://localhost:5174
```

## Main API Routes

```text
GET  /api/ping
GET  /api/merchants
GET  /api/qris?merchant_id=<merchant_uuid>&amount=<amount>
GET  /api/transactions/:id
GET  /metrics
POST /api/transactions/scan
POST /api/transactions/:id/confirm
POST /api/transactions/:id/confirm-sync
POST /api/telemetry
```

## Payment Flow

### 1. Merchant List

```text
GET /api/merchants
```

Returns active merchants from PostgreSQL. The merchant dashboard uses the UUID
`id` as `merchant_id` when generating QRIS payloads.

Seeded merchants:

```text
TEST001 - Kantin FILKOM UB
TEST002 - TESTING STORE
```

### 2. Generate QRIS

```text
GET /api/qris?merchant_id=<merchant_uuid>&amount=<amount>
```

The backend validates the merchant UUID and amount, loads the merchant from
PostgreSQL, caches merchant data in Redis, prefetches related merchants, and
returns a dynamic QRIS payload.

The QRIS payload includes merchant QRID in tag `26.01`, amount in tag `54`,
merchant name in tag `59`, city `MALANG`, and a CRC checksum in tag `63`.

### 3. Customer Scan

```text
POST /api/transactions/scan
```

Request:

```json
{
  "qr_payload": "<qris_payload>",
  "merchant_id": "TEST001",
  "amount": 1000
}
```

The customer app extracts QRID and amount from the scanned QRIS payload, then
sends them to the backend. The backend accepts `merchant_id` as either merchant
UUID or QRID, validates the QR CRC, verifies merchant and amount consistency,
creates a `PENDING` transaction in PostgreSQL, and caches it in Redis for 10
minutes.

### 4. Transaction Status

```text
GET /api/transactions/:id
```

The backend validates the UUID, checks Redis key `transaction:<id>`, falls back
to PostgreSQL on miss or corrupted cache, and returns the transaction response.

### 5. Optimized Async Confirmation

```text
POST /api/transactions/:id/confirm
```

The backend validates the UUID, publishes `transaction_id` to RabbitMQ queue
`payment_confirmations`, and immediately returns:

```json
{
  "data": {
    "transaction_id": "<uuid>",
    "status": "PROCESSING"
  },
  "message": "payment accepted and is being processed in background"
}
```

The payment worker consumes the message, updates the transaction to `SUCCESS`,
and deletes the old Redis transaction cache.

### 6. Baseline Sync Confirmation

```text
POST /api/transactions/:id/confirm-sync
```

The backend updates PostgreSQL to `SUCCESS` during the HTTP request, deletes the
Redis transaction cache, reloads the transaction, and returns the final data.

## Redis Keys

```text
merchant:<qr_id>          TTL 30 minutes
transaction:<uuid>        TTL 10 minutes
```

Redis is used for faster lookups and lower database read load. PostgreSQL
remains authoritative.

## Monitoring

Prometheus scrapes `/metrics` every 15 seconds. Grafana is provisioned with a
dashboard for normal vs rural traffic and backend vs client-perceived latency.

Open:

```text
Prometheus: http://localhost:9090
Targets:    http://localhost:9090/targets
Grafana:    http://localhost:3000
```

Default Grafana credentials come from `.env`; the example above uses
`admin` / `12345`.

Important metrics:

```text
http_requests_total
http_request_duration_seconds
client_request_duration_seconds
transactions_created_total
payment_confirmations_total
payment_confirmation_duration_seconds
payment_worker_processed_total
payment_worker_duration_seconds
cache_lookup_total
cache_write_total
```

The customer app records request round-trip duration with Axios interceptors and
sends it to `POST /api/telemetry`. Metrics include `network_mode`, derived from
port `8080` as `normal` and port `8081` as `rural`.

## Load Testing With K6

Scripts live in `k6/`.

| Command | Scenario |
| --- | --- |
| `./k6/run.sh qris` | QRIS generation load test |
| `./k6/run.sh async` | Optimized async scan + confirm flow |
| `./k6/run.sh sync` | Baseline sync scan + confirm flow |

The scripts run K6 through Docker using the `grafana/k6` image.

## Rural Network Simulation

Configure Toxiproxy after Compose is running:

```bash
./k6/rural_test_setup.sh
```

The proxy listens on `localhost:8081` and forwards to the backend with:

- 500 ms latency
- 100 ms jitter
- 50 KB/s bandwidth, roughly 400 kbps

Compare normal and rural:

```bash
curl http://localhost:8080/api/ping
curl http://localhost:8081/api/ping
```

Run rural K6 tests:

```bash
./k6/run_rural.sh qris
./k6/run_rural.sh async
./k6/run_rural.sh sync
```

Switch Docker customer app mode:

```bash
./scripts/customer-app-mode.sh normal
./scripts/customer-app-mode.sh rural
./scripts/customer-app-mode.sh status
```

Normal mode sends customer app traffic to backend port `8080`. Rural mode sends
traffic through Toxiproxy port `8081` and recreates the customer app container
with `CUSTOMER_APP_API_PORT=8081`.

For manual local customer-app testing through the proxy:

```bash
cd customer-app
VITE_API_PORT=8081 npm run dev -- --host
```

## Phone Camera Notes

Phone camera access can fail on plain LAN HTTP because browsers often require a
secure origin for camera APIs. If the scanner does not open, check browser
permissions and try a browser/device combination that allows camera access for
your test origin.

## Extra Docs

- `report-purpose/flow.txt`
- `report-purpose/flow-mermaid.md`
- `report-purpose/changelog.md`
