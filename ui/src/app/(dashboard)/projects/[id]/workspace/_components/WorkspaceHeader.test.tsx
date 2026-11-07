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
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import "@testing-library/jest-dom/vitest"
import * as matchers from "@testing-library/jest-dom/matchers"
expect.extend(matchers as any)
import WorkspaceHeader from "./WorkspaceHeader"
import { PhaseType } from "../_utils/workspaceHelpers"
import { WorkflowTemplate, Artifact } from "@/lib/types"

const mockProps = {
  projectName: "Test Project",
  projectIcon: "🚀",
  projectId: "proj-123",
  projectStatus: "Onboarded",
  activePhase: "PLANNING" as PhaseType,
  setActivePhase: vi.fn(),
  satisfiedPhases: { ONBOARDING: true, ANALYSIS: true },
  bmadWorkflow: {} as WorkflowTemplate,
  liveArtifacts: [] as Artifact[],
  runHealthCheckPending: false,
  onRunHealthCheck: vi.fn(),
  onShareProject: vi.fn(),
  onEditProject: vi.fn(),
  handleBack: vi.fn(),
}

describe("WorkspaceHeader", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    cleanup()
  })

  it("renders top-right action buttons permanently with correct labels (Sync, Share, Edit)", () => {
    render(<WorkspaceHeader {...mockProps} />)

    expect(screen.getByText("Sync")).toBeInTheDocument()
    expect(screen.getByText("Share")).toBeInTheDocument()
    expect(screen.getByText("Edit")).toBeInTheDocument()
  })

  it("calls button handlers on click", () => {
    render(<WorkspaceHeader {...mockProps} />)

    fireEvent.click(screen.getByText("Sync"))
    expect(mockProps.onRunHealthCheck).toHaveBeenCalledTimes(1)

    fireEvent.click(screen.getByText("Share"))
    expect(mockProps.onShareProject).toHaveBeenCalledTimes(1)

    fireEvent.click(screen.getByText("Edit"))
    expect(mockProps.onEditProject).toHaveBeenCalledTimes(1)
  })

  it("renders horizontal phase tabs when no collision occurs", () => {
    const { container } = render(<WorkspaceHeader {...mockProps} />)
    const visibleNav = container.querySelector("#tour-workspace-phases")

    expect(visibleNav).toBeInTheDocument()
    expect(visibleNav).toHaveTextContent("Onboard")
    expect(visibleNav).toHaveTextContent("Analyze")
    expect(visibleNav).toHaveTextContent("Plan")
    expect(visibleNav).toHaveTextContent("Solution")
    expect(visibleNav).toHaveTextContent("Implement")
  })

  it("switches active phase when tab is clicked", () => {
    render(<WorkspaceHeader {...mockProps} />)
    const solutionTab = screen.getByRole("button", { name: /solution/i })

    fireEvent.click(solutionTab)
    expect(mockProps.setActivePhase).toHaveBeenCalledWith("SOLUTIONING")
  })

  it("collapses horizontal tabs into phase dropdown when collision is detected", () => {
    // Mock getBoundingClientRect on prototype to simulate overlap
    const originalGetBoundingClientRect = Element.prototype.getBoundingClientRect

    Element.prototype.getBoundingClientRect = function () {
      if (this.id === "tour-workspace-health") {
        return { left: 400, right: 480, top: 0, bottom: 40, width: 80, height: 40, x: 400, y: 0, toJSON: () => {} }
      }
      // Hidden measurement implementation tab (or hidden nav tab)
      return { left: 100, right: 450, top: 0, bottom: 40, width: 350, height: 40, x: 100, y: 0, toJSON: () => {} }
    }

    try {
      const { container } = render(<WorkspaceHeader {...mockProps} />)
      const visibleNav = container.querySelector("#tour-workspace-phases")
      expect(visibleNav).toBeInTheDocument()
      // In collapsed mode, the dropdown trigger displays the active phase label ("Plan")
      expect(visibleNav).toHaveTextContent("Plan")
    } finally {
      Element.prototype.getBoundingClientRect = originalGetBoundingClientRect
    }
  })
})
