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
import { render, screen, cleanup } from "@testing-library/react"
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import UsersPage from "./page"
import { useAuthStore } from "@/stores/auth"
import { useAppStore } from "@/stores/app"
import { useQuery, useMutation } from "@tanstack/react-query"

// Mock next/navigation
vi.mock("next/navigation", () => ({
  useRouter: () => ({
    push: vi.fn(),
    replace: vi.fn(),
    back: vi.fn(),
  }),
}))

// Mock sonner
vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

// Mock react-query
vi.mock("@tanstack/react-query", () => ({
  useQueryClient: () => ({
    invalidateQueries: vi.fn(),
  }),
  useQuery: vi.fn(),
  useMutation: vi.fn(),
}))

// Mock auth store and app store
vi.mock("@/stores/auth", () => ({
  useAuthStore: vi.fn(),
}))

vi.mock("@/stores/app", () => ({
  useAppStore: vi.fn(),
}))

describe("IAM / UsersPage", () => {
  const mockSetPageTitle = vi.fn()
  const mockUseAuthStore = useAuthStore as unknown as ReturnType<typeof vi.fn>
  const mockUseAppStore = useAppStore as unknown as ReturnType<typeof vi.fn>
  const mockUseQuery = useQuery as unknown as ReturnType<typeof vi.fn>
  const mockUseMutation = useMutation as unknown as ReturnType<typeof vi.fn>

  beforeEach(() => {
    vi.clearAllMocks()
    mockUseAppStore.mockReturnValue({
      setPageTitle: mockSetPageTitle,
    })
    mockUseMutation.mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
    })
    mockUseQuery.mockReturnValue({
      data: { users: [] },
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    })
  })

  afterEach(() => {
    cleanup()
  })

  it("renders loading state when isCheckingAuth is true", () => {
    mockUseAuthStore.mockReturnValue({
      email: "test@example.com",
      isAdmin: false,
      isCheckingAuth: true,
      selectedOrgId: "org-123",
      permissions: [],
    })

    render(<UsersPage />)
    expect(screen.getByText("Verifying access...")).not.toBeNull()
  })

  it("renders UnauthorizedScreen when user does not have user:list permission", () => {
    mockUseAuthStore.mockReturnValue({
      email: "unauthorized@example.com",
      isAdmin: false,
      isCheckingAuth: false,
      selectedOrgId: "org-123",
      permissions: ["project:read"],
    })

    render(<UsersPage />)

    expect(screen.getByText("Access Restricted")).not.toBeNull()
    expect(
      screen.getByText(
        "You do not have permission to view the organization member directory or access IAM settings."
      )
    ).not.toBeNull()
    expect(screen.getByText("user:list")).not.toBeNull()
  })

  it("renders IAM page when user has user:list permission", () => {
    mockUseAuthStore.mockReturnValue({
      email: "user@example.com",
      isAdmin: false,
      isCheckingAuth: false,
      selectedOrgId: "org-123",
      permissions: ["user:list"],
    })

    render(<UsersPage />)

    expect(screen.queryByText("Access Restricted")).toBeNull()
    expect(screen.getByText("User Management")).not.toBeNull()
    expect(screen.getByText("Member Directory")).not.toBeNull()
  })

  it("renders IAM page when user is admin regardless of permissions array", () => {
    mockUseAuthStore.mockReturnValue({
      email: "admin@example.com",
      isAdmin: true,
      isCheckingAuth: false,
      selectedOrgId: "org-123",
      permissions: [],
    })

    render(<UsersPage />)

    expect(screen.queryByText("Access Restricted")).toBeNull()
    expect(screen.getByText("User Management")).not.toBeNull()
  })
})
