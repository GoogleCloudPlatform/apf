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
import { WorkspacePage } from '../../page-objects/workspace-page';

test.describe('Workspace Detail Page', () => {

  test('should load the workspace page with essential components', async ({ authedPage, sharedProject }) => {
    const workspacePage = new WorkspacePage(authedPage);
    
    // Navigate to the real project workspace
    await workspacePage.goto(sharedProject.id);

    // Verify header is visible
    await expect(workspacePage.header).toBeVisible();

    // Verify Chat Pane is visible (it is the central part of the workspace)
    await expect(workspacePage.chatPane).toBeVisible();

    // Verify Sidebar is visible by default
    await expect(workspacePage.sidebar).toBeVisible();
  });

  test('should allow typing in the chat input', async ({ authedPage, sharedProject }) => {
    const workspacePage = new WorkspacePage(authedPage);
    await workspacePage.goto(sharedProject.id);

    const testMessage = 'Hello, Agent! Help me with my task.';
    
    // Verify input is visible and enable typing
    await expect(workspacePage.chatInput).toBeVisible();
    await workspacePage.chatInput.fill(testMessage);
    
    // Assert the input contains the typed text
    await expect(workspacePage.chatInput).toHaveValue(testMessage);
  });

  // Example of a skipped test that requires backend integration or mocking
  test.skip('should show canvas when an artifact is selected', async ({ authedPage, sharedProject }) => {
    const workspacePage = new WorkspacePage(authedPage);
    await workspacePage.goto(sharedProject.id);

    // Assuming there is an artifact named "README.md" in the list
    await workspacePage.selectArtifact('README.md');

    // Canvas should become visible
    await expect(workspacePage.canvasPane).toBeVisible();

    // Close the canvas
    await workspacePage.canvasCloseButton.click();
    await expect(workspacePage.canvasPane).not.toBeVisible();
  });
});
