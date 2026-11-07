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

import React, { useState, useEffect, useCallback, useRef, useLayoutEffect } from "react"
import ProjectIdentity from "@/components/project/ProjectIdentity"
import { 
  ArrowLeft, 
  Activity, 
  Share2, 
  Edit, 
  Loader2, 
  Lock, 
  Unlock,
  ChevronDown,
  Check
} from "lucide-react"
import { cn } from "@/lib/utils"
import { 
  WorkflowTemplate, 
  Artifact 
} from "@/lib/types"
import { 
  PhaseType, 
  phaseConfig, 
  getPhaseStage 
} from "../_utils/workspaceHelpers"
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from "@/components/ui/dropdown-menu"

interface WorkspaceHeaderProps {
  projectName: string
  projectIcon: string
  projectId: string
  projectStatus?: string
  activePhase: PhaseType
  setActivePhase: (phase: PhaseType) => void
  satisfiedPhases: Record<string, boolean>
  bmadWorkflow: WorkflowTemplate
  liveArtifacts: Artifact[]
  runHealthCheckPending: boolean
  onRunHealthCheck: () => void
  onShareProject: () => void
  onEditProject: () => void
  handleBack: () => void
}

export default function WorkspaceHeader({
  projectName,
  projectIcon,
  projectId,
  projectStatus = "Onboarded",
  activePhase,
  setActivePhase,
  satisfiedPhases,
  bmadWorkflow,
  liveArtifacts,
  runHealthCheckPending,
  onRunHealthCheck,
  onShareProject,
  onEditProject,
  handleBack,
}: WorkspaceHeaderProps) {
  const [isCollapsed, setIsCollapsed] = useState(false)
  const headerRef = useRef<HTMLElement>(null)
  const housekeepingRef = useRef<HTMLButtonElement>(null)
  const hiddenNavRef = useRef<HTMLElement>(null)
  const hiddenImplTabRef = useRef<HTMLButtonElement>(null)

  const phasesList: PhaseType[] = ["ONBOARDING", "ANALYSIS", "PLANNING", "SOLUTIONING", "IMPLEMENTATION"]

  const checkCollision = useCallback(() => {
    if (!hiddenImplTabRef.current || !housekeepingRef.current) return
    const implRight = hiddenImplTabRef.current.getBoundingClientRect().right
    const housekeepingLeft = housekeepingRef.current.getBoundingClientRect().left
    if (implRight === 0 && housekeepingLeft === 0) {
      setIsCollapsed(false)
      return
    }
    setIsCollapsed(implRight >= housekeepingLeft - 8)
  }, [])

  useLayoutEffect(() => {
    checkCollision()
  }, [checkCollision, liveArtifacts, activePhase])

  useEffect(() => {
    checkCollision()

    const rafId = requestAnimationFrame(() => {
      checkCollision()
    })

    if (typeof document !== "undefined" && "fonts" in document) {
      document.fonts.ready.then(() => {
        checkCollision()
      })
    }

    let resizeObserver: ResizeObserver | null = null
    if (typeof ResizeObserver !== "undefined") {
      resizeObserver = new ResizeObserver(() => {
        checkCollision()
      })
      if (headerRef.current) {
        resizeObserver.observe(headerRef.current)
      }
      if (hiddenNavRef.current) {
        resizeObserver.observe(hiddenNavRef.current)
      }
    }

    window.addEventListener("resize", checkCollision)

    return () => {
      cancelAnimationFrame(rafId)
      if (resizeObserver) {
        resizeObserver.disconnect()
      }
      window.removeEventListener("resize", checkCollision)
    }
  }, [checkCollision])

  const renderTabButton = (phase: PhaseType, isHiddenMeasure = false) => {
    const isActive = activePhase === phase
    const stage = getPhaseStage(phase, satisfiedPhases, bmadWorkflow, liveArtifacts)
    const isLocked = stage === "locked"
    
    const inProgressCount = liveArtifacts.filter(a => a.phase === phase && a.status === "In Progress").length
    const readyForReviewCount = liveArtifacts.filter(a => a.phase === phase && a.status === "Staged for Review").length
    const completedCount = liveArtifacts.filter(a => a.phase === phase && a.status === "Complete").length
    const hasNoStartedArtifacts = liveArtifacts.filter(a => a.phase === phase && a.status !== "Not Started").length === 0

    const config = phaseConfig[phase]
    const PhaseIcon = config.icon

    return (
      <button
        key={phase}
        ref={isHiddenMeasure && phase === "IMPLEMENTATION" ? hiddenImplTabRef : undefined}
        id={!isHiddenMeasure ? `tour-workspace-phase-${phase.toLowerCase()}` : undefined}
        onClick={() => {
          setActivePhase(phase)
        }}
        className={cn(
          "relative h-full px-1 text-sm font-bold tracking-wider transition-colors flex items-center gap-2 border-b-4 border-transparent rounded-none shrink-0",
          isActive
            ? "border-primary text-primary"
            : "text-muted-foreground hover:text-foreground hover:border-muted-foreground/20",
          stage === "complete" && !isActive && "text-green-600 dark:text-green-400"
        )}
        title={isLocked ? "This phase is locked because of missing prerequisites" : undefined}
      >
        <PhaseIcon className={cn("h-4 w-4 shrink-0", isActive ? "text-primary" : "text-muted-foreground/60")} />
        <span>
          {config.label}
        </span>

        {isLocked ? (
          <Lock className="h-4 w-4 text-red-500 dark:text-red-400 shrink-0" />
        ) : (
          hasNoStartedArtifacts ? (
            <span title="Ready to start">
              <Unlock className="h-4 w-4 text-green-500 shrink-0" />
            </span>
          ) : (
            <div className="flex gap-1 items-center shrink-0">
              {inProgressCount > 0 && (
                <span 
                  className="inline-flex items-center justify-center h-5 min-w-5 rounded px-1 text-[11px] font-bold leading-none bg-amber-500 text-white" 
                  title={`${inProgressCount} In Progress`}
                >
                  {inProgressCount}
                </span>
              )}
              {readyForReviewCount > 0 && (
                <span 
                  className="inline-flex items-center justify-center h-5 min-w-5 rounded px-1 text-[11px] font-bold leading-none bg-blue-500 text-white" 
                  title={`${readyForReviewCount} Ready for Review`}
                >
                  {readyForReviewCount}
                </span>
              )}
              {completedCount > 0 && (
                <span 
                  className="inline-flex items-center justify-center h-5 min-w-5 rounded px-1 text-[11px] font-bold leading-none bg-green-500 text-white" 
                  title={`${completedCount} Completed`}
                >
                  {completedCount}
                </span>
              )}
            </div>
          )
        )}
      </button>
    )
  }

  const currentConfig = phaseConfig[activePhase]
  const CurrentIcon = currentConfig.icon
  const currentStage = getPhaseStage(activePhase, satisfiedPhases, bmadWorkflow, liveArtifacts)
  const isCurrentLocked = currentStage === "locked"
  const currentInProgressCount = liveArtifacts.filter(a => a.phase === activePhase && a.status === "In Progress").length
  const currentReadyForReviewCount = liveArtifacts.filter(a => a.phase === activePhase && a.status === "Staged for Review").length
  const currentCompletedCount = liveArtifacts.filter(a => a.phase === activePhase && a.status === "Complete").length

  return (
    <header ref={headerRef} className="h-16 px-5 border-b border-border bg-card flex items-center justify-between shrink-0 select-none z-10 relative">
      <div className="flex items-stretch gap-4 min-w-0 flex-1 h-full">
        <button
          onClick={handleBack}
          className="flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground hover:bg-muted border border-input bg-background my-auto shrink-0"
          title="Back to Projects list"
        >
          <ArrowLeft className="h-4 w-4" />
        </button>

        {/* Emojis logo and title segment */}
        <ProjectIdentity
          icon={projectIcon}
          name={projectName}
          id={projectId}
          size="sm"
          className="my-auto shrink-0"
        />

        {/* Vertical boundary spacer */}
        <span className="h-5 w-px bg-border my-auto mx-2 shrink-0" />

        {/* Relative container for accurate hidden and visible tabs positioning */}
        <div className="flex items-stretch h-full relative min-w-0 flex-1">
          {/* Offscreen hidden measurement container aligned with exact tabs start position */}
          <nav
            ref={hiddenNavRef}
            aria-hidden="true"
            className="opacity-0 pointer-events-none absolute left-0 top-0 flex items-stretch gap-6 h-full -z-50"
            style={{ visibility: "hidden" }}
          >
            {phasesList.map((phase) => renderTabButton(phase, true))}
          </nav>

          {/* Stages navigation */}
          <nav id="tour-workspace-phases" className="flex items-center h-full">
            {isCollapsed ? (
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <button
                    className="flex items-center gap-2 h-9 px-3 my-auto rounded-lg border border-input bg-background text-sm font-bold tracking-wide text-foreground hover:bg-muted focus:outline-none shrink-0"
                  >
                    <CurrentIcon className="h-4 w-4 text-primary shrink-0" />
                    <span>{currentConfig.label}</span>
                    {isCurrentLocked ? (
                      <Lock className="h-3.5 w-3.5 text-red-500 shrink-0" />
                    ) : (
                      <div className="flex gap-1 items-center shrink-0">
                        {currentInProgressCount > 0 && (
                          <span className="inline-flex items-center justify-center h-4 min-w-4 rounded px-1 text-[10px] font-bold bg-amber-500 text-white">
                            {currentInProgressCount}
                          </span>
                        )}
                        {currentReadyForReviewCount > 0 && (
                          <span className="inline-flex items-center justify-center h-4 min-w-4 rounded px-1 text-[10px] font-bold bg-blue-500 text-white">
                            {currentReadyForReviewCount}
                          </span>
                        )}
                        {currentCompletedCount > 0 && (
                          <span className="inline-flex items-center justify-center h-4 min-w-4 rounded px-1 text-[10px] font-bold bg-green-500 text-white">
                            {currentCompletedCount}
                          </span>
                        )}
                      </div>
                    )}
                    <ChevronDown className="h-4 w-4 text-muted-foreground ml-1 shrink-0" />
                  </button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="start" className="w-56">
                  {phasesList.map((phase) => {
                    const isSelected = activePhase === phase
                    const stage = getPhaseStage(phase, satisfiedPhases, bmadWorkflow, liveArtifacts)
                    const isLocked = stage === "locked"
                    const config = phaseConfig[phase]
                    const Icon = config.icon

                    const inProg = liveArtifacts.filter(a => a.phase === phase && a.status === "In Progress").length
                    const readyRev = liveArtifacts.filter(a => a.phase === phase && a.status === "Staged for Review").length
                    const comp = liveArtifacts.filter(a => a.phase === phase && a.status === "Complete").length

                    return (
                      <DropdownMenuItem
                        key={phase}
                        onClick={() => setActivePhase(phase)}
                        className={cn("flex items-center justify-between py-2 cursor-pointer", isSelected && "font-bold bg-accent")}
                      >
                        <div className="flex items-center gap-2 min-w-0">
                          <Icon className={cn("h-4 w-4 shrink-0", isSelected ? "text-primary" : "text-muted-foreground")} />
                          <span className="truncate">{config.label}</span>
                        </div>

                        <div className="flex items-center gap-1.5 shrink-0 ml-2">
                          {isLocked ? (
                            <Lock className="h-3.5 w-3.5 text-red-500 shrink-0" />
                          ) : (
                            <div className="flex gap-1 items-center shrink-0">
                              {inProg > 0 && (
                                <span className="inline-flex items-center justify-center h-4 min-w-4 rounded px-1 text-[10px] font-bold bg-amber-500 text-white">
                                  {inProg}
                                </span>
                              )}
                              {readyRev > 0 && (
                                <span className="inline-flex items-center justify-center h-4 min-w-4 rounded px-1 text-[10px] font-bold bg-blue-500 text-white">
                                  {readyRev}
                                </span>
                              )}
                              {comp > 0 && (
                                <span className="inline-flex items-center justify-center h-4 min-w-4 rounded px-1 text-[10px] font-bold bg-green-500 text-white">
                                  {comp}
                                </span>
                              )}
                            </div>
                          )}
                          {isSelected && <Check className="h-4 w-4 text-primary ml-1 shrink-0" />}
                        </div>
                      </DropdownMenuItem>
                    )
                  })}
                </DropdownMenuContent>
              </DropdownMenu>
            ) : (
              <div className="flex items-stretch gap-6 h-full">
                {phasesList.map((phase) => renderTabButton(phase, false))}
              </div>
            )}
          </nav>
        </div>
      </div>

      {/* Top-right dashboard headers actions utilities */}
      <div className="flex items-center gap-2 shrink-0">
        <button
          id="tour-workspace-health"
          ref={housekeepingRef}
          onClick={onRunHealthCheck}
          disabled={runHealthCheckPending}
          className="inline-flex h-8 items-center justify-center gap-1.5 rounded-lg border border-input bg-background px-3 text-xs font-semibold text-foreground hover:bg-muted disabled:opacity-50"
          title="Run active dependency/gate checker diagnostics"
        >
          {runHealthCheckPending ? (
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
          ) : (
            <Activity className="h-3.5 w-3.5" />
          )}
          <span>Sync</span>
        </button>

        <button
          onClick={onShareProject}
          className="inline-flex h-8 items-center justify-center gap-1.5 rounded-lg border border-input bg-background px-3 text-xs font-semibold text-foreground hover:bg-muted"
          title="Share Project"
        >
          <Share2 className="h-3.5 w-3.5" />
          <span>Share</span>
        </button>

        <button
          onClick={onEditProject}
          className="inline-flex h-8 items-center justify-center gap-1.5 rounded-lg border border-input bg-background px-3 text-xs font-semibold text-foreground hover:bg-muted"
          title="Edit Project Details"
        >
          <Edit className="h-3.5 w-3.5" />
          <span>Edit</span>
        </button>
      </div>
    </header>
  )
}

