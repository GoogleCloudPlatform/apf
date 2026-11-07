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

import { test, expect } from '../../fixtures/project-fixture';
import { LoginPage } from '../../page-objects/login-page';
import { SelectOrgPage } from '../../page-objects/select-org-page';

test.describe('Projects Creation Flow', () => {
  test('should create a new project', async ({ authedPage, projectCleanup }) => {
    // Generate a unique timestamped project name to prevent collisions on the live database
    const uniqueProjectName = `E2E Real Project ${Date.now()}`;
    const projectDescription = 'An automated E2E test project created against the live backend.';

    // Register project for automatic API deletion during test teardown
    projectCleanup(uniqueProjectName);

    // 1. Verify we are on the project catalog page and click "Create Project"
    const skipTourButton = authedPage.locator('button:has-text("Skip Tour")');
    if (await skipTourButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await skipTourButton.click();
    }
    const createButton = authedPage.locator('#tour-catalog-create');
    await expect(createButton).toBeVisible();
    await createButton.click();

    // 2. Verify navigation to the new project form
    await authedPage.waitForURL(/\/projects\/new\/details/);

    // 3. Fill out General Configurations
    const nameInput = authedPage.locator('#projectName');
    const descInput = authedPage.locator('#projectDesc');

    await expect(nameInput).toBeVisible();
    await nameInput.fill(uniqueProjectName);
    await descInput.fill(projectDescription);

    // 4. Submit the project creation form to the real backend
    const saveButton = authedPage.locator('#tour-project-save-button');
    await expect(saveButton).toBeEnabled();
    await saveButton.click();

    // 5. Assert successful redirect back to project catalog after API creation completes
    await expect(authedPage).toHaveURL(/\/projects(\?.*)?$/);

    // 6. Verify the newly created project actually appears in the real catalog
    await expect(authedPage.locator(`text="${uniqueProjectName}"`).first()).toBeVisible();
  });

  test('should prevent saving project without a Gemini API key and display error message', async ({ authedPage }) => {
    // 1. Navigate to project creation form
    const createButton = authedPage.locator('#tour-catalog-create');
    await expect(createButton).toBeVisible();
    await createButton.click();
    await authedPage.waitForURL(/\/projects\/new\/details/);

    // 2. Fill in project name
    const nameInput = authedPage.locator('#projectName');
    await expect(nameInput).toBeVisible();
    await nameInput.fill(`Project No Gemini Key ${Date.now()}`);

    // 3. Find Gemini integration card and override with an empty key
    const geminiCard = authedPage.locator('[data-testid="integration-card-gemini"], div:has(h3:has-text("Gemini"))').first();
    if (await geminiCard.isVisible({ timeout: 5000 }).catch(() => false)) {
      const overrideBtn = geminiCard.getByRole('button', { name: /override profile defaults/i }).first();
      if (await overrideBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
        await overrideBtn.click();
        const keyInput = geminiCard.locator('input[id*="gemini_api_key"]').first();
        if (await keyInput.isVisible({ timeout: 2000 }).catch(() => false)) {
          await keyInput.fill('');
        }
      }
    }

    // 4. Click save button
    const saveButton = authedPage.locator('#tour-project-save-button');
    await saveButton.click();

    // 5. Assert clear error is shown and navigation did not occur
    const errorToast = authedPage.getByText(/Gemini API Key/i).first();
    await expect(errorToast).toBeVisible({ timeout: 5000 });
    expect(authedPage.url()).toContain('/projects/new/details');
  });

  test('User with no project:create permission should not be able to create a project', async ({ page }) => {
    const loginPage = new LoginPage(page);
    const selectOrgPage = new SelectOrgPage(page);

    // 1. Log in as user2 (role: viewer, lacking project:create permission)
    await loginPage.goto();
    await loginPage.login('user2', 'user2', process.env.E2E_ORG_NAME || 'The High Council');

    await page.waitForURL(/\/select-org|\/projects/);
    if (page.url().includes('/select-org')) {
      await selectOrgPage.selectOrganization(process.env.E2E_ORG_NAME || 'The High Council');
    }
    await expect(page).toHaveURL(/\/projects/);

    // 2. Attempt to create a project via API directly as user2
    const forbiddenResponse = await page.evaluate(async () => {
      let token = localStorage.getItem('auth_token') || '';
      let orgId = '';

      if (token) {
        try {
          const meRes = await fetch('/api/v1/auth/me', {
            headers: { Authorization: `Bearer ${token}` },
          });
          if (meRes.ok) {
            const me = await meRes.json();
            orgId = me?.organization_id || '';
          }
        } catch (_) {}
      }

      if (!orgId && token) {
        try {
          const orgsRes = await fetch('/api/v1/auth/organizations', {
            headers: { Authorization: `Bearer ${token}` },
          });
          if (orgsRes.ok) {
            const orgsData = await orgsRes.json();
            const firstOrg = (orgsData.organizations || [])[0];
            if (firstOrg?.id) {
              const selectRes = await fetch('/api/v1/auth/select-organization', {
                method: 'POST',
                headers: {
                  'Content-Type': 'application/json',
                  Authorization: `Bearer ${token}`,
                },
                body: JSON.stringify({ org_id: firstOrg.id }),
              });
              if (selectRes.ok) {
                const selectData = await selectRes.json();
                if (selectData?.token) {
                  token = selectData.token;
                  localStorage.setItem('auth_token', token);
                  orgId = firstOrg.id;
                }
              }
            }
          }
        } catch (_) {}
      }

      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
      };
      if (token) headers['Authorization'] = `Bearer ${token}`;
      if (orgId) headers['X-Organization-Id'] = orgId;

      const res = await fetch('/api/v1/projects', {
        method: 'POST',
        headers,
        body: JSON.stringify({
          name: `Unauthorized Project ${Date.now()}`,
          description: 'Should fail due to missing project:create permission.',
        }),
      });

      return {
        status: res.status,
        bodyText: await res.text(),
      };
    });

    // 3. Assert HTTP 403 Forbidden is returned from the backend due to missing project:create permission
    expect(forbiddenResponse.status).toBe(403);
    expect(forbiddenResponse.bodyText).toContain('Forbidden');
  });
});
