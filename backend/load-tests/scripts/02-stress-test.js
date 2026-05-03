import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '30s', target: 100 }, // Naik bertahap ke 100 VU
    { duration: '1m', target: 500 },  // Lonjakan beban ke 500 VU (Stress Test)
    { duration: '30s', target: 0 },   // Turun kembali perlahan
  ],
  thresholds: {
    http_req_duration: ['p(95)<3000'],
    http_req_failed: ['rate<0.01'],   
  },
};

export default function () {
  const url = 'http://localhost:8080/api/qris?amount=50000';
  const res = http.get(url);

  check(res, {
    'Status 200 OK': (r) => r.status === 200,
  });

  // Jeda dikurangi agar tembakan lebih agresif
  sleep(0.5); 
}
