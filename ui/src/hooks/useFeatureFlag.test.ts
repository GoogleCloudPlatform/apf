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

// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import React from "react"
import { useFeatureFlag, fetchFeature } from "./useFeatureFlag"
import api from "@/lib/api"

vi.mock("@/lib/api", () => ({
  default: {
    get: vi.fn(),
  },
}))

describe("useFeatureFlag Hook & fetchFeature", () => {
  let queryClient: QueryClient

  beforeEach(() => {
    vi.clearAllMocks()
    queryClient = new QueryClient({
      defaultOptions: {
        queries: {
          retry: false,
        },
      },
    })
  })

  const wrapper = ({ children }: { children: React.ReactNode }) => (
    React.createElement(QueryClientProvider, { client: queryClient }, children)
  )

  it("fetches feature flag successfully", async () => {
    const mockFeature = {
      id: "canvas_v2",
      enabled: true,
      properties: { show_badge: true, max_retry_count: 3 },
    }

    vi.mocked(api.get).mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => mockFeature,
    } as Response)

    const result = await fetchFeature("canvas_v2")
    expect(result).toEqual(mockFeature)
    expect(api.get).toHaveBeenCalledWith("/api/v1/features/canvas_v2")
  })

  it("handles 404 not found gracefully in fetchFeature", async () => {
    vi.mocked(api.get).mockResolvedValueOnce({
      ok: false,
      status: 404,
      statusText: "Not Found",
    } as Response)

    const result = await fetchFeature("canvas_v2")
    expect(result).toEqual({
      id: "canvas_v2",
      enabled: false,
      properties: {},
    })
  })

  it("returns enabled feature data from hook", async () => {
    const mockFeature = {
      id: "canvas_v2",
      enabled: true,
      properties: { show_badge: true },
    }

    vi.mocked(api.get).mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => mockFeature,
    } as Response)

    const { result } = renderHook(() => useFeatureFlag("canvas_v2"), { wrapper })

    await waitFor(() => {
      expect(result.current.enabled).toBe(true)
    })

    expect(result.current.properties).toEqual({ show_badge: true })
  })

  it("falls back to default disabled properties when API fails", async () => {
    vi.mocked(api.get).mockResolvedValueOnce({
      ok: false,
      status: 500,
      statusText: "Internal Server Error",
    } as Response)

    const { result } = renderHook(
      () => useFeatureFlag("canvas_v2", { default_badge: false }),
      { wrapper }
    )

    await waitFor(() => {
      expect(result.current.enabled).toBe(false)
    })

    expect(result.current.properties).toEqual({ default_badge: false })
  })
})
