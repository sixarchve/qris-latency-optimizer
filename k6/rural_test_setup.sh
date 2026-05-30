#!/bin/bash
# Configure Toxiproxy to simulate a rural 3G connection
# Run this AFTER docker compose up -d

TOXIPROXY_API="http://localhost:8474"

echo "⏳ Waiting for Toxiproxy to be ready..."
until curl -s "$TOXIPROXY_API/version" > /dev/null 2>&1; do
  sleep 1
done
echo "✓ Toxiproxy is ready"

# Delete existing proxy if it exists
curl -s -X DELETE "$TOXIPROXY_API/proxies/golang_rural" > /dev/null 2>&1

# Create proxy: listen on 8081, forward to golang backend on 8080
echo "Creating rural proxy (8081 -> golang:8080)..."
curl -s -X POST "$TOXIPROXY_API/proxies" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "golang_rural",
    "listen": "0.0.0.0:8081",
    "upstream": "golang:8080"
  }'

echo ""

# Add 500ms latency (simulates 3G network delay)
echo "Adding 500ms latency toxic..."
curl -s -X POST "$TOXIPROXY_API/proxies/golang_rural/toxics" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "rural_latency",
    "type": "latency",
    "attributes": {
      "latency": 500,
      "jitter": 100
    }
  }'

echo ""

# Add bandwidth limit (simulates 3G ~400kbps downstream)
echo "Adding bandwidth limit (50KB/s ~ 400kbps)..."
curl -s -X POST "$TOXIPROXY_API/proxies/golang_rural/toxics" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "rural_bandwidth",
    "type": "bandwidth",
    "attributes": {
      "rate": 50
    }
  }'

echo ""
echo "✓ Rural 3G simulation active on port 8081"
echo "  - Latency: 500ms ± 100ms jitter"
echo "  - Bandwidth: ~400kbps"
echo ""
echo "Test it:  curl http://localhost:8081/api/ping"
echo "Compare:  curl http://localhost:8080/api/ping"
