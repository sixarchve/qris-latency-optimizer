Changelog
=========

Backend
-------

1. QRIS payload generation
- Replaced mostly static QR payload generation with input-based generation.
- Payload now uses:
  - merchant name from database
  - merchant QRID from database
  - amount from request
- Added QRIS CRC generation.
- Default city set to MALANG.
- Added QR payload parsing and validation:
  - parse merchant QRID from tag 26.01
  - parse amount from tag 54
  - validate CRC

2. Merchant identifier cleanup
- Clarified merchant identifiers:
  - `ID` = UUID primary key
  - `QRID` = merchant QR identifier, stored in `qr_id`
- Renamed merchant field from confusing `QRIS` to `QRID`.

3. Transaction scan flow hardening
- Added validation for `merchant_id`.
- Allowed `merchant_id` in scan request to be:
  - UUID
  - QRID like `TEST001`
- Added merchant active/existence validation before transaction create.
- Added QR payload validation in scan flow:
  - merchant in QR must match selected merchant
  - amount in QR must match request amount
- Removed panic-prone `uuid.MustParse` usage in scan flow.

4. Transaction status and confirm flow hardening
- Transaction status now:
  - checks Redis first
  - falls back to Postgres
  - deletes corrupted cached transaction payload
- Confirm payment now:
  - checks rows affected
  - returns not found if transaction does not exist
  - reloads updated transaction safely

5. Redis changes
- Redis startup changed from implicit package `init()` to explicit startup call.
- Added merchant cache helpers:
  - `WarmUpCache()`
  - `PrefetchMerchant()`
  - `PrefetchRelatedMerchants()`
  - `GetMerchant()`
  - `CacheMerchant()`
- Warm-up of merchant cache now runs at backend startup.
- Transaction cache uses shared TTL constant.
- Merchant cache used in customer scan flow for QRID lookup.
- Qris generation flow now caches merchant and triggers related merchant prefetch.

6. Database bootstrap
- Removed SQL init dependency from Docker startup.
- Deleted old `backend/init-db/init.sql`.
- Database creation now handled from Go:
  - `pgcrypto` extension creation
  - `AutoMigrate`
  - default merchant seed
- Added Go seed for:
  - `TEST001` / `Kantin FILKOM UB`
  - `TEST002` / `TESTING STORE`

7. Handler/service organization
- Merged server-side transaction status handler into `backend/usecase/service/qris.go`.
- Deleted old separate `backend/usecase/service/transaction.go`.

8. CORS
- Replaced unsafe wildcard CORS setup.
- Added env-based allowed origins using `CORS_ALLOWED_ORIGINS`.
- Safer headers/method configuration.

9. Environment and Compose
- Repo layout shifted to backend-local setup:
  - `backend/.env`
  - `backend/.env_example`
  - `backend/docker-compose.yml`
- LoadEnv currently expects `.env` in backend working directory.
- Compose now includes:
  - Postgres
  - Redis
  - RedisInsight
  - PgAdmin
  - InfluxDB
  - Grafana

10. Asynchronous payment confirmation with RabbitMQ
- Integrated the `github.com/rabbitmq/amqp091-go` message broker package.
- Added `backend/repository/rabbitmq/rabbitmq.go` to handle connections, channel establishment, retry logic (3 attempts with backoff), and publishing message payloads.
- Implemented a background payment consumer worker (`backend/worker/payment_consumer.go`) that consumes from the `payment_confirmations` queue, asynchronously updates the transaction status to `SUCCESS` in PostgreSQL, and invalidates the cached transaction payload in Redis.
- Updated payment confirmation endpoint routing:
  - Split into optimized asynchronous `/api/transactions/:id/confirm` (publishes message to RabbitMQ queue, returning instantly).
  - Maintained synchronous `/api/transactions/:id/confirm-sync` as a baseline reference for direct load-test comparison.

11. Graceful shutdown and improved startup sequence
- Refactored `backend/cmd/main.go` to implement robust graceful shutdown handling using a signal listener (`SIGINT`, `SIGTERM`).
- The server now closes its RabbitMQ channel/connection cleanly and allows active HTTP requests to complete within a 5-second graceful timeout.
- Added clean connection/startup verification messages (`✓`) in terminal logs.

12. CORS and configuration updates
- Refactored CORS configuration in `backend/delivery/handler/cors.go` with dynamic allowed origins logic to support any development origin under ports `:5173` and `:5174` (allowing local LAN testing/IPs) and the monitoring dashboard served on `:8080`.
- Added fallback default value for `RABBITMQ_URL` env variable in `backend/.env_example`.
- Shifted default PostgreSQL timezone setting from `Asia/Shanghai` to `Asia/Jakarta` in `backend/repository/database/pg.go`.


Customer App / Frontend
-----------------------

1. Customer scan behavior
- Customer app scans QR payload.
- Customer app extracts merchant QRID and amount from scanned QR.
- Customer app sends:
  - `qr_payload`
  - `merchant_id`
  - `amount`

2. HTTPS experiment
- Temporary HTTPS dev-server setup for `customer-app` was added, then reverted.
- Current customer app Vite config is back to normal HTTP dev mode.


DevOps, Monitoring & Load Testing
---------------------------------

1. Real-time monitoring dashboard
- Built a highly visual and responsive dual-dashboard frontend under `backend/monitoring/`:
  - `/monitor` (`index.html`): Real-time system monitoring (CPU, RAM, Go runtime, Goroutine count, queue status) and K6 load-test visualizer.
  - `/latency` (`latency.html`): Real-time endpoint request duration histograms and moving average graphs comparing optimized vs non-optimized routes.
- Served dashboards directly from Go/Gin static routes.
- Added a custom latency tracker middleware (`backend/usecase/service/latency_tracker.go`) to transparently capture stats for all API requests.
- Added system and metrics collection REST APIs:
  - `/api/monitor/system` (resource usage, service status).
  - `/api/monitor/live` (live endpoint stats, average/p95 latency, error count).
  - `/api/monitor/k6` / `/api/monitor/k6/data` / `/api/monitor/k6/summary` (K6 testing live ingestion endpoint).

2. Load testing suite (K6 integration)
- Added new K6 load test scenarios under `tests_script/`:
  - `03-polling-test.js`: Simulates clients aggressively checking transaction statuses.
  - `04-payment-confirm.js`: Load tests the optimized, asynchronous, RabbitMQ-backed `/api/transactions/:id/confirm` endpoint.
  - `05-payment-confirm-lama.js`: Load tests the unoptimized, synchronous, DB-blocking `/api/transactions/:id/confirm-sync` endpoint.
  - `06-optimized-dashboard.js` & `07-non-optimized-dashboard.js`: Test configurations matching dashboard visualizations.
- Added unit tests for service layer monitoring (`monitor_test.go`) and payload verification logic (`payload_test.go`).
- Added JSON configuration for Grafana / K6 optimization dashboards (`dashboard-1778674676803.json`).

3. Infrastructure components in Docker
- Re-enabled RabbitMQ container service in `backend/docker-compose.yml` (`guest:guest` auth, standard data port `5672`, and management dashboard on `15672`).
- Preconfigured InfluxDB and Grafana services for load test metrics storage and visual charting.


Docs
----

1. Added flow documentation
- `flow.txt`
- `flow-mermaid.md`
