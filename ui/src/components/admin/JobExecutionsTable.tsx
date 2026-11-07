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

import { useState, useMemo } from "react"
import {
  History,
  Search,
  CheckCircle2,
  AlertCircle,
  Clock,
  ExternalLink,
  Loader2,
  Slash,
  Filter,
  ShieldCheck,
  Zap,
} from "lucide-react"

export interface JobExecutionItem {
  id: string
  name: string
  jobName: string
  dryRun: boolean
  status: string
  createTime?: string
  startTime?: string
  completionTime?: string
  durationSeconds: number
  durationFormatted: string
  errorMessage?: string
  logUri?: string
}

interface JobExecutionsTableProps {
  executions: JobExecutionItem[]
  isLoading: boolean
}

export default function JobExecutionsTable({
  executions,
  isLoading,
}: JobExecutionsTableProps) {
  const [searchQuery, setSearchQuery] = useState("")
  const [statusFilter, setStatusFilter] = useState("ALL")

  const filteredExecutions = useMemo(() => {
    return executions.filter((item) => {
      const matchesSearch =
        searchQuery === "" ||
        item.jobName.toLowerCase().includes(searchQuery.toLowerCase()) ||
        item.id.toLowerCase().includes(searchQuery.toLowerCase())

      const matchesStatus =
        statusFilter === "ALL" || item.status.toUpperCase() === statusFilter.toUpperCase()

      return matchesSearch && matchesStatus
    })
  }, [executions, searchQuery, statusFilter])

  const renderStatusBadge = (status: string, errorMessage?: string) => {
    const s = status.toUpperCase()
    switch (s) {
      case "SUCCEEDED":
        return (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20">
            <CheckCircle2 className="h-3.5 w-3.5" />
            Succeeded
          </span>
        )
      case "FAILED":
        return (
          <span
            className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-rose-500/10 text-rose-600 dark:text-rose-400 border border-rose-500/20"
            title={errorMessage || "Job execution failed"}
          >
            <AlertCircle className="h-3.5 w-3.5" />
            Failed
          </span>
        )
      case "RUNNING":
      case "PENDING":
        return (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/20 animate-pulse">
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
            {s === "PENDING" ? "Pending" : "Running"}
          </span>
        )
      case "CANCELLED":
        return (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-zinc-500/10 text-zinc-600 dark:text-zinc-400 border border-zinc-500/20">
            <Slash className="h-3.5 w-3.5" />
            Cancelled
          </span>
        )
      default:
        return (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-muted text-muted-foreground border border-border">
            <Clock className="h-3.5 w-3.5" />
            {status}
          </span>
        )
    }
  }

  const formatTimestamp = (ts?: string) => {
    if (!ts) return "—"
    try {
      const d = new Date(ts)
      return d.toLocaleString("en-US", {
        month: "short",
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
      })
    } catch {
      return ts
    }
  }

  return (
    <div className="rounded-2xl border border-border bg-card shadow-sm space-y-4 p-6 text-foreground">
      {/* Header & Filters */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div className="space-y-1">
          <h3 className="text-base font-bold text-foreground flex items-center gap-2">
            <History className="h-5 w-5 text-primary" />
            Job Execution History & Live Monitor
          </h3>
          <p className="text-xs text-muted-foreground">
            Real-time execution outcomes and metrics queried live from Google Cloud Run.
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-2.5">
          {/* Search */}
          <div className="relative w-full sm:w-64">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
            <input
              type="text"
              placeholder="Search by job or ID..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-9 pr-3 py-1.5 rounded-lg border border-input bg-background text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-primary"
            />
          </div>

          {/* Status Filter */}
          <div className="flex items-center gap-1.5">
            <Filter className="h-3.5 w-3.5 text-muted-foreground" />
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value)}
              className="px-2.5 py-1.5 rounded-lg border border-input bg-background text-xs text-foreground font-medium focus:outline-none focus:ring-1 focus:ring-primary"
              aria-label="Filter by status"
            >
              <option value="ALL">All Statuses</option>
              <option value="RUNNING">Running / Pending</option>
              <option value="SUCCEEDED">Succeeded</option>
              <option value="FAILED">Failed</option>
              <option value="CANCELLED">Cancelled</option>
            </select>
          </div>
        </div>
      </div>

      {/* Table Content */}
      <div className="overflow-x-auto rounded-xl border border-border">
        <table className="w-full text-left text-xs">
          <thead className="bg-muted/50 border-b border-border text-muted-foreground font-semibold">
            <tr>
              <th className="py-3 px-4">Execution</th>
              <th className="py-3 px-4">Job Name</th>
              <th className="py-3 px-4">Mode</th>
              <th className="py-3 px-4">Status</th>
              <th className="py-3 px-4">Started At</th>
              <th className="py-3 px-4">Duration</th>
              <th className="py-3 px-4 text-right">Logs</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border/60">
            {isLoading ? (
              <tr>
                <td colSpan={7} className="py-12 text-center text-muted-foreground">
                  <div className="flex flex-col items-center justify-center gap-2">
                    <Loader2 className="h-6 w-6 animate-spin text-primary" />
                    <span>Loading execution history...</span>
                  </div>
                </td>
              </tr>
            ) : filteredExecutions.length === 0 ? (
              <tr>
                <td colSpan={7} className="py-12 text-center text-muted-foreground">
                  <div className="flex flex-col items-center justify-center gap-2">
                    <Clock className="h-8 w-8 text-muted-foreground/50 mb-1" />
                    <p className="font-semibold text-foreground text-sm">No Executions Found</p>
                    <p className="text-xs text-muted-foreground">
                      {searchQuery || statusFilter !== "ALL"
                        ? "No executions match the selected search or filter criteria."
                        : "Trigger a job from the catalog above to view execution history."}
                    </p>
                  </div>
                </td>
              </tr>
            ) : (
              filteredExecutions.map((exec) => (
                <tr
                  key={exec.id}
                  className="hover:bg-muted/30 transition-colors"
                >
                  {/* Execution ID */}
                  <td className="py-3 px-4 font-mono font-medium text-foreground">
                    <span title={exec.name}>{exec.id}</span>
                  </td>

                  {/* Job Name */}
                  <td className="py-3 px-4">
                    <span className="font-mono font-semibold text-primary bg-primary/10 px-2 py-0.5 rounded border border-primary/20">
                      {exec.jobName}
                    </span>
                  </td>

                  {/* Mode (Dry Run vs Live) */}
                  <td className="py-3 px-4">
                    {exec.dryRun ? (
                      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-medium bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20">
                        <ShieldCheck className="h-3 w-3" />
                        Dry Run
                      </span>
                    ) : (
                      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-medium bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/20">
                        <Zap className="h-3 w-3" />
                        Live
                      </span>
                    )}
                  </td>

                  {/* Status */}
                  <td className="py-3 px-4">
                    <div className="flex flex-col gap-1">
                      {renderStatusBadge(exec.status, exec.errorMessage)}
                      {exec.errorMessage && (
                        <span className="text-[11px] text-rose-500 font-mono truncate max-w-xs" title={exec.errorMessage}>
                          {exec.errorMessage}
                        </span>
                      )}
                    </div>
                  </td>

                  {/* Started At */}
                  <td className="py-3 px-4 text-muted-foreground whitespace-nowrap">
                    {formatTimestamp(exec.startTime || exec.createTime)}
                  </td>

                  {/* Duration */}
                  <td className="py-3 px-4 text-muted-foreground font-mono whitespace-nowrap">
                    {exec.durationFormatted || "—"}
                  </td>

                  {/* Log Link */}
                  <td className="py-3 px-4 text-right">
                    {exec.logUri ? (
                      <a
                        href={exec.logUri}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="inline-flex items-center gap-1 px-2.5 py-1 rounded-lg border border-input bg-background hover:bg-muted text-foreground text-xs font-medium transition-colors"
                      >
                        <span>Logs</span>
                        <ExternalLink className="h-3 w-3 text-muted-foreground" />
                      </a>
                    ) : (
                      <span className="text-muted-foreground/40 text-xs">—</span>
                    )}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}
