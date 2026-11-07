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
import React from "react"
import { render, screen, cleanup, fireEvent, waitFor } from "@testing-library/react"
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import AIUsageAnalyticsPage from "./page"
import { useAuthStore } from "@/stores/auth"
import { useAppStore } from "@/stores/app"
import { toast } from "sonner"

vi.mock("next/navigation", () => ({
  useRouter: () => ({
    push: vi.fn(),
    replace: vi.fn(),
    back: vi.fn(),
  }),
  usePathname: () => "/admin/ai-usage",
}))

vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

vi.mock("@/stores/auth", () => ({
  useAuthStore: vi.fn(),
}))

vi.mock("@/stores/app", () => ({
  useAppStore: vi.fn(),
}))

describe("AIUsageAnalyticsPage", () => {
  const mockSetPageTitle = vi.fn()
  const mockUseAuthStore = useAuthStore as unknown as ReturnType<typeof vi.fn>
  const mockUseAppStore = useAppStore as unknown as ReturnType<typeof vi.fn>

  const mockTelemetryData = {
    organizationId: "org-123",
    startDate: "2026-07-10",
    endDate: "2026-08-10",
    promptTokens: 15000,
    completionTokens: 5000,
    totalTokens: 20000,
    costUSD: 0.1234,
    turnCount: 50,
    averageTokensPerTurn: 400,
    totalUsers: 2,
    totalProjects: 2,
    userBreakdown: [
      {
        userId: "user-1@example.com",
        promptTokens: 10000,
        completionTokens: 3000,
        totalTokens: 13000,
        costUSD: 0.08,
        sessionCount: 5,
      },
      {
        userId: "user-2@example.com",
        promptTokens: 5000,
        completionTokens: 2000,
        totalTokens: 7000,
        costUSD: 0.0434,
        sessionCount: 3,
      },
    ],
    projectBreakdown: [
      {
        projectId: "proj-alpha",
        promptTokens: 12000,
        completionTokens: 4000,
        totalTokens: 16000,
        costUSD: 0.10,
        sessionCount: 6,
      },
      {
        projectId: "proj-beta",
        promptTokens: 3000,
        completionTokens: 1000,
        totalTokens: 4000,
        costUSD: 0.0234,
        sessionCount: 2,
      },
    ],
    sessionBreakdown: [
      {
        sessionId: "sess-101",
        userId: "user-1@example.com",
        projectId: "proj-alpha",
        promptTokens: 4000,
        completionTokens: 1000,
        totalTokens: 5000,
        costUSD: 0.03,
        turnCount: 12,
      },
    ],
  }

  beforeEach(() => {
    vi.clearAllMocks()

    if (typeof window !== "undefined") {
      window.URL.createObjectURL = vi.fn().mockReturnValue("blob:http://localhost/fake-csv-blob")
      window.URL.revokeObjectURL = vi.fn()
    }

    global.fetch = vi.fn().mockImplementation((url: string) => {
      if (url.includes("format=csv")) {
        return Promise.resolve({
          ok: true,
          status: 200,
          blob: async () => new Blob(["Type,ID,Total Tokens\nSummary,org-123,20000"], { type: "text/csv" }),
        })
      }
      return Promise.resolve({
        ok: true,
        status: 200,
        json: async () => mockTelemetryData,
      })
    }) as any

    mockUseAppStore.mockReturnValue({
      setPageTitle: mockSetPageTitle,
    })

    mockUseAuthStore.mockReturnValue({
      token: "fake-admin-token",
      isSuperAdmin: true,
      isAdmin: true,
      isCheckingAuth: false,
      selectedOrgId: "org-123",
      hasRole: (role: string) => role === "org:admin",
    })
  })

  afterEach(() => {
    cleanup()
  })

  it("renders loading indicator when auth is checking", () => {
    mockUseAuthStore.mockReturnValue({
      token: null,
      isSuperAdmin: false,
      isAdmin: false,
      isCheckingAuth: true,
      selectedOrgId: null,
    })

    render(<AIUsageAnalyticsPage />)
    expect(screen.getByText("Verifying access...")).not.toBeNull()
  })

  it("renders UnauthorizedScreen and does NOT trigger fetch when user is not admin", () => {
    mockUseAuthStore.mockReturnValue({
      token: "fake-user-token",
      isSuperAdmin: false,
      isAdmin: false,
      isCheckingAuth: false,
      selectedOrgId: "org-123",
      hasRole: () => false,
    })

    render(<AIUsageAnalyticsPage />)

    expect(screen.getByText("Admin Access Required")).not.toBeNull()
    expect(global.fetch).not.toHaveBeenCalled()
  })

  it("renders header page title and KPI summary cards for authorized Super Admin", async () => {
    render(<AIUsageAnalyticsPage />)

    expect(mockSetPageTitle).toHaveBeenCalledWith("Admin: AI Usage & Cost")

    await waitFor(() => {
      expect(screen.getByTestId("ai-usage-analytics-page")).not.toBeNull()
    })

    expect(screen.getByTestId("kpi-total-tokens")).not.toBeNull()
    expect(screen.getByText("20,000")).not.toBeNull()
    expect(screen.getByTestId("kpi-total-cost")).not.toBeNull()
    expect(screen.getByText("$0.12")).not.toBeNull()
    expect(screen.getByTestId("kpi-active-turns")).not.toBeNull()
    expect(screen.getByText("50")).not.toBeNull()
    expect(screen.getByTestId("kpi-total-users-projects")).not.toBeNull()
  })

  it("denies access for Org Admin when isSuperAdmin is false", () => {
    mockUseAuthStore.mockReturnValue({
      token: "fake-org-admin-token",
      isSuperAdmin: false,
      isAdmin: true,
      isCheckingAuth: false,
      selectedOrgId: "org-123",
      hasRole: (role: string) => role === "org:admin",
    })

    render(<AIUsageAnalyticsPage />)

    expect(screen.getByText("Admin Access Required")).not.toBeNull()
    expect(global.fetch).not.toHaveBeenCalled()
  })

  it("updates date filter preset and re-fetches telemetry metrics", async () => {
    render(<AIUsageAnalyticsPage />)

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalled()
    })

    const preset7dBtn = screen.getByTestId("preset-7d")
    fireEvent.click(preset7dBtn)

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining("startDate="),
        expect.anything()
      )
    })
  })

  it("shows custom date range inputs when Custom Range preset is selected", async () => {
    render(<AIUsageAnalyticsPage />)

    const customBtn = screen.getByTestId("preset-custom")
    fireEvent.click(customBtn)

    expect(screen.getByLabelText("Start Date:")).not.toBeNull()
    expect(screen.getByLabelText("End Date:")).not.toBeNull()
  })

  it("switches between By User, By Project, and By Session breakdown tabs", async () => {
    render(<AIUsageAnalyticsPage />)

    await waitFor(() => {
      expect(screen.getByText("user-1@example.com")).not.toBeNull()
    })

    // Switch to By Project tab
    const projectTab = screen.getByTestId("tab-project")
    fireEvent.click(projectTab)

    expect(screen.getByText("proj-alpha")).not.toBeNull()
    expect(screen.getByText("proj-beta")).not.toBeNull()

    // Switch to By Session tab
    const sessionTab = screen.getByTestId("tab-session")
    fireEvent.click(sessionTab)

    expect(screen.getByText("sess-101")).not.toBeNull()
  })

  it("filters table results using search input", async () => {
    render(<AIUsageAnalyticsPage />)

    await waitFor(() => {
      expect(screen.getByText("user-1@example.com")).not.toBeNull()
      expect(screen.getByText("user-2@example.com")).not.toBeNull()
    })

    const searchInput = screen.getByTestId("breakdown-search-input")
    fireEvent.change(searchInput, { target: { value: "user-1" } })

    expect(screen.getByText("user-1@example.com")).not.toBeNull()
    expect(screen.queryByText("user-2@example.com")).toBeNull()
  })

  it("triggers CSV export download stream when Export CSV button is clicked", async () => {
    render(<AIUsageAnalyticsPage />)

    await waitFor(() => {
      expect(screen.getByTestId("kpi-total-tokens")).not.toBeNull()
      const btn = screen.getByTestId("export-csv-btn") as HTMLButtonElement
      expect(btn.disabled).toBe(false)
    })

    const exportBtn = screen.getByTestId("export-csv-btn")
    fireEvent.click(exportBtn)

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining("format=csv"),
        expect.anything()
      )
    })

    expect(toast.success).toHaveBeenCalledWith("CSV usage report downloaded successfully")
  })

  it("displays toast error when CSV export fails", async () => {
    mockUseAuthStore.mockReturnValue({
      token: "fake-admin-token",
      isSuperAdmin: true,
      isAdmin: true,
      isCheckingAuth: false,
      selectedOrgId: "org-123",
      hasRole: (role: string) => role === "org:admin",
    })

    global.fetch = vi.fn().mockImplementation((url: string) => {
      if (url.includes("format=csv")) {
        return Promise.resolve({
          ok: false,
          status: 500,
        })
      }
      return Promise.resolve({
        ok: true,
        status: 200,
        json: async () => mockTelemetryData,
      })
    }) as any

    render(<AIUsageAnalyticsPage />)

    await waitFor(() => {
      expect(screen.getByTestId("kpi-total-tokens")).not.toBeNull()
      const btn = screen.getByTestId("export-csv-btn") as HTMLButtonElement
      expect(btn.disabled).toBe(false)
    })

    const exportBtn = screen.getByTestId("export-csv-btn")
    fireEvent.click(exportBtn)

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalled()
    })
  })

  it("renders empty state metrics ($0.0000, 0 tokens) when organization has no logged LLM events", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        promptTokens: 0,
        completionTokens: 0,
        totalTokens: 0,
        costUSD: 0,
        turnCount: 0,
        userBreakdown: [],
        projectBreakdown: [],
        sessionBreakdown: [],
      }),
    }) as any

    render(<AIUsageAnalyticsPage />)

    await waitFor(() => {
      expect(screen.getByTestId("kpi-total-tokens").textContent).toContain("0")
      expect(screen.getByTestId("kpi-total-cost").textContent).toContain("$0.00")
      expect(screen.getByText("No usage telemetry found")).not.toBeNull()
    })
  })

  it("displays error alert banner when telemetry API fetch fails", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 503,
      json: async () => ({ error: { message: "Database connection failed" } }),
    }) as any

    render(<AIUsageAnalyticsPage />)

    await waitFor(() => {
      expect(screen.getByText("Database connection failed")).not.toBeNull()
    })
  })

  it("handles null/undefined fields in breakdown items gracefully without crashing", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        promptTokens: 100,
        completionTokens: 50,
        totalTokens: 150,
        costUSD: 0.001,
        turnCount: 1,
        userBreakdown: [{ userId: null, promptTokens: null, completionTokens: null, totalTokens: null, costUSD: null, sessionCount: null }],
        projectBreakdown: [],
        sessionBreakdown: [],
      }),
    }) as any

    render(<AIUsageAnalyticsPage />)

    await waitFor(() => {
      expect(screen.getByTestId("kpi-total-tokens")).not.toBeNull()
    })
  })
})
