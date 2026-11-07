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

package notifications

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MockEmailProvider is an in-memory test spy implementation of EmailNotificationProvider.
type MockEmailProvider struct {
	mu        sync.RWMutex
	sent      []*EmailNotification
	injectErr error
}

// NewMockEmailProvider creates a new MockEmailProvider.
func NewMockEmailProvider() *MockEmailProvider {
	return &MockEmailProvider{
		sent: make([]*EmailNotification, 0),
	}
}

// Name returns the provider identifier "mock".
func (m *MockEmailProvider) Name() string {
	return "mock"
}

// ValidateConfig verifies provider configuration (always succeeds for mock).
func (m *MockEmailProvider) ValidateConfig() error {
	return nil
}

// Send records the notification in memory or returns an injected error.
func (m *MockEmailProvider) Send(ctx context.Context, notif *EmailNotification) (*SendResult, error) {
	if m == nil {
		return nil, fmt.Errorf("mock provider cannot be nil")
	}
	if notif == nil {
		return nil, fmt.Errorf("notification cannot be nil")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.injectErr != nil {
		return nil, m.injectErr
	}

	reqID := notif.RequestID
	if reqID == "" {
		reqID = uuid.New().String()
	}

	m.sent = append(m.sent, notif)

	return &SendResult{
		Provider:  "mock",
		MessageID: fmt.Sprintf("mock-%s", reqID),
		Status:    SendStatusSuccess,
		Timestamp: time.Now().UTC(),
		Details: map[string]interface{}{
			"sentIndex": len(m.sent) - 1,
		},
	}, nil
}

// SentNotifications returns a copy of all notifications sent through this provider.
func (m *MockEmailProvider) SentNotifications() []*EmailNotification {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*EmailNotification, len(m.sent))
	copy(result, m.sent)
	return result
}

// Clear resets the sent notification history and clears any injected error.
func (m *MockEmailProvider) Clear() {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sent = make([]*EmailNotification, 0)
	m.injectErr = nil
}

// SetError sets a simulated error to be returned on subsequent Send calls.
func (m *MockEmailProvider) SetError(err error) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	m.injectErr = err
}
