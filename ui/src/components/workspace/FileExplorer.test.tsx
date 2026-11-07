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
import React from "react"
import { render, screen, fireEvent, cleanup, waitFor, act, within } from "@testing-library/react"
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import "@testing-library/jest-dom/vitest"
import * as matchers from "@testing-library/jest-dom/matchers"
expect.extend(matchers as any)
import FileExplorer from "./FileExplorer"
import { api } from "@/lib/api"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { Artifact } from "@/lib/types"

vi.mock("next/navigation", () => ({
  useParams: () => ({ id: "proj-123" }),
}))

vi.mock("@/lib/api", () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
    delete: vi.fn(),
  },
}))

function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  })
}

describe("FileExplorer Component", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    cleanup()
  })

  const mockArtifacts: Artifact[] = [
    {
      id: "art-1",
      projectId: "proj-123",
      title: "PRD.md",
      path: "docs/specs/prd/PRD.md",
      phase: "PLANNING",
      status: "In Progress",
      artifactType: "PRD",
      createdAt: "2026-08-14T10:00:00Z",
      updatedAt: "2026-08-14T10:00:00Z",
    },
    {
      id: "art-2",
      projectId: "proj-123",
      title: "Architecture.md",
      path: "docs/specs/arch/Architecture.md",
      phase: "SOLUTIONING",
      status: "Complete",
      artifactType: "ARCHITECTURE",
      createdAt: "2026-08-14T10:00:00Z",
      updatedAt: "2026-08-14T10:00:00Z",
    },
  ]

  it("filters liveArtifacts by activePhase in default view", async () => {
    const mockRootResponse = {
      name: "proj-123",
      path: "",
      type: "folder",
      hasChildren: true,
      children: [],
    }

    vi.mocked(api.get).mockResolvedValueOnce({
      ok: true,
      json: async () => mockRootResponse,
    } as any)

    const queryClient = createTestQueryClient()
    render(
      <QueryClientProvider client={queryClient}>
        <FileExplorer
          activePhase="PLANNING"
          liveArtifacts={mockArtifacts}
        />
      </QueryClientProvider>
    )

    // Should display only Planning phase artifact (PRD.md), not Solutioning (Architecture.md)
    expect(screen.getByText("Planning Files")).toBeDefined()
    expect(screen.getByText("PRD.md")).toBeDefined()
    expect(screen.queryByText("Architecture.md")).toBeNull()

    // Status dot for In Progress should be present
    expect(screen.getByTitle("Status: In Progress")).toBeDefined()
  })

  it("displays <Phase name> Files when unchecked and All Files when checked", async () => {
    const mockRootResponse = {
      name: "proj-123",
      path: "",
      type: "folder",
      hasChildren: true,
      children: [
        { name: "spec.md", path: "spec.md", type: "file" },
      ],
    }

    vi.mocked(api.get).mockResolvedValueOnce({
      ok: true,
      json: async () => mockRootResponse,
    } as any)

    const queryClient = createTestQueryClient()
    const { rerender } = render(
      <QueryClientProvider client={queryClient}>
        <FileExplorer
          activePhase="PLANNING"
          liveArtifacts={mockArtifacts}
          showAllFiles={false}
        />
      </QueryClientProvider>
    )

    // Unchecked: should display "Planning Files"
    expect(screen.getByText("Planning Files")).toBeDefined()

    // Pass showAllFiles={true}
    rerender(
      <QueryClientProvider client={queryClient}>
        <FileExplorer
          activePhase="PLANNING"
          liveArtifacts={mockArtifacts}
          showAllFiles={true}
        />
      </QueryClientProvider>
    )

    // Checked: should display "All Files"
    expect(screen.getByText("All Files")).toBeDefined()
    expect(screen.queryByText("Planning Files")).toBeNull()
  })

  it("switches to tree view and displays status dots for tracked artifacts", async () => {
    const mockRootResponse = {
      name: "proj-123",
      path: "",
      type: "folder",
      hasChildren: true,
      children: [
        {
          name: "docs",
          path: "docs",
          type: "folder",
          hasChildren: true,
          children: [
            {
              name: "specs",
              path: "docs/specs",
              type: "folder",
              hasChildren: true,
              children: [
                {
                  name: "prd",
                  path: "docs/specs/prd",
                  type: "folder",
                  hasChildren: true,
                  children: [
                    { name: "PRD.md", path: "docs/specs/prd/PRD.md", type: "file" },
                  ],
                },
              ],
            },
          ],
        },
        { name: "untracked.ts", path: "untracked.ts", type: "file" },
      ],
    }

    vi.mocked(api.get).mockResolvedValueOnce({
      ok: true,
      json: async () => mockRootResponse,
    } as any)

    const queryClient = createTestQueryClient()
    render(
      <QueryClientProvider client={queryClient}>
        <FileExplorer
          activePhase="PLANNING"
          liveArtifacts={mockArtifacts}
          selectedFilePath="docs/specs/prd/PRD.md"
          showAllFiles={true}
        />
      </QueryClientProvider>
    )

    await waitFor(() => {
      expect(screen.getByRole("tree")).toBeDefined()
    })

    expect(screen.getByText("untracked.ts")).toBeDefined()
  })

  it("triggers dual refresh of files and artifacts when refresh button is clicked", async () => {
    const mockRootResponse = {
      name: "proj-123",
      path: "",
      type: "folder",
      hasChildren: true,
      children: [],
    }

    vi.mocked(api.get).mockResolvedValue({
      ok: true,
      json: async () => mockRootResponse,
    } as any)

    const onRefreshArtifacts = vi.fn().mockResolvedValue(undefined)
    const queryClient = createTestQueryClient()
    render(
      <QueryClientProvider client={queryClient}>
        <FileExplorer
          activePhase="PLANNING"
          liveArtifacts={mockArtifacts}
          onRefreshArtifacts={onRefreshArtifacts}
        />
      </QueryClientProvider>
    )

    const refreshBtn = screen.getByTitle("Refresh files")
    await waitFor(() => {
      expect(refreshBtn).not.toBeDisabled()
    })
    fireEvent.click(refreshBtn)

    await waitFor(() => {
      expect(onRefreshArtifacts).toHaveBeenCalled()
    })
  })

  it("refreshes and opens matching folder via imperative handle refreshFolderForPath", async () => {
    const mockRootResponse = {
      name: "proj-123",
      path: "",
      type: "folder",
      hasChildren: true,
      children: [
        { name: "docs", path: "docs", type: "folder" },
        { name: "AGENTS.md", path: "AGENTS.md", type: "file" },
      ],
    }

    const mockDocsResponse = {
      name: "docs",
      path: "docs",
      type: "folder",
      children: [
        { name: "spec", path: "docs/spec", type: "folder" },
        { name: "new_file.md", path: "docs/new_file.md", type: "file" },
      ],
    }

    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url.includes("path=docs")) {
        return {
          ok: true,
          status: 200,
          json: async () => mockDocsResponse,
        } as any
      }
      return {
        ok: true,
        status: 200,
        json: async () => mockRootResponse,
      } as any
    })

    const explorerRef = React.createRef<any>()
    const queryClient = createTestQueryClient()

    render(
      <QueryClientProvider client={queryClient}>
        <FileExplorer
          ref={explorerRef}
          activePhase="PLANNING"
          liveArtifacts={mockArtifacts}
          showAllFiles={true}
        />
      </QueryClientProvider>
    )

    await waitFor(() => {
      expect(screen.getByText("docs")).toBeDefined()
    })

    // Call refreshFolderForPath with a path under docs
    await act(async () => {
      await explorerRef.current.refreshFolderForPath(
        "docs/spec/brainstorming/brainstorm.md"
      )
    })

    // Expect the subfolder API to have been called for 'docs'
    expect(api.get).toHaveBeenCalledWith(
      expect.stringContaining("path=docs")
    )

    // And 'docs' folder children should now be displayed
    await waitFor(() => {
      expect(screen.getByText("new_file.md")).toBeDefined()
    })
  })

  it("displays and allows selection of hidden files in Show All tree view", async () => {
    const mockRootResponse = {
      name: "proj-123",
      path: "",
      type: "folder",
      hasChildren: true,
      children: [
        { name: ".github", path: ".github", type: "folder", hasChildren: true },
        { name: ".gitignore", path: ".gitignore", type: "file" },
        { name: ".env", path: ".env", type: "file" },
        { name: "README.md", path: "README.md", type: "file" },
      ],
    }

    vi.mocked(api.get).mockResolvedValueOnce({
      ok: true,
      json: async () => mockRootResponse,
    } as any)

    const onFileSelect = vi.fn()
    const queryClient = createTestQueryClient()

    render(
      <QueryClientProvider client={queryClient}>
        <FileExplorer
          activePhase="PLANNING"
          liveArtifacts={mockArtifacts}
          onFileSelect={onFileSelect}
          showAllFiles={true}
        />
      </QueryClientProvider>
    )

    await waitFor(() => {
      expect(screen.getByRole("tree")).toBeDefined()
    })

    // Verify hidden files and folders are visible
    expect(screen.getByText(".github")).toBeDefined()
    expect(screen.getByText(".gitignore")).toBeDefined()
    expect(screen.getByText(".env")).toBeDefined()
    expect(screen.getByText("README.md")).toBeDefined()

    // Click on a hidden file to select it
    fireEvent.click(screen.getByText(".gitignore"))
    expect(onFileSelect).toHaveBeenCalledWith({
      name: ".gitignore",
      path: ".gitignore",
    })
  })

  it("handles master select all / deselect all checkbox and renders selection action bar", async () => {
    const mockRootResponse = {
      name: "proj-123",
      path: "",
      type: "folder",
      hasChildren: true,
      children: [],
    }

    vi.mocked(api.get).mockResolvedValueOnce({
      ok: true,
      json: async () => mockRootResponse,
    } as any)

    const onStageForSync = vi.fn()
    const queryClient = createTestQueryClient()

    render(
      <QueryClientProvider client={queryClient}>
        <FileExplorer
          activePhase="PLANNING"
          liveArtifacts={mockArtifacts}
          onStageForSync={onStageForSync}
        />
      </QueryClientProvider>
    )

    // Master checkbox is present in the header
    const masterCheckbox = screen.getByLabelText("Select all files") as HTMLInputElement
    expect(masterCheckbox.checked).toBe(false)

    // Click master checkbox to select all visible phase files (PRD.md)
    fireEvent.click(masterCheckbox)

    // Selection action bar should appear
    expect(screen.getByText("1 file selected")).toBeDefined()
    expect(screen.getByText("Stage for Sync")).toBeDefined()

    // Click "Stage for Sync"
    fireEvent.click(screen.getByText("Stage for Sync"))
    expect(onStageForSync).toHaveBeenCalledWith(["docs/specs/prd/PRD.md"])

    // Clear selection
    const clearBtn = screen.getByLabelText("Clear selection")
    fireEvent.click(clearBtn)

    expect(screen.queryByText("1 file selected")).toBeNull()
  })

  it("handles batch delete confirmation modal from selection bar", async () => {
    const mockRootResponse = {
      name: "proj-123",
      path: "",
      type: "folder",
      hasChildren: true,
      children: [],
    }

    vi.mocked(api.get).mockResolvedValueOnce({
      ok: true,
      json: async () => mockRootResponse,
    } as any)

    const onBatchDelete = vi.fn().mockResolvedValue(undefined)
    const queryClient = createTestQueryClient()

    render(
      <QueryClientProvider client={queryClient}>
        <FileExplorer
          activePhase="PLANNING"
          liveArtifacts={mockArtifacts}
          onBatchDelete={onBatchDelete}
        />
      </QueryClientProvider>
    )

    // Master select all
    const masterCheckbox = screen.getByLabelText("Select all files")
    fireEvent.click(masterCheckbox)

    // Click batch delete icon in selection bar
    const deleteBtn = screen.getByLabelText("Delete selected files")
    fireEvent.click(deleteBtn)

    // Batch delete confirmation modal should open
    expect(screen.getByRole("dialog")).toBeDefined()
    expect(screen.getByText("Delete 1 File")).toBeDefined()

    // Confirm deletion
    const confirmBtn = screen.getByRole("button", { name: /Delete \(1\)/i })
    fireEvent.click(confirmBtn)

    await waitFor(() => {
      expect(onBatchDelete).toHaveBeenCalledWith([
        expect.objectContaining({
          name: "PRD.md",
          path: "docs/specs/prd/PRD.md",
          artifactId: "art-1",
        }),
      ])
    })
  })

  it("does not render any checkboxes in Show All tree view", async () => {
    const mockRootResponse = {
      name: "proj-123",
      path: "",
      type: "folder",
      hasChildren: true,
      children: [
        { name: "file1.ts", path: "file1.ts", type: "file" },
        { name: "file2.ts", path: "file2.ts", type: "file" },
      ],
    }

    vi.mocked(api.get).mockResolvedValueOnce({
      ok: true,
      json: async () => mockRootResponse,
    } as any)

    const queryClient = createTestQueryClient()

    render(
      <QueryClientProvider client={queryClient}>
        <FileExplorer
          activePhase="PLANNING"
          liveArtifacts={mockArtifacts}
          showAllFiles={true}
        />
      </QueryClientProvider>
    )

    await waitFor(() => {
      expect(screen.getByRole("tree")).toBeDefined()
    })

    // No master checkbox in header
    expect(screen.queryByLabelText("Select all files")).toBeNull()

    // No item checkboxes in the tree items
    expect(screen.queryByRole("checkbox", { name: /Select file1\.ts/i })).toBeNull()
    expect(screen.queryByRole("checkbox", { name: /Select file2\.ts/i })).toBeNull()
  })

  it("does not have left border on selected item in tree view or phase view", async () => {
    const mockRootResponse = {
      name: "proj-123",
      path: "",
      type: "folder",
      hasChildren: true,
      children: [
        { name: "PRD.md", path: "docs/specs/prd/PRD.md", type: "file" },
      ],
    }

    vi.mocked(api.get).mockResolvedValueOnce({
      ok: true,
      json: async () => mockRootResponse,
    } as any)

    const queryClient = createTestQueryClient()

    const { rerender } = render(
      <QueryClientProvider client={queryClient}>
        <FileExplorer
          activePhase="PLANNING"
          liveArtifacts={mockArtifacts}
          selectedFilePath="docs/specs/prd/PRD.md"
          showAllFiles={false}
        />
      </QueryClientProvider>
    )

    // Phase view selected item
    const prdItem = screen.getByText("PRD.md").closest('[role="listitem"]')
    expect(prdItem?.className).not.toContain("border-l")
    expect(prdItem?.className).toContain("bg-primary/10")

    // Rerender with showAllFiles={true}
    rerender(
      <QueryClientProvider client={queryClient}>
        <FileExplorer
          activePhase="PLANNING"
          liveArtifacts={mockArtifacts}
          selectedFilePath="docs/specs/prd/PRD.md"
          showAllFiles={true}
        />
      </QueryClientProvider>
    )

    await waitFor(() => {
      expect(screen.getByRole("tree")).toBeDefined()
    })

    const treeItem = screen.getByText("PRD.md").closest('[role="treeitem"]')
    expect(treeItem?.className).not.toContain("border-l")
    expect(treeItem?.className).toContain("bg-primary/10")
  })

  it("renders + (Upload to Root) button in header in All Files view and triggers file upload", async () => {
    const mockRootResponse = {
      name: "proj-123",
      path: "",
      type: "folder",
      hasChildren: true,
      children: [
        { name: "README.md", path: "README.md", type: "file" },
      ],
    }

    vi.mocked(api.get).mockResolvedValue({
      ok: true,
      json: async () => mockRootResponse,
    } as any)

    vi.mocked(api.post).mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({ status: "SUCCESS", message: "Files uploaded successfully" }),
    } as any)

    const queryClient = createTestQueryClient()
    render(
      <QueryClientProvider client={queryClient}>
        <FileExplorer
          activePhase="PLANNING"
          liveArtifacts={mockArtifacts}
          showAllFiles={true}
        />
      </QueryClientProvider>
    )

    await waitFor(() => {
      expect(screen.getByRole("tree")).toBeDefined()
    })

    const rootUploadBtn = screen.getByLabelText("Upload to root")
    expect(rootUploadBtn).toBeDefined()
    expect(screen.getByTitle("Upload to root")).toBeDefined()

    // Trigger root upload
    fireEvent.click(rootUploadBtn)

    const fileInput = screen.getByLabelText("Upload files") as HTMLInputElement
    const file = new File(["test file content"], "uploaded_root.txt", { type: "text/plain" })

    await act(async () => {
      fireEvent.change(fileInput, { target: { files: [file] } })
    })

    expect(api.post).toHaveBeenCalledWith(
      "/api/v1/projects/proj-123/files/upload",
      expect.any(FormData)
    )
  })

  it("renders folder hover action buttons (+, Refresh, Delete) on folder nodes", async () => {
    const mockRootResponse = {
      name: "proj-123",
      path: "",
      type: "folder",
      hasChildren: true,
      children: [
        { name: "src", path: "src", type: "folder", hasChildren: true, children: [] },
      ],
    }

    vi.mocked(api.get).mockResolvedValueOnce({
      ok: true,
      json: async () => mockRootResponse,
    } as any)

    const queryClient = createTestQueryClient()
    render(
      <QueryClientProvider client={queryClient}>
        <FileExplorer
          activePhase="PLANNING"
          liveArtifacts={mockArtifacts}
          showAllFiles={true}
        />
      </QueryClientProvider>
    )

    await waitFor(() => {
      expect(screen.getByRole("tree")).toBeDefined()
    })

    // Folder node 'src' should have the 3 action buttons
    expect(screen.getByLabelText("Upload to src")).toBeDefined()
    expect(screen.getByLabelText("Refresh src")).toBeDefined()
    expect(screen.getByLabelText("Delete src")).toBeDefined()
  })

  it("uploads files to specific folder when folder + button is clicked", async () => {
    const mockRootResponse = {
      name: "proj-123",
      path: "",
      type: "folder",
      hasChildren: true,
      children: [
        { name: "assets", path: "src/assets", type: "folder", hasChildren: true, children: [] },
      ],
    }

    const mockSubfolderResponse = {
      name: "assets",
      path: "src/assets",
      type: "folder",
      children: [
        { name: "logo.png", path: "src/assets/logo.png", type: "file" },
      ],
    }

    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url.includes("path=src%2Fassets") || url.includes("path=src/assets")) {
        return {
          ok: true,
          status: 200,
          json: async () => mockSubfolderResponse,
        } as any
      }
      return {
        ok: true,
        status: 200,
        json: async () => mockRootResponse,
      } as any
    })

    vi.mocked(api.post).mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({ status: "SUCCESS", message: "Files uploaded successfully" }),
    } as any)

    const queryClient = createTestQueryClient()
    render(
      <QueryClientProvider client={queryClient}>
        <FileExplorer
          activePhase="PLANNING"
          liveArtifacts={mockArtifacts}
          showAllFiles={true}
        />
      </QueryClientProvider>
    )

    await waitFor(() => {
      expect(screen.getByRole("tree")).toBeDefined()
    })

    const folderUploadBtn = screen.getByLabelText("Upload to assets")
    fireEvent.click(folderUploadBtn)

    const fileInput = screen.getByLabelText("Upload files") as HTMLInputElement
    const file = new File(["binary logo"], "logo.png", { type: "image/png" })

    await act(async () => {
      fireEvent.change(fileInput, { target: { files: [file] } })
    })

    expect(api.post).toHaveBeenCalledWith(
      "/api/v1/projects/proj-123/files/upload?targetPath=src%2Fassets",
      expect.any(FormData)
    )
  })

  it("opens folder delete confirmation dialog and deletes folder on confirm", async () => {
    const mockRootResponse = {
      name: "proj-123",
      path: "",
      type: "folder",
      hasChildren: true,
      children: [
        { name: "temp", path: "src/temp", type: "folder", hasChildren: true, children: [] },
      ],
    }

    vi.mocked(api.get).mockResolvedValue({
      ok: true,
      json: async () => mockRootResponse,
    } as any)

    vi.mocked(api.delete).mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({ status: "SUCCESS", message: "Folder deleted successfully" }),
    } as any)

    const queryClient = createTestQueryClient()
    render(
      <QueryClientProvider client={queryClient}>
        <FileExplorer
          activePhase="PLANNING"
          liveArtifacts={mockArtifacts}
          showAllFiles={true}
        />
      </QueryClientProvider>
    )

    await waitFor(() => {
      expect(screen.getByRole("tree")).toBeDefined()
    })

    const deleteBtn = screen.getByLabelText("Delete temp")
    fireEvent.click(deleteBtn)

    // Delete modal should be visible
    expect(screen.getByRole("dialog")).toBeDefined()
    expect(screen.getByRole("heading", { name: "Delete Folder" })).toBeDefined()
    expect(screen.getByText(/Are you sure you want to delete the folder/)).toBeDefined()
    expect(screen.getByText(/"temp"/)).toBeDefined()

    // Confirm deletion
    const confirmDeleteBtn = screen.getByRole("button", { name: "Delete Folder" })
    await act(async () => {
      fireEvent.click(confirmDeleteBtn)
    })

    expect(api.delete).toHaveBeenCalledWith(
      "/api/v1/projects/proj-123/folders?path=src%2Ftemp"
    )

    // Modal should close
    await waitFor(() => {
      expect(screen.queryByText(/Are you sure you want to delete the folder/)).toBeNull()
    })
  })

  it("handles cancel action in folder delete confirmation dialog", async () => {
    const mockRootResponse = {
      name: "proj-123",
      path: "",
      type: "folder",
      hasChildren: true,
      children: [
        { name: "temp", path: "src/temp", type: "folder", hasChildren: true, children: [] },
      ],
    }

    vi.mocked(api.get).mockResolvedValueOnce({
      ok: true,
      json: async () => mockRootResponse,
    } as any)

    const queryClient = createTestQueryClient()
    render(
      <QueryClientProvider client={queryClient}>
        <FileExplorer
          activePhase="PLANNING"
          liveArtifacts={mockArtifacts}
          showAllFiles={true}
        />
      </QueryClientProvider>
    )

    await waitFor(() => {
      expect(screen.getByRole("tree")).toBeDefined()
    })

    const deleteBtn = screen.getByLabelText("Delete temp")
    fireEvent.click(deleteBtn)

    expect(screen.getByRole("dialog")).toBeDefined()

    const cancelBtn = screen.getByRole("button", { name: "Cancel" })
    fireEvent.click(cancelBtn)

    expect(screen.queryByRole("dialog")).toBeNull()
    expect(api.delete).not.toHaveBeenCalled()
  })

  it("displays delete icon on file nodes in tree view and triggers delete file modal and API", async () => {
    const mockRootResponse = {
      name: "proj-123",
      path: "",
      type: "folder",
      hasChildren: true,
      children: [
        { name: "helper.ts", path: "src/utils/helper.ts", type: "file" },
      ],
    }

    vi.mocked(api.get).mockResolvedValueOnce({
      ok: true,
      json: async () => mockRootResponse,
    } as any)

    vi.mocked(api.delete).mockResolvedValueOnce({
      ok: true,
      json: async () => ({ message: "Artifact deleted successfully" }),
    } as any)

    const queryClient = createTestQueryClient()
    render(
      <QueryClientProvider client={queryClient}>
        <FileExplorer
          activePhase="PLANNING"
          liveArtifacts={mockArtifacts}
          showAllFiles={true}
        />
      </QueryClientProvider>
    )

    await waitFor(() => {
      expect(screen.getByRole("tree")).toBeDefined()
    })

    const deleteBtn = screen.getByLabelText("Delete helper.ts")
    expect(deleteBtn).toBeDefined()
    fireEvent.click(deleteBtn)

    // Verify modal appears
    const dialog = screen.getByRole("dialog")
    expect(within(dialog).getByRole("heading", { name: "Delete File" })).toBeDefined()
    expect(within(dialog).getByText(/Are you sure you want to delete the file/)).toBeDefined()

    // Click Confirm Delete
    const confirmDeleteBtn = within(dialog).getByRole("button", { name: "Delete File" })
    fireEvent.click(confirmDeleteBtn)

    await waitFor(() => {
      expect(api.delete).toHaveBeenCalledWith(
        expect.stringContaining("artifacts/src%2Futils%2Fhelper.ts")
      )
    })

    // Modal should close
    await waitFor(() => {
      expect(screen.queryByText("Delete File")).toBeNull()
    })
  })

  it("handles cancel action in file delete confirmation dialog", async () => {
    const mockRootResponse = {
      name: "proj-123",
      path: "",
      type: "folder",
      hasChildren: true,
      children: [
        { name: "test.txt", path: "test.txt", type: "file" },
      ],
    }

    vi.mocked(api.get).mockResolvedValueOnce({
      ok: true,
      json: async () => mockRootResponse,
    } as any)

    const queryClient = createTestQueryClient()
    render(
      <QueryClientProvider client={queryClient}>
        <FileExplorer
          activePhase="PLANNING"
          liveArtifacts={mockArtifacts}
          showAllFiles={true}
        />
      </QueryClientProvider>
    )

    await waitFor(() => {
      expect(screen.getByRole("tree")).toBeDefined()
    })

    const deleteBtn = screen.getByLabelText("Delete test.txt")
    fireEvent.click(deleteBtn)

    expect(screen.getByRole("dialog")).toBeDefined()

    const cancelBtn = screen.getByRole("button", { name: "Cancel" })
    fireEvent.click(cancelBtn)

    expect(screen.queryByRole("dialog")).toBeNull()
    expect(api.delete).not.toHaveBeenCalled()
  })

  it("closes the preview window (notifies onFileSelect) when deleting the currently open file", async () => {
    const mockRootResponse = {
      name: "proj-123",
      path: "",
      type: "folder",
      hasChildren: true,
      children: [
        { name: "open-file.ts", path: "src/open-file.ts", type: "file" },
      ],
    }

    vi.mocked(api.get).mockResolvedValueOnce({
      ok: true,
      json: async () => mockRootResponse,
    } as any)

    vi.mocked(api.delete).mockResolvedValueOnce({
      ok: true,
      json: async () => ({ message: "Artifact deleted successfully" }),
    } as any)

    const onFileSelect = vi.fn()
    const queryClient = createTestQueryClient()
    render(
      <QueryClientProvider client={queryClient}>
        <FileExplorer
          activePhase="PLANNING"
          liveArtifacts={mockArtifacts}
          showAllFiles={true}
          selectedFilePath="src/open-file.ts"
          onFileSelect={onFileSelect}
        />
      </QueryClientProvider>
    )

    await waitFor(() => {
      expect(screen.getByRole("tree")).toBeDefined()
    })

    const deleteBtn = screen.getByLabelText("Delete open-file.ts")
    fireEvent.click(deleteBtn)

    const dialog = screen.getByRole("dialog")
    const confirmDeleteBtn = within(dialog).getByRole("button", { name: "Delete File" })
    fireEvent.click(confirmDeleteBtn)

    await waitFor(() => {
      expect(onFileSelect).toHaveBeenCalledWith({ name: "", path: "" })
    })
  })
})




