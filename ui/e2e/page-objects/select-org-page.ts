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

export class SelectOrgPage {
  readonly page: Page;
  readonly orgDropdownButton: Locator;
  readonly searchInput: Locator;
  readonly enterFoundryButton: Locator;

  constructor(page: Page) {
    this.page = page;
    // Locates the dropdown button under the "Active Organization" label
    this.orgDropdownButton = page.locator('button:has(svg.lucide-chevron-down)').first();
    this.searchInput = page.locator('input[placeholder="Search organizations..."]');
    this.enterFoundryButton = page.locator('button:has-text("ENTER FOUNDRY")');
  }

  async goto() {
    await this.page.goto('/select-org');
  }

  /**
   * Opens the dropdown and selects the organization by name.
   */
  async selectOrganization(orgName: string) {
    await this.page.waitForSelector('text=Active Organization', { timeout: 10000 });
    await this.orgDropdownButton.waitFor({ state: 'visible' });
    await this.orgDropdownButton.click();
    
    const orgOption = this.page.locator('button').filter({ hasText: orgName }).filter({ hasNot: this.page.locator('svg.lucide-chevron-down') }).first();
    await orgOption.waitFor({ state: 'visible' });
    await orgOption.click();
    await this.submit();
  }

  async submit() {
    await this.enterFoundryButton.click();
  }
}
