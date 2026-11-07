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
import { api } from "@/lib/api"
import { useAuthStore } from "./auth"

export type ThemeMode = "light" | "dark" | "system"

interface ThemeState {
  themeMode: ThemeMode
  init: () => void
  setThemeMode: (mode: ThemeMode) => Promise<void>
  toggleTheme: () => Promise<void>
  loadThemeFromBackend: (userId: string) => Promise<void>
  saveThemeToBackend: (userId: string, mode: ThemeMode) => Promise<void>
  _applyTheme: (mode: ThemeMode) => void
}

const THEME_KEY = "theme_mode"

export const useThemeStore = create<ThemeState>((set, get) => ({
  themeMode: "system",

  init: () => {
    if (typeof window === "undefined") return
    const savedTheme = localStorage.getItem(THEME_KEY) as ThemeMode | null
    if (savedTheme === "light" || savedTheme === "dark" || savedTheme === "system") {
      set({ themeMode: savedTheme })
      get()._applyTheme(savedTheme)
    } else {
      set({ themeMode: "system" })
      get()._applyTheme("system")
    }
  },

  setThemeMode: async (mode: ThemeMode) => {
    set({ themeMode: mode })
    if (typeof window !== "undefined") {
      localStorage.setItem(THEME_KEY, mode)
    }
    get()._applyTheme(mode)

    const email = useAuthStore.getState().email
    if (email) {
      await get().saveThemeToBackend(email, mode)
    }
  },

  toggleTheme: async () => {
    const currentMode = get().themeMode
    let isDark = currentMode === "dark"

    if (currentMode === "system" && typeof window !== "undefined") {
      isDark = window.matchMedia("(prefers-color-scheme: dark)").matches
    }

    const newMode: ThemeMode = isDark ? "light" : "dark"
    await get().setThemeMode(newMode)
  },

  loadThemeFromBackend: async (userId: string) => {
    try {
      const response = await api.get(`/api/v1/users/${encodeURIComponent(userId)}/preferences`)
      if (response.ok) {
        const decoded = await response.json()
        const backendTheme = decoded.theme as ThemeMode
        if (backendTheme === "light" || backendTheme === "dark") {
          set({ themeMode: backendTheme })
          if (typeof window !== "undefined") {
            localStorage.setItem(THEME_KEY, backendTheme)
          }
          get()._applyTheme(backendTheme)
        }
      }
    } catch (e) {
      console.warn("Failed to load user theme from backend preferences:", e)
    }
  },

  saveThemeToBackend: async (userId: string, mode: ThemeMode) => {
    try {
      const themeStr = mode === "system" ? "dark" : mode // Default system request to dark on server side
      await api.put(`/api/v1/users/${encodeURIComponent(userId)}/preferences`, {
        theme: themeStr,
      })
    } catch (e) {
      console.warn("Failed to save user theme to backend preferences:", e)
    }
  },

  // Helper function to dynamically add/remove CSS classes on the html root node
  _applyTheme: (mode: ThemeMode) => {
    if (typeof window === "undefined") return
    const root = window.document.documentElement
    root.classList.remove("light", "dark")

    if (mode === "system") {
      const systemTheme = window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light"
      root.classList.add(systemTheme)
    } else {
      root.classList.add(mode)
    }
  },
}))
export default useThemeStore
