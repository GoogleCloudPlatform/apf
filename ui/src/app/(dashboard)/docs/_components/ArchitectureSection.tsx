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

import React from "react"
import { 
  Cpu, 
  Network, 
  HardDrive, 
  Terminal, 
  Radio, 
  ShieldAlert, 
  CheckCircle2, 
  ArrowRight,
  GitBranch
} from "lucide-react"

interface ArchitectureSectionProps {
  id?: string
  sectionRef?: React.RefObject<HTMLDivElement | null>
}

export function ArchitectureSection({ id = "tour-docs-architecture", sectionRef }: ArchitectureSectionProps) {
  return (
    <div id={id} ref={sectionRef} className="space-y-6 scroll-mt-10">
      {/* Section Header */}
      <div className="space-y-2 border-b border-border pb-4 select-none">
        <div className="flex items-center gap-3">
          <Cpu className="h-6 w-6 text-primary" />
          <h2 className="text-base font-bold text-foreground">Agent Swarm Architecture</h2>
        </div>
        <p className="text-[11px] text-muted-foreground leading-relaxed">
          How Foundry orchestrates multi-agent communication, reactive wakeups, sandboxed file mutations, and persistent context loops.
        </p>
      </div>

      {/* Architecture Blocks */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* Subagent Dispatcher & Messaging */}
        <div className="p-5 rounded-2xl border border-border bg-card space-y-3 shadow-sm flex flex-col justify-between">
          <div className="space-y-3">
            <div className="flex items-center gap-3">
              <div className="h-9 w-9 rounded-xl bg-purple-500/10 text-purple-600 dark:text-purple-400 flex items-center justify-center shrink-0">
                <Radio className="h-5 w-5" />
              </div>
              <div className="min-w-0 flex-1">
                <h3 className="text-sm font-bold text-foreground truncate">Reactive Wakeup</h3>
                <span className="text-[10px] text-muted-foreground block truncate">Zero polling async flow</span>
              </div>
            </div>
            <p className="text-xs text-muted-foreground leading-relaxed">
              When an agent launches a background command, cron schedule, or spawns child subagents (e.g. <code className="font-mono text-foreground bg-muted px-1 py-0.5 rounded">research-google</code>), the parent halts and sleeps until reactive completion events fire.
            </p>
          </div>
        </div>

        {/* Sandboxed UV & Python Runner */}
        <div className="p-5 rounded-2xl border border-border bg-card space-y-3 shadow-sm flex flex-col justify-between">
          <div className="space-y-3">
            <div className="flex items-center gap-3">
              <div className="h-9 w-9 rounded-xl bg-blue-500/10 text-blue-600 dark:text-blue-400 flex items-center justify-center shrink-0">
                <Terminal className="h-5 w-5" />
              </div>
              <div className="min-w-0 flex-1">
                <h3 className="text-sm font-bold text-foreground truncate">Sandboxed UV Runner</h3>
                <span className="text-[10px] text-muted-foreground block truncate">Isolated tool execution</span>
              </div>
            </div>
            <p className="text-xs text-muted-foreground leading-relaxed">
              BMad tools and auxiliary Python scripts run inside an ephemeral <code className="font-mono text-foreground bg-muted px-1 py-0.5 rounded">uv run</code> environment with strict workspace boundary validators ensuring zero-trust isolation.
            </p>
          </div>
        </div>

        {/* Multi-Tenant Data Partitioning */}
        <div className="p-5 rounded-2xl border border-border bg-card space-y-3 shadow-sm flex flex-col justify-between">
          <div className="space-y-3">
            <div className="flex items-center gap-3">
              <div className="h-9 w-9 rounded-xl bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 flex items-center justify-center shrink-0">
                <HardDrive className="h-5 w-5" />
              </div>
              <div className="min-w-0 flex-1">
                <h3 className="text-sm font-bold text-foreground truncate">Multi-Tenant Storage</h3>
                <span className="text-[10px] text-muted-foreground block truncate">Organization-scoped data</span>
              </div>
            </div>
            <p className="text-xs text-muted-foreground leading-relaxed">
              Projects, credentials, chat sessions, and artifacts are strictly partitioned by Organization ID. Organization switches instantaneously remap data planes and permissions.
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}
