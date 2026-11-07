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

import "strings"

func extractNodeName(path string) string {
	if path == "" {
		return ""
	}
	segments := strings.Split(path, "/")
	if len(segments) == 0 {
		return ""
	}
	lastSegment := segments[len(segments)-1]
	nodeNameSegs := strings.Split(lastSegment, "@")
	if len(nodeNameSegs) == 0 {
		return ""
	}
	return nodeNameSegs[0]
}

func hasText(event ADKEvent) bool {
	if event.Content != nil && len(event.Content.Parts) > 0 {
		_, ok := event.Content.Parts[0][KeyText]
		return ok
	}
	return false
}

func hasThought(event ADKEvent) bool {
	if event.Content != nil && len(event.Content.Parts) > 0 {
		if _, ok := event.Content.Parts[0][KeyThought]; ok {
			return true
		}
		if _, ok := event.Content.Parts[0][KeyThoughtChunks]; ok {
			return true
		}
		if _, ok := event.Content.Parts[0][KeyThinking]; ok {
			return true
		}
	}
	return false
}

func hasFunctionCall(event ADKEvent) bool {
	if event.Content != nil && len(event.Content.Parts) > 0 {
		_, ok := event.Content.Parts[0][KeyFunctionCall]
		return ok
	}
	return false
}

func hasFunctionResponse(event ADKEvent) bool {
	if event.Content != nil && len(event.Content.Parts) > 0 {
		_, ok := event.Content.Parts[0][KeyFunctionResponse]
		return ok
	}
	return false
}

func hasStateOrArtifactDelta(event ADKEvent) bool {
	if event.Actions != nil {
		_, hasState := event.Actions[ActionStateDelta]
		_, hasArtifact := event.Actions[ActionArtifactDelta]
		return hasState || hasArtifact
	}
	return false
}

func hasTransferSignal(event ADKEvent) bool {
	if event.Actions != nil {
		_, ok := event.Actions[ActionTransfer]
		return ok
	}
	return false
}

func hasEscalationSignal(event ADKEvent) bool {
	if event.Actions != nil {
		_, ok := event.Actions[ActionEscalate]
		return ok
	}
	return false
}

func hasFileWriteNotification(event ADKEvent) bool {
	if event.Actions != nil {
		if stateDelta, ok := event.Actions[ActionStateDelta].(map[string]interface{}); ok {
			for k := range stateDelta {
				if strings.HasPrefix(k, PrefixFileWriteNotif) {
					return true
				}
			}
		}
		if stateDelta, ok := event.Actions[ActionStateDeltaSnake].(map[string]interface{}); ok {
			for k := range stateDelta {
				if strings.HasPrefix(k, PrefixFileWriteNotif) {
					return true
				}
			}
		}
	}
	return false
}

func isFileReaderTool(name string) bool {
	lower := strings.ToLower(name)
	return strings.Contains(lower, "read_file") ||
		strings.Contains(lower, "view_file") ||
		strings.Contains(lower, "file_reader") ||
		lower == "read"
}
