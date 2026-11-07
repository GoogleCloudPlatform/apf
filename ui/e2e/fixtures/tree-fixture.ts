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

import { test as projectTest, expect } from './project-fixture';
import type { Page, Locator } from '@playwright/test';

export type FindFileInTreeOptions = {
  /**
   * Whether to click the located file item in the tree after finding it.
   * Defaults to false.
   */
  click?: boolean;
  /**
   * Maximum wait timeout in milliseconds for locating or expanding elements.
   * Defaults to 10000ms.
   */
  timeout?: number;
};

export type FindFolderInTreeOptions = {
  /**
   * Whether to expand the located folder.
   * Defaults to true.
   */
  expand?: boolean;
  /**
   * Whether to click the folder item in the tree.
   * Defaults to false (unless expanding is needed).
   */
  click?: boolean;
  /**
   * Maximum wait timeout in milliseconds.
   * Defaults to 10000ms.
   */
  timeout?: number;
};

export type FindFileInListOptions = {
  /**
   * Whether to click the located file item in the filtered list.
   * Defaults to false.
   */
  click?: boolean;
  /**
   * Maximum wait timeout in milliseconds.
   * Defaults to 10000ms.
   */
  timeout?: number;
};

export type RefreshFolderInTreeOptions = {
  /**
   * Maximum wait timeout in milliseconds.
   * Defaults to 10000ms.
   */
  timeout?: number;
};

export type CollapseFolderInTreeOptions = {
  /**
   * Maximum wait timeout in milliseconds.
   * Defaults to 10000ms.
   */
  timeout?: number;
};

export type FindFileInTreeFn = (
  filePath: string,
  options?: FindFileInTreeOptions
) => Promise<Locator>;

export type FindFolderInTreeFn = (
  folderPath: string,
  options?: FindFolderInTreeOptions
) => Promise<Locator>;

export type FindFileInListFn = (
  filePathOrName: string,
  options?: FindFileInListOptions
) => Promise<Locator>;

export type RefreshFolderInTreeFn = (
  folderPath: string,
  options?: RefreshFolderInTreeOptions
) => Promise<void>;

export type CollapseFolderInTreeFn = (
  folderPath: string,
  options?: CollapseFolderInTreeOptions
) => Promise<Locator>;

/**
 * Escapes regex special characters in a string.
 */
function escapeRegex(str: string): string {
  return str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

/**
 * Normalizes a file or folder path into clean segments.
 */
function normalizePath(rawPath: string): {
  normalizedPath: string;
  segments: string[];
  folderSegments: string[];
  targetName: string;
} {
  if (!rawPath || typeof rawPath !== 'string' || !rawPath.trim()) {
    throw new Error(
      `tree-fixture: Invalid path provided: "${rawPath}". Must be a non-empty string.`
    );
  }

  const normalizedPath = rawPath
    .replace(/\\/g, '/')
    .replace(/^\.?\//, '')
    .replace(/\/+$/, '');

  const segments = normalizedPath.split('/').filter(Boolean);
  if (segments.length === 0) {
    throw new Error(
      `tree-fixture: Invalid path provided: "${rawPath}". No valid path segments found.`
    );
  }

  const folderSegments = segments.slice(0, -1);
  const targetName = segments[segments.length - 1];

  return { normalizedPath, segments, folderSegments, targetName };
}

/**
 * Ensures the Workspace Sidebar and File Explorer accordion section are expanded.
 */
async function ensureSidebarAndExplorerOpen(page: Page, timeout: number = 5000): Promise<void> {
  const waitBudget = Math.max(timeout, 8000);

  // 1. Expand sidebar if collapsed
  const expandSidebarBtn = page.locator('button[title="Expand Sidebar"]');
  if (await expandSidebarBtn.isVisible().catch(() => false)) {
    await expandSidebarBtn.click();
    await page
      .locator('button[title="Collapse Sidebar"]')
      .waitFor({ state: 'visible', timeout: 5000 })
      .catch(() => {});
  }

  // 2. Expand File Explorer accordion section if closed
  const fileExplorerSection = page.locator('#tour-workspace-file-explorer');
  await fileExplorerSection.waitFor({ state: 'attached', timeout: waitBudget }).catch(() => {});

  const isSectionVisible = await fileExplorerSection.isVisible().catch(() => false);
  if (isSectionVisible) {
    const isExpanded = (await fileExplorerSection.locator('[aria-expanded="true"]').count().catch(() => 0)) > 0;
    if (!isExpanded) {
      await fileExplorerSection.click().catch(() => {});
    }
  } else {
    const sectionHeader = page
      .locator('div, span, h2')
      .filter({ hasText: /^FILE EXPLORER$/i })
      .first();
    if (await sectionHeader.isVisible().catch(() => false)) {
      await sectionHeader.click().catch(() => {});
    }
  }
}

/**
 * Toggles the "Show All" switch to enable (Tree View) or disable (Filtered List View).
 */
async function ensureShowAllFiles(page: Page, enable: boolean, timeout: number = 5000): Promise<void> {
  const waitBudget = Math.max(timeout, 8000);
  const listContainer = page.locator('[role="list"][aria-label="Filtered Files"]');
  const treeContainer = page.locator('[role="tree"][aria-label="Project Files"]');

  if (enable) {
    if (await treeContainer.isVisible().catch(() => false)) {
      return;
    }
  } else {
    if (await listContainer.isVisible().catch(() => false)) {
      return;
    }
  }

  const toggleLabel = page.locator('label:has-text("Show All")').first();
  const showAllFilesToggle = page.locator(
    'input[type="checkbox"][aria-label="Show All"], [role="switch"][aria-label="Show All"], input[type="checkbox"][aria-label="Show All Files"], [role="switch"][aria-label="Show All Files"]'
  );

  // Wait for the toggle or label to be available in the DOM
  try {
    await toggleLabel.waitFor({ state: 'visible', timeout: waitBudget });
  } catch {
    await showAllFilesToggle.waitFor({ state: 'attached', timeout: waitBudget }).catch(() => {});
  }

  const isChecked = await showAllFilesToggle.isChecked().catch(() => false);
  if (isChecked !== enable) {
    if (await toggleLabel.isVisible().catch(() => false)) {
      await toggleLabel.click();
    } else {
      await showAllFilesToggle.setChecked(enable, { force: true }).catch(() => {});
    }
  }
}

/**
 * Waits for workspace loading/initializing indicators to finish and tree container to be ready.
 */
async function waitForTreeReady(page: Page, timeout: number = 10000): Promise<Locator> {
  const waitBudget = Math.max(timeout, 8000);

  // Wait for loading spinners/indicators to detach first before expecting tree container
  const readingIndicator = page.locator('text="Reading workspace directory..."');
  if (await readingIndicator.isVisible().catch(() => false)) {
    await readingIndicator.waitFor({ state: 'hidden', timeout: waitBudget }).catch(() => {});
  }

  const initializingIndicator = page.locator('text="Initializing workspace repository..."');
  if (await initializingIndicator.isVisible().catch(() => false)) {
    await initializingIndicator.waitFor({ state: 'hidden', timeout: waitBudget }).catch(() => {});
  }

  const treeContainer = page.locator('[role="tree"][aria-label="Project Files"]');
  await treeContainer.waitFor({ state: 'visible', timeout: waitBudget });
  return treeContainer;
}

/**
 * Traverses down a folder path in the hierarchical tree view and expands ancestor directories.
 */
export async function findFolderInTree(
  page: Page,
  folderPath: string,
  options: FindFolderInTreeOptions = {}
): Promise<Locator> {
  const { expand = true, click = false, timeout = 10000 } = options;
  const { segments } = normalizePath(folderPath);

  await ensureSidebarAndExplorerOpen(page, timeout);
  await ensureShowAllFiles(page, true, timeout);
  const treeContainer = await waitForTreeReady(page, timeout);

  let currentScope: Locator = treeContainer;
  let lastFolderItem: Locator = treeContainer;

  for (let i = 0; i < segments.length; i++) {
    const segment = segments[i];
    const currentFolderPath = segments.slice(0, i + 1).join('/');
    const isTargetSegment = i === segments.length - 1;

    // Check if an error occurred in current scope
    const errorNotice = currentScope.locator('.text-destructive');
    if ((await errorNotice.count().catch(() => 0)) > 0 && (await errorNotice.first().isVisible().catch(() => false))) {
      const errMsg = (await errorNotice.first().innerText().catch(() => 'Unknown directory error')).trim();
      throw new Error(
        `find_folder_in_tree: Backend error loading folder "${currentFolderPath}": "${errMsg}"`
      );
    }

    const directFolderItem = currentScope
      .locator('> div[role="none"] > div[role="treeitem"][aria-expanded]')
      .filter({
        has: page.locator('span', { hasText: new RegExp(`^${escapeRegex(segment)}$`, 'i') }),
      })
      .first();

    let folderItem = directFolderItem;
    const isDirectVisible = await directFolderItem.isVisible().catch(() => false);
    if (!isDirectVisible) {
      folderItem = currentScope
        .locator('[role="treeitem"][aria-expanded]')
        .filter({
          has: page.locator('span', { hasText: new RegExp(`^${escapeRegex(segment)}$`, 'i') }),
        })
        .first();
    }

    try {
      await folderItem.waitFor({ state: 'visible', timeout });
    } catch {
      const availableItems = await currentScope
        .locator('[role="treeitem"]')
        .allInnerTexts()
        .catch(() => []);
      throw new Error(
        `find_folder_in_tree: Could not find folder "${segment}" (path so far: "${currentFolderPath}") in folderPath "${folderPath}". Available items: [${availableItems.join(', ')}]`
      );
    }

    lastFolderItem = folderItem;
    const isExpanded = (await folderItem.getAttribute('aria-expanded')) === 'true';

    // Expand folder if it's an intermediate directory or if expand=true is requested
    const shouldExpand = !isTargetSegment || expand;
    if (shouldExpand && !isExpanded) {
      await folderItem.click();
      await expect(folderItem).toHaveAttribute('aria-expanded', 'true', { timeout });
    }

    // Wait for subfolder spinner
    const spinner = folderItem.locator('.animate-spin');
    if ((await spinner.count().catch(() => 0)) > 0) {
      await spinner.waitFor({ state: 'detached', timeout }).catch(() => {});
    }

    // If there are more segments to follow, resolve child group container
    if (!isTargetSegment || expand) {
      const groupContainer = folderItem
        .locator('xpath=following-sibling::div[@role="group"]')
        .first();

      try {
        await groupContainer.waitFor({ state: 'visible', timeout });
      } catch {
        const subError = folderItem.locator('xpath=following-sibling::*').locator('.text-destructive');
        if (await subError.isVisible().catch(() => false)) {
          const errText = await subError.innerText().catch(() => '');
          throw new Error(
            `find_folder_in_tree: Failed to expand folder "${segment}": ${errText}`
          );
        }
        throw new Error(
          `find_folder_in_tree: Folder "${segment}" was expanded, but its child group container did not become visible.`
        );
      }

      const groupError = groupContainer.locator('.text-destructive');
      if (await groupError.isVisible().catch(() => false)) {
        const errText = (await groupError.innerText().catch(() => '')).trim();
        throw new Error(
          `find_folder_in_tree: Failed to load folder "${segment}": ${errText}`
        );
      }

      currentScope = groupContainer;
    }
  }

  if (click) {
    await lastFolderItem.scrollIntoViewIfNeeded().catch(() => {});
    await lastFolderItem.click();
  }

  return lastFolderItem;
}

/**
 * Recursively expands folders in the Workspace Explorer file tree to locate the target file.
 *
 * @param page The Playwright Page object
 * @param filePath The relative or absolute file path (e.g., "src/components/workspace/FileExplorer.tsx" or "README.md")
 * @param options Additional options such as click and timeout
 * @returns The Playwright Locator for the target file in the tree
 */
export async function findFileInTree(
  page: Page,
  filePath: string,
  options: FindFileInTreeOptions = {}
): Promise<Locator> {
  const { click = false, timeout = 10000 } = options;
  const { folderSegments, targetName } = normalizePath(filePath);

  await ensureSidebarAndExplorerOpen(page, timeout);
  await ensureShowAllFiles(page, true, timeout);
  const treeContainer = await waitForTreeReady(page, timeout);

  let currentScope: Locator = treeContainer;

  // Navigate through each folder segment sequentially
  for (let i = 0; i < folderSegments.length; i++) {
    const segment = folderSegments[i];
    const currentFolderPath = folderSegments.slice(0, i + 1).join('/');

    // Check for UI error
    const errorNotice = currentScope.locator('.text-destructive');
    if ((await errorNotice.count().catch(() => 0)) > 0 && (await errorNotice.first().isVisible().catch(() => false))) {
      const errMsg = (await errorNotice.first().innerText().catch(() => 'Unknown directory error')).trim();
      throw new Error(
        `find_file_in_tree: Backend error loading folder "${currentFolderPath}": "${errMsg}"`
      );
    }

    const directFolderItem = currentScope
      .locator('> div[role="none"] > div[role="treeitem"][aria-expanded]')
      .filter({
        has: page.locator('span', { hasText: new RegExp(`^${escapeRegex(segment)}$`, 'i') }),
      })
      .first();

    let folderItem = directFolderItem;
    const isDirectVisible = await directFolderItem.isVisible().catch(() => false);
    if (!isDirectVisible) {
      folderItem = currentScope
        .locator('[role="treeitem"][aria-expanded]')
        .filter({
          has: page.locator('span', { hasText: new RegExp(`^${escapeRegex(segment)}$`, 'i') }),
        })
        .first();
    }

    try {
      await folderItem.waitFor({ state: 'visible', timeout });
    } catch {
      const availableItems = await currentScope
        .locator('[role="treeitem"]')
        .allInnerTexts()
        .catch(() => []);
      throw new Error(
        `find_file_in_tree: Could not find folder segment "${segment}" (path so far: "${currentFolderPath}") in filePath "${filePath}". Available items: [${availableItems.join(', ')}]`
      );
    }

    const isExpanded = (await folderItem.getAttribute('aria-expanded')) === 'true';
    if (!isExpanded) {
      await folderItem.click();
      await expect(folderItem).toHaveAttribute('aria-expanded', 'true', { timeout });
    }

    const spinner = folderItem.locator('.animate-spin');
    if ((await spinner.count().catch(() => 0)) > 0) {
      await spinner.waitFor({ state: 'detached', timeout }).catch(() => {});
    }

    const groupContainer = folderItem
      .locator('xpath=following-sibling::div[@role="group"]')
      .first();

    try {
      await groupContainer.waitFor({ state: 'visible', timeout });
    } catch {
      const subError = folderItem.locator('xpath=following-sibling::*').locator('.text-destructive');
      if (await subError.isVisible().catch(() => false)) {
        const errText = await subError.innerText().catch(() => '');
        throw new Error(
          `find_file_in_tree: Failed to expand folder "${segment}": ${errText}`
        );
      }
      throw new Error(
        `find_file_in_tree: Folder "${segment}" was expanded, but its child group container did not become visible.`
      );
    }

    const groupError = groupContainer.locator('.text-destructive');
    if (await groupError.isVisible().catch(() => false)) {
      const errText = (await groupError.innerText().catch(() => '')).trim();
      throw new Error(
        `find_file_in_tree: Failed to load folder "${segment}": ${errText}`
      );
    }

    currentScope = groupContainer;
  }

  // Find the target file in the resolved folder scope
  const directFileItem = currentScope
    .locator('> div[role="none"] > div[role="treeitem"]:not([aria-expanded])')
    .filter({
      has: page.locator('span', { hasText: new RegExp(`^${escapeRegex(targetName)}$`, 'i') }),
    })
    .first();

  let fileItem = directFileItem;
  const isFileDirectVisible = await directFileItem.isVisible().catch(() => false);
  if (!isFileDirectVisible) {
    fileItem = currentScope
      .locator('[role="treeitem"]:not([aria-expanded]), [role="treeitem"]')
      .filter({
        has: page.locator('span', { hasText: new RegExp(`^${escapeRegex(targetName)}$`, 'i') }),
      })
      .first();
  }

  try {
    await fileItem.waitFor({ state: 'visible', timeout });
  } catch {
    const scopeError = currentScope.locator('.text-destructive');
    if (await scopeError.isVisible().catch(() => false)) {
      const errText = (await scopeError.innerText().catch(() => '')).trim();
      throw new Error(
        `find_file_in_tree: Failed to load folder for file "${targetName}": ${errText}`
      );
    }
    const availableFiles = await currentScope
      .locator('[role="treeitem"]')
      .allInnerTexts()
      .catch(() => []);
    const parentDir = folderSegments.join('/') || 'root';
    throw new Error(
      `find_file_in_tree: Could not find file "${targetName}" in folder "${parentDir}" for filePath "${filePath}". Available items: [${availableFiles.join(', ')}]`
    );
  }

  if (click) {
    await fileItem.scrollIntoViewIfNeeded().catch(() => {});
    await fileItem.click();
  }

  return fileItem;
}

/**
 * Locates a file in the flat filtered list view (FilteredFileListing).
 *
 * @param page The Playwright Page object
 * @param filePathOrName The file name or path to find in the filtered list
 * @param options Additional options such as click and timeout
 * @returns The Playwright Locator for the target listitem
 */
export async function findFileInList(
  page: Page,
  filePathOrName: string,
  options: FindFileInListOptions = {}
): Promise<Locator> {
  const { click = false, timeout = 10000 } = options;
  const { targetName } = normalizePath(filePathOrName);

  await ensureSidebarAndExplorerOpen(page, timeout);
  await ensureShowAllFiles(page, false, timeout);

  const listContainer = page.locator('[role="list"][aria-label="Filtered Files"]');
  await listContainer.waitFor({ state: 'visible', timeout });

  const listItem = listContainer
    .locator('[role="listitem"]')
    .filter({
      has: page.locator('span', { hasText: new RegExp(`^${escapeRegex(targetName)}$`, 'i') }),
    })
    .first();

  try {
    await listItem.waitFor({ state: 'visible', timeout });
  } catch {
    const availableItems = await listContainer
      .locator('[role="listitem"]')
      .allInnerTexts()
      .catch(() => []);
    throw new Error(
      `find_file_in_list: Could not find file "${targetName}" in Filtered Files list. Available items: [${availableItems.join(', ')}]`
    );
  }

  if (click) {
    await listItem.scrollIntoViewIfNeeded().catch(() => {});
    await listItem.click();
  }

  return listItem;
}

/**
 * Refreshes a subfolder in the tree view by clicking its refresh button and waiting for completion.
 */
export async function refreshFolderInTree(
  page: Page,
  folderPath: string,
  options: RefreshFolderInTreeOptions = {}
): Promise<void> {
  const { timeout = 10000 } = options;
  const { targetName } = normalizePath(folderPath);

  // Locate and ensure the folder is expanded
  const folderItem = await findFolderInTree(page, folderPath, { expand: true, timeout });

  // Hover over the folder to reveal the refresh button
  await folderItem.hover();

  const refreshBtn = folderItem
    .locator('button')
    .filter({
      has: page.locator(`[aria-label="Refresh ${targetName}"], [title="Refresh ${targetName}"]`),
    })
    .first();

  let targetBtn = refreshBtn;
  if (!(await refreshBtn.isVisible().catch(() => false))) {
    targetBtn = folderItem.locator(`button[title="Refresh ${targetName}"], button[aria-label="Refresh ${targetName}"]`).first();
  }

  await targetBtn.waitFor({ state: 'visible', timeout: Math.min(timeout, 5000) });
  await targetBtn.click();

  // Wait for spinning animation to finish
  const spinner = targetBtn.locator('.animate-spin');
  if ((await spinner.count().catch(() => 0)) > 0) {
    await spinner.waitFor({ state: 'detached', timeout }).catch(() => {});
  }
}

/**
 * Collapses an expanded folder in the tree view.
 */
export async function collapseFolderInTree(
  page: Page,
  folderPath: string,
  options: CollapseFolderInTreeOptions = {}
): Promise<Locator> {
  const { timeout = 10000 } = options;
  const folderItem = await findFolderInTree(page, folderPath, { expand: false, timeout });

  const isExpanded = (await folderItem.getAttribute('aria-expanded')) === 'true';
  if (isExpanded) {
    await folderItem.click();
    await expect(folderItem).toHaveAttribute('aria-expanded', 'false', { timeout });
  }

  return folderItem;
}

export type TreeFixtures = {
  /**
   * Recursively navigates and expands the Workspace Explorer file tree to find a file.
   */
  find_file_in_tree: FindFileInTreeFn;
  /**
   * Alias for find_file_in_tree.
   */
  findFileInTree: FindFileInTreeFn;
  /**
   * Locates and navigates to a folder in the file tree.
   */
  find_folder_in_tree: FindFolderInTreeFn;
  /**
   * Alias for find_folder_in_tree.
   */
  findFolderInTree: FindFolderInTreeFn;
  /**
   * Locates a file in the flat Filtered Files list.
   */
  find_file_in_list: FindFileInListFn;
  /**
   * Alias for find_file_in_list.
   */
  findFileInList: FindFileInListFn;
  /**
   * Refreshes a subfolder in the tree view.
   */
  refresh_folder_in_tree: RefreshFolderInTreeFn;
  /**
   * Alias for refresh_folder_in_tree.
   */
  refreshFolderInTree: RefreshFolderInTreeFn;
  /**
   * Collapses an expanded folder in the tree view.
   */
  collapse_folder_in_tree: CollapseFolderInTreeFn;
  /**
   * Alias for collapse_folder_in_tree.
   */
  collapseFolderInTree: CollapseFolderInTreeFn;
};

export const test = projectTest.extend<TreeFixtures>({
  find_file_in_tree: async ({ authedPage }, use) => {
    const findFn = (filePath: string, options?: FindFileInTreeOptions) => {
      return findFileInTree(authedPage, filePath, options);
    };
    await use(findFn);
  },

  findFileInTree: async ({ authedPage }, use) => {
    const findFn = (filePath: string, options?: FindFileInTreeOptions) => {
      return findFileInTree(authedPage, filePath, options);
    };
    await use(findFn);
  },

  find_folder_in_tree: async ({ authedPage }, use) => {
    const findFn = (folderPath: string, options?: FindFolderInTreeOptions) => {
      return findFolderInTree(authedPage, folderPath, options);
    };
    await use(findFn);
  },

  findFolderInTree: async ({ authedPage }, use) => {
    const findFn = (folderPath: string, options?: FindFolderInTreeOptions) => {
      return findFolderInTree(authedPage, folderPath, options);
    };
    await use(findFn);
  },

  find_file_in_list: async ({ authedPage }, use) => {
    const findFn = (filePathOrName: string, options?: FindFileInListOptions) => {
      return findFileInList(authedPage, filePathOrName, options);
    };
    await use(findFn);
  },

  findFileInList: async ({ authedPage }, use) => {
    const findFn = (filePathOrName: string, options?: FindFileInListOptions) => {
      return findFileInList(authedPage, filePathOrName, options);
    };
    await use(findFn);
  },

  refresh_folder_in_tree: async ({ authedPage }, use) => {
    const refreshFn = (folderPath: string, options?: RefreshFolderInTreeOptions) => {
      return refreshFolderInTree(authedPage, folderPath, options);
    };
    await use(refreshFn);
  },

  refreshFolderInTree: async ({ authedPage }, use) => {
    const refreshFn = (folderPath: string, options?: RefreshFolderInTreeOptions) => {
      return refreshFolderInTree(authedPage, folderPath, options);
    };
    await use(refreshFn);
  },

  collapse_folder_in_tree: async ({ authedPage }, use) => {
    const collapseFn = (folderPath: string, options?: CollapseFolderInTreeOptions) => {
      return collapseFolderInTree(authedPage, folderPath, options);
    };
    await use(collapseFn);
  },

  collapseFolderInTree: async ({ authedPage }, use) => {
    const collapseFn = (folderPath: string, options?: CollapseFolderInTreeOptions) => {
      return collapseFolderInTree(authedPage, folderPath, options);
    };
    await use(collapseFn);
  },
});

export { expect, type Page, type Locator };
