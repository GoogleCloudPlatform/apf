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
import { Handle, Position, useReactFlow } from "@xyflow/react"
import { LucideIcon, Play, Square, Power, Trash2, Zap, Keyboard } from "lucide-react"
import { cn } from "@/lib/utils"

interface TriggerNodeProps {
  id: string
  data: {
    name: string
    displayName?: string
    isRunning?: boolean
    isDisabled?: boolean
    readOnly?: boolean
    interactive?: boolean
    awaitingInput?: boolean
    onInputIconClick?: (id: string) => void
    [key: string]: any
  }
  selected?: boolean
  icon: LucideIcon
  themeClass?: string
  iconClass?: string
}

export const TriggerNode = ({
  id,
  data,
  selected,
  icon: Icon,
  themeClass = "border-purple-300 dark:border-purple-500/70 ring-purple-500/20",
  iconClass = "text-purple-600 dark:text-purple-400"
}: TriggerNodeProps) => {
  const { setNodes, setEdges } = useReactFlow()

  const isRunning = !!data.isRunning
  const isDisabled = !!data.isDisabled
  const isReadOnly = !!data.readOnly
  const isInteractive = !!data.interactive

  const handleToggleStart = (e: React.MouseEvent) => {
    e.stopPropagation()
    setNodes((nds) =>
      nds.map((n) =>
        n.id === id
          ? { ...n, data: { ...n.data, isRunning: !isRunning } }
          : n
      )
    )
  }

  const handleToggleEnable = (e: React.MouseEvent) => {
    e.stopPropagation()
    setNodes((nds) =>
      nds.map((n) =>
        n.id === id
          ? { ...n, data: { ...n.data, isDisabled: !isDisabled } }
          : n
      )
    )
  }

  const handleDelete = (e: React.MouseEvent) => {
    e.stopPropagation()
    setNodes((nds) => nds.filter((n) => n.id !== id))
    setEdges((eds) => eds.filter((e) => e.source !== id && e.target !== id))
  }

  return (
    <div className="relative flex flex-col items-center">
      {/* 1. Selection Toolbar on Top */}
      {selected && !isReadOnly && !isInteractive && (
        <div 
          className="absolute -top-12 left-1/2 -translate-x-1/2 flex items-center gap-1.5 bg-white dark:bg-slate-950/95 border border-slate-200 dark:border-slate-800/90 backdrop-blur-md px-2 py-1 rounded-lg shadow-xl z-50 nodrag select-none"
        >
          {/* Start/Stop Button */}
          <button
            onClick={handleToggleStart}
            className={`p-1.5 rounded hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors ${
              isRunning ? "text-green-500" : "text-slate-500 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-200"
            }`}
            title={isRunning ? "Stop Trigger" : "Fire Trigger"}
          >
            {isRunning ? (
              <Square className="h-3.5 w-3.5 fill-current" />
            ) : (
              <Play className="h-3.5 w-3.5 fill-current" />
            )}
          </button>

          {/* Divider */}
          <div className="w-px h-4 bg-slate-200 dark:bg-slate-800" />

          {/* Enable/Disable Button */}
          <button
            onClick={handleToggleEnable}
            className={`p-1.5 rounded hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors ${
              isDisabled ? "text-red-500" : "text-slate-500 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-200"
            }`}
            title={isDisabled ? "Enable Trigger" : "Disable Trigger"}
          >
            <Power className="h-3.5 w-3.5" />
          </button>

          {/* Divider */}
          <div className="w-px h-4 bg-slate-200 dark:bg-slate-800" />

          {/* Delete Button */}
          <button
            onClick={handleDelete}
            className="p-1.5 rounded hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-500 dark:text-slate-400 hover:text-red-500 dark:hover:text-red-500 transition-colors"
            title="Delete Trigger"
          >
            <Trash2 className="h-3.5 w-3.5" />
          </button>
        </div>
      )}

      {/* 2. Main Node Container */}
      <div
        className={cn(
          "w-[56px] h-[56px] flex items-center justify-center border bg-purple-50/70 border-purple-100 dark:bg-slate-900/95 dark:border-slate-800/90 shadow-lg select-none relative transition-all duration-200",
          selected ? "ring-2 ring-offset-2 ring-offset-background border-purple-400 dark:border-purple-400 shadow-purple-500/10 dark:shadow-purple-500/20" : "",
          isDisabled ? "opacity-40" : "opacity-100",
          themeClass
        )}
        style={{
          borderRadius: "28px 8px 8px 28px", // D-shape: fully rounded left, slightly rounded right
        }}
      >
        {/* Bouncing keyboard icon for HITL input requests */}
        {data.awaitingInput && (
          <button
            onClick={(e) => {
              e.stopPropagation()
              if (data.onInputIconClick) {
                data.onInputIconClick(id)
              }
            }}
            className="absolute -top-3 -left-3 bg-amber-500 hover:bg-amber-600 text-white rounded-full p-1.5 shadow-md border-2 border-background animate-bounce z-50 transition-all hover:scale-110 active:scale-90"
            title="Input Required"
          >
            <Keyboard className="h-4 w-4" />
          </button>
        )}

        {/* Floating lightning icon to the left */}
        <div className="absolute right-full mr-2 top-1/2 -translate-y-1/2 text-orange-500 fill-orange-500 select-none nodrag pointer-events-none">
          <Zap className="h-3.5 w-3.5 fill-current" />
        </div>

        {/* Pulsing indicator if running */}
        {isRunning && (
          <span className="absolute top-1 right-1 flex h-2 w-2">
            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
            <span className="relative inline-flex rounded-full h-2 w-2 bg-green-500"></span>
          </span>
        )}

        {/* Central circular icon wrapper */}
        <div className={cn(
          "w-8 h-8 rounded-full border border-purple-100/80 dark:border-slate-700/45 flex items-center justify-center bg-white/90 dark:bg-slate-950/40 shadow-inner",
          selected ? "border-purple-400/50" : ""
        )}>
          <Icon className={cn("w-4 h-4", iconClass)} />
        </div>

        {/* Output Handle on the flat right edge */}
        <Handle
          type="source"
          position={Position.Right}
          className="w-3.5 h-3.5 !bg-purple-500 border-2 border-background hover:scale-110 transition-transform duration-100"
        />

        {/* Sub-label rendered underneath the node */}
        <div className="absolute -bottom-6 left-1/2 -translate-x-1/2 whitespace-nowrap text-[10px] font-bold text-slate-800 dark:text-slate-300 tracking-wide select-none drop-shadow-sm">
          {data.displayName || data.name}
        </div>
      </div>
    </div>
  )
}
