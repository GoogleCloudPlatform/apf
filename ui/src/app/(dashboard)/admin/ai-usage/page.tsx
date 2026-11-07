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

import { useState, useEffect, useCallback } from "react"
import { useAuthStore } from "@/stores/auth"
import { useAppStore } from "@/stores/app"
import { UnauthorizedScreen } from "@/components/shared/UnauthorizedScreen"
import {
  Cpu,
  Download,
  Search,
  Calendar,
  Users,
  FolderKanban,
  Activity,
  Loader2,
  AlertCircle,
  RefreshCw,
  Zap,
  DollarSign,
  Clock,
  Layers,
  ChevronLeft,
  ChevronRight
} from "lucide-react"
import { toast } from "sonner"

export interface UserBreakdownItem {
  userId: string
  promptTokens: number
  completionTokens: number
  totalTokens: number
  costUSD: number
  sessionCount: number
}

export interface ProjectBreakdownItem {
  projectId: string
  promptTokens: number
  completionTokens: number
  totalTokens: number
  costUSD: number
  sessionCount: number
}

export interface SessionBreakdownItem {
  sessionId: string
  userId?: string
  projectId?: string
  promptTokens: number
  completionTokens: number
  totalTokens: number
  costUSD: number
  turnCount?: number
}

export interface TelemetryUsageData {
  organizationId?: string
  projectId?: string
  userId?: string
  sessionId?: string
  startDate?: string
  endDate?: string
  promptTokens?: number
  completionTokens?: number
  totalTokens?: number
  costUSD?: number
  turnCount?: number
  averageTokensPerTurn?: number
  totalUsers?: number
  totalProjects?: number
  userBreakdown?: UserBreakdownItem[]
  projectBreakdown?: ProjectBreakdownItem[]
  sessionBreakdown?: SessionBreakdownItem[]
  limit?: number
  offset?: number
}

export type DatePreset = "today" | "7d" | "30d" | "custom"
export type BreakdownTab = "user" | "project" | "session"

function formatDateString(date: Date): string {
  return date.toISOString().split("T")[0]
}

function getDateWindow(preset: DatePreset, customStart?: string, customEnd?: string): { startDate: string; endDate: string } {
  const end = new Date()
  const endDate = formatDateString(end)

  if (preset === "today") {
    return { startDate: endDate, endDate }
  }
  if (preset === "7d") {
    const start = new Date()
    start.setDate(start.getDate() - 7)
    return { startDate: formatDateString(start), endDate }
  }
  if (preset === "30d") {
    const start = new Date()
    start.setDate(start.getDate() - 30)
    return { startDate: formatDateString(start), endDate }
  }
  if (preset === "custom") {
    let s = customStart || endDate
    let e = customEnd || endDate
    if (s > e) {
      const temp = s
      s = e
      e = temp
    }
    return {
      startDate: s,
      endDate: e
    }
  }

  // Default to 30 days
  const start = new Date()
  start.setDate(start.getDate() - 30)
  return { startDate: formatDateString(start), endDate }
}

export default function AIUsageAnalyticsPage() {
  const { setPageTitle } = useAppStore()
  const authStore = useAuthStore()
  const { token, isSuperAdmin, isAdmin, isCheckingAuth, selectedOrgId } = authStore

  // RBAC verification: Only Super Admin (isSuperAdmin) can access
  const isAuthorized = isSuperAdmin

  // Date range states
  const [datePreset, setDatePreset] = useState<DatePreset>("30d")
  const defaultDates = getDateWindow("30d")
  const [customStartDate, setCustomStartDate] = useState<string>(defaultDates.startDate)
  const [customEndDate, setCustomEndDate] = useState<string>(defaultDates.endDate)

  // Breakdown view & search states
  const [activeTab, setActiveTab] = useState<BreakdownTab>("user")
  const [searchQuery, setSearchQuery] = useState<string>("")
  const [currentPage, setCurrentPage] = useState<number>(1)
  const itemsPerPage = 50

  // Data fetching states
  const [telemetryData, setTelemetryData] = useState<TelemetryUsageData | null>(null)
  const [isLoading, setIsLoading] = useState<boolean>(true)
  const [error, setError] = useState<string | null>(null)
  const [isExporting, setIsExporting] = useState<boolean>(false)

  // Sync header page title
  useEffect(() => {
    setPageTitle("Admin: AI Usage & Cost")
  }, [setPageTitle])

  // Fetch telemetry function
  const fetchTelemetry = useCallback(async () => {
    if (!token || !isAuthorized) return

    setIsLoading(true)
    setError(null)

    try {
      const { startDate, endDate } = getDateWindow(datePreset, customStartDate, customEndDate)
      const queryParams = new URLSearchParams()

      if (startDate) queryParams.set("startDate", startDate)
      if (endDate) queryParams.set("endDate", endDate)
      if (selectedOrgId) queryParams.set("organizationId", selectedOrgId)
      queryParams.set("limit", "500")

      const response = await fetch(`/api/v1/telemetry/usage?${queryParams.toString()}`, {
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`
        }
      })

      if (!response.ok) {
        const errData = await response.json().catch(() => ({}))
        throw new Error(errData?.error?.message || errData?.message || `Failed to fetch usage analytics (HTTP ${response.status})`)
      }

      const resJson: TelemetryUsageData = await response.json()
      setTelemetryData(resJson)
    } catch (err: any) {
      console.error("Telemetry fetch error:", err)
      setError(err.message || "Failed to load AI token usage telemetry")
    } finally {
      setIsLoading(false)
    }
  }, [token, isAuthorized, datePreset, customStartDate, customEndDate, selectedOrgId])

  // Initial load and filter change trigger
  useEffect(() => {
    if (!isCheckingAuth && isAuthorized) {
      fetchTelemetry()
    }
  }, [isCheckingAuth, isAuthorized, fetchTelemetry])

  // Reset pagination when date filter, search query, or tab changes
  useEffect(() => {
    setCurrentPage(1)
  }, [datePreset, customStartDate, customEndDate, searchQuery, activeTab])

  // Handle CSV export streaming trigger
  const handleExportCSV = async () => {
    if (!token) return

    setIsExporting(true)
    try {
      const { startDate, endDate } = getDateWindow(datePreset, customStartDate, customEndDate)
      const queryParams = new URLSearchParams({ format: "csv" })

      if (startDate) queryParams.set("startDate", startDate)
      if (endDate) queryParams.set("endDate", endDate)
      if (selectedOrgId) queryParams.set("organizationId", selectedOrgId)

      const response = await fetch(`/api/v1/telemetry/usage?${queryParams.toString()}`, {
        headers: {
          Authorization: `Bearer ${token}`
        }
      })

      if (!response.ok) {
        throw new Error(`CSV export failed with status HTTP ${response.status}`)
      }

      const blob = await response.blob()
      const downloadUrl = window.URL.createObjectURL(blob)
      const a = document.createElement("a")
      a.href = downloadUrl
      a.download = `telemetry-usage-${startDate}-to-${endDate}.csv`
      document.body.appendChild(a)
      a.click()
      a.remove()
      window.URL.revokeObjectURL(downloadUrl)

      toast.success("CSV usage report downloaded successfully")
    } catch (err: any) {
      console.error("CSV Export error:", err)
      toast.error(err.message || "Failed to download CSV telemetry report")
    } finally {
      setIsExporting(false)
    }
  }

  // Render authentication loading state
  if (isCheckingAuth) {
    return (
      <div className="flex-1 flex h-[400px] w-full flex-col items-center justify-center gap-3 text-muted-foreground select-none">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
        <span className="text-sm font-medium">Verifying access...</span>
      </div>
    )
  }

  // Render RBAC UnauthorizedScreen for non-admins
  if (!isAuthorized) {
    return (
      <UnauthorizedScreen
        title="Admin Access Required"
        message="This screen is reserved exclusively for System Administrators and Organization Admins."
        requiredPermission="org:admin"
        fullPage={false}
      />
    )
  }

  // KPI card metric values with safe fallbacks
  const totalTokens = telemetryData?.totalTokens ?? 0
  const promptTokens = telemetryData?.promptTokens ?? 0
  const completionTokens = telemetryData?.completionTokens ?? 0
  const costUSD = telemetryData?.costUSD ?? 0
  const turnCount = telemetryData?.turnCount ?? 0
  const averageTokensPerTurn = telemetryData?.averageTokensPerTurn ?? 0
  const totalUsers = telemetryData?.totalUsers ?? (telemetryData?.userBreakdown?.length ?? 0)
  const totalProjects = telemetryData?.totalProjects ?? (telemetryData?.projectBreakdown?.length ?? 0)

  // Extract breakdowns
  const rawUserBreakdown = telemetryData?.userBreakdown ?? []
  const rawProjectBreakdown = telemetryData?.projectBreakdown ?? []
  const rawSessionBreakdown = telemetryData?.sessionBreakdown ?? []

  // Filter breakdown items based on tab & search query
  const query = searchQuery.trim().toLowerCase()

  const filteredUserBreakdown = rawUserBreakdown.filter(item =>
    !query || (item.userId ?? "").toLowerCase().includes(query)
  )

  const filteredProjectBreakdown = rawProjectBreakdown.filter(item =>
    !query || (item.projectId ?? "").toLowerCase().includes(query)
  )

  const filteredSessionBreakdown = rawSessionBreakdown.filter(item =>
    !query ||
    (item.sessionId ?? "").toLowerCase().includes(query) ||
    ((item.userId ?? "").toLowerCase().includes(query)) ||
    ((item.projectId ?? "").toLowerCase().includes(query))
  )

  // Determine current active list and pagination math
  let currentList: any[] = []
  if (activeTab === "user") currentList = filteredUserBreakdown
  else if (activeTab === "project") currentList = filteredProjectBreakdown
  else if (activeTab === "session") currentList = filteredSessionBreakdown

  const totalItems = currentList.length
  const totalPages = Math.max(1, Math.ceil(totalItems / itemsPerPage))
  const safePage = Math.min(currentPage, totalPages)
  const paginatedList = currentList.slice((safePage - 1) * itemsPerPage, safePage * itemsPerPage)

  return (
    <main data-testid="ai-usage-analytics-page" className="flex-1 overflow-y-auto p-8 text-foreground">
      <div className="w-full max-w-[1200px] space-y-8 pb-12">

        {/* Page Header Bar */}
        <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between border-b border-border pb-4">
          <div className="space-y-1">
            <div className="flex items-center gap-2.5">
              <Cpu className="h-6 w-6 text-primary" />
              <h2 className="text-2xl font-bold tracking-tight text-foreground">
                AI Token Usage & Cost Analytics
              </h2>
            </div>
            <p className="text-xs text-muted-foreground leading-relaxed">
              Operational metrics, model token consumption, turn counts, and cost distributions aggregated across users and projects.
            </p>
          </div>

          <div className="flex items-center gap-3">
            <button
              onClick={fetchTelemetry}
              disabled={isLoading}
              className="h-9 px-3 rounded-lg border border-input bg-background hover:bg-muted text-xs font-semibold text-foreground flex items-center gap-2 transition-all shadow-xs disabled:opacity-50"
              title="Refresh telemetry"
            >
              <RefreshCw className={`h-3.5 w-3.5 ${isLoading ? "animate-spin text-primary" : ""}`} />
              Refresh
            </button>

            <button
              data-testid="export-csv-btn"
              onClick={handleExportCSV}
              disabled={isExporting || isLoading}
              className="h-9 px-4 rounded-lg bg-primary hover:bg-primary/90 text-primary-foreground text-xs font-semibold flex items-center gap-2 transition-all shadow-sm disabled:opacity-50"
            >
              {isExporting ? (
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
              ) : (
                <Download className="h-3.5 w-3.5" />
              )}
              Export CSV
            </button>
          </div>
        </div>

        {/* Error Alert Banner */}
        {error && (
          <div className="p-4 rounded-2xl bg-destructive/10 border border-destructive/20 text-destructive text-xs flex items-center justify-between">
            <div className="flex items-center gap-3">
              <AlertCircle className="h-5 w-5 shrink-0" />
              <span className="font-medium">{error}</span>
            </div>
            <button
              onClick={fetchTelemetry}
              className="px-3 py-1 bg-destructive/20 hover:bg-destructive/30 rounded-lg text-xs font-semibold transition-colors"
            >
              Retry
            </button>
          </div>
        )}

        {/* Top KPI Metric Cards Grid */}
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">

          {/* KPI 1: Total Tokens */}
          <div data-testid="kpi-total-tokens" className="bg-card border border-border rounded-2xl p-5 space-y-2 shadow-sm relative overflow-hidden">
            <div className="flex items-center justify-between">
              <span className="text-xs font-bold text-muted-foreground uppercase tracking-wider">
                Total Tokens
              </span>
              <div className="p-2 rounded-xl bg-primary/10 text-primary">
                <Zap className="h-4 w-4" />
              </div>
            </div>
            <div className="space-y-1">
              <span className="text-2xl font-extrabold tracking-tight text-foreground block">
                {totalTokens.toLocaleString()}
              </span>
              <span className="text-[10px] text-muted-foreground block truncate">
                Input: {promptTokens.toLocaleString()} | Output: {completionTokens.toLocaleString()}
              </span>
            </div>
          </div>

          {/* KPI 2: Cost USD */}
          <div data-testid="kpi-total-cost" className="bg-card border border-border rounded-2xl p-5 space-y-2 shadow-sm relative overflow-hidden">
            <div className="flex items-center justify-between">
              <span className="text-xs font-bold text-muted-foreground uppercase tracking-wider">
                Cost (USD)
              </span>
              <div className="p-2 rounded-xl bg-emerald-500/10 text-emerald-500">
                <DollarSign className="h-4 w-4" />
              </div>
            </div>
            <div className="space-y-1">
              <span className="text-2xl font-extrabold tracking-tight text-emerald-600 dark:text-emerald-400 block">
                ${costUSD.toFixed(2)}
              </span>
              <span className="text-[10px] text-muted-foreground block">
                Calculated via model rate cards
              </span>
            </div>
          </div>

          {/* KPI 3: Active Turns */}
          <div data-testid="kpi-active-turns" className="bg-card border border-border rounded-2xl p-5 space-y-2 shadow-sm relative overflow-hidden">
            <div className="flex items-center justify-between">
              <span className="text-xs font-bold text-muted-foreground uppercase tracking-wider">
                Active Turns
              </span>
              <div className="p-2 rounded-xl bg-blue-500/10 text-blue-500">
                <Activity className="h-4 w-4" />
              </div>
            </div>
            <div className="space-y-1">
              <span className="text-2xl font-extrabold tracking-tight text-foreground block">
                {turnCount.toLocaleString()}
              </span>
              <span className="text-[10px] text-muted-foreground block">
                Avg {averageTokensPerTurn.toFixed(1)} tokens / turn
              </span>
            </div>
          </div>

          {/* KPI 4: Total Users & Projects */}
          <div data-testid="kpi-total-users-projects" className="bg-card border border-border rounded-2xl p-5 space-y-2 shadow-sm relative overflow-hidden">
            <div className="flex items-center justify-between">
              <span className="text-xs font-bold text-muted-foreground uppercase tracking-wider">
                Users & Projects
              </span>
              <div className="p-2 rounded-xl bg-purple-500/10 text-purple-500">
                <Users className="h-4 w-4" />
              </div>
            </div>
            <div className="space-y-1">
              <span className="text-2xl font-extrabold tracking-tight text-foreground block">
                {totalUsers} <span className="text-sm font-normal text-muted-foreground">Users</span> / {totalProjects} <span className="text-sm font-normal text-muted-foreground">Projects</span>
              </span>
              <span className="text-[10px] text-muted-foreground block">
                Active telemetry participants
              </span>
            </div>
          </div>

        </div>

        {/* Date Filter & Preset Controls */}
        <div className="bg-card border border-border rounded-2xl p-4 shadow-sm space-y-4">
          <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex items-center gap-2">
              <Calendar className="h-4 w-4 text-primary" />
              <span className="text-xs font-bold text-foreground">Date Range Window:</span>
            </div>

            {/* Preset Buttons */}
            <div className="flex flex-wrap items-center gap-2">
              <button
                data-testid="preset-today"
                onClick={() => setDatePreset("today")}
                className={`px-3 py-1.5 rounded-lg text-xs font-semibold transition-all ${
                  datePreset === "today"
                    ? "bg-primary text-primary-foreground shadow-xs"
                    : "bg-muted/40 hover:bg-muted text-muted-foreground hover:text-foreground"
                }`}
              >
                Today
              </button>
              <button
                data-testid="preset-7d"
                onClick={() => setDatePreset("7d")}
                className={`px-3 py-1.5 rounded-lg text-xs font-semibold transition-all ${
                  datePreset === "7d"
                    ? "bg-primary text-primary-foreground shadow-xs"
                    : "bg-muted/40 hover:bg-muted text-muted-foreground hover:text-foreground"
                }`}
              >
                7D
              </button>
              <button
                data-testid="preset-30d"
                onClick={() => setDatePreset("30d")}
                className={`px-3 py-1.5 rounded-lg text-xs font-semibold transition-all ${
                  datePreset === "30d"
                    ? "bg-primary text-primary-foreground shadow-xs"
                    : "bg-muted/40 hover:bg-muted text-muted-foreground hover:text-foreground"
                }`}
              >
                30D (Default)
              </button>
              <button
                data-testid="preset-custom"
                onClick={() => setDatePreset("custom")}
                className={`px-3 py-1.5 rounded-lg text-xs font-semibold transition-all ${
                  datePreset === "custom"
                    ? "bg-primary text-primary-foreground shadow-xs"
                    : "bg-muted/40 hover:bg-muted text-muted-foreground hover:text-foreground"
                }`}
              >
                Custom Range
              </button>
            </div>
          </div>

          {/* Custom Date Range Inputs */}
          {datePreset === "custom" && (
            <div className="flex flex-wrap items-center gap-4 pt-3 border-t border-border/60">
              <div className="flex items-center gap-2">
                <label htmlFor="custom-start-date" className="text-xs font-medium text-muted-foreground">Start Date:</label>
                <input
                  id="custom-start-date"
                  type="date"
                  value={customStartDate}
                  onChange={(e) => setCustomStartDate(e.target.value)}
                  className="px-3 py-1.5 text-xs rounded-lg border border-input bg-background text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
                />
              </div>

              <div className="flex items-center gap-2">
                <label htmlFor="custom-end-date" className="text-xs font-medium text-muted-foreground">End Date:</label>
                <input
                  id="custom-end-date"
                  type="date"
                  value={customEndDate}
                  onChange={(e) => setCustomEndDate(e.target.value)}
                  className="px-3 py-1.5 text-xs rounded-lg border border-input bg-background text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
                />
              </div>
            </div>
          )}
        </div>

        {/* Tabbed Breakdown Table Section */}
        <div className="bg-card border border-border rounded-2xl shadow-sm overflow-hidden space-y-0">

          {/* Tab Headers & Search Control */}
          <div className="flex flex-col gap-4 p-4 sm:flex-row sm:items-center sm:justify-between border-b border-border bg-slate-50/50 dark:bg-slate-900/30">

            {/* Breakdown Tabs */}
            <div className="flex items-center p-1 bg-muted/60 rounded-xl space-x-1">
              <button
                data-testid="tab-user"
                onClick={() => setActiveTab("user")}
                className={`px-4 py-2 rounded-lg text-xs font-bold transition-all flex items-center gap-2 ${
                  activeTab === "user"
                    ? "bg-background text-foreground shadow-xs"
                    : "text-muted-foreground hover:text-foreground"
                }`}
              >
                <Users className="h-3.5 w-3.5" />
                By User ({filteredUserBreakdown.length})
              </button>

              <button
                data-testid="tab-project"
                onClick={() => setActiveTab("project")}
                className={`px-4 py-2 rounded-lg text-xs font-bold transition-all flex items-center gap-2 ${
                  activeTab === "project"
                    ? "bg-background text-foreground shadow-xs"
                    : "text-muted-foreground hover:text-foreground"
                }`}
              >
                <FolderKanban className="h-3.5 w-3.5" />
                By Project ({filteredProjectBreakdown.length})
              </button>

              <button
                data-testid="tab-session"
                onClick={() => setActiveTab("session")}
                className={`px-4 py-2 rounded-lg text-xs font-bold transition-all flex items-center gap-2 ${
                  activeTab === "session"
                    ? "bg-background text-foreground shadow-xs"
                    : "text-muted-foreground hover:text-foreground"
                }`}
              >
                <Layers className="h-3.5 w-3.5" />
                By Session ({filteredSessionBreakdown.length})
              </button>
            </div>

            {/* Search Input */}
            <div className="relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground/60" />
              <input
                data-testid="breakdown-search-input"
                type="text"
                placeholder={`Search by ${activeTab}...`}
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-9 pr-3 py-1.5 w-full sm:w-[240px] text-xs rounded-lg border border-input bg-background focus:outline-none focus:ring-1 focus:ring-primary/40 focus:border-primary/40 transition-all text-foreground"
              />
            </div>
          </div>

          {/* Table Content */}
          {isLoading ? (
            <div className="flex justify-center items-center py-16 gap-2 text-muted-foreground">
              <Loader2 className="h-6 w-6 animate-spin text-primary" />
              <span className="text-xs font-medium">Loading telemetry breakdown data...</span>
            </div>
          ) : totalItems === 0 ? (
            <div className="p-16 text-center text-muted-foreground text-xs space-y-2 select-none">
              <p className="font-semibold text-foreground text-sm">No usage telemetry found</p>
              <p className="text-muted-foreground">
                There are no recorded token usage events matching the selected date window or search query.
              </p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full border-collapse text-left">
                <thead>
                  <tr className="border-b border-border bg-slate-50 dark:bg-slate-900/40 text-[10px] font-bold uppercase tracking-wider text-muted-foreground/80 select-none">
                    {activeTab === "user" && <th className="p-4">User ID / Email</th>}
                    {activeTab === "project" && <th className="p-4">Project ID</th>}
                    {activeTab === "session" && <th className="p-4">Session ID</th>}
                    {activeTab === "session" && <th className="p-4">User / Project</th>}
                    <th className="p-4 text-right">Input Tokens</th>
                    <th className="p-4 text-right">Output Tokens</th>
                    <th className="p-4 text-right">Total Tokens</th>
                    <th className="p-4 text-right">Cost (USD)</th>
                    {activeTab !== "session" && <th className="p-4 text-right">Sessions</th>}
                    {activeTab === "session" && <th className="p-4 text-right">Turns</th>}
                  </tr>
                </thead>
                <tbody className="divide-y divide-border/60 text-xs">
                  {activeTab === "user" && paginatedList.map((item: UserBreakdownItem, idx: number) => (
                    <tr key={item.userId || idx} className="hover:bg-muted/10 transition-colors">
                      <td className="p-4 font-semibold text-foreground font-mono text-xs">
                        {item.userId || "unknown"}
                      </td>
                      <td className="p-4 text-right font-mono text-muted-foreground">
                        {(item.promptTokens ?? 0).toLocaleString()}
                      </td>
                      <td className="p-4 text-right font-mono text-muted-foreground">
                        {(item.completionTokens ?? 0).toLocaleString()}
                      </td>
                      <td className="p-4 text-right font-mono font-bold text-foreground">
                        {(item.totalTokens ?? 0).toLocaleString()}
                      </td>
                      <td className="p-4 text-right font-mono font-bold text-emerald-600 dark:text-emerald-400">
                        ${(item.costUSD ?? 0).toFixed(2)}
                      </td>
                      <td className="p-4 text-right font-mono text-muted-foreground">
                        {item.sessionCount ?? 0}
                      </td>
                    </tr>
                  ))}

                  {activeTab === "project" && paginatedList.map((item: ProjectBreakdownItem, idx: number) => (
                    <tr key={item.projectId || idx} className="hover:bg-muted/10 transition-colors">
                      <td className="p-4 font-semibold text-foreground font-mono text-xs">
                        {item.projectId || "unknown"}
                      </td>
                      <td className="p-4 text-right font-mono text-muted-foreground">
                        {(item.promptTokens ?? 0).toLocaleString()}
                      </td>
                      <td className="p-4 text-right font-mono text-muted-foreground">
                        {(item.completionTokens ?? 0).toLocaleString()}
                      </td>
                      <td className="p-4 text-right font-mono font-bold text-foreground">
                        {(item.totalTokens ?? 0).toLocaleString()}
                      </td>
                      <td className="p-4 text-right font-mono font-bold text-emerald-600 dark:text-emerald-400">
                        ${(item.costUSD ?? 0).toFixed(2)}
                      </td>
                      <td className="p-4 text-right font-mono text-muted-foreground">
                        {item.sessionCount ?? 0}
                      </td>
                    </tr>
                  ))}

                  {activeTab === "session" && paginatedList.map((item: SessionBreakdownItem, idx: number) => (
                    <tr key={item.sessionId || idx} className="hover:bg-muted/10 transition-colors">
                      <td className="p-4 font-semibold text-foreground font-mono text-xs">
                        {item.sessionId || "unknown"}
                      </td>
                      <td className="p-4 font-mono text-[11px] text-muted-foreground">
                        {item.userId || item.projectId || "—"}
                      </td>
                      <td className="p-4 text-right font-mono text-muted-foreground">
                        {(item.promptTokens ?? 0).toLocaleString()}
                      </td>
                      <td className="p-4 text-right font-mono text-muted-foreground">
                        {(item.completionTokens ?? 0).toLocaleString()}
                      </td>
                      <td className="p-4 text-right font-mono font-bold text-foreground">
                        {(item.totalTokens ?? 0).toLocaleString()}
                      </td>
                      <td className="p-4 text-right font-mono font-bold text-emerald-600 dark:text-emerald-400">
                        ${(item.costUSD ?? 0).toFixed(2)}
                      </td>
                      <td className="p-4 text-right font-mono text-muted-foreground">
                        {item.turnCount ?? 1}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          {/* Pagination Controls */}
          {totalItems > 0 && (
            <div className="flex items-center justify-between p-4 border-t border-border bg-slate-50/50 dark:bg-slate-900/30">
              <span className="text-xs text-muted-foreground">
                Showing {Math.min((currentPage - 1) * itemsPerPage + 1, totalItems)} to {Math.min(currentPage * itemsPerPage, totalItems)} of {totalItems} entries
              </span>

              <div className="flex items-center gap-2">
                <button
                  onClick={() => setCurrentPage(prev => Math.max(prev - 1, 1))}
                  disabled={currentPage === 1}
                  className="h-8 px-3 rounded-lg border border-input bg-background hover:bg-muted text-xs font-semibold text-foreground flex items-center gap-1 transition-all disabled:opacity-40"
                >
                  <ChevronLeft className="h-3.5 w-3.5" />
                  Previous
                </button>

                <span className="text-xs font-medium text-foreground px-2">
                  Page {currentPage} of {totalPages}
                </span>

                <button
                  onClick={() => setCurrentPage(prev => Math.min(prev + 1, totalPages))}
                  disabled={currentPage >= totalPages}
                  className="h-8 px-3 rounded-lg border border-input bg-background hover:bg-muted text-xs font-semibold text-foreground flex items-center gap-1 transition-all disabled:opacity-40"
                >
                  Next
                  <ChevronRight className="h-3.5 w-3.5" />
                </button>
              </div>
            </div>
          )}

        </div>

      </div>
    </main>
  )
}
