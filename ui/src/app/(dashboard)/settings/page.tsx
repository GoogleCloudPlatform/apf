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
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { useRouter } from "next/navigation"
import { toast } from "sonner"
import { cn } from "@/lib/utils"
import { 
  Palette, 
  Key, 
  Moon, 
  Sun, 
  Trash2, 
  CheckCircle2,
  Lock,
  Eye,
  EyeOff,
  Loader2,
  Info,
  User,
  Sliders,
  ChevronDown,
  ChevronUp,
  MinusCircle,
  Circle,
  Zap,
  Save,
  AlertCircle,
  Code,
  Brush,
  Network,
  Layers,
  BarChart2,
  Cable,
  Github,
  Figma,
  RefreshCw,
  Gitlab,
  Trello,
  ListTodo
} from "lucide-react"
import { useAppStore } from "@/stores/app"
import { useAuthStore } from "@/stores/auth"
import { useThemeStore } from "@/stores/theme"
import { api } from "@/lib/api"
import { AppIntegration, AppCredential, CredentialAttribute } from "@/lib/types"
import { useOnboardingStore } from "@/stores/onboarding"
import AppIntegrationsList from "@/components/shared/AppIntegrationsList"
import ConfirmDialog from "@/components/shared/ConfirmDialog"

const CREDENTIAL_MASK_PLACEHOLDER = "••••••••••••••••••••••••••••••••"

export default function SettingsPage() {
  const queryClient = useQueryClient()
  const router = useRouter()
  const { setPageTitle } = useAppStore()
  const { email, selectedOrgId, isAdmin } = useAuthStore()
  const { themeMode, setThemeMode } = useThemeStore()

  // Refs for scrolling to sections
  const preferencesRef = useRef<HTMLDivElement>(null)
  const credentialsRef = useRef<HTMLDivElement>(null)
  const templatesRef = useRef<HTMLDivElement>(null)

  // Active section tracking
  const [activeSection, setActiveSection] = useState<"appearance" | "credentials" | "templates">("appearance")

  // Expand and forms fields states
  const [expandedAppIds, setExpandedAppIds] = useState<Record<string, boolean>>({})
  const [formValues, setFormValues] = useState<Record<string, Record<string, string>>>({})
  const [showPasswordMap, setShowPasswordMap] = useState<Record<string, boolean>>({})
  const [validationStates, setValidationStates] = useState<Record<string, { isValid: boolean; error?: string } | null>>({})
  const [selectedLabel, setSelectedLabel] = useState<string>("All")
  const [isResetConfirmOpen, setIsResetConfirmOpen] = useState(false)
  const [showcaseProjectIdToReset, setShowcaseProjectIdToReset] = useState<string | null>(null)
  const [isRevokeConfirmOpen, setIsRevokeConfirmOpen] = useState(false)
  const [appIdToRevoke, setAppIdToRevoke] = useState<string | null>(null)

  // Sync dyn header title
  useEffect(() => {
    setPageTitle("Settings")
  }, [setPageTitle])

  // Query: Fetch user's projects to check if showcase is provisioned
  const {
    data: projectsData,
    isLoading: isLoadingProjects,
    refetch: refetchProjects,
  } = useQuery({
    queryKey: ["user-projects"],
    queryFn: async () => {
      const res = await api.get("/api/v1/projects")
      if (!res.ok) throw new Error("Failed to load projects")
      return res.json()
    },
  })

  const showcaseProject = projectsData?.myProjects?.find(
    (p: any) => p.id === "foundry-showcase-project" || p.name === "Foundry Showcase"
  )
  const isShowcaseProvisioned = !!showcaseProject

  // Mutation: Provision showcase project
  const provisionShowcaseMutation = useMutation({
    mutationFn: async () => {
      const res = await api.post("/api/v1/projects/provision-showcase", {})
      if (!res.ok) throw new Error("Failed to provision showcase project")
      return res.json()
    },
    onSuccess: () => {
      toast.success("Showcase project provisioned successfully!")
      queryClient.invalidateQueries({ queryKey: ["user-projects"] })
      router.push("/")
    },
    onError: (err: any) => {
      toast.error(`Provisioning failed: ${err.message}`)
    },
  })

  // Mutation: Reset showcase project
  const resetShowcaseMutation = useMutation({
    mutationFn: async (projectId: string) => {
      const res = await api.post(`/api/v1/projects/${projectId}/reset`, {})
      if (!res.ok) throw new Error("Failed to reset showcase project")
      return res.json()
    },
    onSuccess: () => {
      toast.success("Showcase project restored to default state!")
      queryClient.invalidateQueries({ queryKey: ["user-projects"] })
      router.push("/")
    },
    onError: (err: any) => {
      toast.error(`Reset failed: ${err.message}`)
    },
  })

  const handleProvisionShowcase = () => {
    provisionShowcaseMutation.mutate()
  }

  const handleResetShowcase = (projectId: string) => {
    setShowcaseProjectIdToReset(projectId)
    setIsResetConfirmOpen(true)
  }

  const handleConfirmReset = () => {
    if (showcaseProjectIdToReset) {
      resetShowcaseMutation.mutate(showcaseProjectIdToReset, {
        onSettled: () => {
          setIsResetConfirmOpen(false)
          setShowcaseProjectIdToReset(null)
        }
      })
    }
  }

  const isProvisioning = provisionShowcaseMutation.isPending
  const isResetting = resetShowcaseMutation.isPending
  const isPending = isProvisioning || isResetting || isLoadingProjects

  // 1. Query: Fetch standard dynamic list of available workspace integrations
  const {
    data: integrations = [],
    isLoading: isLoadingApps,
    error: appsError,
  } = useQuery<AppIntegration[]>({
    queryKey: ["app-integrations"],
    queryFn: async () => {
      const res = await api.get("/api/v1/config/app-integrations")
      if (!res.ok) throw new Error("Failed to load integrations catalog")
      return res.json()
    },
  })

  // Extract distinct labels from all app_integrations dynamically for visual filtering
  const distinctLabels = [
    "All",
    ...Array.from(new Set(integrations.flatMap((app) => app.labels || [])))
  ]

  // 2. Query: Fetch user's global master keys for each tools in parallel
  const {
    data: globalCredentials,
    isLoading: isLoadingCreds,
    refetch: refetchGlobalCreds,
  } = useQuery<Record<string, AppCredential>>({
    queryKey: ["global-credentials", email],
    queryFn: async () => {
      if (!email) return {}
      const activeApps = (integrations || []).filter((app) => app.isEnabled)
      const credsMap: Record<string, AppCredential> = {}

      await Promise.all(
        activeApps.map(async (app) => {
          try {
            const res = await api.get(`/api/v1/users/${encodeURIComponent(email)}/apps/${app.id}/credentials`)
            if (res.ok) {
              credsMap[app.id] = await res.json()
            }
          } catch (_) {
            // Suppress empty records warnings
          }
        })
      )
      return credsMap
    },
    enabled: !!email && integrations.length > 0,
  })

  // Pre-populate input values from database once loaded
  useEffect(() => {
    if (integrations.length > 0 && (globalCredentials || !isLoadingCreds)) {
      const values: Record<string, Record<string, string>> = {}
      integrations.forEach((app) => {
        const cred = globalCredentials?.[app.id]
        const appValues: Record<string, string> = {}
        
        app.fields.forEach((f) => {
          // Prepopulate from saved user credentials if exists
          const savedField = cred?.fields.find((sf) => sf.name === f.name)
          appValues[f.name] = savedField?.value || ""
        })
        values[app.id] = appValues
      })
      setFormValues(values)

      // If GitHub credentials exist with a PAT but missing email, auto-fetch profile email
      const githubCred = globalCredentials?.["github"]
      const githubPat = githubCred?.fields.find((f) => f.name === "pat")?.value
      const githubEmail = githubCred?.fields.find((f) => f.name === "email")?.value
      if (githubPat && !githubEmail) {
        fetchGitHubProfileEmail(githubPat)
      }
    }
  }, [integrations, globalCredentials, isLoadingCreds])

  // Mutation: Save Master user credentials for a tool
  const saveCredsMutation = useMutation({
    mutationFn: async ({ appId, payload }: { appId: string; payload: AppCredential }) => {
      if (!email) return
      const url = `/api/v1/users/${encodeURIComponent(email)}/apps/${appId}/credentials`
      const res = await api.put(url, payload)
      if (!res.ok) {
        let errData
        try {
          errData = await res.json()
        } catch (_) {}
        throw new Error(errData?.message || errData?.error || "Failed to save credentials")
      }
      return res.json()
    },
    onSuccess: (_, variables) => {
      toast.success(`Successfully configured credentials for ${variables.appId}!`)
      setExpandedAppIds((prev) => ({ ...prev, [variables.appId]: false }))
      queryClient.invalidateQueries({ queryKey: ["global-credentials", email] })
    },
    onError: (err: any) => {
      console.error("Save credentials error:", err)
      toast.error(err?.message || "Failed to save credentials to your profile.")
    },
  })

  // Mutation: Revoke / Delete user master credentials for a tool
  const revokeCredsMutation = useMutation({
    mutationFn: async (appId: string) => {
      if (!email) return
      const url = `/api/v1/users/${encodeURIComponent(email)}/apps/${appId}/credentials`
      const res = await api.delete(url)
      if (!res.ok) throw new Error("Failed to revoke credentials")
      return appId
    },
    onSuccess: (appId) => {
      if (!appId) return
      toast.success(`Revoked credentials for ${appId}!`)
      
      // Clear values in form states
      setFormValues((prev) => {
        const cleared: Record<string, string> = {}
        const app = integrations.find((a) => a.id === appId)
        app?.fields.forEach((f) => { cleared[f.name] = "" })
        return { ...prev, [appId]: cleared }
      })
      setValidationStates((prev) => ({ ...prev, [appId]: null }))
      queryClient.invalidateQueries({ queryKey: ["global-credentials", email] })
    },
    onError: (err) => {
      console.error("Revoke error:", err)
      toast.error("Failed to delete credentials from profile.")
    },
  })

  // Mutation: Validate / Test connection keys
  const validateCredsMutation = useMutation({
    mutationFn: async ({ appId, payload }: { appId: string; payload: AppCredential }) => {
      if (!email) return
      const url = `/api/v1/users/${encodeURIComponent(email)}/apps/${appId}/credentials/validate`
      const res = await api.post(url, payload)
      
      if (!res.ok) {
        let errorData
        try { errorData = await res.json() } catch (_) {}
        throw new Error(errorData?.error || `Failed: Status ${res.status}`)
      }
      return res.json()
    },
    onSuccess: (data, variables) => {
      const isValid = data?.valid === true
      const errorMsg = data?.error
      
      setValidationStates((prev) => ({
        ...prev,
        [variables.appId]: { isValid, error: errorMsg },
      }))

      if (variables.appId === "github" && isValid) {
        const fetchedEmail = (data as any)?.email
        if (fetchedEmail) {
          setFormValues((prev) => ({
            ...prev,
            github: {
              ...(prev.github || {}),
              email: prev.github?.email || fetchedEmail,
            },
          }))
        } else {
          const patVal = variables.payload.fields.find((f) => f.name === "pat")?.value || formValues.github?.pat
          if (!formValues.github?.email) {
            fetchGitHubProfileEmail(patVal)
          }
        }
      }
    },
    onError: (err: any, variables) => {
      setValidationStates((prev) => ({
        ...prev,
        [variables.appId]: { isValid: false, error: err.message },
      }))
    },
  })

  // Mutation: Enable an integration (Admin only)
  const enableIntegrationMutation = useMutation({
    mutationFn: async (appId: string) => {
      if (!selectedOrgId) return
      const url = `/api/v1/admin/organizations/${selectedOrgId}/config/app-integrations/${appId}/enable`
      const res = await api.post(url, {})
      if (!res.ok) throw new Error("Failed to enable integration")
      return appId
    },
    onSuccess: (appId) => {
      if (!appId) return
      toast.success(`Successfully enabled integration!`)
      queryClient.invalidateQueries({ queryKey: ["app-integrations"] })
    },
    onError: (err) => {
      console.error("Enable integration error:", err)
      toast.error("Failed to enable integration.")
    },
  })

  // Mutation: Disable an integration (Admin only)
  const disableIntegrationMutation = useMutation({
    mutationFn: async (appId: string) => {
      if (!selectedOrgId) return
      const url = `/api/v1/admin/organizations/${selectedOrgId}/config/app-integrations/${appId}/disable`
      const res = await api.post(url, {})
      if (!res.ok) throw new Error("Failed to disable integration")
      return appId
    },
    onSuccess: (appId) => {
      if (!appId) return
      toast.success(`Successfully disabled integration!`)
      queryClient.invalidateQueries({ queryKey: ["app-integrations"] })
      if (email) {
        queryClient.invalidateQueries({ queryKey: ["global-credentials", email] })
      }
    },
    onError: (err) => {
      console.error("Disable integration error:", err)
      toast.error("Failed to disable integration.")
    },
  })

  const fetchGitHubProfileEmail = async (token?: string) => {
    try {
      const headers: Record<string, string> = {}
      const hasRealToken = token && token !== CREDENTIAL_MASK_PLACEHOLDER && !/[•·]/.test(token) && token.trim() !== ""
      if (hasRealToken) {
        headers["X-GitHub-Token"] = token
      }
      const res = await api.get("/api/v1/integrations/github/profile", Object.keys(headers).length > 0 ? { headers } : undefined)
      if (res.ok) {
        const data = await res.json()
        if (data.email) {
          setFormValues((prev) => ({
            ...prev,
            github: {
              ...(prev.github || {}),
              email: prev.github?.email || data.email,
            },
          }))
        }
      }
    } catch (err) {
      console.error("Failed to fetch GitHub profile email in settings:", err)
    }
  }

  const handleSave = (appId: string) => {
    const app = integrations.find((a) => a.id === appId)
    if (!app || !email) return

    // Onboarding tour hook
    if (appId === "github") {
      useOnboardingStore.getState().advanceStep("catalog", 9)
      useOnboardingStore.getState().advanceStep("settings", 6)
    }

    const currentValues = formValues[appId] || {}
    const fieldsPayload: CredentialAttribute[] = app.fields
      .filter((f) => f.enabled)
      .map((f) => ({
        ...f,
        value: currentValues[f.name] || "",
      }))

    // Validate required fields
    for (const field of fieldsPayload) {
      if (field.required && !field.value?.trim()) {
        toast.error(`${field.displayName || field.name} is required.`)
        return
      }
    }

    const payload: AppCredential = {
      integrationId: appId,
      fields: fieldsPayload,
    }

    saveCredsMutation.mutate({ appId, payload })
  }

  const handleRevoke = (appId: string) => {
    setAppIdToRevoke(appId)
    setIsRevokeConfirmOpen(true)
  }

  const handleConfirmRevoke = () => {
    if (appIdToRevoke) {
      revokeCredsMutation.mutate(appIdToRevoke, {
        onSettled: () => {
          setIsRevokeConfirmOpen(false)
          setAppIdToRevoke(null)
        }
      })
    }
  }

  const handleValidate = (appId: string) => {
    const app = integrations.find((a) => a.id === appId)
    if (!app || !email) return

    // Onboarding tour hook
    if (appId === "github") {
      useOnboardingStore.getState().advanceStep("catalog", 8)
      useOnboardingStore.getState().advanceStep("settings", 5)
    }

    const currentValues = formValues[appId] || {}
    const fieldsPayload: CredentialAttribute[] = app.fields
      .filter((f) => f.enabled)
      .map((f) => ({
        ...f,
        value: currentValues[f.name] || "",
      }))

    const payload: AppCredential = {
      integrationId: appId,
      fields: fieldsPayload,
    }

    validateCredsMutation.mutate({ appId, payload })
  }

  const handleInputChange = (appId: string, fieldName: string, val: string) => {
    setFormValues((prev) => ({
      ...prev,
      [appId]: {
        ...(prev[appId] || {}),
        [fieldName]: val,
      },
    }))
    // Clear validation status state on edits
    if (validationStates[appId]) {
      setValidationStates((prev) => ({ ...prev, [appId]: null }))
    }

    // If entering PAT for GitHub and email is empty, auto-detect profile email
    if (appId === "github" && fieldName === "pat" && val && val !== CREDENTIAL_MASK_PLACEHOLDER) {
      if (!formValues.github?.email) {
        fetchGitHubProfileEmail(val)
      }
    }
  }

  const togglePasswordVisibility = (appId: string, fieldName: string) => {
    const key = `${appId}-${fieldName}`
    setShowPasswordMap((prev) => ({ ...prev, [key]: !prev[key] }))
  }

  const scrollToSection = (section: "appearance" | "credentials" | "templates") => {
    setActiveSection(section)
    const ref = section === "appearance" ? preferencesRef : section === "credentials" ? credentialsRef : templatesRef
    if (ref.current) {
      ref.current.scrollIntoView({ behavior: "smooth", block: "start" })
    }
  }

  const isGlobalLoading = isLoadingApps || isLoadingCreds

  if (isGlobalLoading) {
    return (
      <div className="flex-1 flex h-[400px] w-full flex-col items-center justify-center gap-3 text-muted-foreground select-none">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
        <span className="text-sm font-medium">Loading settings...</span>
      </div>
    )
  }

  if (appsError) {
    return (
      <div className="flex-1 flex items-center justify-center p-8 min-h-[400px]">
        <div className="w-full max-w-md bg-destructive/10 border border-destructive/20 rounded-2xl p-6 text-center select-none space-y-4">
          <div className="flex justify-center text-destructive">
            <AlertCircle className="h-12 w-12" />
          </div>
          <h3 className="text-lg font-bold text-destructive">
            Failed to load settings
          </h3>
          <p className="text-sm text-muted-foreground leading-relaxed">
            {appsError instanceof Error ? appsError.message : "An unknown error occurred while loading settings parameters."}
          </p>
          <button
            onClick={() => {
              queryClient.refetchQueries({ queryKey: ["app-integrations"] })
              if (email) {
                queryClient.refetchQueries({ queryKey: ["global-credentials", email] })
              }
            }}
            className="inline-flex h-10 items-center justify-center gap-2 rounded-xl bg-destructive px-5 text-sm font-semibold text-destructive-foreground shadow hover:opacity-90 transition-colors"
          >
            <RefreshCw className="h-4 w-4" />
            Retry Loading Settings
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="flex flex-col md:flex-row h-full w-full min-h-[calc(100vh-4rem)]">
      
      {/* 1. Left Sidebar Split Tab Links Column */}
      <aside className="w-full md:w-[260px] border-b md:border-b-0 md:border-r border-border bg-slate-100/50 dark:bg-slate-900/40 py-6 shrink-0 select-none">
        <nav id="tour-settings-nav" className="space-y-3">
          <div className="px-3">
            <button
              onClick={() => scrollToSection("appearance")}
              className={cn(
                "flex items-center w-full px-4 py-3.5 gap-3 rounded-xl text-left transition-all duration-200 border relative",
                activeSection === "appearance"
                  ? "bg-primary/5 border-primary/20 text-primary font-bold"
                  : "bg-transparent border-transparent text-muted-foreground hover:bg-muted/50 hover:text-foreground"
              )}
            >
              {/* Active stripe indicator */}
              <span className={cn(
                "w-1 h-6 rounded shrink-0 transition-colors duration-200",
                activeSection === "appearance" ? "bg-primary" : "bg-transparent"
              )} />
              
              <Palette className={cn(
                "h-5 w-5 shrink-0 transition-colors",
                activeSection === "appearance" ? "text-primary" : "text-muted-foreground"
              )} />
              
              <div className="min-w-0 flex-1">
                <span className={cn(
                  "block text-[13px] leading-tight transition-colors",
                  activeSection === "appearance" ? "font-bold text-primary" : "font-normal text-foreground"
                )}>
                  Visual Preferences
                </span>
                <span className="block text-[10px] font-normal text-muted-foreground truncate mt-0.5">
                  App workspace styling options
                </span>
              </div>
            </button>
          </div>

          <div className="px-3">
            <button
              onClick={() => scrollToSection("credentials")}
              className={cn(
                "flex items-center w-full px-4 py-3.5 gap-3 rounded-xl text-left transition-all duration-200 border relative",
                activeSection === "credentials"
                  ? "bg-primary/5 border-primary/20 text-primary font-bold"
                  : "bg-transparent border-transparent text-muted-foreground hover:bg-muted/50 hover:text-foreground"
              )}
            >
              {/* Active stripe indicator */}
              <span className={cn(
                "w-1 h-6 rounded shrink-0 transition-colors duration-200",
                activeSection === "credentials" ? "bg-primary" : "bg-transparent"
              )} />
              
              <Key className={cn(
                "h-5 w-5 shrink-0 transition-colors",
                activeSection === "credentials" ? "text-primary" : "text-muted-foreground"
              )} />
              
              <div className="min-w-0 flex-1">
                <span className={cn(
                  "block text-[13px] leading-tight transition-colors",
                  activeSection === "credentials" ? "font-bold text-primary" : "font-normal text-foreground"
                )}>
                  Application Credentials
                </span>
                <span className="block text-[10px] font-normal text-muted-foreground truncate mt-0.5">
                  Configure secure global keys
                </span>
              </div>
            </button>
          </div>

          <div className="px-3">
            <button
              onClick={() => scrollToSection("templates")}
              className={cn(
                "flex items-center w-full px-4 py-3.5 gap-3 rounded-xl text-left transition-all duration-200 border relative",
                activeSection === "templates"
                  ? "bg-primary/5 border-primary/20 text-primary font-bold"
                  : "bg-transparent border-transparent text-muted-foreground hover:bg-muted/50 hover:text-foreground"
              )}
            >
              {/* Active stripe indicator */}
              <span className={cn(
                "w-1 h-6 rounded shrink-0 transition-colors duration-200",
                activeSection === "templates" ? "bg-primary" : "bg-transparent"
              )} />
              
              <Layers className={cn(
                "h-5 w-5 shrink-0 transition-colors",
                activeSection === "templates" ? "text-primary" : "text-muted-foreground"
              )} />
              
              <div className="min-w-0 flex-1">
                <span className={cn(
                  "block text-[13px] leading-tight transition-colors",
                  activeSection === "templates" ? "font-bold text-primary" : "font-normal text-foreground"
                )}>
                  Project Templates
                </span>
                <span className="block text-[10px] font-normal text-muted-foreground truncate mt-0.5">
                  Provision showcase projects
                </span>
              </div>
            </button>
          </div>

        </nav>
      </aside>

      <main className="flex-1 overflow-y-auto p-8">
        <div className="w-full max-w-[750px] space-y-12 pb-[calc(100vh-16rem)]">
          
          {/* ================= SECTION A: APPEARANCE PREFERENCES ================= */}
          <div id="tour-settings-appearance" ref={preferencesRef} className="space-y-6 scroll-mt-10 select-none">
            <div className="space-y-2 border-b border-border pb-4">
              <div className="flex items-center gap-3">
                <Palette className="h-6 w-6 text-primary" />
                <h2 className="text-base font-bold text-foreground">Visual Preferences</h2>
              </div>
              <p className="text-[11px] text-muted-foreground leading-relaxed">
                Customize your workspace environment settings. Theme selections are persisted dynamically under your user settings profiles.
              </p>
            </div>

            <div className="space-y-3">
              {/* Premium Dark Option */}
              <button
                type="button"
                onClick={() => setThemeMode("dark")}
                className={cn(
                  "flex items-center justify-between w-full p-4 border rounded-xl text-left transition-all duration-200 shadow-sm",
                  themeMode === "dark"
                    ? "border-2 border-primary bg-primary/5 text-foreground font-bold"
                    : "border-border bg-card text-muted-foreground hover:bg-muted/20"
                )}
              >
                <div className="flex items-center gap-4 flex-1 min-w-0">
                  <div className={cn(
                    "flex h-10 w-10 shrink-0 items-center justify-center rounded-full transition-colors",
                    themeMode === "dark" ? "bg-primary text-primary-foreground" : "bg-slate-100 dark:bg-slate-800 text-muted-foreground"
                  )}>
                    <Moon className="h-5 w-5" />
                  </div>
                  <div className="min-w-0 flex-1">
                    <span className={cn(
                      "text-sm font-bold text-foreground block",
                      themeMode === "dark" ? "text-primary" : "text-foreground"
                    )}>
                      Premium Dark Mode
                    </span>
                    <span className="text-[11px] text-muted-foreground leading-normal block mt-1">
                      Optimized workspace colors to emphasize code details and relieve eye strain.
                    </span>
                  </div>
                </div>
                
                {/* Radio Indicator */}
                <div className={cn(
                  "flex h-4.5 w-4.5 rounded-full border items-center justify-center transition-colors shrink-0",
                  themeMode === "dark" ? "border-primary" : "border-muted-foreground/35"
                )}>
                  {themeMode === "dark" && <div className="h-2.5 w-2.5 rounded-full bg-primary" />}
                </div>
              </button>

              {/* Sleek Light Option */}
              <button
                type="button"
                onClick={() => setThemeMode("light")}
                className={cn(
                  "flex items-center justify-between w-full p-4 border rounded-xl text-left transition-all duration-200 shadow-sm",
                  themeMode === "light"
                    ? "border-2 border-primary bg-primary/5 text-foreground font-bold"
                    : "border-border bg-card text-muted-foreground hover:bg-muted/20"
                )}
              >
                <div className="flex items-center gap-4 flex-1 min-w-0">
                  <div className={cn(
                    "flex h-10 w-10 shrink-0 items-center justify-center rounded-full transition-colors",
                    themeMode === "light" ? "bg-primary text-primary-foreground" : "bg-slate-100 dark:bg-slate-800 text-muted-foreground"
                  )}>
                    <Sun className="h-5 w-5" />
                  </div>
                  <div className="min-w-0 flex-1">
                    <span className={cn(
                      "text-sm font-bold text-foreground block",
                      themeMode === "light" ? "text-primary" : "text-foreground"
                    )}>
                      Sleek Light Mode
                    </span>
                    <span className="text-[11px] text-muted-foreground leading-normal block mt-1">
                      High contrast visual templates focusing on high-daylight clarity workspaces.
                    </span>
                  </div>
                </div>
                
                {/* Radio Indicator */}
                <div className={cn(
                  "flex h-4.5 w-4.5 rounded-full border items-center justify-center transition-colors shrink-0",
                  themeMode === "light" ? "border-primary" : "border-muted-foreground/35"
                )}>
                  {themeMode === "light" && <div className="h-2.5 w-2.5 rounded-full bg-primary" />}
                </div>
              </button>
            </div>
          </div>

          <hr className="border-border" />

          {/* ================= SECTION B: GLOBAL TOOLS CREDENTIALS ================= */}
          <div id="tour-settings-credentials" ref={credentialsRef} className="space-y-6 scroll-mt-10">
            <div className="space-y-2 border-b border-border pb-4 select-none">
              <div className="flex items-center gap-3">
                <Key className="h-6 w-6 text-primary" />
                <h2 className="text-base font-bold text-foreground">Application Credentials</h2>
              </div>
              <p className="text-[11px] text-muted-foreground leading-relaxed">
                Configure your personal developer access keys and platform tokens globally. When project tools are enabled, they link to these master values securely.
              </p>
            </div>

            {integrations.length === 0 ? (
              <div className="flex h-[200px] flex-col items-center justify-center text-center gap-2.5 text-muted-foreground border border-dashed border-border rounded-2xl select-none mt-6">
                <Cable className="h-10 w-10 opacity-30" />
                <span className="text-sm font-medium">No integrations are enabled in the foundry workspace configuration.</span>
              </div>
            ) : (
              <AppIntegrationsList
                mode="user-settings"
                integrations={integrations}
                globalCredentials={globalCredentials}
                expandedAppIds={expandedAppIds}
                setExpandedAppIds={setExpandedAppIds}
                isAdmin={isAdmin}
                
                // Admin toggle handlers
                onEnableIntegration={async (id) => { await enableIntegrationMutation.mutateAsync(id) }}
                onDisableIntegration={async (id) => { await disableIntegrationMutation.mutateAsync(id) }}
                isPendingEnableDisable={enableIntegrationMutation.isPending || disableIntegrationMutation.isPending}
                pendingAppId={enableIntegrationMutation.variables || disableIntegrationMutation.variables || saveCredsMutation.variables?.appId || revokeCredsMutation.variables || validateCredsMutation.variables?.appId}
                
                // Credentials form state & handlers
                formValues={formValues}
                onInputChange={handleInputChange}
                showPasswordMap={showPasswordMap}
                onTogglePasswordVisibility={togglePasswordVisibility}
                validationStates={validationStates}
                onValidateCredentials={async (id) => { handleValidate(id) }}
                onRevokeCredentials={async (id) => { handleRevoke(id) }}
                onSaveCredentials={async (id) => { handleSave(id) }}
                isSavingCredentials={saveCredsMutation.isPending}
                isValidatingCredentials={validateCredsMutation.isPending}
                isRevokingCredentials={revokeCredsMutation.isPending}
              />
            )}
          </div>

          <hr className="border-border" />

          {/* ================= SECTION C: PROJECT TEMPLATES ================= */}
          <div id="tour-settings-templates" ref={templatesRef} className="space-y-6 scroll-mt-10">
            <div className="space-y-2 border-b border-border pb-4 select-none">
              <div className="flex items-center gap-3">
                <Layers className="h-6 w-6 text-primary" />
                <h2 className="text-base font-bold text-foreground">Project Templates</h2>
              </div>
              <p className="text-[11px] text-muted-foreground leading-relaxed">
                Explore pre-configured showcase projects to understand real-world design setups and features.
              </p>
            </div>

            <div className="bg-card border border-border rounded-2xl p-6 space-y-4 shadow-sm">
              <div className="flex items-start justify-between">
                <div className="space-y-1">
                  <h3 className="text-sm font-bold text-foreground">Foundry Showcase Project</h3>
                  <p className="text-xs text-muted-foreground leading-relaxed">
                    A fully-seeded sample project featuring a complete pair programming assistant setup. Includes realistic product specs (PRD, Architecture, UX), sample chat histories, and mock workspace states.
                  </p>
                </div>
                <span className="inline-flex items-center rounded-md bg-blue-50 dark:bg-blue-900/30 px-2 py-1 text-xs font-medium text-blue-700 dark:text-blue-300 ring-1 ring-inset ring-blue-700/10 shrink-0 select-none">
                  Sample Project
                </span>
              </div>

              <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pt-2">
                {isShowcaseProvisioned ? (
                  <div className="flex items-center gap-2 text-xs text-emerald-600 dark:text-emerald-400 font-medium">
                    <CheckCircle2 className="h-4 w-4" />
                    <span>Provisioned (ID: {showcaseProject.id})</span>
                  </div>
                ) : (
                  <div className="text-xs text-muted-foreground">
                    Ready to provision in your active organization.
                  </div>
                )}

                <div className="flex items-center gap-3 self-end sm:self-auto">
                  {isShowcaseProvisioned ? (
                    <button
                      type="button"
                      disabled={isPending}
                      onClick={() => handleResetShowcase(showcaseProject.id)}
                      className="inline-flex h-9 items-center justify-center rounded-xl border border-input bg-background px-4 text-xs font-semibold text-muted-foreground hover:bg-accent hover:text-accent-foreground shadow-sm disabled:opacity-50 transition-colors gap-2"
                    >
                      {isResetting && <Loader2 className="h-3 w-3 animate-spin" />}
                      Restore to default state
                    </button>
                  ) : (
                    <button
                      type="button"
                      disabled={isPending}
                      onClick={handleProvisionShowcase}
                      className="inline-flex h-9 items-center justify-center rounded-xl bg-primary px-4 text-xs font-semibold text-primary-foreground shadow hover:opacity-90 disabled:opacity-50 transition-colors gap-2"
                    >
                      {isProvisioning && <Loader2 className="h-3 w-3 animate-spin" />}
                      Provision Foundry Showcase
                    </button>
                  )}
                </div>
              </div>
            </div>
          </div>

        </div>
      </main>

      <ConfirmDialog
        isOpen={isResetConfirmOpen}
        onClose={() => {
          setIsResetConfirmOpen(false)
          setShowcaseProjectIdToReset(null)
        }}
        onConfirm={handleConfirmReset}
        title="Restore Showcase Project"
        message="Are you sure you want to restore the showcase project to its default state? All your changes, sessions, and messages in this project will be deleted."
        confirmText="Restore"
        cancelText="Cancel"
        isDestructive={true}
        isLoading={isResetting}
      />

      <ConfirmDialog
        isOpen={isRevokeConfirmOpen}
        onClose={() => {
          setIsRevokeConfirmOpen(false)
          setAppIdToRevoke(null)
        }}
        onConfirm={handleConfirmRevoke}
        title="Remove Credentials"
        message="Are you sure you want to remove these secure credentials from your profile? This cannot be undone."
        confirmText="Remove"
        cancelText="Cancel"
        isDestructive={true}
        isLoading={revokeCredsMutation.isPending}
      />
    </div>
  )
}
