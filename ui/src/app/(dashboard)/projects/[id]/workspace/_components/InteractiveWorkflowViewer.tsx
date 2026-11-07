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

import React, { useEffect, useState, useCallback, useRef, useMemo } from "react"
import { X, Play, Loader2, AlertCircle, Terminal, MessageSquare, Workflow, Trash2, ArrowLeft } from "lucide-react"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { api } from "@/lib/api"
import { toast } from "sonner"
import ConfirmDialog from "@/components/shared/ConfirmDialog"
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
import { ResizablePanelGroup, ResizablePanel, ResizableHandle } from "@/components/ui/resizable"
import { useChatStream } from "@/hooks/useChatStream"
import { cn } from "@/lib/utils"
import { DynamicFormModal } from "../../../../workflows/components/DynamicFormModal"
import ChatPane from "@/components/workspace/ChatPane"

interface InteractiveWorkflowViewerProps {
  projectId: string
  runId: string
  workflowId: string | null
  workflowName: string
  onClose: () => void
  onDeleteRun?: (runId: string) => void
}

function extractNodeNameJS(path: string): string {
  if (!path) return ""
  const segments = path.split("/")
  if (segments.length === 0) return ""
  const lastSegment = segments[segments.length - 1]
  const nodeNameSegs = lastSegment.split("@")
  if (nodeNameSegs.length === 0) return ""
  return nodeNameSegs[0]
}

const FULL_PROCESS_NODE: DesignerNode = {
  id: "full_process",
  type: "system",
  data: {
    name: "Full Process",
    displayName: "Full Process",
  },
  position: { x: 0, y: 0 },
}

export default function InteractiveWorkflowViewer({
  projectId,
  runId,
  workflowId,
  workflowName,
  onClose,
  onDeleteRun,
}: InteractiveWorkflowViewerProps) {
  const queryClient = useQueryClient()
  const [isDeleteConfirmOpen, setIsDeleteConfirmOpen] = useState(false)

  const [nodes, setNodes, onNodesChange] = useNodesState<DesignerNode>([])
  const [edges, setEdges, onEdgesChange] = useEdgesState<DesignerEdge>([])
  const [selectedNode, setSelectedNode] = useState<DesignerNode | null>(FULL_PROCESS_NODE)
  const [bottomTab, setBottomTab] = useState<"console" | "chat">("chat")
  const [activeRequest, setActiveRequest] = useState<{
    nodeId: string;
    interruptId: string;
    schema: any;
  } | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)

  const deleteRunMutation = useMutation({
    mutationFn: async () => {
      const res = await api.delete(`/api/v1/projects/${projectId}/workflow-runs/${runId}`)
      if (!res.ok) {
        const errData = await res.json().catch(() => ({}))
        throw new Error(errData.error || "Failed to delete workflow run")
      }
    },
    onSuccess: () => {
      toast.success("Workflow run deleted successfully")
      queryClient.invalidateQueries({ queryKey: ["workflow-runs", projectId] })
      onDeleteRun?.(runId)
      onClose()
    },
    onError: (err: Error) => {
      toast.error(err.message || "Failed to delete workflow run")
    },
  })

  const { messages, isLoading: isStreamLoading, isThinking } = useChatStream(projectId, "implementation", runId, {
    includeNotifications: true,
  })

  useEffect(() => {
    console.log("InteractiveWorkflowViewer: runId =", runId, "messages =", messages)
  }, [runId, messages])

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
    enabled: !!workflowId,
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
          position: { x: vn.position?.x ?? 0, y: vn.position?.y ?? 0 },
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

  // Reset selected node to FULL_PROCESS_NODE when runId or workflowId changes
  useEffect(() => {
    setSelectedNode(FULL_PROCESS_NODE)
  }, [runId, workflowId])

  const handleNodeClick = useCallback((_event: any, node: DesignerNode) => {
    // Only allow clicking logical agent/function/trigger/loop nodes that produce output
    if (node.type === "waypoint" || node.type === "noop" || node.type === "start") {
      return
    }
    setSelectedNode(node)
    setBottomTab("console")
  }, [])

  const handlePaneClick = useCallback(() => {
    setSelectedNode(FULL_PROCESS_NODE)
  }, [])

  // Compute active input requests
  const activeInputRequests = useMemo(() => {
    const requests = new Map<string, any>()
    const resolvedIds = new Set<string>()

    // First pass: find all resolved IDs (messages containing FunctionResponse)
    messages.forEach((m) => {
      if (m.functionResponse?.id) {
        resolvedIds.add(m.functionResponse.id)
      }
    })

    // Second pass: find all active requests that are not resolved
    messages.forEach((m) => {
      if (m.awaitingInput && m.event?.content?.parts?.[0]?.function_call) {
        const funcCall = m.event.content.parts[0].function_call
        if (funcCall.name === "adk_request_input") {
          const interruptId = funcCall.id
          const schema = funcCall.args?.response_schema
          const path = m.event?.node_info?.path
          const extractedName = path ? extractNodeNameJS(path) : ""
          
          const node = nodes.find(n => n.data?.name === extractedName || n.id === extractedName)

          if (node && interruptId && schema && !resolvedIds.has(interruptId)) {
            requests.set(interruptId, {
              nodeId: node.id,
              interruptId,
              schema,
            })
          }
        }
      }
    })

    const active: any[] = []
    requests.forEach((req) => {
      active.push(req)
    })
    return active
  }, [messages, nodes])

  // Wrap nodes to inject awaitingInput status while preserving identity and measurements
  const nodesWithInputStatus = useMemo(() => {
    return nodes.map((node) => {
      const activeReq = activeInputRequests.find((req) => req.nodeId === node.id)
      const isAwaiting = !!activeReq
      if (node.data?.awaitingInput === isAwaiting) {
        return node
      }
      return {
        ...node,
        data: {
          ...node.data,
          awaitingInput: isAwaiting,
        },
      }
    })
  }, [nodes, activeInputRequests])

  const handleInputIconClick = useCallback((nodeId: string) => {
    const req = activeInputRequests.find((r) => r.nodeId === nodeId)
    if (req) {
      setActiveRequest(req)
    }
  }, [activeInputRequests])

  const handleFormSubmit = async (formData: Record<string, any>) => {
    if (!activeRequest || !workflowId) return

    const currentInterruptId = activeRequest.interruptId
    setActiveRequest(null)
    toast.success("Input response submitted. Resuming workflow...")

    try {
      const res = await api.post(`/api/v1/workflows/${workflowId}/runs/${runId}/resume?projectId=${projectId}`, {
        interruptId: currentInterruptId,
        response: formData,
      })

      if (!res.ok) {
        const errText = await res.text()
        throw new Error(errText || "Failed to resume workflow")
      }
    } catch (err: any) {
      console.error("Error resuming workflow:", err)
      toast.error(`Failed to submit response: ${err.message}`)
    }
  }

  return (
    <div className="flex flex-col h-full w-full bg-background overflow-hidden">
      {/* Header */}
      <div className="px-5 py-3.5 border-b border-border flex items-center justify-between bg-muted/5 shrink-0 select-none">
        <div className="flex items-center gap-2.5">
          <button
            onClick={onClose}
            className="flex h-7 w-7 items-center justify-center rounded-lg border border-border bg-background hover:bg-muted text-muted-foreground hover:text-foreground transition-colors shrink-0"
            title="Back to workflow execution history"
          >
            <ArrowLeft className="h-4 w-4" />
          </button>
          <div className="h-7 w-7 rounded-lg bg-primary/10 flex items-center justify-center border border-primary/20 text-primary">
            <Workflow className="h-3.5 w-3.5" />
          </div>
          <div>
            <h2 className="text-xs font-bold text-foreground flex items-center gap-1.5">
              Interactive Viewer: {workflowName}
            </h2>
            <p className="text-[10px] text-muted-foreground mt-0.5 font-mono">
              Active Run ID: {runId} | Click any node to stream its execution logs.
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          {onDeleteRun && (
            <button
              onClick={() => setIsDeleteConfirmOpen(true)}
              className="inline-flex h-7 items-center gap-1.5 px-2.5 rounded-lg border border-destructive/30 bg-destructive/10 text-destructive hover:bg-destructive/20 text-[10px] font-semibold transition-colors"
              title="Delete this workflow run"
            >
              <Trash2 className="h-3 w-3" />
              Delete Run
            </button>
          )}
          <button
            onClick={onClose}
            className="flex h-7 w-7 items-center justify-center rounded-lg hover:bg-muted text-muted-foreground transition-colors"
            title="Close Viewer"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
      </div>

      {/* Split Canvas & Console */}
      <div className="flex-1 min-h-0 relative">
        <ResizablePanelGroup direction="vertical" className="h-full w-full">
          {/* Top Panel: Canvas */}
          <ResizablePanel defaultSize={selectedNode ? 40 : 100} minSize={20} className="h-full relative">
            <WorkflowDesigner
              nodes={nodesWithInputStatus}
              edges={edges}
              interactive={true}
              onNodesChange={onNodesChange}
              onNodeClick={handleNodeClick}
              onPaneClick={handlePaneClick}
              onInputIconClick={handleInputIconClick}
              isLoading={isLoading}
            />
          </ResizablePanel>

          {selectedNode && <ResizableHandle withHandle className="bg-border/60 hover:bg-primary/20 transition-colors" />}

          {/* Bottom Panel: Console / Chat */}
          {selectedNode && (
            <ResizablePanel defaultSize={60} minSize={25} className="h-full flex flex-col bg-background border-t border-border overflow-hidden">
              {/* Unified Bottom Pane Header / Tabs */}
              <div className="px-4 py-2 border-b border-border bg-muted/40 flex items-center justify-between shrink-0 select-none">
                <div className="flex items-center gap-1.5">
                  <button
                    onClick={() => setBottomTab("chat")}
                    className={cn(
                      "inline-flex items-center gap-1.5 px-3 py-1 rounded-md text-xs font-semibold transition-all",
                      bottomTab === "chat"
                        ? "bg-primary/15 text-primary border border-primary/25 shadow-sm"
                        : "text-muted-foreground hover:text-foreground hover:bg-muted/60"
                    )}
                  >
                    <MessageSquare className="h-3.5 w-3.5" />
                    <span>Agent Chat</span>
                  </button>

                  <button
                    onClick={() => setBottomTab("console")}
                    className={cn(
                      "inline-flex items-center gap-1.5 px-3 py-1 rounded-md text-xs font-semibold transition-all",
                      bottomTab === "console"
                        ? "bg-primary/15 text-primary border border-primary/25 shadow-sm"
                        : "text-muted-foreground hover:text-foreground hover:bg-muted/60"
                    )}
                  >
                    <Terminal className="h-3.5 w-3.5" />
                    <span>Console Output</span>
                    {bottomTab === "console" && (
                      <span className="ml-1 text-[10px] font-normal px-1.5 py-0.2 rounded bg-primary/20 text-primary">
                        {selectedNode.data?.displayName || selectedNode.data?.name || selectedNode.id}
                      </span>
                    )}
                  </button>
                </div>

                <div className="flex items-center gap-2">
                  {isThinking && bottomTab === "console" && (
                    <span className="flex items-center gap-1 text-[10px] text-amber-500 bg-amber-500/10 border border-amber-500/20 px-1.5 py-0.5 rounded">
                      <Loader2 className="h-3 w-3 animate-spin" />
                      Executing...
                    </span>
                  )}
                  <button
                    onClick={() => setSelectedNode(null)}
                    className="flex h-6 w-6 items-center justify-center rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
                    title="Close Bottom Pane"
                  >
                    <X className="h-4 w-4" />
                  </button>
                </div>
              </div>

              {/* Tab Content */}
              <div className="flex-1 min-h-0 relative overflow-hidden">
                {bottomTab === "chat" ? (
                  <ChatPane
                    projectId={projectId}
                    activePhase="implementation"
                    sessionId={runId}
                    isConnecting={false}
                    connectionError={null}
                    onRetry={() => {}}
                  />
                ) : (
                  <NodeStreamingConsole
                    node={selectedNode}
                    onClose={() => setSelectedNode(null)}
                    messages={messages}
                    isLoading={isStreamLoading}
                    isThinking={isThinking}
                    showHeader={false}
                  />
                )}
              </div>
            </ResizablePanel>
          )}
        </ResizablePanelGroup>
      </div>
      
      {activeRequest && (
        <DynamicFormModal
          nodeId={activeRequest.nodeId}
          interruptId={activeRequest.interruptId}
          schema={activeRequest.schema}
          onSubmit={handleFormSubmit}
          onClose={() => setActiveRequest(null)}
          isLoading={isSubmitting}
        />
      )}

      <ConfirmDialog
        isOpen={isDeleteConfirmOpen}
        onClose={() => setIsDeleteConfirmOpen(false)}
        onConfirm={() => deleteRunMutation.mutate()}
        title="Delete Workflow Run"
        message={`Are you sure you want to permanently delete execution run "${workflowName}" (${runId}) and all its execution history? This cannot be undone.`}
        confirmText="Delete Run"
        isDestructive={true}
        isLoading={deleteRunMutation.isPending}
      />
    </div>
  )
}

interface NodeStreamingConsoleProps {
  node: DesignerNode
  onClose: () => void
  messages: any[]
  isLoading: boolean
  isThinking: boolean
  showHeader?: boolean
}

function NodeStreamingConsole({ node, onClose, messages, isLoading, isThinking, showHeader = true }: NodeStreamingConsoleProps) {
  const nodeName = node.data?.name || node.id
  const displayName = node.data?.displayName || nodeName
  
  const consoleEndRef = useRef<HTMLDivElement>(null)

  const isFullProcess = node.id === "full_process"

  // Filter messages generated by this specific node, or bypass if viewing full process
  const filteredMessages = isFullProcess
    ? messages
    : messages.filter((m) => m.sender?.toLowerCase() === nodeName.toLowerCase())

  // Auto-scroll to bottom on new streaming output lines
  useEffect(() => {
    if (consoleEndRef.current) {
      consoleEndRef.current.scrollIntoView({ behavior: "smooth" })
    }
  }, [filteredMessages, isThinking])

  return (
    <div className="flex flex-col h-full w-full bg-background text-foreground overflow-hidden font-mono">
      {/* Console Header */}
      {showHeader && (
        <div className="px-4 py-2 border-b border-border bg-muted/40 flex items-center justify-between shrink-0 select-none">
          <div className="flex items-center gap-2">
            <Terminal className="h-3.5 w-3.5 text-green-500 animate-pulse" />
            <span className="text-[10px] font-bold text-muted-foreground">
              Console Output: <span className="text-primary bg-primary/10 dark:bg-primary/25 border border-primary/20 px-1.5 py-0.2 rounded font-sans ml-1">{displayName}</span>
            </span>
          </div>
          <div className="flex items-center gap-1.5">
            {isThinking && (
              <span className="flex items-center gap-1 text-[8px] text-amber-500 bg-amber-500/10 border border-amber-500/20 px-1.5 py-0.2 rounded">
                <Loader2 className="h-2.5 w-2.5 animate-spin" />
                Executing...
              </span>
            )}
            <button
              onClick={onClose}
              className="flex h-5 w-5 items-center justify-center rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
              title="Close Console"
            >
              <X className="h-3.5 w-3.5" />
            </button>
          </div>
        </div>
      )}

      {/* Console Log Stream */}
      <div className="flex-1 min-h-0 overflow-y-auto p-4 select-text">
        {isLoading ? (
          <div className="flex flex-col items-center justify-center h-full text-muted-foreground text-[10px] gap-2">
            <Loader2 className="h-4 w-4 animate-spin text-muted-foreground/70" />
            <span>Connecting to session stream...</span>
          </div>
        ) : filteredMessages.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full text-muted-foreground/80 text-[10px] gap-1.5 select-none">
            <MessageSquare className="h-5 w-5 text-muted-foreground/60" />
            <span>{isFullProcess ? "No outputs recorded for this run session yet." : "No console outputs recorded for this node yet."}</span>
            {isThinking && <span className="text-[9px] text-amber-500/70 animate-pulse">Waiting for execution to reach {displayName}...</span>}
          </div>
        ) : (
          <div className="space-y-2 leading-relaxed text-[11px] whitespace-pre-wrap font-mono">
            {filteredMessages.map((msg, idx) => (
              <div key={msg.id || idx} className="border-b border-border/40 pb-2 last:border-0 last:pb-0">
                {/* Timestamp & Line prefix */}
                <div className="text-[9px] text-muted-foreground/60 mb-0.5 select-none flex items-center gap-1.5">
                  <span>[{new Date(msg.timestamp || "").toLocaleTimeString()}]</span>
                  <span className="text-green-600 dark:text-green-400/80">{isFullProcess ? msg.sender : nodeName}#output&gt;</span>
                </div>
                {/* Output payload text */}
                <div className="text-foreground/90 pl-4 border-l border-green-500/20 dark:border-green-500/10">
                  {msg.text}
                </div>
              </div>
            ))}
            {/* Auto scroll anchor */}
            <div ref={consoleEndRef} />
          </div>
        )}
      </div>
    </div>
  )
}
