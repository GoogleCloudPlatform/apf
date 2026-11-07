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

import { create } from "zustand"
import { useAuthStore } from "./auth"
import { api } from "@/lib/api"
import tourSteps from "./tour-steps.json"

export interface LastActiveStep {
  tourId: TourType
  stepIndex: number
}

export interface TourStatus {
  completed: boolean
  completedSections: string[]
  lastActiveStep: LastActiveStep | null
}

export type TourType = "catalog" | "workspace" | "settings"

export interface TourStep {
  target: string // CSS Selector
  title: string
  description: string
  placement: "top" | "bottom" | "left" | "right" | "center"
  isAction?: boolean
  sectionId?: string
  sectionTitle?: string
}

interface OnboardingState {
  activeTour: TourType | null
  currentStep: number
  completedTours: string[]
  completedSections: string[]
  steps: Record<TourType, TourStep[]>
  lastActiveStep: LastActiveStep | null
  tourStatus: TourStatus | null
  
  // Actions
  startTour: (tour: TourType, startingStep?: number) => void
  nextStep: () => void
  prevStep: () => void
  endTour: () => void
  completeTour: (tour: TourType) => void
  skipSection: () => void
  skipTour: () => void
  loadTourStatusFromBackend: (userId: string) => Promise<void>
  syncTourStateWithBackend: (tourId: TourType, stepIndex: number, completed?: boolean) => Promise<void>
  getSectionFirstStepIndex: (tour: TourType, sectionId: string) => number
  advanceStep: (tourId: TourType, expectedStepIndex: number) => void
  getTourBySectionId: (sectionId: string) => TourType | null
  getCurrentSectionInfo: () => {
    sectionTitle: string
    sectionIndex: number
    totalSections: number
    stepIndex: number
    totalSteps: number
  } | null
}

// Client-safe helper to get completed tours from localStorage
const getSavedCompletedTours = (): string[] => {
  if (typeof window === "undefined") return []
  try {
    const saved = localStorage.getItem("apf_completed_tours")
    const parsed = saved ? JSON.parse(saved) : []
    return Array.isArray(parsed) ? parsed : []
  } catch (e) {
    return []
  }
}

// Client-safe helper to save completed tours to localStorage
const saveCompletedTours = (tours: string[]) => {
  if (typeof window === "undefined") return
  try {
    localStorage.setItem("apf_completed_tours", JSON.stringify(tours))
  } catch (e) {
    // Ignore storage errors
  }
}

// Client-safe helper to get completed sections from localStorage
const getSavedCompletedSections = (): string[] => {
  if (typeof window === "undefined") return []
  try {
    const saved = localStorage.getItem("apf_completed_sections")
    const parsed = saved ? JSON.parse(saved) : []
    return Array.isArray(parsed) ? parsed : []
  } catch (e) {
    return []
  }
}

// Client-safe helper to save completed sections to localStorage
const saveCompletedSections = (sections: string[]) => {
  if (typeof window === "undefined") return
  try {
    localStorage.setItem("apf_completed_sections", JSON.stringify(sections))
  } catch (e) {
    // Ignore storage errors
  }
}

// Client-safe helper to get initial tour status from localStorage
const getInitialTourStatus = (): TourStatus => {
  const completedSections = getSavedCompletedSections()
  const completed = ["catalog", "workspace", "settings"].every(t => completedSections.includes(t))
  return {
    completed,
    completedSections,
    lastActiveStep: null
  }
}

export const useOnboardingStore = create<OnboardingState>((set, get) => ({
  activeTour: null,
  currentStep: 0,
  completedTours: getSavedCompletedTours(),
  completedSections: getSavedCompletedSections(),
  lastActiveStep: null,
  tourStatus: getInitialTourStatus(),
  
  steps: {
    catalog: tourSteps.catalog as TourStep[],
    workspace: tourSteps.workspace as TourStep[],
    settings: (tourSteps.catalog as TourStep[]).filter(
      (step) => step.sectionId === "settings-integration"
    ),
  },

  startTour: (tour, startingStep = 0) => {
    const validStep = Math.max(0, Math.min(startingStep, (get().steps[tour]?.length || 1) - 1))
    set({ activeTour: tour, currentStep: validStep })
    get().syncTourStateWithBackend(tour, validStep)
  },

  nextStep: () => {
    const {
      activeTour,
      currentStep,
      steps,
      completedSections,
      completedTours,
      tourStatus,
    } = get()
    if (!activeTour) return

    const tourSteps = steps[activeTour]
    if (currentStep < tourSteps.length - 1) {
      const nextStepVal = currentStep + 1
      const currentStepObj = tourSteps[currentStep]
      const nextStepObj = tourSteps[nextStepVal]

      const currentSectionId = currentStepObj?.sectionId
      const nextSectionId = nextStepObj?.sectionId

      // Naturally mark section complete if transitioning to a new section
      if (currentSectionId && currentSectionId !== nextSectionId) {
        if (!completedSections.includes(currentSectionId)) {
          const updatedSections = [...completedSections, currentSectionId]
          set({ completedSections: updatedSections })
          saveCompletedSections(updatedSections)
        }
        if (tourStatus?.completed || completedTours.includes(activeTour)) {
          get().completeTour(activeTour)
          return
        }
      }
      
      set({ currentStep: nextStepVal })
      get().syncTourStateWithBackend(activeTour, nextStepVal)
    } else {
      // Reached the end of the tour, mark the last section complete
      const currentStepObj = tourSteps[currentStep]
      const currentSectionId = currentStepObj?.sectionId
      if (currentSectionId && !completedSections.includes(currentSectionId)) {
        const updatedSections = [...completedSections, currentSectionId]
        set({ completedSections: updatedSections })
        saveCompletedSections(updatedSections)
      }
      get().completeTour(activeTour)
    }
  },

  prevStep: () => {
    const { activeTour, currentStep } = get()
    if (!activeTour) return
    if (currentStep > 0) {
      const prevStepVal = currentStep - 1
      set({ currentStep: prevStepVal })
      get().syncTourStateWithBackend(activeTour, prevStepVal)
    }
  },

  endTour: () => {
    const { activeTour } = get()
    if (activeTour) {
      get().completeTour(activeTour)
    } else {
      set({ activeTour: null, currentStep: 0 })
    }
  },

  completeTour: (tour) => {
    const completed = get().completedTours
    if (!completed.includes(tour)) {
      const updated = [...completed, tour]
      set({ completedTours: updated, activeTour: null, currentStep: 0 })
      saveCompletedTours(updated)
      get().syncTourStateWithBackend(tour, 0, true)
    } else {
      set({ activeTour: null, currentStep: 0 })
      get().syncTourStateWithBackend(tour, 0, true)
    }
  },

  skipSection: () => {
    const { activeTour, currentStep, steps, completedSections } = get()
    if (!activeTour) return

    const tourSteps = steps[activeTour]
    const currentStepObj = tourSteps[currentStep]
    const currentSectionId = currentStepObj?.sectionId

    // Mark current section as completed if it exists
    let updatedCompletedSections = completedSections
    if (currentSectionId) {
      if (!completedSections.includes(currentSectionId)) {
        updatedCompletedSections = [...completedSections, currentSectionId]
        set({ completedSections: updatedCompletedSections })
        saveCompletedSections(updatedCompletedSections)
      }
    }

    // Find the next step that has a different sectionId (or undefined)
    let nextStepIndex = -1
    for (let i = currentStep + 1; i < tourSteps.length; i++) {
      if (tourSteps[i].sectionId !== currentSectionId) {
        nextStepIndex = i
        break
      }
    }

    if (nextStepIndex !== -1) {
      set({ currentStep: nextStepIndex })
      get().syncTourStateWithBackend(activeTour, nextStepIndex)
    } else {
      // No more sections, complete the tour
      get().completeTour(activeTour)
    }
  },

  skipTour: () => {
    const { activeTour } = get()
    if (!activeTour) return
    get().completeTour(activeTour)
  },

  loadTourStatusFromBackend: async (userId: string) => {
    if (!userId) return
    try {
      const res = await api.get(`/api/v1/users/${encodeURIComponent(userId)}/tour-status`)
      if (res.ok) {
        const data = await res.json()
        const completedSections = data.completed_sections || []
        const lastActiveStep = data.last_active_step ? {
          tourId: get().getTourBySectionId(data.last_active_step.section_id) || (data.last_active_step.section_id as TourType),
          stepIndex: data.last_active_step.step_index
        } : null

        const tourStatus: TourStatus = {
          completed: data.completed || false,
          completedSections,
          lastActiveStep
        }

        if (data.completed) {
          const allTours: TourType[] = ["catalog", "workspace", "settings"]
          set({
            completedTours: allTours,
            completedSections: data.completed_sections || allTours,
            lastActiveStep: null,
            activeTour: null,
            currentStep: 0,
            tourStatus: {
              completed: true,
              completedSections: data.completed_sections || allTours,
              lastActiveStep: null
            }
          })
          saveCompletedTours(allTours)
          saveCompletedSections(data.completed_sections || allTours)
          return
        }

        const completedTours = completedSections.filter((s: string) => ["catalog", "workspace", "settings"].includes(s))
        
        set({
          completedTours,
          completedSections,
          lastActiveStep,
          tourStatus
        })
        
        saveCompletedTours(completedTours)
        saveCompletedSections(completedSections)
      }
    } catch (e) {
      console.error("Failed to load tour status from backend:", e)
    }
  },

  syncTourStateWithBackend: async (tourId: TourType, stepIndex: number, completed?: boolean) => {
    const { email } = useAuthStore.getState()
    if (!email) return

    const { completedSections: prevSections, lastActiveStep: prevStep, tourStatus: prevStatus, steps } = get()
    
    const tourSteps = steps[tourId]
    const stepObj = tourSteps?.[stepIndex]
    const sectionId = stepObj?.sectionId || tourId

    let updatedCompletedSections = prevSections
    if (completed) {
      const tourSections = Array.from(new Set(tourSteps?.map(s => s.sectionId).filter(Boolean) as string[]))
      const newSections = tourSections.filter(s => !prevSections.includes(s))
      updatedCompletedSections = [...prevSections, ...newSections]
      if (!updatedCompletedSections.includes(tourId)) {
        updatedCompletedSections = [...updatedCompletedSections, tourId]
      }
    }

    const majorTours: TourType[] = ["catalog", "workspace", "settings"]
    const allCompleted = majorTours.every(t => updatedCompletedSections.includes(t))

    const updatedTourStatus: TourStatus = {
      completed: allCompleted,
      completedSections: updatedCompletedSections,
      lastActiveStep: completed ? null : {
        tourId: tourId,
        stepIndex: stepIndex
      }
    }

    // Optimistic Update
    set({
      completedSections: updatedCompletedSections,
      lastActiveStep: updatedTourStatus.lastActiveStep,
      tourStatus: updatedTourStatus
    })
    saveCompletedSections(updatedCompletedSections)

    try {
      const lastActiveStepPayload = completed ? null : {
        section_id: sectionId,
        step_index: stepIndex
      }
      const payload = {
        completed: allCompleted,
        completed_sections: updatedCompletedSections,
        last_active_step: lastActiveStepPayload
      }
      
      const res = await api.put(`/api/v1/users/${encodeURIComponent(email)}/tour-status`, payload)
      if (!res.ok) {
        throw new Error(`Server returned status ${res.status}: ${res.statusText}`)
      }
    } catch (e) {
      console.error("Failed to sync tour state with backend, rolling back:", e)
      
      // Rollback only if store state hasn't moved to a newer request (Bug 5)
      const currentStatus = get().tourStatus
      if (
        currentStatus?.lastActiveStep?.tourId === tourId &&
        currentStatus?.lastActiveStep?.stepIndex === stepIndex
      ) {
        set({
          completedSections: prevSections,
          lastActiveStep: prevStep,
          tourStatus: prevStatus
        })
        saveCompletedSections(prevSections)
      }
    }
  },

  getSectionFirstStepIndex: (tour: TourType, sectionId: string) => {
    const tourSteps = get().steps[tour]
    if (!tourSteps) return 0
    const index = tourSteps.findIndex(s => s.sectionId === sectionId)
    return index !== -1 ? index : 0
  },

  advanceStep: (tourId: TourType, expectedStepIndex: number) => {
    const { activeTour, currentStep, lastActiveStep, steps } = get()
    
    if (activeTour === tourId && currentStep === expectedStepIndex) {
      get().nextStep()
      return
    }
    
    const tourSteps = steps[tourId]

    if (!activeTour && lastActiveStep && lastActiveStep.tourId === tourId && lastActiveStep.stepIndex === expectedStepIndex) {
      const nextStepVal = expectedStepIndex + 1
      if (nextStepVal < (tourSteps?.length || 0)) {
        set({ lastActiveStep: { tourId: tourId, stepIndex: nextStepVal } })
        get().syncTourStateWithBackend(tourId, nextStepVal)
      } else {
        set({ lastActiveStep: null })
        get().completeTour(tourId)
      }
    }
  },

  getTourBySectionId: (sectionId) => {
    const { steps } = get()
    for (const tour of Object.keys(steps) as TourType[]) {
      if (steps[tour].some(step => step.sectionId === sectionId)) {
        return tour
      }
    }
    return null
  },

  getCurrentSectionInfo: () => {
    const { activeTour, currentStep, steps } = get()
    if (!activeTour) return null

    const tourSteps = steps[activeTour]
    if (!tourSteps || tourSteps.length === 0) return null

    const sections: { id: string | undefined; title: string; steps: number[] }[] = []
    
    tourSteps.forEach((step, idx) => {
      const sectionId = step.sectionId
      const sectionTitle = step.sectionTitle || "Overview"
      
      const lastSection = sections[sections.length - 1]
      if (lastSection && lastSection.id === sectionId) {
        lastSection.steps.push(idx)
      } else {
        sections.push({
          id: sectionId,
          title: sectionTitle,
          steps: [idx]
        })
      }
    })

    const currentSectionIdx = sections.findIndex(sec => sec.steps.includes(currentStep))
    if (currentSectionIdx === -1) return null

    const currentSection = sections[currentSectionIdx]
    const stepIndex = currentSection.steps.indexOf(currentStep) + 1

    return {
      sectionTitle: currentSection.title,
      sectionIndex: currentSectionIdx + 1,
      totalSections: sections.length,
      stepIndex,
      totalSteps: currentSection.steps.length
    }
  }
}))
