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

import { Loader2, AlertTriangle } from "lucide-react"

export interface ConfirmDialogProps {
  isOpen: boolean
  onClose: () => void
  onConfirm: () => void
  title: string
  message: string
  confirmText?: string
  cancelText?: string
  isDestructive?: boolean
  isLoading?: boolean
}

export function ConfirmDialog({
  isOpen,
  onClose,
  onConfirm,
  title,
  message,
  confirmText = "Confirm",
  cancelText = "Cancel",
  isDestructive = false,
  isLoading = false,
}: ConfirmDialogProps) {
  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/60 backdrop-blur-sm p-4 select-none">
      <div className="bg-card border border-border rounded-2xl max-w-md w-full p-6 shadow-2xl space-y-5 animate-in fade-in zoom-in-95 duration-200 relative">
        <div className="space-y-1.5 pr-4">
          <h3 className="text-base font-bold text-foreground leading-none flex items-center gap-2">
            {isDestructive && <AlertTriangle className="h-5 w-5 text-destructive" />}
            {title}
          </h3>
          <p className="text-xs text-muted-foreground leading-relaxed">
            {message}
          </p>
        </div>

        <div className="flex justify-end gap-3 pt-2 shrink-0">
          <button
            type="button"
            onClick={onClose}
            disabled={isLoading}
            className="inline-flex h-10 items-center justify-center rounded-xl border border-input bg-background px-4 text-xs font-semibold text-foreground hover:bg-muted disabled:opacity-50 transition-colors"
          >
            {cancelText}
          </button>
          <button
            type="button"
            onClick={onConfirm}
            disabled={isLoading}
            className={`inline-flex h-10 items-center justify-center gap-1.5 rounded-xl px-4 text-xs font-semibold shadow hover:opacity-90 disabled:opacity-50 transition-colors ${
              isDestructive
                ? "bg-destructive text-destructive-foreground"
                : "bg-primary text-primary-foreground"
            }`}
          >
            {isLoading && <Loader2 className="h-4 w-4 animate-spin" />}
            {confirmText}
          </button>
        </div>
      </div>
    </div>
  )
}

export default ConfirmDialog
