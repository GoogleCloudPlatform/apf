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

import { useEffect, useState } from "react"
import { cn } from "@/lib/utils"
import { 
  Sparkles
} from "lucide-react"
import { useAppStore } from "@/stores/app"
import { GettingStartedSection } from "./_components/GettingStartedSection"
import { BmadReferenceSection } from "./_components/BmadReferenceSection"
import { ArchitectureSection } from "./_components/ArchitectureSection"
import { IntegrationsSection } from "./_components/IntegrationsSection"

type DocSection = "getting-started" | "bmad-reference" | "architecture" | "integrations"

export default function DocsPage() {
  const { setPageTitle } = useAppStore()

  // Active section tracking state - defaults to bmad-reference (other sections hidden from display)
  const [activeSection, setActiveSection] = useState<DocSection>("bmad-reference")

  // Sync header page title
  useEffect(() => {
    setPageTitle("Documentation & Reference")
  }, [setPageTitle])

  return (
    <div className="flex flex-col md:flex-row h-full w-full min-h-[calc(100vh-4rem)]">
      
      {/* 1. Left Sidebar Navigation Column */}
      <aside className="w-full md:w-[260px] border-b md:border-b-0 md:border-r border-border bg-slate-100/50 dark:bg-slate-900/40 py-6 shrink-0 select-none">
        <nav id="tour-docs-nav" className="space-y-2">
          
          {/* Tab: BMAD Reference */}
          <div className="px-3">
            <button
              onClick={() => setActiveSection("bmad-reference")}
              className={cn(
                "flex items-center w-full px-4 py-3.5 gap-3 rounded-xl text-left transition-all duration-200 border relative",
                activeSection === "bmad-reference"
                  ? "bg-primary/5 border-primary/20 text-primary font-bold shadow-sm"
                  : "bg-transparent border-transparent text-muted-foreground hover:bg-muted/50 hover:text-foreground"
              )}
              data-testid="docs-nav-bmad-reference"
            >
              <span className={cn(
                "w-1 h-6 rounded shrink-0 transition-colors duration-200",
                activeSection === "bmad-reference" ? "bg-primary" : "bg-transparent"
              )} />
              
              <Sparkles className={cn(
                "h-5 w-5 shrink-0 transition-colors",
                activeSection === "bmad-reference" ? "text-primary" : "text-muted-foreground"
              )} />
              
              <div className="min-w-0 flex-1">
                <span className={cn(
                  "block text-[13px] leading-tight transition-colors",
                  activeSection === "bmad-reference" ? "font-bold text-primary" : "font-normal text-foreground"
                )}>
                  BMAD Reference
                </span>
                <span className="block text-[10px] font-normal text-muted-foreground truncate mt-0.5">
                  Phases, skills & workflows
                </span>
              </div>
            </button>
          </div>

        </nav>
      </aside>

      {/* 2. Main Tab Viewport: Only actively selected tab content is displayed */}
      <main className="flex-1 overflow-y-auto p-6 md:p-8">
        <div className="w-full animate-in fade-in duration-200">
          {activeSection === "getting-started" && <GettingStartedSection />}
          {activeSection === "bmad-reference" && <BmadReferenceSection />}
          {activeSection === "architecture" && <ArchitectureSection />}
          {activeSection === "integrations" && <IntegrationsSection />}
        </div>
      </main>
    </div>
  )
}
