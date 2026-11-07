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

import { useEffect, useState, useRef, useCallback } from "react"
import { useRouter, useSearchParams, usePathname } from "next/navigation"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import { useAppStore } from "@/stores/app"
import { useActivityStore } from "@/stores/activity"
import { api } from "@/lib/api"
import { 
  Project, 
  WorkflowTemplate, 
  Session, 
  Artifact, 
  isPhaseSatisfied,
  determineArtifactType
} from "@/lib/types"
import { 
  PhaseType, 
  getPhaseStage,
  isArtifactMatchingFilePath,
  isPreviewDocumentMatching,
} from "../_utils/workspaceHelpers"
import { FileExplorerHandle } from "@/components/workspace/FileExplorer"
import { extractFilePathFromFileEvent } from "@/lib/fileEventUtils"

export interface UseWorkspaceOptions {
  fileExplorerRef?: React.RefObject<FileExplorerHandle | null>
}

export function useWorkspace(projectId: string, options: UseWorkspaceOptions = {}) {
  const { fileExplorerRef } = options
  const router = useRouter()
  const searchParams = useSearchParams()
  const pathname = usePathname()
  const queryClient = useQueryClient()
  
  const { setPageTitle, isDebugBarOpen, setIsDebugBarOpen, setSessionId, getSessionId } = useAppStore()
  const { subscribe: subscribeActivity, unsubscribe: unsubscribeActivity } = useActivityStore()

  // URL Deep Links parameters stubs support
  const urlPhase = searchParams.get("phase") as PhaseType | null
  const urlArtifact = searchParams.get("artifact")

  // State parameters
  const [activePhase, setActivePhase] = useState<PhaseType>(urlPhase || "ONBOARDING")
  const [activeArtifactTitle, setActiveArtifactTitle] = useState<string | null>(urlArtifact || null)
  const isInitializedRef = useRef(false)

  // Reset initialization flag when projectId changes
  useEffect(() => {
    isInitializedRef.current = false
  }, [projectId])

  const safeSetActivePhase = (phase: PhaseType) => {
    setActivePhase(phase)
    if (typeof window !== "undefined") {
      const params = new URLSearchParams(window.location.search)
      params.set("phase", phase)
      const newUrl = `${window.location.pathname}?${params.toString()}`
      window.history.replaceState(null, "", newUrl)
    }
  }

  const chatPaneRef = useRef<any>(null)
  const isConnectingRef = useRef(false)
  
  const [phaseSessions, setPhaseSessions] = useState<Record<string, string | null>>(() => {
    const initial: Record<string, string | null> = {}
    const phases: PhaseType[] = ["ONBOARDING", "ANALYSIS", "PLANNING", "SOLUTIONING", "IMPLEMENTATION"]
    for (const phase of phases) {
      const sessId = getSessionId(projectId, phase)
      if (sessId) {
        initial[phase] = sessId
      }
    }
    return initial
  })

  const updateActiveSession = (phase: string, sessionId: string | null) => {
    setPhaseSessions((prev) => ({ ...prev, [phase]: sessionId }))
    setSessionId(projectId, phase, sessionId)
  }
  const activeSessionId = phaseSessions[activePhase] || null

  const [isSidebarOpen, setIsSidebarOpen] = useState(true)
  const [isHelpPanelOpen, setIsHelpPanelOpen] = useState(false)
  const [isConnectingSession, setIsConnectingSession] = useState(false)
  const [sessionConnectionError, setSessionConnectionError] = useState<string | null>(null)
  const [sessionToDelete, setSessionToDelete] = useState<Session | null>(null)

  // Left sidebar sections expansion state parameters
  const [isSessionsExpanded, setIsSessionsExpanded] = useState(true)
  const [isActionsExpanded, setIsActionsExpanded] = useState(true)
  const [isArtifactsExpanded, setIsArtifactsExpanded] = useState(true)
  const [isFileExplorerExpanded, setIsFileExplorerExpanded] = useState(true)

  // Debug drawer state parameters
  const [latestAgentInput, setLatestAgentInput] = useState<string>("")
  const [latestAgentOutput, setLatestAgentOutput] = useState<string>("")
  const [debugLogs, setDebugLogs] = useState<string[]>([])
  const [debugPaneHeight, setDebugPaneHeight] = useState(150)
  const [isDraggingDebug, setIsDraggingDebug] = useState(false)

  // Sync dyn activity context stream on mount
  useEffect(() => {
    setPageTitle("Workspace")
    subscribeActivity(projectId)
    return () => {
      unsubscribeActivity(projectId)
    }
  }, [projectId, subscribeActivity, unsubscribeActivity, setPageTitle])

  // Clear states whenever switching phases
  useEffect(() => {
    setActiveArtifactTitle(null)
    setSessionConnectionError(null)
  }, [activePhase])

  // 1. Query: Get project details
  const {
    data: project,
    isLoading: isLoadingProject,
    error: projectError,
    refetch: refetchProject,
  } = useQuery<Project>({
    queryKey: ["project", projectId],
    queryFn: async () => {
      const res = await api.get(`/api/v1/projects/${projectId}`)
      if (!res.ok) throw new Error("Failed to load project details")
      return res.json()
    },
  })

  // Query: Get project artifacts
  const {
    data: liveArtifactsResult = [],
    isLoading: isLoadingArtifacts,
    refetch: refetchArtifacts,
  } = useQuery<Artifact[]>({
    queryKey: ["artifacts", projectId],
    queryFn: async () => {
      const res = await api.get(`/api/v1/projects/${projectId}/artifacts`)
      if (!res.ok) throw new Error("Failed to load project artifacts")
      const data = await res.json()
      return Array.isArray(data) ? data : []
    },
    enabled: !!projectId,
  })

  const liveArtifacts = liveArtifactsResult || []

  // 2. Query: Get workflow templates
  const {
    data: workflows,
    isLoading: isLoadingWorkflows,
  } = useQuery<WorkflowTemplate[]>({
    queryKey: ["workflows", "BMAD"],
    queryFn: async () => {
      const res = await api.get("/api/v1/workflows?type=BMAD")
      if (!res.ok) throw new Error("Failed to load workflows list")
      return res.json()
    },
  })

  // 3. Query: Get active sessions list for active phase
  const {
    data: sessions = [],
    isLoading: isLoadingSessions,
    isFetching: isFetchingSessions,
    refetch: refetchSessions,
  } = useQuery<Session[]>({
    queryKey: ["sessions", projectId, activePhase],
    queryFn: async () => {
      const res = await api.get(`/api/v1/projects/${projectId}/sessions?phase=${activePhase}`)
      if (!res.ok) throw new Error("Failed to load active sessions list")
      const data = await res.json()
      const list = (Array.isArray(data) ? data : []) as Session[]
      // Sort descending: latest first
      return list.sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime())
    },
    enabled: !!projectId && activePhase !== "IMPLEMENTATION",
  })

  const activeSession = sessions.find((s) => s.id === activeSessionId)
  const isAgentInitializing = activeSession?.status === "INITIALIZING"
  const isSessionFailed = activeSession?.status === "FAILED"

  // Resolve standard active variables
  const bmadWorkflow = workflows?.find((w) => w.framework === "BMAD") || workflows?.[0] || {
    id: "",
    name: "Default",
    framework: "BMAD",
    phases: {},
    artifacts: {},
    dependencies: {},
  } as WorkflowTemplate

  // Check state indicators for our gates tabs
  const satisfiedPhases = {
    ONBOARDING: isPhaseSatisfied(bmadWorkflow, "ONBOARDING", liveArtifacts),
    ANALYSIS: isPhaseSatisfied(bmadWorkflow, "ANALYSIS", liveArtifacts),
    PLANNING: isPhaseSatisfied(bmadWorkflow, "PLANNING", liveArtifacts),
    SOLUTIONING: isPhaseSatisfied(bmadWorkflow, "SOLUTIONING", liveArtifacts),
    IMPLEMENTATION: isPhaseSatisfied(bmadWorkflow, "IMPLEMENTATION", liveArtifacts),
  }

  const activePhaseStage = getPhaseStage(activePhase, satisfiedPhases, bmadWorkflow, liveArtifacts)
  const isActivePhaseLocked = activePhaseStage === "locked"

  // Synchronize url phase parameter with state only on initial load
  useEffect(() => {
    if (project && !isInitializedRef.current) {
      isInitializedRef.current = true
      if (urlPhase) {
        if (activePhase !== urlPhase) {
          setActivePhase(urlPhase)
        }
      } else {
        const defaultPhase = project.status === "Onboarded" ? "ANALYSIS" : "ONBOARDING"
        if (activePhase !== defaultPhase) {
          setActivePhase(defaultPhase)
          if (typeof window !== "undefined") {
            const params = new URLSearchParams(window.location.search)
            params.set("phase", defaultPhase)
            const newUrl = `${window.location.pathname}?${params.toString()}`
            window.history.replaceState(null, "", newUrl)
          }
        }
      }
    }
  }, [project, urlPhase, projectId])

  // 4. Mutation: Create dynamic active session stubs
  const connectSessionMutation = useMutation({
    mutationFn: async (forceNew: boolean = false) => {
      setIsConnectingSession(true)
      setSessionConnectionError(null)
      isConnectingRef.current = true
      
      let url = `/api/v1/projects/${projectId}/sessions`
      if (forceNew) {
        url += "?createNew=true"
      }

      const res = await api.post(url, { phase: activePhase })
      if (!res.ok) {
        let errorData
        try { errorData = await res.json() } catch (_) {}
        throw new Error(errorData?.error || `Failed: Status ${res.status}`)
      }
      return res.json()
    },
    onSuccess: (data) => {
      const session = data?.session
      const sessionId = session?.id
      if (sessionId) {
        queryClient.setQueryData<Session[]>(["sessions", projectId, activePhase], (old = []) => [
          session,
          ...old.filter((s) => s.id !== sessionId),
        ])
        updateActiveSession(activePhase, sessionId)
        queryClient.invalidateQueries({ queryKey: ["sessions", projectId, activePhase] })
        addDebugLog(`Connected to active chat session: ${sessionId}`)
      }
      setIsConnectingSession(false)
      isConnectingRef.current = false
    },
    onError: (err: any) => {
      console.warn("Session connection failed:", err)
      const errMsg = err?.message || ""
      const isGeminiErr = /gemini/i.test(errMsg) || /api_key/i.test(errMsg) || /GEMINI_API_KEY_REQUIRED/.test(errMsg)
      setSessionConnectionError(
        isGeminiErr
          ? "Your Gemini API key is missing or invalid. Please configure a valid key in Project Settings or User Settings."
          : errMsg || "Failed to establish active chat session."
      )
      updateActiveSession(activePhase, null)
      setIsConnectingSession(false)
      isConnectingRef.current = false
      addDebugLog(`Session error: ${err.message}`, true)
    },
  })

  // Auto-connect / Auto-create fallback sessions stubs once list is loaded empty
  useEffect(() => {
    if (
      !project ||
      activePhase === "IMPLEMENTATION" ||
      isActivePhaseLocked ||
      isLoadingSessions ||
      isFetchingSessions ||
      sessionConnectionError
    ) return

    const activeSess = phaseSessions[activePhase]
    const validSessions = sessions.filter(s => s.status !== "FAILED")
    const activeSessExists = activeSess && validSessions.some(s => s.id === activeSess)

    if (validSessions.length === 0) {
      if (activeSess) {
        // Clear the deleted/failed/invalid active session
        updateActiveSession(activePhase, null)
      } else if (!isConnectingSession && !connectSessionMutation.isPending && !isConnectingRef.current && !sessionConnectionError) {
        isConnectingRef.current = true
        addDebugLog(`No valid active sessions found for ${activePhase}. Auto-creating default...`)
        connectSessionMutation.mutate(true)
      }
    } else if (!activeSess || !activeSessExists) {
      // Select the most recent valid session
      updateActiveSession(activePhase, validSessions[0].id)
      addDebugLog(`Active session invalid or missing. Selected most recent valid: ${validSessions[0].id}`)
    }
  }, [project, sessions, activePhase, phaseSessions, isLoadingSessions, isFetchingSessions, isConnectingSession, connectSessionMutation.isPending, sessionConnectionError, isActivePhaseLocked])

  // 5. Mutation: Run Project specific Health Check
  const runHealthCheckMutation = useMutation({
    mutationFn: async () => {
      const res = await api.post(`/api/v1/projects/${projectId}/health-check`, {})
      if (!res.ok) throw new Error("Health check failed")
      return res.json()
    },
    onSuccess: (data) => {
      toast.success("Health check completed! Workspace gates updated.")
      refetchProject()
      refetchArtifacts()
      addDebugLog("Health check diagnostics executed successfully.")
    },
    onError: (err) => {
      console.error("Health check error:", err)
      toast.error("Failed to run project health check.")
    },
  })


  // Mutation: Save / Update artifact status
  const saveArtifactMutation = useMutation({
    mutationFn: async (artifact: Artifact) => {
      const res = await api.put(`/api/v1/projects/${projectId}/artifacts/${artifact.id}`, artifact)
      if (!res.ok) throw new Error(`Failed to save artifact: Status ${res.status}`)
      return res.json()
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["project", projectId] })
      queryClient.invalidateQueries({ queryKey: ["artifacts", projectId] })
    }
  })

  // Mutation: Delete artifact
  const deleteArtifactMutation = useMutation({
    mutationFn: async (artifactId: string) => {
      const res = await api.delete(`/api/v1/projects/${projectId}/artifacts/${artifactId}`)
      if (!res.ok) throw new Error(`Failed to delete artifact: Status ${res.status}`)
      return res.json()
    },
    onSuccess: (_, artifactId) => {
      toast.success("Artifact deleted successfully!")
      queryClient.invalidateQueries({ queryKey: ["project", projectId] })
      queryClient.invalidateQueries({ queryKey: ["artifacts", projectId] })
      
      // If the deleted artifact was currently active in the canvas, close it
      let unescapedId = artifactId
      try {
        unescapedId = decodeURIComponent(artifactId)
      } catch (_) {}

      const deletedArt = liveArtifacts.find(
        (a) =>
          a.id === artifactId ||
          a.id === unescapedId ||
          a.path === artifactId ||
          a.path === unescapedId ||
          a.title === artifactId ||
          a.title === unescapedId
      )

      const normActive = (activeArtifactTitle || "").replace(/\\/g, "/").replace(/^\.?\//, "")
      const normDeletedId = artifactId.replace(/\\/g, "/").replace(/^\.?\//, "")
      const normUnescaped = unescapedId.replace(/\\/g, "/").replace(/^\.?\//, "")

      const isDirectMatch =
        Boolean(activeArtifactTitle) &&
        (normActive === normDeletedId ||
          normActive === normUnescaped ||
          normActive.endsWith("/" + normDeletedId) ||
          normActive.endsWith("/" + normUnescaped) ||
          normDeletedId.endsWith("/" + normActive) ||
          normUnescaped.endsWith("/" + normActive))

      const isArtMatch =
        Boolean(deletedArt) &&
        (activeArtifactTitle === deletedArt!.title ||
          activeArtifactTitle === deletedArt!.path ||
          activeArtifactTitle === deletedArt!.id ||
          (Boolean(deletedArt!.path && activeArtifactTitle && activeArtifactTitle.endsWith(deletedArt!.path))) ||
          (Boolean(deletedArt!.path && activeArtifactTitle && deletedArt!.path.endsWith(activeArtifactTitle))))

      const isDocMatching =
        isPreviewDocumentMatching(activeArtifactTitle, liveArtifacts, artifactId) ||
        isPreviewDocumentMatching(activeArtifactTitle, liveArtifacts, unescapedId)

      if (isDirectMatch || isArtMatch || isDocMatching) {
        setActiveArtifactTitle(null)
      }
      addDebugLog(`Deleted artifact: ${artifactId}`)
    },
    onError: (err: any) => {
      console.error("Delete artifact error:", err)
      toast.error(`Failed to delete artifact: ${err.message || "Unknown error"}`)
      addDebugLog(`Error deleting artifact: ${err.message}`, true)
    }
  })

  // Mutation: Delete chat session
  const deleteSessionMutation = useMutation({
    mutationFn: async (sessionId: string) => {
      const res = await api.delete(`/api/v1/projects/${projectId}/sessions/${sessionId}`)
      if (!res.ok) throw new Error(`Failed to delete session: Status ${res.status}`)
      return res.json()
    },
    onSuccess: (_, sessionId) => {
      toast.success("Chat session deleted successfully!")
      queryClient.invalidateQueries({ queryKey: ["sessions", projectId, activePhase] })
      addDebugLog(`Deleted chat session: ${sessionId}`)
    },
    onError: (err: any) => {
      console.error("Delete session error:", err)
      toast.error(`Failed to delete chat session: ${err.message || "Unknown error"}`)
      addDebugLog(`Error deleting chat session: ${err.message}`, true)
    }
  })

  // Mutation: Update chat session title
  const updateSessionTitleMutation = useMutation({
    mutationFn: async ({ sessionId, title }: { sessionId: string; title: string }) => {
      const res = await api.patch(`/api/v1/projects/${projectId}/sessions/${sessionId}`, { title })
      if (!res.ok) throw new Error(`Failed to update session title: Status ${res.status}`)
      return res.json()
    },
    onMutate: async ({ sessionId, title }) => {
      await queryClient.cancelQueries({ queryKey: ["sessions", projectId, activePhase] })
      const prevSessions = queryClient.getQueryData<Session[]>(["sessions", projectId, activePhase])
      if (prevSessions) {
        queryClient.setQueryData<Session[]>(
          ["sessions", projectId, activePhase],
          prevSessions.map((s) => (s.id === sessionId ? { ...s, title } : s))
        )
      }
      return { prevSessions }
    },
    onError: (err: any, _, context) => {
      if (context?.prevSessions) {
        queryClient.setQueryData(["sessions", projectId, activePhase], context.prevSessions)
      }
      console.error("Update session title error:", err)
      addDebugLog(`Error updating session title: ${err.message}`, true)
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ["sessions", projectId, activePhase] })
    },
  })

  const processAgentOutput = (output: string) => {
    try {
      let decoded: any = null
      try {
        decoded = JSON.parse(output)
      } catch {
        // Output may be plain text or non-JSON string
      }

      // Check for file-write / file-update event using structured and text extraction
      const extractedFilePath = extractFilePathFromFileEvent(decoded || output)
      if (extractedFilePath) {
        addDebugLog(`Found file-write notification for path: ${extractedFilePath}. Refreshing File Explorer and artifacts...`)

        // 1. Refresh matching folder in FileExplorer tree view
        if (fileExplorerRef?.current?.refreshFolderForPath) {
          fileExplorerRef.current.refreshFolderForPath(extractedFilePath).catch((err) => {
            console.error("Failed to refresh folder for path:", extractedFilePath, err)
          })
        }

        // 2. If the preview is open, check if the document being shown in the preview
        // is the same document that UI received an update event for.
        const isPreviewOpen = Boolean(activeArtifactTitle)
        const currentOpenArtifact = isPreviewOpen
          ? liveArtifacts.find(
              (a) =>
                a.title === activeArtifactTitle ||
                a.id === activeArtifactTitle ||
                a.path === activeArtifactTitle
            )
          : null

        const isMatch =
          isPreviewOpen &&
          ((currentOpenArtifact &&
            isArtifactMatchingFilePath(currentOpenArtifact, extractedFilePath)) ||
            isPreviewDocumentMatching(activeArtifactTitle, liveArtifacts, extractedFilePath))

        if (isMatch) {
          addDebugLog(
            `Document in preview pane matches updated file (${extractedFilePath}). Automatically refetching document in preview pane...`
          )
          queryClient.invalidateQueries({ queryKey: ["artifact-content", projectId] })
          queryClient.refetchQueries({ queryKey: ["artifact-content", projectId] })
          if (currentOpenArtifact?.id) {
            queryClient.invalidateQueries({
              queryKey: ["artifact-content", projectId, currentOpenArtifact.id],
            })
            queryClient.refetchQueries({
              queryKey: ["artifact-content", projectId, currentOpenArtifact.id],
            })
          }
        }

        refetchArtifacts()
          .then((result) => {
            const updatedArtifacts = (Array.isArray(result?.data)
              ? result.data
              : []) as Artifact[]
            if (activeArtifactTitle) {
              const refreshedOpenArt = updatedArtifacts.find(
                (a) =>
                  a.title === activeArtifactTitle ||
                  a.id === activeArtifactTitle ||
                  a.path === activeArtifactTitle
              )
              if (
                refreshedOpenArt &&
                (isArtifactMatchingFilePath(refreshedOpenArt, extractedFilePath) || isMatch)
              ) {
                queryClient.invalidateQueries({
                  queryKey: ["artifact-content", projectId, refreshedOpenArt.id],
                })
                queryClient.refetchQueries({
                  queryKey: ["artifact-content", projectId, refreshedOpenArt.id],
                })
              }
            }
          })
          .catch(() => {})

        refetchProject()
      }

      if (decoded && decoded.isNotification === true) {
        const details = decoded.notificationDetails
        if (details) {
          const name = details.name
          const text = decoded.text || ""

          if (name === "debug" || name === "error") {
            addDebugLog(text, name === "error")
            if (name === "error") {
              refetchSessions()
            }
          } else if (name === "agent-init") {
            refetchSessions()
          }
        }
        return
      }
    } catch (e) {
      console.error("Error processing agent output:", e)
    }
  }

  const addDebugLog = (msg: string, isError: boolean = false) => {
    const time = new Date().toLocaleTimeString()
    setDebugLogs((prev) => [...prev, `[${time}] ${isError ? "🚨 ERROR: " : "ℹ️ INFO: "}${msg}`])

    if (isError) {
      chatPaneRef.current?.triggerBackendError(msg)
    }
  }

  const handleBack = () => {
    router.push("/projects")
  }

  const handleEditProject = () => {
    router.push(`/projects/${projectId}/details`)
  }

  const handleHousekeeping = () => {
    router.push(`/projects/${projectId}/housekeeping`)
  }

  const handleSwitchSession = (sessionId: string) => {
    setActiveArtifactTitle(null)
    if (phaseSessions[activePhase] === sessionId) return
    updateActiveSession(activePhase, sessionId)
    addDebugLog(`Switched active session context to: ${sessionId}`)
  }

  const handleCreateNewSession = () => {
    setActiveArtifactTitle(null)
    if (isConnectingSession || isConnectingRef.current) return
    isConnectingRef.current = true
    connectSessionMutation.mutate(true)
  }

  const handleUpdateSessionTitle = (sessionId: string, title: string) => {
    updateSessionTitleMutation.mutate({ sessionId, title })
  }

  const handleCreateNewSessionForAction = useCallback(
    async (
      action: { label: string; prompt: string; id?: string; displayPrompt?: string },
      autoSubmit: boolean = false
    ) => {
      setActiveArtifactTitle(null)
      if (isConnectingSession || isConnectingRef.current) return
      isConnectingRef.current = true
      setIsConnectingSession(true)
      setSessionConnectionError(null)

      try {
        const res = await api.post(`/api/v1/projects/${projectId}/sessions?createNew=true`, {
          phase: activePhase,
        })
        if (!res.ok) {
          let errorData
          try {
            errorData = await res.json()
          } catch (_) {}
          throw new Error(errorData?.error || `Failed: Status ${res.status}`)
        }
        const data = await res.json()
        const newSession = data?.session
        const sessionId = newSession?.id
        if (sessionId) {
          queryClient.setQueryData<Session[]>(["sessions", projectId, activePhase], (old = []) => [
            newSession,
            ...old.filter((s) => s.id !== sessionId),
          ])
          updateActiveSession(activePhase, sessionId)
          queryClient.invalidateQueries({ queryKey: ["sessions", projectId, activePhase] })
          addDebugLog(`Connected to active chat session: ${sessionId}`)

          if (action.label) {
            handleUpdateSessionTitle(sessionId, action.label)
          }

          setTimeout(() => {
            if (autoSubmit) {
              chatPaneRef.current?.submitMessage(action.prompt, action.displayPrompt)
            } else {
              chatPaneRef.current?.injectQuickAction(action)
            }
          }, 150)
        }
      } catch (err: any) {
        console.warn("Session connection failed:", err)
        const errMsg = err?.message || ""
        const isGeminiErr =
          /gemini/i.test(errMsg) || /api_key/i.test(errMsg) || /GEMINI_API_KEY_REQUIRED/.test(errMsg)
        setSessionConnectionError(
          isGeminiErr
            ? "Your Gemini API key is missing or invalid. Please configure a valid key in Project Settings or User Settings."
            : errMsg || "Failed to establish active chat session."
        )
        updateActiveSession(activePhase, null)
        addDebugLog(`Session error: ${err.message}`, true)
      } finally {
        setIsConnectingSession(false)
        isConnectingRef.current = false
      }
    },
    [projectId, activePhase, isConnectingSession, queryClient]
  )

  // Debug Console resize handlers
  const handleDebugResizeStart = (e: React.MouseEvent) => {
    e.preventDefault()
    setIsDraggingDebug(true)
  }

  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (!isDraggingDebug) return
      // Calculate delta movement from window height bounds
      const height = window.innerHeight - e.clientY
      setDebugPaneHeight(Math.max(100, Math.min(height, 500)))
    }

    const handleMouseUp = () => {
      setIsDraggingDebug(false)
    }

    if (isDraggingDebug) {
      window.addEventListener("mousemove", handleMouseMove)
      window.addEventListener("mouseup", handleMouseUp)
    }
    return () => {
      window.removeEventListener("mousemove", handleMouseMove)
      window.removeEventListener("mouseup", handleMouseUp)
    }
  }, [isDraggingDebug])



  const handleCrossPhaseSwitch = useCallback((targetPhase: string, promptToExecute?: string) => {
    const normalizedPhase = targetPhase.toUpperCase() as PhaseType
    safeSetActivePhase(normalizedPhase)

    if (promptToExecute && promptToExecute.trim()) {
      setTimeout(() => {
        chatPaneRef.current?.submitMessage(promptToExecute.trim())
      }, 350)
    }
  }, [safeSetActivePhase, chatPaneRef])

  const rawQuickActionsList = bmadWorkflow?.phases?.[activePhase] || []
  const quickActionsList = activePhase === "IMPLEMENTATION"
    ? rawQuickActionsList.filter(
        (action) => action.id !== "dev-story" && action.id !== "code-review"
      )
    : rawQuickActionsList
  const isImplementation = activePhase === "IMPLEMENTATION"

  const isLoading = isLoadingProject || isLoadingWorkflows || isLoadingArtifacts

  return {
    projectId,
    project,
    projectError,
    isLoading,
    activePhase,
    setActivePhase: safeSetActivePhase,
    activeArtifactTitle,
    setActiveArtifactTitle,
    chatPaneRef,
    phaseSessions,
    activeSessionId,
    activeSession,
    isSidebarOpen,
    setIsSidebarOpen,
    isHelpPanelOpen,
    setIsHelpPanelOpen,
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
    setLatestAgentInput,
    setLatestAgentOutput,
    processAgentOutput,
    handleCrossPhaseSwitch,
  }
}
export type UseWorkspaceResult = ReturnType<typeof useWorkspace>
