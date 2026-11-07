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

package integrations

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"google.com/apf/api/db"
)

// GitHubIntegration implements AppIntegration for GitHub
type GitHubIntegration struct{}

func init() {
	Register("GitHubIntegration", &GitHubIntegration{})
}

func (g *GitHubIntegration) Validate(ctx context.Context, cred *db.AppCredential) (bool, error) {
	token := getFieldValue(cred, "pat")
	if token == "" {
		return false, fmt.Errorf("personal access token is missing")
	}

	// Support simulated credentials for offline/local environment
	if IsSimulatedToken(token) {
		return true, nil
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("GitHub status check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return false, nil
	}
	return false, fmt.Errorf("GitHub returned unexpected status code: %d", resp.StatusCode)
}

// FigmaIntegration implements AppIntegration for Figma
type FigmaIntegration struct{}

func init() {
	Register("FigmaIntegration", &FigmaIntegration{})
}

func (f *FigmaIntegration) Validate(ctx context.Context, cred *db.AppCredential) (bool, error) {
	token := getFieldValue(cred, "accessToken")
	if token == "" {
		return false, fmt.Errorf("access token is missing")
	}

	if strings.HasPrefix(token, "simulated_") || token == "mock-token" {
		return true, nil
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.figma.com/v1/me", nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("X-Figma-Token", token)

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("Figma check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return false, nil
	}
	return false, fmt.Errorf("Figma returned status code: %d", resp.StatusCode)
}

// LucidchartIntegration implements AppIntegration for Lucidchart
type LucidchartIntegration struct{}

func init() {
	Register("LucidchartIntegration", &LucidchartIntegration{})
}

func (l *LucidchartIntegration) Validate(ctx context.Context, cred *db.AppCredential) (bool, error) {
	token := getFieldValue(cred, "accessToken")
	if token == "" {
		return false, fmt.Errorf("access token is missing")
	}

	if strings.HasPrefix(token, "simulated_") || token == "mock-token" {
		return true, nil
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.lucid.co/users/me", nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("Lucidchart check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return false, nil
	}
	return false, fmt.Errorf("Lucidchart returned status code: %d", resp.StatusCode)
}

// StitchIntegration implements AppIntegration for Stitch
type StitchIntegration struct{}

func init() {
	Register("StitchIntegration", &StitchIntegration{})
}

func (s *StitchIntegration) Validate(ctx context.Context, cred *db.AppCredential) (bool, error) {
	token := getFieldValue(cred, "accessToken")
	if token == "" {
		return false, fmt.Errorf("access token is missing")
	}
	// Simulate Stitch success for now
	return true, nil
}

// DrawIOIntegration implements AppIntegration for Draw.io
type DrawIOIntegration struct{}

func init() {
	Register("DrawIOIntegration", &DrawIOIntegration{})
}

func (d *DrawIOIntegration) Validate(ctx context.Context, cred *db.AppCredential) (bool, error) {
	token := getFieldValue(cred, "accessToken")
	if token == "" {
		return false, fmt.Errorf("access token is missing")
	}
	// Simulate Draw.io success for now
	return true, nil
}

// GeminiIntegration implements AppIntegration for Gemini AI
type GeminiIntegration struct{}

func init() {
	Register("GeminiIntegration", &GeminiIntegration{})
}

func (g *GeminiIntegration) Validate(ctx context.Context, cred *db.AppCredential) (bool, error) {
	key := getFieldValue(cred, "gemini_api_key")
	if key == "" {
		key = getFieldValue(cred, "accessToken")
	}
	if key == "" {
		return false, fmt.Errorf("Gemini API key is missing")
	}

	if strings.HasPrefix(key, "gemini_simulated_") || strings.HasPrefix(key, "simulated_") || key == "mock-token" || key == "dummy-key" {
		return true, nil
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://generativelanguage.googleapis.com/v1beta/models?key="+key, nil)
	if err != nil {
		return false, err
	}

	resp, err := client.Do(req)
	if err != nil {
		slog.Error("Gemini API check request failed", "error", err)
		return false, fmt.Errorf("Gemini API check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		slog.Info("Gemini API key validation successful")
		return true, nil
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	respBody := strings.TrimSpace(string(bodyBytes))
	slog.Warn("Gemini API key validation failed", "statusCode", resp.StatusCode, "responseBody", respBody)

	if respBody != "" {
		return false, fmt.Errorf("Gemini API returned status code %d: %s", resp.StatusCode, respBody)
	}
	return false, fmt.Errorf("Gemini API returned status code %d", resp.StatusCode)
}

