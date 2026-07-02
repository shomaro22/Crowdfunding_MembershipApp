import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URL, BUYER_EMAIL, BUYER_PASSWORD, jsonHeaders, summaryHandler } from './lib/config.js';
import { stressStages } from './lib/scenarios.js';

export const options = {
  scenarios: {
    stress_auth: {
      executor: 'ramping-vus',
      stages: stressStages,
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.05'],
    http_req_duration: ['p(95)<2000', 'p(99)<4000'],
    checks: ['rate>0.90'],
  },
};

export default function stressAuth() {
  const loginRes = http.post(
    `${BASE_URL}/auth/login`,
    JSON.stringify({ email: BUYER_EMAIL, password: BUYER_PASSWORD }),
    { headers: jsonHeaders() },
  );
  check(loginRes, {
    'stress login': (r) => r.status === 200,
  });
  sleep(0.2);
}

export const handleSummary = summaryHandler('stress-suite');
