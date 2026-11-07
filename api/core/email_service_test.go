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

package core

import (
	"context"
	"errors"
	"testing"

	"google.com/apf/api/notifications"
)

func TestEmailServiceInstance_AdapterDelegation(t *testing.T) {
	mock := notifications.NewMockEmailProvider()
	notifications.SetDefaultProvider(mock)

	cfg := &Configuration{EmailServiceType: "mock"}
	InitializeEmailService(cfg)

	if EmailServiceInstance == nil {
		t.Fatalf("expected EmailServiceInstance to be non-nil")
	}

	err := EmailServiceInstance.SendEmail(
		context.Background(),
		"recipient@example.com",
		"Recipient Name",
		"Test Subject",
		"<p>Hello HTML</p>",
		"Hello Text",
	)
	if err != nil {
		t.Fatalf("expected successful send, got error: %v", err)
	}

	sent := mock.SentNotifications()
	if len(sent) != 1 {
		t.Fatalf("expected 1 sent notification, got %d", len(sent))
	}

	n := sent[0]
	if n.ToEmail != "recipient@example.com" || n.ToName != "Recipient Name" || n.Subject != "Test Subject" || n.HTMLContent != "<p>Hello HTML</p>" || n.TextContent != "Hello Text" {
		t.Fatalf("unexpected notification data: %+v", n)
	}
	if n.RequestID == "" {
		t.Fatalf("expected auto-generated request ID on notification")
	}

	// Test error propagation
	injectedErr := errors.New("simulated dispatch error")
	mock.SetError(injectedErr)

	err = EmailServiceInstance.SendEmail(context.Background(), "fail@example.com", "", "Fail", "", "fail")
	if !errors.Is(err, injectedErr) {
		t.Fatalf("expected injected error %v, got %v", injectedErr, err)
	}
}

func TestEmailServiceInstance_ConsoleDefault(t *testing.T) {
	cfg := &Configuration{EmailServiceType: "CONSOLE"}
	InitializeEmailService(cfg)

	err := EmailServiceInstance.SendEmail(
		context.Background(),
		"console@example.com",
		"Console User",
		"Console Subject",
		"<b>Hi</b>",
		"Hi",
	)
	if err != nil {
		t.Fatalf("expected successful send in console mode, got: %v", err)
	}
}

func TestProviderEmailServiceAdapter_NilReceiverAndProvider(t *testing.T) {
	mock := notifications.NewMockEmailProvider()
	notifications.SetDefaultProvider(mock)

	// 1. Adapter with nil provider should fallback to default provider
	adapterWithNilProvider := NewProviderEmailServiceAdapter(nil)
	err := adapterWithNilProvider.SendEmail(context.Background(), "user1@example.com", "User 1", "Subj 1", "HTML", "Text")
	if err != nil {
		t.Fatalf("expected successful send with nil provider adapter, got: %v", err)
	}

	// 2. Nil adapter receiver should fallback to default provider
	var nilAdapter *ProviderEmailServiceAdapter
	err = nilAdapter.SendEmail(context.Background(), "user2@example.com", "User 2", "Subj 2", "HTML", "Text")
	if err != nil {
		t.Fatalf("expected successful send with nil adapter pointer, got: %v", err)
	}

	if len(mock.SentNotifications()) != 2 {
		t.Errorf("expected 2 sent notifications, got %d", len(mock.SentNotifications()))
	}
}

