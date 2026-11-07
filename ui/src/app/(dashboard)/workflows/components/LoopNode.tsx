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

import { Handle, Position } from "@xyflow/react"
import { BaseNode } from "./BaseNode"
import { RotateCw } from "lucide-react"

interface LoopNodeProps {
  id: string
  data: {
    name: string
    displayName?: string
    description?: string
    [key: string]: any
  }
  selected?: boolean
}

export const LoopNode = ({ id, data, selected }: LoopNodeProps) => {
  return (
    <BaseNode
      id={id}
      selected={selected}
      data={data}
      icon={RotateCw}
      iconColorClass="text-emerald-500 dark:text-emerald-400"
      themeColorClass="border-emerald-500 ring-emerald-500/20"
      bgClass="bg-emerald-50/70 border-emerald-100 dark:bg-slate-900/90 dark:border-slate-800/90"
      label={data.displayName || data.name}
      subtitle="LOOP CONTROL"
      hasSource={false}
    >
      {/* Custom Loop Handles */}
      {/* 1. "done" handle (top-right) */}
      <Handle
        type="source"
        position={Position.Right}
        id="done"
        style={{ top: "30%" }}
        className="w-3 h-3 !bg-white border-2 border-slate-400 dark:!bg-slate-950 dark:border-slate-700 hover:!bg-primary hover:border-primary-foreground transition-all !z-10"
      />
      <span className="absolute left-full top-[30%] -translate-y-1/2 ml-2.5 text-[10px] text-slate-500 dark:text-slate-400 font-bold select-none whitespace-nowrap">
        done
      </span>

      {/* 2. "loop" handle (bottom-right) */}
      <Handle
        type="source"
        position={Position.Right}
        id="loop"
        style={{ top: "70%" }}
        className="w-3 h-3 !bg-white border-2 border-slate-400 dark:!bg-slate-950 dark:border-slate-700 hover:!bg-primary hover:border-primary-foreground transition-all !z-10"
      />
      <span className="absolute left-full top-[70%] -translate-y-1/2 ml-2.5 text-[10px] text-slate-500 dark:text-slate-400 font-bold select-none whitespace-nowrap">
        loop
      </span>
    </BaseNode>
  )
}
