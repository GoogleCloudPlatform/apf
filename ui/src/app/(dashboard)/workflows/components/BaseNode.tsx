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
import { Play, Square, Power, Trash2, LucideIcon, Keyboard } from "lucide-react"

interface BaseNodeProps {
  id: string
  selected?: boolean
  data: {
    name?: string
    displayName?: string
    description?: string
    isRunning?: boolean
    isDisabled?: boolean
    readOnly?: boolean
    interactive?: boolean
    awaitingInput?: boolean
    onInputIconClick?: (nodeId: string) => void
    [key: string]: any
  }
  icon: LucideIcon | string
  iconColorClass?: string
  label: string
  subtitle?: string
  hasTarget?: boolean
  hasSource?: boolean
  targetPosition?: Position
  sourcePosition?: Position
  themeColorClass?: string
  bgClass?: string
  children?: React.ReactNode
}

export const BaseNode = ({
  id,
  selected,
  data,
  icon: Icon,
  iconColorClass = "text-primary",
  label,
  subtitle,
  hasTarget = true,
  hasSource = true,
  targetPosition = Position.Left,
  sourcePosition = Position.Right,
  themeColorClass = "border-primary ring-primary/20",
  bgClass = "bg-slate-50/70 border-slate-200/80 dark:bg-slate-900/90 dark:border-slate-800/90",
  children,
}: BaseNodeProps) => {
  const { setNodes, setEdges } = useReactFlow()

  const isRunning = !!data.isRunning
  const isDisabled = !!data.isDisabled
  const isReadOnly = !!data.readOnly
  const isInteractive = !!data.interactive
  const isAwaitingInput = !!data.awaitingInput

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
    if (id === "start") return
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
            title={isRunning ? "Stop Node" : "Start Node"}
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
            title={isDisabled ? "Enable Node" : "Disable Node"}
          >
            <Power className="h-3.5 w-3.5" />
          </button>

          {/* Divider */}
          {id !== "start" && <div className="w-px h-4 bg-slate-200 dark:bg-slate-800" />}

          {/* Delete Button */}
          {id !== "start" && (
            <button
              onClick={handleDelete}
              className="p-1.5 rounded hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-500 dark:text-slate-400 hover:text-red-500 dark:hover:text-red-500 transition-colors"
              title="Delete Node"
            >
              <Trash2 className="h-3.5 w-3.5" />
            </button>
          )}
        </div>
      )}

      {/* 3. Main Node Container */}
      <div
        className={`w-28 h-28 rounded-2xl border shadow-lg flex items-center justify-center transition-all duration-200 relative ${
          isDisabled ? "opacity-40" : "opacity-100"
        } ${bgClass} ${
          selected
            ? `${themeColorClass} ring-4 ring-opacity-20 scale-[1.03] border-2`
            : "hover:scale-[1.01]"
        }`}
      >
        {/* Input Required Icon */}
        {isAwaitingInput && (
          <button
            onClick={(e) => {
              e.stopPropagation()
              data.onInputIconClick?.(id)
            }}
            className="absolute -top-3 -left-3 bg-amber-500 text-white rounded-full p-1.5 shadow-lg hover:bg-amber-600 transition-colors z-50 nodrag flex items-center justify-center border border-background"
            title="Input Required to Resume Workflow"
          >
            <Keyboard className="h-3.5 w-3.5 animate-bounce" style={{ animationDuration: '2s' }} />
          </button>
        )}
        {/* 2. Handles (nested inside the 112x112 box for perfect vertical centering!) */}
        {hasTarget && (
          <Handle
            type="target"
            position={targetPosition}
            className="w-3 h-3 !bg-white border-2 border-slate-400 dark:!bg-slate-950 dark:border-slate-700 hover:!bg-primary hover:border-primary-foreground transition-all !z-10"
          />
        )}
        {hasSource && (
          <Handle
            type="source"
            position={sourcePosition}
            className="w-3 h-3 !bg-white border-2 border-slate-400 dark:!bg-slate-950 dark:border-slate-700 hover:!bg-primary hover:border-primary-foreground transition-all !z-10"
          />
        )}

        {/* Pulsing indicator if running */}
        {isRunning && (
          <span className="absolute top-2 right-2 flex h-2 w-2">
            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
            <span className="relative inline-flex rounded-full h-2 w-2 bg-green-500"></span>
          </span>
        )}

        {/* Icon */}
        <div className="flex items-center justify-center">
          {typeof Icon === "string" ? (
            <span className="text-3xl select-none">{Icon}</span>
          ) : (
            <Icon className={`h-10 w-10 ${iconColorClass}`} strokeWidth={1.5} />
          )}
        </div>

        {/* Any extra node-specific overlay content */}
        {children}
      </div>

      {/* 4. Label below node */}
      <div className="mt-2 flex flex-col items-center select-none max-w-[140px]">
        <span className="text-xs font-semibold text-slate-800 dark:text-slate-300 text-center truncate w-full px-1">
          {label}
        </span>
        {subtitle && (
          <span className="text-[9px] font-medium text-slate-500 dark:text-slate-400 mt-0.5 uppercase tracking-wider truncate w-full text-center">
            {subtitle}
          </span>
        )}
      </div>
    </div>
  )
}
