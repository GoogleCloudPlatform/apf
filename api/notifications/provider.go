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
	"errors"
	"log/slog"
	"reflect"
	"strings"
	"sync"
	"time"
)

// SendStatus represents the normalized outcome status of a notification send attempt.
type SendStatus string

const (
	SendStatusSuccess   SendStatus = "SUCCESS"
	SendStatusDuplicate SendStatus = "DUPLICATE"
	SendStatusError     SendStatus = "ERROR"
)

// ErrProviderNotFound is returned when an unknown provider name is queried in the registry.
var ErrProviderNotFound = errors.New("notification provider not found")

// EmailNotification represents a normalized email notification request.
type EmailNotification struct {
	ToEmail     string                 `json:"toEmail"`
	ToName      string                 `json:"toName"`
	Subject     string                 `json:"subject"`
	HTMLContent string                 `json:"htmlContent"`
	TextContent string                 `json:"textContent"`
	RequestID   string                 `json:"requestId"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// SendResult represents the normalized response after attempting to dispatch a notification.
type SendResult struct {
	Provider             string                 `json:"provider"`
	MessageID            string                 `json:"messageId"`
	Status               SendStatus             `json:"status"`
	SenderServiceAccount string                 `json:"senderServiceAccount,omitempty"`
	Timestamp            time.Time              `json:"timestamp"`
	Details              map[string]interface{} `json:"details,omitempty"`
}

// EmailNotificationProvider is the pluggable adapter interface for underlying email transports.
type EmailNotificationProvider interface {
	Name() string
	Send(ctx context.Context, notif *EmailNotification) (*SendResult, error)
	ValidateConfig() error
}

var (
	registryMu      sync.RWMutex
	providers       = make(map[string]EmailNotificationProvider)
	defaultProvider EmailNotificationProvider
)

// normalizeProviderName strips whitespace and lowercases the provider name.
func normalizeProviderName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// RegisterProvider registers a notification provider in the registry.
// Name matching is case-insensitive. Existing registrations under the same name are overwritten.
func RegisterProvider(provider EmailNotificationProvider) {
	if provider == nil || normalizeProviderName(provider.Name()) == "" {
		return
	}
	registryMu.Lock()
	defer registryMu.Unlock()

	key := normalizeProviderName(provider.Name())
	providers[key] = provider
}

// GetProvider retrieves a registered provider by name (case-insensitive).
// Returns ErrProviderNotFound if no provider is registered with that name.
func GetProvider(name string) (EmailNotificationProvider, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	key := normalizeProviderName(name)
	p, ok := providers[key]
	if !ok || p == nil {
		return nil, ErrProviderNotFound
	}
	return p, nil
}

// SetDefaultProvider sets the default provider for notification dispatch and registers it.
func SetDefaultProvider(provider EmailNotificationProvider) {
	if provider == nil || normalizeProviderName(provider.Name()) == "" {
		return
	}
	registryMu.Lock()
	defer registryMu.Unlock()

	key := normalizeProviderName(provider.Name())
	providers[key] = provider
	defaultProvider = provider
}

// GetDefaultProvider returns the active default provider.
// If no default provider has been explicitly set, it falls back to the registered "console" provider,
// or instantiates a new ConsoleEmailProvider if none is registered.
func GetDefaultProvider() EmailNotificationProvider {
	registryMu.RLock()
	if defaultProvider != nil {
		p := defaultProvider
		registryMu.RUnlock()
		return p
	}
	if p, ok := providers["console"]; ok && p != nil {
		registryMu.RUnlock()
		return p
	}
	registryMu.RUnlock()

	return NewConsoleEmailProvider()
}

// InitializeProviders registers default built-in providers (Console and Mock) if not already registered,
// and sets the default provider based on configuration or input parameter.
func InitializeProviders(cfg any) {
	if _, err := GetProvider("console"); err != nil {
		RegisterProvider(NewConsoleEmailProvider())
	}
	if _, err := GetProvider("mock"); err != nil {
		RegisterProvider(NewMockEmailProvider())
	}

	var providerName string
	if cfg != nil {
		switch v := cfg.(type) {
		case string:
			providerName = v
		default:
			val := reflect.ValueOf(cfg)
			if val.Kind() == reflect.Pointer {
				val = val.Elem()
			}
			if val.Kind() == reflect.Struct {
				f := val.FieldByName("EmailServiceType")
				if f.IsValid() && f.Kind() == reflect.String {
					providerName = f.String()
				}
			}
		}
	}

	providerName = normalizeProviderName(providerName)
	if providerName != "" {
		if p, err := GetProvider(providerName); err == nil {
			SetDefaultProvider(p)
			slog.Info("Initialized email notification providers", "defaultProvider", p.Name())
			return
		}
	}

	console, _ := GetProvider("console")
	if console == nil {
		console = NewConsoleEmailProvider()
	}
	SetDefaultProvider(console)
	slog.Info("Initialized email notification providers with fallback default", "defaultProvider", console.Name())
}
