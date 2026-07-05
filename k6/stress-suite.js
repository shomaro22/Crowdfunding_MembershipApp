import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URL, BUYER_EMAIL, BUYER_PASSWORD, jsonHeaders, stressThresholdsForEnv, summaryHandler } from './lib/config.js';
import { stressStages } from './lib/scenarios.js';

export const options = {
  scenarios: {
    stress_auth: {
      executor: 'ramping-vus',
      stages: stressStages,
    },
  },
  thresholds: {
    ...stressThresholdsForEnv(),
    checks: ['rate>0.99'],
  },
};

export default function stressAuth() {
  const loginRes = http.post(
    `${BASE_URL}/auth/login`,
    JSON.stringify({ email: BUYER_EMAIL, password: BUYER_PASSWORD }),
    {
      headers: jsonHeaders(),
      responseCallback: http.expectedStatuses(200, 429),
    },
  );
  check(loginRes, {
    'stress login ok or rate limited': (r) => r.status === 200 || r.status === 429,
    'stress login has token when ok': (r) => {
      if (r.status === 429) {
        return true;
      }
      if (r.status !== 200) {
        return false;
      }
      const body = r.json();
      return body && typeof body.auth_token === 'string';
    },
  });
  sleep(0.2);
}

export const handleSummary = summaryHandler('stress-suite');
