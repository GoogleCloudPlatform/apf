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

import React, { useState, useMemo, useCallback } from "react"
import { useQuery } from "@tanstack/react-query"
import { Search, Loader2, AlertCircle, Workflow, ChevronRight } from "lucide-react"
import { api } from "@/lib/api"
import {
  WorkflowTemplate,
  AdkNode,
  Agent,
  FunctionNode,
  JoinNode,
  CustomNode,
  LoopNode,
  ScheduledTrigger,
  ManualTrigger,
  FormTrigger,
} from "@/lib/types"
import {
  WorkflowDesigner,
  DesignerNode,
  DesignerEdge,
} from "@/app/(dashboard)/workflows/components/WorkflowDesigner"
import { cn } from "@/lib/utils"

interface WorkflowSelectorProps {
  onSelect?: (workflow: WorkflowTemplate) => void
  selectedId?: string | null
}

export default function WorkflowSelector({ onSelect, selectedId: initialSelectedId }: WorkflowSelectorProps) {
  const [searchQuery, setSearchQuery] = useState("")
  const [selectedId, setSelectedId] = useState<string | null>(initialSelectedId || null)

  // Fetch workflows
  const {
    data: workflows = [],
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

  // Filter workflows based on search query
  const filteredWorkflows = useMemo(() => {
    return workflows.filter((w) => {
      const name = (w.displayName || w.name || "").toLowerCase()
      const desc = (w.description || "").toLowerCase()
      const query = searchQuery.toLowerCase()
      return name.includes(query) || desc.includes(query)
    })
  }, [workflows, searchQuery])

  // Get currently selected workflow
  const selectedWorkflow = useMemo(() => {
    return workflows.find((w) => w.id === selectedId) || null
  }, [workflows, selectedId])

  // Map WorkflowTemplate nodes/edges to DesignerNode/DesignerEdge for ReactFlow
  const { nodes, edges } = useMemo(() => {
    if (!selectedWorkflow) return { nodes: [], edges: [] }

    const wfData = selectedWorkflow

    if (wfData.framework === "BMAD") {
      // BMAD workflows might not have graph layout, return empty or default
      return { nodes: [], edges: [] }
    }

    if (wfData.layout && wfData.layout.nodes) {
      const restoredNodes = wfData.layout.nodes.map((vn: any) => {
        let nodeData: AdkNode

        if (vn.type === "waypoint" || vn.type === "noop") {
          nodeData = new CustomNode({
            name: vn.data?.name || vn.id,
            displayName: vn.data?.displayName || (vn.type === "noop" ? "Noop Node" : "Waypoint"),
            type: vn.type as any,
          } as any);
          (nodeData as any).orientation = vn.orientation || "LR"
        } else {
          const logicalNode = (wfData.nodes || []).find((ln: any) => ln.id === vn.id)
          const initData = logicalNode || {
            id: vn.id,
            name: vn.data?.name || vn.id,
            displayName: vn.data?.displayName,
            type: vn.type
          }

          if (vn.type === "agent") {
            nodeData = new Agent(initData as any)
          } else if (vn.type === "function") {
            nodeData = new FunctionNode(initData as any)
          } else if (vn.type === "join") {
            nodeData = new JoinNode(initData as any)
          } else if (vn.type === "trigger") {
            nodeData = new ScheduledTrigger(initData as any)
          } else if (vn.type === "manual_trigger") {
            nodeData = new ManualTrigger(initData as any)
          } else if (vn.type === "form_trigger") {
            nodeData = new FormTrigger(initData as any)
          } else if (vn.type === "loop") {
            nodeData = new LoopNode(initData as any)
          } else {
            nodeData = new CustomNode(initData as any)
          }
        }

        return {
          id: vn.id,
          type: vn.type,
          position: vn.position,
          data: nodeData,
        }
      })

      const restoredEdges = (wfData.layout.edges || []).map((ve: any) => ({
        id: ve.id,
        source: ve.source,
        target: ve.target,
        sourceHandle: ve.sourceHandle || undefined,
        targetHandle: ve.targetHandle || undefined,
        label: typeof ve.route === "string" ? ve.route : ve.route ? String(ve.route) : undefined,
        data: { route: ve.route },
      }))

      return { nodes: restoredNodes, edges: restoredEdges }
    } else if (wfData.nodes) {
      // Legacy Fallback
      const restoredNodes = wfData.nodes.map((n: any, index: number) => {
        let nodeData: AdkNode
        if (n.type === "agent") {
          nodeData = new Agent(n)
        } else if (n.type === "function") {
          nodeData = new FunctionNode(n)
        } else if (n.type === "join") {
          nodeData = new JoinNode(n)
        } else if (n.type === "trigger") {
          nodeData = new ScheduledTrigger(n)
        } else if (n.type === "manual_trigger") {
          nodeData = new ManualTrigger(n)
        } else if (n.type === "form_trigger") {
          nodeData = new FormTrigger(n)
        } else if (n.type === "loop") {
          nodeData = new LoopNode(n)
        } else {
          nodeData = new CustomNode(n)
        }

        const x = n.id === "start" ? 84 : 84 + index * 168
        const y = 154

        return {
          id: n.id,
          type: n.type,
          position: { x, y },
          data: nodeData,
        }
      })

      const restoredEdges = (wfData.edges || []).map((e: any) => ({
        id: e.id,
        source: e.fromNode,
        target: e.toNode,
        label: typeof e.route === "string" ? e.route : e.route ? String(e.route) : undefined,
        data: { route: e.route },
      }))

      return { nodes: restoredNodes, edges: restoredEdges }
    }

    return { nodes: [], edges: [] }
  }, [selectedWorkflow])

  const handleSelectWorkflow = (workflow: WorkflowTemplate) => {
    setSelectedId(workflow.id)
    if (onSelect) {
      onSelect(workflow)
    }
  }

  if (isLoading) {
    return (
      <div className="flex h-full w-full items-center justify-center gap-2 text-muted-foreground">
        <Loader2 className="h-6 w-6 animate-spin text-primary" />
        <span className="text-sm">Loading workflows...</span>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex h-full w-full flex-col items-center justify-center gap-2 text-destructive p-6">
        <AlertCircle className="h-8 w-8 opacity-60" />
        <h3 className="font-semibold">Failed to load workflows</h3>
        <p className="text-xs text-muted-foreground">{(error as Error).message}</p>
      </div>
    )
  }

  return (
    <div className="flex h-full w-full overflow-hidden bg-background">
      {/* Left Panel: Search & List */}
      <div className="w-[275px] border-r border-border flex flex-col h-full shrink-0 bg-card">
        {/* Search Header */}
        <div className="p-3 border-b border-border shrink-0">
          <div className="relative">
            <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
            <input
              type="text"
              placeholder="Search workflows..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-9 pr-3 py-1.5 text-xs rounded-md border border-input bg-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
            />
          </div>
        </div>

        {/* Workflows List */}
        <div className="flex-1 overflow-y-auto p-2 space-y-1 select-none">
          {filteredWorkflows.length === 0 ? (
            <div className="text-center py-8 text-xs text-muted-foreground italic">
              No workflows found
            </div>
          ) : (
            filteredWorkflows.map((w) => {
              const isSelected = w.id === selectedId
              return (
                <div
                  key={w.id}
                  onClick={() => handleSelectWorkflow(w)}
                  className={cn(
                    "flex items-center justify-between p-2.5 rounded-lg border text-left text-xs transition-all cursor-pointer",
                    isSelected
                      ? "bg-primary/5 text-primary border-primary font-semibold shadow-sm"
                      : "border-transparent text-muted-foreground hover:bg-muted/50 hover:text-foreground"
                  )}
                >
                  <div className="flex items-center gap-2.5 min-w-0 flex-1">
                    <div className={cn(
                      "p-1.5 rounded-md shrink-0",
                      isSelected ? "bg-primary/10 text-primary" : "bg-muted text-muted-foreground"
                    )}>
                      <Workflow className="h-3.5 w-3.5" />
                    </div>
                    <div className="flex flex-col min-w-0 flex-1">
                      <span className={cn(
                        "font-medium truncate",
                        isSelected ? "text-primary" : "text-foreground"
                      )}>
                        {w.displayName || w.name}
                      </span>
                      <span className="text-[10px] text-muted-foreground/70 font-mono truncate mt-0.5">
                        {w.framework}
                      </span>
                    </div>
                  </div>
                  <ChevronRight className={cn(
                    "h-3.5 w-3.5 opacity-50 shrink-0 transition-transform",
                    isSelected && "translate-x-0.5 opacity-100"
                  )} />
                </div>
              )
            })
          )}
        </div>
      </div>

      {/* Right Panel: Preview */}
      <div className="flex-1 h-full flex flex-col overflow-hidden bg-slate-50 dark:bg-slate-950">
        {selectedWorkflow ? (
          <div className="flex flex-col h-full w-full overflow-hidden">
            {/* Preview Header */}
            <div className="px-4 py-3 border-b border-border bg-card flex items-center justify-between shrink-0">
              <div>
                <h3 className="text-xs font-bold text-foreground">
                  {selectedWorkflow.displayName || selectedWorkflow.name}
                </h3>
                <p className="text-[10px] text-muted-foreground truncate max-w-md mt-0.5">
                  {selectedWorkflow.description || "No description provided."}
                </p>
              </div>
              <span className="inline-flex items-center px-2 py-0.5 rounded-full text-[10px] font-medium bg-primary/10 text-primary">
                {selectedWorkflow.framework}
              </span>
            </div>

            {/* Preview Canvas */}
            <div className="flex-1 min-h-0 relative">
              {selectedWorkflow.framework === "BMAD" ? (
                <div className="absolute inset-0 flex flex-col items-center justify-center text-center p-6 bg-card">
                  <Workflow className="h-8 w-8 text-muted-foreground opacity-40 mb-3" />
                  <h4 className="text-sm font-semibold mb-1">BMAD Workflow</h4>
                  <p className="text-xs text-muted-foreground max-w-xs leading-relaxed">
                    BMAD workflows are phase-based and do not have a visual graph representation in the designer.
                  </p>
                </div>
              ) : (
                <WorkflowDesigner
                  key={selectedWorkflow.id}
                  nodes={nodes}
                  edges={edges}
                  readOnly={true}
                />
              )}
            </div>
          </div>
        ) : (
          <div className="h-full w-full flex flex-col items-center justify-center text-center p-6 select-none">
            <div className="h-12 w-12 rounded-full bg-muted flex items-center justify-center text-muted-foreground/60 mb-3">
              <Workflow className="h-6 w-6" />
            </div>
            <h3 className="font-semibold text-sm text-foreground mb-1">No Workflow Selected</h3>
            <p className="text-xs text-muted-foreground max-w-xs">
              Select a workflow from the list on the left to visualize its execution graph.
            </p>
          </div>
        )}
      </div>
    </div>
  )
}
