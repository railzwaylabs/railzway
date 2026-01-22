import { API_KEY, ORG_ID, ADMIN_SESSION } from '../config/env.js';

export function apiHeaders(extra = {}) {
  const headers = { 'Content-Type': 'application/json', ...extra };
  if (API_KEY) {
    headers.Authorization = `Bearer ${API_KEY}`;
  }
  return headers;
}

export function adminHeaders(extra = {}) {
  const headers = { ...extra };
  if (ADMIN_SESSION) {
    headers.Cookie = `_sid=${ADMIN_SESSION}`;
  }
  if (ORG_ID) {
    headers['X-Org-Id'] = ORG_ID;
  }
  return headers;
}

export function requireApiKey() {
  if (!API_KEY) {
    throw new Error('API_KEY is required');
  }
}

export function requireAdminSession() {
  if (!ADMIN_SESSION) {
    throw new Error('ADMIN_SESSION is required');
  }
}
