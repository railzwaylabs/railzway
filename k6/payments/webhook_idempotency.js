import http from 'k6/http';
import { BASE_URL, PAYMENT_PROVIDER } from '../config/env.js';
import { webhookPayload, webhookHeaders } from '../helpers/payloads.js';
import { assert2xx } from '../helpers/checks.js';

const vus = parseInt(__ENV.WEBHOOK_VUS || '10', 10);

export const options = {
  scenarios: {
    webhook_idempotency: {
      executor: 'per-vu-iterations',
      vus,
      iterations: 1,
      maxDuration: '30s',
    },
  },
  thresholds: {
    'http_req_duration{name:payment_webhook}': ['p(95)<500'],
    'http_req_failed{name:payment_webhook}': ['rate<0.01'],
  },
};

export function setup() {
  if (!PAYMENT_PROVIDER) {
    throw new Error('PAYMENT_PROVIDER is required');
  }

  return {
    provider: PAYMENT_PROVIDER,
    payload: webhookPayload(),
    headers: webhookHeaders(),
  };
}

export default function (data) {
  const res = http.post(
    `${BASE_URL}/api/payments/webhooks/${data.provider}`,
    data.payload,
    {
      headers: data.headers,
      tags: { name: 'payment_webhook' },
    },
  );

  assert2xx(res);
}
