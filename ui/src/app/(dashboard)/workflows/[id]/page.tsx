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

import { use, useEffect, useState, useRef, useCallback } from "react"
import { useRouter } from "next/navigation"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { ArrowLeft, Save, Workflow as WorkflowIcon, Trash2, Plus, Play, HelpCircle, X, ChevronRight, ChevronDown, MousePointerClick, Download, Upload } from "lucide-react"
import { useAppStore } from "@/stores/app"
import { useAuthStore } from "@/stores/auth"
import { toast } from "sonner"
import { api } from "@/lib/api"
import ConfirmDialog from "@/components/shared/ConfirmDialog"
import { isValidPythonIdentifier } from "@/lib/utils"
import {
  ResizablePanelGroup,
  ResizablePanel,
  ResizableHandle,
} from "@/components/ui/resizable"
import {
  useNodesState,
  useEdgesState,
  addEdge,
  reconnectEdge,
  Connection,
  Edge,
  Node,
} from "@xyflow/react"
import "@xyflow/react/dist/style.css"
import {
  WorkflowTemplate,
  AdkNode,
  AdkEdge,
  VisualNode,
  VisualEdge,
  Agent,
  FunctionNode,
  JoinNode,
  CustomNode,
  LoopNode,
  ScheduledTrigger,
  ManualTrigger,
  FormTrigger,
} from "@/lib/types"
import { NodeToolbox, PredefinedAgent } from "../components/NodeToolbox"
import {
  WorkflowDesigner,
  DesignerNode,
  DesignerEdge,
} from "../components/WorkflowDesigner"
import { WorkflowPropertiesPanel } from "../components/WorkflowPropertiesPanel"



export default function WorkflowDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id: workflowId } = use(params)
  const router = useRouter()
  const queryClient = useQueryClient()
  const { setPageTitle } = useAppStore()
  const { isAuthenticated } = useAuthStore()
  const isNew = workflowId === "new"

  const containerRef = useRef<HTMLDivElement>(null)
  const reactFlowInstanceRef = useRef<any>(null)
  const handleSaveRef = useRef<() => void>(() => {})
  const [leftPanelSize, setLeftPanelSize] = useState(22)
  const [leftPanelMinSize, setLeftPanelMinSize] = useState(18)

  // ReactFlow Nodes and Edges State
  const [nodes, setNodes, onNodesChange] = useNodesState<DesignerNode>([])
  const [edges, setEdges, onEdgesChange] = useEdgesState<DesignerEdge>([])

  // Designer State
  const [workflowName, setWorkflowName] = useState("")
  const [workflowDisplayName, setWorkflowDisplayName] = useState("")
  const [workflowDescription, setWorkflowDescription] = useState("")
  const [framework, setFramework] = useState("ADK 2.0")
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null)
  const [selectedEdgeId, setSelectedEdgeId] = useState<string | null>(null)
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false)
  


  // Adjust left panel size dynamically
  useEffect(() => {
    if (containerRef.current) {
      const containerWidth = containerRef.current.getBoundingClientRect().width
      if (containerWidth > 0) {
        setLeftPanelSize((280 / containerWidth) * 100)
        setLeftPanelMinSize((240 / containerWidth) * 100)
      }
    }
  }, [])

  useEffect(() => {
    setPageTitle(isNew ? "Create Workflow" : "Workflow Details")
  }, [setPageTitle, isNew])

  // --- Fetch Workflow from Backend ---
  const { data: workflow, isLoading, error } = useQuery<WorkflowTemplate>({
    queryKey: ["workflow", workflowId],
    queryFn: async () => {
      const res = await api.get(`/api/v1/workflows/${workflowId}`)
      if (!res.ok) {
        throw new Error("Failed to load workflow details")
      }
      return res.json()
    },
    enabled: !isNew,
  })

  // Initialize canvas for new workflows
  const initializeDefaultGraph = useCallback(() => {
    setWorkflowName(isNew ? "" : "Legacy Workflow")
    setWorkflowDisplayName(isNew ? "" : "Legacy Workflow")
    setWorkflowDescription("")
    setNodes([])
    setEdges([])
  }, [isNew])

  const loadWorkflowData = useCallback((wfData: any) => {
    if (!wfData) return

    if (wfData.name) setWorkflowName(wfData.name)
    if (wfData.displayName) setWorkflowDisplayName(wfData.displayName)
    if (wfData.description) setWorkflowDescription(wfData.description)
    if (wfData.framework) setFramework(wfData.framework)

    if (wfData.framework !== "BMAD") {
      if (wfData.layout && wfData.layout.nodes) {
        // A. Restore canvas from Visual Layout (Separated visual state)
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
            const logicalNode = (wfData.nodes || []).find((ln: any) => ln.id === vn.id) || {}
            const initData = {
              ...(vn.data || {}),
              ...logicalNode,
              id: vn.id,
              name: logicalNode.name || vn.data?.name || vn.id,
              displayName: logicalNode.displayName || vn.data?.displayName,
              type: vn.type,
              skills: logicalNode.skills || vn.data?.skills || [],
              tools: logicalNode.tools || vn.data?.tools || [],
              agentId: logicalNode.agentId || vn.data?.agentId,
              model: logicalNode.model || vn.data?.model,
              instruction: logicalNode.instruction || vn.data?.instruction,
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
        // B. Legacy Fallback
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
      } else {
        initializeDefaultGraph()
      }
    } else {
      initializeDefaultGraph()
    }
  }, [setNodes, setEdges, initializeDefaultGraph])

  useEffect(() => {
    if (workflow) {
      loadWorkflowData(workflow)
    }
  }, [workflow, loadWorkflowData])

  useEffect(() => {
    if (error) {
      toast.error("Failed to load workflow details. Mounting fallback canvas.")
      initializeDefaultGraph()
    }
  }, [error, initializeDefaultGraph])

  // Initialize for new workflow
  useEffect(() => {
    if (isNew) {
      initializeDefaultGraph()
    }
  }, [isNew, initializeDefaultGraph])

  // --- Mutate Actions (Save & Delete) ---
  const saveMutation = useMutation({
    mutationFn: async (wf: WorkflowTemplate) => {
      const res = isNew
        ? await api.post("/api/v1/workflows", wf)
        : await api.put(`/api/v1/workflows/${workflowId}`, wf)
      if (!res.ok) {
        throw new Error("Failed to save workflow changes")
      }
      return res.json()
    },
    onSuccess: (data) => {
      toast.success("Workflow saved successfully")
      queryClient.invalidateQueries({ queryKey: ["workflows"] })
      if (isNew && data?.id) {
        queryClient.invalidateQueries({ queryKey: ["workflow", data.id] })
        router.replace(`/workflows/${data.id}`)
      } else {
        queryClient.invalidateQueries({ queryKey: ["workflow", workflowId] })
      }
    },
    onError: (err: Error) => {
      toast.error(err.message || "Failed to save workflow")
    },
  })

  const deleteMutation = useMutation({
    mutationFn: async () => {
      const res = await api.delete(`/api/v1/workflows/${workflowId}`)
      if (!res.ok) {
        throw new Error("Failed to delete workflow")
      }
    },
    onSuccess: () => {
      toast.success("Workflow deleted successfully")
      queryClient.invalidateQueries({ queryKey: ["workflows"] })
      router.push("/workflows")
    },
    onError: (err: Error) => {
      toast.error(err.message || "Failed to delete workflow")
    },
  })

  // --- ReactFlow Connection Handlers ---
  const onConnect = useCallback(
    (params: Connection) => {
      setEdges((eds) => addEdge({ ...params, animated: false }, eds))
    },
    [setEdges]
  )

  const onReconnect = useCallback(
    (oldEdge: DesignerEdge, newConnection: Connection) => {
      setEdges((els) => reconnectEdge(oldEdge, newConnection, els))
    },
    [setEdges]
  )

  const onNodeClick = useCallback((_: any, node: DesignerNode) => {
    if (node.type === "noop") {
      setSelectedNodeId(null)
    } else {
      setSelectedNodeId(node.id)
    }
    setSelectedEdgeId(null)
  }, [])

  const onEdgeClick = useCallback((_: any, edge: DesignerEdge) => {
    setSelectedEdgeId(edge.id)
    setSelectedNodeId(null)
  }, [])

  const onPaneClick = useCallback(() => {
    setSelectedNodeId(null)
    setSelectedEdgeId(null)
  }, [])



  const onEdgeDoubleClick = useCallback(
    (event: React.MouseEvent, edge: DesignerEdge) => {
      if (!reactFlowInstanceRef.current) return

      // Convert screen click coordinates to ReactFlow canvas coordinates
      const position = reactFlowInstanceRef.current.screenToFlowPosition({
        x: event.clientX,
        y: event.clientY,
      })

      // Snap the coordinates to the 14px grid to align with other nodes
      const snappedPosition = {
        x: Math.round(position.x / 14) * 14,
        y: Math.round(position.y / 14) * 14,
      }

      // Find source and target nodes to determine best handle orientations
      const sourceNode = nodes.find((n) => n.id === edge.source)
      const targetNode = nodes.find((n) => n.id === edge.target)

      const srcPos = sourceNode?.position || { x: position.x - 100, y: position.y }
      const tgtPos = targetNode?.position || { x: position.x + 100, y: position.y }

      // Determine best target handle on waypoint (where the incoming line enters)
      const dx1 = srcPos.x - position.x
      const dy1 = srcPos.y - position.y
      let targetHandle = "target-left" // fallback
      if (Math.abs(dx1) > Math.abs(dy1)) {
        targetHandle = dx1 < 0 ? "target-left" : "target-right"
      } else {
        targetHandle = dy1 < 0 ? "target-top" : "target-bottom"
      }

      // Determine best source handle on waypoint (where the outgoing line leaves)
      const dx2 = tgtPos.x - position.x
      const dy2 = tgtPos.y - position.y
      let sourceHandle = "source-right" // fallback
      if (Math.abs(dx2) > Math.abs(dy2)) {
        sourceHandle = dx2 > 0 ? "source-right" : "source-left"
      } else {
        sourceHandle = dy2 > 0 ? "source-bottom" : "source-top"
      }

      // Create new Waypoint Node
      const waypointId = `waypoint-${Date.now()}`
      const newWaypointNode = {
        id: waypointId,
        type: "waypoint",
        position: snappedPosition,
        data: {
          name: "WAYPOINT",
          displayName: "Waypoint",
        },
      }

      // Split the edge: source -> waypoint and waypoint -> target
      const edge1 = {
        id: `edge-${edge.source}-${waypointId}-${Date.now()}`,
        source: edge.source,
        target: waypointId,
        sourceHandle: edge.sourceHandle || undefined,
        targetHandle: targetHandle,
      }

      const edge2 = {
        id: `edge-${waypointId}-${edge.target}-${Date.now() + 1}`,
        source: waypointId,
        target: edge.target,
        sourceHandle: sourceHandle,
        targetHandle: edge.targetHandle || undefined,
        label: edge.label,
        data: edge.data,
      }

      setNodes((nds) => [...nds, newWaypointNode])
      setEdges((eds) => [
        ...eds.filter((e) => e.id !== edge.id), // Remove original edge
        edge1,
        edge2,
      ])

      setSelectedNodeId(waypointId)
      setSelectedEdgeId(null)
      toast.success("Added routing anchor to path")
    },
    [nodes, setNodes, setEdges]
  )

  // --- Designer Action Helpers ---

  // Add Predefined Agent to Canvas
  const handleAddPredefinedAgent = (agent: PredefinedAgent) => {
    const roleSlug = agent.role.toLowerCase().replace(/\s+/g, "_")
    // Generate unique name
    let nodeName = roleSlug
    let counter = 1
    while (nodes.some((n) => n.data?.name === nodeName)) {
      nodeName = `${roleSlug}_${counter}`
      counter++
    }

    const newNodeData = new Agent({
      name: nodeName,
      displayName: agent.role,
      agentId: agent.id,
      description: agent.description,
      model: "gemini-2.5-flash",
      instruction: `You are playing the role of ${agent.role}. ${agent.description}`,
      skills: agent.skills,
      tools: agent.tools,
    })

    const newNode = {
      id: `agent-${Date.now()}`,
      type: "agent",
      position: { x: 250, y: 150 },
      data: newNodeData,
    }

    setNodes((nds) => [...nds, newNode])
    setSelectedNodeId(newNode.id)
    setSelectedEdgeId(null)
    toast.success(`Added Agent node: ${nodeName}`)
  }

  // Add Blank Control Node to Canvas
  const handleAddControlNode = (type: 'agent' | 'function' | 'join' | 'trigger' | 'loop' | 'noop' | 'manual_trigger' | 'form_trigger') => {
    const nodePrefix = type === "agent" ? "llm_agent" : type === "function" ? "custom_fn" : type === "join" ? "join_sync" : type === "loop" ? "loop_items" : type === "noop" ? "noop_route" : type === "manual_trigger" ? "manual_run" : type === "form_trigger" ? "form_run" : "schedule_trigger"
    let nodeName = nodePrefix
    let counter = 1
    while (nodes.some((n) => n.data?.name === nodeName)) {
      nodeName = `${nodePrefix}_${counter}`
      counter++
    }

    let newNodeData: AdkNode
    if (type === "agent") {
      newNodeData = new Agent({
        name: nodeName,
        displayName: "Generic LLM",
        description: "Conversational Agent",
        model: "gemini-2.5-flash",
        instruction: "A helpful assistant.",
      })
    } else if (type === "function") {
      newNodeData = new FunctionNode({
        name: nodeName,
        displayName: "Function Node",
        description: "Python function binding",
        parameterBinding: "state",
      })
    } else if (type === "join") {
      newNodeData = new JoinNode({
        name: nodeName,
        displayName: "Join Node",
        description: "Wait for predecessors",
      })
    } else if (type === "loop") {
      newNodeData = new LoopNode({
        name: nodeName,
        displayName: "Loop Over Items",
        description: "Iterate over a collection",
      })
    } else if (type === "noop") {
      newNodeData = new AdkNode({
        name: nodeName,
        displayName: "Noop Node",
        type: "noop",
        description: "No-operation routing node",
      })
    } else if (type === "manual_trigger") {
      newNodeData = new ManualTrigger({
        name: nodeName,
        displayName: "Manual Trigger",
        description: "Manually trigger the workflow",
      })
    } else if (type === "form_trigger") {
      newNodeData = new FormTrigger({
        name: nodeName,
        displayName: "Form Trigger",
        description: "Trigger the workflow via form submission",
      })
    } else {
      newNodeData = new ScheduledTrigger({
        name: nodeName,
        displayName: "Scheduled Trigger",
        description: "Runs on a recurring schedule",
        triggerInterval: "custom",
        cronExpression: "*/5 * * * *",
      })
    }

    const newNode = {
      id: `${type}-${Date.now()}`,
      type,
      position: { x: 252, y: 154 },
      data: newNodeData,
    }

    setNodes((nds) => [...nds, newNode])
    setSelectedNodeId(newNode.id)
    setSelectedEdgeId(null)
    toast.success(`Created node: ${nodeName}`)
  }

  // Update properties for selected Node
  const handleUpdateNodeData = (field: string, val: any) => {
    if (!selectedNodeId) return
    setNodes((nds) =>
      nds.map((n) => {
        if (n.id === selectedNodeId) {
          let updatedData: AdkNode
          if (n.type === "agent") {
            updatedData = new Agent({
              ...n.data,
              [field]: val,
            } as any)
          } else {
            const Cls = (n.data && typeof n.data.constructor === "function" && n.data.constructor !== Object)
              ? (n.data.constructor as new (init: any) => AdkNode)
              : CustomNode
            updatedData = new Cls({
              ...n.data,
              [field]: val,
            })
          }
          return {
            ...n,
            data: updatedData,
          }
        }
        return n
      })
    )
  }

  // Update edge route label
  const handleUpdateEdgeRoute = (val: string) => {
    if (!selectedEdgeId) return
    setEdges((eds) =>
      eds.map((e) => {
        if (e.id === selectedEdgeId) {
          return {
            ...e,
            label: val || undefined,
            data: { route: val },
          }
        }
        return e
      })
    )
  }

  // Delete selected Node or Edge
  const handleDeleteSelected = useCallback(() => {
    const selectedNodes = nodes.filter((n) => n.selected || n.id === selectedNodeId)
    const selectedEdges = edges.filter((e) => e.selected || e.id === selectedEdgeId)

    if (selectedNodes.length > 0) {
      const selectedNodeIds = new Set(selectedNodes.map((n) => n.id))
      setNodes((nds) => nds.filter((n) => !selectedNodeIds.has(n.id)))
      setEdges((eds) => eds.filter((e) => !selectedNodeIds.has(e.source) && !selectedNodeIds.has(e.target)))
      
      if (selectedNodeId && selectedNodeIds.has(selectedNodeId)) {
        setSelectedNodeId(null)
      }
      toast.success(selectedNodes.length === 1 ? "Node deleted" : `${selectedNodes.length} nodes deleted`)
    } else if (selectedEdges.length > 0) {
      const selectedEdgeIds = new Set(selectedEdges.map((e) => e.id))
      setEdges((eds) => eds.filter((e) => !selectedEdgeIds.has(e.id)))
      
      if (selectedEdgeId && selectedEdgeIds.has(selectedEdgeId)) {
        setSelectedEdgeId(null)
      }
      toast.success(selectedEdges.length === 1 ? "Edge connection deleted" : `${selectedEdges.length} edges deleted`)
    }
  }, [nodes, edges, selectedNodeId, selectedEdgeId, setNodes, setEdges])

  // Keyboard binding for saving workflow or deleting selected node/edge
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "s") {
        event.preventDefault()
        handleSaveRef.current()
        return
      }
      if (event.key === "Delete" || event.key === "Backspace") {
        const activeEl = document.activeElement
        const isInput =
          activeEl &&
          (activeEl.tagName === "INPUT" ||
            activeEl.tagName === "TEXTAREA" ||
            activeEl.tagName === "SELECT" ||
            activeEl.getAttribute("contenteditable") === "true")

        if (!isInput) {
          if (selectedNodeId || selectedEdgeId) {
            event.preventDefault()
            handleDeleteSelected()
          }
        }
      }
    }

    window.addEventListener("keydown", handleKeyDown)
    return () => {
      window.removeEventListener("keydown", handleKeyDown)
    }
  }, [selectedNodeId, selectedEdgeId, handleDeleteSelected])

  // --- Save / Handoff Handler ---
  // --- Graph Traversal Compiler (Visual -> Logical ADK 2.0 Graph) ---
  const compileVisualGraph = (
    rfNodes: Node[],
    rfEdges: Edge[]
  ): { logicalNodes: AdkNode[]; logicalEdges: AdkEdge[] } => {
    // Filter out visual routing nodes and keep only execution nodes
    const executionNodes = rfNodes
      .filter((n) => n.type !== "waypoint" && n.type !== "noop")
      .map((n) => {
        const data = (n.data || {}) as any
        
        // Re-instantiate as pure execution data
        return {
          id: n.id,
          name: n.id === "start" ? "START" : (data.name as string),
          displayName: data.displayName as string || undefined,
          agentId: data.agentId as string || undefined,
          type: n.type as 'agent' | 'function' | 'join' | 'custom' | 'trigger' | 'loop' | 'manual_trigger' | 'form_trigger',
          description: data.description as string || "",
          rerunOnResume: !!data.rerunOnResume,
          waitForOutput: !!data.waitForOutput,
          timeout: data.timeout ? Number(data.timeout) : undefined,
          retryConfig: data.retryConfig ? {
            maxAttempts: data.retryConfig.maxAttempts ? Number(data.retryConfig.maxAttempts) : undefined,
            initialDelay: data.retryConfig.initialDelay ? Number(data.retryConfig.initialDelay) : undefined,
            maxDelay: data.retryConfig.maxDelay ? Number(data.retryConfig.maxDelay) : undefined,
            backoffFactor: data.retryConfig.backoffFactor ? Number(data.retryConfig.backoffFactor) : undefined,
            jitter: data.retryConfig.jitter ? Number(data.retryConfig.jitter) : undefined,
            exceptions: data.retryConfig.exceptions || [],
          } : undefined,
          model: n.type === "agent" ? (data.model as string || "gemini-2.5-flash") : undefined,
          instruction: n.type === "agent" ? (data.instruction as string || "") : undefined,
          skills: n.type === "agent" ? (data.skills || []) : undefined,
          tools: n.type === "agent" ? (data.tools || []) : undefined,
          input: data.input || undefined,
          output: data.output || undefined,
          parameterBinding: n.type === "function" ? (data.parameterBinding as 'state' | 'node_input' || "state") : undefined,
          authConfig: n.type === "function" ? (data.authConfig || {}) : undefined,
          parallelWorker: n.type === "custom" ? !!data.parallelWorker : undefined,
          // Trigger specific mapping
          triggerInterval: n.type === "trigger" ? (data.triggerInterval as any) : undefined,
          intervalValue: n.type === "trigger" ? Number(data.intervalValue) : undefined,
          triggerAtMinute: n.type === "trigger" && data.triggerAtMinute !== undefined ? Number(data.triggerAtMinute) : undefined,
          triggerAtHour: n.type === "trigger" && data.triggerAtHour !== undefined ? Number(data.triggerAtHour) : undefined,
          triggerOnWeekdays: n.type === "trigger" ? (data.triggerOnWeekdays as number[]) : undefined,
          triggerAtDayOfMonth: n.type === "trigger" && data.triggerAtDayOfMonth !== undefined ? Number(data.triggerAtDayOfMonth) : undefined,
          cronExpression: n.type === "trigger" ? (data.cronExpression as string) : undefined,
          exitCondition: n.type === "loop" ? (data.exitCondition || {}) : undefined,
        } as AdkNode
      })

    const resolvedEdges: AdkEdge[] = []
    const visited = new Set<string>()

    const tracePath = (startNodeId: string, currentId: string, accumulatedRoute?: any) => {
      const outgoingEdges = rfEdges.filter((e) => e.source === currentId)

      outgoingEdges.forEach((edge) => {
        const targetNode = rfNodes.find((n) => n.id === edge.target)
        if (!targetNode) return

        // Extract route name from source handle if it's a concrete execution node with named handles
        // (like LoopNode with 'loop'/'done' handles), and no explicit route has been typed.
        const sourceNode = rfNodes.find((n) => n.id === edge.source)
        const isVisualOnly = sourceNode?.type === "waypoint" || sourceNode?.type === "noop"
        const sourceHandle = edge.sourceHandle
        const handleRoute = !isVisualOnly && sourceHandle && !["source", "output", "input", "target"].includes(sourceHandle)
          ? sourceHandle
          : undefined

        const route = edge.data?.route || accumulatedRoute || handleRoute

        if (targetNode.type === "waypoint" || targetNode.type === "noop") {
          const visitKey = `${startNodeId}->${targetNode.id}`
          if (visited.has(visitKey)) return
          visited.add(visitKey)

          tracePath(startNodeId, targetNode.id, route)
        } else {
          resolvedEdges.push({
            id: `${startNodeId}->${targetNode.id}`,
            fromNode: startNodeId,
            toNode: targetNode.id,
            route: route || undefined,
          })
        }
      })
    }

    // Trace paths for all execution nodes (including start)
    rfNodes
      .filter((n) => n.type !== "waypoint" && n.type !== "noop")
      .forEach((n) => tracePath(n.id, n.id))

    // Auto-inject START node for trigger-based/start-less workflows
    const targetNodeIds = new Set(resolvedEdges.map((e) => e.toNode))
    const rootNodes = executionNodes.filter((n) => n.id !== "start" && !targetNodeIds.has(n.id))
    
    if (!executionNodes.some((n) => n.id === "start") && rootNodes.length > 0) {
      // Inject logical START node
      executionNodes.push({
        id: "start",
        name: "START",
        displayName: "START",
        type: "custom",
        description: "Auto-injected entry point",
      } as AdkNode)

      // Connect START to all root nodes
      rootNodes.forEach((rn) => {
        resolvedEdges.push({
          id: `start->${rn.id}`,
          fromNode: "start",
          toNode: rn.id,
        })
      })
    }

    return { logicalNodes: executionNodes, logicalEdges: resolvedEdges }
  }

  // --- Validation Helper ---
  const validateWorkflow = (): boolean => {
    if (!workflowDisplayName.trim() && !workflowName.trim()) {
      toast.error("Please enter a workflow Display Name or ADK Name")
      return false
    }
    if (!workflowName.trim()) {
      toast.error("Workflow ADK Name is required")
      return false
    }
    if (!isValidPythonIdentifier(workflowName)) {
      toast.error("Workflow ADK Name must be a valid Python identifier (starts with letter/underscore, alphanumeric/underscore only, and not a Python keyword)")
      return false
    }

    // Check unique node names and Python identifiers (only for execution nodes)
    const nameSet = new Set<string>()
    for (const node of nodes) {
      if (node.id === "start") continue
      if (node.type === "waypoint" || node.type === "noop") continue // Skip visual routing nodes
      
      const name = node.data?.name as string
      if (!name || !name.trim()) {
        toast.error(`Node ${node.id} has an empty name`)
        return false
      }
      if (!isValidPythonIdentifier(name)) {
        toast.error(`Node name "${name}" is invalid. Must be a valid Python identifier (starts with letter/underscore, alphanumeric/underscore only, and not a Python keyword).`)
        return false
      }
      if (nameSet.has(name)) {
        toast.error(`Duplicate node name detected: "${name}". All node names must be unique.`)
        return false
      }
      nameSet.add(name)
    }

    // Check routing and connectivity (warning for floating execution nodes)
    const activeNodes = nodes.filter((n) => n.id !== "start" && n.type !== "waypoint" && n.type !== "noop")
    const connectedNodeIds = new Set<string>()
    edges.forEach((e) => {
      connectedNodeIds.add(e.source)
      connectedNodeIds.add(e.target)
    })
    const unconnectedNode = activeNodes.find((n) => !connectedNodeIds.has(n.id))
    if (unconnectedNode) {
      toast.warning(`Node "${unconnectedNode.data?.name}" is not connected to any other node.`)
    }

    return true
  }

  // --- Compile Workflow Template Helper ---
  const compileWorkflowTemplate = (): WorkflowTemplate => {
    const { logicalNodes: adkNodes, logicalEdges: adkEdges } = compileVisualGraph(nodes, edges)

    // Serialize Visual Layout State
    const visualNodes: VisualNode[] = nodes.map((n) => {
      const data = n.data || {}
      return {
        id: n.id,
        type: n.type || "custom",
        position: n.position,
        orientation: data.orientation || undefined,
        data: {
          name: data.name || undefined,
          displayName: data.displayName || undefined,
          agentId: (data as any).agentId || undefined,
          skills: (data as any).skills || [],
          tools: (data as any).tools || [],
          model: (data as any).model || undefined,
          instruction: (data as any).instruction || undefined,
        },
      }
    })

    const visualEdges: VisualEdge[] = edges.map((e) => ({
      id: e.id,
      source: e.source,
      target: e.target,
      sourceHandle: e.sourceHandle || undefined,
      targetHandle: e.targetHandle || undefined,
      route: e.data?.route || undefined,
    }))

    return {
      id: isNew ? "" : workflowId,
      name: workflowName,
      displayName: workflowDisplayName,
      description: workflowDescription,
      framework: framework,
      phases: {},
      artifacts: {},
      dependencies: {},
      nodes: adkNodes,
      edges: adkEdges,
      layout: {
        nodes: visualNodes,
        edges: visualEdges,
      },
    }
  }

  // --- Save Handler ---
  const handleSave = () => {
    if (!validateWorkflow()) return
    const finalWorkflow = compileWorkflowTemplate()
    saveMutation.mutate(finalWorkflow)
  }

  useEffect(() => {
    handleSaveRef.current = handleSave
  })

  // --- Export Handler ---
  const handleExport = () => {
    if (!validateWorkflow()) return
    const finalWorkflow = compileWorkflowTemplate()

    try {
      // Convert to JSON and trigger browser download
      const dataStr = "data:text/json;charset=utf-8," + encodeURIComponent(JSON.stringify(finalWorkflow, null, 2))
      const downloadAnchor = document.createElement("a")
      downloadAnchor.setAttribute("href", dataStr)
      
      // Generate a safe snake_case filename
      const safeFileName = workflowName.toLowerCase().replace(/[^a-z0-9_]/g, "_")
      downloadAnchor.setAttribute("download", `${safeFileName}_workflow.json`)
      
      document.body.appendChild(downloadAnchor)
      downloadAnchor.click()
      downloadAnchor.remove()
      
      toast.success("Workflow definition exported successfully!")
    } catch (err) {
      console.error("Failed to export workflow:", err)
      toast.error("Failed to export workflow. Please try again.")
    }
  }

  // --- Import Handler ---
  const handleImport = () => {
    const fileInput = document.createElement("input")
    fileInput.type = "file"
    fileInput.accept = ".json"

    fileInput.onchange = (e: any) => {
      const file = e.target.files?.[0]
      if (!file) return

      const reader = new FileReader()
      reader.onload = (event: any) => {
        try {
          const importedWorkflow = JSON.parse(event.target.result)
          
          if (!importedWorkflow || typeof importedWorkflow !== "object") {
            throw new Error("Invalid file format. Must be a JSON object.")
          }
          
          const hasNodes = (importedWorkflow.nodes && Array.isArray(importedWorkflow.nodes)) || 
                           (importedWorkflow.layout?.nodes && Array.isArray(importedWorkflow.layout.nodes))
                           
          if (!hasNodes) {
            throw new Error("Invalid workflow template. No nodes found.")
          }

          loadWorkflowData(importedWorkflow)
          toast.success("Workflow imported successfully!")
        } catch (err: any) {
          console.error("Failed to import workflow:", err)
          toast.error(`Import failed: ${err.message || "Invalid JSON file"}`)
        }
      }
      reader.readAsText(file)
    }

    fileInput.click()
  }

  const handleDeleteWorkflow = () => {
    setIsDeleteModalOpen(true)
  }

  // Get currently selected items
  return (
    <div className="flex flex-col h-full w-full bg-slate-50 dark:bg-slate-950 overflow-hidden">
      {/* Designer Header */}
      <div className="p-6 border-b border-border bg-card flex items-center justify-between shrink-0 select-none">
        <div className="flex items-center gap-4 flex-1">
          <button
            onClick={() => router.push("/workflows")}
            className="p-2 hover:bg-muted rounded-lg transition-colors text-muted-foreground hover:text-foreground border border-border bg-background"
            title="Back to Workflows"
          >
            <ArrowLeft className="h-5 w-5" />
          </button>
          
          <div className="flex-1 max-w-md">
            <div className="flex items-center gap-2">
              <WorkflowIcon className="h-5 w-5 text-primary" />
              <input
                type="text"
                value={workflowDisplayName}
                onChange={(e) => {
                  const newVal = e.target.value
                  setWorkflowDisplayName(newVal)
                  const oldSlug = workflowDisplayName.toLowerCase().replace(/[^a-z0-9_]/g, "_")
                  if (!workflowName || workflowName === oldSlug) {
                    const newSlug = newVal.toLowerCase().replace(/[^a-z0-9_]/g, "_")
                    if (/^[a-z_]/.test(newSlug)) {
                      setWorkflowName(newSlug)
                    }
                  }
                }}
                placeholder="Untitled ADK Workflow"
                className="text-lg font-bold bg-transparent border-b border-transparent hover:border-border focus:border-primary focus:outline-none px-1 py-0.5 rounded transition-all w-full truncate text-foreground"
              />
            </div>
            <p className="text-muted-foreground text-[10px] uppercase tracking-wider font-semibold mt-0.5 ml-7">
              Target Runtime: {framework} | {isNew ? "New Workflow" : `ID: ${workflowId}`}
            </p>
          </div>
        </div>

        <div className="flex gap-2.5">


          <button
            onClick={() => router.push("/workflows")}
            className="inline-flex h-9 items-center justify-center rounded-lg border border-input bg-background px-4 py-2 text-xs font-semibold text-foreground hover:bg-muted transition-colors"
          >
            Cancel
          </button>

          {!isNew && isAuthenticated && (
            <button
              onClick={handleDeleteWorkflow}
              disabled={deleteMutation.isPending}
              className="inline-flex h-9 items-center justify-center gap-1.5 rounded-lg border border-destructive bg-transparent px-4 py-2 text-xs font-semibold text-destructive hover:bg-destructive/10 transition-colors disabled:opacity-50"
            >
              <Trash2 className="h-3.5 w-3.5" />
              Delete
            </button>
          )}

          <button
            onClick={handleImport}
            className="inline-flex h-9 items-center justify-center gap-1.5 rounded-lg border border-input bg-background px-4 py-2 text-xs font-semibold text-foreground hover:bg-muted transition-colors"
          >
            <Download className="h-3.5 w-3.5" />
            Import
          </button>

          <button
            onClick={handleExport}
            className="inline-flex h-9 items-center justify-center gap-1.5 rounded-lg border border-input bg-background px-4 py-2 text-xs font-semibold text-foreground hover:bg-muted transition-colors"
          >
            <Upload className="h-3.5 w-3.5" />
            Export
          </button>

          <button
            onClick={handleSave}
            disabled={saveMutation.isPending}
            className="inline-flex h-9 items-center justify-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-xs font-semibold text-primary-foreground shadow transition-all hover:opacity-90 disabled:opacity-50"
          >
            <Save className="h-3.5 w-3.5" />
            {isNew ? "Save Workflow" : "Save Changes"}
          </button>
        </div>
      </div>

      {/* Editor Surface split into 3 columns (Toolbox, Canvas, Properties) */}
      <div ref={containerRef} className="flex-1 overflow-hidden flex">
        <ResizablePanelGroup
            key={leftPanelSize}
            direction="horizontal"
            className="flex-1 overflow-hidden"
          >
          {/* 1. Left Panel - Agent & Tools */}
          <ResizablePanel defaultSize={leftPanelSize} minSize={leftPanelMinSize} maxSize={30}>
            <NodeToolbox
              onAddControlNode={handleAddControlNode}
              onAddPredefinedAgent={handleAddPredefinedAgent}
            />
          </ResizablePanel>

          <ResizableHandle withHandle />

          {/* 2. Center Panel - ReactFlow Graph Canvas */}
          <ResizablePanel defaultSize={selectedNodeId || selectedEdgeId ? 53 : 78} minSize={40}>
            <WorkflowDesigner
              nodes={nodes}
              edges={edges}
              onNodesChange={onNodesChange}
              onEdgesChange={onEdgesChange}
              onConnect={onConnect}
              onReconnect={onReconnect}
              onNodeClick={onNodeClick}
              onEdgeClick={onEdgeClick}
              onPaneClick={onPaneClick}
              onEdgeDoubleClick={onEdgeDoubleClick}
              onInit={(instance: any) => {
                reactFlowInstanceRef.current = instance
              }}
              isLoading={isLoading}
            />
          </ResizablePanel>

          <ResizableHandle withHandle />

          {/* 3. Right Panel - Property Editor Drawer / General Settings */}
          <ResizablePanel defaultSize={24} minSize={18} maxSize={35}>
            <WorkflowPropertiesPanel
              selectedNodeId={selectedNodeId}
              selectedEdgeId={selectedEdgeId}
              nodes={nodes}
              edges={edges}
              workflowDisplayName={workflowDisplayName}
              setWorkflowDisplayName={setWorkflowDisplayName}
              workflowName={workflowName}
              setWorkflowName={setWorkflowName}
              workflowDescription={workflowDescription}
              setWorkflowDescription={setWorkflowDescription}
              framework={framework}
              setFramework={setFramework}
              handleUpdateNodeData={handleUpdateNodeData}
              handleUpdateEdgeRoute={handleUpdateEdgeRoute}
              handleDeleteSelected={handleDeleteSelected}
              onPaneClick={onPaneClick}
            />
          </ResizablePanel>
        </ResizablePanelGroup>
      </div>

      <ConfirmDialog
        isOpen={isDeleteModalOpen}
        onClose={() => setIsDeleteModalOpen(false)}
        onConfirm={() => {
          deleteMutation.mutate()
        }}
        title="Delete Workflow"
        message="Are you sure you want to permanently delete this workflow? This cannot be undone."
        confirmText="Delete"
        isDestructive={true}
        isLoading={deleteMutation.isPending}
      />
    </div>
  )
}
