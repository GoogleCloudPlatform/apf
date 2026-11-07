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
import { render, screen, fireEvent, cleanup, waitFor } from "@testing-library/react"
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import CanvasPane from "./CanvasPane"
import { api } from "@/lib/api"
import { toast } from "sonner"
import { Artifact } from "@/lib/types"

afterEach(() => {
  cleanup()
  vi.clearAllMocks()
})

vi.mock("next/navigation", () => ({
  useRouter: () => ({
    push: vi.fn(),
  }),
}))

vi.mock("@/stores/auth", () => ({
  useAuthStore: () => ({
    email: "test@google.com",
    displayName: "Test User",
  }),
}))

vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
  },
}))

vi.mock("./VSCodeArtifactViewer", () => ({
  VSCodeArtifactViewer: ({ content, onContentChange, onSave, isSaving }: any) => (
    <div data-testid="vscode-artifact-viewer">
      <textarea
        data-testid="viewer-editor"
        value={content}
        onChange={(e) => onContentChange?.(e.target.value)}
      />
      <button
        data-testid="viewer-save-btn"
        disabled={isSaving}
        onClick={() => onSave?.(content)}
      >
        {isSaving ? "Saving..." : "Save"}
      </button>
    </div>
  ),
}))

vi.mock("./MermaidDiagram", () => ({
  MermaidDiagram: () => <div data-testid="mermaid" />,
}))

vi.mock("@/components/shared/AlertModal", () => ({
  default: () => <div data-testid="alert-modal" />,
}))

vi.mock("@/components/project/GitSyncModal", () => ({
  GitSyncModal: () => <div data-testid="git-sync-modal" />,
}))

vi.mock("@/components/project/GitStagingModal", () => ({
  GitStagingModal: () => <div data-testid="git-staging-modal" />,
}))

vi.mock("@/components/ui/dropdown-menu", () => ({
  DropdownMenu: ({ children }: any) => <div data-testid="dropdown-menu">{children}</div>,
  DropdownMenuTrigger: ({ children, disabled, title, className }: any) => (
    <button data-testid="dropdown-trigger" disabled={disabled} title={title} className={className}>
      {children}
    </button>
  ),
  DropdownMenuContent: ({ children }: any) => <div data-testid="dropdown-content">{children}</div>,
  DropdownMenuItem: ({ children, onClick, disabled, className, title }: any) => (
    <button data-testid="dropdown-item" onClick={onClick} disabled={disabled} className={className} title={title}>
      {children}
    </button>
  ),
}))

describe("CanvasPane", () => {
  let queryClient: QueryClient

  const mockArtifacts: Artifact[] = [
    {
      id: "art-1",
      title: "PRD",
      path: "docs/spec/prd.md",
      phase: "PLANNING",
      status: "In Progress",
      owner: "Agent",
      content: "",
      artifactType: "prd",
    },
  ]

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    })
  })

  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  it("renders artifact content fetched from the server", async () => {
    vi.spyOn(api, "get").mockImplementation(async (url: string) => {
      if (url.includes("/credentials")) {
        return new Response(JSON.stringify({ isReadOnly: false }), { status: 200 })
      }
      if (url.includes("/content")) {
        return new Response("# Product Requirements Document\n\nInitial version.", { status: 200 })
      }
      return new Response("", { status: 404 })
    })

    render(
      <QueryClientProvider client={queryClient}>
        <CanvasPane
          projectId="proj-1"
          activeArtifactTitle="PRD"
          liveArtifacts={mockArtifacts}
          onClose={vi.fn()}
        />
      </QueryClientProvider>
    )

    await waitFor(() => {
      const textarea = screen.getByTestId("viewer-editor") as HTMLTextAreaElement
      expect(textarea.value).toBe("# Product Requirements Document\n\nInitial version.")
    })
  })

  it("sends modified content to server when Save button is clicked", async () => {
    vi.spyOn(api, "get").mockImplementation(async (url: string) => {
      if (url.includes("/credentials")) {
        return new Response(JSON.stringify({ isReadOnly: false }), { status: 200 })
      }
      if (url.includes("/content")) {
        return new Response("# Initial Content", { status: 200 })
      }
      return new Response("", { status: 404 })
    })

    const putSpy = vi.spyOn(api, "put").mockResolvedValue(
      new Response(JSON.stringify({ message: "File saved successfully", status: "SUCCESS" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    )

    render(
      <QueryClientProvider client={queryClient}>
        <CanvasPane
          projectId="proj-1"
          activeArtifactTitle="PRD"
          liveArtifacts={mockArtifacts}
          onClose={vi.fn()}
        />
      </QueryClientProvider>
    )

    await waitFor(() => {
      expect(screen.getByTestId("viewer-editor")).toBeDefined()
    })

    // Modify content
    const textarea = screen.getByTestId("viewer-editor")
    fireEvent.change(textarea, { target: { value: "# Updated Content from User" } })

    // Click Save button
    const saveBtn = screen.getByTestId("viewer-save-btn")
    fireEvent.click(saveBtn)

    await waitFor(() => {
      expect(putSpy).toHaveBeenCalledWith(
        "/api/v1/projects/proj-1/artifacts/art-1/content",
        "# Updated Content from User"
      )
    })

    await waitFor(() => {
      expect(toast.success).toHaveBeenCalledWith("File saved successfully!")
    })
  })

  it("displays error toast when saving fails", async () => {
    vi.spyOn(api, "get").mockImplementation(async (url: string) => {
      if (url.includes("/credentials")) {
        return new Response(JSON.stringify({ isReadOnly: false }), { status: 200 })
      }
      if (url.includes("/content")) {
        return new Response("# Initial Content", { status: 200 })
      }
      return new Response("", { status: 404 })
    })

    vi.spyOn(api, "put").mockResolvedValue(
      new Response(JSON.stringify({ error: "Storage quota exceeded" }), {
        status: 500,
        headers: { "Content-Type": "application/json" },
      })
    )

    render(
      <QueryClientProvider client={queryClient}>
        <CanvasPane
          projectId="proj-1"
          activeArtifactTitle="PRD"
          liveArtifacts={mockArtifacts}
          onClose={vi.fn()}
        />
      </QueryClientProvider>
    )

    await waitFor(() => {
      expect(screen.getByTestId("viewer-editor")).toBeDefined()
    })

    // Click Save button
    const saveBtn = screen.getByTestId("viewer-save-btn")
    fireEvent.click(saveBtn)

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith("Storage quota exceeded")
    })
  })

  it("renders empty state when no active artifact is selected", () => {
    render(
      <QueryClientProvider client={queryClient}>
        <CanvasPane
          projectId="proj-1"
          activeArtifactTitle={null}
          liveArtifacts={mockArtifacts}
          onClose={vi.fn()}
        />
      </QueryClientProvider>
    )

    expect(screen.getByText("Select an outcome artifact")).toBeDefined()
  })

  it("opens and previews files from File Explorer that are not indexed in Firestore", async () => {
    vi.spyOn(api, "get").mockImplementation(async (url: string) => {
      if (url.includes("/credentials")) {
        return new Response(JSON.stringify({ isReadOnly: false }), { status: 200 })
      }
      if (url.includes("/content")) {
        return new Response("// Unindexed source code\nexport const foo = 'bar';", { status: 200 })
      }
      return new Response("", { status: 404 })
    })

    render(
      <QueryClientProvider client={queryClient}>
        <CanvasPane
          projectId="proj-1"
          activeArtifactTitle="src/components/Button.tsx"
          liveArtifacts={[]} // Not indexed in Firestore
          onClose={vi.fn()}
        />
      </QueryClientProvider>
    )

    await waitFor(() => {
      const textarea = screen.getByTestId("viewer-editor") as HTMLTextAreaElement
      expect(textarea.value).toBe("// Unindexed source code\nexport const foo = 'bar';")
    })
  })

  it("saves files from File Explorer that are not indexed in Firestore", async () => {
    vi.spyOn(api, "get").mockImplementation(async (url: string) => {
      if (url.includes("/credentials")) {
        return new Response(JSON.stringify({ isReadOnly: false }), { status: 200 })
      }
      if (url.includes("/content")) {
        return new Response("// Initial file content", { status: 200 })
      }
      return new Response("", { status: 404 })
    })

    const putSpy = vi.spyOn(api, "put").mockResolvedValue(
      new Response(JSON.stringify({ message: "File saved successfully", status: "SUCCESS" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    )

    render(
      <QueryClientProvider client={queryClient}>
        <CanvasPane
          projectId="proj-1"
          activeArtifactTitle="src/utils/helper.ts"
          liveArtifacts={[]} // Not indexed in Firestore
          onClose={vi.fn()}
        />
      </QueryClientProvider>
    )

    await waitFor(() => {
      expect(screen.getByTestId("viewer-editor")).toBeDefined()
    })

    const textarea = screen.getByTestId("viewer-editor")
    fireEvent.change(textarea, { target: { value: "// Modified helper code" } })

    const saveBtn = screen.getByTestId("viewer-save-btn")
    fireEvent.click(saveBtn)

    await waitFor(() => {
      expect(putSpy).toHaveBeenCalledWith(
        `/api/v1/projects/proj-1/artifacts/${encodeURIComponent("src/utils/helper.ts")}/content`,
        "// Modified helper code"
      )
    })

    await waitFor(() => {
      expect(toast.success).toHaveBeenCalledWith("File saved successfully!")
    })
  })

  it("allows file editing and saving when in read-only mode", async () => {
    vi.spyOn(api, "get").mockImplementation(async (url: string) => {
      if (url.includes("/credentials")) {
        return new Response(JSON.stringify({ isReadOnly: true }), { status: 200 })
      }
      if (url.includes("/content")) {
        return new Response("# Readonly Mode Content", { status: 200 })
      }
      return new Response("", { status: 404 })
    })

    const putSpy = vi.spyOn(api, "put").mockResolvedValue(
      new Response(JSON.stringify({ message: "File saved successfully", status: "SUCCESS" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    )

    render(
      <QueryClientProvider client={queryClient}>
        <CanvasPane
          projectId="proj-1"
          activeArtifactTitle="PRD"
          liveArtifacts={mockArtifacts}
          isReadOnly={true}
          onClose={vi.fn()}
        />
      </QueryClientProvider>
    )

    await waitFor(() => {
      expect(screen.getByTestId("viewer-editor")).toBeDefined()
    })

    const textarea = screen.getByTestId("viewer-editor")
    fireEvent.change(textarea, { target: { value: "# Readonly Mode Content Edited" } })

    const saveBtn = screen.getByTestId("viewer-save-btn")
    fireEvent.click(saveBtn)

    await waitFor(() => {
      expect(putSpy).toHaveBeenCalledWith(
        "/api/v1/projects/proj-1/artifacts/art-1/content",
        "# Readonly Mode Content Edited"
      )
    })

    await waitFor(() => {
      expect(toast.success).toHaveBeenCalledWith("File saved successfully!")
    })
  })

  it("allows local commit (Approve) when in read-only mode", async () => {
    vi.spyOn(api, "get").mockImplementation(async (url: string) => {
      if (url.includes("/credentials")) {
        return new Response(JSON.stringify({ isReadOnly: true }), { status: 200 })
      }
      if (url.includes("/content")) {
        return new Response("# Spec Content", { status: 200 })
      }
      return new Response("", { status: 404 })
    })

    const postSpy = vi.spyOn(api, "post").mockResolvedValue(
      new Response(
        JSON.stringify({
          status: "SUCCESS",
          message: "Specifications successfully approved and saved locally.",
        }),
        {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }
      )
    )

    render(
      <QueryClientProvider client={queryClient}>
        <CanvasPane
          projectId="proj-1"
          activeArtifactTitle="PRD"
          liveArtifacts={mockArtifacts}
          isReadOnly={true}
          onClose={vi.fn()}
        />
      </QueryClientProvider>
    )

    await waitFor(() => {
      expect(screen.getAllByText("Approve").length).toBeGreaterThan(0)
    })

    // The trigger button should NOT be disabled in read-only mode
    const approveTrigger = screen.getByTitle("Read-only mode (Remote push disabled)")
    expect(approveTrigger.hasAttribute("disabled")).toBe(false)

    // The Push menu item should be disabled in read-only mode
    const pushItem = screen.getByTitle("Pushing changes to remote is prohibited in read-only mode")
    expect(pushItem.hasAttribute("disabled")).toBe(true)

    // Click Approve button item in dropdown
    const items = screen.getAllByTestId("dropdown-item")
    const approveItem = items[0] // Approve item
    expect(approveItem.hasAttribute("disabled")).toBe(false)
    fireEvent.click(approveItem)

    await waitFor(() => {
      expect(postSpy).toHaveBeenCalledWith(
        "/api/v1/projects/proj-1/git/commit",
        {
          filePath: "docs/spec/prd.md",
          artifactId: "art-1",
        }
      )
    })
  })

  it("resolves and renders session artifact by name and storagePath with ImageViewer", async () => {
    const sessionArtifacts: Artifact[] = [
      {
        id: "art-sess-1",
        sessionId: "sess-1",
        name: "architecture_diagram.png",
        storagePath: "/artifacts/proj-1/sess-1/architecture_diagram.png",
        storageType: "local_file",
        mimeType: "image/png",
        size: 2500000,
      },
    ]

    vi.spyOn(api, "get").mockImplementation(async (url: string) => {
      if (url.includes("/credentials")) {
        return new Response(JSON.stringify({ isReadOnly: false }), { status: 200 })
      }
      return new Response("", { status: 404 })
    })

    const testQueryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    })

    const { container } = render(
      <QueryClientProvider client={testQueryClient}>
        <CanvasPane
          projectId="proj-sess"
          activeArtifactTitle="architecture_diagram.png"
          liveArtifacts={sessionArtifacts}
          onClose={vi.fn()}
        />
      </QueryClientProvider>
    )

    await waitFor(() => {
      expect(container.querySelector("h2")?.textContent?.trim()).toBe("architecture_diagram.png")
      expect(container.textContent).toContain("/artifacts/proj-1/sess-1/architecture_diagram.png")
      expect(screen.getByTestId("image-viewer-container")).toBeDefined()
      expect(screen.getByTestId("download-image-btn")).toBeDefined()
      expect(screen.getByTestId("image-size-badge").textContent).toBe("2.4 MB")
      expect(screen.queryByTestId("vscode-artifact-viewer")).toBeNull()
      expect(screen.queryByTestId("dropdown-menu")).toBeNull()
    })
  })

  it("measures image dimensions and updates dimension badge in header", async () => {
    const imageArtifact: Artifact = {
      id: "art-img-1",
      title: "mockup.webp",
      path: "assets/mockup.webp",
      phase: "SOLUTIONING",
      status: "In Progress",
      size: 51200,
      mimeType: "image/webp",
    }

    vi.spyOn(api, "get").mockImplementation(async (url: string) => {
      if (url.includes("/credentials")) {
        return new Response(JSON.stringify({ isReadOnly: false }), { status: 200 })
      }
      return new Response(new Blob(["mock-image-data"], { type: "image/webp" }), { status: 200 })
    })

    const createObjectUrlMock = vi.fn(() => "blob:http://localhost/mock-blob-url")
    const revokeObjectUrlMock = vi.fn()
    ;(globalThis as any).URL.createObjectURL = createObjectUrlMock
    ;(globalThis as any).URL.revokeObjectURL = revokeObjectUrlMock
    if (typeof window !== "undefined" && window.URL) {
      window.URL.createObjectURL = createObjectUrlMock
      window.URL.revokeObjectURL = revokeObjectUrlMock
    }

    render(
      <QueryClientProvider client={queryClient}>
        <CanvasPane
          projectId="proj-1"
          activeArtifactTitle="mockup.webp"
          liveArtifacts={[imageArtifact]}
          onClose={vi.fn()}
        />
      </QueryClientProvider>
    )

    await waitFor(() => {
      expect(screen.getByTestId("image-viewer-img")).toBeDefined()
    })

    const img = screen.getByTestId("image-viewer-img") as HTMLImageElement
    Object.defineProperty(img, "naturalWidth", { value: 1280, configurable: true })
    Object.defineProperty(img, "naturalHeight", { value: 720, configurable: true })
    fireEvent.load(img)

    await waitFor(() => {
      const dimBadge = screen.getByTestId("image-dimensions-badge")
      expect(dimBadge.textContent).toBe("1280 × 720 px")
    })
  })

  it("downloads image when Download button is clicked", async () => {
    const imageArtifact: Artifact = {
      id: "art-img-2",
      title: "system_architecture.svg",
      path: "docs/system_architecture.svg",
      size: 15360,
      mimeType: "image/svg+xml",
    }

    vi.spyOn(api, "get").mockImplementation(async (url: string) => {
      if (url.includes("/credentials")) {
        return new Response(JSON.stringify({ isReadOnly: false }), { status: 200 })
      }
      if (url.includes("/content")) {
        return new Response("<svg></svg>", {
          status: 200,
          headers: { "Content-Type": "image/svg+xml" },
        })
      }
      return new Response("", { status: 404 })
    })

    const createObjectUrlMock = vi.fn(() => "blob:http://localhost/mock-blob-url")
    const revokeObjectUrlMock = vi.fn()
    ;(globalThis as any).URL.createObjectURL = createObjectUrlMock
    ;(globalThis as any).URL.revokeObjectURL = revokeObjectUrlMock
    if (typeof window !== "undefined" && window.URL) {
      window.URL.createObjectURL = createObjectUrlMock
      window.URL.revokeObjectURL = revokeObjectUrlMock
    }

    const testQueryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    })

    render(
      <QueryClientProvider client={testQueryClient}>
        <CanvasPane
          projectId="proj-1"
          activeArtifactTitle="system_architecture.svg"
          liveArtifacts={[imageArtifact]}
          onClose={vi.fn()}
        />
      </QueryClientProvider>
    )

    await waitFor(() => {
      expect(screen.getByTestId("download-image-btn")).toBeDefined()
    })

    const downloadBtn = screen.getByTestId("download-image-btn")
    fireEvent.click(downloadBtn)

    await waitFor(() => {
      expect(toast.success).toHaveBeenCalledWith("Image downloaded successfully!")
      expect(createObjectUrlMock).toHaveBeenCalled()
      expect(revokeObjectUrlMock).toHaveBeenCalledWith("blob:http://localhost/mock-blob-url")
    })
  })

  it("displays error toast if image download fails", async () => {
    const imageArtifact: Artifact = {
      id: "art-img-err",
      title: "failed_diagram.png",
      path: "failed_diagram.png",
      size: 5000,
      mimeType: "image/png",
    }

    vi.spyOn(api, "get").mockImplementation(async (url: string) => {
      if (url.includes("/credentials")) {
        return new Response(JSON.stringify({ isReadOnly: false }), { status: 200 })
      }
      if (url.includes("/content")) {
        return new Response("Not Found", { status: 404 })
      }
      return new Response("", { status: 404 })
    })

    const testQueryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    })

    render(
      <QueryClientProvider client={testQueryClient}>
        <CanvasPane
          projectId="proj-1"
          activeArtifactTitle="failed_diagram.png"
          liveArtifacts={[imageArtifact]}
          onClose={vi.fn()}
        />
      </QueryClientProvider>
    )

    await waitFor(() => {
      expect(screen.getByTestId("download-image-btn")).toBeDefined()
    })

    const downloadBtn = screen.getByTestId("download-image-btn")
    fireEvent.click(downloadBtn)

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith(expect.stringContaining("Failed to download image"))
    })
  })
})

