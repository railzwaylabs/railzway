import http from 'k6/http';
import { check } from 'k6';
import { BASE_URL } from '../config/env.js';
import { apiHeaders, requireApiKey } from '../helpers/auth.js';
import { usagePayload, ensureUsageEnv, buildIdempotencyKey } from '../helpers/payloads.js';
import { assert2xx } from '../helpers/checks.js';

let baseID = null;

const iterations = parseInt(__ENV.IDEMPOTENCY_ITERATIONS || '10', 10);

export const options = {
  vus: 1,
  iterations,
  thresholds: {
    'http_req_failed{name:usage_ingest_idempotency}': ['rate<0.01'],
  },
};

export function setup() {
  requireApiKey();
  ensureUsageEnv();

  const idempotencyKey = buildIdempotencyKey('k6-usage-idem');
  const payload = usagePayload({
    idempotencyKey,
    recordedAt: new Date().toISOString(),
  });

  return { payload, idempotencyKey };
}

export default function (data) {
  const res = http.post(`${BASE_URL}/api/usage`, data.payload, {
    headers: apiHeaders(),
    tags: { name: 'usage_ingest_idempotency' },
  });

  assert2xx(res);

  if (res.status >= 200 && res.status < 300) {
    const body = res.json();
    const recordID = body && (body.ID || body.id);

    check(res, {
      'usage id present': () => Boolean(recordID),
      'idempotency returns same record': () => {
        if (!recordID) {
          return false;
        }
        if (baseID === null) {
          baseID = recordID;
          return true;
        }
        return recordID === baseID;
      },
    });
  }
}
