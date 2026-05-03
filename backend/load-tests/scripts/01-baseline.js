import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '10s', target: 10 }, // Naik ke 10 virtual users
    { duration: '30s', target: 10 }, // Tahan 30 detik
    { duration: '10s', target: 0 },  // Selesai
  ],
  thresholds: {
    http_req_duration: ['p(95)<3000'], // Target: p95 di bawah 3 detik
    http_req_failed: ['rate<0.01'],    // Error maksimal 1%
  },
};

export default function () {
  const url = 'http://localhost:8080/api/qris?amount=50000';
  const res = http.get(url);

  check(res, {
    'Status 200 OK': (r) => r.status === 200,
    'Ada Payload QRIS': (r) => r.json().hasOwnProperty('qris_payload')
  });

  sleep(1); 
}
