import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Custom Metrics
const non2xxErrors = new Rate('non_2xx_errors');
const connectionErrors = new Rate('connection_errors');
export const rateLimited = new Rate('rate_limited');
const retryLatency = new Trend('retry_latency');

// Read environment configuration
const targetUrl = __ENV.TARGET_URL || 'http://localhost:8080';
const totalDuration = parseInt(__ENV.TEST_DURATION || '120', 10);

// Calculate stage durations based on total duration (10% ramp-up, 80% steady, 10% ramp-down)
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
  // Alternate between routes: /users (POST) and /contacts (GET)
  const isUsers = Math.random() < 0.5;

  if (isUsers) {
    const url = `${targetUrl}/users`;
    const payload = JSON.stringify({
      name: 'chaos-user',
      timestamp: Date.now(),
    });
    const params = {
      headers: { 'Content-Type': 'application/json' },
      tags: { route: '/users', target: url },
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
      'users status is 200/201': (r) => r.status === 200 || r.status === 201,
    });
  } else {
    const url = `${targetUrl}/contacts`;
    const params = {
      tags: { route: '/contacts', target: url },
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
      'contacts status is 200': (r) => r.status === 200,
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
