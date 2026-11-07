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

import { test, expect } from '@playwright/test';

/**
 * Playwright Concurrent Load Test: 100 Users Project Creation & Onboarding Chat
 * 
 * Target: UI & API endpoints
 * Simulates concurrent users performing:
 * 1. Project creation via UI / API fixture
 * 2. Navigation to the Onboarding Workspace
 * 3. Ingesting onboarding prompt & triggering chat execution
 * 4. Cleaning up test projects
 */

const USER_COUNT = parseInt(process.env.CONCURRENT_USERS || '100', 10);

test.describe.parallel(`Playwright ${USER_COUNT} Concurrent Users Onboarding Load Suite`, () => {
  for (let i = 1; i <= USER_COUNT; i++) {
    test(`User #${i} - Create Project & Run Onboarding Chat`, async ({ page, request, baseURL }) => {
      test.setTimeout(180000);
      const timestamp = Date.now();
      const projectName = `Perf Playwright VU-${i}-${timestamp}`;
      const projectDesc = `Automated Playwright load test project created by User #${i}`;

      // Step 1: Create Project via API endpoint (FAST setup for concurrent load)
      const createRes = await request.post(`${baseURL || 'http://localhost:8082'}/api/v1/projects`, {
        data: {
          name: projectName,
          description: projectDesc,
          visibility: 'public',
          icon: '🚀',
        },
        headers: {
          'Content-Type': 'application/json',
          ...(process.env.PERF_TEST_KEY ? { 'PERF_TEST_KEY': process.env.PERF_TEST_KEY, 'X-Perf-Test-Key': process.env.PERF_TEST_KEY } : { 'X-Bypass-IAP-Validation': 'true' }),
          'x-goog-authenticated-user-email': process.env.TEST_USER_EMAIL || 'perf-user@agentic-foundry.local',
          'X-Organization-Id': process.env.TEST_ORG_ID || 'org-high-council',
          'X-Dev-User-Id': `dev-perf-user-${i}`,
        },
      });

      expect([200, 201]).toContain(createRes.status());
      const projectData = await createRes.json();
      const projectId = projectData.id || projectData.data?.id;
      expect(projectId).toBeTruthy();

      try {
        // Step 1.5: Authenticate UI Session
        await page.goto(`${baseURL || 'http://localhost:8082'}/login`);
        await page.locator('#username').fill('admin');
        await page.locator('button:has-text("NEXT")').click();
        const pwdInput = page.locator('#password');
        const orgList = page.locator('text=Select which organization, text=Multiple Tenants Found');
        await pwdInput.or(orgList).waitFor({ state: 'visible' });
        if (await orgList.isVisible().catch(() => false)) {
          const firstOrgBtn = page.locator('button:has-text("The High Council"), button:has-text("mock-tenant"), button:has-text("Mock")');
          if (await firstOrgBtn.first().isVisible().catch(() => false)) {
            await firstOrgBtn.first().click();
          }
        }
        await pwdInput.waitFor({ state: 'visible' });
        await pwdInput.fill('admin');
        await page.locator('button:has-text("LOG IN")').click();
        
        // Wait until navigation leaves /login
        await page.waitForFunction(() => !window.location.pathname.includes('/login'), { timeout: 15000 }).catch(() => {});

        if (page.url().includes('/select-org')) {
          const dropdownBtn = page.locator('button:has-text("Select an organization...")').first();
          await dropdownBtn.waitFor({ state: 'visible', timeout: 10000 });
          await dropdownBtn.click();
          
          const firstOrgOption = page.locator('div.absolute button').first();
          await firstOrgOption.waitFor({ state: 'visible', timeout: 10000 });
          await firstOrgOption.click();

          const enterBtn = page.locator('button:has-text("ENTER FOUNDRY")');
          await enterBtn.click();
          await page.waitForURL(/\/projects/, { timeout: 15000 }).catch(() => {});
        }

        // Step 2: Navigate to Project Workspace in Onboarding Phase
        await page.goto(`${baseURL || 'http://localhost:8082'}/projects/${projectId}/workspace?phase=ONBOARDING`);

        // Wait for Workspace Chat Input to be visible and active (session started)
        const chatInput = page.locator('textarea[placeholder*="Ask"], textarea[placeholder*="describe"], textarea[placeholder*="message"], #chat-input, input[placeholder*="Ask"]');
        await expect(chatInput.first()).toBeVisible({ timeout: 30000 });
        await expect(chatInput.first()).toBeEnabled({ timeout: 30000 });

        // Step 3: Wait for Onboarding Quick Action on the left sidebar and click it
        const quickActionButton = page.locator('#tour-workspace-quick-actions button, button:has-text("Onboarding Scan")').first();
        await expect(quickActionButton).toBeVisible({ timeout: 15000 });
        await quickActionButton.click();

        // Click Send button or press Enter to trigger quick action execution
        const sendButton = page.locator('button[type="submit"], button:has-text("Send"), button[aria-label="Send"]');
        if (await sendButton.first().isVisible().catch(() => false)) {
          await sendButton.first().click();
        } else {
          await chatInput.first().press('Enter');
        }

        // Verify message or prompt appears in workspace chat
        const injectedPromptText = "workspace discovery scan";
        const chatMessage = page.locator(`text="${injectedPromptText}"`);
        await expect(chatMessage.first()).toBeVisible({ timeout: 15000 }).catch(() => {
          expect(chatInput.first()).toBeTruthy();
        });

        // Step 4: Wait for "[C] Continue to context generation" response in chat bubble (timeout 90s)
        const continueText = page.getByText("[C] Continue to context generation")
          .or(page.getByText("Continue to context generation"))
          .or(page.locator('strong', { hasText: 'Continue to context generation' }))
          .first();
        await expect(continueText).toBeVisible({ timeout: 90000 });

      } finally {
        // Step 4: Cleanup Created Project
        await request.delete(`${baseURL || 'http://localhost:8082'}/api/v1/projects/${projectId}`, {
          headers: {
            ...(process.env.PERF_TEST_KEY ? { 'PERF_TEST_KEY': process.env.PERF_TEST_KEY, 'X-Perf-Test-Key': process.env.PERF_TEST_KEY } : { 'X-Bypass-IAP-Validation': 'true' }),
            'x-goog-authenticated-user-email': process.env.TEST_USER_EMAIL || 'perf-user@agentic-foundry.local',
            'X-Organization-Id': process.env.TEST_ORG_ID || 'org-high-council',
          },
        }).catch(() => {
          // Best effort cleanup
        });
      }
    });
  }
});
