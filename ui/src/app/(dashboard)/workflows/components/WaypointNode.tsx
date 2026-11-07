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
import { Handle, Position } from "@xyflow/react"

export const WaypointNode = ({ selected }: { selected: boolean }) => {
  return (
    <div 
      className="h-7 w-7 flex items-center justify-center cursor-move relative select-none"
      title="Routing Anchor (Drag to bend connection paths)"
    >
      {/* 1. Small visual circle in the exact center of the larger drag area */}
      <div
        className={`h-3 w-3 rounded-full border-2 bg-white dark:bg-slate-950 transition-all ${
          selected
            ? "border-primary shadow-[0_0_8px_rgba(59,130,246,0.5)] scale-110"
            : "border-slate-400 dark:border-slate-600 hover:border-primary hover:scale-110"
        }`}
      />

      {/* 2. Completely hidden, non-interactive TARGET handles on all 4 sides of the visual circle */}
      <Handle
        type="target"
        position={Position.Left}
        id="target-left"
        isConnectable={false}
        className="!opacity-0 !w-0 !h-0 !min-w-0 !min-h-0"
        style={{ left: '8px', top: '50%', transform: 'translate(0, -50%)' }}
      />
      <Handle
        type="target"
        position={Position.Right}
        id="target-right"
        isConnectable={false}
        className="!opacity-0 !w-0 !h-0 !min-w-0 !min-h-0"
        style={{ right: '8px', top: '50%', transform: 'translate(0, -50%)' }}
      />
      <Handle
        type="target"
        position={Position.Top}
        id="target-top"
        isConnectable={false}
        className="!opacity-0 !w-0 !h-0 !min-w-0 !min-h-0"
        style={{ top: '8px', left: '50%', transform: 'translate(-50%, 0)' }}
      />
      <Handle
        type="target"
        position={Position.Bottom}
        id="target-bottom"
        isConnectable={false}
        className="!opacity-0 !w-0 !h-0 !min-w-0 !min-h-0"
        style={{ bottom: '8px', left: '50%', transform: 'translate(-50%, 0)' }}
      />

      {/* 3. Completely hidden, non-interactive SOURCE handles on all 4 sides of the visual circle */}
      <Handle
        type="source"
        position={Position.Left}
        id="source-left"
        isConnectable={false}
        className="!opacity-0 !w-0 !h-0 !min-w-0 !min-h-0"
        style={{ left: '8px', top: '50%', transform: 'translate(0, -50%)' }}
      />
      <Handle
        type="source"
        position={Position.Right}
        id="source-right"
        isConnectable={false}
        className="!opacity-0 !w-0 !h-0 !min-w-0 !min-h-0"
        style={{ right: '8px', top: '50%', transform: 'translate(0, -50%)' }}
      />
      <Handle
        type="source"
        position={Position.Top}
        id="source-top"
        isConnectable={false}
        className="!opacity-0 !w-0 !h-0 !min-w-0 !min-h-0"
        style={{ top: '8px', left: '50%', transform: 'translate(-50%, 0)' }}
      />
      <Handle
        type="source"
        position={Position.Bottom}
        id="source-bottom"
        isConnectable={false}
        className="!opacity-0 !w-0 !h-0 !min-w-0 !min-h-0"
        style={{ bottom: '8px', left: '50%', transform: 'translate(-50%, 0)' }}
      />
    </div>
  )
}
