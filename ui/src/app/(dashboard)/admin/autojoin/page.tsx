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

import { useState, useEffect } from "react"
import { useAuthStore } from "@/stores/auth"
import { useAppStore } from "@/stores/app"
import { 
  UserPlus, 
  Building2, 
  Shield, 
  Plus, 
  Trash2, 
  Edit3, 
  Check, 
  X, 
  RefreshCw, 
  AlertCircle, 
  Search, 
  ToggleLeft, 
  ToggleRight,
  Sparkles,
  ShieldAlert,
  Loader2
} from "lucide-react"
import { cn } from "@/lib/utils"

interface AutojoinRule {
  id: string
  domain: string
  organization_id: string
  default_roles: string[]
  is_active: boolean
  created_at?: string
  updated_at?: string
}

interface Organization {
  id: string
  name: string
}

export default function AutojoinManagementPage() {
  const { setPageTitle } = useAppStore()
  const { token, isSuperAdmin, isCheckingAuth } = useAuthStore()

  // State
  const [rules, setRules] = useState<AutojoinRule[]>([])
  const [orgs, setOrgs] = useState<Organization[]>([])
  const [platformRoles, setPlatformRoles] = useState<{ value: string; label: string; desc: string }[]>([])
  const [searchQuery, setSearchQuery] = useState("")
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Modal State
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [editingRule, setEditingRule] = useState<AutojoinRule | null>(null)
  const [formDomain, setFormDomain] = useState("")
  const [formOrgId, setFormOrgId] = useState("")
  const [formRoles, setFormRoles] = useState<string[]>([])
  const [formIsActive, setFormIsActive] = useState(true)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [modalError, setModalError] = useState<string | null>(null)

  // Delete Modal State
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false)
  const [deleteTargetId, setDeleteTargetId] = useState<string | null>(null)

  // Sync page title
  useEffect(() => {
    setPageTitle("Admin: Autojoin Rules")
  }, [setPageTitle])

  // Fetch initial data
  const fetchData = async () => {
    setIsLoading(true)
    setError(null)
    try {
      // 1. Fetch rules
      const rulesRes = await fetch("/api/v1/admin/autojoin", {
        headers: {
          "Content-Type": "application/json",
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
      })
      if (!rulesRes.ok) throw new Error("Failed to load autojoin rules")
      const rulesData = await rulesRes.json()
      setRules(rulesData || [])

      // 2. Fetch organizations
      const orgsRes = await fetch("/api/v1/admin/organizations", {
        headers: {
          "Content-Type": "application/json",
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
      })
      if (!orgsRes.ok) throw new Error("Failed to load organizations list")
      const orgsData = await orgsRes.json()
      setOrgs(orgsData || [])

      // 3. Fetch platform roles
      const rolesRes = await fetch("/api/v1/admin/roles", {
        headers: {
          "Content-Type": "application/json",
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
      })
      if (!rolesRes.ok) throw new Error("Failed to load platform roles")
      const rolesData = await rolesRes.json()
      const mappedRoles = (rolesData.roles || []).map((r: any) => ({
        value: r.id,
        label: r.name,
        desc: r.description
      }))
      setPlatformRoles(mappedRoles)
    } catch (err: any) {
      console.error(err)
      setError(err.message || "An unexpected error occurred while fetching portal data")
    } finally {
      setIsLoading(false)
    }
  }

  useEffect(() => {
    if (token && isSuperAdmin) {
      fetchData()
    }
  }, [token, isSuperAdmin])

  if (isCheckingAuth) {
    return (
      <div className="flex-1 flex h-[400px] w-full flex-col items-center justify-center gap-3 text-muted-foreground select-none">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
        <span className="text-sm font-medium">Verifying access...</span>
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

  // Toggle active state
  const handleToggleActive = async (rule: AutojoinRule) => {
    const updatedRule = { ...rule, is_active: !rule.is_active }
    try {
      const response = await fetch(`/api/v1/admin/autojoin/${rule.id}`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify(updatedRule),
      })
      if (!response.ok) throw new Error("Failed to update rule status")
      
      // Update locally
      setRules(prev => prev.map(r => r.id === rule.id ? updatedRule : r))
    } catch (err: any) {
      setError(err.message || "Failed to update rule status")
    }
  }

  // Trigger delete confirmation modal
  const handleDeleteRule = (id: string) => {
    setDeleteTargetId(id)
    setIsDeleteModalOpen(true)
  }

  // Actual deletion execution
  const handleDelete = async () => {
    if (!deleteTargetId) return
    try {
      const response = await fetch(`/api/v1/admin/autojoin/${deleteTargetId}`, {
        method: "DELETE",
        headers: {
          "Content-Type": "application/json",
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
      })
      if (!response.ok) throw new Error("Failed to delete mapping rule")

      setRules(prev => prev.filter(r => r.id !== deleteTargetId))
      setIsDeleteModalOpen(false)
      setDeleteTargetId(null)
    } catch (err: any) {
      setError(err.message || "Deletion failed")
    }
  }

  // Open modal for Create
  const openCreateModal = () => {
    setEditingRule(null)
    setFormDomain("")
    setFormOrgId(orgs.length > 0 ? orgs[0].id : "")
    setFormRoles(platformRoles.length > 0 ? [platformRoles[3].value] : ["org:viewer"])
    setFormIsActive(true)
    setModalError(null)
    setIsModalOpen(true)
  }

  // Open modal for Edit
  const openEditModal = (rule: AutojoinRule) => {
    setEditingRule(rule)
    setFormDomain(rule.domain)
    setFormOrgId(rule.organization_id)
    setFormRoles(rule.default_roles || [])
    setFormIsActive(rule.is_active)
    setModalError(null)
    setIsModalOpen(true)
  }

  // Toggle role in form
  const handleToggleRole = (roleValue: string) => {
    setFormRoles(prev => 
      prev.includes(roleValue) 
        ? prev.filter(r => r !== roleValue) 
        : [...prev, roleValue]
    )
  }

  // Submit form
  const handleSubmitForm = async (e: React.FormEvent) => {
    e.preventDefault()
    setModalError(null)
    
    const trimmedDomain = formDomain.trim().toLowerCase()
    const domainRegex = /^(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$/
    
    if (!trimmedDomain) {
      setModalError("Email domain is required")
      return
    }
    if (!domainRegex.test(trimmedDomain)) {
      setModalError("Please enter a valid domain format (e.g., company.com)")
      return
    }
    if (!formOrgId) {
      setModalError("Please select a target organization")
      return
    }
    if (formRoles.length === 0) {
      setModalError("Please select at least one default role")
      return
    }

    setIsSubmitting(true)
    const payload = {
      domain: trimmedDomain,
      organization_id: formOrgId,
      default_roles: formRoles,
      is_active: formIsActive,
    }

    try {
      if (editingRule) {
        // Update
        const response = await fetch(`/api/v1/admin/autojoin/${editingRule.id}`, {
          method: "PUT",
          headers: {
            "Content-Type": "application/json",
            ...(token ? { Authorization: `Bearer ${token}` } : {}),
          },
          body: JSON.stringify(payload),
        })
        if (!response.ok) {
          if (response.status === 409) throw new Error(`An active autojoin rule already exists for domain: ${payload.domain}`)
          throw new Error("Failed to update rule")
        }
        
        setRules(prev => prev.map(r => r.id === editingRule.id ? { ...r, ...payload } : r))
      } else {
        // Create
        const response = await fetch("/api/v1/admin/autojoin", {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            ...(token ? { Authorization: `Bearer ${token}` } : {}),
          },
          body: JSON.stringify(payload),
        })
        if (!response.ok) {
          if (response.status === 409) throw new Error(`An active autojoin rule already exists for domain: ${payload.domain}`)
          throw new Error("Failed to create rule")
        }
        const resData = await response.json()
        
        setRules(prev => [...prev, { id: resData.id, ...payload }])
      }
      setIsModalOpen(false)
    } catch (err: any) {
      setModalError(err.message || "Failed to save autojoin rule")
    } finally {
      setIsSubmitting(false)
    }
  }

  // Helpers
  const getOrgName = (id: string) => {
    const org = orgs.find(o => o.id === id)
    return org ? org.name : id
  }

  const filteredRules = rules.filter(r => 
    r.domain.toLowerCase().includes(searchQuery.toLowerCase()) ||
    getOrgName(r.organization_id).toLowerCase().includes(searchQuery.toLowerCase())
  )

  return (
    <main className="flex-1 overflow-y-auto p-8">
      <div className="w-full max-w-[1000px] space-y-8 pb-12">
        
        {/* Header */}
        <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4 border-b border-border pb-6">
          <div className="space-y-2">
            <div className="flex items-center gap-3">
              <div className="h-10 w-10 rounded-xl bg-primary/10 flex items-center justify-center text-primary">
                <UserPlus className="h-5 w-5" />
              </div>
              <div>
                <h1 className="text-xl font-bold tracking-tight text-foreground flex items-center gap-2">
                  Autojoin Configurations <Sparkles className="h-4 w-4 text-primary animate-pulse" />
                </h1>
                <p className="text-xs text-muted-foreground">
                  Configure claims-based auto-provisioning mapping rules for domains.
                </p>
              </div>
            </div>
          </div>

          <button
            onClick={openCreateModal}
            className="flex h-10 items-center justify-center gap-2 rounded-xl bg-primary hover:opacity-90 text-primary-foreground text-xs font-bold shadow-md transition-all px-4"
          >
            <Plus className="h-4 w-4" /> CREATE RULE
          </button>
        </div>

        {/* Info banner */}
        <div className="p-4 rounded-2xl bg-muted/50 border border-border text-muted-foreground text-xs flex items-start gap-3">
          <Shield className="h-5 w-5 text-primary shrink-0 mt-0.5" />
          <div>
            <span className="font-semibold text-foreground">How it works:</span> When a new user logs in, the platform interceptor extracts their verified email domain. If an active rule matches the domain, they are automatically added to the target organization with all selected default roles, routing them directly to their team dashboard.
          </div>
        </div>

        {/* Error Notification */}
        {error && (
          <div className="p-4 rounded-xl bg-destructive/10 border border-destructive/20 text-destructive text-xs flex items-center gap-3">
            <AlertCircle className="h-5 w-5 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        {/* Main Section */}
        <div className="bg-card border border-border rounded-2xl overflow-hidden shadow-sm">
          
          {/* Controls Bar */}
          <div className="p-4 md:p-6 border-b border-border flex flex-col md:flex-row md:items-center gap-4 bg-muted/20">
            <div className="relative flex-1">
              <Search className="absolute left-3.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <input
                type="text"
                placeholder="Search domain or organization..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-full h-10 pl-10 pr-4 rounded-xl border border-input bg-background text-xs focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all text-foreground"
              />
            </div>
            
            <button 
              onClick={fetchData}
              disabled={isLoading}
              className="h-10 px-4 rounded-xl border border-input bg-background hover:bg-muted text-xs font-semibold flex items-center justify-center gap-2 text-muted-foreground transition-colors disabled:opacity-50"
            >
              <RefreshCw className={`h-3.5 w-3.5 ${isLoading ? 'animate-spin' : ''}`} /> RELOAD RULES
            </button>
          </div>

          {/* Table Container */}
          {isLoading ? (
            <div className="flex flex-col items-center justify-center py-20 gap-4">
              <RefreshCw className="h-8 w-8 text-primary animate-spin" />
              <p className="text-xs text-muted-foreground font-medium">Fetching secure configurations...</p>
            </div>
          ) : filteredRules.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-20 text-center px-6">
              <div className="h-12 w-12 rounded-2xl bg-muted flex items-center justify-center text-muted-foreground mb-4 border border-border">
                <UserPlus className="h-6 w-6" />
              </div>
              <h3 className="font-bold text-foreground text-sm">No autojoin rules found</h3>
              <p className="text-xs text-muted-foreground mt-1 max-w-sm">
                {searchQuery ? "No rules matched your search query." : "Establish mapping rules to automatically provision domains on login."}
              </p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-left border-collapse">
                <thead>
                  <tr className="border-b border-border text-[10px] font-bold text-muted-foreground uppercase bg-muted/30">
                    <th className="py-4 px-6">Email Domain</th>
                    <th className="py-4 px-6">Target Organization</th>
                    <th className="py-4 px-6">Assigned Default Roles</th>
                    <th className="py-4 px-6 text-center">Status</th>
                    <th className="py-4 px-6 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border/60 text-xs">
                  {filteredRules.map((rule) => (
                    <tr key={rule.id} className="hover:bg-muted/20 transition-colors group">
                      
                      {/* Domain */}
                      <td className="py-4 px-6 font-medium text-foreground">
                        <div className="flex items-center gap-2">
                          <span className="h-2 w-2 rounded-full bg-primary animate-pulse" />
                          {rule.domain}
                        </div>
                      </td>

                      {/* Organization */}
                      <td className="py-4 px-6 text-muted-foreground">
                        <div className="flex items-center gap-2">
                          <Building2 className="h-4 w-4 text-muted-foreground" />
                          <span>{getOrgName(rule.organization_id)}</span>
                        </div>
                      </td>

                      {/* Roles */}
                      <td className="py-4 px-6">
                        <div className="flex flex-wrap gap-1.5 max-w-md">
                          {rule.default_roles && rule.default_roles.map((r, idx) => (
                            <span 
                              key={idx} 
                              className="inline-flex items-center px-2 py-0.5 rounded-md text-[10px] font-semibold bg-primary/10 border border-primary/20 text-primary"
                            >
                              {r}
                            </span>
                          ))}
                        </div>
                      </td>

                      {/* Toggle status */}
                      <td className="py-4 px-6 text-center">
                        <button 
                          onClick={() => handleToggleActive(rule)}
                          className="focus:outline-none transition-transform hover:scale-105"
                          title={rule.is_active ? "Deactivate rule" : "Activate rule"}
                        >
                          {rule.is_active ? (
                            <div className="flex items-center justify-center gap-1.5 px-2.5 py-1 rounded-full text-[10px] font-bold bg-emerald-50 dark:bg-emerald-950/30 border border-emerald-200 dark:border-emerald-800/30 text-emerald-600 dark:text-emerald-400">
                              <Check className="h-3 w-3" /> Active
                            </div>
                          ) : (
                            <div className="flex items-center justify-center gap-1.5 px-2.5 py-1 rounded-full text-[10px] font-bold bg-muted border border-border text-muted-foreground">
                              <X className="h-3 w-3" /> Inactive
                            </div>
                          )}
                        </button>
                      </td>

                      {/* Actions */}
                      <td className="py-4 px-6 text-right">
                        <div className="flex items-center justify-end gap-2">
                          <button
                            onClick={() => openEditModal(rule)}
                            className="h-8 w-8 rounded-lg border border-border bg-background hover:bg-muted text-muted-foreground hover:text-foreground flex items-center justify-center transition-all"
                            title="Edit rule"
                          >
                            <Edit3 className="h-3.5 w-3.5" />
                          </button>
                          <button
                            onClick={() => handleDeleteRule(rule.id)}
                            className="h-8 w-8 rounded-lg border border-border bg-background hover:bg-destructive/10 hover:border-destructive/20 text-muted-foreground hover:text-destructive flex items-center justify-center transition-all"
                            title="Delete rule"
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                          </button>
                        </div>
                      </td>

                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

        </div>

      </div>

      {/* Slide-over or Modal Dialog */}
      {isModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-background/80 backdrop-blur-sm transition-opacity">
          <div className="w-full max-w-xl rounded-2xl border border-border bg-card p-6 md:p-8 shadow-2xl space-y-6 text-left text-card-foreground">
            
            {/* Modal Header */}
            <div className="flex items-center justify-between border-b border-border pb-4">
              <h2 className="text-base font-bold text-foreground flex items-center gap-2">
                {editingRule ? "Edit Autojoin Configuration" : "Create Autojoin Rule"}
              </h2>
              <button 
                onClick={() => setIsModalOpen(false)}
                className="h-8 w-8 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground flex items-center justify-center transition-colors"
              >
                <X className="h-4 w-4" />
              </button>
            </div>

            {/* Local Modal Error Banner */}
            {modalError && (
              <div className="p-3.5 rounded-xl bg-destructive/10 border border-destructive/20 text-destructive text-xs flex items-center gap-2.5">
                <AlertCircle className="h-4 w-4 shrink-0" />
                <span>{modalError}</span>
              </div>
            )}

            {/* Form */}
            <form onSubmit={handleSubmitForm} className="space-y-5">
              
              {/* Domain Input */}
              <div className="space-y-1">
                <label className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                  Email Domain
                </label>
                <input
                  type="text"
                  placeholder="e.g. google.com"
                  required
                  value={formDomain}
                  onChange={(e) => setFormDomain(e.target.value)}
                  className="w-full h-11 px-4 rounded-xl border border-input bg-background text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all placeholder:text-muted-foreground/50"
                />
                <p className="text-[10px] text-muted-foreground">
                  Matches domain exactly (case-insensitive). Do not include '@'.
                </p>
              </div>

              {/* Organization Select */}
              <div className="space-y-1">
                <label className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                  Target Organization
                </label>
                <select
                  value={formOrgId}
                  onChange={(e) => setFormOrgId(e.target.value)}
                  className="w-full h-11 px-4 rounded-xl border border-input bg-background text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all"
                >
                  {orgs.map((org) => (
                    <option key={org.id} value={org.id} className="bg-card text-foreground">
                      {org.name} ({org.id})
                    </option>
                  ))}
                </select>
              </div>

              {/* Roles selection grid */}
              <div className="space-y-2">
                <label className="text-xs font-semibold text-muted-foreground uppercase tracking-wider block">
                  Default Assigned Roles
                </label>
                <div className="grid grid-cols-1 gap-2.5 max-h-56 overflow-y-auto pr-1">
                  {platformRoles.map((role) => (
                    <button
                      key={role.value}
                      type="button"
                      onClick={() => handleToggleRole(role.value)}
                      className={`w-full p-3 rounded-xl border text-left flex items-start gap-3 transition-all ${
                        formRoles.includes(role.value)
                          ? "border-primary/60 bg-primary/5 text-foreground"
                          : "border-border hover:border-muted-foreground/30 bg-background text-muted-foreground hover:text-foreground"
                      }`}
                    >
                      <div className={`h-5 w-5 rounded-md border flex items-center justify-center shrink-0 mt-0.5 transition-all ${
                        formRoles.includes(role.value)
                          ? "bg-primary border-primary text-primary-foreground"
                          : "border-input bg-transparent"
                      }`}>
                        {formRoles.includes(role.value) && <Check className="h-3 w-3 stroke-[3]" />}
                      </div>
                      <div>
                        <div className="text-xs font-bold text-foreground">{role.label}</div>
                        <div className="text-[10px] text-muted-foreground mt-0.5 leading-relaxed">{role.desc}</div>
                      </div>
                    </button>
                  ))}
                </div>
              </div>

              {/* Active Toggle */}
              <div className="flex items-center justify-between p-3.5 rounded-xl border border-border bg-muted/20">
                <div>
                  <div className="text-xs font-bold text-foreground">Enable Rule Immediately</div>
                  <div className="text-[10px] text-muted-foreground">Enable this rule for all incoming OIDC handshakes.</div>
                </div>
                <button
                  type="button"
                  onClick={() => setFormIsActive(!formIsActive)}
                  className="focus:outline-none text-muted-foreground hover:text-foreground transition-colors"
                >
                  {formIsActive ? (
                    <ToggleRight className="h-8 w-8 text-primary" />
                  ) : (
                    <ToggleLeft className="h-8 w-8" />
                  )}
                </button>
              </div>

              {/* Submit Buttons */}
              <div className="flex items-center gap-3 pt-4 border-t border-border">
                <button
                  type="button"
                  onClick={() => setIsModalOpen(false)}
                  className="flex-1 h-11 rounded-xl border border-input hover:bg-muted text-muted-foreground text-sm font-semibold transition-colors"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={isSubmitting}
                  className="flex-1 h-11 rounded-xl bg-primary hover:opacity-90 text-primary-foreground text-sm font-bold shadow-md transition-all flex items-center justify-center gap-2 disabled:opacity-50"
                >
                  {isSubmitting ? (
                    <RefreshCw className="h-4 w-4 animate-spin" />
                  ) : (
                    editingRule ? "SAVE CHANGES" : "CREATE RULE"
                  )}
                </button>
              </div>

            </form>

          </div>
        </div>
      )}

      {/* Custom Delete Confirmation Modal Overlay */}
      {isDeleteModalOpen && (
        <div className="fixed inset-0 z-[60] flex items-center justify-center p-4 bg-background/90 backdrop-blur-sm transition-opacity">
          <div className="w-full max-w-md rounded-2xl border border-border bg-card p-6 shadow-2xl space-y-6 text-left text-card-foreground">
            <div className="space-y-2">
              <h3 className="text-base font-bold text-foreground flex items-center gap-2">
                <AlertCircle className="h-5 w-5 text-destructive" /> Delete Autojoin Rule
              </h3>
              <p className="text-xs text-muted-foreground leading-relaxed">
                Are you absolutely sure you want to delete this autojoin mapping? This will immediately prevent new users from this domain from being auto-provisioned. This action cannot be undone.
              </p>
            </div>
            <div className="flex items-center justify-end gap-3 pt-2">
              <button
                type="button"
                onClick={() => {
                  setIsDeleteModalOpen(false)
                  setDeleteTargetId(null)
                }}
                className="h-10 px-4 rounded-xl border border-input bg-background hover:bg-muted text-xs font-bold text-muted-foreground transition-colors"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleDelete}
                className="h-10 px-4 rounded-xl bg-destructive hover:opacity-90 text-xs font-bold text-destructive-foreground shadow-md transition-all"
              >
                Delete Rule
              </button>
            </div>
          </div>
        </div>
      )}
    </main>
  )
}
