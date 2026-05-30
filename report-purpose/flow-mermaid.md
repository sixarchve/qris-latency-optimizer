# QRIS Latency Optimizer Flow

```mermaid
flowchart TD
    A[Backend Start] --> B[Load .env from repo root]
    B --> C[Connect Postgres]
    C --> D[AutoMigrate merchants and transactions]
    D --> E[Seed default merchants]
    E --> F[Connect Redis]
    F --> G[Warm merchant cache]
    G --> G1[Connect RabbitMQ]
    G1 --> G2[Start payment consumer worker]
    G2 --> G3[Start HTTP server with graceful shutdown]

    H[Frontend Merchant Page] --> I[GET /api/merchants]
    I --> J[Query active merchants from Postgres]
    J --> K[Return merchant list]

    L[Frontend Generate QR] --> M[GET /api/qris]
    M --> N[Validate merchant UUID and amount]
    N --> O[Load merchant from Postgres]
    O --> P[Cache merchant in Redis]
    P --> Q[Prefetch related merchants]
    Q --> R[Generate QRIS payload]
    R --> S[Return qris_payload]

    T[Customer Scan QR] --> U[Extract QRID and amount from payload]
    U --> V[POST /api/transactions/scan]
    V --> W[Find merchant by UUID or QRID]
    W --> X[Check Redis merchant cache first]
    X --> Y[Fallback to Postgres if needed]
    Y --> Z[Validate QR CRC, merchant, amount]
    Z --> AA[Create PENDING transaction in Postgres]
    AA --> AB[Cache transaction in Redis]
    AB --> AC[Return transaction_id]

    AD[Customer Check Status] --> AE[GET /api/transactions/:id]
    AE --> AF[Check Redis transaction cache]
    AF -->|Hit| AG[Return cached transaction]
    AF -->|Miss| AH[Query Postgres]
    AH --> AI[Cache transaction in Redis]
    AI --> AJ[Return DB transaction]

    AK[Customer Confirm Payment] --> AL[POST /api/transactions/:id/confirm]
    AL --> AM[Publish transaction_id to RabbitMQ]
    AM --> AN[Return PROCESSING immediately]
    AM --> AO[Payment consumer reads queue]
    AO --> AP[Update status to SUCCESS in Postgres]
    AP --> AQ[Delete Redis transaction cache]

    AR[Baseline Confirm Payment] --> AS[POST /api/transactions/:id/confirm-sync]
    AS --> AT[Update status to SUCCESS in Postgres synchronously]
    AT --> AU[Delete Redis transaction cache]
    AU --> AV[Return SUCCESS transaction]

    AW[Later Status Check] --> AE

    BA[Prometheus Scraper] -->|Every 15s| BB[GET /metrics]
    BB --> BC[Collect http_requests_total]
    BB --> BD[Collect http_request_duration_seconds]
    BB --> BE[Collect Go runtime metrics]

    BF[Customer App Interceptor] --> BG[Measure request round-trip time]
    BG --> BH[POST /api/telemetry]
    BH --> BI[Record client_request_duration_seconds]

    BJ[Grafana Dashboard] --> BA
    BJ --> BI

    CA[K6 Load Test] -->|Normal| CB[Port 8080 - Direct Backend]
    CA -->|Rural| CC[Port 8081 - Through Toxiproxy]
    CC --> CD[Toxiproxy adds 500ms latency + bandwidth limit]
    CD --> CB
```

## Notes

- Postgres is source of truth.
- Redis is cache layer for merchants and transactions.
- QRID like `TEST001` is QR payload merchant identifier.
- Merchant UUID is database primary key.
- Optimized confirm returns `PROCESSING` and finishes through RabbitMQ worker.
- Baseline confirm-sync writes to Postgres before responding.
- Prometheus collects server-side metrics via `/metrics` endpoint.
- Client telemetry measures actual user-perceived latency via Axios interceptors.
- Grafana dashboard compares server latency vs client RTT to visualize rural network lag.
- Toxiproxy simulates rural 3G conditions (500ms latency, ~400kbps bandwidth).
- K6 runs load tests against both normal and rural-simulated endpoints.
