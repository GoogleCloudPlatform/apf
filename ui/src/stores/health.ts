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

interface HealthState {
  isAvailable: boolean
  isChecking: boolean
  isApiUp: boolean
  isAiUp: boolean
  errorMessage: string | null
  checkHealth: () => Promise<boolean>
}

export const useHealthStore = create<HealthState>((set) => ({
  isAvailable: true,
  isChecking: true,
  isApiUp: false,
  isAiUp: false,
  errorMessage: null,

  checkHealth: async () => {
    set({ isChecking: true })
    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), 3000)

    try {
      // Direct health endpoint call (resolved via Next.js Dev Proxy in dev)
      const response = await fetch("/health", {
        method: "GET",
        signal: controller.signal,
      })

      clearTimeout(timeoutId)

      // Service returns 200 (healthy) or 503 (some services down but returns details JSON)
      if (response.status === 200 || response.status === 503) {
        try {
          const data = await response.json()
          if (data && data.services) {
            const isApiUp = data.services.api === "UP"
            const isAiUp = data.services.ai === "UP"
            const isAvailable = isApiUp && isAiUp

            let errorMessage: string | null = null
            if (!isAvailable) {
              const down: string[] = []
              if (!isApiUp) down.push("API")
              if (!isAiUp) down.push("AI")
              errorMessage = `${down.join(" and ")} service${
                down.length > 1 ? "s are" : " is"
              } not running.`
            }

            set({
              isApiUp,
              isAiUp,
              isAvailable,
              errorMessage,
              isChecking: false,
            })
            return isAvailable
          }
        } catch (_) {
          // Parsing failure fallbacks to status codes check below
        }
      }

      const isApiUp = response.status === 200
      set({
        isApiUp,
        isAiUp: false,
        isAvailable: false,
        errorMessage: `Backend reports service status: ${response.status}`,
        isChecking: false,
      })
      return false
    } catch (e) {
      console.warn("Backend health check failed:", e)
      clearTimeout(timeoutId)
      set({
        isApiUp: false,
        isAiUp: false,
        isAvailable: false,
        errorMessage: "Could not connect to backend services. Please make sure they are running.",
        isChecking: false,
      })
      return false
    }
  },
}))
export default useHealthStore
