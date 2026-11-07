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
	"fmt"
	"sync"
	"testing"
)

func resetRegistryForTest() {
	registryMu.Lock()
	defer registryMu.Unlock()
	providers = make(map[string]EmailNotificationProvider)
	defaultProvider = nil
}

func TestProviderInterfaceCompliance(t *testing.T) {
	var _ EmailNotificationProvider = (*ConsoleEmailProvider)(nil)
	var _ EmailNotificationProvider = (*MockEmailProvider)(nil)
}

func TestRegistry_RegisterAndGetProvider(t *testing.T) {
	resetRegistryForTest()
	defer resetRegistryForTest()

	console := NewConsoleEmailProvider()
	RegisterProvider(console)

	// Direct retrieval
	p, err := GetProvider("console")
	if err != nil {
		t.Fatalf("expected to get provider 'console', got error: %v", err)
	}
	if p.Name() != "console" {
		t.Fatalf("expected provider name 'console', got: %s", p.Name())
	}

	// Case-insensitivity & whitespace trimming
	variants := []string{"CONSOLE", "  console  ", "Console", " cOnSoLe "}
	for _, v := range variants {
		found, err := GetProvider(v)
		if err != nil {
			t.Errorf("GetProvider(%q) returned error: %v", v, err)
		}
		if found != console {
			t.Errorf("GetProvider(%q) returned unexpected instance", v)
		}
	}

	// Unknown provider
	_, err = GetProvider("unknown-transport")
	if !errors.Is(err, ErrProviderNotFound) {
		t.Fatalf("expected ErrProviderNotFound, got: %v", err)
	}

	// Registering nil provider is a no-op
	RegisterProvider(nil)
}

func TestRegistry_OverwriteProvider(t *testing.T) {
	resetRegistryForTest()
	defer resetRegistryForTest()

	mock1 := NewMockEmailProvider()
	mock2 := NewMockEmailProvider()

	RegisterProvider(mock1)
	p, err := GetProvider("mock")
	if err != nil || p != mock1 {
		t.Fatalf("expected mock1 registered")
	}

	// Overwrite with mock2
	RegisterProvider(mock2)
	p, err = GetProvider("mock")
	if err != nil || p != mock2 {
		t.Fatalf("expected mock2 to overwrite mock1")
	}
}

func TestRegistry_SetAndGetDefaultProvider(t *testing.T) {
	resetRegistryForTest()
	defer resetRegistryForTest()

	// Default fallback with empty registry
	fallback := GetDefaultProvider()
	if fallback == nil || fallback.Name() != "console" {
		t.Fatalf("expected fallback default provider 'console', got: %v", fallback)
	}

	mock := NewMockEmailProvider()
	SetDefaultProvider(mock)

	def := GetDefaultProvider()
	if def != mock {
		t.Fatalf("expected default provider to be mock instance")
	}

	// Setting nil default should not clear existing default
	SetDefaultProvider(nil)
	if GetDefaultProvider() != mock {
		t.Fatalf("expected default provider to remain mock after nil SetDefaultProvider")
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	resetRegistryForTest()
	defer resetRegistryForTest()

	console := NewConsoleEmailProvider()
	mock := NewMockEmailProvider()
	RegisterProvider(console)
	RegisterProvider(mock)
	SetDefaultProvider(console)

	var wg sync.WaitGroup
	const concurrency = 50

	for i := 0; i < concurrency; i++ {
		wg.Add(3)

		go func(idx int) {
			defer wg.Done()
			_, _ = GetProvider("console")
		}(i)

		go func(idx int) {
			defer wg.Done()
			_ = GetDefaultProvider()
		}(i)

		go func(idx int) {
			defer wg.Done()
			if idx%2 == 0 {
				RegisterProvider(NewMockEmailProvider())
			} else {
				SetDefaultProvider(console)
			}
		}(i)
	}

	wg.Wait()
}

type sampleConfig struct {
	EmailServiceType string
}

func TestInitializeProviders(t *testing.T) {
	resetRegistryForTest()
	defer resetRegistryForTest()

	// 1. With nil cfg -> defaults to console
	InitializeProviders(nil)
	if GetDefaultProvider().Name() != "console" {
		t.Fatalf("expected default provider 'console' for nil config")
	}

	// 2. With string "mock"
	InitializeProviders("mock")
	if GetDefaultProvider().Name() != "mock" {
		t.Fatalf("expected default provider 'mock' for string 'mock'")
	}

	// 3. With struct config
	cfg := sampleConfig{EmailServiceType: "CONSOLE"}
	InitializeProviders(cfg)
	if GetDefaultProvider().Name() != "console" {
		t.Fatalf("expected default provider 'console' for struct config")
	}

	// 4. With pointer to struct config
	cfgPtr := &sampleConfig{EmailServiceType: "mock"}
	InitializeProviders(cfgPtr)
	if GetDefaultProvider().Name() != "mock" {
		t.Fatalf("expected default provider 'mock' for pointer config")
	}

	// 5. With unknown config value -> fallback to console
	InitializeProviders("unknown_custom_provider")
	if GetDefaultProvider().Name() != "console" {
		t.Fatalf("expected fallback to 'console' for unknown provider type")
	}
}

func TestConsoleEmailProvider_Send(t *testing.T) {
	provider := NewConsoleEmailProvider()
	if provider.Name() != "console" {
		t.Fatalf("expected provider name 'console', got: %s", provider.Name())
	}
	if err := provider.ValidateConfig(); err != nil {
		t.Fatalf("expected ValidateConfig to succeed, got: %v", err)
	}

	// Nil notification error
	_, err := provider.Send(context.Background(), nil)
	if err == nil {
		t.Fatalf("expected error sending nil notification")
	}

	// Standard notification
	notif := &EmailNotification{
		ToEmail:     "user@example.com",
		ToName:      "Test User",
		Subject:     "Welcome to APF",
		HTMLContent: "<h1>Hello</h1>",
		TextContent: "Hello world!",
		RequestID:   "req-12345",
	}

	res, err := provider.Send(context.Background(), notif)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != SendStatusSuccess {
		t.Fatalf("expected status %s, got %s", SendStatusSuccess, res.Status)
	}
	if res.Provider != "console" {
		t.Fatalf("expected provider 'console', got: %s", res.Provider)
	}
	if res.MessageID != "console-req-12345" {
		t.Fatalf("expected messageID 'console-req-12345', got: %s", res.MessageID)
	}
	if res.Timestamp.IsZero() {
		t.Fatalf("expected non-zero timestamp")
	}

	// Long text content truncation (> 200 runes)
	longText := ""
	for i := 0; i < 300; i++ {
		longText += "A"
	}
	longNotif := &EmailNotification{
		ToEmail:     "user@example.com",
		Subject:     "Long Text",
		TextContent: longText,
	}
	res2, err := provider.Send(context.Background(), longNotif)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	preview, ok := res2.Details["preview"].(string)
	if !ok {
		t.Fatalf("expected preview string in details")
	}
	if len([]rune(preview)) != 200 {
		t.Fatalf("expected truncated preview of 200 runes, got %d", len([]rune(preview)))
	}

	// Multibyte unicode truncation
	unicodeText := ""
	for i := 0; i < 250; i++ {
		unicodeText += "🚀"
	}
	unicodeNotif := &EmailNotification{
		ToEmail:     "user@example.com",
		Subject:     "Unicode",
		TextContent: unicodeText,
	}
	res3, err := provider.Send(context.Background(), unicodeNotif)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	unicodePreview := res3.Details["preview"].(string)
	if len([]rune(unicodePreview)) != 200 {
		t.Fatalf("expected 200 runes for unicode preview, got: %d", len([]rune(unicodePreview)))
	}
}

func TestMockEmailProvider_SendAndTrack(t *testing.T) {
	mock := NewMockEmailProvider()
	if mock.Name() != "mock" {
		t.Fatalf("expected provider name 'mock', got: %s", mock.Name())
	}
	if err := mock.ValidateConfig(); err != nil {
		t.Fatalf("expected ValidateConfig to succeed, got: %v", err)
	}

	// Nil notification
	_, err := mock.Send(context.Background(), nil)
	if err == nil {
		t.Fatalf("expected error sending nil notification")
	}

	// Send notifications
	notif1 := &EmailNotification{
		ToEmail:   "alpha@example.com",
		Subject:   "Notification 1",
		RequestID: "req-1",
	}
	notif2 := &EmailNotification{
		ToEmail:   "beta@example.com",
		Subject:   "Notification 2",
		RequestID: "req-2",
	}

	res1, err := mock.Send(context.Background(), notif1)
	if err != nil || res1.Status != SendStatusSuccess {
		t.Fatalf("failed to send notif1: %v", err)
	}
	if res1.MessageID != "mock-req-1" {
		t.Fatalf("expected messageID 'mock-req-1', got: %s", res1.MessageID)
	}

	res2, err := mock.Send(context.Background(), notif2)
	if err != nil || res2.Status != SendStatusSuccess {
		t.Fatalf("failed to send notif2: %v", err)
	}

	sent := mock.SentNotifications()
	if len(sent) != 2 {
		t.Fatalf("expected 2 sent notifications, got: %d", len(sent))
	}
	if sent[0].ToEmail != "alpha@example.com" || sent[1].ToEmail != "beta@example.com" {
		t.Fatalf("unexpected sent notifications content: %v", sent)
	}

	// Verify slice copy immutability
	sent[0] = nil
	if mock.SentNotifications()[0] == nil {
		t.Fatalf("expected SentNotifications to return an independent slice copy")
	}

	// Error simulation
	injectedErr := errors.New("simulated network failure")
	mock.SetError(injectedErr)

	_, err = mock.Send(context.Background(), &EmailNotification{ToEmail: "gamma@example.com"})
	if !errors.Is(err, injectedErr) {
		t.Fatalf("expected injected error, got: %v", err)
	}

	// Clear resets sent history and injected error
	mock.Clear()
	if len(mock.SentNotifications()) != 0 {
		t.Fatalf("expected 0 notifications after Clear()")
	}

	res3, err := mock.Send(context.Background(), &EmailNotification{ToEmail: "gamma@example.com"})
	if err != nil || res3.Status != SendStatusSuccess {
		t.Fatalf("expected successful send after Clear(), got error: %v", err)
	}
}

func TestMockEmailProvider_Concurrency(t *testing.T) {
	mock := NewMockEmailProvider()
	const count = 100

	var wg sync.WaitGroup
	wg.Add(count)

	for i := 0; i < count; i++ {
		go func(idx int) {
			defer wg.Done()
			_, _ = mock.Send(context.Background(), &EmailNotification{
				ToEmail:   fmt.Sprintf("user%d@example.com", idx),
				Subject:   fmt.Sprintf("Subject %d", idx),
				RequestID: fmt.Sprintf("req-%d", idx),
			})
		}(i)
	}

	wg.Wait()

	sent := mock.SentNotifications()
	if len(sent) != count {
		t.Fatalf("expected %d sent notifications, got %d", count, len(sent))
	}
}
