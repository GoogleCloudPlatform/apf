// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package workspace

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"google.com/apf/api/db"
	"google.com/apf/api/integrations"
)

// HandleAttachRemoteRepo handles Scenario 1: Local Git Repo -> Remote Connected Git Repo.
// If a local .git repository exists in the workspace folder, it configures the origin remote,
// fetches the base branch, aligns the spec branch, and pushes local commits to origin.
// If no .git exists, it does nothing to the filesystem (workspace was never opened).
func HandleAttachRemoteRepo(ctx context.Context, organizationID string, userId string, project *db.Project, repoURL string, settings map[string]interface{}) error {
	client := getGitClient()
	c, ok := client.(*GitClient)
	if !ok || c == nil {
		return nil
	}

	clonePath := filepath.Join(c.WorkspaceMountPath, organizationID, userId, project.ID)
	gitDir := filepath.Join(clonePath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		slog.Info("HandleAttachRemoteRepo: No local .git repository found (workspace not yet opened), skipping remote attach", "projectId", project.ID)
		return nil
	}

	slog.Info("HandleAttachRemoteRepo: Local .git repository detected. Attaching remote origin", "projectId", project.ID, "repoURL", repoURL)

	// Configure or update origin remote
	_, _ = c.runGit(ctx, clonePath, "", "remote", "remove", defaultRemoteName)
	if _, errRemote := c.runGit(ctx, clonePath, "", "remote", "add", defaultRemoteName, repoURL); errRemote != nil {
		return fmt.Errorf("failed to add origin remote: %w", errRemote)
	}

	accessToken, _ := integrations.ResolveGitHubToken(ctx, userId, project.ID)

	baseBranch := "main"
	if project.BaseBranch != nil && *project.BaseBranch != "" {
		baseBranch = *project.BaseBranch
	} else if currentBranch, errHead := c.runGit(ctx, clonePath, "", "symbolic-ref", "--short", "HEAD"); errHead == nil && currentBranch != "" {
		baseBranch = currentBranch
	}

	specBranch := ResolveUserSpecBranchName(ctx, project.GetSpecBranchPrefix(), userId, clonePath)
	if specBranch == "" || specBranch == baseBranch {
		specBranch = GenerateDynamicBranchName(project.GetSpecBranchPrefix(), "", userId)
	}

	repoCfg := &GitRepoConfig{
		RepoURL:         repoURL,
		RepoAccessToken: accessToken,
		BaseBranch:      baseBranch,
		SpecBranch:      specBranch,
		UserId:          userId,
		ProjectId:       project.ID,
		SpecLocation:    project.GetSpecLocation(),
	}

	// Ensure branches are created and workspace is on spec branch
	if errEnsure := c.ensureBranches(ctx, repoCfg, clonePath); errEnsure != nil {
		slog.Warn("HandleAttachRemoteRepo: Failed to ensure branches during attach", "error", errEnsure)
	}

	// Push commits to origin if token is available
	if accessToken != "" {
		errPush := c.Push(ctx, organizationID, repoCfg)
		if errPush != nil {
			slog.Warn("HandleAttachRemoteRepo: Failed to push local commits to origin", "error", errPush)
		} else {
			slog.Info("HandleAttachRemoteRepo: Successfully pushed commits to origin on spec branch", "projectId", project.ID, "branch", specBranch)
		}
	}

	return nil
}

// HandleDetachRemoteRepo handles Scenario 2: Remote Connected Git Repo -> No Git Repo (Disconnecting Remote).
// If a local .git repository exists, it removes the origin remote, keeping local files and commits intact.
func HandleDetachRemoteRepo(ctx context.Context, organizationID string, userId string, projectId string) error {
	client := getGitClient()
	c, ok := client.(*GitClient)
	if !ok || c == nil {
		return nil
	}

	clonePath := filepath.Join(c.WorkspaceMountPath, organizationID, userId, projectId)
	gitDir := filepath.Join(clonePath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return nil
	}

	if _, errDelete := c.runGit(ctx, clonePath, "", "remote", "remove", defaultRemoteName); errDelete != nil {
		slog.Warn("HandleDetachRemoteRepo: remove remote returned error", "error", errDelete)
	} else {
		slog.Info("HandleDetachRemoteRepo: Successfully removed origin remote", "projectId", projectId)
	}

	return nil
}

// HandleSwitchRemoteRepo handles Scenario 3: Change in Remotely Connected Git Repo (Repo A -> Repo B).
// Purges the local workspace directory and deletes existing Firestore DB artifacts for the project,
// then clones Repo B and synchronizes any discovered artifacts into Firestore DB.
func HandleSwitchRemoteRepo(ctx context.Context, organizationID string, userId string, project *db.Project, repoURL string, settings map[string]interface{}) error {
	client := getGitClient()
	c, ok := client.(*GitClient)
	if !ok || c == nil {
		return nil
	}

	clonePath := filepath.Join(c.WorkspaceMountPath, organizationID, userId, project.ID)
	if _, err := os.Stat(clonePath); err == nil {
		slog.Info("HandleSwitchRemoteRepo: Purging existing local workspace for remote repository switch", "path", clonePath, "projectId", project.ID)
		if err := os.RemoveAll(clonePath); err != nil {
			return fmt.Errorf("failed to purge workspace directory on remote switch: %w", err)
		}
	}

	// Purge all existing artifacts for this project from Firestore DB
	if ArtifactStore != nil {
		existingArtifacts, errList := ArtifactStore.ListArtifactsForProject(ctx, organizationID, project.ID)
		if errList == nil {
			for _, art := range existingArtifacts {
				_ = ArtifactStore.DeleteArtifact(ctx, organizationID, art.ID)
			}
			slog.Info("HandleSwitchRemoteRepo: Successfully purged existing artifacts from Firestore DB", "projectId", project.ID, "count", len(existingArtifacts))
		} else {
			slog.Error("HandleSwitchRemoteRepo: Failed to list existing artifacts for purge", "projectId", project.ID, "error", errList)
		}
	}

	if repoURL == "" {
		return nil
	}

	accessToken, _ := integrations.ResolveGitHubToken(ctx, userId, project.ID)
	baseBranch := "main"
	if b, ok := settings["baseBranch"].(string); ok && b != "" {
		baseBranch = b
	}
	specBranchPrefix := "specs"
	if p, ok := settings["specBranchPrefix"].(string); ok && p != "" {
		specBranchPrefix = p
	}
	specBranch := fmt.Sprintf("%s/%s", specBranchPrefix, baseBranch)

	repoCfg := &GitRepoConfig{
		RepoURL:         repoURL,
		RepoAccessToken: accessToken,
		BaseBranch:      baseBranch,
		SpecBranch:      specBranch,
		UserId:          userId,
		ProjectId:       project.ID,
		SpecLocation:    project.GetSpecLocation(),
	}

	// Clone the new repository (Repo B)
	errClone := c.EnsureRepoInitialized(ctx, organizationID, repoCfg)
	if errClone != nil {
		slog.Warn("HandleSwitchRemoteRepo: Failed to clone new remote repository after switch", "repoURL", repoURL, "error", errClone)
		return nil
	}

	// Discover and synchronize any artifacts present in Repo B into Firestore DB
	added, deleted, errSync := SyncProjectArtifactsFromDisk(ctx, organizationID, userId, project, true)
	if errSync != nil {
		slog.Error("HandleSwitchRemoteRepo: Failed to sync new repository artifacts to Firestore DB", "projectId", project.ID, "error", errSync)
	} else {
		slog.Info("HandleSwitchRemoteRepo: Synchronized new repository artifacts to Firestore DB", "projectId", project.ID, "added", added, "deleted", deleted)
	}

	return nil
}
