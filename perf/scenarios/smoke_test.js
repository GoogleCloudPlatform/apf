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
 * Baseline Smoke Test Scenario
 * Checks core API endpoints: /api/v1/health, /api/v1/auth/me
 * Exports summary reports to reports/summary.json and reports/summary.html
 */

import { sleep } from 'k6';
import { htmlReport } from 'https://raw.githubusercontent.com/benc-uk/k6-reporter/main/dist/bundle.js';
import { createHttpClient } from '../utils/http_client.js';
import { K6_TARGET_URL, httpThresholds } from '../config/profiles.js';

export const options = {
  vus: 1,
  duration: '10s',
  thresholds: httpThresholds,
};

export default function () {
  const client = createHttpClient(K6_TARGET_URL);

  client.get('/api/v1/health');
  sleep(0.5);

  client.get('/api/v1/auth/me');
  sleep(0.5);
}

export function handleSummary(data) {
  return {
    'reports/summary.html': htmlReport(data),
    'reports/summary.json': JSON.stringify(data, null, 2),
  };
}
