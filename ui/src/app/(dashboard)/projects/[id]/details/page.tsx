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

import { use } from "react"
import ProjectIdentity from "@/components/project/ProjectIdentity"
import { ArrowLeft, Trash2, Save, Loader2 } from "lucide-react"
import { useQueryClient } from "@tanstack/react-query"
import { useProjectDetails } from "./_hooks/useProjectDetails"
import GeneralConfigCard from "./_components/GeneralConfigCard"
import BmadConfigCard from "./_components/BmadConfigCard"
import AppCredentialsSection from "./_components/AppCredentialsSection"
import EmojiIconPickerModal from "./_components/EmojiIconPickerModal"
import DeleteProjectModal from "./_components/DeleteProjectModal"
import PermissionWarningModal from "./_components/PermissionWarningModal"
import DuplicateRepoModal from "./_components/DuplicateRepoModal"
import BaseBranchConfirmationModal from "./_components/BaseBranchConfirmationModal"
import RepoChangeWarningModal from "./_components/RepoChangeWarningModal"

interface PageParams {
  id: string
}

export default function ProjectDetailsPage({ params }: { params: Promise<PageParams> }) {
  const { id: projectId } = use(params)
  const queryClient = useQueryClient()
  
  const {
    isNew,
    isLoading,
    name,
    setName,
    description,
    setDescription,
    visibility,
    setVisibility,
    icon,
    setIcon,
    specLocation,
    setSpecLocation,
    baseBranch,
    setBaseBranch,
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
    joinProjectPending,
    handleJoinProject,
    handleGoToProject,
    handleConfirmSave,
    handleSubmit,
    handleDelete,
    handleToggleApp,
    handleFieldChange,
    handleOverrideCredentials,
    handleRevertToMaster,
    handleCancel,
    savePending,
    deletePending,
  } = useProjectDetails(projectId)

  if (isLoading) {
    return (
      <div className="flex h-[350px] w-full flex-col items-center justify-center gap-3 text-muted-foreground select-none">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
        <span className="text-sm font-medium">Loading project configurations...</span>
      </div>
    )
  }

  return (
    <div className="p-6 max-w-4xl space-y-6">
      
      {/* Header operations controls */}
      <div className="flex items-center justify-between border-b border-border pb-4 select-none">
        <div className="flex items-center gap-3">
          <button
            onClick={handleCancel}
            className="flex h-9 w-9 items-center justify-center rounded-lg text-muted-foreground hover:bg-muted hover:text-foreground border border-input bg-background"
          >
            <ArrowLeft className="h-4 w-4" />
          </button>
          
          <ProjectIdentity
            icon={icon}
            name={name}
            id={projectId}
            isNew={isNew}
            onIconClick={() => setIsIconPickerOpen(true)}
            size="lg"
          />
        </div>

        <div className="flex gap-2">
          <button
            onClick={handleCancel}
            className="inline-flex h-9 items-center justify-center rounded-lg border border-input bg-background px-4 py-2 text-xs font-semibold text-foreground hover:bg-muted"
          >
            Cancel
          </button>

          {!isNew && (
            <button
              onClick={() => setIsDeleteConfirmOpen(true)}
              className="inline-flex h-9 items-center justify-center gap-1.5 rounded-lg border border-destructive bg-transparent px-4 py-2 text-xs font-semibold text-destructive hover:bg-destructive/10"
            >
              <Trash2 className="h-3.5 w-3.5" />
              Delete
            </button>
          )}

          <button
            id="tour-project-save-button"
            onClick={(e) => handleSubmit(e)}
            disabled={savePending || isValidatingPermissions || isCheckingRepo}
            className="inline-flex h-9 items-center justify-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-xs font-semibold text-primary-foreground shadow transition-all hover:opacity-90 disabled:opacity-50"
          >
            {savePending || isValidatingPermissions || isCheckingRepo ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <Save className="h-3.5 w-3.5" />
            )}
            {isCheckingRepo ? "Checking..." : isValidatingPermissions ? "Verifying..." : "Save"}
          </button>
        </div>
      </div>

      <form onSubmit={handleSubmit} className="space-y-6">
        <GeneralConfigCard
          name={name}
          setName={setName}
          description={description}
          setDescription={setDescription}
          icon={icon}
          onChooseIcon={() => setIsIconPickerOpen(true)}
          visibility={visibility}
          setVisibility={setVisibility}
          disableVisibility={projectId === "foundry-showcase-project"}
        />

        <BmadConfigCard
          specFolder={specFolder}
          setSpecFolder={setSpecFolder}
          implementationArtifactsPath={implementationArtifactsPath}
          setImplementationArtifactsPath={setImplementationArtifactsPath}
          planningArtifactsPath={planningArtifactsPath}
          setPlanningArtifactsPath={setPlanningArtifactsPath}
          testingArtifactsPath={testingArtifactsPath}
          setTestingArtifactsPath={setTestingArtifactsPath}
        />

        <AppCredentialsSection
          isNew={isNew}
          projectId={projectId}
          appsToShow={appsToShow}
          projectAppSettingsState={projectAppSettingsState}
          expandedAppIds={expandedAppIds}
          setExpandedAppIds={setExpandedAppIds}
          savingAppIds={savingAppIds}
          globalCredentials={globalCredentials}
          handleToggleApp={handleToggleApp}
          handleFieldChange={handleFieldChange}
          handleOverrideCredentials={handleOverrideCredentials}
          handleRevertToMaster={handleRevertToMaster}
          repoUrl={repoUrl}
          setRepoUrl={setRepoUrl}
          specLocation={specLocation}
          setSpecLocation={setSpecLocation}
          baseBranch={baseBranch}
          setBaseBranch={setBaseBranch}
          specBranchPrefix={specBranchPrefix}
          setSpecBranchPrefix={setSpecBranchPrefix}
          email={gitEmail}
          setEmail={setGitEmail}
        />

        {/* Bottom operation controls */}
        <div className="flex items-center justify-end gap-2 border-t border-border pt-6 select-none">
          <button
            type="button"
            onClick={handleCancel}
            className="inline-flex h-9 items-center justify-center rounded-lg border border-input bg-background px-4 py-2 text-xs font-semibold text-foreground hover:bg-muted"
          >
            Cancel
          </button>

          {!isNew && (
            <button
              type="button"
              onClick={() => setIsDeleteConfirmOpen(true)}
              className="inline-flex h-9 items-center justify-center gap-1.5 rounded-lg border border-destructive bg-transparent px-4 py-2 text-xs font-semibold text-destructive hover:bg-destructive/10"
            >
              <Trash2 className="h-3.5 w-3.5" />
              Delete
            </button>
          )}

          <button
            type="submit"
            disabled={savePending || isValidatingPermissions || isCheckingRepo}
            className="inline-flex h-9 items-center justify-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-xs font-semibold text-primary-foreground shadow transition-all hover:opacity-90 disabled:opacity-50"
          >
            {savePending || isValidatingPermissions || isCheckingRepo ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <Save className="h-3.5 w-3.5" />
            )}
            {isCheckingRepo ? "Checking..." : isValidatingPermissions ? "Verifying..." : "Save"}
          </button>
        </div>
      </form>

      <EmojiIconPickerModal
        isOpen={isIconPickerOpen}
        onClose={() => setIsIconPickerOpen(false)}
        selectedIcon={icon}
        onSelectIcon={setIcon}
      />

      <DeleteProjectModal
        isOpen={isDeleteConfirmOpen}
        onClose={() => setIsDeleteConfirmOpen(false)}
        onConfirm={handleDelete}
        projectName={name}
        isPending={deletePending}
      />

      <PermissionWarningModal
        isOpen={isPermissionDialogOpen}
        onClose={() => setIsPermissionDialogOpen(false)}
        onConfirm={handleConfirmSave}
        permissions={permissionsCheck}
        isPending={savePending}
      />

      <DuplicateRepoModal
        isOpen={isDuplicateDialogOpen}
        onClose={() => setIsDuplicateDialogOpen(false)}
        onJoinProject={handleJoinProject}
        onGoToProject={handleGoToProject}
        projectName={duplicateProject?.name || ""}
        repoUrl={repoUrl}
        hasAccess={duplicateProject?.hasAccess || false}
        potentialRole={duplicateProject?.potentialRole || ""}
        isPending={joinProjectPending}
      />

      <BaseBranchConfirmationModal
        isOpen={isBaseBranchModalOpen}
        onClose={() => setIsBaseBranchModalOpen(false)}
        onConfirm={handleConfirmBaseBranch}
        baseBranch={missingBaseBranchName || baseBranch}
        repoUrl={repoUrl}
        isPending={savePending}
      />

      <RepoChangeWarningModal
        isOpen={isRepoChangeModalOpen}
        onClose={() => setIsRepoChangeModalOpen(false)}
        onConfirm={handleConfirmRepoChange}
        initialRepoUrl={initialRepoUrl}
        newRepoUrl={repoUrl}
        isPending={savePending || isValidatingPermissions}
      />
    </div>
  )
}

