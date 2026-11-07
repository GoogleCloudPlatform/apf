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
import { HardDrive, Server, RefreshCw, Activity, CheckCircle2, XCircle, Loader2 } from "lucide-react"

export interface NFSVolumeStats {
  totalBytes: number
  availableBytes: number
  usedBytes: number
  mountPath: string
  isHealthy: boolean
}

export interface ActiveJobState {
  status: string
  executionName?: string
  jobName?: string
}

interface NFSVolumeStatsCardProps {
  volume?: NFSVolumeStats
  totalFolders: number
  isLoading: boolean
  onAuditClick: (dryRun?: boolean) => void
  isAuditing: boolean
  activeJob?: ActiveJobState | null
}

export function formatBytes(bytes: number, decimals: number = 2): string {
  if (bytes === 0) return "0 Bytes"
  const k = 1024
  const dm = decimals < 0 ? 0 : decimals
  const sizes = ["Bytes", "KB", "MB", "GB", "TB", "PB"]
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + " " + sizes[i]
}

export default function NFSVolumeStatsCard({
  volume,
  totalFolders,
  isLoading,
  onAuditClick,
  isAuditing,
  activeJob,
}: NFSVolumeStatsCardProps) {
  const [dryRun, setDryRun] = useState(false)
  const total = volume?.totalBytes || 1
  const used = volume?.usedBytes || 0
  const available = volume?.availableBytes || 0
  const usedPercent = Math.min(Math.round((used / total) * 100), 100)

  const isAuditJobRunning = isAuditing || (activeJob?.jobName === "nfs-audit" && activeJob.status === "PENDING")

  return (
    <div className="bg-card border border-border rounded-2xl p-6 shadow-sm space-y-6">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <div className="h-12 w-12 rounded-2xl bg-primary/10 flex items-center justify-center text-primary shrink-0">
            <HardDrive className="h-6 w-6" />
          </div>
          <div>
            <h2 className="text-base font-bold text-foreground flex flex-wrap items-center gap-2">
              NFS Volume Health & Capacity
              {volume?.isHealthy ? (
                <span className="inline-flex items-center gap-1 text-[10px] font-bold px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-500 border border-emerald-500/20">
                  <CheckCircle2 className="h-3 w-3" /> Healthy
                </span>
              ) : (
                <span className="inline-flex items-center gap-1 text-[10px] font-bold px-2 py-0.5 rounded-full bg-destructive/10 text-destructive border border-destructive/20">
                  <XCircle className="h-3 w-3" /> Degraded
                </span>
              )}
              {isAuditJobRunning && (
                <span
                  data-testid="audit-running-badge"
                  className="inline-flex items-center gap-1 text-[10px] font-bold px-2 py-0.5 rounded-full bg-primary/10 text-primary border border-primary/20 animate-pulse"
                >
                  <Loader2 className="h-3 w-3 animate-spin" /> AUDIT IN PROGRESS
                </span>
              )}
            </h2>
            <p className="text-xs text-muted-foreground font-mono mt-0.5">
              Mount: {volume?.mountPath || "~/workspaces"}
            </p>
          </div>
        </div>

        <div className="flex items-center gap-3">
          <label className="flex items-center gap-1.5 text-xs text-muted-foreground cursor-pointer select-none">
            <input
              type="checkbox"
              checked={dryRun}
              onChange={(e) => setDryRun(e.target.checked)}
              className="rounded border-input text-primary focus:ring-primary/20"
            />
            <span className="font-mono text-[11px]">Dry Run</span>
          </label>

          <button
            onClick={() => onAuditClick(dryRun)}
            disabled={isAuditJobRunning || isLoading}
            aria-label="Run Storage Audit Now"
            className="flex h-10 items-center justify-center gap-2 rounded-xl border border-input bg-background hover:bg-muted text-foreground text-xs font-semibold px-4 transition-all disabled:opacity-50"
          >
            {isAuditJobRunning ? (
              <>
                <Loader2 className="h-3.5 w-3.5 animate-spin text-primary" />
                <span>AUDITING...</span>
              </>
            ) : (
              <>
                <RefreshCw className="h-3.5 w-3.5" />
                <span>Run Storage Audit Now</span>
              </>
            )}
          </button>
        </div>
      </div>

      {/* Progress bar */}
      <div className="space-y-2">
        <div className="flex items-center justify-between text-xs font-medium">
          <span className="text-muted-foreground">Volume Consumption</span>
          <span className="text-foreground font-bold">{usedPercent}% Used</span>
        </div>
        <div className="h-3 w-full bg-muted/60 rounded-full overflow-hidden p-0.5 border border-border">
          <div
            className={`h-full rounded-full transition-all duration-500 ${
              usedPercent > 90
                ? "bg-destructive"
                : usedPercent > 75
                ? "bg-amber-500"
                : "bg-primary"
            }`}
            style={{ width: `${usedPercent}%` }}
          />
        </div>
      </div>

      {/* Metrics Grid */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 pt-2">
        <div className="p-3.5 rounded-xl bg-muted/30 border border-border space-y-1">
          <div className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider">
            Total Capacity
          </div>
          <div className="text-sm font-extrabold text-foreground">
            {formatBytes(total)}
          </div>
        </div>

        <div className="p-3.5 rounded-xl bg-muted/30 border border-border space-y-1">
          <div className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider">
            Used Storage
          </div>
          <div className="text-sm font-extrabold text-foreground">
            {formatBytes(used)}
          </div>
        </div>

        <div className="p-3.5 rounded-xl bg-muted/30 border border-border space-y-1">
          <div className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider">
            Available Space
          </div>
          <div className="text-sm font-extrabold text-emerald-500">
            {formatBytes(available)}
          </div>
        </div>

        <div className="p-3.5 rounded-xl bg-muted/30 border border-border space-y-1">
          <div className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider">
            Organizations
          </div>
          <div className="text-sm font-extrabold text-foreground flex items-center gap-1.5">
            <Server className="h-4 w-4 text-primary" /> {totalFolders}
          </div>
        </div>
      </div>
    </div>
  )
}
