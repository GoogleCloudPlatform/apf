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

import { X, AlertCircle, AlertTriangle } from "lucide-react"
import { cn } from "@/lib/utils"

export interface AlertModalProps {
  type: "error" | "warning"
  title: string
  description?: string
  details?: string
  onClose: () => void
  action?: {
    label: string
    onClick: () => void
  }
}

export function AlertModal({
  type,
  title,
  description,
  details,
  onClose,
  action,
}: AlertModalProps) {
  const isError = type === "error"
  const Icon = isError ? AlertCircle : AlertTriangle

  return (
    <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/60 backdrop-blur-sm p-4 select-none">
      <div className="bg-card border border-border rounded-2xl max-w-md w-full p-6 shadow-2xl space-y-5 animate-in fade-in zoom-in-95 duration-200 relative">
        {/* Close button top-right */}
        <button
          type="button"
          onClick={onClose}
          className="absolute top-4 right-4 text-muted-foreground hover:text-foreground transition-colors"
          aria-label="Close Alert"
        >
          <X className="h-4 w-4" />
        </button>

        {/* Icon & Title / Description Area */}
        <div className="flex items-start gap-4">
          <div
            className={cn(
              "flex h-12 w-12 shrink-0 items-center justify-center rounded-full",
              isError
                ? "bg-destructive/10 text-destructive"
                : "bg-amber-500/10 text-amber-600 dark:text-amber-400"
            )}
          >
            <Icon className="h-6 w-6" />
          </div>
          <div className="space-y-1.5 pr-4">
            <h3 className="text-base font-bold text-foreground leading-none">
              {title}
            </h3>
            {description && (
              <p className="text-xs text-muted-foreground leading-relaxed">
                {description}
              </p>
            )}
          </div>
        </div>

        {/* Grey Code Box for Technical Details */}
        {details && (
          <div className="p-3 rounded-lg bg-muted/50 border border-border/50 font-mono text-[10px] text-muted-foreground break-words max-h-24 overflow-y-auto">
            <span className="font-semibold text-foreground block mb-0.5">
              Error Details:
            </span>
            {details}
          </div>
        )}

        {/* Footer Actions Tray */}
        <div className="flex justify-end gap-3 pt-2 shrink-0">
          {action && (
            <button
              type="button"
              onClick={action.onClick}
              className="inline-flex h-9 items-center justify-center rounded-lg bg-primary px-4 text-xs font-bold text-primary-foreground hover:opacity-90 shadow transition-opacity"
            >
              {action.label}
            </button>
          )}
          <button
            type="button"
            onClick={onClose}
            className="inline-flex h-9 items-center justify-center rounded-lg border border-input bg-background px-4 text-xs font-semibold text-foreground hover:bg-muted transition-colors"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  )
}

export default AlertModal
