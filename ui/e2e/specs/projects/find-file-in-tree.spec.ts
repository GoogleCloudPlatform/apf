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

import { test, expect } from '../../fixtures/tree-fixture';
import { WorkspacePage } from '../../page-objects/workspace-page';

test.describe('find_file_in_tree Fixture', () => {

  test.beforeEach(async ({ authedPage }) => {
    // Intercept project file requests with a deterministic mock file tree structure
    await authedPage.route('**/api/v1/projects/*/files*', async (route) => {
      const url = new URL(route.request().url());
      const subpath = url.searchParams.get('path') || '';

      if (!subpath) {
        // Root directory
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            name: 'mock-project',
            path: '',
            type: 'folder',
            hasChildren: true,
            children: [
              { name: 'src', path: 'src', type: 'folder', hasChildren: true },
              { name: 'docs', path: 'docs', type: 'folder', hasChildren: true },
              { name: '.github', path: '.github', type: 'folder', hasChildren: true },
              { name: '.gitignore', path: '.gitignore', type: 'file' },
              { name: 'package.json', path: 'package.json', type: 'file' },
              { name: 'README.md', path: 'README.md', type: 'file' },
            ],
          }),
        });
      }

      if (subpath === '.github') {
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            name: '.github',
            path: '.github',
            type: 'folder',
            hasChildren: true,
            children: [
              { name: 'workflows', path: '.github/workflows', type: 'folder', hasChildren: true },
            ],
          }),
        });
      }

      if (subpath === '.github/workflows' || subpath === '.github%2Fworkflows') {
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            name: 'workflows',
            path: '.github/workflows',
            type: 'folder',
            hasChildren: true,
            children: [
              { name: 'ci.yml', path: '.github/workflows/ci.yml', type: 'file' },
            ],
          }),
        });
      }

      if (subpath === 'src') {
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            name: 'src',
            path: 'src',
            type: 'folder',
            hasChildren: true,
            children: [
              { name: 'components', path: 'src/components', type: 'folder', hasChildren: true },
              { name: 'index.ts', path: 'src/index.ts', type: 'file' },
            ],
          }),
        });
      }

      if (subpath === 'src/components' || subpath === 'src%2Fcomponents') {
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            name: 'components',
            path: 'src/components',
            type: 'folder',
            hasChildren: true,
            children: [
              { name: 'workspace', path: 'src/components/workspace', type: 'folder', hasChildren: true },
              { name: 'Button.tsx', path: 'src/components/Button.tsx', type: 'file' },
            ],
          }),
        });
      }

      if (subpath === 'src/components/workspace' || subpath === 'src%2Fcomponents%2Fworkspace') {
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            name: 'workspace',
            path: 'src/components/workspace',
            type: 'folder',
            hasChildren: true,
            children: [
              { name: 'FileExplorer.tsx', path: 'src/components/workspace/FileExplorer.tsx', type: 'file' },
              { name: 'FileIcon.tsx', path: 'src/components/workspace/FileIcon.tsx', type: 'file' },
            ],
          }),
        });
      }

      if (subpath === 'docs') {
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            name: 'docs',
            path: 'docs',
            type: 'folder',
            hasChildren: true,
            children: [
              { name: 'architecture.md', path: 'docs/architecture.md', type: 'file' },
            ],
          }),
        });
      }

      return route.fulfill({
        status: 404,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Not found' }),
      });
    });
  });

  test('should find a root-level file in the workspace tree', async ({
    authedPage,
    sharedProject,
    find_file_in_tree,
  }) => {
    const workspacePage = new WorkspacePage(authedPage);
    await workspacePage.goto(sharedProject.id);

    // Locate root-level README.md
    const fileLocator = await find_file_in_tree('README.md');
    await expect(fileLocator).toBeVisible();
    await expect(fileLocator).toContainText('README.md');
  });

  test('should recursively expand folders to find deeply nested files', async ({
    authedPage,
    sharedProject,
    find_file_in_tree,
  }) => {
    const workspacePage = new WorkspacePage(authedPage);
    await workspacePage.goto(sharedProject.id);

    // Locate deeply nested FileExplorer.tsx
    const fileLocator = await find_file_in_tree('src/components/workspace/FileExplorer.tsx');
    await expect(fileLocator).toBeVisible();
    await expect(fileLocator).toContainText('FileExplorer.tsx');
  });

  test('should normalize path formats with leading slashes and relative dots', async ({
    authedPage,
    sharedProject,
    find_file_in_tree,
  }) => {
    const workspacePage = new WorkspacePage(authedPage);
    await workspacePage.goto(sharedProject.id);

    // Leading slash
    const file1 = await find_file_in_tree('/docs/architecture.md');
    await expect(file1).toBeVisible();

    // Leading ./
    const file2 = await find_file_in_tree('./src/components/Button.tsx');
    await expect(file2).toBeVisible();
  });

  test('should automatically expand collapsed FILE EXPLORER section', async ({
    authedPage,
    sharedProject,
    find_file_in_tree,
  }) => {
    const workspacePage = new WorkspacePage(authedPage);
    await workspacePage.goto(sharedProject.id);

    // Collapse the File Explorer section first
    const fileExplorerHeader = authedPage
      .locator('#tour-workspace-file-explorer')
      .locator('div')
      .first();
    await fileExplorerHeader.click();

    // Verify tree is collapsed/hidden
    await expect(workspacePage.fileTree).not.toBeVisible();

    // find_file_in_tree should auto-expand the section and find the file
    const fileLocator = await find_file_in_tree('package.json');
    await expect(fileLocator).toBeVisible();
  });

  test('should support click option to select the file upon finding it', async ({
    authedPage,
    sharedProject,
    find_file_in_tree,
  }) => {
    const workspacePage = new WorkspacePage(authedPage);
    await workspacePage.goto(sharedProject.id);

    const fileLocator = await find_file_in_tree('src/components/workspace/FileIcon.tsx', {
      click: true,
    });

    await expect(fileLocator).toBeVisible();
    await expect(fileLocator).toHaveAttribute('aria-selected', 'true');
  });

  test('should work via WorkspacePage page object method', async ({
    authedPage,
    sharedProject,
  }) => {
    const workspacePage = new WorkspacePage(authedPage);
    await workspacePage.goto(sharedProject.id);

    const fileLocator = await workspacePage.findFileInTree('src/index.ts');
    await expect(fileLocator).toBeVisible();
  });

  test('clicking on a file in the file explorer should not collapse the folder tree', async ({
    authedPage,
    sharedProject,
    find_file_in_tree,
  }) => {
    const workspacePage = new WorkspacePage(authedPage);
    await workspacePage.goto(sharedProject.id);

    // Expand deeply nested folder and click FileExplorer.tsx
    const fileLocator = await find_file_in_tree('src/components/workspace/FileExplorer.tsx', {
      click: true,
    });

    // Verify file is selected and visible
    await expect(fileLocator).toBeVisible();
    await expect(fileLocator).toHaveAttribute('aria-selected', 'true');

    // Verify parent folders and sibling files remain expanded and visible in the tree
    const siblingFile = authedPage.locator('[role="treeitem"]').filter({
      has: authedPage.locator('span', { hasText: /^FileIcon\.tsx$/ }),
    });
    await expect(siblingFile).toBeVisible();

    const parentFolder = authedPage.locator('[role="treeitem"][aria-expanded="true"]').filter({
      has: authedPage.locator('span', { hasText: /^workspace$/ }),
    });
    await expect(parentFolder).toBeVisible();
    await expect(parentFolder).toHaveAttribute('aria-expanded', 'true');

    const grandParentFolder = authedPage.locator('[role="treeitem"][aria-expanded="true"]').filter({
      has: authedPage.locator('span', { hasText: /^components$/ }),
    });
    await expect(grandParentFolder).toBeVisible();
    await expect(grandParentFolder).toHaveAttribute('aria-expanded', 'true');

    const rootFolder = authedPage.locator('[role="treeitem"][aria-expanded="true"]').filter({
      has: authedPage.locator('span', { hasText: /^src$/ }),
    });
    await expect(rootFolder).toBeVisible();
    await expect(rootFolder).toHaveAttribute('aria-expanded', 'true');
  });

  test('should display FilteredFileListing by default and switch to Tree View when Show All is toggled', async ({
    authedPage,
    sharedProject,
  }) => {
    const workspacePage = new WorkspacePage(authedPage);
    await workspacePage.goto(sharedProject.id);

    // Filtered mode header is rendered by default ("Onboarding Files")
    await expect(authedPage.locator('#tour-workspace-file-explorer')).toBeVisible();

    // Tree view should not be visible initially
    const treeView = authedPage.locator('[role="tree"][aria-label="Project Files"]');
    await expect(treeView).not.toBeVisible();

    // Toggle Show All ON
    const toggleLabel = authedPage.locator('label:has-text("Show All")').first();
    await toggleLabel.click();

    // Standard tree view should now be visible (role="tree")
    await expect(treeView).toBeVisible();
    await expect(treeView.locator('[role="treeitem"]').filter({ hasText: /^src$/ })).toBeVisible();
    await expect(treeView.locator('text=package.json')).toBeVisible();

    // Toggle Show All OFF
    await toggleLabel.click();
    await expect(treeView).not.toBeVisible();
  });

  test('should throw a descriptive error when target file does not exist', async ({
    authedPage,
    sharedProject,
    find_file_in_tree,
  }) => {
    const workspacePage = new WorkspacePage(authedPage);
    await workspacePage.goto(sharedProject.id);

    await expect(
      find_file_in_tree('src/components/non-existent-file.ts', { timeout: 3000 })
    ).rejects.toThrowError(/find_file_in_tree: Could not find file "non-existent-file.ts"/);
  });

  test('should find and expand a folder using find_folder_in_tree', async ({
    authedPage,
    sharedProject,
    find_folder_in_tree,
  }) => {
    const workspacePage = new WorkspacePage(authedPage);
    await workspacePage.goto(sharedProject.id);

    const folderLocator = await find_folder_in_tree('src/components/workspace');
    await expect(folderLocator).toBeVisible();
    await expect(folderLocator).toContainText('workspace');
    await expect(folderLocator).toHaveAttribute('aria-expanded', 'true');
  });

  test('should collapse an expanded folder using collapse_folder_in_tree', async ({
    authedPage,
    sharedProject,
    find_folder_in_tree,
    collapse_folder_in_tree,
  }) => {
    const workspacePage = new WorkspacePage(authedPage);
    await workspacePage.goto(sharedProject.id);

    // Expand folder first
    const folderLocator = await find_folder_in_tree('src/components');
    await expect(folderLocator).toHaveAttribute('aria-expanded', 'true');

    // Collapse folder
    await collapse_folder_in_tree('src/components');
    await expect(folderLocator).toHaveAttribute('aria-expanded', 'false');
  });

  test('should find and select a file in the filtered list using find_file_in_list', async ({
    authedPage,
    sharedProject,
    find_file_in_list,
  }) => {
    await authedPage.route('**/api/v1/projects/*/artifacts', async (route) => {
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          {
            id: 'art-readme',
            projectId: sharedProject.id,
            title: 'README.md',
            path: 'README.md',
            phase: 'ONBOARDING',
            status: 'Complete',
            artifactType: 'DOC',
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
          },
        ]),
      });
    });

    const workspacePage = new WorkspacePage(authedPage);
    await workspacePage.goto(sharedProject.id);

    const listItem = await find_file_in_list('README.md', { click: true });
    await expect(listItem).toBeVisible();
    await expect(listItem).toHaveAttribute('aria-selected', 'true');
  });

  test('should refresh a subfolder in the tree view using refresh_folder_in_tree', async ({
    authedPage,
    sharedProject,
    refresh_folder_in_tree,
  }) => {
    const workspacePage = new WorkspacePage(authedPage);
    await workspacePage.goto(sharedProject.id);

    let refreshCalled = false;
    await authedPage.route('**/api/v1/projects/*/files?path=src%2Fcomponents', async (route) => {
      refreshCalled = true;
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          name: 'components',
          path: 'src/components',
          type: 'folder',
          hasChildren: true,
          children: [
            { name: 'workspace', path: 'src/components/workspace', type: 'folder', hasChildren: true },
            { name: 'Button.tsx', path: 'src/components/Button.tsx', type: 'file' },
          ],
        }),
      });
    });

    await refresh_folder_in_tree('src/components');
    expect(refreshCalled).toBe(true);
  });

  test('should throw descriptive error when target folder does not exist', async ({
    authedPage,
    sharedProject,
    find_folder_in_tree,
  }) => {
    const workspacePage = new WorkspacePage(authedPage);
    await workspacePage.goto(sharedProject.id);

    await expect(
      find_folder_in_tree('src/non-existent-folder', { timeout: 3000 })
    ).rejects.toThrowError(/find_folder_in_tree: Could not find folder "non-existent-folder"/);
  });

  test('should propagate backend API errors when subfolder fails to load', async ({
    authedPage,
    sharedProject,
    find_file_in_tree,
  }) => {
    const workspacePage = new WorkspacePage(authedPage);
    await workspacePage.goto(sharedProject.id);

    // Override the route for src/components to return a 500 error
    await authedPage.route('**/api/v1/projects/*/files?path=src%2Fcomponents', async (route) => {
      return route.fulfill({
        status: 500,
        contentType: 'text/plain',
        body: 'Internal Server Error: disk unreadable',
      });
    });

    await expect(
      find_file_in_tree('src/components/Button.tsx', { timeout: 4000 })
    ).rejects.toThrowError(/Failed to (load|expand)/);
  });

  test('should find hidden files and hidden folders in the tree view when showing all files', async ({
    authedPage,
    sharedProject,
    find_file_in_tree,
    find_folder_in_tree,
  }) => {
    const workspacePage = new WorkspacePage(authedPage);
    await workspacePage.goto(sharedProject.id);

    // Locate root-level hidden file .gitignore
    const gitignoreLocator = await find_file_in_tree('.gitignore');
    await expect(gitignoreLocator).toBeVisible();
    await expect(gitignoreLocator).toContainText('.gitignore');

    // Locate nested file within hidden folder .github/workflows/ci.yml
    const ciLocator = await find_file_in_tree('.github/workflows/ci.yml');
    await expect(ciLocator).toBeVisible();
    await expect(ciLocator).toContainText('ci.yml');

    // Verify .github folder is expanded
    const githubFolder = await find_folder_in_tree('.github');
    await expect(githubFolder).toBeVisible();
    await expect(githubFolder).toHaveAttribute('aria-expanded', 'true');
  });

});
