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

import { useAuthStore } from "@/stores/auth"

interface SSEOptions {
  method?: "GET" | "POST"
  body?: string
  headers?: Record<string, string>
  onMessage: (event: string, data: string) => void
  onError?: (error: any) => void
  onOpen?: () => void
  onKeepAlive?: () => void
}

/**
 * Custom SSE parser using browser Fetch and Streams API.
 * Bypasses the default browser EventSource header restrictions.
 */
export function connectSSE(urlOrFn: string | (() => string), options: SSEOptions): () => void {
  let isAborted = false
  let abortController = new AbortController()
  let reconnectTimeoutId: NodeJS.Timeout | null = null
  let readTimeoutId: NodeJS.Timeout | null = null
  let reconnectAttempts = 0

  const resetReadTimeout = (url: string) => {
    if (readTimeoutId) {
      clearTimeout(readTimeoutId)
    }
    readTimeoutId = setTimeout(() => {
      console.warn(`SSE connection to ${url} timed out (no data received for 60s). Reconnecting...`)
      abortController.abort()
    }, 60000)
  }

  const clearReadTimeout = () => {
    if (readTimeoutId) {
      clearTimeout(readTimeoutId)
      readTimeoutId = null
    }
  }

  const connect = async () => {
    if (isAborted) return

    const url = typeof urlOrFn === "function" ? urlOrFn() : urlOrFn
    const headers: Record<string, string> = { ...options.headers }
    const token = useAuthStore.getState().token
    if (token) {
      headers["Authorization"] = `Bearer ${token}`
    }

    // Inject active organization ID if present
    const selectedOrgId = useAuthStore.getState().selectedOrgId
    if (selectedOrgId) {
      headers["X-Organization-Id"] = selectedOrgId
    }

    const sanitizedHeaders: Record<string, string> = {}
    for (const [key, value] of Object.entries(headers)) {
      if (
        value !== undefined &&
        value !== null &&
        typeof key === "string" &&
        !/[^\x00-\xFF]/.test(key) &&
        !/[^\x00-\xFF]/.test(String(value))
      ) {
        sanitizedHeaders[key] = String(value)
      }
    }

    try {
      console.log("SSE Fetching:", url, "Headers:", sanitizedHeaders)
      const response = await fetch(url, {
        method: options.method || "GET",
        body: options.body,
        headers: sanitizedHeaders,
        signal: abortController.signal,
      })

      if (!response.ok) {
        if (response.status === 401) {
          useAuthStore.getState().handleUnauthorized()
          return
        }
        const error = new Error(`SSE HTTP Error status: ${response.status}`)
        ;(error as any).status = response.status
        throw error
      }

      if (options.onOpen) {
        options.onOpen()
      }

      // Reset reconnect attempts on successful handshake
      reconnectAttempts = 0

      const reader = response.body?.getReader()
      if (!reader) {
        throw new Error("Response body is not readable")
      }

      resetReadTimeout(url)

      const decoder = new TextDecoder()
      let buffer = ""
      let currentEvent = "message"
      let dataBuffer: string[] = []

      while (true) {
        const { done, value } = await reader.read()
        if (done || isAborted) {
          clearReadTimeout()
          if (!isAborted && done) {
            console.warn(`SSE stream to ${url} closed cleanly by server. Reconnecting...`)
            scheduleReconnect()
          }
          break
        }

        resetReadTimeout(url)

        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split(/\r?\n/)
        
        // Keep the trailing open line fragment in the buffer
        buffer = lines.pop() || ""

        for (const line of lines) {
          if (line.trim() === "") {
            // Emptiness marks message block dispatch boundary
            if (dataBuffer.length > 0) {
              const dataStr = dataBuffer.join("\n")
              options.onMessage(currentEvent, dataStr)
              dataBuffer = []
              currentEvent = "message" // Reset default event type
            }
            continue
          }

          if (line.startsWith("event:")) {
            currentEvent = line.substring(line.startsWith("event: ") ? 7 : 6).trim()
          } else if (line.startsWith("data:")) {
            dataBuffer.push(line.substring(line.startsWith("data: ") ? 6 : 5))
          } else if (line.startsWith(":")) {
            options.onKeepAlive?.()
          }
        }
      }
    } catch (error: any) {
      clearReadTimeout()
      if (error.name === "AbortError") {
        if (isAborted) return
      }

      if (options.onError) {
        options.onError(error)
      }
      
      // Stop reconnecting on fatal HTTP status errors
      const status = error.status
      if (status && (status === 400 || status === 403 || status === 404)) {
        console.error(`SSE connection permanently terminated due to fatal HTTP status: ${status}`)
        return
      }
      
      scheduleReconnect()
    }
  }

  const scheduleReconnect = () => {
    if (isAborted) return
    
    reconnectAttempts++
    // Exponential backoff capped at 60 seconds with simple limit safety
    const delaySeconds = Math.min(Math.pow(2, Math.min(reconnectAttempts, 6)) * 2, 60)
    const jitterMs = Math.floor(Math.random() * 1000)
    const delay = delaySeconds * 1000 + jitterMs

    const url = typeof urlOrFn === "function" ? urlOrFn() : urlOrFn
    console.log(`SSE connection lost to ${url}. Reconnecting in ${delaySeconds}s (attempt ${reconnectAttempts})`)

    reconnectTimeoutId = setTimeout(() => {
      // Re-trigger new connection cycle
      abortController = new AbortController()
      connect()
    }, delay)
  }

  // Kick off the initial connection loop
  connect()

  // Return unsubscribe/abort callback
  return () => {
    isAborted = true
    clearReadTimeout()
    abortController.abort()
    if (reconnectTimeoutId) {
      clearTimeout(reconnectTimeoutId)
    }
  }
}
export default connectSSE
