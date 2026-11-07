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

import React, { useState, useMemo } from "react"
import { useQuery } from "@tanstack/react-query"
import { Play, Loader2, Workflow, AlertCircle, Plus, Link2Off, Eye } from "lucide-react"
import { useParams } from "next/navigation"
import { api } from "@/lib/api"
import { WorkflowTemplate } from "@/lib/types"
import { toast } from "sonner"
import WorkflowPreviewModal from "./WorkflowPreviewModal"
import ComingSoonModal from "@/components/shared/ComingSoonModal"

interface WorkflowsControlProps {
  linkedWorkflowIds?: string[]
  addDebugLog?: (msg: string, isError?: boolean) => void
  onUnlink?: (id: string) => void
}

export default function WorkflowsControl({ linkedWorkflowIds = [], addDebugLog, onUnlink }: WorkflowsControlProps) {
  const [invokingId, setInvokingId] = useState<string | null>(null)
  const [workflowToUnlink, setWorkflowToUnlink] = useState<WorkflowTemplate | null>(null)
  const [workflowToPreview, setWorkflowToPreview] = useState<WorkflowTemplate | null>(null)
  const [isComingSoonOpen, setIsComingSoonOpen] = useState(false)
  const params = useParams()
  const projectId = params?.id as string

  // Fetch workflows
  const {
    data: allWorkflows,
    isLoading,
    error,
  } = useQuery<WorkflowTemplate[]>({
    queryKey: ["workflows", "ADK"],
    queryFn: async () => {
      const res = await api.get("/api/v1/workflows?type=ADK")
      if (!res.ok) {
        throw new Error(`HTTP error status: ${res.status}`)
      }
      return res.json()
    },
  })

  // Filter to only show linked ones
  const workflows = useMemo(() => {
    if (!allWorkflows) return []
    return allWorkflows.filter((w) => linkedWorkflowIds.includes(w.id))
  }, [allWorkflows, linkedWorkflowIds])

  const handleInvokeWorkflow = async (id: string) => {
    setInvokingId(id)
    const toastId = toast.loading(`Invoking workflow ${id}...`)
    try {
      const res = await api.post(`/api/v1/workflows/${id}/run?projectId=${projectId}`, "Start Execution")
      if (!res.ok) {
        const errorData = await res.json().catch(() => ({}))
        throw new Error(errorData.error || `HTTP error status: ${res.status}`)
      }
      const data = await res.json()
      toast.success("Workflow executed successfully!", {
        id: toastId,
        description: data.output ? (data.output.length > 80 ? data.output.slice(0, 80) + "..." : data.output) : "Success",
      })
      if (data.output) {
        addDebugLog?.(`Workflow ${id} executed successfully. Output:\n${data.output}`)
      } else {
        addDebugLog?.(`Workflow ${id} executed successfully.`)
      }
    } catch (err: any) {
      console.error("Failed to invoke workflow:", err)
      toast.error(err.message || "Failed to invoke workflow", { id: toastId })
      addDebugLog?.(`Failed to invoke workflow ${id}: ${err.message}`, true)
    } finally {
      setInvokingId(null)
    }
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-6 text-muted-foreground">
        <Loader2 className="h-4 w-4 animate-spin text-primary mr-2" />
        <span className="text-xs">Loading workflows...</span>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex items-center gap-2 p-3 text-destructive border border-destructive/20 bg-destructive/5 rounded-lg text-xs">
        <AlertCircle className="h-4 w-4 shrink-0" />
        <span className="truncate">{(error as Error).message || "Failed to load workflows"}</span>
      </div>
    )
  }

  if (!allWorkflows || allWorkflows.length === 0) {
    return (
      <div className="text-center py-6 border border-dashed border-border rounded-lg bg-muted/10">
        <p className="text-xs text-muted-foreground italic mb-2">No workflows available</p>
        <a
          href="/workflows"
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center gap-1 text-[10px] font-semibold text-primary hover:underline"
        >
          <Plus className="h-3 w-3" /> Create one
        </a>
      </div>
    )
  }

  if (workflows.length === 0) {
    return (
      <div className="text-center py-6 border border-dashed border-border rounded-lg bg-muted/10 p-3">
        <p className="text-xs text-muted-foreground italic mb-1">No workflows linked</p>
        <p className="text-[9px] text-muted-foreground/70 leading-normal">
          Click the "Link" button in the header to associate workflows with this project.
        </p>
      </div>
    )
  }

  return (
    <>
      <div className="space-y-1.5 max-h-[200px] overflow-y-auto pr-1">
        {workflows.map((workflow) => (
          <div
            key={workflow.id}
            className="flex items-center justify-between p-2 rounded-md border border-border/50 bg-card hover:bg-muted/20 transition-colors group"
          >
            <div
              onClick={() => setWorkflowToPreview(workflow)}
              className="flex items-center gap-2 min-w-0 flex-1 cursor-pointer group/title"
            >
              <Workflow className="h-3.5 w-3.5 text-muted-foreground group-hover/title:text-primary shrink-0 transition-colors" />
              <div className="flex flex-col min-w-0">
                <span className="text-xs font-medium truncate group-hover/title:text-primary transition-colors" title={workflow.displayName || workflow.name}>
                  {workflow.displayName || workflow.name}
                </span>
                <span className="text-[9px] text-muted-foreground/70 font-mono truncate">
                  {workflow.framework}
                </span>
              </div>
            </div>
            <div className="flex items-center gap-1 shrink-0">
              {onUnlink && (
                <button
                  onClick={() => setWorkflowToUnlink(workflow)}
                  disabled={invokingId !== null}
                  className="p-1 rounded text-muted-foreground hover:text-destructive hover:bg-destructive/10 disabled:opacity-50 transition-colors"
                  title="Unlink Workflow"
                >
                  <Link2Off className="h-3.5 w-3.5" />
                </button>
              )}
              <button
                onClick={() => setWorkflowToPreview(workflow)}
                className="p-1 rounded text-muted-foreground hover:text-primary hover:bg-primary/10 transition-colors"
                title="Preview Workflow"
              >
                <Eye className="h-3.5 w-3.5" />
              </button>
              <button
                onClick={() => setIsComingSoonOpen(true)}
                disabled={invokingId !== null}
                className="p-1 rounded text-muted-foreground hover:text-primary hover:bg-primary/10 disabled:opacity-50 transition-colors"
                title="Invoke Workflow"
              >
                {invokingId === workflow.id ? (
                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                ) : (
                  <Play className="h-3.5 w-3.5 fill-current" />
                )}
              </button>
            </div>
          </div>
        ))}
      </div>

      {/* Confirmation Modal */}
      {workflowToUnlink && (
        <div className="fixed inset-0 z-[100] flex items-center justify-center p-4 bg-slate-950/40 backdrop-blur-sm">
          <div className="w-full max-w-md rounded-xl border border-border bg-card p-5 shadow-2xl animate-in fade-in zoom-in-95 duration-150 select-none">
            <h3 className="text-sm font-bold text-foreground mb-1.5 flex items-center gap-2">
              <Link2Off className="h-4 w-4 text-destructive" />
              Unlink Workflow?
            </h3>
            <p className="text-xs text-muted-foreground mb-5 leading-normal">
              Are you sure you want to unlink <span className="font-semibold text-foreground">"{workflowToUnlink.displayName || workflowToUnlink.name}"</span> from this project's implementation phase? 
              This will not delete the workflow template itself.
            </p>
            <div className="flex justify-end gap-2">
              <button
                onClick={() => setWorkflowToUnlink(null)}
                className="inline-flex h-8 items-center justify-center rounded-lg border border-input bg-background px-3 text-[11px] font-semibold text-foreground hover:bg-muted transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={() => {
                  if (onUnlink && workflowToUnlink) {
                    onUnlink(workflowToUnlink.id)
                    setWorkflowToUnlink(null)
                  }
                }}
                className="inline-flex h-8 items-center justify-center rounded-lg bg-destructive px-3 text-[11px] font-semibold text-destructive-foreground shadow hover:bg-destructive/90 transition-colors"
              >
                Unlink
              </button>
            </div>
          </div>
        </div>
      )}

      <WorkflowPreviewModal
        isOpen={!!workflowToPreview}
        onClose={() => setWorkflowToPreview(null)}
        workflowId={workflowToPreview?.id || null}
        workflowName={workflowToPreview?.displayName || workflowToPreview?.name || ""}
        onUnlink={() => {
          if (workflowToPreview && onUnlink) {
            onUnlink(workflowToPreview.id)
          }
        }}
      />

      <ComingSoonModal
        isOpen={isComingSoonOpen}
        onClose={() => setIsComingSoonOpen(false)}
        featureName="Invoke Workflow"
      />
    </>
  )
}
