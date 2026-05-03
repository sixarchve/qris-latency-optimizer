# QRIS Latency Optimizer 🚀

This project is a full-stack QRIS payment system designed to handle extremely low-latency API responses. It implements a cache-aside architecture using Redis to optimize transaction status polling, drastically reducing the load on the primary database.

## 📂 Project Structure

This repository is organized as a monorepo containing both the UI and the server:

- **`/backend`**: The Go backend (using the Gin framework). Handles dynamic QR generation, transaction lifecycle, and caching logic.
- **`/frontend`**: The React + Vite UI. Provides the graphical interface simulating the customer and merchant interaction.

## 🛠️ Tech Stack

- **Backend**: Go, Gin, GORM
- **Frontend**: React, Vite
- **Databases/Infra**: PostgreSQL (Persistence), Redis (Caching), Docker

## 🚀 How to Run

**1. Start the Infrastructure (Database & Redis)**
```bash
docker-compose up -d
```

**2. Start the Backend API**
Open a terminal and run:
```bash
cd backend
go run cmd/main.go
```
*(The backend runs on http://localhost:8080)*

**3. Start the Frontend UI**
Open a new terminal and run:
```bash
cd frontend
npm run dev
```
*(The frontend runs on http://localhost:5173)*

## 📚 Architectural Details (Clean Architecture)

The backend follows Clean Architecture principles:
- **`usecase/customer`**: Contains endpoints mimicking customer actions (e.g., scanning the QR code, simulating payment confirmation).
- **`usecase/service`**: Contains endpoints for the merchant backend (e.g., generating the dynamic QR code string, checking transaction status).
- **Latency Optimization**: The `GetTransactionStatus` API queries Redis first. If there's a cache hit, it returns immediately. On a cache miss, it fetches from PostgreSQL and re-populates Redis. When a payment is confirmed, the Redis cache is instantly invalidated to prevent stale data.
