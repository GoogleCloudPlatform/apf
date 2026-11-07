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

import { useState, useEffect } from "react"
import { useAuthStore } from "@/stores/auth"
import { useAppStore } from "@/stores/app"
import { HardDrive, ShieldAlert, Loader2, Sparkles, RefreshCw, AlertCircle, PlayCircle } from "lucide-react"
import NFSVolumeStatsCard, { NFSVolumeStats, ActiveJobState } from "@/components/admin/NFSVolumeStatsCard"
import FolderMetadataTable from "@/components/admin/FolderMetadataTable"
import EditQuotaModal, { NFSFolderMetadata } from "@/components/admin/EditQuotaModal"
import HangingProjectsSection from "@/components/admin/HangingProjectsSection"
import { toast } from "sonner"

export default function NFSAdminPage() {
  const { setPageTitle } = useAppStore()
  const { token, isSuperAdmin, isCheckingAuth } = useAuthStore()

  const [volume, setVolume] = useState<NFSVolumeStats | undefined>()
  const [folders, setFolders] = useState<NFSFolderMetadata[]>([])
  const [organizations, setOrganizations] = useState<{ id: string; name: string }[]>([])
  const [totalFolders, setTotalFolders] = useState<number>(0)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [isAuditing, setIsAuditing] = useState(false)
  const [activeJob, setActiveJob] = useState<ActiveJobState | null>(null)

  // Modal state
  const [selectedFolder, setSelectedFolder] = useState<NFSFolderMetadata | null>(null)
  const [isModalOpen, setIsModalOpen] = useState(false)

  useEffect(() => {
    setPageTitle("Admin: NFS Volume & Folder Metadata")
  }, [setPageTitle])

  const fetchData = async () => {
    setIsLoading(true)
    setError(null)
    try {
      // 1. Fetch volume status
      const volRes = await fetch("/api/v1/admin/nfs/volumes", {
        headers: {
          "Content-Type": "application/json",
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
      })
      if (!volRes.ok) throw new Error("Failed to load NFS volume stats")
      const volData = await volRes.json()
      setVolume(volData.volume)

      // 2. Fetch folder metadata
      const foldersRes = await fetch("/api/v1/admin/nfs/folders", {
        headers: {
          "Content-Type": "application/json",
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
      })
      if (!foldersRes.ok) throw new Error("Failed to load NFS folder metadata")
      const foldersData = await foldersRes.json()
      setFolders(foldersData.folders || [])
      setTotalFolders(foldersData.total || 0)

      // 3. Fetch organizations for name resolution
      try {
        const orgsRes = await fetch("/api/v1/admin/organizations", {
          headers: {
            "Content-Type": "application/json",
            ...(token ? { Authorization: `Bearer ${token}` } : {}),
          },
        })
        if (orgsRes.ok) {
          const orgsData = await orgsRes.json()
          setOrganizations(orgsData || [])
        }
      } catch (err) {
        console.warn("Failed to load organizations for name lookup:", err)
      }
    } catch (err: any) {
      console.error(err)
      setError(err.message || "An unexpected error occurred while fetching NFS metadata")
    } finally {
      setIsLoading(false)
    }
  }

  useEffect(() => {
    if (token && isSuperAdmin) {
      fetchData()
    }
  }, [token, isSuperAdmin])

  if (isCheckingAuth) {
    return (
      <div className="flex-1 flex h-[400px] w-full flex-col items-center justify-center gap-3 text-muted-foreground select-none">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
        <span className="text-sm font-medium">Verifying super-admin access...</span>
      </div>
    )
  }

  if (!isSuperAdmin) {
    return (
      <div className="flex-1 flex flex-col items-center justify-center p-8 min-h-[calc(100vh-4rem)] select-none">
        <div className="w-full max-w-md bg-destructive/10 border border-destructive/20 rounded-2xl p-8 text-center space-y-6">
          <div className="flex justify-center text-destructive">
            <ShieldAlert className="h-16 w-16 animate-bounce" />
          </div>
          <div className="space-y-2">
            <h3 className="text-xl font-bold text-foreground">Access Restricted</h3>
            <p className="text-sm text-muted-foreground leading-relaxed">
              This screen is reserved exclusively for Parent Project System Administrators.
            </p>
          </div>
        </div>
      </div>
    )
  }

  const handleTriggerJob = async (jobName: string, dryRun: boolean = false) => {
    if (jobName === "nfs-audit") {
      setIsAuditing(true)
    }
    setError(null)
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
        throw new Error(errData.message || `Job execution trigger failed for ${jobName}`)
      }
      const data = await res.json()
      const jobState: ActiveJobState = {
        jobName: data.jobName || jobName,
        status: data.status || "PENDING",
        executionName: data.executionName,
      }
      setActiveJob(jobState)
      toast.success(`Job execution initiated: ${jobName} (${data.executionName || "PENDING"})`)
      setTimeout(() => {
        setActiveJob((prev) => (prev?.executionName === jobState.executionName ? null : prev))
      }, 10000)
      await fetchData()
    } catch (err: any) {
      console.error(err)
      setError(err.message || `Failed to trigger job execution: ${jobName}`)
      toast.error(err.message || `Failed to trigger job execution: ${jobName}`)
    } finally {
      if (jobName === "nfs-audit") {
        setIsAuditing(false)
      }
    }
  }

  const handleTriggerAudit = async (dryRun: boolean = false) => {
    await handleTriggerJob("nfs-audit", dryRun)
  }

  const handleOpenEditModal = (folder: NFSFolderMetadata) => {
    setSelectedFolder(folder)
    setIsModalOpen(true)
  }

  const handleSaveMetadata = async (
    folderId: string,
    updates: {
      quotaSizeBytes?: number
      isReadOnly?: boolean
      retentionPolicy?: string
    }
  ) => {
    const res = await fetch(`/api/v1/admin/nfs/folders/${folderId}`, {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
      body: JSON.stringify(updates),
    })

    if (!res.ok) throw new Error("Failed to update folder metadata")

    const updatedFolder = await res.json()
    setFolders((prev) =>
      prev.map((f) => (f.folderId === folderId ? { ...f, ...updatedFolder } : f))
    )
  }

  return (
    <main data-testid="nfs-admin-page" className="flex-1 overflow-y-auto p-8">
      <div className="w-full max-w-[1100px] space-y-8 pb-12">
        
        {/* Header */}
        <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4 border-b border-border pb-6">
          <div className="flex items-center gap-3">
            <div className="h-10 w-10 rounded-xl bg-primary/10 flex items-center justify-center text-primary">
              <HardDrive className="h-5 w-5" />
            </div>
            <div>
              <h1 className="text-xl font-bold tracking-tight text-foreground flex items-center gap-2">
                NFS Storage Governance <Sparkles className="h-4 w-4 text-primary animate-pulse" />
              </h1>
              <p className="text-xs text-muted-foreground">
                Manage volume capacity, tenant folder quotas, read-only locks, and on-demand housekeeping jobs.
              </p>
            </div>
          </div>

          <div className="flex items-center gap-3">
            {activeJob && (
              <div
                data-testid="active-job-indicator"
                className="flex items-center gap-2 px-3 py-1.5 rounded-xl bg-primary/10 border border-primary/20 text-primary text-xs font-semibold"
              >
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
                <span>Job Active: {activeJob.jobName} ({activeJob.status})</span>
              </div>
            )}

            <button
              onClick={fetchData}
              disabled={isLoading}
              className="flex h-10 items-center justify-center gap-2 rounded-xl border border-input bg-background hover:bg-muted text-xs font-semibold px-4 text-muted-foreground transition-all disabled:opacity-50"
            >
              <RefreshCw className={`h-3.5 w-3.5 ${isLoading ? "animate-spin" : ""}`} />
              RELOAD DATA
            </button>
          </div>
        </div>

        {/* Error Alert */}
        {error && (
          <div className="p-4 rounded-xl bg-destructive/10 border border-destructive/20 text-destructive text-xs flex items-center gap-3">
            <AlertCircle className="h-5 w-5 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        {/* Volume Health & Capacity Overview Card */}
        <NFSVolumeStatsCard
          volume={volume}
          totalFolders={totalFolders}
          isLoading={isLoading}
          onAuditClick={handleTriggerAudit}
          isAuditing={isAuditing}
          activeJob={activeJob}
        />

        {/* Folder Metadata Table */}
        <FolderMetadataTable
          folders={folders}
          organizations={organizations}
          isLoading={isLoading}
          onEditFolder={handleOpenEditModal}
        />

        {/* Hanging Projects Discovery & Cleanup Section */}
        <HangingProjectsSection
          token={token}
          onProjectsDeleted={fetchData}
          onRunCleanupJob={(dryRun) => handleTriggerJob("hanging-cleanup", dryRun)}
          activeJob={activeJob}
        />

      </div>

      {/* Edit Quota & Lock Modal */}
      <EditQuotaModal
        folder={selectedFolder}
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        onSave={handleSaveMetadata}
      />
    </main>
  )
}
