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
import { render, screen, cleanup, fireEvent } from "@testing-library/react"
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import OrganizationDetailPage from "./page"
import { useAuthStore } from "@/stores/auth"
import { useAppStore } from "@/stores/app"

let mockSearchParamsGet = vi.fn().mockReturnValue(null)

vi.mock("next/navigation", () => ({
  useRouter: () => ({
    push: vi.fn(),
    replace: vi.fn(),
    back: vi.fn(),
  }),
  useSearchParams: () => ({
    get: (param: string) => mockSearchParamsGet(param),
  }),
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

describe("OrganizationDetailPage", () => {
  const mockSetPageTitle = vi.fn()
  const mockUseAuthStore = useAuthStore as unknown as ReturnType<typeof vi.fn>
  const mockUseAppStore = useAppStore as unknown as ReturnType<typeof vi.fn>

  beforeEach(() => {
    vi.clearAllMocks()
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ name: "Existing Org", tenant_id: "existing-tenant-id" }),
    }) as any
    mockSearchParamsGet = vi.fn().mockReturnValue(null) // default: isNew = true
    mockUseAppStore.mockReturnValue({
      setPageTitle: mockSetPageTitle,
    })
    mockUseAuthStore.mockReturnValue({
      token: "fake-jwt-token",
      isSuperAdmin: true,
      isCheckingAuth: false,
    })
  })

  afterEach(() => {
    cleanup()
  })

  it("renders Display Name input when creating a new organization", () => {
    render(<OrganizationDetailPage />)
    
    const displayNameLabel = screen.getByLabelText("Display Name")
    expect(displayNameLabel).not.toBeNull()
  })

  it("auto-populates Organization Name with lowercased display name and replaces spaces/special chars (except underscores) with hyphens", () => {
    render(<OrganizationDetailPage />)

    const displayNameInput = screen.getByLabelText("Display Name") as HTMLInputElement
    const orgNameInput = screen.getByLabelText("Organization Name") as HTMLInputElement

    // Test case 1: Standard name with spaces
    fireEvent.change(displayNameInput, { target: { value: "Acme Corporation" } })
    expect(orgNameInput.value).toBe("acme-corporation")

    // Test case 2: Name with underscores (underscores should be preserved)
    fireEvent.change(displayNameInput, { target: { value: "My_Awesome_Org" } })
    expect(orgNameInput.value).toBe("my_awesome_org")

    // Test case 3: Name with special characters and spaces
    fireEvent.change(displayNameInput, { target: { value: "Super & Special @ Org!" } })
    expect(orgNameInput.value).toBe("super---special---org-")
  })

  it("does not render Display Name input when editing an existing organization", () => {
    mockSearchParamsGet.mockReturnValue("existing-org-id")
    render(<OrganizationDetailPage />)

    expect(screen.queryByLabelText("Display Name")).toBeNull()
  })
})
