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

import { useEffect, useState } from "react"
import { useRouter } from "next/navigation"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import { useAppStore } from "@/stores/app"
import { useAuthStore } from "@/stores/auth"
import { api } from "@/lib/api"
import { 
  Project, 
  AppIntegration, 
  UserProject, 
  AppCredential, 
  CredentialAttribute 
} from "@/lib/types"
import { useOnboardingStore } from "@/stores/onboarding"

export const availableEmojis = [
  "🚀", "🛠️", "📦", "🎨", "📊", "🔒", "🌐", "📱", "💻", "🖥️",
  "🕹️", "🎮", "🔌", "🔋", "🔍", "🎯", "🏆", "💡", "🔥", "⭐",
  "🌈", "☀️", "🌙", "☁️", "🌧️", "❄️", "⚓", "✈️", "⛵", "🚲",
  "⏰", "⌛", "⏳", "🔑", "🎁", "🎈", "🎉", "🎊", "📚", "📝",
  "📌", "📍", "✂️", "📐", "📅"
]

export function useProjectDetails(projectId: string) {
  const router = useRouter()
  const queryClient = useQueryClient()
  const { setPageTitle } = useAppStore()
  const { email } = useAuthStore()
  const isNew = projectId === "new"

  // Form Field States
  const [name, setName] = useState("")
  const [description, setDescription] = useState("")
  const [icon, setIcon] = useState("🚀")
  const [specLocation, setSpecLocation] = useState("docs/spec")
  const [baseBranch, setBaseBranch] = useState("main")
  const [specBranchPrefix, setSpecBranchPrefix] = useState("specs")
  const [visibility, setVisibility] = useState("public")

  // BMAD Configuration States
  const [specFolder, setSpecFolder] = useState("docs/spec")
  const [implementationArtifactsPath, setImplementationArtifactsPath] = useState("implementation")
  const [planningArtifactsPath, setPlanningArtifactsPath] = useState("plans")
  const [testingArtifactsPath, setTestingArtifactsPath] = useState("test")

  // Additional Repo fields (backward compatibility/direct overrides)
  const [repoUrl, setRepoUrl] = useState("")
  const [repoAccessToken, setRepoAccessToken] = useState("")
  const [gitEmail, setGitEmail] = useState("")

  // Integrations states
  const [projectAppSettingsState, setProjectAppSettingsState] = useState<Record<string, UserProject>>({})
  const [expandedAppIds, setExpandedAppIds] = useState<Record<string, boolean>>({})
  const [savingAppIds, setSavingAppIds] = useState<Record<string, boolean>>({})
  const [isIconPickerOpen, setIsIconPickerOpen] = useState(false)
  const [isDeleteConfirmOpen, setIsDeleteConfirmOpen] = useState(false)
  const [isPermissionDialogOpen, setIsPermissionDialogOpen] = useState(false)
  const [permissionsCheck, setPermissionsCheck] = useState<{read: boolean, write: boolean, pullRequest: boolean, baseBranchExists?: boolean} | null>(null)
  const [isValidatingPermissions, setIsValidatingPermissions] = useState(false)
  const [isDuplicateDialogOpen, setIsDuplicateDialogOpen] = useState(false)
  const [duplicateProject, setDuplicateProject] = useState<{ id: string, name: string, hasAccess: boolean, potentialRole: string } | null>(null)
  const [isCheckingRepo, setIsCheckingRepo] = useState(false)
  const [isBaseBranchModalOpen, setIsBaseBranchModalOpen] = useState(false)
  const [missingBaseBranchName, setMissingBaseBranchName] = useState("")
  const [isRepoChangeModalOpen, setIsRepoChangeModalOpen] = useState(false)

  // Sync dyn header title
  useEffect(() => {
    setPageTitle(isNew ? "Create Project" : "Project Settings")
  }, [isNew, setPageTitle])

  // 1. Query: Get project details (only if NOT creating dynamic new)
  const {
    data: project,
    isLoading: isLoadingProject,
  } = useQuery<Project>({
    queryKey: ["project", projectId],
    queryFn: async () => {
      const res = await api.get(`/api/v1/projects/${projectId}`)
      if (!res.ok) throw new Error("Failed to load project profile")
      return res.json()
    },
    enabled: !isNew,
  })

  // 2. Query: Get global app integrations list config
  const {
    data: appIntegrations,
    isLoading: isLoadingApps,
  } = useQuery<AppIntegration[]>({
    queryKey: ["app-integrations"],
    queryFn: async () => {
      const res = await api.get("/api/v1/config/app-integrations")
      if (!res.ok) throw new Error("Failed to load app integrations")
      return res.json()
    },
  })

  // 3. Query: Get active user global app credentials in parallel to check inherit status
  const {
    data: globalCredentials,
    isLoading: isLoadingGlobalCreds,
  } = useQuery<Record<string, AppCredential>>({
    queryKey: ["global-credentials", email],
    queryFn: async () => {
      if (!email) return {}
      const activeApps = (appIntegrations || []).filter((app) => app.isEnabled)
      const credsMap: Record<string, AppCredential> = {}
      
      await Promise.all(
        activeApps.map(async (app) => {
          try {
            const res = await api.get(`/api/v1/users/${encodeURIComponent(email)}/apps/${app.id}/credentials`)
            if (res.ok) {
              credsMap[app.id] = await res.json()
            }
          } catch (_) {
            // Suppress missing credential logs
          }
        })
      )
      return credsMap
    },
    enabled: !!email && !!appIntegrations && appIntegrations.length > 0,
  })

  // 4. Query: Get project specific app credentials configuration states
  const {
    data: projectAppSettings,
    isLoading: isLoadingProjectSettings,
  } = useQuery<Record<string, UserProject>>({
    queryKey: ["project-app-settings", projectId, email],
    queryFn: async () => {
      if (!email || isNew) return {}
      const activeApps = (appIntegrations || []).filter((app) => app.isEnabled)
      const settingsMap: Record<string, UserProject> = {}

      await Promise.all(
        activeApps.map(async (app) => {
          try {
            const res = await api.get(
              `/api/v1/users/${encodeURIComponent(email)}/projects/${projectId}/apps/${app.id}/credentials`
            )
            if (res.ok) {
              settingsMap[app.id] = await res.json()
            }
          } catch (_) {
            // Suppress missing integrations configs
          }
        })
      )
      return settingsMap
    },
    enabled: !!email && !isNew && !!appIntegrations && appIntegrations.length > 0,
  })

  const [initialRepoUrl, setInitialRepoUrl] = useState("")

  // Set initial form states once data loads
  useEffect(() => {
    if (project && !isNew) {
      setName(project.name)
      setDescription(project.description)
      setIcon(availableEmojis.includes(project.icon) ? project.icon : "🚀")
      setBaseBranch(project.baseBranch || "main")
      setVisibility(project.visibility || "public")
      setRepoAccessToken(project.repoAccessToken || "")

      const loadedSpecFolder = project.bmadConfig?.specFolder || project.specLocation || "docs/spec"
      setSpecFolder(loadedSpecFolder)

      const stripSpecPrefix = (val: string | undefined, fallback: string) => {
        if (!val) return fallback
        const cleanSpec = loadedSpecFolder.replace(/^\/+|\/+$/g, "")
        const cleanVal = val.replace(/^\/+|\/+$/g, "")
        if (cleanVal.startsWith(cleanSpec + "/")) {
          return cleanVal.slice(cleanSpec.length + 1)
        }
        if (cleanVal === cleanSpec) {
          return fallback
        }
        return cleanVal
      }

      const rawImpl = project.bmadConfig?.implementationArtifactsPath || project.implementationArtifactsLocation
      const rawPlan = project.bmadConfig?.planningArtifactsPath || project.planningArtifactsLocation
      const rawTest = project.bmadConfig?.testingArtifactsPath || project.testingArtifactsLocation

      setImplementationArtifactsPath(stripSpecPrefix(rawImpl, "implementation"))
      setPlanningArtifactsPath(stripSpecPrefix(rawPlan, "plans"))
      setTestingArtifactsPath(stripSpecPrefix(rawTest, "test"))

      const githubIntegration = project.app_integrations?.find(
        (integration) => integration.integration_id === "github"
      )
      if (githubIntegration) {
        const loadedUrl = githubIntegration.settings?.repoUrl || ""
        setRepoUrl(loadedUrl)
        setInitialRepoUrl(loadedUrl)
        setSpecLocation(githubIntegration.settings?.specLocation || "docs/spec")
        setSpecBranchPrefix(githubIntegration.settings?.specBranchPrefix || "specs")
        setGitEmail(githubIntegration.settings?.email || "")
      } else {
        setRepoUrl("")
        setInitialRepoUrl("")
        setSpecLocation("docs/spec")
        setSpecBranchPrefix("specs")
        setGitEmail("")
      }
    }
  }, [project, isNew])

  // Automatically fetch profile email if GitHub is enabled and gitEmail is empty
  useEffect(() => {
    const githubSetting = projectAppSettingsState["github"]
    const isGithubEnabled = githubSetting?.enabled || false
    if (!isGithubEnabled || gitEmail) return

    let cancelled = false
    async function fetchProfileEmail() {
      try {
        const customToken = githubSetting?.fields?.find((f) => f.name === "pat")?.value || ""
        const isOverridden = githubSetting?.isOverridden
        const hasRealToken = isOverridden && customToken && !/[•·]/.test(customToken) && customToken.trim() !== ""
        const headers: Record<string, string> = {}
        if (hasRealToken) {
          headers["X-GitHub-Token"] = customToken
        }
        const res = await api.get("/api/v1/integrations/github/profile", Object.keys(headers).length > 0 ? { headers } : undefined)
        if (res.ok && !cancelled) {
          const data = await res.json()
          if (data.email) {
            setGitEmail(data.email)
          }
        }
      } catch (err) {
        console.error("Failed to fetch profile email in useProjectDetails:", err)
      }
    }
    fetchProfileEmail()
    return () => {
      cancelled = true
    }
  }, [projectAppSettingsState, gitEmail])

  // Map initial integration toggle check status states once credentials/settings load
  useEffect(() => {
    if (appIntegrations && (isNew || projectAppSettings)) {
      const initialSettings: Record<string, UserProject> = {}
      const initialExpanded: Record<string, boolean> = {}

      appIntegrations.forEach((app) => {
        const setting = projectAppSettings?.[app.id]
        const globalCred = globalCredentials?.[app.id]
        const hasGlobal = !!globalCred

        if (setting) {
          if (setting.enabled) {
            initialExpanded[app.id] = true
          }
          if (setting.enabled && !setting.isOverridden) {
            // Inherited mode: ensure fields are populated from global creds!
            const copiedFields = globalCred?.fields.map((f) => ({
              ...f,
              value: f.value || "",
            })) || app.fields.map((f) => ({
              ...f,
              value: "",
            }))

            initialSettings[app.id] = {
              ...setting,
              fields: copiedFields,
            }
          } else {
            // Overridden or disabled mode: use fields from setting directly
            initialSettings[app.id] = {
              ...setting,
              fields: setting.fields || [],
            }
          }
        } else {
          // Default initialization
          const initialFields = globalCred?.fields.map((f) => ({
            ...f,
            value: f.value || "",
          })) || app.fields.map((f) => ({
            ...f,
            value: "",
          }))

          const isGitHubDefault = isNew && app.id === "github"
          const isGeminiDefault = isNew && app.id === "gemini" && hasGlobal

          initialSettings[app.id] = {
            userId: email || "",
            projectId: projectId,
            appId: app.id,
            enabled: isGitHubDefault || isGeminiDefault,
            credentialId: hasGlobal ? app.id : "",
            isOverridden: false,
            fields: initialFields,
          }

          if (isGitHubDefault || isGeminiDefault) {
            initialExpanded[app.id] = true
          }
        }
      })

      setProjectAppSettingsState(initialSettings)
      setExpandedAppIds((prev) => {
        const next = { ...prev }
        Object.entries(initialExpanded).forEach(([appId, isExpanded]) => {
          if (next[appId] === undefined) {
            next[appId] = isExpanded
          }
        })
        return next
      })
    }
  }, [appIntegrations, projectAppSettings, globalCredentials, isNew, email, projectId])

  // Mutation: Save / Update Project Profile configurations
  const saveProjectMutation = useMutation({
    mutationFn: async (payload: any) => {
      let res
      if (isNew) {
        res = await api.post("/api/v1/projects", payload)
      } else {
        res = await api.put(`/api/v1/projects/${projectId}`, payload)
      }

      if (!res.ok) throw new Error("Server rejected project details")
      return res.json()
    },
    onSuccess: async (data) => {
      const savedId = isNew ? data.id : projectId
      let warningMessage = ""

      try {
        const githubSetting = projectAppSettingsState["github"]
        const githubEnabled = githubSetting?.enabled || false
        const hasGithubConn = Boolean(githubEnabled && repoUrl.trim())

        // 1. Save GitHub integration settings first (blocking)
        if (hasGithubConn) {
          const res = await api.post(`/api/v1/projects/${savedId}/integrations`, {
            integration_id: "github",
            enabled: true,
            settings: {
              repoUrl: repoUrl.trim(),
              specLocation: specLocation.trim() || "docs/specs",
              baseBranch: baseBranch.trim(),
              specBranchPrefix: specBranchPrefix.trim(),
              email: gitEmail.trim(),
            },
          })
          if (!res.ok) {
            throw new Error("Failed to save GitHub integration settings")
          }
        }

        // 2. Save bulk project app settings after GitHub integration is saved
        if (email && savedId) {
          const visibleAppIds = new Set(appsToShow.map((app) => app.id))

          const bulkPayload = Object.entries(projectAppSettingsState)
            .filter(([appId, setting]) => (setting.enabled || appId === "github") && visibleAppIds.has(appId))
            .map(([appId, setting]) => {
              if (appId === "github") {
                return {
                  appId,
                  enabled: hasGithubConn,
                  credentialId: setting.credentialId,
                  isOverridden: setting.isOverridden,
                  fields: setting.fields,
                  isReadOnly: !hasGithubConn ? true : Boolean(setting.isReadOnly),
                }
              }
              return {
                appId,
                enabled: setting.enabled,
                credentialId: setting.credentialId,
                isOverridden: setting.isOverridden,
                fields: setting.fields,
                isReadOnly: Boolean(setting.isReadOnly),
              }
            })

          if (bulkPayload.length > 0) {
            const res = await api.put(
              `/api/v1/users/${encodeURIComponent(email)}/projects/${savedId}/apps/credentials`,
              bulkPayload
            )
            if (!res.ok) {
              throw new Error("Failed to bulk save project app credentials")
            }
            const resData = await res.json()
            if (resData.status === "WARNING") {
              warningMessage = resData.message
            }
          }
        }

        if (warningMessage) {
          toast.warning(warningMessage, {
            duration: 10000,
          })
        } else {
          toast.success(isNew ? "Project created successfully!" : "Project updated successfully!")
        }

      } catch (e: any) {
        console.error("Failed to complete post-creation setup:", e)
        toast.error(e.message || "Failed to complete project setup details.")
      }

      queryClient.invalidateQueries({ queryKey: ["projects"] })
      queryClient.invalidateQueries({ queryKey: ["project", savedId] })
      router.push("/projects")
    },
    onError: (err) => {
      console.error("Save error:", err)
      toast.error("Failed to save project configurations.")
    },
  })

  // Mutation: Delete existing project
  const deleteProjectMutation = useMutation({
    mutationFn: async () => {
      const res = await api.delete(`/api/v1/projects/${projectId}`)
      if (!res.ok) throw new Error("Delete command failed")
      return res
    },
    onSuccess: () => {
      toast.success("Project deleted successfully!")
      // Clear selections parameters cache
      if (useAppStore.getState().lastSelectedProjectId === projectId) {
        useAppStore.getState().setLastSelectedProjectId(null)
      }
      queryClient.invalidateQueries({ queryKey: ["projects"] })
      router.push("/projects")
    },
    onError: (err) => {
      console.error("Delete error:", err)
      toast.error("Failed to delete project.")
    },
  })

  // Mutation: Join duplicate project automatically
  const joinProjectMutation = useMutation({
    mutationFn: async (targetProjectId: string) => {
      const res = await api.post(`/api/v1/projects/${targetProjectId}/join`, {})
      if (!res.ok) {
        const data = await res.json()
        throw new Error(data.message || "Failed to join project")
      }
      return res.json() as Promise<{ status: string, role: string }>
    },
    onSuccess: (data, targetProjectId) => {
      toast.success(`Successfully joined project as ${data.role}!`)
      setIsDuplicateDialogOpen(false)
      queryClient.invalidateQueries({ queryKey: ["projects"] })
      router.push(`/projects/${targetProjectId}/workspace`)
    },
    onError: (err: any) => {
      console.error("Join project error:", err)
      toast.error(err.message || "Failed to join project automatically.")
    },
  })

  const performSave = (payload: any) => {
    saveProjectMutation.mutate(payload)
  }

  const buildSavePayload = () => ({
    name: name.trim(),
    description: description.trim(),
    icon,
    repoAccessToken: repoAccessToken.trim(),
    baseBranch: baseBranch.trim(),
    visibility,
    bmadConfig: {
      specFolder: specFolder.trim() || "docs/spec",
      implementationArtifactsPath: implementationArtifactsPath.trim() || "implementation",
      planningArtifactsPath: planningArtifactsPath.trim() || "plans",
      testingArtifactsPath: testingArtifactsPath.trim() || "test",
    },
  })

  const hasValidGeminiKey = () => {
    const geminiSetting = projectAppSettingsState["gemini"]
    const globalGemini = globalCredentials?.["gemini"]

    const globalKeyVal = globalGemini?.fields?.find(
      (f) => f.name === "gemini_api_key" || f.name === "apiKey"
    )?.value?.trim()
    const hasGlobalKey = Boolean(
      globalGemini && (globalKeyVal !== undefined ? Boolean(globalKeyVal) : true)
    )

    if (geminiSetting) {
      if (!geminiSetting.enabled) {
        return false
      }
      if (geminiSetting.isOverridden) {
        const overriddenKey = geminiSetting.fields?.find(
          (f) => f.name === "gemini_api_key" || f.name === "apiKey"
        )?.value?.trim()
        return Boolean(overriddenKey)
      }
      const inheritedKey = geminiSetting.fields?.find(
        (f) => f.name === "gemini_api_key" || f.name === "apiKey"
      )?.value?.trim()
      return hasGlobalKey || Boolean(inheritedKey)
    }

    return hasGlobalKey
  }

  const checkPermissionsAndSave = async () => {
    if (!hasValidGeminiKey()) {
      toast.error(
        "A project cannot be saved without a valid Gemini API Key. Please provide a Gemini API Key in Application Credentials or configure it in Settings."
      )
      return
    }

    const githubSetting = projectAppSettingsState["github"]
    const githubEnabled = githubSetting?.enabled || false
    const hasGithubConn = Boolean(githubEnabled && repoUrl.trim())

    const payload = buildSavePayload()

    if (!hasGithubConn) {
      performSave(payload)
      return
    }

    // 1. Check if repo already exists (only when creating a NEW project)
    if (isNew) {
      setIsCheckingRepo(true)
      try {
        const checkRes = await api.get(`/api/v1/projects/check-repo?repoUrl=${encodeURIComponent(repoUrl.trim())}`)
        if (!checkRes.ok) {
          const checkData = await checkRes.json().catch(() => ({}))
          throw new Error(checkData.message || checkData.error || "Failed to check repository availability")
        }
        const checkData = await checkRes.json() as { exists: boolean, projectId?: string, projectName?: string, hasAccess?: boolean, potentialRole?: string }
        
        if (checkData.exists) {
          setDuplicateProject({
            id: checkData.projectId!,
            name: checkData.projectName!,
            hasAccess: checkData.hasAccess!,
            potentialRole: checkData.potentialRole || "",
          })
          setIsDuplicateDialogOpen(true)
          setIsCheckingRepo(false)
          return // STOP FLOW
        }
      } catch (err: any) {
        console.error("Repo check error:", err)
        toast.error(err.message || "Failed to verify repository availability.")
        setIsCheckingRepo(false)
        return // STOP FLOW
      }
      setIsCheckingRepo(false)
    }

    // 2. Continue with permission validation
    setIsValidatingPermissions(true)
    try {
      let token = ""
      const githubSetting = projectAppSettingsState["github"]
      const patField = githubSetting?.fields?.find(f => f.name === "pat")

      if (githubSetting?.enabled && githubSetting?.isOverridden && patField?.value?.trim() && !/[•·]/.test(patField.value)) {
        token = patField.value.trim()
      } else if (repoAccessToken.trim() && !/[•·]/.test(repoAccessToken)) {
        token = repoAccessToken.trim()
      } else if (patField?.value?.trim() && !/[•·]/.test(patField.value)) {
        token = patField.value.trim()
      }

      const res = await api.post("/api/v1/integrations/github/validate-permissions", {
        repoUrl: repoUrl.trim(),
        token: token || undefined,
        projectId: isNew ? undefined : projectId,
        baseBranch: baseBranch.trim(),
      })

      if (!res.ok) {
        throw new Error("Failed to validate repository permissions")
      }

      const data = await res.json() as { read: boolean, write: boolean, pullRequest: boolean, baseBranchExists?: boolean, baseBranch?: string }
      setPermissionsCheck(data)

      if (isNew && data.baseBranchExists === false) {
        setMissingBaseBranchName(baseBranch.trim())
        setIsBaseBranchModalOpen(true)
        return
      }

      if (data.write && data.pullRequest) {
        performSave(payload)
      } else {
        setIsPermissionDialogOpen(true)
      }
    } catch (e: any) {
      console.error("Permission check failed:", e)
      toast.error(e.message || "Failed to verify repository permissions. Please check your token.")
    } finally {
      setIsValidatingPermissions(false)
    }
  }

  const handleConfirmBaseBranch = () => {
    setIsBaseBranchModalOpen(false)
    const payload = buildSavePayload()
    if (permissionsCheck && (!permissionsCheck.write || !permissionsCheck.pullRequest)) {
      setIsPermissionDialogOpen(true)
    } else {
      performSave(payload)
    }
  }

  const handleSubmit = (e?: React.FormEvent) => {
    if (e) e.preventDefault()
    if (!name.trim()) {
      toast.error("Project Name is required")
      return
    }

    const githubSetting = projectAppSettingsState["github"]
    const githubEnabled = githubSetting?.enabled || false
    if (githubEnabled) {
      if (!repoUrl.trim()) {
        toast.error("Please either disable Git integration or select a repository.")
        return
      }
      if (!gitEmail.trim()) {
        toast.error("Git Author Email is required when GitHub integration is enabled.")
        return
      }
    }

    if (!hasValidGeminiKey()) {
      toast.error(
        "A project cannot be saved without a valid Gemini API Key. Please provide a Gemini API Key in Application Credentials or configure it in Settings."
      )
      return
    }

    const isRepoChanged = !isNew && Boolean(repoUrl.trim()) && repoUrl.trim() !== initialRepoUrl
    if (isRepoChanged) {
      setIsRepoChangeModalOpen(true)
      return
    }

    checkPermissionsAndSave()
  }

  const handleConfirmRepoChange = () => {
    setIsRepoChangeModalOpen(false)
    checkPermissionsAndSave()
  }

  const handleConfirmSave = () => {
    setIsPermissionDialogOpen(false)
    const payload = buildSavePayload()
    performSave(payload)
  }

  const handleDelete = () => {
    deleteProjectMutation.mutate()
  }

  const handleToggleApp = async (appId: string, enabled: boolean) => {
    if (savingAppIds[appId]) return
    if (!email) {
      toast.error("User email is missing. Please sign in again.")
      return
    }

    // Onboarding tour check
    if (appId === "github" && enabled) {
      useOnboardingStore.getState().advanceStep("catalog", 13)
    }

    const app = appIntegrations?.find((a) => a.id === appId)
    if (!app) return

    const existing = projectAppSettingsState[app.id]
    const globalCred = globalCredentials?.[app.id]
    const hasGlobal = !!globalCred

    let updatedFields: CredentialAttribute[] = []
    if (enabled) {
      if (hasGlobal) {
        updatedFields = globalCred.fields.map((f) => ({
          ...f,
          value: f.value || "",
        }))
      } else {
        updatedFields = app.fields.map((f) => ({
          ...f,
          value: "",
        }))
      }
    }

    const updated: UserProject = {
      userId: email,
      projectId: projectId,
      appId: appId,
      enabled: enabled,
      credentialId: enabled ? (hasGlobal ? appId : "") : "",
      isOverridden: false,
      fields: updatedFields,
    }

    setProjectAppSettingsState((prev) => ({
      ...prev,
      [appId]: updated,
    }))

    if (enabled) {
      setExpandedAppIds((prev) => ({ ...prev, [appId]: true }))
    }

    if (isNew) {
      return
    }

    try {
      setSavingAppIds((prev) => ({ ...prev, [appId]: true }))
      const res = await api.put(
        `/api/v1/users/${encodeURIComponent(email)}/projects/${projectId}/apps/${appId}/credentials`,
        updated
      )
      if (!res.ok) throw new Error("Failed to update project integration")
      toast.success(`Successfully ${enabled ? "enabled" : "disabled"} ${app.name}!`)
    } catch (err: any) {
      console.error(err)
      toast.error(err.message || "Failed to update integration state")
      if (existing) {
        setProjectAppSettingsState((prev) => ({
          ...prev,
          [appId]: existing,
        }))
      }
    } finally {
      setSavingAppIds((prev) => ({ ...prev, [appId]: false }))
    }
  }

  const handleFieldChange = (appId: string, fieldName: string, value: string) => {
    if (appId === "github" && fieldName === "pat") {
      setRepoAccessToken(value)
    }
    setProjectAppSettingsState((prev) => {
      const current = prev[appId]
      if (!current) return prev
      const hasField = current.fields?.some((f) => f.name === fieldName)
      let updatedFields = current.fields?.map((f) => {
        if (f.name === fieldName) {
          return { ...f, value: value }
        }
        return f
      }) || []

      if (!hasField) {
        const appField = appIntegrations?.find((a) => a.id === appId)?.fields?.find((f) => f.name === fieldName)
        const newAttr: CredentialAttribute = {
          name: fieldName,
          displayName: appField?.displayName || fieldName,
          enabled: true,
          helpText: appField?.helpText || "",
          isSecret: appField?.isSecret ?? true,
          value: value,
        }
        updatedFields = [...updatedFields, newAttr]
      }

      return {
        ...prev,
        [appId]: {
          ...current,
          fields: updatedFields,
        },
      }
    })
  }

  const handleOverrideCredentials = (appId: string) => {
    const app = appIntegrations?.find((a) => a.id === appId)
    if (!app) return
    const globalCred = globalCredentials?.[appId]

    if (appId === "github") {
      setRepoAccessToken("")
    }

    const copiedFields = globalCred?.fields.map((f) => ({
      ...f,
      value: f.isSecret ? "" : (f.value || ""),
    })) || app.fields.map((f) => ({
      ...f,
      value: "",
    }))

    setProjectAppSettingsState((prev) => {
      const setting = prev[appId]
      if (!setting) return prev

      return {
        ...prev,
        [appId]: {
          ...setting,
          isOverridden: true,
          fields: copiedFields,
        },
      }
    })
  }

  const handleRevertToMaster = async (appId: string) => {
    if (savingAppIds[appId]) return
    if (!email) return

    if (appId === "github") {
      setRepoAccessToken("")
    }

    const app = appIntegrations?.find((a) => a.id === appId)
    if (!app) return

    const existing = projectAppSettingsState[appId]
    const globalCred = globalCredentials?.[appId]
    const copiedFields = globalCred?.fields.map((f) => ({
      ...f,
      value: f.value || "",
    })) || []

    const reverted: UserProject = {
      userId: email,
      projectId: projectId,
      appId: appId,
      enabled: existing?.enabled || false,
      credentialId: appId,
      isOverridden: false,
      fields: copiedFields,
    }

    setProjectAppSettingsState((prev) => ({
      ...prev,
      [appId]: reverted,
    }))

    if (isNew) {
      return
    }

    try {
      setSavingAppIds((prev) => ({ ...prev, [appId]: true }))
      const res = await api.put(
        `/api/v1/users/${encodeURIComponent(email)}/projects/${projectId}/apps/${appId}/credentials`,
        reverted
      )
      if (!res.ok) throw new Error("Failed to revert settings to master")
      toast.success(`Successfully reverted ${app.name} to Master Inherited!`)
    } catch (err: any) {
      console.error(err)
      toast.error(err.message || "Failed to revert settings")
      if (existing) {
        setProjectAppSettingsState((prev) => ({
          ...prev,
          [appId]: existing,
        }))
      }
    } finally {
      setSavingAppIds((prev) => ({ ...prev, [appId]: false }))
    }
  }


  const handleCancel = () => {
    router.back()
  }

  const isLoading =
    (!isNew && isLoadingProject) ||
    isLoadingApps ||
    (!isNew && isLoadingProjectSettings) ||
    isLoadingGlobalCreds

  const appsToShow = appIntegrations?.filter((app) => app.isEnabled) || []

  return {
    isNew,
    projectId,
    isLoading,
    name,
    setName,
    description,
    setDescription,
    icon,
    setIcon,
    specLocation,
    setSpecLocation,
    baseBranch,
    setBaseBranch,
    visibility,
    setVisibility,
    specBranchPrefix,
    setSpecBranchPrefix,
    specFolder,
    setSpecFolder,
    implementationArtifactsPath,
    setImplementationArtifactsPath,
    planningArtifactsPath,
    setPlanningArtifactsPath,
    testingArtifactsPath,
    setTestingArtifactsPath,
    repoUrl,
    setRepoUrl,
    repoAccessToken,
    setRepoAccessToken,
    gitEmail,
    setGitEmail,
    projectAppSettingsState,
    expandedAppIds,
    setExpandedAppIds,
    savingAppIds,
    isIconPickerOpen,
    setIsIconPickerOpen,
    isDeleteConfirmOpen,
    setIsDeleteConfirmOpen,
    appsToShow,
    globalCredentials,
    isPermissionDialogOpen,
    setIsPermissionDialogOpen,
    permissionsCheck,
    isValidatingPermissions,
    isDuplicateDialogOpen,
    setIsDuplicateDialogOpen,
    duplicateProject,
    isCheckingRepo,
    isBaseBranchModalOpen,
    setIsBaseBranchModalOpen,
    missingBaseBranchName,
    handleConfirmBaseBranch,
    initialRepoUrl,
    isRepoChangeModalOpen,
    setIsRepoChangeModalOpen,
    handleConfirmRepoChange,
    joinProjectPending: joinProjectMutation.isPending,
    handleJoinProject: () => {
      if (duplicateProject) {
        joinProjectMutation.mutate(duplicateProject.id)
      }
    },
    handleGoToProject: () => {
      if (duplicateProject) {
        router.push(`/projects/${duplicateProject.id}/workspace`)
      }
    },
    handleConfirmSave,
    handleSubmit,
    hasValidGeminiKey,
    handleDelete,
    handleToggleApp,
    handleFieldChange,
    handleOverrideCredentials,
    handleRevertToMaster,
    handleCancel,
    savePending: saveProjectMutation.isPending,
    deletePending: deleteProjectMutation.isPending,
  }
}
