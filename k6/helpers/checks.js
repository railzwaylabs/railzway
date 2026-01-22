import { check } from 'k6';

export function assert2xx(res, label = 'status is 2xx') {
  return check(res, {
    [label]: (r) => r.status >= 200 && r.status < 300,
  });
}

export function assertLatency(res, maxMs, label) {
  const name = label || `latency < ${maxMs}ms`;
  return check(res, {
    [name]: (r) => r.timings.duration < maxMs,
  });
}

export function assertStatus(res, expected, label) {
  const name = label || `status is ${expected}`;
  return check(res, {
    [name]: (r) => r.status === expected,
  });
}
