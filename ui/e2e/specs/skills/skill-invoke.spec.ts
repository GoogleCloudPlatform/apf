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

import { test, expect } from '../../fixtures/project-fixture';
import type { Page, Locator } from '@playwright/test';
import { FailureMonitor } from '../../helpers/failure-monitor';

export interface SkillInvokeConfig {
  tabName: string;          // e.g. 'ONBOARDING', 'ANALYSIS', 'PLANNING', 'SOLUTIONING', 'IMPLEMENTATION'
  quickActionName: string;  // e.g. 'Help', 'Brainstorming', 'Create PRD', 'Architecture', 'Sprint Planning'
  projectName?: string;
  failOnDetectedError?: boolean;
  testTimeoutMs?: number;        // Overall Playwright test timeout (default: 180,000ms / 3 mins)
  turnTimeoutMs?: number;        // Single agent turn timeout (default: 45,000ms / 45 secs)
  stableWindowMs?: number;       // Stability wait window (default: 5,000ms / 5 secs)
}

export type SkillInvokerConfig = SkillInvokeConfig;

const MESSAGE_SELECTOR = 'div.mr-auto .rounded-2xl .prose, div.mr-auto .prose, div.mr-auto .rounded-2xl, [data-testid="chat-message"], .message';

/**
 * Helper function to determine if the initial agent turn is complete:
 * 1. Evaluates display state (no active streaming ThinkingBlock, stop button, or spinner).
 * 2. Waits for chat pane display content to remain unchanged for the stability window.
 * 3. Returns the final displayed agent response text.
 */
async function waitForAgentTurnToComplete(
  page: Page,
  chatPane: Locator,
  failureMonitor: FailureMonitor,
  timeoutMs = 45000,
  stableWindowMs = 5000
): Promise<string> {
  const startTime = Date.now();
  let lastDisplayText = '';
  let stableCount = 0;
  const pollIntervalMs = 1000;
  const requiredStableCycles = Math.max(2, Math.ceil(stableWindowMs / pollIntervalMs));

  console.log(`[E2E Skill Invoke] Waiting for agent turn completion (Display stability window: ${stableWindowMs}ms)...`);

  while (Date.now() - startTime < timeoutMs) {
    await page.waitForTimeout(pollIntervalMs);

    // Record all messages for failure monitoring
    const allMessages = chatPane.locator('.prose, [data-testid="chat-message"], .message, div.rounded-2xl');
    const allCount = await allMessages.count();
    for (let i = 0; i < allCount; i++) {
      const text = (await allMessages.nth(i).textContent()) || '';
      if (text.trim()) {
        failureMonitor.recordAndCheck('agent', text.trim());
      }
    }

    // Locate agent response messages specifically
    const agentMessages = chatPane.locator(MESSAGE_SELECTOR);
    let count = await agentMessages.count();

    if (count === 0) {
      const fallbackItems = chatPane.locator('.prose, div.rounded-2xl');
      count = await fallbackItems.count();
    }

    if (count > 0) {
      const currentDisplayText = (await agentMessages.nth(count - 1).textContent()) || '';

      const hasSpinners = await chatPane.locator('.animate-spin').first().isVisible().catch(() => false);
      const hasStopBtn = await chatPane.locator('button[title*="Stop active agent processing"], button[title*="Stop"]').first().isVisible().catch(() => false);
      const hasThinkingText = await chatPane.locator('text=/is processing|Agent is thinking|Initializing agent reasoning/i').first().isVisible().catch(() => false);
      const isThinking = hasSpinners || hasStopBtn || hasThinkingText;

      const optionButtons = chatPane.locator('button.option-btn, button[data-option]');
      const hasOptions = (await optionButtons.count()) > 0;

      if (hasOptions && !isThinking) {
        console.log('[E2E Skill Invoke] Interactive option buttons detected in chat pane.');
        return currentDisplayText;
      }

      if (!isThinking && currentDisplayText === lastDisplayText && currentDisplayText.trim().length > 0) {
        stableCount++;
        if (stableCount >= requiredStableCycles) {
          console.log(`[E2E Skill Invoke] Turn completion detected! Display content stable for ${stableWindowMs / 1000}s. Count: ${count}`);
          return currentDisplayText;
        }
      } else {
        lastDisplayText = currentDisplayText;
        stableCount = 0;
      }
    } else {
      const fullText = (await chatPane.textContent()) || '';
      if (fullText.length > 50 && !fullText.includes('Message thread is empty')) {
        const hasSpinners = await chatPane.locator('.animate-spin').first().isVisible().catch(() => false);
        const hasStopBtn = await chatPane.locator('button[title*="Stop active agent processing"], button[title*="Stop"]').first().isVisible().catch(() => false);
        const hasThinkingText = await chatPane.locator('text=/is processing|Agent is thinking|Initializing agent reasoning/i').first().isVisible().catch(() => false);
        const isThinking = hasSpinners || hasStopBtn || hasThinkingText;

        if (!isThinking && fullText === lastDisplayText) {
          stableCount++;
          if (stableCount >= requiredStableCycles) {
            console.log(`[E2E Skill Invoke] Turn completion detected via full text fallback!`);
            return fullText;
          }
        } else {
          lastDisplayText = fullText;
          stableCount = 0;
        }
      }
    }
  }

  console.log('[E2E Skill Invoke] Warning: Turn timeout reached before display stability was confirmed.');
  return lastDisplayText;
}

/**
 * Parameterized E2E test runner for quick-action skill invocation.
 * Verifies that:
 * 1. The skill is invoked from the UI quick action and returns non-empty response.
 * 2. No redirection of skill / phase switch is suggested for skills in their canonical tab.
 */
export function runSkillInvokeE2ETest(config: SkillInvokeConfig) {
  test(`Skill Invocation - Tab: [${config.tabName}], Action: [${config.quickActionName}]`, async ({
    authedPage,
    projectCleanup,
  }, testInfo) => {
    test.setTimeout(config.testTimeoutMs || 180000);

    const failureMonitor = new FailureMonitor();
    const randomSlug = Math.random().toString(36).substring(2, 8);
    const projectName = config.projectName
      ? `${config.projectName}-${randomSlug}`
      : `Skill Invoke ${config.tabName} ${Date.now()}-${randomSlug}`;
    projectCleanup(projectName);

    // 1. Override workflows at the UI network tier so no phase is locked and all tabs are accessible
    await authedPage.route('**/api/v1/workflows*', async (route) => {
      try {
        const response = await route.fetch();
        const json = await response.json();
        if (Array.isArray(json)) {
          const modified = json.map((wf: any) => {
            if (wf.framework === 'BMAD' || wf.phases) {
              return {
                ...wf,
                dependencies: {
                  ONBOARDING: [],
                  ANALYSIS: [],
                  PLANNING: [],
                  SOLUTIONING: [],
                  IMPLEMENTATION: [],
                },
              };
            }
            return wf;
          });
          await route.fulfill({ json: modified });
          return;
        }
        await route.fulfill({ json });
      } catch (_) {
        await route.continue();
      }
    });

    // 2. Ensure we are on /projects page
    if (!authedPage.url().includes('/projects')) {
      await authedPage.goto('/projects');
    }

    // Dismiss tour if present
    const skipTourButton = authedPage.locator('button:has-text("Skip Tour")');
    if (await skipTourButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await skipTourButton.click();
    }

    // 3. Create Project
    const createButton = authedPage.locator('#tour-catalog-create');
    await expect(createButton).toBeVisible({ timeout: 15000 });
    await createButton.click();

    await authedPage.waitForURL(/\/projects\/new\/details/);

    const nameInput = authedPage.locator('#projectName');
    await expect(nameInput).toBeVisible();
    await nameInput.fill(projectName);

    const descInput = authedPage.locator('#projectDesc');
    if (await descInput.isVisible().catch(() => false)) {
      await descInput.fill('Skill invoke E2E test project.');
    }

    // Uncheck GitHub integration if present/checked
    const githubToggle = authedPage.locator('#tour-project-toggle-github, input[type="checkbox"][id*="github"]').first();
    if (await githubToggle.isVisible({ timeout: 5000 }).catch(() => false)) {
      if (await githubToggle.isChecked().catch(() => false)) {
        await githubToggle.uncheck().catch(async () => {
          await githubToggle.click();
        });
      }
    }

    const saveButton = authedPage.locator('#tour-project-save-button');
    await expect(saveButton).toBeEnabled();
    await saveButton.click();

    await authedPage.waitForURL(/\/projects(\?.*)?$/, { timeout: 15000 });
    const projectHeading = authedPage.getByRole('heading', { name: projectName });
    await expect(projectHeading).toBeVisible({ timeout: 15000 });

    // Open Workspace for this specific project card
    const projectCard = authedPage
      .locator('.tour-project-card, [data-testid="project-card"]')
      .filter({ has: projectHeading })
      .first();

    await expect(projectCard).toBeVisible({ timeout: 15000 });

    const openWorkspaceBtn = projectCard
      .locator('button:has-text("Onboarding"), button:has-text("Open Workspace"), button.tour-project-launch-button')
      .first();

    if (await openWorkspaceBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
      await openWorkspaceBtn.click();
    } else {
      await projectCard.click();
    }

    await authedPage.waitForURL(/\/projects\/.*\/workspace/, { timeout: 15000 });

    // 4. Force target phase on UI tier and Navigate to Target Phase Tab
    const currentWorkspaceUrl = authedPage.url();
    const workspaceIdMatch = currentWorkspaceUrl.match(/\/projects\/([^/?#]+)\/workspace/);
    if (workspaceIdMatch && workspaceIdMatch[1]) {
      const pId = workspaceIdMatch[1];
      await authedPage.goto(`/projects/${pId}/workspace?phase=${config.tabName.toUpperCase()}&force=true`);
      await authedPage.waitForURL(new RegExp(`/projects/${pId}/workspace`), { timeout: 15000 });
    }

    const normalizedTab = config.tabName.toLowerCase();
    const phaseTab = authedPage.locator(`#tour-workspace-phase-${normalizedTab}`);

    if (await phaseTab.isVisible({ timeout: 2000 }).catch(() => false)) {
      await phaseTab.click();
    } else {
      const phasesNav = authedPage.locator('#tour-workspace-phases');
      const dropdownTrigger = phasesNav.locator('button').first();
      if (await dropdownTrigger.isVisible({ timeout: 2000 }).catch(() => false)) {
        await dropdownTrigger.click();
        await authedPage.waitForTimeout(300);
        const menuItem = authedPage.locator('[role="menuitem"]').filter({
          hasText: new RegExp(config.tabName.slice(0, 4), 'i'),
        }).first();
        if (await menuItem.isVisible({ timeout: 2000 }).catch(() => false)) {
          await menuItem.click();
        }
      } else {
        const genericTab = authedPage.locator(`[role="tab"]:has-text("${config.tabName}"), button:has-text("${config.tabName}")`).first();
        if (await genericTab.isVisible({ timeout: 3000 }).catch(() => false)) {
          await genericTab.click();
        }
      }
    }

    // If on IMPLEMENTATION tab, ensure the "Chat & Quick Actions" sub-tab is active
    if (config.tabName.toUpperCase() === 'IMPLEMENTATION') {
      const chatSubTab = authedPage.locator('button:has-text("Chat & Quick Actions")');
      if (await chatSubTab.isVisible({ timeout: 3000 }).catch(() => false)) {
        await chatSubTab.click();
      }
    }

    const chatPane = authedPage.locator('#tour-workspace-chat');
    await expect(chatPane).toBeVisible({ timeout: 15000 });

    const textarea = chatPane.locator('textarea').first();

    // 5. Ensure active session exists and is ready
    const isTextareaReady = await textarea.isVisible({ timeout: 10000 }).catch(() => false);
    if (!isTextareaReady) {
      // Force session creation via New Session button in Workspace Explorer or via API fallback
      const newSessionBtn = authedPage.locator('#tour-workspace-chat-section button:has-text("New")');
      if (await newSessionBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
        await newSessionBtn.click();
      } else {
        const urlMatch = authedPage.url().match(/\/projects\/([^/]+)\/workspace/);
        const currentProjectId = urlMatch ? urlMatch[1] : '';
        if (currentProjectId) {
          await authedPage.evaluate(async ({ pId, phase }) => {
            const token = localStorage.getItem('auth_token');
            let orgId = '';
            try {
              const meRes = await fetch('/api/v1/auth/me', {
                headers: { ...(token ? { Authorization: `Bearer ${token}` } : {}) },
              });
              if (meRes.ok) {
                const me = await meRes.json();
                orgId = me?.organization_id || '';
              }
            } catch (_) {}
            await fetch(`/api/v1/projects/${pId}/sessions?createNew=true`, {
              method: 'POST',
              headers: {
                'Content-Type': 'application/json',
                ...(token ? { Authorization: `Bearer ${token}` } : {}),
                ...(orgId ? { 'X-Organization-Id': orgId } : {}),
              },
              body: JSON.stringify({ phase }),
            });
          }, { pId: currentProjectId, phase: config.tabName.toUpperCase() }).catch(() => {});
        }
      }
    }

    const retryBtn = authedPage.locator('button:has-text("Retry Connection")');
    if (await retryBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
      await retryBtn.click().catch(() => {});
    }

    await expect(textarea).toBeVisible({ timeout: 15000 });
    await expect(textarea).toBeEnabled({ timeout: 20000 });

    // 6. Locate Quick Action Button under #tour-workspace-quick-actions
    const quickActionsContainer = chatPane.locator('#tour-workspace-quick-actions');
    let actionBtn = quickActionsContainer
      .locator('button')
      .filter({ hasText: new RegExp(`^${config.quickActionName}$`, 'i') })
      .first();

    if (!(await actionBtn.isVisible({ timeout: 3000 }).catch(() => false))) {
      actionBtn = quickActionsContainer
        .locator('button')
        .filter({ hasText: new RegExp(config.quickActionName, 'i') })
        .first();
    }

    if (!(await actionBtn.isVisible({ timeout: 3000 }).catch(() => false))) {
      actionBtn = chatPane.locator(`button:has-text("${config.quickActionName}")`).first();
    }

    await expect(actionBtn, `Quick action button for "${config.quickActionName}" should be visible`).toBeVisible({ timeout: 15000 });

    // 7. Click Quick Action Button to select action
    failureMonitor.recordAndCheck('user', `[Trigger Quick Action] Tab: ${config.tabName}, Action: ${config.quickActionName}`);
    await actionBtn.click();
    await authedPage.waitForTimeout(500);

    const pill = chatPane.locator('span', { hasText: new RegExp(config.quickActionName, 'i') }).first();
    const isPillVisible = await pill.isVisible({ timeout: 2000 }).catch(() => false);
    const injectedText = await textarea.inputValue().catch(() => '');

    if (isPillVisible) {
      failureMonitor.recordAndCheck('user', `[Selected Quick Action Pill] ${config.quickActionName}`);
    } else if (injectedText.trim()) {
      failureMonitor.recordAndCheck('user', `[Injected Quick Action Prompt] ${injectedText.trim()}`);
    }

    // 8. Submit Message
    const sendButton = chatPane.locator('button[type="submit"]').first();
    await expect(sendButton).toBeEnabled({ timeout: 10000 });
    await sendButton.click();

    // 9. Wait for initial Agent Turn to complete
    const lastAgentOutput = await waitForAgentTurnToComplete(
      authedPage,
      chatPane,
      failureMonitor,
      config.turnTimeoutMs || 45000,
      config.stableWindowMs || 5000
    );

    // 10. Verification Assertions:
    // Assertion 1: Skill got invoked and delivered a non-empty response
    expect(
      lastAgentOutput.trim().length,
      `Skill "${config.quickActionName}" in phase [${config.tabName}] should be invoked and produce a non-empty agent response.`
    ).toBeGreaterThan(0);

    // Assertion 2: No phase switch redirection suggested
    const phaseSwitchCard = chatPane.locator('text=/Recommended Phase Switch/i');
    const isPhaseSwitchVisible = await phaseSwitchCard.isVisible().catch(() => false);
    expect(
      isPhaseSwitchVisible,
      `Skill "${config.quickActionName}" in phase [${config.tabName}] should NOT suggest a phase redirect.`
    ).toBe(false);

    // Verify recorded messages contain no suggest_phase_switch redirection
    const recorded = failureMonitor.getRecordedMessages();
    const redirectionMessage = recorded.find(
      (m) =>
        m.text.includes('suggest_phase_switch') ||
        m.text.includes('SWITCH_PHASE:') ||
        /Recommended Phase Switch/i.test(m.text)
    );
    expect(
      redirectionMessage,
      `No phase switch redirection should be emitted for skill "${config.quickActionName}" in phase [${config.tabName}].`
    ).toBeUndefined();

    // 11. Enrich test report with diagnostics
    failureMonitor.enrichTestReport(testInfo, config.quickActionName, config.tabName);

    const detectedFailures = failureMonitor.getDetectedFailures();
    if (config.failOnDetectedError !== false && detectedFailures.length > 0) {
      const firstFail = detectedFailures[0];
      expect.soft(
        detectedFailures.length,
        `Detected ${detectedFailures.length} failure indication(s) during skill invocation. First match at Step ${firstFail.index}: "${firstFail.text}"`
      ).toBe(0);
    }
  });
}

// Canonical list of all PhaseActions defined in workflows.go
export const ALL_PHASE_ACTIONS = [
  // ONBOARDING
  { tabName: 'ONBOARDING', quickActionName: 'Project Context' },
  { tabName: 'ONBOARDING', quickActionName: 'Help' },

  // ANALYSIS
  { tabName: 'ANALYSIS', quickActionName: 'Talk to Business Analyst (Mary)' },
  { tabName: 'ANALYSIS', quickActionName: 'Brainstorming' },
  { tabName: 'ANALYSIS', quickActionName: 'Deep Recon' },
  { tabName: 'ANALYSIS', quickActionName: 'Forge Idea' },
  { tabName: 'ANALYSIS', quickActionName: 'PRFAQ Challenge' },
  { tabName: 'ANALYSIS', quickActionName: 'Product Brief' },
  { tabName: 'ANALYSIS', quickActionName: 'Help' },

  // PLANNING
  { tabName: 'PLANNING', quickActionName: 'Talk to Product Manager (John)' },
  { tabName: 'PLANNING', quickActionName: 'Talk to UX Designer (Sally)' },
  { tabName: 'PLANNING', quickActionName: 'Create PRD' },
  { tabName: 'PLANNING', quickActionName: 'Spec Distillation' },
  { tabName: 'PLANNING', quickActionName: 'UX Design' },
  { tabName: 'PLANNING', quickActionName: 'Test Design' },
  { tabName: 'PLANNING', quickActionName: 'Review' },
  { tabName: 'PLANNING', quickActionName: 'Help' },

  // SOLUTIONING
  { tabName: 'SOLUTIONING', quickActionName: 'Talk to System Architect (Winston)' },
  { tabName: 'SOLUTIONING', quickActionName: 'Create Epics and Stories' },
  { tabName: 'SOLUTIONING', quickActionName: 'Architecture' },
  { tabName: 'SOLUTIONING', quickActionName: 'Review' },
  { tabName: 'SOLUTIONING', quickActionName: 'Help' },

  // IMPLEMENTATION
  { tabName: 'IMPLEMENTATION', quickActionName: 'Sprint Planning' },
  { tabName: 'IMPLEMENTATION', quickActionName: 'Correct Course' },
  { tabName: 'IMPLEMENTATION', quickActionName: 'Talk to Test Architect (Murat)' },
  { tabName: 'IMPLEMENTATION', quickActionName: 'Retrospective' },
  { tabName: 'IMPLEMENTATION', quickActionName: 'Review' },
  { tabName: 'IMPLEMENTATION', quickActionName: 'Help' },
];

export const runSkillInvokerE2ETest = runSkillInvokeE2ETest;

test.describe('Skill Invoke E2E Suite', () => {
  for (const action of ALL_PHASE_ACTIONS) {
    runSkillInvokeE2ETest({
      tabName: action.tabName,
      quickActionName: action.quickActionName,
    });
  }
});
