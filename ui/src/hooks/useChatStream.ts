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

import { useState, useEffect, useRef, useCallback } from "react"
import { connectSSE } from "@/lib/sse"
import { api } from "@/lib/api"
import { Message, PhaseSwitchSuggestion, Artifact } from "@/lib/types"

function sortMessagesChronologically(msgs: Message[]): Message[] {
  return [...msgs].sort((a, b) => {
    const timeA = a.timestamp ? new Date(a.timestamp).getTime() : 0
    const timeB = b.timestamp ? new Date(b.timestamp).getTime() : 0
    if (timeA !== timeB && timeA > 0 && timeB > 0) {
      return timeA - timeB
    }
    if (a.isUser && !b.isUser) return -1
    if (!a.isUser && b.isUser) return 1
    return 0
  })
}

function extractPhaseSwitchSuggestion(msg: Message): PhaseSwitchSuggestion | undefined {
  if (msg.phaseSwitchSuggestion) return msg.phaseSwitchSuggestion
  if (msg.notificationDetails?.params?.tool_name === "suggest_phase_switch") {
    try {
      const contentStr = msg.notificationDetails.params.content
      if (typeof contentStr === "string") {
        const parsed = JSON.parse(contentStr)
        if (parsed && parsed.targetPhase) {
          return {
            targetPhase: parsed.targetPhase,
            skillId: parsed.skillId,
            reason: parsed.reason,
          }
        }
      } else if (typeof contentStr === "object" && contentStr?.targetPhase) {
        return {
          targetPhase: contentStr.targetPhase,
          skillId: contentStr.skillId,
          reason: contentStr.reason,
        }
      }
    } catch (_) {}
  }
  return undefined
}

function isAccordionStep(msg: Message): boolean {
  if (msg.isUser) return false
  if (msg.isTransient === true) return true
  if (msg.notificationDetails != null && msg.notificationDetails.name !== "error") return true
  if (msg.text && (
    /^(?:\*\*)?Executing tool:/i.test(msg.text) ||
    /^(?:\*\*)?Read file:/i.test(msg.text) ||
    /^(?:\*\*)?Tool result/i.test(msg.text)
  )) return true
  return false
}

export interface UseChatStreamOptions {
  onLatestAgentOutput?: (output: string) => void
  onAgentInputSent?: (input: string) => void
  includeNotifications?: boolean
  streamEndpoint?: string
  sendMessageEndpoint?:
    | string
    | ((text: string, defaultPayload: Record<string, any>) => { endpoint: string; payload?: Record<string, any> })
  customSendMessage?: (text: string) => Promise<boolean>
  additionalPayload?: Record<string, any>
}

export function useChatStream(
  projectId: string,
  phase: string,
  sessionId: string | null,
  options: UseChatStreamOptions = {}
) {
  const [messages, setMessages] = useState<Message[]>([])
  const [isLoading, setIsLoading] = useState<boolean>(true)
  const [isThinking, setIsThinking] = useState<boolean>(false)
  const [latestThought, setLatestThought] = useState<Message | null>(null)
  const [thinkingSteps, setThinkingSteps] = useState<Message[]>([])
  const [error, setError] = useState<{ title: string; description?: string; details?: string } | null>(null)

  const activeSubRef = useRef<(() => void) | null>(null)
  const thinkingTimeoutRef = useRef<NodeJS.Timeout | null>(null)
  const activityTimeoutRef = useRef<NodeJS.Timeout | null>(null)
  const transientNotificationTimeoutRef = useRef<NodeJS.Timeout | null>(null)
  const resetActivityTimeoutRef = useRef<(() => void) | null>(null)
  const resetThinkingTimeoutRef = useRef<(() => void) | null>(null)
  const retryConnectionRef = useRef<(() => void) | null>(null)

  const isThinkingRef = useRef(isThinking)
  isThinkingRef.current = isThinking

  const thinkingStepsRef = useRef<Message[]>(thinkingSteps)
  thinkingStepsRef.current = thinkingSteps

  const thinkingStartTimeRef = useRef<number | null>(null)

  const messagesRef = useRef<Message[]>(messages)
  messagesRef.current = messages

  const connectionTimeRef = useRef<Date | null>(null)
  const recentSwitchSuggestionRef = useRef<PhaseSwitchSuggestion | undefined>(undefined)
  
  // Cache targets in refs to bypass stale closures inside standard callbacks
  const optionsRef = useRef(options)
  optionsRef.current = options

  const clearThinkingTimeout = useCallback(() => {
    if (thinkingTimeoutRef.current) {
      clearTimeout(thinkingTimeoutRef.current)
      thinkingTimeoutRef.current = null
    }
  }, [])

  const resetThinkingTimeout = useCallback(() => {
    if (thinkingTimeoutRef.current) {
      clearTimeout(thinkingTimeoutRef.current)
    }
    thinkingTimeoutRef.current = setTimeout(() => {
      setIsThinking(false)
      setError({
        title: "Connection Timeout",
        description: "The AI agent took too long to respond. Please verify your request and try again.",
      })
    }, 90000)
  }, [])

  resetThinkingTimeoutRef.current = resetThinkingTimeout

  const connect = useCallback((isReconnect = false) => {
    // Terminate any preceding stream connections
    if (activeSubRef.current) {
      activeSubRef.current()
      activeSubRef.current = null
    }

    if (!isReconnect) {
      clearThinkingTimeout()
    }

    if (!sessionId) {
      recentSwitchSuggestionRef.current = undefined
      setMessages([])
      setIsLoading(false)
      setIsThinking(false)
      setLatestThought(null)
      setThinkingSteps([])
      setError(null)
      return
    }

    // Prepare fresh transition states
    if (!isReconnect) {
      recentSwitchSuggestionRef.current = undefined
      connectionTimeRef.current = new Date()
      setMessages([])
      setIsLoading(true)
      setIsThinking(false)
      setLatestThought(null)
      setThinkingSteps([])
      setError(null)
    } else {
      setError(null)
    }

    let url = optionsRef.current.streamEndpoint || `/api/v1/projects/${projectId}/messages/stream?phase=${phase}&sessionId=${sessionId}`

    if (isReconnect && messagesRef.current.length > 0) {
      const lastMsg = messagesRef.current[messagesRef.current.length - 1]
      if (lastMsg.timestamp) {
        url += `&since=${encodeURIComponent(lastMsg.timestamp)}`
      }
    }
    
    const unsubscribeSSE = connectSSE(url, {
      onOpen: () => {
        setIsLoading(false)
        setError(null)
        if (isThinkingRef.current) {
          resetActivityTimeoutRef.current?.()
          resetThinkingTimeoutRef.current?.()
        }
      },
      onKeepAlive: () => {
        if (isThinkingRef.current) {
          resetActivityTimeoutRef.current?.()
        }
      },
      onMessage: (event, dataStr) => {
        setIsLoading(false)
        setError(null)
        if (isThinkingRef.current) {
          resetActivityTimeoutRef.current?.()
          resetThinkingTimeoutRef.current?.()
        }
        try {
          const decoded = JSON.parse(dataStr)
          
          // Ignore agent-init system messages so clean initial state does not show thinking steps
          if (decoded.notificationDetails?.name === "agent-init") {
            return
          }

          const isNotification = decoded.isNotification === true
          
          const msgTime = decoded.timestamp ? new Date(decoded.timestamp) : new Date()
          const isMsgHistorical = connectionTimeRef.current
            ? msgTime.getTime() < connectionTimeRef.current.getTime()
            : decoded.isHistorical === true

          const extractedSuggestion = extractPhaseSwitchSuggestion(decoded as Message)
          if (extractedSuggestion) {
            recentSwitchSuggestionRef.current = extractedSuggestion
          }

          const isStep = isAccordionStep(decoded as Message)

          const suggestionToAttach = !decoded.isUser
            ? (decoded.phaseSwitchSuggestion || (!isStep ? recentSwitchSuggestionRef.current : undefined))
            : undefined

          if (!isStep && !decoded.isUser && recentSwitchSuggestionRef.current) {
            recentSwitchSuggestionRef.current = undefined
          }

          const msg = {
            ...decoded,
            isHistorical: isMsgHistorical,
            phaseSwitchSuggestion: suggestionToAttach,
            rawContent: JSON.stringify(decoded, null, 2)
          } as Message

          // Synchronously store tool executions, file reads, and intermediate progress in thinkingStepsRef
          if (isStep) {
            setLatestThought(msg)
            const exists = thinkingStepsRef.current.some((m) => {
              if (m.id && msg.id) return m.id === msg.id
              return m.text === msg.text && m.sender === msg.sender && m.timestamp === msg.timestamp
            })
            if (!exists) {
              thinkingStepsRef.current = [...thinkingStepsRef.current, msg]
              setThinkingSteps([...thinkingStepsRef.current])
            }
            if (transientNotificationTimeoutRef.current) {
              clearTimeout(transientNotificationTimeoutRef.current)
            }
            if (!isThinkingRef.current) {
              transientNotificationTimeoutRef.current = setTimeout(() => {
                setLatestThought(null)
              }, 10000)
            }
          }

          // Append main final messages to the chat list
          if (!isStep || optionsRef.current.includeNotifications) {
            const durationSec = thinkingStartTimeRef.current
              ? Math.max(1, Math.round((Date.now() - thinkingStartTimeRef.current) / 1000))
              : undefined

            const sortedThinkingSteps =
              thinkingStepsRef.current.length > 0
                ? sortMessagesChronologically([...thinkingStepsRef.current])
                : undefined

            const enrichedMsg: Message = {
              ...msg,
              phaseSwitchSuggestion: msg.phaseSwitchSuggestion,
              thinkingSteps: !msg.isUser ? sortedThinkingSteps : undefined,
              thinkingDurationSec: !msg.isUser ? durationSec : undefined,
            }

            setMessages((prev) => {
              // Replace optimistic message with the real one if it matches
              const optimisticIndex = prev.findIndex(
                (m) =>
                  m.id?.startsWith("optimistic-") &&
                  (m.text === enrichedMsg.text ||
                    (enrichedMsg.displayText && m.displayText === enrichedMsg.displayText) ||
                    (enrichedMsg.displayPrompt && m.text === enrichedMsg.displayPrompt) ||
                    (enrichedMsg.prompt && m.prompt === enrichedMsg.prompt) ||
                    (enrichedMsg.prompt && m.text === enrichedMsg.prompt))
              )
              if (optimisticIndex !== -1) {
                const updated = [...prev]
                updated[optimisticIndex] = enrichedMsg
                return sortMessagesChronologically(updated)
              }

              // Standard de-duplication safe-check
              const exists = prev.some((m) => {
                if (m.id && enrichedMsg.id) return m.id === enrichedMsg.id
                return (
                  m.sender === enrichedMsg.sender &&
                  (m.text === enrichedMsg.text || (enrichedMsg.prompt && m.prompt === enrichedMsg.prompt)) &&
                  m.timestamp === enrichedMsg.timestamp
                )
              })
              if (exists) return prev
              return sortMessagesChronologically([...prev, enrichedMsg])
            })
          }

          // Clear thinking steps at start of user turn
          if (msg.isUser) {
            recentSwitchSuggestionRef.current = undefined
            thinkingStepsRef.current = []
            setThinkingSteps([])
            setLatestThought(null)
          }

          const isErrorStep = msg.notificationDetails?.name === "error"
          if (!msg.isUser && (!isStep || isErrorStep)) {
            setIsThinking(false)
            setLatestThought(null)
            thinkingStepsRef.current = []
            setThinkingSteps([])
            thinkingStartTimeRef.current = null
            if (isErrorStep) {
              if (!isMsgHistorical && !msg.isHistorical) {
                const params = msg.notificationDetails?.params || {}
                setError({
                  title: params.title || msg.text || "An error occurred",
                  description: params.description || "An unexpected error occurred while processing the request.",
                  details: params.details || "",
                })
              }
            } else {
              setError(null)
            }
            clearThinkingTimeout()
          }

          // Dynamic feedback hook for UI sync handlers
          if (msg.isHistorical !== true && !msg.isUser && msg.rawContent) {
            optionsRef.current.onLatestAgentOutput?.(msg.rawContent)
          }
        } catch (e) {
          console.error("Error parsing stream message string:", e)
        }
      },
      onError: (err) => {
        console.warn("Chat stream parser connection dropped:", err)
        // Set silent reconnect indicators
        setIsLoading(false)
      },
    })

    activeSubRef.current = unsubscribeSSE
  }, [projectId, phase, sessionId, clearThinkingTimeout])

  const retryConnection = useCallback(() => {
    connect(true)
  }, [connect])

  retryConnectionRef.current = retryConnection

  const resetActivityTimeout = useCallback(() => {
    if (activityTimeoutRef.current) {
      clearTimeout(activityTimeoutRef.current)
    }
    activityTimeoutRef.current = setTimeout(() => {
      console.warn("No message or keepalive received from API for 45 seconds. Reconnecting stream...")
      retryConnectionRef.current?.()
    }, 45000)
  }, [])

  resetActivityTimeoutRef.current = resetActivityTimeout

  const clearActivityTimeout = useCallback(() => {
    if (activityTimeoutRef.current) {
      clearTimeout(activityTimeoutRef.current)
      activityTimeoutRef.current = null
    }
  }, [])

  useEffect(() => {
    if (isThinking) {
      resetActivityTimeout()
    } else {
      clearActivityTimeout()
    }
    return () => clearActivityTimeout()
  }, [isThinking, resetActivityTimeout, clearActivityTimeout])

  // Subscribes whenever targets mutate
  useEffect(() => {
    connect()
    return () => {
      if (activeSubRef.current) {
        activeSubRef.current()
        activeSubRef.current = null
      }
      clearThinkingTimeout()
      if (activityTimeoutRef.current) {
        clearTimeout(activityTimeoutRef.current)
        activityTimeoutRef.current = null
      }
      if (transientNotificationTimeoutRef.current) {
        clearTimeout(transientNotificationTimeoutRef.current)
        transientNotificationTimeoutRef.current = null
      }
    }
  }, [connect])

  const sendMessage = useCallback(
    async (text: string, displayText?: string, artifacts?: Artifact[]): Promise<boolean> => {
      const trimmedText = text.trim()
      const hasArtifacts = !!(artifacts && artifacts.length > 0)
      if ((!trimmedText && !hasArtifacts) || !sessionId) return false

      const artifactIds = artifacts?.map((a) => a.id).filter((id): id is string => Boolean(id))

      const defaultText = hasArtifacts ? "Uploaded attachment(s)" : ""
      const effectiveText = trimmedText || defaultText
      const bubbleText = (displayText && displayText.trim()) ? displayText.trim() : effectiveText

      setIsThinking(true)
      thinkingStartTimeRef.current = Date.now()
      setLatestThought(null)
      setThinkingSteps([])
      setError(null)

      // Add optimistic user message to the list immediately
      const optimisticMsgId = `optimistic-${Date.now()}`
      const optimisticMsg: Message = {
        id: optimisticMsgId,
        sender: "User",
        text: bubbleText,
        displayText: bubbleText,
        displayPrompt: bubbleText,
        prompt: effectiveText,
        isUser: true,
        timestamp: new Date().toISOString(),
        artifacts: hasArtifacts ? artifacts : undefined,
        artifactIds: artifactIds && artifactIds.length > 0 ? artifactIds : undefined,
      }
      setMessages((prev) => sortMessagesChronologically([...prev, optimisticMsg]))

      // Start thinking fallback timeout
      resetThinkingTimeout()

      try {
        if (optionsRef.current.customSendMessage) {
          const success = await optionsRef.current.customSendMessage(effectiveText)
          if (!success) {
            setMessages((prev) => prev.filter((m) => m.id !== optimisticMsgId))
            setIsThinking(false)
            clearThinkingTimeout()
          }
          return success
        }

        let payload: Record<string, any> = {
          sender: "User",
          text: bubbleText,
          displayText: bubbleText,
          displayPrompt: bubbleText,
          prompt: effectiveText,
          isUser: true,
          phase: phase,
          sessionId: sessionId,
          ...(hasArtifacts ? { artifacts, artifactIds } : {}),
          ...(optionsRef.current.additionalPayload || {}),
        }

        let endpoint = `/api/v1/projects/${projectId}/messages?sessionId=${sessionId}`

        if (typeof optionsRef.current.sendMessageEndpoint === "function") {
          const customConfig = optionsRef.current.sendMessageEndpoint(trimmedText, payload)
          endpoint = customConfig.endpoint
          if (customConfig.payload !== undefined) {
            payload = customConfig.payload
          }
        } else if (typeof optionsRef.current.sendMessageEndpoint === "string") {
          endpoint = optionsRef.current.sendMessageEndpoint
        }

        // Invoke the agent input logger callback for the dynamic debug console tabs
        optionsRef.current.onAgentInputSent?.(JSON.stringify({ endpoint, ...payload }, null, 2))

        const response = await api.post(endpoint, payload)

        if (response.status === 201) {
          return true
        } else {
          // Remove optimistic message if delivery failed
          setMessages((prev) => prev.filter((m) => m.id !== optimisticMsgId))
          setIsThinking(false)
          setError({
            title: "Delivery Failed",
            description: "The message could not be delivered to the API backend.",
            details: `HTTP Status ${response.status}`
          })
          clearThinkingTimeout()
          return false
        }
      } catch (e) {
        // Remove optimistic message if delivery failed
        setMessages((prev) => prev.filter((m) => m.id !== optimisticMsgId))
        setIsThinking(false)
        setError({
          title: "Delivery Failed",
          description: "A network error occurred while sending your message to the API backend.",
          details: e instanceof Error ? e.message : String(e)
        })
        clearThinkingTimeout()
        return false
      }
    },
    [projectId, phase, sessionId, resetThinkingTimeout, clearThinkingTimeout]
  )

  const cancelMessage = useCallback(async () => {
    if (!sessionId || !isThinking) return

    setIsThinking(false)
    setLatestThought(null)
    setThinkingSteps([])
    clearThinkingTimeout()

    try {
      await api.post(`/api/v1/projects/${projectId}/messages/cancel?sessionId=${sessionId}`)
    } catch (e) {
      console.warn("Failed to propagate cancellation to backend:", e)
    }
  }, [projectId, sessionId, isThinking, clearThinkingTimeout])

  const triggerError = useCallback((errObj: { title: string; description?: string; details?: string } | null) => {
    if (errObj === null) {
      setError(null)
      return
    }
    if (!isThinking) return
    setIsThinking(false)
    setError(errObj)
    clearThinkingTimeout()
  }, [isThinking, clearThinkingTimeout])

  return {
    messages,
    isLoading,
    isThinking,
    latestThought,
    thinkingSteps,
    error,
    sendMessage,
    cancelMessage,
    retryConnection,
    triggerError,
  }
}
export default useChatStream
