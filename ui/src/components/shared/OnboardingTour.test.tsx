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
import { render, waitFor, cleanup } from "@testing-library/react"
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import OnboardingTour from "./OnboardingTour"
import { useOnboardingStore } from "@/stores/onboarding"

// Mock router/navigation
const mockPush = vi.fn()
const mockGet = vi.fn()
let currentPath = "/projects"

vi.mock("next/navigation", () => ({
  usePathname: () => currentPath,
  useRouter: () => ({ push: mockPush }),
  useSearchParams: () => ({
    get: mockGet
  }),
  useParams: () => {
    const match = currentPath.match(/\/projects\/([^\/]+)/)
    return match ? { id: match[1] } : {}
  }
}))

// Mock app store
vi.mock("@/stores/app", () => ({
  useAppStore: () => ({
    lastSelectedProjectId: "proj-123"
  })
}))

// Mock react-query
vi.mock("@tanstack/react-query", () => ({
  useQuery: () => ({ data: { id: "proj-123", name: "Test Project" } }),
  useQueryClient: () => ({})
}))

describe("OnboardingTour Component", () => {
  const mockStartTour = vi.fn()
  
  beforeEach(() => {
    vi.clearAllMocks()
    currentPath = "/projects"
    mockGet.mockReturnValue(null)
    
    // Reset onboarding store with mocked actions
    useOnboardingStore.setState({
      activeTour: null,
      currentStep: 0,
      completedTours: [],
      completedSections: [],
      lastActiveStep: null,
      steps: {
        catalog: [
          { target: "#step1", title: "Step 1", description: "Desc 1", placement: "bottom", sectionId: "catalog-overview" },
          { target: "#step2", title: "Step 2", description: "Desc 2", placement: "bottom", sectionId: "catalog-overview" }
        ],
        workspace: [
          { target: "#w1", title: "W1", description: "", placement: "bottom", sectionId: "workspace-overview" }
        ],
        settings: [
          { target: "#s1", title: "S1", description: "", placement: "bottom", sectionId: "settings-overview" }
        ],
      },
      startTour: mockStartTour,
      nextStep: vi.fn(),
      prevStep: vi.fn(),
      endTour: vi.fn(),
      completeTour: vi.fn(),
      advanceStep: vi.fn()
    })
  })

  afterEach(() => {
    cleanup()
  })

  it("auto-starts tour if not completed and on matching page", async () => {
    render(<OnboardingTour />)

    await waitFor(() => {
      expect(mockStartTour).toHaveBeenCalledWith("catalog", 0)
    }, { timeout: 2000 })
  })

  it("does not auto-start if already completed", async () => {
    useOnboardingStore.setState({
      completedSections: ["catalog-overview"]
    })
    
    render(<OnboardingTour />)

    await new Promise(r => setTimeout(r, 1800))
    expect(mockStartTour).not.toHaveBeenCalled()
  })

  it("resumes from lastActiveStep if set", async () => {
    useOnboardingStore.setState({
      lastActiveStep: {
        tourId: "catalog",
        stepIndex: 1
      }
    })

    render(<OnboardingTour />)

    await waitFor(() => {
      expect(mockStartTour).toHaveBeenCalledWith("catalog", 1)
    }, { timeout: 2000 })
  })


  it("auto-advances tour when navigating to projects list from #tour-leftbar-projects step", async () => {
    const mockNextStep = vi.fn()
    useOnboardingStore.setState({
      activeTour: "settings",
      currentStep: 0,
      steps: {
        ...useOnboardingStore.getState().steps,
        settings: [
          { target: "#tour-leftbar-projects", title: "Back to Projects", description: "", placement: "bottom", sectionId: "settings-integration" }
        ]
      },
      nextStep: mockNextStep
    })

    currentPath = "/settings"
    const { rerender } = render(<OnboardingTour />)

    // Simulate navigation to /projects
    currentPath = "/projects"
    rerender(<OnboardingTour />)

    await waitFor(() => {
      expect(mockNextStep).toHaveBeenCalled()
    })
  })
})
