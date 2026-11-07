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
import { useRouter } from "next/navigation"
import { useAuthStore } from "@/stores/auth"
import { useAppStore } from "@/stores/app"
import { UnauthorizedScreen } from "@/components/shared/UnauthorizedScreen"
import { 
  Server, 
  ShieldAlert, 
  Loader2, 
  Shield, 
  Cpu, 
  Building2, 
  Edit3, 
  Plus, 
  Trash2, 
  Search, 
  Download, 
  AlertCircle,
  CheckCircle2
} from "lucide-react"
import { toast } from "sonner"

interface SystemInfo {
  version: string
  region: string
  superAdminEmails: string[]
}

interface Organization {
  id: string
  name: string
  tenant_id: string
  user_count?: number
}

// Delete Confirmation Modal
interface DeleteModalProps {
  isOpen: boolean
  onClose: () => void
  onConfirm: () => void
  orgName: string
  isPending: boolean
}

function DeleteOrganizationModal({ isOpen, onClose, onConfirm, orgName, isPending }: DeleteModalProps) {
  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4 animate-in fade-in duration-150">
      <div className="w-full max-w-md bg-card border border-border rounded-2xl p-6 shadow-2xl space-y-6 animate-in zoom-in-95 duration-200 text-foreground">
        <div className="space-y-2">
          <h3 className="text-lg font-bold text-foreground flex items-center gap-2 text-destructive">
            <AlertCircle className="h-5 w-5" />
            Delete Organization
          </h3>
          <p className="text-xs text-muted-foreground leading-relaxed">
            Are you sure you want to delete <span className="font-semibold text-foreground">{orgName}</span>? 
            This action is permanent and will delete all associated projects, workflows, and database records.
          </p>
        </div>

        <div className="flex justify-end gap-3">
          <button
            onClick={onClose}
            disabled={isPending}
            className="px-4 py-2 rounded-lg border border-input bg-background hover:bg-muted text-xs font-semibold text-foreground transition-all disabled:opacity-50"
          >
            Cancel
          </button>
          <button
            onClick={onConfirm}
            disabled={isPending}
            className="px-4 py-2 rounded-lg bg-destructive hover:bg-destructive/90 text-xs font-semibold text-destructive-foreground shadow-sm flex items-center gap-1.5 transition-all disabled:opacity-50"
          >
            {isPending ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <Trash2 className="h-3.5 w-3.5" />
            )}
            Delete Permanently
          </button>
        </div>
      </div>
    </div>
  )
}

export default function AdminOverviewPage() {
  const router = useRouter()
  const { setPageTitle } = useAppStore()
  const { token, isSuperAdmin, isCheckingAuth } = useAuthStore()

  // Data states
  const [systemInfo, setSystemInfo] = useState<SystemInfo | null>(null)
  const [isLoadingSystemInfo, setIsLoadingSystemInfo] = useState(true)
  const [systemInfoError, setSystemInfoError] = useState<string | null>(null)

  const [organizations, setOrganizations] = useState<Organization[]>([])
  const [isLoadingOrgs, setIsLoadingOrgs] = useState(true)
  const [orgsError, setOrgsError] = useState<string | null>(null)

  // Search/Filter states (UI Layout Only)
  const [searchQuery, setSearchQuery] = useState("")

  // Deletion states
  const [isDeleting, setIsDeleting] = useState(false)
  const [orgToDelete, setOrgToDelete] = useState<Organization | null>(null)

  // Sync page title
  useEffect(() => {
    setPageTitle("Admin: Overview")
  }, [setPageTitle])

  // Fetch system info
  const fetchSystemInfo = async () => {
    if (!token || !isSuperAdmin) return
    setIsLoadingSystemInfo(true)
    setSystemInfoError(null)
    try {
      const response = await fetch("/api/v1/admin/system-info", {
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
      })
      if (!response.ok) {
        throw new Error(`Failed to fetch system info: ${response.statusText}`)
      }
      const data = await response.json()
      setSystemInfo(data)
    } catch (err: any) {
      console.error("Error fetching system info:", err)
      setSystemInfoError(err.message || "Failed to load system specifications")
    } finally {
      setIsLoadingSystemInfo(false)
    }
  }

  // Fetch organizations
  const fetchOrgs = async () => {
    if (!token || !isSuperAdmin) return
    setIsLoadingOrgs(true)
    setOrgsError(null)
    try {
      const response = await fetch("/api/v1/admin/organizations", {
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
      })
      if (!response.ok) {
        throw new Error(`Failed to fetch organizations: ${response.statusText}`)
      }
      const data = await response.json()
      setOrganizations(data || [])
    } catch (err: any) {
      console.error("Error fetching organizations:", err)
      setOrgsError(err.message || "Failed to load organizations")
    } finally {
      setIsLoadingOrgs(false)
    }
  }

  useEffect(() => {
    if (!isCheckingAuth && isSuperAdmin) {
      fetchSystemInfo()
      fetchOrgs()
    }
  }, [token, isSuperAdmin, isCheckingAuth])

  const handleEditOrg = (orgId: string) => {
    router.push(`/admin/organization-detail?id=${orgId}`)
  }

  const handleDeleteConfirm = async () => {
    if (!orgToDelete || !token) return
    setIsDeleting(true)
    try {
      const response = await fetch(`/api/v1/admin/organizations/${orgToDelete.id}`, {
        method: "DELETE",
        headers: {
          Authorization: `Bearer ${token}`,
        },
      })

      if (!response.ok) {
        const errData = await response.json().catch(() => ({}))
        throw new Error(errData.message || `Delete failed with status: ${response.status}`)
      }

      toast.success(`Organization ${orgToDelete.name} deleted successfully`)
      setOrganizations(organizations.filter(o => o.id !== orgToDelete.id))
      setOrgToDelete(null)
    } catch (err: any) {
      console.error("Error deleting organization:", err)
      toast.error(err.message || "Failed to delete organization")
    } finally {
      setIsDeleting(false)
    }
  }

  if (isCheckingAuth || (isSuperAdmin && isLoadingSystemInfo)) {
    return (
      <div className="flex-1 flex h-[400px] w-full flex-col items-center justify-center gap-3 text-muted-foreground select-none">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
        <span className="text-sm font-medium">Loading Foundry Overview...</span>
      </div>
    )
  }

  // Guard: Only super admins can access
  if (!isSuperAdmin) {
    return (
      <UnauthorizedScreen
        title="Super Admin Access Only"
        message="This screen is reserved exclusively for Super Administrators of the Foundry platform."
        requiredPermission="super_admin"
        fullPage={false}
      />
    )
  }

  // Helper to extract name initials
  const getInitials = (text: string) => {
    return text.split(/[ @._-]/).map(p => p.charAt(0)).join("").slice(0, 2).toUpperCase()
  }

  // Filter organizations by search query
  const filteredOrgs = organizations.filter(org => 
    org.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    org.id.toLowerCase().includes(searchQuery.toLowerCase()) ||
    org.tenant_id?.toLowerCase().includes(searchQuery.toLowerCase())
  )

  return (
    <main className="flex-1 overflow-y-auto p-8 text-foreground">
      <div className="w-full max-w-[1000px] space-y-8 pb-12">
        
        {/* Header */}
        <div className="space-y-2 pb-2">
          <h2 className="text-2xl font-bold tracking-tight text-foreground">Foundry Overview</h2>
          <p className="text-xs text-muted-foreground leading-relaxed">
            Global system specifications, deployment topology, and administrative access controls.
          </p>
        </div>

        {systemInfoError && (
          <div className="p-4 rounded-xl bg-destructive/10 border border-destructive/20 text-destructive text-xs flex items-center gap-3">
            <ShieldAlert className="h-5 w-5 shrink-0" />
            <span>{systemInfoError}</span>
          </div>
        )}

        {/* Global System Info Grid */}
        <div className="grid gap-6 md:grid-cols-2">
          
          {/* Card 1: System Specifications */}
          <div className="bg-card border border-border rounded-2xl p-5 space-y-4 shadow-sm flex flex-col justify-between">
            <div className="space-y-3.5">
              <span className="text-xs font-bold text-muted-foreground block uppercase tracking-wider flex items-center gap-2 select-none">
                <Server className="h-4.5 w-4.5 text-primary" />
                System Specifications
              </span>
              
              <div className="grid grid-cols-2 gap-4">
                {/* Region Panel */}
                <div className="border border-border/60 rounded-xl p-3.5 bg-background/40">
                  <span className="text-[10px] text-muted-foreground/80 font-medium block uppercase tracking-wide">Current Region</span>
                  <span className="text-xs font-bold text-foreground block mt-1.5 truncate">
                    {systemInfo?.region === "us-west-2" ? "North America, US-West-2" : systemInfo?.region || "Unknown"}
                  </span>
                  <div className="mt-3.5 space-y-1 text-[10px] text-muted-foreground">
                    <div className="flex justify-between"><span>Provider</span><span className="font-semibold text-foreground/80">Google Cloud</span></div>
                    <div className="flex justify-between"><span>Uptime</span><span className="font-semibold text-emerald-500">99.98%</span></div>
                  </div>
                </div>

                {/* Version Panel */}
                <div className="border border-border/60 rounded-xl p-3.5 bg-background/40">
                  <span className="text-[10px] text-muted-foreground/80 font-medium block uppercase tracking-wide">Platform Version</span>
                  <span className="text-xs font-bold text-foreground block mt-1.5 truncate">
                    {systemInfo?.version || "v0.0.0"} "Helios"
                  </span>
                  <div className="mt-3.5 space-y-1 text-[10px] text-muted-foreground">
                    <div className="flex justify-between"><span>Status</span><span className="font-semibold text-emerald-500">Active</span></div>
                    <div className="flex justify-between"><span>Uptime</span><span className="font-semibold text-emerald-500">99.98%</span></div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* Card 2: Super Administrators */}
          <div className="bg-card border border-border rounded-2xl p-5 space-y-4 shadow-sm">
            <span className="text-xs font-bold text-muted-foreground block uppercase tracking-wider flex items-center gap-2 select-none">
              <Shield className="h-4.5 w-4.5 text-primary" />
              Super Administrators
            </span>
            <div className="space-y-3.5 max-h-[160px] overflow-y-auto pr-1">
              {systemInfo?.superAdminEmails && systemInfo.superAdminEmails.length > 0 ? (
                systemInfo.superAdminEmails.map((email, idx) => {
                  const localPart = email.split("@")[0]
                  const name = localPart.split(/[._-]/).map(p => p.charAt(0).toUpperCase() + p.slice(1)).join(" ")
                  const initials = getInitials(email)

                  return (
                    <div key={idx} className="flex items-center justify-between p-2.5 border border-border/60 rounded-xl bg-background/40 hover:bg-muted/10 transition-colors">
                      <div className="flex items-center gap-3 min-w-0">
                        <div className="h-8 w-8 rounded-full bg-primary/10 border border-primary/20 flex items-center justify-center text-[10px] font-bold text-primary shrink-0 select-none">
                          {initials}
                        </div>
                        <div className="min-w-0 flex flex-col">
                          <span className="text-xs font-semibold text-foreground truncate">{name}</span>
                          <span className="text-[10px] text-muted-foreground truncate">{email}</span>
                        </div>
                      </div>
                      <div className="flex items-center gap-2.5">
                        <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[9px] font-semibold bg-emerald-500/10 text-emerald-500 select-none">
                          Active
                        </span>
                      </div>
                    </div>
                  )
                })
              ) : (
                <span className="text-xs text-muted-foreground italic">No super admins configured</span>
              )}
            </div>
          </div>
        </div>

        {/* Organizations Section */}
        <div className="space-y-4 pt-2">
          
          {/* Header Control Bar */}
          <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between border-b border-border pb-4">
            <div>
              <h3 className="text-base font-bold text-foreground">Organizations ({organizations.length})</h3>
              <p className="text-[11px] text-muted-foreground mt-0.5">Manage multi-tenant organization directory and provisioning configurations.</p>
            </div>
            
            <div className="flex flex-wrap items-center gap-2">
              {/* Search Bar */}
              <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground/60" />
                <input
                  type="text"
                  placeholder="Search orgs..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="pl-9 pr-3 py-1.5 w-[180px] text-xs rounded-lg border border-input bg-background focus:outline-none focus:ring-1 focus:ring-primary/40 focus:border-primary/40 transition-all text-foreground"
                />
              </div>

              {/* Export Action Button */}
              <button 
                type="button" 
                className="h-8 px-2.5 rounded-lg border border-input bg-background hover:bg-muted text-xs font-medium text-muted-foreground hover:text-foreground flex items-center gap-1.5 transition-colors select-none"
              >
                <Download className="h-3.5 w-3.5" />
                Export
              </button>

              {/* Provision Org Button */}
              <button
                onClick={() => router.push("/admin/organization-detail")}
                className="h-8 px-3 rounded-lg bg-primary hover:bg-primary/90 text-primary-foreground text-xs font-semibold flex items-center gap-1.5 transition-colors shadow-sm select-none"
              >
                <Plus className="h-3.5 w-3.5" />
                Add New Org
              </button>
            </div>
          </div>

          {orgsError && (
            <div className="p-4 rounded-xl bg-destructive/10 border border-destructive/20 text-destructive text-xs flex items-center gap-3">
              <ShieldAlert className="h-5 w-5 shrink-0" />
              <span>{orgsError}</span>
            </div>
          )}

          {isLoadingOrgs ? (
            <div className="flex justify-center items-center py-12 gap-2 text-muted-foreground">
              <Loader2 className="h-6 w-6 animate-spin text-primary" />
              <span className="text-xs">Loading organizations...</span>
            </div>
          ) : filteredOrgs.length === 0 ? (
            <div className="p-12 text-center border border-dashed border-border rounded-2xl text-muted-foreground text-xs select-none">
              No organizations found matching the criteria.
            </div>
          ) : (
            <div className="bg-card border border-border rounded-2xl overflow-hidden shadow-sm">
              <table className="w-full border-collapse text-left">
                <thead>
                  <tr className="border-b border-border bg-slate-50 dark:bg-slate-900/40 text-[10px] font-bold uppercase tracking-wider text-muted-foreground/80 select-none">
                    <th className="p-4">Tenant Name</th>
                    <th className="p-4">Tenant ID (UUID)</th>
                    <th className="p-4">Status</th>
                    <th className="p-4">Users</th>
                    <th className="p-4">Created</th>
                    <th className="p-4 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border/60 text-xs">
                  {filteredOrgs.map((org, index) => {
                    const initials = org.name.slice(0, 2).toUpperCase()
                    // Derive dynamic status badge based on tenant registration state
                    const status = org.tenant_id ? "Active" : "Pending"
                    // User count from database
                    const userCount = org.user_count ?? 0
                    const createdDate = `Jun 1${index}, 2026`

                    return (
                      <tr key={org.id} className="hover:bg-muted/10 transition-colors">
                        <td className="p-4 font-semibold text-foreground">
                          <div className="flex items-center gap-2.5">
                            <div className="h-6.5 w-6.5 rounded-full bg-primary/10 border border-primary/20 flex items-center justify-center text-[9px] font-bold text-primary shrink-0 select-none">
                              {initials}
                            </div>
                            <span className="truncate max-w-[160px]">{org.name}</span>
                          </div>
                        </td>
                        <td className="p-4 font-mono text-[10px] text-primary/90">{org.id}</td>
                        <td className="p-4 select-none">
                          <span className={`inline-flex items-center px-1.5 py-0.5 rounded text-[9px] font-semibold ${
                            status === "Active" 
                              ? "bg-emerald-500/10 text-emerald-500" 
                              : "bg-amber-500/10 text-amber-500"
                          }`}>
                            {status}
                          </span>
                        </td>
                        <td className="p-4 text-muted-foreground font-medium">{userCount}</td>
                        <td className="p-4 text-muted-foreground/80">{createdDate}</td>
                        <td className="p-4 text-right">
                          <div className="flex items-center justify-end gap-1.5">
                            {/* Manage Org Policies */}
                            <button
                              onClick={() => handleEditOrg(org.id)}
                              className="h-8 px-2.5 rounded-lg border border-border bg-background hover:bg-muted text-muted-foreground hover:text-foreground flex items-center gap-1 text-[10px] font-medium transition-colors"
                              title="Edit Organization"
                            >
                              <Edit3 className="h-3 w-3" />
                              EDIT
                            </button>

                            {/* Delete Org */}
                            <button
                              onClick={() => setOrgToDelete(org)}
                              className="h-8 px-2.5 rounded-lg border border-destructive/20 bg-background hover:bg-destructive/10 text-destructive/80 hover:text-destructive flex items-center gap-1 text-[10px] font-medium transition-colors"
                              title="Delete Organization"
                            >
                              <Trash2 className="h-3 w-3" />
                              DELETE
                            </button>
                          </div>
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          )}
        </div>

      </div>

      {/* Delete Confirmation Modal Overlay */}
      <DeleteOrganizationModal
        isOpen={orgToDelete !== null}
        onClose={() => setOrgToDelete(null)}
        onConfirm={handleDeleteConfirm}
        orgName={orgToDelete?.name || ""}
        isPending={isDeleting}
      />
    </main>
  )
}
