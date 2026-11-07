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

package db

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

type NotificationDetails struct {
	Name   string                 `json:"name" firestore:"name"`
	Params map[string]interface{} `json:"params" firestore:"params"`
}

type NodeInfo struct {
	Path string `json:"path" firestore:"path"`
}

type EventContent struct {
	Role  string                   `json:"role" firestore:"role"`
	Parts []map[string]interface{} `json:"parts" firestore:"parts"`
}

type ADKEvent struct {
	Author       string                 `json:"author" firestore:"author"`
	InvocationID string                 `json:"invocation_id" firestore:"invocation_id"`
	Content      *EventContent          `json:"content,omitempty" firestore:"content,omitempty"`
	Partial      bool                   `json:"partial" firestore:"partial"`
	TurnComplete bool                   `json:"turn_complete" firestore:"turnComplete"`
	Actions      map[string]interface{} `json:"actions,omitempty" firestore:"actions,omitempty"`
	NodeInfo     *NodeInfo              `json:"node_info,omitempty" firestore:"nodeInfo,omitempty"`
}

type FunctionResponse struct {
	Name     string                 `json:"name" firestore:"name"`
	ID       string                 `json:"id" firestore:"id"`
	Response map[string]interface{} `json:"response" firestore:"response"`
}

type PhaseSwitchSuggestion struct {
	TargetPhase     string `json:"targetPhase" firestore:"targetPhase"`
	SkillID         string `json:"skillId,omitempty" firestore:"skillId,omitempty"`
	Reason          string `json:"reason,omitempty" firestore:"reason,omitempty"`
	SuggestedPrompt string `json:"suggestedPrompt,omitempty" firestore:"suggestedPrompt,omitempty"`
}

// Message represents a chat message.
type Message struct {
	ID                    string                 `json:"id,omitempty" firestore:"id,omitempty"`
	OrganizationID        string                 `json:"organizationId,omitempty" firestore:"organizationId,omitempty"`
	ProjectId             string                 `json:"projectId,omitempty" firestore:"projectId,omitempty"`
	Sender                string                 `json:"sender" firestore:"sender"`
	Text                  string                 `json:"text" firestore:"text"`
	Prompt                string                 `json:"prompt,omitempty" firestore:"prompt,omitempty"`
	DisplayText           string                 `json:"displayText,omitempty" firestore:"displayText,omitempty"`
	DisplayPrompt         string                 `json:"displayPrompt,omitempty" firestore:"displayPrompt,omitempty"`
	Actions               []string               `json:"actions,omitempty" firestore:"actions,omitempty"`
	PhaseSwitchSuggestion *PhaseSwitchSuggestion `json:"phaseSwitchSuggestion,omitempty" firestore:"phaseSwitchSuggestion,omitempty"`
	Phase                 string                 `json:"phase,omitempty" firestore:"phase,omitempty"`
	SessionId             string                 `json:"sessionId,omitempty" firestore:"sessionId,omitempty"`
	Timestamp             time.Time              `json:"timestamp" firestore:"timestamp"`
	ExpireAt              time.Time              `json:"expireAt,omitempty" firestore:"expireAt,omitempty"`
	NotificationDetails   *NotificationDetails   `json:"notificationDetails,omitempty" firestore:"notificationDetails,omitempty"`
	Event                 *ADKEvent              `json:"event,omitempty" firestore:"event,omitempty"`
	IsUser                bool                   `json:"isUser" firestore:"isUser"`
	IsTransient           bool                   `json:"isTransient" firestore:"isTransient"`
	AwaitingInput         bool                   `json:"awaitingInput,omitempty" firestore:"awaitingInput,omitempty"`
	IsNotification        bool                   `json:"isNotification,omitempty" firestore:"isNotification,omitempty"`
	IsHistorical          bool                   `json:"isHistorical,omitempty" firestore:"isHistorical,omitempty"`
	FunctionResponse      *FunctionResponse      `json:"functionResponse,omitempty" firestore:"functionResponse,omitempty"`
	ArtifactIDs           []string               `json:"artifactIds,omitempty" firestore:"artifactIds,omitempty"`
	Artifacts             []Artifact             `json:"artifacts,omitempty" firestore:"artifacts,omitempty"`
}

// MessageStore interface for messaging data operations.
type MessageStore interface {
	CreateMessage(ctx context.Context, organizationID string, projectId string, userId string, m *Message) (*Message, error)
	ListMessages(ctx context.Context, organizationID string, projectId string, userId string, sessionId string, includeTransient bool) ([]Message, error)
	StreamMessages(ctx context.Context, organizationID string, projectId string, userId string, sessionId string, since time.Time) (<-chan Message, <-chan error)
	StreamProjectActivities(ctx context.Context, organizationID string, projectId string, since time.Time) (<-chan Message, <-chan error)
}

// FirestoreMessageStore implements MessageStore for Firestore.
type FirestoreMessageStore struct {
	client       *firestore.Client
	sessionStore SessionStore
}

// NewFirestoreMessageStore creates a new FirestoreMessageStore.
func NewFirestoreMessageStore(client *firestore.Client, sessStore SessionStore) *FirestoreMessageStore {
	return &FirestoreMessageStore{client: client, sessionStore: sessStore}
}

func (s *FirestoreMessageStore) CreateMessage(ctx context.Context, organizationID string, projectId string, userId string, m *Message) (*Message, error) {
	if m == nil {
		return nil, fmt.Errorf("message cannot be nil")
	}
	msgCopy := *m
	msgCopy.ProjectId = projectId
	msgCopy.OrganizationID = organizationID
	if msgCopy.SessionId == "" {
		msgCopy.SessionId = "project_global_" + projectId
	}
	if msgCopy.Timestamp.IsZero() {
		msgCopy.Timestamp = time.Now()
	}
	col := organizationCollection(organizationID, s.client, "sessions")
	docRef := col.Doc(msgCopy.SessionId).Collection("messages").NewDoc()
	msgCopy.ID = docRef.ID
	_, err := docRef.Set(ctx, &msgCopy)
	return &msgCopy, err
}

func (s *FirestoreMessageStore) ListMessages(ctx context.Context, organizationID string, projectId string, userId string, sessionId string, includeTransient bool) ([]Message, error) {
	var messages []Message
	if sessionId == "" {
		slog.Warn("Session ID is empty, returning no messages", "projectId", projectId, "userId", userId)
		return messages, nil
	}

	col := organizationCollection(organizationID, s.client, "sessions")
	q := col.Doc(sessionId).Collection("messages").OrderBy("timestamp", firestore.Asc)
	if !includeTransient {
		q = q.Where("isTransient", "==", false)
	}

	iter := q.Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var m Message
		if err := doc.DataTo(&m); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, nil
}

func (s *FirestoreMessageStore) StreamMessages(ctx context.Context, organizationID string, projectId string, userId string, sessionId string, since time.Time) (<-chan Message, <-chan error) {
	msgChan := make(chan Message, 100)
	errChan := make(chan error, 10)

	go func() {
		defer close(msgChan)
		defer close(errChan)

		col := organizationCollection(organizationID, s.client, "sessions")

		var query firestore.Query
		query = col.Doc(sessionId).Collection("messages").OrderBy("timestamp", firestore.Asc)
		if !since.IsZero() {
			query = query.Where("timestamp", ">", since)
		}

		it := query.Snapshots(ctx)
		defer it.Stop()

		for {
			snap, err := it.Next()
			if err == iterator.Done {
				return
			}
			if err != nil {
				select {
				case errChan <- err:
				case <-ctx.Done():
				}
				return
			}
			var addedMsgs []Message
			for _, change := range snap.Changes {
				if change.Kind == firestore.DocumentAdded {
					var msg Message
					if err := change.Doc.DataTo(&msg); err != nil {
						continue
					}
					addedMsgs = append(addedMsgs, msg)
				}
			}
			sort.SliceStable(addedMsgs, func(i, j int) bool {
				return addedMsgs[i].Timestamp.Before(addedMsgs[j].Timestamp)
			})
			for _, msg := range addedMsgs {
				select {
				case msgChan <- msg:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return msgChan, errChan
}

func (s *FirestoreMessageStore) StreamProjectActivities(ctx context.Context, organizationID string, projectId string, since time.Time) (<-chan Message, <-chan error) {
	msgChan := make(chan Message, 100)
	errChan := make(chan error, 10)

	go func() {
		defer close(msgChan)
		defer close(errChan)

		colGroupQuery := organizationCollectionGroup(organizationID, s.client, "messages")

		if since.IsZero() {
			// Fetch the 100th latest message's timestamp to use as the 'since' threshold
			limitQuery := colGroupQuery.
				Where("projectId", "==", projectId).
				Where("isNotification", "==", true).
				OrderBy("timestamp", firestore.Desc).
				Limit(100)
			docs, err := limitQuery.Documents(ctx).GetAll()
			if err == nil && len(docs) > 0 {
				var oldestMsg Message
				if err := docs[len(docs)-1].DataTo(&oldestMsg); err == nil {
					since = oldestMsg.Timestamp.Add(-1 * time.Millisecond)
				}
			}
		}
		if since.IsZero() {
			since = time.Now().Add(-24 * time.Hour) // Default fallback window
		}

		query := colGroupQuery.Where("projectId", "==", projectId).Where("isNotification", "==", true).OrderBy("timestamp", firestore.Asc)
		if !since.IsZero() {
			query = query.Where("timestamp", ">", since)
		}

		it := query.Snapshots(ctx)
		defer it.Stop()

		for {
			snap, err := it.Next()
			if err == iterator.Done {
				return
			}
			if err != nil {
				select {
				case errChan <- err:
				case <-ctx.Done():
				}
				return
			}
			var addedMsgs []Message
			for _, change := range snap.Changes {
				if change.Kind == firestore.DocumentAdded {
					var msg Message
					if err := change.Doc.DataTo(&msg); err != nil {
						slog.Warn("Failed to decode message document in activity stream", "error", err, "docId", change.Doc.Ref.ID)
						continue
					}
					addedMsgs = append(addedMsgs, msg)
				}
			}
			sort.SliceStable(addedMsgs, func(i, j int) bool {
				return addedMsgs[i].Timestamp.Before(addedMsgs[j].Timestamp)
			})
			for _, msg := range addedMsgs {
				select {
				case msgChan <- msg:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return msgChan, errChan
}
