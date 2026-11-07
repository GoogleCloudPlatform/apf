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

import { useState } from "react"
import { useRouter } from "next/navigation"
import { useAppStore } from "@/stores/app"
import { useOnboardingStore } from "@/stores/onboarding"
import { api } from "@/lib/api"
import { 
  Folder, 
  Code, 
  Terminal, 
  Wrench, 
  FileText, 
  Edit3, 
  ArrowUpRight,
  ShieldAlert,
  Unplug,
  Share2,
  Loader2
} from "lucide-react"
import { 
  Project, 
  WorkflowTemplate, 
  getPhaseStatus
} from "@/lib/types"
import { ShareProjectModal } from "./ShareProjectModal"

interface ProjectCardProps {
  project: Project
  bmadWorkflow: WorkflowTemplate
  isBrowse?: boolean
  onRequestAccess?: (projectId: string) => Promise<void>
}

// Icon mappings matching Flutter's availableIcons structure
const iconMap: Record<string, React.ComponentType<{ className?: string }>> = {
  folder: Folder,
  code: Code,
  build: Wrench,
  terminal: Terminal,
  source: FileText,
}

export function ProjectCard({
  project,
  bmadWorkflow,
  isBrowse = false,
  onRequestAccess,
}: ProjectCardProps) {
  const router = useRouter()
  const { setLastSelectedProjectId } = useAppStore()
  const [isShareOpen, setIsShareOpen] = useState(false)
  const [isProvisioning, setIsProvisioning] = useState(false)

  const isShowcase = project.id === "foundry-showcase-project" || project.id.startsWith("showcase-") || project.name === "Foundry Showcase"
  const statusNormalized = (project.status || "New").toLowerCase()
  const LegacyIcon = project.icon ? iconMap[project.icon] : null

  const handleLaunch = async () => {
    if (isBrowse) {
      if (isShowcase) {
        try {
          setIsProvisioning(true)
          await onRequestAccess?.(project.id)
        } finally {
          setIsProvisioning(false)
        }
      }
      return
    }
    setLastSelectedProjectId(project.id)

    useOnboardingStore.getState().advanceStep("catalog", 17)
    
    if (project.status === "New") {
      try {
        await api.put(`/api/v1/projects/${project.id}`, { status: "Onboarding" })
        router.push(`/projects/${project.id}/workspace?phase=ONBOARDING`)
      } catch (err) {
        console.error("Failed to update status to Onboarding:", err)
        router.push(`/projects/${project.id}/workspace?phase=ONBOARDING`)
      }
    } else if (project.status === "Onboarding") {
      router.push(`/projects/${project.id}/workspace?phase=ONBOARDING`)
    } else {
      router.push(`/projects/${project.id}/workspace`)
    }
  }

  const handleEdit = (e: React.MouseEvent) => {
    e.stopPropagation()
    router.push(`/projects/${project.id}/details`)
  }

  // Get status badge styling
  const getBadgeStyle = () => {
    switch (project.status) {
      case "Onboarded":
        return "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20"
      case "Onboarding":
        return "bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/20"
      case "New":
      default:
        return "bg-blue-500/10 text-blue-600 dark:text-blue-400 border-blue-500/20"
    }
  }

  return (
    <div 
      onClick={handleLaunch}
      data-testid={isShowcase ? "showcase-project-card" : "project-card"}
      className="flex flex-col rounded-xl border border-border bg-card text-card-foreground shadow-sm overflow-hidden min-h-[225px] hover:border-muted-foreground/20 active:scale-[0.98] transition-all duration-200 tour-project-card cursor-pointer touch-manipulation w-full"
    >
      
      {/* Upper header section */}
      <div className="p-4 sm:p-5 flex items-start gap-3 sm:gap-4 border-b border-border bg-muted/20">
        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-xl select-none text-primary">
          {LegacyIcon ? (
            <LegacyIcon className="h-5 w-5" />
          ) : (
            project.icon || "🚀"
          )}
        </div>
        
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-1.5 flex-wrap">
            <h3 className="text-sm font-semibold tracking-tight text-foreground truncate">
              {project.name}
            </h3>
            {project.isReadOnly && (
              <span title="Read-only repository access" className="flex shrink-0">
                <Unplug className="h-4 w-4 text-muted-foreground" />
              </span>
            )}
            {/* Status Badge with test ID */}
            <span
              data-testid={`project-card-status-${statusNormalized}`}
              className={`inline-flex items-center rounded-md px-1.5 py-0.5 text-[10px] font-semibold border select-none ${getBadgeStyle()}`}
            >
              {project.status || "New"}
            </span>
            {isShowcase && (
              <span className="inline-flex items-center rounded-md bg-blue-50 dark:bg-blue-900/30 px-1.5 py-0.5 text-[9px] font-medium text-blue-700 dark:text-blue-300 ring-1 ring-inset ring-blue-700/10 shrink-0 select-none">
                Sample Project
              </span>
            )}
          </div>
          <span className="font-mono text-[10px] text-muted-foreground block mt-0.5">
            ID: {project.id}
          </span>
        </div>

        {!isBrowse && (
          <div className="flex items-center gap-1 shrink-0">
            <button
              onClick={(e) => {
                e.stopPropagation()
                setIsShareOpen(true)
              }}
              className="flex h-10 w-10 min-h-[44px] min-w-[44px] items-center justify-center rounded-lg text-muted-foreground hover:bg-muted hover:text-foreground transition-colors touch-manipulation"
              title="Share Project"
            >
              <Share2 className="h-4 w-4" />
            </button>
            <button
              onClick={handleEdit}
              className="flex h-10 w-10 min-h-[44px] min-w-[44px] items-center justify-center rounded-lg text-muted-foreground hover:bg-muted hover:text-foreground transition-colors touch-manipulation"
              title="Edit Project Configuration"
            >
              <Edit3 className="h-4 w-4" />
            </button>
          </div>
        )}
      </div>

      {/* Description & actions body section */}
      <div className="flex-1 p-4 sm:p-5 flex flex-col justify-between gap-4">
        <p className="text-xs text-muted-foreground line-clamp-3 leading-relaxed">
          {project.description || "No project description provided."}
        </p>

        <div className="space-y-4">
          {/* Real-time Governance Gates Progress indicators */}
          {!isBrowse && bmadWorkflow.id && (
            <div className="flex items-center justify-end gap-1.5" title="Governance Gates (Onboarding, Analysis, Planning, Solutioning)">
              {["ONBOARDING", "ANALYSIS", "PLANNING", "SOLUTIONING"].map((phase) => {
                const status = getPhaseStatus(bmadWorkflow, phase, project.artifacts)
                let colorClass = "bg-muted/40 border-muted-foreground/10" // Default (required-unstarted)

                if (status === "complete") {
                  colorClass = "bg-green-500 border-green-600 shadow-[0_0_4px_rgba(34,197,94,0.4)]"
                } else if (status === "in-progress") {
                  colorClass = "bg-amber-500 border-amber-600 shadow-[0_0_4px_rgba(245,158,11,0.4)]"
                } else if (status === "optional-unstarted") {
                  colorClass = "bg-yellow-500 border-yellow-600 shadow-[0_0_4px_rgba(234,179,8,0.4)]"
                }

                return (
                  <span
                    key={phase}
                    className={`h-2 w-5 rounded border transition-colors duration-300 ${colorClass}`}
                  />
                )
              })}
            </div>
          )}

          {/* Action Call Controls Button */}
          <div className="flex items-center justify-end">
            {isBrowse ? (
              <button
                onClick={async (e) => {
                  e.stopPropagation()
                  try {
                    setIsProvisioning(true)
                    await onRequestAccess?.(project.id)
                  } finally {
                    setIsProvisioning(false)
                  }
                }}
                data-testid="project-card-primary-cta"
                disabled={isProvisioning}
                className="inline-flex min-h-[48px] min-w-[48px] items-center justify-center gap-1.5 rounded-lg bg-primary/10 px-4 py-2 text-xs font-semibold text-primary hover:bg-primary/20 transition-all touch-manipulation disabled:opacity-50"
              >
                {isProvisioning && (
                  <Loader2 className="h-4 w-4 animate-spin text-primary" data-testid="showcase-loading-spinner" />
                )}
                {isShowcase
                  ? "Provision Showcase"
                  : "Request Access"}
                {!isProvisioning && <ArrowUpRight className="h-3.5 w-3.5" />}
              </button>
            ) : (
              <button
                onClick={handleLaunch}
                data-testid="project-card-primary-cta"
                className="inline-flex min-h-[48px] min-w-[48px] items-center justify-center rounded-lg border border-input bg-background px-4 py-2 text-xs font-semibold text-foreground hover:bg-muted active:bg-muted/80 transition-all tour-project-launch-button touch-manipulation"
              >
                {project.status === "New" ? "Onboard" : project.status === "Onboarding" ? "Onboard" : "Open Workspace"}
              </button>
            )}
          </div>
        </div>
      </div>
      <ShareProjectModal
        isOpen={isShareOpen}
        onClose={() => setIsShareOpen(false)}
        projectName={project.name}
        projectId={project.id}
        isTiedToRepo={project.app_integrations?.some(
          (i) => i.integration_id === "github" && i.enabled && i.settings?.repoUrl
        )}
      />
    </div>
  )
}
export default ProjectCard
