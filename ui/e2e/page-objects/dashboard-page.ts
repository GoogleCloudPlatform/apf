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

export class DashboardPage {
  readonly page: Page;
  readonly leftBar: Locator;
  readonly logo: Locator;
  readonly projectsNavItem: Locator;
  readonly workflowsNavItem: Locator;
  readonly settingsNavItem: Locator;
  readonly iamNavItem: Locator;
  readonly sidebarCollapseButton: Locator;

  constructor(page: Page) {
    this.page = page;
    this.leftBar = page.locator('#tour-leftbar');
    this.logo = page.locator('img[alt="Agentic Foundry Logo"]').first();
    this.projectsNavItem = page.locator('#tour-leftbar-projects');
    this.workflowsNavItem = page.locator('#tour-leftbar-workflows');
    this.settingsNavItem = page.locator('#tour-leftbar-settings');
    this.iamNavItem = page.locator('#tour-leftbar-iam');
    this.sidebarCollapseButton = this.leftBar.locator('button:has(svg.lucide-chevron-left), button:has(svg.lucide-chevron-right)').last();
  }

  async dismissTourIfPresent() {
    const skipTourButton = this.page.locator('button:has-text("Skip Tour")');
    if (await skipTourButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await skipTourButton.click();
    }
  }

  async goto() {
    await this.page.goto('/projects');
    await this.dismissTourIfPresent();
  }

  async navigateToProjects() {
    await this.dismissTourIfPresent();
    await this.projectsNavItem.click();
  }

  async navigateToWorkflows() {
    await this.dismissTourIfPresent();
    await this.workflowsNavItem.click();
  }

  async navigateToSettings() {
    await this.dismissTourIfPresent();
    await this.settingsNavItem.click();
  }

  async navigateToIAM() {
    await this.dismissTourIfPresent();
    await this.iamNavItem.waitFor({ state: 'visible' });
    await this.iamNavItem.click();
  }
}
