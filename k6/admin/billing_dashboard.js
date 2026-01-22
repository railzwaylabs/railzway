import http from 'k6/http';
import { BASE_URL } from '../config/env.js';
import { adminHeaders, requireAdminSession } from '../helpers/auth.js';
import { assert2xx } from '../helpers/checks.js';

const vus = parseInt(__ENV.DASHBOARD_VUS || '15', 10);
const duration = __ENV.DASHBOARD_DURATION || '2m';

export const options = {
  scenarios: {
    billing_dashboard: {
      executor: 'constant-vus',
      vus,
      duration,
    },
  },
  thresholds: {
    'http_req_duration{name:admin_billing_customers}': ['p(95)<500'],
    'http_req_duration{name:admin_billing_cycles}': ['p(95)<500'],
    'http_req_failed{name:admin_billing_customers}': ['rate<0.01'],
    'http_req_failed{name:admin_billing_cycles}': ['rate<0.01'],
  },
};

export function setup() {
  requireAdminSession();
}

export default function () {
  const headers = adminHeaders();

  const customers = http.get(`${BASE_URL}/admin/billing/customers`, {
    headers,
    tags: { name: 'admin_billing_customers' },
  });

  assert2xx(customers);

  const cycles = http.get(`${BASE_URL}/admin/billing/cycles`, {
    headers,
    tags: { name: 'admin_billing_cycles' },
  });

  assert2xx(cycles);
}
