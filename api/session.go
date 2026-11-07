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

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	firestoregorilla "github.com/GoogleCloudPlatform/firestore-gorilla-sessions"
	"github.com/google/uuid"
	"google.com/apf/api/agent"
	core "google.com/apf/api/core"
	"google.com/apf/api/db"
	"google.com/apf/api/integrations"
	"google.com/apf/api/workspace"
)

type SessionResponse struct {
	Session db.Session `json:"session"`
}

// SessionHandler encapsulates dependencies for session endpoints.
type SessionHandler struct {
	SessionStore       db.SessionStore
	Store              db.MessageStore
	SessionCookieStore *firestoregorilla.Store
	ArtifactStore      db.ArtifactStore
	ArtifactService    core.ArtifactService
	AppConfig          *core.Configuration
}

// NewSessionHandler creates a new SessionHandler instance.
func NewSessionHandler(sessionStore db.SessionStore, store db.MessageStore, cookieStore *firestoregorilla.Store) *SessionHandler {
	return &SessionHandler{
		SessionStore:       sessionStore,
		Store:              store,
		SessionCookieStore: cookieStore,
	}
}

// WithArtifactStore sets the ArtifactStore dependency.
func (h *SessionHandler) WithArtifactStore(store db.ArtifactStore) *SessionHandler {
	h.ArtifactStore = store
	return h
}

// WithArtifactService sets the ArtifactService dependency.
func (h *SessionHandler) WithArtifactService(svc core.ArtifactService) *SessionHandler {
	h.ArtifactService = svc
	return h
}

// WithAppConfig sets the Configuration dependency.
func (h *SessionHandler) WithAppConfig(cfg *core.Configuration) *SessionHandler {
	h.AppConfig = cfg
	return h
}

func (h *SessionHandler) connectSessionHandler(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("id")
	claims := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		sendJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized: Organization ID missing"})
		return
	}

	var sess db.Session
	if err := json.NewDecoder(r.Body).Decode(&sess); err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
		return
	}

	if h.SessionCookieStore != nil {
		session, err := h.SessionCookieStore.Get(r, "session-auth")
		if err == nil {
			session.Values["claims"] = claims
			session.Values["projectId"] = projectId
			session.Values["phase"] = sess.Phase
			session.Save(r, w)
		}
	}

	createNew := r.URL.Query().Get("createNew") == "true"
	project, err := store.GetProject(r.Context(), organizationID, projectId)
	if err != nil {
		sendJSON(w, http.StatusNotFound, map[string]string{"error": fmt.Sprintf("Project not found: %v", err)})
		return
	}

	var connectedSession db.Session
	var statusCode int

	var sessions []db.Session
	if !createNew {
		sessions, err = h.SessionStore.ListSessionsForProject(r.Context(), organizationID, projectId, claims.Subject, sess.Phase)
		if err != nil && !errors.Is(err, db.ErrNotFound) {
			sendJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Database error listing sessions: %v", err)})
			return
		}
	}

	if len(sessions) > 0 && !createNew {
		connectedSession = sessions[0]
		statusCode = http.StatusOK
	} else {
		sess.UserId = claims.Subject
		sess.ProjectId = projectId
		sess.CreatedAt = time.Now()
		sess.LastActiveAt = time.Now()
		sess.Status = "INITIALIZING"

		prefix := project.GetSpecBranchPrefix()
		cfg := core.NewConfig()
		clonePath := filepath.Join(cfg.WorkspaceMountPath, organizationID, claims.Subject, projectId)
		sess.SpecBranchName = workspace.ResolveUserSpecBranchName(r.Context(), prefix, claims.Subject, clonePath)

		if err := h.SessionStore.CreateSession(r.Context(), organizationID, &sess); err != nil {
			sendJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to create session: %v", err)})
			return
		}
		connectedSession = sess
		statusCode = http.StatusCreated
	}

	isNewSession := (statusCode == http.StatusCreated)

	cfg := core.NewConfig()
	bgCtx := r.Context()
	if claims != nil {
		bgCtx = context.WithValue(bgCtx, core.UserClaimsKey, claims)
	}
	sessCtx, cancel := context.WithTimeout(bgCtx, cfg.AgentHTTPTimeout)
	defer cancel()

	errInit := h.initializeAgentSession(sessCtx, organizationID, projectId, connectedSession.ID, connectedSession.Phase, claims.Subject)
	if errInit != nil {
		slog.Error("Failed to initialize session", "sessionId", connectedSession.ID, "error", errInit)
		if isNewSession {
			slog.Info("Rolling back failed new session creation", "sessionId", connectedSession.ID)
			_ = h.SessionStore.DeleteSession(bgCtx, organizationID, connectedSession.ID)
		} else {
			_ = h.SessionStore.UpdateSessionStatus(bgCtx, organizationID, connectedSession.ID, "FAILED")
		}
		agent.ProcessAgentError(bgCtx, "Agent Initialization Failed", "The connection to the AI agent could not be established.", errInit.Error(), projectId, organizationID, claims.Subject, connectedSession.Phase, connectedSession.ID)
		sendJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": fmt.Sprintf("Failed to initialize agent session: %v", errInit),
			"code":  "SESSION_INIT_FAILED",
		})
		return
	}

	_ = h.SessionStore.UpdateSessionStatus(bgCtx, organizationID, connectedSession.ID, "ACTIVE")
	connectedSession.Status = "ACTIVE"

	if h.Store != nil {
		systemText := fmt.Sprintf("Phase %s session started", connectedSession.Phase)
		msg := &db.Message{
			Sender:         "System",
			Text:           systemText,
			IsUser:         false,
			Phase:          connectedSession.Phase,
			SessionId:      "project_global_" + projectId,
			Timestamp:      time.Now(),
			IsTransient:    true,
			ExpireAt:       time.Now().Add(24 * time.Hour),
			IsNotification: true,
			NotificationDetails: &db.NotificationDetails{
				Name: "session-connect",
				Params: map[string]interface{}{
					"sessionId": connectedSession.ID,
					"phase":     connectedSession.Phase,
				},
			},
		}
		if _, err := h.Store.CreateMessage(bgCtx, organizationID, projectId, claims.Subject, msg); err != nil {
			slog.Error("Failed to broadcast session connection", "error", err)
		}
	}

	sendJSON(w, statusCode, SessionResponse{
		Session: connectedSession,
	})
}

func (h *SessionHandler) initializeAgentSession(ctx context.Context, orgID, projID, sessID, phase, userId string) error {
	appName := "APF Help Agent"
	switch phase {
	case "ONBOARDING":
		appName = "APF Overwatch Agent"
	case "ANALYSIS":
		appName = "APF Analysis Agent"
	case "PLANNING":
		appName = "APF Planning Agent"
	case "SOLUTIONING":
		appName = "APF Solutioning Agent"
	case "IMPLEMENTATION":
		appName = "APF Implementation Agent"
	}

	cfg := core.NewConfig()

	// Ensure workspace directory exists
	clonePath := filepath.Join(cfg.WorkspaceMountPath, orgID, userId, projID)
	if err := os.MkdirAll(clonePath, 0755); err != nil {
		return fmt.Errorf("failed to create workspace directory: %w", err)
	}

	project, err := store.GetProject(ctx, orgID, projID)
	if err != nil {
		return fmt.Errorf("failed to get project for git init: %w", err)
	}

	repoURL := project.GetRepoUrl()
	var accessToken string
	var errToken error
	if repoURL != "" {
		accessToken, errToken = integrations.ResolveGitHubToken(ctx, userId, projID)
		if errToken != nil {
			return fmt.Errorf("failed to resolve github token for git init: %w", errToken)
		}
	}

	baseBranch, specBranch, errResolve := workspace.ResolveSpecBranchNoRequest(ctx, orgID, projID, userId, phase)
	if errResolve != nil {
		return fmt.Errorf("failed to resolve spec branch for git init: %w", errResolve)
	}
	if baseBranch == "" || baseBranch == ".invalid" || strings.HasPrefix(baseBranch, ".") {
		baseBranch = "main"
	}
	if specBranch == "" || specBranch == ".invalid" || strings.HasPrefix(specBranch, ".") {
		specBranch = baseBranch
	}

	gitRepoCfg := &workspace.GitRepoConfig{
		RepoURL:         repoURL,
		RepoAccessToken: accessToken,
		BaseBranch:      baseBranch,
		SpecBranch:      specBranch,
		UserId:          userId,
		ProjectId:       projID,
		SpecLocation:    project.GetSpecLocation(),
	}

	errInit := workspace.GetGitClient().EnsureRepoInitialized(ctx, orgID, gitRepoCfg)
	if errInit != nil {
		return fmt.Errorf("failed to initialize repository: %w", errInit)
	}

	proxy := agent.NewAgentProxy(cfg.AgentBaseURL, cfg)

	sessCtx, cancel := context.WithTimeout(ctx, cfg.AgentHTTPTimeout)
	defer cancel()
	if orgID != "" {
		sessCtx = core.WithOrganizationID(sessCtx, orgID)
	}

	exists, err := proxy.GetSession(sessCtx, appName, sessID, userId)
	if err != nil {
		slog.Error("Failed to check if agent session exists", "error", err)
		return fmt.Errorf("failed to check if agent session exists: %w", err)
	} else if !exists {
		if err := proxy.CreateSession(sessCtx, appName, sessID, projID, userId, orgID); err != nil {
			slog.Error("Failed to initialize agent session dynamically", "error", err)
			return fmt.Errorf("failed to create agent session: %w", err)
		}
		_, err := proxy.RunAgent(sessCtx, appName, sessID, projID, userId, "init")
		if err != nil {
			slog.Error("Failed to invoke agent", "error", err)
			agent.ProcessAgentError(sessCtx, "Agent Initialization failed", "Failed to invoke initialization on the AI agent.", err.Error(), projID, orgID, userId, phase, sessID)
			return fmt.Errorf("failed to invoke agent initialization: %w", err)
		}

		if h.Store != nil {
			msg := &db.Message{
				Sender:         "System",
				Text:           fmt.Sprintf("%s initialized and ready", appName),
				IsUser:         false,
				Phase:          phase,
				SessionId:      sessID,
				Timestamp:      time.Now(),
				IsTransient:    true,
				ExpireAt:       time.Now().Add(24 * time.Hour),
				IsNotification: true,
				NotificationDetails: &db.NotificationDetails{
					Name: "agent-init",
					Params: map[string]interface{}{
						"agentName": appName,
						"phase":     phase,
					},
				},
			}
			if _, err := h.Store.CreateMessage(sessCtx, orgID, projID, "System", msg); err != nil {
				slog.Error("Failed to broadcast agent initialization", "error", err)
			}
		}
	}
	return nil
}

// TODO: Not Currently used. Will be used when we start supporting a use case where we need to list multiple sessions
func (h *SessionHandler) listSessionsHandler(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("id")
	claims := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	phase := r.URL.Query().Get("phase")

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		sendJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized: Organization ID missing"})
		return
	}

	sessions, err := h.SessionStore.ListSessionsForProject(r.Context(), organizationID, projectId, claims.Subject, phase)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to list sessions: %v", err)})
		return
	}

	sendJSON(w, http.StatusOK, sessions)
}

// deleteSessionHandler deletes a previous chat session.
func (h *SessionHandler) deleteSessionHandler(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("id")
	sessionId := r.PathValue("sessionId")
	claims := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		sendJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized: Organization ID missing"})
		return
	}

	// Fetch sessions to find details (like Phase) before deleting for broadcast info
	sessions, err := h.SessionStore.ListSessionsForProject(r.Context(), organizationID, projectId, claims.Subject, "")
	var targetSess *db.Session
	if err == nil {
		for _, s := range sessions {
			if s.ID == sessionId {
				targetSess = &s
				break
			}
		}
	}

	if err := h.SessionStore.DeleteSession(r.Context(), organizationID, sessionId); err != nil {
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to delete session: %v", err)})
		return
	}

	// Broadcast session delete event
	if h.Store != nil && targetSess != nil {
		msg := &db.Message{
			Sender:         claims.Subject,
			Text:           fmt.Sprintf("Session for phase %s deleted", targetSess.Phase),
			IsUser:         false,
			Phase:          targetSess.Phase,
			SessionId:      "project_global_" + projectId,
			Timestamp:      time.Now(),
			IsTransient:    true,
			ExpireAt:       time.Now().Add(24 * time.Hour),
			IsNotification: true,
			NotificationDetails: &db.NotificationDetails{
				Name: "session-delete",
				Params: map[string]interface{}{
					"sessionId": sessionId,
					"phase":     targetSess.Phase,
				},
			},
		}
		if _, err := h.Store.CreateMessage(r.Context(), organizationID, projectId, claims.Subject, msg); err != nil {
			slog.Error("Failed to broadcast session deletion", "error", err)
		}
	}

	sendJSON(w, http.StatusOK, map[string]string{"message": "Session deleted successfully"})
}

type UpdateSessionRequest struct {
	Title  string `json:"title,omitempty"`
	Status string `json:"status,omitempty"`
}

// updateSessionHandler updates session properties like title or status.
func (h *SessionHandler) updateSessionHandler(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("id")
	sessionId := r.PathValue("sessionId")
	claims := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		sendJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized: Organization ID missing"})
		return
	}

	var req UpdateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
		return
	}

	if req.Title != "" {
		if err := h.SessionStore.UpdateSessionTitle(r.Context(), organizationID, sessionId, req.Title); err != nil {
			sendJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to update session title: %v", err)})
			return
		}
	}
	if req.Status != "" {
		if err := h.SessionStore.UpdateSessionStatus(r.Context(), organizationID, sessionId, req.Status); err != nil {
			sendJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to update session status: %v", err)})
			return
		}
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"message":   "Session updated successfully",
		"sessionId": sessionId,
		"projectId": projectId,
		"userId":    claims.Subject,
	})
}

const maxUploadFileSize = 10 * 1024 * 1024 // 10MB limit

// uploadSessionArtifactHandler handles multipart file uploads for chat sessions.
func (h *SessionHandler) uploadSessionArtifactHandler(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("id")
	sessionId := r.PathValue("sessionId")

	if projectId == "" || sessionId == "" {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Project ID and Session ID are required"})
		return
	}

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		sendJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized: Organization ID missing"})
		return
	}

	// Immediate check on ContentLength if provided
	if r.ContentLength > maxUploadFileSize {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "File size exceeds 10MB limit"})
		return
	}

	// Limit request body reading to maxUploadFileSize + 1KB buffer for multipart headers
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadFileSize+1024)

	if err := r.ParseMultipartForm(maxUploadFileSize); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) || strings.Contains(err.Error(), "request body too large") || strings.Contains(err.Error(), "http: request body too large") {
			sendJSON(w, http.StatusBadRequest, map[string]string{"error": "File size exceeds 10MB limit"})
			return
		}
		if errors.Is(err, http.ErrNotMultipart) {
			sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid multipart request"})
			return
		}
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Failed to parse multipart form: %v", err)})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "File is required"})
		return
	}
	defer file.Close()

	if header.Size > maxUploadFileSize {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "File size exceeds 10MB limit"})
		return
	}
	if header.Size == 0 {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "File is empty"})
		return
	}

	// Strict filename sanitization using filepath.Base
	filename := filepath.Base(filepath.Clean(header.Filename))
	if idx := strings.LastIndex(filename, "\\"); idx != -1 {
		filename = filename[idx+1:]
	}
	filename = strings.TrimSpace(filename)
	if filename == "" || filename == "." || filename == ".." {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid filename"})
		return
	}

	// Detect MIME type from extension or sniff from the first 512 bytes
	mimeType := mime.TypeByExtension(filepath.Ext(filename))
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Failed to read file"})
		return
	}
	sniffedData := buf[:n]
	if mimeType == "" {
		mimeType = http.DetectContentType(sniffedData)
	}

	if headerType := header.Header.Get("Content-Type"); headerType != "" && (mimeType == "application/octet-stream" || mimeType == "text/plain; charset=utf-8") {
		mimeType = headerType
	}

	combinedReader := io.MultiReader(bytes.NewReader(sniffedData), file)
	limitReader := io.LimitReader(combinedReader, maxUploadFileSize+1)

	// Resolve ArtifactService (use injected service if provided, else determine based on config)
	artService := h.ArtifactService
	if artService == nil {
		cfg := h.AppConfig
		if cfg == nil {
			cfg = core.NewConfig()
		}
		isCloud := strings.EqualFold(cfg.ArtifactServiceType, "GCS") || (cfg.Environment != "local" && cfg.Environment != "test" && cfg.Environment != "")
		if isCloud {
			if cfg.GcsArtifactBucket == "" {
				slog.Error("GCS_ARTIFACT_BUCKET is required in cloud environments (staging/production)")
				sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Cloud artifact storage bucket not configured"})
				return
			}
			gcsClient, err := storage.NewClient(r.Context())
			if err != nil {
				slog.Error("Failed to initialize GCS client", "error", err)
				sendJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to initialize storage: %v", err)})
				return
			}
			defer gcsClient.Close()
			artService = core.NewGcsArtifactService(gcsClient, cfg.GcsArtifactBucket)
		} else {
			// Local development mode only: use local filesystem artifact storage
			localPath := cfg.GetLocalArtifactPath()
			artService = core.NewFileSystemArtifactService(localPath)
		}
	}

	storageType, storagePath, bytesWritten, err := artService.SaveSessionArtifact(r.Context(), projectId, sessionId, filename, limitReader)
	if err != nil {
		slog.Error("Failed to save session artifact", "projectId", projectId, "sessionId", sessionId, "filename", filename, "error", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to save artifact: %v", err)})
		return
	}

	if bytesWritten > maxUploadFileSize {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "File size exceeds 10MB limit"})
		return
	}

	artStore := h.ArtifactStore
	if artStore == nil {
		artStore = workspace.ArtifactStore
	}

	art := &db.Artifact{
		ID:          uuid.New().String(),
		ProjectId:   projectId,
		SessionId:   sessionId,
		Name:        filename,
		MimeType:    mimeType,
		Size:        bytesWritten,
		StorageType: storageType,
		StoragePath: storagePath,
		CreatedAt:   time.Now().UTC(),
	}

	if artStore != nil {
		if err := artStore.CreateArtifact(r.Context(), organizationID, art); err != nil {
			slog.Error("Failed to persist session artifact metadata in Firestore", "error", err)
			sendJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to persist artifact metadata: %v", err)})
			return
		}
	}

	sendJSON(w, http.StatusCreated, art)
}

