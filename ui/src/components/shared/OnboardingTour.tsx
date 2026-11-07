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

"use client"

import React, { useEffect, useState, useRef } from "react"
import { usePathname, useRouter, useSearchParams, useParams } from "next/navigation"
import { X, ArrowLeft, ArrowRight, Check, Sparkles } from "lucide-react"
import { useOnboardingStore, TourStep, TourType } from "@/stores/onboarding"
import { useAppStore } from "@/stores/app"
import { cn } from "@/lib/utils"
import { toast } from "sonner"
import { useQuery } from "@tanstack/react-query"
import { Project } from "@/lib/types"
import { api } from "@/lib/api"

export default function OnboardingTour() {
  const pathname = usePathname()
  const router = useRouter()
  const searchParams = useSearchParams()
  const params = useParams()
  const { lastSelectedProjectId } = useAppStore()

  const rawProjectId = (params?.id as string) || null
  const isWorkflowRoute = pathname?.startsWith("/workflows")
  const projectId = isWorkflowRoute ? null : (rawProjectId && rawProjectId.trim() !== "" ? rawProjectId : null)
  const isValidProjectId = projectId && projectId !== "new" && projectId !== "create" && /^[a-zA-Z0-9_-]+$/.test(projectId)

  const { data: project, error: projectQueryError } = useQuery<Project>({
    queryKey: ["project", projectId],
    queryFn: async () => {
      if (!projectId) throw new Error("No project ID")
      try {
        const res = await api.get(`/api/v1/projects/${projectId}`)
        if (!res.ok) throw new Error(`Failed to load project details: ${res.statusText}`)
        return await res.json()
      } catch (err) {
        console.error("Error fetching project details in OnboardingTour:", err)
        throw err
      }
    },
    enabled: !!isValidProjectId
  })

  useEffect(() => {
    if (projectQueryError) {
      toast.error("Failed to load project details for onboarding tour")
    }
  }, [projectQueryError])

  const {
    activeTour,
    currentStep,
    steps,
    nextStep,
    prevStep,
    endTour,
    completedTours,
    startTour,
    lastActiveStep,
    skipSection,
    skipTour,
    getCurrentSectionInfo,
    tourStatus,
    completedSections,
    getSectionFirstStepIndex,
    getTourBySectionId
  } = useOnboardingStore()

  const sectionInfo = getCurrentSectionInfo()

  const [targetRect, setTargetRect] = useState<DOMRect | null>(null)
  const [bubbleStyle, setBubbleStyle] = useState<React.CSSProperties>({})
  const [arrowStyle, setArrowStyle] = useState<React.CSSProperties>({})
  const bubbleRef = useRef<HTMLDivElement>(null)
  const [isVisible, setIsVisible] = useState(false)

  const activeStepsList = activeTour ? steps[activeTour] : []
  const currentStepData: TourStep | undefined = activeStepsList[currentStep]

  const completedSectionsJoined = completedSections.join(",")
  const completedToursJoined = completedTours.join(",")
  const projectStatus = project?.status
  const projectIdVal = project?.id
  const tourCompleted = tourStatus?.completed
  const lastActiveStepTourId = lastActiveStep?.tourId
  const lastActiveStepIndex = lastActiveStep?.stepIndex

  // Auto-trigger tour on page load if not completed yet
  useEffect(() => {
    if (activeTour) return
    if (tourCompleted) return

    const isCatalogPage = pathname === "/projects" || pathname === "/projects/"
    const isWorkspacePage = pathname.includes("/workspace")
    const isSettingsPage = pathname === "/settings" || pathname === "/settings/"
    const urlPhase = searchParams.get("phase")
    const phase = urlPhase || (projectStatus ? (projectStatus === "Onboarded" ? "ANALYSIS" : "ONBOARDING") : null)

    let targetTour: TourType | null = null
    let targetSection: string | null = null

    if (isCatalogPage) {
      targetTour = "catalog"
      targetSection = "catalog-overview"
    } else if (isSettingsPage) {
      targetTour = "settings"
      targetSection = "settings-overview"
    } else if (isWorkspacePage) {
      if (!phase) return // wait for project details if phase query param is missing
      targetTour = "workspace"
      targetSection = "workspace-overview"
    }

    if (!targetTour || !targetSection) return

    // Resume logic using granular sectionId page matching
    if (lastActiveStepTourId && lastActiveStepIndex !== undefined) {
      const step = steps[lastActiveStepTourId]?.[lastActiveStepIndex]
      const granularSectionId = step?.sectionId || lastActiveStepTourId

      const pageMatchesLastActive = 
        (granularSectionId === "catalog-overview" && isCatalogPage) ||
        (granularSectionId === "settings-integration" && isSettingsPage) ||
        (granularSectionId === "project-creation" && pathname.includes("/details")) ||
        (granularSectionId === "settings-overview" && isSettingsPage) ||
        (granularSectionId === "workspace-overview" && isWorkspacePage)

      if (pageMatchesLastActive) {
        const timer = setTimeout(() => {
          startTour(lastActiveStepTourId, lastActiveStepIndex)
        }, 1500)
        return () => clearTimeout(timer)
      }
    }

    // Auto-start logic if section not completed and parent tour not completed
    if (!completedSections.includes(targetSection) && !completedTours.includes(targetTour)) {
      const startStep = getSectionFirstStepIndex(targetTour, targetSection)
      const timer = setTimeout(() => {
        startTour(targetTour!, startStep)
      }, 1500)
      return () => clearTimeout(timer)
    }
  }, [
    pathname,
    completedSectionsJoined,
    completedToursJoined,
    activeTour,
    startTour,
    lastActiveStepTourId,
    lastActiveStepIndex,
    tourCompleted,
    searchParams,
    getSectionFirstStepIndex,
    projectStatus,
    projectIdVal,
    steps
  ])


  // Handle auto-advancing the tour based on pathname changes
  useEffect(() => {
    if (!activeTour || !currentStepData) return

    const isCatalogPage = pathname === "/projects" || pathname === "/projects/"
    const isSettingsPage = pathname === "/settings" || pathname === "/settings/"
    const isDetailsPage = pathname.includes("/details")

    const target = currentStepData.target

    if (target === "#tour-leftbar-settings" && isSettingsPage) {
      nextStep()
    } else if (target === "#tour-leftbar-projects" && isCatalogPage) {
      nextStep()
    } else if (target === "#tour-catalog-create" && isDetailsPage) {
      nextStep()
    } else if (target === "#tour-project-save-button" && isCatalogPage) {
      nextStep()
    }
  }, [pathname, activeTour, currentStepData, nextStep])

  // Handle reactive step behaviors (opening panels, expanding items)
  useEffect(() => {
    if (!activeTour || !currentStepData) return

    const selector = currentStepData.target

    // 1. Handle Debug console step: Auto-open Debug Panel
    if (selector === "#tour-workspace-debug") {
      const { isDebugBarOpen, setIsDebugBarOpen } = useAppStore.getState()
      if (!isDebugBarOpen) {
        setIsDebugBarOpen(true)
      }
    }

    // 2. Handle Canvas step: Auto-select first outcome artifact if closed
    if (selector === "#tour-workspace-canvas") {
      const canvasEl = document.querySelector("#tour-workspace-canvas")
      if (!canvasEl) {
        // Let's try to click the first folder and file to reveal the canvas!
        const folderHeader = document.querySelector("div[class*='FolderCard'] div") as HTMLElement | null
        if (folderHeader) {
          folderHeader.click()
        }
        setTimeout(() => {
          const fileNode = document.querySelector("div[class*='FileNode'] span") as HTMLElement | null
          if (fileNode) {
            fileNode.click()
          }
        }, 300)
      }
    }

    // Expand the Left Bar (if it is collapsed) before the step 2 (Global Navigation menu)
    if (selector === "#tour-leftbar") {
      const { isLeftBarExpanded, setIsLeftBarExpanded } = useAppStore.getState()
      if (!isLeftBarExpanded) {
        setIsLeftBarExpanded(true)
      }
    }

    // Collapse the Left Bar once on the settings page
    if (selector === "#tour-settings-nav") {
      const { isLeftBarExpanded, setIsLeftBarExpanded } = useAppStore.getState()
      if (isLeftBarExpanded) {
        setIsLeftBarExpanded(false)
      }
    }

    // 3. Start the GitHub section as collapsed when on settings screen
    if (selector === "#tour-settings-card-github" || selector === "#tour-settings-credentials") {
      const githubCard = document.querySelector("#tour-settings-card-github") as HTMLElement | null
      if (githubCard) {
        const isExpanded = githubCard.querySelector("div[class*='border-t']") !== null
        if (isExpanded) {
          const headerClickRow = githubCard.querySelector("div[class*='cursor-pointer']") as HTMLElement | null
          if (headerClickRow) {
            headerClickRow.click()
          }
        }
      }
    }

    // 5. Handle onboard step: Auto-switch back to "My Projects" tab to show onboard button
    if (selector === ".tour-project-launch-button") {
      const myProjectsTab = document.querySelector("#tour-tab-my-projects") as HTMLElement | null
      if (myProjectsTab) {
        myProjectsTab.click()
      }
    }
  }, [activeTour, currentStep, currentStepData, nextStep])


  // Recalculate positioning on step changes, scroll, resize, or page mutations
  useEffect(() => {
    if (!activeTour || !currentStepData) {
      setTargetRect(null)
      setIsVisible(false)
      return
    }

    const updatePosition = () => {
      // Special case: if target card is highlighted and catalog page is empty, target the fallback
      let selector = currentStepData.target
      let el = document.querySelector(selector) as HTMLElement | null

      if (!el && (selector === "#tour-projects-list" || selector === ".tour-project-card" || selector === ".tour-project-launch-button")) {
        // Fallback to the catalog header or page element if no cards exist
        el = document.querySelector("#tour-catalog-header")
      }

      if (el) {
        el.scrollIntoView({ behavior: "smooth", block: "center" })
        
        // Slight timeout to wait for smooth scrolling to finish
        setTimeout(() => {
          let rect = el!.getBoundingClientRect()
          
          if (selector === "#tour-project-github-repo") {
            const dropdownEl = document.querySelector("#tour-project-github-repo-dropdown") as HTMLElement | null
            if (dropdownEl) {
              const dropdownRect = dropdownEl.getBoundingClientRect()
              const left = Math.min(rect.left, dropdownRect.left)
              const top = Math.min(rect.top, dropdownRect.top)
              const right = Math.max(rect.right, dropdownRect.right)
              const bottom = Math.max(rect.bottom, dropdownRect.bottom)
              
              rect = new DOMRect(left, top, right - left, bottom - top)
            }
          }
          
          setTargetRect(rect)
          setIsVisible(true)
        }, 300)
      } else {
        // Target not found on this page (might be another page or hidden)
        setTargetRect(null)
        setIsVisible(false)
      }
    }

    updatePosition()
    window.addEventListener("resize", updatePosition)
    window.addEventListener("scroll", updatePosition, true)

    // Mutation observer to handle delayed rendering of elements
    const observer = new MutationObserver(updatePosition)
    observer.observe(document.body, { childList: true, subtree: true })

    return () => {
      window.removeEventListener("resize", updatePosition)
      window.removeEventListener("scroll", updatePosition, true)
      observer.disconnect()
    }
  }, [activeTour, currentStep, currentStepData, pathname])

  // Bubble placement logic
  useEffect(() => {
    if (!targetRect || !bubbleRef.current || !currentStepData) return

    const bubble = bubbleRef.current
    const bubbleRect = bubble.getBoundingClientRect()
    const placement = currentStepData.placement

    let top = 0
    let left = 0
    const margin = 16

    // Spotlight highlight position
    const targetTop = targetRect.top + window.scrollY
    const targetLeft = targetRect.left + window.scrollX
    const targetWidth = targetRect.width
    const targetHeight = targetRect.height

    let arrowTop = ""
    let arrowLeft = ""
    let arrowTransform = ""
    let arrowBorder = ""

    if (placement === "bottom") {
      top = targetTop + targetHeight + margin
      left = targetLeft + targetWidth / 2 - bubbleRect.width / 2
      arrowTop = `-${6}px`
      arrowLeft = "calc(50% - 6px)"
      arrowTransform = "rotate(45deg)"
    } else if (placement === "top") {
      top = targetTop - bubbleRect.height - margin
      left = targetLeft + targetWidth / 2 - bubbleRect.width / 2
      arrowTop = `calc(100% - 6px)`
      arrowLeft = "calc(50% - 6px)"
      arrowTransform = "rotate(45deg)"
    } else if (placement === "left") {
      top = targetTop + targetHeight / 2 - bubbleRect.height / 2
      left = targetLeft - bubbleRect.width - margin
      arrowTop = "calc(50% - 6px)"
      arrowLeft = `calc(100% - 6px)`
      arrowTransform = "rotate(45deg)"
    } else if (placement === "right") {
      top = targetTop + targetHeight / 2 - bubbleRect.height / 2
      left = targetLeft + targetWidth + margin
      arrowTop = "calc(50% - 6px)"
      arrowLeft = `-${6}px`
      arrowTransform = "rotate(45deg)"
    } else {
      // Center viewport fallback
      top = window.innerHeight / 2 - bubbleRect.height / 2
      left = window.innerWidth / 2 - bubbleRect.width / 2
    }

    // Boundary constraints (don't let bubble overflow viewport)
    const maxLeft = window.innerWidth - bubbleRect.width - margin
    const maxTop = window.innerHeight - bubbleRect.height - margin
    left = Math.max(margin, Math.min(left, maxLeft))
    top = Math.max(margin, Math.min(top, maxTop))

    setBubbleStyle({
      position: "absolute",
      top: `${top}px`,
      left: `${left}px`,
      zIndex: 10000
    })

    setArrowStyle({
      position: "absolute",
      top: arrowTop,
      left: arrowLeft,
      width: "12px",
      height: "12px",
      transform: arrowTransform,
      zIndex: -1
    })
  }, [targetRect, currentStep, currentStepData])

  const handleNext = () => {
    if (currentStepData?.target === "#tour-settings-card-github") {
      const targetIdx = activeStepsList.findIndex(s => s.target === "#tour-leftbar-projects")
      if (activeTour && targetIdx !== -1) {
        startTour(activeTour, targetIdx)
        return
      }
    }

    if (activeTour === "catalog") {
      // Step 3 (0-indexed 2): Click Settings button -> Go to Settings
      if (currentStep === 2) {
        router.push("/settings")
      }
      // Step 11 (0-indexed 10): Back to Projects list -> Go to Projects
      else if (currentStep === 10) {
        router.push("/projects")
      }
      // Step 12 (0-indexed 11): Create Project button -> Go to details
      else if (currentStep === 11) {
        router.push("/projects/new/details")
      }
      // Step 15 (0-indexed 14): Save Project -> Redirect to /projects
      else if (currentStep === 14) {
        router.push("/projects")
      }
      // Step 16 (0-indexed 15): Toggle to List View -> programmatically click the toggle
      else if (currentStep === 15) {
        const toggleBtn = document.querySelector("#tour-toggle-list-view") as HTMLElement | null
        if (toggleBtn) {
          toggleBtn.click()
        }
      }
      // Step 17 (0-indexed 16): Toggle Shared Projects -> programmatically click the tab
      else if (currentStep === 16) {
        const tabBtn = document.querySelector("#tour-tab-shared-projects") as HTMLElement | null
        if (tabBtn) {
          tabBtn.click()
        }
      }
      // Step 18 (0-indexed 17): Onboard button click -> route to workspace onboarding
      else if (currentStep === 17) {
        if (lastSelectedProjectId) {
          router.push(`/projects/${lastSelectedProjectId}/workspace?phase=ONBOARDING`)
        } else {
          console.warn("lastSelectedProjectId is missing in tour routing fallback.")
          toast.error("Could not find the target project workspace. Returning to catalog.")
          router.push("/projects")
        }
      }
    }
    nextStep()
  }

  if (!activeTour || !currentStepData || !isVisible) return null

  const isAlreadyCompleted =
    tourStatus?.completed || completedTours.includes(activeTour)

  const nextStepData = activeStepsList[currentStep + 1]
  const isLastStep =
    !nextStepData ||
    (!!isAlreadyCompleted &&
      !!currentStepData.sectionId &&
      nextStepData.sectionId !== currentStepData.sectionId)

  return (
    <div className="fixed inset-0 z-[9999] pointer-events-none select-none">
      {/* Semi-transparent Spotlight Backdrop */}
      {targetRect && (
        <div
          className="absolute inset-0 bg-black/45 dark:bg-black/65 pointer-events-auto transition-opacity duration-300 ease-in-out"
          style={{
            clipPath: `polygon(
              0% 0%, 
              0% 100%, 
              ${targetRect.left}px 100%, 
              ${targetRect.left}px ${targetRect.top}px, 
              ${targetRect.right}px ${targetRect.top}px, 
              ${targetRect.right}px ${targetRect.bottom}px, 
              ${targetRect.left}px ${targetRect.bottom}px, 
              ${targetRect.left}px 100%, 
              100% 100%, 
              100% 0%
            )`
          }}
          onClick={endTour}
        />
      )}

      {/* Spotlight Highlight Ring */}
      {targetRect && (
        <div
          className={cn(
            "absolute border-2 rounded-lg pointer-events-none transition-all duration-300 ease-out",
            currentStepData?.isAction 
              ? "border-purple-500/80 shadow-[0_0_15px_rgba(168,85,247,0.5)] animate-pulse" 
              : "border-primary/80 animate-pulse-slow"
          )}
          style={{
            top: `${targetRect.top + window.scrollY - 4}px`,
            left: `${targetRect.left + window.scrollX - 4}px`,
            width: `${targetRect.width + 8}px`,
            height: `${targetRect.height + 8}px`,
            zIndex: 9998
          }}
        />
      )}

      {/* Glassmorphic Walkthrough Bubble */}
      <div
        ref={bubbleRef}
        style={bubbleStyle}
        className="pointer-events-auto w-[330px] rounded-2xl p-5 bg-white/85 dark:bg-slate-950/90 border border-slate-200/50 dark:border-slate-800/70 backdrop-blur-lg shadow-[0_20px_50px_rgba(0,0,0,0.3)] text-slate-900 dark:text-slate-100 transition-all duration-300"
      >
        {/* Inner Arrow element */}
        <div
          style={arrowStyle}
          className="bg-white/85 dark:bg-slate-950/90 border-t border-l border-slate-200/50 dark:border-slate-800/70"
        />

        {/* Bubble Header */}
        <div className="flex items-start justify-between gap-4 mb-2.5">
          <h4 className="text-sm font-bold tracking-wide flex items-center gap-1.5 text-primary dark:text-purple-400">
            <Sparkles className="h-4 w-4 animate-spin-slow" />
            {currentStepData.title}
          </h4>
          <button
            onClick={endTour}
            className="p-1 rounded-full text-muted-foreground hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
          >
            <X className="h-3.5 w-3.5" />
          </button>
        </div>

        {/* Description */}
        <p className="text-xs text-slate-600 dark:text-slate-300 leading-relaxed mb-4">
          {currentStepData.description}
        </p>

        {/* Footer */}
        <div className="flex flex-col gap-3">
          {/* Step / Section indicators */}
          <div className="flex items-center justify-between text-[10px] font-mono font-semibold text-muted-foreground tracking-wider border-b border-slate-100 dark:border-slate-800/50 pb-2">
            {sectionInfo ? (
              <span className="truncate max-w-[180px]" title={sectionInfo.sectionTitle}>
                {sectionInfo.sectionTitle} ({sectionInfo.sectionIndex}/{sectionInfo.totalSections})
              </span>
            ) : (
              <span>Step {currentStep + 1} of {activeStepsList.length}</span>
            )}
            {sectionInfo && (
              <span>
                Step {sectionInfo.stepIndex} of {sectionInfo.totalSteps}
              </span>
            )}
          </div>

          {/* Navigation & Skip Controls */}
          <div className="flex items-center justify-between select-none">
            {/* Skip Actions */}
            <div className="flex items-center gap-1.5">
              {sectionInfo && sectionInfo.totalSections > 1 && (
                <button
                  onClick={skipSection}
                  className="inline-flex h-7 items-center justify-center rounded-md px-2 text-[10px] font-semibold border border-slate-200 dark:border-slate-800 hover:bg-slate-50 dark:hover:bg-slate-900 focus-visible:ring-2 focus-visible:ring-primary focus-visible:outline-none pointer-events-auto transition-colors"
                >
                  Skip Section
                </button>
              )}
              <button
                onClick={skipTour}
                className="inline-flex h-7 items-center justify-center rounded-md px-2 text-[10px] font-semibold border border-slate-200 dark:border-slate-800 hover:bg-slate-50 dark:hover:bg-slate-900 focus-visible:ring-2 focus-visible:ring-primary focus-visible:outline-none pointer-events-auto transition-colors"
              >
                Skip Tour
              </button>
            </div>

            {/* Next / Finish Action */}
            <button
              onClick={handleNext}
              disabled={currentStepData.isAction}
              className={cn(
                "inline-flex h-7 items-center justify-center gap-1 rounded-md px-3 text-[11px] font-bold shadow transition-all focus-visible:ring-2 focus-visible:ring-primary focus-visible:outline-none",
                currentStepData.isAction
                  ? "bg-slate-200 dark:bg-slate-800 text-slate-400 dark:text-slate-600 cursor-not-allowed opacity-60"
                  : "bg-primary text-primary-foreground hover:opacity-95"
              )}
              title={currentStepData.isAction ? "Click the highlighted button to proceed" : undefined}
            >
              {isLastStep ? (
                <>
                  Got it
                  <Check className="h-3 w-3" />
                </>
              ) : (
                <>
                  Next
                  <ArrowRight className="h-3 w-3" />
                </>
              )}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
