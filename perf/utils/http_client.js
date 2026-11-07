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
 * Custom k6 HTTP wrapper with metric tagging, default headers, and diagnostic logging
 */

import http from 'k6/http';
import { check } from 'k6';
import { getAuthHeaders } from './auth.js';

export function createHttpClient(baseUrl, defaultHeaders = {}) {
  const authHeaders = getAuthHeaders();
  const headers = Object.assign({ 'Content-Type': 'application/json' }, authHeaders, defaultHeaders);

  return {
    get(path, tags = {}) {
      const res = http.get(`${baseUrl}${path}`, {
        headers,
        tags: Object.assign({ name: path }, tags),
      });

      if (res.status !== 200 && res.status !== 204) {
        if (res.status === 302 || res.status === 301 || res.status === 401 || res.status === 403) {
          const loc = res.headers ? (res.headers['Location'] || res.headers['location'] || '') : '';
          console.warn(`[IAP AUTH ERROR] Path: ${path}, Status: ${res.status}. Staging endpoint returned IAP challenge (Location: ${loc}). Set STAGING_AUTH_TOKEN with a valid IAP OIDC token.`);
        } else {
          console.error(`[HTTP GET ERROR] Path: ${path}, Status: ${res.status}, Body: ${res.body}`);
        }
      }

      check(res, { 'status is 200/204': (r) => r.status === 200 || r.status === 204 });
      return res;
    },
    post(path, body, tags = {}) {
      const res = http.post(`${baseUrl}${path}`, JSON.stringify(body), {
        headers,
        tags: Object.assign({ name: path }, tags),
      });

      if (res.status !== 200 && res.status !== 201) {
        console.error(`[HTTP POST ERROR] Path: ${path}, Status: ${res.status}, Body: ${res.body}`);
      }

      check(res, { 'status is 200/201': (r) => r.status === 200 || r.status === 201 });
      return res;
    },
  };
}

export default { createHttpClient };
