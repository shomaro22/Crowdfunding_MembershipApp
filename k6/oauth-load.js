import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URL, DEFAULT_THRESHOLDS, summaryHandler } from './lib/config.js';
import { spikeStages } from './lib/scenarios.js';

export const options = {
  scenarios: {
    oauth_spike: {
      executor: 'ramping-vus',
      stages: spikeStages,
    },
  },
  thresholds: {
    ...DEFAULT_THRESHOLDS,
    http_req_failed: ['rate<0.1'],
  },
};

export default function oauthLoad() {
  const startRes = http.get(`${BASE_URL}/auth/google?from=/`, {
    redirects: 0,
    responseCallback: http.expectedStatuses(200, 302, 404, 503),
  });
  check(startRes, {
    'google start redirect or disabled': (r) =>
      r.status === 302 || r.status === 503 || r.status === 200 || r.status === 404,
  });

  const exchangeRes = http.post(
    `${BASE_URL}/auth/oauth/exchange`,
    JSON.stringify({ code: 'invalid-load-test-code' }),
    {
      headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      responseCallback: http.expectedStatuses(400, 401, 404, 503),
    },
  );
  check(exchangeRes, {
    'oauth exchange rejects invalid code or disabled': (r) =>
      r.status === 400 || r.status === 401 || r.status === 503 || r.status === 404,
  });

  const callbackRes = http.get(`${BASE_URL}/auth/callback?error=oauth_failed`, {
    redirects: 0,
    responseCallback: http.expectedStatuses(200, 302),
  });
  check(callbackRes, {
    'oauth callback error redirect': (r) => r.status === 302 || r.status === 200,
  });

  sleep(0.2);
}

export const handleSummary = summaryHandler('oauth-load');
