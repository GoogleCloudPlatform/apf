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
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { useAuthStore } from "./auth"

describe("Auth Store - Organization Auto-selection & Management", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    useAuthStore.setState({
      token: null,
      email: null,
      displayName: null,
      isCheckingAuth: false,
      isAuthenticated: false,
      selectedOrgId: null,
      organizations: [],
    })
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it("loadOrganizations fetches organizations, updates store state, and returns the array", async () => {
    const mockOrgs = [
      { id: "org-1", name: "Primary Org" },
      { id: "org-2", name: "Secondary Org" },
    ]

    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ organizations: mockOrgs }),
    } as any)

    useAuthStore.setState({ token: "test-token", isAuthenticated: true })

    const result = await useAuthStore.getState().loadOrganizations()

    expect(global.fetch).toHaveBeenCalledWith("/api/v1/auth/organizations", expect.anything())
    expect(result).toEqual(mockOrgs)
    expect(useAuthStore.getState().organizations).toEqual(mockOrgs)
  })

  it("selectOrganization posts org_id, updates token and selectedOrgId in store and localStorage", async () => {
    const mockNewToken = "org-scoped-jwt-token"

    global.fetch = vi.fn().mockImplementation((url: string) => {
      if (url === "/api/v1/auth/select-organization") {
        return Promise.resolve({
          ok: true,
          json: async () => ({ token: mockNewToken }),
        })
      }
      if (url === "/api/v1/auth/me") {
        return Promise.resolve({
          ok: true,
          json: async () => ({
            sub: "test@example.com",
            organization_id: "org-solo",
            roles: ["org:admin"],
          }),
        })
      }
      return Promise.resolve({
        ok: true,
        json: async () => ({}),
      })
    })

    useAuthStore.setState({ token: "initial-token", isAuthenticated: true })

    const success = await useAuthStore.getState().selectOrganization("org-solo")

    expect(success).toBe(true)
    expect(useAuthStore.getState().selectedOrgId).toBe("org-solo")
    expect(useAuthStore.getState().token).toBe(mockNewToken)
    expect(localStorage.getItem("auth_token")).toBe(mockNewToken)
  })

  it("login automatically selects single organization when user belongs to exactly 1 org", async () => {
    const mockOrgs = [{ id: "org-single", name: "Single Corp" }]
    const mockLoginToken = "user-login-token"
    const mockOrgToken = "org-scoped-token-single"

    global.fetch = vi.fn().mockImplementation((url: string) => {
      if (url === "/api/v1/auth/login") {
        return Promise.resolve({
          ok: true,
          json: async () => ({ token: mockLoginToken }),
        })
      }
      if (url === "/api/v1/auth/organizations") {
        return Promise.resolve({
          ok: true,
          json: async () => ({ organizations: mockOrgs }),
        })
      }
      if (url === "/api/v1/auth/select-organization") {
        return Promise.resolve({
          ok: true,
          json: async () => ({ token: mockOrgToken }),
        })
      }
      if (url === "/api/v1/auth/me") {
        return Promise.resolve({
          ok: true,
          json: async () => ({
            sub: "user@example.com",
            organization_id: "org-single",
            roles: ["org:admin"],
          }),
        })
      }
      return Promise.resolve({
        ok: true,
        json: async () => ({}),
      })
    })

    const loginSuccess = await useAuthStore.getState().login("user@example.com", "password123")

    expect(loginSuccess).toBe(true)
    expect(useAuthStore.getState().selectedOrgId).toBe("org-single")
  })
})
