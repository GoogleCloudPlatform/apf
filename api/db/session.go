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
	"sort"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

type Session struct {
	ID             string    `json:"id" firestore:"-"`
	UserId         string    `json:"userId" firestore:"userId"`
	ProjectId      string    `json:"projectId" firestore:"projectId"`
	Phase          string    `json:"phase" firestore:"phase"`
	CreatedAt      time.Time `json:"createdAt" firestore:"createdAt"`
	LastActiveAt   time.Time `json:"lastActiveAt" firestore:"lastActiveAt"`
	Status         string    `json:"status" firestore:"status"`
	SpecBranchName string    `json:"specBranchName" firestore:"specBranchName"`
	Title          string    `json:"title,omitempty" firestore:"title,omitempty"`
}

type SessionStore interface {
	CreateSession(ctx context.Context, organizationID string, s *Session) error
	ListSessionsForProject(ctx context.Context, organizationID string, projectId string, userId string, phase string) ([]Session, error)
	UpdateSessionStatus(ctx context.Context, organizationID string, id string, status string) error
	UpdateSessionTitle(ctx context.Context, organizationID string, id string, title string) error
	DeleteSession(ctx context.Context, organizationID string, id string) error
}

type FirestoreSessionStore struct {
	client *firestore.Client
}

func NewFirestoreSessionStore(client *firestore.Client) *FirestoreSessionStore {
	return &FirestoreSessionStore{client: client}
}

func (s *FirestoreSessionStore) CreateSession(ctx context.Context, organizationID string, sess *Session) error {
	col := organizationCollection(organizationID, s.client, "sessions")
	var ref *firestore.DocumentRef
	var err error
	if sess.ID == "" {
		ref, _, err = col.Add(ctx, sess)
	} else {
		ref = col.Doc(sess.ID)
		_, err = ref.Set(ctx, sess)
	}
	if err != nil {
		return err
	}
	sess.ID = ref.ID
	return nil
}

func (s *FirestoreSessionStore) ListSessionsForProject(ctx context.Context, organizationID string, projectId string, userId string, phase string) ([]Session, error) {
	sessions := []Session{}
	col := organizationCollection(organizationID, s.client, "sessions")
	query := col.
		Where("projectId", "==", projectId).
		Where("userId", "==", userId)
	if phase != "" {
		query = query.Where("phase", "==", phase)
	}
	iter := query.Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var sess Session
		if err := doc.DataTo(&sess); err != nil {
			return nil, err
		}
		sess.ID = doc.Ref.ID
		sessions = append(sessions, sess)
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].CreatedAt.After(sessions[j].CreatedAt)
	})

	return sessions, nil
}

func (s *FirestoreSessionStore) UpdateSessionStatus(ctx context.Context, organizationID string, id string, status string) error {
	col := organizationCollection(organizationID, s.client, "sessions")
	_, err := col.Doc(id).Update(ctx, []firestore.Update{
		{Path: "status", Value: status},
		{Path: "lastActiveAt", Value: time.Now()},
	})
	return err
}

func (s *FirestoreSessionStore) UpdateSessionTitle(ctx context.Context, organizationID string, id string, title string) error {
	col := organizationCollection(organizationID, s.client, "sessions")
	_, err := col.Doc(id).Update(ctx, []firestore.Update{
		{Path: "title", Value: title},
		{Path: "lastActiveAt", Value: time.Now()},
	})
	return err
}

func (s *FirestoreSessionStore) DeleteSession(ctx context.Context, organizationID string, id string) error {
	col := organizationCollection(organizationID, s.client, "sessions")
	// First delete all messages in the subcollection to prevent orphaned subcollection documents
	messagesRef := col.Doc(id).Collection("messages")
	iter := messagesRef.DocumentRefs(ctx)
	for {
		docRef, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to list messages for deletion: %w", err)
		}
		if _, err := docRef.Delete(ctx); err != nil {
			return fmt.Errorf("failed to delete message: %w", err)
		}
	}

	// Then delete the session document itself
	_, err := col.Doc(id).Delete(ctx)
	return err
}
