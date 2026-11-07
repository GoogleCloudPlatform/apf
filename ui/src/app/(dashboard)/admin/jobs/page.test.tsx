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
import AdminJobsPage from "./page"
import { useAuthStore } from "@/stores/auth"
import { useAppStore } from "@/stores/app"
import { toast } from "sonner"

vi.mock("next/navigation", () => ({
  useRouter: () => ({
    push: vi.fn(),
    replace: vi.fn(),
    back: vi.fn(),
  }),
  usePathname: () => "/admin/jobs",
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

describe("AdminJobsPage", () => {
  const mockSetPageTitle = vi.fn()
  const mockUseAuthStore = useAuthStore as unknown as ReturnType<typeof vi.fn>
  const mockUseAppStore = useAppStore as unknown as ReturnType<typeof vi.fn>

  const mockJobsCatalog = [
    {
      name: "global-workspace-provisioning",
      description: "Provision shared skills and framework assets into global workspace",
      parameters: [
        {
          name: "dryRun",
          type: "bool",
          description: "Simulate operations without writing files",
          required: false,
        },
      ],
    },
    {
      name: "nfs-audit",
      description: "Scan tenant workspaces and update folder metadata",
      parameters: [
        {
          name: "dryRun",
          type: "bool",
          description: "Simulate scan without persisting metadata",
          required: false,
        },
      ],
    },
    {
      name: "hanging-cleanup",
      description: "Purge dangling workspace folders from deleted projects",
      parameters: [
        {
          name: "dryRun",
          type: "bool",
          description: "Scan orphaned folders without deleting files",
          required: false,
        },
      ],
    },
  ]

  const mockExecutionsList = {
    executions: [
      {
        id: "exec-101",
        name: "projects/test-proj/locations/us-central1/jobs/apf-stg-maintenance/executions/exec-101",
        jobName: "global-workspace-provisioning",
        dryRun: true,
        status: "SUCCEEDED",
        createTime: "2026-08-31T12:00:00Z",
        startTime: "2026-08-31T12:00:02Z",
        completionTime: "2026-08-31T12:00:15Z",
        durationSeconds: 13,
        durationFormatted: "13s",
        logUri: "https://console.cloud.google.com/logs/query;query=exec-101",
      },
      {
        id: "exec-102",
        name: "projects/test-proj/locations/us-central1/jobs/apf-stg-maintenance/executions/exec-102",
        jobName: "nfs-audit",
        dryRun: false,
        status: "FAILED",
        createTime: "2026-08-31T12:10:00Z",
        startTime: "2026-08-31T12:10:02Z",
        completionTime: "2026-08-31T12:10:45Z",
        durationSeconds: 43,
        durationFormatted: "43s",
        errorMessage: "NFS mount timeout while scanning tenant folders",
        logUri: "https://console.cloud.google.com/logs/query;query=exec-102",
      },
      {
        id: "exec-103",
        name: "projects/test-proj/locations/us-central1/jobs/apf-stg-maintenance/executions/exec-103",
        jobName: "hanging-cleanup",
        dryRun: true,
        status: "RUNNING",
        createTime: "2026-08-31T12:20:00Z",
        startTime: "2026-08-31T12:20:02Z",
        durationSeconds: 15,
        durationFormatted: "15s",
      },
    ],
    nextPageToken: "",
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockUseAppStore.mockReturnValue({
      setPageTitle: mockSetPageTitle,
    })
    mockUseAuthStore.mockReturnValue({
      token: "mock-token",
      isSuperAdmin: true,
      isCheckingAuth: false,
    })

    global.fetch = vi.fn().mockImplementation((url: string, init?: RequestInit) => {
      if (url.includes("/api/v1/admin/jobs/executions")) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: () => Promise.resolve(mockExecutionsList),
        } as Response)
      }
      if (url.includes("/run")) {
        // Run endpoint
        return Promise.resolve({
          ok: true,
          status: 202,
          json: () =>
            Promise.resolve({
              status: "PENDING",
              executionName: "projects/test-proj/locations/us-central1/jobs/apf-stg-maintenance/executions/exec-new",
              jobName: "global-workspace-provisioning",
            }),
        } as Response)
      }
      if (url === "/api/v1/admin/jobs" || url.startsWith("/api/v1/admin/jobs")) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: () => Promise.resolve(mockJobsCatalog),
        } as Response)
      }
      return Promise.reject(new Error(`Unhandled URL: ${url}`))
    })
  })

  afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
  })

  it("renders UnauthorizedScreen when user is not a super administrator", async () => {
    mockUseAuthStore.mockReturnValue({
      token: "mock-token",
      isSuperAdmin: false,
      isCheckingAuth: false,
    })

    render(<AdminJobsPage />)

    expect(screen.getByText("Super Administrator Access Required")).toBeDefined()
    expect(global.fetch).not.toHaveBeenCalled()
  })

  it("renders page header, KPI cards, jobs catalog, and execution history for Super Admin", async () => {
    render(<AdminJobsPage />)

    await waitFor(() => {
      expect(screen.getByText("System Jobs & Maintenance")).toBeDefined()
    })

    // Verify KPI cards
    await waitFor(() => {
      expect(screen.getByText("Registered Jobs")).toBeDefined()
      expect(screen.getByText("Active Executions")).toBeDefined()
      expect(screen.getByText("Succeeded Runs")).toBeDefined()
      expect(screen.getByText("Failed Runs")).toBeDefined()
    })

    // Verify catalog items rendered
    expect(screen.getAllByText("global-workspace-provisioning").length).toBeGreaterThan(0)
    expect(screen.getAllByText("nfs-audit").length).toBeGreaterThan(0)
    expect(screen.getAllByText("hanging-cleanup").length).toBeGreaterThan(0)

    // Verify execution items in table
    expect(screen.getByText("exec-101")).toBeDefined()
    expect(screen.getByText("exec-102")).toBeDefined()
    expect(screen.getByText("exec-103")).toBeDefined()

    // Verify status badges
    expect(screen.getAllByText("Succeeded").length).toBeGreaterThan(0)
    expect(screen.getAllByText("Failed").length).toBeGreaterThan(0)
    expect(screen.getAllByText("Running").length).toBeGreaterThan(0)
  })

  it("opens JobRunModal when clicking 'Configure & Run' and dispatches dry-run execution", async () => {
    render(<AdminJobsPage />)

    await waitFor(() => {
      expect(screen.getAllByText("global-workspace-provisioning").length).toBeGreaterThan(0)
    })

    const runButtons = screen.getAllByRole("button", { name: /Configure & Run/i })
    fireEvent.click(runButtons[0])

    // Verify modal appears
    await waitFor(() => {
      expect(screen.getByRole("dialog")).toBeDefined()
      expect(screen.getByText(/Run System Job:/i)).toBeDefined()
    })

    // Verify dry run is checked by default
    const dryRunCheckbox = screen.getByLabelText(/Dry Run Mode/i) as HTMLInputElement
    expect(dryRunCheckbox.checked).toBe(true)

    // Submit execution
    const submitBtn = screen.getByRole("button", { name: /Execute Dry Run/i })
    fireEvent.click(submitBtn)

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        "/api/v1/admin/jobs/global-workspace-provisioning/run",
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify({ dryRun: true }),
        })
      )
      expect(toast.success).toHaveBeenCalledWith(
        expect.stringContaining("Job global-workspace-provisioning triggered successfully")
      )
    })
  })

  it("toggles dry-run off in modal and executes live job", async () => {
    render(<AdminJobsPage />)

    await waitFor(() => {
      expect(screen.getAllByText("nfs-audit").length).toBeGreaterThan(0)
    })

    const runButtons = screen.getAllByRole("button", { name: /Configure & Run/i })
    fireEvent.click(runButtons[1]) // nfs-audit

    await waitFor(() => {
      expect(screen.getByRole("dialog")).toBeDefined()
    })

    const dryRunCheckbox = screen.getByLabelText(/Dry Run Mode/i) as HTMLInputElement
    fireEvent.click(dryRunCheckbox)
    expect(dryRunCheckbox.checked).toBe(false)

    // Warning alert should be shown
    expect(screen.getByText(/Live execution will apply actual mutations/i)).toBeDefined()

    const submitBtn = screen.getByRole("button", { name: /Execute Live Job/i })
    fireEvent.click(submitBtn)

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        "/api/v1/admin/jobs/nfs-audit/run",
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify({ dryRun: false }),
        })
      )
      expect(toast.success).toHaveBeenCalled()
    })
  })

  it("filters executions table by search query", async () => {
    render(<AdminJobsPage />)

    await waitFor(() => {
      expect(screen.getByText("exec-101")).toBeDefined()
      expect(screen.getByText("exec-102")).toBeDefined()
    })

    const searchInput = screen.getByPlaceholderText("Search by job or ID...")
    fireEvent.change(searchInput, { target: { value: "nfs-audit" } })

    expect(screen.queryByText("exec-101")).toBeNull()
    expect(screen.getByText("exec-102")).toBeDefined()
  })

  it("filters executions table by status dropdown", async () => {
    render(<AdminJobsPage />)

    await waitFor(() => {
      expect(screen.getByText("exec-101")).toBeDefined()
      expect(screen.getByText("exec-102")).toBeDefined()
    })

    const filterSelect = screen.getByLabelText("Filter by status")
    fireEvent.change(filterSelect, { target: { value: "FAILED" } })

    expect(screen.queryByText("exec-101")).toBeNull()
    expect(screen.getByText("exec-102")).toBeDefined()
  })

  it("refreshes catalog and execution history when clicking Refresh Data button", async () => {
    render(<AdminJobsPage />)

    await waitFor(() => {
      expect(screen.getByText("Refresh Data")).toBeDefined()
    })

    const refreshBtn = screen.getByRole("button", { name: /Refresh Data/i })
    fireEvent.click(refreshBtn)

    await waitFor(() => {
      expect(toast.success).toHaveBeenCalledWith("Job catalog and execution history refreshed")
    })
  })

  it("renders log link with target=_blank and rel=noopener noreferrer", async () => {
    render(<AdminJobsPage />)

    await waitFor(() => {
      expect(screen.getByText("exec-101")).toBeDefined()
    })

    const logLinks = screen.getAllByRole("link", { name: /Logs/i })
    expect(logLinks.length).toBeGreaterThan(0)
    expect(logLinks[0].getAttribute("href")).toBe(
      "https://console.cloud.google.com/logs/query;query=exec-101"
    )
    expect(logLinks[0].getAttribute("target")).toBe("_blank")
    expect(logLinks[0].getAttribute("rel")).toBe("noopener noreferrer")
  })
})
