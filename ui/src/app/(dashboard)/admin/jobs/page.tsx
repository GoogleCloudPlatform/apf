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

import { useState, useEffect, useCallback, useRef } from "react"
import { useAuthStore } from "@/stores/auth"
import { useAppStore } from "@/stores/app"
import {
  Terminal,
  RefreshCw,
  AlertCircle,
  Loader2,
  CheckCircle2,
  AlertTriangle,
  PlayCircle,
  Activity,
  Layers,
} from "lucide-react"
import { UnauthorizedScreen } from "@/components/shared/UnauthorizedScreen"
import JobsCatalogCard from "@/components/admin/JobsCatalogCard"
import JobRunModal, { JobCatalogItem } from "@/components/admin/JobRunModal"
import JobExecutionsTable, { JobExecutionItem } from "@/components/admin/JobExecutionsTable"
import { toast } from "sonner"

export default function AdminJobsPage() {
  const { setPageTitle } = useAppStore()
  const { token, isSuperAdmin, isCheckingAuth } = useAuthStore()

  const [jobs, setJobs] = useState<JobCatalogItem[]>([])
  const [executions, setExecutions] = useState<JobExecutionItem[]>([])
  const [isLoadingJobs, setIsLoadingJobs] = useState(true)
  const [isLoadingExecutions, setIsLoadingExecutions] = useState(true)
  const [isRefreshing, setIsRefreshing] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Run Modal State
  const [selectedJob, setSelectedJob] = useState<JobCatalogItem | null>(null)
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [isTriggering, setIsTriggering] = useState(false)

  const pollingTimerRef = useRef<NodeJS.Timeout | null>(null)

  useEffect(() => {
    setPageTitle("Admin: System Jobs & Execution History")
  }, [setPageTitle])

  const fetchJobs = useCallback(async () => {
    setIsLoadingJobs(true)
    try {
      const res = await fetch("/api/v1/admin/jobs", {
        headers: {
          "Content-Type": "application/json",
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
      })
      if (!res.ok) {
        throw new Error(`Failed to load jobs catalog (status ${res.status})`)
      }
      const data = await res.json()
      setJobs(Array.isArray(data) ? data : [])
    } catch (err: any) {
      console.error(err)
      setError(err.message || "Failed to load jobs catalog")
    } finally {
      setIsLoadingJobs(false)
    }
  }, [token])

  const fetchExecutions = useCallback(
    async (showLoading = false) => {
      if (showLoading) setIsLoadingExecutions(true)
      try {
        const res = await fetch("/api/v1/admin/jobs/executions", {
          headers: {
            "Content-Type": "application/json",
            ...(token ? { Authorization: `Bearer ${token}` } : {}),
          },
        })
        if (!res.ok) {
          throw new Error(`Failed to load executions (status ${res.status})`)
        }
        const data = await res.json()
        setExecutions(data.executions || [])
      } catch (err: any) {
        console.error(err)
      } finally {
        if (showLoading) setIsLoadingExecutions(false)
      }
    },
    [token]
  )

  const handleRefresh = async () => {
    setIsRefreshing(true)
    await Promise.all([fetchJobs(), fetchExecutions(false)])
    setIsRefreshing(false)
    toast.success("Job catalog and execution history refreshed")
  }

  // Initial Load
  useEffect(() => {
    if (isSuperAdmin) {
      fetchJobs()
      fetchExecutions(true)
    }
  }, [isSuperAdmin, fetchJobs, fetchExecutions])

  // Active Execution Polling (every 5 seconds while jobs are running/pending)
  useEffect(() => {
    const hasActiveExecutions = executions.some(
      (e) => e.status.toUpperCase() === "RUNNING" || e.status.toUpperCase() === "PENDING"
    )

    if (hasActiveExecutions) {
      pollingTimerRef.current = setInterval(() => {
        fetchExecutions(false)
      }, 5000)
    } else {
      if (pollingTimerRef.current) {
        clearInterval(pollingTimerRef.current)
        pollingTimerRef.current = null
      }
    }

    return () => {
      if (pollingTimerRef.current) {
        clearInterval(pollingTimerRef.current)
        pollingTimerRef.current = null
      }
    }
  }, [executions, fetchExecutions])

  const handleOpenRunModal = (job: JobCatalogItem) => {
    setSelectedJob(job)
    setIsModalOpen(true)
  }

  const handleCloseRunModal = () => {
    if (isTriggering) return
    setIsModalOpen(false)
    setSelectedJob(null)
  }

  const handleExecuteJob = async (jobName: string, dryRun: boolean) => {
    setIsTriggering(true)
    try {
      const res = await fetch(`/api/v1/admin/jobs/${encodeURIComponent(jobName)}/run`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify({ dryRun }),
      })

      if (!res.ok) {
        const errData = await res.json().catch(() => ({}))
        throw new Error(errData.message || `Failed to trigger job (status ${res.status})`)
      }

      const data = await res.json()
      toast.success(
        `Job ${jobName} triggered successfully${dryRun ? " (Dry Run)" : ""}! Execution ID: ${
          data.executionName?.split("/").pop() || "in-flight"
        }`
      )
      setIsModalOpen(false)
      setSelectedJob(null)

      // Refresh executions to show the newly queued job immediately
      await fetchExecutions(false)
    } catch (err: any) {
      console.error(err)
      toast.error(err.message || "Failed to trigger job execution")
    } finally {
      setIsTriggering(false)
    }
  }

  // Access Control check
  if (isCheckingAuth) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[60vh] gap-3">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
        <p className="text-sm text-muted-foreground font-medium">Verifying super administrator permissions...</p>
      </div>
    )
  }

  if (!isSuperAdmin) {
    return (
      <UnauthorizedScreen
        title="Super Administrator Access Required"
        message="System maintenance jobs, direct execution triggers, and operational audit telemetry are restricted to Parent-Project Super Administrators."
        requiredPermission="super_admin"
      />
    )
  }

  // Summary Metrics
  const activeCount = executions.filter(
    (e) => e.status.toUpperCase() === "RUNNING" || e.status.toUpperCase() === "PENDING"
  ).length
  const succeededCount = executions.filter((e) => e.status.toUpperCase() === "SUCCEEDED").length
  const failedCount = executions.filter((e) => e.status.toUpperCase() === "FAILED").length

  return (
    <div className="w-full max-w-7xl space-y-8 p-6 text-foreground">
      {/* Top Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div className="space-y-1">
          <h1 className="text-2xl font-bold tracking-tight text-foreground flex items-center gap-2.5">
            <Terminal className="h-7 w-7 text-primary" />
            System Jobs & Maintenance
          </h1>
          <p className="text-sm text-muted-foreground">
            Trigger on-demand batch operations, inspect container catalog capabilities, and monitor Cloud Run execution history.
          </p>
        </div>

        <button
          onClick={handleRefresh}
          disabled={isRefreshing || isLoadingJobs || isLoadingExecutions}
          className="inline-flex items-center justify-center gap-2 px-4 py-2 rounded-xl border border-input bg-card hover:bg-muted text-xs font-semibold text-foreground shadow-sm transition-all disabled:opacity-50"
        >
          <RefreshCw className={`h-3.5 w-3.5 ${isRefreshing ? "animate-spin" : ""}`} />
          <span>Refresh Data</span>
        </button>
      </div>

      {/* Error Alert */}
      {error && (
        <div className="p-4 rounded-xl border border-destructive/30 bg-destructive/10 text-destructive flex items-start gap-3 text-xs">
          <AlertCircle className="h-5 w-5 shrink-0 mt-0.5" />
          <div className="space-y-1">
            <p className="font-semibold">Failed to load system jobs data</p>
            <p>{error}</p>
          </div>
        </div>
      )}

      {/* Summary KPI Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* Total Registered Jobs */}
        <div className="p-5 rounded-2xl border border-border bg-card shadow-sm space-y-2">
          <div className="flex items-center justify-between text-muted-foreground">
            <span className="text-xs font-semibold uppercase tracking-wider">Registered Jobs</span>
            <Layers className="h-4 w-4 text-primary" />
          </div>
          <div className="text-2xl font-bold text-foreground">
            {isLoadingJobs ? <Loader2 className="h-6 w-6 animate-spin text-primary" /> : jobs.length}
          </div>
          <p className="text-[11px] text-muted-foreground">Maintenance routines in binary</p>
        </div>

        {/* Active In-Flight Executions */}
        <div className="p-5 rounded-2xl border border-border bg-card shadow-sm space-y-2">
          <div className="flex items-center justify-between text-muted-foreground">
            <span className="text-xs font-semibold uppercase tracking-wider">Active Executions</span>
            <Activity className={`h-4 w-4 ${activeCount > 0 ? "text-blue-500 animate-pulse" : "text-muted-foreground"}`} />
          </div>
          <div className="text-2xl font-bold text-foreground flex items-center gap-2">
            {isLoadingExecutions ? (
              <Loader2 className="h-6 w-6 animate-spin text-primary" />
            ) : (
              <>
                {activeCount}
                {activeCount > 0 && (
                  <span className="text-xs font-semibold px-2 py-0.5 rounded-full bg-blue-500/10 text-blue-500 animate-pulse">
                    Running
                  </span>
                )}
              </>
            )}
          </div>
          <p className="text-[11px] text-muted-foreground">Cloud Run tasks currently processing</p>
        </div>

        {/* Succeeded Executions */}
        <div className="p-5 rounded-2xl border border-border bg-card shadow-sm space-y-2">
          <div className="flex items-center justify-between text-muted-foreground">
            <span className="text-xs font-semibold uppercase tracking-wider">Succeeded Runs</span>
            <CheckCircle2 className="h-4 w-4 text-emerald-500" />
          </div>
          <div className="text-2xl font-bold text-foreground">
            {isLoadingExecutions ? <Loader2 className="h-6 w-6 animate-spin text-primary" /> : succeededCount}
          </div>
          <p className="text-[11px] text-muted-foreground">Completed with zero errors</p>
        </div>

        {/* Failed Executions */}
        <div className="p-5 rounded-2xl border border-border bg-card shadow-sm space-y-2">
          <div className="flex items-center justify-between text-muted-foreground">
            <span className="text-xs font-semibold uppercase tracking-wider">Failed Runs</span>
            <AlertTriangle className="h-4 w-4 text-rose-500" />
          </div>
          <div className="text-2xl font-bold text-foreground">
            {isLoadingExecutions ? <Loader2 className="h-6 w-6 animate-spin text-primary" /> : failedCount}
          </div>
          <p className="text-[11px] text-muted-foreground">Executions requiring administrative attention</p>
        </div>
      </div>

      {/* Registered Jobs Catalog */}
      <JobsCatalogCard
        jobs={jobs}
        onSelectJob={handleOpenRunModal}
        isLoading={isLoadingJobs}
      />

      {/* Historical Executions & Live Monitor Table */}
      <JobExecutionsTable
        executions={executions}
        isLoading={isLoadingExecutions}
      />

      {/* Run Job Modal */}
      <JobRunModal
        isOpen={isModalOpen}
        job={selectedJob}
        onClose={handleCloseRunModal}
        onRun={handleExecuteJob}
        isRunning={isTriggering}
      />
    </div>
  )
}
