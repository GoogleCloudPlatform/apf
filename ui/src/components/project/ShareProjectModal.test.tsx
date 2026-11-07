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
import { render, screen, fireEvent, cleanup, waitFor } from "@testing-library/react"
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import * as matchers from "@testing-library/jest-dom/matchers"
expect.extend(matchers)
import { ShareProjectModal } from "./ShareProjectModal"
import { toast } from "sonner"
import { api } from "@/lib/api"

// Mock sonner toast
vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
  },
}))

// Mock navigator.clipboard
const mockWriteText = vi.fn()
Object.defineProperty(navigator, "clipboard", {
  value: {
    writeText: mockWriteText,
  },
  writable: true,
})

// Mock API client
vi.mock("@/lib/api", () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}))

// Mock auth store
vi.mock("@/stores/auth", () => ({
  useAuthStore: () => ({
    email: "owner@example.com",
  }),
}))

describe("ShareProjectModal", () => {
  const mockOnClose = vi.fn()
  const projectId = "test-project-id"

  const mockMembers = [
    {
      id: "1",
      name: "Project Owner",
      email: "owner@example.com",
      role: "Owner",
      isPending: false,
    },
    {
      id: "2",
      name: "Alex Morgan",
      email: "viewer@example.com",
      role: "Viewer",
      isPending: false,
    }
  ]

  beforeEach(() => {
    vi.clearAllMocks()
    // Default mock implementation for GET members with logging
    api.get.mockImplementation(async (url) => {
      console.log(`[Mock API GET] Calling ${url}`)
      return {
        ok: true,
        json: async () => {
          console.log(`[Mock API GET] Resolved JSON for ${url}`)
          return mockMembers
        },
      }
    })
  })

  afterEach(() => {
    cleanup()
  })

  it("renders correctly when open in standard state", async () => {
    render(
      <ShareProjectModal 
        isOpen={true} 
        onClose={mockOnClose} 
        projectName="Requirements Specification" 
        projectId={projectId}
      />
    )

    // Check title
    expect(screen.getByText('Share "Requirements Specification"')).toBeInTheDocument()

    // Check helper text / label
    expect(screen.getByText("Add people by email address")).toBeInTheDocument()

    // Check default users (wait for them to load)
    await waitFor(() => {
      expect(screen.getByText("Project Owner")).toBeInTheDocument()
    })
    expect(screen.getByText("owner@example.com")).toBeInTheDocument()
    expect(screen.getByText("Alex Morgan")).toBeInTheDocument()
    expect(screen.getByText("viewer@example.com")).toBeInTheDocument()
  })

  it("converts a valid email to a chip when hitting space bar", async () => {
    render(
      <ShareProjectModal 
        isOpen={true} 
        onClose={mockOnClose} 
        projectName="Requirements Specification" 
        projectId={projectId}
      />
    )

    // Wait for initial load to complete
    await waitFor(() => {
      expect(screen.getByText("Project Owner")).toBeInTheDocument()
    })

    const input = screen.getByPlaceholderText("Enter email addresses...")
    
    // Type email
    fireEvent.change(input, { target: { value: "anakin.skywalker@example.com" } })
    
    // Fire space keydown
    fireEvent.keyDown(input, { key: " " })

    // Verify chip is rendered
    expect(screen.getByText("Anakin Skywalker")).toBeInTheDocument()
    expect(input.value).toBe("") // Input should be cleared

    // Verify modal transitioned into invitation state (Notify people appears)
    expect(screen.getByText("Notify people")).toBeInTheDocument()
    expect(screen.getByPlaceholderText("Message")).toBeInTheDocument()
  })

  it("shows error toast when entering an invalid email address", async () => {
    render(
      <ShareProjectModal 
        isOpen={true} 
        onClose={mockOnClose} 
        projectName="Requirements Specification" 
        projectId={projectId}
      />
    )

    await waitFor(() => {
      expect(screen.getByText("Project Owner")).toBeInTheDocument()
    })

    const input = screen.getByPlaceholderText("Enter email addresses...")
    fireEvent.change(input, { target: { value: "invalid-email" } })
    fireEvent.keyDown(input, { key: "Enter" })

    // Verify toast.error was called
    expect(toast.error).toHaveBeenCalledWith("Please enter a valid email address.")

    // Verify chip NOT rendered
    expect(screen.queryByText("invalid-email")).not.toBeInTheDocument()
  })

  it("converts a valid email to a chip on blur", async () => {
    render(
      <ShareProjectModal 
        isOpen={true} 
        onClose={mockOnClose} 
        projectName="Requirements Specification" 
        projectId={projectId}
      />
    )

    await waitFor(() => {
      expect(screen.getByText("Project Owner")).toBeInTheDocument()
    })

    const input = screen.getByPlaceholderText("Enter email addresses...")
    
    // Type email
    fireEvent.change(input, { target: { value: "padme.amidala@example.com" } })
    
    // Simulate blur
    fireEvent.blur(input)

    // Verify chip is rendered
    expect(screen.getByText("Padme Amidala")).toBeInTheDocument()
    expect(input.value).toBe("") // Input should be cleared
  })

  it("toggles the message textbox when Notify People is clicked", async () => {
    render(
      <ShareProjectModal 
        isOpen={true} 
        onClose={mockOnClose} 
        projectName="Requirements Specification" 
        projectId={projectId}
      />
    )

    await waitFor(() => {
      expect(screen.getByText("Project Owner")).toBeInTheDocument()
    })

    const input = screen.getByPlaceholderText("Enter email addresses...")
    fireEvent.change(input, { target: { value: "luke@skywalker.com" } })
    fireEvent.keyDown(input, { key: " " })

    // Check message box is visible by default
    const messageBox = screen.getByPlaceholderText("Message")
    expect(messageBox).toBeInTheDocument()

    // Uncheck Notify People
    const checkbox = screen.getByLabelText("Notify people")
    fireEvent.click(checkbox)

    // Message box should be hidden
    expect(screen.queryByPlaceholderText("Message")).not.toBeInTheDocument()
  })

  it("allows setting a pending role for the invitation", async () => {
    render(
      <ShareProjectModal 
        isOpen={true} 
        onClose={mockOnClose} 
        projectName="Requirements Specification" 
        projectId={projectId}
      />
    )

    await waitFor(() => {
      expect(screen.getByText("Project Owner")).toBeInTheDocument()
    })

    const input = screen.getByPlaceholderText("Enter email addresses...")
    fireEvent.change(input, { target: { value: "leia@organa.com" } })
    fireEvent.keyDown(input, { key: " " })

    // Default role selector next to chip input should show Editor
    const roleSelector = screen.getByRole("button", { name: /Editor/i })
    expect(roleSelector).toBeInTheDocument()

    // Click it to open dropdown
    fireEvent.click(roleSelector)

    // Select Viewer
    const viewerOption = screen.getByRole("button", { name: "Viewer" })
    fireEvent.click(viewerOption)

    // Role selector should now update to Viewer
    expect(screen.getByRole("button", { name: /Viewer/i })).toBeInTheDocument()
  })

  it("clears pending chips and returns to main view on Back button or Cancel click", async () => {
    render(
      <ShareProjectModal 
        isOpen={true} 
        onClose={mockOnClose} 
        projectName="Requirements Specification" 
        projectId={projectId}
      />
    )

    await waitFor(() => {
      expect(screen.getByText("Project Owner")).toBeInTheDocument()
    })

    const input = screen.getByPlaceholderText("Enter email addresses...")
    fireEvent.change(input, { target: { value: "chewbacca@kashyyyk.com" } })
    fireEvent.keyDown(input, { key: " " })

    expect(screen.getByText("Chewbacca")).toBeInTheDocument()

    // Click Back button
    const backBtn = screen.getByTitle("Go back")
    fireEvent.click(backBtn)

    // Pending chip should be cleared, returning to main list view
    expect(screen.queryByText("Chewbacca")).not.toBeInTheDocument()
    expect(screen.queryByText("Notify people")).not.toBeInTheDocument()
    expect(screen.getByText("People with access")).toBeInTheDocument()
  })

  it("adds users to the permanent list upon clicking Send", async () => {
    let getCallCount = 0
    api.get.mockImplementation(async (url) => {
      getCallCount++
      console.log(`[Mock API GET - Send Test] Call #${getCallCount} to ${url}`)
      return {
        ok: true,
        json: async () => {
          if (getCallCount === 1) {
            console.log(`[Mock API GET - Send Test] Call #${getCallCount} resolved to initial members`)
            return mockMembers
          }
          console.log(`[Mock API GET - Send Test] Call #${getCallCount} resolved to updated members (with Obiwan)`)
          return [
            ...mockMembers,
            { id: "3", name: "Obiwan", email: "obiwan@kenobi.com", role: "Editor", isPending: true }
          ]
        },
      }
    })

    api.post.mockImplementation(async (url, data) => {
      console.log(`[Mock API POST] Calling ${url} with`, data)
      return {
        ok: true,
        json: async () => {
          console.log(`[Mock API POST] Resolved JSON for ${url}`)
          return { status: "SUCCESS", addedCount: 1 }
        },
      }
    })

    render(
      <ShareProjectModal 
        isOpen={true} 
        onClose={mockOnClose} 
        projectName="Requirements Specification" 
        projectId={projectId}
      />
    )

    await waitFor(() => {
      expect(screen.getByText("Project Owner")).toBeInTheDocument()
    })

    const input = screen.getByPlaceholderText("Enter email addresses...")
    fireEvent.change(input, { target: { value: "obiwan@kenobi.com" } })
    fireEvent.keyDown(input, { key: " " })

    // Click Send
    const sendBtn = screen.getByRole("button", { name: "Send" })
    console.log("[Test] Clicking Send button")
    fireEvent.click(sendBtn)

    // Verify API called
    expect(api.post).toHaveBeenCalledWith(`/api/v1/projects/${projectId}/members`, {
      emails: ["obiwan@kenobi.com"],
      role: "Editor",
    })

    // Verify user added to the list of people with access
    console.log("[Test] Waiting for obiwan@kenobi.com to appear in UI...")
    await waitFor(() => {
      expect(screen.getByText("obiwan@kenobi.com")).toBeInTheDocument()
    })
    console.log("[Test] obiwan@kenobi.com appeared in UI!")
    expect(screen.getByText("Obiwan")).toBeInTheDocument()

    // Verify invitation state cleared
    expect(screen.queryByText("Notify people")).not.toBeInTheDocument()
  })

  it("disables role selectors and shows tooltip when isTiedToRepo is true", async () => {
    render(
      <ShareProjectModal 
        isOpen={true} 
        onClose={mockOnClose} 
        projectName="Requirements Specification" 
        projectId={projectId}
        isTiedToRepo={true}
      />
    )

    await waitFor(() => {
      expect(screen.getByText("Project Owner")).toBeInTheDocument()
    })

    // 1. Check existing member's role button (Alex Morgan is Viewer by default)
    const ericRoleBtn = screen.getByRole("button", { name: /Viewer/i })
    expect(ericRoleBtn).toBeDisabled()
    expect(ericRoleBtn).toHaveAttribute(
      "title", 
      "User role depends on their role in associated code repository"
    )

    // 2. Go to inviting state by adding a chip
    const input = screen.getByPlaceholderText("Enter email addresses...")
    fireEvent.change(input, { target: { value: "luke@skywalker.com" } })
    fireEvent.keyDown(input, { key: " " })

    // 3. Check pending role selector button (now the only "Editor" button visible)
    const pendingRoleBtn = screen.getByRole("button", { name: /Editor/i })
    expect(pendingRoleBtn).toBeDisabled()
    expect(pendingRoleBtn).toHaveAttribute(
      "title", 
      "User role depends on their role in associated code repository"
    )
  })

  it("calls API to update role when role is changed", async () => {
    api.put.mockImplementation(async (url, data) => {
      console.log(`[Mock API PUT] Calling ${url} with`, data)
      return {
        ok: true,
        json: async () => {
          console.log(`[Mock API PUT] Resolved JSON for ${url}`)
          return { status: "SUCCESS" }
        },
      }
    })

    render(
      <ShareProjectModal 
        isOpen={true} 
        onClose={mockOnClose} 
        projectName="Requirements Specification" 
        projectId={projectId}
      />
    )

    await waitFor(() => {
      expect(screen.getByText("Project Owner")).toBeInTheDocument()
    })

    // Find Alex Morgan's role button
    const ericRoleBtn = screen.getByRole("button", { name: /Viewer/i })
    fireEvent.click(ericRoleBtn)

    // Select Editor from dropdown
    const editorOption = screen.getByRole("button", { name: "Editor" })
    console.log("[Test] Clicking Editor option in dropdown")
    fireEvent.click(editorOption)

    // Verify API called
    expect(api.put).toHaveBeenCalledWith(`/api/v1/projects/${projectId}/members/2`, {
      role: "Editor",
    })

    // Verify role updated in UI
    console.log("[Test] Waiting for Editor button to appear in UI...")
    await waitFor(() => {
      expect(screen.getByRole("button", { name: /Editor/i })).toBeInTheDocument()
    })
    console.log("[Test] Editor button appeared in UI!")
  })

  it("calls API to remove member when remove button is clicked", async () => {
    api.delete.mockResolvedValue({
      ok: true,
      json: async () => ({ status: "SUCCESS" }),
    })

    render(
      <ShareProjectModal 
        isOpen={true} 
        onClose={mockOnClose} 
        projectName="Requirements Specification" 
        projectId={projectId}
      />
    )

    await waitFor(() => {
      expect(screen.getByText("Project Owner")).toBeInTheDocument()
    })

    // Find remove button for Alex Morgan (id: 2)
    // The button has title "Revoke access for Alex Morgan"
    const removeBtn = screen.getByTitle("Revoke access for Alex Morgan")
    fireEvent.click(removeBtn)

    // Verify API called
    expect(api.delete).toHaveBeenCalledWith(`/api/v1/projects/${projectId}/members/2`)

    // Verify member removed from UI
    await waitFor(() => {
      expect(screen.queryByText("Alex Morgan")).not.toBeInTheDocument()
    })
  })
})
