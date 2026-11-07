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

import React, { useState } from "react"
import { useRouter } from "next/navigation"
import { toast } from "sonner"
import { cn } from "@/lib/utils"
import { 
  Cable, 
  Loader2, 
  AlertCircle, 
  RefreshCw, 
  CheckCircle2, 
  Power, 
  PowerOff,
  Github,
  Figma,
  Network,
  Layers,
  BarChart2,
  Gitlab,
  Trello,
  ListTodo,
  ChevronUp,
  ChevronDown,
  Circle,
  MinusCircle,
  Zap,
  Save,
  Trash2,
  Lock,
  Eye,
  EyeOff,
  User,
  Sliders,
  Link2Off,
  Settings,
  Edit3,
  Info,
  Sparkles,
  Undo2,
  Mail
} from "lucide-react"
import { AppIntegration, AppCredential, UserProject, CredentialAttribute } from "@/lib/types"
import GitHubIntegrationForm from "@/components/project/integrations/GitHubIntegrationForm"
import AlertModal from "@/components/shared/AlertModal"

// Centralized icon helpers
export const getAppIcon = (iconName?: string) => {
  if (!iconName) return Cable
  switch (iconName.toLowerCase()) {
    case 'github':
      return Github
    case 'figma':
      return Figma
    case 'lucidchart':
      return Network
    case 'stitch':
      return Layers
    case 'drawio':
      return BarChart2
    case 'gemini':
      return Sparkles
    case 'gitlab':
      return Gitlab
    case 'jira':
      return Trello
    case 'linear':
      return ListTodo
    default:
      return Cable
  }
}

const getFieldIcon = (fieldName: string) => {
  switch (fieldName) {
    case "username":
      return <User className="h-4.5 w-4.5 text-muted-foreground" />
    case "email":
      return <Mail className="h-4.5 w-4.5 text-muted-foreground" />
    case "password":
      return <Lock className="h-4.5 w-4.5 text-muted-foreground" />
    case "accessToken":
    case "pat":
      return <KeyIcon className="h-4.5 w-4.5 text-muted-foreground" />
    case "refreshToken":
      return <RefreshCw className="h-4.5 w-4.5 text-muted-foreground" />
    default:
      return <Sliders className="h-4.5 w-4.5 text-muted-foreground" />
  }
}

// KeyIcon alias to avoid naming conflict with lucide Key if we import it (wait, we didn't import Key, but we imported Key from lucide in settings page. Let's just use Sliders/Cable/Zap or import Key if needed. Wait, we have Key in Lucide! Let's import it or use Zap/Cable. Let's just use lucide Key but alias it if we need to. Ah, we didn't import Key in this file's imports! So we can just import Key from lucide-react. Let's check imports: Cable, Loader2... no Key! Let's add Key to imports if we want to use it. Or just use Sliders/Zap.)
// Let's import Key from lucide-react!
// Wait, in my imports list above: there is no Key. Let's add it. Oh, wait, I can just use a styled lucide Key.
// Let's make sure it is imported. I will add Key to the imports in the code.
// Actually, let's look at the imports in the CodeContent. I will add Key there.

const KeyIcon = (props: React.ComponentProps<typeof Cable>) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    width="24"
    height="24"
    viewBox="24"
    fill="none"
    stroke="currentColor"
    strokeWidth="2"
    strokeLinecap="round"
    strokeLinejoin="round"
    {...props}
  >
    <path d="m21 2-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0 3 3L22 7l-3-3m-3.5 3.5L19 4" />
  </svg>
)

const CREDENTIAL_MASK_PLACEHOLDER = "••••••••••••••••••••••••••••••••"

export interface AppIntegrationsListProps {
  mode: "admin-catalog" | "user-settings" | "project-override"
  
  // Data
  integrations: AppIntegration[]
  globalCredentials?: Record<string, AppCredential>
  
  // Shared state (expansion)
  expandedAppIds: Record<string, boolean>
  setExpandedAppIds: React.Dispatch<React.SetStateAction<Record<string, boolean>>>
  
  // ================= MODE: admin-catalog & user-settings =================
  isAdmin?: boolean
  onEnableIntegration?: (appId: string) => Promise<void>
  onDisableIntegration?: (appId: string) => Promise<void>
  isPendingEnableDisable?: boolean
  pendingAppId?: string
  
  // ================= MODE: user-settings =================
  formValues?: Record<string, Record<string, string>>
  onInputChange?: (appId: string, fieldName: string, value: string) => void
  showPasswordMap?: Record<string, boolean>
  onTogglePasswordVisibility?: (appId: string, fieldName: string) => void
  validationStates?: Record<string, { isValid: boolean; error?: string } | null>
  onValidateCredentials?: (appId: string) => Promise<void>
  onRevokeCredentials?: (appId: string) => Promise<void>
  onSaveCredentials?: (appId: string) => Promise<void>
  isSavingCredentials?: boolean
  isValidatingCredentials?: boolean
  isRevokingCredentials?: boolean
  
  // ================= MODE: project-override =================
  projectId?: string
  isNewProject?: boolean
  projectAppSettingsState?: Record<string, UserProject>
  savingAppIds?: Record<string, boolean>
  onToggleProjectApp?: (appId: string, enabled: boolean) => Promise<void>
  onOverrideProjectCredentials?: (appId: string) => void
  onRevertProjectToMaster?: (appId: string) => Promise<void>
  onProjectFieldChange?: (appId: string, fieldName: string, value: string) => void
  
  // GitHub specific project settings
  githubSettings?: {
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
  }
}

export default function AppIntegrationsList({
  mode,
  integrations,
  globalCredentials,
  expandedAppIds,
  setExpandedAppIds,
  isAdmin = false,
  onEnableIntegration,
  onDisableIntegration,
  isPendingEnableDisable = false,
  pendingAppId,
  formValues = {},
  onInputChange,
  showPasswordMap = {},
  onTogglePasswordVisibility,
  validationStates = {},
  onValidateCredentials,
  onRevokeCredentials,
  onSaveCredentials,
  isSavingCredentials = false,
  isValidatingCredentials = false,
  isRevokingCredentials = false,
  projectId = "",
  isNewProject = false,
  projectAppSettingsState = {},
  savingAppIds = {},
  onToggleProjectApp,
  onOverrideProjectCredentials,
  onRevertProjectToMaster,
  onProjectFieldChange,
  githubSettings
}: AppIntegrationsListProps) {
  const router = useRouter()
  const [selectedLabel, setSelectedLabel] = useState<string>("All")
  const [confirmingApp, setConfirmingApp] = useState<AppIntegration | null>(null)

  if (integrations.length === 0) {
    return (
      <div className="flex h-[200px] flex-col items-center justify-center text-center gap-2.5 text-muted-foreground border border-dashed border-border rounded-2xl select-none">
        <Cable className="h-10 w-10 opacity-30" />
        <span className="text-sm font-medium">No integrations available.</span>
      </div>
    )
  }

  // Extract distinct labels dynamically for filtering (only for catalog and settings modes)
  const distinctLabels = [
    "All",
    ...Array.from(new Set(integrations.flatMap((app) => app.labels || [])))
  ]

  const filteredApps = mode === "project-override"
    ? integrations
    : integrations.filter((app) => selectedLabel === "All" || app.labels?.includes(selectedLabel))

  const handleToggleEnableDisable = async (app: AppIntegration) => {
    if (app.isEnabled) {
      setConfirmingApp(app)
    } else {
      if (onEnableIntegration) {
        await onEnableIntegration(app.id)
      }
    }
  }

  // ================= RENDER METHOD: admin-catalog =================
  const renderAdminCatalogCard = (app: AppIntegration) => {
    const AppIcon = getAppIcon(app.icon)
    const isAppPending = isPendingEnableDisable && pendingAppId === app.id

    return (
      <div
        key={app.id}
        data-testid={`integration-card-${app.id}`}
        className="rounded-2xl border bg-card text-card-foreground shadow-sm overflow-hidden border-border p-5 flex items-center justify-between gap-6 transition-all duration-200 hover:shadow-sm relative"
      >
        {/* Left Side: Icon & Info Column */}
        <div className="flex items-start gap-4 flex-1 min-w-0">
          <div className={cn(
            "flex h-10 w-10 shrink-0 items-center justify-center rounded-full transition-colors",
            app.isEnabled 
              ? "bg-primary/10 text-primary" 
              : "bg-slate-100 dark:bg-slate-800 text-slate-500"
          )}>
            <AppIcon className="h-[22px] w-[22px]" />
          </div>

          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <h3 className="text-base font-bold text-foreground leading-tight truncate animate-fade-in">
                {app.name}
              </h3>
              {app.labels && app.labels.length > 0 && (
                <span className="inline-flex items-center text-[9px] font-bold px-2 py-0.5 rounded bg-muted text-muted-foreground uppercase tracking-wider shrink-0">
                  {app.labels[0]}
                </span>
              )}
            </div>
            <p className="text-[11px] text-muted-foreground mt-1 leading-normal line-clamp-1">
              {app.description}
            </p>
            
            {/* Status Indicator */}
            <div className="flex items-center gap-1.5 mt-2 select-none">
              <span className={`h-2 w-2 rounded-full ${app.isEnabled ? "bg-green-500 animate-pulse" : "bg-slate-300 dark:bg-slate-700"}`} />
              <span className="text-[10px] font-semibold text-muted-foreground">
                {app.isEnabled ? "Active / Enabled" : "Inactive / Disabled"}
              </span>
            </div>
          </div>
        </div>

        {/* Right Side: Toggle Button */}
        <div className="shrink-0 select-none">
          <button
            type="button"
            onClick={() => handleToggleEnableDisable(app)}
            disabled={isAppPending}
            className={cn(
              "inline-flex h-9 items-center justify-center gap-1.5 rounded-xl px-4 text-xs font-bold transition-all duration-200",
              app.isEnabled
                ? "bg-green-50 border border-green-200 text-green-600 hover:bg-red-50 hover:border-red-200 hover:text-red-600 dark:bg-green-500/10 dark:border-green-500/20 dark:text-green-400 dark:hover:bg-red-500/10 dark:hover:border-red-500/20 dark:hover:text-red-400 group"
                : "bg-primary text-primary-foreground shadow hover:opacity-90 disabled:opacity-50"
            )}
          >
            {isAppPending ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : app.isEnabled ? (
              <>
                <CheckCircle2 className="h-3.5 w-3.5 shrink-0 group-hover:hidden" />
                <PowerOff className="h-3.5 w-3.5 shrink-0 hidden group-hover:block" />
                <span className="group-hover:hidden">Active</span>
                <span className="hidden group-hover:block">Disable</span>
              </>
            ) : (
              <>
                <Power className="h-3.5 w-3.5 shrink-0" />
                <span>Enable</span>
              </>
            )}
          </button>
        </div>
      </div>
    )
  }

  // ================= RENDER METHOD: user-settings =================
  const renderUserSettingsCard = (app: AppIntegration) => {
    const isEnabled = app.isEnabled
    const isExpanded = expandedAppIds[app.id] || false
    const isConfigured = !!globalCredentials?.[app.id]
    const currentValues = formValues[app.id] || {}
    
    const isSaving = isSavingCredentials && pendingAppId === app.id
    const isRevoking = isRevokingCredentials && pendingAppId === app.id
    const isValidating = isValidatingCredentials && pendingAppId === app.id
    const isPendingToggle = isPendingEnableDisable && pendingAppId === app.id

    const validation = validationStates[app.id]
    const AppIcon = getAppIcon(app.icon)

    const renderField = (f: CredentialAttribute) => {
      if (!f.enabled) return null
      
      const currentVal = currentValues[f.name] || ""
      const isSecret = f.isSecret
      const showKey = `${app.id}-${f.name}`
      const showPassword = showPasswordMap[showKey] || false
      const isPlaceholder = isSecret && currentVal === CREDENTIAL_MASK_PLACEHOLDER
      
      let PrefixIcon = Sliders
      if (f.name === "username") {
        PrefixIcon = User
      } else if (f.name === "password") {
        PrefixIcon = Lock
      } else if (f.name === "accessToken" || f.name === "pat") {
        PrefixIcon = KeyIcon as any
      } else if (f.name === "refreshToken") {
        PrefixIcon = RefreshCw
      } else if (f.isSecret) {
        PrefixIcon = KeyIcon as any
      }

      return (
        <div key={f.name} className="space-y-1.5">
          <label className="text-xs font-semibold text-foreground/80 block" htmlFor={`${app.id}-${f.name}`}>
            {f.displayName} {f.required && <span className="text-destructive">*</span>}
          </label>
          
          <div className="relative">
            <div className="absolute left-3.5 top-1/2 -translate-y-1/2 text-muted-foreground select-none pointer-events-none">
              <PrefixIcon className="h-4.5 w-4.5" />
            </div>
            
            <input
              id={app.id === "github" && f.name === "pat" ? "tour-secrets-modal-input" : `tour-settings-input-${app.id}-${f.name}`}
              type={isSecret && (!showPassword || isPlaceholder) ? "password" : "text"}
              placeholder={isSecret ? "••••••••••••••••••••••••••••••••" : f.helpText || `Enter ${f.displayName.toLowerCase()}`}
              value={currentVal}
              onChange={(e) => onInputChange?.(app.id, f.name, e.target.value)}
              onFocus={(e) => e.target.select()}
              className={cn(
                "w-full h-12 pl-11 rounded-xl border border-input bg-background text-sm transition-all focus:outline-none focus:ring-2 focus:ring-primary/25 focus:border-primary",
                isSecret ? "pr-11" : "pr-4"
              )}
            />
            
            {isSecret && (
              <button
                type="button"
                onClick={() => {
                  if (isPlaceholder) {
                    toast("Saved credentials cannot be revealed for security. You can overwrite them by typing a new value.")
                    return
                  }
                  onTogglePasswordVisibility?.(app.id, f.name)
                }}
                title={isPlaceholder ? "Saved credentials cannot be revealed" : showPassword ? "Hide password" : "Show password"}
                className={cn(
                  "absolute right-2 top-1/2 -translate-y-1/2 p-1.5 rounded-lg text-muted-foreground hover:text-foreground hover:bg-muted transition-colors z-10",
                  isPlaceholder && "opacity-50 cursor-not-allowed hover:bg-transparent hover:text-muted-foreground"
                )}
              >
                {showPassword && !isPlaceholder ? <EyeOff className="h-4.5 w-4.5" /> : <Eye className="h-4.5 w-4.5" />}
              </button>
            )}
          </div>
          
          {f.helpText && (
            <p className="text-[11px] text-muted-foreground leading-normal pl-1">
              {f.helpText}
            </p>
          )}
        </div>
      )
    }

    return (
      <div
        key={app.id}
        id={`tour-settings-card-${app.id}`}
        data-testid={`integration-card-${app.id}`}
        className="rounded-2xl border bg-card text-card-foreground shadow-sm overflow-hidden transition-all duration-200 border-border tour-settings-credential-card relative"
      >
        {/* Top Right Labels Badges */}
        {app.labels && app.labels.length > 0 && (
          <div className="absolute top-4 right-12 flex items-center gap-1.5 z-10 pointer-events-none">
            {app.labels.map((label) => (
              <span
                key={label}
                className="inline-flex items-center text-[10px] font-bold px-2.5 py-0.5 rounded-md border bg-primary/5 border-primary/10 text-primary/80 dark:text-primary/90 shrink-0 shadow-sm"
              >
                {label}
              </span>
            ))}
          </div>
        )}

        {/* Header Banner */}
        <div 
          onClick={() => setExpandedAppIds((prev) => ({ ...prev, [app.id]: !isExpanded }))}
          className="p-5 flex items-center justify-between gap-4 select-none cursor-pointer hover:bg-muted/5 transition-colors"
        >
          <div className="flex items-start gap-4 flex-1 min-w-0">
            <div className={cn(
              "flex h-10 w-10 shrink-0 items-center justify-center rounded-full transition-colors",
              isConfigured 
                ? "bg-primary/10 text-primary" 
                : "bg-slate-100 dark:bg-slate-800 text-slate-500"
            )}>
              <AppIcon className="h-[22px] w-[22px]" />
            </div>

            <div className="min-w-0 flex-1">
              <h3 className="text-base font-bold text-foreground leading-tight">
                {app.name}
              </h3>
              <p className="text-[11px] text-muted-foreground mt-1 leading-normal">
                {app.description}
              </p>
              
              <div className="mt-3">
                <span className={cn(
                  "inline-flex items-center gap-1 text-[10px] font-bold px-2.5 py-0.5 rounded-full border shrink-0",
                  !isEnabled
                    ? "bg-slate-100 dark:bg-slate-800/60 text-slate-500 border-slate-200 dark:border-slate-700/80"
                    : isConfigured
                    ? "bg-green-500/10 border-green-500/20 text-green-600 dark:text-green-400"
                    : "bg-amber-500/10 border-amber-500/20 text-amber-600 dark:text-amber-400"
                )}>
                  {!isEnabled ? (
                    <>
                      <MinusCircle className="h-3 w-3 shrink-0" />
                      Disabled
                    </>
                  ) : isConfigured ? (
                    <>
                      <CheckCircle2 className="h-3 w-3 shrink-0" />
                      Configured
                    </>
                  ) : (
                    <>
                      <Circle className="h-3 w-3 shrink-0" />
                      Not Configured
                    </>
                  )}
                </span>
              </div>
            </div>
          </div>

          <div className="shrink-0 p-1">
            {isExpanded ? (
              <ChevronUp className="h-5 w-5 text-muted-foreground" />
            ) : (
              <ChevronDown className="h-5 w-5 text-muted-foreground" />
            )}
          </div>
        </div>

        {/* Validation Status Banner */}
        {isExpanded && validation && (
          <div className={cn(
            "mx-5 p-3.5 rounded-xl border flex items-center gap-3 animate-slide-down text-xs leading-normal select-none mb-3",
            validation.isValid 
              ? "bg-green-500/10 border-green-500/20 text-green-700 dark:text-green-400"
              : "bg-destructive/10 border-destructive/20 text-destructive"
          )}>
            {validation.isValid ? (
              <>
                <CheckCircle2 className="h-4.5 w-4.5 shrink-0" />
                <span>Successfully verified connection to {app.name} API endpoints!</span>
              </>
            ) : (
              <>
                <AlertCircle className="h-4.5 w-4.5 shrink-0" />
                <span>Connection failed: {validation.error || "Invalid credential values."}</span>
              </>
            )}
          </div>
        )}

        {/* Expanded Edit Form */}
        {isExpanded && (
          <div className="p-5 border-t border-border bg-slate-50/50 dark:bg-slate-900/10 space-y-5 animate-slide-down">
            {!isEnabled ? (
              <div className="flex flex-col items-start gap-4 py-2 w-full">
                <div className="flex items-center gap-3 text-xs leading-normal text-muted-foreground select-none">
                  <Info className="h-5 w-5 text-muted-foreground shrink-0" />
                  <span className="italic">
                    {isAdmin 
                      ? "This integration is currently disabled for your organization." 
                      : "This integration is currently disabled by your administrator."}
                  </span>
                </div>
                {isAdmin && onEnableIntegration && (
                  <button
                    type="button"
                    onClick={() => onEnableIntegration(app.id)}
                    disabled={isPendingToggle}
                    className="inline-flex h-10 items-center justify-center gap-1.5 rounded-xl bg-primary px-5 text-xs font-semibold text-primary-foreground shadow hover:opacity-90 disabled:opacity-50 transition-colors"
                  >
                    {isPendingToggle ? (
                      <Loader2 className="h-4 w-4 animate-spin text-primary-foreground" />
                    ) : (
                      <Zap className="h-4 w-4" />
                    )}
                    Enable Integration
                  </button>
                )}
              </div>
            ) : (
              <>
                <div className="space-y-4">
                  {app.fields.filter(f => f.enabled).map(renderField)}
                </div>

                <div className="flex flex-wrap items-center justify-end gap-3 pt-4 border-t border-border/50 select-none w-full">
                  {/* Validate Link */}
                  {onValidateCredentials && (
                    <button
                      type="button"
                      onClick={() => onValidateCredentials(app.id)}
                      disabled={isValidating || isSaving || isRevoking || isPendingToggle}
                      className="inline-flex h-10 items-center justify-center gap-1.5 rounded-xl border border-primary bg-background px-4 text-xs font-semibold text-primary hover:bg-primary/5 disabled:opacity-50 transition-colors"
                    >
                      {isValidating ? (
                        <Loader2 className="h-4 w-4 animate-spin text-primary" />
                      ) : (
                        <Zap className="h-4 w-4" />
                      )}
                      Validate Link
                    </button>
                  )}

                  {/* Revoke (only if configured) */}
                  {isConfigured && onRevokeCredentials && (
                    <button
                      type="button"
                      onClick={() => onRevokeCredentials(app.id)}
                      disabled={isRevoking || isSaving || isValidating || isPendingToggle}
                      className="inline-flex h-10 items-center justify-center gap-1.5 rounded-xl border border-destructive bg-background px-4 text-xs font-semibold text-destructive hover:bg-destructive/5 disabled:opacity-50 transition-colors"
                    >
                      {isRevoking ? (
                        <Loader2 className="h-4 w-4 animate-spin text-destructive" />
                      ) : (
                        <Trash2 className="h-4 w-4" />
                      )}
                      Revoke
                    </button>
                  )}

                  {/* Save Credentials */}
                  {onSaveCredentials && (
                    <button
                      type="button"
                      onClick={() => onSaveCredentials(app.id)}
                      disabled={isSaving || isValidating || isRevoking || isPendingToggle}
                      className="inline-flex h-10 items-center justify-center gap-1.5 rounded-xl bg-primary px-4 text-xs font-semibold text-primary-foreground shadow hover:opacity-90 disabled:opacity-50 transition-colors"
                    >
                      {isSaving ? (
                        <Loader2 className="h-4 w-4 animate-spin text-primary-foreground" />
                      ) : (
                        <Save className="h-4 w-4" />
                      )}
                      Save Credentials
                    </button>
                  )}
                </div>
              </>
            )}
          </div>
        )}
      </div>
    )
  }

  // ================= RENDER METHOD: project-override =================
  const renderProjectOverrideCard = (app: AppIntegration) => {
    const setting = projectAppSettingsState[app.id]
    const isEnabled = setting?.enabled || false
    const isExpanded = expandedAppIds[app.id] || false
    const hasGlobal = !!globalCredentials?.[app.id]
    const isOverridden = setting?.isOverridden || false

    let badgeText = isNewProject ? "Not Configured" : "Not Enabled"
    let badgeColor = isNewProject 
      ? "bg-amber-500/10 border-amber-500/20 text-amber-600 dark:text-amber-400" 
      : "bg-muted/50 border-muted-foreground/10 text-muted-foreground"
    let BadgeIcon = Circle
    let avatarColor = "bg-muted text-muted-foreground"

    if (isEnabled) {
      avatarColor = "bg-primary/10 text-primary"
      if (!hasGlobal && !isOverridden) {
        badgeText = "Credentials Missing"
        badgeColor = "bg-amber-500/10 border-amber-500/30 text-amber-600 dark:text-amber-400"
        BadgeIcon = AlertCircle
      } else {
        badgeText = "Active & Connected"
        badgeColor = "bg-green-500/10 border-green-500/30 text-green-600 dark:text-green-400"
        BadgeIcon = CheckCircle2
      }
    }

    const AppIcon = getAppIcon(app.icon)

    return (
      <div
        key={app.id}
        className={`rounded-xl border bg-card text-card-foreground shadow-sm overflow-visible transition-all duration-200 ${
          isEnabled ? "border-border" : "border-border/60 opacity-80"
        }`}
      >
        {/* Banner header row */}
        <div 
          onClick={() => setExpandedAppIds((prev) => ({ ...prev, [app.id]: !isExpanded }))}
          className="p-4 flex items-center justify-between gap-4 select-none cursor-pointer hover:bg-muted/5 transition-colors"
        >
          <div className="flex items-center gap-3.5 flex-1 min-w-0">
            <div className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-full transition-colors ${avatarColor}`}>
              <AppIcon className="h-5 w-5" />
            </div>
            
            <div className="flex-1 min-w-0 flex flex-col gap-1.5">
              <h3 className="text-sm font-semibold text-foreground truncate">
                {app.name}
              </h3>
              <p className="text-[10px] text-muted-foreground leading-normal truncate">
                {app.description}
              </p>
              <div className="flex">
                <span className={`inline-flex items-center gap-1 text-[9px] font-bold uppercase border px-2 py-0.5 rounded-full ${badgeColor}`}>
                  <BadgeIcon className="h-2.5 w-2.5 shrink-0" />
                  {badgeText}
                </span>
              </div>
            </div>
          </div>

          {/* Action Checkbox Toggle */}
          <div className="flex items-center gap-4" onClick={(e) => e.stopPropagation()}>
            {onToggleProjectApp && (
              <input
                type="checkbox"
                id={app.id === "github" ? "tour-project-toggle-github" : undefined}
                checked={isEnabled}
                disabled={savingAppIds[app.id]}
                onChange={(e) => onToggleProjectApp(app.id, e.target.checked)}
                className="h-6 w-11 rounded-full bg-muted border border-border appearance-none cursor-pointer transition-colors relative checked:bg-primary before:content-[''] before:absolute before:h-4 before:w-4 before:rounded-full before:bg-card before:left-[3px] before:top-[3px] before:transition-transform checked:before:translate-x-5"
              />
            )}

            <button
              type="button"
              onClick={() => setExpandedAppIds((prev) => ({ ...prev, [app.id]: !isExpanded }))}
              className="flex h-7 w-7 items-center justify-center rounded hover:bg-muted text-muted-foreground"
            >
              {isExpanded ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
            </button>
          </div>
        </div>

        {/* Expanded body block */}
        {isExpanded && (
          <div className="p-5 border-t border-border bg-muted/20 space-y-4 animate-slide-down rounded-b-xl">
            {!isEnabled ? (
              <span className="text-xs text-muted-foreground leading-normal block">
                This integration is currently disabled for this project. Toggle the switch to activate it.
              </span>
            ) : (
              <>
                {/* CASE 1: Global Keys Missing & Not Overridden */}
                {!hasGlobal && !isOverridden ? (
                  <div className="flex items-start gap-3 p-3 rounded-lg border border-amber-500/20 bg-amber-500/5">
                    <Link2Off className="h-5 w-5 text-amber-500 shrink-0 mt-0.5" />
                    <div className="flex-1 space-y-2">
                      <h4 className="text-xs font-bold text-foreground">Global Credential Missing</h4>
                      <p className="text-[11px] text-muted-foreground leading-relaxed">
                        ⚠️ Warning: No master credentials configured globally. You can configure global credentials in Settings or enter custom credentials below to override for this project.
                      </p>
                      <div className="flex flex-wrap gap-2 pt-1">
                        <button
                          type="button"
                          onClick={() => router.push("/settings")}
                          className="inline-flex h-8 items-center gap-1.5 rounded-md border border-primary bg-background px-3 text-xs font-semibold text-primary hover:bg-primary/5"
                        >
                          <Settings className="h-3.5 w-3.5" />
                          Configure in Settings Now
                        </button>
                        {onOverrideProjectCredentials && (
                          <button
                            type="button"
                            onClick={() => {
                              onOverrideProjectCredentials(app.id)
                              setTimeout(() => {
                                document.getElementById(`${app.id}-pat`)?.focus()
                              }, 50)
                            }}
                            className="inline-flex h-8 items-center gap-1.5 rounded-md bg-primary px-3 text-xs font-semibold text-primary-foreground shadow hover:opacity-90"
                          >
                            <Edit3 className="h-3.5 w-3.5" />
                            Override Scoped Credentials Now
                          </button>
                        )}
                      </div>
                    </div>
                  </div>
                ) : (
                  /* CASE 2: Connected - fields lists available */
                  <div className="space-y-4">
                    {isOverridden && (
                      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        {setting?.fields?.map((f) => {
                          if (!f.enabled) return null
                          const isEditable = !savingAppIds[app.id]

                          return (
                            <div key={f.name} className="space-y-1.5">
                              <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider" htmlFor={`${app.id}-${f.name}`}>
                                {f.displayName}
                              </label>
                              <div className="relative">
                                <div className="absolute left-3 top-1/2 -translate-y-1/2">
                                  {getFieldIcon(f.name)}
                                </div>
                                <input
                                  id={`${app.id}-${f.name}`}
                                  type={f.isSecret ? "password" : "text"}
                                  disabled={!isEditable}
                                  placeholder={f.isSecret ? "" : f.helpText || `Enter ${f.displayName.toLowerCase()}`}
                                  value={f.value || ""}
                                  onBlur={() => {
                                    if (f.name === "pat" && !f.value?.trim() && hasGlobal && onRevertProjectToMaster) {
                                      onRevertProjectToMaster(app.id)
                                    }
                                  }}
                                  onChange={(e) => onProjectFieldChange?.(app.id, f.name, e.target.value)}
                                  className="w-full h-9 pl-9 pr-3 rounded border border-input bg-card text-xs focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary disabled:opacity-75 disabled:bg-muted/30"
                                />
                              </div>
                              {f.helpText && isEditable && (
                                <p className="text-[10px] text-muted-foreground">{f.helpText}</p>
                              )}
                            </div>
                          )
                        })}
                      </div>
                    )}

                    {/* Overridden vs Inherited footer banners/controls */}
                    {!isOverridden ? (
                      <div className="space-y-3 pt-2">
                        <div className="flex items-start gap-2">
                          <Info className="h-4 w-4 text-blue-500 shrink-0 mt-0.5" />
                          <p className="text-xs text-muted-foreground leading-normal">
                            Using inherited credentials from your master profile globally. Changes to master credentials will apply automatically.
                          </p>
                        </div>
                        {onOverrideProjectCredentials && (
                          <button
                            type="button"
                            disabled={savingAppIds[app.id]}
                            onClick={() => {
                              onOverrideProjectCredentials(app.id)
                              setTimeout(() => {
                                document.getElementById(`${app.id}-pat`)?.focus()
                              }, 50)
                            }}
                            className="inline-flex h-8 items-center gap-1.5 rounded-md border border-input bg-background px-3 text-xs font-semibold text-foreground hover:bg-muted"
                          >
                            <Edit3 className="h-3.5 w-3.5 text-muted-foreground" />
                            Override profile defaults
                          </button>
                        )}
                      </div>
                    ) : (
                      hasGlobal && onRevertProjectToMaster ? (
                        <div className="flex items-center justify-start gap-2.5 pt-2">
                          {savingAppIds[app.id] && (
                            <Loader2 className="h-4 w-4 animate-spin text-primary" />
                          )}
                          <button
                            type="button"
                            disabled={savingAppIds[app.id]}
                            onClick={() => onRevertProjectToMaster(app.id)}
                            className="inline-flex h-8 items-center gap-1.5 rounded-md text-xs font-semibold text-destructive hover:bg-destructive/5 px-3"
                          >
                            <Undo2 className="h-3.5 w-3.5" />
                            Revert to profile defaults
                          </button>
                        </div>
                      ) : null
                    )}
                    
                    {/* GitHub specific project configs (Repo URL, etc.) */}
                    {app.id === "github" && githubSettings && (
                      <div className="border-t border-border/50 pt-4 mt-4">
                        <GitHubIntegrationForm
                          minimal
                          projectId={projectId}
                          repoUrl={githubSettings.repoUrl}
                          setRepoUrl={githubSettings.setRepoUrl}
                          specLocation={githubSettings.specLocation}
                          setSpecLocation={githubSettings.setSpecLocation}
                          baseBranch={githubSettings.baseBranch}
                          setBaseBranch={githubSettings.setBaseBranch}
                          specBranchPrefix={githubSettings.specBranchPrefix}
                          setSpecBranchPrefix={githubSettings.setSpecBranchPrefix}
                          email={githubSettings.email}
                          setEmail={githubSettings.setEmail}
                          enabled={isEnabled}
                          hasCredentials={hasGlobal || isOverridden}
                          customToken={projectAppSettingsState["github"]?.fields?.find((f) => f.name === "pat")?.value || ""}
                          isOverridden={isOverridden}
                        />
                      </div>
                    )}
                  </div>
                )}
              </>
            )}
          </div>
        )}
      </div>
    )
  }

  // ================= MAIN RENDER =================
  return (
    <div className="space-y-4">
      {/* Dynamic Filter Chips (Only for admin-catalog and user-settings modes) */}
      {mode !== "project-override" && distinctLabels.length > 1 && (
        <div className="flex flex-wrap gap-2 pb-2 select-none animate-fade-in">
          {distinctLabels.map((label) => {
            const isSelected = selectedLabel === label
            return (
              <button
                key={label}
                type="button"
                onClick={() => setSelectedLabel(label)}
                className={cn(
                  "inline-flex items-center justify-center px-3.5 py-1.5 rounded-full text-xs font-semibold border transition-all duration-200 shadow-sm cursor-pointer",
                  isSelected
                    ? "bg-primary border-primary text-primary-foreground font-bold scale-[1.02]"
                    : "bg-background border-border text-muted-foreground hover:text-foreground hover:bg-muted/40"
                )}
              >
                {label}
              </button>
            )
          })}
        </div>
      )}

      {/* List / Grid Container */}
      <div className={cn(
        "w-full",
        mode === "admin-catalog" 
          ? "space-y-4" 
          : mode === "project-override"
          ? "space-y-3.5"
          : "space-y-4"
      )}>
        {filteredApps.map((app) => {
          switch (mode) {
            case "admin-catalog":
              return renderAdminCatalogCard(app)
            case "user-settings":
              return renderUserSettingsCard(app)
            case "project-override":
              return renderProjectOverrideCard(app)
            default:
              return null
          }
        })}
      </div>

      {/* Reusable Premium Alert/Confirm Modal for Disabling */}
      {confirmingApp && (
        <AlertModal
          type="warning"
          title={`Disable ${confirmingApp.name}?`}
          description={`Are you sure you want to disable ${confirmingApp.name}? Team members will no longer be able to use it, but their credentials will be preserved if you re-enable it later.`}
          onClose={() => setConfirmingApp(null)}
          action={{
            label: "Disable Integration",
            onClick: () => {
              if (onDisableIntegration) {
                onDisableIntegration(confirmingApp.id)
              }
              setConfirmingApp(null)
            }
          }}
        />
      )}
    </div>
  )
}
