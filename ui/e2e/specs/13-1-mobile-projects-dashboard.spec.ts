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

/**
 * ATDD Red-Phase Acceptance Tests: Story 13.1 Mobile Projects Dashboard Screen
 * Goal: Verify mobile responsiveness, touch targets >= 48x48px, status badges, and showcase provisioning.
 * Phase: RED (Tests scaffolded with test.skip() for TDD Red-Green-Refactor implementation).
 */

test.describe.skip('Story 13.1: Mobile Projects Dashboard Screen (ATDD Red Phase)', () => {

  test.use({
    viewport: { width: 375, height: 667 }, // iPhone SE / Mobile Viewport
  });

  test('AC 13.1.1: Mobile Breakpoint Grid & Touch Target Compliance (>= 48x48px)', async ({ page }) => {
    // 1. Navigate to Projects Dashboard on mobile breakpoint (< 768px)
    await page.goto('/projects');

    // 2. Verify Project Cards grid formats into a single column
    const projectGrid = page.getByTestId('projects-grid');
    await expect(projectGrid).toBeVisible();
    await expect(projectGrid).toHaveClass(/grid-cols-1/);

    // 3. Verify touch target bounding box is at least 48x48px for project cards & primary buttons
    const primaryButton = page.getByTestId('project-card-primary-cta').first();
    const box = await primaryButton.boundingBox();

    expect(box).not.toBeNull();
    if (box) {
      expect(box.width).toBeGreaterThanOrEqual(48);
      expect(box.height).toBeGreaterThanOrEqual(48);
    }
  });

  test('AC 13.1.2: Dynamic Project Status Badges & Touch CTA Labels', async ({ page }) => {
    await page.goto('/projects');

    // Verify New status project badge & 'Onboard' CTA label
    const newProjectCard = page.getByTestId('project-card-status-new').first();
    await expect(newProjectCard).toBeVisible();
    await expect(newProjectCard.getByRole('button', { name: /onboard/i })).toBeVisible();

    // Verify Onboarded status project badge & 'Open Workspace' CTA label
    const onboardedProjectCard = page.getByTestId('project-card-status-onboarded').first();
    await expect(onboardedProjectCard).toBeVisible();
    await expect(onboardedProjectCard.getByRole('button', { name: /open workspace/i })).toBeVisible();
  });

  test('AC 13.1.3: Showcase Project Mobile Provisioning & Routing', async ({ page }) => {
    await page.goto('/projects');

    const showcaseCard = page.getByTestId('showcase-project-card');
    await expect(showcaseCard).toBeVisible();

    const showcaseCta = showcaseCard.getByRole('button', { name: /provision showcase|onboard/i });
    await expect(showcaseCta).toBeVisible();

    // Tap showcase CTA and verify async loading spinner + navigation to workspace
    await showcaseCta.click();
    await expect(showcaseCard.getByTestId('showcase-loading-spinner')).toBeVisible();

    // Verification URL redirection
    await page.waitForURL(/\/projects\/.*\/workspace/);
    expect(page.url()).toContain('/workspace');
  });

  test('AC 13.1.4: Small Viewport (375px) Zero Horizontal Overflow', async ({ page }) => {
    await page.goto('/projects');

    // Verify header tabs stack/scroll cleanly without document horizontal scroll
    const scrollWidth = await page.evaluate(() => document.documentElement.scrollWidth);
    const clientWidth = await page.evaluate(() => document.documentElement.clientWidth);

    expect(scrollWidth).toBeLessThanOrEqual(clientWidth);
  });
});
