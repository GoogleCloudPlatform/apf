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

// @ts-nocheck
// @vitest-environment jsdom
import React from "react"
import { render, screen, fireEvent, cleanup } from "@testing-library/react"
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import * as matchers from "@testing-library/jest-dom/matchers"
expect.extend(matchers)
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import TopBar from "./TopBar"

import { api } from "@/lib/api"

const mockPush = vi.fn()
vi.mock("next/navigation", () => ({
  usePathname: () => "/projects/test-proj-1/workspace",
  useSearchParams: () => new URLSearchParams(),
  useRouter: () => ({
    push: mockPush,
  }),
}))

const mockUseFeature = vi.fn()
vi.mock("@/hooks/useFeature", () => ({
  useFeature: (featureId: string) => mockUseFeature(featureId),
}))

vi.mock("@/lib/api", () => ({
  api: {
    get: vi.fn().mockImplementation(async (url: string) => {
      if (url.includes("/api/v1/version")) {
        return {
          ok: true,
          json: async () => ({ version: "v1.0.43" }),
        }
      }
      return {
        ok: true,
        json: async () => ({ id: "test-proj-1", name: "Test Project" }),
      }
    }),
  },
}))

describe("TopBar Component", () => {
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
    ;(api.get as any).mockImplementation(async (url: string) => {
      if (url.includes("/api/v1/version")) {
        return {
          ok: true,
          json: async () => ({ version: "v1.0.43" }),
        }
      }
      return {
        ok: true,
        json: async () => ({ id: "test-proj-1", name: "Test Project" }),
      }
    })
  })

  afterEach(() => {
    cleanup()
  })

  const renderComponent = () =>
    render(
      <QueryClientProvider client={queryClient}>
        <TopBar />
      </QueryClientProvider>
    )

  it("renders alpha notice when show_alpha_notice is true for app_banners", () => {
    mockUseFeature.mockReturnValue({
      enabled: true,
      properties: { show_alpha_notice: true },
      isLoading: false,
      isError: false,
      error: null,
    })

    renderComponent()

    expect(mockUseFeature).toHaveBeenCalledWith("app_banners")
    expect(
      screen.getByText(/You're using the alpha version of APF\./i)
    ).toBeInTheDocument()
    expect(screen.getByText("Learn more")).toBeInTheDocument()
    expect(screen.getByText("file a bug")).toBeInTheDocument()
  })

  it("does not render alpha notice when show_alpha_notice is false", () => {
    mockUseFeature.mockReturnValue({
      enabled: true,
      properties: { show_alpha_notice: false },
      isLoading: false,
      isError: false,
      error: null,
    })

    renderComponent()

    expect(
      screen.queryByText(/You're using the alpha version of APF\./i)
    ).not.toBeInTheDocument()
  })

  it("does not render alpha notice when app_banners feature is disabled", () => {
    mockUseFeature.mockReturnValue({
      enabled: false,
      properties: { show_alpha_notice: true },
      isLoading: false,
      isError: false,
      error: null,
    })

    expect(
      screen.queryByText(/You're using the alpha version of APF\./i)
    ).not.toBeInTheDocument()
  })

  it("links Learn More to the APF documentation repository", () => {
    mockUseFeature.mockReturnValue({
      enabled: true,
      properties: { show_alpha_notice: true },
      isLoading: false,
      isError: false,
      error: null,
    })

    renderComponent()

    const learnMoreLink = screen.getByTestId("banner-learn-more-link")
    expect(learnMoreLink).toHaveAttribute(
      "href",
      "https://github.com/GoogleCloudPlatform/apf"
    )
    expect(learnMoreLink).toHaveAttribute("target", "_blank")
  })

  it("opens FeedbackModal with bug pre-selected when clicking file a bug", () => {
    mockUseFeature.mockReturnValue({
      enabled: true,
      properties: { show_alpha_notice: true },
      isLoading: false,
      isError: false,
      error: null,
    })

    renderComponent()

    const fileBugLink = screen.getByTestId("banner-file-bug-link")
    fireEvent.click(fileBugLink)

    expect(screen.getByRole("heading", { name: "Submit Feedback" })).toBeInTheDocument()
    const bugBtn = screen.getByTestId("feedback-type-bug")
    expect(bugBtn.className).toContain("border-red-500")
  })

  it("renders version label when show_version is true for app_banners", async () => {
    mockUseFeature.mockReturnValue({
      enabled: true,
      properties: { show_version: true },
      isLoading: false,
      isError: false,
      error: null,
    })

    renderComponent()

    const versionLabel = await screen.findByTestId("topbar-version-label")
    expect(versionLabel).toBeInTheDocument()
    expect(screen.getByText("v1.0.43")).toBeInTheDocument()
  })

  it("does not render version label when show_version is false", () => {
    mockUseFeature.mockReturnValue({
      enabled: true,
      properties: { show_version: false },
      isLoading: false,
      isError: false,
      error: null,
    })

    renderComponent()

    expect(screen.queryByTestId("topbar-version-label")).not.toBeInTheDocument()
    expect(screen.queryByText("v1.0.43")).not.toBeInTheDocument()
  })

  it("does not render version label when app_banners feature is disabled", () => {
    mockUseFeature.mockReturnValue({
      enabled: false,
      properties: { show_version: true },
      isLoading: false,
      isError: false,
      error: null,
    })

    renderComponent()

    expect(screen.queryByTestId("topbar-version-label")).not.toBeInTheDocument()
  })

  it("renders both alpha banner notice and version label side by side when both flags are true", async () => {
    mockUseFeature.mockReturnValue({
      enabled: true,
      properties: { show_alpha_notice: true, show_version: true },
      isLoading: false,
      isError: false,
      error: null,
    })

    renderComponent()

    expect(
      screen.getByText(/You're using the alpha version of APF\./i)
    ).toBeInTheDocument()
    const versionLabel = await screen.findByTestId("topbar-version-label")
    expect(versionLabel).toBeInTheDocument()
    expect(screen.getByText("v1.0.43")).toBeInTheDocument()
  })

  it("formats version with 'v' prefix when retrieved without 'v'", async () => {
    ;(api.get as any).mockImplementation(async (url: string) => {
      if (url.includes("/api/v1/version")) {
        return {
          ok: true,
          json: async () => ({ version: "2.5.0" }),
        }
      }
      return {
        ok: true,
        json: async () => ({ id: "test-proj-1", name: "Test Project" }),
      }
    })

    mockUseFeature.mockReturnValue({
      enabled: true,
      properties: { show_version: true },
      isLoading: false,
      isError: false,
      error: null,
    })

    renderComponent()

    const versionLabel = await screen.findByTestId("topbar-version-label")
    expect(versionLabel).toBeInTheDocument()
    expect(screen.getByText("v2.5.0")).toBeInTheDocument()
  })

  it("navigates to /docs when the search lens button is clicked", () => {
    mockUseFeature.mockReturnValue({
      enabled: false,
      properties: {},
      isLoading: false,
      isError: false,
      error: null,
    })

    renderComponent()

    const searchButton = screen.getByTestId("topbar-docs-search-button")
    expect(searchButton).toBeInTheDocument()
    fireEvent.click(searchButton)

    expect(mockPush).toHaveBeenCalledWith("/docs")
  })
})

