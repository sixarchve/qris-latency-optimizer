#!/bin/bash
# Run K6 load test through TOXIPROXY (rural 3G simulation)
# Usage: ./k6/run_rural.sh <test>
#   ./k6/run_rural.sh qris
#   ./k6/run_rural.sh async
#   ./k6/run_rural.sh sync

set -e

TEST=$1
BASE_URL="http://host.docker.internal:8081"

case "$TEST" in
  qris)
    SCRIPT="qris_generation.js"
    ;;
  async)
    SCRIPT="scan_async_payment.js"
    ;;
  sync)
    SCRIPT="scan_sync_payment.js"
    ;;
  *)
    echo "Usage: ./k6/run_rural.sh <qris|async|sync>"
    exit 1
    ;;
esac

echo "🌾 Running K6 load test: $SCRIPT (Rural 3G via Toxiproxy)"
echo "   Target: $BASE_URL (proxied through Toxiproxy)"
echo ""
echo "   Make sure you ran ./k6/rural_test_setup.sh first!"
echo ""

docker run --rm \
  --add-host=host.docker.internal:host-gateway \
  -v "$(pwd)/k6:/scripts" \
  -e BASE_URL="$BASE_URL" \
  grafana/k6 run "/scripts/$SCRIPT"
