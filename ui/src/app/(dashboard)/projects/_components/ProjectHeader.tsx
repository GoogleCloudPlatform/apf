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

import { LayoutGrid, List } from "lucide-react"
import { useOnboardingStore } from "@/stores/onboarding"
import { useRouter } from "next/navigation"

interface ProjectHeaderProps {
  activeTab: "my" | "shared"
  setActiveTab: (tab: "my" | "shared") => void
  viewMode: "card" | "list"
  setViewMode: (mode: "card" | "list") => void
  onSharedProjectsClick?: () => void
}

export function ProjectHeader({
  activeTab,
  setActiveTab,
  viewMode,
  setViewMode,
  onSharedProjectsClick,
}: ProjectHeaderProps) {
  const router = useRouter()
  return (
    <div 
      id="tour-catalog-tabs" 
      data-testid="projects-header-tabs"
      className="border-b border-border pb-2 select-none flex flex-col sm:flex-row justify-between items-start sm:items-center gap-3 sm:gap-6 w-full max-w-full overflow-x-hidden"
    >
      {/* Navigation tabs */}
      <div className="flex items-center gap-2 sm:gap-6 overflow-x-auto w-full sm:w-auto">
        <button
          id="tour-tab-my-projects"
          onClick={() => setActiveTab("my")}
          className={`min-h-[48px] px-3 py-2 text-sm font-semibold relative transition-colors touch-manipulation flex items-center justify-center ${
            activeTab === "my"
              ? "text-primary font-bold"
              : "text-muted-foreground hover:text-foreground"
          }`}
        >
          My Projects
          {activeTab === "my" && (
            <span className="absolute bottom-0 left-0 right-0 h-1 bg-primary rounded-t" />
          )}
        </button>
        
        <button
          id="tour-tab-shared-projects"
          onClick={() => {
            if (onSharedProjectsClick) {
              onSharedProjectsClick()
            } else {
              setActiveTab("shared")
            }
            useOnboardingStore.getState().advanceStep("catalog", 16)
          }}
          className={`min-h-[48px] px-3 py-2 text-sm font-semibold relative transition-colors touch-manipulation flex items-center justify-center ${
            activeTab === "shared"
              ? "text-primary font-bold"
              : "text-muted-foreground hover:text-foreground"
          }`}
        >
          Shared Projects
          {activeTab === "shared" && (
            <span className="absolute bottom-0 left-0 right-0 h-1 bg-primary rounded-t" />
          )}
        </button>
      </div>

      {/* View Mode Toggle */}
      <div className="flex items-center gap-2 shrink-0">
        <button
          onClick={() => setViewMode("card")}
          className={`min-h-[48px] min-w-[48px] p-2 rounded-lg transition-colors touch-manipulation flex items-center justify-center ${
            viewMode === "card"
              ? "bg-muted text-foreground"
              : "text-muted-foreground hover:text-foreground"
          }`}
          title="Card View"
        >
          <LayoutGrid className="h-5 w-5" />
        </button>
        <button
          id="tour-toggle-list-view"
          onClick={() => {
            setViewMode("list")
            useOnboardingStore.getState().advanceStep("catalog", 15)
          }}
          className={`min-h-[48px] min-w-[48px] p-2 rounded-lg transition-colors touch-manipulation flex items-center justify-center ${
            viewMode === "list"
              ? "bg-muted text-foreground"
              : "text-muted-foreground hover:text-foreground"
          }`}
          title="List View"
        >
          <List className="h-5 w-5" />
        </button>
      </div>
    </div>
  )
}
export default ProjectHeader
