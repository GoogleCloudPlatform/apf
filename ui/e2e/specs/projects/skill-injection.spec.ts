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

test.describe('Skill Injection E2E Flow', () => {
  test('should create project with apf-test repo, navigate to housekeeping, and inject skills', async ({
    authedPage,
    projectCleanup,
  }) => {
    const projectName = `Skill Injection Test ${Date.now()}`;
    projectCleanup(projectName);

    // Ensure we are on /projects page
    if (!authedPage.url().includes('/projects')) {
      await authedPage.goto('/projects');
    }

    // 1. Dismiss tour if present & click Create Project
    const skipTourButton = authedPage.locator('button:has-text("Skip Tour")');
    if (await skipTourButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await skipTourButton.click();
    }
    const createButton = authedPage.locator('#tour-catalog-create');
    await expect(createButton).toBeVisible({ timeout: 15000 });
    await createButton.click();

    // 2. Fill project name
    await authedPage.waitForURL(/\/projects\/new\/details/);
    const nameInput = authedPage.locator('#projectName');
    await expect(nameInput).toBeVisible();
    await nameInput.fill(projectName);

    // 3. Select repo apf-test
    const repoDropdown = authedPage.locator('#tour-project-github-repo');
    await expect(repoDropdown).toBeVisible();
    await repoDropdown.click();

    const repoSearchInput = authedPage.locator('input[placeholder="Search repositories..."]');
    await expect(repoSearchInput).toBeVisible();
    await repoSearchInput.fill('apf-test');

    // Select exact repo option for apf-test if available, or first matching repo option
    const dropdownOptions = authedPage.locator('#tour-project-github-repo-dropdown button');
    const exactRepoOption = dropdownOptions.filter({ hasText: /^apf-test$/i }).first();
    const repoOption = (await exactRepoOption.isVisible().catch(() => false))
      ? exactRepoOption
      : dropdownOptions.filter({ hasText: 'apf-test' }).first();

    await expect(repoOption).toBeVisible({ timeout: 10000 });
    await repoOption.click();

    // Save project
    const saveButton = authedPage.locator('#tour-project-save-button');
    await expect(saveButton).toBeEnabled();
    await saveButton.click();

    // Handle modal if opened (Permission Warning Modal "Save Anyway" or Duplicate Repo Modal)
    const saveAnywayButton = authedPage.locator('button:has-text("Save Anyway")');
    const goToProjectButton = authedPage.locator('button:has-text("Go to Project")');
    const joinProjectButton = authedPage.locator('button:has-text("Join Project")');

    try {
      await Promise.race([
        saveAnywayButton.waitFor({ state: 'visible', timeout: 8000 }).then(() => saveAnywayButton.click()),
        goToProjectButton.waitFor({ state: 'visible', timeout: 8000 }).then(() => goToProjectButton.click()),
        joinProjectButton.waitFor({ state: 'visible', timeout: 8000 }).then(() => joinProjectButton.click()),
      ]);
    } catch (_) {
      // Modal didn't appear or direct navigation occurred
    }

    // 4. Wait for dashboard or workspace to show newly created project
    await authedPage.waitForURL(/\/projects(\?.*)?$/, { timeout: 15000 });
    const projectCardText = authedPage.locator(`text="${projectName}"`).first();
    await expect(projectCardText).toBeVisible({ timeout: 15000 });

    // 5. Click Onboarding button on project card
    const projectHeading = authedPage.getByRole('heading', { name: projectName });
    await expect(projectHeading).toBeVisible({ timeout: 15000 });

    const cardContainer = authedPage
      .locator('div')
      .filter({ has: projectHeading })
      .filter({ has: authedPage.locator('button') })
      .first();

    const onboardingButton = cardContainer
      .locator('button:has-text("Onboarding"), button:has-text("Open Workspace"), button.tour-project-launch-button')
      .first();

    await expect(onboardingButton).toBeVisible({ timeout: 15000 });
    await onboardingButton.click();

    await authedPage.waitForURL(/\/projects\/.*\/workspace/);

    // 6. Navigate to housekeeping
    const housekeepingBtn = authedPage.locator('#tour-workspace-health');
    await expect(housekeepingBtn).toBeVisible();
    await housekeepingBtn.click();

    await authedPage.waitForURL(/\/projects\/.*\/housekeeping/);

    // 7. Click Inject and Pull Request
    const directInjectAndPRBtn = authedPage.locator('button:has-text("Inject and Pull Request")').first();

    if (await directInjectAndPRBtn.isVisible().catch(() => false)) {
      await directInjectAndPRBtn.click();
    } else {
      // Open "More injection options" dropdown if project is in Read-Only mode
      const moreOptionsBtn = authedPage.locator('button[title="More injection options"]');
      await expect(moreOptionsBtn).toBeVisible({ timeout: 10000 });
      await moreOptionsBtn.click();

      const dropdownInjectAndPR = authedPage.locator('[role="menuitem"]:has-text("Inject and Pull Request"), button:has-text("Inject and Pull Request")').first();
      await expect(dropdownInjectAndPR).toBeVisible({ timeout: 10000 });
      await dropdownInjectAndPR.click();
    }

    // 8. Wait for confirmation toast
    const confirmationToast = authedPage
      .locator('[data-sonner-toast], [role="status"], [role="alert"]')
      .filter({ hasText: /skills|injected|success/i })
      .first();
    await expect(confirmationToast).toBeVisible({ timeout: 15000 });
  });
});
