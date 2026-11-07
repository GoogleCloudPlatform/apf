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

package agent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

const (
	transientExpiration = 72 * time.Hour

	SenderMiddleware = "Middleware"
	SenderAgent      = "Agent"
	SenderUser       = "user"

	NotificationError       = "error"
	NotificationThought     = "thought"
	NotificationToolResult  = "tool-result"
	NotificationStateUpdate = "state-update"
	NotificationTransfer    = "agent-transfer"
	NotificationEscalate    = "escalate"
	NotificationFileWrite   = "file-write"

	ActionStateDelta      = "stateDelta"
	ActionStateDeltaSnake = "state_delta"
	ActionArtifactDelta   = "artifactDelta"
	ActionTransfer        = "transfer_to_agent"
	ActionEscalate        = "escalate"
	PrefixFileWriteNotif  = "file-write-notification:"

	KeyText             = "text"
	KeyThought          = "thought"
	KeyThoughtChunks    = "thought_chunks"
	KeyThinking         = "thinking"
	KeyFunctionCall     = "function_call"
	KeyFunctionResponse = "functionResponse"
	KeyName             = "name"
	KeyArgs             = "args"
	KeyResponse         = "response"

	DefaultFallbackReply     = "Empty response / Connection failed"
	DefaultExecutingToolText = "Executing tool call"
)

var (
	recentToolCallArgsMu         sync.Mutex
	recentToolCallArgs           = make(map[string]string)
	recentToolCallNames          = make(map[string]string)
	recentPhaseSwitchSuggestions = make(map[string]*db.PhaseSwitchSuggestion)

	toolArgCandidateKeys = []string{
		"path", "file", "filePath", "query", "command",
		"targetFile", "TargetFile", "AbsolutePath", "SearchDirectory",
	}
	toolRespCandidateKeys = []string{
		"path", "filePath", "file", "filename", "targetFile", "AbsolutePath",
	}
)

// Store is the global message store instance.
var Store db.MessageStore
var config *core.Configuration

// OnFileWriteHook is an optional hook triggered when a file write notification is handled.
var OnFileWriteHook func(ctx context.Context, projectID string, organizationID string, phase string, filePath string)

// SetArtifactStore sets the global artifact store instance (no-op retained for compatibility).
func SetArtifactStore(s db.ArtifactStore) {}

// SetMessageStore sets the global message store instance.
func SetMessageStore(s db.MessageStore) {
	Store = s
}

// SetConfig sets the global configuration instance.
func SetConfig(c *core.Configuration) {
	config = c
}

type NodeInfo = db.NodeInfo
type ADKEvent = db.ADKEvent
type EventContent = db.EventContent

// EventContext holds metadata about the active agent session and request context.
type EventContext struct {
	ProjectID      string
	OrganizationID string
	UserID         string
	Phase          string
	SessionID      string
}

func ProcessAgentError(ctx context.Context, title, description, details string, projectID string, organizationID string, userID string, phase string, sessionID string) {
	eventCtx := EventContext{
		ProjectID:      projectID,
		OrganizationID: organizationID,
		UserID:         userID,
		Phase:          phase,
		SessionID:      sessionID,
	}
	ProcessAgentErrorCtx(ctx, title, description, details, eventCtx)
}

// ProcessAgentErrorCtx records an agent error message using an EventContext.
func ProcessAgentErrorCtx(ctx context.Context, title, description, details string, eventCtx EventContext) {
	slog.Info("Processing agent error", "projectId", eventCtx.ProjectID)

	lowerDetails := strings.ToLower(details)
	if strings.Contains(details, "GEMINI_API_KEY_REQUIRED") || strings.Contains(lowerDetails, "gemini_api_key") || strings.Contains(lowerDetails, "api_key_invalid") || strings.Contains(lowerDetails, "api key not valid") {
		title = "Gemini API Key Missing or Invalid"
		description = "The AI agent could not execute because your Gemini API key is unconfigured or invalid. Please configure a valid Gemini API key in Project Settings or User Settings."
	}

	agentMsg := db.Message{
		Sender:         SenderMiddleware,
		Text:           title,
		IsUser:         false,
		Phase:          eventCtx.Phase,
		SessionId:      eventCtx.SessionID,
		Timestamp:      time.Now(),
		IsTransient:    true,
		ExpireAt:       time.Now().Add(transientExpiration),
		IsNotification: true,
		NotificationDetails: &db.NotificationDetails{
			Name: NotificationError,
			Params: map[string]interface{}{
				"title":       title,
				"description": description,
				"details":     details,
			},
		},
	}

	if Store != nil {
		// Use context.WithoutCancel to avoid request cancellation while preserving organization ID claims
		detachedCtx := context.WithoutCancel(ctx)
		if _, err := Store.CreateMessage(detachedCtx, eventCtx.OrganizationID, eventCtx.ProjectID, eventCtx.UserID, &agentMsg); err != nil {
			slog.Warn(fmt.Sprintf("Failed to save agent message to Firestore: %v\n", err))
		}
	}
}

// processEvent handles a single ADK event and returns true if any pattern handler matched the event.
func processEvent(ctx context.Context, event ADKEvent, eventCtx EventContext, payload []byte) bool {
	slog.Info("API tier received event", "time", time.Now().Format("15:04:05.000"), "author", event.Author, "partial", event.Partial, "turnComplete", event.TurnComplete)
	return DefaultDispatcher.Dispatch(ctx, event, eventCtx)
}

// ProcessAgentEvent handles processing of a single real-time ADKEvent.
func ProcessAgentEvent(ctx context.Context, event ADKEvent, projectID string, organizationID string, userID string, phase string, sessionID string) {
	eventCtx := EventContext{
		ProjectID:      projectID,
		OrganizationID: organizationID,
		UserID:         userID,
		Phase:          phase,
		SessionID:      sessionID,
	}
	ProcessAgentEventCtx(ctx, event, eventCtx)
}

// ProcessAgentEventCtx handles processing of a single real-time ADKEvent using an EventContext.
func ProcessAgentEventCtx(ctx context.Context, event ADKEvent, eventCtx EventContext) {
	processEvent(ctx, event, eventCtx, nil)
}
