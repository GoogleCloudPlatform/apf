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

// @vitest-environment jsdom
import { renderHook, act, waitFor } from "@testing-library/react"
import { describe, it, expect, vi, beforeEach } from "vitest"
import "@testing-library/jest-dom/vitest"
import { useProjectDetails } from "./useProjectDetails"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import React from "react"
import { toast } from "sonner"

vi.mock("next/navigation", () => ({
  useRouter: () => ({
    push: vi.fn(),
    back: vi.fn(),
  }),
}))

vi.mock("sonner", () => ({
  toast: {
    error: vi.fn(),
    warning: vi.fn(),
    success: vi.fn(),
  },
}))

vi.mock("@/stores/app", () => ({
  useAppStore: () => ({
    setPageTitle: vi.fn(),
  }),
}))

vi.mock("@/stores/auth", () => ({
  useAuthStore: () => ({
    email: "test@example.com",
  }),
}))

const mockApiGet = vi.fn()
const mockApiPost = vi.fn()
const mockApiPut = vi.fn()

vi.mock("@/lib/api", () => ({
  api: {
    get: (...args: any[]) => mockApiGet(...args),
    post: (...args: any[]) => mockApiPost(...args),
    put: (...args: any[]) => mockApiPut(...args),
  },
}))

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  })
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  )
}

describe("useProjectDetails hook", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockApiGet.mockImplementation((url: string) => {
      if (url === "/api/v1/config/app-integrations") {
        return Promise.resolve({
          ok: true,
          json: () =>
            Promise.resolve([
              { id: "github", name: "GitHub", isEnabled: true, fields: [] },
              { id: "gemini", name: "Gemini", isEnabled: true, fields: [{ name: "gemini_api_key", value: "" }] },
            ]),
        })
      }
      if (url.includes("/apps/gemini/credentials")) {
        return Promise.resolve({
          ok: true,
          json: () =>
            Promise.resolve({
              integration_id: "gemini",
              enabled: true,
              credentialId: "gemini",
              isOverridden: false,
              fields: [{ name: "gemini_api_key", value: "gemini_simulated_global_key" }],
            }),
        })
      }
      if (url.includes("/credentials")) {
        return Promise.resolve({
          ok: true,
          json: () =>
            Promise.resolve({
              enabled: true,
              credentialId: "github",
              isOverridden: false,
              fields: [],
            }),
        })
      }
      if (url.includes("/api/v1/integrations/github/profile")) {
        return Promise.resolve({
          ok: true,
          json: () =>
            Promise.resolve({
              connected: true,
              username: "octocat",
              email: "octocat@github.com",
            }),
        })
      }
      if (url.includes("/api/v1/projects/existing-1")) {
        return Promise.resolve({
          ok: true,
          json: () =>
            Promise.resolve({
              id: "existing-1",
              name: "Existing Project",
              description: "Desc",
              icon: "🚀",
              app_integrations: [
                {
                  integration_id: "github",
                  enabled: true,
                  settings: { repoUrl: "https://github.com/org/repo-old", email: "octocat@github.com" },
                },
                {
                  integration_id: "gemini",
                  enabled: true,
                  settings: {},
                },
              ],
            }),
        })
      }
      if (url.includes("/api/v1/projects/empty-repo-project")) {
        return Promise.resolve({
          ok: true,
          json: () =>
            Promise.resolve({
              id: "empty-repo-project",
              name: "Empty Repo Project",
              description: "Desc",
              icon: "🚀",
              app_integrations: [
                {
                  integration_id: "gemini",
                  enabled: true,
                  settings: {},
                },
                {
                  integration_id: "github",
                  enabled: true,
                  settings: { email: "octocat@github.com" },
                },
              ],
            }),
        })
      }
      return Promise.reject(new Error("Unknown endpoint"))
    })
  })

  it("Requirement 1: Prevents saving project if repo is not selected and git is enabled", async () => {
    const { result } = renderHook(() => useProjectDetails("new"), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isLoading).toBe(false))

    act(() => {
      result.current.setName("New Project Name")
      result.current.setRepoUrl("")
    })

    act(() => {
      result.current.handleSubmit()
    })

    expect(toast.error).toHaveBeenCalledWith(
      "Please either disable Git integration or select a repository."
    )
    expect(mockApiPost).not.toHaveBeenCalled()
  })

  it("Prevents saving project when Gemini API key is missing and shows clear error message", async () => {
    mockApiGet.mockImplementation((url: string) => {
      if (url === "/api/v1/config/app-integrations") {
        return Promise.resolve({
          ok: true,
          json: () =>
            Promise.resolve([
              { id: "github", name: "GitHub", isEnabled: true, fields: [] },
              { id: "gemini", name: "Gemini", isEnabled: true, fields: [{ name: "gemini_api_key", value: "" }] },
            ]),
        })
      }
      if (url.includes("/apps/gemini/credentials")) {
        // No global gemini credentials
        return Promise.resolve({
          ok: false,
          status: 404,
          json: () => Promise.reject(new Error("Not found")),
        })
      }
      if (url.includes("/api/v1/integrations/github/profile")) {
        return Promise.resolve({
          ok: true,
          json: () =>
            Promise.resolve({
              connected: true,
              username: "octocat",
              email: "octocat@github.com",
            }),
        })
      }
      if (url.includes("/credentials")) {
        return Promise.resolve({
          ok: true,
          json: () =>
            Promise.resolve({
              enabled: true,
              credentialId: "github",
              isOverridden: false,
              fields: [],
            }),
        })
      }
      return Promise.reject(new Error("Unknown endpoint"))
    })

    const { result } = renderHook(() => useProjectDetails("new"), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isLoading).toBe(false))

    act(() => {
      result.current.setName("New Project Without Gemini")
      result.current.setRepoUrl("https://github.com/org/repo")
      result.current.setGitEmail("octocat@github.com")
    })

    act(() => {
      result.current.handleSubmit()
    })

    expect(toast.error).toHaveBeenCalledWith(
      "A project cannot be saved without a valid Gemini API Key. Please provide a Gemini API Key in Application Credentials or configure it in Settings."
    )
    expect(mockApiPost).not.toHaveBeenCalled()
  })

  it("Prevents saving existing project when Gemini is disabled or overridden with empty key", async () => {
    mockApiGet.mockImplementation((url: string) => {
      if (url === "/api/v1/config/app-integrations") {
        return Promise.resolve({
          ok: true,
          json: () =>
            Promise.resolve([
              { id: "github", name: "GitHub", isEnabled: true, fields: [] },
              { id: "gemini", name: "Gemini", isEnabled: true, fields: [{ name: "gemini_api_key", value: "" }] },
            ]),
        })
      }
      if (url.includes("/api/v1/projects/existing-1")) {
        return Promise.resolve({
          ok: true,
          json: () =>
            Promise.resolve({
              id: "existing-1",
              name: "Existing Project",
              description: "Desc",
              icon: "🚀",
              app_integrations: [
                {
                  integration_id: "github",
                  enabled: true,
                  settings: { repoUrl: "https://github.com/org/repo-old" },
                },
              ],
            }),
        })
      }
      if (url.includes("/credentials")) {
        return Promise.resolve({
          ok: true,
          json: () =>
            Promise.resolve({
              enabled: false,
              credentialId: "gemini",
              isOverridden: true,
              fields: [{ name: "gemini_api_key", value: "" }],
            }),
        })
      }
      return Promise.reject(new Error("Unknown endpoint"))
    })

    const { result } = renderHook(() => useProjectDetails("existing-1"), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isLoading).toBe(false))

    act(() => {
      result.current.setRepoUrl("https://github.com/org/repo-old")
      result.current.handleOverrideCredentials("gemini")
      result.current.handleFieldChange("gemini", "gemini_api_key", "")
    })

    act(() => {
      result.current.handleSubmit()
    })

    expect(toast.error).toHaveBeenCalledWith(
      "A project cannot be saved without a valid Gemini API Key. Please provide a Gemini API Key in Application Credentials or configure it in Settings."
    )
    expect(mockApiPut).not.toHaveBeenCalled()
  })

  it("Allows saving project when a valid Gemini API key is provided via override", async () => {
    mockApiGet.mockImplementation((url: string) => {
      if (url === "/api/v1/config/app-integrations") {
        return Promise.resolve({
          ok: true,
          json: () =>
            Promise.resolve([
              { id: "github", name: "GitHub", isEnabled: true, fields: [] },
              { id: "gemini", name: "Gemini", isEnabled: true, fields: [{ name: "gemini_api_key", value: "" }] },
            ]),
        })
      }
      if (url.includes("/api/v1/projects/check-repo")) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ exists: false }),
        })
      }
      if (url.includes("/api/v1/integrations/github/profile")) {
        return Promise.resolve({
          ok: true,
          json: () =>
            Promise.resolve({
              connected: true,
              username: "octocat",
              email: "octocat@github.com",
            }),
        })
      }
      if (url.includes("/credentials")) {
        return Promise.resolve({
          ok: false,
          status: 404,
          json: () => Promise.reject(new Error("Not found")),
        })
      }
      return Promise.reject(new Error("Unknown endpoint"))
    })

    mockApiPost.mockImplementation((url: string) => {
      if (url.includes("/validate-permissions")) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ read: true, write: true, pullRequest: true, baseBranchExists: true }),
        })
      }
      if (url === "/api/v1/projects") {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ id: "new-proj-123" }),
        })
      }
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve({}),
      })
    })

    const { result } = renderHook(() => useProjectDetails("new"), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isLoading).toBe(false))

    await act(async () => {
      result.current.setName("New Project Valid Gemini")
      result.current.setRepoUrl("https://github.com/org/repo")
      await result.current.handleToggleApp("gemini", true)
      result.current.handleOverrideCredentials("gemini")
      result.current.handleFieldChange("gemini", "gemini_api_key", "AIzaSyTestValidGeminiKey")
    })

    expect(result.current.hasValidGeminiKey()).toBe(true)

    await act(async () => {
      result.current.handleSubmit()
    })

    await waitFor(() => {
      expect(mockApiPost).toHaveBeenCalledWith(
        "/api/v1/projects",
        expect.objectContaining({
          name: "New Project Valid Gemini",
        })
      )
    })
  })

  it("Requirement 2: Shows warning modal when repo is changed on existing project (from another repo)", async () => {
    const { result } = renderHook(() => useProjectDetails("existing-1"), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isLoading).toBe(false))

    act(() => {
      result.current.setRepoUrl("https://github.com/org/repo-new")
    })

    expect(result.current.isRepoChangeModalOpen).toBe(false)

    act(() => {
      result.current.handleSubmit()
    })

    expect(result.current.isRepoChangeModalOpen).toBe(true)
    expect(mockApiPut).not.toHaveBeenCalled()
  })

  it("Requirement 2: Shows warning modal when repo is changed on existing project (from empty repo)", async () => {
    const { result } = renderHook(() => useProjectDetails("empty-repo-project"), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isLoading).toBe(false))

    act(() => {
      result.current.setRepoUrl("https://github.com/org/newly-added-repo")
    })

    expect(result.current.isRepoChangeModalOpen).toBe(false)

    act(() => {
      result.current.handleSubmit()
    })

    expect(result.current.isRepoChangeModalOpen).toBe(true)
    expect(mockApiPut).not.toHaveBeenCalled()
  })

  it("Validates that Git Author Email is required when GitHub is enabled", async () => {
    const { result } = renderHook(() => useProjectDetails("new"), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isLoading).toBe(false))

    act(() => {
      result.current.setName("New Project No Email")
      result.current.setRepoUrl("https://github.com/org/repo")
      result.current.setGitEmail("")
    })

    act(() => {
      result.current.handleSubmit()
    })

    expect(toast.error).toHaveBeenCalledWith(
      "Git Author Email is required when GitHub integration is enabled."
    )
    expect(mockApiPost).not.toHaveBeenCalled()
  })

  it("Loads existing Git Author Email from project AppIntegrations", async () => {
    const { result } = renderHook(() => useProjectDetails("existing-1"), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isLoading).toBe(false))

    expect(result.current.gitEmail).toBe("octocat@github.com")
  })
})
