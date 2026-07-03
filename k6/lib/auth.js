import http from 'k6/http';
import { BASE_URL, BUYER_EMAIL, BUYER_PASSWORD, jsonHeaders } from './config.js';

export function loginBuyer() {
  const res = http.post(
    `${BASE_URL}/auth/login`,
    JSON.stringify({ email: BUYER_EMAIL, password: BUYER_PASSWORD }),
    {
      headers: jsonHeaders(),
      responseCallback: http.expectedStatuses(200, 429),
    },
  );
  if (res.status !== 200) {
    return null;
  }
  const body = res.json();
  if (!body || !body.auth_token) {
    return null;
  }
  return body.auth_token;
}

export function loginAdminWithTotp(totpCode) {
  const loginRes = http.post(
    `${BASE_URL}/auth/login`,
    JSON.stringify({ email: __ENV.ADMIN_EMAIL || 'admin@example.com', password: __ENV.ADMIN_PASSWORD || 'password123' }),
    {
      headers: jsonHeaders(),
      responseCallback: http.expectedStatuses(200, 429),
    },
  );
  if (loginRes.status !== 200) {
    return null;
  }
  const pending = loginRes.json();
  if (!pending || !pending.requires_2fa) {
    return pending && pending.auth_token ? pending.auth_token : null;
  }
  if (!totpCode) {
    return null;
  }
  const challengeRes = http.post(
    `${BASE_URL}/auth/2fa/challenge`,
    JSON.stringify({ code: totpCode }),
    {
      headers: jsonHeaders(),
      cookies: loginRes.cookies,
      responseCallback: http.expectedStatuses(200, 429),
    },
  );
  if (challengeRes.status !== 200) {
    return null;
  }
  const body = challengeRes.json();
  return body && body.auth_token ? body.auth_token : null;
}
