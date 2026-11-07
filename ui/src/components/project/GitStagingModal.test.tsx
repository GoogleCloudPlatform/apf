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

// @ts-nocheck
// @vitest-environment jsdom
import React from "react"
import { render, screen, fireEvent, cleanup } from "@testing-library/react"
import { describe, it, expect, vi, afterEach } from "vitest"
import * as matchers from "@testing-library/jest-dom/matchers"
expect.extend(matchers)
import { GitStagingModal, GitFileChange } from "./GitStagingModal"

describe("GitStagingModal Component", () => {
  afterEach(() => {
    cleanup()
  })

  const mockFiles: GitFileChange[] = [
    { path: "docs/spec/prd.md", status: "modified", isChecked: true },
    { path: "docs/spec/architecture.md", status: "untracked", isChecked: false },
  ]

  it("renders changed file list with current document checked by default", () => {
    render(
      <GitStagingModal
        isOpen={true}
        onClose={vi.fn()}
        currentFilePath="docs/spec/prd.md"
        changedFiles={mockFiles}
        onConfirmSync={vi.fn()}
      />
    )

    expect(screen.getByText(/docs\/spec\/prd.md/i)).toBeInTheDocument()
    expect(screen.getByText(/Current Document/i)).toBeInTheDocument()
    expect(screen.getByText(/docs\/spec\/architecture.md/i)).toBeInTheDocument()
  })

  it("calls onConfirmSync with selected paths when Push Changes is clicked", () => {
    const handleConfirmSync = vi.fn()
    render(
      <GitStagingModal
        isOpen={true}
        onClose={vi.fn()}
        currentFilePath="docs/spec/prd.md"
        changedFiles={mockFiles}
        onConfirmSync={handleConfirmSync}
      />
    )

    const confirmButton = screen.getByText(/^Push Changes \(/i)
    fireEvent.click(confirmButton)

    expect(handleConfirmSync).toHaveBeenCalledWith(["docs/spec/prd.md"], "sync", expect.any(Function), "push")
  })

  it("renders Request Review button", () => {
    const handleConfirmSync = vi.fn()
    render(
      <GitStagingModal
        isOpen={true}
        onClose={vi.fn()}
        currentFilePath="docs/spec/prd.md"
        changedFiles={mockFiles}
        onConfirmSync={handleConfirmSync}
      />
    )

    const reviewBtn = screen.getByText(/^Request Review/i)

    expect(reviewBtn).toBeInTheDocument()

    fireEvent.click(reviewBtn)
    expect(handleConfirmSync).toHaveBeenCalledWith(["docs/spec/prd.md"], "request_review", expect.any(Function), "push")
  })

  it("ensures only one accordion section is open at a time", () => {
    const handleConfirmSync = vi.fn()
    render(
      <GitStagingModal
        isOpen={true}
        onClose={vi.fn()}
        currentFilePath="docs/spec/prd.md"
        changedFiles={mockFiles}
        onConfirmSync={handleConfirmSync}
      />
    )

    // Initially File List accordion is open
    expect(screen.getByText(/Select All \(2 items\)/i)).toBeInTheDocument()

    // Trigger action -> switches active accordion exclusively to Execution Stages
    const reviewBtn = screen.getByText(/^Request Review/i)
    fireEvent.click(reviewBtn)

    // Execution stages is open, File list is closed
    expect(screen.getByText(/🚀 Execution Stages/i)).toBeInTheDocument()
    expect(screen.queryByText(/Select All \(2 items\)/i)).not.toBeInTheDocument()

    // Clicking File List header opens File List and closes Execution Stages
    const fileAccordionHeader = screen.getByText(/📁 Changed Files/i)
    fireEvent.click(fileAccordionHeader)

    expect(screen.getByText(/Select All \(2 items\)/i)).toBeInTheDocument()
    expect(screen.queryByText(/Create Safety Snapshot/i)).not.toBeInTheDocument()
  })

  it("toggles sync direction when direction badge toggle button is clicked", () => {
    const handleConfirmSync = vi.fn()
    render(
      <GitStagingModal
        isOpen={true}
        onClose={vi.fn()}
        currentFilePath="docs/spec/prd.md"
        changedFiles={mockFiles}
        onConfirmSync={handleConfirmSync}
      />
    )

    // Default direction is Push
    expect(screen.getByText(/Local to Remote/i)).toBeInTheDocument()
    expect(screen.getByText(/Your Branch \(Source\)/i)).toBeInTheDocument()

    // Click direction toggle badge button
    const toggleBtn = screen.getByTitle(/Click to toggle sync direction/i)
    fireEvent.click(toggleBtn)

    // Toggled direction is Pull
    expect(screen.getByText(/Remote to Local/i)).toBeInTheDocument()
    expect(screen.getByText(/Your Branch \(Destination\)/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /^Pull Changes/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Refresh/i })).toBeInTheDocument()
  })

  it("calls onPreviewChanges when Preview Changes button is clicked in pull mode", () => {
    const handlePreviewChanges = vi.fn()
    render(
      <GitStagingModal
        isOpen={true}
        onClose={vi.fn()}
        currentFilePath="docs/spec/prd.md"
        changedFiles={mockFiles}
        onConfirmSync={vi.fn()}
        onPreviewChanges={handlePreviewChanges}
      />
    )

    // Swap to pull mode
    const toggleBtn = screen.getByTitle(/Click to toggle sync direction/i)
    fireEvent.click(toggleBtn)

    const refreshBtn = screen.getByRole('button', { name: /Refresh/i })
    expect(refreshBtn).toBeInTheDocument()

    fireEvent.click(refreshBtn)
    expect(handlePreviewChanges).toHaveBeenCalled()
  })

  it("disables Pull Changes button when there are no files to pull or no selection", () => {
    render(
      <GitStagingModal
        isOpen={true}
        onClose={vi.fn()}
        currentFilePath={null}
        changedFiles={[]}
        onConfirmSync={vi.fn()}
      />
    )

    // Swap to pull mode
    const toggleBtn = screen.getByTitle(/Click to toggle sync direction/i)
    fireEvent.click(toggleBtn)

    const pullBtn = screen.getByRole('button', { name: /Pull Changes/i })
    expect(pullBtn).toBeDisabled()
  })

  it("disables Pull Changes button when onPreviewChanges returns unmergeable status", async () => {
    const handlePreviewChanges = vi.fn().mockResolvedValue({ mergeable: false })
    render(
      <GitStagingModal
        isOpen={true}
        onClose={vi.fn()}
        currentFilePath="docs/spec/prd.md"
        changedFiles={mockFiles}
        onConfirmSync={vi.fn()}
        onPreviewChanges={handlePreviewChanges}
      />
    )

    // Swap to pull mode
    const toggleBtn = screen.getByTitle(/Click to toggle sync direction/i)
    fireEvent.click(toggleBtn)

    const refreshBtn = screen.getByRole('button', { name: /Refresh/i })
    fireEvent.click(refreshBtn)

    const pullBtn = screen.getByRole('button', { name: /Pull Changes/i })
    expect(pullBtn).toBeDisabled()
  })

  it("triggers onRefetchStatus when modal opens to launch diff analysis after modal open", () => {
    const handleRefetchStatus = vi.fn().mockResolvedValue([])
    render(
      <GitStagingModal
        isOpen={true}
        onClose={vi.fn()}
        currentFilePath={null}
        changedFiles={[]}
        onConfirmSync={vi.fn()}
        onRefetchStatus={handleRefetchStatus}
      />
    )

    expect(handleRefetchStatus).toHaveBeenCalled()
  })

  it("shows loading spinner while diff analysis is in progress on modal open", () => {
    // Return unresolved promise to test loading state
    const handleRefetchStatus = vi.fn().mockImplementation(() => new Promise(() => {}))
    render(
      <GitStagingModal
        isOpen={true}
        onClose={vi.fn()}
        currentFilePath={null}
        changedFiles={[]}
        onConfirmSync={vi.fn()}
        onRefetchStatus={handleRefetchStatus}
      />
    )

    expect(screen.getByText(/Analyzing workspace changes.../i)).toBeInTheDocument()
  })

  it("shows only Done button after push process completes successfully", async () => {
    const handleConfirmSync = vi.fn().mockImplementation(async (_paths, _action, onProgress) => {
      onProgress([
        { id: "snapshot", label: "Create Safety Snapshot", status: "completed" },
        { id: "commit", label: "Approving changes locally", status: "completed" },
        { id: "fetch", label: "Fetch Remote Changes", status: "completed" },
        { id: "sync", label: "Merged Changes", status: "completed" },
      ])
    })

    render(
      <GitStagingModal
        isOpen={true}
        onClose={vi.fn()}
        currentFilePath="docs/spec/prd.md"
        changedFiles={mockFiles}
        onConfirmSync={handleConfirmSync}
      />
    )

    const pushBtn = screen.getByRole("button", { name: /^Push Changes/i })
    fireEvent.click(pushBtn)

    expect(screen.getByRole("button", { name: "Done" })).toBeInTheDocument()
    expect(screen.queryByRole("button", { name: /^Push Changes/i })).not.toBeInTheDocument()
    expect(screen.queryByRole("button", { name: "Cancel" })).not.toBeInTheDocument()
  })
})
