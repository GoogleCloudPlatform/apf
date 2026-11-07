/**
 * Copyright 2026 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/**
 * Story 14.3: Distributed Multi-Cluster Load Test Harness (200 Concurrent Users)
 * Target Environment: Staging (https://staging.deltaproductfoundry.app)
 *
 * Execution Profile:
 * - 2m ramp up to 50 VUs
 * - 5m ramp up to 200 VUs
 * - 5m sustained hold at 200 VUs
 * - 1m ramp down to 0 VUs
 */

import { sleep } from 'k6';
import { htmlReport } from 'https://raw.githubusercontent.com/benc-uk/k6-reporter/main/dist/bundle.js';
import { createHttpClient } from '../utils/http_client.js';
import { K6_TARGET_URL, httpThresholds, rampProfiles, TARGET_ENV } from '../config/profiles.js';

export const options = {
  stages: rampProfiles[TARGET_ENV] || [
    { duration: '2m', target: 50 },
    { duration: '5m', target: 200 },
    { duration: '5m', target: 200 },
    { duration: '1m', target: 0 },
  ],
  thresholds: httpThresholds,
};

export default function () {
  const client = createHttpClient(K6_TARGET_URL);

  // 1. Health check endpoint
  client.get('/api/v1/health');
  sleep(1);

  // 2. User identity check
  client.get('/api/v1/auth/me');
  sleep(1);
}

export function handleSummary(data) {
  return {
    'reports/summary.html': htmlReport(data),
    'reports/summary.json': JSON.stringify(data, null, 2),
  };
}
