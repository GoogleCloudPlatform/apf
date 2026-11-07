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

import React, { useState, useEffect } from "react"
import { createPortal } from "react-dom"
import { useMutation, useQuery } from "@tanstack/react-query"
import { toast } from "sonner"
import { 
  X, 
  Shield, 
  Loader2, 
  Mail, 
  User, 
  Trash2
} from "lucide-react"
import { api } from "@/lib/api"
import { cn } from "@/lib/utils"

export interface Member {
  id: string
  userId?: string
  email: string
  role: string
  roles: string[]
  status: 'active' | 'invited' | 'pending'
  displayName?: string
  lastLogin?: string
}

export interface UserDetailModalProps {
  isOpen: boolean
  onClose: () => void
  orgId: string
  member?: Member | null
  onSuccess?: () => void
  onDelete?: (member: Member) => void
}

const generateRandomPassword = () => {
  const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
  let result = ""
  for (let i = 0; i < 10; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length))
  }
  return result
}

export function UserDetailModal({
  isOpen,
  onClose,
  orgId,
  member = null,
  onSuccess,
  onDelete,
}: UserDetailModalProps) {
  const isEditMode = !!member

  const [email, setEmail] = useState("")
  const [displayName, setDisplayName] = useState("")
  const [selectedRoles, setSelectedRoles] = useState<string[]>(["viewer"])
  const [mounted, setMounted] = useState(false)

  useEffect(() => {
    setMounted(true)
  }, [])

  // Populate state based on mode and open status
  useEffect(() => {
    if (isOpen) {
      if (member) {
        setEmail(member.email || "")
        setDisplayName(member.displayName || "")
        setSelectedRoles(member.roles && member.roles.length > 0 ? member.roles : [member.role || "viewer"])
      } else {
        setEmail("")
        setDisplayName("")
        setSelectedRoles(["viewer"])
      }
    }
  }, [isOpen, member])

  const { data: rolesData, isLoading: isLoadingRoles } = useQuery({
    queryKey: ["org-roles", orgId],
    queryFn: async () => {
      const res = await api.get(`/api/v1/admin/organizations/${orgId}/roles`)
      if (!res.ok) throw new Error("Failed to load roles")
      return res.json()
    },
    enabled: isOpen && !!orgId,
  })
  const roles = rolesData?.roles || []

  // Mutation: Add Member
  const addMemberMutation = useMutation({
    mutationFn: async (payload: { email: string; password: string; displayName?: string; roles: string[] }) => {
      const res = await api.post(`/api/v1/admin/organizations/${orgId}/users`, payload)
      if (!res.ok) {
        let errData
        try { errData = await res.json() } catch (_) {}
        throw new Error(errData?.message || "Failed to add member")
      }
      return res.json()
    },
    onSuccess: () => {
      toast.success(`Member ${email} successfully added to organization!`)
      setEmail("")
      setDisplayName("")
      setSelectedRoles(["viewer"])
      if (onSuccess) onSuccess()
      onClose()
    },
    onError: (err: any) => {
      toast.error(err.message || "Failed to add member. Please try again.")
    },
  })

  // Mutation: Update Member Role
  const updateMemberMutation = useMutation({
    mutationFn: async (payload: { userId: string; role: string }) => {
      const res = await api.put(`/api/v1/admin/organizations/${orgId}/users/${encodeURIComponent(payload.userId)}/role`, { role: payload.role })
      if (!res.ok) {
        let errData
        try { errData = await res.json() } catch (_) {}
        throw new Error(errData?.message || "Failed to update member role")
      }
      return res.json()
    },
    onSuccess: () => {
      toast.success(`Updated role for ${email}`)
      if (onSuccess) onSuccess()
      onClose()
    },
    onError: (err: any) => {
      toast.error(err.message || "Failed to update member role")
    },
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (selectedRoles.length === 0) {
      toast.error("Please select at least one role.")
      return
    }

    if (isEditMode && member) {
      updateMemberMutation.mutate({
        userId: member.id,
        role: selectedRoles[0],
      })
    } else {
      if (!email) {
        toast.error("Please enter a valid email address.")
        return
      }
      addMemberMutation.mutate({ 
        email, 
        password: generateRandomPassword(), 
        displayName: displayName || undefined, 
        roles: selectedRoles 
      })
    }
  }

  if (!isOpen || !mounted) return null

  const isPending = addMemberMutation.isPending || updateMemberMutation.isPending

  const modalContent = (
    <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/60 backdrop-blur-sm animate-in fade-in-0 duration-200 select-none p-4">
      <div className="relative w-full max-w-md border border-border bg-card p-6 shadow-2xl rounded-2xl animate-in zoom-in-95 duration-200">
        
        {/* Close Button */}
        <button
          onClick={onClose}
          className="absolute right-4 top-4 rounded-lg p-1.5 text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
          aria-label="Close Modal"
        >
          <X className="h-4 w-4" />
        </button>

        {/* Header */}
        <div className="space-y-1.5 mb-6">
          <h2 className="text-lg font-bold text-foreground">
            {isEditMode ? "Edit Member" : "Add New Member"}
          </h2>
          <p className="text-xs text-muted-foreground">
            {isEditMode 
              ? "Update organization roles and view access details for this member."
              : "Directly provision a new active user account and assign their organization roles."
            }
          </p>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} className="space-y-4">
          
          {/* Email */}
          <div className="space-y-1.5">
            <label className="text-xs font-semibold text-foreground/80 block" htmlFor="member-email">
              Email Address
            </label>
            <div className="relative">
              <div className="absolute left-3.5 top-1/2 -translate-y-1/2 text-muted-foreground select-none pointer-events-none">
                <Mail className="h-4.5 w-4.5" />
              </div>
              <input
                id="member-email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="developer@example.com"
                required
                disabled={isEditMode}
                className="w-full h-12 pl-11 pr-4 rounded-xl border border-input bg-background text-sm transition-all focus:outline-none focus:ring-2 focus:ring-primary/25 focus:border-primary placeholder:text-muted-foreground/60 disabled:opacity-60 disabled:cursor-not-allowed"
              />
            </div>
          </div>

          {/* Display Name */}
          <div className="space-y-1.5">
            <label className="text-xs font-semibold text-foreground/80 block" htmlFor="member-name">
              Display Name <span className="text-muted-foreground/60 font-normal">(Optional)</span>
            </label>
            <div className="relative">
              <div className="absolute left-3.5 top-1/2 -translate-y-1/2 text-muted-foreground select-none pointer-events-none">
                <User className="h-4.5 w-4.5" />
              </div>
              <input
                id="member-name"
                type="text"
                value={displayName}
                onChange={(e) => setDisplayName(e.target.value)}
                placeholder="Anakin Skywalker"
                disabled={isEditMode}
                className="w-full h-12 pl-11 pr-4 rounded-xl border border-input bg-background text-sm transition-all focus:outline-none focus:ring-2 focus:ring-primary/25 focus:border-primary placeholder:text-muted-foreground/60 disabled:opacity-60 disabled:cursor-not-allowed"
              />
            </div>
          </div>

          {/* Status & Last Login info (in Edit mode) */}
          {isEditMode && member && (
            <div className="grid grid-cols-2 gap-3 p-3 bg-muted/30 border border-border rounded-xl text-xs">
              <div>
                <span className="text-[10px] text-muted-foreground font-medium block">Status</span>
                <span className={cn(
                  "inline-flex items-center px-2 py-0.5 rounded-full text-[10px] font-semibold capitalize mt-0.5",
                  member.status === "active"
                    ? "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400"
                    : member.status === "invited"
                      ? "bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400"
                      : "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400"
                )}>
                  {member.status}
                </span>
              </div>
              <div>
                <span className="text-[10px] text-muted-foreground font-medium block">Last Login</span>
                <span className="text-foreground font-medium text-[11px] block mt-0.5">
                  {member.lastLogin ? new Date(member.lastLogin).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' }) : "Never"}
                </span>
              </div>
            </div>
          )}

          {/* Roles Floating Divs */}
          <div className="space-y-1.5">
            <span className="text-xs font-semibold text-foreground/80 block select-none">
              Organization Roles
            </span>
            <div className="flex flex-wrap gap-2 pt-1">
              {isLoadingRoles ? (
                <div className="flex items-center justify-center py-2 text-xs text-muted-foreground gap-2">
                  <Loader2 className="h-4 w-4 animate-spin text-primary" />
                  <span>Loading roles...</span>
                </div>
              ) : roles.length === 0 ? (
                <div className="text-xs text-muted-foreground py-1">No roles available.</div>
              ) : (
                roles.map((r: any) => {
                  const isSelected = selectedRoles.includes(r.id)
                  return (
                    <button
                      key={r.id}
                      type="button"
                      onClick={() => {
                        if (isSelected) {
                          setSelectedRoles(selectedRoles.filter((id) => id !== r.id))
                        } else {
                          setSelectedRoles([...selectedRoles, r.id])
                        }
                      }}
                      className={cn(
                        "inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs transition-all select-none cursor-pointer",
                        isSelected
                          ? "bg-purple-100 text-purple-800 border border-purple-300 dark:bg-purple-950/60 dark:text-purple-300 dark:border-purple-700/60 shadow-sm font-semibold"
                          : "bg-muted/40 text-muted-foreground border border-border hover:border-muted-foreground/30 hover:text-foreground hover:bg-muted/70 font-medium"
                      )}
                    >
                      {isSelected ? (
                        <Shield className="h-3.5 w-3.5 text-purple-600 dark:text-purple-400 shrink-0" />
                      ) : (
                        <div className="h-2 w-2 rounded-full bg-muted-foreground/30 shrink-0" />
                      )}
                      <span>{r.name}</span>
                      {r.isCustom && <span className="text-[10px] opacity-70">(Custom)</span>}
                    </button>
                  )
                })
              )}
            </div>
          </div>

          {/* Action Buttons */}
          <div className="flex items-center justify-between pt-4 select-none">
            {/* Delete / Revoke button (Only in Edit mode) */}
            {isEditMode && member && onDelete ? (
              <button
                type="button"
                onClick={() => {
                  onClose()
                  onDelete(member)
                }}
                className="inline-flex h-11 items-center gap-1.5 rounded-xl border border-destructive/20 bg-destructive/10 px-4 text-xs font-semibold text-destructive hover:bg-destructive/20 transition-colors"
              >
                <Trash2 className="h-4 w-4" />
                <span>Revoke</span>
              </button>
            ) : (
              <div />
            )}

            <div className="flex items-center gap-3">
              <button
                type="button"
                onClick={onClose}
                className="inline-flex h-11 items-center justify-center rounded-xl border border-input bg-background px-5 text-xs font-semibold text-foreground shadow-sm hover:bg-muted transition-colors"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={isPending}
                className="inline-flex h-11 items-center justify-center gap-2 rounded-xl bg-primary px-5 text-xs font-semibold text-primary-foreground shadow hover:opacity-90 disabled:opacity-50 transition-colors"
              >
                {isPending && <Loader2 className="h-3.5 w-3.5 animate-spin" />}
                {isEditMode ? "Save Changes" : "Add Member"}
              </button>
            </div>
          </div>

        </form>
      </div>
    </div>
  )

  return createPortal(modalContent, document.body)
}

// Alias for backwards compatibility
export const AddMemberModal = UserDetailModal
export type AddMemberModalProps = UserDetailModalProps

export default UserDetailModal
