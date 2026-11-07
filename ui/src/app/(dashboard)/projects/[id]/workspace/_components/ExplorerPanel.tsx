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

import React from "react"
import { 
  ChevronLeft, 
  Plus, 
  MessageSquare, 
  Trash2, 
  Loader2,
  Link,
  ChevronRight,
  FileText,
  Image as ImageIcon,
  Eye
} from "lucide-react"
import { cn, formatSessionDate, formatFileSize } from "@/lib/utils"
import { Session, Artifact, WorkflowTemplate, Project } from "@/lib/types"
import ExplorerSection from "./ExplorerSection"
import WorkflowsControl from "./WorkflowsControl"
import { PhaseType } from "../_utils/workspaceHelpers"
import SkeletonLoader from "./SkeletonLoader"
import FileExplorer, { FileExplorerHandle } from "@/components/workspace/FileExplorer"

function isImageArtifact(art: Artifact): boolean {
  if (art.mimeType && art.mimeType.toLowerCase().startsWith("image/")) return true
  const name = (art.name || art.title || art.path || "").toLowerCase()
  return /\.(png|jpg|jpeg|webp|gif|svg)$/i.test(name)
}

interface ExplorerPanelProps {
  project?: Project | null
  activePhase: PhaseType
  isSidebarOpen: boolean
  setIsSidebarOpen: (open: boolean) => void
  isLocked?: boolean
  isImplementation: boolean
  isSessionsExpanded: boolean
  setIsSessionsExpanded: (expanded: boolean) => void
  isArtifactsExpanded?: boolean
  setIsArtifactsExpanded?: (expanded: boolean) => void
  isFileExplorerExpanded?: boolean
  setIsFileExplorerExpanded?: (expanded: boolean) => void
  isConnectingSession: boolean
  isLoadingSessions?: boolean
  isLoading?: boolean
  sessions: Session[]
  activeSessionId: string | null
  handleCreateNewSession: () => void
  handleSwitchSession: (id: string) => void
  setSessionToDelete: (session: Session | null) => void
  addDebugLog: (msg: string, isError?: boolean) => void
  liveArtifacts: Artifact[]
  bmadWorkflow?: WorkflowTemplate
  activeArtifactTitle: string | null
  setActiveArtifactTitle: (title: string | null) => void
  deleteArtifact: (id: string) => void
  refetchArtifacts?: () => Promise<unknown> | void
  onLinkWorkflows?: () => void
  onUnlinkWorkflow?: (id: string) => void
  linkedWorkflowIds?: string[]
  fileExplorerRef?: React.Ref<FileExplorerHandle>
  onStageForSync?: (paths: string[]) => void
  onBatchDelete?: (items: any[]) => Promise<unknown> | void
}

export default function ExplorerPanel({
  project,
  activePhase,
  isSidebarOpen,
  setIsSidebarOpen,
  isLocked = false,
  isImplementation,
  isSessionsExpanded,
  setIsSessionsExpanded,
  isArtifactsExpanded,
  setIsArtifactsExpanded,
  isFileExplorerExpanded: isFileExplorerExpandedProp,
  setIsFileExplorerExpanded: setIsFileExplorerExpandedProp,
  isConnectingSession,
  isLoadingSessions,
  isLoading,
  sessions,
  activeSessionId,
  handleCreateNewSession,
  handleSwitchSession,
  setSessionToDelete,
  addDebugLog,
  liveArtifacts,
  bmadWorkflow,
  activeArtifactTitle,
  setActiveArtifactTitle,
  deleteArtifact,
  refetchArtifacts,
  onLinkWorkflows,
  onUnlinkWorkflow,
  linkedWorkflowIds,
  fileExplorerRef,
  onStageForSync,
  onBatchDelete,
}: ExplorerPanelProps) {
  const [isWorkflowsExpanded, setIsWorkflowsExpanded] = React.useState(true)
  const [isFileExplorerExpandedInternal, setIsFileExplorerExpandedInternal] = React.useState(true)
  const isFileExplorerExpanded = isFileExplorerExpandedProp ?? isFileExplorerExpandedInternal
  const setIsFileExplorerExpanded = setIsFileExplorerExpandedProp ?? setIsFileExplorerExpandedInternal
  const [showAllFiles, setShowAllFiles] = React.useState(false)
  const [expandedSessionIds, setExpandedSessionIds] = React.useState<Record<string, boolean>>({})

  const toggleSessionExpand = (sessionId: string) => {
    setExpandedSessionIds((prev) => ({
      ...prev,
      [sessionId]: !prev[sessionId],
    }))
  }

  return (
    <div className="h-full flex flex-col overflow-hidden">
      {/* Header containing the title Workspace Explorer */}
      <div className="p-4 border-b border-border select-none shrink-0">
        <div className="flex justify-between items-center">
          <h2 className="text-xs font-bold text-foreground tracking-wider uppercase">
            WORKSPACE EXPLORER
          </h2>
          <button
            onClick={() => setIsSidebarOpen(false)}
            className="flex h-6 w-6 items-center justify-center rounded hover:bg-muted text-muted-foreground"
            title="Collapse Sidebar"
          >
            <ChevronLeft className="h-4 w-4" />
          </button>
        </div>
      </div>

      {/* Main Unified Content Area */}
      <div className="flex-1 overflow-y-auto p-4 space-y-3 select-none">
        {/* Block A: Chat sessions list */}
        <ExplorerSection
          id="tour-workspace-chat-section"
          title="CHAT SESSIONS"
          isExpanded={isSessionsExpanded}
          onToggle={() => setIsSessionsExpanded(!isSessionsExpanded)}
          trailingHeaderWidget={
            <button
              onClick={handleCreateNewSession}
              disabled={isConnectingSession || isLocked}
              className="inline-flex h-5 items-center gap-0.5 text-[9px] font-bold text-primary border border-primary/20 bg-primary/5 px-2 rounded-full hover:bg-primary/10 disabled:opacity-50 transition-colors"
            >
              <Plus className="h-3 w-3" />
              New
            </button>
          }
        >
          {isLoading || isLoadingSessions ? (
            <SkeletonLoader mode="list" count={3} />
          ) : isLocked ? (
            <span className="text-[10px] text-muted-foreground/60 italic block py-1">
              Sessions are unavailable while locked.
            </span>
          ) : isConnectingSession && sessions.length === 0 ? (
            <div className="flex items-center justify-center py-4">
              <Loader2 className="h-4 w-4 animate-spin text-primary" />
            </div>
          ) : sessions.length === 0 ? (
            <span className="text-[10px] text-muted-foreground/60 italic block py-1">
              No active sessions listed.
            </span>
          ) : (
            <div className="space-y-1 max-h-[240px] overflow-y-auto pr-1">
              {sessions.map((s) => {
                const isActive = activeSessionId === s.id
                const sessionDisplayName = s.title || formatSessionDate(s.createdAt)
                const sessionArtifacts = (liveArtifacts || []).filter((art) => art.sessionId === s.id)
                const hasArtifacts = sessionArtifacts.length > 0
                const isExpanded = !!expandedSessionIds[s.id]

                return (
                  <div key={s.id} className="space-y-1">
                    <div
                      className={cn(
                        "group flex items-center justify-between w-full h-8 px-2.5 rounded-md border text-left text-xs transition-colors cursor-pointer select-none",
                        isActive
                          ? "bg-primary/5 text-primary border-primary font-semibold"
                          : "border-border/50 text-muted-foreground hover:bg-muted hover:text-foreground"
                      )}
                      onClick={() => {
                        handleSwitchSession(s.id)
                        if (hasArtifacts) {
                          toggleSessionExpand(s.id)
                        }
                      }}
                      title={sessionDisplayName}
                    >
                      <div className="flex items-center gap-1.5 min-w-0 flex-1">
                        {isActive ? (
                          <MessageSquare className="h-3.5 w-3.5 text-primary fill-primary shrink-0" />
                        ) : (
                          <MessageSquare className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                        )}
                        <span className="truncate flex-1 leading-none">
                          {sessionDisplayName}
                        </span>
                        {s.status === "FAILED" ? (
                          <span className="text-[8px] font-bold bg-destructive/15 text-destructive px-1 rounded-sm shrink-0 flex items-center leading-none" title="Initialization failed (e.g. missing API key)">
                            FAILED
                          </span>
                        ) : s.title ? (
                          <span className="text-[9px] font-normal text-muted-foreground/70 shrink-0 flex items-center leading-none pt-[2px]">
                            {formatSessionDate(s.createdAt)}
                          </span>
                        ) : null}
                      </div>
                      <div className="flex items-center shrink-0 ml-1.5 gap-0.5">
                        {hasArtifacts && (
                          <button
                            type="button"
                            data-testid={`session-chevron-${s.id}`}
                            onClick={(e) => {
                              e.stopPropagation()
                              toggleSessionExpand(s.id)
                            }}
                            className="p-0.5 rounded text-muted-foreground/60 hover:text-foreground hover:bg-muted transition-colors flex items-center justify-center"
                            title={isExpanded ? "Collapse Artifacts" : "Expand Artifacts"}
                            aria-expanded={isExpanded}
                          >
                            <ChevronRight
                              className={cn(
                                "h-3.5 w-3.5 transition-transform duration-200 ease-in-out",
                                isExpanded && "rotate-90"
                              )}
                            />
                          </button>
                        )}
                        <button
                          onClick={(e) => {
                            e.stopPropagation()
                            setSessionToDelete(s)
                          }}
                          className="p-0.5 rounded text-muted-foreground/50 hover:text-destructive hover:bg-destructive/10 transition-colors flex items-center justify-center"
                          title="Delete Session"
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </button>
                      </div>
                    </div>

                    {hasArtifacts && (
                      <div
                        data-testid={`session-artifacts-${s.id}`}
                        className={cn(
                          "overflow-hidden transition-all duration-200 ease-in-out pl-3 pr-1 space-y-0.5",
                          isExpanded ? "max-h-60 opacity-100 py-0.5" : "max-h-0 opacity-0 py-0 pointer-events-none"
                        )}
                      >
                        {sessionArtifacts.map((art, idx) => {
                          const isImg = isImageArtifact(art)
                          const artName = art.name || art.title || art.path || "Untitled Artifact"
                          const isArtActive =
                            activeArtifactTitle &&
                            (activeArtifactTitle === art.name ||
                              activeArtifactTitle === art.title ||
                              activeArtifactTitle === art.path ||
                              activeArtifactTitle === art.id)

                          return (
                            <div
                              key={art.id || idx}
                              data-testid={`session-artifact-item-${art.id || art.name}`}
                              className={cn(
                                "group/art flex items-center justify-between w-full h-7 px-2 rounded-md text-xs transition-colors cursor-pointer select-none",
                                isArtActive
                                  ? "bg-primary/10 text-primary font-medium"
                                  : "text-muted-foreground hover:bg-muted/60 hover:text-foreground"
                              )}
                              onClick={(e) => {
                                e.stopPropagation()
                                const targetTitle = art.name || art.title || art.path || art.id
                                if (targetTitle) {
                                  setActiveArtifactTitle(targetTitle)
                                }
                              }}
                              title={`Preview ${artName}`}
                            >
                              <div className="flex items-center gap-1.5 min-w-0 flex-1">
                                {isImg ? (
                                  <ImageIcon className="h-3.5 w-3.5 text-blue-500 dark:text-blue-400 shrink-0" />
                                ) : (
                                  <FileText className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                                )}
                                <span className="truncate flex-1 text-[11px] leading-none">
                                  {artName}
                                </span>
                                {art.size !== undefined && art.size !== null ? (
                                  <span className="text-[10px] text-muted-foreground/70 shrink-0 font-mono">
                                    ({formatFileSize(art.size)})
                                  </span>
                                ) : null}
                              </div>
                              <button
                                type="button"
                                data-testid={`preview-artifact-${art.id || art.name}`}
                                onClick={(e) => {
                                  e.stopPropagation()
                                  const targetTitle = art.name || art.title || art.path || art.id
                                  if (targetTitle) {
                                    setActiveArtifactTitle(targetTitle)
                                  }
                                }}
                                className="opacity-0 group-hover/art:opacity-100 p-0.5 rounded text-muted-foreground hover:text-primary hover:bg-primary/10 transition-all shrink-0 ml-1 flex items-center justify-center"
                                title="Open in Preview"
                              >
                                <Eye className="h-3 w-3" />
                              </button>
                            </div>
                          )
                        })}
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          )}
        </ExplorerSection>

        {/* Block B: Unified File Explorer with Phase Artifacts */}
        <ExplorerSection
          id="tour-workspace-file-explorer"
          title="FILE EXPLORER"
          isExpanded={isFileExplorerExpanded}
          onToggle={() => setIsFileExplorerExpanded(!isFileExplorerExpanded)}
          trailingHeaderWidget={
            <label
              htmlFor="toggle-show-all-files"
              className="flex items-center gap-1.5 cursor-pointer text-[10px] text-muted-foreground hover:text-foreground select-none"
              title="Show All (Tree View)"
              onClick={(e) => e.stopPropagation()}
            >
              <span className="text-[9px] font-medium leading-none">Show All</span>
              <div className="relative inline-flex items-center">
                <input
                  id="toggle-show-all-files"
                  type="checkbox"
                  role="switch"
                  checked={showAllFiles}
                  onChange={(e) => setShowAllFiles(e.target.checked)}
                  className="sr-only peer"
                  aria-label="Show All"
                />
                <div className="w-7 h-4 bg-muted peer-focus-visible:ring-1 peer-focus-visible:ring-ring rounded-full peer peer-checked:after:translate-x-3 peer-checked:after:border-primary-foreground after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-foreground/70 peer-checked:after:bg-primary-foreground after:rounded-full after:h-3 after:w-3 after:transition-all peer-checked:bg-primary" />
              </div>
            </label>
          }
        >
          <FileExplorer
            ref={fileExplorerRef}
            activePhase={activePhase}
            liveArtifacts={liveArtifacts}
            showAllFiles={showAllFiles}
            onShowAllFilesChange={setShowAllFiles}
            onFileSelect={(file) => {
              if (!file || (!file.path && !file.name)) {
                setActiveArtifactTitle(null)
              } else {
                setActiveArtifactTitle(file.path || file.name)
              }
            }}
            selectedFilePath={activeArtifactTitle}
            onDeleteArtifact={async (artifactId) => {
              deleteArtifact(artifactId)
            }}
            onBatchDelete={onBatchDelete}
            onStageForSync={onStageForSync}
            onRefreshArtifacts={refetchArtifacts}
            isLoadingArtifacts={isLoading}
          />
        </ExplorerSection>

        {/* Block C: Workflows list (only in Implementation phase) */}
        {isImplementation && (
          <ExplorerSection
            id="tour-workspace-workflows"
            title="WORKFLOWS"
            isExpanded={isWorkflowsExpanded}
            onToggle={() => setIsWorkflowsExpanded(!isWorkflowsExpanded)}
            trailingHeaderWidget={
              onLinkWorkflows ? (
                <button
                  onClick={onLinkWorkflows}
                  className="inline-flex h-5 items-center gap-0.5 text-[9px] font-bold text-primary border border-primary/20 bg-primary/5 px-2 rounded-full hover:bg-primary/10 transition-colors"
                  title="Link Workflow"
                >
                  <Link className="h-3 w-3" />
                  Link
                </button>
              ) : undefined
            }
          >
            <WorkflowsControl
              linkedWorkflowIds={linkedWorkflowIds}
              addDebugLog={addDebugLog}
              onUnlink={onUnlinkWorkflow}
            />
          </ExplorerSection>
        )}
      </div>
    </div>
  )
}
