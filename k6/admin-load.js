import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URL, summaryHandler, thresholdsForEnv } from './lib/config.js';
import { loginAdminWithTotp } from './lib/auth.js';
import { rampUpStages } from './lib/scenarios.js';

export const options = {
  scenarios: {
    admin_read: {
      executor: 'ramping-vus',
      stages: rampUpStages,
    },
  },
  thresholds: thresholdsForEnv(),
};

function resolveAdminToken() {
  const totpCode = (__ENV.ADMIN_TOTP_CODE || '').trim();
  if (totpCode) {
    const token = loginAdminWithTotp(totpCode);
    if (token) {
      return token;
    }
  }
  const envToken = (__ENV.ADMIN_AUTH_TOKEN || '').trim();
  if (envToken) {
    return envToken;
  }
  return '';
}

export function setup() {
  const adminToken = resolveAdminToken();
  if (!adminToken) {
    console.error('admin-load setup: failed to obtain admin token (check ADMIN_TOTP_CODE / credentials)');
  }
  return { adminToken };
}

function adminGet(path, token) {
  return http.get(`${BASE_URL}${path}`, {
    headers: { Accept: 'application/json', Authorization: `Bearer ${token}` },
    responseCallback: http.expectedStatuses(200, 429),
  });
}

export default function adminLoad(data) {
  const token = data.adminToken;
  if (!token) {
    const forbidden = http.get(`${BASE_URL}/admin/stats`, {
      headers: { Accept: 'application/json', Authorization: 'Bearer invalid' },
      responseCallback: http.expectedStatuses(401, 403),
    });
    check(forbidden, {
      'admin without auth blocked': (r) => r.status === 401 || r.status === 403,
    });
    sleep(1);
    return;
  }

  const stats = adminGet('/admin/stats', token);
  check(stats, {
    'admin stats': (r) => r.status === 200 || r.status === 429,
  });

  const users = adminGet('/admin/users?limit=20&offset=0', token);
  check(users, {
    'admin users': (r) => r.status === 200 || r.status === 429,
  });

  const reports = adminGet('/admin/reports?limit=20&offset=0', token);
  check(reports, {
    'admin reports': (r) => r.status === 200 || r.status === 429,
  });

  sleep(0.5);
}

export const handleSummary = summaryHandler('admin-load');
