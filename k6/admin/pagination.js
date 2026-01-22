import http from 'k6/http';
import { check } from 'k6';
import { BASE_URL } from '../config/env.js';
import { adminHeaders, requireAdminSession } from '../helpers/auth.js';
import { assert2xx } from '../helpers/checks.js';

const pageSize = parseInt(__ENV.PAGE_SIZE || '50', 10);
const maxPages = parseInt(__ENV.MAX_PAGES || '10', 10);

export const options = {
  vus: 1,
  iterations: 1,
  thresholds: {
    'http_req_failed{name:admin_customers_page}': ['rate<0.01'],
  },
};

export function setup() {
  requireAdminSession();
}

export default function () {
  const seenIDs = new Set();
  const seenTokens = new Set();
  let pageToken = '';
  let pages = 0;
  let hasMore = true;

  while (hasMore && pages < maxPages) {
    const tokenParam = pageToken ? `&page_token=${encodeURIComponent(pageToken)}` : '';
    const url = `${BASE_URL}/admin/customers?page_size=${pageSize}${tokenParam}`;

    const res = http.get(url, {
      headers: adminHeaders(),
      tags: { name: 'admin_customers_page' },
    });

    assert2xx(res);

    if (res.status < 200 || res.status >= 300) {
      break;
    }

    const body = res.json();
    const data = body && body.data ? body.data : {};
    const customers = Array.isArray(data.customers) ? data.customers : [];

    let duplicates = 0;
    const beforeCount = seenIDs.size;

    for (const customer of customers) {
      const id = customer && (customer.ID || customer.id);
      if (!id) {
        continue;
      }
      if (seenIDs.has(id)) {
        duplicates += 1;
        continue;
      }
      seenIDs.add(id);
    }

    check(res, {
      'no duplicate customer ids': () => duplicates === 0,
      'page adds new rows': () => customers.length === 0 || seenIDs.size > beforeCount,
      'next page token advances': () => !data.next_page_token || !seenTokens.has(data.next_page_token),
    });

    if (data.next_page_token) {
      seenTokens.add(data.next_page_token);
    }

    pageToken = data.next_page_token || '';
    hasMore = Boolean(data.has_more && pageToken);
    pages += 1;
  }
}
