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
import { render, screen, cleanup, waitFor } from "@testing-library/react"
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import SelectOrganizationPage from "./page"
import { useAuthStore } from "@/stores/auth"

const mockPush = vi.fn()

vi.mock("next/navigation", () => ({
  useRouter: () => ({
    push: mockPush,
  }),
}))

vi.mock("@/stores/auth", () => ({
  useAuthStore: vi.fn(),
}))

describe("SelectOrganizationPage", () => {
  const mockLoadOrganizations = vi.fn()
  const mockSelectOrganization = vi.fn()
  const mockLogout = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    cleanup()
  })

  it("automatically selects and launches when user belongs to only 1 organization", async () => {
    const singleOrg = [{ id: "org-solo", name: "Solo Workspace", tenant_id: "tenant-solo" }]
    mockLoadOrganizations.mockResolvedValue(singleOrg)
    mockSelectOrganization.mockResolvedValue(true)

    ;(useAuthStore as any).mockReturnValue({
      organizations: singleOrg,
      loadOrganizations: mockLoadOrganizations,
      selectOrganization: mockSelectOrganization,
      logout: mockLogout,
      isCheckingAuth: false,
    })

    render(<SelectOrganizationPage />)

    // Should load organizations
    expect(mockLoadOrganizations).toHaveBeenCalledTimes(1)

    // Should automatically call selectOrganization with the single org's ID
    await waitFor(() => {
      expect(mockSelectOrganization).toHaveBeenCalledWith("org-solo")
    })

    // Should redirect to /projects
    await waitFor(() => {
      expect(mockPush).toHaveBeenCalledWith("/projects")
    })
  })

  it("does not auto-select and displays dropdown when user belongs to multiple organizations", async () => {
    const multiOrgs = [
      { id: "org-1", name: "Alpha Corp" },
      { id: "org-2", name: "Beta Corp" },
    ]
    mockLoadOrganizations.mockResolvedValue(multiOrgs)

    ;(useAuthStore as any).mockReturnValue({
      organizations: multiOrgs,
      loadOrganizations: mockLoadOrganizations,
      selectOrganization: mockSelectOrganization,
      logout: mockLogout,
      isCheckingAuth: false,
    })

    render(<SelectOrganizationPage />)

    await waitFor(() => {
      expect(mockLoadOrganizations).toHaveBeenCalledTimes(1)
    })

    // Should NOT automatically select or redirect
    expect(mockSelectOrganization).not.toHaveBeenCalled()
    expect(mockPush).not.toHaveBeenCalled()

    // Should display the selection interface
    expect(screen.getByText("Select an organization...")).toBeDefined()
    expect(screen.getByText("ENTER FOUNDRY")).toBeDefined()
  })

  it("displays no authorized organizations message when user belongs to 0 organizations", async () => {
    mockLoadOrganizations.mockResolvedValue([])

    ;(useAuthStore as any).mockReturnValue({
      organizations: [],
      loadOrganizations: mockLoadOrganizations,
      selectOrganization: mockSelectOrganization,
      logout: mockLogout,
      isCheckingAuth: false,
    })

    render(<SelectOrganizationPage />)

    await waitFor(() => {
      expect(screen.getByText("No authorized organizations found for your account.")).toBeDefined()
    })

    expect(mockSelectOrganization).not.toHaveBeenCalled()
    expect(mockPush).not.toHaveBeenCalled()
  })

  it("displays error banner when auto-selection fails", async () => {
    const singleOrg = [{ id: "org-solo", name: "Solo Workspace" }]
    mockLoadOrganizations.mockResolvedValue(singleOrg)
    mockSelectOrganization.mockResolvedValue(false)

    ;(useAuthStore as any).mockReturnValue({
      organizations: singleOrg,
      loadOrganizations: mockLoadOrganizations,
      selectOrganization: mockSelectOrganization,
      logout: mockLogout,
      isCheckingAuth: false,
    })

    render(<SelectOrganizationPage />)

    await waitFor(() => {
      expect(mockSelectOrganization).toHaveBeenCalledWith("org-solo")
    })

    await waitFor(() => {
      expect(screen.getByText(/Failed to auto-select organization "Solo Workspace"/)).toBeDefined()
    })

    expect(mockPush).not.toHaveBeenCalledWith("/projects")
  })
})
