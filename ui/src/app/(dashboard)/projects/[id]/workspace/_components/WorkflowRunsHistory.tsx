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

import React from "react"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { Play, Loader2, Calendar, User, Hash, AlertCircle, RefreshCw, Workflow, Trash2 } from "lucide-react"
import { api } from "@/lib/api"
import { toast } from "sonner"
import { cn } from "@/lib/utils"
import ConfirmDialog from "@/components/shared/ConfirmDialog"

interface WorkflowRun {
  id: string
  userId: string
  projectId: string
  workflowId: string
  startDate: string
  workflowName: string
  status?: string
  awaitingInput?: boolean
}

function formatStatusLabel(status?: string): string {
  if (!status) return "Completed"
  return status
    .replace(/_/g, " ")
    .toLowerCase()
    .replace(/\b\w/g, (char) => char.toUpperCase())
}

interface WorkflowRunsHistoryProps {
  projectId: string
  selectedRunId?: string | null
  onSelectRun?: (runId: string, workflowId: string, workflowName: string) => void
  onDeleteRun?: (runId: string) => void
  compact?: boolean
}

export default function WorkflowRunsHistory({
  projectId,
  selectedRunId = null,
  onSelectRun,
  onDeleteRun,
  compact = false,
}: WorkflowRunsHistoryProps) {
  const queryClient = useQueryClient()
  const [runToDelete, setRunToDelete] = React.useState<WorkflowRun | null>(null)

  const deleteRunMutation = useMutation({
    mutationFn: async (runId: string) => {
      const res = await api.delete(`/api/v1/projects/${projectId}/workflow-runs/${runId}`)
      if (!res.ok) {
        const errData = await res.json().catch(() => ({}))
        throw new Error(errData.error || "Failed to delete workflow run")
      }
    },
    onSuccess: (_, deletedId) => {
      toast.success("Workflow run deleted successfully")
      queryClient.invalidateQueries({ queryKey: ["workflow-runs", projectId] })
      if (selectedRunId === deletedId) {
        onDeleteRun?.(deletedId)
      }
      setRunToDelete(null)
    },
    onError: (error: Error) => {
      toast.error(error.message || "Failed to delete workflow run")
    },
  })

  // Fetch prior workflow runs
  const {
    data: runs,
    isLoading,
    error,
    refetch,
    isRefetching,
  } = useQuery<WorkflowRun[]>({
    queryKey: ["workflow-runs", projectId],
    queryFn: async () => {
      const res = await api.get(`/api/v1/projects/${projectId}/workflow-runs`)
      if (!res.ok) {
        throw new Error(`Failed to load workflow runs: Status ${res.status}`)
      }
      return res.json()
    },
    refetchInterval: 5000, // Poll every 5 seconds to show new runs automatically
  })

  const formatDateTime = (dateStr: string) => {
    try {
      const date = new Date(dateStr)
      return date.toLocaleString(undefined, {
        dateStyle: "medium",
        timeStyle: "short",
      })
    } catch (e) {
      return dateStr
    }
  }

  const handleCopyId = (id: string) => {
    navigator.clipboard.writeText(id)
    toast.success("Run ID copied to clipboard!")
  }

  if (isLoading) {
    return (
      <div className="flex h-full w-full flex-col items-center justify-center text-center p-6 select-none">
        <Loader2 className="h-7 w-7 animate-spin text-primary mb-3" />
        <h3 className="text-xs font-bold text-foreground mb-0.5">Loading execution history...</h3>
        <p className="text-[10px] text-muted-foreground">Fetching prior workflow runs.</p>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex h-full w-full flex-col items-center justify-center text-center p-6 max-w-sm mx-auto">
        <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-destructive/10 text-destructive mb-3 border border-destructive/20">
          <AlertCircle className="h-5.5 w-5.5" />
        </div>
        <h3 className="text-xs font-bold text-foreground mb-0.5">Failed to load history</h3>
        <p className="text-[10px] text-muted-foreground mb-3">
          {(error as Error).message || "An error occurred."}
        </p>
        <button
          onClick={() => refetch()}
          className="inline-flex h-8 items-center justify-center gap-1 rounded-lg border border-input bg-background px-3 text-[10px] font-semibold text-foreground hover:bg-muted transition-colors"
        >
          <RefreshCw className="h-3 w-3" />
          Retry
        </button>
      </div>
    )
  }

  if (!runs || runs.length === 0) {
    return (
      <div className="flex h-full w-full flex-col items-center justify-center text-center p-6 select-none">
        <div className="flex h-14 w-14 items-center justify-center rounded-xl bg-slate-100 dark:bg-slate-950 text-muted-foreground/80 mb-5 border border-border">
          <Workflow className="h-7 w-7 text-primary" />
        </div>
        <h2 className="text-sm font-bold text-foreground mb-1.5">No Workflows Executed</h2>
        <p className="text-[11px] text-muted-foreground max-w-[240px] leading-relaxed">
          No workflows have been executed in this project yet. Run a linked workflow from the sidebar on the left.
        </p>
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full w-full bg-card overflow-hidden">
      {/* Header */}
      <div className={cn(
        "border-b border-border flex items-center justify-between shrink-0 bg-muted/5 select-none", 
        compact ? "px-4 py-3" : "px-6 py-4"
      )}>
        <div>
          <h2 className="text-xs font-bold text-foreground flex items-center gap-1.5">
            <Workflow className="h-4 w-4 text-primary" />
            {compact ? "Executions" : "Workflow Execution History"}
          </h2>
          {!compact && (
            <p className="text-[11px] text-muted-foreground mt-0.5">
              Real-time status of all workflows executed within this project workspace.
            </p>
          )}
        </div>
        <button
          onClick={() => refetch()}
          disabled={isRefetching}
          className="inline-flex h-6 items-center gap-1 text-[9px] font-bold border border-border bg-background px-2.5 rounded-md hover:bg-muted text-muted-foreground transition-colors disabled:opacity-50"
          title="Refresh History"
        >
          <RefreshCw className={`h-2.5 w-2.5 ${isRefetching ? "animate-spin" : ""}`} />
          {compact ? "" : "Refresh"}
        </button>
      </div>

      {/* Runs List */}
      <div className={cn("flex-1 min-h-0 overflow-y-auto", compact ? "p-3" : "p-6")}>
        <div className={cn("space-y-2.5 w-full", compact ? "" : "max-w-4xl mr-auto")}>
          {runs.map((run) => {
            const isSelected = run.id === selectedRunId
            const isAwaitingInput = run.awaitingInput || run.status === "AWAITING_INPUT"
            const isRunning = run.status === "RUNNING"
            const isError = run.status === "ERROR"
            return (
              <div
                key={run.id}
                onClick={() => onSelectRun?.(run.id, run.workflowId, run.workflowName)}
                className={cn(
                  "flex flex-col rounded-xl border transition-all duration-200 gap-3",
                  compact ? "p-3 gap-2" : "flex-row md:items-center justify-between p-4 gap-4",
                  isSelected 
                    ? "border-primary bg-primary/5 shadow-sm ring-1 ring-primary/20" 
                    : isAwaitingInput
                      ? "border-amber-500/50 bg-amber-500/5 hover:bg-amber-500/10 shadow-sm"
                      : "border-border/60 bg-card hover:bg-muted/15",
                  onSelectRun ? "cursor-pointer" : ""
                )}
              >
                <div className="flex items-start gap-3 min-w-0 flex-1">
                  <div className={cn(
                    "flex items-center justify-center rounded-lg border shrink-0 mt-0.5",
                    compact ? "h-7 w-7 bg-primary/5 text-primary border-primary/10" : "h-9 w-9 bg-primary/10 text-primary border-primary/20"
                  )}>
                    <Workflow className={compact ? "h-3.5 w-3.5" : "h-4.5 w-4.5"} />
                  </div>
                  <div className="flex flex-col min-w-0 gap-0.5">
                    <span className={cn("font-bold truncate text-foreground", compact ? "text-[11px]" : "text-xs")} title={run.workflowName}>
                      {run.workflowName}
                    </span>
                    <div className="flex flex-wrap items-center gap-x-2.5 gap-y-1 text-[9px] text-muted-foreground">
                      <span className="flex items-center gap-1">
                        <Calendar className="h-2.5 w-2.5" />
                        {formatDateTime(run.startDate)}
                      </span>
                      {!compact && (
                        <span className="flex items-center gap-1 truncate max-w-[150px]" title={run.userId}>
                          <User className="h-2.5 w-2.5" />
                          {run.userId}
                        </span>
                      )}
                    </div>
                  </div>
                </div>

                <div className={cn(
                  "flex items-center shrink-0",
                  compact ? "justify-between w-full mt-1 border-t border-border/40 pt-2" : "gap-2.5"
                )}>
                  {!compact ? (
                    <>
                      <div 
                        onClick={(e) => {
                          e.stopPropagation() // Prevent row click from triggering selection
                          handleCopyId(run.id)
                        }}
                        className="flex items-center gap-1 text-[9px] font-mono border border-border/80 bg-muted/35 hover:bg-muted hover:border-border px-2 py-1 rounded-md cursor-pointer transition-colors"
                        title="Click to copy Run ID"
                      >
                        <Hash className="h-2.5 w-2.5 text-muted-foreground/80" />
                        <span className="text-muted-foreground truncate max-w-[80px]">{run.id}</span>
                      </div>
                      {isAwaitingInput ? (
                        <span className="inline-flex items-center gap-1.5 text-[10px] font-bold text-amber-700 dark:text-amber-300 bg-amber-500/15 border border-amber-500/30 px-2.5 py-0.5 rounded-full select-none animate-pulse">
                          <span className="h-1.5 w-1.5 rounded-full bg-amber-500 animate-ping" />
                          {formatStatusLabel(run.status || "WAITING")}
                        </span>
                      ) : isRunning ? (
                        <span className="inline-flex items-center gap-1 text-[10px] font-semibold text-blue-700 dark:text-blue-400 bg-blue-500/10 border border-blue-500/20 px-2.5 py-0.5 rounded-full select-none">
                          <Loader2 className="h-2.5 w-2.5 animate-spin" />
                          {formatStatusLabel(run.status || "RUNNING")}
                        </span>
                      ) : isError ? (
                        <span className="text-[10px] font-semibold text-destructive bg-destructive/10 border border-destructive/20 px-2.5 py-0.5 rounded-full select-none">
                          {formatStatusLabel(run.status || "FAILED")}
                        </span>
                      ) : (
                        <span className="text-[10px] font-semibold text-green-700 dark:text-green-400 bg-green-500/10 border border-green-500/20 px-2.5 py-0.5 rounded-full select-none">
                          {formatStatusLabel(run.status || "COMPLETED")}
                        </span>
                      )}
                      <button
                        type="button"
                        onClick={(e) => {
                          e.stopPropagation()
                          setRunToDelete(run)
                        }}
                        disabled={deleteRunMutation.isPending && runToDelete?.id === run.id}
                        className="flex h-7 w-7 items-center justify-center rounded-lg border border-destructive/20 bg-destructive/5 text-destructive hover:bg-destructive/15 transition-colors disabled:opacity-50"
                        title="Delete this workflow run"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </button>
                    </>
                  ) : (
                    <>
                      <span className="text-[8px] font-mono text-muted-foreground">ID: {run.id.substring(0, 8)}...</span>
                      <div className="flex items-center gap-1.5">
                        {isAwaitingInput ? (
                          <span className="inline-flex items-center gap-1 text-[8px] font-bold text-amber-700 dark:text-amber-300 bg-amber-500/15 border border-amber-500/30 px-1.5 py-0.2 rounded-full select-none animate-pulse" title="Waiting for user input">
                            <span className="h-1 w-1 rounded-full bg-amber-500 animate-ping" />
                            {formatStatusLabel(run.status || "WAITING")}
                          </span>
                        ) : isRunning ? (
                          <span className="inline-flex items-center gap-1 text-[8px] font-semibold text-blue-700 dark:text-blue-400 bg-blue-500/10 border border-blue-500/20 px-1.5 py-0.2 rounded-full select-none">
                            <Loader2 className="h-2 w-2 animate-spin" />
                            {formatStatusLabel(run.status || "RUNNING")}
                          </span>
                        ) : isError ? (
                          <span className="text-[8px] font-semibold text-destructive bg-destructive/10 border border-destructive/20 px-1.5 py-0.2 rounded-full select-none">
                            {formatStatusLabel(run.status || "FAILED")}
                          </span>
                        ) : (
                          <span className="text-[8px] font-semibold text-green-700 dark:text-green-400 bg-green-500/10 border border-green-500/20 px-1.5 py-0.2 rounded-full select-none">
                            {formatStatusLabel(run.status || "COMPLETED")}
                          </span>
                        )}
                        <button
                          type="button"
                          onClick={(e) => {
                            e.stopPropagation()
                            setRunToDelete(run)
                          }}
                          disabled={deleteRunMutation.isPending && runToDelete?.id === run.id}
                          className="flex h-6 w-6 items-center justify-center rounded-md border border-destructive/20 bg-destructive/5 text-destructive hover:bg-destructive/15 transition-colors disabled:opacity-50"
                          title="Delete run"
                        >
                          <Trash2 className="h-3 w-3" />
                        </button>
                      </div>
                    </>
                  )}
                </div>
              </div>
            )
          })}
        </div>
      </div>

      <ConfirmDialog
        isOpen={!!runToDelete}
        onClose={() => setRunToDelete(null)}
        onConfirm={() => {
          if (runToDelete) {
            deleteRunMutation.mutate(runToDelete.id)
          }
        }}
        title="Delete Workflow Run"
        message={`Are you sure you want to permanently delete execution run "${runToDelete?.workflowName || runToDelete?.id}" and its execution logs? This action cannot be undone.`}
        confirmText="Delete Run"
        isDestructive={true}
        isLoading={deleteRunMutation.isPending}
      />
    </div>
  )
}
