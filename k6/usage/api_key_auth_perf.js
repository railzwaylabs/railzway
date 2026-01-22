import http from 'k6/http';
import { check } from 'k6';
import { Trend } from 'k6/metrics';
import { BASE_URL, AUTH_COMPARE } from '../config/env.js';
import { apiHeaders, requireApiKey } from '../helpers/auth.js';
import { assert2xx } from '../helpers/checks.js';

const vus = parseInt(__ENV.AUTH_VUS || '5', 10);
const duration = __ENV.AUTH_DURATION || '1m';
const compareUnauth = AUTH_COMPARE;

const authOverheadMs = new Trend('auth_overhead_ms');
const authOverheadRatio = new Trend('auth_overhead_ratio');

const thresholds = {
  'http_req_duration{name:auth_with_key}': ['p(95)<200'],
  'http_req_failed{name:auth_with_key}': ['rate<0.01'],
};

if (compareUnauth) {
  thresholds.auth_overhead_ms = ['p(95)<50'];
  thresholds.auth_overhead_ratio = ['p(95)<2'];
}

export const options = {
  scenarios: {
    auth_perf: {
      executor: 'constant-vus',
      vus,
      duration,
    },
  },
  thresholds,
};

export function setup() {
  requireApiKey();
}

export default function () {
  const url = `${BASE_URL}/api/customers?page_size=1`;

  const resAuth = http.get(url, {
    headers: apiHeaders(),
    tags: { name: 'auth_with_key' },
  });

  assert2xx(resAuth);

  if (compareUnauth) {
    const resNoAuth = http.get(url, {
      tags: { name: 'auth_without_key' },
    });

    check(resNoAuth, {
      'unauth is 401 or 403': (r) => r.status === 401 || r.status === 403,
    });

    const authDuration = resAuth.timings.duration;
    const noAuthDuration = resNoAuth.timings.duration || 1;

    authOverheadMs.add(authDuration - noAuthDuration);
    authOverheadRatio.add(authDuration / Math.max(noAuthDuration, 1));
  }
}
