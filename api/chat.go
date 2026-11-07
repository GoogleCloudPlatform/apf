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
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"google.com/apf/api/agent"
	core "google.com/apf/api/core"
	"google.com/apf/api/db"
	"google.com/apf/api/workspace"
)

type UserMessageResponse struct {
	Message db.Message `json:"message"`
}

// Store is the global instance of db.MessageStore for backwards compatibility.
var Store db.MessageStore

// ChatHandler encapsulates dependencies for chat endpoints.
type ChatHandler struct {
	Store            db.MessageStore
	ArtifactService  core.ArtifactService
	ArtifactStore    db.ArtifactStore
	AppConfig        *core.Configuration
	activeExecutions map[string]context.CancelFunc
	mu               sync.Mutex
}

// NewChatHandler creates a new ChatHandler instance.
func NewChatHandler(store db.MessageStore) *ChatHandler {
	return &ChatHandler{
		Store:            store,
		activeExecutions: make(map[string]context.CancelFunc),
	}
}

// WithArtifactService attaches an ArtifactService to ChatHandler.
func (h *ChatHandler) WithArtifactService(svc core.ArtifactService) *ChatHandler {
	h.ArtifactService = svc
	return h
}

// WithArtifactStore attaches an ArtifactStore to ChatHandler.
func (h *ChatHandler) WithArtifactStore(store db.ArtifactStore) *ChatHandler {
	h.ArtifactStore = store
	return h
}

// WithAppConfig attaches application configuration to ChatHandler.
func (h *ChatHandler) WithAppConfig(cfg *core.Configuration) *ChatHandler {
	h.AppConfig = cfg
	return h
}

// ListMessagesHandler lists all messages for a workspace.
func (h *ChatHandler) ListMessagesHandler(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("id")
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		http.Error(w, "Unauthorized context", http.StatusUnauthorized)
		return
	}
	sessionId := r.URL.Query().Get("sessionId")

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		http.Error(w, "Unauthorized: Organization ID missing", http.StatusUnauthorized)
		return
	}

	messages, err := h.Store.ListMessages(r.Context(), organizationID, projectId, claims.Subject, sessionId, false)
	if err != nil {
		slog.Error("Error listing messages", "error", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"message": fmt.Sprintf("Failed to list messages: %v", err), "status": "ERROR"})
		return
	}

	sendJSON(w, http.StatusOK, messages)
}

// UserMessageHandler adds a new message and auto-generates a response.
func (h *ChatHandler) UserMessageHandler(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("id")
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		http.Error(w, "Unauthorized context", http.StatusUnauthorized)
		return
	}

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		http.Error(w, "Unauthorized: Organization ID missing", http.StatusUnauthorized)
		return
	}

	var msg db.Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		return
	}

	createdMsg, err := h.Store.CreateMessage(r.Context(), organizationID, projectId, claims.Subject, &msg)
	if err != nil {
		slog.Error("Failed to save user message to Firestore", "error", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to save message"})
		return
	}

	if createdMsg.IsUser {
		claims, _ := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)

		go func(userMsg db.Message, projId string, orgId string, userId string, claims *core.UserClaims) {
			appName := "APF Help Agent"
			switch userMsg.Phase {
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

			sessId := userMsg.SessionId
			cfg := h.AppConfig
			if cfg == nil {
				cfg = core.NewConfig()
			}
			proxy := agent.NewAgentProxy(cfg.AgentBaseURL, cfg)

			// Use independent background context with timeout so background goroutine isn't cancelled by HTTP handler return,
			// but ensure it is properly cancellable by timeout.
			bgCtx := context.Background()
			if claims != nil {
				bgCtx = context.WithValue(bgCtx, core.UserClaimsKey, claims)
			}
			if orgId != "" {
				bgCtx = core.WithOrganizationID(bgCtx, orgId)
			}
			ctx, cancel := context.WithTimeout(bgCtx, cfg.AgentHTTPTimeout)
			defer cancel()
			h.RegisterExecution(projId, sessId, cancel)
			defer h.UnregisterExecution(projId, sessId)

			// Verify if the session exists on the AI tier (it may have been lost due to AI tier restarts,
			// or failed to initialize previously). If not, dynamically create and initialize it.
			exists, err := proxy.GetSession(ctx, appName, sessId, userId)
			if err != nil {
				slog.Error("Failed to check if agent session exists in UserMessageHandler", "error", err)
				agent.ProcessAgentError(ctx, "Failed to stream response", "The API backend could not verify the AI agent session status.", err.Error(), projId, orgId, userId, userMsg.Phase, sessId)
				return
			} else if !exists {
				slog.Info("Agent session not found on AI tier. Dynamically initializing session...", "sessionId", sessId)
				if err := proxy.CreateSession(ctx, appName, sessId, projId, userId, orgId); err != nil {
					slog.Error("Failed to initialize agent session dynamically in UserMessageHandler", "error", err)
					agent.ProcessAgentError(ctx, "Failed to stream response", "The API backend could not dynamically create the AI agent session.", err.Error(), projId, orgId, userId, userMsg.Phase, sessId)
					return
				}
				_, err := proxy.RunAgent(ctx, appName, sessId, projId, userId, "init")
				if err != nil {
					slog.Error("Failed to run agent initialization dynamically in UserMessageHandler", "error", err)
					agent.ProcessAgentError(ctx, "Failed to stream response", "The API backend could not dynamically initialize the AI agent session.", err.Error(), projId, orgId, userId, userMsg.Phase, sessId)
					return
				}
				// Update session status to ACTIVE in database
				if sessionStore != nil {
					if err := sessionStore.UpdateSessionStatus(bgCtx, orgId, sessId, "ACTIVE"); err != nil {
						slog.Warn("Failed to update session status to ACTIVE", "sessionId", sessId, "error", err)
					}
				}
			}

			// Resolve ArtifactService
			artService := h.ArtifactService
			if artService == nil {
				isCloud := strings.EqualFold(cfg.ArtifactServiceType, "GCS") || (cfg.Environment != "local" && cfg.Environment != "test" && cfg.Environment != "")
				if isCloud {
					if cfg.GcsArtifactBucket != "" {
						gcsClient, err := storage.NewClient(bgCtx)
						if err != nil {
							slog.Error("Failed to initialize GCS client in UserMessageHandler", "error", err)
						} else {
							defer gcsClient.Close()
							artService = core.NewGcsArtifactService(gcsClient, cfg.GcsArtifactBucket)
						}
					} else {
						slog.Error("GCS_ARTIFACT_BUCKET is required in cloud environments (staging/production)")
					}
				} else {
					// Local development mode only: use local filesystem artifact storage
					localPath := cfg.GetLocalArtifactPath()
					artService = core.NewFileSystemArtifactService(localPath)
				}
			}

			promptToRun := userMsg.Prompt
			if promptToRun == "" {
				promptToRun = userMsg.Text
			}

			artifacts := userMsg.Artifacts
			existingIDMap := make(map[string]bool)
			for _, a := range artifacts {
				if a.ID != "" {
					existingIDMap[a.ID] = true
				}
			}
			var missingIDs []string
			for _, id := range userMsg.ArtifactIDs {
				if !existingIDMap[id] {
					missingIDs = append(missingIDs, id)
				}
			}
			if len(missingIDs) > 0 {
				artStore := h.ArtifactStore
				if artStore == nil {
					artStore = workspace.ArtifactStore
				}
				if artStore != nil {
					allArts, errArts := artStore.ListArtifactsForProject(ctx, orgId, projId)
					if errArts != nil {
						slog.Warn("Failed to list artifacts for project in UserMessageHandler", "error", errArts, "projectId", projId)
					} else {
						idMap := make(map[string]bool)
						for _, id := range missingIDs {
							idMap[id] = true
						}
						for _, a := range allArts {
							if idMap[a.ID] {
								artifacts = append(artifacts, a)
							}
						}
					}
				}
			}

			promptParts, err := agent.BuildPromptParts(ctx, promptToRun, artifacts, artService)
			if err != nil {
				slog.Error("Failed to build prompt parts with artifacts in UserMessageHandler", "error", err)
				promptParts = []agent.ProxyMessagePart{{Text: promptToRun}}
			}

			bodyStream, err := proxy.RunAgentStreamingWithParts(ctx, appName, sessId, projId, userId, promptParts)
			if err != nil {
				if errors.Is(ctx.Err(), context.Canceled) || strings.Contains(err.Error(), "context canceled") {
					slog.Info("Agent streaming connection cancelled before start", "sessionId", sessId)
					if h.Store != nil {
						cancelMsg := &db.Message{
							Sender:    appName,
							Text:      "Message cancelled",
							IsUser:    false,
							Phase:     userMsg.Phase,
							SessionId: sessId,
							Timestamp: time.Now(),
						}
						_, _ = h.Store.CreateMessage(bgCtx, orgId, projId, userId, cancelMsg)
					}
					return
				}
				slog.Error("Failed to invoke agent streaming in UserMessageHandler", "error", err)
				agent.ProcessAgentError(ctx, "Failed to stream response", "The API backend could not initiate the connection to the AI agent.", err.Error(), projId, orgId, userId, userMsg.Phase, sessId)
				return
			}
			defer bodyStream.Close()

			turnCompleted := false
			scanner := bufio.NewScanner(bodyStream)
			const maxCapacity = 32 * 1024 * 1024 // 32 MB max line buffer for long-running agent streaming chunks
			buf := make([]byte, 64*1024)
			scanner.Buffer(buf, maxCapacity)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "data: ") {
					dataStr := strings.TrimPrefix(line, "data: ")
					var event agent.ADKEvent
					if err := json.Unmarshal([]byte(dataStr), &event); err != nil {
						slog.Error("Failed to unmarshal streaming ADKEvent", "error", err, "data", dataStr)
						agent.ProcessAgentError(ctx, "Malformed stream payload received", "The AI server sent a malformed streaming block that could not be parsed.", fmt.Sprintf("Raw details: %v", err), projId, orgId, userId, userMsg.Phase, sessId)
						continue
					}
					if event.TurnComplete || (!event.Partial && event.Content != nil && len(event.Content.Parts) > 0) {
						turnCompleted = true
					}
					slog.Info("API tier received streaming chunk", "time", time.Now().Format("15:04:05.000"), "author", event.Author, "partial", event.Partial, "turnComplete", event.TurnComplete)
					agent.ProcessAgentEvent(ctx, event, projId, orgId, userId, userMsg.Phase, sessId)
				}
			}

			if !turnCompleted {
				isTimeout := errors.Is(ctx.Err(), context.DeadlineExceeded) || (scanner.Err() != nil && strings.Contains(scanner.Err().Error(), "context deadline exceeded"))
				isCanceled := errors.Is(ctx.Err(), context.Canceled) || (scanner.Err() != nil && strings.Contains(scanner.Err().Error(), "context canceled"))
				if isCanceled || isTimeout {
					if isTimeout {
						slog.Warn("Agent stream generation timed out", "sessionId", sessId)
						agent.ProcessAgentError(ctx, "AI agent request timed out", "The AI agent request exceeded the maximum allowed execution time.", "Timeout limit reached", projId, orgId, userId, userMsg.Phase, sessId)
					} else {
						slog.Info("Agent stream generation cancelled by user", "sessionId", sessId)
						if h.Store != nil {
							cancelMsg := &db.Message{
								Sender:    appName,
								Text:      "Message cancelled",
								IsUser:    false,
								Phase:     userMsg.Phase,
								SessionId: sessId,
								Timestamp: time.Now(),
							}
							_, _ = h.Store.CreateMessage(bgCtx, orgId, projId, userId, cancelMsg)
						}
					}
				} else if scanner.Err() != nil {
					slog.Error("Error reading ADK agent SSE stream", "error", scanner.Err())
					agent.ProcessAgentError(ctx, "AI agent stream connection interrupted", "The connection to the AI agent stream was dropped mid-generation.", scanner.Err().Error(), projId, orgId, userId, userMsg.Phase, sessId)
				} else {
					slog.Error("Agent SSE stream closed without turn completion", "sessionId", sessId)
					agent.ProcessAgentError(ctx, "AI agent execution failed", "The AI agent stream ended without completing the response.", "No completion event received from AI tier", projId, orgId, userId, userMsg.Phase, sessId)
				}
			}

		}(*createdMsg, projectId, organizationID, claims.Subject, claims)
	}

	sendJSON(w, http.StatusCreated, UserMessageResponse{
		Message: msg,
	})
}

// StreamMessageHandler handles real-time message streaming via SSE.
func (h *ChatHandler) StreamMessageHandler(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("id")
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok {
		http.Error(w, "Unauthorized context", http.StatusUnauthorized)
		return
	}
	userId := claims.Subject

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		http.Error(w, "Unauthorized: Organization ID missing", http.StatusUnauthorized)
		return
	}

	sessionId := r.URL.Query().Get("sessionId")
	sinceStr := r.URL.Query().Get("since")
	var since time.Time
	if sinceStr != "" {
		var err error
		since, err = time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			http.Error(w, "Invalid since parameter", http.StatusBadRequest)
			return
		}
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no")

	connTime := time.Now()

	// Write StatusOK and flush headers immediately so client gets response status and doesn't timeout.
	w.WriteHeader(http.StatusOK)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	if sessionId == "" {
		sessionId = projectId
	}
	msgChan, errChan := h.Store.StreamMessages(r.Context(), organizationID, projectId, userId, sessionId, since)

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-msgChan:
			if !ok {
				return
			}
			if msg.Timestamp.Before(connTime) {
				msg.IsHistorical = true
			}
			jsonBytes, err := json.Marshal(msg)
			if err != nil {
				slog.Error("Failed to marshal message in stream", "error", err)
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", string(jsonBytes))
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		case err := <-errChan:
			slog.Error("Firestore snapshot error", "error", err)
			return
		case <-ticker.C:
			// Send heartbeat
			fmt.Fprintf(w, ": keepalive\n\n")
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		case <-r.Context().Done():
			return
		}
	}
}

func (h *ChatHandler) RegisterExecution(projectId, sessionId string, cancel context.CancelFunc) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.activeExecutions == nil {
		h.activeExecutions = make(map[string]context.CancelFunc)
	}
	key := projectId + ":" + sessionId
	h.activeExecutions[key] = cancel
}

func (h *ChatHandler) UnregisterExecution(projectId, sessionId string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.activeExecutions == nil {
		return
	}
	key := projectId + ":" + sessionId
	delete(h.activeExecutions, key)
}

func (h *ChatHandler) CancelMessageHandler(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("id")
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		http.Error(w, "Unauthorized context", http.StatusUnauthorized)
		return
	}
	sessionId := r.URL.Query().Get("sessionId")
	if sessionId == "" {
		sessionId = projectId
	}

	h.mu.Lock()
	key := projectId + ":" + sessionId
	cancel, exists := h.activeExecutions[key]
	h.mu.Unlock()

	if exists && cancel != nil {
		cancel()
		h.UnregisterExecution(projectId, sessionId)
		slog.Info("Cancelled active execution", "projectId", projectId, "sessionId", sessionId, "user", claims.Subject)
	}

	sendJSON(w, http.StatusOK, map[string]string{"message": "Execution cancelled", "status": "CANCELLED"})
}
