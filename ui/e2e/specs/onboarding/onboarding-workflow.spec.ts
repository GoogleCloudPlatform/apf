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
import { WorkspacePage } from '../../page-objects/workspace-page';

test.describe('Onboarding Workflow in Workspace', () => {

  test('should load workspace in Onboarding phase and display Onboarding quick actions', async ({ authedPage, sharedProject }) => {
    const workspacePage = new WorkspacePage(authedPage);
    await workspacePage.goto(sharedProject.id, 'ONBOARDING');

    // 1. Verify Header & Onboarding Phase Tab
    await expect(workspacePage.header).toBeVisible();
    await expect(workspacePage.onboardingPhaseTab).toBeVisible();

    // 2. Verify Sidebar is open with Explorer panel
    await expect(workspacePage.sidebar).toBeVisible();

    // 3. Verify Quick Actions section and Onboarding Scan action button
    await expect(workspacePage.quickActionsSection).toBeVisible();
    await expect(workspacePage.onboardingScanButton).toBeVisible();
  });

  test('should inject Onboarding Scan quick action into chat input', async ({ authedPage, sharedProject }) => {
    const workspacePage = new WorkspacePage(authedPage);
    await workspacePage.goto(sharedProject.id, 'ONBOARDING');

    // 1. Verify chat input is initially empty
    await expect(workspacePage.chatInput).toBeVisible();

    // 2. Click "Onboarding Scan" quick action button
    await workspacePage.clickQuickAction('Onboarding Scan');

    // 3. Assert the prompt text is injected into chat input
    await expect(workspacePage.chatInput).toHaveValue(/Generate project context by conducting a complete workspace discovery scan/i);
  });

  test('should execute health check diagnostic in onboarding workspace', async ({ authedPage, sharedProject }) => {
    const workspacePage = new WorkspacePage(authedPage);
    await workspacePage.goto(sharedProject.id, 'ONBOARDING');

    // 1. Verify health check button is present
    await expect(workspacePage.healthCheckButton).toBeVisible();

    // 2. Trigger health check
    await workspacePage.runHealthCheck();

    // 3. Assert toast notification indicates health check completed successfully
    const successToast = authedPage.locator('text=Health check completed');
    await expect(successToast).toBeVisible({ timeout: 10000 });
  });



  test('should submit Onboarding Scan quick action and wait up to 5 minutes for project-context.md under Generated Artifacts > Project Context', async ({ authedPage, sharedProject }) => {
    // 1. Set 5-minute timeout for this long-running agent test
    test.setTimeout(300000);

    const workspacePage = new WorkspacePage(authedPage);
    await workspacePage.goto(sharedProject.id, 'ONBOARDING');

    // 2. Click "Onboarding Scan" quick action button (injects prompt into chat input)
    await workspacePage.clickQuickAction('Onboarding Scan');
    await expect(workspacePage.chatInput).toHaveValue(/Generate project context/i);

    // 3. Wait for the send button to be enabled (after session initialization completes) and click it
    await expect(workspacePage.sendButton).toBeEnabled({ timeout: 15000 });
    await workspacePage.sendButton.click();

    // 4. Verify Generated Artifacts section is visible
    await expect(workspacePage.artifactsSection).toBeVisible();

    // 5. Locate Project Context folder in Generated Artifacts
    const projectContextFolder = workspacePage.artifactsSection.locator('text=PROJECT CONTEXT');
    await expect(projectContextFolder).toBeVisible();

    // 6. Wait for up to 5 minutes (300,000 ms) for project-context.md to appear under Project Context
    const projectContextFile = workspacePage.artifactsSection.locator('text=project-context.md');

    await expect(async () => {
      // If the Project Context folder is collapsed, click to expand it
      if (await projectContextFolder.isVisible() && !(await projectContextFile.isVisible())) {
        await projectContextFolder.click().catch(() => {});
      }
      await expect(projectContextFile).toBeVisible();
    }).toPass({
      timeout: 300000,
      intervals: [2000, 5000, 10000],
    });
  });

  test('should launch onboarding workspace directly from project card in catalog', async ({ authedPage, createProject }) => {
    // 1. Create a fresh project via API
    const project = await createProject(`Onboarding Catalog Test ${Date.now()}`);
    const workspacePage = new WorkspacePage(authedPage);

    // 2. Navigate to project catalog
    await authedPage.goto('/projects');

    // 3. Locate the created project card
    const card = authedPage.locator('.tour-project-card').filter({ hasText: project.name });
    await expect(card).toBeVisible();

    // 4. Click the onboarding launch button on the card
    const launchButton = card.locator('.tour-project-launch-button');
    await expect(launchButton).toBeVisible();
    await launchButton.click();

    // 5. Assert redirect to workspace in Onboarding phase
    await authedPage.waitForURL(new RegExp(`/projects/${project.id}/workspace`));
    await expect(workspacePage.onboardingPhaseTab).toBeVisible();
    await expect(workspacePage.onboardingScanButton).toBeVisible();
  });
});
