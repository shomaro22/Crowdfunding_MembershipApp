import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URL, DEFAULT_THRESHOLDS, checkOK, summaryHandler } from './lib/config.js';
import { loginBuyer } from './lib/auth.js';
import { rampUpStages } from './lib/scenarios.js';

export const options = {
  scenarios: {
    project_list: {
      executor: 'ramping-vus',
      stages: rampUpStages,
    },
  },
  thresholds: DEFAULT_THRESHOLDS,
};

const searchTerms = ['デモ', 'ファン', 'クラブ', 'イラスト'];

export default function projectListLoad() {
  const token = loginBuyer();
  const headers = token
    ? { Accept: 'application/json', Authorization: `Bearer ${token}` }
    : { Accept: 'application/json' };

  const listRes = http.get(`${BASE_URL}/projects?limit=10&offset=0`, { headers });
  checkOK(listRes, 'projects page 1');

  const page2 = http.get(`${BASE_URL}/projects?limit=10&offset=10`, { headers });
  check(page2, {
    'projects page 2 ok or empty': (r) => r.status === 200,
  });

  const term = searchTerms[__VU % searchTerms.length];
  const searchRes = http.get(`${BASE_URL}/projects?search=${encodeURIComponent(term)}&limit=5`, { headers });
  checkOK(searchRes, 'projects search');
  check(searchRes, {
    'search returns array': (r) => {
      const body = r.json();
      return body && Array.isArray(body.projects);
    },
  });

  const featured = http.get(`${BASE_URL}/projects/featured`, { headers });
  checkOK(featured, 'projects featured');

  sleep(0.4);
}

export const handleSummary = summaryHandler('project-list');
