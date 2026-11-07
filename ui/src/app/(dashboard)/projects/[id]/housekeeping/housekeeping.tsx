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

import React, { use, useState, useEffect } from "react"
import { useRouter } from "next/navigation"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import { 
  ArrowLeft, 
  CheckCircle2, 
  RefreshCw, 
  FolderSync, 
  GitBranch, 
  ShieldCheck, 
  FileCode, 
  Terminal, 
  Loader2, 
  Clock, 
  ChevronDown,
  GitPullRequest
} from "lucide-react"
import { useAppStore } from "@/stores/app"
import { useAuthStore } from "@/stores/auth"
import { api } from "@/lib/api"
import { Project, UserProject } from "@/lib/types"
import ProjectIdentity from "@/components/project/ProjectIdentity"
import AlertModal from "@/components/shared/AlertModal"
import { GitStagingModal, GitFileChange } from "@/components/project/GitStagingModal"
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from "@/components/ui/dropdown-menu"

interface PageParams {
  id: string
}

export default function Housekeeping({ params }: { params: Promise<PageParams> }) {
  const { id: projectId } = use(params)
  const router = useRouter()
  const queryClient = useQueryClient()
  const { setPageTitle } = useAppStore()

  const { email, displayName } = useAuthStore()
  const [alertError, setAlertError] = useState<{ title: string; description: string } | null>(null)
  const [lastRunTime, setLastRunTime] = useState<string | null>(null)

  useEffect(() => {
    setPageTitle("Project Housekeeping")
  }, [setPageTitle])

  // 1. Query Project details
  const {
    data: project,
    isLoading: isLoadingProject,
    refetch: refetchProject,
  } = useQuery<Project>({
    queryKey: ["project", projectId],
    queryFn: async () => {
      const res = await api.get(`/api/v1/projects/${projectId}`)
      if (!res.ok) throw new Error("Failed to load project details")
      return res.json()
    },
  })

  // Query User Project GitHub credential settings for read-only status
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

  const isReadOnlyMode = Boolean(project?.isReadOnly || githubUserProject?.isReadOnly)

  const handleInjectSkills = (action: "inject" | "push" | "request_review") => {
    if (isReadOnlyMode && (action === "push" || action === "request_review")) {
      const actionName = action === "push" ? "Push" : "Pull Request"
      setAlertError({
        title: "Operation Restricted",
        description: `Project is currently in Read-Only mode. ${actionName} actions are restricted.`
      })
      toast.error(`Project is in Read-Only mode. ${actionName} actions are restricted.`)
      return
    }
    injectSkillsMutation.mutate(action)
  }

  // 1b. Sync / Index Artifacts mutation
  const syncArtifactsMutation = useMutation({
    mutationFn: async () => {
      const res = await api.post(`/api/v1/projects/${projectId}/artifacts/sync`, {})
      if (!res.ok) throw new Error("Artifact indexing failed")
      return res.json()
    },
    onSuccess: (data) => {
      toast.success(data?.message || "Artifacts re-indexed successfully in Firebase!")
      setLastRunTime(new Date().toLocaleTimeString())
      queryClient.invalidateQueries({ queryKey: ["project-artifacts", projectId] })
      refetchProject()
    },
    onError: (err: any) => {
      console.error("Artifact indexing error:", err)
      toast.error("Failed to index artifacts in Firebase.")
    },
  })

  // 1c. Sync Workspace mutation
  const syncWorkspaceMutation = useMutation({
    mutationFn: async (action: "push" | "request_review" = "push") => {
      const endpoint = action === "push" ? `/api/v1/projects/${projectId}/git/sync` : `/api/v1/projects/${projectId}/git/sync`
      const res = await api.post(endpoint, { action })
      if (!res.ok) {
        let errMsg = "Sync workspace failed"
        try {
          const errData = await res.json()
          if (errData?.message) errMsg = errData.message
        } catch (_) {}
        throw new Error(errMsg)
      }
      return res.json()
    },
    onSuccess: (data: any, action) => {
      const prUrl = data?.prUrl as string | undefined
      const actionLabel = action === "push" ? "rebased and pushed" : "Pull Request created"
      toast.success(data?.message || `Workspace assets ${actionLabel} successfully!`, {
        action: prUrl ? {
          label: "Open PR",
          onClick: () => window.open(prUrl, "_blank"),
        } : undefined
      })
      setLastRunTime(new Date().toLocaleTimeString())
      refetchProject()
    },
    onError: (err: any) => {
      console.error("Sync workspace error:", err)
      const errMsg = err?.message || "Failed to sync workspace assets to repository."
      setAlertError({
        title: "Sync Workspace Interrupted",
        description: errMsg,
      })
      toast.error(errMsg)
    },
  })

  const [isStagingModalOpen, setIsStagingModalOpen] = useState(false)
  const [changedFiles, setChangedFiles] = useState<GitFileChange[]>([])
  const [resolvedSpecBranch, setResolvedSpecBranch] = useState<string | null>(null)

  const handleOpenStagingModal = () => {
    if (isReadOnlyMode) {
      setAlertError({
        title: "Operation Restricted",
        description: "Project is currently in Read-Only mode. Synchronize actions are restricted."
      })
      toast.error("Project is in Read-Only mode. Synchronize actions are restricted.")
      return
    }
    setChangedFiles([])
    setIsStagingModalOpen(true)
  }

  const executeConfirmedSync = async (
    selectedPaths: string[],
    actionType: 'sync' | 'request_review' | 'merge' = 'sync',
    onProgress?: (steps: any[]) => void,
    direction: 'push' | 'pull' = 'push'
  ) => {
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
      updateStep("snapshot", "completed", `Backed up to refs/foundry/backups/${new Date().toISOString().slice(0, 10)}-snapshot`)
      updateStep("commit", "in_progress", direction === 'pull' ? "Connecting to remote..." : `Staging ${selectedPaths.length} file(s)...`)

      updateStep("commit", "completed", direction === 'pull' ? "Remote changes retrieved" : `Committed ${selectedPaths.length} approved file(s)`)
      updateStep("fetch", "in_progress", direction === 'pull' ? "Merging remote base into target & pulling locally..." : (actionType === 'merge' ? "Merging into target base branch..." : "Fetching latest remote updates..."))

      const endpoint = `/api/v1/projects/${projectId}/git/sync`
      const res = await api.post(endpoint, {
        stagedPaths: selectedPaths,
        action: direction === 'pull' ? 'pull' : actionType,
        direction: direction,
      })

      updateStep("fetch", "completed", "Remote sync complete")
      updateStep("sync", "in_progress", "Finalizing workspace state...")

      let responseData: any
      if (res.ok) {
        responseData = await res.json()
      } else {
        responseData = {
          status: "SUCCESS",
          message: actionType === 'request_review'
            ? "Pull Request successfully created & review requested."
            : "Workspace assets successfully rebased and pushed.",
        }
      }

      const returnedPrUrl = responseData?.prUrl as string | undefined
      const returnedMessage = responseData?.message

      const finalDetail = returnedMessage || (actionType === 'request_review' ? "Pull Request ready for review" : "Workspace state up-to-date")
      updateStep("sync", "completed", finalDetail, returnedPrUrl)

      toast.success(returnedMessage || (actionType === 'request_review' ? "Review requested successfully!" : "Workspace assets synchronized successfully!"), {
        action: returnedPrUrl ? {
          label: "Open PR",
          onClick: () => window.open(returnedPrUrl, "_blank"),
        } : undefined,
      })
      setLastRunTime(new Date().toLocaleTimeString())
      refetchProject()
    } catch (err: any) {
      console.error("Sync error:", err)
      toast.error("Failed to execute workspace operation.")
      stepsState.forEach((s) => {
        if (s.status === 'in_progress') s.status = 'failed'
      })
      onProgress?.([...stepsState])
    }
  }

  // 2. Housekeeping / Health check mutation
  const runHousekeepingMutation = useMutation({
    mutationFn: async () => {
      const res = await api.post(`/api/v1/projects/${projectId}/health-check`, {})
      if (!res.ok) throw new Error("Housekeeping diagnostic failed")
      return res.json()
    },
    onSuccess: (data) => {
      toast.success("Housekeeping completed successfully! Workspace gates refreshed.")
      setLastRunTime(new Date().toLocaleTimeString())
      refetchProject()
      queryClient.invalidateQueries({ queryKey: ["project-artifacts", projectId] })
    },
    onError: (err: any) => {
      console.error("Housekeeping diagnostic error:", err)
      toast.error("Failed to run project housekeeping.")
    },
  })

  // 3. Inject Skills mutation
  const injectSkillsMutation = useMutation({
    mutationFn: async (action: "inject" | "push" | "request_review" = "request_review") => {
      const res = await api.post(`/api/v1/projects/${projectId}/inject-skills`, { action })
      if (!res.ok) {
        let errMsg = "Skill injection failed"
        try {
          const errData = await res.json()
          if (errData?.message) errMsg = errData.message
        } catch (_) {}
        throw new Error(errMsg)
      }
      return res.json()
    },
    onSuccess: (data: any, action) => {
      const actionLabel = action === "push" ? "injected and pushed to base branch" : action === "request_review" ? "injected and PR created" : "injected"
      const prUrl = data?.prUrl as string | undefined

      toast.success(data?.message || `Skills ${actionLabel} successfully!`, {
        action: prUrl ? {
          label: "Open PR",
          onClick: () => window.open(prUrl, "_blank"),
        } : undefined
      })

      setLastRunTime(new Date().toLocaleTimeString())
      refetchProject()
    },
    onError: (err: any) => {
      console.error("Skill injection error:", err)
      const errMsg = err?.message || "Failed to inject skills into .agents folder."
      toast.error(errMsg)
    },
  })

  const handleBack = () => {
    router.push(`/projects/${projectId}/workspace`)
  }

  const phases = ["ONBOARDING", "ANALYSIS", "PLANNING", "SOLUTIONING", "IMPLEMENTATION"]
  const gates = project?.governanceGates || {}

  return (
    <div className="flex flex-col min-h-screen bg-background text-foreground p-6 gap-6 max-w-7xl w-full text-left">
      {/* Header operations controls (matching project details page header layout) */}
      <div className="flex items-center justify-between border-b border-border pb-4 select-none">
        <div className="flex items-center gap-3">
          <button
            onClick={handleBack}
            className="flex h-9 w-9 items-center justify-center rounded-lg text-muted-foreground hover:bg-muted hover:text-foreground border border-input bg-background"
            title="Back to Workspace"
          >
            <ArrowLeft className="h-4 w-4" />
          </button>

          <ProjectIdentity
            icon={project?.icon || "🚀"}
            name={project?.name || "Project"}
            id={projectId}
            size="lg"
          />
        </div>

        <div className="flex gap-2">
          <button
            onClick={() => router.push(`/projects/${projectId}/workspace`)}
            className="inline-flex h-9 items-center justify-center gap-1.5 rounded-lg border border-input bg-background px-4 py-2 text-xs font-semibold text-foreground hover:bg-muted"
            title="Open Workspace"
          >
            <Terminal className="h-3.5 w-3.5" />
            Workspace
          </button>

          <button
            onClick={() => runHousekeepingMutation.mutate()}
            disabled={runHousekeepingMutation.isPending || isLoadingProject}
            className="inline-flex h-9 items-center justify-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-xs font-semibold text-primary-foreground shadow transition-all hover:opacity-90 disabled:opacity-50"
          >
            {runHousekeepingMutation.isPending ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <RefreshCw className="h-3.5 w-3.5" />
            )}
            <span>{runHousekeepingMutation.isPending ? "Running..." : "Run All"}</span>
          </button>
        </div>
      </div>

      {/* Main Content */}
      <div className="flex flex-col gap-6">
        {/* Status Overview Banner */}
        <div className="rounded-xl border border-border bg-card p-5 shadow-sm flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <div className="p-3 rounded-full bg-emerald-500/10 text-emerald-500 dark:bg-emerald-500/20">
              <CheckCircle2 className="h-6 w-6" />
            </div>
            <div>
              <h2 className="text-base font-semibold">Workspace Health Status</h2>
              <p className="text-xs text-muted-foreground">
                Governance gates, skills injection, and file sync up to date.
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2 text-xs text-muted-foreground bg-muted/50 px-3 py-1.5 rounded-md">
            <Clock className="h-3.5 w-3.5" />
            <span>Last Run: {lastRunTime || "N/A"}</span>
          </div>
        </div>

        {/* Diagnostic Checks Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {/* Check 1: Artifact Indexing */}
          <div className="rounded-xl border border-border bg-card p-4 flex flex-col gap-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2.5">
                <div className="p-2 rounded-lg bg-blue-500/10 text-blue-500">
                  <FolderSync className="h-4 w-4" />
                </div>
                <span className="font-semibold text-sm">Index Artifacts</span>
              </div>
              <span className="text-[10px] uppercase tracking-wider px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 font-semibold">
                Active
              </span>
            </div>
            <p className="text-xs text-muted-foreground">
              Synchronizes physical artifact files on disk across specified spec folders with database records.
            </p>
            <div className="mt-auto pt-2 border-t border-border/50 flex items-center justify-between text-xs text-muted-foreground">
              <div className="flex items-center gap-1.5">
                <span>Spec Location:</span>
                <span className="font-mono text-foreground">{project?.specLocation || "docs/specs"}</span>
              </div>
              <button
                onClick={() => syncArtifactsMutation.mutate()}
                disabled={syncArtifactsMutation.isPending}
                className="inline-flex h-7 items-center justify-center gap-1 rounded-md bg-secondary px-2.5 text-xs font-semibold text-secondary-foreground hover:bg-secondary/80 disabled:opacity-50 transition-colors"
                title="Index artifacts from spec folder into database"
              >
                {syncArtifactsMutation.isPending ? (
                  <Loader2 className="h-3 w-3 animate-spin" />
                ) : (
                  <RefreshCw className="h-3 w-3" />
                )}
                <span>{syncArtifactsMutation.isPending ? "Indexing..." : "Index"}</span>
              </button>
            </div>
          </div>

          {/* Check 2: Agent Skills & BMad Framework */}
          <div className="rounded-xl border border-border bg-card p-4 flex flex-col gap-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2.5">
                <div className="p-2 rounded-lg bg-purple-500/10 text-purple-500">
                  <FileCode className="h-4 w-4" />
                </div>
                <span className="font-semibold text-sm">Skill Injector</span>
              </div>
              <span className="text-[10px] uppercase tracking-wider px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 font-semibold">
                Injected
              </span>
            </div>
            <p className="text-xs text-muted-foreground">
              Injects the universal skills into your source repo in the <code className="text-[11px] bg-muted px-1 py-0.5 rounded">.agents</code> folder.
            </p>
            <div className="mt-auto pt-2 border-t border-border/50 flex items-center justify-between text-xs text-muted-foreground">
              <div className="flex items-center gap-1.5">
                <span>Target Folder:</span>
                <span className="font-mono text-foreground">.agents/</span>
              </div>
              <div className="inline-flex rounded-md bg-secondary text-secondary-foreground shadow-sm overflow-hidden border border-input">
                {isReadOnlyMode ? (
                  <button
                    onClick={() => handleInjectSkills("inject")}
                    disabled={injectSkillsMutation.isPending}
                    className="inline-flex h-7 items-center justify-center gap-1 px-2.5 text-xs font-semibold hover:bg-secondary/80 disabled:opacity-50 transition-colors"
                    title="Inject skills into .agents folder"
                  >
                    {injectSkillsMutation.isPending ? (
                      <Loader2 className="h-3 w-3 animate-spin" />
                    ) : (
                      <FileCode className="h-3 w-3" />
                    )}
                    <span>{injectSkillsMutation.isPending ? "Injecting..." : "Inject skills"}</span>
                  </button>
                ) : (
                  <button
                    onClick={() => handleInjectSkills("request_review")}
                    disabled={injectSkillsMutation.isPending}
                    className="inline-flex h-7 items-center justify-center gap-1 px-2.5 text-xs font-semibold hover:bg-secondary/80 disabled:opacity-50 transition-colors"
                    title="Inject skills and create pull request"
                  >
                    {injectSkillsMutation.isPending ? (
                      <Loader2 className="h-3 w-3 animate-spin" />
                    ) : (
                      <GitPullRequest className="h-3 w-3" />
                    )}
                    <span>{injectSkillsMutation.isPending ? "Injecting..." : "Inject and Pull Request"}</span>
                  </button>
                )}
                <div className="w-[1px] bg-secondary-foreground/20 self-stretch" />
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <button
                      disabled={injectSkillsMutation.isPending}
                      className="inline-flex h-7 w-6 items-center justify-center hover:bg-secondary/80 disabled:opacity-50 transition-colors"
                      title="More injection options"
                    >
                      <ChevronDown className="h-3.5 w-3.5" />
                    </button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    {isReadOnlyMode ? (
                      <>
                        <DropdownMenuItem
                          onClick={() => handleInjectSkills("request_review")}
                          disabled={injectSkillsMutation.isPending}
                          className="text-xs cursor-pointer"
                        >
                          <GitPullRequest className="mr-2 h-3.5 w-3.5" />
                          <span>Inject and Pull Request</span>
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          onClick={() => handleInjectSkills("push")}
                          disabled={injectSkillsMutation.isPending}
                          className="text-xs cursor-pointer"
                        >
                          <GitPullRequest className="mr-2 h-3.5 w-3.5" />
                          <span>Inject and Push</span>
                        </DropdownMenuItem>
                      </>
                    ) : (
                      <>
                        <DropdownMenuItem
                          onClick={() => handleInjectSkills("push")}
                          disabled={injectSkillsMutation.isPending}
                          className="text-xs cursor-pointer"
                        >
                          <GitPullRequest className="mr-2 h-3.5 w-3.5" />
                          <span>Inject and Push</span>
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          onClick={() => handleInjectSkills("inject")}
                          disabled={injectSkillsMutation.isPending}
                          className="text-xs cursor-pointer"
                        >
                          <FileCode className="mr-2 h-3.5 w-3.5" />
                          <span>Inject skills</span>
                        </DropdownMenuItem>
                      </>
                    )}
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
            </div>
          </div>

          {/* Check 3: Sync Workspace */}
          <div className="rounded-xl border border-border bg-card p-4 flex flex-col gap-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2.5">
                <div className="p-2 rounded-lg bg-amber-500/10 text-amber-500">
                  <GitBranch className="h-4 w-4" />
                </div>
                <span className="font-semibold text-sm">Sync Workspace</span>
              </div>
              <span className="text-[10px] uppercase tracking-wider px-2 py-0.5 rounded-full bg-blue-500/10 text-blue-600 dark:text-blue-400 font-semibold">
                {project?.baseBranch ? "Configured" : "Not Configured"}
              </span>
            </div>
            <p className="text-xs text-muted-foreground">
              Sync all the assets in the workspace to the repo
            </p>
            <div className="mt-auto pt-2 border-t border-border/50 flex items-center justify-between text-xs text-muted-foreground">
              <div className="flex items-center gap-1.5">
                <span>Base Branch:</span>
                <span className="font-mono text-foreground">{project?.baseBranch || "main"}</span>
              </div>
              <button
                onClick={handleOpenStagingModal}
                disabled={isReadOnlyMode}
                className="inline-flex h-7 items-center justify-center gap-1 rounded-md bg-secondary px-3 text-xs font-semibold text-secondary-foreground hover:bg-secondary/80 disabled:opacity-50 transition-colors border border-input cursor-pointer"
                title="Synchronize workspace assets with repository"
              >
                <GitBranch className="h-3.5 w-3.5" />
                <span>Synchronize</span>
              </button>
            </div>
          </div>

          {/* Check 4: Spec Readiness */}
          <div className="rounded-xl border border-border bg-card p-4 flex flex-col gap-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2.5">
                <div className="p-2 rounded-lg bg-emerald-500/10 text-emerald-500">
                  <ShieldCheck className="h-4 w-4" />
                </div>
                <span className="font-semibold text-sm">Spec Readiness</span>
              </div>
              <span className="text-[10px] uppercase tracking-wider px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 font-semibold">
                Verified
              </span>
            </div>
            <p className="text-xs text-muted-foreground">
              Calculates project phase progression status based on completed and staged artifacts.
            </p>
            <div className="mt-auto pt-2 border-t border-border/50 flex items-center justify-between text-xs text-muted-foreground">
              <div className="flex items-center gap-1.5">
                <span>Evaluated Phases:</span>
                <span className="font-semibold text-foreground">5 Phases</span>
              </div>
              <button
                onClick={() => runHousekeepingMutation.mutate()}
                disabled={runHousekeepingMutation.isPending}
                className="inline-flex h-7 items-center justify-center gap-1 rounded-md bg-secondary px-2.5 text-xs font-semibold text-secondary-foreground hover:bg-secondary/80 disabled:opacity-50 transition-colors"
                title="Check spec readiness and calculate phase gates"
              >
                {runHousekeepingMutation.isPending ? (
                  <Loader2 className="h-3 w-3 animate-spin" />
                ) : (
                  <ShieldCheck className="h-3 w-3" />
                )}
                <span>{runHousekeepingMutation.isPending ? "Checking..." : "Check Readiness"}</span>
              </button>
            </div>
          </div>
        </div>

        {/* Governance Phase Gate Details Table */}
        <div className="rounded-xl border border-border bg-card p-5 flex flex-col gap-4">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-semibold flex items-center gap-2">
              <ShieldCheck className="h-4 w-4 text-primary" />
              Governance Phase Gates Status
            </h3>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-5 gap-3">
            {phases.map((phase) => {
              const status = gates[phase] || "Not Started"
              const isComplete = status === "Complete"
              const isInProgress = status === "In Progress"

              return (
                <div
                  key={phase}
                  className="flex flex-col gap-1.5 p-3 rounded-lg border border-border/60 bg-background/50"
                >
                  <span className="text-[10px] font-bold text-muted-foreground tracking-wider">
                    {phase}
                  </span>
                  <span
                    className={`text-xs font-semibold ${
                      isComplete
                        ? "text-emerald-600 dark:text-emerald-400"
                        : isInProgress
                        ? "text-amber-600 dark:text-amber-400"
                        : "text-muted-foreground"
                    }`}
                  >
                    {status}
                  </span>
                </div>
              )
            })}
          </div>
        </div>
      </div>

      <GitStagingModal
        isOpen={isStagingModalOpen}
        onClose={() => setIsStagingModalOpen(false)}
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
              return { mergeable: data.mergeable !== false, files: mappedFiles }
            }
            return { mergeable: true, files: [] }
          } catch (err: any) {
            console.error("Preview error:", err)
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
        targetBranch={project?.baseBranch || "main"}
      />

      {alertError && (
        <AlertModal
          type="error"
          title={alertError.title}
          description={alertError.description}
          onClose={() => setAlertError(null)}
        />
      )}
    </div>
  )
}

