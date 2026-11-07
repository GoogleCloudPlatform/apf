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
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import * as matchers from "@testing-library/jest-dom/matchers"
expect.extend(matchers)
import DocsPage from "./page"
import { useAppStore } from "@/stores/app"

// Mock next/navigation
vi.mock("next/navigation", () => ({
  usePathname: () => "/docs",
  useSearchParams: () => new URLSearchParams(),
  useRouter: () => ({
    push: vi.fn(),
  }),
}))

describe("DocsPage Component", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    // Mock clipboard
    Object.assign(navigator, {
      clipboard: {
        writeText: vi.fn().mockImplementation(() => Promise.resolve()),
      },
    })
  })

  afterEach(() => {
    cleanup()
  })

  it("renders BMAD Reference navigation tab and displays BMAD Reference content by default", () => {
    render(<DocsPage />)

    // BMAD Reference tab is present in navigation
    expect(screen.getByTestId("docs-nav-bmad-reference")).toBeInTheDocument()

    // BMAD Reference content should be visible by default
    expect(screen.getByRole("heading", { name: "BMAD Reference" })).toBeInTheDocument()
  })

  it("hides getting-started, architecture, and integrations navigation tabs from display", () => {
    render(<DocsPage />)

    expect(screen.queryByTestId("docs-nav-getting-started")).not.toBeInTheDocument()
    expect(screen.queryByTestId("docs-nav-architecture")).not.toBeInTheDocument()
    expect(screen.queryByTestId("docs-nav-integrations")).not.toBeInTheDocument()

    expect(screen.queryByRole("heading", { name: "Getting Started with Foundry" })).not.toBeInTheDocument()
    expect(screen.queryByRole("heading", { name: "Agent Swarm Architecture" })).not.toBeInTheDocument()
    expect(screen.queryByRole("heading", { name: "Integrations & Credentials" })).not.toBeInTheDocument()
  })

  it("sets the page title to 'Documentation & Reference' on load", () => {
    render(<DocsPage />)
    const title = useAppStore.getState().pageTitle
    expect(title).toBe("Documentation & Reference")
  })

  it("filters BMAD skills when typing in the search input on BMAD Reference tab", () => {
    render(<DocsPage />)

    // Switch to BMAD Reference
    fireEvent.click(screen.getByTestId("docs-nav-bmad-reference"))

    const searchInput = screen.getByTestId("bmad-search-input")
    expect(searchInput).toBeInTheDocument()

    // Initial check: APF-supported skills present by default
    expect(screen.getByTestId("bmad-skill-card-bmad-brainstorming")).toBeInTheDocument()
    expect(screen.getByTestId("bmad-skill-card-bmad-prd")).toBeInTheDocument()

    // Type "Brainstorm"
    fireEvent.change(searchInput, { target: { value: "Brainstorm" } })

    expect(screen.getByTestId("bmad-skill-card-bmad-brainstorming")).toBeInTheDocument()
    expect(screen.queryByTestId("bmad-skill-card-bmad-prd")).not.toBeInTheDocument()

    // Search by shortcut code "PRD"
    fireEvent.change(searchInput, { target: { value: "PRD" } })
    expect(screen.getByTestId("bmad-skill-card-bmad-prd")).toBeInTheDocument()

    // Non-existent search shows empty state
    fireEvent.change(searchInput, { target: { value: "xyznonexistentquery999" } })
    expect(screen.getByText("No matching BMAD skills found")).toBeInTheDocument()
  })

  it("toggles visibility of IDE based skills with the 'Show IDE based skills' toggle", () => {
    render(<DocsPage />)

    // Switch to BMAD Reference
    fireEvent.click(screen.getByTestId("docs-nav-bmad-reference"))

    const ideToggle = screen.getByTestId("bmad-toggle-ide-skills")
    expect(ideToggle).toBeInTheDocument()
    expect(ideToggle.checked).toBe(false)

    // IDE-based skills (apf_supported: false) should NOT be visible by default
    expect(screen.queryByTestId("bmad-skill-card-bmad-agent-builder")).not.toBeInTheDocument()
    expect(screen.queryByTestId("bmad-skill-card-bmad-customize")).not.toBeInTheDocument()
    expect(screen.queryByTestId("bmad-skill-card-bmad-workflow-builder")).not.toBeInTheDocument()

    // APF-supported skills should be visible by default
    expect(screen.getByTestId("bmad-skill-card-bmad-prd")).toBeInTheDocument()
    expect(screen.getByTestId("bmad-skill-card-bmad-build")).toBeInTheDocument()

    // Turn ON the toggle
    fireEvent.click(ideToggle)
    expect(ideToggle.checked).toBe(true)

    // Now IDE-based skills should be visible
    expect(screen.getByTestId("bmad-skill-card-bmad-agent-builder")).toBeInTheDocument()
    expect(screen.getByTestId("bmad-skill-card-bmad-customize")).toBeInTheDocument()
    expect(screen.getByTestId("bmad-skill-card-bmad-workflow-builder")).toBeInTheDocument()

    // Turn OFF the toggle
    fireEvent.click(ideToggle)
    expect(ideToggle.checked).toBe(false)
    expect(screen.queryByTestId("bmad-skill-card-bmad-agent-builder")).not.toBeInTheDocument()
  })

  it("filters BMAD skills by phase buttons", () => {
    render(<DocsPage />)

    // Switch to BMAD Reference
    fireEvent.click(screen.getByTestId("docs-nav-bmad-reference"))

    const implPhaseFilter = screen.getByTestId("bmad-phase-filter-4-implementation")
    fireEvent.click(implPhaseFilter)

    expect(screen.getByTestId("bmad-skill-card-bmad-retrospective")).toBeInTheDocument()
    expect(screen.queryByTestId("bmad-skill-card-bmad-prd")).not.toBeInTheDocument()

    const planPhaseFilter = screen.getByTestId("bmad-phase-filter-2-planning")
    fireEvent.click(planPhaseFilter)

    expect(screen.getByTestId("bmad-skill-card-bmad-prd")).toBeInTheDocument()
    expect(screen.queryByTestId("bmad-skill-card-bmad-retrospective")).not.toBeInTheDocument()
  })

  it("filters BMAD skills by module dropdown", () => {
    render(<DocsPage />)

    // Switch to BMAD Reference
    fireEvent.click(screen.getByTestId("docs-nav-bmad-reference"))

    const moduleSelect = screen.getByTestId("bmad-module-select")
    fireEvent.change(moduleSelect, { target: { value: "Test Architecture Enterprise" } })

    expect(screen.getByTestId("bmad-skill-card-bmad-testarch-test-design")).toBeInTheDocument()
    expect(screen.queryByTestId("bmad-skill-card-bmad-prd")).not.toBeInTheDocument()
  })

  it("opens the skill detail modal when clicking a skill card", () => {
    render(<DocsPage />)

    // Switch to BMAD Reference
    fireEvent.click(screen.getByTestId("docs-nav-bmad-reference"))

    const prdCard = screen.getByTestId("bmad-skill-card-bmad-prd")
    fireEvent.click(prdCard)

    // Modal should be visible with title and detailed contracts
    expect(screen.getByRole("dialog")).toBeInTheDocument()
    expect(screen.getByText("Workflow Pipeline Position")).toBeInTheDocument()
    expect(screen.getByText("Output Deliverables & Contracts")).toBeInTheDocument()
    expect(screen.getByText("Example Usages & Prompts")).toBeInTheDocument()
  })

  it("displays sandbox icon for sandboxed skills and hides it for non-sandboxed skills", () => {
    render(<DocsPage />)

    // Switch to BMAD Reference
    fireEvent.click(screen.getByTestId("docs-nav-bmad-reference"))

    // bmad-build requires sandbox -> should show sandbox icon
    expect(screen.getByTestId("bmad-icon-sandbox-bmad-build")).toBeInTheDocument()

    // bmad-prd does not require sandbox -> should NOT show sandbox icon
    expect(screen.queryByTestId("bmad-icon-sandbox-bmad-prd")).not.toBeInTheDocument()

    // Web icons should not exist on any skills
    expect(screen.queryByTestId("bmad-icon-web-bmad-build")).not.toBeInTheDocument()
    expect(screen.queryByTestId("bmad-icon-web-bmad-prd")).not.toBeInTheDocument()
  })
})
