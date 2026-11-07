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
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import { cn } from "@/lib/utils"
import { 
  Shield, 
  UserX, 
  Plus, 
  Loader2,
  AlertCircle,
  Users,
  Pencil,
  X,
  Search
} from "lucide-react"
import { useAppStore } from "@/stores/app"
import { useAuthStore } from "@/stores/auth"
import { api } from "@/lib/api"
import ConfirmDialog from "@/components/shared/ConfirmDialog"
import UserDetailModal from "@/components/shared/UserDetailModal"
import { UnauthorizedScreen } from "@/components/shared/UnauthorizedScreen"

interface Member {
  id: string
  userId?: string
  email: string
  role: string
  roles: string[]
  status: 'active' | 'invited' | 'pending'
  displayName?: string
  lastLogin?: string
}

export default function UsersPage() {
  const queryClient = useQueryClient()
  const { setPageTitle } = useAppStore()
  const { email, isAdmin, isCheckingAuth, selectedOrgId, permissions } = useAuthStore()

  const userPermissions = permissions || []
  const hasPermission = (perm: string) => isAdmin || userPermissions.includes(perm)

  // Tab & search state
  const [activeTab, setActiveTab] = useState<"members" | "roles">("members")
  const [searchQuery, setSearchQuery] = useState("")
  const [isAddModalOpen, setIsAddModalOpen] = useState(false)
  const [memberToRevoke, setMemberToRevoke] = useState<Member | null>(null)
  const [memberToEdit, setMemberToEdit] = useState<Member | null>(null)

  // Sync page title
  useEffect(() => {
    setPageTitle("Users")
  }, [setPageTitle])

  // 1. Query: Fetch organization members
  const {
    data: membersData,
    isLoading: isLoadingMembers,
    error: membersError,
    refetch: refetchMembers,
  } = useQuery({
    queryKey: ["org-members", selectedOrgId, searchQuery],
    queryFn: async () => {
      if (!selectedOrgId) return { users: [] }
      const params = searchQuery ? `?q=${encodeURIComponent(searchQuery)}` : ""
      const res = await api.get(`/api/v1/admin/organizations/${selectedOrgId}/users${params}`)
      if (!res.ok) throw new Error("Failed to load member directory")
      return res.json()
    },
    enabled: !!selectedOrgId && hasPermission("user:list"),
  })
  const members: Member[] = membersData?.users || []

  // Client-side search filtering
  const filteredMembers = members.filter((member: Member) => {
    if (!searchQuery.trim()) return true
    const q = searchQuery.toLowerCase().trim()
    const nameMatch = (member.displayName || "").toLowerCase().includes(q)
    const emailMatch = (member.email || "").toLowerCase().includes(q)
    const idMatch = (member.id || "").toLowerCase().includes(q) || (member.userId || "").toLowerCase().includes(q)
    const statusMatch = (member.status || "").toLowerCase().includes(q)
    const rolesMatch = (member.roles || [member.role || ""]).some((r) => r.toLowerCase().includes(q))
    return nameMatch || emailMatch || idMatch || statusMatch || rolesMatch
  })

  // 2. Query: Fetch roles
  const {
    data: rolesData,
    isLoading: isLoadingRoles,
    refetch: refetchRoles,
  } = useQuery({
    queryKey: ["org-roles", selectedOrgId],
    queryFn: async () => {
      if (!selectedOrgId) return { roles: [] }
      const res = await api.get(`/api/v1/admin/organizations/${selectedOrgId}/roles`)
      if (!res.ok) throw new Error("Failed to load roles directory")
      return res.json()
    },
    enabled: !!selectedOrgId && (hasPermission("role:list") || hasPermission("user:list")),
  })
  const roles = rolesData?.roles || []

  // Mutation: Update Role
  const updateRoleMutation = useMutation({
    mutationFn: async ({ userId, role }: { userId: string; role: string }) => {
      const res = await api.put(`/api/v1/admin/organizations/${selectedOrgId}/users/${encodeURIComponent(userId)}/role`, { role })
      if (!res.ok) {
        let errData
        try { errData = await res.json() } catch (_) {}
        throw new Error(errData?.message || "Failed to update role")
      }
      return { userId, role }
    },
    onSuccess: (data) => {
      toast.success(`Updated role for user to ${data.role}`)
      refetchMembers()
    },
    onError: (err: any) => {
      toast.error(err.message || "Failed to update role")
    }
  })

  // Mutation: Remove User
  const removeUserMutation = useMutation({
    mutationFn: async (userId: string) => {
      const res = await api.delete(`/api/v1/admin/organizations/${selectedOrgId}/users/${encodeURIComponent(userId)}`)
      if (!res.ok) {
        let errData
        try { errData = await res.json() } catch (_) {}
        throw new Error(errData?.message || "Failed to remove user")
      }
      return userId
    },
    onSuccess: () => {
      toast.success("User removed from organization")
      refetchMembers()
      setMemberToRevoke(null)
    },
    onError: (err: any) => {
      toast.error(err.message || "Failed to remove user")
    }
  })

  if (isCheckingAuth) {
    return (
      <div className="flex-1 flex h-[400px] w-full flex-col items-center justify-center gap-3 text-muted-foreground select-none">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
        <span className="text-sm font-medium">Verifying access...</span>
      </div>
    )
  }

  if (!hasPermission("user:list")) {
    return (
      <UnauthorizedScreen
        title="Access Restricted"
        message="You do not have permission to view the organization member directory or access IAM settings."
        requiredPermission="user:list"
        fullPage={false}
      />
    )
  }

  return (
    <div className="flex flex-col h-full w-full p-8 space-y-6">
      
      {/* Header and Title */}
      <div className="space-y-1 select-none">
        <h1 className="text-2xl font-bold text-foreground flex items-center gap-3">
          <Users className="h-7 w-7 text-primary" />
          User Management
        </h1>
        <p className="text-xs text-muted-foreground leading-relaxed">
          Manage your organization's members, access levels, roles, and granular permissions.
        </p>
      </div>

      {/* Tabs Menu Navigation */}
      <div className="flex border-b border-border select-none">
        <button
          onClick={() => setActiveTab("members")}
          className={cn(
            "px-6 py-3 text-xs font-bold border-b-2 transition-all duration-200",
            activeTab === "members"
              ? "border-primary text-primary font-bold"
              : "border-transparent text-muted-foreground hover:text-foreground"
          )}
        >
          Member Directory
        </button>
        <button
          onClick={() => setActiveTab("roles")}
          className={cn(
            "px-6 py-3 text-xs font-bold border-b-2 transition-all duration-200",
            activeTab === "roles"
              ? "border-primary text-primary font-bold"
              : "border-transparent text-muted-foreground hover:text-foreground"
          )}
        >
          Roles & Permissions
        </button>
      </div>

      {/* Main Content Area */}
      <div className="flex-1 w-full max-w-[1200px]">
        {activeTab === "members" ? (
          /* ================= MEMBER DIRECTORY ================= */
          <div className="space-y-6">
            <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-4 select-none">
              <div>
                <h2 className="text-base font-bold text-foreground">Members List</h2>
                <p className="text-[11px] text-muted-foreground">
                  Active organization users and their corresponding role mappings.
                </p>
              </div>
              <div className="flex items-center gap-3">
                {/* Search Input */}
                <div className="relative flex-1 sm:w-72">
                  <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                  <input
                    type="text"
                    aria-label="Search users"
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    placeholder="Search users by name, email, or role..."
                    className="w-full h-9 pl-9 pr-8 rounded-xl border border-input bg-background text-xs placeholder:text-muted-foreground/60 focus:outline-none focus:ring-1 focus:ring-primary focus:border-primary transition-all"
                  />
                  {searchQuery && (
                    <button
                      type="button"
                      onClick={() => setSearchQuery("")}
                      className="absolute right-2.5 top-1/2 -translate-y-1/2 p-0.5 rounded-full hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
                      title="Clear search"
                    >
                      <X className="h-3.5 w-3.5" />
                    </button>
                  )}
                </div>

                {hasPermission("user:create") && (
                  <button
                    type="button"
                    onClick={() => setIsAddModalOpen(true)}
                    className="inline-flex h-9 shrink-0 items-center justify-center gap-1.5 rounded-xl bg-primary px-4 text-xs font-semibold text-primary-foreground shadow hover:opacity-90 transition-colors"
                  >
                    <Plus className="h-4 w-4" />
                    Add Member
                  </button>
                )}
              </div>
            </div>

            {searchQuery && (
              <div className="flex items-center justify-between text-xs text-muted-foreground bg-muted/20 px-3 py-1.5 rounded-lg border border-border/50 select-none">
                <span>
                  Showing <strong>{filteredMembers.length}</strong> of {members.length} members matching &ldquo;{searchQuery}&rdquo;
                </span>
                <button
                  type="button"
                  onClick={() => setSearchQuery("")}
                  className="text-primary hover:underline text-[11px] font-medium"
                >
                  Clear search
                </button>
              </div>
            )}

            {isLoadingMembers ? (
              <div className="flex h-[150px] items-center justify-center gap-2 text-muted-foreground">
                <Loader2 className="h-5 w-5 animate-spin text-primary" />
                <span className="text-xs">Loading members...</span>
              </div>
            ) : membersError ? (
              <div className="p-4 bg-destructive/10 border border-destructive/20 rounded-xl text-center text-xs text-destructive flex items-center justify-center gap-2">
                <AlertCircle className="h-4 w-4" />
                Failed to load member directory: {membersError instanceof Error ? membersError.message : "Unknown error"}
              </div>
            ) : members.length === 0 && !searchQuery.trim() ? (
              <div className="text-center py-8 text-xs text-muted-foreground border border-dashed border-border rounded-xl select-none">
                No members found in this organization.
              </div>
            ) : filteredMembers.length === 0 ? (
              <div className="text-center py-12 px-4 border border-dashed border-border rounded-xl select-none space-y-3">
                <Search className="h-8 w-8 text-muted-foreground/50 mx-auto" />
                <p className="text-xs text-muted-foreground">
                  No matching users found for &ldquo;<span className="text-foreground font-medium">{searchQuery}</span>&rdquo;
                </p>
                <button
                  type="button"
                  onClick={() => setSearchQuery("")}
                  className="inline-flex items-center gap-1 text-xs text-primary font-medium hover:underline"
                >
                  Clear search query
                </button>
              </div>
            ) : (
              <div className="border border-border rounded-xl overflow-hidden bg-card shadow-sm">
                <table className="w-full text-left border-collapse text-xs">
                  <thead>
                    <tr className="bg-slate-50 dark:bg-slate-900/50 border-b border-border text-muted-foreground font-medium select-none">
                      <th className="p-4">Name / Email</th>
                      <th className="p-4">Role</th>
                      <th className="p-4">Status</th>
                      <th className="p-4">Last Login</th>
                      <th className="p-4 text-right"></th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-border">
                    {filteredMembers.map((member: Member) => {
                      const isSelf = member.email === email
                      return (
                        <tr key={member.id} className="hover:bg-muted/10 transition-colors">
                          <td className="p-4">
                            <div className="font-medium text-foreground">
                              {member.displayName || member.email.split('@')[0]}
                            </div>
                            <div className="text-[10px] text-muted-foreground mt-0.5">{member.email}</div>
                            {member.userId && (
                              <div className="text-[9px] text-muted-foreground font-mono mt-0.5">{member.userId}</div>
                            )}
                          </td>
                          <td className="p-4">
                            <div className="flex flex-wrap gap-1.5 items-center">
                              {(member.roles && member.roles.length > 0 ? member.roles : [member.role || "viewer"]).map((r) => {
                                const isSpecial = r === "admin" || r === "owner"
                                const isEditor = r === "editor" || r === "developer"
                                return (
                                  <div
                                    key={r}
                                    className={cn(
                                      "inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-[10px] font-semibold tracking-wide capitalize shadow-xs border transition-all select-none",
                                      isSpecial
                                        ? "bg-purple-100 text-purple-800 border-purple-200 dark:bg-purple-950/60 dark:text-purple-300 dark:border-purple-800"
                                        : isEditor
                                          ? "bg-indigo-100 text-indigo-800 border-indigo-200 dark:bg-indigo-950/60 dark:text-indigo-300 dark:border-indigo-800"
                                          : "bg-slate-100 text-slate-700 border-slate-200 dark:bg-slate-800/80 dark:text-slate-300 dark:border-slate-700"
                                    )}
                                  >
                                    <Shield className="h-3 w-3 shrink-0 text-current opacity-80" />
                                    <span>{r}</span>
                                  </div>
                                )
                              })}
                            </div>
                          </td>
                          <td className="p-4">
                            <span className={cn(
                              "inline-flex items-center px-2 py-0.5 rounded-full text-[10px] font-semibold capitalize",
                              member.status === "active"
                                ? "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400"
                                : member.status === "invited"
                                  ? "bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400"
                                  : "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400" // pending
                            )}>
                              {member.status}
                            </span>
                          </td>
                          <td className="p-4 text-muted-foreground text-[11px]">
                            {member.lastLogin ? (
                              <span>
                                {new Date(member.lastLogin).toLocaleDateString(undefined, {
                                  month: 'short',
                                  day: 'numeric',
                                  year: 'numeric',
                                  hour: '2-digit',
                                  minute: '2-digit',
                                })}
                              </span>
                            ) : (
                              <span className="text-muted-foreground/60 italic">Never</span>
                            )}
                          </td>
                          <td className="p-4 text-right">
                            <div className="flex items-center justify-end">
                              {/* Edit Action */}
                              <button
                                type="button"
                                onClick={() => setMemberToEdit(member)}
                                className="inline-flex h-8 items-center gap-1 rounded-lg border border-border bg-background px-2.5 text-[11px] font-medium text-foreground hover:bg-muted/50 transition-colors"
                                title="Edit Member"
                              >
                                <Pencil className="h-3.5 w-3.5 text-muted-foreground" />
                                <span>Edit</span>
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
        ) : (
          /* ================= ROLES & PERMISSIONS ================= */
          <div className="space-y-6">
            <div className="flex items-center justify-between select-none">
              <div>
                <h2 className="text-base font-bold text-foreground">Roles & Permissions Reference</h2>
                <p className="text-[11px] text-muted-foreground">
                  View organization predefined roles and their associated system permissions.
                </p>
              </div>
            </div>

            {/* Roles List */}
            <div className="border border-border rounded-xl overflow-hidden bg-card shadow-sm">
              <table className="w-full text-left border-collapse text-xs">
                <thead>
                  <tr className="bg-slate-50 dark:bg-slate-900/50 border-b border-border text-muted-foreground font-medium select-none">
                    <th className="p-4 w-1/3">Role</th>
                    <th className="p-4 w-2/3">Permissions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {roles.map((role: any) => (
                    <tr key={role.id} className="hover:bg-muted/10 transition-colors">
                      <td className="p-4 align-top">
                        <div className="font-medium text-foreground flex items-center gap-2">
                          <Shield className="h-4 w-4 text-primary shrink-0" />
                          {role.name}
                        </div>
                        {role.description && (
                          <p className="text-[10px] text-muted-foreground mt-1 leading-relaxed">
                            {role.description}
                          </p>
                        )}
                      </td>
                      <td className="p-4 align-top">
                        <div className="flex flex-wrap gap-1.5">
                          {role.permissions.map((p: any) => (
                            <span key={p} className="px-2 py-0.5 rounded-lg bg-muted text-[9px] font-semibold font-mono text-muted-foreground">
                              {p}
                            </span>
                          ))}
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </div>      {selectedOrgId && (
        <>
          <UserDetailModal
            isOpen={isAddModalOpen}
            onClose={() => setIsAddModalOpen(false)}
            orgId={selectedOrgId}
            onSuccess={() => {
              refetchMembers()
            }}
          />
          <UserDetailModal
            isOpen={!!memberToEdit}
            onClose={() => setMemberToEdit(null)}
            orgId={selectedOrgId}
            member={memberToEdit}
            onSuccess={() => {
              refetchMembers()
            }}
            onDelete={(m) => {
              setMemberToRevoke(m)
            }}
          />
        </>
      )}
      <ConfirmDialog
        isOpen={!!memberToRevoke}
        onClose={() => setMemberToRevoke(null)}
        onConfirm={() => {
          if (memberToRevoke) {
            removeUserMutation.mutate(memberToRevoke.id)
          }
        }}
        title="Revoke Access"
        message={`Are you sure you want to revoke access for ${memberToRevoke?.email}? This will permanently remove them from the organization and delete their profile.`}
        confirmText="Revoke Access"
        isDestructive
        isLoading={removeUserMutation.isPending}
      />
    </div>
  )
}
