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

import { useState } from "react"
import { Play, Loader2, AlertTriangle, ShieldCheck, X } from "lucide-react"

export interface JobParameterInfo {
  name: string
  type: string
  description: string
  required: boolean
  defaultValue?: any
}

export interface JobCatalogItem {
  name: string
  description: string
  parameters: JobParameterInfo[]
}

interface JobRunModalProps {
  isOpen: boolean
  job: JobCatalogItem | null
  onClose: () => void
  onRun: (jobName: string, dryRun: boolean) => Promise<void>
  isRunning: boolean
}

export default function JobRunModal({
  isOpen,
  job,
  onClose,
  onRun,
  isRunning,
}: JobRunModalProps) {
  const [dryRun, setDryRun] = useState(true)

  if (!isOpen || !job) return null

  const hasDryRunParam = job.parameters?.some((p) => p.name === "dryRun") ?? true

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    await onRun(job.name, dryRun)
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4 animate-in fade-in duration-150"
      role="dialog"
      aria-modal="true"
      aria-labelledby="modal-title"
    >
      <div className="w-full max-w-lg bg-card border border-border rounded-2xl p-6 shadow-2xl space-y-6 animate-in zoom-in-95 duration-200 text-foreground">
        {/* Header */}
        <div className="flex items-start justify-between">
          <div className="space-y-1">
            <h3 id="modal-title" className="text-lg font-bold text-foreground flex items-center gap-2">
              <Play className="h-5 w-5 text-primary" />
              Run System Job: <span className="font-mono text-primary">{job.name}</span>
            </h3>
            <p className="text-xs text-muted-foreground leading-relaxed">
              {job.description || "Execute this housekeeping operation on the underlying infrastructure."}
            </p>
          </div>
          <button
            onClick={onClose}
            disabled={isRunning}
            className="p-1 rounded-lg text-muted-foreground hover:text-foreground hover:bg-muted transition-colors disabled:opacity-50"
            aria-label="Close"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} className="space-y-5">
          {hasDryRunParam && (
            <div className="p-4 rounded-xl border border-border bg-muted/40 space-y-3">
              <div className="flex items-start gap-3">
                <input
                  type="checkbox"
                  id="dry-run-checkbox"
                  checked={dryRun}
                  onChange={(e) => setDryRun(e.target.checked)}
                  disabled={isRunning}
                  className="mt-1 h-4 w-4 rounded border-border text-primary focus:ring-primary"
                />
                <label htmlFor="dry-run-checkbox" className="text-xs space-y-1 cursor-pointer select-none">
                  <span className="font-semibold text-foreground flex items-center gap-1.5">
                    <ShieldCheck className="h-4 w-4 text-emerald-500" />
                    Dry Run Mode (Recommended)
                  </span>
                  <p className="text-muted-foreground">
                    Simulate execution to inspect actions without deleting files, changing quotas, or altering storage state.
                  </p>
                </label>
              </div>
            </div>
          )}

          {!dryRun && (
            <div className="p-3.5 rounded-xl border border-amber-500/30 bg-amber-500/10 text-amber-600 dark:text-amber-400 flex items-start gap-2.5 text-xs">
              <AlertTriangle className="h-4 w-4 shrink-0 mt-0.5" />
              <span>
                <strong>Warning:</strong> Live execution will apply actual mutations to workspaces, files, or metadata.
              </span>
            </div>
          )}

          {/* Action Buttons */}
          <div className="flex justify-end gap-3 pt-2">
            <button
              type="button"
              onClick={onClose}
              disabled={isRunning}
              className="px-4 py-2 rounded-lg border border-input bg-background hover:bg-muted text-xs font-semibold text-foreground transition-all disabled:opacity-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isRunning}
              className={`px-4 py-2 rounded-lg text-xs font-semibold shadow-sm flex items-center gap-1.5 transition-all disabled:opacity-50 ${
                dryRun
                  ? "bg-primary text-primary-foreground hover:bg-primary/90"
                  : "bg-amber-600 text-white hover:bg-amber-700"
              }`}
            >
              {isRunning ? (
                <>
                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                  Dispatching...
                </>
              ) : (
                <>
                  <Play className="h-3.5 w-3.5" />
                  {dryRun ? "Execute Dry Run" : "Execute Live Job"}
                </>
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
