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
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

// StreamActivityHandler handles real-time project-wide activity streaming via SSE.
func StreamActivityHandler(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("id")
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		http.Error(w, "Unauthorized context", http.StatusUnauthorized)
		return
	}

	sinceStr := r.URL.Query().Get("since")
	var since time.Time
	if sinceStr != "" {
		var err error
		since, err = time.Parse(time.RFC3339Nano, sinceStr)
		if err != nil {
			// Fallback to standard RFC3339
			since, err = time.Parse(time.RFC3339, sinceStr)
			if err != nil {
				http.Error(w, "Invalid since parameter", http.StatusBadRequest)
				return
			}
		}
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	// Subtask 3.10: Removed wildcard CORS header; handled by corsMiddleware

	w.WriteHeader(http.StatusOK)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	connTime := time.Now()
	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		http.Error(w, "Unauthorized context: missing organization ID", http.StatusUnauthorized)
		return
	}
	streamCtx, cancel := context.WithCancel(r.Context())
	msgChan, errChan := Store.StreamProjectActivities(streamCtx, organizationID, projectId, since)
	defer func() {
		cancel()
		for range msgChan {
		}
		for range errChan {
		}
	}()

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
				slog.Error("Failed to marshal message in activity stream", "error", err)
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", string(jsonBytes))
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		case err := <-errChan:
			if err == nil {
				return
			}
			slog.Error("Firestore snapshot error in activity stream", "error", err)
			// Subtask 3.11: Send SSE error framing on stream error
			fmt.Fprintf(w, "event: error\ndata: {\"error\": %q}\n\n", err.Error())
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
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

// IngestActivityHandler handles external CLI/IDE developer activity ingest.
func IngestActivityHandler(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("id")
	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		sendJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized context"})
		return
	}

	// Subtask 3.8: Add payload size limit and field validation
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024) // 1MB limit

	var msg db.Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON payload"})
		return
	}

	if msg.Sender == "" || msg.Text == "" {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Sender and Text fields are required"})
		return
	}

	// Prevent spoofing "System" or agent personas by validating msg.Sender matches the authenticated user
	if msg.Sender != claims.Subject && msg.Sender != claims.Username {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Sender must match your authenticated user identity to prevent spoofing"})
		return
	}

	msg.IsUser = false
	msg.IsNotification = true
	msg.IsTransient = true

	// Subtask 3.9: Preserve historical timestamps if provided
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	} else if msg.Timestamp.After(time.Now()) {
		msg.Timestamp = time.Now() // Clamp future timestamps
	}
	if msg.ExpireAt.IsZero() {
		msg.ExpireAt = time.Now().Add(24 * time.Hour)
	}

	// Validate NotificationDetails structure size and length limits to prevent document bloat
	if msg.NotificationDetails != nil {
		if len(msg.NotificationDetails.Name) > 100 {
			msg.NotificationDetails.Name = msg.NotificationDetails.Name[:100]
		}
		if len(msg.NotificationDetails.Params) > 20 {
			sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Too many parameters in notification details"})
			return
		}
	}

	// Prevent external ingest payloads from overriding SessionId to arbitrary paths.
	// All ingested activities must route to the global project feed channel.
	msg.SessionId = "project_global_" + projectId

	organizationID, errOrg := core.GetOrganizationID(r.Context())
	if errOrg != nil {
		sendJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized context: missing organization ID"})
		return
	}

	createdMsg, err := Store.CreateMessage(r.Context(), organizationID, projectId, claims.Subject, &msg)
	if err != nil {
		slog.Error("Failed to save ingested activity to Firestore", "error", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to save activity"})
		return
	}

	sendJSON(w, http.StatusCreated, createdMsg)
}
