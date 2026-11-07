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
import HangingProjectsSection, { HangingProjectItem } from "./HangingProjectsSection"

describe("HangingProjectsSection", () => {
  const mockProjects: HangingProjectItem[] = [
    {
      indexKey: "org1-user1-proj1",
      orgId: "org1",
      userId: "user1",
      projectId: "proj1",
      path: "/workspaces/organizations/org1/user1/proj1",
      sizeBytes: 1024 * 1024 * 50,
      fileCount: 25,
    },
    {
      indexKey: "org1-user2-proj2",
      orgId: "org1",
      userId: "user2",
      projectId: "proj2",
      path: "/workspaces/organizations/org1/user2/proj2",
      sizeBytes: 1024 * 1024 * 10,
      fileCount: 5,
    },
  ]

  beforeEach(() => {
    vi.restoreAllMocks()
  })

  afterEach(() => {
    cleanup()
  })

  it("fetches and renders hanging projects on mount", async () => {
    global.fetch = vi.fn().mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        status: "SUCCESS",
        hangingProjects: mockProjects,
        total: mockProjects.length,
      }),
    } as Response)

    render(<HangingProjectsSection token="mock-token" />)

    expect(await screen.findByText("org1-user1-proj1")).not.toBeNull()
    expect(screen.getByText("org1-user2-proj2")).not.toBeNull()
    expect(screen.getByText("2 Orphaned")).not.toBeNull()
  })

  it("successfully deletes all selected hanging projects and shows success banner", async () => {
    global.fetch = vi
      .fn()
      // Initial fetch
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          status: "SUCCESS",
          hangingProjects: mockProjects,
          total: 2,
        }),
      } as Response)
      // Delete call
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          status: "SUCCESS",
          message: "Successfully deleted 2 hanging project folder(s)",
          deletedCount: 2,
          freedBytes: 1024 * 1024 * 60,
          errors: [],
        }),
      } as Response)
      // Re-fetch after deletion
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          status: "SUCCESS",
          hangingProjects: [],
          total: 0,
        }),
      } as Response)

    render(<HangingProjectsSection token="mock-token" />)

    // Wait for initial load
    expect(await screen.findByText("org1-user1-proj1")).not.toBeNull()

    // Click "Purge All"
    fireEvent.click(screen.getByText(/Purge All/i))

    // Confirm in modal
    const confirmButton = screen.getByText(/Confirm & Delete/i)
    fireEvent.click(confirmButton)

    // Expect success message
    expect(
      await screen.findByText(/Successfully deleted 2 hanging project folder\(s\)/i)
    ).not.toBeNull()
  })

  it("reflects a warning in the UI when the delete operation fails on some folders", async () => {
    global.fetch = vi
      .fn()
      // Initial fetch
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          status: "SUCCESS",
          hangingProjects: mockProjects,
          total: 2,
        }),
      } as Response)
      // Delete call with partial failure and errors
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          status: "WARNING",
          message: "Deleted 1 hanging project folder(s), but failed to delete 1 folder(s)",
          deletedCount: 1,
          freedBytes: 1024 * 1024 * 50,
          errors: [
            "Failed to remove directory /workspaces/organizations/org1/user2/proj2: permission denied",
          ],
        }),
      } as Response)
      // Re-fetch after deletion
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          status: "SUCCESS",
          hangingProjects: [mockProjects[1]],
          total: 1,
        }),
      } as Response)

    render(<HangingProjectsSection token="mock-token" />)

    expect(await screen.findByText("org1-user1-proj1")).not.toBeNull()

    // Open confirmation modal for all projects
    fireEvent.click(screen.getByText(/Purge All/i))

    // Confirm deletion
    const confirmButton = screen.getByText(/Confirm & Delete/i)
    fireEvent.click(confirmButton)

    // Should display warning that system was unable to delete some requested folders
    await waitFor(() => {
      const warningAlert = screen.getByRole("alert", { name: /warning message/i })
      expect(warningAlert).not.toBeNull()
      expect(warningAlert.textContent).toContain(
        "Warning: The system was unable to delete some of the requested folders"
      )
      expect(warningAlert.textContent).toContain("permission denied")
    })

    // Should also show success for the deleted folder(s)
    const successStatus = screen.getByRole("status", { name: /success message/i })
    expect(successStatus).not.toBeNull()
    expect(successStatus.textContent).toContain("Successfully deleted 1 hanging project folder(s)")
  })

  it("reflects a warning in the UI when all requested folder deletions fail", async () => {
    global.fetch = vi
      .fn()
      // Initial fetch
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          status: "SUCCESS",
          hangingProjects: [mockProjects[0]],
          total: 1,
        }),
      } as Response)
      // Delete call with complete failure
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          status: "ERROR",
          message: "Failed to delete 1 hanging project folder(s)",
          deletedCount: 0,
          freedBytes: 0,
          errors: [
            "Refusing to delete /tmp/outside: path must be at project level (org/user/project)",
          ],
        }),
      } as Response)
      // Re-fetch after deletion
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          status: "SUCCESS",
          hangingProjects: [mockProjects[0]],
          total: 1,
        }),
      } as Response)

    render(<HangingProjectsSection token="mock-token" />)

    expect(await screen.findByText("org1-user1-proj1")).not.toBeNull()

    // Open confirmation modal
    fireEvent.click(screen.getByText(/Purge All/i))

    // Confirm deletion
    const confirmButton = screen.getByText(/Confirm & Delete/i)
    fireEvent.click(confirmButton)

    // Should display warning alert
    await waitFor(() => {
      const warningAlert = screen.getByRole("alert", { name: /warning message/i })
      expect(warningAlert).not.toBeNull()
      expect(warningAlert.textContent).toContain(
        "Warning: The system was unable to delete the requested folders"
      )
      expect(warningAlert.textContent).toContain("path must be at project level")
    })
  })
})
