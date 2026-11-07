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

import { useState, useEffect, Suspense } from "react"
import { useRouter, useSearchParams } from "next/navigation"
import { useAuthStore } from "@/stores/auth"
import { auth } from "@/lib/firebase"
import Image from "next/image"
import { Lock, User, RefreshCw, Globe, ArrowLeft } from "lucide-react"

/**
 * LoginScreen Next.js port.
 * Renders credentials submission forms with status selectors and error alerts.
 */
function LoginContent() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const { login, selectOrganization, discoverTenant } = useAuthStore()
  
  const urlTenantId = searchParams.get("tenantId") || ""
  
  // Phase management
  const [phase, setPhase] = useState<"email" | "organization" | "password">("email")
  const [usernameOrEmail, setUsernameOrEmail] = useState("")
  const [password, setPassword] = useState("")
  const [discoveredOrgs, setDiscoveredOrgs] = useState<{ id: string; name: string; tenant_id?: string }[]>([])
  const [selectedOrg, setSelectedOrg] = useState<{ id: string; name: string; tenant_id?: string } | null>(null)
  const [activeTenantId, setActiveTenantId] = useState("")
  const [activeOrgId, setActiveOrgId] = useState("")
  const [isLoading, setIsLoading] = useState(false)
  const [errorMessage, setErrorMessage] = useState("")

  // If a tenantId is pre-provided in the URL, skip discovery and go straight to password phase!
  useEffect(() => {
    if (urlTenantId) {
      console.log("[Login Page] URL tenant ID detected, skipping discovery:", urlTenantId)
      setActiveTenantId(urlTenantId)
      setSelectedOrg({ id: urlTenantId, name: urlTenantId, tenant_id: urlTenantId })
      setPhase("password")
      auth.tenantId = urlTenantId
    } else {
      auth.tenantId = null
    }
  }, [urlTenantId])

  // Sync tenantId scope to Firebase Auth whenever activeTenantId changes
  useEffect(() => {
    if (activeTenantId) {
      console.log("[Login Page] Scoping Firebase Auth SDK to tenant ID:", activeTenantId)
      auth.tenantId = activeTenantId
    } else {
      console.log("[Login Page] Clearing tenant scope from Firebase Auth SDK")
      auth.tenantId = null
    }
  }, [activeTenantId])

  const handleNext = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!usernameOrEmail.trim()) {
      setErrorMessage("Please enter your email or username.")
      return
    }

    setIsLoading(true)
    setErrorMessage("")

    // Local simulated bypass check (admin and super-admin skip discovery)
    const trimmedInput = usernameOrEmail.trim()
    const isMockBypass = trimmedInput === "admin" || trimmedInput === "super-admin" || trimmedInput === "org-owner" || trimmedInput.startsWith("org-owner") || trimmedInput.startsWith("user1") || trimmedInput.startsWith("user2")
    if (isMockBypass) {
      setIsLoading(false)
      setActiveTenantId("")
      setActiveOrgId("")
      setSelectedOrg(null)
      setPhase("password")
      return
    }

    try {
      const orgs = await discoverTenant(trimmedInput)
      setIsLoading(false)

      if (orgs.length === 0) {
        setErrorMessage("No organization found for this email address.")
      } else if (orgs.length === 1) {
        setSelectedOrg(orgs[0])
        setActiveTenantId(orgs[0].tenant_id || orgs[0].id)
        setActiveOrgId(orgs[0].id)
        setPhase("password")
      } else {
        setDiscoveredOrgs(orgs)
        setPhase("organization")
      }
    } catch (err: any) {
      console.error("[Login Page] Discovery error:", err)
      setIsLoading(false)
      setErrorMessage("Failed to look up organizations. Please try again.")
    }
  }

  const handleOrgSelect = (org: { id: string; name: string; tenant_id?: string }) => {
    setSelectedOrg(org)
    setActiveTenantId(org.tenant_id || org.id)
    setActiveOrgId(org.id)
    setPhase("password")
  }

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!password.trim()) {
      setErrorMessage("Please enter your password.")
      return
    }

    setIsLoading(true)
    setErrorMessage("")

    try {
      const success = await login(usernameOrEmail.trim(), password.trim(), activeTenantId)

      if (success) {
        if (activeTenantId) {
          // Resolve the correct Firestore Org ID (from discovery state, or fall back to store claims)
          const orgId = activeOrgId || useAuthStore.getState().selectedOrgId || activeTenantId
          console.log("[Login Page] Selecting organization silo:", orgId)
          const orgSuccess = await selectOrganization(orgId)
          if (orgSuccess) {
            router.push("/projects")
          } else {
            console.warn("[Login Page] Organization selection failed. Rolling back login.")
            await useAuthStore.getState().logout()
            setIsLoading(false)
            setErrorMessage("Failed to bind session to organization silo.")
          }
        } else {
          const storeState = useAuthStore.getState()
          if (storeState.selectedOrgId) {
            router.push("/projects")
          } else {
            const orgs = await storeState.loadOrganizations()
            if (orgs.length === 1) {
              const orgSuccess = await selectOrganization(orgs[0].id)
              if (orgSuccess) {
                router.push("/projects")
              } else {
                router.push("/select-org")
              }
            } else {
              router.push("/select-org")
            }
          }
        }
      } else {
        setIsLoading(false)
        setErrorMessage("Invalid username or password.")
      }
    } catch (err: any) {
      console.error("[Login Page] Login error:", err)
      setIsLoading(false)
      setErrorMessage(err.message || "An unexpected error occurred during login.")
    }
  }

  const handleBack = () => {
    setErrorMessage("")
    if (phase === "password" && discoveredOrgs.length > 1 && !urlTenantId) {
      setPhase("organization")
    } else {
      setPhase("email")
      if (!urlTenantId) {
        setActiveTenantId("")
        setActiveOrgId("")
        setSelectedOrg(null)
      }
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-slate-50 dark:bg-slate-950 p-6">
      <div className="w-full max-w-[400px] rounded-2xl border border-border bg-card p-8 shadow-xl text-card-foreground relative overflow-hidden">
        
        {/* Universal Back Navigation Arrow */}
        {phase !== "email" && !urlTenantId && (
          <button
            onClick={handleBack}
            className="absolute top-6 left-6 text-muted-foreground hover:text-foreground transition-colors p-1 rounded-md hover:bg-muted"
            title="Go Back"
          >
            <ArrowLeft className="h-4 w-4" />
          </button>
        )}

        <div className="flex flex-col items-center">
          
          {/* Visual Modern Icon Logo Group */}
          <div className="flex h-16 w-16 items-center justify-center mb-5">
            <Image
              src="/images/agentic-foundry-logo.png"
              alt="Agentic Foundry Logo"
              width={64}
              height={64}
              className="object-contain"
              priority
            />
          </div>

          <h1 className="text-xl font-bold tracking-tight text-foreground mb-1">
            Agentic Product Foundry
          </h1>
          <p className="text-xs text-muted-foreground mb-8 uppercase tracking-widest font-semibold">
            Enterprise Ready AIDLC
          </p>

          {/* Phase 1: Email Entry Form */}
          {phase === "email" && (
            <form onSubmit={handleNext} className="w-full space-y-5">
              <div className="space-y-2">
                <label className="text-xs font-semibold text-muted-foreground uppercase" htmlFor="username">
                  Email or Username
                </label>
                <div className="relative">
                  <span className="absolute inset-y-0 left-0 flex items-center pl-3 text-muted-foreground">
                    <User className="h-4 w-4" />
                  </span>
                  <input
                    id="username"
                    type="text"
                    placeholder="Enter your email or username"
                    value={usernameOrEmail}
                    onChange={(e) => setUsernameOrEmail(e.target.value)}
                    disabled={isLoading}
                    className="w-full h-11 pl-10 pr-4 rounded-md border border-input bg-transparent text-sm focus:outline-none focus:ring-2 focus:ring-primary/40 focus:border-primary disabled:opacity-50"
                    autoFocus
                    autoComplete="email"
                  />
                </div>
              </div>

              {errorMessage && (
                <div className="text-xs font-medium text-destructive text-center py-1">
                  {errorMessage}
                </div>
              )}

              <button
                type="submit"
                disabled={isLoading}
                className="flex w-full h-11 items-center justify-center gap-2 rounded-md bg-primary text-primary-foreground text-sm font-bold shadow-md transition-all hover:opacity-90 disabled:opacity-50 mt-2"
              >
                {isLoading ? (
                  <RefreshCw className="h-4 w-4 animate-spin" />
                ) : (
                  "NEXT"
                )}
              </button>
            </form>
          )}

          {/* Phase 2: Organization Selector List */}
          {phase === "organization" && (
            <div className="w-full space-y-4">
              <div className="text-center space-y-1">
                <h2 className="text-sm font-bold text-foreground">Multiple Tenants Found</h2>
                <p className="text-xs text-muted-foreground">Select which organization you want to access:</p>
              </div>

              <div className="max-h-[200px] overflow-y-auto pr-1 space-y-2">
                {discoveredOrgs.map((org) => (
                  <button
                    key={org.id}
                    onClick={() => handleOrgSelect(org)}
                    className="flex w-full items-center gap-3 px-4 py-3 rounded-lg border border-border bg-card hover:bg-muted text-left text-sm font-medium transition-colors"
                  >
                    <Globe className="h-4 w-4 text-primary" />
                    <div className="flex-1 min-w-0">
                      <div className="truncate text-foreground">{org.name}</div>
                      <div className="text-[10px] text-muted-foreground font-mono truncate">{org.id}</div>
                    </div>
                  </button>
                ))}
              </div>

              {errorMessage && (
                <div className="text-xs font-medium text-destructive text-center py-1">
                  {errorMessage}
                </div>
              )}
            </div>
          )}

          {/* Phase 3: Password Entry Form */}
          {phase === "password" && (
            <form onSubmit={handleLogin} className="w-full space-y-5">
              
              {/* Organization Silo Badge */}
              {selectedOrg && (
                <div className="flex flex-col items-center justify-center gap-1 p-3 rounded-xl bg-primary/5 border border-primary/10 text-xs text-primary font-medium">
                  <div className="flex items-center gap-1.5 font-bold">
                    <Globe className="h-4 w-4" />
                    <span>{selectedOrg.name}</span>
                  </div>
                  <span className="text-[9px] text-muted-foreground font-mono">{selectedOrg.id}</span>
                </div>
              )}

              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <label className="text-xs font-semibold text-muted-foreground uppercase" htmlFor="password">
                    Password
                  </label>
                  {!urlTenantId && (
                    <span className="text-[10px] text-muted-foreground truncate max-w-[180px]">
                      as <code className="font-mono text-foreground font-semibold">{usernameOrEmail}</code>
                    </span>
                  )}
                </div>
                <div className="relative">
                  <span className="absolute inset-y-0 left-0 flex items-center pl-3 text-muted-foreground">
                    <Lock className="h-4 w-4" />
                  </span>
                  <input
                    id="password"
                    type="password"
                    placeholder="Enter your password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    disabled={isLoading}
                    className="w-full h-11 pl-10 pr-4 rounded-md border border-input bg-transparent text-sm focus:outline-none focus:ring-2 focus:ring-primary/40 focus:border-primary disabled:opacity-50"
                    autoFocus
                  />
                </div>
              </div>

              {errorMessage && (
                <div className="text-xs font-medium text-destructive text-center py-1">
                  {errorMessage}
                </div>
              )}

              <button
                type="submit"
                disabled={isLoading}
                className="flex w-full h-11 items-center justify-center gap-2 rounded-md bg-primary text-primary-foreground text-sm font-bold shadow-md transition-all hover:opacity-90 disabled:opacity-50 mt-2"
              >
                {isLoading ? (
                  <RefreshCw className="h-4 w-4 animate-spin" />
                ) : (
                  "LOG IN"
                )}
              </button>
            </form>
          )}

          <p className="text-[10px] text-muted-foreground/60 text-center mt-6">
            Default credentials for off-line development are <code className="font-mono bg-muted px-1.5 py-0.5 rounded text-foreground">admin / admin</code>.
          </p>
        </div>
      </div>
    </main>
  )
}


export default function LoginPage() {
  return (
    <Suspense fallback={
      <main className="flex min-h-screen items-center justify-center bg-slate-50 dark:bg-slate-950 p-6">
        <div className="flex flex-col items-center gap-2 text-muted-foreground">
          <RefreshCw className="h-8 w-8 animate-spin" />
          <span className="text-sm font-medium">Loading Login Portal...</span>
        </div>
      </main>
    }>
      <LoginContent />
    </Suspense>
  )
}
