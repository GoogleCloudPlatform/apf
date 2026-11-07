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

import React, { useState, useEffect } from "react"
import { X, ShieldAlert, Lock, Unlock, HardDrive, AlertCircle, Loader2 } from "lucide-react"
import { formatBytes } from "./NFSVolumeStatsCard"

export interface NFSFolderMetadata {
  folderId: string
  organizationId: string
  organizationName?: string
  projectId: string
  workspacePath: string
  quotaSizeBytes: number
  usedSizeBytes: number
  fileCount: number
  isReadOnly?: boolean
  retentionPolicy: string
  lastAuditedAt?: string
  createdAt?: string
  updatedAt?: string
}

interface EditQuotaModalProps {
  folder: NFSFolderMetadata | null
  isOpen: boolean
  onClose: () => void
  onSave: (
    folderId: string,
    updates: {
      quotaSizeBytes?: number
      retentionPolicy?: string
    }
  ) => Promise<void>
}

export default function EditQuotaModal({
  folder,
  isOpen,
  onClose,
  onSave,
}: EditQuotaModalProps) {
  const [quotaGB, setQuotaGB] = useState<number>(10)
  const [retentionPolicy, setRetentionPolicy] = useState<string>("90d")
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (folder) {
      const gb = Math.round((folder.quotaSizeBytes / (1024 * 1024 * 1024)) * 100) / 100
      setQuotaGB(gb || 10)
      setRetentionPolicy(folder.retentionPolicy || "90d")
      setError(null)
    }
  }, [folder])

  if (!isOpen || !folder) return null

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)

    if (quotaGB <= 0) {
      setError("Storage quota must be greater than 0 GB")
      return
    }

    const quotaSizeBytes = Math.round(quotaGB * 1024 * 1024 * 1024)

    setIsSubmitting(true)
    try {
      await onSave(folder.folderId, {
        quotaSizeBytes,
        retentionPolicy,
      })
      onClose()
    } catch (err: any) {
      setError(err.message || "Failed to update folder metadata")
    } finally {
      setIsSubmitting(false)
    }
  }

  const presets = [5, 10, 25, 50, 100]

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-background/80 backdrop-blur-sm transition-opacity">
      <div className="w-full max-w-lg rounded-2xl border border-border bg-card p-6 md:p-8 shadow-2xl space-y-6 text-left text-card-foreground">
        
        {/* Header */}
        <div className="flex items-center justify-between border-b border-border pb-4">
          <div className="flex items-center gap-3">
            <div className="h-10 w-10 rounded-xl bg-primary/10 flex items-center justify-center text-primary shrink-0">
              <HardDrive className="h-5 w-5" />
            </div>
            <div>
              <h2 className="text-base font-bold text-foreground">
                Edit Organization Quota
              </h2>
              <p className="text-xs text-muted-foreground font-mono truncate max-w-[280px]">
                {folder.folderId}
              </p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="h-8 w-8 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground flex items-center justify-center transition-colors"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Error Alert */}
        {error && (
          <div className="p-3.5 rounded-xl bg-destructive/10 border border-destructive/20 text-destructive text-xs flex items-center gap-2.5">
            <AlertCircle className="h-4 w-4 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        {/* Current Info Card */}
        <div className="p-3.5 rounded-xl bg-muted/40 border border-border text-xs space-y-1">
          <div className="flex justify-between">
            <span className="text-muted-foreground">Org ID:</span>
            <span className="font-semibold text-foreground">{folder.organizationId}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">Used Storage:</span>
            <span className="font-bold text-foreground">{formatBytes(folder.usedSizeBytes)} ({folder.fileCount} files)</span>
          </div>
        </div>

        <form onSubmit={handleSubmit} className="space-y-5">
          
          {/* Quota Limit Input */}
          <div className="space-y-2">
            <label className="text-xs font-semibold text-muted-foreground uppercase tracking-wider block">
              Storage Quota Limit (GB)
            </label>
            <div className="flex items-center gap-2">
              <input
                type="number"
                step="0.1"
                min="0.5"
                required
                value={quotaGB}
                onChange={(e) => setQuotaGB(parseFloat(e.target.value) || 0)}
                className="w-full h-11 px-4 rounded-xl border border-input bg-background text-sm font-bold text-foreground focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all"
              />
              <span className="text-xs font-bold text-muted-foreground px-2">GB</span>
            </div>

            {/* Presets */}
            <div className="flex flex-wrap gap-2 pt-1">
              {presets.map((p) => (
                <button
                  key={p}
                  type="button"
                  onClick={() => setQuotaGB(p)}
                  className={`px-3 py-1 rounded-lg border text-xs font-semibold transition-all ${
                    quotaGB === p
                      ? "bg-primary text-primary-foreground border-primary"
                      : "bg-background border-border hover:bg-muted text-muted-foreground"
                  }`}
                >
                  {p} GB
                </button>
              ))}
            </div>
          </div>

          {/* Retention Policy */}
          <div className="space-y-1">
            <label className="text-xs font-semibold text-muted-foreground uppercase tracking-wider block">
              Data Retention Policy
            </label>
            <select
              value={retentionPolicy}
              onChange={(e) => setRetentionPolicy(e.target.value)}
              className="w-full h-11 px-4 rounded-xl border border-input bg-background text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all"
            >
              <option value="30d">30 Days Retention (30d)</option>
              <option value="90d">90 Days Retention (90d)</option>
              <option value="180d">180 Days Retention (180d)</option>
              <option value="365d">365 Days Retention (365d)</option>
              <option value="indefinite">Indefinite (No Auto-Purge)</option>
            </select>
          </div>

          {/* Submit buttons */}
          <div className="flex items-center gap-3 pt-4 border-t border-border">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 h-11 rounded-xl border border-input hover:bg-muted text-muted-foreground text-sm font-semibold transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isSubmitting}
              className="flex-1 h-11 rounded-xl bg-primary hover:opacity-90 text-primary-foreground text-sm font-bold shadow-md transition-all flex items-center justify-center gap-2 disabled:opacity-50"
            >
              {isSubmitting ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                "SAVE METADATA"
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
