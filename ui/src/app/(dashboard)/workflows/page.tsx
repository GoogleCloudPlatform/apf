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

import { useEffect, useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { useRouter } from "next/navigation"
import { Loader2, AlertCircle, Workflow, Layers, FileText, LayoutGrid, List, Plus, Edit3, Play, ShieldAlert } from "lucide-react"
import { useAppStore } from "@/stores/app"
import { useAuthStore } from "@/stores/auth"
import { useActivityStore } from "@/stores/activity"
import { toast } from "sonner"
import { api } from "@/lib/api"
import { WorkflowTemplate } from "@/lib/types"

export default function WorkflowsPage() {
  const router = useRouter()
  const { setPageTitle } = useAppStore()
  const { isAuthenticated, isCheckingAuth, isAdmin, permissions } = useAuthStore()
  const { isPanelOpen } = useActivityStore()

  const userPermissions = permissions || []
  const hasPermission = (perm: string) => isAdmin || userPermissions.includes(perm)

  const [viewMode, setViewMode] = useState<"card" | "list">("card")
  const [invokingId, setInvokingId] = useState<string | null>(null)

  const handleInvokeWorkflow = async (id: string) => {
    setInvokingId(id)
    const toastId = toast.loading(`Invoking workflow ${id}...`)
    try {
      const res = await api.post(`/api/v1/workflows/${id}/run`, "Start Execution")
      if (!res.ok) {
        const errorData = await res.json().catch(() => ({}))
        throw new Error(errorData.error || `HTTP error status: ${res.status}`)
      }
      const data = await res.json()
      toast.success("Workflow executed successfully!", {
        id: toastId,
        description: data.output ? (data.output.length > 80 ? data.output.slice(0, 80) + "..." : data.output) : "Success",
      })
      console.log("Workflow output:", data.output)
    } catch (err: any) {
      console.error("Failed to invoke workflow:", err)
      toast.error(err.message || "Failed to invoke workflow", { id: toastId })
    } finally {
      setInvokingId(null)
    }
  }

  const handleCreateWorkflow = () => {
    router.push("/workflows/new")
  }

  const handleEditWorkflow = (id: string) => {
    router.push(`/workflows/${id}`)
  }

  useEffect(() => {
    setPageTitle("Workflows")
  }, [setPageTitle])

  const {
    data: workflows,
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
    enabled: isAuthenticated && hasPermission("project:list"),
  })

  if (!isCheckingAuth && isAuthenticated && !hasPermission("project:list")) {
    return (
      <div className="flex-1 flex flex-col items-center justify-center p-8 min-h-[calc(100vh-4rem)] select-none">
        <div className="w-full max-w-md bg-destructive/10 border border-destructive/20 rounded-2xl p-8 text-center space-y-6">
          <div className="flex justify-center text-destructive">
            <ShieldAlert className="h-16 w-16" />
          </div>
          <div className="space-y-2">
            <h3 className="text-xl font-bold text-foreground">
              Access Restricted
            </h3>
            <p className="text-sm text-muted-foreground leading-relaxed">
              You do not have permission to view workflows.
            </p>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="p-6 space-y-6 w-full">
      <div className="flex items-center justify-between border-b border-border pb-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Workflow Templates</h1>
          <p className="text-muted-foreground text-sm">
            Browse available agentic workflow templates.
          </p>
        </div>
        
        {/* View Mode Toggle */}
        {!isLoading && !error && workflows && workflows.length > 0 && (
          <div className="flex items-center gap-2">
            <button
              onClick={() => setViewMode("card")}
              className={`p-1.5 rounded-lg transition-colors ${
                viewMode === "card"
                  ? "bg-muted text-foreground"
                  : "text-muted-foreground hover:text-foreground"
              }`}
              title="Card View"
            >
              <LayoutGrid className="h-4 w-4" />
            </button>
            <button
              onClick={() => setViewMode("list")}
              className={`p-1.5 rounded-lg transition-colors ${
                viewMode === "list"
                  ? "bg-muted text-foreground"
                  : "text-muted-foreground hover:text-foreground"
              }`}
              title="List View"
            >
              <List className="h-4 w-4" />
            </button>
          </div>
        )}
      </div>

      {isLoading ? (
        <div className="flex h-[300px] w-full flex-col items-center justify-center gap-3 text-muted-foreground select-none">
          <Loader2 className="h-8 w-8 animate-spin text-primary" />
          <span className="text-sm font-medium">Loading workflows...</span>
        </div>
      ) : error ? (
        <div className="flex h-[250px] w-full flex-col items-center justify-center text-center gap-2 text-destructive max-w-md mx-auto p-4 border border-destructive/20 bg-destructive/5 rounded-xl">
          <AlertCircle className="h-8 w-8 opacity-60" />
          <h3 className="font-semibold text-sm">Failed to load workflows</h3>
          <p className="text-xs text-muted-foreground leading-normal">
            {(error as Error).message || "An unexpected error occurred."}
          </p>
        </div>
      ) : !workflows || workflows.length === 0 ? (
        <div className="flex h-[250px] w-full flex-col items-center justify-center text-center gap-2 border border-dashed rounded-xl p-8">
          <Workflow className="h-8 w-8 text-muted-foreground opacity-60" />
          <h3 className="font-semibold text-sm">No workflows found</h3>
          <p className="text-xs text-muted-foreground">
            There are no workflow templates available.
          </p>
        </div>
      ) : viewMode === "card" ? (
        <div className="grid grid-cols-[repeat(auto-fill,minmax(325px,1fr))] gap-4 pb-12">
          {workflows.map((workflow) => {
            const phaseCount = Object.keys(workflow.phases || {}).length
            const artifactCount = Object.values(workflow.artifacts || {}).reduce(
              (acc, arr) => acc + arr.length,
              0
            )

            return (
              <div
                key={workflow.id}
                className="flex flex-col rounded-xl border border-border bg-card text-card-foreground shadow-sm overflow-hidden min-h-[225px] hover:border-muted-foreground/20 transition-all duration-200"
              >
                {/* Upper header section */}
                <div className="p-5 flex items-start gap-4 border-b border-border bg-muted/20">
                  <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
                    <Workflow className="h-5 w-5" />
                  </div>
                  
                  <div className="flex-1 min-w-0">
                    <h3 className="text-sm font-semibold tracking-tight text-foreground truncate" title={workflow.displayName || workflow.name}>
                      {workflow.displayName || workflow.name}
                    </h3>
                    <div className="flex flex-col gap-0.5 mt-0.5">
                      <span className="font-mono text-[9px] text-muted-foreground/70">
                        ID: {workflow.id}
                      </span>
                    </div>
                  </div>

                </div>

                {/* Body section */}
                <div className="flex-1 p-5 flex flex-col justify-between gap-4">
                  <div className="text-xs text-muted-foreground leading-relaxed">
                    <p>{workflow.description || `Templates for orchestrating agentic workflows using ${workflow.framework} framework.`}</p>
                  </div>

                  <div className="pt-4 border-t border-border flex justify-between items-center">
                    <span className="inline-flex shrink-0 items-center px-2.5 py-0.5 rounded-full text-[10px] font-semibold bg-primary/10 text-primary">
                      {workflow.framework}
                    </span>
                    <button
                      onClick={() => handleEditWorkflow(workflow.id)}
                      className="inline-flex items-center justify-center gap-1.5 rounded-lg bg-primary px-3 py-1.5 text-xs font-semibold text-primary-foreground shadow-sm hover:bg-primary/90 active:scale-95 transition-all duration-200"
                    >
                      <Edit3 className="h-3.5 w-3.5" />
                      Design
                    </button>
                  </div>
                </div>
              </div>
            )
          })}
        </div>
      ) : (
        // List View (Table)
        <div className="border border-border rounded-xl overflow-hidden bg-card">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="border-b border-border bg-muted/20 text-xs font-semibold text-muted-foreground">
                <th className="p-4">Workflow Name</th>
                <th className="p-4">Framework</th>
                <th className="p-4">Phases</th>
                <th className="p-4">Required Artifacts</th>
                <th className="p-4 text-right">Actions</th>
              </tr>
            </thead>
            <tbody>
              {workflows.map((workflow) => {
                const phaseCount = Object.keys(workflow.phases || {}).length
                const artifactCount = Object.values(workflow.artifacts || {}).reduce(
                  (acc, arr) => acc + arr.length,
                  0
                )

                return (
                  <tr key={workflow.id} className="border-b border-border/50 hover:bg-muted/5 transition-colors last:border-0">
                    <td className="p-4">
                      <div className="font-semibold text-sm">{workflow.displayName || workflow.name}</div>
                      <div className="text-xs text-muted-foreground flex flex-col gap-0.5 mt-0.5">
                        {workflow.displayName && workflow.displayName !== workflow.name && (
                          <span>ADK Name: <code className="text-[10px] bg-muted/50 px-1 py-0.5 rounded">{workflow.name}</code></span>
                        )}
                        <span>ID: {workflow.id}</span>
                      </div>
                    </td>
                    <td className="p-4">
                      <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-primary/10 text-primary">
                        {workflow.framework}
                      </span>
                    </td>
                    <td className="p-4 text-sm text-muted-foreground">
                      <div className="flex items-center gap-1.5">
                        <Layers className="h-4 w-4" />
                        <span>{phaseCount} Phases</span>
                      </div>
                    </td>
                    <td className="p-4 text-sm text-muted-foreground">
                      <div className="flex items-center gap-1.5">
                        <FileText className="h-4 w-4" />
                        <span>{artifactCount} Artifacts</span>
                      </div>
                    </td>
                    <td className="p-4 text-right">
                      <div className="flex items-center justify-end gap-2">
                        <button
                          onClick={() => handleInvokeWorkflow(workflow.id)}
                          disabled={invokingId !== null}
                          className="inline-flex h-8 px-2.5 items-center justify-center gap-1 rounded-lg bg-primary/10 text-primary hover:bg-primary/20 text-xs font-medium disabled:opacity-50 disabled:pointer-events-none transition-colors"
                          title="Invoke Workflow"
                        >
                          {invokingId === workflow.id ? (
                            <Loader2 className="h-3.5 w-3.5 animate-spin" />
                          ) : (
                            <Play className="h-3.5 w-3.5 fill-current" />
                          )}
                          Invoke
                        </button>
                        <button
                          onClick={() => handleEditWorkflow(workflow.id)}
                          className="inline-flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
                          title="Edit Workflow"
                        >
                          <Edit3 className="h-4 w-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}

      {/* Floating Action Button (FAB) */}
      {isAuthenticated && (
        <button
          onClick={handleCreateWorkflow}
          className={`fixed bottom-6 z-50 inline-flex h-12 items-center justify-center gap-2 rounded-full bg-primary px-6 text-sm font-semibold text-primary-foreground shadow-lg transition-all duration-200 hover:scale-105 active:scale-95 ${
            isPanelOpen ? "right-[364px]" : "right-6"
          }`}
        >
          <Plus className="h-5 w-5" />
          Create Workflow
        </button>
      )}
    </div>
  )
}
