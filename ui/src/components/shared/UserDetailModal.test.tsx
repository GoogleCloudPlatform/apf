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
import { render, screen, fireEvent, cleanup } from "@testing-library/react"
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { UserDetailModal } from "./UserDetailModal"
import { useMutation, useQuery } from "@tanstack/react-query"
import { toast } from "sonner"

// Mock sonner toast
vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

// Mock react-query
const mockMutate = vi.fn()
vi.mock("@tanstack/react-query", () => ({
  useMutation: vi.fn(),
  useQuery: vi.fn(),
}))

describe("UserDetailModal Component", () => {
  const mockOnClose = vi.fn()
  const mockOnSuccess = vi.fn()
  const mockUseMutation = useMutation as any
  const mockUseQuery = useQuery as any

  beforeEach(() => {
    vi.clearAllMocks()
    // Default mock implementation (success)
    mockUseMutation.mockReturnValue({
      mutate: mockMutate,
      isPending: false,
    })
    mockUseQuery.mockReturnValue({
      data: {
        roles: [
          { id: "viewer", name: "Viewer", isCustom: false },
          { id: "editor", name: "Editor", isCustom: false },
          { id: "admin", name: "Admin", isCustom: false },
        ]
      },
      isLoading: false,
    })
  })

  afterEach(() => {
    cleanup()
  })

  it("does not render when isOpen is false", () => {
    const { container } = render(
      <UserDetailModal isOpen={false} onClose={mockOnClose} orgId="org-123" />
    )
    expect(container.firstChild).toBeNull()
  })

  it("renders correctly when isOpen is true in Add mode", () => {
    render(
      <UserDetailModal isOpen={true} onClose={mockOnClose} orgId="org-123" />
    )
    
    expect(screen.queryByText("Add New Member")).not.toBeNull()
    expect(screen.queryByLabelText("Email Address")).not.toBeNull()
    expect(screen.queryByLabelText("Display Name (Optional)")).not.toBeNull()
    expect(screen.queryByText("Organization Roles")).not.toBeNull()
    expect(screen.queryByRole("button", { name: "Add Member" })).not.toBeNull()
    expect(screen.queryByRole("button", { name: "Cancel" })).not.toBeNull()
  })

  it("calls onClose when Cancel button is clicked", () => {
    render(
      <UserDetailModal isOpen={true} onClose={mockOnClose} orgId="org-123" />
    )
    
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }))
    expect(mockOnClose).toHaveBeenCalledTimes(1)
  })

  it("calls onClose when Close icon button is clicked", () => {
    render(
      <UserDetailModal isOpen={true} onClose={mockOnClose} orgId="org-123" />
    )
    
    fireEvent.click(screen.getByRole("button", { name: "Close Modal" }))
    expect(mockOnClose).toHaveBeenCalledTimes(1)
  })

  it("calls mutate with correct values on form submission in Add mode", () => {
    render(
      <UserDetailModal isOpen={true} onClose={mockOnClose} orgId="org-123" />
    )
    
    const emailInput = screen.getByLabelText("Email Address")
    const nameInput = screen.getByLabelText("Display Name (Optional)")
    
    fireEvent.change(emailInput, { target: { value: "test@example.com" } })
    fireEvent.change(nameInput, { target: { value: "Anakin Skywalker" } })
    
    // Select "Editor" role chip
    const editorChip = screen.getByRole("button", { name: "Editor" })
    fireEvent.click(editorChip)
    
    fireEvent.click(screen.getByRole("button", { name: "Add Member" }))
    
    expect(mockMutate).toHaveBeenCalledWith({
      email: "test@example.com",
      password: expect.any(String),
      displayName: "Anakin Skywalker",
      roles: ["viewer", "editor"],
    })
  })

  it("displays loading spinner when addition is pending", () => {
    mockUseMutation.mockReturnValue({
      mutate: mockMutate,
      isPending: true,
    })
    
    render(
      <UserDetailModal isOpen={true} onClose={mockOnClose} orgId="org-123" />
    )
    
    const submitButton = screen.getByRole("button", { name: "Add Member" })
    expect(submitButton.hasAttribute("disabled")).toBe(true)
  })

  it("handles mutation success correctly", async () => {
    mockUseMutation.mockImplementation((options: any) => {
      const onSuccessCallback = options?.onSuccess
      return {
        mutate: (variables: any) => {
          if (onSuccessCallback) onSuccessCallback(variables)
        },
        isPending: false,
      }
    })

    render(
      <UserDetailModal isOpen={true} onClose={mockOnClose} orgId="org-123" onSuccess={mockOnSuccess} />
    )
    
    fireEvent.change(screen.getByLabelText("Email Address"), { target: { value: "success@example.com" } })
    fireEvent.click(screen.getByRole("button", { name: "Add Member" }))
    
    expect(toast.success).toHaveBeenCalledWith("Member success@example.com successfully added to organization!")
    expect(mockOnSuccess).toHaveBeenCalledTimes(1)
    expect(mockOnClose).toHaveBeenCalledTimes(1)
    expect((screen.getByLabelText("Email Address") as HTMLInputElement).value).toBe("")
  })

  it("handles mutation error correctly", async () => {
    mockUseMutation.mockImplementation((options: any) => {
      const onErrorCallback = options?.onError
      return {
        mutate: () => {
          if (onErrorCallback) onErrorCallback(new Error("Failed to add member"))
        },
        isPending: false,
      }
    })

    render(
      <UserDetailModal isOpen={true} onClose={mockOnClose} orgId="org-123" />
    )
    
    fireEvent.change(screen.getByLabelText("Email Address"), { target: { value: "error@example.com" } })
    fireEvent.click(screen.getByRole("button", { name: "Add Member" }))
    
    expect(toast.error).toHaveBeenCalledWith("Failed to add member")
    expect(mockOnClose).not.toHaveBeenCalled()
  })

  it("renders correctly in Edit mode with existing member and Revoke button", () => {
    const mockOnDelete = vi.fn()
    const testMember = {
      id: "user-123",
      email: "existing@example.com",
      displayName: "Existing User",
      role: "editor",
      roles: ["editor"],
      status: "active" as const,
      lastLogin: "2026-07-07T12:00:00Z"
    }

    render(
      <UserDetailModal
        isOpen={true}
        onClose={mockOnClose}
        orgId="org-123"
        member={testMember}
        onDelete={mockOnDelete}
      />
    )

    expect(screen.queryByText("Edit Member")).not.toBeNull()
    expect((screen.getByLabelText("Email Address") as HTMLInputElement).value).toBe("existing@example.com")
    expect(screen.queryByLabelText("Password")).toBeNull() // Password omitted in edit mode
    expect(screen.queryByRole("button", { name: "Revoke" })).not.toBeNull()
    expect(screen.queryByRole("button", { name: "Save Changes" })).not.toBeNull()
  })

  it("calls onDelete when Revoke button is clicked in Edit mode", () => {
    const mockOnDelete = vi.fn()
    const testMember = {
      id: "user-123",
      email: "existing@example.com",
      role: "editor",
      roles: ["editor"],
      status: "active" as const
    }

    render(
      <UserDetailModal
        isOpen={true}
        onClose={mockOnClose}
        orgId="org-123"
        member={testMember}
        onDelete={mockOnDelete}
      />
    )

    fireEvent.click(screen.getByRole("button", { name: "Revoke" }))
    expect(mockOnClose).toHaveBeenCalledTimes(1)
    expect(mockOnDelete).toHaveBeenCalledWith(testMember)
  })
})
