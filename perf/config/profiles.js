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
 * k6 Configuration Profiles & Environment Target Resolver
 * Supports Local, Staging, and Production execution profiles.
 */

export const TARGET_ENV = __ENV.TARGET_ENV || 'local';

export const K6_TARGET_URL = __ENV.K6_TARGET_URL || __ENV.STAGING_URL || (
  TARGET_ENV === 'staging'
    ? 'https://staging.deltaproductfoundry.app'
    : TARGET_ENV === 'prod'
    ? 'https://deltaproductfoundry.app'
    : 'http://localhost:8081'
);

export const K6_P95_THRESHOLD_MS = parseInt(
  __ENV.K6_P95_THRESHOLD_MS || (TARGET_ENV === 'staging' ? '800' : '500'),
  10
);

export const httpThresholds = {
  http_req_duration: [`p(95)<${K6_P95_THRESHOLD_MS}`],
  http_req_failed: ['rate<0.01'],
};

export const wsThresholds = {
  ws_frame_latency_ms: ['p(95)<300'],
  ws_connecting_errors: ['count<1'],
};

export const thresholds = Object.assign({}, httpThresholds);

export const rampProfiles = {
  local: [
    { duration: '30s', target: 5 },
    { duration: '1m', target: 5 },
    { duration: '10s', target: 0 },
  ],
  staging_smoke: [
    { duration: '30s', target: 5 },
    { duration: '1m', target: 5 },
    { duration: '10s', target: 0 },
  ],
  staging: [
    { duration: '1m', target: 50 },
    { duration: '3m', target: 200 },
    { duration: '5m', target: 200 },
    { duration: '1m', target: 0 },
  ],
  stress: [
    { duration: '2m', target: 50 },
    { duration: '5m', target: 200 },
    { duration: '3m', target: 0 },
  ],
};

export function getProfile(env = TARGET_ENV) {
  return {
    targetUrl: K6_TARGET_URL,
    thresholds,
    stages: rampProfiles[env] || rampProfiles[TARGET_ENV] || rampProfiles.local,
  };
}

export default {
  TARGET_ENV,
  K6_TARGET_URL,
  K6_P95_THRESHOLD_MS,
  httpThresholds,
  wsThresholds,
  thresholds,
  rampProfiles,
  getProfile,
};
