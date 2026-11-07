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
 * k6 Authentication Helper
 * Supports Dual-Mode Authentication:
 * 1. OIDC / Firebase REST API / Token authentication for Staging / Prod
 * 2. Mock JWT & BYPASS_IAP_VALIDATION headers for Local Dev
 */

import http from 'k6/http';

export const BYPASS_IAP_VALIDATION = __ENV.BYPASS_IAP_VALIDATION || 'true';
export const MOCK_AUTH_HEADER = __ENV.MOCK_AUTH_HEADER || 'Bearer iap-authenticated';
export const PERF_SA_EMAIL = __ENV.PERF_SA_EMAIL || __ENV.STAGING_PERF_SA_EMAIL || 'perf-testing@example.iam.gserviceaccount.com';

export function getAuthHeaders(env = __ENV.TARGET_ENV || 'local') {
  if (env === 'staging' || env === 'prod') {
    const bearerToken = __ENV.STAGING_AUTH_TOKEN || __ENV.STAGING_BEARER_TOKEN || __ENV.FIREBASE_ID_TOKEN || __ENV.IAP_TOKEN;
    const userEmail = __ENV.TEST_USER_EMAIL || PERF_SA_EMAIL;

    if (!bearerToken) {
      console.warn(`[AUTH WARN] STAGING_AUTH_TOKEN is empty. Requests to staging (${env}) will fail IAP authentication unless STAGING_AUTH_TOKEN=<token> is provided.`);
    }

    const formattedToken = bearerToken ? (bearerToken.startsWith('Bearer ') ? bearerToken : `Bearer ${bearerToken}`) : '';

    const headers = {
      'Proxy-Authorization': formattedToken,
      'X-Organization-Id': __ENV.TEST_ORG_ID || 'apf-stg-org-001',
      'X-Goog-Authenticated-User-Email': userEmail,
    };

    if (__ENV.STAGING_SESSION_TOKEN) {
      headers['Authorization'] = __ENV.STAGING_SESSION_TOKEN.startsWith('Bearer ') ? __ENV.STAGING_SESSION_TOKEN : `Bearer ${__ENV.STAGING_SESSION_TOKEN}`;
    }

    return headers;
  }

  return {
    'Authorization': MOCK_AUTH_HEADER,
    'X-Bypass-IAP-Validation': BYPASS_IAP_VALIDATION,
    'x-goog-authenticated-user-email': __ENV.TEST_USER_EMAIL || 'perf-test@agentic-foundry.com',
    'X-Organization-Id': __ENV.TEST_ORG_ID || 'mock-tenant-123',
    'X-Dev-User-Id': 'dev-perf-user-001',
  };
}

export default {
  BYPASS_IAP_VALIDATION,
  MOCK_AUTH_HEADER,
  PERF_SA_EMAIL,
  getAuthHeaders,
};
