import ws from 'k6/ws';
import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URL, DEFAULT_THRESHOLDS, jsonHeaders, summaryHandler } from './lib/config.js';
import { loginBuyer } from './lib/auth.js';
import { rampUpStages } from './lib/scenarios.js';

export const options = {
  scenarios: {
    ws_smoke: {
      executor: 'ramping-vus',
      stages: rampUpStages,
    },
  },
  thresholds: DEFAULT_THRESHOLDS,
};

export default function wsLoad() {
  const token = loginBuyer();
  if (!token) {
    sleep(1);
    return;
  }

  const wsTokenRes = http.post(`${BASE_URL}/ws/token`, null, { headers: jsonHeaders(token) });
  if (wsTokenRes.status !== 200) {
    sleep(1);
    return;
  }
  const wsToken = wsTokenRes.json('ws_token');
  const wsURL = `${BASE_URL.replace('http', 'ws')}/ws?ws_token=${wsToken}`;

  const res = ws.connect(wsURL, {}, (socket) => {
    socket.on('open', () => {
      socket.setTimeout(() => socket.close(), 2000);
    });
    socket.on('message', () => {
      check(true, { 'ws message received': () => true });
    });
  });

  check(res, { 'ws connected': (r) => r && r.url === wsURL });

  const notifications = http.get(`${BASE_URL}/me/notifications`, { headers: jsonHeaders(token) });
  check(notifications, {
    'notifications after ws': (r) => r.status === 200,
  });

  sleep(0.5);
}

export const handleSummary = summaryHandler('ws-load');
