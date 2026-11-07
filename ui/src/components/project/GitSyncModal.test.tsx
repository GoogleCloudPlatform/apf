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
import { GitSyncModal, GitSyncChecklistResponse } from "./GitSyncModal"

describe("GitSyncModal Component", () => {
  afterEach(() => {
    cleanup()
  })

  const mockResponse: GitSyncChecklistResponse = {
    status: "CONFLICT_DETECTED",
    message: "Rebase collision detected on spec branch.",
    dynamicBranches: {
      baseBranch: "main",
      specBranch: "specs/user1",
      backupRef: "refs/foundry/backups/20260731-150000-a1b2c3d",
      shadowBranch: "refs/foundry/shadow-rebase-12345",
    },
    steps: [
      { id: "step1", label: "Create Safety Snapshot", status: "completed" },
      { id: "step2", label: "Fetch Remote Updates", status: "completed" },
      { id: "step3", label: "Rebase on Shadow Branch", status: "failed", detail: "Merge conflict in docs/spec/prd.md" },
    ],
    conflictInfo: {
      conflictingFiles: ["docs/spec/prd.md"],
      pullRequestUrl: "https://github.com/example/repo/pull/42",
      cliInstructions: "git checkout specs/user1\ngit rebase origin/main",
      nextStepRecommendation: "Resolve conflicts via Pull Request or use Visual Card Chooser.",
    },
  }

  it("renders branch metadata badges and step checklist", () => {
    render(<GitSyncModal isOpen={true} onClose={vi.fn()} data={mockResponse} />)

    expect(screen.getByText(/Base: main/i)).toBeInTheDocument()
    expect(screen.getByText(/Spec: specs\/user1/i)).toBeInTheDocument()
    expect(screen.getByText(/Create Safety Snapshot/i)).toBeInTheDocument()
    expect(screen.getByText(/Rebase collision detected/i)).toBeInTheDocument()
  })

  it("renders conflict resolution callout with GitHub PR button and Visual Chooser trigger", () => {
    render(<GitSyncModal isOpen={true} onClose={vi.fn()} data={mockResponse} />)

    expect(screen.getByText(/Resolve Conflict on GitHub PR/i)).toBeInTheDocument()
    expect(screen.getByText(/Visual File Chooser/i)).toBeInTheDocument()
    expect(screen.getAllByText(/docs\/spec\/prd.md/i).length).toBeGreaterThan(0)
  })

  it("opens visual file chooser sub-view when Visual File Chooser button is clicked", () => {
    render(<GitSyncModal isOpen={true} onClose={vi.fn()} data={mockResponse} />)

    const chooserButton = screen.getByText(/Visual File Chooser/i)
    fireEvent.click(chooserButton)

    expect(screen.getByText(/Visual File Conflict Chooser/i)).toBeInTheDocument()
    expect(screen.getByText(/Keep My Version/i)).toBeInTheDocument()
    expect(screen.getByText(/Keep Remote Version/i)).toBeInTheDocument()
  })
})
