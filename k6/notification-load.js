import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URL, summaryHandler, thresholdsForEnv } from './lib/config.js';
import { loginBuyer } from './lib/auth.js';
import { rampUpStages } from './lib/scenarios.js';

export const options = {
  scenarios: {
    notifications: {
      executor: 'ramping-vus',
      stages: rampUpStages,
    },
  },
  thresholds: thresholdsForEnv(),
};

export function setup() {
  const token = loginBuyer();
  if (!token) {
    throw new Error('notification-load setup: buyer login failed');
  }
  return { token };
}

export default function notificationLoad(data) {
  const headers = { Accept: 'application/json', Authorization: `Bearer ${data.token}` };

  const unreadRes = http.get(`${BASE_URL}/me/notifications`, { headers });
  check(unreadRes, {
    'notifications 200': (r) => r.status === 200,
    'unread_messages number': (r) => {
      const body = r.json();
      return body && typeof body.unread_messages === 'number';
    },
  });

  const threadsRes = http.get(`${BASE_URL}/messages?page=1&per_page=10&messages_per_thread=5`, { headers });
  check(threadsRes, {
    'message threads 200': (r) => r.status === 200,
    'threads array': (r) => {
      const body = r.json();
      return body && Array.isArray(body.threads);
    },
  });

  sleep(0.3);
}

export const handleSummary = summaryHandler('notification-load');
