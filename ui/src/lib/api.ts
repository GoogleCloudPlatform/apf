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
import type { Artifact, FeedbackPayload, FeedbackResponse } from "./types"

interface RequestOptions extends RequestInit {
  headers?: Record<string, string>
  timeout?: number
}

/**
 * Standard client API wrapper mimicking Flutter's ApiClient.
 * Auto-injects Authorization headers and handles 401 actions globally.
 */
class ApiClient {
  private async request(
    url: string,
    method: string,
    body?: any,
    options: RequestOptions = {}
  ): Promise<Response> {
    const headers = { ...options.headers }
    const isFormData = typeof FormData !== "undefined" && body instanceof FormData

    // Set default Content-Type if not provided, has a body, and not FormData
    if (!isFormData && !headers["Content-Type"] && !headers["content-type"]) {
      headers["Content-Type"] = "application/json"
    }

    // Inject active JWT token from Zustand auth store state
    const token = useAuthStore.getState().token
    if (token) {
      headers["Authorization"] = `Bearer ${token}`
    }

    // Inject active organization ID if present
    const selectedOrgId = useAuthStore.getState().selectedOrgId
    if (selectedOrgId) {
      headers["X-Organization-Id"] = selectedOrgId
    }

    // Sanitize headers: HTTP headers must be ISO-8859-1 (ByteString). Strip invalid characters or null/undefined entries.
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

    const timeoutMs = options.timeout ??
      (process.env.NEXT_PUBLIC_HTTP_TIMEOUT ? parseInt(process.env.NEXT_PUBLIC_HTTP_TIMEOUT, 10) : 300000) // Default 5 minutes

    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), timeoutMs)

    const config: RequestInit = {
      ...options,
      method,
      headers: sanitizedHeaders,
      signal: controller.signal,
    }

    if (body !== undefined) {
      config.body = isFormData ? body : (typeof body === "string" ? body : JSON.stringify(body))
    }

    try {
      const response = await fetch(url, config)
      clearTimeout(timeoutId)

      // Globally handle unauthorized responses to sign user out
      if (response.status === 401) {
        useAuthStore.getState().handleUnauthorized()
      }

      return response
    } catch (error: any) {
      clearTimeout(timeoutId)
      if (error instanceof Error && error.name === "AbortError") {
        throw new Error(`Request timed out after ${timeoutMs / 1000}s`)
      }
      console.error(`API Client Error [${method} ${url}]:`, error)
      throw error
    }
  }

  public get(url: string, options?: RequestOptions): Promise<Response> {
    return this.request(url, "GET", undefined, options)
  }

  public post(url: string, body?: any, options?: RequestOptions): Promise<Response> {
    return this.request(url, "POST", body, options)
  }

  public put(url: string, body?: any, options?: RequestOptions): Promise<Response> {
    return this.request(url, "PUT", body, options)
  }

  public patch(url: string, body?: any, options?: RequestOptions): Promise<Response> {
    return this.request(url, "PATCH", body, options)
  }

  public delete(url: string, options?: RequestOptions): Promise<Response> {
    return this.request(url, "DELETE", undefined, options)
  }
}

export const api = new ApiClient()
export default api

export async function uploadSessionArtifact(
  projectId: string,
  sessionId: string,
  file: File
): Promise<Artifact> {
  const formData = new FormData()
  formData.append("file", file)

  const response = await api.post(
    `/api/v1/projects/${encodeURIComponent(projectId)}/sessions/${encodeURIComponent(sessionId)}/artifacts/upload`,
    formData
  )
  if (!response.ok) {
    let errorMsg = `Failed to upload artifact (${response.status})`
    try {
      const text = await response.text()
      if (text) {
        try {
          const errorData = JSON.parse(text)
          errorMsg = errorData.error || errorData.message || text
        } catch {
          errorMsg = text
        }
      }
    } catch {}
    throw new Error(errorMsg)
  }
  return response.json()
}

export async function submitFeedback(payload: FeedbackPayload): Promise<FeedbackResponse> {
  const response = await api.post('/api/v1/feedback', payload)
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}))
    throw new Error(errorData.error || errorData.message || `Failed to submit feedback (${response.status})`)
  }
  return response.json()
}

