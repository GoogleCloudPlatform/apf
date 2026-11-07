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

import { useEffect, useState, useRef } from "react"
import { useRouter } from "next/navigation"
import { useAuthStore } from "@/stores/auth"
import { Building2, ArrowRight, RefreshCw, LogOut, ChevronDown, Search, Check } from "lucide-react"

/**
 * SelectOrganizationPage renders the organization picker screen.
 * Restricts access until the authenticated user has targeted a tenant scope.
 */
export default function SelectOrganizationPage() {
  const router = useRouter()
  const { organizations, loadOrganizations, selectOrganization, logout, isCheckingAuth } = useAuthStore()

  const [selectedId, setSelectedId] = useState("")
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [isLoadingOrgs, setIsLoadingOrgs] = useState(true)
  const [isAutoSelecting, setIsAutoSelecting] = useState(false)
  const [autoSelectError, setAutoSelectError] = useState("")
  const [autoSelectedOrgName, setAutoSelectedOrgName] = useState("")
  const [isOpen, setIsOpen] = useState(false)
  const [searchQuery, setSearchQuery] = useState("")
  const dropdownRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    let isMounted = true

    const fetchOrgs = async () => {
      setIsLoadingOrgs(true)
      setAutoSelectError("")
      const orgs = await loadOrganizations()
      if (!isMounted) return

      if (orgs.length === 1) {
        const singleOrg = orgs[0]
        setSelectedId(singleOrg.id)
        setAutoSelectedOrgName(singleOrg.name)
        setIsAutoSelecting(true)
        setIsLoadingOrgs(false)

        const success = await selectOrganization(singleOrg.id)
        if (!isMounted) return
        if (success) {
          router.push("/projects")
          return
        } else {
          setIsAutoSelecting(false)
          setAutoSelectError(`Failed to auto-select organization "${singleOrg.name}". Please try selecting manually.`)
        }
      } else {
        setIsLoadingOrgs(false)
      }
    }

    fetchOrgs()
    return () => {
      isMounted = false
    }
  }, [loadOrganizations, selectOrganization, router])

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false)
      }
    }
    document.addEventListener("mousedown", handleClickOutside)
    return () => document.removeEventListener("mousedown", handleClickOutside)
  }, [])

  const filteredOrgs = organizations.filter(org =>
    org.name.toLowerCase().includes(searchQuery.toLowerCase())
  )

  const selectedOrg = organizations.find(org => org.id === selectedId)

  const handleSelect = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!selectedId) return

    setIsSubmitting(true)
    const success = await selectOrganization(selectedId)
    if (success) {
      router.push("/projects")
    } else {
      setIsSubmitting(false)
    }
  }

  const handleLogout = async () => {
    await logout()
    router.push("/login")
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-slate-50 dark:bg-slate-950 p-6">
      <div className="w-full max-w-[440px] rounded-2xl border border-border bg-card p-8 shadow-xl text-card-foreground">
        <div className="flex flex-col items-center">
          
          {/* Organisation Icon */}
          <div className="flex h-12 w-12 items-center justify-center mb-4 rounded-xl bg-primary/10 text-primary">
            <Building2 className="h-6 w-6" />
          </div>

          <h1 className="text-xl font-bold tracking-tight text-foreground mb-1 text-center">
            Select Organization
          </h1>
          <p className="text-sm text-muted-foreground text-center mb-8">
            Choose an organization to enter the workspace
          </p>

          {isLoadingOrgs ? (
            <div className="flex flex-col items-center py-8 gap-3">
              <RefreshCw className="h-6 w-6 animate-spin text-primary" />
              <span className="text-xs text-muted-foreground">Loading authorized organizations...</span>
            </div>
          ) : isAutoSelecting ? (
            <div className="flex flex-col items-center py-8 gap-3">
              <RefreshCw className="h-6 w-6 animate-spin text-primary" />
              <span className="text-sm font-medium text-foreground">
                Launching {autoSelectedOrgName || "workspace"}...
              </span>
              <span className="text-xs text-muted-foreground">Auto-connecting to organization</span>
            </div>
          ) : (
            <form onSubmit={handleSelect} className="w-full space-y-6">
              {autoSelectError && (
                <div className="text-xs font-medium text-destructive text-center py-2 px-3 bg-destructive/10 rounded-md border border-destructive/20">
                  {autoSelectError}
                </div>
              )}
              <div className="space-y-2 relative" ref={dropdownRef}>
                <label className="text-xs font-semibold text-muted-foreground uppercase" htmlFor="organization">
                  Active Organization
                </label>
                
                {organizations.length === 0 ? (
                  <div className="text-sm text-center py-4 text-destructive bg-destructive/10 rounded-md border border-destructive/20">
                    No authorized organizations found for your account.
                  </div>
                ) : (
                  <>
                    <button
                      type="button"
                      onClick={() => setIsOpen(!isOpen)}
                      className="flex h-11 w-full items-center justify-between rounded-md border border-input bg-card px-4 text-sm focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all text-foreground"
                    >
                      <div className="flex items-center gap-3 truncate">
                        <Building2 className="h-4 w-4 text-muted-foreground shrink-0" />
                        <span className="truncate">
                          {selectedOrg ? selectedOrg.name : "Select an organization..."}
                        </span>
                      </div>
                      <ChevronDown className="h-4 w-4 text-muted-foreground shrink-0" />
                    </button>

                    {isOpen && (
                      <div className="absolute z-50 mt-1 w-full rounded-md border border-border bg-card p-2 shadow-lg max-h-60 overflow-y-auto space-y-2">
                        {/* Search Input */}
                        <div className="relative">
                          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                          <input
                            type="text"
                            placeholder="Search organizations..."
                            value={searchQuery}
                            onChange={(e) => setSearchQuery(e.target.value)}
                            className="w-full h-9 pl-9 pr-3 rounded border border-input bg-muted/20 text-sm focus:outline-none focus:ring-1 focus:ring-primary focus:border-primary"
                          />
                        </div>

                        {/* List of Orgs */}
                        <div className="space-y-0.5">
                          {filteredOrgs.length === 0 ? (
                            <div className="text-center py-4 text-muted-foreground text-sm">
                              No organizations found
                            </div>
                          ) : (
                            filteredOrgs.map((org) => {
                              const isSelected = org.id === selectedId
                              return (
                                <button
                                  key={org.id}
                                  type="button"
                                  onClick={() => {
                                    setSelectedId(org.id)
                                    setIsOpen(false)
                                    setSearchQuery("")
                                  }}
                                  className="flex w-full items-center justify-between rounded px-3 py-2 text-left text-sm hover:bg-muted/50 text-foreground transition-colors"
                                >
                                  <div className="flex items-center gap-3 min-w-0">
                                    <Building2 className="h-4 w-4 text-muted-foreground shrink-0" />
                                    <span className="font-medium truncate">{org.name}</span>
                                  </div>
                                  {isSelected && <Check className="h-4 w-4 text-primary shrink-0 ml-2" />}
                                </button>
                              )
                            })
                          )}
                        </div>
                      </div>
                    )}
                  </>
                )}
              </div>

              {/* Submit and logout controls */}
              <div className="space-y-3 pt-2">
                <button
                  type="submit"
                  disabled={isSubmitting || !selectedId}
                  className="flex w-full h-11 items-center justify-center gap-2 rounded-md bg-primary text-primary-foreground text-sm font-bold shadow-md transition-all hover:opacity-90 disabled:opacity-50"
                >
                  {isSubmitting ? (
                    <RefreshCw className="h-4 w-4 animate-spin" />
                  ) : (
                    <>
                      ENTER FOUNDRY <ArrowRight className="h-4 w-4" />
                    </>
                  )}
                </button>

                <button
                  type="button"
                  onClick={handleLogout}
                  className="flex w-full h-10 items-center justify-center gap-2 rounded-md border border-input bg-transparent text-xs font-semibold text-muted-foreground transition-all hover:bg-muted/50"
                >
                  <LogOut className="h-3.5 w-3.5" /> LOG OUT
                </button>
              </div>
            </form>
          )}
        </div>
      </div>
    </main>
  )
}
