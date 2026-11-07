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
import NFSAdminPage from "./page"
import { useAuthStore } from "@/stores/auth"
import { useAppStore } from "@/stores/app"
import { toast } from "sonner"

vi.mock("next/navigation", () => ({
  useRouter: () => ({
    push: vi.fn(),
    replace: vi.fn(),
    back: vi.fn(),
  }),
  usePathname: () => "/admin/nfs",
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

describe("NFSAdminPage", () => {
  const mockSetPageTitle = vi.fn()
  const mockUseAuthStore = useAuthStore as unknown as ReturnType<typeof vi.fn>
  const mockUseAppStore = useAppStore as unknown as ReturnType<typeof vi.fn>

  const mockVolumeData = {
    volume: {
      totalBytes: 1024 * 1024 * 1024 * 500, // 500 GB
      usedBytes: 1024 * 1024 * 1024 * 100,  // 100 GB
      availableBytes: 1024 * 1024 * 1024 * 400, // 400 GB
      mountPath: "/workspaces",
      isHealthy: true,
    },
    totalFolders: 2,
    status: "SUCCESS",
  }

  const mockFoldersData = {
    folders: [
      {
        folderId: "folder-1",
        organizationId: "org-1",
        projectId: "proj-1",
        workspacePath: "/workspaces/organizations/org-1/user-1/proj-1",
        quotaSizeBytes: 1024 * 1024 * 1024 * 10,
        usedSizeBytes: 1024 * 1024 * 1024 * 2,
        fileCount: 150,
        retentionPolicy: "30d",
        lastAuditedAt: "2026-08-20T10:00:00Z",
        createdAt: "2026-08-01T10:00:00Z",
        updatedAt: "2026-08-20T10:00:00Z",
      },
    ],
    total: 1,
    status: "SUCCESS",
  }

  const mockOrgsData = [
    { id: "org-1", name: "Acme Corp" },
  ]

  const mockHangingData = {
    hangingProjects: [
      {
        indexKey: "org1-user1-proj1",
        orgId: "org1",
        userId: "user1",
        projectId: "proj1",
        path: "/workspaces/organizations/org1/user1/proj1",
        sizeBytes: 1024 * 1024 * 50,
        fileCount: 12,
      },
    ],
    total: 1,
    status: "SUCCESS",
  }

  beforeEach(() => {
    vi.clearAllMocks()

    mockUseAppStore.mockReturnValue({
      setPageTitle: mockSetPageTitle,
    })

    mockUseAuthStore.mockReturnValue({
      token: "mock-super-admin-token",
      isSuperAdmin: true,
      isCheckingAuth: false,
    })

    global.fetch = vi.fn().mockImplementation((url: string) => {
      if (url.includes("/api/v1/admin/nfs/volumes")) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => mockVolumeData,
        })
      }
      if (url.includes("/api/v1/admin/nfs/folders")) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => mockFoldersData,
        })
      }
      if (url.includes("/api/v1/admin/organizations")) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => mockOrgsData,
        })
      }
      if (url.includes("/api/v1/admin/nfs/hanging-projects")) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => mockHangingData,
        })
      }
      return Promise.resolve({
        ok: true,
        status: 200,
        json: async () => ({ status: "SUCCESS" }),
      })
    }) as any
  })

  afterEach(() => {
    cleanup()
  })

  it("renders loading indicator when verifying access", () => {
    mockUseAuthStore.mockReturnValue({
      token: null,
      isSuperAdmin: false,
      isCheckingAuth: true,
    })

    render(<NFSAdminPage />)
    expect(screen.getByText("Verifying super-admin access...")).not.toBeNull()
  })

  it("renders Access Restricted when user is not super admin", () => {
    mockUseAuthStore.mockReturnValue({
      token: "mock-user-token",
      isSuperAdmin: false,
      isCheckingAuth: false,
    })

    render(<NFSAdminPage />)
    expect(screen.getByText("Access Restricted")).not.toBeNull()
    expect(global.fetch).not.toHaveBeenCalled()
  })

  it("renders NFS admin page and loads volume and folder data for Super Admin", async () => {
    render(<NFSAdminPage />)

    expect(mockSetPageTitle).toHaveBeenCalledWith("Admin: NFS Volume & Folder Metadata")

    await waitFor(() => {
      expect(screen.getByTestId("nfs-admin-page")).not.toBeNull()
    })

    expect(screen.getByText("NFS Volume Health & Capacity")).not.toBeNull()
    expect(screen.getByText("Hanging Projects Discovery & Cleanup")).not.toBeNull()
    expect(screen.getByText("org1-user1-proj1")).not.toBeNull()
  })

  it("triggers On-Demand Storage Audit job execution (POST /api/v1/admin/jobs/nfs-audit/run) and displays active job badge", async () => {
    global.fetch = vi.fn().mockImplementation((url: string, opts?: any) => {
      if (url.includes("/api/v1/admin/jobs/nfs-audit/run")) {
        return Promise.resolve({
          ok: true,
          status: 202,
          json: async () => ({
            status: "PENDING",
            executionName: "projects/test-p/locations/us-c1/jobs/apf-stg-maintenance/executions/exec-audit-99",
            jobName: "nfs-audit",
          }),
        })
      }
      if (url.includes("/api/v1/admin/nfs/volumes")) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => mockVolumeData,
        })
      }
      if (url.includes("/api/v1/admin/nfs/folders")) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => mockFoldersData,
        })
      }
      if (url.includes("/api/v1/admin/organizations")) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => mockOrgsData,
        })
      }
      if (url.includes("/api/v1/admin/nfs/hanging-projects")) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => mockHangingData,
        })
      }
      return Promise.resolve({
        ok: true,
        status: 200,
        json: async () => ({ status: "SUCCESS" }),
      })
    }) as any

    render(<NFSAdminPage />)

    await waitFor(() => {
      const btn = screen.getByText("Run Storage Audit Now").closest("button") as HTMLButtonElement
      expect(btn).not.toBeNull()
      expect(btn.disabled).toBe(false)
    })

    const auditBtn = screen.getByText("Run Storage Audit Now")
    fireEvent.click(auditBtn)

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        "/api/v1/admin/jobs/nfs-audit/run",
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify({ dryRun: false }),
        })
      )
    })

    await waitFor(() => {
      expect(toast.success).toHaveBeenCalledWith(
        expect.stringContaining("Job execution initiated: nfs-audit")
      )
    })

    expect(screen.getByTestId("active-job-indicator")).not.toBeNull()
    expect(screen.getByText(/Job Active: nfs-audit \(PENDING\)/i)).not.toBeNull()
  })

  it("triggers On-Demand Storage Audit job with dryRun flag enabled", async () => {
    global.fetch = vi.fn().mockImplementation((url: string) => {
      if (url.includes("/api/v1/admin/jobs/nfs-audit/run")) {
        return Promise.resolve({
          ok: true,
          status: 202,
          json: async () => ({
            status: "PENDING",
            executionName: "projects/test-p/locations/us-c1/jobs/apf-stg-maintenance/executions/exec-audit-dry",
            jobName: "nfs-audit",
          }),
        })
      }
      if (url.includes("/api/v1/admin/nfs/volumes")) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => mockVolumeData,
        })
      }
      if (url.includes("/api/v1/admin/nfs/folders")) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => mockFoldersData,
        })
      }
      if (url.includes("/api/v1/admin/organizations")) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => mockOrgsData,
        })
      }
      if (url.includes("/api/v1/admin/nfs/hanging-projects")) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => mockHangingData,
        })
      }
      return Promise.resolve({
        ok: true,
        status: 200,
        json: async () => ({ status: "SUCCESS" }),
      })
    }) as any

    render(<NFSAdminPage />)

    await waitFor(() => {
      const btn = screen.getByText("Run Storage Audit Now").closest("button") as HTMLButtonElement
      expect(btn).not.toBeNull()
      expect(btn.disabled).toBe(false)
    })

    // Toggle the Dry Run checkbox in NFSVolumeStatsCard
    const dryRunLabels = screen.getAllByText("Dry Run")
    const auditDryRunCheckbox = dryRunLabels[0].parentElement?.querySelector("input") as HTMLInputElement
    fireEvent.click(auditDryRunCheckbox)

    const auditBtn = screen.getByText("Run Storage Audit Now")
    fireEvent.click(auditBtn)

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        "/api/v1/admin/jobs/nfs-audit/run",
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify({ dryRun: true }),
        })
      )
    })
  })

  it("triggers On-Demand Cleanup job execution (POST /api/v1/admin/jobs/hanging-cleanup/run) with dryRun flag", async () => {
    global.fetch = vi.fn().mockImplementation((url: string, opts?: any) => {
      if (url.includes("/api/v1/admin/jobs/hanging-cleanup/run")) {
        return Promise.resolve({
          ok: true,
          status: 202,
          json: async () => ({
            status: "PENDING",
            executionName: "projects/test-p/locations/us-c1/jobs/apf-stg-maintenance/executions/exec-cleanup-77",
            jobName: "hanging-cleanup",
          }),
        })
      }
      if (url.includes("/api/v1/admin/nfs/volumes")) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => mockVolumeData,
        })
      }
      if (url.includes("/api/v1/admin/nfs/folders")) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => mockFoldersData,
        })
      }
      if (url.includes("/api/v1/admin/organizations")) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => mockOrgsData,
        })
      }
      if (url.includes("/api/v1/admin/nfs/hanging-projects")) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => mockHangingData,
        })
      }
      return Promise.resolve({
        ok: true,
        status: 200,
        json: async () => ({ status: "SUCCESS" }),
      })
    }) as any

    render(<NFSAdminPage />)

    await waitFor(() => {
      const btn = screen.getByText("Run Cleanup").closest("button") as HTMLButtonElement
      expect(btn).not.toBeNull()
      expect(btn.disabled).toBe(false)
    })

    // Find and check the Cleanup Dry Run checkbox (in HangingProjectsSection)
    const dryRunLabels = screen.getAllByText("Dry Run")
    const cleanupDryRunCheckbox = dryRunLabels[dryRunLabels.length - 1].parentElement?.querySelector("input") as HTMLInputElement
    fireEvent.click(cleanupDryRunCheckbox)

    const cleanupBtn = screen.getByText("Run Cleanup")
    fireEvent.click(cleanupBtn)

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        "/api/v1/admin/jobs/hanging-cleanup/run",
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify({ dryRun: true }),
        })
      )
    })

    await waitFor(() => {
      expect(toast.success).toHaveBeenCalledWith(
        expect.stringContaining("Job execution initiated: hanging-cleanup")
      )
    })
  })

  it("handles job execution API error and shows alert and toast error", async () => {
    global.fetch = vi.fn().mockImplementation((url: string) => {
      if (url.includes("/api/v1/admin/jobs/nfs-audit/run")) {
        return Promise.resolve({
          ok: false,
          status: 500,
          json: async () => ({
            status: "ERROR",
            message: "Cloud Run API quota exceeded",
          }),
        })
      }
      if (url.includes("/api/v1/admin/nfs/volumes")) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => mockVolumeData,
        })
      }
      if (url.includes("/api/v1/admin/nfs/folders")) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => mockFoldersData,
        })
      }
      if (url.includes("/api/v1/admin/organizations")) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => mockOrgsData,
        })
      }
      if (url.includes("/api/v1/admin/nfs/hanging-projects")) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => mockHangingData,
        })
      }
      return Promise.resolve({
        ok: true,
        status: 200,
        json: async () => ({ status: "SUCCESS" }),
      })
    }) as any

    render(<NFSAdminPage />)

    await waitFor(() => {
      const btn = screen.getByText("Run Storage Audit Now").closest("button") as HTMLButtonElement
      expect(btn).not.toBeNull()
      expect(btn.disabled).toBe(false)
    })

    const auditBtn = screen.getByText("Run Storage Audit Now")
    fireEvent.click(auditBtn)

    await waitFor(() => {
      expect(screen.getByText("Cloud Run API quota exceeded")).not.toBeNull()
    })

    expect(toast.error).toHaveBeenCalledWith("Cloud Run API quota exceeded")
  })
})
