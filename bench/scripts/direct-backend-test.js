import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Custom Metrics
const non2xxErrors = new Rate('non_2xx_errors');
const connectionErrors = new Rate('connection_errors');
export const rateLimited = new Rate('rate_limited');
const retryLatency = new Trend('retry_latency');

// Read environment configuration
const defaultBackends = [
  'http://localhost:3001',
  'http://localhost:3002',
  'http://localhost:3003',
  'http://localhost:3004',
  'http://localhost:3005',
  'http://localhost:3006',
].join(',');

const rawBackends = __ENV.BACKEND_URLS || defaultBackends;
const backendUrls = rawBackends.split(',').map((b) => b.trim()).filter(Boolean);
const totalDuration = parseInt(__ENV.TEST_DURATION || '120', 10);

const rampTime = Math.max(1, Math.floor(totalDuration * 0.1));
const steadyTime = Math.max(1, totalDuration - (2 * rampTime));

export const options = {
  stages: [
    { duration: `${rampTime}s`, target: 50 },
    { duration: `${steadyTime}s`, target: 50 },
    { duration: `${rampTime}s`, target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(50)<500', 'p(95)<1500', 'p(99)<3000'],
    http_req_failed: ['rate<0.10'],
    non_2xx_errors: ['rate<0.10'],
    connection_errors: ['rate<0.05'],
  },
};

export default function () {
  // Pick a random backend directly (no gateway load balancing)
  const backend = backendUrls[Math.floor(Math.random() * backendUrls.length)];

  // Map backend port to expected endpoint:
  // Ports 3001, 3002, 3003 -> POST /
  // Ports 3004, 3005, 3006 -> GET /contacts
  const isPost = backend.includes(':3001') || backend.includes(':3002') || backend.includes(':3003');

  if (isPost) {
    const url = `${backend}/`;
    const payload = JSON.stringify({ name: 'direct-test', timestamp: Date.now() });
    const params = {
      headers: { 'Content-Type': 'application/json' },
      tags: { backend: backend, target: url, route: '/' },
    };

    const res = http.post(url, payload, params);

    const is2xx = res.status >= 200 && res.status < 300;
    const isConnErr = res.status === 0;

    non2xxErrors.add(!is2xx);
    connectionErrors.add(isConnErr);
    rateLimited.add(res.status === 429);

    if (res.headers['X-Retry-Count'] || res.headers['x-retry-count']) {
      retryLatency.add(res.timings.duration);
    }

    check(res, {
      'direct POST status is 200/201': (r) => r.status === 200 || r.status === 201,
    });
  } else {
    const url = `${backend}/contacts`;
    const params = {
      tags: { backend: backend, target: url, route: '/contacts' },
    };

    const res = http.get(url, params);

    const is2xx = res.status >= 200 && res.status < 300;
    const isConnErr = res.status === 0;

    non2xxErrors.add(!is2xx);
    connectionErrors.add(isConnErr);
    rateLimited.add(res.status === 429);

    if (res.headers['X-Retry-Count'] || res.headers['x-retry-count']) {
      retryLatency.add(res.timings.duration);
    }

    check(res, {
      'direct GET status is 200': (r) => r.status === 200,
    });
  }

  sleep(0.05);
}

export function handleSummary(data) {
  const metrics = data.metrics;
  const p50 = metrics.http_req_duration?.values['p(50)']?.toFixed(2) || '0.00';
  const p95 = metrics.http_req_duration?.values['p(95)']?.toFixed(2) || '0.00';
  const p99 = metrics.http_req_duration?.values['p(99)']?.toFixed(2) || '0.00';

  const rateLimitedPct = ((metrics.rate_limited?.values.rate || 0) * 100).toFixed(2);
  const non2xxPct = ((metrics.non_2xx_errors?.values.rate || 0) * 100).toFixed(2);
  const connErrPct = ((metrics.connection_errors?.values.rate || 0) * 100).toFixed(2);

  console.log(`\n======================================`);
  console.log(`Test Summary`);
  console.log(`======================================`);
  console.log(`Rate Limited:       ${rateLimitedPct}%`);
  console.log(`Non-2xx Errors:     ${non2xxPct}%`);
  console.log(`Connection Errors:  ${connErrPct}%`);
  console.log(`Latency p(50):      ${p50} ms`);
  console.log(`Latency p(95):      ${p95} ms`);
  console.log(`Latency p(99):      ${p99} ms`);
  console.log(`======================================\n`);
}
