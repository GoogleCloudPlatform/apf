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
import { IamPage } from '../../page-objects/iam-page';
import { LoginPage } from '../../page-objects/login-page';
import { SelectOrgPage } from '../../page-objects/select-org-page';

test.describe('IAM / User Management Page', () => {

  test('should load the IAM page with essential components', async ({ authedPage }) => {
    const iamPage = new IamPage(authedPage);

    // 1. Navigate to IAM page
    await iamPage.goto();

    // 2. Verify header title is visible
    await expect(iamPage.titleHeader).toBeVisible();

    // 3. Verify Member Directory tab is selected by default
    await expect(iamPage.memberDirectoryTab).toBeVisible();

    // 4. Verify Members table or directory list is visible
    await expect(iamPage.membersTable).toBeVisible();
  });

  test('should switch between Member Directory and Roles & Permissions tabs', async ({ authedPage }) => {
    const iamPage = new IamPage(authedPage);

    await iamPage.goto();

    // 1. Switch to Roles & Permissions tab
    await iamPage.switchTab('roles');

    // 2. Verify Roles table is visible and header reflects roles reference
    await expect(authedPage.locator('h2:has-text("Roles & Permissions Reference")')).toBeVisible();
    await expect(iamPage.rolesTable).toBeVisible();

    // 3. Switch back to Member Directory tab
    await iamPage.switchTab('members');

    // 4. Verify Members List header is visible again
    await expect(authedPage.locator('h2:has-text("Members List")')).toBeVisible();
    await expect(iamPage.membersTable).toBeVisible();
  });

  test('should allow authorized users to open the Add Member modal', async ({ authedPage }) => {
    const iamPage = new IamPage(authedPage);

    await iamPage.goto();

    // 1. Verify Add Member button is visible for admin
    await expect(iamPage.addMemberButton).toBeVisible();

    // 2. Click Add Member button
    await iamPage.openAddMemberModal();

    // 3. Assert Add Member modal opens and displays input fields
    const modalHeader = authedPage.locator('h2:has-text("Add New Member"), h3:has-text("Add New Member")');
    await expect(modalHeader).toBeVisible();
  });

  test('should filter members when using search input', async ({ authedPage }) => {
    const iamPage = new IamPage(authedPage);

    await iamPage.goto();

    // 1. Verify search input is visible
    await expect(iamPage.searchInput).toBeVisible();

    // 2. Type non-matching search query
    await iamPage.searchUsers('nonexistentuser999');

    // 3. Verify empty search state message appears
    await expect(authedPage.locator('text=No matching users found')).toBeVisible();

    // 4. Clear search query
    await iamPage.searchInput.fill('');

    // 5. Verify members table is visible again
    await expect(iamPage.membersTable).toBeVisible();
  });

  test('should not be able to navigate to iam page when list user permission is not present (use project-creator user)', async ({ page }) => {
    const { loginAsUser } = await import('../../fixtures/login-fixture');
    const iamPage = new IamPage(page);

    await loginAsUser(page, 'project-creator@the-high-council.com');
    await iamPage.goto();

    await expect(iamPage.accessRestrictedTitle).toBeVisible();
    await expect(iamPage.requiredPermissionBadge).toBeVisible();
    await expect(iamPage.titleHeader).not.toBeVisible();
  });

  test('should be able to navigate to iam page when userlist permission is available (use the user-admin role)', async ({ page }) => {
    const { loginAsUser } = await import('../../fixtures/login-fixture');
    const iamPage = new IamPage(page);

    await loginAsUser(page, 'user-admin@the-high-council.com');
    await iamPage.goto();

    await expect(iamPage.titleHeader).toBeVisible();
    await expect(iamPage.memberDirectoryTab).toBeVisible();
    await expect(iamPage.accessRestrictedTitle).not.toBeVisible();
  });

});
