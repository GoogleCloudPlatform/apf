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

import { test, expect } from '../../fixtures/login-fixture';
import { DashboardPage } from '../../page-objects/dashboard-page';
import { LoginPage } from '../../page-objects/login-page';
import { SelectOrgPage } from '../../page-objects/select-org-page';

test.describe('Dashboard Layout & Navigation', () => {

  test('should load the dashboard layout with left sidebar and branding', async ({ authedPage }) => {
    const dashboardPage = new DashboardPage(authedPage);

    // 1. Navigate to the main dashboard
    await dashboardPage.goto();

    // 2. Verify Left Sidebar is visible
    await expect(dashboardPage.leftBar).toBeVisible();

    // 3. Verify Logo / Branding is displayed
    await expect(dashboardPage.logo).toBeVisible();

    // 4. Verify primary navigation items are visible
    await expect(dashboardPage.projectsNavItem).toBeVisible();
    await expect(dashboardPage.workflowsNavItem).toBeVisible();
    await expect(dashboardPage.settingsNavItem).toBeVisible();
  });

  test('should navigate to Workflows section via left sidebar', async ({ authedPage }) => {
    const dashboardPage = new DashboardPage(authedPage);
    await dashboardPage.goto();

    // Click Workflows navigation item
    await dashboardPage.navigateToWorkflows();

    // Assert URL redirect to /workflows
    await expect(authedPage).toHaveURL(/\/workflows/);
  });

  test('should navigate to IAM section via left sidebar', async ({ authedPage }) => {
    const dashboardPage = new DashboardPage(authedPage);
    await dashboardPage.goto();

    // Click IAM navigation item
    await dashboardPage.navigateToIAM();

    // Assert URL redirect to /iam
    await expect(authedPage).toHaveURL(/\/iam/);
  });

  test('should navigate to Settings section via left sidebar', async ({ authedPage }) => {
    const dashboardPage = new DashboardPage(authedPage);
    await dashboardPage.goto();

    // Click Settings navigation item
    await dashboardPage.navigateToSettings();

    // Assert URL redirect to /settings
    await expect(authedPage).toHaveURL(/\/settings/);
  });

  test('should be able to see the IAM icon if the user has user:list permission', async ({ page }) => {
    const loginPage = new LoginPage(page);
    const selectOrgPage = new SelectOrgPage(page);
    const dashboardPage = new DashboardPage(page);

    // 1. Log in as a user with user management permissions (org-owner@the-high-council.com)
    await loginPage.goto();
    await loginPage.login('org-owner@the-high-council.com', 'password', process.env.E2E_ORG_NAME || 'The High Council');

    await page.waitForURL(/\/select-org|\/projects/);
    if (page.url().includes('/select-org')) {
      await selectOrgPage.selectOrganization(process.env.E2E_ORG_NAME || 'The High Council');
    }
    await expect(page).toHaveURL(/\/projects/);

    // 2. Assert that the IAM icon / nav item is visible in the left sidebar
    await expect(dashboardPage.iamNavItem).toBeVisible();
  });

  test('should not be able to see the IAM icon if the user does not have user:list permission', async ({ page }) => {
    const dashboardPage = new DashboardPage(page);

    // 1. Authenticate with a restricted user session lacking user:list permission
    await page.goto('/login');
    await page.evaluate(() => {
      localStorage.setItem('auth_token', 'simulated-jwt-token-for-restricted-user');
    });

    await dashboardPage.goto();

    // 2. Assert that the IAM icon / nav item is NOT visible in the left sidebar
    await expect(dashboardPage.iamNavItem).not.toBeVisible();
  });

  test('should not be able to see Projects and Workflows navigation items and corresponding route should also be guarded for no-role user', async ({ page }) => {
    const loginPage = new LoginPage(page);
    const selectOrgPage = new SelectOrgPage(page);
    const dashboardPage = new DashboardPage(page);

    // 1. Log in as no-role@the-high-council.com
    await loginPage.goto();
    await loginPage.login('no-role@the-high-council.com', 'password', process.env.E2E_ORG_NAME || 'The High Council');

    await page.waitForURL(/\/select-org|\/projects/);
    if (page.url().includes('/select-org')) {
      await selectOrgPage.selectOrganization(process.env.E2E_ORG_NAME || 'The High Council');
    }

    // 2. Assert Projects and Workflows navigation items are NOT visible in the left sidebar
    await expect(dashboardPage.projectsNavItem).not.toBeVisible();
    await expect(dashboardPage.workflowsNavItem).not.toBeVisible();

    // 3. Assert /projects route is guarded
    await page.goto('/projects');
    await expect(page.locator('text=Access Restricted')).toBeVisible();
    await expect(page.locator('#tour-catalog-create')).not.toBeVisible();

    // 4. Assert /workflows route is guarded
    await page.goto('/workflows');
    await expect(page.locator('text=Access Restricted')).toBeVisible();
    await expect(page.locator('text=Create Workflow')).not.toBeVisible();
  });

});
