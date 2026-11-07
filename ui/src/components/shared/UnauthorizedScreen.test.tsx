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
import { UnauthorizedScreen } from "./UnauthorizedScreen"
import { useAuthStore } from "@/stores/auth"

// Mock next/navigation
const mockPush = vi.fn()
const mockBack = vi.fn()

vi.mock("next/navigation", () => ({
  useRouter: () => ({
    push: mockPush,
    back: mockBack,
  }),
}))

// Mock auth store
vi.mock("@/stores/auth", () => ({
  useAuthStore: vi.fn(),
}))

describe("UnauthorizedScreen Component", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    ;(useAuthStore as any).mockReturnValue({
      email: "user@example.com",
      selectedOrgId: "org-showcase-123",
      isSuperAdmin: false,
      permissions: ["user:read"],
    })
  })

  afterEach(() => {
    cleanup()
  })

  it("renders default title and message correctly", () => {
    render(<UnauthorizedScreen />)

    expect(screen.queryByText("Access Restricted")).not.toBeNull()
    expect(screen.queryByText(/You do not have the required permissions/)).not.toBeNull()
    expect(screen.queryByText("user@example.com")).not.toBeNull()
    expect(screen.queryByText("org-showcase-123")).not.toBeNull()
    expect(screen.queryByText("Standard Member")).not.toBeNull()
  })

  it("renders custom title, message, and required permission", () => {
    render(
      <UnauthorizedScreen
        title="Admin Area Only"
        message="Custom restriction notice"
        requiredPermission="org:update"
      />
    )

    expect(screen.queryByText("Admin Area Only")).not.toBeNull()
    expect(screen.queryByText("Custom restriction notice")).not.toBeNull()
    expect(screen.queryByText("org:update")).not.toBeNull()
  })

  it("renders Super Admin status when user is super admin", () => {
    ;(useAuthStore as any).mockReturnValue({
      email: "admin@example.com",
      selectedOrgId: "org-1",
      isSuperAdmin: true,
      permissions: [],
    })

    render(<UnauthorizedScreen />)

    expect(screen.queryByText("Super Admin")).not.toBeNull()
  })

  it("navigates back when Go Back is clicked", () => {
    Object.defineProperty(window, "history", {
      value: { length: 2, back: vi.fn() },
      writable: true,
    })

    render(<UnauthorizedScreen />)

    const goBackBtn = screen.getByRole("button", { name: /Go Back/i })
    fireEvent.click(goBackBtn)

    expect(mockBack).toHaveBeenCalledTimes(1)
  })

  it("navigates to projects when Projects button is clicked", () => {
    render(<UnauthorizedScreen />)

    const projectsBtn = screen.getByRole("button", { name: /Projects/i })
    fireEvent.click(projectsBtn)

    expect(mockPush).toHaveBeenCalledWith("/projects")
  })

  it("navigates to select-org when Switch Org button is clicked", () => {
    render(<UnauthorizedScreen />)

    const switchOrgBtn = screen.getByRole("button", { name: /Switch Org/i })
    fireEvent.click(switchOrgBtn)

    expect(mockPush).toHaveBeenCalledWith("/select-org")
  })
})
