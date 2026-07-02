import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URL, DEFAULT_THRESHOLDS, summaryHandler } from './lib/config.js';
import { loginBuyer } from './lib/auth.js';
import { soakStages } from './lib/scenarios.js';

export const options = {
  scenarios: {
    checkout_soak: {
      executor: 'ramping-vus',
      stages: soakStages,
    },
  },
  thresholds: DEFAULT_THRESHOLDS,
};

export default function checkoutLoad() {
  const token = loginBuyer();
  if (!token) {
    sleep(1);
    return;
  }
  const headers = { 'Content-Type': 'application/json', Accept: 'application/json', Authorization: `Bearer ${token}` };

  const projectsRes = http.get(`${BASE_URL}/projects?limit=1&status=ACTIVE`, { headers });
  if (projectsRes.status !== 200) {
    sleep(1);
    return;
  }
  const projects = projectsRes.json('projects');
  if (!projects || projects.length === 0) {
    sleep(1);
    return;
  }

  const projectID = projects[0].id;
  const detailRes = http.get(`${BASE_URL}/projects/${projectID}`, { headers });
  check(detailRes, { 'project detail': (r) => r.status === 200 });

  const plans = detailRes.json('plans');
  const activePlan = plans && plans.find((p) => p.is_active);
  if (!activePlan) {
    sleep(1);
    return;
  }

  const intentRes = http.post(
    `${BASE_URL}/payments/intent`,
    JSON.stringify({ project_id: projectID, plan_id: activePlan.id }),
    { headers },
  );
  check(intentRes, {
    'intent created or conflict': (r) => r.status === 200 || r.status === 409 || r.status === 403,
  });

  sleep(0.5);
}

export const handleSummary = summaryHandler('checkout-load');
