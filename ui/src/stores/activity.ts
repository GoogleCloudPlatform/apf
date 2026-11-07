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

import { create } from "zustand"
import { toast } from "sonner"
import React from "react"
import { connectSSE } from "@/lib/sse"
import { Message } from "@/lib/types"

interface ActivityState {
  unreadCount: number
  activities: Message[]
  isPanelOpen: boolean
  currentProjectId: string | null

  // Caches for background tab persistence
  projectActivities: Record<string, Message[]>
  projectUnreadCounts: Record<string, number>
  streamSubscriptions: Record<string, () => void>
  subscriberCounts: Record<string, number>
  connectionTimes: Record<string, Date>

  // Actions
  subscribe: (projectId: string) => void
  unsubscribe: (projectId: string) => void
  togglePanel: () => void
  getLatestTimestamp: (projectId: string) => Date | null
  clearAll: () => void
}

// Map dynamic roles to standard theme color classes for Sonner badges
export function getPersonaColorClass(sender: string, phase?: string): string {
  const s = sender.toLowerCase()

  if (
    s.includes("agent") ||
    s.includes("pm") ||
    s.includes("ux") ||
    s.includes("architect") ||
    s.includes("dev") ||
    s.includes("apf")
  ) {
    return "bg-blue-500 border-blue-600"
  } else if (
    s.includes("system") ||
    s.includes("middleware")
  ) {
    return "bg-slate-600 border-slate-700"
  } else {
    return "bg-green-500 border-green-600"
  }
}

export const useActivityStore = create<ActivityState>((set, get) => ({
  unreadCount: 0,
  activities: [],
  isPanelOpen: false,
  currentProjectId: null,

  projectActivities: {},
  projectUnreadCounts: {},
  streamSubscriptions: {},
  subscriberCounts: {},
  connectionTimes: {},

  getLatestTimestamp: (projectId) => {
    const list = get().projectActivities[projectId]
    if (!list || list.length === 0) return null
    // The latest messages are inserted at index 0 (latest at top)
    const latestMsg = list[0]
    if (!latestMsg.timestamp) return null
    const date = new Date(latestMsg.timestamp)
    return isNaN(date.getTime()) ? null : date
  },

  subscribe: (projectId) => {
    const state = get()
    const currentSubs = { ...state.streamSubscriptions }
    const currentCounts = { ...state.subscriberCounts }
    const currentTimes = { ...state.connectionTimes }

    const count = (currentCounts[projectId] || 0) + 1
    currentCounts[projectId] = count

    // Set the current project context dynamically
    if (state.currentProjectId !== projectId) {
      set({
        currentProjectId: projectId,
        activities: state.projectActivities[projectId] || [],
        unreadCount: state.projectUnreadCounts[projectId] || 0,
      })
    }

    // If stream subscription is already alive for this project, just exit
    if (currentSubs[projectId]) {
      set({ subscriberCounts: currentCounts })
      return
    }

    if (!currentTimes[projectId]) {
      currentTimes[projectId] = new Date()
    }

    const getUrl = () => {
      const latestTime = get().getLatestTimestamp(projectId)
      // Fallback to 24h before connection time if no local history cache is available
      const sinceDate = latestTime || new Date(currentTimes[projectId].getTime() - 24 * 60 * 60 * 1000)
      const sinceParam = encodeURIComponent(sinceDate.toISOString())
      return `/api/v1/projects/${projectId}/activities/stream?since=${sinceParam}`
    }

    const unsubscribeSSE = connectSSE(getUrl, {
      onMessage: (event, dataStr) => {
        if (event === "error") {
          console.warn(`Server sent error events in activity stream: ${dataStr}`)
          return
        }

        try {
          const msg = JSON.parse(dataStr) as Message
          
          // Format structural message data matching original Dart logic
          const list = [...(get().projectActivities[projectId] || [])]
          
          // De-duplicate items
          const exists = list.some((m) => {
            if (m.id && msg.id) return m.id === msg.id
            return (
              m.sender === msg.sender &&
              m.text === msg.text &&
              m.timestamp === msg.timestamp
            )
          })

          if (!exists) {
            // Insert at index 0 (latest at top)
            list.unshift(msg)
            
            // Sort to guarantee correct order descending
            list.sort((a, b) => {
              const timeA = a.timestamp ? new Date(a.timestamp).getTime() : 0
              const timeB = b.timestamp ? new Date(b.timestamp).getTime() : 0
              return timeB - timeA
            })

            const initialConnectTime = get().connectionTimes[projectId]
            const msgTime = msg.timestamp ? new Date(msg.timestamp) : new Date()
            const isHistorical = initialConnectTime ? msgTime.getTime() < initialConnectTime.getTime() : msg.isHistorical === true

            set((s) => {
              const updatedActivities = { ...s.projectActivities, [projectId]: list }
              const updatedUnread = { ...s.projectUnreadCounts }

              if (projectId === s.currentProjectId) {
                updatedUnread[projectId] = 0
                return {
                  projectActivities: updatedActivities,
                  projectUnreadCounts: updatedUnread,
                  activities: list,
                  unreadCount: 0,
                }
              } else {
                if (!isHistorical) {
                  updatedUnread[projectId] = (updatedUnread[projectId] || 0) + 1
                }
                return {
                  projectActivities: updatedActivities,
                  projectUnreadCounts: updatedUnread,
                }
              }
            })
          }
        } catch (e) {
          // Suppress parsing errors for robustness
        }
      },
      onError: (err) => {
        console.warn(`Activity stream connection error for project ${projectId}:`, err)
      },
    })

    currentSubs[projectId] = unsubscribeSSE
    set({
      streamSubscriptions: currentSubs,
      subscriberCounts: currentCounts,
      connectionTimes: currentTimes,
    })
  },

  unsubscribe: (projectId) => {
    const state = get()
    const currentCounts = { ...state.subscriberCounts }
    const currentSubs = { ...state.streamSubscriptions }
    const currentTimes = { ...state.connectionTimes }
    const currentActivities = { ...state.projectActivities }
    const currentUnread = { ...state.projectUnreadCounts }

    const count = (currentCounts[projectId] || 0) - 1

    if (count <= 0) {
      delete currentCounts[projectId]
      if (currentSubs[projectId]) {
        currentSubs[projectId]() // Invoke abort callback
        delete currentSubs[projectId]
      }
      delete currentTimes[projectId]
      delete currentActivities[projectId]
      delete currentUnread[projectId]

      const nextActiveProject = state.currentProjectId === projectId ? null : state.currentProjectId

      set({
        streamSubscriptions: currentSubs,
        subscriberCounts: currentCounts,
        connectionTimes: currentTimes,
        projectActivities: currentActivities,
        projectUnreadCounts: currentUnread,
        currentProjectId: nextActiveProject,
        activities: nextActiveProject ? currentActivities[nextActiveProject] || [] : [],
        unreadCount: nextActiveProject ? currentUnread[nextActiveProject] || 0 : 0,
      })
    } else {
      currentCounts[projectId] = count
      set({ subscriberCounts: currentCounts })
    }
  },

  togglePanel: () => {
    set((state) => {
      const nextOpenState = !state.isPanelOpen
      const updatedUnread = { ...state.projectUnreadCounts }
      let nextUnreadCount = state.unreadCount

      if (nextOpenState) {
        // Mark all as read when opening panel
        nextUnreadCount = 0
        if (state.currentProjectId) {
          updatedUnread[state.currentProjectId] = 0
        }
      }

      return {
        isPanelOpen: nextOpenState,
        projectUnreadCounts: updatedUnread,
        unreadCount: nextUnreadCount,
      }
    })
  },

  clearAll: () => {
    const state = get()
    // Invoke abort callbacks on all active streams
    Object.values(state.streamSubscriptions).forEach((unsub) => {
      if (unsub) unsub()
    })

    set({
      unreadCount: 0,
      activities: [],
      isPanelOpen: false,
      currentProjectId: null,
      projectActivities: {},
      projectUnreadCounts: {},
      streamSubscriptions: {},
      subscriberCounts: {},
      connectionTimes: {},
    })
    console.log("Activity stream service fully cleared and closed.")
  },
}))
export default useActivityStore
