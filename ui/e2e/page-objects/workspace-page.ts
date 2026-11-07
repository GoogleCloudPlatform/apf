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
import {
  findFileInTree,
  findFolderInTree,
  findFileInList,
  refreshFolderInTree,
  collapseFolderInTree,
  FindFileInTreeOptions,
  FindFolderInTreeOptions,
  FindFileInListOptions,
  RefreshFolderInTreeOptions,
  CollapseFolderInTreeOptions,
} from '../fixtures/tree-fixture';

export class WorkspacePage {
  readonly page: Page;
  
  // Header Elements
  readonly header: Locator;
  readonly backButton: Locator;
  readonly projectTitle: Locator;
  
  // Sidebar (Explorer) Elements
  readonly sidebar: Locator;
  readonly sidebarToggle: Locator;
  readonly newSessionButton: Locator;
  readonly sessionList: Locator;
  readonly artifactList: Locator;
  readonly fileExplorerSection: Locator;
  readonly fileTree: Locator;
  
  // Chat Elements
  readonly chatPane: Locator;
  readonly chatInput: Locator;
  readonly sendButton: Locator;
  readonly messages: Locator;
  
  // Canvas Elements
  readonly canvasPane: Locator;
  readonly canvasCloseButton: Locator;

  // Onboarding & Phase Navigation Elements
  readonly onboardingPhaseTab: Locator;
  readonly analysisPhaseTab: Locator;
  readonly planningPhaseTab: Locator;
  readonly healthCheckButton: Locator;
  readonly quickActionsSection: Locator;
  readonly onboardingScanButton: Locator;
  readonly artifactsSection: Locator;

  constructor(page: Page) {
    this.page = page;
    
    // Header
    this.header = page.locator('header').filter({ has: page.locator('#tour-workspace-phases') });
    this.backButton = this.header.locator('button[title="Back to Projects list"]');
    this.projectTitle = this.header.locator('h1, span.font-bold');
    
    // Phase Tabs & Health Check
    this.onboardingPhaseTab = page.locator('#tour-workspace-phase-onboarding');
    this.analysisPhaseTab = page.locator('#tour-workspace-phase-analysis');
    this.planningPhaseTab = page.locator('#tour-workspace-phase-planning');
    this.healthCheckButton = page.locator('#tour-workspace-health');
    
    // Sidebar
    this.sidebar = page.locator('.border-r.bg-card').filter({ hasText: 'WORKSPACE EXPLORER' }); 
    this.sidebarToggle = page.locator('button[title="Collapse Sidebar"], button[title="Expand Sidebar"]');
    this.newSessionButton = page.locator('button:has-text("New Session"), button:has-text("Create")');
    this.sessionList = page.locator('text=Sessions'); // Adjust based on ExplorerPanel headers
    this.quickActionsSection = page.locator('#tour-workspace-quick-actions');
    this.onboardingScanButton = this.quickActionsSection.locator('button');
    this.artifactsSection = page.locator('#tour-workspace-file-explorer');
    this.artifactList = page.locator('#tour-workspace-file-explorer [role="list"]');
    this.fileExplorerSection = page.locator('#tour-workspace-file-explorer');
    this.fileTree = page.locator('[role="tree"][aria-label="Project Files"]');
    
    // Chat
    this.chatPane = page.locator('#tour-workspace-chat');
    this.chatInput = this.chatPane.locator('textarea, input[type="text"]');
    this.sendButton = this.chatPane.locator('button[type="submit"]');
    this.messages = this.chatPane.locator('[data-testid="chat-message"], .message'); // Adjust based on ChatPane implementation
    
    // Canvas
    this.canvasPane = page.locator('#tour-workspace-canvas');
    this.canvasCloseButton = this.canvasPane.locator('button:has(svg)'); // Typically an 'X' close button
  }

  async goto(projectId: string, phase?: string) {
    const query = phase ? `?phase=${phase}` : '';
    await this.page.goto(`/projects/${projectId}/workspace${query}`);
  }

  async selectPhase(phase: 'ONBOARDING' | 'ANALYSIS' | 'PLANNING' | 'SOLUTIONING' | 'IMPLEMENTATION') {
    await this.page.locator(`#tour-workspace-phase-${phase.toLowerCase()}`).click();
  }

  async runHealthCheck() {
    await this.healthCheckButton.click();
  }

  async clickQuickAction(name: string) {
    await this.page.locator(`button:has-text("${name}")`).click();
  }

  async sendMessage(message: string) {
    await this.chatInput.waitFor({ state: 'visible' });
    await this.chatInput.fill(message);
    await this.sendButton.click();
  }

  async toggleSidebar() {
    const collapseBtn = this.page.locator('button[title="Collapse Sidebar"]');
    const expandBtn = this.page.locator('button[title="Expand Sidebar"]');
    if (await collapseBtn.isVisible()) {
      await collapseBtn.click();
    } else {
      await expandBtn.click();
    }
  }

  async createNewSession() {
    await this.newSessionButton.click();
  }

  async selectArtifact(name: string) {
    await this.page.locator(`text=${name}`).click();
  }

  /**
   * Recursively expands folders in the Workspace Explorer file tree until the target file is found.
   */
  async findFileInTree(filePath: string, options?: FindFileInTreeOptions) {
    return await findFileInTree(this.page, filePath, options);
  }

  /**
   * Traverses and locates a folder in the file tree.
   */
  async findFolderInTree(folderPath: string, options?: FindFolderInTreeOptions) {
    return await findFolderInTree(this.page, folderPath, options);
  }

  /**
   * Locates a file in the flat Filtered Files list.
   */
  async findFileInList(filePathOrName: string, options?: FindFileInListOptions) {
    return await findFileInList(this.page, filePathOrName, options);
  }

  /**
   * Refreshes a subfolder in the tree view.
   */
  async refreshFolderInTree(folderPath: string, options?: RefreshFolderInTreeOptions) {
    return await refreshFolderInTree(this.page, folderPath, options);
  }

  /**
   * Collapses an expanded folder in the tree view.
   */
  async collapseFolderInTree(folderPath: string, options?: CollapseFolderInTreeOptions) {
    return await collapseFolderInTree(this.page, folderPath, options);
  }
}


