import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URL, SMOKE_THRESHOLDS, checkOK, summaryHandler } from './lib/config.js';
import { smokeStages } from './lib/scenarios.js';

export const options = {
  scenarios: {
    smoke: {
      executor: 'ramping-vus',
      stages: smokeStages,
    },
  },
  thresholds: SMOKE_THRESHOLDS,
};

export default function smoke() {
  const health = http.get(`${BASE_URL}/health`);
  checkOK(health, 'health');

  const projects = http.get(`${BASE_URL}/projects?limit=5`);
  checkOK(projects, 'list projects');
  check(projects, {
    'projects array': (r) => {
      const body = r.json();
      return body && Array.isArray(body.projects);
    },
  });

  sleep(0.5);
}

export const handleSummary = summaryHandler('smoke');
