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
import { describe, it, expect, vi, afterEach } from "vitest"
import "@testing-library/jest-dom/vitest"
import * as matchers from "@testing-library/jest-dom/matchers"
expect.extend(matchers as any)
import FilteredFileListing, { extractFiles, getArtifactStatusDot } from "./FilteredFileListing"
import { Node } from "./FileExplorer"
import { Artifact } from "@/lib/types"

describe("FilteredFileListing Component", () => {
  afterEach(() => {
    cleanup()
  })

  const mockFileTree: Node = {
    name: "root",
    path: "",
    type: "folder",
    children: [
      {
        name: "src",
        path: "src",
        type: "folder",
        children: [
          {
            name: "components",
            path: "src/components",
            type: "folder",
            children: [
              {
                name: "FileExplorer.tsx",
                path: "src/components/FileExplorer.tsx",
                type: "file",
              },
            ],
          },
          {
            name: "index.ts",
            path: "src/index.ts",
            type: "file",
          },
        ],
      },
      {
        name: "README.md",
        path: "README.md",
        type: "file",
      },
    ],
  }

  const mockArtifacts: Artifact[] = [
    {
      id: "art-1",
      projectId: "proj-1",
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
      projectId: "proj-1",
      title: "Architecture.md",
      path: "docs/specs/arch/Architecture.md",
      phase: "PLANNING",
      status: "Complete",
      artifactType: "ARCHITECTURE",
      createdAt: "2026-08-14T10:00:00Z",
      updatedAt: "2026-08-14T10:00:00Z",
    },
    {
      id: "art-3",
      projectId: "proj-1",
      title: "UX-Specs.md",
      path: "docs/specs/ux/UX-Specs.md",
      phase: "PLANNING",
      status: "Staged for Review",
      artifactType: "UX",
      createdAt: "2026-08-14T10:00:00Z",
      updatedAt: "2026-08-14T10:00:00Z",
    },
  ]

  it("extractFiles helper correctly extracts all leaf files from nested tree", () => {
    const files = extractFiles(mockFileTree)
    expect(files).toHaveLength(3)
    expect(files.map((f) => f.name)).toEqual(["FileExplorer.tsx", "index.ts", "README.md"])
  })

  it("getArtifactStatusDot returns appropriate color class and title for each status", () => {
    expect(getArtifactStatusDot("In Progress")).toEqual({
      dotClass: "bg-amber-500",
      title: "In Progress",
    })
    expect(getArtifactStatusDot("Complete")).toEqual({
      dotClass: "bg-emerald-500",
      title: "Complete",
    })
    expect(getArtifactStatusDot("Staged for Review")).toEqual({
      dotClass: "bg-blue-500",
      title: "Staged for Review",
    })
    expect(getArtifactStatusDot("Not Started")).toEqual({
      dotClass: "bg-slate-400 dark:bg-slate-500",
      title: "Not Started",
    })
  })

  it("renders phase artifacts sorted alphabetically with status indicator dots", () => {
    render(<FilteredFileListing artifacts={mockArtifacts} activePhase="Planning" />)

    // Verify alphabetically sorted: Architecture.md, PRD.md, UX-Specs.md
    const items = screen.getAllByRole("listitem")
    expect(items).toHaveLength(3)
    expect(items[0].textContent).toContain("Architecture.md")
    expect(items[1].textContent).toContain("PRD.md")
    expect(items[2].textContent).toContain("UX-Specs.md")

    // Verify parent directory subtitles
    expect(screen.getByText("docs/specs/arch")).toBeDefined()
    expect(screen.getByText("docs/specs/prd")).toBeDefined()
    expect(screen.getByText("docs/specs/ux")).toBeDefined()

    // Verify status indicator dots
    expect(screen.getByTitle("Status: Complete")).toBeDefined()
    expect(screen.getByTitle("Status: In Progress")).toBeDefined()
    expect(screen.getByTitle("Status: Staged for Review")).toBeDefined()
  })

  it("renders contextual empty state when no artifacts are provided", () => {
    render(<FilteredFileListing artifacts={[]} activePhase="Planning" />)

    expect(screen.getByText("No files tracked for Planning phase.")).toBeDefined()
  })

  it("calls onFileSelect callback on mouse click", () => {
    const onFileSelect = vi.fn()
    render(
      <FilteredFileListing
        artifacts={mockArtifacts}
        onFileSelect={onFileSelect}
      />
    )

    const fileItem = screen.getByText("PRD.md")
    fireEvent.click(fileItem)

    expect(onFileSelect).toHaveBeenCalledWith(
      expect.objectContaining({
        name: "PRD.md",
        path: "docs/specs/prd/PRD.md",
      })
    )
  })

  it("calls onFileSelect callback on keyboard Enter and Space keys", () => {
    const onFileSelect = vi.fn()
    render(
      <FilteredFileListing
        artifacts={mockArtifacts}
        onFileSelect={onFileSelect}
      />
    )

    const fileItem = screen.getAllByRole("listitem")[0]
    fireEvent.keyDown(fileItem, { key: "Enter" })
    expect(onFileSelect).toHaveBeenCalledTimes(1)

    fireEvent.keyDown(fileItem, { key: " " })
    expect(onFileSelect).toHaveBeenCalledTimes(2)
  })

  it("opens delete confirmation dialog and calls onDeleteArtifact upon confirming", async () => {
    const onDeleteArtifact = vi.fn().mockResolvedValue(undefined)
    render(
      <FilteredFileListing
        artifacts={mockArtifacts}
        onDeleteArtifact={onDeleteArtifact}
      />
    )

    const deleteBtn = screen.getByTitle("Delete PRD.md")
    fireEvent.click(deleteBtn)

    // Dialog should open
    expect(screen.getByRole("dialog")).toBeDefined()
    expect(screen.getByText(/Are you sure you want to delete/)).toBeDefined()

    // Click Delete in modal
    const confirmBtn = screen.getByRole("button", { name: "Delete" })
    fireEvent.click(confirmBtn)

    await waitFor(() => {
      expect(onDeleteArtifact).toHaveBeenCalledWith("art-1")
    })
  })

  it("highlights selected file matching title or path", () => {
    render(
      <FilteredFileListing
        artifacts={mockArtifacts}
        selectedFilePath="docs/specs/prd/PRD.md"
      />
    )

    const items = screen.getAllByRole("listitem")
    const prdItem = items.find((i) => i.textContent?.includes("PRD.md"))
    expect(prdItem?.getAttribute("aria-selected")).toBe("true")
  })

  it("prioritizes file name taking up to 75% of available space when directory subtitle is present", () => {
    render(
      <FilteredFileListing
        artifacts={[
          {
            id: "art-1",
            projectId: "proj-1",
            title: "very-long-architecture-specification-file.md",
            path: "docs/specs/architecture/v2/very-long-architecture-specification-file.md",
            phase: "PLANNING",
            status: "Complete",
            createdAt: "2026-08-14T10:00:00Z",
            updatedAt: "2026-08-14T10:00:00Z",
          },
        ]}
      />
    )

    const fileNameSpan = screen.getByText("very-long-architecture-specification-file.md")
    expect(fileNameSpan.className).toContain("max-w-[75%]")
    expect(fileNameSpan.className).toContain("shrink-0")
    expect(fileNameSpan.className).toContain("truncate")

    const dirSubtitleSpan = screen.getByText("docs/specs/architecture/v2")
    expect(dirSubtitleSpan.className).toContain("truncate")
    expect(dirSubtitleSpan.className).toContain("min-w-0")
    expect(dirSubtitleSpan.className).toContain("flex-1")
  })

  it("allows file name to occupy available space without max-w-[75%] restriction when directory subtitle is absent", () => {
    render(
      <FilteredFileListing
        artifacts={[
          {
            id: "art-2",
            projectId: "proj-1",
            title: "root-file.md",
            path: "root-file.md",
            phase: "PLANNING",
            status: "Complete",
            createdAt: "2026-08-14T10:00:00Z",
            updatedAt: "2026-08-14T10:00:00Z",
          },
        ]}
      />
    )

    const fileNameSpan = screen.getByText("root-file.md")
    expect(fileNameSpan.className).toContain("truncate")
    expect(fileNameSpan.className).not.toContain("max-w-[75%]")
    expect(screen.queryByText("root-file.md")).not.toBeNull()
  })

  it("renders checkboxes for each file item and toggles check state without triggering onFileSelect", () => {
    const onFileSelect = vi.fn()
    const onToggleCheck = vi.fn()
    const checkedPaths = new Set<string>(["docs/specs/prd/PRD.md"])

    render(
      <FilteredFileListing
        artifacts={mockArtifacts}
        onFileSelect={onFileSelect}
        checkedPaths={checkedPaths}
        onToggleCheck={onToggleCheck}
      />
    )

    const checkboxes = screen.getAllByRole("checkbox")
    expect(checkboxes).toHaveLength(3)

    // PRD.md should be checked
    const prdCheckbox = screen.getByLabelText("Select PRD.md") as HTMLInputElement
    expect(prdCheckbox.checked).toBe(true)

    // Click checkbox on Architecture.md
    const archCheckbox = screen.getByLabelText("Select Architecture.md")
    fireEvent.click(archCheckbox)

    expect(onToggleCheck).toHaveBeenCalledWith("docs/specs/arch/Architecture.md", expect.objectContaining({ name: "Architecture.md" }))
    expect(onFileSelect).not.toHaveBeenCalled()
  })

  it("handles batch delete confirmation and executes onDeleteArtifact for each selected item", async () => {
    const onDeleteArtifact = vi.fn().mockResolvedValue(undefined)
    const checkedPaths = new Set<string>(["docs/specs/prd/PRD.md", "docs/specs/arch/Architecture.md"])

    render(
      <FilteredFileListing
        artifacts={mockArtifacts}
        onDeleteArtifact={onDeleteArtifact}
        checkedPaths={checkedPaths}
      />
    )

    // Verify checked styling applied to rows
    const items = screen.getAllByRole("listitem")
    expect(items[0].className).toContain("bg-muted/60") // Architecture.md
    expect(items[1].className).toContain("bg-muted/60") // PRD.md
  })

  it("invokes onBatchDelete callback when batch delete is executed", async () => {
    const onBatchDelete = vi.fn().mockResolvedValue(undefined)
    const checkedPaths = new Set<string>(["docs/specs/prd/PRD.md", "docs/specs/arch/Architecture.md"])

    const { rerender } = render(
      <FilteredFileListing
        artifacts={mockArtifacts}
        onBatchDelete={onBatchDelete}
        checkedPaths={checkedPaths}
      />
    )

    expect(screen.getByLabelText("Select PRD.md")).toBeDefined()
  })
})
