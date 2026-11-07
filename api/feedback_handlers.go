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
	"encoding/json"
	"net/http"
	"strings"
	"unicode/utf8"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

// FeedbackHandler handles REST endpoints for user feedback.
type FeedbackHandler struct {
	store db.FeedbackStore
}

// NewFeedbackHandler creates a new FeedbackHandler with the provided store.
func NewFeedbackHandler(store db.FeedbackStore) *FeedbackHandler {
	return &FeedbackHandler{store: store}
}

// CreateFeedbackRequest defines the incoming payload for feedback submission.
type CreateFeedbackRequest struct {
	Type           string `json:"type"`
	Subject        string `json:"subject"`
	Message        string `json:"message"`
	UserEmail      string `json:"userEmail,omitempty"`
	OrganizationID string `json:"organizationId,omitempty"`
	ProjectID      string `json:"projectId,omitempty"`
}

// CreateFeedbackResponse defines the successful response payload.
type CreateFeedbackResponse struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// FeedbackErrorResponse defines error details returned to the caller.
type FeedbackErrorResponse struct {
	Code  string `json:"code"`
	Error string `json:"error"`
}

// CreateFeedbackHandler processes POST requests to create new feedback.
func (h *FeedbackHandler) CreateFeedbackHandler(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.store == nil {
		sendJSON(w, http.StatusInternalServerError, FeedbackErrorResponse{
			Code:  "INTERNAL_ERROR",
			Error: "Feedback store uninitialized",
		})
		return
	}

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		sendJSON(w, http.StatusMethodNotAllowed, FeedbackErrorResponse{
			Code:  "METHOD_NOT_ALLOWED",
			Error: "Only POST method is allowed",
		})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1048576)

	var req CreateFeedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, FeedbackErrorResponse{
			Code:  "INVALID_FEEDBACK_PAYLOAD",
			Error: "Invalid JSON request payload",
		})
		return
	}

	feedbackType := strings.TrimSpace(req.Type)
	subject := strings.TrimSpace(req.Subject)
	message := strings.TrimSpace(req.Message)

	// Validate type
	if feedbackType != "bug" && feedbackType != "feature_request" && feedbackType != "general" {
		sendJSON(w, http.StatusBadRequest, FeedbackErrorResponse{
			Code:  "INVALID_FEEDBACK_PAYLOAD",
			Error: "Feedback type must be 'bug', 'feature_request', or 'general'",
		})
		return
	}

	// Validate subject
	if subject == "" {
		sendJSON(w, http.StatusBadRequest, FeedbackErrorResponse{
			Code:  "INVALID_FEEDBACK_PAYLOAD",
			Error: "Subject is required",
		})
		return
	}
	if utf8.RuneCountInString(subject) > 200 {
		sendJSON(w, http.StatusBadRequest, FeedbackErrorResponse{
			Code:  "INVALID_FEEDBACK_PAYLOAD",
			Error: "Subject must not exceed 200 characters",
		})
		return
	}

	// Validate message
	if message == "" {
		sendJSON(w, http.StatusBadRequest, FeedbackErrorResponse{
			Code:  "INVALID_FEEDBACK_PAYLOAD",
			Error: "Message is required",
		})
		return
	}
	if utf8.RuneCountInString(message) > 4000 {
		sendJSON(w, http.StatusBadRequest, FeedbackErrorResponse{
			Code:  "INVALID_FEEDBACK_PAYLOAD",
			Error: "Message must not exceed 4000 characters",
		})
		return
	}

	// Context defaults extraction
	userEmail := strings.TrimSpace(req.UserEmail)
	if userEmail == "" {
		if claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims); ok && claims != nil {
			if claims.Username != "" {
				userEmail = claims.Username
			} else if claims.Subject != "" {
				userEmail = claims.Subject
			}
		}
	}

	orgID := strings.TrimSpace(req.OrganizationID)
	if orgID == "" {
		if ctxOrgID, err := core.GetOrganizationID(r.Context()); err == nil && ctxOrgID != "" {
			orgID = ctxOrgID
		}
	}

	fb := &db.Feedback{
		Type:           feedbackType,
		Subject:        subject,
		Message:        message,
		UserEmail:      userEmail,
		OrganizationID: orgID,
		ProjectID:      strings.TrimSpace(req.ProjectID),
	}

	created, err := h.store.CreateFeedback(r.Context(), fb)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, FeedbackErrorResponse{
			Code:  "INTERNAL_ERROR",
			Error: "Failed to persist feedback",
		})
		return
	}

	sendJSON(w, http.StatusCreated, CreateFeedbackResponse{
		ID:      created.ID,
		Status:  "success",
		Message: "Feedback submitted successfully",
	})
}
