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
import { GitBranch, AlertTriangle, Loader2 } from "lucide-react"

interface BaseBranchConfirmationModalProps {
  isOpen: boolean
  onClose: () => void
  onConfirm: () => void
  baseBranch: string
  repoUrl: string
  isPending: boolean
}

export default function BaseBranchConfirmationModal({
  isOpen,
  onClose,
  onConfirm,
  baseBranch,
  repoUrl,
  isPending,
}: BaseBranchConfirmationModalProps) {
  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm select-none p-4">
      <div className="w-full max-w-[460px] rounded-2xl border border-border bg-card p-6 shadow-2xl text-card-foreground animate-in zoom-in-95 duration-200">
        
        <div className="flex items-center gap-3 mb-4">
          <div className="h-10 w-10 rounded-full bg-amber-500/10 flex items-center justify-center text-amber-500 shrink-0">
            <GitBranch className="h-5 w-5" />
          </div>
          <div>
            <h3 className="text-base font-bold text-foreground">
              Base Branch Not Found
            </h3>
            <p className="text-xs text-muted-foreground">
              Confirmation required before project creation
            </p>
          </div>
        </div>

        <div className="space-y-3 mb-6">
          <div className="rounded-xl border border-amber-500/20 bg-amber-500/5 p-4 text-xs text-muted-foreground leading-relaxed">
            The base branch <strong className="text-foreground font-semibold">{baseBranch}</strong> was not found on the remote repository:
            <div className="mt-1.5 font-mono text-[11px] text-foreground/90 bg-muted/60 px-2.5 py-1 rounded-md border border-border/50 truncate">
              {repoUrl}
            </div>
          </div>

          <p className="text-xs text-muted-foreground leading-relaxed">
            Would you like to create <strong className="text-foreground font-semibold">{baseBranch}</strong> on remote and push it during project creation?
          </p>
        </div>

        <div className="flex justify-end gap-2.5 pt-2">
          <button
            type="button"
            onClick={onClose}
            disabled={isPending}
            className="inline-flex h-9 items-center justify-center rounded-lg border border-input bg-background px-4 py-2 text-xs font-semibold text-foreground hover:bg-muted transition-colors disabled:opacity-50"
          >
            Cancel
          </button>

          <button
            type="button"
            onClick={onConfirm}
            disabled={isPending}
            className="inline-flex h-9 items-center justify-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-xs font-semibold text-primary-foreground shadow hover:opacity-90 disabled:opacity-50 transition-colors"
          >
            {isPending ? (
              <>
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
                Creating...
              </>
            ) : (
              "Confirm & Create Branch"
            )}
          </button>
        </div>
      </div>
    </div>
  )
}
