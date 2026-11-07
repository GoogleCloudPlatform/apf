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
import { Loader2, AlertCircle, ExternalLink, UserPlus } from "lucide-react"

interface DuplicateRepoModalProps {
  isOpen: boolean
  onClose: () => void
  onJoinProject: () => void
  onGoToProject: () => void
  projectName: string
  repoUrl: string
  hasAccess: boolean
  potentialRole: string // "Editor", "Viewer", or ""
  isPending: boolean // join pending
}

export default function DuplicateRepoModal({
  isOpen,
  onClose,
  onJoinProject,
  onGoToProject,
  projectName,
  repoUrl,
  hasAccess,
  potentialRole,
  isPending,
}: DuplicateRepoModalProps) {
  if (!isOpen) return null

  const hasGitAccess = potentialRole === "Editor" || potentialRole === "Viewer"

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm select-none">
      <div className="w-full max-w-[420px] rounded-xl border border-border bg-card p-6 shadow-2xl text-card-foreground animate-in zoom-in-95 duration-200">
        <div className="flex items-center gap-2 text-amber-500 mb-2">
          <AlertCircle className="h-5 w-5" />
          <h3 className="text-base font-bold text-foreground">
            Repository Already Connected
          </h3>
        </div>
        
        <p className="text-xs text-muted-foreground leading-relaxed mb-4">
          The repository <span className="font-semibold text-foreground break-all">{repoUrl}</span> is already connected to another project: <span className="font-semibold text-foreground">{projectName}</span>.
        </p>

        {hasAccess ? (
          <p className="text-xs text-muted-foreground leading-relaxed mb-6">
            You already have access to this project. You can go directly to its workspace.
          </p>
        ) : hasGitAccess ? (
          <p className="text-xs text-muted-foreground leading-relaxed mb-6">
            Based on your GitHub repository permissions, you can automatically join this project as a <span className="font-semibold text-foreground">{potentialRole}</span>.
          </p>
        ) : (
          <p className="text-xs text-muted-foreground leading-relaxed mb-6">
            You do not have access to this project, and we could not verify your permissions on the GitHub repository. Please ensure you have access to the repository on GitHub or contact the project owner.
          </p>
        )}
        
        <div className="flex justify-end gap-2.5">
          <button
            type="button"
            onClick={onClose}
            className="inline-flex h-9 items-center justify-center rounded-lg border border-input bg-background px-4 py-2 text-xs font-semibold text-foreground hover:bg-muted"
          >
            Cancel
          </button>
          
          {hasAccess ? (
            <button
              type="button"
              onClick={onGoToProject}
              className="inline-flex h-9 items-center justify-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-xs font-semibold text-primary-foreground shadow hover:opacity-90"
            >
              Go to Project
              <ExternalLink className="h-3.5 w-3.5" />
            </button>
          ) : hasGitAccess ? (
            <button
              type="button"
              onClick={onJoinProject}
              disabled={isPending}
              className="inline-flex h-9 items-center justify-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-xs font-semibold text-primary-foreground shadow hover:opacity-90 disabled:opacity-50"
            >
              {isPending ? (
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
              ) : (
                <>
                  Join Project
                  <UserPlus className="h-3.5 w-3.5" />
                </>
              )}
            </button>
          ) : null}
        </div>
      </div>
    </div>
  )
}
