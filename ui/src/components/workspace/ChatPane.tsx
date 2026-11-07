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

import { useState, useEffect, useRef, useCallback, forwardRef, useImperativeHandle } from "react"
import ReactMarkdown from "react-markdown"
import remarkGfm from "remark-gfm"
import { 
  Send, 
  Terminal, 
  User, 
  HelpCircle, 
  RefreshCw,
  Sparkles,
  ArrowDown,
  Loader2,
  X,
  Paperclip,
  Square,
  Lock,
  ChevronDown,
  ChevronRight,
  CheckCircle2,
  AlertTriangle,
  Mic,
  MicOff,
  FileText,
  Image as ImageIcon
} from "lucide-react"
import { toast } from "sonner"
import { cn, formatSessionDate, formatFileSize } from "@/lib/utils"
import { useAuthStore } from "@/stores/auth"
import { useActivityStore, getPersonaColorClass } from "@/stores/activity"
import { Message, PhaseSwitchSuggestion, PhaseAction, Artifact } from "@/lib/types"
import { uploadSessionArtifact } from "@/lib/api"
import { useChatStream } from "@/hooks/useChatStream"
import { useSpeechRecognition } from "@/hooks/useSpeechRecognition"
import EngagingLoader from "@/components/shared/EngagingLoader"
import ScrollArea from "@/components/ui/scroll-area"
import AlertModal from "@/components/shared/AlertModal"
import { ThinkingBlock } from "@/components/workspace/ThinkingBlock"
import PhaseSwitchCard from "@/components/workspace/PhaseSwitchCard"
import { phaseConfig, PhaseType } from "@/app/(dashboard)/projects/[id]/workspace/_utils/workspaceHelpers"

export interface StagedFile {
  id: string
  file: File
  name: string
  size: number
  isImage: boolean
  previewUrl?: string
}

const ACCEPTED_FILE_EXTENSIONS = ".txt,.md,.pdf,.csv,.json,.yaml,.yml,.py,.go,.ts,.tsx,.js,.html,.png,.jpg,.jpeg,.webp,.gif,.svg"
const ACCEPTED_EXTENSIONS_SET = new Set(ACCEPTED_FILE_EXTENSIONS.split(",").map((ext) => ext.trim().toLowerCase()))

function isAllowedFileExtension(filename: string): boolean {
  const dotIdx = filename.lastIndexOf(".")
  if (dotIdx === -1 || dotIdx === filename.length - 1) return false
  const ext = filename.substring(dotIdx).toLowerCase()
  return ACCEPTED_EXTENSIONS_SET.has(ext)
}

const MAX_FILE_SIZE_BYTES = 10 * 1024 * 1024 // 10MB

function isImageFile(file: File): boolean {
  const name = (file.name || "").toLowerCase()
  return (file.type && file.type.startsWith("image/")) || /\.(png|jpg|jpeg|webp|gif|svg)$/i.test(name)
}

const formatQuickActionLabel = (actionId: string): string => {
  if (!actionId) return ""
  if (actionId === "bmad-help") return "Help"
  return actionId
    .split("-")
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ")
}

const getQuickActionLabel = (action: PhaseAction): string => {
  if (
    action.id === "bmad-help" ||
    action.name === "BMad Help" ||
    action.name === "BMadHelp" ||
    action.name === "Bmad Help" ||
    action.name === "bmad-help" ||
    action.name === "Help"
  ) {
    return "Help"
  }
  return action.name || formatQuickActionLabel(action.id)
}

export interface ChatPaneEndpointConfig {
  streamEndpoint?: string
  sendMessageEndpoint?:
    | string
    | ((text: string, defaultPayload: Record<string, any>) => { endpoint: string; payload?: Record<string, any> })
  customSendMessage?: (text: string) => Promise<boolean>
  additionalPayload?: Record<string, any>
}

export interface ChatPaneProps {
  projectId: string
  activePhase: string
  sessionId: string | null
  sessionTitle?: string
  sessionCreatedAt?: string
  onUpdateSessionTitle?: (title: string) => void
  isConnecting: boolean
  connectionError: string | null
  isAgentInitializing?: boolean
  isSessionFailed?: boolean
  isLocked?: boolean
  quickActions?: PhaseAction[]
  onRetry: () => void
  onLatestAgentOutput?: (output: string) => void
  onAgentInputSent?: (input: string) => void
  onThinkingStateChange?: (isThinking: boolean) => void
  onCrossPhaseSwitch?: (targetPhase: string, promptToExecute?: string) => void
  onRequestNewSession?: (
    action: { label: string; prompt: string; id?: string; displayPrompt?: string },
    autoSubmit?: boolean
  ) => void
  streamEndpoint?: string
  sendMessageEndpoint?:
    | string
    | ((text: string, defaultPayload: Record<string, any>) => { endpoint: string; payload?: Record<string, any> })
  customSendMessage?: (text: string) => Promise<boolean>
  additionalPayload?: Record<string, any>
  endpointConfig?: ChatPaneEndpointConfig
}

export interface QuickActionPill {
  id?: string
  label: string
  prompt: string
  displayPrompt?: string
}

export interface ChatPaneRef {
  injectMessage: (message: string) => void
  injectQuickAction: (action: { label: string; prompt: string; id?: string; displayPrompt?: string }) => void
  submitMessage: (message: string, displayPrompt?: string) => void
  triggerBackendError: (message: string) => void
}

export const ChatPane = forwardRef<ChatPaneRef, ChatPaneProps>(({
  projectId,
  activePhase,
  sessionId,
  sessionTitle,
  sessionCreatedAt,
  onUpdateSessionTitle,
  isConnecting: isSessionConnecting,
  connectionError: sessionConnectionError,
  isAgentInitializing = false,
  isSessionFailed = false,
  isLocked = false,
  quickActions,
  onRetry,
  onLatestAgentOutput,
  onAgentInputSent,
  onThinkingStateChange,
  onCrossPhaseSwitch,
  onRequestNewSession,
  streamEndpoint,
  sendMessageEndpoint,
  customSendMessage,
  additionalPayload,
  endpointConfig,
}, ref) => {
  const [inputText, setInputText] = useState("")
  const [selectedQuickAction, setSelectedQuickAction] = useState<QuickActionPill | null>(null)
  const [isAnimationCancelled, setIsAnimationCancelled] = useState(false)
  const [isHistoryExpanded, setIsHistoryExpanded] = useState(false)
  const [stagedFiles, setStagedFiles] = useState<StagedFile[]>([])
  const [isUploading, setIsUploading] = useState(false)
  const [isDraggingOver, setIsDraggingOver] = useState(false)
  const inputRef = useRef<HTMLTextAreaElement>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const chatContainerRef = useRef<HTMLDivElement>(null)
  const shouldRefocusRef = useRef(false)
  const prevThinkingRef = useRef(false)
  const shouldFocusWhenReadyRef = useRef(false)
  const isUserScrolledUpRef = useRef(false)

  const stagedFilesRef = useRef<StagedFile[]>(stagedFiles)
  stagedFilesRef.current = stagedFiles

  const stageFiles = useCallback((files: FileList | File[]) => {
    const newStaged: StagedFile[] = []
    for (const file of Array.from(files)) {
      if (file.size > MAX_FILE_SIZE_BYTES) {
        toast.error("File exceeds 10MB limit")
        continue
      }
      if (!isAllowedFileExtension(file.name)) {
        toast.error(`Unsupported file type: ${file.name}`)
        continue
      }
      const isImg = isImageFile(file)
      const previewUrl = isImg ? URL.createObjectURL(file) : undefined
      newStaged.push({
        id: `${file.name}-${file.size}-${Date.now()}-${Math.random().toString(36).substring(2, 7)}`,
        file,
        name: file.name,
        size: file.size,
        isImage: isImg,
        previewUrl,
      })
    }
    if (newStaged.length > 0) {
      setStagedFiles((prev) => [...prev, ...newStaged])
    }
  }, [])

  const removeStagedFile = useCallback((fileId: string) => {
    setStagedFiles((prev) => {
      const target = prev.find((f) => f.id === fileId)
      if (target?.previewUrl) {
        URL.revokeObjectURL(target.previewUrl)
      }
      return prev.filter((f) => f.id !== fileId)
    })
  }, [])

  // Reset staged files on session or project switch
  useEffect(() => {
    setStagedFiles((prev) => {
      prev.forEach((sf) => {
        if (sf.previewUrl) {
          URL.revokeObjectURL(sf.previewUrl)
        }
      })
      return []
    })
  }, [sessionId, projectId])

  const isMountedRef = useRef(true)

  // Clean up object URLs on component unmount
  useEffect(() => {
    isMountedRef.current = true
    return () => {
      isMountedRef.current = false
      stagedFilesRef.current.forEach((sf) => {
        if (sf.previewUrl) {
          URL.revokeObjectURL(sf.previewUrl)
        }
      })
    }
  }, [])

  // Auto-resize textarea height up to 6 lines max
  useEffect(() => {
    const textarea = inputRef.current
    if (!textarea) return

    textarea.style.height = "auto"

    const computed = window.getComputedStyle(textarea)
    let lineHeight = parseFloat(computed.lineHeight)
    if (isNaN(lineHeight) || lineHeight <= 0) {
      const fontSize = parseFloat(computed.fontSize) || 14
      lineHeight = fontSize * 1.4
    }

    const paddingTop = parseFloat(computed.paddingTop) || 8
    const paddingBottom = parseFloat(computed.paddingBottom) || 8
    const borderTop = parseFloat(computed.borderTopWidth) || 1
    const borderBottom = parseFloat(computed.borderBottomWidth) || 1

    const maxHeight = lineHeight * 6 + paddingTop + paddingBottom + borderTop + borderBottom
    const newHeight = Math.min(textarea.scrollHeight, maxHeight)

    textarea.style.height = `${newHeight}px`
  }, [inputText])



  useImperativeHandle(ref, () => ({
    injectMessage: (message: string) => {
      setInputText(message)
      inputRef.current?.focus()
    },
    injectQuickAction: (action: { label: string; prompt: string; id?: string; displayPrompt?: string }) => {
      setSelectedQuickAction({
        label: action.label,
        prompt: action.prompt,
        id: action.id,
        displayPrompt: action.displayPrompt,
      })
      inputRef.current?.focus()
    },
    submitMessage: (message: string, displayPrompt?: string) => {
      if (message && message.trim()) {
        const trimmedMessage = message.trim()
        const trimmedDisplayPrompt = displayPrompt?.trim()
        if (sessionId && !sessionTitle && onUpdateSessionTitle) {
          const display = trimmedDisplayPrompt || trimmedMessage
          const words = display.replace(/\s+/g, " ").trim().split(" ").filter(Boolean)
          const newTitle = words.slice(0, 6).join(" ")
          if (newTitle) {
            onUpdateSessionTitle(newTitle)
          }
        }
        if (trimmedDisplayPrompt && trimmedDisplayPrompt !== trimmedMessage) {
          sendMessage(trimmedMessage, trimmedDisplayPrompt)
        } else {
          sendMessage(trimmedMessage)
        }
      }
    },
    triggerBackendError: (message: string) => {
      triggerError({ title: "System Error", description: message })
    }
  }))
  const scrollAreaViewportRef = useRef<HTMLDivElement>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const [showScrollButton, setShowScrollButton] = useState(false)
  const [showLoader, setShowLoader] = useState(false)

  // Subscribes to dynamic SSE messages stream
  const {
    messages,
    isLoading: isStreamLoading,
    isThinking,
    latestThought,
    thinkingSteps,
    error: streamError,
    sendMessage,
    cancelMessage,
    retryConnection,
    triggerError,
  } = useChatStream(projectId, activePhase, sessionId, {
    onLatestAgentOutput,
    onAgentInputSent,
    streamEndpoint: streamEndpoint || endpointConfig?.streamEndpoint,
    sendMessageEndpoint: sendMessageEndpoint || endpointConfig?.sendMessageEndpoint,
    customSendMessage: customSendMessage || endpointConfig?.customSendMessage,
    additionalPayload: {
      ...(endpointConfig?.additionalPayload || {}),
      ...(additionalPayload || {}),
    },
  })

  const baseTextRef = useRef("")
  const inputTextRef = useRef(inputText)

  useEffect(() => {
    inputTextRef.current = inputText
  }, [inputText])

  const {
    isListening,
    isSupported: isSpeechSupported,
    startListening: startVoiceCapture,
    stopListening: stopVoiceCapture,
  } = useSpeechRecognition({
    onResult: (spokenText) => {
      const base = baseTextRef.current.trim()
      const combined = base ? `${base} ${spokenText}` : spokenText
      setInputText(combined)
    },
    onError: (errMessage) => {
      triggerError({ title: "Voice Input Error", description: errMessage })
    },
  })

  const toggleVoiceInput = useCallback(() => {
    if (isThinking || isSessionConnecting || isAgentInitializing || isSessionFailed || !isSpeechSupported) {
      return
    }
    if (isListening) {
      stopVoiceCapture()
    } else {
      baseTextRef.current = inputTextRef.current
      startVoiceCapture()
      inputRef.current?.focus()
    }
  }, [isThinking, isSessionConnecting, isAgentInitializing, isSessionFailed, isSpeechSupported, isListening, stopVoiceCapture, startVoiceCapture])

  // Global keyboard shortcut to toggle voice input: Alt+V or Ctrl/Cmd+Shift+M
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      const isAltV = e.altKey && !e.ctrlKey && !e.metaKey && e.key.toLowerCase() === "v"
      const isCtrlShiftM = (e.ctrlKey || e.metaKey) && e.shiftKey && e.key.toLowerCase() === "m"
      const isEscapeWhileListening = e.key === "Escape" && isListening

      if (isAltV || isCtrlShiftM) {
        e.preventDefault()
        toggleVoiceInput()
      } else if (isEscapeWhileListening) {
        e.preventDefault()
        stopVoiceCapture()
      }
    }

    window.addEventListener("keydown", handleKeyDown)
    return () => window.removeEventListener("keydown", handleKeyDown)
  }, [toggleVoiceInput, isListening, stopVoiceCapture])

  // Bubble up thinking state updates to parent view containers
  useEffect(() => {
    if (onThinkingStateChange) {
      onThinkingStateChange(isThinking)
    }
  }, [isThinking, onThinkingStateChange])

  // Set a brief 200ms delay to avoid loader flashing on fast connections
  useEffect(() => {
    const needsLoader = isSessionConnecting || (isStreamLoading && messages.length === 0)
    if (!needsLoader) {
      setShowLoader(false)
      return
    }

    const timer = setTimeout(() => {
      setShowLoader(true)
    }, 200)

    return () => clearTimeout(timer)
  }, [isSessionConnecting, isStreamLoading, messages.length])

  // Auto scroll down to bottom helper
  const scrollToBottom = (behavior: ScrollBehavior = "smooth", force = false) => {
    if (!force && isUserScrolledUpRef.current) return

    const executeScroll = () => {
      if (scrollAreaViewportRef.current) {
        const viewport = scrollAreaViewportRef.current
        if (typeof viewport.scrollTo === "function") {
          viewport.scrollTo({ top: viewport.scrollHeight, behavior: "smooth" })
        } else {
          viewport.scrollTop = viewport.scrollHeight
        }
      }
      messagesEndRef.current?.scrollIntoView({ behavior })
    }

    executeScroll()
    requestAnimationFrame(() => {
      executeScroll()
    })
  }

  const handleScrollBottom = () => {
    scrollToBottom("auto")
  }

  // Auto-scroll on initial load or incoming message chunks
  useEffect(() => {
    scrollToBottom("auto")
  }, [messages, isThinking])

  // Auto-scroll when thinking steps or latest thought stream in
  useEffect(() => {
    if (isThinking || (thinkingSteps && thinkingSteps.length > 0) || latestThought) {
      scrollToBottom("auto")
    }
  }, [thinkingSteps, latestThought, isThinking])

  // Track thinking state transitions to prepare focus restoration
  useEffect(() => {
    const wasThinking = prevThinkingRef.current
    prevThinkingRef.current = isThinking

    if (wasThinking && !isThinking && shouldRefocusRef.current) {
      shouldRefocusRef.current = false
      shouldFocusWhenReadyRef.current = true
    }
  }, [isThinking])

  // Execute focus when all conditions are met (not thinking, not connecting, and flag is set)
  useEffect(() => {
    if (shouldFocusWhenReadyRef.current && !isThinking && !isSessionConnecting) {
      shouldFocusWhenReadyRef.current = false
      
      const activeEl = document.activeElement
      const isFocusLostOrInChat = 
        !activeEl || 
        activeEl === document.body || 
        activeEl.tagName === "BODY" ||
        chatContainerRef.current?.contains(activeEl)

      if (isFocusLostOrInChat && inputRef.current) {
        inputRef.current.focus()
      }
    }
  }, [isThinking, isSessionConnecting])

  // Detect scroll status to display custom floating "scroll down" button
  const handleScroll = (e: React.UIEvent<HTMLDivElement>) => {
    const target = e.currentTarget
    const threshold = 150
    const isNearBottom = 
      target.scrollHeight - target.scrollTop - target.clientHeight < threshold
    isUserScrolledUpRef.current = !isNearBottom
    setShowScrollButton(!isNearBottom && target.scrollHeight > target.clientHeight)
  }

  const handleSend = async (e?: React.FormEvent) => {
    if (e) e.preventDefault()
    const text = inputText.trim()
    const hasAttachments = stagedFiles.length > 0
    if ((!text && !selectedQuickAction && !hasAttachments) || isThinking || isUploading) return

    if (isListening) {
      stopVoiceCapture()
    }

    const currentQuickAction = selectedQuickAction
    let promptToSend = text
    let displayPromptToShow = text

    if (currentQuickAction) {
      promptToSend = text ? `${currentQuickAction.prompt} ${text}` : currentQuickAction.prompt
      if (currentQuickAction.displayPrompt) {
        displayPromptToShow = text ? `${currentQuickAction.displayPrompt} ${text}` : currentQuickAction.displayPrompt
      } else {
        displayPromptToShow = promptToSend
      }
    }

    if (!promptToSend && hasAttachments) {
      promptToSend = "Uploaded attachment(s)"
      displayPromptToShow = promptToSend
    }

    // If the session is in default state (no explicit title yet), assign name on first message
    if (sessionId && !sessionTitle && onUpdateSessionTitle) {
      let newTitle = ""
      if (currentQuickAction?.label) {
        newTitle = currentQuickAction.label
      } else if (displayPromptToShow) {
        const words = displayPromptToShow.replace(/\s+/g, " ").trim().split(" ").filter(Boolean)
        newTitle = words.slice(0, 6).join(" ")
      } else if (text) {
        const words = text.replace(/\s+/g, " ").trim().split(" ").filter(Boolean)
        newTitle = words.slice(0, 6).join(" ")
      }
      if (newTitle) {
        onUpdateSessionTitle(newTitle)
      }
    }

    // Upload staged files if any
    let uploadedArtifacts: Artifact[] = []
    if (stagedFiles.length > 0) {
      if (!projectId || !sessionId) {
        toast.error("Missing project or session ID to upload attachments")
        return
      }
      setIsUploading(true)
      try {
        uploadedArtifacts = await Promise.all(
          stagedFiles.map((sf) => uploadSessionArtifact(projectId, sessionId, sf.file))
        )
      } catch (err: any) {
        if (isMountedRef.current) {
          setIsUploading(false)
          toast.error(err.message || "Failed to upload file(s)")
        }
        return // Abort message send, keep staged files preserved
      }
      if (isMountedRef.current) {
        setIsUploading(false)
      }
    }

    // Revoke object URLs and clear staged files
    stagedFiles.forEach((sf) => {
      if (sf.previewUrl) URL.revokeObjectURL(sf.previewUrl)
    })
    setStagedFiles([])

    setInputText("")
    setSelectedQuickAction(null)
    isUserScrolledUpRef.current = false
    scrollToBottom("auto", true)
    
    // Evaluate if the focus was currently inside the chat form before accepting the send action
    const activeEl = document.activeElement
    const wasFocusedInChatForm = !!(activeEl && inputRef.current?.form?.contains(activeEl))

    const success = (displayPromptToShow && displayPromptToShow !== promptToSend)
      ? (uploadedArtifacts.length > 0
          ? await sendMessage(promptToSend, displayPromptToShow, uploadedArtifacts)
          : await sendMessage(promptToSend, displayPromptToShow))
      : (uploadedArtifacts.length > 0
          ? await sendMessage(promptToSend, undefined, uploadedArtifacts)
          : await sendMessage(promptToSend))

    if (success) {
      if (wasFocusedInChatForm) {
        shouldRefocusRef.current = true
      }
    } else {
      // Restore input text and quick action if delivery failed
      setInputText(text)
      if (currentQuickAction) {
        setSelectedQuickAction(currentQuickAction)
      }
    }
  }

  const handleQuickActionClick = (action: PhaseAction) => {
    const label = getQuickActionLabel(action)
    const injectionTarget = action.prompt || `bmad-${action.id}`

    // If session already has conversation history, launch a fresh session for this action
    if (messages.length > 0 && onRequestNewSession) {
      onRequestNewSession({
        label,
        prompt: injectionTarget,
        id: action.id,
        displayPrompt: action.displayPrompt,
      }, false)
      return
    }

    if (selectedQuickAction?.id === action.id || selectedQuickAction?.label === label) {
      setSelectedQuickAction(null)
    } else {
      setSelectedQuickAction({
        label,
        prompt: injectionTarget,
        id: action.id,
        displayPrompt: action.displayPrompt,
      })
    }
    inputRef.current?.focus()
  }

  const handleActionChipClick = (actionText: string) => {
    const chipLabel = actionText.startsWith("bmad-")
      ? actionText.substring(5).split("-").map((w) => w.charAt(0).toUpperCase() + w.slice(1)).join(" ")
      : actionText
    const matchedAction = quickActions?.find(
      (a) => a.id === actionText || a.prompt === actionText || `bmad-${a.id}` === actionText || a.name === chipLabel
    )
    const actionPayload = {
      label: chipLabel,
      prompt: matchedAction?.prompt || actionText,
      id: matchedAction?.id,
      displayPrompt: matchedAction?.displayPrompt,
    }

    // If session already has conversation history, launch a fresh session and submit the chip prompt
    if (messages.length > 0 && onRequestNewSession) {
      onRequestNewSession(actionPayload, true)
      return
    }

    setSelectedQuickAction(actionPayload)
    inputRef.current?.focus()
  }

  const formattedPhase = activePhase
    ? activePhase.toLowerCase()
    : "workspace"

  const upperPhase = (activePhase || "").toUpperCase() as PhaseType
  const PhaseIcon = phaseConfig[upperPhase]?.icon

  const isInitialEmptyState =
    !!sessionId &&
    !isSessionConnecting &&
    !isStreamLoading &&
    messages.length === 0 &&
    !isThinking &&
    !showLoader &&
    !sessionConnectionError &&
    !isLocked

  const isLoading = isStreamLoading && messages.length === 0
  const activeError = sessionId ? streamError : null

  const renderComposer = (variant: "bottom" | "centered" = "bottom") => {
    const isCentered = variant === "centered"

    return (
      <form
        onSubmit={handleSend}
        onDragOver={(e) => {
          e.preventDefault()
          e.stopPropagation()
          if (isThinking || isUploading || isSessionConnecting || isAgentInitializing || isSessionFailed) return
          setIsDraggingOver(true)
        }}
        onDragEnter={(e) => {
          e.preventDefault()
          e.stopPropagation()
          if (isThinking || isUploading || isSessionConnecting || isAgentInitializing || isSessionFailed) return
          setIsDraggingOver(true)
        }}
        onDragLeave={(e) => {
          e.preventDefault()
          e.stopPropagation()
          if (!e.currentTarget.contains(e.relatedTarget as Node)) {
            setIsDraggingOver(false)
          }
        }}
        onDrop={(e) => {
          e.preventDefault()
          e.stopPropagation()
          setIsDraggingOver(false)
          if (isThinking || isUploading || isSessionConnecting || isAgentInitializing || isSessionFailed) return
          if (e.dataTransfer.files && e.dataTransfer.files.length > 0) {
            stageFiles(e.dataTransfer.files)
          }
        }}
        className={cn(
          "flex flex-col select-none transition-colors",
          isCentered ? "gap-3" : "p-4 gap-2.5",
          isDraggingOver && "border-2 border-dashed border-primary/60 bg-primary/5 rounded-xl"
        )}
      >
        <input
          ref={fileInputRef}
          type="file"
          multiple
          accept={ACCEPTED_FILE_EXTENSIONS}
          className="hidden"
          data-testid="chat-file-input"
          onChange={(e) => {
            if (e.target.files && e.target.files.length > 0) {
              stageFiles(e.target.files)
              e.target.value = ""
            }
          }}
        />

        {/* Live Voice Listening Indicator Banner */}
        {isListening && (
          <div className="flex items-center justify-between px-4 py-2 bg-red-500/10 border-b border-red-500/30 text-xs text-red-600 dark:text-red-400 animate-pulse select-none rounded-lg">
            <span className="flex items-center gap-2 font-medium">
              <span className="h-2 w-2 rounded-full bg-red-500 animate-ping" />
              Listening... Speak clearly into your microphone
            </span>
            <button
              type="button"
              onClick={stopVoiceCapture}
              className="text-xs underline hover:opacity-80 font-medium"
            >
              Stop (ESC / Alt+V)
            </button>
          </div>
        )}

        {/* Staged File Attachment Badges */}
        {stagedFiles.length > 0 && (
          <div className="flex flex-wrap gap-2 px-1" data-testid="staged-attachments-list">
            {stagedFiles.map((sf) => (
              <div
                key={sf.id}
                data-testid={`staged-chip-${sf.name}`}
                className="inline-flex items-center gap-2 rounded-lg border border-border bg-card px-2.5 py-1 text-xs shadow-xs animate-fade-in"
              >
                {sf.isImage && sf.previewUrl ? (
                  <img
                    src={sf.previewUrl}
                    alt={sf.name}
                    className="h-6 w-6 rounded object-cover border border-border shrink-0"
                  />
                ) : (
                  <FileText className="h-4 w-4 text-primary shrink-0" />
                )}
                <span className="max-w-[150px] truncate font-medium text-foreground" title={sf.name}>
                  {sf.name}
                </span>
                <span className="text-[10px] text-muted-foreground">
                  ({formatFileSize(sf.size)})
                </span>
                <button
                  type="button"
                  disabled={isUploading}
                  onClick={() => removeStagedFile(sf.id)}
                  className="ml-0.5 rounded-full p-0.5 text-muted-foreground hover:bg-muted hover:text-foreground disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                  title={`Remove ${sf.name}`}
                  aria-label={`Remove ${sf.name}`}
                >
                  <X className="h-3.5 w-3.5" />
                </button>
              </div>
            ))}
          </div>
        )}

        <div className="flex gap-3 items-end w-full">
          <button
            type="button"
            id="tour-chat-file-upload-button"
            onClick={() => fileInputRef.current?.click()}
            disabled={isThinking || isUploading || isSessionConnecting || isAgentInitializing || isSessionFailed}
            className={cn(
              "flex h-10 w-10 shrink-0 items-center justify-center rounded-lg border border-input bg-card text-muted-foreground hover:bg-muted hover:text-foreground shadow-sm transition-colors mb-[1px]",
              (isThinking || isUploading || isSessionConnecting || isAgentInitializing || isSessionFailed) && "opacity-50 cursor-not-allowed"
            )}
            title="Attach files"
            aria-label="Attach files"
          >
            <Paperclip className="h-4.5 w-4.5" />
          </button>
          <div
            className={cn(
              "flex-1 min-h-[40px] px-3.5 py-2 rounded-lg border border-input bg-card text-sm leading-normal focus-within:ring-2 focus-within:ring-primary/40 focus-within:border-primary flex flex-row items-center gap-2 transition-all",
              (isThinking || isUploading || isSessionConnecting || isAgentInitializing || isSessionFailed) && "opacity-50"
            )}
          >
            {selectedQuickAction && (
              <span className="inline-flex items-center gap-1.5 text-xs font-semibold bg-primary/10 text-primary border border-primary/20 px-2.5 py-0.5 rounded-full shadow-sm select-none shrink-0">
                <Sparkles className="h-3 w-3 shrink-0 text-primary" />
                <span>{selectedQuickAction.label}</span>
                <button
                  type="button"
                  onClick={() => {
                    setSelectedQuickAction(null)
                    inputRef.current?.focus()
                  }}
                  className="ml-0.5 rounded-full p-0.5 hover:bg-primary/20 text-primary/80 hover:text-primary transition-colors"
                  title="Remove quick action"
                  aria-label="Remove quick action"
                >
                  <X className="h-3 w-3" />
                </button>
              </span>
            )}
            <textarea
              ref={inputRef}
              rows={1}
              placeholder={
                isAgentInitializing 
                  ? "Agent is initializing..." 
                  : isSessionFailed 
                    ? "Agent initialization failed." 
                    : isThinking 
                      ? "Agent is processing request..." 
                      : isUploading
                        ? "Uploading attachments..."
                        : isListening
                          ? "Listening to your voice..."
                          : selectedQuickAction
                            ? "Add extra details (optional) or press Send..."
                            : "Ask a question, enter a command, or describe a goal..."
              }
              value={inputText}
              onChange={(e) => setInputText(e.target.value)}
              onFocus={() => {
                if (!isAnimationCancelled) {
                  setIsAnimationCancelled(true)
                }
              }}
              onKeyDown={(e) => {
                if (e.key === "Escape" && isListening) {
                  e.preventDefault()
                  stopVoiceCapture()
                  return
                }
                if (e.key === "Enter" && !e.shiftKey && !e.nativeEvent.isComposing) {
                  e.preventDefault()
                  handleSend()
                }
              }}
              disabled={isThinking || isUploading || isSessionConnecting || isAgentInitializing || isSessionFailed}
              className="flex-1 bg-transparent border-0 p-0 text-sm leading-normal focus:outline-none focus:ring-0 disabled:opacity-50 resize-none overflow-y-auto"
            />
          </div>
          {/* Voice Input Microphone Button */}
          <button
            type="button"
            onClick={toggleVoiceInput}
            disabled={isThinking || isUploading || isSessionConnecting || isAgentInitializing || isSessionFailed || !isSpeechSupported}
            title={
              !isSpeechSupported
                ? "Voice input is not supported in this browser (try Chrome/Edge)"
                : isListening
                ? "Listening... Click or press ESC / Alt+V to stop"
                : "Start voice input (Alt+V or Ctrl+Shift+M)"
            }
            className={cn(
              "flex h-10 w-10 shrink-0 items-center justify-center rounded-lg border border-input bg-card hover:bg-muted shadow-sm transition-colors mb-[1px]",
              isListening && "bg-red-500/10 border-red-500 text-red-500 dark:text-red-400 animate-pulse",
              (!isSpeechSupported || isUploading) && "opacity-50 cursor-not-allowed"
            )}
          >
            {isListening ? (
              <MicOff className="h-4.5 w-4.5" />
            ) : (
              <Mic className="h-4.5 w-4.5" />
            )}
          </button>
          {isThinking && (
            <button
              type="button"
              onClick={cancelMessage}
              className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg border border-input bg-card hover:bg-muted shadow-sm transition-colors animate-fade-in mb-[1px]"
              title="Stop active agent processing"
            >
              <Square className="h-3.5 w-3.5 fill-slate-500 text-slate-500" />
            </button>
          )}
          <button
            type="submit"
            disabled={isThinking || isUploading || (!inputText.trim() && !selectedQuickAction && stagedFiles.length === 0) || isSessionConnecting || isAgentInitializing || isSessionFailed}
            className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground shadow hover:opacity-90 disabled:opacity-40 transition-opacity mb-[1px]"
          >
            {isUploading ? (
              <Loader2 className="h-4.5 w-4.5 animate-spin" />
            ) : (
              <Send className="h-4.5 w-4.5" />
            )}
          </button>
        </div>

        {/* Quick action buttons under Chat prompt text box */}
        {quickActions && quickActions.length > 0 && (
          <div
            id="tour-workspace-quick-actions"
            className="flex flex-wrap items-center gap-2 pt-1 pl-[52px]"
          >
            {quickActions.map((action) => {
              const label = getQuickActionLabel(action)
              const isSelected = selectedQuickAction?.id === action.id || selectedQuickAction?.label === label
              return (
                <button
                  key={action.id}
                  type="button"
                  title={action.displayPrompt || label}
                  onClick={() => handleQuickActionClick(action)}
                  disabled={isThinking || isSessionConnecting || isAgentInitializing || isSessionFailed}
                  className={cn(
                    "relative inline-flex h-7 items-center gap-1.5 border px-3 rounded-full text-xs font-medium transition-all shadow-xs select-none disabled:opacity-50 disabled:pointer-events-none hover:shadow-sm active:scale-[0.98]",
                    !isAnimationCancelled && "animate-neon-glow-once-4s",
                    isSelected
                      ? "border-primary bg-primary/10 text-primary font-semibold"
                      : "border-border/80 bg-background/80 hover:bg-background hover:border-primary/50 text-foreground"
                  )}
                >
                  {/* Subtle glowing neon border snake animation (runs once after 4s delay, cancelled on focus) */}
                  {!isAnimationCancelled && (
                    <svg
                      className="pointer-events-none absolute -inset-[1px] h-[calc(100%+2px)] w-[calc(100%+2px)] rounded-full overflow-visible z-10"
                      aria-hidden="true"
                    >
                      <defs>
                        <linearGradient id="neon-snake-gradient" x1="0%" y1="0%" x2="100%" y2="100%">
                          <stop offset="0%" stopColor="hsl(var(--primary))" stopOpacity="0" />
                          <stop offset="70%" stopColor="hsl(var(--primary))" stopOpacity="0.6" />
                          <stop offset="100%" stopColor="hsl(var(--primary))" stopOpacity="0.85" />
                        </linearGradient>
                      </defs>
                      <rect
                        x="1"
                        y="1"
                        width="calc(100% - 2px)"
                        height="calc(100% - 2px)"
                        rx="14"
                        fill="none"
                        stroke="url(#neon-snake-gradient)"
                        strokeWidth="1.5"
                        pathLength="100"
                        className="animate-border-snake-4s"
                      />
                    </svg>
                  )}
                  <Sparkles className={cn("h-3 w-3", isSelected ? "text-primary" : "text-primary/70")} />
                  <span>{label}</span>
                </button>
              )
            })}
          </div>
        )}
      </form>
    )
  }

  return (
    <div ref={chatContainerRef} className="flex flex-col h-full min-h-0 bg-slate-50 dark:bg-slate-950 relative overflow-hidden">
      
      {/* Top loader bar for active session connections */}
      {isSessionConnecting && (
        <div className="w-full h-[3px] bg-primary/10 overflow-hidden shrink-0 relative z-20">
          <div 
            className="h-full bg-primary rounded-r animate-pulse" 
            style={{ 
              width: '45%', 
            }} 
          />
        </div>
      )}
      
      {/* Scrollable messages log frame container OR Centered Initial Empty State */}
      {isInitialEmptyState ? (
        <div className="flex-1 min-h-0 flex flex-col justify-center items-center px-6 pt-8 pb-36 sm:pb-44 relative overflow-y-auto">
          <div className="w-full max-w-2xl md:max-w-3xl relative animate-fade-in">
            {/* Main prompt headings greeting sitting directly above the card */}
            <div className="absolute bottom-full left-0 mb-2 w-full text-left select-none">
              <h2 className="flex items-center gap-2 text-base sm:text-lg font-semibold tracking-tight text-foreground">
                {PhaseIcon && (
                  <PhaseIcon className="h-5 w-5 shrink-0 text-primary" data-testid="phase-header-icon" />
                )}
                <span>Hi, where should we start today in the {formattedPhase} phase?</span>
              </h2>
            </div>

            {/* Central actions prompt input card block */}
            <div className="w-full rounded-2xl border border-border bg-card shadow-lg p-4 flex flex-col gap-3.5 hover:border-muted-foreground/20 transition-all duration-300">
              {renderComposer("centered")}
            </div>
          </div>

          {/* Reusable AlertModal Component */}
          {activeError && (
            <AlertModal
              type="error"
              title={activeError.title}
              description={activeError.description}
              details={activeError.details}
              onClose={() => triggerError(null)}
              action={
                (sessionConnectionError && /gemini|api_key|GEMINI_API_KEY_REQUIRED/i.test(sessionConnectionError))
                  ? {
                      label: "Go to Settings",
                      onClick: () => {
                        if (typeof window !== "undefined") {
                          window.location.href = "/settings"
                        }
                      },
                    }
                  : (sessionConnectionError || isSessionFailed ? onRetry : retryConnection)
                  ? {
                      label: "Retry",
                      onClick: sessionConnectionError || isSessionFailed ? onRetry : retryConnection,
                    }
                  : undefined
              }
            />
          )}
        </div>
      ) : (
        <div 
          className="flex-1 min-h-0 overflow-y-auto p-5 pb-12 space-y-5"
          onScroll={handleScroll}
          ref={scrollAreaViewportRef}
        >
          {showLoader || isSessionConnecting || (isStreamLoading && messages.length === 0) ? (
            <div className="flex h-full w-full flex-col items-center justify-center gap-3 text-muted-foreground select-none">
              <Loader2 className="h-6 w-6 animate-spin text-primary" />
              <span className="text-xs font-medium">
                {isSessionConnecting ? "Creating a new session" : "Loading session..."}
              </span>
            </div>
          ) : sessionConnectionError ? (
            <div className="flex h-full w-full flex-col items-center justify-center text-center gap-4 max-w-md mx-auto select-none p-5 animate-fade-in">
              <div className="flex h-12 w-12 items-center justify-center rounded-full bg-destructive/10 text-destructive shadow-sm">
                <HelpCircle className="h-6 w-6" />
              </div>
              <h3 className="font-bold text-base text-foreground">Failed to Initialize Session</h3>
              <p className="text-xs text-muted-foreground leading-relaxed">
                {sessionConnectionError}
              </p>
              <button
                onClick={onRetry}
                className="inline-flex h-9 items-center justify-center gap-1.5 text-xs font-bold bg-primary text-primary-foreground px-4 rounded-lg hover:opacity-90 shadow-sm transition-opacity"
              >
                <RefreshCw className="h-3.5 w-3.5" />
                Retry Connection
              </button>
            </div>
          ) : isLocked ? (
            <div className="flex h-full w-full flex-col items-center justify-center text-center gap-4 max-w-md mx-auto select-none p-5 animate-fade-in">
              <div className="flex h-12 w-12 items-center justify-center rounded-full bg-red-100 dark:bg-red-950/50 text-red-500 shadow-sm border border-red-200 dark:border-red-900/50">
                <Lock className="h-6 w-6" />
              </div>
              <h3 className="font-bold text-base text-foreground">Phase is Locked</h3>
              <p className="text-xs text-muted-foreground leading-relaxed">
                This phase is locked because its prerequisite phases are not completed yet. 
                Please satisfy all the gates in the previous phases to unlock and start a session here.
              </p>
            </div>
          ) : !sessionId ? (
            <div className="flex h-full w-full flex-col items-center justify-center text-center gap-2.5 text-muted-foreground max-w-sm mx-auto select-none">
              <Sparkles className="h-10 w-10 text-primary/40 animate-pulse" />
              <h3 className="font-bold text-sm text-foreground">No active chat session</h3>
              <p className="text-xs leading-normal">
                Establish a new chat session from the left-side Workspace Explorer sidebar to start prompting the agent.
              </p>
            </div>
          ) : (
            <div className="space-y-5">
              {messages.map((msg, idx) => {
                const isUser = msg.isUser
                const colorClass = getPersonaColorClass(msg.sender, msg.phase)
                const userInitial = msg.sender && msg.sender.length > 0 ? msg.sender[0].toUpperCase() : "?"

                return (
                  <div
                    key={msg.id || idx}
                    className={cn(
                      "flex gap-3 max-w-[85%] md:max-w-[78%] animate-fade-in",
                      isUser ? "ml-auto flex-row-reverse" : "mr-auto"
                    )}
                  >
                    {/* Left Role/User Avatar Circle */}
                    <div
                      className={cn(
                        "flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-white text-xs font-bold shadow-sm select-none",
                        isUser 
                          ? "bg-slate-600 border border-slate-700" 
                          : colorClass
                      )}
                    >
                      {isUser ? <User className="h-4 w-4" /> : userInitial}
                    </div>

                    {/* Message Bubble Column */}
                    <div className="flex flex-col gap-1.5 min-w-0">
                      {/* Header line metadata */}
                      <div className={cn(
                        "flex items-center gap-2 px-1 text-[10px] text-muted-foreground",
                        isUser ? "justify-end" : "justify-start"
                      )}>
                        <span className="font-bold text-foreground">
                          {isUser ? "You" : msg.sender}
                        </span>
                        {msg.timestamp && (
                          <span>• {formatSessionDate(msg.timestamp)}</span>
                        )}
                      </div>

                      {/* Phase 3: Completed Collapsible Thinking Process */}
                      {!isUser && ((msg.thinkingSteps && msg.thinkingSteps.length > 0) || msg.thinkingText) && (
                        <ThinkingBlock
                          isStreaming={false}
                          thinkingSteps={msg.thinkingSteps}
                          thinkingText={msg.thinkingText}
                          durationSec={msg.thinkingDurationSec}
                          onScrollBottom={handleScrollBottom}
                        />
                      )}

                      {/* Text Body Card */}
                      <div
                        className={cn(
                          "rounded-2xl px-4 py-3 border text-sm shadow-sm min-w-0 flex flex-col gap-2",
                          isUser
                            ? "bg-primary border-primary text-primary-foreground rounded-tr-none"
                            : "bg-card border-border text-card-foreground rounded-tl-none"
                        )}
                      >
                        {/* Attached Artifacts Display */}
                        {msg.artifacts && msg.artifacts.length > 0 && (
                          <div className="flex flex-wrap gap-1.5 pb-1">
                            {msg.artifacts.map((art, artIdx) => {
                              const artName = (art.name || "").toLowerCase()
                              const isImg = (art.mimeType && art.mimeType.startsWith("image/")) || /\.(png|jpg|jpeg|webp|gif|svg)$/i.test(artName)
                              return (
                                <div
                                  key={art.id || artIdx}
                                  className={cn(
                                    "inline-flex items-center gap-1.5 rounded-lg px-2.5 py-1 text-xs border max-w-full shadow-2xs",
                                    isUser
                                      ? "bg-primary-foreground/15 border-primary-foreground/20 text-primary-foreground"
                                      : "bg-muted/80 border-border text-foreground"
                                  )}
                                >
                                  {isImg ? (
                                    <ImageIcon className="h-3.5 w-3.5 shrink-0" />
                                  ) : (
                                    <FileText className="h-3.5 w-3.5 shrink-0" />
                                  )}
                                  <span className="truncate max-w-[160px] font-medium" title={art.name || art.title || "Attachment"}>
                                    {art.name || art.title || "Attachment"}
                                  </span>
                                  {art.size !== undefined && art.size > 0 && (
                                    <span className={cn(
                                      "text-[10px]",
                                      isUser ? "text-primary-foreground/75" : "text-muted-foreground"
                                    )}>
                                      ({formatFileSize(art.size)})
                                    </span>
                                  )}
                                </div>
                              )
                            })}
                          </div>
                        )}

                        {/* Markdown processing panel */}
                        <div className="prose prose-slate dark:prose-invert prose-xs max-w-none break-words w-full min-w-0 leading-relaxed space-y-2 prose-headings:font-bold prose-p:my-1 prose-ul:list-disc prose-ul:pl-4 prose-ol:list-decimal prose-ol:pl-4 prose-pre:bg-muted prose-pre:border prose-pre:p-2.5 prose-pre:rounded-lg prose-pre:whitespace-pre-wrap prose-pre:break-words prose-code:font-mono prose-code:text-xs prose-code:break-words prose-table:w-full prose-table:border-collapse prose-th:border prose-th:p-1.5 prose-td:border prose-td:p-1.5">
                          <ReactMarkdown remarkPlugins={[remarkGfm]}>
                            {msg.displayText || msg.displayPrompt || msg.text}
                          </ReactMarkdown>
                        </div>
                      </div>

                      {/* Phase Switch Suggestion Banner */}
                      {(() => {
                        const switchSuggestion: PhaseSwitchSuggestion | undefined = msg.phaseSwitchSuggestion || (() => {
                          const switchAction = msg.actions?.find((a) => a.startsWith("SWITCH_PHASE:"))
                          if (!switchAction) return undefined
                          const parts = switchAction.split(":")
                          return {
                            targetPhase: parts[1] || "PLANNING",
                            targetPhaseLabel: undefined,
                            skillId: parts[2] || undefined,
                            reason: parts.slice(3).join(":") || undefined,
                            suggestedPrompt: undefined,
                          }
                        })()

                        if (!switchSuggestion) return null

                        return (
                          <PhaseSwitchCard
                            targetPhase={switchSuggestion.targetPhase}
                            targetPhaseLabel={switchSuggestion.targetPhaseLabel}
                            skillId={switchSuggestion.skillId}
                            reason={switchSuggestion.reason}
                            suggestedPrompt={switchSuggestion.suggestedPrompt}
                            onConfirmSwitch={(targetPhase, promptToExecute) => {
                              onCrossPhaseSwitch?.(targetPhase, promptToExecute)
                            }}
                          />
                        )
                      })()}

                      {/* Action chips choices list */}
                      {msg.actions && msg.actions.filter(a => !a.startsWith("SWITCH_PHASE:")).length > 0 && (
                        <div className="flex flex-wrap gap-1.5 mt-1">
                          {msg.actions
                            .filter(a => !a.startsWith("SWITCH_PHASE:"))
                            .map((action, actionIdx) => (
                              <button
                                key={actionIdx}
                                onClick={() => handleActionChipClick(action)}
                                className="inline-flex items-center gap-1 text-[10px] font-semibold border border-primary/20 bg-primary/5 text-primary px-3 py-1 rounded-full hover:bg-primary/10 transition-colors"
                              >
                                <Terminal className="h-3 w-3 shrink-0" />
                                {action.startsWith("bmad-") ? action.substring(5) : action}
                              </button>
                            ))}
                        </div>
                      )}
                    </div>
                  </div>
                )
              })}
            </div>
          )}

          {/* Dynamic loading state & Phase 1 live thinking block */}
          {(() => {
            const rawActiveStep = (thinkingSteps && thinkingSteps.length > 0)
              ? thinkingSteps[thinkingSteps.length - 1]
              : latestThought

            const activeStep = rawActiveStep?.notificationDetails?.name === "agent-init" ? null : rawActiveStep

            return (isThinking || activeStep) ? (
              <div className="flex items-start gap-3 mr-auto max-w-[85%] animate-fade-in mt-4 group w-full">
                <div
                  className={cn(
                    "flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-white text-xs font-bold shadow-sm select-none",
                    activeStep
                      ? getPersonaColorClass(activeStep.sender, activeStep.phase)
                      : "bg-primary animate-pulse"
                  )}
                >
                  {activeStep?.sender ? activeStep.sender[0].toUpperCase() : "?"}
                </div>
                <div className="flex flex-col gap-1.5 flex-1 min-w-0">
                  <span className="text-[10px] text-muted-foreground px-1 font-medium">
                    {activeStep?.sender ? `${activeStep.sender} is processing...` : "Agent is thinking..."}
                  </span>

                  {/* Important attention card if attention required */}
                  {activeStep && (activeStep.awaitingInput || activeStep.notificationDetails?.name === "error") && (
                    <div className="rounded-2xl px-4 py-3 border-2 border-amber-500/40 bg-amber-500/10 dark:bg-amber-950/20 text-foreground text-sm rounded-tl-none shadow-md max-w-lg min-w-0 flex-1 animate-pulse">
                      <div className="flex items-center gap-2.5 text-xs font-bold text-amber-600 dark:text-amber-400 min-w-0">
                        <AlertTriangle className="h-4 w-4 shrink-0" />
                        <span className="break-words">{activeStep.text}</span>
                      </div>
                    </div>
                  )}

                  {/* Collapsible Thinking / Reasoning Container (Phases 1 & 2) */}
                  <ThinkingBlock
                    isStreaming={isThinking}
                    thinkingSteps={
                      thinkingSteps && thinkingSteps.length > 0
                        ? thinkingSteps
                        : activeStep
                        ? [activeStep]
                        : undefined
                    }
                    thinkingText={
                      !thinkingSteps?.length && !activeStep
                        ? "Initializing agent reasoning..."
                        : undefined
                    }
                    onScrollBottom={handleScrollBottom}
                  />
                </div>
              </div>
            ) : null
          })()}

          {/* Reusable AlertModal Component */}
          {activeError && (
            <AlertModal
              type="error"
              title={activeError.title}
              description={activeError.description}
              details={activeError.details}
              onClose={() => triggerError(null)}
              action={
                (sessionConnectionError && /gemini|api_key|GEMINI_API_KEY_REQUIRED/i.test(sessionConnectionError))
                  ? {
                      label: "Go to Settings",
                      onClick: () => {
                        if (typeof window !== "undefined") {
                          window.location.href = "/settings"
                        }
                      },
                    }
                  : (sessionConnectionError || isSessionFailed ? onRetry : retryConnection)
                  ? {
                      label: "Retry",
                      onClick: sessionConnectionError || isSessionFailed ? onRetry : retryConnection,
                    }
                  : undefined
              }
            />
          )}

          {/* Scroll anchor target element */}
          <div ref={messagesEndRef} className="h-12 w-full shrink-0" />
        </div>
      )}

      {/* Floating scroll down button */}
      {showScrollButton && !isInitialEmptyState && (
        <button
          onClick={() => {
            isUserScrolledUpRef.current = false
            scrollToBottom("smooth", true)
          }}
          className="absolute bottom-24 right-8 flex h-9 w-9 items-center justify-center rounded-full bg-card border border-border text-muted-foreground hover:bg-muted hover:text-foreground shadow-md transition-all z-20 animate-bounce"
          title="Scroll to bottom"
        >
          <ArrowDown className="h-4 w-4" />
        </button>
      )}

      {/* COMPOSER INPUT SECTION FOOTER BAR (when not in centered initial state) */}
      {sessionId && !isInitialEmptyState && (
        <div className="flex flex-col border-t border-border bg-slate-100 dark:bg-slate-900 shrink-0 relative z-10">
          {renderComposer("bottom")}
        </div>
      )}
    </div>
  )
})
export default ChatPane
export { Loader2 as Loader2Icon }
