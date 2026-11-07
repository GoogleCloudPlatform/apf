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
import { EdgeProps, getSmoothStepPath, EdgeLabelRenderer, useReactFlow } from "@xyflow/react"
import { getSmartPath } from "./smartRouting"

export const N8nEdge = ({
  id,
  source,
  target,
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  style = {},
  markerEnd,
  label,
  selected,
}: EdgeProps) => {
  const { getNodes } = useReactFlow()
  const nodes = getNodes()

  // Calculate high-performance obstacle-avoiding path (inspired by libavoid)
  const edgePath = getSmartPath(sourceX, sourceY, targetX, targetY, nodes, source, target)

  // Still use standard xyflow smoothstep path helper ONLY to find the ideal center coordinates for the label
  const [, labelX, labelY] = getSmoothStepPath({
    sourceX,
    sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition,
  })

  return (
    <>
      <style>{`
        @keyframes n8n-flow {
          0% {
            offset-distance: 0%;
          }
          100% {
            offset-distance: 100%;
          }
        }
        .n8n-edge-pulse {
          animation: n8n-flow 4s linear infinite;
        }
      `}</style>

      {/* 1. Base thick path for hover area & background */}
      <path
        id={id}
        style={style}
        className={`fill-none transition-all stroke-slate-200 dark:stroke-slate-800/90 cursor-pointer ${
          selected ? "!stroke-primary/50" : "hover:stroke-slate-300 dark:hover:stroke-slate-700"
        }`}
        strokeWidth={selected ? 6 : 4}
        d={edgePath}
        markerEnd={markerEnd}
      />

      {/* 2. Inner thin core path for glowing/accent line */}
      <path
        className={`fill-none transition-all ${
          selected ? "stroke-primary" : "stroke-slate-400 dark:stroke-slate-600/80"
        }`}
        strokeWidth={1.5}
        d={edgePath}
      />

      {/* 3. Flowing animated pulse dot along the path */}
      <circle
        r="3"
        className="n8n-edge-pulse fill-primary dark:fill-emerald-400 shadow-sm"
        style={{
          offsetPath: `path('${edgePath}')`,
        }}
      />

      {/* 4. Glassmorphic Label Badge in the center */}
      {label && (
        <EdgeLabelRenderer>
          <div
            style={{
              position: "absolute",
              transform: `translate(-50%, -50%) translate(${labelX}px,${labelY}px)`,
              pointerEvents: "all",
            }}
            className="nodrag nopan bg-white/90 dark:bg-slate-950/90 text-slate-700 dark:text-slate-200 border border-slate-200 dark:border-slate-800/80 backdrop-blur-md px-2 py-0.5 rounded-md text-[9px] font-bold shadow-md select-none transition-all hover:scale-105"
          >
            {label}
          </div>
        </EdgeLabelRenderer>
      )}
    </>
  )
}
