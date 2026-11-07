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

import { Page, Locator } from '@playwright/test';

export class IamPage {
  readonly page: Page;
  readonly titleHeader: Locator;
  readonly memberDirectoryTab: Locator;
  readonly rolesPermissionsTab: Locator;
  readonly addMemberButton: Locator;
  readonly searchInput: Locator;
  readonly membersTable: Locator;
  readonly rolesTable: Locator;
  readonly accessRestrictedTitle: Locator;
  readonly requiredPermissionBadge: Locator;

  constructor(page: Page) {
    this.page = page;
    this.titleHeader = page.locator('h1:has-text("User Management")');
    this.memberDirectoryTab = page.locator('button:has-text("Member Directory")');
    this.rolesPermissionsTab = page.locator('button:has-text("Roles & Permissions")');
    this.addMemberButton = page.locator('button:has-text("Add Member")');
    this.searchInput = page.locator('input[aria-label="Search users"], input[placeholder*="Search users"]');
    this.membersTable = page.locator('table').first();
    this.rolesTable = page.locator('table').first();
    this.accessRestrictedTitle = page.locator('h1:has-text("Access Restricted")');
    this.requiredPermissionBadge = page.locator('text=user:list');
  }

  async goto() {
    await this.page.goto('/iam');
  }

  async switchTab(tab: 'members' | 'roles') {
    if (tab === 'members') {
      await this.memberDirectoryTab.click();
    } else {
      await this.rolesPermissionsTab.click();
    }
  }

  async openAddMemberModal() {
    await this.addMemberButton.click();
  }

  async searchUsers(query: string) {
    await this.searchInput.fill(query);
  }
}
