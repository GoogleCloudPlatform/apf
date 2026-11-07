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
import { useSearchParams, useRouter } from "next/navigation"
import { useAuthStore } from "@/stores/auth"
import { useAppStore } from "@/stores/app"
import { Building2, ArrowLeft, Trash2, Save, Loader2, AlertCircle, CheckCircle2, ShieldAlert } from "lucide-react"
import { toast } from "sonner"

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
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm p-4">
      <div className="w-full max-w-md bg-card border border-border rounded-2xl p-6 shadow-lg space-y-6 animate-in fade-in zoom-in-95 duration-150">
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

function OrganizationDetailContent() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const orgId = searchParams.get("id")
  const isNew = !orgId

  const { setPageTitle } = useAppStore()
  const { token, isSuperAdmin, isCheckingAuth } = useAuthStore()

  // Form states
  const [displayName, setDisplayName] = useState("")
  const [name, setName] = useState("")
  const [ownerEmail, setOwnerEmail] = useState("")
  const [tenantId, setTenantId] = useState("")

  // Loading & pending states
  const [isLoading, setIsLoading] = useState(false)
  const [isSaving, setIsSaving] = useState(false)
  const [isDeleting, setIsDeleting] = useState(false)
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false)

  // Status banners
  const [error, setError] = useState<string | null>(null)
  const [successData, setSuccessData] = useState<{ orgId: string; ownerUid?: string } | null>(null)

  // Sync page title
  useEffect(() => {
    setPageTitle(isNew ? "Admin: Provision Organization" : "Admin: Organization Details")
  }, [isNew, setPageTitle])

  // Fetch organization details if in Edit mode
  useEffect(() => {
    const fetchOrgDetails = async () => {
      if (isNew || !token || !isSuperAdmin) return
      setIsLoading(true)
      setError(null)
      try {
        const response = await fetch(`/api/v1/admin/organizations/${orgId}`, {
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${token}`,
          },
        })
        if (!response.ok) {
          throw new Error(`Failed to load organization: ${response.statusText}`)
        }
        const data = await response.json()
        setName(data.name || "")
        setTenantId(data.tenant_id || "")
      } catch (err: any) {
        console.error("Error fetching org details:", err)
        setError(err.message || "Failed to load organization specifications")
        toast.error("Failed to load organization details")
      } finally {
        setIsLoading(false)
      }
    }

    if (!isCheckingAuth && isSuperAdmin && !isNew) {
      fetchOrgDetails()
    }
  }, [orgId, isNew, token, isSuperAdmin, isCheckingAuth])

  if (isCheckingAuth || isLoading) {
    return (
      <div className="flex-1 flex h-[400px] w-full flex-col items-center justify-center gap-3 text-muted-foreground select-none">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
        <span className="text-sm font-medium">
          {isLoading ? "Loading organization details..." : "Verifying access..."}
        </span>
      </div>
    )
  }

  // Guard: Only super admins can access
  if (!isSuperAdmin) {
    return (
      <div className="flex-1 flex flex-col items-center justify-center p-8 min-h-[calc(100vh-4rem)] select-none">
        <div className="w-full max-w-md bg-destructive/10 border border-destructive/20 rounded-2xl p-8 text-center space-y-6">
          <div className="flex justify-center text-destructive">
            <ShieldAlert className="h-16 w-16 animate-bounce" />
          </div>
          <div className="space-y-2">
            <h3 className="text-xl font-bold text-foreground">
              Access Restricted
            </h3>
            <p className="text-sm text-muted-foreground leading-relaxed">
              This screen is reserved exclusively for Super Administrators.
            </p>
          </div>
        </div>
      </div>
    )
  }

  const handleCancel = () => {
    router.push("/admin/overview")
  }

  const validateForm = () => {
    const trimmedName = name.trim()
    if (!trimmedName) {
      setError("Organization Name is required")
      return false
    }

    // Cloud Identity Platform Tenant ID constraints:
    // Alphanumeric, underscores, hyphens, 4-20 characters
    const tenantNameRegex = /^[a-zA-Z0-9_-]{4,20}$/
    if (!tenantNameRegex.test(trimmedName)) {
      setError("Organization Name must be 4-20 characters and contain only letters, numbers, underscores, or hyphens (no spaces)")
      return false
    }

    if (isNew) {
      const trimmedEmail = ownerEmail.trim()
      if (!trimmedEmail) {
        setError("Owner Email is required")
        return false
      }
      const emailRegex = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/
      if (!emailRegex.test(trimmedEmail)) {
        setError("A valid Owner Email address is required")
        return false
      }
    }
    return true
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    setSuccessData(null)

    if (!validateForm()) return

    setIsSaving(true)

    try {
      const url = isNew ? "/api/v1/organizations" : `/api/v1/admin/organizations/${orgId}`
      const method = isNew ? "POST" : "PUT"
      
      const payload = isNew 
        ? { name: name.trim(), ownerEmail: ownerEmail.trim() }
        : { name: name.trim() }

      const response = await fetch(url, {
        method,
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify(payload),
      })

      let data: any = {}
      const contentType = response.headers.get("content-type")
      if (contentType && contentType.includes("application/json")) {
        data = await response.json()
      } else {
        const text = await response.text()
        data = { message: text || `Server returned status ${response.status}` }
      }

      if (response.ok) {
        if (isNew) {
          const newOrgId = data.data?.orgId || data.id || ""
          const ownerUid = data.data?.ownerUid || data.ownerUid || ""
          setSuccessData({
            orgId: newOrgId,
            ownerUid,
          })
          toast.success("Organization provisioned successfully!")
          setDisplayName("")
          setName("")
          setOwnerEmail("")
        } else {
          toast.success("Organization updated successfully")
          router.push("/admin/overview")
        }
      } else {
        setError(data.message || `Failed to ${isNew ? "provision" : "update"} organization`)
      }
    } catch (err) {
      console.error(`Error saving organization:`, err)
      setError("Network error: Failed to connect to server")
    } finally {
      setIsSaving(false)
    }
  }

  const handleDelete = async () => {
    if (isNew) return
    setIsDeleting(true)
    setError(null)
    try {
      const response = await fetch(`/api/v1/admin/organizations/${orgId}`, {
        method: "DELETE",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
      })

      let data: any = {}
      const contentType = response.headers.get("content-type")
      if (contentType && contentType.includes("application/json")) {
        data = await response.json()
      } else {
        const text = await response.text()
        data = { message: text }
      }

      if (response.ok) {
        toast.success("Organization deleted successfully")
        setIsDeleteModalOpen(false)
        router.push("/admin/overview")
      } else {
        setError(data.message || "Failed to delete organization")
        setIsDeleteModalOpen(false)
      }
    } catch (err) {
      console.error("Error deleting organization:", err)
      setError("Network error: Failed to connect to server")
      setIsDeleteModalOpen(false)
    } finally {
      setIsDeleting(false)
    }
  }

  return (
    <main className="flex-1 overflow-y-auto p-8">
      <div className="w-full max-w-[800px] space-y-6 pb-12">
        
        {/* Header (Similar to Project Details) */}
        <div className="flex items-center justify-between border-b border-border pb-4 select-none">
          <div className="flex items-center gap-3">
            <button
              onClick={handleCancel}
              className="flex h-9 w-9 items-center justify-center rounded-lg text-muted-foreground hover:bg-muted hover:text-foreground border border-input bg-background transition-all"
              title="Back to Overview"
            >
              <ArrowLeft className="h-4 w-4" />
            </button>
            
            <div className="flex items-center gap-2.5">
              <div className="h-10 w-10 rounded-xl bg-primary/10 text-primary flex items-center justify-center">
                <Building2 className="h-5 w-5" />
              </div>
              <div>
                <h2 className="text-sm font-bold text-foreground">
                  {isNew ? "New Organization" : name}
                </h2>
                {!isNew && (
                  <span className="text-[10px] font-mono text-muted-foreground">ID: {orgId}</span>
                )}
              </div>
            </div>
          </div>

          <div className="flex gap-2">
            <button
              type="button"
              onClick={handleCancel}
              className="inline-flex h-9 items-center justify-center rounded-lg border border-input bg-background px-4 py-2 text-xs font-semibold text-foreground hover:bg-muted transition-all"
            >
              Cancel
            </button>

            {!isNew && (
              <button
                type="button"
                onClick={() => setIsDeleteModalOpen(true)}
                className="inline-flex h-9 items-center justify-center gap-1.5 rounded-lg border border-destructive bg-transparent px-4 py-2 text-xs font-semibold text-destructive hover:bg-destructive/10 transition-all"
              >
                <Trash2 className="h-3.5 w-3.5" />
                Delete
              </button>
            )}

            <button
              type="button"
              onClick={handleSubmit}
              disabled={isSaving}
              className="inline-flex h-9 items-center justify-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-xs font-semibold text-primary-foreground shadow transition-all hover:opacity-90 disabled:opacity-50"
            >
              {isSaving ? (
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
              ) : (
                <Save className="h-3.5 w-3.5" />
              )}
              {isNew ? "Provision" : "Save"}
            </button>
          </div>
        </div>

        {/* Success Banner */}
        {successData && (
          <div className="p-4 rounded-xl bg-emerald-50 dark:bg-emerald-950/30 border border-emerald-200 dark:border-emerald-800/30 text-emerald-800 dark:text-emerald-300 text-xs space-y-2">
            <div className="flex items-center gap-2 font-semibold">
              <CheckCircle2 className="h-4 w-4 text-emerald-600 dark:text-emerald-400" />
              <span>Organization provisioned successfully!</span>
            </div>
            <div className="pl-6 space-y-1 font-mono text-[10px]">
              <div><span className="font-semibold select-none text-muted-foreground">Org ID:</span> {successData.orgId}</div>
              {successData.ownerUid && (
                <div><span className="font-semibold select-none text-muted-foreground">Owner UID:</span> {successData.ownerUid}</div>
              )}
            </div>
          </div>
        )}

        {/* Error Banner */}
        {error && (
          <div className="p-4 rounded-xl bg-destructive/10 border border-destructive/20 text-destructive text-xs flex items-start gap-2">
            <AlertCircle className="h-4 w-4 mt-0.5 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        {/* Form Body (Card Layout) */}
        <form onSubmit={handleSubmit} className="space-y-6">
          
          <div className="bg-card border border-border rounded-2xl p-6 space-y-4 shadow-sm">
            <div className="flex items-center gap-2 border-b border-border pb-2 select-none">
              <Building2 className="h-4 w-4 text-primary" />
              <h3 className="text-xs font-bold text-foreground">General Settings</h3>
            </div>

            <div className="space-y-4">
              {/* Display Name (Create Mode Only) */}
              {isNew && (
                <div className="space-y-1">
                  <label className="text-xs font-semibold text-muted-foreground uppercase" htmlFor="display-name">
                    Display Name
                  </label>
                  <input
                    id="display-name"
                    type="text"
                    placeholder="e.g. Acme Corp"
                    disabled={isSaving}
                    value={displayName}
                    onChange={(e) => {
                      const val = e.target.value
                      setDisplayName(val)
                      setName(val.toLowerCase().replace(/[^a-z0-9_]/g, "-"))
                    }}
                    className="flex h-11 w-full rounded-md border border-input bg-background px-4 text-sm focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all text-foreground disabled:opacity-60"
                  />
                  <p className="text-[10px] text-muted-foreground">
                    Display name for the organization.
                  </p>
                </div>
              )}

              {/* Organization Name */}
              <div className="space-y-1">
                <label className="text-xs font-semibold text-muted-foreground uppercase" htmlFor="org-name">
                  Organization Name
                </label>
                <input
                  id="org-name"
                  type="text"
                  placeholder="e.g. acme-corp"
                  disabled={isSaving}
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="flex h-11 w-full rounded-md border border-input bg-background px-4 text-sm focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all text-foreground disabled:opacity-60"
                />
                <p className="text-[10px] text-muted-foreground">
                  Must be 4-20 characters, containing only letters, numbers, underscores, or hyphens (no spaces).
                </p>
              </div>

              {/* Owner Email (Create Mode Only) */}
              {isNew && (
                <div className="space-y-1">
                  <label className="text-xs font-semibold text-muted-foreground uppercase" htmlFor="owner-email">
                    Owner Email Address
                  </label>
                  <input
                    id="owner-email"
                    type="email"
                    placeholder="e.g. admin@acme.com"
                    disabled={isSaving}
                    value={ownerEmail}
                    onChange={(e) => setOwnerEmail(e.target.value)}
                    className="flex h-11 w-full rounded-md border border-input bg-background px-4 text-sm focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all text-foreground disabled:opacity-60"
                  />
                  <p className="text-[10px] text-muted-foreground">
                    This user will be created as the initial Owner of the organization.
                  </p>
                </div>
              )}

              {/* Tenant ID (Edit Mode Only - Read Only) */}
              {!isNew && (
                <div className="space-y-1">
                  <label className="text-xs font-semibold text-muted-foreground uppercase">
                    Tenant ID
                  </label>
                  <code className="flex h-11 w-full items-center rounded-md border border-input bg-muted px-4 text-xs text-muted-foreground font-mono select-all">
                    {tenantId || "none"}
                  </code>
                  <p className="text-[10px] text-muted-foreground">
                    The unique system tenant identifier. This value is immutable.
                  </p>
                </div>
              )}
            </div>
          </div>

          {!isNew && (
            <div className="bg-card border border-destructive/20 rounded-2xl p-6 space-y-4 shadow-sm">
              <div className="flex items-center gap-2 border-b border-border pb-2 select-none text-destructive">
                <Trash2 className="h-4 w-4" />
                <h3 className="text-xs font-bold">Danger Zone</h3>
              </div>
              <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                <div className="space-y-1">
                  <p className="text-xs font-semibold text-foreground">Delete this organization</p>
                  <p className="text-[10px] text-muted-foreground leading-relaxed">
                    Permanently delete this organization, its Firebase tenant, and all associated data. 
                    This action is irreversible.
                  </p>
                </div>
                <button
                  type="button"
                  onClick={() => setIsDeleteModalOpen(true)}
                  className="px-4 py-2 bg-destructive hover:bg-destructive/90 text-destructive-foreground rounded-lg text-xs font-semibold shadow-sm transition-all shrink-0"
                >
                  Delete Organization
                </button>
              </div>
            </div>
          )}

        </form>

        <DeleteOrganizationModal
          isOpen={isDeleteModalOpen}
          onClose={() => setIsDeleteModalOpen(false)}
          onConfirm={handleDelete}
          orgName={name}
          isPending={isDeleting}
        />

      </div>
    </main>
  )
}

export default function OrganizationDetailPage() {
  return (
    <Suspense fallback={
      <div className="flex-1 flex h-[400px] w-full flex-col items-center justify-center gap-3 text-muted-foreground select-none">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
        <span className="text-sm font-medium">Loading Organization Details...</span>
      </div>
    }>
      <OrganizationDetailContent />
    </Suspense>
  )
}
