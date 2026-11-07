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
import { render, screen, fireEvent, cleanup, waitFor } from "@testing-library/react"
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import LoginPage from "./page"
import { useAuthStore } from "@/stores/auth"

const mockPush = vi.fn()
const mockGetSearchParams = vi.fn().mockReturnValue(new URLSearchParams())

vi.mock("next/navigation", () => ({
  useRouter: () => ({
    push: mockPush,
  }),
  useSearchParams: () => mockGetSearchParams(),
}))

vi.mock("@/stores/auth", () => {
  const store = {
    login: vi.fn(),
    selectOrganization: vi.fn(),
    discoverTenant: vi.fn(),
    loadOrganizations: vi.fn(),
    logout: vi.fn(),
    selectedOrgId: null,
  }
  const useAuthStoreMock: any = vi.fn(() => store)
  useAuthStoreMock.getState = vi.fn(() => store)
  return {
    useAuthStore: useAuthStoreMock,
  }
})

vi.mock("@/lib/firebase", () => ({
  auth: { tenantId: null },
  isFirebaseMock: true,
}))

describe("LoginPage Component", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useAuthStore.getState().selectedOrgId = null
  })

  afterEach(() => {
    cleanup()
  })

  it("renders username/email input and advances on submit", async () => {
    ;(useAuthStore.getState().discoverTenant as any).mockResolvedValue([])

    render(<LoginPage />)

    const input = screen.getByPlaceholderText(/Enter your email or username/i)
    fireEvent.change(input, { target: { value: "admin" } })

    const nextBtn = screen.getByRole("button", { name: /NEXT/i })
    fireEvent.click(nextBtn)

    await waitFor(() => {
      expect(screen.getByPlaceholderText(/Enter your password/i)).toBeDefined()
    })
  })

  it("auto-selects single organization and launches /projects when login succeeds", async () => {
    ;(useAuthStore.getState().login as any).mockResolvedValue(true)
    ;(useAuthStore.getState().loadOrganizations as any).mockResolvedValue([
      { id: "org-solo", name: "Solo Org" },
    ])
    ;(useAuthStore.getState().selectOrganization as any).mockResolvedValue(true)

    render(<LoginPage />)

    // Phase 1: Email/Username
    const userInput = screen.getByPlaceholderText(/Enter your email or username/i)
    fireEvent.change(userInput, { target: { value: "admin" } })
    fireEvent.click(screen.getByRole("button", { name: /NEXT/i }))

    // Phase 2: Password
    await waitFor(() => {
      expect(screen.getByPlaceholderText(/Enter your password/i)).toBeDefined()
    })

    const passInput = screen.getByPlaceholderText(/Enter your password/i)
    fireEvent.change(passInput, { target: { value: "admin" } })
    fireEvent.click(screen.getByRole("button", { name: /LOG IN/i }))

    await waitFor(() => {
      expect(useAuthStore.getState().selectOrganization).toHaveBeenCalledWith("org-solo")
      expect(mockPush).toHaveBeenCalledWith("/projects")
    })
  })

  it("navigates to /select-org when login succeeds and user has multiple organizations", async () => {
    ;(useAuthStore.getState().login as any).mockResolvedValue(true)
    ;(useAuthStore.getState().loadOrganizations as any).mockResolvedValue([
      { id: "org-1", name: "Org 1" },
      { id: "org-2", name: "Org 2" },
    ])

    render(<LoginPage />)

    const userInput = screen.getByPlaceholderText(/Enter your email or username/i)
    fireEvent.change(userInput, { target: { value: "admin" } })
    fireEvent.click(screen.getByRole("button", { name: /NEXT/i }))

    await waitFor(() => {
      expect(screen.getByPlaceholderText(/Enter your password/i)).toBeDefined()
    })

    const passInput = screen.getByPlaceholderText(/Enter your password/i)
    fireEvent.change(passInput, { target: { value: "admin" } })
    fireEvent.click(screen.getByRole("button", { name: /LOG IN/i }))

    await waitFor(() => {
      expect(mockPush).toHaveBeenCalledWith("/select-org")
    })
  })
})
