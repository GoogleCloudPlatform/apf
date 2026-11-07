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
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"google.com/apf/api/db"
)

func formatToolCallText(event ADKEvent) string {
	if event.Content == nil || len(event.Content.Parts) == 0 {
		return DefaultExecutingToolText
	}
	funcCall, ok := event.Content.Parts[0][KeyFunctionCall].(map[string]interface{})
	if !ok {
		return DefaultExecutingToolText
	}
	name, _ := funcCall[KeyName].(string)
	if name == "" {
		return DefaultExecutingToolText
	}

	var argSummary string
	if args, ok := funcCall[KeyArgs].(map[string]interface{}); ok {
		for _, key := range toolArgCandidateKeys {
			if val, exists := args[key]; exists {
				if s, ok := val.(string); ok && s != "" {
					argSummary = s
					break
				}
			}
		}
	}

	if argSummary != "" {
		return fmt.Sprintf("Executing tool: `%s` (`%s`)", name, argSummary)
	}
	return fmt.Sprintf("Executing tool: `%s`", name)
}

func extractResponseContentStr(respObj interface{}) string {
	if respObj == nil {
		return ""
	}
	if s, ok := respObj.(string); ok {
		return s
	}
	if m, ok := respObj.(map[string]interface{}); ok {
		for _, key := range []string{"result", "content", "output", "text", "lines"} {
			if val, exists := m[key]; exists {
				if s, ok := val.(string); ok && s != "" {
					return s
				}
			}
		}
		if respVal, exists := m[KeyResponse]; exists {
			return extractResponseContentStr(respVal)
		}
		bytes, err := json.MarshalIndent(m, "", "  ")
		if err == nil {
			return string(bytes)
		}
	}
	return fmt.Sprintf("%v", respObj)
}

func formatToolResultText(event ADKEvent) (string, string, string, bool, string) {
	toolName := ""
	contentStr := ""
	fileName := ""

	if event.Content != nil && len(event.Content.Parts) > 0 {
		if funcResp, ok := event.Content.Parts[0][KeyFunctionResponse].(map[string]interface{}); ok {
			if name, ok := funcResp[KeyName].(string); ok {
				toolName = name
			}
			if resp, exists := funcResp[KeyResponse]; exists {
				contentStr = extractResponseContentStr(resp)
				if respMap, ok := resp.(map[string]interface{}); ok {
					for _, key := range toolRespCandidateKeys {
						if val, exists := respMap[key]; exists {
							if s, ok := val.(string); ok && s != "" {
								fileName = s
								break
							}
						}
					}
				}
			}
			if fileName == "" {
				for _, key := range toolRespCandidateKeys {
					if val, exists := funcResp[key]; exists {
						if s, ok := val.(string); ok && s != "" {
							fileName = s
							break
						}
					}
				}
			}
		}
	}

	if event.InvocationID != "" {
		recentToolCallArgsMu.Lock()
		if fileName == "" {
			fileName = recentToolCallArgs[event.InvocationID]
		}
		if toolName == "" {
			toolName = recentToolCallNames[event.InvocationID]
		}
		delete(recentToolCallArgs, event.InvocationID)
		delete(recentToolCallNames, event.InvocationID)
		recentToolCallArgsMu.Unlock()
	}

	isFileReader := isFileReaderTool(toolName)

	if contentStr == "" {
		if isFileReader && fileName != "" {
			return fmt.Sprintf("Read file: %s", fileName), toolName, "", isFileReader, fileName
		}
		if toolName != "" {
			return fmt.Sprintf("Tool result (`%s`) received", toolName), toolName, "", isFileReader, fileName
		}
		return "Tool result received", "", "", false, ""
	}

	var formattedText string
	if isFileReader {
		if fileName != "" {
			formattedText = fmt.Sprintf("Read file: %s\n```\n%s\n```", fileName, contentStr)
		} else {
			formattedText = fmt.Sprintf("Read file (`%s`):\n```\n%s\n```", toolName, contentStr)
		}
	} else if strings.Contains(contentStr, "\n") || len(contentStr) > 100 {
		if toolName != "" {
			formattedText = fmt.Sprintf("Tool result (`%s`):\n```\n%s\n```", toolName, contentStr)
		} else {
			formattedText = fmt.Sprintf("Tool result received:\n```\n%s\n```", contentStr)
		}
	} else {
		if toolName != "" {
			formattedText = fmt.Sprintf("Tool result (`%s`): `%s`", toolName, contentStr)
		} else {
			formattedText = fmt.Sprintf("Tool result received: `%s`", contentStr)
		}
	}

	return formattedText, toolName, contentStr, isFileReader, fileName
}

func handleToolCallRequest(ctx context.Context, event ADKEvent, eventCtx EventContext) {
	slog.Info("Shell: Handling Tool Call Request")

	awaitingInput := false
	var toolName string
	var argSummary string
	if event.Content != nil && len(event.Content.Parts) > 0 {
		if funcCall, ok := event.Content.Parts[0][KeyFunctionCall].(map[string]interface{}); ok {
			if name, ok := funcCall[KeyName].(string); ok {
				toolName = name
				if name == "adk_request_input" {
					awaitingInput = true
					if db.WorkflowRunStoreInstance != nil && eventCtx.SessionID != "" {
						_ = db.WorkflowRunStoreInstance.UpdateWorkflowRunStatus(ctx, eventCtx.OrganizationID, eventCtx.SessionID, "AWAITING_INPUT")
					}
				}
			}
			if args, ok := funcCall[KeyArgs].(map[string]interface{}); ok {
				for _, key := range toolArgCandidateKeys {
					if val, exists := args[key]; exists {
						if s, ok := val.(string); ok && s != "" {
							argSummary = s
							break
						}
					}
				}
			}
		}
	}

	if event.InvocationID != "" {
		recentToolCallArgsMu.Lock()
		if argSummary != "" {
			recentToolCallArgs[event.InvocationID] = argSummary
		}
		if toolName != "" {
			recentToolCallNames[event.InvocationID] = toolName
		}
		recentToolCallArgsMu.Unlock()
	}

	if toolName == "directory_reader" && (config == nil || !config.IsDebug()) {
		slog.Debug("Suppressing directory_reader tool call notification because debug mode is false")
		return
	}

	msg := &db.Message{
		Sender:         SenderAgent,
		Text:           formatToolCallText(event),
		IsUser:         false,
		Phase:          eventCtx.Phase,
		SessionId:      eventCtx.SessionID,
		Timestamp:      time.Now(),
		IsTransient:    true,
		ExpireAt:       time.Now().Add(transientExpiration),
		IsNotification: true,
		AwaitingInput:  awaitingInput,
		Event:          &event,
	}

	if Store != nil {
		if _, err := Store.CreateMessage(ctx, eventCtx.OrganizationID, eventCtx.ProjectID, eventCtx.UserID, msg); err != nil {
			slog.Error("Failed to save tool call notification to Firestore", "error", err, "projectId", eventCtx.ProjectID, "sessionId", eventCtx.SessionID)
		}
	}
}

func extractPhaseSwitchSuggestionFromEvent(event ADKEvent, contentStr string) *db.PhaseSwitchSuggestion {
	var suggestion db.PhaseSwitchSuggestion
	if err := json.Unmarshal([]byte(contentStr), &suggestion); err == nil && suggestion.TargetPhase != "" {
		return &suggestion
	}
	if event.Content != nil && len(event.Content.Parts) > 0 {
		if funcResp, ok := event.Content.Parts[0][KeyFunctionResponse].(map[string]interface{}); ok {
			if respMap, ok := funcResp[KeyResponse].(map[string]interface{}); ok {
				targetPhase, _ := respMap["targetPhase"].(string)
				skillId, _ := respMap["skillId"].(string)
				reason, _ := respMap["reason"].(string)
				if targetPhase != "" {
					return &db.PhaseSwitchSuggestion{
						TargetPhase: targetPhase,
						SkillID:     skillId,
						Reason:      reason,
					}
				}
			}
		}
	}
	return nil
}

func handleToolResultProvided(ctx context.Context, event ADKEvent, eventCtx EventContext) {
	slog.Info("Shell: Handling Tool Result Provided")

	formattedText, toolName, contentStr, isFileReader, fileName := formatToolResultText(event)

	if toolName == "directory_reader" && (config == nil || !config.IsDebug()) {
		slog.Debug("Suppressing directory_reader tool result notification because debug mode is false")
		return
	}

	isTransient := true
	isNotification := !isFileReader
	var phaseSwitchSuggestion *db.PhaseSwitchSuggestion

	if toolName == "suggest_phase_switch" {
		phaseSwitchSuggestion = extractPhaseSwitchSuggestionFromEvent(event, contentStr)
		if phaseSwitchSuggestion != nil && eventCtx.SessionID != "" {
			recentToolCallArgsMu.Lock()
			recentPhaseSwitchSuggestions[eventCtx.SessionID] = phaseSwitchSuggestion
			recentToolCallArgsMu.Unlock()
		}
	}

	msg := &db.Message{
		Sender:                SenderAgent,
		Text:                  formattedText,
		IsUser:                false,
		Phase:                 eventCtx.Phase,
		SessionId:             eventCtx.SessionID,
		Timestamp:             time.Now(),
		IsTransient:           isTransient,
		ExpireAt:              time.Now().Add(transientExpiration),
		IsNotification:        isNotification,
		PhaseSwitchSuggestion: phaseSwitchSuggestion,
		NotificationDetails: &db.NotificationDetails{
			Name: NotificationToolResult,
			Params: map[string]interface{}{
				"invocation_id":  event.InvocationID,
				"tool_name":      toolName,
				"content":        contentStr,
				"is_file_reader": isFileReader,
				"file_name":      fileName,
			},
		},
		Event: &event,
	}
	if Store != nil {
		if _, err := Store.CreateMessage(ctx, eventCtx.OrganizationID, eventCtx.ProjectID, eventCtx.UserID, msg); err != nil {
			slog.Error("Failed to save tool result notification to Firestore", "error", err, "projectId", eventCtx.ProjectID, "sessionId", eventCtx.SessionID)
		}
	}
}
