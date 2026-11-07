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
  Unplug,
  Share2
} from "lucide-react"
import { 
  Project, 
  WorkflowTemplate, 
  getPhaseStatus
} from "@/lib/types"
import { ShareProjectModal } from "./ShareProjectModal"

interface ProjectRowProps {
  project: Project
  bmadWorkflow: WorkflowTemplate
  isBrowse?: boolean
  onRequestAccess?: (projectId: string) => Promise<void>
}

const iconMap: Record<string, React.ComponentType<{ className?: string }>> = {
  folder: Folder,
  code: Code,
  build: Wrench,
  terminal: Terminal,
  source: FileText,
}

export function ProjectRow({
  project,
  bmadWorkflow,
  isBrowse = false,
  onRequestAccess,
}: ProjectRowProps) {
  const router = useRouter()
  const { setLastSelectedProjectId } = useAppStore()
  const [isShareOpen, setIsShareOpen] = useState(false)

  const LegacyIcon = project.icon ? iconMap[project.icon] : null

  const handleLaunch = async () => {
    if (isBrowse) {
      if (project.id === "foundry-showcase-project" || project.name === "Foundry Showcase") {
        onRequestAccess?.(project.id)
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

  return (
    <tr className="border-b border-border hover:bg-muted/10 transition-colors align-middle">
      <td className="p-4">
        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10 text-lg select-none text-primary">
          {LegacyIcon ? (
            <LegacyIcon className="h-4 w-4" />
          ) : (
            project.icon || "🚀"
          )}
        </div>
      </td>
      <td className="p-4">
        <div className="flex flex-col">
          <span className="text-sm font-semibold text-foreground flex items-center gap-1.5">
            {project.name}
            {project.isReadOnly && (
              <span title="Read-only repository access" className="flex shrink-0">
                <Unplug className="h-3.5 w-3.5 text-muted-foreground" />
              </span>
            )}
            {(project.id.startsWith("showcase-") || project.name === "Foundry Showcase") && (
              <span className="inline-flex items-center rounded-md bg-blue-50 dark:bg-blue-900/30 px-1.5 py-0.5 text-[9px] font-medium text-blue-700 dark:text-blue-300 ring-1 ring-inset ring-blue-700/10 shrink-0 select-none">
                Sample Project
              </span>
            )}
          </span>
          <span className="font-mono text-[9px] text-muted-foreground">
            ID: {project.id}
          </span>
        </div>
      </td>
      <td className="p-4 max-w-xs md:max-w-md">
        <p className="text-xs text-muted-foreground truncate">
          {project.description || "No project description provided."}
        </p>
      </td>
      {!isBrowse && (
        <td className="p-4 text-right">
          {bmadWorkflow.id && (
            <div className="flex items-center justify-end gap-1" title="Governance Gates">
              {["ONBOARDING", "ANALYSIS", "PLANNING", "SOLUTIONING"].map((phase) => {
                const status = getPhaseStatus(bmadWorkflow, phase, project.artifacts)
                let colorClass = "bg-muted/40 border-muted-foreground/10"

                if (status === "complete") {
                  colorClass = "bg-green-500 border-green-600"
                } else if (status === "in-progress") {
                  colorClass = "bg-amber-500 border-amber-600"
                } else if (status === "optional-unstarted") {
                  colorClass = "bg-yellow-500 border-yellow-600"
                }

                return (
                  <span
                    key={phase}
                    className={`h-1.5 w-3 rounded border ${colorClass}`}
                    title={phase}
                  />
                )
              })}
            </div>
          )}
        </td>
      )}
      <td className="p-4 text-right">
        <div className="flex items-center justify-end gap-2">
          {!isBrowse && (
            <>
              <button
                onClick={(e) => {
                  e.stopPropagation()
                  setIsShareOpen(true)
                }}
                className="flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
                title="Share Project"
              >
                <Share2 className="h-4 w-4" />
              </button>
              <button
                onClick={handleEdit}
                className="flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
                title="Edit Project Configuration"
              >
                <Edit3 className="h-4 w-4" />
              </button>
            </>
          )}
          {isBrowse ? (
            <button
              onClick={(e) => {
                e.stopPropagation()
                onRequestAccess?.(project.id)
              }}
              className="flex items-center gap-1 text-xs font-semibold text-primary hover:underline"
            >
              {project.id === "foundry-showcase-project" || project.name === "Foundry Showcase"
                ? "Provision Showcase"
                : "Request"}
              <ArrowUpRight className="h-3.5 w-3.5" />
            </button>
          ) : (
            <button
              onClick={handleLaunch}
              className="inline-flex h-8 items-center justify-center rounded-lg border border-input bg-background px-3 text-xs font-semibold text-foreground hover:bg-muted transition-all tour-project-launch-button"
            >
              {project.status === "New" ? "Onboarding" : "Open Workspace"}
            </button>
          )}
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
      </td>
    </tr>
  )
}

export default ProjectRow
