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

import { FolderPlus, Plus } from "lucide-react"

interface EmptyProjectsStateProps {
  activeTab: "my" | "shared"
  onCreateProject: () => void
}

export function EmptyProjectsState({
  activeTab,
  onCreateProject,
}: EmptyProjectsStateProps) {
  return (
    <div className="flex h-[350px] w-full flex-col items-center justify-center text-center gap-3 border border-dashed border-border bg-muted/20 rounded-2xl p-8 max-w-lg mx-auto select-none mt-6">
      <FolderPlus className="h-12 w-12 text-muted-foreground opacity-40" />
      <h3 className="font-bold text-base text-foreground">No projects found</h3>
      <p className="text-xs text-muted-foreground leading-relaxed max-w-xs">
        {activeTab === "my"
          ? "You haven't joined or created any projects yet. Create one now or browse the Shared catalog tab!"
          : "There are no public shared projects listed in the Foundry database right now."}
      </p>
      {activeTab === "my" && (
        <button
          onClick={onCreateProject}
          className="inline-flex h-9 items-center justify-center gap-2 rounded-lg bg-primary px-4 py-2 text-xs font-semibold text-primary-foreground shadow transition-all hover:opacity-90 mt-2"
        >
          <Plus className="h-4 w-4" />
          Get Started
        </button>
      )}
    </div>
  )
}
export default EmptyProjectsState
