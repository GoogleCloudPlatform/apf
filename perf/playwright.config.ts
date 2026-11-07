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

import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './scenarios',
  testMatch: '**/*.spec.ts',
  fullyParallel: true,
  workers: process.env.PLAYWRIGHT_WORKERS ? parseInt(process.env.PLAYWRIGHT_WORKERS, 10) : 10,
  retries: 0,
  timeout: 120000, // 2 minutes per test user
  reporter: [
    ['html', { outputFolder: 'reports/playwright-report', open: 'never' }],
    ['json', { outputFile: 'reports/playwright-summary.json' }],
    ['list'],
  ],
  use: {
    baseURL: process.env.K6_TARGET_URL || process.env.BASE_URL || 'http://localhost:8082',
    extraHTTPHeaders: process.env.PERF_TEST_KEY ? {
      'PERF_TEST_KEY': process.env.PERF_TEST_KEY,
      'X-Perf-Test-Key': process.env.PERF_TEST_KEY,
    } : {},
    trace: 'off',
    video: 'off',
    screenshot: 'only-on-failure',
    headless: true,
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});
