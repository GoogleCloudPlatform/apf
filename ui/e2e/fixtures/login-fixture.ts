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

import fs from 'fs';
import path from 'path';
import { test as base, expect, Page } from '@playwright/test';
import { LoginPage } from '../page-objects/login-page';
import { SelectOrgPage } from '../page-objects/select-org-page';

export function getGeminiApiKey(): string {
  if (process.env.GEMINI_API_KEY) return process.env.GEMINI_API_KEY;
  if (process.env.NEXT_PUBLIC_GEMINI_API_KEY) return process.env.NEXT_PUBLIC_GEMINI_API_KEY;

  // Try reading from root config.env or ui/.env.local
  try {
    const configPath = path.resolve(__dirname, '../../../config.env');
    if (fs.existsSync(configPath)) {
      const content = fs.readFileSync(configPath, 'utf-8');
      const match = content.match(/^GEMINI_API_KEY=(.*)$/m);
      if (match && match[1].trim()) {
        return match[1].trim();
      }
    }
  } catch (_) {}

  return '';
}

export async function authenticatePage(page: Page) {
  const loginPage = new LoginPage(page);
  const selectOrgPage = new SelectOrgPage(page);

  // 1. Navigate to login
  await loginPage.goto();

  // 2. Perform Login & Org Selection
  const username = process.env.E2E_USERNAME || 'admin';
  const password = process.env.E2E_PASSWORD || 'admin';
  const orgName = process.env.E2E_ORG_NAME || 'The High Council';

  await loginPage.login(username, password, orgName);

  // 3. Wait for redirect to either /select-org or /projects
  await page.waitForURL(/\/select-org|\/projects/);

  // 4. Handle Organization Selection if redirected to the separate /select-org page
  if (page.url().includes('/select-org')) {
    await selectOrgPage.selectOrganization(orgName);
  }

  // 5. Assert we have successfully entered the app (should be on /projects)
  await expect(page).toHaveURL(/\/projects/);

  const apiKey = getGeminiApiKey();

  // 6. Ensure token is scoped to an active organization/tenant and seed Gemini credentials
  await page.evaluate(async (geminiKey) => {
    let token = localStorage.getItem('auth_token') || '';
    let orgId = '';
    let userId = '';
    if (!token) return;
    try {
      const meRes = await fetch('/api/v1/auth/me', {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (meRes.ok) {
        const me = await meRes.json();
        if (me?.organization_id) {
          orgId = me.organization_id;
        }
        if (me?.subject || me?.username || me?.email) {
          userId = me.subject || me.username || me.email;
        }
      }

      if (!orgId) {
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
                localStorage.setItem('auth_token', selectData.token);
                token = selectData.token;
                orgId = firstOrg.id;
              }
            }
          }
        }
      }

      try {
        const targetUsers = Array.from(
          new Set(['admin', userId, 'org-owner@the-high-council.com', 'developer@agentic-foundry.local'])
        ).filter(Boolean);

        for (const u of targetUsers) {
          await fetch(`/api/v1/users/${u}/apps/gemini/credentials`, {
            method: 'PUT',
            headers: {
              'Content-Type': 'application/json',
              Authorization: `Bearer ${token}`,
              ...(orgId ? { 'X-Organization-Id': orgId } : {}),
            },
            body: JSON.stringify({
              integration_id: 'gemini',
              enabled: true,
              credential_id: 'gemini',
              is_overridden: false,
              fields: [
                {
                  name: 'gemini_api_key',
                  display_name: 'Gemini API Key',
                  value: geminiKey,
                  is_secret: true,
                  enabled: true,
                },
              ],
            }),
          });
        }
      } catch (_) {}
    } catch (_) {}
  }, apiKey);
  await page.reload();
}

export async function loginAsUser(
  page: Page,
  username: string,
  password = 'password',
  orgName = 'The High Council'
) {
  const loginPage = new LoginPage(page);
  const selectOrgPage = new SelectOrgPage(page);

  await loginPage.goto();
  await loginPage.login(username, password, orgName);
  await page.waitForURL(/\/select-org|\/projects/);

  if (page.url().includes('/select-org')) {
    try {
      await selectOrgPage.selectOrganization(orgName);
    } catch (_) {}
  }

  const apiKey = getGeminiApiKey();

  await page.evaluate(async (geminiKey) => {
    let token = localStorage.getItem('auth_token') || '';
    let orgId = '';
    let userId = '';
    if (!token) return;
    try {
      const meRes = await fetch('/api/v1/auth/me', {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (meRes.ok) {
        const me = await meRes.json();
        if (me?.organization_id) {
          orgId = me.organization_id;
        }
        if (me?.subject || me?.username || me?.email) {
          userId = me.subject || me.username || me.email;
        }
      }

      if (!orgId) {
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
                localStorage.setItem('auth_token', selectData.token);
                token = selectData.token;
                orgId = firstOrg.id;
              }
            }
          }
        }
      }

      try {
        const targetUsers = Array.from(
          new Set(['admin', userId, 'org-owner@the-high-council.com', 'developer@agentic-foundry.local'])
        ).filter(Boolean);

        for (const u of targetUsers) {
          await fetch(`/api/v1/users/${u}/apps/gemini/credentials`, {
            method: 'PUT',
            headers: {
              'Content-Type': 'application/json',
              Authorization: `Bearer ${token}`,
              ...(orgId ? { 'X-Organization-Id': orgId } : {}),
            },
            body: JSON.stringify({
              integration_id: 'gemini',
              enabled: true,
              credential_id: 'gemini',
              is_overridden: false,
              fields: [
                {
                  name: 'gemini_api_key',
                  display_name: 'Gemini API Key',
                  value: geminiKey,
                  is_secret: true,
                  enabled: true,
                },
              ],
            }),
          });
        }
      } catch (_) {}
    } catch (_) {}
  }, apiKey);
  await page.reload();
}

// Define the custom fixtures type
type AuthFixtures = {
  authedPage: Page;
};

// Extend the base test to include the 'authedPage' fixture
export const test = base.extend<AuthFixtures>({
  authedPage: async ({ page }, use) => {
    await authenticatePage(page);
    await use(page);
  },
});

export { expect, type Page };
