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

import { test as authTest, expect, type Page } from './login-fixture';
import { authenticatePage } from './login-fixture';

export type CreatedProject = {
  id: string;
  name: string;
  description?: string;
  icon?: string;
  status?: string;
  visibility?: string;
};

export async function createProjectApi(
  page: Page,
  name: string = `E2E Fixture Project ${Date.now()}`,
  description: string = 'Created automatically by Playwright createProject fixture.',
  options: { icon?: string; visibility?: string } = {}
): Promise<CreatedProject> {
  const payload = {
    name,
    description,
    icon: options.icon || '🚀',
    visibility: options.visibility || 'public',
    repoAccessToken: '',
    baseBranch: 'main',
  };

  return await page.evaluate(async (projectData: typeof payload) => {
    let token = localStorage.getItem('auth_token') || '';
    let orgId = '';
    if (token) {
      try {
        const authRes = await fetch('/api/v1/auth/me', {
          headers: { Authorization: `Bearer ${token}` },
        });
        if (authRes.ok) {
          const authData = await authRes.json();
          orgId = authData?.organization_id || '';
        }
      } catch (_) {}
    }

    if (!orgId && token) {
      try {
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
                token = selectData.token;
                localStorage.setItem('auth_token', token);
                orgId = firstOrg.id;
              }
            }
          }
        }
      } catch (_) {}
    }

    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };
    if (token) headers['Authorization'] = `Bearer ${token}`;
    if (orgId) headers['X-Organization-Id'] = orgId;

    const res = await fetch('/api/v1/projects', {
      method: 'POST',
      headers,
      body: JSON.stringify(projectData),
    });

    if (!res.ok) {
      const errText = await res.text();
      throw new Error(`Failed to create project via API: HTTP ${res.status} - ${errText}`);
    }

    return await res.json();
  }, payload);
}

export async function deleteProjectApi(page: Page, projectId: string): Promise<void> {
  await page.evaluate(async (id: string) => {
    let token = localStorage.getItem('auth_token') || '';
    let orgId = '';
    if (token) {
      try {
        const authRes = await fetch('/api/v1/auth/me', {
          headers: { Authorization: `Bearer ${token}` },
        });
        if (authRes.ok) {
          const authData = await authRes.json();
          orgId = authData?.organization_id || '';
        }
      } catch (_) {}
    }

    if (!orgId && token) {
      try {
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
                token = selectData.token;
                localStorage.setItem('auth_token', token);
                orgId = firstOrg.id;
              }
            }
          }
        }
      } catch (_) {}
    }

    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };
    if (token) headers['Authorization'] = `Bearer ${token}`;
    if (orgId) headers['X-Organization-Id'] = orgId;

    const res = await fetch(`/api/v1/projects/${id}`, {
      method: 'DELETE',
      headers,
    });

    if (!res.ok) {
      console.warn(`Failed to delete project ${id}: HTTP status ${res.status}`);
    }
  }, projectId);
}

export async function deleteProjectByNameApi(page: Page, projectName: string): Promise<void> {
  await page.evaluate(async (name: string) => {
    let token = localStorage.getItem('auth_token') || '';
    let orgId = '';
    if (token) {
      try {
        const authRes = await fetch('/api/v1/auth/me', {
          headers: { Authorization: `Bearer ${token}` },
        });
        if (authRes.ok) {
          const authData = await authRes.json();
          orgId = authData?.organization_id || '';
        }
      } catch (_) {}
    }

    if (!orgId && token) {
      try {
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
                token = selectData.token;
                localStorage.setItem('auth_token', token);
                orgId = firstOrg.id;
              }
            }
          }
        }
      } catch (_) {}
    }

    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };
    if (token) headers['Authorization'] = `Bearer ${token}`;
    if (orgId) headers['X-Organization-Id'] = orgId;

    try {
      const res = await fetch('/api/v1/projects', { headers });
      if (!res.ok) return;
      const data = await res.json();
      const allProjects = [...(data.myProjects || []), ...(data.browseProjects || [])];
      const target = allProjects.find((p: any) => p.name === name || p.id === name);
      
      if (target && target.id) {
        await fetch(`/api/v1/projects/${target.id}`, {
          method: 'DELETE',
          headers,
        });
      }
    } catch (e) {
      console.warn(`Error deleting project by name ${name}:`, e);
    }
  }, projectName);
}

export type ProjectFixtures = {
  /**
   * Creates a real project on the backend via API and automatically registers it for teardown cleanup.
   */
  createProject: (
    name?: string,
    description?: string,
    options?: { icon?: string; visibility?: string }
  ) => Promise<CreatedProject>;

  /**
   * Deletes a project by its unique project ID via the backend API.
   */
  deleteProject: (projectId: string) => Promise<void>;

  /**
   * Finds a project by its display name (or ID) in the project catalog and deletes it via the backend API.
   */
  deleteProjectByName: (projectName: string) => Promise<void>;

  /**
   * Registers a project ID or project name to be automatically deleted after the test completes.
   */
  projectCleanup: (projectNameOrId: string) => void;
};

export type ProjectWorkerFixtures = {
  /**
   * A single project created once at worker/suite start and cleaned up at worker teardown.
   */
  sharedProject: CreatedProject;
};

export const test = authTest.extend<ProjectFixtures, ProjectWorkerFixtures>({
  sharedProject: [
    async ({ browser }, use) => {
      const page = await browser.newPage();
      await authenticatePage(page);
      const project = await createProjectApi(page, `Shared E2E Project ${Date.now()}`);

      await use(project);

      try {
        await deleteProjectApi(page, project.id);
        await deleteProjectByNameApi(page, project.name);
      } catch (e) {
        console.error(`Failed to clean up shared project ${project.id}:`, e);
      } finally {
        await page.close();
      }
    },
    { scope: 'worker' },
  ],

  createProject: async ({ authedPage, projectCleanup }, use) => {
    const createProjectFn = async (
      name?: string,
      description?: string,
      options?: { icon?: string; visibility?: string }
    ): Promise<CreatedProject> => {
      const result = await createProjectApi(authedPage, name, description, options);

      // Automatically register the created project for deletion in test teardown
      if (result && result.id) {
        projectCleanup(result.id);
      } else if (result && result.name) {
        projectCleanup(result.name);
      }

      return result;
    };

    await use(createProjectFn);
  },

  deleteProject: async ({ authedPage }, use) => {
    const deleteProjectFn = async (projectId: string) => {
      await deleteProjectApi(authedPage, projectId);
    };

    await use(deleteProjectFn);
  },

  deleteProjectByName: async ({ authedPage }, use) => {
    const deleteProjectByNameFn = async (projectName: string) => {
      await deleteProjectByNameApi(authedPage, projectName);
    };

    await use(deleteProjectByNameFn);
  },

  projectCleanup: async ({ deleteProject, deleteProjectByName }, use) => {
    const itemsToDelete: string[] = [];

    const trackFn = (projectNameOrId: string) => {
      if (!itemsToDelete.includes(projectNameOrId)) {
        itemsToDelete.push(projectNameOrId);
      }
    };

    await use(trackFn);

    // Teardown: Automatically delete all tracked projects after the test finishes
    for (const item of itemsToDelete) {
      try {
        await deleteProject(item);
        await deleteProjectByName(item);
      } catch (e) {
        console.error(`Failed to clean up project ${item} during teardown:`, e);
      }
    }
  },
});

export { expect, type Page };
