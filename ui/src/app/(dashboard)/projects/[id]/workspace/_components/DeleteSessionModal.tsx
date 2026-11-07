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

interface DeleteSessionModalProps {
  isOpen: boolean
  onClose: () => void
  onConfirm: () => void
  sessionDate: string
}

export default function DeleteSessionModal({
  isOpen,
  onClose,
  onConfirm,
  sessionDate,
}: DeleteSessionModalProps) {
  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/40 backdrop-blur-sm select-none">
      <div className="w-full max-w-sm rounded-lg border border-border bg-card p-6 shadow-lg animate-in fade-in zoom-in-95 duration-150">
        <h3 className="text-sm font-bold text-foreground mb-1.5">
          Delete Chat Session
        </h3>
        <p className="text-xs text-muted-foreground mb-6 leading-relaxed">
          Are you sure you want to delete the session <span className="font-semibold text-foreground">"{sessionDate}"</span>? This will permanently remove the session and all its message history.
        </p>
        <div className="flex justify-end gap-2">
          <button
            onClick={onClose}
            className="inline-flex h-9 items-center justify-center rounded-md border border-input bg-background px-4 text-xs font-semibold text-foreground hover:bg-muted transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={onConfirm}
            className="inline-flex h-9 items-center justify-center rounded-md bg-destructive px-4 text-xs font-semibold text-destructive-foreground hover:bg-destructive/90 transition-colors shadow"
          >
            Delete
          </button>
        </div>
      </div>
    </div>
  )
}
