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
import {
  FolderX,
  Trash2,
  RefreshCw,
  Search,
  AlertTriangle,
  CheckCircle,
  HardDrive,
  Info,
  X,
  Loader2,
  FolderMinus,
} from "lucide-react"
import { formatBytes, ActiveJobState } from "./NFSVolumeStatsCard"
import { toast } from "sonner"

export interface HangingProjectItem {
  indexKey: string
  orgId: string
  userId: string
  projectId: string
  path: string
  sizeBytes: number
  fileCount: number
}

interface HangingProjectsSectionProps {
  token: string | null
  onProjectsDeleted?: () => void
  onRunCleanupJob?: (dryRun: boolean) => Promise<void>
  activeJob?: ActiveJobState | null
}

export default function HangingProjectsSection({
  token,
  onProjectsDeleted,
  onRunCleanupJob,
  activeJob,
}: HangingProjectsSectionProps) {
  const [hangingProjects, setHangingProjects] = useState<HangingProjectItem[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [isDeleting, setIsDeleting] = useState(false)
  const [isRunningCleanupJob, setIsRunningCleanupJob] = useState(false)
  const [isCleanupDryRun, setIsCleanupDryRun] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [warningMsg, setWarningMsg] = useState<string | null>(null)
  const [successMsg, setSuccessMsg] = useState<string | null>(null)

  const [searchQuery, setSearchQuery] = useState("")
  const [selectedKeys, setSelectedKeys] = useState<Set<string>>(new Set())

  // Confirmation Modal state
  const [isConfirmOpen, setIsConfirmOpen] = useState(false)
  const [projectsToDelete, setProjectsToDelete] = useState<HangingProjectItem[]>([])

  const isCleanupJobActive =
    isRunningCleanupJob ||
    (activeJob?.jobName === "hanging-cleanup" && activeJob.status === "PENDING")

  const fetchHangingProjects = async (clearMessages = true) => {
    setIsLoading(true)
    if (clearMessages) {
      setError(null)
      setWarningMsg(null)
      setSuccessMsg(null)
    }
    try {
      const res = await fetch("/api/v1/admin/nfs/hanging-projects", {
        headers: {
          "Content-Type": "application/json",
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
      })
      if (!res.ok) throw new Error("Failed to discover hanging projects")
      const data = await res.json()
      setHangingProjects(data.hangingProjects || [])
      setSelectedKeys(new Set())
    } catch (err: any) {
      console.error(err)
      setError(err.message || "Failed to scan for hanging projects")
    } finally {
      setIsLoading(false)
    }
  }

  useEffect(() => {
    if (token) {
      fetchHangingProjects()
    }
  }, [token])

  const filteredProjects = hangingProjects.filter((p) => {
    const query = searchQuery.toLowerCase()
    return (
      p.indexKey.toLowerCase().includes(query) ||
      p.orgId.toLowerCase().includes(query) ||
      p.userId.toLowerCase().includes(query) ||
      p.projectId.toLowerCase().includes(query) ||
      p.path.toLowerCase().includes(query)
    )
  })

  const toggleSelectAll = () => {
    if (selectedKeys.size === filteredProjects.length && filteredProjects.length > 0) {
      setSelectedKeys(new Set())
    } else {
      setSelectedKeys(new Set(filteredProjects.map((p) => p.indexKey)))
    }
  }

  const toggleSelect = (indexKey: string) => {
    const next = new Set(selectedKeys)
    if (next.has(indexKey)) {
      next.delete(indexKey)
    } else {
      next.add(indexKey)
    }
    setSelectedKeys(next)
  }

  const handleOpenConfirmSingle = (project: HangingProjectItem) => {
    setProjectsToDelete([project])
    setIsConfirmOpen(true)
  }

  const handleOpenConfirmSelected = () => {
    const toDelete = hangingProjects.filter((p) => selectedKeys.has(p.indexKey))
    if (toDelete.length === 0) return
    setProjectsToDelete(toDelete)
    setIsConfirmOpen(true)
  }

  const handleOpenConfirmAll = () => {
    if (hangingProjects.length === 0) return
    setProjectsToDelete(hangingProjects)
    setIsConfirmOpen(true)
  }

  const handleTriggerCleanupJob = async () => {
    setIsRunningCleanupJob(true)
    setError(null)
    setWarningMsg(null)
    setSuccessMsg(null)
    try {
      if (onRunCleanupJob) {
        await onRunCleanupJob(isCleanupDryRun)
      } else {
        const res = await fetch("/api/v1/admin/jobs/hanging-cleanup/run", {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            ...(token ? { Authorization: `Bearer ${token}` } : {}),
          },
          body: JSON.stringify({ dryRun: isCleanupDryRun }),
        })
        if (!res.ok) {
          const errData = await res.json().catch(() => ({}))
          throw new Error(errData.message || "Failed to trigger cleanup job execution")
        }
        const data = await res.json()
        const msg = `Cleanup job execution initiated (${data.executionName || "PENDING"})`
        setSuccessMsg(msg)
        toast.success(msg)
      }
      await fetchHangingProjects(false)
      if (onProjectsDeleted) {
        onProjectsDeleted()
      }
    } catch (err: any) {
      console.error(err)
      setError(err.message || "Failed to trigger cleanup job")
      toast.error(err.message || "Failed to trigger cleanup job")
    } finally {
      setIsRunningCleanupJob(false)
    }
  }

  const executeDeletion = async () => {
    setIsDeleting(true)
    setError(null)
    setWarningMsg(null)
    setSuccessMsg(null)
    try {
      const keys = projectsToDelete.map((p) => p.indexKey)
      const paths = projectsToDelete.map((p) => p.path)
      const res = await fetch("/api/v1/admin/nfs/hanging-projects/delete", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify({ indexKeys: keys, paths: paths }),
      })
      if (!res.ok) throw new Error("Deletion of hanging project(s) failed")
      const data = await res.json()

      const deletedCount = typeof data.deletedCount === "number" ? data.deletedCount : 0
      const hasErrors = Array.isArray(data.errors) && data.errors.length > 0

      if (deletedCount > 0) {
        const msg = `Successfully deleted ${deletedCount} hanging project folder(s) (${formatBytes(
          data.freedBytes || 0
        )} freed).`
        setSuccessMsg(msg)
        toast.success(msg)
      }

      if (hasErrors) {
        const errorDetails = data.errors.join("; ")
        if (deletedCount > 0) {
          setWarningMsg(
            `Warning: The system was unable to delete some of the requested folders (${data.errors.length} failed): ${errorDetails}`
          )
        } else {
          setWarningMsg(
            `Warning: The system was unable to delete the requested folders (${data.errors.length} failed): ${errorDetails}`
          )
        }
      } else if (deletedCount === 0) {
        setWarningMsg("No hanging project folders were deleted.")
      }

      setIsConfirmOpen(false)
      await fetchHangingProjects(false)
      if (onProjectsDeleted) {
        onProjectsDeleted()
      }
    } catch (err: any) {
      console.error(err)
      setError(err.message || "Failed to delete hanging projects")
      toast.error(err.message || "Failed to delete hanging projects")
    } finally {
      setIsDeleting(false)
    }
  }

  const totalReclaimable = hangingProjects.reduce((acc, p) => acc + (p.sizeBytes || 0), 0)
  const selectedReclaimable = projectsToDelete.reduce((acc, p) => acc + (p.sizeBytes || 0), 0)

  return (
    <div className="bg-card border border-border rounded-2xl overflow-hidden shadow-sm space-y-0">
      {/* Header Bar */}
      <div className="p-6 border-b border-border flex flex-col md:flex-row md:items-center justify-between gap-4 bg-muted/20">
        <div className="flex items-center gap-3">
          <div className="h-10 w-10 rounded-xl bg-amber-500/10 flex items-center justify-center text-amber-500 shrink-0 border border-amber-500/20">
            <FolderMinus className="h-5 w-5" />
          </div>
          <div>
            <h2 className="text-base font-bold text-foreground flex flex-wrap items-center gap-2">
              Hanging Projects Discovery & Cleanup
              <span className="text-[10px] font-extrabold px-2 py-0.5 rounded-full bg-amber-500/10 text-amber-500 border border-amber-500/20">
                {hangingProjects.length} Orphaned
              </span>
              {isCleanupJobActive && (
                <span
                  data-testid="cleanup-running-badge"
                  className="inline-flex items-center gap-1 text-[10px] font-bold px-2 py-0.5 rounded-full bg-primary/10 text-primary border border-primary/20 animate-pulse"
                >
                  <Loader2 className="h-3 w-3 animate-spin" /> CLEANUP IN PROGRESS
                </span>
              )}
            </h2>
            <p className="text-xs text-muted-foreground mt-0.5">
              Identifies disk folders (orgId-userId-projectId) absent from DB index and purges them safely.
            </p>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <label className="flex items-center gap-1.5 text-xs text-muted-foreground cursor-pointer select-none mr-1">
            <input
              type="checkbox"
              checked={isCleanupDryRun}
              onChange={(e) => setIsCleanupDryRun(e.target.checked)}
              className="rounded border-input text-primary focus:ring-primary/20"
            />
            <span className="font-mono text-[11px]">Dry Run</span>
          </label>

          <button
            onClick={handleTriggerCleanupJob}
            disabled={isLoading || isDeleting || isCleanupJobActive}
            aria-label="Run Cleanup"
            className="flex h-9 items-center justify-center gap-1.5 rounded-xl border border-primary/20 bg-primary/10 hover:bg-primary/20 text-primary text-xs font-semibold px-3 transition-all disabled:opacity-50"
          >
            {isCleanupJobActive ? (
              <>
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
                <span>RUNNING CLEANUP...</span>
              </>
            ) : (
              <>
                <RefreshCw className="h-3.5 w-3.5" />
                <span>Run Cleanup</span>
              </>
            )}
          </button>

          {hangingProjects.length > 0 && (
            <button
              onClick={handleOpenConfirmAll}
              disabled={isLoading || isDeleting || isCleanupJobActive}
              className="flex h-9 items-center justify-center gap-1.5 rounded-xl bg-destructive/10 hover:bg-destructive/20 border border-destructive/20 text-destructive text-xs font-semibold px-3 transition-all disabled:opacity-50"
            >
              <Trash2 className="h-3.5 w-3.5" /> Purge All ({hangingProjects.length})
            </button>
          )}

          <button
            onClick={() => fetchHangingProjects()}
            disabled={isLoading || isDeleting || isCleanupJobActive}
            className="flex h-9 items-center justify-center gap-1.5 rounded-xl border border-input bg-background hover:bg-muted text-xs font-semibold px-3 text-muted-foreground transition-all disabled:opacity-50"
          >
            <RefreshCw className={`h-3.5 w-3.5 ${isLoading ? "animate-spin" : ""}`} />
            DISCOVER HANGING FOLDERS
          </button>
        </div>
      </div>

      {/* Messages */}
      {(error || warningMsg || successMsg) && (
        <div className="p-4 border-b border-border space-y-3 bg-muted/5">
          {error && (
            <div
              role="alert"
              aria-label="Error message"
              className="p-4 rounded-xl bg-destructive/10 border border-destructive/20 text-destructive text-xs flex items-center gap-3"
            >
              <AlertTriangle className="h-4 w-4 shrink-0" />
              <span>{error}</span>
            </div>
          )}

          {warningMsg && (
            <div
              role="alert"
              aria-label="Warning message"
              className="p-4 rounded-xl bg-amber-500/10 border border-amber-500/20 text-amber-500 text-xs flex items-center gap-3"
            >
              <AlertTriangle className="h-4 w-4 shrink-0" />
              <span>{warningMsg}</span>
            </div>
          )}

          {successMsg && (
            <div
              role="status"
              aria-label="Success message"
              className="p-4 rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-emerald-500 text-xs flex items-center gap-3"
            >
              <CheckCircle className="h-4 w-4 shrink-0" />
              <span>{successMsg}</span>
            </div>
          )}
        </div>
      )}

      {/* Filter / Bulk Action Bar */}
      <div className="p-4 border-b border-border flex flex-col md:flex-row md:items-center justify-between gap-4 bg-background">
        <div className="relative flex-1 max-w-md">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
          <input
            type="text"
            placeholder="Filter by org, user, project ID, or index key..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full h-9 pl-9 pr-3 rounded-lg border border-input bg-background text-xs focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary text-foreground"
          />
        </div>

        <div className="flex items-center gap-3">
          {selectedKeys.size > 0 && (
            <button
              onClick={handleOpenConfirmSelected}
              className="flex h-9 items-center justify-center gap-1.5 rounded-lg bg-destructive text-destructive-foreground text-xs font-semibold px-3 hover:bg-destructive/90 transition-all shadow-sm"
            >
              <Trash2 className="h-3.5 w-3.5" /> Delete Selected ({selectedKeys.size})
            </button>
          )}

          <div className="text-xs text-muted-foreground font-medium">
            Total Reclaimable: <span className="font-bold text-foreground">{formatBytes(totalReclaimable)}</span>
          </div>
        </div>
      </div>

      {/* Table Content */}
      {isLoading ? (
        <div className="flex flex-col items-center justify-center py-16 gap-3 text-muted-foreground">
          <Loader2 className="h-7 w-7 text-amber-500 animate-spin" />
          <p className="text-xs font-medium">Scanning workspace filesystem and building index diff...</p>
        </div>
      ) : filteredProjects.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-14 text-center px-6">
          <CheckCircle className="h-8 w-8 text-emerald-500 mb-2" />
          <h3 className="font-bold text-foreground text-xs">No hanging project folders found</h3>
          <p className="text-xs text-muted-foreground mt-1 max-w-md">
            {searchQuery
              ? "No hanging project folders matched your search criteria."
              : "All workspace project folders on disk match active database records."}
          </p>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="border-b border-border text-[10px] font-bold text-muted-foreground uppercase bg-muted/30">
                <th className="py-3 px-4 w-10 text-center">
                  <input
                    type="checkbox"
                    checked={
                      selectedKeys.size === filteredProjects.length && filteredProjects.length > 0
                    }
                    onChange={toggleSelectAll}
                    className="rounded border-input text-primary focus:ring-primary/20"
                  />
                </th>
                <th className="py-3 px-4">Index Key (org-user-project)</th>
                <th className="py-3 px-4">Disk Path</th>
                <th className="py-3 px-4">Size</th>
                <th className="py-3 px-4">Files</th>
                <th className="py-3 px-4 text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border/60 text-xs">
              {filteredProjects.map((proj) => {
                const isSelected = selectedKeys.has(proj.indexKey)
                return (
                  <tr
                    key={proj.indexKey}
                    className={`hover:bg-muted/20 transition-colors ${
                      isSelected ? "bg-primary/5" : ""
                    }`}
                  >
                    <td className="py-3 px-4 text-center">
                      <input
                        type="checkbox"
                        checked={isSelected}
                        onChange={() => toggleSelect(proj.indexKey)}
                        className="rounded border-input text-primary focus:ring-primary/20"
                      />
                    </td>

                    <td className="py-3 px-4 font-mono font-bold text-foreground">
                      {proj.indexKey}
                    </td>

                    <td className="py-3 px-4">
                      <div
                        className="font-mono text-[11px] text-muted-foreground truncate max-w-xs"
                        title={proj.path}
                      >
                        {proj.path}
                      </div>
                    </td>

                    <td className="py-3 px-4 font-semibold text-foreground">
                      {formatBytes(proj.sizeBytes || 0)}
                    </td>

                    <td className="py-3 px-4 text-muted-foreground">
                      {proj.fileCount || 0}
                    </td>

                    <td className="py-3 px-4 text-right">
                      <button
                        onClick={() => handleOpenConfirmSingle(proj)}
                        className="h-7 px-2.5 rounded-md border border-destructive/20 bg-destructive/10 text-destructive hover:bg-destructive/20 text-[11px] font-semibold transition-all inline-flex items-center gap-1"
                      >
                        <Trash2 className="h-3 w-3" /> Delete
                      </button>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}

      {/* Confirmation Modal */}
      {isConfirmOpen && (
        <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4">
          <div className="bg-card border border-border rounded-2xl w-full max-w-lg shadow-xl overflow-hidden flex flex-col max-h-[85vh]">
            {/* Modal Header */}
            <div className="p-5 border-b border-border flex items-center justify-between bg-muted/30">
              <div className="flex items-center gap-2 text-destructive font-bold text-sm">
                <AlertTriangle className="h-4 w-4" /> Confirm Hanging Project Deletion
              </div>
              <button
                onClick={() => setIsConfirmOpen(false)}
                className="text-muted-foreground hover:text-foreground p-1 rounded-lg"
              >
                <X className="h-4 w-4" />
              </button>
            </div>

            {/* Modal Body */}
            <div className="p-6 space-y-4 overflow-y-auto flex-1 text-xs">
              <p className="text-muted-foreground leading-relaxed">
                You are about to permanently delete{" "}
                <strong className="text-foreground">{projectsToDelete.length}</strong> hanging project
                folder(s) from disk.
              </p>

              <div className="p-3.5 rounded-xl bg-destructive/10 border border-destructive/20 text-destructive space-y-1.5">
                <div className="font-bold flex items-center gap-1.5">
                  <Info className="h-4 w-4" /> Safety Guarantee
                </div>
                <p className="text-[11px] leading-relaxed text-destructive/90">
                  Only the specific project folder(s) will be deleted. Organization and User parent
                  directories (e.g. <span className="font-mono">{projectsToDelete[0]?.orgId}</span> or{" "}
                  <span className="font-mono">{projectsToDelete[0]?.orgId}/{projectsToDelete[0]?.userId}</span>) will <strong>NEVER</strong> be deleted.
                </p>
              </div>

              <div className="space-y-2">
                <div className="font-bold text-foreground text-[11px] uppercase tracking-wider">
                  Target Folders To Delete ({projectsToDelete.length}):
                </div>
                <div className="max-h-40 overflow-y-auto border border-border rounded-xl divide-y divide-border/60 font-mono text-[11px] bg-muted/20">
                  {projectsToDelete.map((p) => (
                    <div key={p.indexKey} className="p-2.5 flex items-center justify-between">
                      <span className="text-foreground truncate max-w-xs" title={p.path}>
                        {p.path}
                      </span>
                      <span className="text-muted-foreground font-semibold shrink-0 ml-2">
                        {formatBytes(p.sizeBytes || 0)}
                      </span>
                    </div>
                  ))}
                </div>
              </div>

              <div className="p-3 rounded-xl bg-muted/40 border border-border flex items-center justify-between text-xs font-semibold">
                <span className="text-muted-foreground">Total Reclaimable Space:</span>
                <span className="text-foreground font-bold">{formatBytes(selectedReclaimable)}</span>
              </div>
            </div>

            {/* Modal Footer */}
            <div className="p-5 border-t border-border flex items-center justify-end gap-3 bg-muted/20">
              <button
                onClick={() => setIsConfirmOpen(false)}
                disabled={isDeleting}
                className="h-9 px-4 rounded-xl border border-input bg-background hover:bg-muted text-xs font-semibold text-muted-foreground transition-all disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                onClick={executeDeletion}
                disabled={isDeleting}
                className="h-9 px-4 rounded-xl bg-destructive text-destructive-foreground hover:bg-destructive/90 text-xs font-semibold flex items-center justify-center gap-2 transition-all shadow-sm disabled:opacity-50"
              >
                {isDeleting ? (
                  <>
                    <Loader2 className="h-3.5 w-3.5 animate-spin" />
                    Deleting...
                  </>
                ) : (
                  <>
                    <Trash2 className="h-3.5 w-3.5" />
                    Confirm & Delete ({projectsToDelete.length})
                  </>
                )}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
