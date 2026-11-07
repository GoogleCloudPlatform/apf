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
import { Loader2 } from "lucide-react"

interface DeleteProjectModalProps {
  isOpen: boolean
  onClose: () => void
  onConfirm: () => void
  projectName: string
  isPending: boolean
}

export default function DeleteProjectModal({
  isOpen,
  onClose,
  onConfirm,
  projectName,
  isPending,
}: DeleteProjectModalProps) {
  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm select-none">
      <div className="w-full max-w-[400px] rounded-xl border border-border bg-card p-6 shadow-2xl text-card-foreground animate-in zoom-in-95 duration-200">
        <h3 className="text-base font-bold text-destructive mb-2">Delete Project</h3>
        <p className="text-xs text-muted-foreground leading-relaxed mb-6">
          Are you sure you want to permanently delete <strong className="text-foreground">{projectName}</strong>? This action is final and will wipe all associated workspace properties database histories.
        </p>

        <div className="flex justify-end gap-2.5">
          <button
            type="button"
            onClick={onClose}
            className="inline-flex h-9 items-center justify-center rounded-lg border border-input bg-background px-4 py-2 text-xs font-semibold text-foreground hover:bg-muted"
          >
            Cancel
          </button>
          
          <button
            type="button"
            onClick={onConfirm}
            disabled={isPending}
            className="inline-flex h-9 items-center justify-center gap-1.5 rounded-lg bg-destructive px-4 py-2 text-xs font-semibold text-destructive-foreground shadow hover:opacity-90 disabled:opacity-50"
          >
            {isPending ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              "Delete Project"
            )}
          </button>
        </div>
      </div>
    </div>
  )
}
