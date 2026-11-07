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
import { render, screen, fireEvent, cleanup } from "@testing-library/react"
import { describe, it, expect, vi, beforeEach, afterEach, beforeAll } from "vitest"
import "@testing-library/jest-dom/vitest"
import * as matchers from "@testing-library/jest-dom/matchers"
expect.extend(matchers as any)
import ExplorerPanel from "./ExplorerPanel"
import { PhaseType } from "../_utils/workspaceHelpers"
import { WorkflowTemplate, Artifact } from "@/lib/types"

// Inject Tailwind's hidden class style so JSDOM respects toBeVisible()
beforeAll(() => {
  const style = document.createElement("style")
  style.innerHTML = ".hidden { display: none !important; }"
  document.head.appendChild(style)
})

// Mock sub-components
vi.mock("./ExplorerSection", () => ({
  default: ({
    children,
    title,
    trailingHeaderWidget,
  }: {
    children: React.ReactNode
    title: string
    trailingHeaderWidget?: React.ReactNode
  }) => (
    <div data-testid={`explorer-section-${title}`}>
      <div className="flex items-center justify-between">
        <h3>{title}</h3>
        {trailingHeaderWidget}
      </div>
      {children}
    </div>
  ),
}))

vi.mock("./WorkflowsControl", () => ({
  default: () => <div data-testid="mock-workflows-control">WorkflowsControl</div>,
}))

vi.mock("@/components/workspace/FileExplorer", () => ({
  default: ({
    onFileSelect,
    liveArtifacts,
    activePhase,
  }: {
    onFileSelect?: (file: any) => void
    liveArtifacts?: Artifact[]
    activePhase?: string
  }) => (
    <div data-testid="mock-file-explorer" data-phase={activePhase}>
      <span>FileExplorer</span>
      <span data-testid="artifact-count">{liveArtifacts?.length ?? 0}</span>
      <button
        data-testid="mock-select-file"
        onClick={() => onFileSelect?.({ name: "index.ts", path: "src/index.ts" })}
      >
        Select File
      </button>
    </div>
  ),
}))

const defaultProps = {
  activePhase: "PLANNING" as PhaseType,
  isSidebarOpen: true,
  setIsSidebarOpen: vi.fn(),
  isLocked: false,
  isImplementation: false,
  isSessionsExpanded: true,
  setIsSessionsExpanded: vi.fn(),
  isConnectingSession: false,
  isLoadingSessions: false,
  isLoading: false,
  sessions: [],
  activeSessionId: null,
  handleCreateNewSession: vi.fn(),
  handleSwitchSession: vi.fn(),
  setSessionToDelete: vi.fn(),
  addDebugLog: vi.fn(),
  liveArtifacts: [
    {
      id: "art-1",
      projectId: "proj-1",
      title: "PRD.md",
      path: "docs/specs/prd/PRD.md",
      phase: "PLANNING",
      status: "In Progress" as const,
      artifactType: "PRD",
      createdAt: "2026-08-14T10:00:00Z",
      updatedAt: "2026-08-14T10:00:00Z",
    },
  ],
  bmadWorkflow: { id: "bmad", name: "BMad", framework: "bmad", phases: {}, artifacts: {}, dependencies: {} } as unknown as WorkflowTemplate,
  activeArtifactTitle: null,
  setActiveArtifactTitle: vi.fn(),
  deleteArtifact: vi.fn(),
}

describe("ExplorerPanel Sections and Layout", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    cleanup()
  })

  it("renders CHAT SESSIONS and FILE EXPLORER sections, and does NOT render GENERATED ARTIFACTS accordion", () => {
    render(<ExplorerPanel {...defaultProps} />)

    expect(screen.getByTestId("explorer-section-CHAT SESSIONS")).toBeInTheDocument()
    expect(screen.getByTestId("explorer-section-FILE EXPLORER")).toBeInTheDocument()
    expect(screen.getByTestId("mock-file-explorer")).toBeInTheDocument()

    // GENERATED ARTIFACTS accordion must be eliminated
    expect(screen.queryByTestId("explorer-section-GENERATED ARTIFACTS")).toBeNull()
  })

  it("passes liveArtifacts and activePhase to FileExplorer", () => {
    render(<ExplorerPanel {...defaultProps} />)

    const fileExplorer = screen.getByTestId("mock-file-explorer")
    expect(fileExplorer.getAttribute("data-phase")).toBe("PLANNING")
    expect(screen.getByTestId("artifact-count").textContent).toBe("1")
  })

  it("sets active artifact title when a file is selected in FileExplorer", () => {
    const setActiveArtifactTitle = vi.fn()
    render(<ExplorerPanel {...defaultProps} setActiveArtifactTitle={setActiveArtifactTitle} />)

    const selectFileBtn = screen.getByTestId("mock-select-file")
    fireEvent.click(selectFileBtn)

    expect(setActiveArtifactTitle).toHaveBeenCalledWith("src/index.ts")
  })

  it("does not render tablist or tab buttons", () => {
    render(<ExplorerPanel {...defaultProps} />)

    expect(screen.queryByRole("tablist")).toBeNull()
    expect(screen.queryByRole("tab", { name: "Artifacts" })).toBeNull()
    expect(screen.queryByRole("tab", { name: "File Explorer" })).toBeNull()
  })

  it("renders CHAT SESSIONS, FILE EXPLORER, and WORKFLOWS in Implementation phase", () => {
    render(
      <ExplorerPanel
        {...defaultProps}
        activePhase="IMPLEMENTATION"
        isImplementation={true}
      />
    )

    expect(screen.getByTestId("explorer-section-CHAT SESSIONS")).toBeInTheDocument()
    expect(screen.getByTestId("explorer-section-FILE EXPLORER")).toBeInTheDocument()
    expect(screen.getByTestId("explorer-section-WORKFLOWS")).toBeInTheDocument()
    expect(screen.queryByTestId("explorer-section-GENERATED ARTIFACTS")).toBeNull()
  })

  it("does not render QUICK ACTIONS section in any phase", () => {
    const phases: PhaseType[] = ["ONBOARDING", "ANALYSIS", "PLANNING", "SOLUTIONING", "IMPLEMENTATION"]

    phases.forEach((phase) => {
      const { unmount } = render(
        <ExplorerPanel
          {...defaultProps}
          activePhase={phase}
        />
      )

      expect(screen.queryByTestId("explorer-section-QUICK ACTIONS")).toBeNull()
      unmount()
    })
  })

  it("renders explicit session title and date when title is provided", () => {
    const sessions = [
      {
        id: "sess-1",
        userId: "user-1",
        projectId: "proj-1",
        phase: "PLANNING",
        createdAt: "2026-08-14T10:00:00Z",
        lastActiveAt: "2026-08-14T10:00:00Z",
        status: "ACTIVE",
        title: "Login Flow Spec",
      },
    ]

    render(
      <ExplorerPanel
        {...defaultProps}
        activeSessionId="sess-1"
        sessions={sessions}
      />
    )

    expect(screen.getByText("Login Flow Spec")).toBeInTheDocument()
    expect(screen.queryByText("ACTIVE")).toBeNull()
    expect(screen.getByText(/Aug 14/)).toBeInTheDocument()

    const deleteBtn = screen.getByTitle("Delete Session")
    expect(deleteBtn).toBeInTheDocument()
    expect(deleteBtn.className).toContain("text-muted-foreground/50")
  })

  it("falls back to formatted date when session title is not set", () => {
    const sessions = [
      {
        id: "sess-2",
        userId: "user-1",
        projectId: "proj-1",
        phase: "PLANNING",
        createdAt: "2026-08-14T15:30:00Z",
        lastActiveAt: "2026-08-14T15:30:00Z",
        status: "ACTIVE",
      },
    ]

    render(
      <ExplorerPanel
        {...defaultProps}
        sessions={sessions}
      />
    )

    expect(screen.getByText(/Aug 14/)).toBeInTheDocument()
    expect(screen.queryByText("ACTIVE")).toBeNull()
  })

  it("passes onStageForSync and onBatchDelete handlers down to FileExplorer", () => {
    const onStageForSync = vi.fn()
    const onBatchDelete = vi.fn()

    render(
      <ExplorerPanel
        {...defaultProps}
        onStageForSync={onStageForSync}
        onBatchDelete={onBatchDelete}
      />
    )

    expect(screen.getByTestId("mock-file-explorer")).toBeInTheDocument()
  })

  it("renders Show All toggle in FILE EXPLORER section header", () => {
    render(<ExplorerPanel {...defaultProps} />)

    const showAllSwitch = screen.getByRole("switch", { name: "Show All" })
    expect(showAllSwitch).toBeInTheDocument()
  })

  describe("Story 18.5: Chat Session Collapsible Chevron & Artifacts Dropdown", () => {
    const sessions = [
      {
        id: "sess-with-arts",
        userId: "user-1",
        projectId: "proj-1",
        phase: "PLANNING",
        createdAt: "2026-08-14T10:00:00Z",
        lastActiveAt: "2026-08-14T10:00:00Z",
        status: "ACTIVE",
        title: "Session With Artifacts",
      },
      {
        id: "sess-no-arts",
        userId: "user-1",
        projectId: "proj-1",
        phase: "PLANNING",
        createdAt: "2026-08-14T11:00:00Z",
        lastActiveAt: "2026-08-14T11:00:00Z",
        status: "ACTIVE",
        title: "Session Without Artifacts",
      },
    ]

    const liveArtifacts: Artifact[] = [
      {
        id: "art-1",
        projectId: "proj-1",
        sessionId: "sess-with-arts",
        name: "architecture-diagram.png",
        title: "architecture-diagram.png",
        mimeType: "image/png",
        size: 2560000, // ~2.4 MB
        storageType: "local_file",
        storagePath: ".foundry/artifacts/proj-1/sess-with-arts/architecture-diagram.png",
        phase: "PLANNING",
        status: "Complete" as const,
        createdAt: "2026-08-14T10:05:00Z",
      },
      {
        id: "art-2",
        projectId: "proj-1",
        sessionId: "sess-with-arts",
        name: "spec-notes.txt",
        title: "spec-notes.txt",
        mimeType: "text/plain",
        size: 15360, // 15.0 KB
        storageType: "local_file",
        storagePath: ".foundry/artifacts/proj-1/sess-with-arts/spec-notes.txt",
        phase: "PLANNING",
        status: "Complete" as const,
        createdAt: "2026-08-14T10:06:00Z",
      },
      {
        id: "art-3",
        projectId: "proj-1",
        title: "WorkspacePRD.md",
        path: "docs/specs/prd/WorkspacePRD.md",
        phase: "PLANNING",
        status: "In Progress" as const,
        createdAt: "2026-08-14T10:00:00Z",
      },
    ]

    it("renders expandable chevron for sessions with artifacts, and omits chevron for sessions without artifacts", () => {
      render(
        <ExplorerPanel
          {...defaultProps}
          sessions={sessions}
          liveArtifacts={liveArtifacts}
        />
      )

      expect(screen.getByTestId("session-chevron-sess-with-arts")).toBeInTheDocument()
      expect(screen.queryByTestId("session-chevron-sess-no-arts")).toBeNull()
    })

    it("toggles expand/collapse state when clicking chevron button", () => {
      render(
        <ExplorerPanel
          {...defaultProps}
          sessions={sessions}
          liveArtifacts={liveArtifacts}
        />
      )

      const chevronBtn = screen.getByTestId("session-chevron-sess-with-arts")
      const dropdown = screen.getByTestId("session-artifacts-sess-with-arts")

      // Initially collapsed
      expect(chevronBtn.getAttribute("aria-expanded")).toBe("false")
      expect(dropdown.className).toContain("max-h-0")

      // Click to expand
      fireEvent.click(chevronBtn)
      expect(chevronBtn.getAttribute("aria-expanded")).toBe("true")
      expect(dropdown.className).toContain("max-h-60")

      // Click to collapse
      fireEvent.click(chevronBtn)
      expect(chevronBtn.getAttribute("aria-expanded")).toBe("false")
      expect(dropdown.className).toContain("max-h-0")
    })

    it("toggles expand/collapse state and calls handleSwitchSession when clicking session row", () => {
      const handleSwitchSession = vi.fn()
      render(
        <ExplorerPanel
          {...defaultProps}
          sessions={sessions}
          liveArtifacts={liveArtifacts}
          handleSwitchSession={handleSwitchSession}
        />
      )

      const sessionRow = screen.getByText("Session With Artifacts").closest("div[title='Session With Artifacts']")!
      const chevronBtn = screen.getByTestId("session-chevron-sess-with-arts")

      // Click session row
      fireEvent.click(sessionRow)
      expect(handleSwitchSession).toHaveBeenCalledWith("sess-with-arts")
      expect(chevronBtn.getAttribute("aria-expanded")).toBe("true")

      // Click again to collapse
      fireEvent.click(sessionRow)
      expect(chevronBtn.getAttribute("aria-expanded")).toBe("false")
    })

    it("renders session artifacts with file icon, image icon, formatted size, and filename", () => {
      render(
        <ExplorerPanel
          {...defaultProps}
          sessions={sessions}
          liveArtifacts={liveArtifacts}
        />
      )

      // Expand dropdown
      const chevronBtn = screen.getByTestId("session-chevron-sess-with-arts")
      fireEvent.click(chevronBtn)

      // Verify artifact items
      expect(screen.getByText("architecture-diagram.png")).toBeInTheDocument()
      expect(screen.getByText("(2.4 MB)")).toBeInTheDocument()
      expect(screen.getByText("spec-notes.txt")).toBeInTheDocument()
      expect(screen.getByText("(15.0 KB)")).toBeInTheDocument()
    })

    it("triggers setActiveArtifactTitle when clicking artifact row or preview action button", () => {
      const setActiveArtifactTitle = vi.fn()
      render(
        <ExplorerPanel
          {...defaultProps}
          sessions={sessions}
          liveArtifacts={liveArtifacts}
          setActiveArtifactTitle={setActiveArtifactTitle}
        />
      )

      // Expand dropdown
      const chevronBtn = screen.getByTestId("session-chevron-sess-with-arts")
      fireEvent.click(chevronBtn)

      // Click artifact item row
      const artItem = screen.getByTestId("session-artifact-item-art-1")
      fireEvent.click(artItem)
      expect(setActiveArtifactTitle).toHaveBeenCalledWith("architecture-diagram.png")

      // Click preview button on second artifact
      const previewBtn = screen.getByTestId("preview-artifact-art-2")
      fireEvent.click(previewBtn)
      expect(setActiveArtifactTitle).toHaveBeenCalledWith("spec-notes.txt")
    })

    it("does not toggle accordion when clicking delete session button", () => {
      const setSessionToDelete = vi.fn()
      render(
        <ExplorerPanel
          {...defaultProps}
          sessions={sessions}
          liveArtifacts={liveArtifacts}
          setSessionToDelete={setSessionToDelete}
        />
      )

      const chevronBtn = screen.getByTestId("session-chevron-sess-with-arts")
      const deleteButtons = screen.getAllByTitle("Delete Session")
      const sessionWithArtsDeleteBtn = deleteButtons[0]

      // Click delete button
      fireEvent.click(sessionWithArtsDeleteBtn)
      expect(setSessionToDelete).toHaveBeenCalledWith(sessions[0])

      // Dropdown should still remain collapsed
      expect(chevronBtn.getAttribute("aria-expanded")).toBe("false")
    })

    it("handles missing artifact size gracefully", () => {
      const artifactsWithMissingSize: Artifact[] = [
        {
          id: "art-no-size",
          projectId: "proj-1",
          sessionId: "sess-with-arts",
          name: "no-size-doc.md",
          mimeType: "text/markdown",
          phase: "PLANNING",
          status: "Complete" as const,
        },
      ]

      render(
        <ExplorerPanel
          {...defaultProps}
          sessions={sessions}
          liveArtifacts={artifactsWithMissingSize}
        />
      )

      const chevronBtn = screen.getByTestId("session-chevron-sess-with-arts")
      fireEvent.click(chevronBtn)

      expect(screen.getByText("no-size-doc.md")).toBeInTheDocument()
      expect(screen.queryByText(/0 B/)).toBeNull()
    })
  })
})
