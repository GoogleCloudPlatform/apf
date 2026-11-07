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

interface AppState {
  lastSelectedProjectId: string | null
  userName: string | null
  projectPhaseSessions: Record<string, Record<string, string>>
  isDebugBarOpen: boolean
  pageTitle: string

  // Actions
  setLastSelectedProjectId: (id: string | null) => void
  setUserName: (name: string | null) => void
  setSessionId: (projectId: string, phase: string, sessionId: string | null) => void
  getSessionId: (projectId: string, phase: string) => string | null
  setIsDebugBarOpen: (isOpen: boolean) => void
  toggleDebugBar: () => void
  isLeftBarExpanded: boolean
  setIsLeftBarExpanded: (isExpanded: boolean) => void
  setPageTitle: (title: string) => void
}

export const useAppStore = create<AppState>((set, get) => ({
  lastSelectedProjectId: null,
  userName: null,
  projectPhaseSessions: {},
  isDebugBarOpen: false,
  pageTitle: "Dashboard",

  setLastSelectedProjectId: (id) => set({ lastSelectedProjectId: id }),
  setUserName: (name) => set({ userName: name }),
  setPageTitle: (title) => set({ pageTitle: title }),

  setSessionId: (projectId, phase, sessionId) => {
    set((state) => {
      const updatedSessions = { ...state.projectPhaseSessions }
      if (!updatedSessions[projectId]) {
        updatedSessions[projectId] = {}
      }

      if (sessionId === null) {
        const { [phase]: _, ...remaining } = updatedSessions[projectId]
        updatedSessions[projectId] = remaining
      } else {
        updatedSessions[projectId] = {
          ...updatedSessions[projectId],
          [phase]: sessionId,
        }
      }

      return { projectPhaseSessions: updatedSessions }
    })
  },

  getSessionId: (projectId, phase) => {
    return get().projectPhaseSessions[projectId]?.[phase] || null
  },

  setIsDebugBarOpen: (isOpen) => set({ isDebugBarOpen: isOpen }),
  toggleDebugBar: () => set((state) => ({ isDebugBarOpen: !state.isDebugBarOpen })),
  isLeftBarExpanded: false,
  setIsLeftBarExpanded: (isExpanded) => set({ isLeftBarExpanded: isExpanded }),
}))
export default useAppStore
