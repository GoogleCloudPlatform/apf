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
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import "@testing-library/jest-dom/vitest"
import { useWorkspace } from "./useWorkspace"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import React from "react"

const mockPush = vi.fn()
const mockGetSearchParam = vi.fn()

vi.mock("next/navigation", () => ({
  useRouter: () => ({
    push: mockPush,
  }),
  useSearchParams: () => ({
    get: (key: string) => mockGetSearchParam(key),
  }),
  usePathname: () => "/projects/proj-123/workspace",
}))

vi.mock("sonner", () => ({
  toast: {
    error: vi.fn(),
    success: vi.fn(),
    info: vi.fn(),
    loading: vi.fn(),
  },
}))

vi.mock("@/stores/app", () => ({
  useAppStore: () => ({
    setPageTitle: vi.fn(),
    isDebugBarOpen: false,
    setIsDebugBarOpen: vi.fn(),
    setSessionId: vi.fn(),
    getSessionId: vi.fn(),
  }),
}))

vi.mock("@/stores/activity", () => ({
  useActivityStore: () => ({
    subscribe: vi.fn(),
    unsubscribe: vi.fn(),
  }),
  getPersonaColorClass: () => "bg-blue-500",
}))

const mockApiGet = vi.fn()
const mockApiPost = vi.fn()
const mockApiPut = vi.fn()
const mockApiDelete = vi.fn()
const mockApiPatch = vi.fn()

vi.mock("@/lib/api", () => ({
  api: {
    get: (...args: any[]) => mockApiGet(...args),
    post: (...args: any[]) => mockApiPost(...args),
    put: (...args: any[]) => mockApiPut(...args),
    delete: (...args: any[]) => mockApiDelete(...args),
    patch: (...args: any[]) => mockApiPatch(...args),
  },
}))

const mockArtifacts = [
  {
    id: "art-prd",
    projectId: "proj-123",
    workflowId: "bmad",
    title: "PRD",
    path: "docs/specs/prd.md",
    phase: "PLANNING",
    status: "Complete" as const,
    owner: "agent",
    content: "# PRD Content",
    required: true,
    isCustom: false,
    artifactType: "product-req-doc",
  },
  {
    id: "art-arch",
    projectId: "proj-123",
    workflowId: "bmad",
    title: "Architecture",
    path: "docs/specs/architecture.md",
    phase: "SOLUTIONING",
    status: "Complete" as const,
    owner: "agent",
    content: "# Architecture Content",
    required: true,
    isCustom: false,
    artifactType: "architecture-doc",
  },
]

describe("useWorkspace hook - file-write auto-refetch in preview pane", () => {
  let queryClient: QueryClient

  const createWrapper = () => {
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
      },
    })
    return ({ children }: { children: React.ReactNode }) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    )
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockGetSearchParam.mockReturnValue(null)

    mockApiGet.mockImplementation((url: string) => {
      if (url.includes("/artifacts")) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve(mockArtifacts),
        })
      }
      if (url.includes("/workflows")) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve([]),
        })
      }
      if (url.includes("/sessions")) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve([]),
        })
      }
      if (url.includes("/api/v1/projects/proj-123")) {
        return Promise.resolve({
          ok: true,
          json: () =>
            Promise.resolve({
              id: "proj-123",
              name: "Test Project",
              status: "Active",
            }),
        })
      }
      return Promise.reject(new Error(`Unhandled GET url: ${url}`))
    })

    mockApiPost.mockImplementation(() =>
      Promise.resolve({
        ok: true,
        json: () => Promise.resolve({ session: null }),
      })
    )

    mockApiPatch.mockImplementation(() =>
      Promise.resolve({
        ok: true,
        json: () => Promise.resolve({}),
      })
    )
  })

  afterEach(() => {
    queryClient?.clear()
  })

  it("automatically refetches preview document content when preview is open and updated document matches", async () => {
    const wrapper = createWrapper()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const refetchSpy = vi.spyOn(queryClient, "refetchQueries")

    const { result } = renderHook(() => useWorkspace("proj-123"), { wrapper })

    await waitFor(() => expect(result.current.isLoading).toBe(false))

    // User opens the PRD artifact in the preview pane
    act(() => {
      result.current.setActiveArtifactTitle("PRD")
    })

    expect(result.current.activeArtifactTitle).toBe("PRD")

    // Simulate receiving a file-write event for the PRD file
    const fileWriteEventPayload = JSON.stringify({
      isNotification: true,
      notificationDetails: {
        name: "file-write",
        params: {
          filePath: "docs/specs/prd.md",
        },
      },
      text: "File updated: docs/specs/prd.md",
    })

    act(() => {
      result.current.processAgentOutput(fileWriteEventPayload)
    })

    // Expect queryClient to invalidate and refetch artifact-content queries
    expect(invalidateSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        queryKey: ["artifact-content", "proj-123"],
      })
    )
    expect(refetchSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        queryKey: ["artifact-content", "proj-123"],
      })
    )

    // Also expect debug log to record the preview refetch
    expect(
      result.current.debugLogs.some((log) =>
        log.includes("Document in preview pane matches updated file")
      )
    ).toBe(true)
  })

  it("does not refetch preview document content when preview is open with a non-matching document", async () => {
    const wrapper = createWrapper()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const refetchSpy = vi.spyOn(queryClient, "refetchQueries")

    const { result } = renderHook(() => useWorkspace("proj-123"), { wrapper })

    await waitFor(() => expect(result.current.isLoading).toBe(false))

    // User opens the Architecture artifact in the preview pane
    act(() => {
      result.current.setActiveArtifactTitle("Architecture")
    })

    invalidateSpy.mockClear()
    refetchSpy.mockClear()

    // Simulate receiving a file-write event for the PRD file
    const fileWriteEventPayload = JSON.stringify({
      isNotification: true,
      notificationDetails: {
        name: "file-write",
        params: {
          filePath: "docs/specs/prd.md",
        },
      },
      text: "File updated: docs/specs/prd.md",
    })

    act(() => {
      result.current.processAgentOutput(fileWriteEventPayload)
    })

    // artifact-content queries should NOT have been invalidated or refetched
    expect(invalidateSpy).not.toHaveBeenCalledWith(
      expect.objectContaining({
        queryKey: ["artifact-content", "proj-123"],
      })
    )
    expect(refetchSpy).not.toHaveBeenCalledWith(
      expect.objectContaining({
        queryKey: ["artifact-content", "proj-123"],
      })
    )
  })

  it("invokes fileExplorerRef.refreshFolderForPath when receiving Tool result (file_writer)", async () => {
    const wrapper = createWrapper()
    const mockRefreshFolderForPath = vi.fn().mockResolvedValue(undefined)
    const fileExplorerRef = {
      current: {
        refreshFolderForPath: mockRefreshFolderForPath,
        refreshAll: vi.fn(),
      },
    }

    const { result } = renderHook(
      () => useWorkspace("proj-123", { fileExplorerRef: fileExplorerRef as any }),
      { wrapper }
    )

    await waitFor(() => expect(result.current.isLoading).toBe(false))

    const toolResultPayload = `Tool result (file_writer):

{
  "filePath": "AGENTS.md",
  "message": "File written successfully"
}`

    act(() => {
      result.current.processAgentOutput(toolResultPayload)
    })

    expect(mockRefreshFolderForPath).toHaveBeenCalledWith("AGENTS.md")
  })

  it("invokes fileExplorerRef.refreshFolderForPath when receiving File updated: path message", async () => {
    const wrapper = createWrapper()
    const mockRefreshFolderForPath = vi.fn().mockResolvedValue(undefined)
    const fileExplorerRef = {
      current: {
        refreshFolderForPath: mockRefreshFolderForPath,
        refreshAll: vi.fn(),
      },
    }

    const { result } = renderHook(
      () => useWorkspace("proj-123", { fileExplorerRef: fileExplorerRef as any }),
      { wrapper }
    )

    await waitFor(() => expect(result.current.isLoading).toBe(false))

    const fileUpdatedPayload = JSON.stringify({
      text: "File updated: docs/spec/brainstorming/brainstorm-stock-screening-app-2026-08-19/.memlog.md",
      isUser: false,
    })

    act(() => {
      result.current.processAgentOutput(fileUpdatedPayload)
    })

    expect(mockRefreshFolderForPath).toHaveBeenCalledWith(
      "docs/spec/brainstorming/brainstorm-stock-screening-app-2026-08-19/.memlog.md"
    )
  })

  it("updates artifact section without error when preview window is closed (activeArtifactTitle is null)", async () => {
    const wrapper = createWrapper()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const refetchSpy = vi.spyOn(queryClient, "refetchQueries")

    const { result } = renderHook(() => useWorkspace("proj-123"), { wrapper })

    await waitFor(() => expect(result.current.isLoading).toBe(false))

    expect(result.current.activeArtifactTitle).toBeNull()

    invalidateSpy.mockClear()
    refetchSpy.mockClear()

    // Simulate receiving a file-write event when preview is closed
    const fileWriteEventPayload = JSON.stringify({
      isNotification: true,
      notificationDetails: {
        name: "file-write",
        params: {
          filePath: "docs/specs/prd.md",
        },
      },
      text: "File updated: docs/specs/prd.md",
    })

    act(() => {
      result.current.processAgentOutput(fileWriteEventPayload)
    })

    // Should NOT trigger preview content refetches
    expect(invalidateSpy).not.toHaveBeenCalledWith(
      expect.objectContaining({
        queryKey: ["artifact-content", "proj-123"],
      })
    )
    expect(refetchSpy).not.toHaveBeenCalledWith(
      expect.objectContaining({
        queryKey: ["artifact-content", "proj-123"],
      })
    )

    // Should log the file-write notification for artifacts
    expect(
      result.current.debugLogs.some((log) =>
        log.includes("Found file-write notification for path: docs/specs/prd.md")
      )
    ).toBe(true)
  })

  it("filters out FAILED sessions when resolving active session", async () => {
    mockApiGet.mockImplementation((url: string) => {
      if (url.includes("/sessions")) {
        return Promise.resolve({
          ok: true,
          json: () =>
            Promise.resolve([
              {
                id: "sess-failed-1",
                projectId: "proj-123",
                phase: "ONBOARDING",
                status: "FAILED",
                createdAt: new Date().toISOString(),
              },
            ]),
        })
      }
      if (url.includes("/api/v1/projects/proj-123")) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ id: "proj-123", name: "Test Project", status: "Active" }),
        })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
    })

    const wrapper = createWrapper()
    const { result } = renderHook(() => useWorkspace("proj-123"), { wrapper })

    await waitFor(() => expect(result.current.isLoading).toBe(false))
    await waitFor(() => expect(result.current.isLoadingSessions).toBe(false))

    // The failed session must NOT be selected as active session ID
    expect(result.current.activeSessionId).toBeNull()
  })

  it("properly flags Gemini API key error with user guidance message on connection failure", async () => {
    mockApiPost.mockImplementation((url: string) => {
      if (url.includes("/sessions")) {
        return Promise.resolve({
          ok: false,
          status: 500,
          json: () =>
            Promise.resolve({
              error: "GEMINI_API_KEY_REQUIRED: Gemini API key is required. Please configure your key in User Secrets Settings or Project Settings.",
              code: "SESSION_INIT_FAILED",
            }),
        })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve({}) })
    })

    const wrapper = createWrapper()
    const { result } = renderHook(() => useWorkspace("proj-123"), { wrapper })

    await waitFor(() => expect(result.current.isLoading).toBe(false))

    act(() => {
      result.current.handleCreateNewSession()
    })

    await waitFor(() => {
      expect(result.current.sessionConnectionError).toContain("Gemini API key is missing or invalid")
      expect(result.current.activeSessionId).toBeNull()
    })
  })

  it("closes the canvas (resets activeArtifactTitle to null) when switching to another chat session", async () => {
    const wrapper = createWrapper()
    const { result } = renderHook(() => useWorkspace("proj-123"), { wrapper })

    await waitFor(() => expect(result.current.isLoading).toBe(false))

    // Open canvas
    act(() => {
      result.current.setActiveArtifactTitle("PRD.md")
    })
    expect(result.current.activeArtifactTitle).toBe("PRD.md")

    // Click on a different chat session
    act(() => {
      result.current.handleSwitchSession("sess-2")
    })

    // Canvas should now be closed
    expect(result.current.activeArtifactTitle).toBeNull()
  })

  it("closes the canvas (resets activeArtifactTitle to null) even when clicking on the currently active chat session", async () => {
    const wrapper = createWrapper()
    const { result } = renderHook(() => useWorkspace("proj-123"), { wrapper })

    await waitFor(() => expect(result.current.isLoading).toBe(false))

    // Open canvas
    act(() => {
      result.current.setActiveArtifactTitle("PRD.md")
    })
    expect(result.current.activeArtifactTitle).toBe("PRD.md")

    // Click on the same session
    act(() => {
      result.current.handleSwitchSession(result.current.activeSessionId || "sess-1")
    })

    // Canvas should be closed
    expect(result.current.activeArtifactTitle).toBeNull()
  })

  it("closes the canvas (resets activeArtifactTitle to null) when creating a new chat session", async () => {
    const wrapper = createWrapper()
    const { result } = renderHook(() => useWorkspace("proj-123"), { wrapper })

    await waitFor(() => expect(result.current.isLoading).toBe(false))

    // Open canvas
    act(() => {
      result.current.setActiveArtifactTitle("Architecture.md")
    })
    expect(result.current.activeArtifactTitle).toBe("Architecture.md")

    // Create a new session
    act(() => {
      result.current.handleCreateNewSession()
    })

    // Canvas should be closed
    expect(result.current.activeArtifactTitle).toBeNull()
  })

  it("handleCreateNewSessionForAction creates a new session, sets title, and switches active session", async () => {
    let currentSessions = [
      {
        id: "sess-existing",
        projectId: "proj-123",
        phase: "ONBOARDING",
        status: "ACTIVE",
        createdAt: new Date().toISOString(),
        lastActiveAt: new Date().toISOString(),
      },
    ]

    mockApiGet.mockImplementation((url: string) => {
      if (url.includes("/artifacts")) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve(mockArtifacts),
        })
      }
      if (url.includes("/workflows")) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve([]),
        })
      }
      if (url.includes("/sessions")) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve(currentSessions),
        })
      }
      if (url.includes("/api/v1/projects/proj-123")) {
        return Promise.resolve({
          ok: true,
          json: () =>
            Promise.resolve({
              id: "proj-123",
              name: "Test Project",
              status: "Active",
            }),
        })
      }
      return Promise.reject(new Error(`Unhandled GET url: ${url}`))
    })

    mockApiPost.mockImplementation((url: string) => {
      if (url.includes("/sessions?createNew=true")) {
        const newSession = {
          id: "sess-action-new",
          projectId: "proj-123",
          phase: "ONBOARDING",
          title: "",
          status: "ACTIVE",
          createdAt: new Date().toISOString(),
          lastActiveAt: new Date().toISOString(),
        }
        currentSessions = [newSession, ...currentSessions]
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ session: newSession }),
        })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve({}) })
    })

    const wrapper = createWrapper()
    const { result } = renderHook(() => useWorkspace("proj-123"), { wrapper })

    await waitFor(() => expect(result.current.isLoading).toBe(false))
    await waitFor(() => expect(result.current.activeSessionId).toBe("sess-existing"))

    await act(async () => {
      await result.current.handleCreateNewSessionForAction(
        {
          id: "bmad-prd",
          label: "Create PRD",
          prompt: "/bmad-prd",
          displayPrompt: "Create PRD Document",
        },
        false
      )
    })

    expect(mockApiPost).toHaveBeenCalledWith(
      expect.stringContaining("/sessions?createNew=true"),
      expect.objectContaining({ phase: expect.any(String) })
    )
    expect(result.current.activeSessionId).toBe("sess-action-new")
  })

  it("closes the preview window (resets activeArtifactTitle to null) when the open artifact is deleted", async () => {
    mockApiDelete.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ message: "Artifact deleted successfully" }),
    })

    const wrapper = createWrapper()
    const { result } = renderHook(() => useWorkspace("proj-123"), { wrapper })

    await waitFor(() => expect(result.current.isLoading).toBe(false))

    act(() => {
      result.current.setActiveArtifactTitle("PRD")
    })
    expect(result.current.activeArtifactTitle).toBe("PRD")

    await act(async () => {
      await result.current.deleteArtifactMutation.mutateAsync("art-prd")
    })

    expect(result.current.activeArtifactTitle).toBeNull()
  })

  it("closes the preview window (resets activeArtifactTitle to null) when an open unindexed file path is deleted", async () => {
    mockApiDelete.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ message: "Artifact deleted successfully" }),
    })

    const wrapper = createWrapper()
    const { result } = renderHook(() => useWorkspace("proj-123"), { wrapper })

    await waitFor(() => expect(result.current.isLoading).toBe(false))

    act(() => {
      result.current.setActiveArtifactTitle("src/utils/helper.ts")
    })
    expect(result.current.activeArtifactTitle).toBe("src/utils/helper.ts")

    await act(async () => {
      await result.current.deleteArtifactMutation.mutateAsync("src/utils/helper.ts")
    })

    expect(result.current.activeArtifactTitle).toBeNull()
  })
})


