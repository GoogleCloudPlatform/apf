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

"use client"

import { useEffect, useState } from "react"
import { usePathname, useRouter } from "next/navigation"
import { useAuthStore } from "@/stores/auth"
import { useHealthStore } from "@/stores/health"
import { useThemeStore } from "@/stores/theme"

interface RouteGuardProps {
  children: React.ReactNode
}

export function RouteGuard({ children }: RouteGuardProps) {
  const router = useRouter()
  const pathname = usePathname()

  const { token, isCheckingAuth, isAuthenticated, selectedOrgId, isSuperAdmin, init: initAuth } = useAuthStore()
  const { isAvailable, isChecking: isCheckingHealth, checkHealth } = useHealthStore()
  const { init: initTheme } = useThemeStore()

  const [isInitializing, setIsInitializing] = useState(true)

  const isDebug = process.env.NODE_ENV === "development" || typeof window === "undefined"

  // Run global environment initializations
  useEffect(() => {
    const initializeApp = async () => {
      // Fire in parallel
      await Promise.all([checkHealth(), initAuth()])
      initTheme() // Load saved visual theme options
      setIsInitializing(false)
    }

    initializeApp()
  }, [checkHealth, initAuth, initTheme])

  // Process route gating actions whenever state variables mutate
  useEffect(() => {
    if (isInitializing) return

    const isNotAvailablePage = pathname === "/not-available"
    const isLoginPage = pathname === "/login"
    const isLoadingPage = pathname === "/loading"
    const isRootPage = pathname === "/"

    // 1. Gating Step: If backend is DOWN
    if (!isAvailable) {
      if (!isNotAvailablePage) {
        router.replace("/not-available")
      }
      return
    }

    // If backend is healthy but client is stuck on /not-available, push back to loading
    if (isAvailable && isNotAvailablePage) {
      router.replace("/")
      return
    }

    // 2. Gating Step: Wait for auth parsing routines
    if (isCheckingAuth) {
      if (!isLoadingPage && !isRootPage) {
        router.replace("/")
      }
      return
    }

    // 3. Gating Step: Route gating targets based on login state
    if (!isAuthenticated) {
      if (isDebug) {
        if (!isLoginPage) {
          router.replace("/login")
        }
      } else {
        // In production, redirect to root to trigger IAP login if session is lost
        if (pathname !== "/") {
          router.replace("/")
        }
      }
      return
    }

    const isSelectOrgPage = pathname === "/select-org"
    const isUnauthorizedPage = pathname === "/unauthorized"
    const isAdminPath = pathname ? pathname.startsWith("/admin") : false

    if (isUnauthorizedPage) {
      return
    }

    if (isSuperAdmin) {
      // Super Admin can always access admin paths
      if (isAdminPath) {
        return
      }
      // For other paths, they need an organization selected
      if (!selectedOrgId) {
        if (!isSelectOrgPage) {
          router.replace("/select-org")
        }
      } else {
        if (isLoginPage || isLoadingPage || isRootPage || isSelectOrgPage) {
          router.replace("/projects")
        }
      }
    } else {
      // Regular User
      if (isAdminPath) {
        // Cannot access admin paths -> Redirect to /unauthorized screen
        router.replace("/unauthorized?reason=superadmin")
      } else if (!selectedOrgId) {
        if (!isSelectOrgPage) {
          router.replace("/select-org")
        }
      } else {
        if (isLoginPage || isLoadingPage || isRootPage || isSelectOrgPage) {
          router.replace("/projects")
        }
      }
    }
  }, [isInitializing, isAvailable, isCheckingAuth, isAuthenticated, selectedOrgId, pathname, router, isSuperAdmin])

  const isLoginPage = pathname === "/login"
  const isSelectOrgPage = pathname === "/select-org"
  const isUnauthorizedPage = pathname === "/unauthorized"
  const isLoadingPage = pathname === "/loading"
  const isNotAvailablePage = pathname === "/not-available"
  const isRootPage = pathname === "/"

  const isAdminPath = pathname ? pathname.startsWith("/admin") : false

  // Determine if we should show the loader to prevent rendering children during pending redirects
  const showLoader =
    isInitializing ||
    isCheckingHealth ||
    isCheckingAuth ||
    // Backend down, redirecting to /not-available
    (!isAvailable && !isNotAvailablePage) ||
    // Backend up, redirecting away from /not-available
    (isAvailable && isNotAvailablePage) ||
    // Not authenticated, redirecting to login/root
    (!isAuthenticated && (isDebug ? (!isLoginPage && !isLoadingPage && !isRootPage) : !isRootPage)) ||
    // Authenticated:
    (isAuthenticated && !isUnauthorizedPage && (
      // Case 1: Regular user trying to access admin paths (redirecting away)
      (!isSuperAdmin && isAdminPath) ||
      // Case 2: Any user on a allowed non-admin path, but missing org (redirecting to /select-org)
      (!isAdminPath && !selectedOrgId && !isSelectOrgPage) ||
      // Case 3: Any user has org, but is on auth/select-org pages (redirecting to /projects)
      (selectedOrgId && (isLoginPage || isLoadingPage || isRootPage || isSelectOrgPage))
    ))

  // Render full-screen loader while loading environment parameters or redirecting
  if (showLoader) {
    return (
      <div className="flex h-screen w-screen flex-col items-center justify-center bg-background text-foreground gap-4">
        <div className="relative h-10 w-10 animate-spin rounded-full border-4 border-muted border-t-primary" />
        <span className="text-sm text-muted-foreground font-medium animate-pulse">
          Connecting to Foundry Workspace...
        </span>
      </div>
    )
  }

  // Otherwise permit child page nodes rendering
  return <>{children}</>
}
export default RouteGuard
