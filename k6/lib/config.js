import { check } from 'k6';
import { textSummary } from 'https://jslib.k6.io/k6-summary/0.0.5/index.js';
import { htmlReport } from 'https://raw.githubusercontent.com/benc-uk/k6-reporter/main/dist/bundle.js';

const ENV_PROFILES = {
  development: {
    baseUrl: 'http://localhost',
    buyerEmail: 'buyer@example.com',
    buyerPassword: 'password123',
    adminEmail: 'admin@example.com',
    adminPassword: 'password123',
    adminTotpSecret: 'JBSWY3DPEHPK3PXP',
  },
  staging: {
    baseUrl: __ENV.STAGING_URL || 'https://staging.example.com',
    buyerEmail: __ENV.BUYER_EMAIL || 'buyer@example.com',
    buyerPassword: __ENV.BUYER_PASSWORD || 'password123',
    adminEmail: __ENV.ADMIN_EMAIL || 'admin@example.com',
    adminPassword: __ENV.ADMIN_PASSWORD || 'password123',
    adminTotpSecret: __ENV.ADMIN_TOTP_SECRET || 'JBSWY3DPEHPK3PXP',
  },
  production: {
    baseUrl: __ENV.PROD_URL || 'https://example.com',
    buyerEmail: __ENV.BUYER_EMAIL,
    buyerPassword: __ENV.BUYER_PASSWORD,
    adminEmail: __ENV.ADMIN_EMAIL,
    adminPassword: __ENV.ADMIN_PASSWORD,
    adminTotpSecret: __ENV.ADMIN_TOTP_SECRET,
  },
};

const ACTIVE_ENV = __ENV.K6_ENV || 'development';
export const profile = ENV_PROFILES[ACTIVE_ENV] || ENV_PROFILES.development;

export const BASE_URL = __ENV.BASE_URL || profile.baseUrl;
export const BUYER_EMAIL = __ENV.BUYER_EMAIL || profile.buyerEmail;
export const BUYER_PASSWORD = __ENV.BUYER_PASSWORD || profile.buyerPassword;
export const ADMIN_EMAIL = __ENV.ADMIN_EMAIL || profile.adminEmail;
export const ADMIN_PASSWORD = __ENV.ADMIN_PASSWORD || profile.adminPassword;
export const ADMIN_TOTP_SECRET = __ENV.ADMIN_TOTP_SECRET || profile.adminTotpSecret;

export const DEFAULT_THRESHOLDS = {
  http_req_failed: ['rate<0.02'],
  http_req_duration: ['p(95)<800', 'p(99)<1500'],
  checks: ['rate>0.95'],
};

export const SMOKE_THRESHOLDS = {
  http_req_failed: ['rate<0.01'],
  http_req_duration: ['p(95)<500', 'p(99)<1000'],
  checks: ['rate>0.99'],
};

export const STRESS_THRESHOLDS = {
  http_req_failed: ['rate<0.05'],
  http_req_duration: ['p(95)<2000', 'p(99)<4000'],
};

export function jsonHeaders(token = '') {
  const headers = {
    'Content-Type': 'application/json',
    Accept: 'application/json',
  };
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  return headers;
}

export function checkOK(res, name) {
  const ok = check(res, {
    [`${name} status 2xx`]: (r) => r.status >= 200 && r.status < 300,
  });
  if (!ok) {
    console.error(`${name} failed: ${res.status} ${String(res.body).slice(0, 200)}`);
  }
  return ok;
}

export function summaryHandler(prefix = 'k6') {
  return function handleSummary(data) {
    return {
      [`k6/reports/${prefix}-summary.json`]: JSON.stringify(data, null, 2),
      [`k6/reports/${prefix}-summary.txt`]: textSummary(data, { indent: ' ', enableColors: false }),
      [`k6/reports/${prefix}-report.html`]: htmlReport(data),
      stdout: textSummary(data, { indent: ' ', enableColors: true }),
    };
  };
}
