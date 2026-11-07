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
import { test, expect } from '../../fixtures/tree-fixture';
import type { Page, Locator } from '@playwright/test';
import { generateLLMResponse } from '../../helpers/llm-responder';
import { FailureMonitor } from '../../helpers/failure-monitor';

export interface ConversationTurn {
  turnIndex: number;
  agentKeyword?: string;
  userReply: string;
}

export interface ConversationFixture {
  tabName: string;
  quickActionName: string;
  description?: string;
  fileNamePattern?: string;
  expectedFilePattern?: string;
  turns: ConversationTurn[];
}

export interface SkillTestConfig {
  tabName: string;          // e.g. 'ONBOARDING', 'ANALYSIS', 'PLANNING', 'SOLUTIONING', 'IMPLEMENTATION'
  quickActionName: string;  // e.g. 'Help', 'BMad Help', 'Architecture', 'Create PRD'
  projectName?: string;
  maxInteractions?: number;
  fileNamePattern?: string | RegExp;     // File name pattern to verify in generated artifacts (input parameter)
  expectedFilePattern?: string | RegExp; // Synonym / alias for flexibility
  failOnDetectedError?: boolean;
  testTimeoutMs?: number;         // Overall Playwright test timeout (default: 240,000ms / 4 mins)
  interactionTimeoutMs?: number;  // Interaction loop time limit (default: 240,000ms / 4 mins)
  turnTimeoutMs?: number;         // Single agent turn timeout (default: 45,000ms / 45 secs)
  stableWindowMs?: number;        // Stability wait window (default: 5,000ms / 5 secs)
  fixtureName?: string;           // Optional conversation fixture file name in e2e/fixtures/conversations
}

export type SkillCompleteFlowConfig = SkillTestConfig;

const MESSAGE_SELECTOR = 'div.mr-auto .rounded-2xl .prose, div.mr-auto .prose, div.mr-auto .rounded-2xl, [data-testid="chat-message"], .message';

/**
 * Loads a conversation fixture JSON file if provided
 */
function loadFixture(fixtureName: string): ConversationFixture | null {
  try {
    const filePath = path.join(__dirname, '../../fixtures/conversations', fixtureName);
    if (fs.existsSync(filePath)) {
      return JSON.parse(fs.readFileSync(filePath, 'utf-8'));
    }
  } catch (e) {
    console.warn(`Failed to load fixture ${fixtureName}:`, e);
  }
  return null;
}

/**
 * Helper function to determine if the agent turn is complete based on renderlogic.md:
 * 1. Evaluates display state (no active streaming ThinkingBlock, stop button, or spinner).
 * 2. Waits for chat pane display content to remain unchanged for 5 consecutive seconds.
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
  const pollIntervalMs = 1000; // Poll display every 1 second
  const requiredStableCycles = Math.max(2, Math.ceil(stableWindowMs / pollIntervalMs));

  console.log(`[E2E Flow] Waiting for agent turn completion (Display stability window: ${stableWindowMs}ms)...`);

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
        console.log('[E2E Flow] Interactive option buttons detected in chat pane.');
        return currentDisplayText;
      }

      if (!isThinking && currentDisplayText === lastDisplayText && currentDisplayText.trim().length > 0) {
        stableCount++;
        if (stableCount >= requiredStableCycles) {
          console.log(`[E2E Flow] Turn completion detected! Display content stable for ${stableWindowMs / 1000}s. Count: ${count}`);
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
            console.log(`[E2E Flow] Turn completion detected via full text fallback!`);
            return fullText;
          }
        } else {
          lastDisplayText = fullText;
          stableCount = 0;
        }
      }
    }
  }

  console.log('[E2E Flow] Warning: Turn timeout reached before display stability was confirmed.');
  return lastDisplayText;
}

/**
 * Parameterized E2E test runner function.
 * Provide tabName, quickActionName, and optional fixtureName to execute an interactive E2E skill flow test.
 */
export function runSkillCompleteFlowE2ETest(config: SkillTestConfig) {
  test(`Skill Complete Flow - Tab: [${config.tabName}], Action: [${config.quickActionName}]`, async ({
    authedPage,
    projectCleanup,
    find_file_in_list,
  }, testInfo) => {
    test.setTimeout(config.testTimeoutMs || 240000);

    const fixture = config.fixtureName ? loadFixture(config.fixtureName) : null;
    const failureMonitor = new FailureMonitor();
    const projectName = config.projectName || `Skill Complete Flow ${config.tabName} ${Date.now()}`;
    projectCleanup(projectName);

    // Intercept SSE and API streams in the browser to capture real-time file write event responses
    await authedPage.addInitScript(() => {
      (window as any).__capturedFileWriteEvents = [];
      const originalFetch = window.fetch;
      window.fetch = async function (...args) {
        const response = await originalFetch.apply(this, args);
        const url = String(args[0] || '');
        if (
          url.includes('/messages/stream') ||
          url.includes('/activities/stream') ||
          url.includes('/messages') ||
          url.includes('/artifacts')
        ) {
          try {
            const clone = response.clone();
            const reader = clone.body?.getReader();
            if (reader) {
              const decoder = new TextDecoder();
              (async () => {
                try {
                  while (true) {
                    const { done, value } = await reader.read();
                    if (done) break;
                    const chunk = decoder.decode(value, { stream: true });
                    const notifRegex = /file-write-notification:[^"]*"\s*:\s*"([^"]+)"/g;
                    let m: RegExpExecArray | null;
                    while ((m = notifRegex.exec(chunk)) !== null) {
                      if (m[1]) (window as any).__capturedFileWriteEvents.push(m[1].trim());
                    }
                    const filePathRegex = /"filePath"\s*:\s*"([^"]+)"/g;
                    while ((m = filePathRegex.exec(chunk)) !== null) {
                      if (m[1]) (window as any).__capturedFileWriteEvents.push(m[1].trim());
                    }
                    const textRegex = /File updated:\s*([^\n\r"\\`]+)/g;
                    while ((m = textRegex.exec(chunk)) !== null) {
                      if (m[1]) (window as any).__capturedFileWriteEvents.push(m[1].trim());
                    }
                  }
                } catch (_) {}
              })();
            }
          } catch (_) {}
        }
        return response;
      };
    });

    // 1. Ensure we are on /projects page
    if (!authedPage.url().includes('/projects')) {
      await authedPage.goto('/projects');
    }

    // Dismiss tour if present
    const skipTourButton = authedPage.locator('button:has-text("Skip Tour")');
    if (await skipTourButton.isVisible({ timeout: 2000 }).catch(() => false)) {
      await skipTourButton.click();
    }

    // 2. Create Project (without Git Repo and with GitHub integration unchecked)
    const createButton = authedPage.locator('#tour-catalog-create');
    await expect(createButton).toBeVisible({ timeout: 15000 });
    await createButton.click();

    await authedPage.waitForURL(/\/projects\/new\/details/);

    const nameInput = authedPage.locator('#projectName');
    await expect(nameInput).toBeVisible();
    await nameInput.fill(projectName);

    const descInput = authedPage.locator('#projectDesc');
    if (await descInput.isVisible().catch(() => false)) {
      await descInput.fill('Skill complete flow test project created without git repo.');
    }

    // Uncheck GitHub integration if enabled/checked
    const githubToggle = authedPage.locator('#tour-project-toggle-github, input[type="checkbox"][id*="github"]').first();
    if (await githubToggle.isVisible({ timeout: 5000 }).catch(() => false)) {
      if (await githubToggle.isChecked().catch(() => false)) {
        await githubToggle.uncheck().catch(async () => {
          await githubToggle.click();
        });
      }
    }

    // Submit form without filling GitHub repository
    const saveButton = authedPage.locator('#tour-project-save-button');
    await expect(saveButton).toBeEnabled();
    await saveButton.click();

    // Wait for project catalog or workspace navigation
    await authedPage.waitForURL(/\/projects(\?.*)?$/, { timeout: 15000 });
    const projectHeading = authedPage.getByRole('heading', { name: projectName });
    await expect(projectHeading).toBeVisible({ timeout: 15000 });

    // Open Workspace
    const cardContainer = authedPage
      .locator('div')
      .filter({ has: projectHeading })
      .filter({ has: authedPage.locator('button') })
      .first();

    const openWorkspaceBtn = cardContainer
      .locator('button:has-text("Onboarding"), button:has-text("Open Workspace"), button.tour-project-launch-button')
      .first();

    await expect(openWorkspaceBtn).toBeVisible({ timeout: 15000 });
    await openWorkspaceBtn.click();

    await authedPage.waitForURL(/\/projects\/.*\/workspace/, { timeout: 15000 });

    // 3. Navigate to Tab & Ensure Chat Textbox is Enabled
    const normalizedTab = config.tabName.toLowerCase();
    const phaseTab = authedPage.locator(`#tour-workspace-phase-${normalizedTab}`);

    if (await phaseTab.isVisible({ timeout: 5000 }).catch(() => false)) {
      await phaseTab.click();
    } else {
      const genericTab = authedPage.locator(`[role="tab"]:has-text("${config.tabName}"), button:has-text("${config.tabName}")`).first();
      if (await genericTab.isVisible({ timeout: 3000 }).catch(() => false)) {
        await genericTab.click();
      }
    }

    const chatPane = authedPage.locator('#tour-workspace-chat');
    const textarea = chatPane.locator('textarea').first();
    const sendButton = chatPane.locator('button[type="submit"]').first();

    await expect(chatPane).toBeVisible({ timeout: 15000 });
    await expect(textarea).toBeVisible({ timeout: 15000 });
    await expect(textarea).toBeEnabled({ timeout: 20000 });

    // Locate quick action button under Chat prompt text box (#tour-workspace-quick-actions in chatPane)
    const quickActionsContainer = chatPane.locator('#tour-workspace-quick-actions');
    const actionAliases: Record<string, string[]> = {
      scan: ['Project Context', 'Scan'],
    };
    const targetNames = actionAliases[config.quickActionName.toLowerCase()] || [config.quickActionName];

    let actionBtn = quickActionsContainer
      .getByRole('button', { name: new RegExp(`^(${targetNames.join('|')})$`, 'i') })
      .first();

    if (!(await actionBtn.isVisible({ timeout: 3000 }).catch(() => false))) {
      actionBtn = quickActionsContainer
        .getByRole('button', { name: new RegExp(targetNames.join('|'), 'i') })
        .first();
    }

    if (!(await actionBtn.isVisible({ timeout: 3000 }).catch(() => false))) {
      actionBtn = chatPane
        .getByRole('button', { name: new RegExp(targetNames.join('|'), 'i') })
        .first();
    }

    await expect(actionBtn).toBeVisible({ timeout: 15000 });

    // Click quick action button to select quick action
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

    // Click Send Message button after quick action click
    await expect(sendButton).toBeEnabled({ timeout: 10000 });
    await sendButton.click();

    // Wait until final response is received for the sent quick action message
    let lastAgentOutput = await waitForAgentTurnToComplete(
      authedPage,
      chatPane,
      failureMonitor,
      config.turnTimeoutMs || 45000,
      config.stableWindowMs || 5000
    );

    // 4. Continued Dynamic / Fixture-Driven Response Loop (concludes after number of turns are done)
    let interactions = 0;
    const maxInteractions = config.maxInteractions || (fixture ? fixture.turns.length : 6);
    const interactionStartTime = Date.now();
    const maxInteractionDurationMs = config.interactionTimeoutMs || 240000;

    while (interactions < maxInteractions) {
      const elapsedMs = Date.now() - interactionStartTime;
      if (elapsedMs > maxInteractionDurationMs) {
        console.log(`[E2E Flow] Interaction time limit reached (${Math.round(elapsedMs / 1000)}s). Concluding test loop.`);
        break;
      }

      // Check for interactive option buttons in chat
      const optionButtons = chatPane.locator('button.option-btn, button[data-option], .message button');
      const optionCount = await optionButtons.count();

      if (optionCount > 0) {
        const firstOption = optionButtons.first();
        if (await firstOption.isVisible().catch(() => false)) {
          const optionText = (await firstOption.textContent()) || 'Selected Option';
          failureMonitor.recordAndCheck('user', `[Selected Option] ${optionText.trim()}`);
          await firstOption.click();
          interactions++;

          lastAgentOutput = await waitForAgentTurnToComplete(
            authedPage,
            chatPane,
            failureMonitor,
            config.turnTimeoutMs || 45000,
            config.stableWindowMs || 5000
          );
          continue;
        }
      }

      // Determine response text: use fixture turn userReply if available, or call generateLLMResponse
      let responseText = '';
      if (fixture && fixture.turns[interactions]) {
        responseText = fixture.turns[interactions].userReply;
        console.log(`[E2E Flow] Turn ${interactions + 1}/${maxInteractions}: Using sample conversation fixture reply: "${responseText}"`);
      } else {
        console.log(`[E2E Flow] Turn ${interactions + 1}/${maxInteractions}: Invoking generateLLMResponse...`);
        responseText = await generateLLMResponse({
          skillName: config.quickActionName,
          tabName: config.tabName,
          lastAgentMessage: lastAgentOutput,
          chatHistory: [lastAgentOutput],
        });
      }

      if (await textarea.isVisible().catch(() => false)) {
        failureMonitor.recordAndCheck('user', responseText);
        await textarea.fill(responseText);

        await expect(sendButton).toBeEnabled({ timeout: 5000 });
        await sendButton.click();
        interactions++;

        lastAgentOutput = await waitForAgentTurnToComplete(
          authedPage,
          chatPane,
          failureMonitor,
          config.turnTimeoutMs || 45000,
          config.stableWindowMs || 5000
        );
      } else {
        break;
      }
    }

    console.log(`[E2E Flow] Completed all ${interactions} turn(s). Concluding conversational turns.`);

    // 5. Use tree-fixture to navigate, locate the generated file in the tree, click it, and assert content in CanvasPane
    const patternInput =
      config.fileNamePattern ||
      config.expectedFilePattern ||
      fixture?.fileNamePattern ||
      fixture?.expectedFilePattern ||
      (config.tabName?.toUpperCase() === 'ONBOARDING' && /scan/i.test(config.quickActionName)
        ? 'agents.md'
        : undefined);
    const patternRegex = patternInput
      ? (typeof patternInput === 'string' ? new RegExp(patternInput, 'i') : patternInput)
      : null;

    // Collect file write events captured in-page via SSE/fetch streams
    const capturedInPage: string[] = await authedPage.evaluate(() => {
      return (window as any).__capturedFileWriteEvents || [];
    }).catch(() => []);

    const allFileWrites: string[] = [...capturedInPage];

    // Also extract from recorded failureMonitor chat messages
    for (const msg of failureMonitor.getRecordedMessages()) {
      const match = msg.text.match(/File updated:\s*([^\n\r`]+)/i);
      if (match && match[1]) {
        allFileWrites.push(match[1].trim());
      }
      const readMatch = msg.text.match(/Read file:\s*`?([^`\n\r*]+)`?/i);
      if (readMatch && readMatch[1]) {
        allFileWrites.push(readMatch[1].trim());
      }
    }

    // Also query backend artifacts & project files API for registered live artifacts/files
    const workspaceUrl = authedPage.url();
    const projectIdMatch = workspaceUrl.match(/\/projects\/([^/]+)\/workspace/);
    const currentProjectId = projectIdMatch ? projectIdMatch[1] : '';

    if (currentProjectId) {
      const backendArtifacts: any[] = await authedPage.evaluate(async (pId) => {
        try {
          const token = localStorage.getItem('auth_token');
          const res = await fetch(`/api/v1/projects/${pId}/artifacts`, {
            headers: token ? { Authorization: `Bearer ${token}` } : {},
          });
          if (res.ok) {
            return await res.json();
          }
        } catch (_) {}
        return [];
      }, currentProjectId).catch(() => []);

      for (const art of backendArtifacts) {
        if (art.path) allFileWrites.push(art.path);
        else if (art.title) allFileWrites.push(art.title);
      }
    }

    // Determine target file from the last file write event response (matching pattern if provided)
    let matchingWrites = allFileWrites;
    if (patternRegex) {
      const filtered = allFileWrites.filter((p) => {
        const fn = p.split('/').pop() || p;
        return patternRegex.test(fn) || patternRegex.test(p);
      });
      if (filtered.length > 0) {
        matchingWrites = filtered;
      }
    }

    let fileToLocate = matchingWrites.length > 0 ? matchingWrites[matchingWrites.length - 1] : '';
    if (!fileToLocate && patternInput && typeof patternInput === 'string' && !patternInput.includes('*')) {
      fileToLocate = patternInput;
    }

    // If still no path, fallback to querying project files
    if (!fileToLocate && currentProjectId) {
      const backendFiles: any[] = await authedPage.evaluate(async (pId) => {
        try {
          const token = localStorage.getItem('auth_token');
          const res = await fetch(`/api/v1/projects/${pId}/files`, {
            headers: token ? { Authorization: `Bearer ${token}` } : {},
          });
          if (res.ok) {
            return await res.json();
          }
        } catch (_) {}
        return [];
      }, currentProjectId).catch(() => []);

      const flattenFiles = (nodes: any[]): string[] => {
        const result: string[] = [];
        for (const n of nodes) {
          if (n.type === 'file') result.push(n.path || n.name);
          if (n.children) result.push(...flattenFiles(n.children));
        }
        return result;
      };
      const allProjectFiles = flattenFiles(backendFiles);
      if (patternRegex) {
        const matching = allProjectFiles.filter((p) => {
          const fn = p.split('/').pop() || p;
          return patternRegex.test(fn) || patternRegex.test(p);
        });
        if (matching.length > 0) {
          fileToLocate = matching[matching.length - 1];
        }
      }
      if (!fileToLocate && allProjectFiles.length > 0) {
        fileToLocate = allProjectFiles[allProjectFiles.length - 1];
      }
    }

    const targetFileName = fileToLocate ? (fileToLocate.split('/').pop() || fileToLocate) : '';

    console.log(`[E2E Flow] Target file to locate in tree: "${fileToLocate}", target file name: "${targetFileName}"`);

    expect(
      fileToLocate,
      `Expected a target file to be identified for tree navigation. Captured writes: ${JSON.stringify(allFileWrites)}`
    ).toBeTruthy();

    if (patternRegex) {
      expect(targetFileName).toMatch(patternRegex);
    }

    // Locate the generated/changed file in the default File Explorer view (FilteredFileListing) without toggling Show All
    const fileLocator = await find_file_in_list(fileToLocate, { click: true, timeout: 20000 });
    await expect(fileLocator, `File "${fileToLocate}" should be visible in the file explorer view`).toBeVisible({ timeout: 10000 });

    // Verify Canvas / Preview Pane is opened after clicking the file in the file explorer
    const canvasPane = authedPage.locator('#tour-workspace-canvas');
    await expect(canvasPane, 'Canvas preview pane should open after clicking the file in the file explorer').toBeVisible({ timeout: 15000 });

    if (targetFileName) {
      const escapedTarget = targetFileName.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
      const canvasHeader = canvasPane.locator('h2, span, div').filter({ hasText: new RegExp(escapedTarget, 'i') }).first();
      await expect(canvasHeader, `Canvas header should display the file name "${targetFileName}"`).toBeVisible({ timeout: 10000 });
    }

    // Wait for the content viewer in CanvasPane to finish loading
    const viewerContainer = canvasPane.locator(
      '.monaco-editor, .prose, [data-testid="vscode-artifact-viewer"], pre, textarea'
    ).first();
    await expect(viewerContainer, 'Preview pane content viewer should be open and visible').toBeVisible({ timeout: 20000 });

    // Read the content of the file in the canvaspane
    let fileContent = '';
    const contentCandidates = [
      canvasPane.locator('.monaco-editor .view-lines'),
      canvasPane.locator('.prose'),
      canvasPane.locator('pre code'),
      canvasPane.locator('textarea[data-testid="viewer-editor"], textarea'),
      viewerContainer,
      canvasPane,
    ];

    for (const loc of contentCandidates) {
      if (await loc.isVisible({ timeout: 1000 }).catch(() => false)) {
        const text = (await loc.textContent()) || (await loc.inputValue().catch(() => '')) || '';
        if (text.trim() && !text.includes('Downloading file specs content...')) {
          fileContent = text.trim();
          break;
        }
      }
    }

    console.log(`[E2E Flow] Successfully read file content from CanvasPane (${fileContent.length} chars):\n${fileContent.slice(0, 300)}...`);
    expect(fileContent.length, 'File content read from CanvasPane should not be empty').toBeGreaterThan(0);
    await expect(canvasPane.locator('text=Failed to load content')).not.toBeVisible();

    // 6. Enrich Test Report with Chat Transcript and Failure Diagnostics
    failureMonitor.enrichTestReport(testInfo, config.quickActionName, config.tabName);

    const capturedMessages = failureMonitor.getRecordedMessages();
    expect(capturedMessages.length).toBeGreaterThan(0);

    const detectedFailures = failureMonitor.getDetectedFailures();
    if (config.failOnDetectedError !== false && detectedFailures.length > 0) {
      const firstFail = detectedFailures[0];
      expect.soft(
        detectedFailures.length,
        `Detected ${detectedFailures.length} failure indication(s) during skill chat flow. First match at Step ${firstFail.index}: "${firstFail.text}"`
      ).toBe(0);
    }
  });
}

export const runSkillE2ETest = runSkillCompleteFlowE2ETest;

// Automatically create test cases for all JSON conversation fixture files in e2e/fixtures/conversations/
test.describe('Skill Complete Flow E2E Suite', () => {
  const fixturesDir = path.join(__dirname, '../../fixtures/conversations');
  if (fs.existsSync(fixturesDir)) {
    const fixtureFiles = fs.readdirSync(fixturesDir).filter((f) => f.endsWith('.json'));
    for (const file of fixtureFiles) {
      const fix = loadFixture(file);
      if (fix) {
        runSkillCompleteFlowE2ETest({
          tabName: fix.tabName,
          quickActionName: fix.quickActionName,
          fixtureName: file,
          fileNamePattern: fix.fileNamePattern || fix.expectedFilePattern,
        });
      }
    }
  } else {
    // Fallback default test if directory is empty
    runSkillCompleteFlowE2ETest({
      tabName: 'ONBOARDING',
      quickActionName: 'Help',
      fileNamePattern: '.*\\.md',
    });
  }
});

