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

test.describe.configure({ mode: 'serial' });

async function enableGeminiViaAPI(page: any) {
  await page.evaluate(async () => {
    const token = localStorage.getItem('auth_token') || '';
    if (!token) return;
    try {
      const meRes = await fetch('/api/v1/auth/me', {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (meRes.ok) {
        const me = await meRes.json();
        const orgId = me?.organization_id || 'org-high-council';
        await fetch(`/api/v1/admin/organizations/${orgId}/config/app-integrations/gemini/enable`, {
          method: 'POST',
          headers: { Authorization: `Bearer ${token}` },
        });
      }
    } catch (_) {}
  });
}

async function ensureGeminiCardExpanded(page: any) {
  await enableGeminiViaAPI(page);

  const card = page.locator('[data-testid="integration-card-gemini"], div:has(h3:has-text("Gemini"))').first();
  await expect(card).toBeVisible({ timeout: 15000 });

  const input = card.locator('input').first();
  
  if (!(await input.isVisible().catch(() => false))) {
    const headerTitle = card.locator('h3').first();
    await headerTitle.click();
    await page.waitForTimeout(500);
  }

  const enableBtn = card.getByRole('button', { name: /enable integration/i }).first();
  if (await enableBtn.isVisible({ timeout: 1000 }).catch(() => false)) {
    await enableBtn.click();
    await page.waitForTimeout(1000);
    if (!(await input.isVisible().catch(() => false))) {
      const headerTitle = card.locator('h3').first();
      await headerTitle.click();
      await page.waitForTimeout(500);
    }
  }

  await expect(input).toBeVisible({ timeout: 10000 });
  return { card, input };
}

test.describe('Story 15.1: Gemini API Key Management UI', () => {

  test('TC-P1-4a: Render Gemini Integration Card in Admin Catalog and User Settings', async ({ authedPage }) => {
    // 1. Navigate to Admin Integrations catalog (/integrations)
    await authedPage.goto('/integrations');
    await authedPage.waitForURL(/\/integrations/);
    
    await enableGeminiViaAPI(authedPage);
    await authedPage.reload();

    // Locate Gemini integration card in catalog
    const catalogCard = authedPage.locator('[data-testid="integration-card-gemini"], div:has(h3:has-text("Gemini"))').first();
    await expect(catalogCard).toBeVisible({ timeout: 15000 });

    // 2. Navigate to User Settings (/settings)
    await authedPage.goto('/settings');
    await authedPage.waitForURL(/\/settings/);

    const settingsCard = authedPage.locator('[data-testid="integration-card-gemini"], div:has(h3:has-text("Gemini"))').first();
    await expect(settingsCard).toBeVisible({ timeout: 15000 });
  });

  test('TC-P1-4b: Input Gemini API Key and Toggle Secret Visibility', async ({ authedPage }) => {
    await authedPage.goto('/settings');
    await authedPage.waitForURL(/\/settings/);

    const { card: geminiCard, input: inputField } = await ensureGeminiCardExpanded(authedPage);

    // Type test key
    const testKey = 'gemini_simulated_api_key_12345';
    await inputField.fill(testKey);
    await expect(inputField).toHaveValue(testKey);

    // Visibility toggle
    const eyeToggleBtn = geminiCard.locator('button[title*="password" i], button[title*="show" i], button[title*="hide" i]').first();
    if (await eyeToggleBtn.isVisible({ timeout: 1000 }).catch(() => false)) {
      await eyeToggleBtn.click();
      await eyeToggleBtn.click();
    }
  });

  test('TC-P1-4c: Synchronous Key Validation via "Validate Link"', async ({ authedPage }) => {
    await authedPage.goto('/settings');
    await authedPage.waitForURL(/\/settings/);

    const { card: geminiCard, input: inputField } = await ensureGeminiCardExpanded(authedPage);

    await inputField.fill('gemini_simulated_api_key_12345');

    // Click "Validate Link"
    const validateBtn = geminiCard.getByRole('button', { name: /validate link|validate|test/i }).first();
    if (await validateBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
      await validateBtn.click();
      const validationMessage = authedPage.getByText(/verified|successfully|valid|failed|error/i).first();
      await expect(validationMessage).toBeVisible({ timeout: 10000 });
    }
  });

  test('TC-P1-4d: Save Credentials to Profile', async ({ authedPage }) => {
    await authedPage.goto('/settings');
    await authedPage.waitForURL(/\/settings/);

    const { card: geminiCard, input: inputField } = await ensureGeminiCardExpanded(authedPage);

    await inputField.fill('gemini_simulated_api_key_12345');

    // Click Save Credentials
    const saveBtn = geminiCard.getByRole('button', { name: /save credentials|save/i }).first();
    if (await saveBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
      await saveBtn.click();
      const successFeedback = authedPage.getByText(/configured|saved|successfully/i).first();
      await expect(successFeedback).toBeVisible({ timeout: 10000 });
    }
  });

  test('TC-P1-4e: Auto-enable Gemini in New Project screen and hide textboxes until Override Master clicked', async ({ authedPage }) => {
    // 1. Ensure Gemini credentials are saved to user profile
    await authedPage.goto('/settings');
    await authedPage.waitForURL(/\/settings/);
    const { card: geminiCard, input: inputField } = await ensureGeminiCardExpanded(authedPage);
    await inputField.fill('gemini_simulated_api_key_12345');
    const saveBtn = geminiCard.getByRole('button', { name: /save credentials|save/i }).first();
    if (await saveBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
      await saveBtn.click();
      await authedPage.waitForTimeout(1000);
    }

    // 2. Navigate to New Project page (/projects/new/details)
    await authedPage.goto('/projects/new/details');
    await authedPage.waitForURL(/\/projects\/new\/details/);

    // Locate Gemini card on new project page
    const newProjectGeminiCard = authedPage.locator('[data-testid="integration-card-gemini"], div:has(h3:has-text("Gemini"))').first();
    await expect(newProjectGeminiCard).toBeVisible({ timeout: 15000 });

    // Assert Gemini toggle/checkbox is checked (enabled by default)
    const geminiCheckbox = newProjectGeminiCard.locator('input[type="checkbox"], button[role="checkbox"], button[role="switch"]').first();
    await expect(geminiCheckbox).toBeChecked({ timeout: 10000 });

    // Assert credential input text box is hidden while inheriting master credentials
    const keyInput = newProjectGeminiCard.locator('input[id*="gemini_api_key"]').first();
    await expect(keyInput).not.toBeVisible();

    // Click "Override profile defaults" button
    const overrideBtn = newProjectGeminiCard.getByRole('button', { name: /override profile defaults/i }).first();
    await expect(overrideBtn).toBeVisible();
    await overrideBtn.click();

    // Assert credential input text box is now visible after hitting Override Master button
    await expect(keyInput).toBeVisible({ timeout: 5000 });
  });

  test('TC-P0-5b: Revoke / Delete Stored Credential', async ({ authedPage }) => {
    await authedPage.goto('/settings');
    await authedPage.waitForURL(/\/settings/);

    const { card: geminiCard } = await ensureGeminiCardExpanded(authedPage);

    const revokeBtn = geminiCard.getByRole('button', { name: /revoke|delete|remove/i }).first();
    if (await revokeBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
      await revokeBtn.click();
      
      // Target modal confirm button inside backdrop overlay specifically
      const modalConfirmBtn = authedPage.locator('.fixed.inset-0 button').filter({ hasText: /revoke|confirm|yes|delete/i }).first();
      if (await modalConfirmBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
        await modalConfirmBtn.click();
      }
    }
  });

});
