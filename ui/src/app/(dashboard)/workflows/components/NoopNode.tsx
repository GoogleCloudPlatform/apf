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

import React from "react"
import { Handle, Position, useReactFlow, useUpdateNodeInternals } from "@xyflow/react"
import { ArrowRight, Trash2 } from "lucide-react"

interface NoopNodeProps {
  id: string
  data: {
    name: string
    displayName?: string
    orientation?: 'LR' | 'RL' | 'TB' | 'BT'
    readOnly?: boolean
    interactive?: boolean
    [key: string]: any
  }
  selected?: boolean
}

export const NoopNode = ({ id, data, selected }: NoopNodeProps) => {
  const { setNodes, setEdges } = useReactFlow()
  const updateNodeInternals = useUpdateNodeInternals()

  const handleDelete = (e: React.MouseEvent) => {
    e.stopPropagation()
    setNodes((nds) => nds.filter((n) => n.id !== id))
    setEdges((eds) => eds.filter((e) => e.source !== id && e.target !== id))
  }
  const orientation = data.orientation || 'LR'
  const isReadOnly = !!data.readOnly
  const isInteractive = !!data.interactive

  // Double click cycles orientation clockwise: LR -> TB -> RL -> BT -> LR
  const handleDoubleClick = (e: React.MouseEvent) => {
    e.stopPropagation() // Prevent triggering background canvas double-clicks
    
    const nextOrientation = 
      orientation === 'LR' ? 'TB' :
      orientation === 'TB' ? 'RL' :
      orientation === 'RL' ? 'BT' : 'LR'

    // 1. Update node data state
    setNodes((nds) =>
      nds.map((n) =>
        n.id === id
          ? { ...n, data: { ...n.data, orientation: nextOrientation } }
          : n
      )
    )

    // 2. Force React Flow to re-measure handle bounds after DOM updates
    requestAnimationFrame(() => {
      updateNodeInternals(id)
    })
  }

  // Determine handle positions and icon rotation based on current orientation
  let targetPos = Position.Left
  let sourcePos = Position.Right
  let rotationClass = "rotate-0" // Left-to-Right

  if (orientation === "TB") {
    targetPos = Position.Top
    sourcePos = Position.Bottom
    rotationClass = "rotate-90" // Top-to-Bottom
  } else if (orientation === "RL") {
    targetPos = Position.Right
    sourcePos = Position.Left
    rotationClass = "rotate-180" // Right-to-Left
  } else if (orientation === "BT") {
    targetPos = Position.Bottom
    sourcePos = Position.Top
    rotationClass = "-rotate-90" // Bottom-to-Top
  }

  return (
    <div 
      className="relative flex flex-col items-center select-none"
      onDoubleClick={handleDoubleClick}
      title="Double-click to cycle orientation clockwise (LR ➔ TB ➔ RL ➔ BT)"
    >
      {selected && !isReadOnly && !isInteractive && (
        <div 
          className="absolute -top-12 left-1/2 -translate-x-1/2 flex items-center bg-white dark:bg-slate-950/95 border border-slate-200 dark:border-slate-800/90 backdrop-blur-md p-1 rounded-lg shadow-xl z-50 nodrag select-none"
        >
          <button
            onClick={handleDelete}
            className="p-1 rounded hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-500 dark:text-slate-400 hover:text-red-500 dark:hover:text-red-500 transition-colors cursor-pointer"
            title="Delete Node"
          >
            <Trash2 className="h-3.5 w-3.5" />
          </button>
        </div>
      )}
      {/* Main Node Box: 56px x 56px (w-14 h-14) */}
      <div
        className={`w-14 h-14 rounded-xl border shadow-md flex items-center justify-center transition-all duration-200 relative bg-white border-slate-200/80 dark:bg-slate-900/90 dark:border-slate-800/90 ${
          selected
            ? "border-primary ring-4 ring-primary/20 scale-[1.03] border-2"
            : "hover:scale-[1.01]"
        }`}
      >
        {/* Dynamic target (input) and source (output) handles based on orientation (visually hidden but functional) */}
        <Handle
          type="target"
          position={targetPos}
          id="target"
          className="!opacity-0 !z-10"
        />
        <Handle
          type="source"
          position={sourcePos}
          id="source"
          className="!opacity-0 !z-10"
        />

        {/* Center redirect icon with CSS rotation */}
        <ArrowRight className={`h-5 w-5 text-slate-400 dark:text-slate-500 transition-transform duration-300 ${rotationClass}`} strokeWidth={2} />
      </div>
    </div>
  )
}
