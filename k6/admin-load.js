import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URL, DEFAULT_THRESHOLDS, summaryHandler } from './lib/config.js';
import { loginAdminWithTotp } from './lib/auth.js';
import { rampUpStages } from './lib/scenarios.js';

export const options = {
  scenarios: {
    admin_read: {
      executor: 'ramping-vus',
      stages: rampUpStages,
    },
  },
  thresholds: DEFAULT_THRESHOLDS,
};

export function setup() {
  const envToken = __ENV.ADMIN_AUTH_TOKEN || '';
  if (envToken) {
    return { adminToken: envToken };
  }
  const totpCode = __ENV.ADMIN_TOTP_CODE || '';
  const adminToken = loginAdminWithTotp(totpCode);
  return { adminToken: adminToken || '' };
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

  const headers = { Accept: 'application/json', Authorization: `Bearer ${token}` };

  const stats = http.get(`${BASE_URL}/admin/stats`, { headers });
  check(stats, { 'admin stats': (r) => r.status === 200 });

  const users = http.get(`${BASE_URL}/admin/users?limit=20&offset=0`, { headers });
  check(users, { 'admin users': (r) => r.status === 200 });

  const reports = http.get(`${BASE_URL}/admin/reports?limit=20&offset=0`, { headers });
  check(reports, { 'admin reports': (r) => r.status === 200 });

  sleep(0.5);
}

export const handleSummary = summaryHandler('admin-load');
