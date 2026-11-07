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
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// ConsoleEmailProvider implements EmailNotificationProvider for local development and testing by logging structured emails.
type ConsoleEmailProvider struct{}

// NewConsoleEmailProvider creates a new ConsoleEmailProvider.
func NewConsoleEmailProvider() *ConsoleEmailProvider {
	return &ConsoleEmailProvider{}
}

// Name returns the provider identifier "console".
func (c *ConsoleEmailProvider) Name() string {
	return "console"
}

// ValidateConfig verifies that the provider configuration is valid.
func (c *ConsoleEmailProvider) ValidateConfig() error {
	return nil
}

// Send logs structured email notification details via slog and returns a successful SendResult.
func (c *ConsoleEmailProvider) Send(ctx context.Context, notif *EmailNotification) (*SendResult, error) {
	if notif == nil {
		return nil, fmt.Errorf("notification cannot be nil")
	}

	reqID := notif.RequestID
	if reqID == "" {
		reqID = uuid.New().String()
	}

	preview := truncatePreview(notif.TextContent, 200)
	if preview == "" && notif.HTMLContent != "" {
		preview = "(HTML content only)"
	}

	slog.Info("Email notification dispatched (CONSOLE mode)",
		"to", notif.ToEmail,
		"toName", notif.ToName,
		"subject", notif.Subject,
		"requestId", reqID,
		"htmlContentLength", len(notif.HTMLContent),
		"textContentLength", len(notif.TextContent),
		"preview", preview,
	)

	return &SendResult{
		Provider:  "console",
		MessageID: fmt.Sprintf("console-%s", reqID),
		Status:    SendStatusSuccess,
		Timestamp: time.Now().UTC(),
		Details: map[string]interface{}{
			"preview":           preview,
			"htmlContentLength": len(notif.HTMLContent),
			"textContentLength": len(notif.TextContent),
		},
	}, nil
}

func truncatePreview(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes])
}
