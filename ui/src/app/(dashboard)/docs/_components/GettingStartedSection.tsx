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
  BookOpen, 
  FolderKanban, 
  Layers, 
  Sparkles, 
  ShieldCheck, 
  Zap, 
  Workflow, 
  Terminal, 
  CheckCircle2 
} from "lucide-react"

interface GettingStartedSectionProps {
  id?: string
  sectionRef?: React.RefObject<HTMLDivElement | null>
}

export function GettingStartedSection({ id = "tour-docs-getting-started", sectionRef }: GettingStartedSectionProps) {
  return (
    <div id={id} ref={sectionRef} className="space-y-6 scroll-mt-10">
      {/* Section Header */}
      <div className="space-y-2 border-b border-border pb-4 select-none">
        <div className="flex items-center gap-3">
          <BookOpen className="h-6 w-6 text-primary" />
          <h2 className="text-base font-bold text-foreground">Getting Started with Foundry</h2>
        </div>
        <p className="text-[11px] text-muted-foreground leading-relaxed">
          Agentic Product Foundry (APF) is an enterprise multi-agent pair programming and autonomous software engineering platform built on Google infrastructure.
        </p>
      </div>

      {/* Core Concepts Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-4">
        {/* Concept 1: Autonomous Agent Swarms */}
        <div className="p-5 rounded-2xl border border-border bg-card space-y-3 shadow-sm flex flex-col justify-between">
          <div className="space-y-3">
            <div className="flex items-center gap-3">
              <div className="h-9 w-9 rounded-xl bg-primary/10 text-primary flex items-center justify-center shrink-0">
                <Sparkles className="h-5 w-5" />
              </div>
              <div className="min-w-0 flex-1">
                <h3 className="text-sm font-bold text-foreground truncate">Autonomous Swarms</h3>
                <span className="text-[10px] text-muted-foreground block truncate">Collaborative execution</span>
              </div>
            </div>
            <p className="text-xs text-muted-foreground leading-relaxed">
              Collaborate with specialized personas including Architect, PM, Developer, TEA, and UX Designer in coordinated swarm workflows.
            </p>
          </div>
        </div>

        {/* Concept 2: Visual Canvas Workspace */}
        <div className="p-5 rounded-2xl border border-border bg-card space-y-3 shadow-sm flex flex-col justify-between">
          <div className="space-y-3">
            <div className="flex items-center gap-3">
              <div className="h-9 w-9 rounded-xl bg-blue-500/10 text-blue-600 dark:text-blue-400 flex items-center justify-center shrink-0">
                <Layers className="h-5 w-5" />
              </div>
              <div className="min-w-0 flex-1">
                <h3 className="text-sm font-bold text-foreground truncate">Interactive Canvas</h3>
                <span className="text-[10px] text-muted-foreground block truncate">Live DAG visualizer</span>
              </div>
            </div>
            <p className="text-xs text-muted-foreground leading-relaxed">
              Visualize project architecture, code nodes, trigger connections, and live agent chat streams on an interactive canvas powered by XYFlow.
            </p>
          </div>
        </div>

        {/* Concept 3: Sandboxed Execution */}
        <div className="p-5 rounded-2xl border border-border bg-card space-y-3 shadow-sm flex flex-col justify-between">
          <div className="space-y-3">
            <div className="flex items-center gap-3">
              <div className="h-9 w-9 rounded-xl bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 flex items-center justify-center shrink-0">
                <ShieldCheck className="h-5 w-5" />
              </div>
              <div className="min-w-0 flex-1">
                <h3 className="text-sm font-bold text-foreground truncate">Zero-Trust Sandbox</h3>
                <span className="text-[10px] text-muted-foreground block truncate">Secure guardrails</span>
              </div>
            </div>
            <p className="text-xs text-muted-foreground leading-relaxed">
              All agent file modifications, terminal commands, and tool executions run in isolated environments with strict workspace boundary enforcement.
            </p>
          </div>
        </div>

        {/* Concept 4: Enterprise Toolset */}
        <div className="p-5 rounded-2xl border border-border bg-card space-y-3 shadow-sm flex flex-col justify-between">
          <div className="space-y-3">
            <div className="flex items-center gap-3">
              <div className="h-9 w-9 rounded-xl bg-purple-500/10 text-purple-600 dark:text-purple-400 flex items-center justify-center shrink-0">
                <Zap className="h-5 w-5" />
              </div>
              <div className="min-w-0 flex-1">
                <h3 className="text-sm font-bold text-foreground truncate">Extensible Tools</h3>
                <span className="text-[10px] text-muted-foreground block truncate">Model Context Protocol</span>
              </div>
            </div>
            <p className="text-xs text-muted-foreground leading-relaxed">
              Directly connect your Git repositories (GitHub/GitLab), task boards (Jira/Trello), Figma designs, and MCP servers with global master keys.
            </p>
          </div>
        </div>
      </div>

      {/* Quick Start Guide Card */}
      <div className="bg-slate-100/70 dark:bg-slate-900/60 border border-border rounded-2xl p-6 space-y-4 shadow-sm">
        <h3 className="text-sm font-bold text-foreground flex items-center gap-2">
          <Workflow className="h-4 w-4 text-primary" />
          Standard Feature Development Lifecycle
        </h3>
        
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3 text-xs">
          <div className="p-3.5 bg-card border border-border rounded-xl space-y-1.5">
            <span className="font-bold text-primary block">1. Discover</span>
            <p className="text-[11px] text-muted-foreground leading-relaxed">
              Brainstorm or challenge your concept with <code className="font-mono text-foreground bg-muted px-1 py-0.5 rounded">/bmad-brainstorming</code> or <code className="font-mono text-foreground bg-muted px-1 py-0.5 rounded">/bmad-prfaq</code>.
            </p>
          </div>

          <div className="p-3.5 bg-card border border-border rounded-xl space-y-1.5">
            <span className="font-bold text-primary block">2. Plan & Spec</span>
            <p className="text-[11px] text-muted-foreground leading-relaxed">
              Synthesize your PRD with <code className="font-mono text-foreground bg-muted px-1 py-0.5 rounded">/bmad-prd</code> and lock invariants with <code className="font-mono text-foreground bg-muted px-1 py-0.5 rounded">/bmad-spec</code>.
            </p>
          </div>

          <div className="p-3.5 bg-card border border-border rounded-xl space-y-1.5">
            <span className="font-bold text-primary block">3. Architect</span>
            <p className="text-[11px] text-muted-foreground leading-relaxed">
              Design the technical spine with <code className="font-mono text-foreground bg-muted px-1 py-0.5 rounded">/bmad-architecture</code> and scaffold epics.
            </p>
          </div>

          <div className="p-3.5 bg-card border border-border rounded-xl space-y-1.5">
            <span className="font-bold text-primary block">4. Build & Verify</span>
            <p className="text-[11px] text-muted-foreground leading-relaxed">
              Drive TDD loops with <code className="font-mono text-foreground bg-muted px-1 py-0.5 rounded">/bmad-build</code>, review diffs, and generate automated tests.
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}
