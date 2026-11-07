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
	"log/slog"

	"github.com/google/uuid"
	"google.com/apf/api/notifications"
)

// EmailService defines the legacy interface for sending emails.
type EmailService interface {
	SendEmail(ctx context.Context, toEmail, toName, subject, htmlContent, textContent string) error
}

// Global instance of EmailService
var EmailServiceInstance EmailService

// ProviderEmailServiceAdapter adapts an EmailNotificationProvider to the legacy EmailService interface.
type ProviderEmailServiceAdapter struct {
	provider notifications.EmailNotificationProvider
}

// NewProviderEmailServiceAdapter creates a new adapter wrapping the given provider.
// If provider is nil, it delegates dynamically to notifications.GetDefaultProvider().
func NewProviderEmailServiceAdapter(provider notifications.EmailNotificationProvider) *ProviderEmailServiceAdapter {
	return &ProviderEmailServiceAdapter{provider: provider}
}

// SendEmail delegates the email sending request to the underlying EmailNotificationProvider.
func (a *ProviderEmailServiceAdapter) SendEmail(ctx context.Context, toEmail, toName, subject, htmlContent, textContent string) error {
	var p notifications.EmailNotificationProvider
	if a != nil {
		p = a.provider
	}
	if p == nil {
		p = notifications.GetDefaultProvider()
	}

	notif := &notifications.EmailNotification{
		ToEmail:     toEmail,
		ToName:      toName,
		Subject:     subject,
		HTMLContent: htmlContent,
		TextContent: textContent,
		RequestID:   uuid.New().String(),
	}

	_, err := p.Send(ctx, notif)
	return err
}

// InitializeEmailService initializes the global EmailServiceInstance based on configuration,
// delegating to the notifications provider registry.
func InitializeEmailService(cfg *Configuration) {
	if cfg != nil {
		notifications.InitializeProviders(cfg)
	}

	defaultProvider := notifications.GetDefaultProvider()
	slog.Info("Initialized legacy EmailServiceInstance wrapping default notification provider",
		"provider", defaultProvider.Name())
	EmailServiceInstance = NewProviderEmailServiceAdapter(defaultProvider)
}
