import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Trend, Rate } from 'k6/metrics';

// --- Configuration ---
const BASE_URL = 'http://localhost'; // Ensure this is accessible, e.g., via minikube tunnel or /etc/hosts
const VUS = 1000; // Number of virtual users
const DURATION = '1m'; // Test duration
const RAMP_UP_TIME = '10s';
const RAMP_DOWN_TIME = '5s';

// --- Metrics ---
const healthCheckTime = new Trend('health_check_time');
const videoListTime = new Trend('video_list_time');
const errorRate = new Rate('errors');

// --- Test Data ---
// No video file needed for /health endpoint

// --- Test Scenarios ---
export const options = {
  stages: [
    { duration: RAMP_UP_TIME, target: VUS }, // Ramp-up to VUS
    { duration: DURATION, target: VUS },     // Stay at VUS for DURATION
    { duration: RAMP_DOWN_TIME, target: 0 }, // Ramp-down to 0
  ],
  thresholds: {
    'http_req_failed': ['rate<0.01'], // http errors should be less than 1%
    'http_req_duration': ['p(95)<500'],  // 95% of requests should be below 500ms (health checks should be fast)
    'health_check_time': ['p(95)<200'],  // 95% of health checks should be below 200ms
    'errors': ['rate<0.01'], // Custom error rate
  },
};

export default function () {
  group('Health Check Endpoint', function () {
    const res = http.get(`${BASE_URL}/health`);

    const checkRes = check(res, {
      'is status 200': (r) => r.status === 200,
      'body contains status healthy': (r) => r.body && r.body.includes('"status":"healthy"'),
    });

    if (!checkRes) {
      errorRate.add(1);
      console.error(`Health check failed: ${res.status} - ${res.body}`);
    }

    healthCheckTime.add(res.timings.duration);
    sleep(1); // Wait 1 second between iterations
  });
}

export function teardown(data) {
  console.log('Test finished.');
}
