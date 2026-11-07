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
import { ChevronDown, ChevronRight } from "lucide-react"

interface ExplorerSectionProps {
  id?: string
  title: string
  isExpanded: boolean
  onToggle: () => void
  children: React.ReactNode
  trailingHeaderWidget?: React.ReactNode
}

export default function ExplorerSection({
  id,
  title,
  isExpanded,
  onToggle,
  children,
  trailingHeaderWidget,
}: ExplorerSectionProps) {
  return (
    <div id={id} className="mb-3 border border-border/50 rounded-md bg-slate-50/50 dark:bg-slate-900/50 overflow-hidden">
      <div
        onClick={onToggle}
        className="flex items-center justify-between px-2.5 py-2 cursor-pointer hover:bg-slate-100/50 dark:hover:bg-slate-800/50 transition-colors select-none"
      >
        <div className="flex items-center gap-1.5 min-w-0">
          {isExpanded ? (
            <ChevronDown className="h-4 w-4 text-muted-foreground shrink-0" />
          ) : (
            <ChevronRight className="h-4 w-4 text-muted-foreground shrink-0" />
          )}
          <span className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest truncate">
            {title}
          </span>
        </div>
        {trailingHeaderWidget && (
          <div onClick={(e) => e.stopPropagation()} className="shrink-0 ml-2">
            {trailingHeaderWidget}
          </div>
        )}
      </div>
      {isExpanded && (
        <>
          <div className="h-px bg-border/50" />
          <div className="p-2.5 space-y-2">{children}</div>
        </>
      )}
    </div>
  )
}
