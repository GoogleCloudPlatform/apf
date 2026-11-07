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

import React, { use, useState, useEffect, useRef } from "react"
import { useActivityStore, getPersonaColorClass } from "@/stores/activity"
import { Message, WorkflowTemplate } from "@/lib/types"
import { 
  Loader2, 
  AlertCircle, 
  ArrowLeft, 
  Terminal, 
  GitBranch, 
  X,
  ChevronRight,
  MessageSquare,
  FolderTree
} from "lucide-react"
import { toast } from "sonner"
import { cn, formatSessionDate } from "@/lib/utils"
import ChatPane from "@/components/workspace/ChatPane"
import CanvasPane from "@/components/workspace/CanvasPane"

import {
  ResizablePanelGroup,
  ResizablePanel,
  ResizableHandle,
} from "@/components/ui/resizable"
import { ImperativePanelHandle } from "react-resizable-panels"

import { useWorkspace } from "./_hooks/useWorkspace"
import WorkspaceHeader from "./_components/WorkspaceHeader"
import ExplorerPanel from "./_components/ExplorerPanel"
import DeleteSessionModal from "./_components/DeleteSessionModal"
import WorkflowSelectorModal from "./_components/WorkflowSelectorModal"
import WorkflowRunsHistory from "./_components/WorkflowRunsHistory"
import InteractiveWorkflowViewer from "./_components/InteractiveWorkflowViewer"
import { PhaseType } from "./_utils/workspaceHelpers"

import { api } from "@/lib/api"
import { FileExplorerHandle } from "@/components/workspace/FileExplorer"
import { GitStagingModal, GitFileChange } from "@/components/project/GitStagingModal"
import { useAuthStore } from "@/stores/auth"

interface PageParams {
  id: string
}

export default function WorkspacePage({ params }: { params: Promise<PageParams> }) {
  const { id: projectId } = use(params)
  const fileExplorerRef = useRef<FileExplorerHandle>(null)
  
  const {
    project,
    projectError,
    isLoading,
    activePhase,
    setActivePhase,
    activeArtifactTitle,
    setActiveArtifactTitle,
    chatPaneRef,
    activeSessionId,
    isSidebarOpen,
    setIsSidebarOpen,
    isConnectingSession,
    isAgentInitializing,
    sessionConnectionError,
    isSessionFailed,
    sessionToDelete,
    setSessionToDelete,
    isSessionsExpanded,
    setIsSessionsExpanded,
    isActionsExpanded,
    setIsActionsExpanded,
    isArtifactsExpanded,
    setIsArtifactsExpanded,
    isFileExplorerExpanded,
    setIsFileExplorerExpanded,
    debugLogs,
    isDebugBarOpen,
    setIsDebugBarOpen,
    debugPaneHeight,
    handleDebugResizeStart,
    isDraggingDebug,
    liveArtifacts,
    refetchProject,
    refetchArtifacts,
    bmadWorkflow,
    satisfiedPhases,
    isActivePhaseLocked,
    quickActionsList,
    isImplementation,
    sessions,
    isLoadingSessions,
    connectSessionMutation,
    runHealthCheckMutation,
    deleteArtifactMutation,
    deleteSessionMutation,
    updateSessionTitleMutation,
    addDebugLog,
    handleBack,
    handleEditProject,
    handleHousekeeping,
    handleSwitchSession,
    handleCreateNewSession,
    handleCreateNewSessionForAction,
    handleUpdateSessionTitle,
    activeSession,
    setLatestAgentOutput,
    processAgentOutput,
    handleCrossPhaseSwitch,
  } = useWorkspace(projectId, { fileExplorerRef })

  const { email, displayName } = useAuthStore()
  const sidebarRef = React.useRef<ImperativePanelHandle>(null)
  const [isWorkflowSelectorOpen, setIsWorkflowSelectorOpen] = useState(false)
  const [selectedRunId, setSelectedRunId] = useState<string | null>(null)
  const [selectedRunWorkflowId, setSelectedRunWorkflowId] = useState<string | null>(null)
  const [selectedRunWorkflowName, setSelectedRunWorkflowName] = useState<string>("")
  const [implementationTab, setImplementationTab] = useState<"chat" | "runs">("chat")

  const [isExplorerStagingModalOpen, setIsExplorerStagingModalOpen] = useState(false)
  const [explorerStagedFiles, setExplorerStagedFiles] = useState<GitFileChange[]>([])

  const handleExplorerStageForSync = (paths: string[]) => {
    const fileChanges: GitFileChange[] = paths.map((p) => ({
      path: p,
      status: "modified",
      isChecked: true,
    }))
    setExplorerStagedFiles(fileChanges)
    setIsExplorerStagingModalOpen(true)
  }

  const handleExplorerBatchDelete = async (items: any[]) => {
    for (const item of items) {
      const targetIdOrPath = item.artifactId || item.path || item.name
      if (targetIdOrPath) {
        await deleteArtifactMutation.mutateAsync(targetIdOrPath)
      }
    }
    refetchArtifacts()
    refetchProject()
  }

  const userBranch = email && email.startsWith("specs/")
    ? email
    : (() => {
        const userEmail = email?.includes("@") ? email : displayName?.includes("@") ? displayName : ""
        return userEmail ? `specs/${userEmail.split("@")[0]}` : "specs/user"
      })()

  const handleLinkWorkflow = async (workflow: WorkflowTemplate) => {
    if (!project) return

    const currentLinked = project.linkedWorkflows || []
    if (currentLinked.includes(workflow.id)) {
      toast.info(`Workflow "${workflow.displayName || workflow.name}" is already linked.`)
      return
    }

    const updatedLinked = [...currentLinked, workflow.id]

    const toastId = toast.loading("Linking workflow...")
    try {
      const res = await api.put(`/api/v1/projects/${projectId}`, {
        linkedWorkflows: updatedLinked,
      })

      if (!res.ok) {
        throw new Error(`Failed to link workflow: Status ${res.status}`)
      }

      toast.success(`Linked workflow "${workflow.displayName || workflow.name}" successfully!`, { id: toastId })
      refetchProject()
    } catch (err: any) {
      console.error("Failed to link workflow:", err)
      toast.error(err.message || "Failed to link workflow", { id: toastId })
    }
  }

  const handleUnlinkWorkflow = async (workflowId: string) => {
    if (!project) return

    const currentLinked = project.linkedWorkflows || []
    const updatedLinked = currentLinked.filter((id) => id !== workflowId)

    const toastId = toast.loading("Unlinking workflow...")
    try {
      const res = await api.put(`/api/v1/projects/${projectId}`, {
        linkedWorkflows: updatedLinked,
      })

      if (!res.ok) {
        throw new Error(`Failed to unlink workflow: Status ${res.status}`)
      }

      toast.success("Unlinked workflow successfully!", { id: toastId })
      refetchProject()
    } catch (err: any) {
      console.error("Failed to unlink workflow:", err)
      toast.error(err.message || "Failed to unlink workflow", { id: toastId })
    }
  }

  React.useEffect(() => {
    const sidebar = sidebarRef.current
    if (sidebar) {
      if (isSidebarOpen) {
        sidebar.expand()
      } else {
        sidebar.collapse()
      }
    }
  }, [isSidebarOpen])

  // Automatically collapse/expand the sidebar explorer when the interactive viewer is opened/closed
  React.useEffect(() => {
    if (selectedRunId) {
      setIsSidebarOpen(false)
    } else {
      setIsSidebarOpen(true)
    }
  }, [selectedRunId, setIsSidebarOpen])

  interface NotificationItem extends Message {
    keyId: string
    isFadingOut?: boolean
  }
  const { activities, isPanelOpen } = useActivityStore()
  const [liveNotifications, setLiveNotifications] = useState<NotificationItem[]>([])
  const prevActivitiesLengthRef = useRef(activities.length)

  const pendingQueueRef = useRef<NotificationItem[]>([])
  const isProcessingQueueRef = useRef(false)
  const processingTimerRef = useRef<NodeJS.Timeout | null>(null)
  const liveNotificationsRef = useRef<NotificationItem[]>([])

  useEffect(() => {
    liveNotificationsRef.current = liveNotifications
  }, [liveNotifications])

  const pendingRemovalsRef = useRef<string[]>([])
  const isProcessingRemovalsRef = useRef(false)
  const removalTimerRef = useRef<NodeJS.Timeout | null>(null)

  const processNextRemoval = () => {
    if (pendingRemovalsRef.current.length === 0) {
      isProcessingRemovalsRef.current = false
      return
    }

    isProcessingRemovalsRef.current = true
    const nextKeyId = pendingRemovalsRef.current.shift()
    if (nextKeyId) {
      setLiveNotifications((prev) =>
        prev.map((n) => (n.keyId === nextKeyId ? { ...n, isFadingOut: true } : n))
      )
      setTimeout(() => {
        setLiveNotifications((prev) => prev.filter((n) => n.keyId !== nextKeyId))
      }, 200)
    }

    removalTimerRef.current = setTimeout(() => {
      processNextRemoval()
    }, 500)
  }

  const dismissNotification = (keyId: string) => {
    if (!pendingRemovalsRef.current.includes(keyId)) {
      pendingRemovalsRef.current.push(keyId)
    }
    if (!isProcessingRemovalsRef.current) {
      processNextRemoval()
    }
  }

  const processNextNotification = () => {
    if (pendingQueueRef.current.length === 0) {
      isProcessingQueueRef.current = false
      return
    }

    isProcessingQueueRef.current = true

    const activeNotifications = liveNotificationsRef.current.filter((n) => !n.isFadingOut)

    if (activeNotifications.length >= 4) {
      const oldest = activeNotifications[0]
      dismissNotification(oldest.keyId)

      setTimeout(() => {
        const next = pendingQueueRef.current.shift()
        if (next) {
          setLiveNotifications((prev) => [...prev, next])
        }

        processingTimerRef.current = setTimeout(() => {
          processNextNotification()
        }, 500)
      }, 200)

      return
    }

    const next = pendingQueueRef.current.shift()
    if (next) {
      setLiveNotifications((prev) => [...prev, next])
    }

    processingTimerRef.current = setTimeout(() => {
      processNextNotification()
    }, 500)
  }

  useEffect(() => {
    if (isPanelOpen) {
      setLiveNotifications([])
      pendingQueueRef.current = []
      isProcessingQueueRef.current = false
      if (processingTimerRef.current) {
        clearTimeout(processingTimerRef.current)
        processingTimerRef.current = null
      }
      pendingRemovalsRef.current = []
      isProcessingRemovalsRef.current = false
      if (removalTimerRef.current) {
        clearTimeout(removalTimerRef.current)
        removalTimerRef.current = null
      }
      return
    }

    if (activities.length > prevActivitiesLengthRef.current) {
      const newItems = activities.slice(0, activities.length - prevActivitiesLengthRef.current)
      
      const newNotifications: NotificationItem[] = []
      for (let i = newItems.length - 1; i >= 0; i--) {
        const item = newItems[i]
        if (item && !item.isHistorical) {
          newNotifications.push({ 
            ...item, 
            keyId: `${item.id || Math.random()}-${Date.now()}-${i}` 
          })
        }
      }

      if (newNotifications.length > 0) {
        pendingQueueRef.current = [...pendingQueueRef.current, ...newNotifications]
        if (!isProcessingQueueRef.current) {
          processNextNotification()
        }
      }
    }
    prevActivitiesLengthRef.current = activities.length
  }, [activities, isPanelOpen])

  useEffect(() => {
    return () => {
      if (processingTimerRef.current) {
        clearTimeout(processingTimerRef.current)
      }
      if (removalTimerRef.current) {
        clearTimeout(removalTimerRef.current)
      }
    }
  }, [])



  const [isAgentRunning, setIsAgentRunning] = useState(false)

  const [sidebarPercentages, setSidebarPercentages] = React.useState({
    defaultSize: 20,
    minSize: 10,
  })

  React.useEffect(() => {
    if (typeof window !== "undefined") {
      const width = window.innerWidth
      if (width > 0) {
        const defaultPct = (350 / width) * 100
        const minPct = (150 / width) * 100
        setSidebarPercentages({
          defaultSize: defaultPct,
          minSize: minPct,
        })
      }
    }
  }, [])

  // Poll artifacts list every 5 seconds when the agent is running
  React.useEffect(() => {
    if (!isAgentRunning) {
      return
    }

    const interval = setInterval(() => {
      refetchArtifacts()
    }, 5000)

    return () => clearInterval(interval)
  }, [refetchArtifacts, isAgentRunning])



  // Check if target project is fully loaded
  if (isLoading) {
    return (
      <div className="flex h-screen w-screen flex-col items-center justify-center gap-3 text-muted-foreground bg-slate-50 dark:bg-slate-950 select-none">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
        <span className="text-sm font-medium">Provisioning workspace view...</span>
      </div>
    )
  }

  if (projectError || !project) {
    return (
      <div className="flex h-screen w-screen flex-col items-center justify-center text-center gap-3 bg-slate-50 dark:bg-slate-950 p-6 select-none">
        <AlertCircle className="h-10 w-10 text-destructive animate-bounce" />
        <h2 className="text-lg font-bold text-foreground">Failed to enter workspace</h2>
        <p className="text-xs text-muted-foreground max-w-sm">
          Could not download workspace profile configurations. Make sure the API service is active.
        </p>
        <button
          onClick={handleBack}
          className="flex h-9 items-center gap-2 border border-input bg-background px-4 rounded-lg hover:bg-muted text-xs font-semibold"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to Projects
        </button>
      </div>
    )
  }

  const hasCanvas = !!activeArtifactTitle
  const sidebarPct = isSidebarOpen ? sidebarPercentages.defaultSize : 0
  const canvasPct = hasCanvas ? 45 : 0
  const chatPct = 100 - sidebarPct - canvasPct

  return (
    <div className="flex flex-col h-full bg-slate-50 dark:bg-slate-950 overflow-hidden relative">
      
      <WorkspaceHeader
        projectName={project.name}
        projectIcon={project.icon}
        projectId={projectId}
        projectStatus={project.status}
        activePhase={activePhase}
        setActivePhase={setActivePhase}
        satisfiedPhases={satisfiedPhases}
        bmadWorkflow={bmadWorkflow}
        liveArtifacts={liveArtifacts}
        runHealthCheckPending={runHealthCheckMutation.isPending}
        onRunHealthCheck={handleHousekeeping}
        onShareProject={() => toast.info("Share Project feature coming soon!")}
        onEditProject={handleEditProject}
        handleBack={handleBack}
      />

      {/* ================= MAIN SPLIT VIEW viewport GROUP ================= */}
      <div className="flex-1 w-full h-full overflow-hidden relative">
        <ResizablePanelGroup id="workspace-main-panel-group" direction="horizontal">
          
          {/* COLUMN 1: Explorer Sidebar context pane - Width ~20% */}
          {/* COLUMN 1: Explorer Sidebar context pane - Width ~20% */}
          <ResizablePanel
            ref={sidebarRef}
            collapsible={true}
            defaultSize={sidebarPercentages.defaultSize}
            minSize={sidebarPercentages.minSize}
            onCollapse={() => setIsSidebarOpen(false)}
            onExpand={() => setIsSidebarOpen(true)}
            className={cn(
              "h-full border-r border-border bg-card flex flex-col overflow-hidden transition-all",
              !isSidebarOpen && "max-w-0 border-r-0 shrink-0"
            )}
          >
            <ExplorerPanel
              key={projectId}
              project={project}
              activePhase={activePhase}
              isSidebarOpen={isSidebarOpen}
              setIsSidebarOpen={setIsSidebarOpen}
              isLocked={isActivePhaseLocked}
              isImplementation={isImplementation}
              isSessionsExpanded={isSessionsExpanded}
              setIsSessionsExpanded={setIsSessionsExpanded}
              isArtifactsExpanded={isArtifactsExpanded}
              setIsArtifactsExpanded={setIsArtifactsExpanded}
              isFileExplorerExpanded={isFileExplorerExpanded}
              setIsFileExplorerExpanded={setIsFileExplorerExpanded}
              isConnectingSession={isConnectingSession}
              isLoadingSessions={isLoadingSessions}
              isLoading={isLoading}
              sessions={sessions}
              activeSessionId={activeSessionId}
              handleCreateNewSession={handleCreateNewSession}
              handleSwitchSession={handleSwitchSession}
              setSessionToDelete={setSessionToDelete}
              addDebugLog={addDebugLog}
              liveArtifacts={liveArtifacts}
              bmadWorkflow={bmadWorkflow}
              activeArtifactTitle={activeArtifactTitle}
              setActiveArtifactTitle={setActiveArtifactTitle}
              deleteArtifact={(id) => deleteArtifactMutation.mutate(id)}
              refetchArtifacts={refetchArtifacts}
              onLinkWorkflows={() => setIsWorkflowSelectorOpen(true)}
              onUnlinkWorkflow={handleUnlinkWorkflow}
              linkedWorkflowIds={project?.linkedWorkflows}
              fileExplorerRef={fileExplorerRef}
              onStageForSync={handleExplorerStageForSync}
              onBatchDelete={handleExplorerBatchDelete}
            />
          </ResizablePanel>

          {/* Split drag handler handle bar */}
          {isSidebarOpen && <ResizableHandle withHandle />}

          {/* Closed Sidebar visual drawer strip tab */}
          {!isSidebarOpen && (
            <div className="h-full w-10 border-r border-border bg-card flex flex-col items-center py-4 select-none shrink-0">
              <button
                onClick={() => setIsSidebarOpen(true)}
                className="flex h-7 w-7 items-center justify-center rounded hover:bg-muted text-muted-foreground"
                title="Expand Sidebar"
              >
                <ChevronRight className="h-4 w-4" />
              </button>
            </div>
          )}

          {/* RIGHT SIDE OF WORKSPACE EXPLORER: Full CanvasPane when open, Chat when closed */}
          {activeArtifactTitle ? (
            <ResizablePanel
              id="tour-workspace-canvas"
              defaultSize={100 - sidebarPct}
              minSize={30}
              className="h-full bg-card flex flex-col overflow-hidden"
            >
              <CanvasPane
                projectId={projectId}
                activeArtifactTitle={activeArtifactTitle}
                liveArtifacts={liveArtifacts}
                isReadOnly={project?.isReadOnly}
                onClose={() => setActiveArtifactTitle(null)}
                onSyncCompleted={() => {
                  refetchProject()
                  refetchArtifacts()
                  addDebugLog(`Outcome artifact sync update completed for: "${activeArtifactTitle}"`)
                }}
              />
            </ResizablePanel>
          ) : (
            <ResizablePanel
              defaultSize={100 - sidebarPct}
              minSize={20}
              className="h-full flex flex-col overflow-hidden relative"
            >
              {/* Primary chat views screen */}
              <div id="tour-workspace-chat" className="flex-1 min-h-0 relative flex flex-col overflow-hidden">
                {isImplementation && !selectedRunId && (
                  <div className="flex items-center gap-2 px-4 py-2 border-b border-border bg-muted/20 select-none shrink-0">
                    <button
                      onClick={() => setImplementationTab("chat")}
                      className={cn(
                        "flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-semibold transition-colors",
                        implementationTab === "chat"
                          ? "bg-primary text-primary-foreground"
                          : "text-muted-foreground hover:text-foreground hover:bg-muted"
                      )}
                    >
                      <MessageSquare className="h-3.5 w-3.5" />
                      Chat & Quick Actions
                    </button>
                    <button
                      onClick={() => setImplementationTab("runs")}
                      className={cn(
                        "flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-semibold transition-colors",
                        implementationTab === "runs"
                          ? "bg-primary text-primary-foreground"
                          : "text-muted-foreground hover:text-foreground hover:bg-muted"
                      )}
                    >
                      <FolderTree className="h-3.5 w-3.5" />
                      Workflow Runs
                    </button>
                  </div>
                )}
                {selectedRunId ? (
                  <div className="h-full w-full flex overflow-hidden bg-card">
                    <InteractiveWorkflowViewer
                      projectId={projectId}
                      runId={selectedRunId}
                      workflowId={selectedRunWorkflowId}
                      workflowName={selectedRunWorkflowName}
                      onClose={() => {
                        setSelectedRunId(null)
                        setSelectedRunWorkflowId(null)
                        setSelectedRunWorkflowName("")
                      }}
                      onDeleteRun={(runId) => {
                        if (runId === selectedRunId) {
                          setSelectedRunId(null)
                          setSelectedRunWorkflowId(null)
                          setSelectedRunWorkflowName("")
                        }
                      }}
                    />
                  </div>
                ) : (!isImplementation || implementationTab === "chat") ? (
                  <ChatPane
                    key={`${activePhase}_${activeSessionId || "none"}`}
                    ref={chatPaneRef}
                    projectId={projectId}
                    activePhase={activePhase}
                    sessionId={activeSessionId}
                    sessionTitle={activeSession?.title}
                    sessionCreatedAt={activeSession?.createdAt}
                    onUpdateSessionTitle={(title) => {
                      if (activeSessionId) {
                        handleUpdateSessionTitle(activeSessionId, title)
                      }
                    }}
                    isConnecting={isConnectingSession}
                    connectionError={sessionConnectionError}
                    isAgentInitializing={isAgentInitializing}
                    isSessionFailed={isSessionFailed}
                    isLocked={isActivePhaseLocked}
                    quickActions={quickActionsList}
                    onRetry={() => connectSessionMutation.mutate(false)}
                    onLatestAgentOutput={(output) => {
                      setLatestAgentOutput(output)
                      processAgentOutput(output)
                    }}
                    onAgentInputSent={() => {
                      addDebugLog("User composer payload post submitted.")
                    }}
                    onThinkingStateChange={setIsAgentRunning}
                    onCrossPhaseSwitch={handleCrossPhaseSwitch}
                    onRequestNewSession={handleCreateNewSessionForAction}
                  />
                ) : (
                  <div className="h-full w-full flex overflow-hidden bg-card">
                    <WorkflowRunsHistory
                      projectId={projectId}
                      selectedRunId={selectedRunId}
                      onSelectRun={(runId, wfId, wfName) => {
                        setSelectedRunId(runId)
                        setSelectedRunWorkflowId(wfId)
                        setSelectedRunWorkflowName(wfName)
                      }}
                      onDeleteRun={(runId) => {
                        if (runId === selectedRunId) {
                          setSelectedRunId(null)
                          setSelectedRunWorkflowId(null)
                          setSelectedRunWorkflowName("")
                        }
                      }}
                      compact={false}
                    />
                  </div>
                )}
              </div>

              {/* Draggable resizable bottom console drawers container */}
              {isDebugBarOpen && (
                <div
                  className="border-t border-border bg-card flex flex-col overflow-hidden select-none shrink-0"
                  style={{ height: `${debugPaneHeight}px` }}
                >
                  {/* Horizontal resize handle drag bar */}
                  <div
                    onMouseDown={handleDebugResizeStart}
                    className={cn(
                      "h-1.5 w-full bg-border border-y border-border cursor-ns-resize hover:bg-primary/30 transition-colors",
                      isDraggingDebug ? "bg-primary/40" : ""
                    )}
                  />
                  
                  {/* Console tabs selectors and close button headers */}
                  <div className="h-10 px-4 border-b border-border flex items-center justify-between bg-muted/10 shrink-0">
                    <div className="flex items-center gap-1.5">
                      <span className="text-[10px] font-extrabold tracking-wider text-muted-foreground uppercase mr-3">
                        DEBUG CONSOLE
                      </span>
                      <span className="text-[9px] font-bold border border-destructive/30 bg-destructive/10 text-destructive px-2 py-0.5 rounded-full">
                        Developer Output Logs
                      </span>
                    </div>
                    <button
                      onClick={() => setIsDebugBarOpen(false)}
                      className="flex h-6 w-6 items-center justify-center rounded hover:bg-muted text-muted-foreground"
                      title="Close Drawer"
                    >
                      <X className="h-4 w-4" />
                    </button>
                  </div>

                  {/* Console tabs content logs Scroll Areas */}
                  <div className="flex-1 p-4 font-mono text-[10px] overflow-y-auto bg-slate-950 text-slate-200">
                    {debugLogs.length === 0 ? (
                      <span className="text-slate-500 italic">Console is clean. No background event logs registered yet.</span>
                    ) : (
                      <div className="space-y-1">
                        {debugLogs.map((log, idx) => (
                          <div key={idx} className="whitespace-pre-wrap leading-normal break-words">
                            {log}
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
              )}
            </ResizablePanel>
          )}
        </ResizablePanelGroup>
      </div>
      
      <DeleteSessionModal
        isOpen={!!sessionToDelete}
        onClose={() => setSessionToDelete(null)}
        onConfirm={() => {
          if (sessionToDelete) {
            deleteSessionMutation.mutate(sessionToDelete.id)
            setSessionToDelete(null)
          }
        }}
        sessionDate={sessionToDelete ? (sessionToDelete.title || formatSessionDate(sessionToDelete.createdAt)) : ""}
      />

      <WorkflowSelectorModal
        isOpen={isWorkflowSelectorOpen}
        onClose={() => setIsWorkflowSelectorOpen(false)}
        onLink={handleLinkWorkflow}
        projectName={project?.name}
      />

      <GitStagingModal
        isOpen={isExplorerStagingModalOpen}
        onClose={() => setIsExplorerStagingModalOpen(false)}
        changedFiles={explorerStagedFiles}
        onConfirmSync={async (selectedPaths, actionType, onProgress, direction) => {
          try {
            const endpoint = actionType === 'merge' ? `/api/v1/projects/${projectId}/git/merge` : `/api/v1/projects/${projectId}/git/sync`
            const res = await api.post(endpoint, {
              stagedPaths: selectedPaths,
              action: direction === 'pull' ? 'pull' : actionType,
              direction: direction || 'push',
            })
            if (!res.ok) {
              let errMsg = "Failed to sync changes."
              try {
                const errData = await res.json()
                if (errData?.message) errMsg = errData.message
              } catch (_) {}
              throw new Error(errMsg)
            }
            toast.success(actionType === 'request_review' ? "Review requested successfully!" : "Changes merged and synchronized successfully!")
            refetchProject()
            refetchArtifacts()
          } catch (err: any) {
            console.error("Explorer sync error:", err)
            toast.error(err?.message || "Failed to execute sync operation.")
          }
        }}
        userBranch={userBranch}
        targetBranch="main"
      />

      {/* Live dynamic notification floating bubbles stack */}
      {liveNotifications.length > 0 && (
        <div className="absolute bottom-[76px] right-4 mr-4 mb-4 z-30 flex flex-col gap-2 max-w-[320px] select-none pointer-events-none">
          {liveNotifications.map((notification) => (
            <NotificationBubble
              key={notification.keyId}
              notification={notification}
              onDismiss={() => dismissNotification(notification.keyId)}
            />
          ))}
        </div>
      )}
    </div>
  )
}
export type { PhaseType }
export const dynamicParams = true

interface NotificationBubbleProps {
  notification: {
    keyId: string
    isFadingOut?: boolean
    sender: string
    phase?: string
    text: string
  }
  onDismiss: () => void
}

function NotificationBubble({ notification, onDismiss }: NotificationBubbleProps) {
  useEffect(() => {
    const timer = setTimeout(() => {
      onDismiss()
    }, 3000)
    return () => clearTimeout(timer)
  }, [onDismiss])

  return (
    <div
      className={cn(
        "bg-card border border-border rounded-xl p-3 shadow-xl flex gap-3 pointer-events-auto transition-all duration-150 transform",
        notification.isFadingOut
          ? "opacity-0 scale-95 -translate-y-2"
          : "animate-in slide-in-from-bottom-2 fade-in duration-200"
      )}
    >
      <div className={cn("h-2 w-2 rounded-full mt-1.5 shrink-0", getPersonaColorClass(notification.sender, notification.phase))} />
      <div className="flex flex-col gap-0.5 flex-1 min-w-0">
        <div className="flex justify-between items-center gap-2">
          <span className="text-xs font-bold text-foreground truncate">
            {notification.sender || "System Activity"}
          </span>
          {notification.phase && (
            <span className="text-[8.5px] font-semibold text-muted-foreground uppercase bg-muted/55 px-1 rounded shrink-0">
              {notification.phase}
            </span>
          )}
        </div>
        <p className="text-[10px] text-muted-foreground leading-normal line-clamp-2 break-words">
          {notification.text}
        </p>
      </div>
      <button
        onClick={onDismiss}
        className="text-muted-foreground/60 hover:text-foreground shrink-0 self-start p-0.5 rounded hover:bg-muted/55 transition-colors"
      >
        <X className="h-3 w-3" />
      </button>
    </div>
  )
}
