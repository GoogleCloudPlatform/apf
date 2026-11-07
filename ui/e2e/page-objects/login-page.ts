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

export class LoginPage {
  readonly page: Page;
  readonly usernameInput: Locator;
  readonly nextButton: Locator;
  readonly passwordInput: Locator;
  readonly loginButton: Locator;
  readonly orgListContainer: Locator;

  constructor(page: Page) {
    this.page = page;
    // Phase 1 elements
    this.usernameInput = page.locator('#username');
    this.nextButton = page.locator('button:has-text("NEXT")');
    
    // Phase 2 (Inline Org Selection) elements
    this.orgListContainer = page.locator('text=Multiple Tenants Found, text=Select which organization');
    
    // Phase 3 elements
    this.passwordInput = page.locator('#password');
    this.loginButton = page.locator('button:has-text("LOG IN")');
  }

  async goto() {
    await this.page.goto('/login');
    await this.page.waitForLoadState('domcontentloaded');
    if (!this.page.url().includes('/login')) {
      await this.page.evaluate(() => {
        localStorage.clear();
        sessionStorage.clear();
      });
      await this.page.goto('/login');
      await this.page.reload();
    }
    await this.usernameInput.waitFor({ state: 'visible' });
  }

  /**
   * Selects an organization from the inline list during the login flow.
   */
  async selectOrgInline(orgName: string) {
    const orgButton = this.page.locator(`button:has-text("${orgName}")`);
    await orgButton.waitFor({ state: 'visible' });
    await orgButton.click();
  }

  /**
   * Performs the full login flow, handling optional inline organization selection.
   */
  async login(username: string, password: string, orgName?: string) {
    // 1. Phase 1: Enter username and click Next
    await this.usernameInput.fill(username);
    await this.nextButton.click();

    // 2. Optional Phase 2: Inline Org Selection or Phase 3: Password Input
    // Wait for either the password input or the org list container to be visible.
    const passwordOrOrg = this.passwordInput.or(this.orgListContainer);
    await passwordOrOrg.waitFor({ state: 'visible' });

    if (await this.orgListContainer.isVisible()) {
      if (orgName) {
        await this.selectOrgInline(orgName);
      } else {
        throw new Error("Organization selection is required but no orgName was provided.");
      }
    }

    // 3. Phase 3: Enter password and click Log In
    await this.passwordInput.waitFor({ state: 'visible' });
    await this.passwordInput.fill(password);
    await this.loginButton.click();
  }
}
