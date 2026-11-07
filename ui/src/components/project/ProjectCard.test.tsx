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
import { render, screen, fireEvent, waitFor, cleanup } from "@testing-library/react"
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import * as matchers from "@testing-library/jest-dom/matchers"
expect.extend(matchers)
import { ProjectCard } from "./ProjectCard"
import { Project, WorkflowTemplate } from "@/lib/types"

// Mock dependencies
const mockPush = vi.fn()
vi.mock("next/navigation", () => ({
  useRouter: () => ({
    push: mockPush,
  }),
}))

const mockSetLastSelectedProjectId = vi.fn()
vi.mock("@/stores/app", () => ({
  useAppStore: () => ({
    setLastSelectedProjectId: mockSetLastSelectedProjectId,
  }),
}))

const mockPut = vi.fn()
vi.mock("@/lib/api", () => ({
  api: {
    put: (...args: any[]) => mockPut(...args),
  },
}))

const mockBmadWorkflow: WorkflowTemplate = {
  id: "bmad",
  name: "BMad Workflow",
  framework: "bmad",
  phases: {},
  artifacts: {},
  dependencies: {},
}

const mockProjectTemplate: Project = {
  id: "project-123",
  name: "Test Project",
  description: "A test project description",
  status: "New",
  artifacts: [],
  icon: "🚀",
  repoAccessToken: "",
  baseBranch: "main",
  governanceGates: {},
}

describe("ProjectCard", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    cleanup()
  })

  it("renders Onboard label when project status is New", () => {
    const project = { ...mockProjectTemplate, status: "New" }
    render(<ProjectCard project={project} bmadWorkflow={mockBmadWorkflow} />)

    const button = screen.getByRole("button", { name: "Onboard" })
    expect(button).toBeInTheDocument()
    expect(screen.getByTestId("project-card-status-new")).toBeInTheDocument()
  })

  it("renders Onboard label when project status is Onboarding", () => {
    const project = { ...mockProjectTemplate, status: "Onboarding" }
    render(<ProjectCard project={project} bmadWorkflow={mockBmadWorkflow} />)

    const button = screen.getByRole("button", { name: "Onboard" })
    expect(button).toBeInTheDocument()
    expect(screen.getByTestId("project-card-status-onboarding")).toBeInTheDocument()
  })

  it("renders Open Workspace label when project status is Onboarded", () => {
    const project = { ...mockProjectTemplate, status: "Onboarded" }
    render(<ProjectCard project={project} bmadWorkflow={mockBmadWorkflow} />)

    const button = screen.getByRole("button", { name: "Open Workspace" })
    expect(button).toBeInTheDocument()
    expect(screen.getByTestId("project-card-status-onboarded")).toBeInTheDocument()
  })

  it("performs API transition to Onboarding and redirects on New click", async () => {
    const project = { ...mockProjectTemplate, status: "New" }
    mockPut.mockResolvedValueOnce({ status: "Onboarding" })

    render(<ProjectCard project={project} bmadWorkflow={mockBmadWorkflow} />)

    const button = screen.getByRole("button", { name: "Onboard" })
    fireEvent.click(button)

    expect(mockSetLastSelectedProjectId).toHaveBeenCalledWith("project-123")
    expect(mockPut).toHaveBeenCalledWith("/api/v1/projects/project-123", { status: "Onboarding" })

    await waitFor(() => {
      expect(mockPush).toHaveBeenCalledWith("/projects/project-123/workspace?phase=ONBOARDING")
    })
  })

  it("redirects directly to Onboarding workspace on Onboarding click", () => {
    const project = { ...mockProjectTemplate, status: "Onboarding" }

    render(<ProjectCard project={project} bmadWorkflow={mockBmadWorkflow} />)

    const button = screen.getByRole("button", { name: "Onboard" })
    fireEvent.click(button)

    expect(mockSetLastSelectedProjectId).toHaveBeenCalledWith("project-123")
    expect(mockPut).not.toHaveBeenCalled()
    expect(mockPush).toHaveBeenCalledWith("/projects/project-123/workspace?phase=ONBOARDING")
  })

  it("redirects to standard workspace on Onboarded/other click", () => {
    const project = { ...mockProjectTemplate, status: "Onboarded" }

    render(<ProjectCard project={project} bmadWorkflow={mockBmadWorkflow} />)

    const button = screen.getByRole("button", { name: "Open Workspace" })
    fireEvent.click(button)

    expect(mockSetLastSelectedProjectId).toHaveBeenCalledWith("project-123")
    expect(mockPut).not.toHaveBeenCalled()
    expect(mockPush).toHaveBeenCalledWith("/projects/project-123/workspace")
  })

  it("renders 4 progress indicators", () => {
    const project = { ...mockProjectTemplate, status: "New" }
    const { container } = render(<ProjectCard project={project} bmadWorkflow={mockBmadWorkflow} />)
    
    // The indicators container has title "Governance Gates..."
    const indicatorsContainer = container.querySelector('[title^="Governance Gates"]')
    expect(indicatorsContainer).toBeInTheDocument()
    expect(indicatorsContainer).toHaveAttribute("title", "Governance Gates (Onboarding, Analysis, Planning, Solutioning)")
    
    // Check there are 4 span elements inside it
    const indicators = indicatorsContainer?.querySelectorAll('span')
    expect(indicators?.length).toBe(4)
  })
})
