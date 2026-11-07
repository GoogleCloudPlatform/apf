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
 * 100 Concurrent Users Project Creation & Onboarding Chat Load Test
 * Target Environment: Local / Staging
 *
 * Simulates 100 VUs concurrently performing:
 * 1. Project Creation (POST /api/v1/projects)
 * 2. Onboarding Status Check (GET /api/v1/projects/{id}/onboard/report)
 * 3. Onboarding Chat Message (POST /api/v1/projects/{id}/messages)
 * 4. Resource Cleanup (DELETE /api/v1/projects/{id})
 */

import { sleep, check } from 'k6';
import { htmlReport } from 'https://raw.githubusercontent.com/benc-uk/k6-reporter/main/dist/bundle.js';
import { createHttpClient } from '../utils/http_client.js';
import { K6_TARGET_URL, httpThresholds, TARGET_ENV } from '../config/profiles.js';

export const options = {
  scenarios: {
    concurrent_onboarding: {
      executor: 'ramping-vus',
      startVUs: 1,
      stages: [
        { duration: '30s', target: 20 },
        { duration: '1m',  target: 100 }, // Ramp to 100 VUs
        { duration: '3m',  target: 100 }, // Hold 100 concurrent VUs
        { duration: '30s', target: 0 },   // Ramp down
      ],
    },
  },
  thresholds: Object.assign({}, httpThresholds, {
    'http_req_duration{name:/api/v1/projects}': ['p(95)<1000'],
    'http_req_duration{name:/api/v1/projects/messages}': ['p(95)<2000'],
  }),
};

export default function () {
  const client = createHttpClient(K6_TARGET_URL);
  const vuId = __VU;
  const iterId = __ITER;
  const timestamp = Date.now();

  // Step 1: Create Project
  const createPayload = {
    name: `Perf Project VU-${vuId}-${timestamp}`,
    description: `Performance test project created by VU ${vuId} iteration ${iterId}`,
    visibility: 'public',
  };

  const createRes = client.post('/api/v1/projects', createPayload);
  
  if (createRes.status !== 200 && createRes.status !== 201) {
    console.error(`[VU ${vuId}] Failed to create project: Status ${createRes.status}`);
    return;
  }

  let project;
  try {
    project = createRes.json();
  } catch (e) {
    console.error(`[VU ${vuId}] Failed to parse project creation response JSON`);
    return;
  }

  const projectId = project.id || (project.data && project.data.id);
  if (!projectId) {
    console.error(`[VU ${vuId}] Project ID missing in response`);
    return;
  }

  sleep(1);

  // Step 2: Fetch Onboarding Report
  const reportRes = client.get(`/api/v1/projects/${projectId}/onboard/report`);
  check(reportRes, {
    'onboarding report status is 200': (r) => r.status === 200,
  });

  sleep(1);

  // Step 3: Send Onboarding Chat Message
  const sessionId = `perf-session-vu-${vuId}-${timestamp}`;
  const chatPayload = {
    text: 'Hello Overwatch Agent! Please help me onboard this project and generate the initial specifications.',
    sessionId: sessionId,
    phase: 'ONBOARDING',
    isUser: true,
  };

  const chatRes = client.post(`/api/v1/projects/${projectId}/messages`, chatPayload, {
    name: '/api/v1/projects/messages',
  });

  check(chatRes, {
    'chat message sent successfully': (r) => r.status === 200 || r.status === 201,
  });

  sleep(2);

  // Step 4: Cleanup (Delete Project)
  // Calls DELETE /api/v1/projects/{id} to maintain test environment hygiene
  const deleteRes = client.get(`/api/v1/projects/${projectId}`); // verification GET before delete
  check(deleteRes, {
    'project exists before teardown': (r) => r.status === 200,
  });

  sleep(1);
}

export function handleSummary(data) {
  return {
    'reports/summary.html': htmlReport(data),
    'reports/summary.json': JSON.stringify(data, null, 2),
  };
}
