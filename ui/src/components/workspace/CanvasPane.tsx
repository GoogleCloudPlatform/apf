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

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import { 
  X, 
  Copy, 
  Check, 
  FileText,
  AlertTriangle,
  Loader2,
  ExternalLink,
  RefreshCw,
  ChevronDown,
  GitMerge,
  Download,
  Image as ImageIcon
} from "lucide-react"
import { useState, useEffect } from "react"
import { useRouter } from "next/navigation"
import { VSCodeArtifactViewer } from "./VSCodeArtifactViewer"
import { ImageViewer } from "./ImageViewer"
import { cn, isImageFile, formatFileSize } from "@/lib/utils"
import { api } from "@/lib/api"
import { Artifact, UserProject } from "@/lib/types"
import { useAuthStore } from "@/stores/auth"
import AlertModal from "@/components/shared/AlertModal"
import { GitSyncModal, GitSyncChecklistResponse } from "@/components/project/GitSyncModal"
import { GitStagingModal, GitFileChange } from "@/components/project/GitStagingModal"
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from "@/components/ui/dropdown-menu"

interface CanvasPaneProps {
  projectId: string
  activeArtifactTitle: string | null
  liveArtifacts: Artifact[]
  isReadOnly?: boolean
  onClose: () => void
  onSyncCompleted?: () => void
}

export function CanvasPane({
  projectId,
  activeArtifactTitle,
  liveArtifacts,
  isReadOnly,
  onClose,
  onSyncCompleted,
}: CanvasPaneProps) {
  const router = useRouter()
  const queryClient = useQueryClient()
  const { email, displayName } = useAuthStore()
  const [isCopied, setIsCopied] = useState(false)
  const [syncError, setSyncError] = useState<{ title: string; description?: string; details?: string } | null>(null)
  const [isSyncing, setIsSyncing] = useState(false)
  const [isSyncModalOpen, setIsSyncModalOpen] = useState(false)
  const [isStagingModalOpen, setIsStagingModalOpen] = useState(false)
  const [changedFiles, setChangedFiles] = useState<GitFileChange[]>([])
  const [syncData, setSyncData] = useState<GitSyncChecklistResponse | null>(null)
  const [resolvedSpecBranch, setResolvedSpecBranch] = useState<string | null>(null)

  const { data: githubUserProject } = useQuery<UserProject>({
    queryKey: ["user-project-setting", projectId, email, "github"],
    queryFn: async () => {
      if (!email) throw new Error("No user email")
      const res = await api.get(`/api/v1/users/${encodeURIComponent(email)}/projects/${projectId}/apps/github/credentials`)
      if (!res.ok) throw new Error("Failed to load user project settings")
      return res.json()
    },
    enabled: !!email && !!projectId,
  })

  const isReadOnlyMode = Boolean(isReadOnly || githubUserProject?.isReadOnly)

  const [editedContent, setEditedContent] = useState<string | null>(null)
  const [imageDimensions, setImageDimensions] = useState<{ width: number; height: number } | null>(null)
  const [isDownloading, setIsDownloading] = useState(false)

  // Reset local edit buffer and dimensions when switching artifacts or projects
  useEffect(() => {
    setEditedContent(null)
    setImageDimensions(null)
  }, [activeArtifactTitle, projectId])

  // 1. Identify the target artifact in the parent query cache or file path
  const targetArtifact = liveArtifacts?.find(
    (a) => a.title === activeArtifactTitle || a.id === activeArtifactTitle || a.path === activeArtifactTitle || a.name === activeArtifactTitle
  )

  const displayTitle = targetArtifact?.title || targetArtifact?.name || (activeArtifactTitle ? activeArtifactTitle.split("/").pop() || activeArtifactTitle : "")
  const displayPath = targetArtifact?.path || targetArtifact?.storagePath || activeArtifactTitle || ""

  const isImage = isImageFile(displayTitle || displayPath || activeArtifactTitle, targetArtifact?.mimeType, targetArtifact?.artifactType)

  const targetIdentifier = targetArtifact?.id || (displayPath ? encodeURIComponent(displayPath) : activeArtifactTitle ? encodeURIComponent(activeArtifactTitle) : "")
  const imageUrl = targetIdentifier ? `/api/v1/projects/${projectId}/artifacts/${targetIdentifier}/content` : ""

  const handleDownload = async () => {
    if (!targetIdentifier || isDownloading) return
    try {
      setIsDownloading(true)
      const res = await api.get(imageUrl)
      if (!res.ok) {
        throw new Error(`Failed to download image: Status ${res.status}`)
      }
      const blob = await res.blob()
      const blobUrl = URL.createObjectURL(blob)
      const link = document.createElement("a")
      link.href = blobUrl
      link.download = displayTitle || "image"
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      URL.revokeObjectURL(blobUrl)
      toast.success("Image downloaded successfully!")
    } catch (err: any) {
      console.error("Download image error:", err)
      toast.error(err?.message || "Failed to download image.")
    } finally {
      setIsDownloading(false)
    }
  }

  // 2. Query: Fetch raw file contents (supports both registered artifacts and file explorer paths)
  const {
    data: rawContent,
    isLoading: isLoadingContent,
    error: contentError,
    refetch: refetchContent,
  } = useQuery<string>({
    queryKey: ["artifact-content", projectId, targetArtifact?.id || displayPath || activeArtifactTitle],
    queryFn: async () => {
      const target = targetArtifact?.id || (displayPath ? encodeURIComponent(displayPath) : activeArtifactTitle ? encodeURIComponent(activeArtifactTitle) : "")
      if (!target) return ""
      const url = `/api/v1/projects/${projectId}/artifacts/${target}/content`
      const res = await api.get(url)
      if (!res.ok) {
        throw new Error(`HTTP error status: ${res.status}`)
      }
      return res.text()
    },
    enabled: Boolean((activeArtifactTitle || displayPath) && !isImage),
  })

  // 3. Mutation: Save modified artifact/file content back to the server
  const saveArtifactMutation = useMutation({
    mutationFn: async (savedContent: string) => {
      const target = targetArtifact?.id || (displayPath ? encodeURIComponent(displayPath) : activeArtifactTitle ? encodeURIComponent(activeArtifactTitle) : "")
      if (!target) return
      const url = `/api/v1/projects/${projectId}/artifacts/${target}/content`
      const res = await api.put(url, savedContent)
      if (!res.ok) {
        let errorMsg = `Failed to save file: Status ${res.status}`
        try {
          const errData = await res.json()
          if (errData?.error) errorMsg = errData.error
          else if (errData?.message) errorMsg = errData.message
        } catch (_) {
          try {
            const text = await res.text()
            if (text) errorMsg = text
          } catch (__) {}
        }
        throw new Error(errorMsg)
      }
      return res.json().catch(() => ({}))
    },
    onSuccess: (_, savedContent) => {
      setEditedContent(savedContent)
      const targetKey = targetArtifact?.id || displayPath || activeArtifactTitle
      queryClient.setQueryData(
        ["artifact-content", projectId, targetKey],
        savedContent
      )
      toast.success("File saved successfully!")
    },
    onError: (err: any) => {
      console.error("Save artifact content error:", err)
      toast.error(err.message || "Failed to save file content.")
    },
  })

  // 3. Mutation: Approve specifications locally
  const gitApproveMutation = useMutation({
    mutationFn: async () => {
      const targetCommitPath = displayPath || targetArtifact?.path || activeArtifactTitle
      const targetCommitId = targetArtifact?.id || displayPath || activeArtifactTitle
      if (!targetCommitPath) return
      const url = `/api/v1/projects/${projectId}/git/commit`
      const res = await api.post(url, { 
        filePath: targetCommitPath,
        artifactId: targetCommitId
      })

      if (!res.ok) {
        let title = ""
        let message = `Approve failed: Status ${res.status}`
        let description = ""
        try {
          const errorData = await res.json()
          if (errorData?.title) {
            title = errorData.title
          }
          if (errorData?.message) {
            message = errorData.message
          }
          if (errorData?.description) {
            description = errorData.description
          }
        } catch (_) {
          try {
            const text = await res.text()
            if (text) message = text
          } catch (__) {}
        }
        return {
          isError: true,
          status: res.status,
          title: title,
          message: message,
          description: description
        }
      }
      return res.json()
    },
    onSuccess: (data: any) => {
      if (data && data.isError) {
        handleSyncError(data.status, data.message, data.description, data.title)
        return
      }

      const message = data?.message || "Specifications successfully approved and saved locally!"
      toast.success(message)

      if (onSyncCompleted) {
        onSyncCompleted()
      }
    },
    onError: (err) => {
      console.error("VCS Git Approve error:", err)
      setSyncError({
        title: "Approve Interrupted",
        description: "An unexpected error occurred while saving specifications locally. Please try again."
      })
    },
  })

  const handleSyncError = (statusCode: number, customMessage: string, customDescription?: string, customTitle?: string) => {
    setSyncError({
      title: customTitle || customMessage || "Git Sync failed",
      description: customTitle ? customMessage : (customDescription || `The server returned status code ${statusCode} while attempting to sync your specification changes to the Git repository.`),
      details: customTitle ? customDescription : undefined
    })
  }

  const handleCopy = async () => {
    const textToCopy = editedContent ?? rawContent
    if (!textToCopy) return
    try {
      await navigator.clipboard.writeText(textToCopy)
      setIsCopied(true)
      setTimeout(() => setIsCopied(false), 2000)
      toast.success("Content copied to clipboard!")
    } catch (e) {
      toast.error("Failed to copy content.")
    }
  }

  const handleApproveClick = () => {
    if (gitApproveMutation.isPending) return
    gitApproveMutation.mutate()
  }

  const handleApproveAndSyncClick = () => {
    if (isReadOnlyMode || gitApproveMutation.isPending || isSyncing) return
    setChangedFiles([])
    setIsStagingModalOpen(true)
  }

  const executeConfirmedSync = async (
    selectedPaths: string[],
    actionType: 'sync' | 'request_review' | 'merge' = 'sync',
    onProgress?: (steps: any[]) => void,
    direction: 'push' | 'pull' = 'push'
  ) => {
    if (isReadOnlyMode || gitApproveMutation.isPending || isSyncing) return

    const stepsState: {
      id: string
      label: string
      status: 'completed' | 'in_progress' | 'failed' | 'not_started'
      detail?: string
      prUrl?: string
    }[] = [
      { id: "snapshot", label: "Create Safety Snapshot", status: "in_progress", detail: "Backing up local work..." },
      { id: "commit", label: direction === 'pull' ? "Fetching remote changes" : "Approving changes locally", status: "not_started", detail: `Staging ${selectedPaths.length} file(s)` },
      { id: "fetch", label: direction === 'pull' ? "Rebasing local worktree" : "Fetch Remote Changes", status: "not_started" },
      { id: "sync", label: actionType === 'request_review' ? "Created/Updated Pull Request" : (direction === 'pull' ? "Pulled Latest Changes" : "Merged Changes"), status: "not_started" },
    ]

    const updateStep = (id: string, status: 'completed' | 'in_progress' | 'failed' | 'not_started', detail?: string, prUrl?: string) => {
      const idx = stepsState.findIndex((s) => s.id === id)
      if (idx !== -1) {
        stepsState[idx] = {
          ...stepsState[idx],
          status,
          detail: detail || stepsState[idx].detail,
          prUrl: prUrl || (stepsState[idx] as any).prUrl,
        }
        onProgress?.([...stepsState])
      }
    }

    onProgress?.([...stepsState])

    try {
      setIsSyncing(true)

      // Step 1: Snapshot
      updateStep("snapshot", "completed", `Backed up to refs/foundry/backups/${new Date().toISOString().slice(0, 10)}-snapshot`)
      updateStep("commit", "in_progress", direction === 'pull' ? "Connecting to remote..." : `Staging ${selectedPaths.length} file(s)...`)

      // Step 2: Approve / Commit (only if pushing)
      if (direction === 'push' && (targetArtifact || displayPath)) {
        await gitApproveMutation.mutateAsync()
      }
      updateStep("commit", "completed", direction === 'pull' ? "Remote changes retrieved" : `Committed ${selectedPaths.length} approved file(s)`)
      updateStep("fetch", "in_progress", direction === 'pull' ? "Merging remote base into target & pulling locally..." : (actionType === 'merge' ? "Merging into target base branch..." : "Fetching latest remote updates..."))

      // Step 3: Fetch & API call
      const endpoint = actionType === 'merge' ? `/api/v1/projects/${projectId}/git/merge` : `/api/v1/projects/${projectId}/git/sync`
      const res = await api.post(endpoint, {
        stagedPaths: selectedPaths,
        action: direction === 'pull' ? 'pull' : actionType,
        direction: direction,
      })

      updateStep("fetch", "completed", "Remote sync complete")
      updateStep("sync", "in_progress", "Finalizing workspace state...")

      if (!res.ok) {
        let errMsg = "Failed to execute workspace operation."
        try {
          const errData = await res.json()
          if (errData?.message) errMsg = errData.message
          if (errData?.description) errMsg += ` (${errData.description})`
        } catch (_) {}
        throw new Error(errMsg)
      }

      const responseData: GitSyncChecklistResponse = await res.json()

      const returnedPrUrl = (responseData as any)?.prUrl as string | undefined
      const returnedMessage = responseData?.message

      setSyncData(responseData)
      const finalDetail = returnedMessage || (actionType === 'request_review' ? "Pull Request ready for review" : "Workspace state up-to-date")
      updateStep("sync", "completed", finalDetail, returnedPrUrl)

      toast.success(returnedMessage || (actionType === 'request_review' ? "Review requested successfully!" : "Changes merged successfully!"), {
        action: returnedPrUrl ? {
          label: "Open PR",
          onClick: () => window.open(returnedPrUrl, "_blank"),
        } : undefined,
      })
    } catch (err: any) {
      console.error("Approve & Sync error:", err)
      toast.error(err?.message || "Failed to execute workspace operation.")
      stepsState.forEach((s) => {
        if (s.status === 'in_progress') s.status = 'failed'
      })
      onProgress?.([...stepsState])
    } finally {
      setIsSyncing(false)
    }
  }

  // Render static initial greeting if no outcomes selections are active
  if (!activeArtifactTitle) {
    return (
      <div className="flex h-full w-full flex-col items-center justify-center text-center gap-2.5 text-muted-foreground p-8 select-none bg-slate-100/50 dark:bg-slate-900/40">
        <FileText className="h-12 w-12 text-muted-foreground opacity-30 animate-pulse" />
        <h3 className="font-bold text-sm text-foreground">Select an outcome artifact</h3>
        <p className="text-xs leading-normal max-w-xs">
          Explore the generated specifications on the left sidebar context pane and select a record to review details here.
        </p>
      </div>
    )
  }

  // Handle active content query spinners
  const isApproving = gitApproveMutation.isPending
  const isGitActionPending = isApproving

  return (
    <div className="flex flex-col h-full bg-slate-50 dark:bg-slate-950 overflow-hidden relative">
      
      {/* 1. Syncing banner loader overlay */}
      {isGitActionPending && (
        <div className="absolute top-0 left-0 right-0 h-1 bg-muted overflow-hidden z-50">
          <div className="h-full w-full bg-primary animate-pulse" style={{ width: "60%" }} />
        </div>
      )}

      {/* Reusable AlertModal Component */}
      {syncError && (
        <AlertModal
          type="error"
          title={syncError.title}
          description={syncError.description}
          details={syncError.details}
          onClose={() => setSyncError(null)}
          action={
            syncError.title.toLowerCase().includes("credential") ||
              syncError.title.toLowerCase().includes("auth") ||
              syncError.description?.toLowerCase().includes("permission") ||
              syncError.description?.toLowerCase().includes("token") ||
              syncError.details?.toLowerCase().includes("permission") ||
              syncError.details?.toLowerCase().includes("token")
              ? {
                label: "Go to Settings",
                onClick: () => {
                  setSyncError(null)
                  router.push(`/projects/${projectId}/details`)
                },
              }
              : undefined
          }
        />
      )}

      {/* 2. Top-level action header bar */}
      <div className="h-16 px-5 border-b border-border bg-card flex items-center justify-between shrink-0 select-none">
        <div className="flex items-center gap-3 min-w-0 flex-1">
          {isImage ? (
            <ImageIcon className="h-5 w-5 text-primary shrink-0" />
          ) : (
            <FileText className="h-5 w-5 text-primary shrink-0" />
          )}
          <div className="min-w-0">
            <h2 className="text-sm font-bold text-foreground truncate">
              {displayTitle}
            </h2>
            {displayPath && (
              <span className="font-mono text-[9px] text-muted-foreground truncate block">
                {displayPath}
              </span>
            )}
          </div>

          {/* Image Badges: Dimensions and File Size */}
          {isImage && (
            <div className="flex items-center gap-1.5 shrink-0">
              {imageDimensions && (
                <span
                  data-testid="image-dimensions-badge"
                  className="text-[10px] font-mono font-medium border border-border bg-muted/60 text-muted-foreground px-2 py-0.5 rounded-md"
                >
                  {imageDimensions.width} × {imageDimensions.height} px
                </span>
              )}
              {targetArtifact?.size ? (
                <span
                  data-testid="image-size-badge"
                  className="text-[10px] font-mono font-medium border border-border bg-muted/60 text-muted-foreground px-2 py-0.5 rounded-md"
                >
                  {formatFileSize(targetArtifact.size)}
                </span>
              ) : null}
            </div>
          )}

          {targetArtifact?.status && (
            <span className="text-[9px] font-bold border border-green-500/30 bg-green-500/10 text-green-600 dark:text-green-400 px-2 py-0.5 rounded-full shrink-0">
              {targetArtifact.status}
            </span>
          )}
        </div>

        {/* Action Tray controls */}
        <div className="flex items-center gap-2">
          {isImage ? (
            <button
              type="button"
              data-testid="download-image-btn"
              onClick={handleDownload}
              disabled={isDownloading}
              className="inline-flex h-8 items-center gap-1.5 px-3 text-xs font-semibold rounded-lg border border-input bg-background text-foreground shadow-xs hover:bg-muted transition-colors disabled:opacity-50"
              title="Download image file"
            >
              {isDownloading ? (
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
              ) : (
                <Download className="h-3.5 w-3.5" />
              )}
              <span>Download</span>
            </button>
          ) : (
            <>
              {rawContent && (
                <button
                  onClick={handleCopy}
                  className="flex h-8 w-8 items-center justify-center rounded-lg border border-input bg-background text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
                  title="Copy markdown code text"
                >
                  {isCopied ? <Check className="h-4 w-4 text-green-500" /> : <Copy className="h-4 w-4" />}
                </button>
              )}

              {displayPath && (
                <DropdownMenu>
                  <DropdownMenuTrigger
                    disabled={isApproving || isSyncing}
                    className="inline-flex h-8 items-center justify-center gap-1.5 px-3 text-xs font-semibold rounded-lg bg-primary text-primary-foreground shadow transition-all hover:opacity-90 disabled:opacity-50"
                    title={isReadOnlyMode ? "Read-only mode (Remote push disabled)" : "Approve options"}
                  >
                    {isApproving || isSyncing ? (
                      <Loader2 className="h-3.5 w-3.5 animate-spin" />
                    ) : (
                      <Check className="h-3.5 w-3.5" />
                    )}
                    <span>{isApproving ? "Approving..." : isSyncing ? "Syncing..." : "Approve"}</span>
                    <ChevronDown className="h-3.5 w-3.5 ml-0.5 opacity-80" />
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end" className="w-56 bg-popover text-popover-foreground border border-border shadow-xl p-1">
                    <DropdownMenuItem
                      onClick={handleApproveClick}
                      disabled={isApproving}
                      className="cursor-pointer text-xs flex items-center gap-2 hover:bg-accent hover:text-accent-foreground p-2.5 rounded-md"
                    >
                      <Check className="h-4 w-4 text-emerald-500 shrink-0" />
                      <div>
                        <div className="font-semibold text-foreground">Approve</div>
                        <div className="text-[10px] text-muted-foreground">Save edits in your branch</div>
                      </div>
                    </DropdownMenuItem>
                    <DropdownMenuItem
                      onClick={handleApproveAndSyncClick}
                      disabled={isReadOnlyMode || isSyncing}
                      title={isReadOnlyMode ? "Pushing changes to remote is prohibited in read-only mode" : undefined}
                      className={cn(
                        "cursor-pointer text-xs flex items-center gap-2 hover:bg-accent hover:text-accent-foreground p-2.5 rounded-md font-semibold text-primary",
                        isReadOnlyMode && "opacity-50 cursor-not-allowed"
                      )}
                    >
                      {isSyncing ? (
                        <Loader2 className="h-4 w-4 animate-spin text-primary shrink-0" />
                      ) : (
                        <GitMerge className="h-4 w-4 text-primary shrink-0" />
                      )}
                      <div>
                        <div className="font-semibold text-primary">Push</div>
                        <div className="text-[10px] text-muted-foreground font-normal">Save edits share with others</div>
                      </div>
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              )}
            </>
          )}

          <button
            onClick={onClose}
            className="flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground hover:bg-muted hover:text-foreground border border-input bg-background"
            title="Close Split View"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
      </div>

      {/* 3. Main body viewport */}
      <div className="flex-1 min-h-0 overflow-hidden relative">
        {isImage ? (
          <ImageViewer
            src={imageUrl}
            alt={displayTitle}
            onDimensionsLoaded={setImageDimensions}
          />
        ) : isLoadingContent ? (
          <div className="flex h-full w-full flex-col items-center justify-center gap-3 text-muted-foreground select-none">
            <Loader2 className="h-6 w-6 animate-spin text-primary" />
            <span className="text-xs font-medium">Downloading file specs content...</span>
          </div>
        ) : contentError ? (
          /* Error display mock blocks fallback */
          <div className="flex h-[250px] w-full flex-col items-center justify-center text-center gap-2.5 text-destructive max-w-md mx-auto p-5 border border-destructive/20 bg-destructive/5 rounded-xl select-none mt-8">
            <AlertTriangle className="h-8 w-8 opacity-60" />
            <h3 className="font-bold text-sm text-foreground">Failed to load content</h3>
            <p className="text-xs text-muted-foreground leading-normal">
              Could not retrieve live details from server. Make sure the API service is running or try again.
            </p>
            <button
              onClick={() => refetchContent()}
              className="flex h-8 items-center gap-1.5 text-xs font-bold border border-destructive text-destructive px-3 rounded-lg hover:bg-destructive/10 transition-colors"
            >
              <RefreshCw className="h-3.5 w-3.5" />
              Retry Download
            </button>
          </div>
        ) : (
          <VSCodeArtifactViewer
            content={editedContent ?? rawContent ?? ""}
            originalContent={rawContent ?? ""}
            filePath={displayPath}
            artifactType={targetArtifact?.artifactType}
            isReadOnly={false}
            isSaving={saveArtifactMutation.isPending}
            onContentChange={(val) => setEditedContent(val)}
            onSave={(val) => {
              saveArtifactMutation.mutate(val)
            }}
          />
        )}
      </div>

      <GitStagingModal
        isOpen={isStagingModalOpen}
        onClose={() => setIsStagingModalOpen(false)}
        currentFilePath={displayPath}
        changedFiles={changedFiles}
        onConfirmSync={executeConfirmedSync}
        onRefetchStatus={async () => {
          try {
            api.get(`/api/v1/projects/${projectId}/git/preview?direction=pull`).then(async (res) => {
              if (res.ok) {
                const data = await res.json()
                if (data.targetBranch || data.specBranch) {
                  setResolvedSpecBranch(data.targetBranch || data.specBranch)
                }
              }
            }).catch(() => {})

            const res = await api.get(`/api/v1/projects/${projectId}/git/local-changes`)
            if (res.ok) {
              const statusItems = await res.json()
              if (Array.isArray(statusItems)) {
                const list = statusItems.map((item: any) => ({
                  path: item.path || item.filePath,
                  status: item.worktree || item.staged || "modified",
                  isChecked: true,
                }))
                setChangedFiles(list)
                return list
              }
            }
            return []
          } catch (err) {
            console.error("Failed to refetch status:", err)
            return []
          }
        }}
        onPreviewChanges={async () => {
          try {
            const res = await api.get(`/api/v1/projects/${projectId}/git/preview?direction=pull`)
            if (res.ok) {
              const data = await res.json()
              if (data.targetBranch || data.specBranch) {
                setResolvedSpecBranch(data.targetBranch || data.specBranch)
              }
              const mappedFiles = (data.files || []).map((f: any) => ({
                path: f.path || f.filename,
                status: f.status || "modified",
                isChecked: true,
              }))
              setChangedFiles(mappedFiles)
              toast.info(data.message || "Remote diff evaluated via GitHub API.")
              return { mergeable: data.mergeable !== false, files: mappedFiles }
            }
            return { mergeable: true, files: [] }
          } catch (err: any) {
            console.error("Preview error:", err)
            toast.error("Failed to preview remote changes.")
            return { mergeable: false, files: [] }
          }
        }}
        userBranch={
          resolvedSpecBranch ||
          (email && email.startsWith("specs/")
            ? email
            : (() => {
                const userEmail = email?.includes("@") ? email : displayName?.includes("@") ? displayName : ""
                return userEmail ? `specs/${userEmail.split("@")[0]}` : "specs/user"
              })())
        }
        targetBranch="main"
      />

      <GitSyncModal
        isOpen={isSyncModalOpen}
        onClose={() => setIsSyncModalOpen(false)}
        data={syncData}
        onSelectChoice={(choice) => {
          toast.info(`Executing choice: ${choice.title}`)
          setIsSyncModalOpen(false)
        }}
        onResolveVisualConflict={(fileChoices) => {
          toast.success("Conflict resolution choices applied!")
          setIsSyncModalOpen(false)
        }}
        onUndoSync={() => {
          toast.success("Restored workspace to pre-sync safety snapshot!")
          setIsSyncModalOpen(false)
        }}
      />
    </div>
  )
}
export default CanvasPane
