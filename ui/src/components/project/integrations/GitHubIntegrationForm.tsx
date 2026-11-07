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
import { useRouter } from "next/navigation"
import { AlertTriangle, Key, Save, Loader2 } from "lucide-react"
import { toast } from "sonner"
import { api } from "@/lib/api"
import SearchableRepoDropdown from "./SearchableRepoDropdown"
import RepoChangeWarningModal from "@/app/(dashboard)/projects/[id]/details/_components/RepoChangeWarningModal"

interface GitHubIntegrationFormProps {
  projectId: string
  onSaveSuccess?: () => void
  repoUrl?: string
  setRepoUrl?: (url: string) => void
  specLocation?: string
  setSpecLocation?: (path: string) => void
  baseBranch?: string
  setBaseBranch?: (branch: string) => void
  specBranchPrefix?: string
  setSpecBranchPrefix?: (prefix: string) => void
  email?: string
  setEmail?: (email: string) => void
  enabled?: boolean
  setEnabled?: (enabled: boolean) => void
  minimal?: boolean
  hasCredentials?: boolean
  customToken?: string
  isOverridden?: boolean
}

export default function GitHubIntegrationForm({
  projectId,
  onSaveSuccess,
  repoUrl: parentRepoUrl,
  setRepoUrl: parentSetRepoUrl,
  specLocation: parentSpecLocation,
  setSpecLocation: parentSetSpecLocation,
  baseBranch: parentBaseBranch,
  setBaseBranch: parentSetBaseBranch,
  specBranchPrefix: parentSpecBranchPrefix,
  setSpecBranchPrefix: parentSetSpecBranchPrefix,
  email: parentEmail,
  setEmail: parentSetEmail,
  enabled: parentEnabled,
  setEnabled: parentSetEnabled,
  minimal,
  hasCredentials,
  customToken,
  isOverridden,
}: GitHubIntegrationFormProps) {
  const router = useRouter()
  const [isConnected, setIsConnected] = useState<boolean | null>(null)
  const [isLoadingStatus, setIsLoadingStatus] = useState(true)

  // Form settings states
  const [localRepoUrl, setLocalRepoUrl] = useState("")
  const [localSpecLocation, setLocalSpecLocation] = useState("docs/specs")
  const [localBaseBranch, setLocalBaseBranch] = useState("main")
  const [localSpecBranchPrefix, setLocalSpecBranchPrefix] = useState("spec-")
  const [localEmail, setLocalEmail] = useState("")
  const [localEnabled, setLocalEnabled] = useState(true)

  const repoUrl = parentRepoUrl !== undefined ? parentRepoUrl : localRepoUrl
  const setRepoUrl = parentSetRepoUrl !== undefined ? parentSetRepoUrl : setLocalRepoUrl

  const specLocation = parentSpecLocation !== undefined ? parentSpecLocation : localSpecLocation
  const setSpecLocation = parentSetSpecLocation !== undefined ? parentSetSpecLocation : setLocalSpecLocation

  const baseBranch = parentBaseBranch !== undefined ? parentBaseBranch : localBaseBranch
  const setBaseBranch = parentSetBaseBranch !== undefined ? parentSetBaseBranch : setLocalBaseBranch

  const specBranchPrefix = parentSpecBranchPrefix !== undefined ? parentSpecBranchPrefix : localSpecBranchPrefix
  const setSpecBranchPrefix = parentSetSpecBranchPrefix !== undefined ? parentSetSpecBranchPrefix : setLocalSpecBranchPrefix

  const email = parentEmail !== undefined ? parentEmail : localEmail
  const setEmail = parentSetEmail !== undefined ? parentSetEmail : setLocalEmail

  const enabled = parentEnabled !== undefined ? parentEnabled : localEnabled
  const setEnabled = parentSetEnabled !== undefined ? parentSetEnabled : setLocalEnabled

  const [isSaving, setIsSaving] = useState(false)

  // Check if GitHub credential is configured
  useEffect(() => {
    if (hasCredentials !== undefined) {
      setIsConnected(hasCredentials)
      setIsLoadingStatus(false)
      return
    }

    async function checkStatus() {
      try {
        const res = await api.get("/api/v1/integrations/github/status")
        if (res.ok) {
          const data = await res.json()
          setIsConnected(data.connected)
        } else {
          setIsConnected(false)
        }
      } catch (err) {
        console.error(err)
        setIsConnected(false)
      } finally {
        setIsLoadingStatus(false)
      }
    }
    checkStatus()
  }, [hasCredentials])

  // Automatically fetch profile email if connected or custom token is provided and email is empty
  useEffect(() => {
    if (isConnected !== true) return
    if (email) return

    let cancelled = false
    async function fetchProfile() {
      try {
        const headers: Record<string, string> = {}
        const hasRealToken = (isOverridden ?? true) && customToken && !/[•·]/.test(customToken) && customToken.trim() !== ""
        if (hasRealToken) {
          headers["X-GitHub-Token"] = customToken
        }
        const res = await api.get("/api/v1/integrations/github/profile", Object.keys(headers).length > 0 ? { headers } : undefined)
        if (res.ok && !cancelled) {
          const data = await res.json()
          if (data.email) {
            setEmail(data.email)
          }
        }
      } catch (err) {
        console.error("Failed to fetch GitHub profile email:", err)
      }
    }
    fetchProfile()
    return () => {
      cancelled = true
    }
  }, [isConnected, customToken, isOverridden, email, setEmail])

  const [initialRepoUrl, setInitialRepoUrl] = useState("")

  // Load existing settings for this integration from project AppIntegrations
  useEffect(() => {
    if (projectId === "new") return
    if (parentRepoUrl !== undefined) return // Skip loading if parent controls it

    async function loadProjectSettings() {
      try {
        const res = await api.get(`/api/v1/projects/${projectId}`)
        if (res.ok) {
          const data = await res.json()
          const githubIntegration = data.app_integrations?.find(
            (integration: any) => integration.integration_id === "github"
          )
          if (githubIntegration) {
            const loadedUrl = githubIntegration.settings?.repoUrl || ""
            setEnabled(githubIntegration.enabled)
            setRepoUrl(loadedUrl)
            setInitialRepoUrl(loadedUrl)
            setSpecLocation(githubIntegration.settings?.specLocation || "docs/specs")
            setBaseBranch(githubIntegration.settings?.baseBranch || "main")
            setSpecBranchPrefix(githubIntegration.settings?.specBranchPrefix || "spec-")
            setEmail(githubIntegration.settings?.email || "")
          }
        }
      } catch (err) {
        console.error("Failed to load project integration settings:", err)
      }
    }
    loadProjectSettings()
  }, [projectId, parentRepoUrl])

  const [isRepoChangeModalOpen, setIsRepoChangeModalOpen] = useState(false)

  const executeSave = async () => {
    setIsSaving(true)
    try {
      const res = await api.post(`/api/v1/projects/${projectId}/integrations`, {
        integration_id: "github",
        enabled: enabled,
        settings: {
          repoUrl: repoUrl.trim(),
          specLocation: specLocation.trim(),
          baseBranch: baseBranch.trim(),
          specBranchPrefix: specBranchPrefix.trim(),
          email: email.trim(),
        },
      })

      if (!res.ok) {
        const errData = await res.json()
        throw new Error(errData.message || "Failed to save integration settings")
      }

      toast.success("GitHub integration settings saved successfully!")
      if (onSaveSuccess) onSaveSuccess()
    } catch (err: any) {
      console.error(err)
      toast.error(err.message || "Failed to save integration settings")
    } finally {
      setIsSaving(false)
    }
  }

  const handleSave = async () => {
    // Validation
    if (!repoUrl.trim() || !specLocation.trim() || !baseBranch.trim() || !specBranchPrefix.trim()) {
      toast.error("Repository and branch settings are required to enable integration.")
      return
    }

    const isRepoChanged = projectId !== "new" && Boolean(repoUrl.trim()) && repoUrl.trim() !== initialRepoUrl
    if (isRepoChanged) {
      setIsRepoChangeModalOpen(true)
      return
    }

    await executeSave()
  }

  if (isLoadingStatus) {
    return (
      <div className="flex items-center justify-center py-6 gap-2 text-muted-foreground text-xs">
        <Loader2 className="h-4 w-4 animate-spin text-primary" />
        Loading integration settings...
      </div>
    )
  }

  if (minimal) {
    return (
      <div className="space-y-4">
        {/* CASE 1: Secrets are missing or invalid */}
        {isConnected === false ? (
          <div className="flex items-start gap-3 p-3.5 rounded-lg border border-amber-500/20 bg-amber-500/5 leading-relaxed">
            <AlertTriangle className="h-4.5 w-4.5 text-amber-500 shrink-0 mt-0.5 animate-pulse" />
            <div className="flex-1 space-y-2">
              <h4 className="text-xs font-bold text-foreground">GitHub Credentials Required</h4>
              <p className="text-[10px] text-muted-foreground leading-relaxed">
                To configure GitHub repository connections, you must first link your personal access token (PAT) in your User Settings.
              </p>
              <button
                type="button"
                id="tour-user-secrets-button"
                onClick={() => router.push("/settings")}
                className="inline-flex h-7.5 items-center gap-1 rounded-md bg-primary px-2.5 py-1.5 text-[10px] font-semibold text-primary-foreground shadow hover:opacity-90"
              >
                <Key className="h-3 w-3" />
                Go to Settings to link PAT
              </button>
            </div>
          </div>
        ) : (
          /* CASE 2: Connected - Render the settings form */
          <div className="space-y-4">
            <SearchableRepoDropdown
              value={repoUrl}
              onChange={setRepoUrl}
              onSelectRepo={(url, defaultBranch) => {
                if (defaultBranch && setBaseBranch) {
                  setBaseBranch(defaultBranch)
                }
              }}
              disabled={!enabled}
              customToken={customToken}
              isOverridden={isOverridden}
            />

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 select-none">
              <div className="space-y-1.5">
                <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider block">
                  Base Git Branch
                </label>
                <input
                  type="text"
                  placeholder="main"
                  value={baseBranch}
                  disabled={!enabled}
                  onChange={(e) => setBaseBranch(e.target.value)}
                  className="w-full h-9 px-3 rounded border border-input bg-card text-xs focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary disabled:opacity-50 disabled:bg-muted/30"
                />
              </div>

              <div className="space-y-1.5">
                <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider block">
                  Spec Branch Prefix
                </label>
                <input
                  type="text"
                  placeholder="spec-"
                  value={specBranchPrefix}
                  disabled={!enabled}
                  onChange={(e) => setSpecBranchPrefix(e.target.value)}
                  className="w-full h-9 px-3 rounded border border-input bg-card text-xs focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary disabled:opacity-50 disabled:bg-muted/30"
                />
              </div>
            </div>
          </div>
        )}
      </div>
    )
  }

  return (
    <div className="rounded-xl border border-border bg-card text-card-foreground p-5 space-y-5 shadow-sm">
      <div className="flex items-center justify-between select-none">
        <div>
          <h3 className="text-sm font-bold text-foreground">GitHub Integration Configuration</h3>
          <p className="text-[11px] text-muted-foreground leading-normal">
            Scope settings, path definitions and repository connections for project automation.
          </p>
        </div>

        <div className="flex items-center gap-3">
          <span className="text-xs text-muted-foreground">Integration Enabled</span>
          <input
            type="checkbox"
            checked={enabled}
            onChange={(e) => setEnabled(e.target.checked)}
            className="h-6 w-11 rounded-full bg-muted border border-border appearance-none cursor-pointer transition-colors relative checked:bg-primary before:content-[''] before:absolute before:h-4 before:w-4 before:rounded-full before:bg-card before:left-[3px] before:top-[3px] before:transition-transform checked:before:translate-x-5"
          />
        </div>
      </div>

      {/* CASE 1: Secrets are missing or invalid */}
      {isConnected === false ? (
        <div className="flex items-start gap-3 p-4 rounded-lg border border-amber-500/20 bg-amber-500/5 leading-relaxed">
          <AlertTriangle className="h-5 w-5 text-amber-500 shrink-0 mt-0.5 animate-pulse" />
          <div className="flex-1 space-y-2.5">
            <h4 className="text-xs font-bold text-foreground">GitHub Credentials Required</h4>
            <p className="text-[11px] text-muted-foreground">
              To enable and configure project-level GitHub repository connections, you must first link your personal access token (PAT) in your User Settings profile.
            </p>
            <button
              type="button"
              id="tour-user-secrets-button"
              onClick={() => router.push("/settings")}
              className="inline-flex h-8 items-center gap-1.5 rounded-md bg-primary px-3 text-xs font-semibold text-primary-foreground shadow hover:opacity-90"
            >
              <Key className="h-3.5 w-3.5" />
              Go to Settings to link PAT
            </button>
          </div>
        </div>
      ) : (
        /* CASE 2: Connected - Render the settings form */
        <div className="space-y-4">
          <SearchableRepoDropdown
            value={repoUrl}
            onChange={setRepoUrl}
            onSelectRepo={(url, defaultBranch) => {
              if (defaultBranch && setBaseBranch) {
                setBaseBranch(defaultBranch)
              }
            }}
            disabled={!enabled}
            customToken={customToken}
            isOverridden={isOverridden}
          />

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 select-none">
            <div className="space-y-1.5">
              <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider block">
                Base Git Branch
              </label>
              <input
                type="text"
                placeholder="main"
                value={baseBranch}
                disabled={!enabled}
                onChange={(e) => setBaseBranch(e.target.value)}
                className="w-full h-9 px-3 rounded border border-input bg-card text-xs focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary disabled:opacity-50 disabled:bg-muted/30"
              />
            </div>

            <div className="space-y-1.5">
              <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider block">
                Spec Branch Prefix
              </label>
              <input
                type="text"
                placeholder="spec-"
                value={specBranchPrefix}
                disabled={!enabled}
                onChange={(e) => setSpecBranchPrefix(e.target.value)}
                className="w-full h-9 px-3 rounded border border-input bg-card text-xs focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary disabled:opacity-50 disabled:bg-muted/30"
              />
            </div>
          </div>

          {projectId !== "new" && (
            <div className="flex items-center justify-end pt-2">
              <button
                type="button"
                onClick={handleSave}
                disabled={isSaving}
                className="inline-flex h-9 items-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-xs font-semibold text-primary-foreground shadow transition-all hover:opacity-90 disabled:opacity-50"
              >
                {isSaving ? (
                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                ) : (
                  <Save className="h-3.5 w-3.5" />
                )}
                Save Integration Settings
              </button>
            </div>
          )}
        </div>
      )}
      <RepoChangeWarningModal
        isOpen={isRepoChangeModalOpen}
        onClose={() => setIsRepoChangeModalOpen(false)}
        onConfirm={() => {
          setIsRepoChangeModalOpen(false)
          executeSave()
        }}
        initialRepoUrl={initialRepoUrl}
        newRepoUrl={repoUrl}
        isPending={isSaving}
      />
    </div>
  )
}
