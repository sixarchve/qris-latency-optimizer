# QRIS Latency Optimizer Flow

```mermaid
flowchart TD
    A[Backend Start] --> B[Load repo-root .env or container env]
    B --> C[Connect PostgreSQL]
    C --> D[Create pgcrypto extension]
    D --> E[AutoMigrate merchants and transactions]
    E --> F[Seed default merchants]
    F --> G[Connect Redis]
    G --> H[Warm merchant cache]
    H --> I[Connect RabbitMQ]
    I --> IQ[Declare payment_confirmations and merchant_notifications]
    IQ --> WH[Start WebSocket hub]
    WH --> J[Start payment and notification workers]
    J --> K[Start Gin HTTP server on 8080]

    MD[Merchant Dashboard] --> ML[GET /api/merchants]
    ML --> MP[Query active merchants from PostgreSQL]
    MP --> MR[Return merchant UUIDs and QRIDs]

    MD --> Q1[GET /api/qris]
    Q1 --> Q2[Validate merchant UUID and amount]
    Q2 --> Q3[Load merchant from PostgreSQL]
    Q3 --> Q4[Cache merchant in Redis]
    Q4 --> Q5[Prefetch related merchants]
    Q5 --> Q6[Generate QRIS payload with CRC]
    Q6 --> Q7[Return qris_payload]

    CA[Customer App] --> S1[Scan QRIS payload]
    S1 --> S2[Extract QRID tag 26.01 and amount tag 54]
    S2 --> S3[POST /api/transactions/scan]
    S3 --> S4[Find merchant by UUID or QRID]
    S4 --> S5{Merchant in Redis?}
    S5 -->|Yes| S6[Use cached merchant]
    S5 -->|No| S7[Query PostgreSQL and cache merchant]
    S6 --> S8[Validate QR CRC, merchant, amount]
    S7 --> S8
    S8 --> S9[Create PENDING transaction in PostgreSQL]
    S9 --> S10[Cache transaction in Redis]
    S10 --> S11[Return transaction_id]

    CA --> ST1[GET /api/transactions/:id]
    ST1 --> ST2{Transaction in Redis?}
    ST2 -->|Hit| ST3[Return cached transaction]
    ST2 -->|Miss or corrupt| ST4[Query PostgreSQL]
    ST4 --> ST5[Cache fresh transaction]
    ST5 --> ST6[Return DB transaction]

    CA --> AC1[POST /api/transactions/:id/confirm]
    AC1 --> AC2[Publish transaction_id to RabbitMQ]
    AC2 --> AC3[Return PROCESSING immediately]
    AC2 --> AC4[Worker consumes payment_confirmations queue]
    AC4 --> AC5[Update PostgreSQL status to SUCCESS]
    AC5 --> AC6[Delete Redis transaction cache]
    AC6 --> NQ[Publish merchant notification]
    NQ --> NW[Notification worker consumes merchant_notifications]
    NW --> WS[Push transaction_notification over /ws]
    WS --> MD
    AC6 --> ST1

    CA --> SC1[POST /api/transactions/:id/confirm-sync]
    SC1 --> SC2[Update PostgreSQL status to SUCCESS during request]
    SC2 --> SC3[Delete Redis transaction cache]
    SC3 --> SN[Publish merchant notification]
    SN --> NW
    SC3 --> SC4[Reload transaction]
    SC4 --> SC5[Return SUCCESS]

    PR[Prometheus] -->|Every 15s| MT[GET /metrics]
    MT --> M1[HTTP latency and request metrics]
    MT --> M2[Payment confirmation metrics]
    MT --> M3[Worker metrics]
    MT --> M4[Cache metrics]
    MT --> M5[Go runtime metrics]

    CA --> T1[Axios request/response interceptor]
    T1 --> T2[Measure client round-trip time]
    T2 --> T3[POST /api/telemetry]
    T3 --> T4[Record client_request_duration_seconds]

    GF[Grafana Dashboard] --> PR
    GF --> T4

    MD --> WSC[GET /ws?merchant_id]
    WSC --> WH
    MD --> WSS[GET /api/ws/status]
    WSS --> WH

    K6[K6 Tests] --> N1[Normal target port 8080]
    K6 --> R1[Rural target port 8081]
    R1 --> TX[Toxiproxy latency and bandwidth toxics]
    TX --> N1
```

## Notes

- PostgreSQL is the source of truth.
- Redis caches active merchants and recent transactions.
- RabbitMQ powers the optimized asynchronous confirmation path.
- RabbitMQ also carries merchant notification events.
- `/ws?merchant_id=<uuid>` streams successful payment notifications to the
  merchant dashboard.
- `/api/ws/status` exposes connection and pending-notification counts.
- `/confirm` returns `PROCESSING`; the worker later writes `SUCCESS`.
- `/confirm-sync` is the baseline synchronous path.
- Customer telemetry measures user-perceived request duration.
- Port `8080` is treated as `normal`; port `8081` is treated as `rural`.
- Toxiproxy rural mode adds 500ms latency, 100ms jitter, and about 400kbps
  bandwidth.
