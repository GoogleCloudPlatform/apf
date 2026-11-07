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
import { FolderGit2 } from "lucide-react"

interface BmadConfigCardProps {
  specFolder: string
  setSpecFolder: (v: string) => void
  implementationArtifactsPath: string
  setImplementationArtifactsPath: (v: string) => void
  planningArtifactsPath: string
  setPlanningArtifactsPath: (v: string) => void
  testingArtifactsPath: string
  setTestingArtifactsPath: (v: string) => void
}

export default function BmadConfigCard({
  specFolder,
  setSpecFolder,
  implementationArtifactsPath,
  setImplementationArtifactsPath,
  planningArtifactsPath,
  setPlanningArtifactsPath,
  testingArtifactsPath,
  setTestingArtifactsPath,
}: BmadConfigCardProps) {
  const specPrefix = (specFolder.trim().replace(/\/+$/, "") || "docs/spec") + "/"

  return (
    <div id="tour-bmad-config-card" className="rounded-xl border border-border bg-card p-6 shadow-sm space-y-5">
      <div className="flex items-center gap-2 border-b border-border pb-3 mb-2">
        <FolderGit2 className="h-4 w-4 text-primary" />
        <h2 className="text-sm font-semibold text-foreground">BMAD Configuration</h2>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-5">

        {/* Spec Folder */}
        <div className="space-y-1.5 md:col-span-2">
          <label className="text-xs font-semibold text-muted-foreground uppercase" htmlFor="specFolder">
            Spec Folder
          </label>
          <input
            id="specFolder"
            type="text"
            placeholder="docs/spec"
            value={specFolder}
            onChange={(e) => setSpecFolder(e.target.value)}
            className="w-full h-10 px-3.5 rounded-md border border-input bg-transparent text-sm focus:outline-none focus:ring-2 focus:ring-primary/40 focus:border-primary"
          />
        </div>

        {/* Planning Artifacts Folder */}
        <div className="space-y-1.5">
          <label className="text-xs font-semibold text-muted-foreground uppercase" htmlFor="planningArtifactsPath">
            Planning Artifacts Folder
          </label>
          <div className="flex rounded-md border border-input bg-transparent focus-within:ring-2 focus-within:ring-primary/40 focus-within:border-primary overflow-hidden">
            <span className="inline-flex items-center px-3 bg-muted text-muted-foreground border-r border-input text-xs font-mono select-none shrink-0">
              {specPrefix}
            </span>
            <input
              id="planningArtifactsPath"
              type="text"
              placeholder="plans"
              value={planningArtifactsPath}
              onChange={(e) => setPlanningArtifactsPath(e.target.value)}
              className="flex-1 h-10 px-3 bg-transparent text-sm focus:outline-none min-w-0"
            />
          </div>
        </div>

        {/* Implementation Artifacts Folder */}
        <div className="space-y-1.5">
          <label className="text-xs font-semibold text-muted-foreground uppercase" htmlFor="implementationArtifactsPath">
            Implementation Artifacts Folder
          </label>
          <div className="flex rounded-md border border-input bg-transparent focus-within:ring-2 focus-within:ring-primary/40 focus-within:border-primary overflow-hidden">
            <span className="inline-flex items-center px-3 bg-muted text-muted-foreground border-r border-input text-xs font-mono select-none shrink-0">
              {specPrefix}
            </span>
            <input
              id="implementationArtifactsPath"
              type="text"
              placeholder="implementation"
              value={implementationArtifactsPath}
              onChange={(e) => setImplementationArtifactsPath(e.target.value)}
              className="flex-1 h-10 px-3 bg-transparent text-sm focus:outline-none min-w-0"
            />
          </div>
        </div>

        {/* Testing Artifacts Folder */}
        <div className="space-y-1.5 md:col-span-2">
          <label className="text-xs font-semibold text-muted-foreground uppercase" htmlFor="testingArtifactsPath">
            Testing Artifacts Folder
          </label>
          <div className="flex rounded-md border border-input bg-transparent focus-within:ring-2 focus-within:ring-primary/40 focus-within:border-primary overflow-hidden">
            <span className="inline-flex items-center px-3 bg-muted text-muted-foreground border-r border-input text-xs font-mono select-none shrink-0">
              {specPrefix}
            </span>
            <input
              id="testingArtifactsPath"
              type="text"
              placeholder="test"
              value={testingArtifactsPath}
              onChange={(e) => setTestingArtifactsPath(e.target.value)}
              className="flex-1 h-10 px-3 bg-transparent text-sm focus:outline-none min-w-0"
            />
          </div>
        </div>
      </div>
    </div>
  )
}
