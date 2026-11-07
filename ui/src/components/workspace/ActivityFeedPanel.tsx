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

import { useActivityStore, getPersonaColorClass } from "@/stores/activity"
import { X, BellOff, Clock } from "lucide-react"
import { ScrollArea } from "@/components/ui/scroll-area"

/**
 * Formats ISO timestamps into friendly relative human distances ("5m ago").
 */
function formatTimeDistance(timestampStr?: string): string {
  if (!timestampStr || timestampStr.startsWith("0001-01-01")) {
    return "Just now"
  }
  const dt = new Date(timestampStr)
  if (isNaN(dt.getTime()) || dt.getFullYear() <= 1970) return "Just now"

  const now = new Date()
  const diffMs = now.getTime() - dt.getTime()
  const diffSec = Math.floor(diffMs / 1000)
  const diffMin = Math.floor(diffSec / 60)
  const diffHour = Math.floor(diffMin / 60)

  if (diffSec < 60) {
    return "Just now"
  } else if (diffMin < 60) {
    return `${diffMin}m ago`
  } else if (diffHour < 24) {
    return `${diffHour}h ago`
  } else {
    return dt.toLocaleDateString(undefined, {
      month: "numeric",
      day: "numeric",
      year: "numeric",
    })
  }
}

/**
 * ActivityFeedPanel Next.js port.
 * Collapsible side-drawer listing global real-time service events.
 */
export function ActivityFeedPanel() {
  const { activities, togglePanel } = useActivityStore()

  return (
    <aside className="w-[340px] h-full bg-slate-50 dark:bg-slate-950 border-l border-border flex flex-col shrink-0 animate-slide-in-right shadow-lg z-40 relative">
      {/* Panel Header */}
      <div className="flex h-16 items-center justify-between px-5 border-b border-border select-none">
        <span className="text-xs font-bold tracking-wider text-foreground">
          NOTIFICATION CENTER
        </span>
        <button
          onClick={togglePanel}
          className="flex h-7 w-7 items-center justify-center rounded-lg text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
          title="Close Panel"
        >
          <X className="h-4 w-4" />
        </button>
      </div>

      {/* Notifications List Scrollable viewport */}
      <ScrollArea className="flex-1">
        {activities.length === 0 ? (
          <div className="flex h-[200px] flex-col items-center justify-center text-center gap-2 text-muted-foreground select-none p-4">
            <BellOff className="h-8 w-8 opacity-40" />
            <span className="text-sm font-medium italic">No activities yet.</span>
          </div>
        ) : (
          <div className="grid grid-cols-1 min-w-0 w-full gap-3.5 p-4 pb-8">
            {activities.map((msg, idx) => {
              const colorClass = getPersonaColorClass(msg.sender, msg.phase)
              const senderLabel = msg.sender || "System Activity"

              return (
                <div
                  key={msg.id || idx}
                  className="rounded-lg border border-border bg-card p-3 shadow-sm text-card-foreground hover:border-muted-foreground/30 transition-colors"
                >
                  {/* Card Header */}
                  <div className="flex items-center gap-2 mb-1.5">
                    {/* Role Indicator Pin */}
                    <div className={`h-2 w-2 rounded-full shrink-0 ${colorClass}`} />
                    
                    <span className="font-bold text-foreground text-xs truncate flex-1">
                      {senderLabel}
                    </span>

                    {msg.phase && (
                      <span className="text-[9px] font-bold text-muted-foreground uppercase border border-border px-1.5 py-0.5 rounded bg-muted/30 shrink-0">
                        {msg.phase}
                      </span>
                    )}
                  </div>

                  {/* Body Text */}
                  <p className="text-[11px] text-foreground/80 leading-normal break-words whitespace-pre-wrap">
                    {msg.displayText || msg.displayPrompt || msg.text}
                  </p>

                  {/* Optional Time-stamp relative distance tag */}
                  {msg.timestamp && (
                    <div className="flex items-center gap-1 text-[9px] text-muted-foreground/70 mt-2">
                      <Clock className="h-3 w-3" />
                      <span>{formatTimeDistance(msg.timestamp)}</span>
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        )}
      </ScrollArea>
    </aside>
  )
}
export default ActivityFeedPanel
