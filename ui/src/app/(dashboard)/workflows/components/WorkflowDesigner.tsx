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

import React, { useRef, useEffect, useState, useMemo, useCallback } from "react"
import { Play, MousePointerClick, Code, Workflow } from "lucide-react"
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  Handle,
  Position,
  Node,
  Edge,
  ReactFlowProvider,
  useReactFlow,
} from "@xyflow/react"
import "@xyflow/react/dist/style.css"
import { TriggerNode } from "./TriggerNode"
import { LlmNode } from "./LlmNode"
import { FunctionNode as FunctionNodeComponent } from "./FunctionNode"
import { JoinNode as JoinNodeComponent } from "./JoinNode"
import { ScheduledTrigger as ScheduledTriggerComponent } from "../nodes/triggers/ScheduledTrigger"
import { LoopNode as LoopNodeComponent } from "./LoopNode"
import { N8nEdge } from "./N8nEdge"
import { WaypointNode } from "./WaypointNode"
import { NoopNode } from "./NoopNode"
import { FormTrigger as FormTriggerComponent } from "../nodes/triggers/FormTrigger"
import { cn } from "@/lib/utils"

export interface DesignerNodeData {
  name: string
  displayName?: string
  agentId?: string
  description?: string
  model?: string
  instruction?: string
  parameterBinding?: 'state' | 'node_input'
  authConfig?: Record<string, any>
  parallelWorker?: boolean
  rerunOnResume?: boolean
  waitForOutput?: boolean
  timeout?: number
  retryConfig?: {
    maxAttempts?: number
    initialDelay?: number
    maxDelay?: number
    backoffFactor?: number
    jitter?: number
    exceptions?: string[]
  }
  [key: string]: any
}

export interface DesignerEdgeData {
  route?: string | string[] | boolean | number
  [key: string]: any
}

export type DesignerNode = Node<DesignerNodeData>
export type DesignerEdge = Edge<DesignerEdgeData>

// --- Custom ReactFlow Node Components ---
const StartNode = () => (
  <div className="w-[84px] h-[28px] flex items-center justify-center gap-1.5 rounded-full border border-green-500 bg-green-500/10 text-green-700 dark:text-green-400 font-bold text-xs shadow-md select-none relative">
    <div className="h-2 w-2 rounded-full bg-green-500 animate-pulse" />
    START
    <Handle type="source" position={Position.Right} className="w-3 h-3 !bg-green-500 border border-background" />
  </div>
)

const ManualTriggerNode = ({ id, data, selected }: any) => (
  <TriggerNode
    id={id}
    data={data}
    selected={selected}
    icon={MousePointerClick}
  />
)

const nodeTypes = {
  start: StartNode,
  agent: LlmNode,
  function: FunctionNodeComponent,
  join: JoinNodeComponent,
  trigger: ScheduledTriggerComponent,
  loop: LoopNodeComponent,
  waypoint: WaypointNode,
  noop: NoopNode,
  manual_trigger: ManualTriggerNode,
  form_trigger: FormTriggerComponent,
}

const edgeTypes = {
  smoothstep: N8nEdge,
}

export interface WorkflowDesignerProps {
  nodes: DesignerNode[]
  edges: DesignerEdge[]
  onNodesChange?: any
  onEdgesChange?: any
  onConnect?: any
  onReconnect?: any
  onNodeClick?: any
  onEdgeClick?: any
  onPaneClick?: any
  onEdgeDoubleClick?: any
  onInit?: any
  readOnly?: boolean
  interactive?: boolean
  isLoading?: boolean
  onInputIconClick?: (nodeId: string) => void
}

function WorkflowDesignerCanvas({
  nodes,
  edges,
  onNodesChange,
  onEdgesChange,
  onConnect,
  onReconnect,
  onNodeClick,
  onEdgeClick,
  onPaneClick,
  onEdgeDoubleClick,
  onInit,
  readOnly = false,
  interactive = false,
  isLoading = false,
  onInputIconClick,
}: WorkflowDesignerProps) {
  const { fitView, getNodes, getZoom, setCenter } = useReactFlow()
  const isFirstFitRef = useRef(true)
  const [viewMode, setViewMode] = useState<'design' | 'code'>('design')

  const isDraggable = !readOnly && !interactive
  const isConnectable = !readOnly && !interactive
  const isSelectable = !readOnly || interactive
  const isClickable = !readOnly || interactive

  const containerRef = useRef<HTMLDivElement>(null)
  const nodeMeasuredMapRef = useRef<Map<string, { width: number; height: number }>>(new Map())

  const handleNodesChange = useCallback((changes: any[]) => {
    changes.forEach((change) => {
      if (change.type === 'dimensions' && change.dimensions) {
        nodeMeasuredMapRef.current.set(change.id, change.dimensions)
      }
    })
    if (onNodesChange) {
      onNodesChange(changes)
    }
  }, [onNodesChange])

  const cleanJson = useMemo(() => {
    const cleanNodes = nodes.map((n) => ({
      id: n.id,
      type: n.type,
      position: n.position,
      data: n.data,
    }))
    const cleanEdges = edges.map((e) => ({
      id: e.id,
      source: e.source,
      target: e.target,
      label: e.label,
      data: e.data,
    }))
    return JSON.stringify({ nodes: cleanNodes, edges: cleanEdges }, null, 2)
  }, [nodes, edges])

  const processedNodes = useMemo(() => {
    return nodes.map((n) => {
      const cachedMeasured = n.measured || nodeMeasuredMapRef.current.get(n.id)
      return {
        ...n,
        ...(cachedMeasured ? { measured: cachedMeasured } : {}),
        data: {
          ...n.data,
          readOnly,
          interactive,
          onInputIconClick,
        },
      }
    })
  }, [nodes, readOnly, interactive, onInputIconClick])

  // Programmatic auto-fit once nodes are loaded and measured
  // We use a 250ms timeout to allow parent modal entrance animations (usually 150ms) to complete
  // and stabilize container dimensions before fitting the view.
  useEffect(() => {
    if (nodes && nodes.length > 0 && isFirstFitRef.current) {
      isFirstFitRef.current = false
      const timer = setTimeout(() => {
        fitView({ padding: 0.2, maxZoom: 1 })
      }, 250)
      return () => clearTimeout(timer)
    }
  }, [nodes, fitView])

  return (
    <div ref={containerRef} className="h-full w-full bg-slate-50 dark:bg-slate-950 relative flex flex-col overflow-hidden">
      <style>{`
        .react-flow__handle {
          transition: scale 0.15s cubic-bezier(0.4, 0, 0.2, 1), background-color 0.15s ease, box-shadow 0.15s ease !important;
        }
        /* Expand target drag area without changing visual size */
        .react-flow__handle::after {
          content: '';
          position: absolute;
          top: 50%;
          left: 50%;
          transform: translate(-50%, -50%);
          width: 28px;
          height: 28px;
          background: transparent;
          cursor: crosshair;
        }
        .react-flow__handle:hover {
          scale: 1.6 !important;
        }
        .react-flow__handle[class*="!bg-green-500"]:hover {
          box-shadow: 0 0 0 5px rgba(34, 197, 94, 0.25) !important;
        }
        .react-flow__handle[class*="!bg-amber-500"]:hover {
          box-shadow: 0 0 0 5px rgba(245, 158, 11, 0.25) !important;
        }
        .react-flow__handle[class*="!bg-teal-500"]:hover {
          box-shadow: 0 0 0 5px rgba(20, 184, 166, 0.25) !important;
        }
        .react-flow__handle[class*="!bg-primary"]:hover {
          box-shadow: 0 0 0 5px rgba(59, 130, 246, 0.25) !important;
        }
        /* Standard React Flow Edge Styling */
        .react-flow__edge-path {
          stroke: #94a3b8; /* slate-400 */
          stroke-width: 2px;
          transition: stroke 0.15s ease, stroke-width 0.15s ease;
        }
        .dark .react-flow__edge-path {
          stroke: #475569; /* slate-600 */
        }
        .react-flow__edge.selected .react-flow__edge-path {
          stroke: #3b82f6; /* primary / blue-500 */
          stroke-width: 3px;
        }
        .react-flow__edge:hover .react-flow__edge-path {
          stroke: #64748b; /* slate-500 */
        }
        .dark .react-flow__edge:hover .react-flow__edge-path {
          stroke: #94a3b8; /* slate-400 */
        }
        
        /* Interactive edge label styling */
        .react-flow__edge-textbg {
          fill: #ffffff;
          fill-opacity: 0.9;
          rx: 6;
          ry: 6;
        }
        .dark .react-flow__edge-textbg {
          fill: #0f172a;
          fill-opacity: 0.9;
        }
        .react-flow__edge-text {
          fill: #334155; /* slate-700 */
          font-size: 10px;
          font-weight: 600;
        }
        .dark .react-flow__edge-text {
          fill: #e2e8f0; /* slate-200 */
        }
      `}</style>

      {/* Floating Toggle Controls */}
      <div className="absolute top-4 right-4 z-30 flex items-center gap-1 bg-background/90 backdrop-blur-sm border border-border p-1 rounded-lg shadow-sm select-none">
        <button
          onClick={() => setViewMode('design')}
          className={cn(
            "flex items-center gap-1 px-2.5 py-1 rounded-md text-[10px] font-bold transition-all",
            viewMode === 'design'
              ? "bg-primary text-primary-foreground shadow-sm"
              : "text-muted-foreground hover:text-foreground hover:bg-muted"
          )}
        >
          <Workflow className="h-3 w-3" />
          Design
        </button>
        <button
          onClick={() => setViewMode('code')}
          className={cn(
            "flex items-center gap-1 px-2.5 py-1 rounded-md text-[10px] font-bold transition-all",
            viewMode === 'code'
              ? "bg-primary text-primary-foreground shadow-sm"
              : "text-muted-foreground hover:text-foreground hover:bg-muted"
          )}
        >
          <Code className="h-3 w-3" />
          Code
        </button>
      </div>

      {isLoading ? (
        <div className="absolute inset-0 flex flex-col items-center justify-center bg-background/80 z-50">
          <div className="h-8 w-8 border-4 border-primary border-t-transparent rounded-full animate-spin" />
          <p className="text-xs font-semibold text-muted-foreground mt-3">Loading designer layout...</p>
        </div>
      ) : null}

      {viewMode === 'design' ? (
        <ReactFlow<DesignerNode, DesignerEdge>
          nodes={processedNodes}
          edges={edges}
          nodeTypes={nodeTypes}
          edgeTypes={edgeTypes}
          onNodesChange={handleNodesChange}
          onEdgesChange={isDraggable ? onEdgesChange : undefined}
          onConnect={isConnectable ? onConnect : undefined}
          onReconnect={isConnectable ? onReconnect : undefined}
          onNodeClick={isClickable ? onNodeClick : undefined}
          onEdgeClick={readOnly ? undefined : onEdgeClick}
          onPaneClick={readOnly ? undefined : onPaneClick}
          onEdgeDoubleClick={readOnly ? undefined : onEdgeDoubleClick}
          onInit={onInit}
          defaultEdgeOptions={{ type: 'smoothstep' }}
          snapToGrid={true}
          snapGrid={[14, 14]}
          fitView
          fitViewOptions={{ maxZoom: 1 }}
          maxZoom={4}
          minZoom={0.2}
          proOptions={{ hideAttribution: true }}
          nodesDraggable={isDraggable}
          nodesConnectable={isConnectable}
          elementsSelectable={isSelectable}
        >
          <Background gap={14} />
          <Controls />
          {!readOnly && (
            <MiniMap
              position="bottom-right"
              style={{
                width: 160,
                height: 120,
              }}
              className="!bg-card border border-border shadow-md rounded-xl overflow-hidden !z-20"
              nodeColor={(node) => {
                if (node.type === 'agent') return '#3b82f6'
                if (node.type === 'trigger' || node.type === 'manual_trigger' || node.type === 'form_trigger') return '#a855f7'
                if (node.type === 'loop') return '#10b981'
                if (node.type === 'function') return '#06b6d4'
                if (node.type === 'join') return '#f59e0b'
                return '#94a3b8'
              }}
              nodeStrokeWidth={0}
              maskColor="rgba(148, 163, 184, 0.1)"
            />
          )}
        </ReactFlow>
      ) : (
        <div className="flex-1 w-full h-full p-6 font-mono text-[11px] overflow-auto bg-transparent text-foreground select-text">
          <pre className="h-full w-full leading-normal">
            <code>{cleanJson}</code>
          </pre>
        </div>
      )}
    </div>
  )
}

export function WorkflowDesigner(props: WorkflowDesignerProps) {
  return (
    <ReactFlowProvider>
      <WorkflowDesignerCanvas {...props} />
    </ReactFlowProvider>
  )
}
