export const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
export const API_KEY = __ENV.API_KEY || 'vk_live_key_F98QDZLSN9J4_f2fcf6e5ca11cd463b182956bf5d338f13186ea52ec84f1e4acd24db0d6a9d79';
export const ORG_ID = __ENV.ORG_ID || '2008111837777760256';

export const CUSTOMER_ID = __ENV.CUSTOMER_ID || '2008111839082188800';
export const METER_CODE = __ENV.METER_CODE || 'hybrid-usage-meter-teste2e-ratinghybrid';

export const ADMIN_SESSION = __ENV.ADMIN_SESSION || '';

export const PAYMENT_PROVIDER = __ENV.PAYMENT_PROVIDER || '';
export const WEBHOOK_PAYLOAD = __ENV.WEBHOOK_PAYLOAD || '';
export const WEBHOOK_HEADERS = __ENV.WEBHOOK_HEADERS || '';

export const AUTH_COMPARE = String(__ENV.AUTH_COMPARE || '').toLowerCase() === 'true';
