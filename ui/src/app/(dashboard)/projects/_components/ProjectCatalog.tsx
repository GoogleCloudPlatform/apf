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

import { Project, WorkflowTemplate } from "@/lib/types"
import ProjectCard from "@/components/project/ProjectCard"
import ProjectRow from "@/components/project/ProjectRow"

interface ProjectCatalogProps {
  viewMode: "card" | "list"
  activeProjects: Project[]
  bmadWorkflow: WorkflowTemplate
  isBrowse: boolean
  onRequestAccess: (projectId: string) => Promise<void>
}

export function ProjectCatalog({
  viewMode,
  activeProjects,
  bmadWorkflow,
  isBrowse,
  onRequestAccess,
}: ProjectCatalogProps) {
  if (viewMode === "card") {
    return (
      <div 
        id="tour-projects-list" 
        data-testid="projects-grid"
        className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-[repeat(auto-fill,minmax(390px,1fr))] gap-4 pb-12 w-full max-w-full overflow-x-hidden"
      >
        {activeProjects.map((project) => (
          <ProjectCard
            key={project.id}
            project={project}
            bmadWorkflow={bmadWorkflow}
            isBrowse={isBrowse}
            onRequestAccess={onRequestAccess}
          />
        ))}
      </div>
    )
  }

  return (
    <div className="border border-border rounded-xl overflow-hidden bg-card pb-12">
      <table className="w-full text-left border-collapse">
        <thead>
          <tr className="border-b border-border bg-muted/20 text-xs font-semibold text-muted-foreground">
            <th className="p-4 w-12"></th>
            <th className="p-4">Project</th>
            <th className="p-4">Description</th>
            {!isBrowse && <th className="p-4 text-right">Stage</th>}
            <th className="p-4 text-right">Actions</th>
          </tr>
        </thead>
        <tbody>
          {activeProjects.map((project) => (
            <ProjectRow
              key={project.id}
              project={project}
              bmadWorkflow={bmadWorkflow}
              isBrowse={isBrowse}
              onRequestAccess={onRequestAccess}
            />
          ))}
        </tbody>
      </table>
    </div>
  )
}
export default ProjectCatalog
