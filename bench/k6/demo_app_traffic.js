import http from 'k6/http';
import { check } from 'k6';

const TARGET = __ENV.DEMO_URL || 'http://localhost:8080/api/demo';
const RATE = parseInt(__ENV.RATE || '200', 10);
const DURATION = __ENV.DURATION || '2m';

export const options = {
  scenarios: {
    constant_load: {
      executor: 'constant-arrival-rate',
      rate: RATE,
      timeUnit: '1s',
      duration: DURATION,
      preAllocatedVUs: 50,
      maxVUs: 500,
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.50'],
  },
};

export default function () {
  const res = http.get(TARGET);
  check(res, { 'got a response': (r) => r.status > 0 });
}
