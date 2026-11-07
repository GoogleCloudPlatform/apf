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

import React, { useEffect, useState, useCallback } from "react"
import { X, Eye, Link2Off } from "lucide-react"
import { useQuery } from "@tanstack/react-query"
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
import { WorkflowDesigner, DesignerNode, DesignerEdge } from "../../../../workflows/components/WorkflowDesigner"
import { useNodesState, useEdgesState } from "@xyflow/react"

interface WorkflowPreviewModalProps {
  isOpen: boolean
  onClose: () => void
  workflowId: string | null
  workflowName: string
  onUnlink: () => void
}

export default function WorkflowPreviewModal({
  isOpen,
  onClose,
  workflowId,
  workflowName,
  onUnlink,
}: WorkflowPreviewModalProps) {
  const [nodes, setNodes, onNodesChange] = useNodesState<DesignerNode>([])
  const [edges, setEdges, onEdgesChange] = useEdgesState<DesignerEdge>([])

  // Fetch Workflow details
  const { data: workflow, isLoading } = useQuery<WorkflowTemplate>({
    queryKey: ["workflow", workflowId],
    queryFn: async () => {
      if (!workflowId) throw new Error("No workflow ID provided")
      const res = await api.get(`/api/v1/workflows/${workflowId}`)
      if (!res.ok) {
        throw new Error("Failed to load workflow details")
      }
      return res.json()
    },
    enabled: isOpen && !!workflowId,
  })

  const loadWorkflowData = useCallback((wfData: any) => {
    if (!wfData) return

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

      setNodes(restoredNodes)
      setEdges(restoredEdges)
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

      setNodes(restoredNodes)
      setEdges(restoredEdges)
    }
  }, [])

  useEffect(() => {
    if (workflow) {
      loadWorkflowData(workflow)
    }
  }, [workflow, loadWorkflowData])

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-[100] flex items-center justify-center p-4 md:p-6 bg-slate-950/40 backdrop-blur-sm select-none animate-in fade-in duration-150">
      <div className="w-full max-w-[92vw] h-[85vh] rounded-xl border border-border bg-card shadow-2xl flex flex-col overflow-hidden animate-in zoom-in-95 duration-150">
        {/* Modal Header */}
        <div className="px-5 py-4 border-b border-border flex items-center justify-between bg-muted/10 shrink-0">
          <div>
            <h2 className="text-sm font-bold text-foreground flex items-center gap-2">
              <Eye className="h-4 w-4 text-primary" />
              Preview Workflow: {workflowName}
            </h2>
            <p className="text-xs text-muted-foreground mt-0.5">
              {workflow?.description || "Visual preview of the linked workflow graph."}
            </p>
          </div>
          <button
            onClick={onClose}
            className="flex h-7 w-7 items-center justify-center rounded-lg hover:bg-muted text-muted-foreground transition-colors"
            title="Close Dialog"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Modal Content (Canvas) */}
        <div className="flex-1 min-h-0 relative">
          <WorkflowDesigner
            nodes={nodes}
            edges={edges}
            readOnly={true}
            onNodesChange={onNodesChange}
            isLoading={isLoading}
          />
        </div>

        {/* Modal Footer */}
        <div className="px-5 py-3 border-t border-border flex justify-between items-center bg-muted/5 shrink-0">
          <span className="text-[10px] text-muted-foreground font-mono">
            Framework: {workflow?.framework || "ADK 2.0"}
          </span>
          <div className="flex gap-2">
            <button
              onClick={onClose}
              className="inline-flex h-9 items-center justify-center rounded-lg border border-input bg-background px-4 text-xs font-semibold text-foreground hover:bg-muted transition-colors"
            >
              Close
            </button>
            <button
              onClick={() => {
                onUnlink()
                onClose()
              }}
              className="inline-flex h-9 items-center justify-center gap-1.5 rounded-lg bg-destructive px-4 text-xs font-semibold text-destructive-foreground shadow hover:bg-destructive/90 transition-all duration-200"
            >
              <Link2Off className="h-3.5 w-3.5" />
              Unlink Workflow
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
