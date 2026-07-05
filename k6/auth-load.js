import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URL, BUYER_EMAIL, BUYER_PASSWORD, jsonHeaders, summaryHandler, thresholdsForEnv } from './lib/config.js';
import { rampUpStages, stressStages } from './lib/scenarios.js';

const profile = __ENV.K6_PROFILE || 'ramp';

export const options = {
  scenarios: {
    auth_load: {
      executor: 'ramping-vus',
      stages: profile === 'stress' ? stressStages : rampUpStages,
    },
  },
  thresholds: thresholdsForEnv(),
};

export default function authLoad() {
  const loginRes = http.post(
    `${BASE_URL}/auth/login`,
    JSON.stringify({ email: BUYER_EMAIL, password: BUYER_PASSWORD }),
    {
      headers: jsonHeaders(),
      responseCallback: http.expectedStatuses(200, 429),
    },
  );

  check(loginRes, {
    'login ok or rate limited': (r) => r.status === 200 || r.status === 429,
    'login has token when ok': (r) => {
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
  if (loginRes.status !== 200) {
    sleep(1);
    return;
  }

  const token = loginRes.json('auth_token');
  const meRes = http.get(`${BASE_URL}/me`, { headers: jsonHeaders(token) });
  check(meRes, { 'me ok': (r) => r.status === 200 });

  const refreshRes = http.post(`${BASE_URL}/auth/refresh`, null, {
    headers: jsonHeaders(),
    cookies: loginRes.cookies,
    responseCallback: http.expectedStatuses(200, 204, 429),
  });
  check(refreshRes, {
    'refresh ok or rate limited': (r) => r.status === 200 || r.status === 204 || r.status === 429,
  });

  sleep(0.3);
}

export const handleSummary = summaryHandler('auth-load');
