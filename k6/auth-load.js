import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URL, BUYER_EMAIL, BUYER_PASSWORD, DEFAULT_THRESHOLDS, jsonHeaders, summaryHandler } from './lib/config.js';
import { rampUpStages, stressStages } from './lib/scenarios.js';

const profile = __ENV.K6_PROFILE || 'ramp';

export const options = {
  scenarios: {
    auth_load: {
      executor: 'ramping-vus',
      stages: profile === 'stress' ? stressStages : rampUpStages,
    },
  },
  thresholds: DEFAULT_THRESHOLDS,
};

export default function authLoad() {
  const loginRes = http.post(
    `${BASE_URL}/auth/login`,
    JSON.stringify({ email: BUYER_EMAIL, password: BUYER_PASSWORD }),
    { headers: jsonHeaders() },
  );

  const loginOk = check(loginRes, {
    'login ok': (r) => r.status === 200,
    'login has token': (r) => {
      const body = r.json();
      return body && typeof body.auth_token === 'string';
    },
  });
  if (!loginOk) {
    sleep(1);
    return;
  }

  const token = loginRes.json('auth_token');
  const meRes = http.get(`${BASE_URL}/me`, { headers: jsonHeaders(token) });
  check(meRes, { 'me ok': (r) => r.status === 200 });

  const refreshRes = http.post(`${BASE_URL}/auth/refresh`, null, {
    headers: jsonHeaders(),
    cookies: loginRes.cookies,
  });
  check(refreshRes, {
    'refresh ok or rate limited': (r) => r.status === 200 || r.status === 429,
  });

  sleep(0.3);
}

export const handleSummary = summaryHandler('auth-load');
