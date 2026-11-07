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
import { AlertTriangle, Loader2 } from "lucide-react"

interface RepoChangeWarningModalProps {
  isOpen: boolean
  onClose: () => void
  onConfirm: () => void
  initialRepoUrl?: string
  newRepoUrl: string
  isPending?: boolean
}

export default function RepoChangeWarningModal({
  isOpen,
  onClose,
  onConfirm,
  initialRepoUrl,
  newRepoUrl,
  isPending,
}: RepoChangeWarningModalProps) {
  if (!isOpen) return null

  const isSwitching = Boolean(initialRepoUrl && initialRepoUrl.trim())

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm select-none p-4">
      <div className="w-full max-w-[460px] rounded-2xl border border-border bg-card p-6 shadow-2xl text-card-foreground animate-in zoom-in-95 duration-200">
        
        <div className="flex items-center gap-3 mb-4">
          <div className="h-10 w-10 rounded-full bg-amber-500/10 flex items-center justify-center text-amber-500 shrink-0">
            <AlertTriangle className="h-5 w-5" />
          </div>
          <div>
            <h3 className="text-base font-bold text-foreground">
              {isSwitching ? "Change Repository" : "Connect Repository"}
            </h3>
            <p className="text-xs text-muted-foreground">
              Warning: Workspace data will be overwritten
            </p>
          </div>
        </div>

        <div className="space-y-3 mb-6">
          <div className="rounded-xl border border-amber-500/20 bg-amber-500/5 p-4 text-xs text-muted-foreground leading-relaxed">
            {isSwitching ? (
              <>
                Changing your repository from{" "}
                <strong className="text-foreground font-semibold font-mono text-[11px] block truncate my-1">
                  {initialRepoUrl}
                </strong>
                to{" "}
                <strong className="text-foreground font-semibold font-mono text-[11px] block truncate my-1">
                  {newRepoUrl}
                </strong>
                will delete your currently cached local workspace data. Any unsynced local changes will be lost.
              </>
            ) : (
              <>
                Connecting repository{" "}
                <strong className="text-foreground font-semibold font-mono text-[11px] block truncate my-1">
                  {newRepoUrl}
                </strong>
                will replace your current workspace data with the contents of the repository. Any unsynced local changes will be lost.
              </>
            )}
          </div>

          <p className="text-xs text-muted-foreground leading-relaxed">
            Do you want to proceed with this change?
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
                Proceeding...
              </>
            ) : (
              "Proceed"
            )}
          </button>
        </div>
      </div>
    </div>
  )
}
