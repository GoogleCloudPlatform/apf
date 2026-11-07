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
	"regexp"

	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"google.com/apf/api/agent"
	core "google.com/apf/api/core"
	"google.com/apf/api/db"
	"google.com/apf/api/integrations"

	"cloud.google.com/go/storage"
)

var ArtifactStore db.ArtifactStore
var SessionStore db.SessionStore
var AppConfig *core.Configuration
var ConfigStore db.ConfigStore
var UserProjectStore db.UserProjectStore
var NFSStore db.NFSStore
var UserStore db.UserStore

// AsyncRunner defines the function signature used to execute background tasks.
type AsyncRunner func(fn func())

var defaultAsyncRunner AsyncRunner = func(fn func()) {
	go fn()
}

// SetDefaultAsyncRunner configures the package-level async runner (e.g. for deterministic unit testing).
func SetDefaultAsyncRunner(runner AsyncRunner) {
	if runner == nil {
		defaultAsyncRunner = func(fn func()) { go fn() }
	} else {
		defaultAsyncRunner = runner
	}
}

func runAsync(fn func()) {
	if defaultAsyncRunner != nil {
		defaultAsyncRunner(fn)
	} else {
		go fn()
	}
}

// Handler manages workspace-related operations with dependency injection.
type Handler struct {
	ArtifactStore    db.ArtifactStore
	SessionStore     db.SessionStore
	ConfigStore      db.ConfigStore
	UserProjectStore db.UserProjectStore
	NFSStore         db.NFSStore
	UserStore        db.UserStore
	AppConfig        *core.Configuration
	GitClient        GitClientInterface
	AsyncRunner      AsyncRunner
}

// NewHandler creates a new Handler instance with injected dependencies.
func NewHandler(
	artifactStore db.ArtifactStore,
	sessionStore db.SessionStore,
	configStore db.ConfigStore,
	userProjectStore db.UserProjectStore,
	nfsStore db.NFSStore,
	userStore db.UserStore,
	appConfig *core.Configuration,
) *Handler {
	return &Handler{
		ArtifactStore:    artifactStore,
		SessionStore:     sessionStore,
		ConfigStore:      configStore,
		UserProjectStore: userProjectStore,
		NFSStore:         nfsStore,
		UserStore:        userStore,
		AppConfig:        appConfig,
		AsyncRunner:      defaultAsyncRunner,
	}
}

var gitClient GitClientInterface
var gitClientOnce sync.Once

// getWorkspaceMountPath resolves the workspace mount path from AppConfig or core.NewConfig().
// If WorkspaceMountPath is missing or empty in configuration, it logs a clear error and panics.
func getWorkspaceMountPath() string {
	if AppConfig != nil {
		return AppConfig.GetWorkspaceMountPath()
	}
	cfg := core.NewConfig()
	if cfg != nil {
		return cfg.GetWorkspaceMountPath()
	}
	slog.Error("CRITICAL CONFIGURATION ERROR: WorkspaceMountPath is missing or empty in configuration")
	panic("Configuration error: WorkspaceMountPath is missing or empty in configuration")
}

func getGitClient() GitClientInterface {
	if gitClient == nil {
		gitClientOnce.Do(func() {
			mountPath := getWorkspaceMountPath()
			c := NewGitClient(mountPath)
			if NFSStore != nil {
				c.WithNFSStore(NFSStore)
			}
			gitClient = c
		})
	}
	return gitClient
}

func GetGitClient() GitClientInterface {
	return getGitClient()
}

// SetGitClient sets the active GitClientInterface instance (useful for testing).
func SetGitClient(c GitClientInterface) {
	gitClient = c
}

// ResetGitClient resets the cached GitClientInterface instance.
func ResetGitClient() {
	gitClient = nil
	gitClientOnce = sync.Once{}
}

func getFileSystemArtifactService(mountPath string) *core.FileSystemArtifactService {
	svc := core.NewFileSystemArtifactService(mountPath)
	if NFSStore != nil {
		svc.WithNFSStore(NFSStore)
	}
	return svc
}

type Response struct {
	Title       string `json:"title,omitempty"`
	Message     string `json:"message"`
	Status      string `json:"status"`
	Description string `json:"description,omitempty"`
}

func sendJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, x-goog-authenticated-user-email, x-goog-authenticated-user-id, X-Organization-Id")
	w.WriteHeader(statusCode)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			slog.Error("Error encoding response", "error", err)
		}
	}
}

func init() {
	agent.OnFileWriteHook = func(ctx context.Context, projectId string, organizationID string, phase string, filePath string) {
		slog.Info("OnFileWriteHook: File write detected", "projectId", projectId, "phase", phase, "filePath", filePath)

		cleanPath := filepath.ToSlash(filepath.Clean(filePath))
		if strings.HasPrefix(cleanPath, ".tmp/") || cleanPath == ".tmp" || strings.Contains(cleanPath, "/.tmp/") {
			slog.Debug("OnFileWriteHook: Skipping temporary scratch file", "projectId", projectId, "filePath", filePath)
			return
		}

		// Ensure artifact is registered and set to In Progress in Firestore
		if err := SyncOrRegisterArtifact(ctx, organizationID, projectId, phase, filePath); err != nil {
			slog.Error("OnFileWriteHook: Failed to sync or register artifact in Firestore", "projectId", projectId, "filePath", filePath, "error", err)
		}

		// Recalculate gates to transition phase status to In Progress
		if err := recalculateGates(ctx, organizationID, projectId); err != nil {
			slog.Error("OnFileWriteHook: Failed to recalculate gates after file write", "projectId", projectId, "error", err)
		}

		// Commit the file changes locally on the API tier
		claims, ok := ctx.Value(core.UserClaimsKey).(*core.UserClaims)
		if !ok || claims == nil {
			slog.Warn("OnFileWriteHook: Failed to get user claims from context, skipping git commit")
			return
		}
		userId := claims.Subject

		project, err := db.ProjectStoreInstance.GetProject(ctx, organizationID, projectId)
		if err != nil {
			slog.Error("OnFileWriteHook: Failed to get project", "projectId", projectId, "error", err)
			return
		}

		repoURL := project.GetRepoUrl()
		var accessToken string
		if repoURL != "" {
			var errToken error
			accessToken, errToken = integrations.ResolveGitHubToken(ctx, userId, projectId)
			if errToken != nil {
				slog.Error("OnFileWriteHook: Failed to resolve GitHub token", "projectId", projectId, "userId", userId, "error", errToken)
				return
			}
		}

		baseBranch, specBranch, errResolve := ResolveSpecBranchNoRequest(ctx, organizationID, projectId, userId, phase)
		if errResolve != nil {
			slog.Error("OnFileWriteHook: Failed to resolve spec branch", "projectId", projectId, "error", errResolve)
			return
		}

		email := project.GetGitHubEmail()
		if email == "" {
			if resEmail, err := integrations.ResolveGitHubEmail(ctx, userId, projectId); err == nil && resEmail != "" {
				email = resEmail
			}
		}
		if email == "" && claims != nil {
			email = claims.Subject
		}
		name := email
		if atIdx := strings.Index(email, "@"); atIdx != -1 {
			name = email[:atIdx]
		}

		gitRepoCfg := &GitRepoConfig{
			RepoURL:         repoURL,
			RepoAccessToken: accessToken,
			BaseBranch:      baseBranch,
			SpecBranch:      specBranch,
			UserId:          userId,
			ProjectId:       projectId,
			SpecLocation:    project.GetSpecLocation(),
			AuthorEmail:     email,
			AuthorName:      name,
		}

		client := getGitClient()

		commitMsg := fmt.Sprintf("Auto-commit artifact: %s", filepath.Base(filePath))

		errCommit := client.LocalCommit(ctx, organizationID, gitRepoCfg, filePath, commitMsg, name, email)
		if errCommit != nil {
			slog.Error("OnFileWriteHook: Failed to perform local git commit", "projectId", projectId, "filePath", filePath, "error", errCommit)
		} else {
			slog.Info("OnFileWriteHook: Local git commit completed successfully", "projectId", projectId, "filePath", filePath)
		}
	}
}

// ListArtifactsHandler lists all artifacts for a project.
func ListArtifactsHandler(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("id")

	organizationID, err := core.GetOrganizationID(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized context", http.StatusUnauthorized)
		return
	}

	projectArtifacts, err := ArtifactStore.ListArtifactsForProject(r.Context(), organizationID, projectId)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to list artifacts: %v", err)})
		return
	}

	sendJSON(w, http.StatusOK, projectArtifacts)
}

// UpdateArtifactHandler updates an artifact.
func UpdateArtifactHandler(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("id")
	artifactId := r.PathValue("artifactId")

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		http.Error(w, "Unauthorized context", http.StatusUnauthorized)
		return
	}

	var a db.Artifact
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
		return
	}
	a.ID = artifactId
	a.ProjectId = projectId

	// Apply Path and Title updates
	if a.Title != "" && strings.Contains(a.Title, "/") {
		// Title is the last string after the last / in the path
		a.Path = a.Title
		a.Title = path.Base(a.Title)
	}

	if err := ArtifactStore.UpdateArtifact(r.Context(), organizationID, &a); err != nil {
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to update artifact: %v", err)})
		return
	}

	// Broadcast internal artifact update event
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	userId := "System"
	if ok && claims != nil && claims.Subject != "" {
		userId = claims.Subject
	}

	msg := &db.Message{
		Sender:         userId,
		Text:           fmt.Sprintf("Artifact '%s' updated to %s", a.Title, a.Status),
		IsUser:         false,
		Phase:          a.Phase,
		SessionId:      "project_global_" + projectId,
		Timestamp:      time.Now(),
		IsTransient:    true,
		ExpireAt:       time.Now().Add(24 * time.Hour),
		IsNotification: true,
		NotificationDetails: &db.NotificationDetails{
			Name: "artifact-update",
			Params: map[string]interface{}{
				"artifactId": a.ID,
				"title":      a.Title,
				"status":     a.Status,
			},
		},
	}

	if agent.Store != nil {
		if _, err := agent.Store.CreateMessage(r.Context(), organizationID, projectId, userId, msg); err != nil {
			slog.Error("Failed to broadcast artifact update notification", "error", err)
		}
	}

	// Trigger gate status calculation asynchronously
	detachedCtx := context.WithoutCancel(r.Context())
	store := db.ProjectStoreInstance
	artStore := ArtifactStore
	agentStore := agent.Store
	runAsync(func() {
		if err := recalculateGatesWithStores(detachedCtx, organizationID, projectId, store, artStore, agentStore); err != nil {
			slog.Error("Failed to recalculate gates after artifact update", "projectId", projectId, "error", err)
		}
	})

	sendJSON(w, http.StatusOK, a)
}

// DeleteArtifactHandler deletes an artifact or workspace file.
func DeleteArtifactHandler(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("id")
	artifactId := r.PathValue("artifactId")

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		http.Error(w, "Unauthorized context", http.StatusUnauthorized)
		return
	}

	unescapedArtifactId := artifactId
	if unescaped, errUnescape := url.PathUnescape(artifactId); errUnescape == nil && unescaped != "" {
		unescapedArtifactId = unescaped
	}

	// Get artifact to get path and phase
	var targetArt *db.Artifact
	if ArtifactStore != nil {
		if arts, err := ArtifactStore.ListArtifactsForProject(r.Context(), organizationID, projectId); err == nil {
			for i := range arts {
				a := &arts[i]
				if a.ID == artifactId || a.Path == artifactId || a.Title == artifactId ||
					a.ID == unescapedArtifactId || a.Path == unescapedArtifactId || a.Title == unescapedArtifactId ||
					a.Name == artifactId || a.Name == unescapedArtifactId ||
					a.StoragePath == artifactId || a.StoragePath == unescapedArtifactId {
					targetArt = a
					break
				}
			}
		}
	}

	targetPath := ""
	if targetArt != nil && targetArt.Path != "" {
		targetPath = targetArt.Path
	} else if targetArt != nil && targetArt.StoragePath != "" {
		targetPath = targetArt.StoragePath
	} else {
		clean := filepath.Clean(unescapedArtifactId)
		cleanLower := strings.ToLower(clean)
		if !strings.HasPrefix(clean, "..") && !filepath.IsAbs(clean) && cleanLower != ".git" && !strings.HasPrefix(cleanLower, ".git/") && !strings.HasPrefix(cleanLower, ".git"+string(filepath.Separator)) {
			targetPath = clean
		}
	}

	if targetArt == nil && targetPath == "" {
		sendJSON(w, http.StatusNotFound, map[string]string{"error": "Artifact or file not found"})
		return
	}

	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	userId := "System"
	if ok && claims != nil && claims.Subject != "" {
		userId = claims.Subject
	}

	// Delete physical file
	appCfg := core.NewConfig()
	var svc core.ArtifactService
	var svcErr error

	if appCfg.ArtifactServiceType == "GCS" {
		cfg, err := ConfigStore.GetGlobalConfig(r.Context())
		if err != nil {
			slog.Error("Failed to get global config for GCS delete", "error", err)
		} else {
			bucketName := strings.TrimPrefix(cfg.GCSWorkspaceBucket, "gs://")
			client, err := storage.NewClient(r.Context())
			if err != nil {
				slog.Error("Failed to create storage client for GCS delete", "error", err)
			} else {
				defer client.Close()
				svc = core.NewGcsArtifactService(client, bucketName)
			}
		}
	} else {
		svc = getFileSystemArtifactService(appCfg.WorkspaceMountPath)
	}

	if svc != nil && targetPath != "" {
		svcErr = svc.DeleteArtifact(r.Context(), organizationID, userId, projectId, targetPath)
		if svcErr != nil {
			if errors.Is(svcErr, os.ErrNotExist) || strings.Contains(svcErr.Error(), "object doesn't exist") || strings.Contains(svcErr.Error(), "no such file") {
				if targetArt == nil {
					sendJSON(w, http.StatusNotFound, map[string]string{"error": "Artifact not found"})
					return
				}
				slog.Info("Physical artifact file already gone", "path", targetPath)
			} else {
				sendJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to delete physical file: %v", svcErr)})
				return
			}
		}
	}

	// Delete from DB if registered artifact
	if targetArt != nil {
		if err := ArtifactStore.DeleteArtifact(r.Context(), organizationID, targetArt.ID); err != nil {
			sendJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to delete artifact from DB: %v", err)})
			return
		}

		// Broadcast internal artifact delete event
		msg := &db.Message{
			Sender:         userId,
			Text:           fmt.Sprintf("Artifact '%s' deleted", targetArt.Title),
			IsUser:         false,
			Phase:          targetArt.Phase,
			SessionId:      "project_global_" + projectId,
			Timestamp:      time.Now(),
			IsTransient:    true,
			ExpireAt:       time.Now().Add(24 * time.Hour),
			IsNotification: true,
			NotificationDetails: &db.NotificationDetails{
				Name: "artifact-delete",
				Params: map[string]interface{}{
					"artifactId": targetArt.ID,
					"title":      targetArt.Title,
				},
			},
		}

		if agent.Store != nil {
			if _, err := agent.Store.CreateMessage(r.Context(), organizationID, projectId, userId, msg); err != nil {
				slog.Error("Failed to broadcast artifact delete notification", "error", err)
			}
		}

		// Trigger gate status calculation asynchronously
		detachedCtx := context.WithoutCancel(r.Context())
		store := db.ProjectStoreInstance
		artStore := ArtifactStore
		agentStore := agent.Store
		runAsync(func() {
			if err := recalculateGatesWithStores(detachedCtx, organizationID, projectId, store, artStore, agentStore); err != nil {
				slog.Error("Failed to recalculate gates after artifact deletion", "projectId", projectId, "error", err)
			}
		})
	}

	sendJSON(w, http.StatusOK, map[string]string{"message": "Artifact deleted successfully"})
}

// GetArtifactContentHandler reads the raw file from GCS or local filesystem and returns it.
func GetArtifactContentHandler(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("id")
	artifactId := r.PathValue("artifactId")

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		http.Error(w, "Unauthorized context: missing organization ID", http.StatusUnauthorized)
		return
	}

	unescapedArtifactId := artifactId
	if unescaped, errUnescape := url.PathUnescape(artifactId); errUnescape == nil && unescaped != "" {
		unescapedArtifactId = unescaped
	}

	var targetArt *db.Artifact
	if ArtifactStore != nil {
		if arts, err := ArtifactStore.ListArtifactsForProject(r.Context(), organizationID, projectId); err == nil {
			for i := range arts {
				a := &arts[i]
				if a.ID == artifactId || a.Path == artifactId || a.Title == artifactId ||
					a.ID == unescapedArtifactId || a.Path == unescapedArtifactId || a.Title == unescapedArtifactId ||
					a.Name == artifactId || a.Name == unescapedArtifactId ||
					a.StoragePath == artifactId || a.StoragePath == unescapedArtifactId {
					targetArt = a
					break
				}
			}
		}
	}

	targetPath := ""
	if targetArt != nil && targetArt.Path != "" {
		targetPath = targetArt.Path
	} else if targetArt != nil && targetArt.StoragePath != "" {
		// Session artifact with explicit storage path
	} else {
		clean := filepath.Clean(unescapedArtifactId)
		if !strings.HasPrefix(clean, "..") && !filepath.IsAbs(clean) {
			targetPath = clean
		}
	}

	if targetArt == nil && targetPath == "" {
		http.Error(w, "Artifact or file not found", http.StatusNotFound)
		return
	}

	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	userId := "System"
	if ok && claims != nil && claims.Subject != "" {
		userId = claims.Subject
	}

	appCfg := core.NewConfig()
	var data []byte
	var readErr error

	if targetArt != nil && targetArt.StoragePath != "" {
		if appCfg.ArtifactServiceType == "GCS" || targetArt.StorageType == "gcs" {
			cfg, err := ConfigStore.GetGlobalConfig(r.Context())
			if err != nil {
				http.Error(w, "Failed to get global config", http.StatusInternalServerError)
				return
			}
			bucketName := strings.TrimPrefix(cfg.GCSWorkspaceBucket, "gs://")

			client, err := storage.NewClient(r.Context())
			if err != nil {
				http.Error(w, "Failed to create storage client", http.StatusInternalServerError)
				return
			}
			defer client.Close()

			svc := core.NewGcsArtifactService(client, bucketName)
			data, readErr = svc.ReadSessionArtifact(r.Context(), targetArt.StorageType, targetArt.StoragePath)
		} else {
			svc := getFileSystemArtifactService(appCfg.WorkspaceMountPath)
			if appCfg.LocalArtifactPath != "" {
				svc = svc.WithLocalArtifactPath(appCfg.LocalArtifactPath)
			}
			data, readErr = svc.ReadSessionArtifact(r.Context(), targetArt.StorageType, targetArt.StoragePath)
		}
	} else {
		if appCfg.ArtifactServiceType == "GCS" {
			cfg, err := ConfigStore.GetGlobalConfig(r.Context())
			if err != nil {
				http.Error(w, "Failed to get global config", http.StatusInternalServerError)
				return
			}
			bucketName := strings.TrimPrefix(cfg.GCSWorkspaceBucket, "gs://")

			client, err := storage.NewClient(r.Context())
			if err != nil {
				http.Error(w, "Failed to create storage client", http.StatusInternalServerError)
				return
			}
			defer client.Close()

			svc := core.NewGcsArtifactService(client, bucketName)
			data, readErr = svc.GetArtifact(r.Context(), organizationID, userId, projectId, targetPath)
		} else {
			svc := getFileSystemArtifactService(appCfg.WorkspaceMountPath)
			data, readErr = svc.GetArtifact(r.Context(), organizationID, userId, projectId, targetPath)
		}
	}

	if readErr != nil {
		http.Error(w, fmt.Sprintf("Failed to read artifact: %v", readErr), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	if targetArt != nil && targetArt.MimeType != "" {
		w.Header().Set("Content-Type", targetArt.MimeType)
	} else if mimeType := mime.TypeByExtension(filepath.Ext(targetPath)); mimeType != "" {
		w.Header().Set("Content-Type", mimeType)
	} else if mimeType := mime.TypeByExtension(filepath.Ext(unescapedArtifactId)); mimeType != "" {
		w.Header().Set("Content-Type", mimeType)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	w.Write(data)
}

// SaveArtifactContentHandler writes modified file/artifact content back to storage.
func SaveArtifactContentHandler(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("id")
	artifactId := r.PathValue("artifactId")

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		http.Error(w, "Unauthorized context: missing organization ID", http.StatusUnauthorized)
		return
	}

	unescapedArtifactId := artifactId
	if unescaped, errUnescape := url.PathUnescape(artifactId); errUnescape == nil && unescaped != "" {
		unescapedArtifactId = unescaped
	}

	var targetArt *db.Artifact
	if ArtifactStore != nil {
		if arts, err := ArtifactStore.ListArtifactsForProject(r.Context(), organizationID, projectId); err == nil {
			for i := range arts {
				a := &arts[i]
				if a.ID == artifactId || a.Path == artifactId || a.Title == artifactId ||
					a.ID == unescapedArtifactId || a.Path == unescapedArtifactId || a.Title == unescapedArtifactId {
					targetArt = a
					break
				}
			}
		}
	}

	targetPath := ""
	if targetArt != nil && targetArt.Path != "" {
		targetPath = targetArt.Path
	} else {
		clean := filepath.Clean(unescapedArtifactId)
		if !strings.HasPrefix(clean, "..") && !filepath.IsAbs(clean) {
			targetPath = clean
		}
	}

	if targetPath == "" {
		http.Error(w, "Artifact or file not found", http.StatusNotFound)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var data []byte
	var jsonPayload struct {
		Content *string `json:"content"`
		Data    *string `json:"data"`
	}
	if len(bodyBytes) > 0 && json.Unmarshal(bodyBytes, &jsonPayload) == nil && (jsonPayload.Content != nil || jsonPayload.Data != nil) {
		if jsonPayload.Content != nil {
			data = []byte(*jsonPayload.Content)
		} else {
			data = []byte(*jsonPayload.Data)
		}
	} else {
		data = bodyBytes
	}

	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	userId := "System"
	if ok && claims != nil && claims.Subject != "" {
		userId = claims.Subject
	}

	appCfg := core.NewConfig()
	var writeErr error

	if appCfg.ArtifactServiceType == "GCS" {
		cfg, err := ConfigStore.GetGlobalConfig(r.Context())
		if err != nil {
			http.Error(w, "Failed to get global config", http.StatusInternalServerError)
			return
		}
		bucketName := strings.TrimPrefix(cfg.GCSWorkspaceBucket, "gs://")

		client, err := storage.NewClient(r.Context())
		if err != nil {
			http.Error(w, "Failed to create storage client", http.StatusInternalServerError)
			return
		}
		defer client.Close()

		svc := core.NewGcsArtifactService(client, bucketName)
		writeErr = svc.WriteArtifact(r.Context(), organizationID, userId, projectId, targetPath, data)
	} else {
		svc := getFileSystemArtifactService(appCfg.WorkspaceMountPath)
		writeErr = svc.WriteArtifact(r.Context(), organizationID, userId, projectId, targetPath, data)
	}

	if writeErr != nil {
		http.Error(w, fmt.Sprintf("Failed to write artifact: %v", writeErr), http.StatusInternalServerError)
		return
	}

	phase := ""
	if targetArt != nil {
		phase = targetArt.Phase
	}
	if phase == "" {
		project, _ := db.ProjectStoreInstance.GetProject(r.Context(), organizationID, projectId)
		phase = db.DetermineArtifactPhaseForProject(targetPath, project)
	}

	if agent.OnFileWriteHook != nil {
		agent.OnFileWriteHook(r.Context(), projectId, organizationID, phase, targetPath)
	} else {
		if err := SyncOrRegisterArtifact(r.Context(), organizationID, projectId, phase, targetPath); err != nil {
			slog.Error("Failed to sync or register artifact after save", "projectId", projectId, "filePath", targetPath, "error", err)
		}
		detachedCtx := context.WithoutCancel(r.Context())
		store := db.ProjectStoreInstance
		artStore := ArtifactStore
		agentStore := agent.Store
		runAsync(func() {
			if err := recalculateGatesWithStores(detachedCtx, organizationID, projectId, store, artStore, agentStore); err != nil {
				slog.Error("Failed to recalculate gates after saving artifact", "projectId", projectId, "error", err)
			}
		})
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"message": "File saved successfully",
		"path":    targetPath,
		"status":  "SUCCESS",
	})
}

func calculateGates(artifacts []db.Artifact) map[string]string {
	gates := make(map[string]string)

	phaseArtifacts := make(map[string][]db.Artifact)
	for _, a := range artifacts {
		phaseArtifacts[a.Phase] = append(phaseArtifacts[a.Phase], a)
	}

	phases := []string{"ONBOARDING", "ANALYSIS", "PLANNING", "SOLUTIONING", "IMPLEMENTATION"}

	for _, phase := range phases {
		arts := phaseArtifacts[phase]
		if len(arts) == 0 {
			gates[phase] = "Not Started"
			continue
		}

		hasOpenedGate := false
		anyWorkDone := false

		for _, a := range arts {
			if a.Status == db.ArtifactStatusComplete || a.Status == db.ArtifactStatusStagedForReview {
				hasOpenedGate = true
			}
			if a.Status != db.ArtifactStatusNotStarted {
				anyWorkDone = true
			}
		}

		if hasOpenedGate {
			gates[phase] = "Complete"
		} else if anyWorkDone {
			gates[phase] = "In Progress"
		} else {
			gates[phase] = "Not Started"
		}
	}

	return gates
}

// recalculateGates recalculates the project governance gates based on artifact completion status,
// updates the project record, and broadcasts the status update.
func recalculateGates(ctx context.Context, organizationID string, projectId string) error {
	return recalculateGatesWithStores(ctx, organizationID, projectId, db.ProjectStoreInstance, ArtifactStore, agent.Store)
}

// recalculateGatesWithStores recalculates governance gates using explicitly passed store instances to prevent data races.
func recalculateGatesWithStores(
	ctx context.Context,
	organizationID string,
	projectId string,
	store db.ProjectStore,
	artStore db.ArtifactStore,
	agentStore db.MessageStore,
) error {
	if artStore == nil || store == nil {
		return nil
	}

	arts, err := artStore.ListArtifactsForProject(ctx, organizationID, projectId)
	if err != nil {
		return fmt.Errorf("failed to list artifacts for gate calculation: %w", err)
	}

	gates := calculateGates(arts)

	project, err := store.GetProject(ctx, organizationID, projectId)
	if err != nil || project == nil {
		return fmt.Errorf("failed to get project for gate calculation: %w", err)
	}

	project.GovernanceGates = &gates
	if err := store.UpdateProject(ctx, organizationID, project); err != nil {
		return fmt.Errorf("failed to update project gates: %w", err)
	}

	if agentStore != nil {
		msg := &db.Message{
			Sender:         "System",
			Text:           "Project governance gates recalculated successfully",
			IsUser:         false,
			SessionId:      "project_global_" + projectId,
			Timestamp:      time.Now(),
			IsTransient:    true,
			ExpireAt:       time.Now().Add(24 * time.Hour),
			IsNotification: true,
			NotificationDetails: &db.NotificationDetails{
				Name: "governance-update",
				Params: map[string]interface{}{
					"projectId": projectId,
				},
			},
		}
		if _, err := agentStore.CreateMessage(ctx, organizationID, projectId, "System", msg); err != nil {
			slog.Error("Failed to broadcast governance update", "error", err)
		}
	}

	return nil
}

func GitPullHandler(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("id")
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized: User claims missing", Status: "ERROR"})
		return
	}
	userId := claims.Subject

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized: Organization ID missing", Status: "ERROR"})
		return
	}

	project, err := db.ProjectStoreInstance.GetProject(r.Context(), organizationID, projectId)
	if err != nil {
		sendJSON(w, http.StatusNotFound, Response{Message: fmt.Sprintf("Project not found: %v", err), Status: "ERROR"})
		return
	}
	repoURL := project.GetRepoUrl()
	if repoURL == "" {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Repository URL is empty on project configuration", Status: "ERROR"})
		return
	}

	accessToken, errToken := integrations.ResolveGitHubToken(r.Context(), userId, projectId)
	if errToken != nil {
		if errors.Is(errToken, integrations.ErrGitHubCredsNotConfigured) {
			sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: " + errToken.Error(), Status: "ERROR"})
		} else {
			sendJSON(w, http.StatusInternalServerError, Response{Message: "Internal Error: " + errToken.Error(), Status: "ERROR"})
		}
		return
	}

	baseBranch, specBranch, errResolve := resolveSpecBranch(r.Context(), organizationID, projectId, userId, r)
	if errResolve != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to resolve spec branch: %v", errResolve), Status: "ERROR"})
		return
	}

	gitRepoCfg := &GitRepoConfig{
		RepoURL:         repoURL,
		RepoAccessToken: accessToken,
		BaseBranch:      baseBranch,
		SpecBranch:      specBranch,
		UserId:          userId,
		ProjectId:       projectId,
		SpecLocation:    project.GetSpecLocation(),
	}

	client := getGitClient()
	errPull := client.Pull(r.Context(), organizationID, gitRepoCfg)
	if errPull != nil {
		if errors.Is(errPull, ErrWriteAccessDenied) {
			sendJSON(w, http.StatusForbidden, Response{
				Title:       "Access Denied",
				Message:     "Failed to pull from remote repository: write access not granted or token expired.",
				Description: "Please reconnect your GitHub credentials in Settings.",
				Status:      "ERROR",
			})
			return
		}
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to pull from remote: %v", errPull), Status: "ERROR"})
		return
	}

	// Resynchronize database with pulled repository contents
	added, deleted, errSync := SyncProjectArtifactsFromDisk(r.Context(), organizationID, userId, project, true)
	if errSync != nil {
		slog.Error("pullProjectSpecHandler: Failed to synchronize artifacts after pull", "projectId", projectId, "error", errSync)
	} else {
		slog.Info("pullProjectSpecHandler: Synchronized artifacts after pull", "projectId", projectId, "added", added, "deleted", deleted)
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"status":      "SUCCESS",
		"message":     "Workspace specifications successfully re-hydrated from remote Git branch",
		"stagedFiles": 0,
		"branch":      specBranch,
	})
}

type GitSyncRequest struct {
	FilePath    string   `json:"filePath,omitempty"`
	ArtifactID  string   `json:"artifactId,omitempty"`
	Action      string   `json:"action,omitempty"`
	Direction   string   `json:"direction,omitempty"`
	StagedPaths []string `json:"stagedPaths,omitempty"`
}

// GitStatusHandler returns the working tree status (modified, untracked, added, deleted files).
func GitStatusHandler(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("id")
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized: User claims missing", Status: "ERROR"})
		return
	}
	userId := claims.Subject

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized: Organization ID missing", Status: "ERROR"})
		return
	}

	project, err := db.ProjectStoreInstance.GetProject(r.Context(), organizationID, projectId)
	if err != nil {
		sendJSON(w, http.StatusNotFound, Response{Message: fmt.Sprintf("Project not found: %v", err), Status: "ERROR"})
		return
	}

	baseBranch, targetBranch, _ := resolveSpecBranch(r.Context(), organizationID, projectId, userId, r)

	repoCfg := &GitRepoConfig{
		RepoURL:      project.GetRepoUrl(),
		BaseBranch:   baseBranch,
		SpecBranch:   targetBranch,
		UserId:       userId,
		ProjectId:    projectId,
		SpecLocation: project.GetSpecLocation(),
	}

	statusItems, err := getGitClient().GetWorkspaceStatus(r.Context(), organizationID, repoCfg)
	if err != nil {
		slog.Error("Failed to fetch git status", "projectId", projectId, "error", err)
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to get git status: %v", err), Status: "ERROR"})
		return
	}

	sendJSON(w, http.StatusOK, statusItems)
}

func GitSyncHandler(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("id")
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized: User claims missing", Status: "ERROR"})
		return
	}
	userId := claims.Subject

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized: Organization ID missing", Status: "ERROR"})
		return
	}

	var reqBody GitSyncRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid request payload", Status: "ERROR"})
		return
	}

	project, err := db.ProjectStoreInstance.GetProject(r.Context(), organizationID, projectId)
	if err != nil {
		sendJSON(w, http.StatusNotFound, Response{Message: fmt.Sprintf("Project not found: %v", err), Status: "ERROR"})
		return
	}

	isReadOnly := false
	userProj, errUserProj := UserProjectStore.GetProjectAppSetting(r.Context(), organizationID, userId, projectId, "github")
	if errUserProj == nil && userProj != nil {
		isReadOnly = userProj.IsReadOnly
	}

	if isReadOnly {
		sendJSON(w, http.StatusForbidden, Response{
			Title:       "Operation Forbidden",
			Message:     "Remote sync operations are disabled because your workspace is running in Read-Only Mode (Read-Only Repository Access).",
			Description: "Pushing changes to the remote repository is not permitted when in read-only mode.",
			Status:      "ERROR",
		})
		return
	}

	repoURL := project.GetRepoUrl()
	if repoURL == "" {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Repository URL is empty on project configuration", Status: "ERROR"})
		return
	}

	// 1. Resolve Auth Token
	accessToken, errToken := integrations.ResolveGitHubToken(r.Context(), userId, projectId)
	if errToken != nil {
		if errors.Is(errToken, integrations.ErrGitHubCredsNotConfigured) {
			sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: " + errToken.Error(), Status: "ERROR"})
		} else {
			sendJSON(w, http.StatusInternalServerError, Response{Message: "Internal Error: " + errToken.Error(), Status: "ERROR"})
		}
		return
	}

	// 2. Resolve target and base branch configurations
	baseBranch, targetBranch, errResolve := resolveSpecBranch(r.Context(), organizationID, projectId, userId, r)
	if errResolve != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to resolve spec branch: %v", errResolve), Status: "ERROR"})
		return
	}

	if targetBranch == baseBranch {
		sendJSON(w, http.StatusBadRequest, Response{
			Message: "Base branch and target specification branch cannot be identical for gated PR synchronization.",
			Status:  "ERROR",
		})
		return
	}

	email := project.GetGitHubEmail()
	if email == "" {
		if resEmail, err := integrations.ResolveGitHubEmail(r.Context(), userId, projectId); err == nil && resEmail != "" {
			email = resEmail
		}
	}
	if email == "" && claims != nil {
		email = claims.Subject
	}
	name := email
	if atIdx := strings.Index(email, "@"); atIdx != -1 {
		name = email[:atIdx]
	}

	gitRepoCfg := &GitRepoConfig{
		RepoURL:         repoURL,
		RepoAccessToken: accessToken,
		BaseBranch:      baseBranch,
		SpecBranch:      targetBranch,
		UserId:          userId,
		ProjectId:       projectId,
		SpecLocation:    project.GetSpecLocation(),
		AuthorEmail:     email,
		AuthorName:      name,
	}

	client := getGitClient()

	if reqBody.Direction == "pull" || reqBody.Action == "pull" {
		slog.Info("GitSyncHandler: executing Pull Mode", "projectId", projectId, "baseBranch", baseBranch, "targetBranch", targetBranch)
		errPull := client.Pull(r.Context(), organizationID, gitRepoCfg)
		if errPull != nil {
			if strings.Contains(errPull.Error(), "conflict") || strings.Contains(errPull.Error(), "rebase failed") {
				sendJSON(w, http.StatusConflict, Response{
					Title:       "Merge Conflict",
					Message:     fmt.Sprintf("Conflict detected while rebasing '%s' onto latest '%s'. Your local changes have been safely preserved.", targetBranch, baseBranch),
					Description: errPull.Error(),
					Status:      "ERROR",
				})
				return
			}
			sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to pull remote changes: %v", errPull), Status: "ERROR"})
			return
		}

		sendJSON(w, http.StatusOK, map[string]interface{}{
			"status":     "SUCCESS",
			"message":    fmt.Sprintf("Pulled changes from '%s' into '%s' and updated local workspace.", baseBranch, targetBranch),
			"branchName": targetBranch,
		})
		return
	}

	if len(reqBody.StagedPaths) > 0 {
		for _, p := range reqBody.StagedPaths {
			_ = client.StagePath(r.Context(), organizationID, gitRepoCfg, p)
		}
		// Also stage .gitignore if present in workspace
		_ = client.StagePath(r.Context(), organizationID, gitRepoCfg, ".gitignore")
		commitMsg := fmt.Sprintf("spec(sync): approve and stage %d file(s)", len(reqBody.StagedPaths))
		_ = client.LocalCommit(r.Context(), organizationID, gitRepoCfg, "", commitMsg, name, email)
	} else if reqBody.FilePath != "" {
		_ = client.StagePath(r.Context(), organizationID, gitRepoCfg, ".gitignore")
		fileName := filepath.Base(reqBody.FilePath)
		commitMsg := fmt.Sprintf("spec(sync): approve and stage %s", fileName)
		errCommit := client.LocalCommit(r.Context(), organizationID, gitRepoCfg, reqBody.FilePath, commitMsg, name, email)
		if errCommit != nil {
			sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to commit locally: %v", errCommit), Status: "ERROR"})
			return
		}
		_ = updateArtifactStatus(r.Context(), organizationID, projectId, reqBody.ArtifactID, reqBody.FilePath, db.ArtifactStatusStagedForReview)
	} else {
		// Full workspace asset sync
		pathsToStage := []string{".agents", ".gitignore"}
		if spec := project.GetSpecLocation(); spec != "" {
			pathsToStage = append(pathsToStage, spec)
		}
		if planning := project.GetPlanningArtifactsLocation(); planning != "" {
			pathsToStage = append(pathsToStage, planning)
		}
		if impl := project.GetImplementationArtifactsLocation(); impl != "" {
			pathsToStage = append(pathsToStage, impl)
		}

		for _, p := range pathsToStage {
			_ = client.StagePath(r.Context(), organizationID, gitRepoCfg, p)
		}
		commitMsg := "spec(sync): sync workspace assets and configuration"
		_ = client.LocalCommit(r.Context(), organizationID, gitRepoCfg, "", commitMsg, name, email)
	}

	// Step: Pull latest main and rebase spec/* before pushing to remote
	isConflictFallback := false
	errPull := client.Pull(r.Context(), organizationID, gitRepoCfg)
	if errPull != nil {
		if reqBody.Action == "request_review" {
			slog.Warn("Pull/rebase encountered conflict, proceeding with pushing spec branch for PR creation", "error", errPull)
		} else {
			slog.Warn("Pull/rebase encountered conflict during Push mode, falling back to PR creation", "error", errPull)
			reqBody.Action = "request_review"
			isConflictFallback = true
		}
	}

	errPush := client.Push(r.Context(), organizationID, gitRepoCfg)
	if errPush != nil {
		if errors.Is(errPush, ErrWriteAccessDenied) {
			sendJSON(w, http.StatusForbidden, Response{
				Title:       "Access Denied",
				Message:     "Failed to push to remote repository: write access not granted or token expired.",
				Description: "Please reconnect your GitHub credentials in Settings.",
				Status:      "ERROR",
			})
			return
		}
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to push to remote: %v", errPush), Status: "ERROR"})
		return
	}

	// Trigger gate status calculation asynchronously
	detachedCtx := context.WithoutCancel(r.Context())
	store := db.ProjectStoreInstance
	artStore := ArtifactStore
	agentStore := agent.Store
	runAsync(func() {
		if err := recalculateGatesWithStores(detachedCtx, organizationID, projectId, store, artStore, agentStore); err != nil {
			slog.Error("Failed to recalculate gates after sync", "projectId", projectId, "error", err)
		}
	})

	var prURL string
	var prNumber int
	var isExistingPR bool
	owner, repoName, errParse := ParseGitHubURL(repoURL)
	if errParse == nil && owner != "" && repoName != "" {
		prClient := &integrations.GitHubPRClient{Token: accessToken}
		projectName := ""
		if project.Name != nil {
			projectName = *project.Name
		}
		slog.Info("GitSyncHandler: evaluated action for PR", "action", reqBody.Action, "owner", owner, "repo", repoName, "head", targetBranch, "base", baseBranch)
		if reqBody.Action == "request_review" {
			prPayload := integrations.PullRequestPayload{
				Title: fmt.Sprintf("feat(workspace): sync workspace assets for project %s", projectName),
				Head:  targetBranch,
				Base:  baseBranch,
				Body: fmt.Sprintf(
					"### Workspace Asset Sync Report\n\n- **Synced By:** @%s\n- **Timestamp:** %s UTC\n\n*This Pull Request was automatically generated by Agentic Foundry Workspace Sync.*",
					name,
					time.Now().UTC().Format("2006-01-02 15:04:05"),
				),
			}

			prResp, errPR := prClient.CreatePullRequest(r.Context(), owner, repoName, prPayload)
			if errPR != nil {
				slog.Warn("Failed to create Pull Request after workspace sync", "error", errPR, "head", targetBranch, "base", baseBranch)
				if integrations.IsDuplicatePullRequestError(errPR) {
					isExistingPR = true
					headParam := fmt.Sprintf("%s:%s", owner, targetBranch)
					if prs, errList := prClient.ListPullRequests(r.Context(), owner, repoName, headParam); errList == nil && len(prs) > 0 {
						prURL = prs[0].HTMLURL
						prNumber = prs[0].Number
						slog.Info("Found existing Pull Request for branch", "prURL", prURL, "prNumber", prNumber)
					} else if errList != nil {
						slog.Warn("Failed to list existing Pull Requests", "error", errList)
					}
				}
			} else if prResp != nil {
				prURL = prResp.HTMLURL
				prNumber = prResp.Number
				slog.Info("Successfully created Pull Request", "prURL", prURL, "prNumber", prNumber)
			}
		} else if reqBody.Action == "sync" || reqBody.Action == "merge" || reqBody.Action == "push" {
			mergeMsg := fmt.Sprintf("chore(merge): merge %s into %s", targetBranch, baseBranch)
			slog.Info("Attempting branch merge with base branch", "head", targetBranch, "base", baseBranch)
			errMerge := prClient.MergeBranch(r.Context(), owner, repoName, baseBranch, targetBranch, mergeMsg)
			if errMerge != nil {
				slog.Warn("Branch merge into base branch failed or encountered conflict, falling back to Pull Request creation", "error", errMerge)
				prPayload := integrations.PullRequestPayload{
					Title: fmt.Sprintf("feat(workspace): sync workspace assets for project %s (Review Fallback)", projectName),
					Head:  targetBranch,
					Base:  baseBranch,
					Body: fmt.Sprintf(
						"### Workspace Asset Sync Report (Automatic PR Fallback)\n\nAutomated direct merge to '%s' encountered an issue: %v.\n\nPlease review and merge via this Pull Request.\n\n- **Synced By:** @%s\n- **Timestamp:** %s UTC",
						baseBranch,
						errMerge,
						name,
						time.Now().UTC().Format("2006-01-02 15:04:05"),
					),
				}
				if prResp, errPR := prClient.CreatePullRequest(r.Context(), owner, repoName, prPayload); errPR == nil && prResp != nil {
					prURL = prResp.HTMLURL
					prNumber = prResp.Number
				} else if errPR != nil && integrations.IsDuplicatePullRequestError(errPR) {
					isExistingPR = true
					headParam := fmt.Sprintf("%s:%s", owner, targetBranch)
					if prs, errList := prClient.ListPullRequests(r.Context(), owner, repoName, headParam); errList == nil && len(prs) > 0 {
						prURL = prs[0].HTMLURL
						prNumber = prs[0].Number
					}
				}
			} else {
				slog.Info("Successfully merged target branch into base branch", "head", targetBranch, "base", baseBranch)
				_ = client.FastForwardBaseToSpec(r.Context(), organizationID, gitRepoCfg)
			}
		}
	} else {
		slog.Warn("Failed to parse GitHub URL for PR creation", "repoURL", repoURL, "error", errParse)
	}

	respData := map[string]interface{}{
		"status":     "SUCCESS",
		"message":    fmt.Sprintf("Pushed changes to '%s' and merged into '%s'.", targetBranch, baseBranch),
		"branchName": targetBranch,
	}
	if prURL != "" {
		respData["prUrl"] = prURL
		respData["prNumber"] = prNumber
		respData["isExistingPr"] = isExistingPR
		if isConflictFallback {
			respData["message"] = fmt.Sprintf("Merge conflict detected against '%s'. Created Pull Request #%d for review.", baseBranch, prNumber)
		} else if isExistingPR {
			respData["message"] = fmt.Sprintf("Review request already exists: Pull Request #%d", prNumber)
		} else {
			respData["message"] = fmt.Sprintf("Workspace assets synced and Pull Request #%d created.", prNumber)
		}
	}
	sendJSON(w, http.StatusOK, respData)
}

func GitMergeHandler(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("id")
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized: User claims missing", Status: "ERROR"})
		return
	}
	userId := claims.Subject

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized: Organization ID missing", Status: "ERROR"})
		return
	}

	var reqBody GitSyncRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid request payload", Status: "ERROR"})
		return
	}
	if reqBody.FilePath == "" {
		sendJSON(w, http.StatusBadRequest, Response{Message: "filePath is required", Status: "ERROR"})
		return
	}

	project, err := db.ProjectStoreInstance.GetProject(r.Context(), organizationID, projectId)
	if err != nil {
		sendJSON(w, http.StatusNotFound, Response{Message: fmt.Sprintf("Project not found: %v", err), Status: "ERROR"})
		return
	}

	isReadOnly := false
	userProj, errUserProj := UserProjectStore.GetProjectAppSetting(r.Context(), organizationID, userId, projectId, "github")
	if errUserProj == nil && userProj != nil {
		isReadOnly = userProj.IsReadOnly
	}

	if isReadOnly {
		sendJSON(w, http.StatusForbidden, Response{
			Title:       "Operation Forbidden",
			Message:     "Remote sync operations are disabled because your workspace is running in Read-Only Mode (Read-Only Repository Access).",
			Description: "Pushing changes to the remote repository is not permitted when in read-only mode.",
			Status:      "ERROR",
		})
		return
	}

	repoURL := project.GetRepoUrl()
	if repoURL == "" {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Repository URL is empty on project configuration", Status: "ERROR"})
		return
	}

	owner, repoName, errParse := ParseGitHubURL(repoURL)
	if errParse != nil {
		sendJSON(w, http.StatusBadRequest, Response{Message: errParse.Error(), Status: "ERROR"})
		return
	}

	accessToken, errToken := integrations.ResolveGitHubToken(r.Context(), userId, projectId)
	if errToken != nil {
		if errors.Is(errToken, integrations.ErrGitHubCredsNotConfigured) {
			sendJSON(w, http.StatusForbidden, Response{Message: "Forbidden: " + errToken.Error(), Status: "ERROR"})
		} else {
			sendJSON(w, http.StatusInternalServerError, Response{Message: "Internal Error: " + errToken.Error(), Status: "ERROR"})
		}
		return
	}

	baseBranch, targetBranch, errResolve := resolveSpecBranch(r.Context(), organizationID, projectId, userId, r)
	if errResolve != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to resolve spec branch: %v", errResolve), Status: "ERROR"})
		return
	}

	if targetBranch == baseBranch {
		sendJSON(w, http.StatusBadRequest, Response{
			Message: "Base branch and target specification branch cannot be identical for merging.",
			Status:  "ERROR",
		})
		return
	}

	gitRepoCfg := &GitRepoConfig{
		RepoURL:         repoURL,
		RepoAccessToken: accessToken,
		BaseBranch:      baseBranch,
		SpecBranch:      targetBranch,
		UserId:          userId,
		ProjectId:       projectId,
		SpecLocation:    project.GetSpecLocation(),
	}

	client := getGitClient()
	errPush := client.Push(r.Context(), organizationID, gitRepoCfg)
	if errPush != nil {
		if errors.Is(errPush, ErrWriteAccessDenied) {
			sendJSON(w, http.StatusForbidden, Response{
				Title:       "Access Denied",
				Message:     "Failed to push to remote repository: write access not granted or token expired.",
				Description: "Please reconnect your GitHub credentials in Settings.",
				Status:      "ERROR",
			})
			return
		}
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to push to remote: %v", errPush), Status: "ERROR"})
		return
	}

	prClient := &integrations.GitHubPRClient{Token: accessToken}

	// Verify target remote branch exists before attempting merge
	exists, errBranch := prClient.CheckBranchExists(r.Context(), owner, repoName, targetBranch)
	if errBranch != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to verify remote branch: %v", errBranch), Status: "ERROR"})
		return
	}
	if !exists {
		sendJSON(w, http.StatusConflict, Response{
			Title:   "Unable to merge",
			Message: fmt.Sprintf("Staging branch '%s' is not available on remote repository", targetBranch),
			Description: "This typically happens when the credentials used for Git operations don't have " +
				"enough permissions on the code repository. Please verify your token and try again.",
			Status: "ERROR"})
		return
	}

	fileName := filepath.Base(reqBody.FilePath)
	commitMessage := fmt.Sprintf("spec(sync): merge approved %s", fileName)

	errMerge := prClient.MergeBranch(r.Context(), owner, repoName, baseBranch, targetBranch, commitMessage)
	if errMerge != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to merge branch: %v", errMerge), Status: "ERROR"})
		return
	}

	_ = client.FastForwardBaseToSpec(r.Context(), organizationID, gitRepoCfg)

	// Transition specific Artifact database status to 'Complete'
	_ = updateArtifactStatus(r.Context(), organizationID, projectId, reqBody.ArtifactID, reqBody.FilePath, db.ArtifactStatusComplete)

	// Trigger gate status calculation asynchronously
	detachedCtx := context.WithoutCancel(r.Context())
	store := db.ProjectStoreInstance
	artStore := ArtifactStore
	agentStore := agent.Store
	runAsync(func() {
		if err := recalculateGatesWithStores(detachedCtx, organizationID, projectId, store, artStore, agentStore); err != nil {
			slog.Error("Failed to recalculate gates after merge", "projectId", projectId, "error", err)
		}
	})

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"status":     "SUCCESS",
		"message":    "Specifications successfully merged.",
		"branchName": targetBranch,
	})
}

func GitCommitHandler(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("id")
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized: User claims missing", Status: "ERROR"})
		return
	}
	userId := claims.Subject

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized: Organization ID missing", Status: "ERROR"})
		return
	}

	var reqBody GitSyncRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid request payload", Status: "ERROR"})
		return
	}
	if reqBody.FilePath == "" {
		sendJSON(w, http.StatusBadRequest, Response{Message: "filePath is required", Status: "ERROR"})
		return
	}

	project, err := db.ProjectStoreInstance.GetProject(r.Context(), organizationID, projectId)
	if err != nil {
		sendJSON(w, http.StatusNotFound, Response{Message: fmt.Sprintf("Project not found: %v", err), Status: "ERROR"})
		return
	}

	baseBranch, targetBranch, errResolve := resolveSpecBranch(r.Context(), organizationID, projectId, userId, r)
	if errResolve != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to resolve spec branch: %v", errResolve), Status: "ERROR"})
		return
	}

	fileName := filepath.Base(reqBody.FilePath)

	email := project.GetGitHubEmail()
	if email == "" {
		if resEmail, err := integrations.ResolveGitHubEmail(r.Context(), userId, projectId); err == nil && resEmail != "" {
			email = resEmail
		}
	}
	if email == "" && claims != nil {
		email = claims.Subject
	}
	name := email
	if atIdx := strings.Index(email, "@"); atIdx != -1 {
		name = email[:atIdx]
	}

	repoURL := project.GetRepoUrl()

	gitRepoCfg := &GitRepoConfig{
		RepoURL:      repoURL,
		BaseBranch:   baseBranch,
		SpecBranch:   targetBranch,
		UserId:       userId,
		ProjectId:    projectId,
		SpecLocation: project.GetSpecLocation(),
		AuthorEmail:  email,
		AuthorName:   name,
	}

	client := getGitClient()
	commitMsg := fmt.Sprintf("spec(approve): approve %s", fileName)
	errCommit := client.LocalCommit(r.Context(), organizationID, gitRepoCfg, reqBody.FilePath, commitMsg, name, email)
	if errCommit != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to commit locally: %v", errCommit), Status: "ERROR"})
		return
	}

	// Transition specific Artifact database status to 'Complete'
	_ = updateArtifactStatus(r.Context(), organizationID, projectId, reqBody.ArtifactID, reqBody.FilePath, db.ArtifactStatusComplete)

	// Trigger gate status calculation asynchronously
	detachedCtx := context.WithoutCancel(r.Context())
	store := db.ProjectStoreInstance
	artStore := ArtifactStore
	agentStore := agent.Store
	runAsync(func() {
		if err := recalculateGatesWithStores(detachedCtx, organizationID, projectId, store, artStore, agentStore); err != nil {
			slog.Error("Failed to recalculate gates after commit", "projectId", projectId, "error", err)
		}
	})

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"status":     "SUCCESS",
		"message":    "Specifications successfully approved and saved locally.",
		"branchName": targetBranch,
	})
}

func ParseGitHubURL(repoURL string) (owner string, repo string, err error) {
	if !strings.HasPrefix(repoURL, "https://github.com/") {
		return "", "", fmt.Errorf("invalid repository URL: only HTTPS github.com URLs are supported")
	}
	trimmed := strings.TrimPrefix(repoURL, "https://github.com/")
	trimmed = strings.TrimSuffix(trimmed, ".git")
	trimmed = strings.TrimRight(trimmed, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("failed to parse owner and repository name from URL")
	}
	return parts[0], parts[1], nil
}

func updateArtifactStatus(ctx context.Context, organizationID string, projectId, artifactId, filePath string, status db.ArtifactStatus) error {
	artifacts, err := ArtifactStore.ListArtifactsForProject(ctx, organizationID, projectId)
	if err != nil {
		slog.Error("Failed to list artifacts to update status", "projectId", projectId, "error", err)
		return err
	}

	var targetArt *db.Artifact
	baseName := filepath.Base(filePath)

	for i := range artifacts {
		if (artifactId != "" && artifacts[i].ID == artifactId) || artifacts[i].Path == filePath || artifacts[i].Title == baseName {
			targetArt = &artifacts[i]
			break
		}
	}

	if targetArt == nil {
		slog.Warn("Target artifact record not found for status update", "projectId", projectId, "filePath", filePath)
		return fmt.Errorf("artifact not found")
	}

	targetArt.Status = status
	if errUpdate := ArtifactStore.UpdateArtifact(ctx, organizationID, targetArt); errUpdate != nil {
		slog.Error("Failed to update Firestore status for artifact", "artifactId", targetArt.ID, "error", errUpdate)
		return errUpdate
	}

	slog.Info("Successfully transitioned artifact status", "artifactId", targetArt.ID, "filePath", filePath, "status", status)
	return nil
}

func SyncOrRegisterArtifact(ctx context.Context, organizationID string, projectId, phase, filePath string) error {
	artifacts, err := ArtifactStore.ListArtifactsForProject(ctx, organizationID, projectId)
	if err != nil {
		return fmt.Errorf("failed to list artifacts to update status: %w", err)
	}

	var targetArt *db.Artifact
	baseName := filepath.Base(filePath)

	for i := range artifacts {
		if artifacts[i].Path == filePath || artifacts[i].Title == baseName {
			targetArt = &artifacts[i]
			break
		}
	}

	if targetArt != nil {
		needsUpdate := false
		if targetArt.Status == db.ArtifactStatusNotStarted {
			targetArt.Status = db.ArtifactStatusInProgress
			needsUpdate = true
		}
		if targetArt.Path != filePath && filePath != "" {
			targetArt.Path = filePath
			needsUpdate = true
		}
		if needsUpdate {
			if errUpdate := ArtifactStore.UpdateArtifact(ctx, organizationID, targetArt); errUpdate != nil {
				slog.Error("Failed to update Firestore status for artifact to In Progress", "artifactId", targetArt.ID, "error", errUpdate)
				return errUpdate
			}
			slog.Info("Successfully transitioned artifact status to In Progress", "artifactId", targetArt.ID, "filePath", filePath)
		}
		return nil
	}

	// New artifact: automatically register in Firestore on the API tier
	if phase == "" {
		project, _ := db.ProjectStoreInstance.GetProject(ctx, organizationID, projectId)
		phase = db.DetermineArtifactPhaseForProject(filePath, project)
	}
	newId := fmt.Sprintf("%s_%s", projectId, baseName)
	newArt := &db.Artifact{
		ID:           newId,
		ProjectId:    projectId,
		Title:        baseName,
		Path:         filePath,
		Phase:        phase,
		Status:       db.ArtifactStatusInProgress,
		Owner:        "agent",
		Content:      "",
		ArtifactType: db.DetermineArtifactType(filePath),
	}

	if errCreate := ArtifactStore.CreateArtifact(ctx, organizationID, newArt); errCreate != nil {
		slog.Error("Failed to register new artifact in Firestore", "artifactId", newId, "filePath", filePath, "error", errCreate)
		return errCreate
	}

	slog.Info("Successfully created and registered new artifact in Firestore", "artifactId", newId, "filePath", filePath, "phase", phase)
	return nil
}

func resolveSpecBranch(ctx context.Context, organizationID string, projectId, userId string, r *http.Request) (baseBranch, specBranch string, err error) {
	project, err := db.ProjectStoreInstance.GetProject(ctx, organizationID, projectId)
	if err != nil {
		return "", "", fmt.Errorf("failed to get project: %w", err)
	}

	base := "main"
	if project.BaseBranch != nil && *project.BaseBranch != "" {
		base = *project.BaseBranch
	}

	// Default to base branch if no session is active or resolved
	spec := base

	var reqSessionId string
	var reqPhase string

	// Read query parameters
	if qSessionId := r.URL.Query().Get("sessionId"); qSessionId != "" {
		reqSessionId = qSessionId
	}
	if qPhase := r.URL.Query().Get("phase"); qPhase != "" {
		reqPhase = qPhase
	}

	// Read optional JSON body (parse safely to avoid disrupting downstream/upstream parsing)
	if r.Method == http.MethodPost && r.Body != nil {
		var bodyMap map[string]interface{}
		if bodyBytes, errRead := io.ReadAll(r.Body); errRead == nil {
			// Restore the body reader for subsequent HTTP parsing steps
			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			_ = json.Unmarshal(bodyBytes, &bodyMap)
			if bodyMap != nil {
				if val, ok := bodyMap["sessionId"].(string); ok && val != "" {
					reqSessionId = val
				}
				if val, ok := bodyMap["phase"].(string); ok && val != "" {
					reqPhase = val
				}
				if val, ok := bodyMap["filePath"].(string); ok && val != "" && reqPhase == "" {
					// Infer phase from the path of the specific artifact target if phase is unspecified
					if arts, errList := ArtifactStore.ListArtifactsForProject(ctx, organizationID, projectId); errList == nil {
						baseName := filepath.Base(val)
						for _, a := range arts {
							if a.Path == val || a.Title == baseName {
								reqPhase = a.Phase
								break
							}
						}
					}
				}
			}
		}
	}

	var targetSession *db.Session
	if reqSessionId != "" {
		sessions, errList := SessionStore.ListSessionsForProject(ctx, organizationID, projectId, userId, "")
		if errList == nil {
			for _, s := range sessions {
				if s.ID == reqSessionId {
					targetSession = &s
					break
				}
			}
		}
	} else if reqPhase != "" {
		sessions, errList := SessionStore.ListSessionsForProject(ctx, organizationID, projectId, userId, reqPhase)
		if errList == nil && len(sessions) > 0 {
			targetSession = &sessions[0]
		}
	} else {
		sessions, errList := SessionStore.ListSessionsForProject(ctx, organizationID, projectId, userId, "")
		if errList == nil && len(sessions) > 0 {
			targetSession = &sessions[0]
		}
	}

	mountPath := getWorkspaceMountPath()
	clonePath := filepath.Join(mountPath, organizationID, userId, projectId)
	if fsBranch := getGitBranchFromFS(clonePath); fsBranch != "" && fsBranch != base {
		return base, fsBranch, nil
	}

	prefix := project.GetSpecBranchPrefix()
	if targetSession != nil && targetSession.SpecBranchName != "" {
		spec = targetSession.SpecBranchName
	} else {
		spec = ResolveUserSpecBranchName(ctx, prefix, userId, clonePath)
	}

	return base, spec, nil
}

func ResolveSpecBranchNoRequest(ctx context.Context, organizationID string, projectId, userId, phase string) (baseBranch, specBranch string, err error) {
	project, err := db.ProjectStoreInstance.GetProject(ctx, organizationID, projectId)
	if err != nil {
		return "", "", fmt.Errorf("failed to get project: %w", err)
	}

	base := "main"
	if project.BaseBranch != nil && *project.BaseBranch != "" {
		base = *project.BaseBranch
	}

	prefix := project.GetSpecBranchPrefix()
	spec := base

	mountPath := getWorkspaceMountPath()
	clonePath := filepath.Join(mountPath, organizationID, userId, projectId)
	if fsBranch := getGitBranchFromFS(clonePath); fsBranch != "" && fsBranch != base {
		return base, fsBranch, nil
	}

	var targetSession *db.Session
	if phase != "" {
		sessions, errList := SessionStore.ListSessionsForProject(ctx, organizationID, projectId, userId, phase)
		if errList == nil && len(sessions) > 0 {
			targetSession = &sessions[0]
		}
	}

	if targetSession == nil {
		sessions, errList := SessionStore.ListSessionsForProject(ctx, organizationID, projectId, userId, "")
		if errList == nil && len(sessions) > 0 {
			targetSession = &sessions[0]
		}
	}

	if targetSession != nil && targetSession.SpecBranchName != "" {
		spec = targetSession.SpecBranchName
	} else {
		spec = ResolveUserSpecBranchName(ctx, prefix, userId, clonePath)
	}

	return base, spec, nil
}

// ProjectHealthCheckHandler runs system-wide checks on workspace health, synchronizing artifacts from disk, recalculating governance gates, and injecting skills.
func ProjectHealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("id")

	organizationID, err := core.GetOrganizationID(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized context", http.StatusUnauthorized)
		return
	}

	claims, _ := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	userID := "simulated-admin-uid"
	if claims != nil && claims.Subject != "" {
		userID = claims.Subject
	} else if AppConfig != nil && AppConfig.SimulatedUserID != "" {
		userID = AppConfig.SimulatedUserID
	}

	// 1. Synchronize project artifacts from disk to Firestore before gate calculation
	if db.ProjectStoreInstance != nil {
		if project, errProject := db.ProjectStoreInstance.GetProject(r.Context(), organizationID, projectId); errProject == nil && project != nil {
			added, deleted, errSync := SyncProjectArtifactsFromDisk(r.Context(), organizationID, userID, project, false)
			if errSync != nil {
				slog.Warn("Failed to sync artifacts from disk during health check", "projectId", projectId, "error", errSync)
			} else {
				slog.Info("Successfully synced artifacts during health check", "projectId", projectId, "added", added, "deleted", deleted)
			}
		}
	}

	// 2. Recalculate project governance gates
	if err := recalculateGates(r.Context(), organizationID, projectId); err != nil {
		slog.Error("Failed to recalculate gates during health check", "projectId", projectId, "error", err)
		sendJSON(w, http.StatusInternalServerError, Response{
			Message: fmt.Sprintf("Failed to run health check: %v", err),
			Status:  "ERROR",
		})
		return
	}

	// 2. Inject skills and framework into project workspace
	if err := injectSkills(r.Context(), organizationID, userID, projectId); err != nil {
		slog.Error("Failed to inject skills during health check", "projectId", projectId, "error", err)
		sendJSON(w, http.StatusInternalServerError, Response{
			Message: fmt.Sprintf("Failed to inject skills during health check: %v", err),
			Status:  "ERROR",
		})
		return
	}

	// 3. Inject config updates into .agents/_bmad configuration files
	if err := injectConfigUpdates(r.Context(), organizationID, userID, projectId); err != nil {
		slog.Error("Failed to inject config updates during health check", "projectId", projectId, "error", err)
		sendJSON(w, http.StatusInternalServerError, Response{
			Message: fmt.Sprintf("Failed to inject config updates during health check: %v", err),
			Status:  "ERROR",
		})
		return
	}

	// 4. Stage .agents folder for commit if git repository is configured
	if db.ProjectStoreInstance != nil {
		if project, err := db.ProjectStoreInstance.GetProject(r.Context(), organizationID, projectId); err == nil && project != nil && project.GetRepoUrl() != "" {
			accessToken, _ := integrations.ResolveGitHubToken(r.Context(), userID, projectId)
			baseBranch, specBranch, _ := ResolveSpecBranchNoRequest(r.Context(), organizationID, projectId, userID, "ANALYSIS")
			if baseBranch == "" {
				baseBranch = "main"
			}
			if specBranch == "" {
				specBranch = "specs/" + userID
			}

			repoCfg := &GitRepoConfig{
				RepoURL:         project.GetRepoUrl(),
				RepoAccessToken: accessToken,
				BaseBranch:      baseBranch,
				SpecBranch:      specBranch,
				UserId:          userID,
				ProjectId:       projectId,
				SpecLocation:    project.GetSpecLocation(),
			}

			client := getGitClient()
			if err := client.StagePath(r.Context(), organizationID, repoCfg, ".agents"); err != nil {
				slog.Warn("Failed to stage .agents for commit during health check", "projectId", projectId, "error", err)
			}
		}
	}

	sendJSON(w, http.StatusOK, Response{
		Message: "Health check completed successfully, project gates recalculated, skills and config injected.",
		Status:  "SUCCESS",
	})
}

// InjectSkillsHandler prepares and copies universal skills and framework config into the project's .agents directory.
func InjectSkillsHandler(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("id")

	organizationID, err := core.GetOrganizationID(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized context", http.StatusUnauthorized)
		return
	}

	claims, _ := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	userID := "simulated-admin-uid"
	if claims != nil && claims.Subject != "" {
		userID = claims.Subject
	} else if AppConfig != nil && AppConfig.SimulatedUserID != "" {
		userID = AppConfig.SimulatedUserID
	}
	var reqBody struct {
		Action string `json:"action"`
	}
	_ = json.NewDecoder(r.Body).Decode(&reqBody)

	if db.ProjectStoreInstance != nil {
		if project, err := db.ProjectStoreInstance.GetProject(r.Context(), organizationID, projectId); err == nil && project != nil {
			if project.IsReadOnly && (reqBody.Action == "push" || reqBody.Action == "request_review") {
				actionName := "Push"
				if reqBody.Action == "request_review" {
					actionName = "Pull Request"
				}
				sendJSON(w, http.StatusForbidden, Response{
					Message: fmt.Sprintf("Project is in read-only mode. %s operations are restricted.", actionName),
					Status:  "ERROR",
				})
				return
			}
		}
	}

	if err := injectSkills(r.Context(), organizationID, userID, projectId); err != nil {
		slog.Error("Failed to inject skills", "projectId", projectId, "error", err)
		sendJSON(w, http.StatusInternalServerError, Response{
			Message: fmt.Sprintf("Failed to inject skills: %v", err),
			Status:  "ERROR",
		})
		return
	}

	var prURL string
	var prNumber int

	if reqBody.Action == "push" || reqBody.Action == "request_review" {
		if db.ProjectStoreInstance != nil {
			if project, err := db.ProjectStoreInstance.GetProject(r.Context(), organizationID, projectId); err == nil && project != nil && project.GetRepoUrl() != "" {
				accessToken, _ := integrations.ResolveGitHubToken(r.Context(), userID, projectId)
				baseBranch, specBranch, _ := ResolveSpecBranchNoRequest(r.Context(), organizationID, projectId, userID, "ANALYSIS")
				if baseBranch == "" {
					baseBranch = "main"
				}
				if specBranch == "" {
					specBranch = "specs/" + userID
				}

				injectEmail := project.GetGitHubEmail()
				if injectEmail == "" {
					if resEmail, err := integrations.ResolveGitHubEmail(r.Context(), userID, projectId); err == nil && resEmail != "" {
						injectEmail = resEmail
					}
				}
				if injectEmail == "" {
					injectEmail = userID
				}
				injectName := injectEmail
				if atIdx := strings.Index(injectEmail, "@"); atIdx != -1 {
					injectName = injectEmail[:atIdx]
				}

				repoCfg := &GitRepoConfig{
					RepoURL:         project.GetRepoUrl(),
					RepoAccessToken: accessToken,
					BaseBranch:      baseBranch,
					SpecBranch:      specBranch,
					UserId:          userID,
					ProjectId:       projectId,
					SpecLocation:    project.GetSpecLocation(),
					AuthorEmail:     injectEmail,
					AuthorName:      injectName,
				}

				client := getGitClient()
				if err := client.StagePath(r.Context(), organizationID, repoCfg, ".agents"); err != nil {
					slog.Warn("Failed to stage .agents for push during skill injection", "projectId", projectId, "error", err)
				}
				commitMsg := "feat(skills): inject universal skills into .agents"
				if errCommit := client.LocalCommit(r.Context(), organizationID, repoCfg, "", commitMsg, injectName, injectEmail); errCommit != nil {
					slog.Warn("Failed to commit .agents for push during skill injection", "projectId", projectId, "error", errCommit)
				}
				if err := client.Push(r.Context(), organizationID, repoCfg); err != nil {
					slog.Warn("Failed to push .agents changes to remote", "projectId", projectId, "error", err)
				}

				if reqBody.Action == "request_review" {
					owner, repoName, errParse := ParseGitHubURL(project.GetRepoUrl())
					if errParse == nil && owner != "" && repoName != "" {
						prClient := &integrations.GitHubPRClient{Token: accessToken}
						prPayload := integrations.PullRequestPayload{
							Title: fmt.Sprintf("feat(skills): inject universal skills into .agents"),
							Head:  specBranch,
							Base:  baseBranch,
							Body: fmt.Sprintf(
								"### Skill Injection Report\n\n- **Injected By:** @%s\n- **Timestamp:** %s UTC\n\n*This Pull Request was automatically generated by Agentic Foundry Skill Injector.*",
								userID,
								time.Now().UTC().Format("2006-01-02 15:04:05"),
							),
						}
						prResp, errPR := prClient.CreatePullRequest(r.Context(), owner, repoName, prPayload)
						if errPR != nil {
							slog.Warn("Failed to create Pull Request after skills push", "error", errPR, "owner", owner, "repo", repoName, "head", specBranch, "base", baseBranch)
							if integrations.IsDuplicatePullRequestError(errPR) {
								headParam := fmt.Sprintf("%s:%s", owner, specBranch)
								if prs, errList := prClient.ListPullRequests(r.Context(), owner, repoName, headParam); errList == nil && len(prs) > 0 {
									prURL = prs[0].HTMLURL
									prNumber = prs[0].Number
								}
							}
						} else if prResp != nil {
							prURL = prResp.HTMLURL
							prNumber = prResp.Number
							slog.Info("Successfully created Pull Request for skill injection", "prURL", prURL, "prNumber", prNumber)
						}
					}
				} else if reqBody.Action == "push" {
					owner, repoName, errParse := ParseGitHubURL(project.GetRepoUrl())
					if errParse == nil && owner != "" && repoName != "" {
						prClient := &integrations.GitHubPRClient{Token: accessToken}

						cmp, errCmp := prClient.CompareBranches(r.Context(), owner, repoName, baseBranch, specBranch)
						if errCmp == nil && cmp != nil && !cmp.Mergeable {
							slog.Warn("Skill injection merge conflict detected. Falling back to Pull Request creation.", "specBranch", specBranch, "baseBranch", baseBranch)
							prPayload := integrations.PullRequestPayload{
								Title: "feat(skills): inject universal skills into .agents (Conflict Review)",
								Head:  specBranch,
								Base:  baseBranch,
								Body:  fmt.Sprintf("### Skill Injection Report (Merge Conflict Detected)\n\nAutomated merge to '%s' failed due to merge conflicts. Please review and resolve conflicts in this Pull Request.", baseBranch),
							}
							if prResp, errPR := prClient.CreatePullRequest(r.Context(), owner, repoName, prPayload); errPR == nil && prResp != nil {
								prURL = prResp.HTMLURL
								prNumber = prResp.Number
							}
						} else {
							mergeMsg := fmt.Sprintf("chore(skills): merge injected skills from %s into %s", specBranch, baseBranch)
							if errMerge := prClient.MergeBranch(r.Context(), owner, repoName, baseBranch, specBranch, mergeMsg); errMerge != nil {
								slog.Warn("Failed to merge injected skills into base branch, creating PR fallback", "error", errMerge)
								prPayload := integrations.PullRequestPayload{
									Title: "feat(skills): inject universal skills into .agents",
									Head:  specBranch,
									Base:  baseBranch,
									Body:  fmt.Sprintf("### Skill Injection Report\n\nAutomated merge encountered an issue: %v. Please review this Pull Request.", errMerge),
								}
								if prResp, errPR := prClient.CreatePullRequest(r.Context(), owner, repoName, prPayload); errPR == nil && prResp != nil {
									prURL = prResp.HTMLURL
									prNumber = prResp.Number
								}
							} else {
								slog.Info("Successfully merged injected skills into base branch", "specBranch", specBranch, "baseBranch", baseBranch)
								_ = client.Pull(r.Context(), organizationID, repoCfg)
							}
						}
					}
				}
			}
		}
	}

	msg := "Universal skills and framework config successfully prepared and copied into .agents folder."
	if reqBody.Action == "push" {
		msg = "Universal skills injected, prepared, and pushed to specification branch."
	} else if reqBody.Action == "request_review" {
		if prURL != "" {
			msg = fmt.Sprintf("Universal skills injected and Pull Request #%d created.", prNumber)
		} else {
			msg = "Universal skills injected and pushed to staging branch for Pull Request."
		}
	}

	respData := map[string]interface{}{
		"status":  "SUCCESS",
		"message": msg,
	}
	if prURL != "" {
		respData["prUrl"] = prURL
		respData["prNumber"] = prNumber
	}

	sendJSON(w, http.StatusOK, respData)
}


// VerifySkillsPresence checks if .agents, .agents/_bmad, and at least one
// .agents/skills/bmad-* subfolder exist in the workspace directory.
// Returns true if all exist, false otherwise.
func VerifySkillsPresence(workspaceDir string) bool {
	agentsDir := filepath.Join(workspaceDir, ".agents")
	if info, err := os.Stat(agentsDir); err != nil || !info.IsDir() {
		return false
	}

	bmadDir := filepath.Join(agentsDir, "_bmad")
	if info, err := os.Stat(bmadDir); err != nil || !info.IsDir() {
		return false
	}

	skillsDir := filepath.Join(agentsDir, "skills")
	if info, err := os.Stat(skillsDir); err != nil || !info.IsDir() {
		return false
	}

	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "bmad-") {
			return true
		}
	}

	return false
}

// injectSkills ensures that the project source tree has a .agents folder at its root,
// containing the _bmad framework configuration and updated skills from BMAD_ROOT_PATH.
func injectSkills(ctx context.Context, organizationID, userID, projectID string) error {
	mountPath := getWorkspaceMountPath()
	projectRoot := filepath.Join(mountPath, organizationID, userID, projectID)
	return InjectSkillsForPath(ctx, projectRoot, organizationID, userID, projectID)
}

// InjectSkillsForPath ensures that projectRoot has a .agents folder at its root,
// containing the _bmad framework configuration and updated skills from BMAD_ROOT_PATH.
func InjectSkillsForPath(ctx context.Context, projectRoot, organizationID, userID, projectID string) error {
	if err := os.MkdirAll(projectRoot, 0755); err != nil {
		return fmt.Errorf("failed to ensure project root directory: %w", err)
	}

	// Only copy skill files if they are not already present on disk
	if !VerifySkillsPresence(projectRoot) {
		// Prefer GLOBAL_WORKSPACE_PATH if available on the shared workspace mount
		assetSource := ""
		globalPath := ""
		if AppConfig != nil && AppConfig.GlobalWorkspacePath != "" {
			globalPath = AppConfig.GlobalWorkspacePath
		}
		if globalPath == "" {
			globalPath = os.Getenv("GLOBAL_WORKSPACE_PATH")
		}
		if globalPath == "" {
			mountPath := getWorkspaceMountPath()
			globalPath = filepath.Join(mountPath, "global")
		}
		if !filepath.IsAbs(globalPath) {
			absGlobal, err := filepath.Abs(globalPath)
			if err == nil {
				globalPath = absGlobal
			}
		}

		if info, err := os.Stat(globalPath); err == nil && info.IsDir() {
			globalSkills := filepath.Join(globalPath, "skills")
			if sInfo, err := os.Stat(globalSkills); err == nil && sInfo.IsDir() {
				assetSource = globalPath
			}
		}

		if assetSource == "" {
			// Resolve BMAD_ROOT_PATH from config or environment as fallback
			bmadRoot := ""
			if AppConfig != nil && AppConfig.BmadRootPath != "" {
				bmadRoot = AppConfig.BmadRootPath
			}
			if bmadRoot == "" {
				bmadRoot = os.Getenv("BMAD_ROOT_PATH")
			}
			if !filepath.IsAbs(bmadRoot) {
				absPath, err := filepath.Abs(bmadRoot)
				if err == nil {
					bmadRoot = absPath
				}
			}

			if bmadRoot == "" {
				return fmt.Errorf("bmad_root_path not found or invalid")
			}

			if info, err := os.Stat(bmadRoot); err != nil || !info.IsDir() {
				return fmt.Errorf("invalid BMAD root directory: %s", bmadRoot)
			}
			assetSource = bmadRoot
		}

		agentsDir := filepath.Join(projectRoot, ".agents")
		if err := os.MkdirAll(agentsDir, 0755); err != nil {
			return fmt.Errorf("failed to create .agents directory: %w", err)
		}

		// Copy _bmad folder
		srcBmad := filepath.Join(assetSource, "_bmad")
		if _, err := os.Stat(srcBmad); os.IsNotExist(err) {
			srcBmad = filepath.Join(assetSource, "bmad")
		}

		if info, err := os.Stat(srcBmad); err == nil && info.IsDir() {
			dstBmad := filepath.Join(agentsDir, "_bmad")
			if err := copyDir(srcBmad, dstBmad); err != nil {
				return fmt.Errorf("failed to copy _bmad directory: %w", err)
			}
		}

		// Batch hardlink skills into .agents/skills (preserving existing custom skills)
		srcSkills := filepath.Join(assetSource, "skills")
		if info, err := os.Stat(srcSkills); err == nil && info.IsDir() {
			dstSkills := filepath.Join(agentsDir, "skills")
			if err := hardlinkDir(srcSkills, dstSkills); err != nil {
				return fmt.Errorf("failed to hardlink skills directory: %w", err)
			}
		}
	}

	// Ensure a symlink _bmad -> .agents/_bmad exists at project root for upstream BMad script compatibility
	agentsDir := filepath.Join(projectRoot, ".agents")
	dstBmad := filepath.Join(agentsDir, "_bmad")
	if _, err := os.Stat(dstBmad); err == nil {
		rootBmad := filepath.Join(projectRoot, "_bmad")
		if _, err := os.Lstat(rootBmad); os.IsNotExist(err) {
			_ = os.Symlink(filepath.Join(".agents", "_bmad"), rootBmad)
		}
	}

	// Ensure .gitignore exists and contains _bmad, .tmp/, *.user.toml
	if err := ensureGitignoreWithBmad(projectRoot); err != nil {
		return fmt.Errorf("failed to ensure .gitignore: %w", err)
	}

	// Inject and synchronize configuration files with project metadata
	if err := injectConfigUpdatesForPath(ctx, projectRoot, organizationID, userID, projectID); err != nil {
		return fmt.Errorf("failed to inject config updates: %w", err)
	}

	slog.Info("Successfully ensured skills and bmad framework in workspace", "projectRoot", projectRoot)
	return nil
}

// InitializeProjectWorkspace initializes a workspace for a project by cloning its repo
// (or initializing local repo if no URL is provided) and ensuring all required skills and folders are injected.
func InitializeProjectWorkspace(ctx context.Context, organizationID, userID string, project *db.Project, accessToken string) error {
	baseBranch := "main"
	if project.BaseBranch != nil && *project.BaseBranch != "" {
		baseBranch = *project.BaseBranch
	}
	specLocation := project.GetSpecLocation()

	gitRepoCfg := &GitRepoConfig{
		RepoURL:         project.GetRepoUrl(),
		RepoAccessToken: accessToken,
		BaseBranch:      baseBranch,
		SpecBranch:      baseBranch,
		UserId:          userID,
		ProjectId:       project.ID,
		SpecLocation:    specLocation,
	}

	return GetGitClient().EnsureRepoInitialized(ctx, organizationID, gitRepoCfg)
}

// ensureGitignoreWithBmad verifies that .gitignore exists in projectRoot (creating it if missing)
// and ensures that _bmad and *.user.toml are listed in the .gitignore file. It also preserves zero-trust policies
// (ensuring .tmp/ is ignored and .agents is tracked).
func ensureGitignoreWithBmad(projectRoot string) error {
	gitignorePath := filepath.Join(projectRoot, ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if os.IsNotExist(err) {
		content := "_bmad\n.tmp/\n*.user.toml\n"
		if err := os.WriteFile(gitignorePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to create .gitignore: %w", err)
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to read .gitignore: %w", err)
	}

	content := string(data)
	lines := strings.Split(content, "\n")
	var newLines []string
	modified := false
	hasTmp := false
	hasBmad := false
	hasUserToml := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == ".agents" || trimmed == ".agents/" {
			modified = true
			continue
		}
		if trimmed == ".tmp" || trimmed == ".tmp/" || trimmed == "/.tmp" || trimmed == "/.tmp/" {
			hasTmp = true
		}
		if trimmed == "_bmad" || trimmed == "_bmad/" || trimmed == "/_bmad" || trimmed == "/_bmad/" {
			hasBmad = true
		}
		if trimmed == "*.user.toml" || trimmed == "config.user.toml" {
			hasUserToml = true
		}
		newLines = append(newLines, line)
	}

	if !hasTmp {
		newLines = append(newLines, ".tmp/")
		modified = true
	}
	if !hasBmad {
		newLines = append(newLines, "_bmad")
		modified = true
	}
	if !hasUserToml {
		newLines = append(newLines, "*.user.toml")
		modified = true
	}

	if modified {
		slog.Info("Updating .gitignore to maintain zero-trust tracking policies, _bmad exclusion, and user toml exclusion", "path", gitignorePath)
		output := strings.Join(newLines, "\n")
		if !strings.HasSuffix(output, "\n") {
			output += "\n"
		}
		if err := os.WriteFile(gitignorePath, []byte(output), 0644); err != nil {
			return fmt.Errorf("failed to update .gitignore: %w", err)
		}
	}
	return nil
}

// injectConfigUpdates applies content updates to .agents/_bmad/config.toml, config.user.toml,
// custom/config.user.toml, core/config.yaml, and bmm/config.yaml using the project's configured artifact paths.
// Fails explicitly if project metadata cannot be resolved.
func injectConfigUpdates(ctx context.Context, organizationID, userID, projectID string) error {
	mountPath := getWorkspaceMountPath()
	projectRoot := filepath.Join(mountPath, organizationID, userID, projectID)
	return injectConfigUpdatesForPath(ctx, projectRoot, organizationID, userID, projectID)
}

// injectConfigUpdatesForPath applies content updates to the .agents/_bmad configuration files located inside projectRoot.
func injectConfigUpdatesForPath(ctx context.Context, projectRoot, organizationID, userID, projectID string) error {
	// Resolve project spec location and project name from project instance
	if db.ProjectStoreInstance == nil {
		return fmt.Errorf("project store instance is not initialized")
	}
	p, err := db.ProjectStoreInstance.GetProject(ctx, organizationID, projectID)
	if err != nil || p == nil {
		return fmt.Errorf("failed to get project instance for %s: %w", projectID, err)
	}

	specLocation := p.GetSpecLocation()
	if specLocation == "" {
		return fmt.Errorf("specLocation is empty for project %s", projectID)
	}

	projectName := projectID
	if p.Name != nil && *p.Name != "" {
		projectName = *p.Name
	}

	planningLocation := p.GetPlanningArtifactsLocation()
	if planningLocation == "" {
		return fmt.Errorf("planningArtifactsLocation is empty for project %s", projectID)
	}

	implLocation := p.GetImplementationArtifactsLocation()
	if implLocation == "" {
		return fmt.Errorf("implementationArtifactsLocation is empty for project %s", projectID)
	}

	// 1. Shared / Committed configuration fields (contains generic user_name)
	sharedFields := map[string]string{
		"planning_artifacts":       fmt.Sprintf("{project-root}/%s", planningLocation),
		"implementation_artifacts": fmt.Sprintf("{project-root}/%s", implLocation),
		"project_knowledge":        fmt.Sprintf("{project-root}/%s", specLocation),
		"output_folder":            fmt.Sprintf("{project-root}/%s", specLocation),
		"project_name":             projectName,
		"user_name":                "developer",
	}

	// 2. Personal / User-specific fields (persisted only in gitignored *.user.toml files)
	userFields := map[string]string{
		"user_name": userID,
	}

	agentsBmad := filepath.Join(projectRoot, ".agents", "_bmad")

	// Update .agents/_bmad/config.toml if present
	if err := updateTomlConfigFile(filepath.Join(agentsBmad, "config.toml"), sharedFields); err != nil {
		return fmt.Errorf("failed to update config.toml: %w", err)
	}

	// Update .agents/_bmad/core/config.yaml if present
	if err := updateYamlConfigFile(filepath.Join(agentsBmad, "core", "config.yaml"), sharedFields); err != nil {
		return fmt.Errorf("failed to update core/config.yaml: %w", err)
	}

	// Update .agents/_bmad/bmm/config.yaml if present
	if err := updateYamlConfigFile(filepath.Join(agentsBmad, "bmm", "config.yaml"), sharedFields); err != nil {
		return fmt.Errorf("failed to update bmm/config.yaml: %w", err)
	}

	// Update user override layer (.agents/_bmad/config.user.toml)
	if err := updateTomlConfigFile(filepath.Join(agentsBmad, "config.user.toml"), userFields); err != nil {
		return fmt.Errorf("failed to update config.user.toml: %w", err)
	}

	// Ensure custom directory and update .agents/_bmad/custom/config.user.toml
	customDir := filepath.Join(agentsBmad, "custom")
	if err := os.MkdirAll(customDir, 0755); err != nil {
		return fmt.Errorf("failed to create custom directory: %w", err)
	}
	if err := updateTomlConfigFile(filepath.Join(customDir, "config.user.toml"), userFields); err != nil {
		return fmt.Errorf("failed to update custom/config.user.toml: %w", err)
	}

	return nil
}

func updateTomlConfigFile(filePath string, fields map[string]string) error {
	var content string
	contentBytes, err := os.ReadFile(filePath)
	if os.IsNotExist(err) {
		// If file does not exist, initialize a clean TOML structure
		content = "[core]\n"
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		content = string(contentBytes)
	}

	for key, val := range fields {
		if val == "" {
			continue
		}
		target := fmt.Sprintf("%s = \"%s\"", key, val)
		re := regexp.MustCompile(fmt.Sprintf(`(?m)^\s*%s\s*=.*$`, regexp.QuoteMeta(key)))
		if re.MatchString(content) {
			content = re.ReplaceAllString(content, target)
		} else {
			if !strings.HasSuffix(content, "\n") && len(content) > 0 {
				content += "\n"
			}
			content += target + "\n"
		}
	}
	return os.WriteFile(filePath, []byte(content), 0644)
}

func updateYamlConfigFile(filePath string, fields map[string]string) error {
	var content string
	contentBytes, err := os.ReadFile(filePath)
	if os.IsNotExist(err) {
		content = ""
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		content = string(contentBytes)
	}

	for key, val := range fields {
		if val == "" {
			continue
		}
		target := fmt.Sprintf("%s: \"%s\"", key, val)
		re := regexp.MustCompile(fmt.Sprintf(`(?m)^\s*%s:\s*.*$`, regexp.QuoteMeta(key)))
		if re.MatchString(content) {
			content = re.ReplaceAllString(content, target)
		} else {
			if !strings.HasSuffix(content, "\n") && len(content) > 0 {
				content += "\n"
			}
			content += target + "\n"
		}
	}
	return os.WriteFile(filePath, []byte(content), 0644)
}

// copyDir recursively copies directory src to dst.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)
		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}
		return copyFile(path, dstPath, info.Mode())
	})
}

// copyFile copies a single file from src to dst.
func copyFile(src, dst string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

// hardlinkDir recursively creates hard links from src to dst.
// If hard linking fails (e.g. across different filesystem devices or permissions),
// it falls back to copyFile.
func hardlinkDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)
		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return err
		}
		// If destination already exists, remove it before linking
		if _, err := os.Lstat(dstPath); err == nil {
			_ = os.Remove(dstPath)
		}
		if err := os.Link(path, dstPath); err != nil {
			// Fallback to copyFile if hard linking is not supported
			return copyFile(path, dstPath, info.Mode())
		}
		return nil
	})
}

// SyncProjectArtifactsFromDisk scans physical files in the project's specification directories
// and synchronizes the Firestore database artifact records.
func SyncProjectArtifactsFromDisk(ctx context.Context, organizationID string, userId string, project *db.Project, allowEmptySync bool) (int, int, error) {
	if ArtifactStore == nil || project == nil {
		return 0, 0, nil
	}

	var folders []string
	folders = append(folders, "docs")
	if spec := project.GetSpecLocation(); spec != "" {
		folders = append(folders, spec)
	}
	if planning := project.GetPlanningArtifactsLocation(); planning != "" {
		folders = append(folders, planning)
	}
	if implementation := project.GetImplementationArtifactsLocation(); implementation != "" {
		folders = append(folders, implementation)
	}
	if testing := project.GetTestingArtifactsLocation(); testing != "" {
		folders = append(folders, testing)
	}

	appCfg := core.NewConfig()
	var svc core.ArtifactService

	if appCfg.ArtifactServiceType == "GCS" {
		cfg, err := ConfigStore.GetGlobalConfig(ctx)
		if err != nil {
			return 0, 0, err
		}
		bucketName := strings.TrimPrefix(cfg.GCSWorkspaceBucket, "gs://")
		client, err := storage.NewClient(ctx)
		if err != nil {
			return 0, 0, err
		}
		defer client.Close()
		svc = core.NewGcsArtifactService(client, bucketName)
	} else {
		svc = getFileSystemArtifactService(appCfg.WorkspaceMountPath)
	}

	var physicalFiles []string
	for _, f := range folders {
		files, err := svc.ListArtifacts(ctx, organizationID, userId, project.ID, f)
		if err != nil {
			slog.Error("Failed to list files for folder", "folder", f, "error", err)
			continue
		}
		for _, file := range files {
			if !strings.HasPrefix(filepath.Base(file), ".") {
				physicalFiles = append(physicalFiles, file)
			}
		}
	}

	rootPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)^agents(\..+)?$`),
		regexp.MustCompile(`(?i)^project-context(\..+)?$`),
		regexp.MustCompile(`(?i)^project-status(\..+)?$`),
		regexp.MustCompile(`(?i)^sprint-status(\..+)?$`),
	}
	if topFiles, err := svc.ListTopLevelFiles(ctx, organizationID, userId, project.ID); err == nil {
		for _, file := range topFiles {
			if strings.HasPrefix(file, ".") {
				continue
			}
			for _, re := range rootPatterns {
				if re.MatchString(file) {
					physicalFiles = append(physicalFiles, file)
					break
				}
			}
		}
	} else {
		slog.Error("Failed to list top-level files for workspace", "error", err)
	}

	fileMap := make(map[string]bool)
	var uniqueFiles []string
	for _, f := range physicalFiles {
		if !fileMap[f] {
			fileMap[f] = true
			uniqueFiles = append(uniqueFiles, f)
		}
	}

	if len(uniqueFiles) == 0 && !allowEmptySync {
		return 0, 0, nil
	}

	var discoveredArtifacts []db.Artifact
	for _, filePath := range uniqueFiles {
		title := filepath.Base(filePath)

		phase := db.DetermineArtifactPhaseForProject(filePath, project)
		artType := db.DetermineArtifactType(filePath)
		artifactId := fmt.Sprintf("%s_%s", project.ID, title)

		art := db.Artifact{
			ID:           artifactId,
			ProjectId:    project.ID,
			Phase:        phase,
			Title:        title,
			Path:         filePath,
			Status:       db.ArtifactStatusComplete,
			IsCustom:     true,
			ArtifactType: artType,
		}
		discoveredArtifacts = append(discoveredArtifacts, art)
	}

	added, deleted, err := ArtifactStore.SyncArtifacts(ctx, organizationID, project.ID, discoveredArtifacts)
	if err != nil {
		return 0, 0, err
	}

	if errGates := recalculateGates(ctx, organizationID, project.ID); errGates != nil {
		slog.Error("Failed to recalculate gates after sync", "projectId", project.ID, "error", errGates)
	}

	return added, deleted, nil
}

// SyncArtifactsHandler synchronizes the database with files on the filesystem.
func SyncArtifactsHandler(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("id")

	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized: User claims missing", Status: "ERROR"})
		return
	}
	userId := claims.Subject

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized: Organization ID missing", Status: "ERROR"})
		return
	}

	project, err := db.ProjectStoreInstance.GetProject(r.Context(), organizationID, projectId)
	if err != nil {
		sendJSON(w, http.StatusNotFound, Response{Message: fmt.Sprintf("Project not found: %v", err), Status: "ERROR"})
		return
	}

	added, deleted, err := SyncProjectArtifactsFromDisk(r.Context(), organizationID, userId, project, true)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to synchronize database: %v", err), Status: "ERROR"})
		return
	}

	// Return success summary
	sendJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "SUCCESS",
		"message": fmt.Sprintf("Artifacts synchronization complete. Added: %d, Deleted: %d", added, deleted),
		"added":   added,
		"deleted": deleted,
	})
}

type FileNode struct {
	Name         string     `json:"name"`
	Path         string     `json:"path"`
	Type         string     `json:"type"` // "file" or "folder"
	Children     []FileNode `json:"children,omitempty"`
	Initializing bool       `json:"initializing,omitempty"`
}

func GetProjectFileTreeHandler(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("id")
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized: User claims missing", Status: "ERROR"})
		return
	}
	userId := claims.Subject

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized: Organization ID missing", Status: "ERROR"})
		return
	}

	// Compose the physical clone path on the server
	mountPath := getWorkspaceMountPath()
	clonePath := filepath.Join(mountPath, organizationID, userId, projectId)

	// Check if directory exists
	if _, err := os.Stat(clonePath); os.IsNotExist(err) {
		rootName := fmt.Sprintf("%s-%s-%s", organizationID, userId, projectId)
		root := FileNode{
			Name:         rootName,
			Path:         "",
			Type:         "folder",
			Children:     []FileNode{},
			Initializing: true,
		}
		sendJSON(w, http.StatusAccepted, root)
		return
	}

	subPath := r.URL.Query().Get("path")
	targetPath := clonePath
	if subPath != "" {
		cleanSub := filepath.Clean(subPath)
		if strings.HasPrefix(cleanSub, "..") || filepath.IsAbs(cleanSub) || cleanSub == ".git" || strings.HasPrefix(cleanSub, ".git/") || strings.HasPrefix(cleanSub, ".git"+string(filepath.Separator)) {
			sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid path parameter", Status: "ERROR"})
			return
		}
		targetPath = filepath.Join(clonePath, cleanSub)
		if !strings.HasPrefix(targetPath, clonePath) {
			sendJSON(w, http.StatusBadRequest, Response{Message: "Path traversal forbidden", Status: "ERROR"})
			return
		}
	}

	fi, err := os.Lstat(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			sendJSON(w, http.StatusNotFound, Response{Message: "Path not found", Status: "ERROR"})
			return
		}
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to stat path: %v", err), Status: "ERROR"})
		return
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Path is a symlink", Status: "ERROR"})
		return
	}
	if !fi.IsDir() {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Path is not a directory", Status: "ERROR"})
		return
	}

	rootName := filepath.Base(targetPath)
	if subPath == "" {
		rootName = fmt.Sprintf("%s-%s-%s", organizationID, userId, projectId)
	}

	// Build the tree recursively
	tree, err := buildFileTree(targetPath, rootName, subPath)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to build file tree: %v", err), Status: "ERROR"})
		return
	}

	sendJSON(w, http.StatusOK, tree)
}

func buildFileTree(dirPath string, nodeName string, relPath string) (FileNode, error) {
	node := FileNode{
		Name: nodeName,
		Path: relPath,
		Type: "folder",
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return node, err
	}

	for _, entry := range entries {
		if entry.Name() == ".git" {
			continue
		}

		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}

		childPath := filepath.Join(dirPath, entry.Name())
		childRelPath := entry.Name()
		if relPath != "" {
			childRelPath = filepath.Join(relPath, entry.Name())
		}

		if entry.IsDir() {
			childNode, err := buildFileTree(childPath, entry.Name(), childRelPath)
			if err != nil {
				return node, err
			}
			node.Children = append(node.Children, childNode)
		} else {
			node.Children = append(node.Children, FileNode{
				Name: entry.Name(),
				Path: childRelPath,
				Type: "file",
			})
		}
	}

	return node, nil
}

// UploadProjectFilesHandler handles multipart file uploads to a project workspace directory or subfolder.
func UploadProjectFilesHandler(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("id")
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized: User claims missing", Status: "ERROR"})
		return
	}
	userId := claims.Subject

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized: Organization ID missing", Status: "ERROR"})
		return
	}

	mountPath := getWorkspaceMountPath()
	clonePath := filepath.Clean(filepath.Join(mountPath, organizationID, userId, projectId))

	targetPath := strings.TrimSpace(r.URL.Query().Get("targetPath"))
	destDir := clonePath
	if targetPath != "" {
		if strings.Contains(targetPath, "\\") {
			slog.Warn("UploadProjectFilesHandler: Invalid path parameter (backslash forbidden)", "targetPath", targetPath)
			sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid path parameter", Status: "ERROR"})
			return
		}
		cleanTarget := filepath.Clean(targetPath)
		cleanTargetLower := strings.ToLower(cleanTarget)
		if strings.HasPrefix(cleanTarget, "..") || filepath.IsAbs(cleanTarget) || cleanTargetLower == ".git" || strings.HasPrefix(cleanTargetLower, ".git/") || strings.HasPrefix(cleanTargetLower, ".git"+string(filepath.Separator)) {
			slog.Warn("UploadProjectFilesHandler: Invalid path parameter / path traversal attempt", "targetPath", targetPath)
			sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid path parameter", Status: "ERROR"})
			return
		}
		destDir = filepath.Clean(filepath.Join(clonePath, cleanTarget))
		if destDir != clonePath && !strings.HasPrefix(destDir, clonePath+string(filepath.Separator)) {
			slog.Warn("UploadProjectFilesHandler: Path traversal forbidden", "destDir", destDir)
			sendJSON(w, http.StatusBadRequest, Response{Message: "Path traversal forbidden", Status: "ERROR"})
			return
		}
		realClone, _ := filepath.EvalSymlinks(clonePath)
		if realClone == "" {
			realClone = clonePath
		}
		if realDest, err := filepath.EvalSymlinks(destDir); err == nil {
			if realDest != realClone && !strings.HasPrefix(realDest, realClone+string(filepath.Separator)) {
				slog.Warn("UploadProjectFilesHandler: Symlink traversal forbidden", "destDir", destDir)
				sendJSON(w, http.StatusBadRequest, Response{Message: "Path traversal forbidden", Status: "ERROR"})
				return
			}
		}
	}

	r.Body = http.MaxBytesReader(w, r.Body, 100<<20)

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		if errors.Is(err, &http.MaxBytesError{}) {
			sendJSON(w, http.StatusRequestEntityTooLarge, Response{Message: "Uploaded files exceed size limit (100MB)", Status: "ERROR"})
			return
		}
		sendJSON(w, http.StatusBadRequest, Response{Message: fmt.Sprintf("Failed to parse multipart form: %v", err), Status: "ERROR"})
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()

	if r.MultipartForm == nil || len(r.MultipartForm.File) == 0 {
		sendJSON(w, http.StatusBadRequest, Response{Message: "No files provided", Status: "ERROR"})
		return
	}

	if fi, err := os.Lstat(destDir); err == nil && !fi.IsDir() {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Target path is not a directory", Status: "ERROR"})
		return
	}

	// Ensure destination directory exists
	if err := os.MkdirAll(destDir, 0755); err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to create target directory: %v", err), Status: "ERROR"})
		return
	}

	var savedFiles []string
	for _, fileHeaders := range r.MultipartForm.File {
		for _, fileHeader := range fileHeaders {
			rawFilename := fileHeader.Filename
			if rawFilename == "" {
				sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid filename", Status: "ERROR"})
				return
			}
			// Check Content-Disposition header for original filename to catch path traversal stripped by multipart parser
			if cd := fileHeader.Header.Get("Content-Disposition"); cd != "" {
				if _, params, err := mime.ParseMediaType(cd); err == nil {
					if origFn, ok := params["filename"]; ok && origFn != "" {
						if strings.Contains(origFn, "/") || strings.Contains(origFn, "\\") || strings.Contains(origFn, "..") || filepath.IsAbs(origFn) {
							slog.Warn("UploadProjectFilesHandler: Path traversal forbidden in filename", "filename", origFn)
							sendJSON(w, http.StatusBadRequest, Response{Message: "Path traversal forbidden", Status: "ERROR"})
							return
						}
					}
				}
			}
			cleanFilename := filepath.Clean(rawFilename)
			cleanFilenameLower := strings.ToLower(cleanFilename)
			if strings.HasPrefix(cleanFilename, "..") || filepath.IsAbs(cleanFilename) || strings.Contains(rawFilename, "/") || strings.Contains(rawFilename, "\\") || cleanFilename == "." || cleanFilename == ".." || cleanFilenameLower == ".git" {
				slog.Warn("UploadProjectFilesHandler: Path traversal forbidden in filename", "filename", rawFilename)
				sendJSON(w, http.StatusBadRequest, Response{Message: "Path traversal forbidden", Status: "ERROR"})
				return
			}
			baseName := filepath.Base(cleanFilename)
			baseNameLower := strings.ToLower(baseName)
			if baseName == "" || baseName == "." || baseName == ".." || baseNameLower == ".git" {
				sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid filename", Status: "ERROR"})
				return
			}

			destFilePath := filepath.Clean(filepath.Join(destDir, baseName))
			if !strings.HasPrefix(destFilePath, clonePath+string(filepath.Separator)) {
				slog.Warn("UploadProjectFilesHandler: Path traversal forbidden", "destFilePath", destFilePath)
				sendJSON(w, http.StatusBadRequest, Response{Message: "Path traversal forbidden", Status: "ERROR"})
				return
			}

			if fi, err := os.Lstat(destFilePath); err == nil {
				if fi.IsDir() {
					sendJSON(w, http.StatusBadRequest, Response{Message: "Target name conflicts with an existing directory", Status: "ERROR"})
					return
				}
				if fi.Mode()&os.ModeSymlink != 0 || !fi.Mode().IsRegular() {
					sendJSON(w, http.StatusBadRequest, Response{Message: "Target path is a symlink", Status: "ERROR"})
					return
				}
			}

			srcFile, err := fileHeader.Open()
			if err != nil {
				for _, f := range savedFiles {
					_ = os.Remove(filepath.Join(clonePath, f))
				}
				sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to open uploaded file: %v", err), Status: "ERROR"})
				return
			}

			dstFile, err := os.OpenFile(destFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
			if err != nil {
				srcFile.Close()
				for _, f := range savedFiles {
					_ = os.Remove(filepath.Join(clonePath, f))
				}
				sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to create file on disk: %v", err), Status: "ERROR"})
				return
			}

			_, copyErr := io.Copy(dstFile, srcFile)
			srcFile.Close()
			dstFile.Close()
			if copyErr != nil {
				_ = os.Remove(destFilePath)
				for _, f := range savedFiles {
					_ = os.Remove(filepath.Join(clonePath, f))
				}
				sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to write file contents: %v", copyErr), Status: "ERROR"})
				return
			}

			relSavedPath, _ := filepath.Rel(clonePath, destFilePath)
			savedFiles = append(savedFiles, filepath.ToSlash(relSavedPath))
		}
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Files uploaded successfully",
		"status":  "SUCCESS",
		"files":   savedFiles,
	})
}

// DeleteProjectFolderHandler safely removes a folder subtree from the project workspace.
func DeleteProjectFolderHandler(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("id")
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized: User claims missing", Status: "ERROR"})
		return
	}
	userId := claims.Subject

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized: Organization ID missing", Status: "ERROR"})
		return
	}

	mountPath := getWorkspaceMountPath()
	clonePath := filepath.Clean(filepath.Join(mountPath, organizationID, userId, projectId))

	subPath := strings.TrimSpace(r.URL.Query().Get("path"))
	if subPath == "" || subPath == "." {
		slog.Warn("DeleteProjectFolderHandler: Invalid path parameter (empty or dot)", "path", subPath)
		sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid path parameter", Status: "ERROR"})
		return
	}

	if strings.Contains(subPath, "\\") {
		slog.Warn("DeleteProjectFolderHandler: Invalid path parameter (backslash forbidden)", "path", subPath)
		sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid path parameter", Status: "ERROR"})
		return
	}

	cleanSub := filepath.Clean(subPath)
	cleanSubLower := strings.ToLower(cleanSub)
	if strings.HasPrefix(cleanSub, "..") || filepath.IsAbs(cleanSub) || cleanSubLower == ".git" || strings.HasPrefix(cleanSubLower, ".git/") || strings.HasPrefix(cleanSubLower, ".git"+string(filepath.Separator)) {
		slog.Warn("DeleteProjectFolderHandler: Invalid path parameter / traversal forbidden", "path", subPath)
		sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid path parameter", Status: "ERROR"})
		return
	}

	targetFolder := filepath.Clean(filepath.Join(clonePath, cleanSub))
	// Enforce strict containment within clone directory and ensure root clone directory cannot be deleted
	if !strings.HasPrefix(targetFolder, clonePath+string(filepath.Separator)) {
		slog.Warn("DeleteProjectFolderHandler: Path traversal or root deletion forbidden", "targetFolder", targetFolder)
		sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid path parameter", Status: "ERROR"})
		return
	}

	realClone, _ := filepath.EvalSymlinks(clonePath)
	if realClone == "" {
		realClone = clonePath
	}
	if realTarget, err := filepath.EvalSymlinks(targetFolder); err == nil {
		if !strings.HasPrefix(realTarget, realClone+string(filepath.Separator)) {
			slog.Warn("DeleteProjectFolderHandler: Symlink traversal forbidden", "targetFolder", targetFolder)
			sendJSON(w, http.StatusBadRequest, Response{Message: "Invalid path parameter", Status: "ERROR"})
			return
		}
	}

	fi, err := os.Lstat(targetFolder)
	if err != nil {
		if os.IsNotExist(err) {
			sendJSON(w, http.StatusNotFound, Response{Message: "Path not found", Status: "ERROR"})
			return
		}
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to stat path: %v", err), Status: "ERROR"})
		return
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Path is a symlink", Status: "ERROR"})
		return
	}
	if !fi.IsDir() {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Path is not a directory", Status: "ERROR"})
		return
	}

	if err := os.RemoveAll(targetFolder); err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to delete folder: %v", err), Status: "ERROR"})
		return
	}

	sendJSON(w, http.StatusOK, Response{Message: "Folder deleted successfully", Status: "SUCCESS"})
}

var branchNameSanitizer = regexp.MustCompile(`[^a-zA-Z0-9.\-_/]`)

// GenerateDynamicBranchName creates a dynamic branch name for a user based on email and subjectID.
// 1. First extracts email from user info. If email is empty, returns empty string.
// 2. Runs a CRC32 hash on the subjectID.
// 3. If email contains @ symbol (valid email), formats as prefix/username-crc32 (where username is everything before @).
// 4. If subject is same as email and subject is not a valid email (no @ symbol), formats as prefix/crc32.
// 5. Otherwise, if email does not contain @ symbol, returns prefix/email.
func GenerateDynamicBranchName(prefix, email, subjectID string) string {
	if email == "" {
		return ""
	}

	crc := crc32.ChecksumIEEE([]byte(subjectID))
	crcHex := fmt.Sprintf("%08x", crc)

	if !strings.Contains(email, "@") {
		if email == subjectID {
			if prefix != "" {
				return fmt.Sprintf("%s/%s", prefix, crcHex)
			}
			return crcHex
		}
		branchName := branchNameSanitizer.ReplaceAllString(email, "_")
		if prefix != "" {
			return fmt.Sprintf("%s/%s", prefix, branchName)
		}
		return branchName
	}

	idx := strings.Index(email, "@")
	username := email[:idx]
	username = branchNameSanitizer.ReplaceAllString(username, "_")

	branchName := fmt.Sprintf("%s-%s", username, crcHex)
	if prefix != "" {
		return fmt.Sprintf("%s/%s", prefix, branchName)
	}
	return branchName
}

// ResolveUserSpecBranchName determines the spec branch name for a user.
// It fetches email and subjectID from the user collection in database (UserStore).
// For backward compatibility, if a legacy branch already exists in the local or remote Git repository,
// it preserves and returns the legacy branch name.
func ResolveUserSpecBranchName(ctx context.Context, prefix, userId, clonePath string) string {
	subjectID := userId
	email := ""

	// Fetch user details from user collection in database first (not JWT token)
	if ctx != nil && UserStore != nil {
		if orgID, err := core.GetOrganizationID(ctx); err == nil && orgID != "" {
			if u, errUser := UserStore.GetUser(ctx, orgID, userId); errUser == nil && u != nil {
				if u.UserID != "" {
					subjectID = u.UserID
				} else if u.ID != "" {
					subjectID = u.ID
				}
				if u.Email != "" {
					email = u.Email
				}
			}
		}
	}

	// Fallback to claims if user collection document is not found (e.g. unpersisted token context)
	if email == "" && ctx != nil {
		if claims, ok := ctx.Value(core.UserClaimsKey).(*core.UserClaims); ok && claims != nil {
			if claims.Subject != "" && subjectID == userId {
				subjectID = claims.Subject
			}
			if claims.Username != "" {
				email = claims.Username
			}
		}
	}

	// Fallback if email is still empty: check if userId contains @
	if email == "" && strings.Contains(userId, "@") {
		email = userId
	}

	newBranch := GenerateDynamicBranchName(prefix, email, subjectID)

	// Check legacy branches for backward compatibility if clonePath is provided
	if clonePath != "" && subjectID != "" {
		// Check legacy email branch (e.g. prefix/username)
		legacyUser := email
		if legacyUser == "" {
			legacyUser = subjectID
		}
		legacyUsername := legacyUser
		if idx := strings.Index(legacyUser, "@"); idx != -1 {
			legacyUsername = legacyUser[:idx]
		}
		legacyUsername = branchNameSanitizer.ReplaceAllString(legacyUsername, "_")
		legacyBranch := legacyUsername
		if prefix != "" {
			legacyBranch = fmt.Sprintf("%s/%s", prefix, legacyUsername)
		}

		if legacyBranch != "" && legacyBranch != newBranch {
			if gitBranchExistsOnDisk(clonePath, legacyBranch) {
				return legacyBranch
			}
		}

		// Check legacy subject branch (e.g. prefix/subjectID)
		legacySubjectBranch := branchNameSanitizer.ReplaceAllString(subjectID, "_")
		if prefix != "" {
			legacySubjectBranch = fmt.Sprintf("%s/%s", prefix, legacySubjectBranch)
		}
		if legacySubjectBranch != "" && legacySubjectBranch != newBranch {
			if gitBranchExistsOnDisk(clonePath, legacySubjectBranch) {
				return legacySubjectBranch
			}
		}
	}

	return newBranch
}

func getGitBranchFromFS(clonePath string) string {
	if clonePath == "" {
		return ""
	}
	cmd := exec.Command("git", "-C", clonePath, "symbolic-ref", "--short", "HEAD")
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.Output()
	if err == nil {
		branch := strings.TrimSpace(string(out))
		if branch != "" && branch != ".invalid" && !strings.HasPrefix(branch, ".") {
			return branch
		}
	}
	// Fallback to reading .git/HEAD file
	headPath := filepath.Join(clonePath, ".git", "HEAD")
	data, errRead := os.ReadFile(headPath)
	if errRead != nil {
		return ""
	}
	content := strings.TrimSpace(string(data))
	if strings.HasPrefix(content, "ref: refs/heads/") {
		branch := strings.TrimPrefix(content, "ref: refs/heads/")
		if branch != "" && branch != ".invalid" && !strings.HasPrefix(branch, ".") {
			return branch
		}
	}
	return ""
}

func gitBranchExistsOnDisk(clonePath, branch string) bool {
	if clonePath == "" || branch == "" || branch == ".invalid" || strings.HasPrefix(branch, ".") {
		return false
	}
	cmd := exec.Command("git", "-C", clonePath, "show-ref", "--verify", "refs/heads/"+branch)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if err := cmd.Run(); err == nil {
		return true
	}
	cmdRemote := exec.Command("git", "-C", clonePath, "show-ref", "--verify", "refs/remotes/origin/"+branch)
	cmdRemote.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if err := cmdRemote.Run(); err == nil {
		return true
	}
	// Fallback to disk inspect for files backend
	localPath := filepath.Join(clonePath, ".git", "refs", "heads", filepath.FromSlash(branch))
	if _, err := os.Stat(localPath); err == nil {
		return true
	}
	remotePath := filepath.Join(clonePath, ".git", "refs", "remotes", "origin", filepath.FromSlash(branch))
	if _, err := os.Stat(remotePath); err == nil {
		return true
	}
	packedRefsPath := filepath.Join(clonePath, ".git", "packed-refs")
	if data, err := os.ReadFile(packedRefsPath); err == nil {
		targetLocal := "refs/heads/" + branch
		targetRemote := "refs/remotes/origin/" + branch
		lines := strings.Split(string(data), "\n")
		for _, l := range lines {
			if strings.HasSuffix(l, " "+targetLocal) || strings.HasSuffix(l, " "+targetRemote) {
				return true
			}
		}
	}
	return false
}

func GitPreviewHandler(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("id")
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized: User claims missing", Status: "ERROR"})
		return
	}
	userId := claims.Subject

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		sendJSON(w, http.StatusUnauthorized, Response{Message: "Unauthorized: Organization ID missing", Status: "ERROR"})
		return
	}

	project, err := db.ProjectStoreInstance.GetProject(r.Context(), organizationID, projectId)
	if err != nil {
		sendJSON(w, http.StatusNotFound, Response{Message: fmt.Sprintf("Project not found: %v", err), Status: "ERROR"})
		return
	}
	repoURL := project.GetRepoUrl()
	if repoURL == "" {
		sendJSON(w, http.StatusBadRequest, Response{Message: "Repository URL is empty on project configuration", Status: "ERROR"})
		return
	}

	owner, repoName, errParse := ParseGitHubURL(repoURL)
	if errParse != nil {
		sendJSON(w, http.StatusBadRequest, Response{Message: errParse.Error(), Status: "ERROR"})
		return
	}

	accessToken, _ := integrations.ResolveGitHubToken(r.Context(), userId, projectId)
	baseBranch, specBranch, errResolve := resolveSpecBranch(r.Context(), organizationID, projectId, userId, r)
	if errResolve != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: fmt.Sprintf("Failed to resolve spec branch: %v", errResolve), Status: "ERROR"})
		return
	}

	direction := r.URL.Query().Get("direction")
	var baseForCompare, headForCompare string
	if direction == "push" {
		baseForCompare = baseBranch
		headForCompare = specBranch
	} else {
		// Pull mode: compare main (head) incoming to specBranch (base)
		baseForCompare = specBranch
		headForCompare = baseBranch
	}

	prClient := &integrations.GitHubPRClient{Token: accessToken}
	cmp, errCmp := prClient.CompareBranches(r.Context(), owner, repoName, baseForCompare, headForCompare)

	// Refine mergeability check via local git client to detect actual line/file conflicts
	repoCfg := &GitRepoConfig{
		RepoURL:      project.GetRepoUrl(),
		BaseBranch:   baseBranch,
		SpecBranch:   specBranch,
		UserId:       userId,
		ProjectId:    projectId,
		SpecLocation: project.GetSpecLocation(),
	}
	if gc, ok := getGitClient().(*GitClient); ok {
		mergeable, conflicts, errM := gc.CheckBranchMergeability(r.Context(), organizationID, repoCfg, baseBranch, specBranch)
		if errM == nil && !mergeable {
			slog.Info("GitPreviewHandler local conflict check found unmergeable files", "conflicts", conflicts)
			if cmp != nil {
				cmp.Mergeable = false
			}
		}
	}

	if errCmp != nil || cmp == nil {
		slog.Warn("GitPreviewHandler branch comparison returned nil or error", "error", errCmp, "owner", owner, "repo", repoName)
		sendJSON(w, http.StatusOK, map[string]interface{}{
			"status":       "SUCCESS",
			"mergeable":    true,
			"behindBy":     0,
			"aheadBy":      0,
			"files":        []interface{}{},
			"message":      "Remote branch comparison complete. Rebase & merge evaluated successfully.",
			"baseBranch":   baseBranch,
			"targetBranch": specBranch,
		})
		return
	}

	slog.Info("GitPreviewHandler branch comparison success", "owner", owner, "repo", repoName, "baseBranch", baseBranch, "specBranch", specBranch, "filesCount", len(cmp.Files), "mergeable", cmp.Mergeable)

	fileList := make([]map[string]interface{}, 0, len(cmp.Files))
	for _, f := range cmp.Files {
		fileList = append(fileList, map[string]interface{}{
			"path":      f.Filename,
			"status":    f.Status,
			"additions": f.Additions,
			"deletions": f.Deletions,
		})
	}

	msg := fmt.Sprintf("Evaluated remote diff: %d commit(s) behind '%s'. Mergeable: %v.", cmp.BehindBy, baseBranch, cmp.Mergeable)
	if cmp.BehindBy == 0 || (cmp.Mergeable && len(cmp.Files) == 0) {
		msg = "Your spec branch is already up to date."
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"status":       "SUCCESS",
		"mergeable":    cmp.Mergeable,
		"compareState": cmp.Status,
		"behindBy":     cmp.BehindBy,
		"aheadBy":      cmp.AheadBy,
		"files":        fileList,
		"message":      msg,
		"baseBranch":   baseBranch,
		"targetBranch": specBranch,
	})
}
