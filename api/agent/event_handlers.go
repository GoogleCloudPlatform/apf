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
	"log/slog"
	"strings"
	"time"

	"google.com/apf/api/db"
)

func handleUserInput(ctx context.Context, event ADKEvent, eventCtx EventContext) {
	slog.Info("Shell: Handling User Input")
}

func handleAgentFinalTextResponse(ctx context.Context, event ADKEvent, eventCtx EventContext) {
	slog.Info("Handling Agent Final Text Response")

	var text string
	if event.Content != nil && len(event.Content.Parts) > 0 {
		text, _ = event.Content.Parts[0][KeyText].(string)
	}

	sender := event.Author
	if event.NodeInfo != nil && event.NodeInfo.Path != "" {
		if nodeName := extractNodeName(event.NodeInfo.Path); nodeName != "" {
			sender = nodeName
		}
	}

	var phaseSwitchSuggestion *db.PhaseSwitchSuggestion
	if eventCtx.SessionID != "" {
		recentToolCallArgsMu.Lock()
		if pss, exists := recentPhaseSwitchSuggestions[eventCtx.SessionID]; exists {
			phaseSwitchSuggestion = pss
			delete(recentPhaseSwitchSuggestions, eventCtx.SessionID)
		}
		recentToolCallArgsMu.Unlock()
	}

	msg := &db.Message{
		Sender:                sender,
		Text:                  text,
		IsUser:                false,
		Phase:                 eventCtx.Phase,
		SessionId:             eventCtx.SessionID,
		Timestamp:             time.Now(),
		PhaseSwitchSuggestion: phaseSwitchSuggestion,
	}

	if _, err := Store.CreateMessage(ctx, eventCtx.OrganizationID, eventCtx.ProjectID, eventCtx.UserID, msg); err != nil {
		slog.Error("Failed to create agent message", "error", err)
	}
}

func handleAgentStreamingTextResponse(ctx context.Context, event ADKEvent, eventCtx EventContext) {
	slog.Info("Shell: Handling Agent Streaming Text Response")

	var text string
	if event.Content != nil && len(event.Content.Parts) > 0 {
		text, _ = event.Content.Parts[0][KeyText].(string)
	}

	sender := event.Author
	if event.NodeInfo != nil && event.NodeInfo.Path != "" {
		if nodeName := extractNodeName(event.NodeInfo.Path); nodeName != "" {
			sender = nodeName
		}
	}

	msg := &db.Message{
		Sender:    sender,
		Text:      text,
		IsUser:    false,
		Phase:     eventCtx.Phase,
		SessionId: eventCtx.SessionID,
		Timestamp: time.Now(),
	}

	if _, err := Store.CreateMessage(ctx, eventCtx.OrganizationID, eventCtx.ProjectID, eventCtx.UserID, msg); err != nil {
		slog.Error("Failed to create agent message", "error", err)
	}
}

func handleAgentStreamingThoughtResponse(ctx context.Context, event ADKEvent, eventCtx EventContext) {
	slog.Info("Shell: Handling Agent Streaming Thought Response")

	var thought string
	if event.Content != nil && len(event.Content.Parts) > 0 {
		if t, ok := event.Content.Parts[0][KeyThought].(string); ok {
			thought = t
		} else if t, ok := event.Content.Parts[0][KeyThoughtChunks].(string); ok {
			thought = t
		} else if t, ok := event.Content.Parts[0][KeyThinking].(string); ok {
			thought = t
		}
	}

	msg := &db.Message{
		Sender:         event.Author,
		Text:           thought,
		IsUser:         false,
		Phase:          eventCtx.Phase,
		SessionId:      eventCtx.SessionID,
		Timestamp:      time.Now(),
		IsTransient:    true,
		ExpireAt:       time.Now().Add(transientExpiration),
		IsNotification: true,
		NotificationDetails: &db.NotificationDetails{
			Name:   NotificationThought,
			Params: map[string]interface{}{},
		},
	}

	if _, err := Store.CreateMessage(ctx, eventCtx.OrganizationID, eventCtx.ProjectID, eventCtx.UserID, msg); err != nil {
		slog.Error("Failed to create agent thought message", "error", err)
	}
}

func handleStateArtifactUpdateOnly(ctx context.Context, event ADKEvent, eventCtx EventContext) {
	slog.Info("Shell: Handling State/Artifact Update Only")
	msg := &db.Message{
		Sender:         SenderAgent,
		Text:           "State or artifact updated",
		IsUser:         false,
		Phase:          eventCtx.Phase,
		SessionId:      eventCtx.SessionID,
		Timestamp:      time.Now(),
		IsTransient:    true,
		ExpireAt:       time.Now().Add(transientExpiration),
		IsNotification: true,
		NotificationDetails: &db.NotificationDetails{
			Name:   NotificationStateUpdate,
			Params: map[string]interface{}{},
		},
	}
	if Store != nil {
		if _, err := Store.CreateMessage(ctx, eventCtx.OrganizationID, eventCtx.ProjectID, eventCtx.UserID, msg); err != nil {
			slog.Error("Failed to save state update notification to Firestore", "error", err, "projectId", eventCtx.ProjectID, "sessionId", eventCtx.SessionID)
		}
	}
}

func handleAgentTransferSignal(ctx context.Context, event ADKEvent, eventCtx EventContext) {
	slog.Info("Shell: Handling Agent Transfer Signal")
	msg := &db.Message{
		Sender:         SenderAgent,
		Text:           "Transferring agent session",
		IsUser:         false,
		Phase:          eventCtx.Phase,
		SessionId:      eventCtx.SessionID,
		Timestamp:      time.Now(),
		IsTransient:    true,
		ExpireAt:       time.Now().Add(transientExpiration),
		IsNotification: true,
		NotificationDetails: &db.NotificationDetails{
			Name:   NotificationTransfer,
			Params: map[string]interface{}{},
		},
	}
	if Store != nil {
		if _, err := Store.CreateMessage(ctx, eventCtx.OrganizationID, eventCtx.ProjectID, eventCtx.UserID, msg); err != nil {
			slog.Error("Failed to save transfer notification to Firestore", "error", err, "projectId", eventCtx.ProjectID, "sessionId", eventCtx.SessionID)
		}
	}
}

func handleLoopEscalationSignal(ctx context.Context, event ADKEvent, eventCtx EventContext) {
	slog.Info("Shell: Handling Loop Escalation Signal")
	msg := &db.Message{
		Sender:         SenderAgent,
		Text:           "Escalating task",
		IsUser:         false,
		Phase:          eventCtx.Phase,
		SessionId:      eventCtx.SessionID,
		Timestamp:      time.Now(),
		IsTransient:    true,
		ExpireAt:       time.Now().Add(transientExpiration),
		IsNotification: true,
		NotificationDetails: &db.NotificationDetails{
			Name:   NotificationEscalate,
			Params: map[string]interface{}{},
		},
	}
	if Store != nil {
		if _, err := Store.CreateMessage(ctx, eventCtx.OrganizationID, eventCtx.ProjectID, eventCtx.UserID, msg); err != nil {
			slog.Error("Failed to save loop escalation notification to Firestore", "error", err, "projectId", eventCtx.ProjectID, "sessionId", eventCtx.SessionID)
		}
	}
}

func handleFileWriteNotification(ctx context.Context, event ADKEvent, eventCtx EventContext) {
	slog.Info("Handling File Write Notification")

	var stateDelta map[string]interface{}
	if delta, ok := event.Actions[ActionStateDelta].(map[string]interface{}); ok {
		stateDelta = delta
	} else if delta, ok := event.Actions[ActionStateDeltaSnake].(map[string]interface{}); ok {
		stateDelta = delta
	}

	if stateDelta == nil {
		slog.Warn("StateDelta is empty in file-write-notification")
		return
	}

	seenFilePaths := make(map[string]bool)
	var filePaths []string

	// Check for unique prefix keys
	for k, v := range stateDelta {
		if strings.HasPrefix(k, PrefixFileWriteNotif) {
			if pathStr, ok := v.(string); ok && pathStr != "" {
				if !seenFilePaths[pathStr] {
					seenFilePaths[pathStr] = true
					filePaths = append(filePaths, pathStr)
				}
			}
		}
	}

	if len(filePaths) == 0 {
		slog.Warn("No file paths found in file-write-notification")
		return
	}

	for _, filePath := range filePaths {
		processSingleFileWrite(ctx, event, eventCtx, filePath)
	}
}

func processSingleFileWrite(ctx context.Context, event ADKEvent, eventCtx EventContext, filePath string) {
	msg := &db.Message{
		Sender:         SenderAgent,
		Text:           "File updated: " + filePath,
		IsUser:         false,
		Phase:          eventCtx.Phase,
		SessionId:      eventCtx.SessionID,
		Timestamp:      time.Now(),
		IsTransient:    true,
		ExpireAt:       time.Now().Add(transientExpiration),
		IsNotification: true,
		NotificationDetails: &db.NotificationDetails{
			Name: NotificationFileWrite,
			Params: map[string]interface{}{
				"filePath": filePath,
			},
		},
	}

	if _, err := Store.CreateMessage(ctx, eventCtx.OrganizationID, eventCtx.ProjectID, eventCtx.UserID, msg); err != nil {
		slog.Error("Failed to create transient message", "error", err, "filePath", filePath)
	}

	if OnFileWriteHook != nil {
		OnFileWriteHook(ctx, eventCtx.ProjectID, eventCtx.OrganizationID, eventCtx.Phase, filePath)
	}
}
