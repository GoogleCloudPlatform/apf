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

import React, { useState } from "react"
import { X, Link } from "lucide-react"
import { WorkflowTemplate } from "@/lib/types"
import WorkflowSelector from "./WorkflowSelector"

interface WorkflowSelectorModalProps {
  isOpen: boolean
  onClose: () => void
  onLink: (workflow: WorkflowTemplate) => void
  projectName?: string
}

export default function WorkflowSelectorModal({
  isOpen,
  onClose,
  onLink,
  projectName,
}: WorkflowSelectorModalProps) {
  const [selectedWorkflow, setSelectedWorkflow] = useState<WorkflowTemplate | null>(null)

  if (!isOpen) return null

  const handleLink = () => {
    if (selectedWorkflow) {
      onLink(selectedWorkflow)
      onClose()
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 md:p-6 bg-slate-950/40 backdrop-blur-sm select-none">
      <div className="w-full max-w-[92vw] h-[85vh] rounded-xl border border-border bg-card shadow-2xl flex flex-col overflow-hidden animate-in fade-in zoom-in-95 duration-150">
        {/* Modal Header */}
        <div className="px-5 py-4 border-b border-border flex items-center justify-between bg-muted/10 shrink-0">
          <div>
            <h2 className="text-sm font-bold text-foreground flex items-center gap-2">
              <Link className="h-4 w-4 text-primary" />
              Link Workflow to Project
            </h2>
            <p className="text-xs text-muted-foreground mt-0.5">
              Select a workflow template to associate with {projectName ? <span className="font-semibold text-foreground">{projectName}</span> : "this project"}.
            </p>
          </div>
          <button
            onClick={onClose}
            className="flex h-7 w-7 items-center justify-center rounded-lg hover:bg-muted text-muted-foreground transition-colors"
            title="Close Dialog"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Modal Content (Workflow Selector) */}
        <div className="flex-1 min-h-0">
          <WorkflowSelector onSelect={setSelectedWorkflow} />
        </div>

        {/* Modal Footer */}
        <div className="px-5 py-3 border-t border-border flex justify-end gap-2 bg-muted/5 shrink-0">
          <button
            onClick={onClose}
            className="inline-flex h-9 items-center justify-center rounded-lg border border-input bg-background px-4 text-xs font-semibold text-foreground hover:bg-muted transition-colors"
          >
            Cancel
          </button>
          <button
            disabled={!selectedWorkflow}
            onClick={handleLink}
            className="inline-flex h-9 items-center justify-center gap-1.5 rounded-lg bg-primary px-4 text-xs font-semibold text-primary-foreground shadow hover:bg-primary/90 disabled:opacity-50 disabled:pointer-events-none transition-all duration-200"
          >
            <Link className="h-3.5 w-3.5" />
            Link Workflow
          </button>
        </div>
      </div>
    </div>
  )
}
