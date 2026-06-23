import http from 'k6/http';
import { check } from 'k6';

const TARGET = __ENV.AGENT_URL || 'http://localhost:9100/metrics';

export const options = {
  scenarios: {
    ramp: {
      executor: 'ramping-vus',
      startVUs: 1,
      stages: [
        { duration: '30s', target: 20 },
        { duration: '1m', target: 50 },
        { duration: '30s', target: 0 },
      ],
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<200', 'p(99)<500'],
    http_req_failed: ['rate<0.01'],
  },
};

export default function () {
  const res = http.get(TARGET);
  check(res, {
    'status is 200': (r) => r.status === 200,
    'exposes system_ metrics': (r) => r.body && r.body.indexOf('system_') !== -1,
  });
}
