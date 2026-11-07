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

import React from "react"
import { Layers } from "lucide-react"
import { AppIntegration, AppCredential, UserProject } from "@/lib/types"
import AppIntegrationsList from "@/components/shared/AppIntegrationsList"

interface AppCredentialsSectionProps {
  isNew: boolean
  projectId: string
  appsToShow: AppIntegration[]
  projectAppSettingsState: Record<string, UserProject>
  expandedAppIds: Record<string, boolean>
  setExpandedAppIds: React.Dispatch<React.SetStateAction<Record<string, boolean>>>
  savingAppIds: Record<string, boolean>
  globalCredentials?: Record<string, AppCredential>
  handleToggleApp: (appId: string, enabled: boolean) => Promise<void>
  handleFieldChange: (appId: string, fieldName: string, value: string) => void
  handleOverrideCredentials: (appId: string) => void
  handleRevertToMaster: (appId: string) => Promise<void>
  
  // GitHub specific settings props
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

export default function AppCredentialsSection({
  isNew,
  projectId,
  appsToShow,
  projectAppSettingsState,
  expandedAppIds,
  setExpandedAppIds,
  savingAppIds,
  globalCredentials,
  handleToggleApp,
  handleFieldChange,
  handleOverrideCredentials,
  handleRevertToMaster,
  repoUrl,
  setRepoUrl,
  specLocation,
  setSpecLocation,
  baseBranch,
  setBaseBranch,
  specBranchPrefix,
  setSpecBranchPrefix,
  email,
  setEmail,
}: AppCredentialsSectionProps) {
  if (appsToShow.length === 0) return null

  return (
    <div className="space-y-4">
      <div className="space-y-1 select-none">
        <div className="flex items-center gap-2 border-b border-border pb-3 mb-2">
          <Layers className="h-5 w-5 text-primary" />
          <h2 className="text-sm font-semibold text-foreground">Application Credentials</h2>
        </div>
        <p className="text-[11px] text-muted-foreground leading-relaxed">
          {isNew
            ? "Choose which of your user-level active integrations you want to enable for this new project."
            : "Configure platform tool status settings for this project. Application settings are completely scoped to your user account and are independent of other members."}
        </p>
      </div>

      <AppIntegrationsList
        mode="project-override"
        integrations={appsToShow}
        globalCredentials={globalCredentials}
        expandedAppIds={expandedAppIds}
        setExpandedAppIds={setExpandedAppIds}
        
        projectId={projectId}
        isNewProject={isNew}
        projectAppSettingsState={projectAppSettingsState}
        savingAppIds={savingAppIds}
        onToggleProjectApp={handleToggleApp}
        onOverrideProjectCredentials={handleOverrideCredentials}
        onRevertProjectToMaster={handleRevertToMaster}
        onProjectFieldChange={handleFieldChange}
        
        githubSettings={{
          repoUrl,
          setRepoUrl,
          specLocation,
          setSpecLocation,
          baseBranch,
          setBaseBranch,
          specBranchPrefix,
          setSpecBranchPrefix,
          email,
          setEmail,
        }}
      />
    </div>
  )
}
