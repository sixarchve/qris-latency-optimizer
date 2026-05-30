import http from 'k6/http';
import { check, sleep } from 'k6';

// ── Configuration ──────────────────────────────────────────
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export const options = {
  stages: [
    { duration: '10s', target: 20 },
    { duration: '30s', target: 20 },
    { duration: '10s', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'],
  },
};

// ── Setup: Get merchant UUID and generate a QRIS payload ──
export function setup() {
  const res = http.get(`${BASE_URL}/api/merchants`);
  const body = JSON.parse(res.body);
  let merchants = body;
  if (body.data) merchants = body.data;

  if (!merchants || merchants.length === 0) {
    throw new Error('No merchants found.');
  }

  const merchant = merchants[0];
  const merchantID = merchant.ID || merchant.id;
  console.log(`Using merchant: ${merchant.merchant_name} (${merchantID})`);

  // Generate a QRIS payload to use in scan requests
  const qrisRes = http.get(`${BASE_URL}/api/qris?merchant_id=${merchantID}&amount=10000`);
  const qrisBody = JSON.parse(qrisRes.body);

  return {
    merchantID: merchantID,
    qrPayload: qrisBody.qris_payload,
    merchantQRID: merchant.qr_id || 'TEST001',
  };
}

// ── Main test: Scan + Async Confirm ────────────────────────
export default function (data) {
  const headers = { 'Content-Type': 'application/json' };

  // Step 1: Scan QR (create PENDING transaction)
  const scanRes = http.post(
    `${BASE_URL}/api/transactions/scan`,
    JSON.stringify({
      qr_payload: data.qrPayload,
      merchant_id: data.merchantQRID,
      amount: 10000,
    }),
    { headers }
  );

  const scanOk = check(scanRes, {
    'scan status is 201': (r) => r.status === 201,
  });

  if (!scanOk) {
    console.error(`Scan failed: ${scanRes.status} ${scanRes.body}`);
    return;
  }

  const scanBody = JSON.parse(scanRes.body);
  const txID = scanBody.data.transaction_id;

  // Step 2: Confirm payment (ASYNC via RabbitMQ)
  const confirmRes = http.post(
    `${BASE_URL}/api/transactions/${txID}/confirm`,
    null,
    { headers }
  );

  check(confirmRes, {
    'confirm status is 200': (r) => r.status === 200,
    'status is PROCESSING': (r) => JSON.parse(r.body).data.status === 'PROCESSING',
  });

  sleep(0.5);
}
