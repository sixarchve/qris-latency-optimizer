Current Repository Report
=========================

This document summarizes the current QRIS Latency Optimizer repository. It is
not a branch diff; it is a concise report of the implemented system.


1. Backend
----------

- Backend entrypoint is backend/cmd/main.go.
- API framework is Gin.
- The code is organized into delivery handlers, middleware, usecases, domain
  entities/interfaces, and repository implementations.
- Startup sequence:
  - load configuration
  - connect PostgreSQL
  - create pgcrypto extension
  - run GORM AutoMigrate
  - seed default merchants
  - connect Redis
  - warm Redis merchant cache
  - connect RabbitMQ
  - start payment consumer worker
  - start HTTP server with graceful shutdown
- Main route registration is in backend/delivery/handler/router.go.


2. Database
-----------

- PostgreSQL is the source of truth.
- Tables are created through GORM AutoMigrate.
- Default seeded merchants:
  - TEST001 / Kantin FILKOM UB
  - TEST002 / TESTING STORE
- Merchant has two important identifiers:
  - ID: UUID primary key
  - QRID: QR merchant identifier stored in qr_id
- Transaction status starts as PENDING and becomes SUCCESS after confirmation.


3. QRIS Payload
---------------

- QRIS payload code is in backend/internal/qris/payload.go.
- Generation uses merchant database data and request amount.
- Payload includes merchant QRID in tag 26.01.
- Payload includes amount in tag 54.
- Payload includes merchant name and city MALANG.
- CRC is generated and validated.
- Tests exist in backend/internal/qris/payload_test.go.


4. Transaction Flow
-------------------

- Customer scan endpoint:
  - POST /api/transactions/scan
- Scan request includes:
  - qr_payload
  - merchant_id
  - amount
- merchant_id can be UUID or QRID.
- Backend validates:
  - merchant exists
  - QR payload parses successfully
  - QR CRC is valid
  - QR merchant matches selected merchant
  - QR amount matches submitted amount
- Backend creates a PENDING transaction in PostgreSQL.
- Backend caches the transaction in Redis.


5. Optimized vs Baseline Confirmation
-------------------------------------

Optimized endpoint:
- POST /api/transactions/:id/confirm
- Validates transaction UUID.
- Publishes transaction_id to RabbitMQ queue payment_confirmations.
- Returns PROCESSING immediately.
- Worker updates PostgreSQL status to SUCCESS.
- Worker invalidates Redis transaction cache.

Baseline endpoint:
- POST /api/transactions/:id/confirm-sync
- Validates transaction UUID.
- Updates PostgreSQL status to SUCCESS during the HTTP request.
- Deletes Redis transaction cache.
- Reloads and returns the updated transaction.

Purpose:
- The async path demonstrates lower request latency by moving the write work to
  RabbitMQ and the worker.
- The sync path remains available for direct comparison.


6. Redis Caching
----------------

Redis is optional. If Redis is unavailable, the application continues using
PostgreSQL.

Merchant cache:
- Key: merchant:<qr_id>
- TTL: 30 minutes
- Warmed on startup.
- Used for QRID lookup during scan.
- Also populated during QRIS generation.

Transaction cache:
- Key: transaction:<transaction_id>
- TTL: 10 minutes
- Used for status polling.
- Corrupted cached transaction payloads are deleted and reloaded from
  PostgreSQL.

Metrics:
- cache_lookup_total
- cache_write_total


7. Frontend Applications
------------------------

Merchant dashboard:
- Path: frontend/
- Runs on port 5173.
- Lists merchants and generates QRIS payloads.

Customer app:
- Path: customer-app/
- Runs on port 5174.
- Scans QRIS payloads.
- Extracts merchant QRID and amount from the payload.
- Creates transactions.
- Confirms payment using the optimized async endpoint.
- Checks transaction status.
- Sends client latency telemetry with Axios interceptors.
- API port is controlled by VITE_API_PORT, supplied by
  CUSTOMER_APP_API_PORT in Docker Compose.


8. Monitoring
-------------

Prometheus:
- Service is defined in docker-compose.yml.
- Config is prometheus.yml.
- Scrapes backend /metrics every 15 seconds.

Grafana:
- Service is defined in docker-compose.yml.
- Datasource provisioning is under grafana/provisioning/datasources/.
- Dashboard provisioning is under grafana/provisioning/dashboards/.
- Dashboard JSON is grafana/dashboards/golang-metrics.json.

Metrics currently exposed:
- http_requests_total
- http_request_duration_seconds
- client_request_duration_seconds
- transactions_created_total
- payment_confirmations_total
- payment_confirmation_duration_seconds
- payment_worker_processed_total
- payment_worker_duration_seconds
- cache_lookup_total
- cache_write_total


9. Rural Network Simulation
---------------------------

Toxiproxy:
- Service is defined in docker-compose.yml.
- Management API: localhost:8474.
- Rural proxy port: localhost:8081.

Setup script:
- k6/rural_test_setup.sh

Configured toxics:
- 500ms latency
- 100ms jitter
- 50KB/s bandwidth, approximately 400kbps

Customer app mode script:
- scripts/customer-app-mode.sh normal
- scripts/customer-app-mode.sh rural
- scripts/customer-app-mode.sh status


10. Load Testing
----------------

K6 scripts:
- k6/qris_generation.js
- k6/scan_async_payment.js
- k6/scan_sync_payment.js

Normal network runner:
- ./k6/run.sh qris
- ./k6/run.sh async
- ./k6/run.sh sync

Rural network runner:
- ./k6/run_rural.sh qris
- ./k6/run_rural.sh async
- ./k6/run_rural.sh sync

The runners use Docker image grafana/k6 and target host.docker.internal.


11. Operational URLs
--------------------

- Backend: http://localhost:8080
- Merchant dashboard: http://localhost:5173
- Customer app: http://localhost:5174
- PostgreSQL: localhost:5432
- Redis: localhost:6379
- RedisInsight: http://localhost:5540
- RabbitMQ management: http://localhost:15672
- pgAdmin: http://localhost:5050
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000
- Toxiproxy management: http://localhost:8474


12. Documentation Files
-----------------------

- README.md: operator and developer guide.
- report-purpose/flow.txt: detailed text flow.
- report-purpose/flow-mermaid.md: Mermaid architecture flow.
- report-purpose/changelog.md: current repository report.
