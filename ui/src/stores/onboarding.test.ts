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

import { describe, it, expect, beforeEach, afterEach, vi } from "vitest"
import { useOnboardingStore, TourStep } from "./onboarding"
import { api } from "@/lib/api"

// Mock api client
vi.mock("@/lib/api", () => ({
  api: {
    get: vi.fn(() => Promise.resolve({ ok: true, json: () => Promise.resolve({}) } as any)),
    put: vi.fn(() => Promise.resolve({ ok: true, json: () => Promise.resolve({}) } as any))
  }
}))

// Mock auth store
vi.mock("./auth", () => ({
  useAuthStore: {
    getState: () => ({ email: "test@example.com" })
  }
}))

const mockSteps: Record<string, TourStep[]> = {
  catalog: [
    { target: "#1", title: "S1 Step 1", description: "", placement: "bottom", sectionId: "sec-1", sectionTitle: "Section 1" },
    { target: "#2", title: "S1 Step 2", description: "", placement: "bottom", sectionId: "sec-1", sectionTitle: "Section 1" },
    { target: "#3", title: "S2 Step 1", description: "", placement: "bottom", sectionId: "sec-2", sectionTitle: "Section 2" },
    { target: "#4", title: "S2 Step 2", description: "", placement: "bottom", sectionId: "sec-2", sectionTitle: "Section 2" },
    { target: "#5", title: "No Sec Step", description: "", placement: "bottom" }, // Undefined sectionId
    { target: "#6", title: "S3 Step 1", description: "", placement: "bottom", sectionId: "sec-3", sectionTitle: "Section 3" },
  ]
}

describe("Onboarding Store Actions", () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    // Reset store state before each test
    useOnboardingStore.setState({
      activeTour: null,
      currentStep: 0,
      completedTours: [],
      completedSections: [],
      steps: mockSteps as any // Use mock steps for testing
    })
    if (typeof window !== "undefined") {
      localStorage.clear()
    }
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe("skipSection", () => {
    it("skips to the first step of the next section", () => {
      const store = useOnboardingStore
      store.getState().startTour("catalog", 0) // Start at sec-1, step 1
      
      store.getState().skipSection()
      
      expect(store.getState().currentStep).toBe(2) // Should jump to S2 Step 1 (index 2)
      expect(store.getState().completedSections).toContain("sec-1")
    })

    it("skips to step with undefined section if it is next", () => {
      const store = useOnboardingStore
      store.getState().startTour("catalog", 2) // Start at sec-2, step 1 (index 2)
      
      store.getState().skipSection()
      
      expect(store.getState().currentStep).toBe(4) // Should jump to No Sec Step (index 4)
      expect(store.getState().completedSections).toContain("sec-2")
    })

    it("skips from undefined section to next section", () => {
      const store = useOnboardingStore
      store.getState().startTour("catalog", 4) // Start at index 4 (undefined section)
      
      store.getState().skipSection()
      
      expect(store.getState().currentStep).toBe(5) // Should jump to sec-3 step 1 (index 5)
      // Since it was undefined, it shouldn't add "undefined" to completedSections
      expect(store.getState().completedSections).not.toContain("undefined")
      expect(store.getState().completedSections).not.toContain(undefined)
    })

    it("completes the tour if there is no next section/step", () => {
      const store = useOnboardingStore
      store.getState().startTour("catalog", 5) // Start at sec-3 (index 5)
      
      store.getState().skipSection()
      
      expect(store.getState().activeTour).toBeNull()
      expect(store.getState().currentStep).toBe(0)
      expect(store.getState().completedTours).toContain("catalog")
      expect(store.getState().completedSections).toContain("sec-3")
    })

    it("does nothing if no active tour", () => {
      const store = useOnboardingStore
      store.getState().skipSection()
      expect(store.getState().currentStep).toBe(0)
      expect(store.getState().completedSections).toEqual([])
    })
  })

  describe("skipTour", () => {
    it("resets tour state and marks tour as completed", () => {
      const store = useOnboardingStore
      store.getState().startTour("catalog", 1)
      
      store.getState().skipTour()
      
      expect(store.getState().activeTour).toBeNull()
      expect(store.getState().currentStep).toBe(0)
      expect(store.getState().completedTours).toContain("catalog")
    })
  })

  describe("getSectionFirstStepIndex", () => {
    it("returns correct index for existing section", () => {
      const store = useOnboardingStore
      const index = store.getState().getSectionFirstStepIndex("catalog", "sec-2")
      expect(index).toBe(2)
    })

    it("returns 0 if section does not exist", () => {
      const store = useOnboardingStore
      const index = store.getState().getSectionFirstStepIndex("catalog", "non-existent")
      expect(index).toBe(0)
    })
  })

  describe("getCurrentSectionInfo", () => {
    it("returns correct info for step in first section", () => {
      const store = useOnboardingStore
      store.getState().startTour("catalog", 1) // sec-1, step 2 (index 1)
      
      const info = store.getState().getCurrentSectionInfo()
      expect(info).toEqual({
        sectionTitle: "Section 1",
        sectionIndex: 1,
        totalSections: 4, // sec-1, sec-2, default, sec-3
        stepIndex: 2,
        totalSteps: 2
      })
    })

    it("returns correct info for step in section with undefined id", () => {
      const store = useOnboardingStore
      store.getState().startTour("catalog", 4) // index 4 (undefined section)
      
      const info = store.getState().getCurrentSectionInfo()
      expect(info).toEqual({
        sectionTitle: "Overview", // fallback title
        sectionIndex: 3,
        totalSections: 4,
        stepIndex: 1,
        totalSteps: 1
      })
    })

    it("returns null if no active tour", () => {
      const store = useOnboardingStore
      expect(store.getState().getCurrentSectionInfo()).toBeNull()
    })
  })

  describe("loadTourStatusFromBackend", () => {
    it("fetches status and updates state", async () => {
      const store = useOnboardingStore
      const mockData = {
        completed: false,
        completed_sections: ["catalog"],
        last_active_step: {
          section_id: "workspace",
          step_index: 2
        }
      }
      
      vi.mocked(api.get).mockResolvedValueOnce({
        ok: true,
        json: async () => mockData
      } as any)

      await store.getState().loadTourStatusFromBackend("test@example.com")

      expect(api.get).toHaveBeenCalledWith("/api/v1/users/test%40example.com/tour-status")
      expect(store.getState().completedTours).toContain("catalog")
      expect(store.getState().lastActiveStep).toEqual({
        tourId: "workspace",
        stepIndex: 2
      })
      expect(store.getState().tourStatus).toEqual({
        completed: false,
        completedSections: ["catalog"],
        lastActiveStep: {
          tourId: "workspace",
          stepIndex: 2
        }
      })
    })

    it("resets state if tour is completed on backend", async () => {
      const store = useOnboardingStore
      const mockData = {
        completed: true,
        completed_sections: ["catalog", "workspace", "settings"],
        last_active_step: null
      }
      
      vi.mocked(api.get).mockResolvedValueOnce({
        ok: true,
        json: async () => mockData
      } as any)

      await store.getState().loadTourStatusFromBackend("test@example.com")

      expect(store.getState().completedTours).toContain("catalog")
      expect(store.getState().completedTours).toContain("workspace")
      expect(store.getState().lastActiveStep).toBeNull()
      expect(store.getState().activeTour).toBeNull()
      expect(store.getState().tourStatus).toEqual({
        completed: true,
        completedSections: ["catalog", "workspace", "settings"],
        lastActiveStep: null
      })
    })
  })

  describe("syncTourStateWithBackend", () => {
    it("sends put request with current status", async () => {
      const store = useOnboardingStore
      store.setState({ completedSections: ["catalog"] })
      
      vi.mocked(api.put).mockResolvedValueOnce({ ok: true } as any)

      await store.getState().syncTourStateWithBackend("workspace", 1)

      expect(api.put).toHaveBeenCalledWith(
        "/api/v1/users/test%40example.com/tour-status",
        expect.objectContaining({
          completed: false,
          completed_sections: ["catalog"],
          last_active_step: {
            section_id: "workspace",
            step_index: 1
          }
        })
      )
      expect(store.getState().tourStatus).toEqual({
        completed: false,
        completedSections: ["catalog"],
        lastActiveStep: {
          tourId: "workspace",
          stepIndex: 1
        }
      })
    })

    it("marks completed if all major tours are completed", async () => {
      const store = useOnboardingStore
      store.setState({ completedSections: ["catalog", "workspace"] })
      
      vi.mocked(api.put).mockResolvedValueOnce({ ok: true } as any)

      await store.getState().syncTourStateWithBackend("settings", 0, true)

      expect(api.put).toHaveBeenCalledWith(
        "/api/v1/users/test%40example.com/tour-status",
        expect.objectContaining({
          completed: true,
          completed_sections: ["catalog", "workspace", "settings"]
        })
      )
      expect(store.getState().tourStatus).toEqual({
        completed: true,
        completedSections: ["catalog", "workspace", "settings"],
        lastActiveStep: null
      })
    })
  })

  describe("advanceStep", () => {
    it("advances active tour if matches", () => {
      const store = useOnboardingStore
      store.setState({
        activeTour: "catalog",
        currentStep: 1
      })
      
      const spyNext = vi.spyOn(store.getState(), "nextStep")
      
      store.getState().advanceStep("catalog", 1)
      
      expect(spyNext).toHaveBeenCalled()
      spyNext.mockRestore()
    })

    it("advances background step if matches lastActiveStep and not active", () => {
      const store = useOnboardingStore
      store.setState({
        activeTour: null,
        lastActiveStep: {
          tourId: "catalog",
          stepIndex: 1
        }
      })
      
      vi.mocked(api.put).mockResolvedValueOnce({ ok: true } as any)

      store.getState().advanceStep("catalog", 1)
      
      expect(store.getState().lastActiveStep).toEqual({
        tourId: "catalog",
        stepIndex: 2
      })
      expect(api.put).toHaveBeenCalled()
    })

    it("does nothing if doesn't match", () => {
      const store = useOnboardingStore
      store.setState({
        activeTour: "catalog",
        currentStep: 1
      })
      const spyNext = vi.spyOn(store.getState(), "nextStep")
      
      store.getState().advanceStep("catalog", 2)
      
      expect(spyNext).not.toHaveBeenCalled()
      spyNext.mockRestore()
    })
  })
})
