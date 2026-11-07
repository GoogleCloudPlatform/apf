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

import { create } from "zustand"
import { auth, signInWithEmailLink, updatePassword, signInWithPopup, isFirebaseMock } from "@/lib/firebase"

export interface Organization {
  id: string
  name: string
  tenant_id?: string
}

interface AuthState {
  token: string | null
  email: string | null
  displayName: string | null
  isCheckingAuth: boolean
  isRedirectingToIap: boolean
  
  // Reactive State Fields (replaces broken JS getters)
  isAuthenticated: boolean
  isAdmin: boolean
  isOwner: boolean
  isSuperAdmin: boolean
  canCreateProject: boolean
  permissions: string[]

  // Organization Fields
  selectedOrgId: string | null
  organizations: Organization[]

  // Actions
  init: () => Promise<void>
  login: (username: string, password: string, tenantId?: string) => Promise<boolean>
  discoverTenant: (email: string) => Promise<Organization[]>
  logout: () => Promise<void>
  switchOrganization: () => void
  handleUnauthorized: () => void
  _checkAuthStatus: (tokenToCheck: string | null) => Promise<void>
  loadOrganizations: () => Promise<Organization[]>
  selectOrganization: (orgId: string) => Promise<boolean>
  acceptInvitation: (
    email: string,
    tenantId: string,
    password?: string,
    token?: string,
    oobCode?: string
  ) => Promise<boolean>
  acceptInvitationWithGoogle: (email: string, tenantId: string) => Promise<boolean>
}

const TOKEN_KEY = "auth_token"

function parseJwt(token: string) {
  try {
    const base64Url = token.split(".")[1]
    const base64 = base64Url.replace(/-/g, "+").replace(/_/g, "/")
    const jsonPayload = decodeURIComponent(
      atob(base64)
        .split("")
        .map((c) => "%" + ("00" + c.charCodeAt(0).toString(16)).slice(-2))
        .join("")
    )
    return JSON.parse(jsonPayload)
  } catch (e) {
    return null
  }
}

// Helper utility to calculate dynamic role permissions based on the active token block
function getDerivedState(token: string | null) {
  const isAuthenticated = token !== null
  const jwtClaims = token ? parseJwt(token) : null

  const isSuperAdmin = (token !== null && token.startsWith("simulated-jwt-token-for-super-admin")) || (jwtClaims?.is_super_admin ?? false)
  const roles: string[] = jwtClaims?.roles || []
  const isOwner = (token !== null && (token.startsWith("simulated-jwt-token-for-admin") || token.startsWith("simulated-jwt-token-for-owner") || token.includes("org-owner"))) || roles.includes("org:owner") || isSuperAdmin
  const isAdmin = (token !== null && (token.startsWith("simulated-jwt-token-for-admin") || token.includes("org-owner") || token.includes("org-admin"))) || roles.includes("org:admin") || isOwner || isSuperAdmin
  const isUser1 = token !== null && token.startsWith("simulated-jwt-token-for-user1")
  
  let permissions: string[] = jwtClaims?.permissions || []
  if (permissions.length === 0 && token !== null) {
    if (token.startsWith("simulated-jwt-token-for-super-admin") || token.startsWith("simulated-jwt-token-for-admin") || token.includes("org-owner") || token.includes("org-admin") || token.includes("user-admin")) {
      permissions = [
        "org:create", "org:read", "org:update", "org:delete", "org:list",
        "project:create", "project:read", "project:update", "project:delete", "project:list",
        "user:create", "user:read", "user:update", "user:delete", "user:list",
        "role:create", "role:read", "role:update", "role:delete", "role:list"
      ]
    } else if (token.startsWith("simulated-jwt-token-for-user1")) {
      permissions = [
        "org:read", "org:list",
        "project:create", "project:read", "project:list",
        "user:read", "user:list",
        "role:read", "role:list"
      ]
    } else if (token.startsWith("simulated-jwt-token-for-user2")) {
      permissions = [
        "org:read", "org:list",
        "project:read", "project:list",
        "user:read", "user:list",
        "role:read", "role:list"
      ]
    }
  }

  return {
    token,
    isAuthenticated,
    isAdmin,
    isOwner,
    isSuperAdmin,
    canCreateProject: isAdmin || isUser1 || isSuperAdmin,
    permissions,
  }
}

export const useAuthStore = create<AuthState>((set, get) => ({
  // Initial state values
  token: null,
  email: null,
  displayName: null,
  isCheckingAuth: true,
  isRedirectingToIap: false,
  isAuthenticated: false,
  isAdmin: false,
  isOwner: false,
  isSuperAdmin: false,
  canCreateProject: false,
  permissions: [],
  selectedOrgId: null,
  organizations: [],

  init: async () => {
    if (typeof window === "undefined") return

    const savedToken = localStorage.getItem(TOKEN_KEY)
    if (savedToken) {
      set(getDerivedState(savedToken))
    }

    // Run remote status validation
    await get()._checkAuthStatus(savedToken)

    // Clear selectedOrgId if initializing directly on the select-org page
    if (window.location.pathname === "/select-org") {
      set({ selectedOrgId: null })
    }

    // Load organizations if authenticated
    if (get().isAuthenticated) {
      const orgs = await get().loadOrganizations()
      if (!get().selectedOrgId && orgs.length === 1 && typeof window !== "undefined" && window.location.pathname !== "/unauthorized") {
        await get().selectOrganization(orgs[0].id)
      }
    }
  },

  _checkAuthStatus: async (tokenToCheck: string | null) => {
    const currentToken = tokenToCheck || get().token
    try {
      const response = await fetch("/api/v1/auth/me", {
        method: "GET",
        headers: currentToken ? { Authorization: `Bearer ${currentToken}` } : {},
      })

      if (response.ok) {
        const decoded = await response.json()
        const resolvedToken = currentToken || "iap-authenticated"
        
        const roles = decoded.roles || []
        const isOwner = roles.includes("org:owner") || roles.includes("owner") || decoded.is_super_admin || false
        const isAdmin = roles.includes("org:admin") || roles.includes("admin") || isOwner
        const isSuperAdmin = decoded.is_super_admin || false
        const canCreateProject = isAdmin || roles.includes("project_creator") || roles.includes("project:admin") || roles.includes("project:editor")

        set({
          token: resolvedToken,
          isAuthenticated: true,
          isAdmin,
          isOwner,
          isSuperAdmin,
          canCreateProject,
          permissions: decoded.permissions || [],
          email: decoded.sub || "",
          displayName: decoded.username || "",
          selectedOrgId: decoded.organization_id || null,
          isRedirectingToIap: false,
          isCheckingAuth: false,
        })

        // Fire dynamic theme initialization and tour status loading in the background
        if (decoded.sub) {
          const { useThemeStore } = await import("./theme")
          useThemeStore.getState().loadThemeFromBackend(decoded.sub)

          const { useOnboardingStore } = await import("./onboarding")
          useOnboardingStore.getState().loadTourStatusFromBackend(decoded.sub)
        }
      } else if (response.status === 401) {
        get().handleUnauthorized()
      } else {
        set({ isCheckingAuth: false })
      }
    } catch (e) {
      console.warn("Authentication profile check failed:", e)
      set({ isCheckingAuth: false })
    }
  },

  login: async (username, password, tenantId) => {
    set({ isRedirectingToIap: false })
    
    // Local developer bypass: Skip real Firebase Auth for local mock admin accounts and simulated test users
    const isMockBypass = username === "super-admin" || username === "admin" || username.startsWith("org-owner") || username.startsWith("user") || !username.includes("@")
    
    // If running with real Firebase Auth Emulator, authenticate using the Client SDK!
    if (!isFirebaseMock && !isMockBypass) {
      try {
        if (!auth) {
          throw new Error("Firebase Auth Client is not initialized. Please verify your configuration.")
        }
        // Explicitly scope the Firebase Auth instance to the target tenant ID
        if (tenantId) {
          auth.tenantId = tenantId
          console.log(`[Auth Store] Scoped Firebase Auth to provided tenant ID: ${tenantId}`)
        } else if (!auth.tenantId) {
          const orgs = await get().discoverTenant(username)
          if (orgs && orgs.length > 0) {
            auth.tenantId = orgs[0].tenant_id || orgs[0].id
            console.log(`[Auth Store] Scoped Firebase Auth to autodiscovered tenant ID: ${auth.tenantId}`)
          }
        }

        const { signInWithEmailAndPassword } = await import("firebase/auth")
        const userCredential = await signInWithEmailAndPassword(auth as any, username, password)
        const idToken = await userCredential.user.getIdToken()
        
        localStorage.setItem(TOKEN_KEY, idToken)
        set({
          ...getDerivedState(idToken),
          isCheckingAuth: false,
        })
        await get()._checkAuthStatus(idToken)
        if (!get().selectedOrgId) {
          const orgs = await get().loadOrganizations()
          if (orgs.length === 1) {
            await get().selectOrganization(orgs[0].id)
          }
        }
        return true
      } catch (e: any) {
        console.error("[Firebase Client] Login failed:", e)
        throw new Error(e.message || "Authentication failed")
      }
    }

    try {
      const response = await fetch("/api/v1/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password }),
      })

      if (response.ok) {
        const decoded = await response.json()
        if (decoded && decoded.token) {
          const newToken = decoded.token
          localStorage.setItem(TOKEN_KEY, newToken)
          
          set({
            ...getDerivedState(newToken),
            isCheckingAuth: false,
          })
          
          await get()._checkAuthStatus(newToken)
          if (!get().selectedOrgId) {
            const orgs = await get().loadOrganizations()
            if (orgs.length === 1) {
              await get().selectOrganization(orgs[0].id)
            }
          }
          return true
        }
      }
      return false
    } catch (e) {
      console.warn("Backend auth failed, evaluating simulated login bypass:", e)
      // Standard Local simulated sandbox bypass (equivalent to Flutter AuthService check)
      if (process.env.NODE_ENV === "development" && username === "admin" && password === "admin") {
        const simulatedToken = "simulated-jwt-token-for-admin"
        localStorage.setItem(TOKEN_KEY, simulatedToken)
        
        set({
          ...getDerivedState(simulatedToken),
          email: "admin@agentic-foundry.local",
          displayName: "Administrator (Local Debug)",
          isCheckingAuth: false,
        })
        const orgs = await get().loadOrganizations()
        if (orgs.length === 1) {
          await get().selectOrganization(orgs[0].id)
        }
        return true
      } else if (process.env.NODE_ENV === "development" && username === "super-admin" && password === "super-admin") {
        const simulatedToken = "simulated-jwt-token-for-super-admin"
        localStorage.setItem(TOKEN_KEY, simulatedToken)
        
        set({
          ...getDerivedState(simulatedToken),
          email: "super-admin@agentic-foundry.local",
          displayName: "Super Administrator (Local Debug)",
          isCheckingAuth: false,
        })
        const orgs = await get().loadOrganizations()
        if (orgs.length === 1) {
          await get().selectOrganization(orgs[0].id)
        }
        return true
      }
      return false
    }
  },

  discoverTenant: async (email: string) => {
    try {
      const response = await fetch(`/api/v1/auth/discover?email=${encodeURIComponent(email)}`, {
        method: "GET",
        headers: { "Content-Type": "application/json" },
      })
      if (response.ok) {
        const data = await response.json()
        return data.organizations || []
      }
      return []
    } catch (e) {
      console.error("[Auth Store] Tenant discovery failed:", e)
      return []
    }
  },

  logout: async () => {
    const isDebug = process.env.NODE_ENV === "development" || typeof window === "undefined"

    if (!isDebug) {
      // In production, redirect to IAP logout directly to clear session.
      // After clearing, redirect to the application root, which will trigger IAP re-auth (IdP login page).
      const redirectUrl = window.location.origin
      window.location.href = `${redirectUrl}?gcp-iap-mode=CLEAR_LOGIN_COOKIE`
      return
    }

    const currentToken = get().token
    localStorage.removeItem(TOKEN_KEY)
    
    // Reset state parameters
    set({
      ...getDerivedState(null),
      email: null,
      displayName: null,
      isCheckingAuth: false,
      isRedirectingToIap: false,
      selectedOrgId: null,
      organizations: [],
    })

    if (currentToken) {
      try {
        await fetch("/api/v1/auth/logout", {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${currentToken}`,
          },
        })
      } catch (e) {
        console.warn("Logout endpoint call failed:", e)
      }
    }
  },

  switchOrganization: () => {
    set({ selectedOrgId: null })
  },

  handleUnauthorized: () => {
    localStorage.removeItem(TOKEN_KEY)
    
    set({
      ...getDerivedState(null),
      email: null,
      displayName: null,
      isCheckingAuth: false,
      selectedOrgId: null,
      organizations: [],
    })

    // Avoid loops and only trigger IAP redirects in non-debug deployments
    const isDebug = process.env.NODE_ENV === "development" || typeof window === "undefined"
    if (!isDebug && !get().isRedirectingToIap) {
      try {
        const currentUrl = new URL(window.location.href)
        if (!currentUrl.searchParams.has("gcp-iap-mode")) {
          set({ isRedirectingToIap: true })
          currentUrl.searchParams.set("gcp-iap-mode", "CLEAR_LOGIN_COOKIE")
          window.location.href = currentUrl.toString()
        }
      } catch (e) {
        // Fallback for relative or malformed URLs
        if (!window.location.href.includes("gcp-iap-mode")) {
          set({ isRedirectingToIap: true })
          const separator = window.location.href.includes("?") ? "&" : "?"
          window.location.href = `${window.location.href}${separator}gcp-iap-mode=CLEAR_LOGIN_COOKIE`
        }
      }
    }
  },

  loadOrganizations: async (): Promise<Organization[]> => {
    const currentToken = get().token
    try {
      const response = await fetch("/api/v1/auth/organizations", {
        method: "GET",
        headers: currentToken ? { Authorization: `Bearer ${currentToken}` } : {},
      })
      if (response.ok) {
        const data = await response.json()
        const orgs = data.organizations || []
        set({ organizations: orgs })
        return orgs
      }
    } catch (e) {
      console.warn("Failed to load organizations:", e)
    }
    return []
  },

  selectOrganization: async (orgId: string) => {
    const currentToken = get().token
    try {
      const response = await fetch("/api/v1/auth/select-organization", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...(currentToken ? { Authorization: `Bearer ${currentToken}` } : {}),
        },
        body: JSON.stringify({ org_id: orgId }),
      })
      if (response.ok) {
        const data = await response.json()
        if (data && data.token) {
          const orgToken = data.token
          localStorage.setItem(TOKEN_KEY, orgToken)
          set({ selectedOrgId: orgId, token: orgToken })
          await get()._checkAuthStatus(orgToken)
          return true
        }
      }
      return false
    } catch (e) {
      console.warn("Failed to select organization:", e)
      return false
    }
  },

  acceptInvitation: async (email, tenantId, password, token, oobCode) => {
    try {
      if (!auth) {
        throw new Error("Firebase Auth Client is not initialized. Please verify your configuration.")
      }
      // Scopes the auth client instance (real or mock) to the tenant ID
      auth.tenantId = tenantId

      // If mock mode, seed the mock email onto the auth client so wrappers can reconcile correctly
      if (isFirebaseMock) {
        (auth as any).mockEmail = email
      }

      // Call signInWithEmailLink via pluggable wrapper
      const href = typeof window !== "undefined" ? window.location.href : ""
      const result = await signInWithEmailLink(auth, email, href)

      if (!result || !result.user) {
        throw new Error("Authentication failed: No current user after sign in.")
      }

      // If a password is provided, update the user's credentials (wrapper handles mock gracefully)
      if (password) {
        await updatePassword(result.user, password)
      }

      const idToken = await result.user.getIdToken()

      let orgId = tenantId
      try {
        const payload = JSON.parse(atob(idToken.split(".")[1]))
        if (payload.organization_id) {
          orgId = payload.organization_id
          console.log(`[Auth Store] Resolved Firestore Org ID from token claims: ${orgId}`)
        }
      } catch (e) {
        console.warn("[Auth Store] Failed to decode ID token payload to extract organization_id, falling back to tenantId:", e)
      }

      // Set temporary state to allow selectOrganization to fetch organizations in the same context
      set({ token: idToken, email: email, isAuthenticated: true })

      // Call selectOrganization with the resolved Firestore Org ID
      const success = await get().selectOrganization(orgId)

      if (success) {
        localStorage.setItem(TOKEN_KEY, idToken)
        return true
      } else {
        console.warn("[Auth Store] Organization selection failed during invitation acceptance. Rolling back.")
        // Rollback state on organization selection failure
        localStorage.removeItem(TOKEN_KEY)
        set({ token: null, email: null, isAuthenticated: false })
        return false
      }
    } catch (e) {
      console.warn("Invitation acceptance failed:", e)
      // Rollback state on general failure
      localStorage.removeItem(TOKEN_KEY)
      set({ token: null, email: null, isAuthenticated: false })
      return false
    }
  },

  acceptInvitationWithGoogle: async (email, tenantId) => {
    try {
      if (!auth) {
        throw new Error("Firebase Auth Client is not initialized. Please verify your configuration.")
      }
      // Scopes the auth client instance (real or mock) to the tenant ID
      auth.tenantId = tenantId

      // If mock mode, seed the mock email onto the auth client so mock Google Auth resolves invitation reconciliation
      if (isFirebaseMock) {
        (auth as any).mockEmail = email
      }

      // Import dynamic google provider wrapper safely
      const { GoogleAuthProvider } = await import("@/lib/firebase")
      const provider = new GoogleAuthProvider()
      
      const result = await signInWithPopup(auth, provider)

      if (!result || !result.user) {
        throw new Error("Google authentication failed: No user returned.")
      }

      if (!result.user.email) {
        throw new Error("Google authentication failed: Email is missing from provider profile.")
      }

      // Critical Security Check: Ensure the authenticated Google email matches the invited email address
      if (result.user.email.toLowerCase() !== email.toLowerCase()) {
        throw new Error(`Google account email (${result.user.email}) does not match the invited email address (${email}).`)
      }

      const idToken = await result.user.getIdToken()

      let orgId = tenantId
      try {
        const payload = JSON.parse(atob(idToken.split(".")[1]))
        if (payload.organization_id) {
          orgId = payload.organization_id
          console.log(`[Auth Store] Resolved Firestore Org ID from Google token claims: ${orgId}`)
        }
      } catch (e) {
        console.warn("[Auth Store] Failed to decode Google ID token payload to extract organization_id, falling back to tenantId:", e)
      }

      // Set temporary state to allow selectOrganization to fetch organizations in the same context
      set({ token: idToken, email: result.user.email, isAuthenticated: true })

      // Call selectOrganization with the resolved Firestore Org ID
      const success = await get().selectOrganization(orgId)

      if (success) {
        localStorage.setItem(TOKEN_KEY, idToken)
        return true
      } else {
        console.warn("[Auth Store] Organization selection failed during Google invitation acceptance. Rolling back.")
        // Rollback state on organization selection failure
        localStorage.removeItem(TOKEN_KEY)
        set({ token: null, email: null, isAuthenticated: false })
        return false
      }
    } catch (e: any) {
      console.warn("Google invitation acceptance failed:", e)
      // Rollback state on general failure
      localStorage.removeItem(TOKEN_KEY)
      set({ token: null, email: null, isAuthenticated: false })
      throw e
    }
  },
}))
export default useAuthStore
