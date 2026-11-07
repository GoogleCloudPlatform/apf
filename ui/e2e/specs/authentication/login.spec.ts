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
import { LoginPage } from '../../page-objects/login-page';
import { SelectOrgPage } from '../../page-objects/select-org-page';

test.describe('Authentication & Organization Selection', () => {

  test('admin should be able to select an organization', async ({ page }) => {
    const loginPage = new LoginPage(page);
    const selectOrgPage = new SelectOrgPage(page);

    await loginPage.goto();

    // 1. Enter admin username
    await loginPage.usernameInput.fill('admin');
    await loginPage.nextButton.click();

    const targetOrg = 'The High Council';

    // 2. Handle inline organization selection if it appears on the login card
    if (await loginPage.orgListContainer.isVisible()) {
      await loginPage.selectOrgInline(targetOrg);
    }

    // 3. Enter admin password
    await loginPage.passwordInput.waitFor({ state: 'visible' });
    await loginPage.passwordInput.fill('admin');
    await loginPage.loginButton.click();

    // 4. Handle redirect-based organization selection if redirected to /select-org
    await page.waitForURL(/\/(select-org|projects)/);
    if (page.url().includes('/select-org')) {
      await selectOrgPage.selectOrganization(targetOrg);
      await selectOrgPage.submit();
    }

    // 5. Assert redirection to the projects dashboard page
    await expect(page).toHaveURL(/\/projects/);
  });

});
