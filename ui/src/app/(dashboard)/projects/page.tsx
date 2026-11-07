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
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import { Plus, HelpCircle, Loader2, ShieldAlert } from "lucide-react"
import { useAppStore } from "@/stores/app"
import { useAuthStore } from "@/stores/auth"
import { useActivityStore } from "@/stores/activity"
import { api } from "@/lib/api"
import { Project, WorkflowTemplate } from "@/lib/types"
import ProjectHeader from "./_components/ProjectHeader"
import EmptyProjectsState from "./_components/EmptyProjectsState"
import ProjectCatalog from "./_components/ProjectCatalog"
import ComingSoonModal from "@/components/shared/ComingSoonModal"

interface ProjectsApiResponse {
  myProjects: Project[]
  browseProjects: Project[]
}

export default function ProjectsPage() {
  const router = useRouter()
  const queryClient = useQueryClient()
  const { setPageTitle } = useAppStore()
  const { isAuthenticated, isCheckingAuth, isAdmin, permissions } = useAuthStore()
  const { isPanelOpen } = useActivityStore()

  const userPermissions = permissions || []
  const hasPermission = (perm: string) => isAdmin || userPermissions.includes(perm)

  const [activeTab, setActiveTab] = useState<"my" | "shared">("my")
  const [viewMode, setViewMode] = useState<"card" | "list">("card")
  const [isSharedComingSoonOpen, setIsSharedComingSoonOpen] = useState(false)

  // Sync dyn header title on mount
  useEffect(() => {
    setPageTitle("Project List")
  }, [setPageTitle])

  // Query: Get both member and browseable shared projects
  const {
    data: projectsData,
    isLoading: isLoadingProjects,
    error: projectsError,
  } = useQuery<ProjectsApiResponse>({
    queryKey: ["projects"],
    queryFn: async () => {
      const res = await api.get("/api/v1/projects")
      if (!res.ok) {
        throw new Error(`HTTP error status: ${res.status}`)
      }
      const data = await res.json()
      // Fallbacks mapping to empty arrays if null/missing from API
      return {
        myProjects: data.myProjects || [],
        browseProjects: data.browseProjects || [],
      }
    },
  })

  // Query: Get workflows list to resolve milestones definitions
  const {
    data: workflowsData,
    isLoading: isLoadingWorkflows,
  } = useQuery<WorkflowTemplate[]>({
    queryKey: ["workflows", "BMAD"],
    queryFn: async () => {
      const res = await api.get("/api/v1/workflows?type=BMAD")
      if (!res.ok) {
        throw new Error(`HTTP error status: ${res.status}`)
      }
      return res.json()
    },
  })

  // Mutation: Send membership access requests
  const requestAccessMutation = useMutation({
    mutationFn: async (projectId: string) => {
      const res = await api.post(`/api/v1/projects/${projectId}/requests`, {
        requestedRole: "member",
      })
      if (!res.ok) {
        throw new Error(`HTTP error status: ${res.status}`)
      }
      return res
    },
    onSuccess: () => {
      toast.success("Access requested successfully! Your request is pending approval.")
      queryClient.invalidateQueries({ queryKey: ["projects"] })
    },
    onError: (err) => {
      console.error("Access request failed:", err)
      toast.error("Failed to request project access. Please try again.")
    },
  })

  const handleRequestAccess = async (projectId: string) => {
    if (projectId === "foundry-showcase-project") {
      try {
        const res = await api.post("/api/v1/projects/provision-showcase", {})
        if (!res.ok) throw new Error("Failed to provision showcase project")
        toast.success("Showcase project provisioned successfully!")
        queryClient.invalidateQueries({ queryKey: ["projects"] })
        queryClient.invalidateQueries({ queryKey: ["user-projects"] })
      } catch (err: any) {
        toast.error(`Provisioning failed: ${err.message}`)
      }
      return
    }
    await requestAccessMutation.mutateAsync(projectId)
  }

  const handleCreateProject = () => {
    router.push("/projects/new/details")
  }

  const myProjects = projectsData?.myProjects || []
  const browseProjects = projectsData?.browseProjects || []

  // Resolve standard BMAD workflow definitions (equivalent to Dart firstWhere mapping)
  const bmadWorkflow = workflowsData?.find(
    (w) => w.name === "Standard BMAD Workflow"
  ) || workflowsData?.[0] || {
    id: "",
    name: "Default",
    framework: "BMAD",
    phases: {},
    artifacts: {},
    dependencies: {},
  } as WorkflowTemplate

  const activeProjects = activeTab === "my" ? myProjects : browseProjects
  const isBrowse = activeTab === "shared"

  const isLoading = isLoadingProjects || isLoadingWorkflows

  if (!isCheckingAuth && isAuthenticated && !hasPermission("project:list")) {
    return (
      <div className="flex-1 flex flex-col items-center justify-center p-8 min-h-[calc(100vh-4rem)] select-none">
        <div className="w-full max-w-md bg-destructive/10 border border-destructive/20 rounded-2xl p-8 text-center space-y-6">
          <div className="flex justify-center text-destructive">
            <ShieldAlert className="h-16 w-16" />
          </div>
          <div className="space-y-2">
            <h3 className="text-xl font-bold text-foreground">
              Access Restricted
            </h3>
            <p className="text-sm text-muted-foreground leading-relaxed">
              You do not have permission to view projects.
            </p>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div id="tour-catalog-header" className="p-6 space-y-6 w-full">
      <ProjectHeader
        activeTab={activeTab}
        setActiveTab={setActiveTab}
        viewMode={viewMode}
        setViewMode={setViewMode}
        onSharedProjectsClick={() => setIsSharedComingSoonOpen(true)}
      />

      {/* Main Content Pane Grid */}
      {isLoading ? (
        <div className="flex h-[300px] w-full flex-col items-center justify-center gap-3 text-muted-foreground select-none">
          <Loader2 className="h-8 w-8 animate-spin text-primary" />
          <span className="text-sm font-medium">Loading project catalog...</span>
        </div>
      ) : projectsError ? (
        <div className="flex h-[250px] w-full flex-col items-center justify-center text-center gap-2 text-destructive max-w-md mx-auto p-4 border border-destructive/20 bg-destructive/5 rounded-xl">
          <HelpCircle className="h-8 w-8 opacity-60" />
          <h3 className="font-semibold text-sm">Failed to load projects</h3>
          <p className="text-xs text-muted-foreground leading-normal">
            {(projectsError as Error).message || "An unexpected network error occurred while querying the server."}
          </p>
        </div>
      ) : activeProjects.length === 0 ? (
        <EmptyProjectsState
          activeTab={activeTab}
          onCreateProject={handleCreateProject}
        />
      ) : (
        <ProjectCatalog
          viewMode={viewMode}
          activeProjects={activeProjects}
          bmadWorkflow={bmadWorkflow}
          isBrowse={isBrowse}
          onRequestAccess={handleRequestAccess}
        />
      )}

      {/* Floating Action Button (FAB) matching Flutter */}
      {isAuthenticated && (
        <button
          id="tour-catalog-create"
          onClick={handleCreateProject}
          className={`fixed bottom-6 z-50 inline-flex h-12 items-center justify-center gap-2 rounded-full bg-primary px-6 text-sm font-semibold text-primary-foreground shadow-lg transition-all duration-200 hover:scale-105 active:scale-95 ${
            isPanelOpen ? "right-[364px]" : "right-6"
          }`}
        >
          <Plus className="h-5 w-5" />
          Create Project
        </button>
      )}

      <ComingSoonModal
        isOpen={isSharedComingSoonOpen}
        onClose={() => setIsSharedComingSoonOpen(false)}
        featureName="Shared Projects"
      />
    </div>
  )
}
