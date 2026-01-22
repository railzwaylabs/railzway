import http from 'k6/http';
import { check } from 'k6';
import { BASE_URL } from '../config/env.js';
import { apiHeaders, requireApiKey } from '../helpers/auth.js';
import { usagePayload, ensureUsageEnv, buildIdempotencyKey } from '../helpers/payloads.js';
import { assert2xx } from '../helpers/checks.js';

const vus = parseInt(__ENV.BURST_VUS || '30', 10);

export const options = {
  scenarios: {
    ingest_burst: {
      executor: 'per-vu-iterations',
      vus,
      iterations: 1,
      maxDuration: '30s',
    },
  },
  thresholds: {
    'http_req_duration{name:usage_ingest_burst}': ['p(95)<300'],
    'http_req_failed{name:usage_ingest_burst}': ['rate<0.01'],
  },
};

export function setup() {
  requireApiKey();
  ensureUsageEnv();

  const idempotencyKey = buildIdempotencyKey('k6-usage-burst');
  const payload = usagePayload({
    idempotencyKey,
    recordedAt: new Date().toISOString(),
  });

  return { payload };
}

export default function (data) {
  const res = http.post(`${BASE_URL}/api/usage`, data.payload, {
    headers: apiHeaders(),
    tags: { name: 'usage_ingest_burst' },
  });

  assert2xx(res);
  check(res, {
    'no 5xx responses': (r) => r.status < 500,
  });
}
