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
import { useRouter } from "next/navigation"
import { toast } from "sonner"
import { Cable, Loader2, AlertCircle, RefreshCw } from "lucide-react"
import { useAuthStore } from "@/stores/auth"
import { useAppStore } from "@/stores/app"
import { api } from "@/lib/api"
import { AppIntegration } from "@/lib/types"
import AppIntegrationsList from "@/components/shared/AppIntegrationsList"

export default function IntegrationsPage() {
  const router = useRouter()
  const queryClient = useQueryClient()
  const { setPageTitle } = useAppStore()
  const { isAuthenticated, isCheckingAuth, isAdmin, selectedOrgId } = useAuthStore()

  // Expanded state (not utilized in admin-catalog but required by component contract)
  const [expandedAppIds, setExpandedAppIds] = useState<Record<string, boolean>>({})

  // Sync page title
  useEffect(() => {
    setPageTitle("App Integrations")
  }, [setPageTitle])

  // Role gate: redirect if not admin/owner
  useEffect(() => {
    if (!isCheckingAuth && isAuthenticated && !isAdmin) {
      router.replace("/projects")
      toast.error("Access denied: Organization administrators only.")
    }
  }, [isAdmin, isCheckingAuth, isAuthenticated, router])

  // Query: Fetch integrations catalog (global merged with org state)
  const {
    data: integrations = [],
    isLoading,
    error,
  } = useQuery<AppIntegration[]>({
    queryKey: ["app-integrations"],
    queryFn: async () => {
      const res = await api.get("/api/v1/config/app-integrations")
      if (!res.ok) throw new Error("Failed to load integrations catalog")
      return res.json()
    },
    enabled: isAuthenticated && isAdmin && !!selectedOrgId,
  })

  // Mutation: Enable an integration
  const enableMutation = useMutation({
    mutationFn: async (appId: string) => {
      if (!selectedOrgId) return
      const url = `/api/v1/admin/organizations/${selectedOrgId}/config/app-integrations/${appId}/enable`
      const res = await api.post(url, {})
      if (!res.ok) throw new Error("Failed to enable integration")
      return appId
    },
    onSuccess: (appId) => {
      if (!appId) return
      toast.success(`Successfully enabled ${appId} integration!`)
      queryClient.invalidateQueries({ queryKey: ["app-integrations"] })
    },
    onError: (err) => {
      console.error("Enable error:", err)
      toast.error("Failed to enable integration.")
    },
  })

  // Mutation: Disable an integration
  const disableMutation = useMutation({
    mutationFn: async (appId: string) => {
      if (!selectedOrgId) return
      const url = `/api/v1/admin/organizations/${selectedOrgId}/config/app-integrations/${appId}/disable`
      const res = await api.post(url, {})
      if (!res.ok) throw new Error("Failed to disable integration")
      return appId
    },
    onSuccess: (appId) => {
      if (!appId) return
      toast.success(`Successfully disabled ${appId} integration!`)
      queryClient.invalidateQueries({ queryKey: ["app-integrations"] })
    },
    onError: (err) => {
      console.error("Disable error:", err)
      toast.error("Failed to disable integration.")
    },
  })

  const isPending = enableMutation.isPending || disableMutation.isPending
  const pendingAppId = enableMutation.variables || disableMutation.variables

  if (isCheckingAuth || isLoading) {
    return (
      <div className="flex h-[400px] w-full flex-col items-center justify-center gap-3 text-muted-foreground select-none">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
        <span className="text-sm font-medium">Loading integrations catalog...</span>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex items-center justify-center p-8 min-h-[400px]">
        <div className="w-full max-w-md bg-destructive/10 border border-destructive/20 rounded-2xl p-6 text-center space-y-4">
          <div className="flex justify-center text-destructive">
            <AlertCircle className="h-12 w-12" />
          </div>
          <h3 className="text-lg font-bold text-destructive">Failed to load integrations</h3>
          <p className="text-sm text-muted-foreground leading-relaxed">
            {error instanceof Error ? error.message : "An unknown error occurred."}
          </p>
          <button
            onClick={() => queryClient.refetchQueries({ queryKey: ["app-integrations"] })}
            className="inline-flex h-10 items-center justify-center gap-2 rounded-xl bg-destructive px-5 text-sm font-semibold text-destructive-foreground shadow hover:opacity-90 transition-colors"
          >
            <RefreshCw className="h-4 w-4" />
            Retry
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="p-8 max-w-5xl space-y-8">
      {/* Header section */}
      <div className="space-y-2 border-b border-border pb-6 select-none">
        <div className="flex items-center gap-3">
          <Cable className="h-7 w-7 text-primary" />
          <h1 className="text-2xl font-bold text-foreground">App Integrations</h1>
        </div>
        <p className="text-sm text-muted-foreground leading-relaxed max-w-3xl">
          Manage third-party integrations available for your organization. 
          Enabling an integration allows team members to configure their personal access tokens and link tools to their workspace projects.
        </p>
      </div>

      {/* Reusable Integrations Catalog Component */}
      <AppIntegrationsList
        mode="admin-catalog"
        integrations={integrations}
        expandedAppIds={expandedAppIds}
        setExpandedAppIds={setExpandedAppIds}
        isAdmin={isAdmin}
        onEnableIntegration={async (id) => { await enableMutation.mutateAsync(id) }}
        onDisableIntegration={async (id) => { await disableMutation.mutateAsync(id) }}
        isPendingEnableDisable={isPending}
        pendingAppId={pendingAppId}
      />
    </div>
  )
}
