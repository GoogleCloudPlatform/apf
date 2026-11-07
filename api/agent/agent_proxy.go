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

package agent

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
	"google.com/apf/api/integrations"
)

type userSecretsBridge struct {
	UserID               string `json:"userId"`
	FigmaConfigured      bool   `json:"figmaConfigured"`
	LucidchartConfigured bool   `json:"lucidchartConfigured"`
	StitchConfigured     bool   `json:"stitchConfigured"`
	DrawioConfigured     bool   `json:"drawioConfigured"`
	GeminiConfigured     bool   `json:"geminiConfigured"`
	FigmaApiKey          string `json:"figmaApiKey"`
	LucidchartApiKey     string `json:"lucidchartApiKey"`
	StitchApiKey         string `json:"stitchApiKey"`
	DrawioApiKey         string `json:"drawioApiKey"`
	GeminiApiKey         string `json:"geminiApiKey"`
}


// AgentProxy handles communication with the ADK agent.
type AgentProxy struct {
	BaseURL        string
	ProxyPublicKey *rsa.PublicKey
	cfg            *core.Configuration
}

// NewAgentProxy creates a new AgentProxy instance with a specific base URL.
func NewAgentProxy(baseURL string, cfg *core.Configuration) *AgentProxy {
	p := &AgentProxy{BaseURL: baseURL, cfg: cfg}
	pubPEM := os.Getenv("SECRET_ENCRYPTION_PUBLIC_KEY")
	if strings.TrimSpace(pubPEM) == "" {
		isTest := strings.HasSuffix(os.Args[0], ".test") || strings.Contains(os.Args[0], "/_test/") || flag.Lookup("test.v") != nil
		if !isTest {
			panic("SECRET_ENCRYPTION_PUBLIC_KEY environment variable is required")
		}
	} else {
		pubPEM = strings.ReplaceAll(pubPEM, `\n`, "\n")
		block, _ := pem.Decode([]byte(pubPEM))
		if block != nil {
			pub, err := x509.ParsePKIXPublicKey(block.Bytes)
			if err == nil {
				if rsaPub, ok := pub.(*rsa.PublicKey); ok {
					p.ProxyPublicKey = rsaPub
				}
			}
		}
	}
	return p
}

// EncryptSecretsForProxy encrypts the user secrets using standard JWE compact serialization for the detached agent proxy.
func (p *AgentProxy) EncryptSecretsForProxy(secrets *userSecretsBridge) (string, error) {
	if secrets == nil {
		return "", nil
	}
	data, err := json.Marshal(secrets)
	if err != nil {
		return "", err
	}
	if p.ProxyPublicKey == nil {
		return "", errors.New("proxy public key not configured")
	}

	// 1. Generate 32-byte AES CEK
	cek := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, cek); err != nil {
		return "", err
	}

	// 2. Encrypt CEK with RSA-OAEP
	encryptedCEK, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, p.ProxyPublicKey, cek, nil)
	if err != nil {
		return "", err
	}

	// 3. Encrypt data with AES-GCM using CEK
	block, err := aes.NewCipher(cek)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	iv := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	// 4. Construct standard JWE compact serialization (RFC 7516)
	header := []byte(`{"alg":"RSA-OAEP-256","enc":"A256GCM"}`)
	part1 := base64.RawURLEncoding.EncodeToString(header)
	part2 := base64.RawURLEncoding.EncodeToString(encryptedCEK)
	part3 := base64.RawURLEncoding.EncodeToString(iv)

	// GCM Seal appends auth tag to ciphertext. AAD is ASCII of part1.
	sealed := gcm.Seal(nil, iv, data, []byte(part1))
	tagSize := gcm.Overhead()
	actualCiphertext := sealed[:len(sealed)-tagSize]
	authTag := sealed[len(sealed)-tagSize:]

	part4 := base64.RawURLEncoding.EncodeToString(actualCiphertext)
	part5 := base64.RawURLEncoding.EncodeToString(authTag)

	return fmt.Sprintf("%s.%s.%s.%s.%s", part1, part2, part3, part4, part5), nil
}

func (p *AgentProxy) populateAuthAndSecrets(ctx context.Context, targetMap map[string]interface{}, projectId, userId string) error {
	if targetMap == nil {
		return errors.New("targetMap cannot be nil")
	}
	if userId == "" {
		return errors.New("invalid userId: cannot be empty")
	}

	claims, ok := ctx.Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		return errors.New("unauthorized context: missing user claims")
	}
	if claims.Token != "" {
		targetMap["userToken"] = claims.Token
	}

	organizationID, errOrg := core.GetOrganizationID(ctx)
	if errOrg != nil {
		return fmt.Errorf("failed to get organization ID: %w", errOrg)
	}

	// 1. Resolve and update dynamic project/user scoped app credentials (e.g. GitHub) inside state payload
	if projectId != "" {
		cred, errResolve := db.ResolveAppCredential(ctx, organizationID, nil, nil, userId, projectId, "github")
		if errResolve == nil && cred != nil {
			pat := db.GetFieldValue(cred, "pat")
			if pat != "" {
				var stateMap map[string]interface{}
				if existingState, exists := targetMap["state"]; exists {
					if sm, ok := existingState.(map[string]interface{}); ok {
						stateMap = sm
					}
				}
				if stateMap == nil {
					stateMap = map[string]interface{}{}
					targetMap["state"] = stateMap
				}
				stateMap["repoAccessToken"] = pat
				slog.Info("Dynamically updated GORM session state payload with active GitHub credentials", "projectId", projectId)
			}
		}
	}

	// 2. Resolve and bridge dynamic app credentials back to the legacy JWE compact secrets structure
	secrets := &userSecretsBridge{UserID: userId}

	if cred, err := db.ResolveAppCredential(ctx, organizationID, nil, nil, userId, projectId, "figma"); err == nil && cred != nil {
		if val := db.GetFieldValue(cred, "accessToken"); val != "" {
			secrets.FigmaApiKey = val
			secrets.FigmaConfigured = true
		}
	}
	if cred, err := db.ResolveAppCredential(ctx, organizationID, nil, nil, userId, projectId, "lucidchart"); err == nil && cred != nil {
		if val := db.GetFieldValue(cred, "accessToken"); val != "" {
			secrets.LucidchartApiKey = val
			secrets.LucidchartConfigured = true
		}
	}
	if cred, err := db.ResolveAppCredential(ctx, organizationID, nil, nil, userId, projectId, "stitch"); err == nil && cred != nil {
		if val := db.GetFieldValue(cred, "accessToken"); val != "" {
			secrets.StitchApiKey = val
			secrets.StitchConfigured = true
		}
	}
	if cred, err := db.ResolveAppCredential(ctx, organizationID, nil, nil, userId, projectId, "drawio"); err == nil && cred != nil {
		if val := db.GetFieldValue(cred, "accessToken"); val != "" {
			secrets.DrawioApiKey = val
			secrets.DrawioConfigured = true
		}
	}
	if cred, err := db.ResolveAppCredential(ctx, organizationID, nil, nil, userId, projectId, "gemini"); err == nil && cred != nil {
		val := db.GetFieldValue(cred, "gemini_api_key")
		if val == "" {
			val = db.GetFieldValue(cred, "accessToken")
		}
		if val != "" {
			secrets.GeminiApiKey = val
			secrets.GeminiConfigured = true
		}
	}

	if !secrets.GeminiConfigured {
		return fmt.Errorf("GEMINI_API_KEY_REQUIRED: Gemini API key is required. Please configure your key in User Secrets Settings or Project Settings.")
	}

	if secrets.FigmaConfigured || secrets.LucidchartConfigured || secrets.StitchConfigured || secrets.DrawioConfigured || secrets.GeminiConfigured {
		encryptedSecrets, err := p.EncryptSecretsForProxy(secrets)
		if err != nil {
			return fmt.Errorf("failed to encrypt user secrets for proxy: %w", err)
		}
		targetMap["userSecretsEncrypted"] = encryptedSecrets
		var stateMap map[string]interface{}
		if existingState, exists := targetMap["state"]; exists && existingState != nil {
			if sm, ok := existingState.(map[string]interface{}); ok {
				stateMap = sm
			}
		}
		if stateMap == nil {
			stateMap = map[string]interface{}{}
			targetMap["state"] = stateMap
		}
		stateMap["userSecretsEncrypted"] = encryptedSecrets
	}
	return nil
}

// GetJWESecretsForUser resolves current project-scoped or global AppCredentials and packages them into the JWE payload.
func (p *AgentProxy) GetJWESecretsForUser(ctx context.Context, projectId, userId string) (string, error) {
	if userId == "" {
		return "", errors.New("userId is required")
	}

	organizationID, errOrg := core.GetOrganizationID(ctx)
	if errOrg != nil {
		return "", fmt.Errorf("failed to get organization ID: %w", errOrg)
	}

	secrets := &userSecretsBridge{UserID: userId}

	if cred, err := db.ResolveAppCredential(ctx, organizationID, nil, nil, userId, projectId, "figma"); err == nil && cred != nil {
		if val := db.GetFieldValue(cred, "accessToken"); val != "" {
			secrets.FigmaApiKey = val
			secrets.FigmaConfigured = true
		}
	}
	if cred, err := db.ResolveAppCredential(ctx, organizationID, nil, nil, userId, projectId, "lucidchart"); err == nil && cred != nil {
		if val := db.GetFieldValue(cred, "accessToken"); val != "" {
			secrets.LucidchartApiKey = val
			secrets.LucidchartConfigured = true
		}
	}
	if cred, err := db.ResolveAppCredential(ctx, organizationID, nil, nil, userId, projectId, "stitch"); err == nil && cred != nil {
		if val := db.GetFieldValue(cred, "accessToken"); val != "" {
			secrets.StitchApiKey = val
			secrets.StitchConfigured = true
		}
	}
	if cred, err := db.ResolveAppCredential(ctx, organizationID, nil, nil, userId, projectId, "drawio"); err == nil && cred != nil {
		if val := db.GetFieldValue(cred, "accessToken"); val != "" {
			secrets.DrawioApiKey = val
			secrets.DrawioConfigured = true
		}
	}
	if cred, err := db.ResolveAppCredential(ctx, organizationID, nil, nil, userId, projectId, "gemini"); err == nil && cred != nil {
		val := db.GetFieldValue(cred, "gemini_api_key")
		if val == "" {
			val = db.GetFieldValue(cred, "accessToken")
		}
		if val != "" {
			secrets.GeminiApiKey = val
			secrets.GeminiConfigured = true
		}
	}


	if !secrets.FigmaConfigured && !secrets.LucidchartConfigured && !secrets.StitchConfigured && !secrets.DrawioConfigured && !secrets.GeminiConfigured {
		return "", nil
	}

	return p.EncryptSecretsForProxy(secrets)
}


// CreateSession initializes a session with the agent.
func (p *AgentProxy) CreateSession(ctx context.Context, appName, sessionId, projectId, userId, organizationID string) error {
	initURL := fmt.Sprintf("%s/api/apps/%s/users/%s/sessions/%s", p.BaseURL, url.PathEscape(appName), url.PathEscape(userId), url.PathEscape(sessionId))

	statePayload, err := p.buildSessionStatePayload(ctx, projectId, userId, organizationID)
	if err != nil {
		return err
	}

	initPayload := map[string]interface{}{
		"state": statePayload,
	}
	initBytes, err := json.Marshal(initPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal init payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", initURL, bytes.NewReader(initBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	if bytes.Contains(bodyBytes, []byte(fmt.Sprintf("session %s already exists", sessionId))) {
		return nil
	}

	slog.Error("Unexpected error creating session", "code", resp.StatusCode, "body", string(bodyBytes))
	return fmt.Errorf("unexpected error creating session: Body: %s", string(bodyBytes))
}

func (p *AgentProxy) buildSessionStatePayload(ctx context.Context, projectId, userId, organizationID string) (map[string]interface{}, error) {
	statePayload := map[string]interface{}{
		"projectId":      projectId,
		"organizationId": organizationID,
	}

	jweSecrets, errJWE := p.GetJWESecretsForUser(ctx, projectId, userId)
	if errJWE == nil && jweSecrets != "" {
		statePayload["userSecretsEncrypted"] = jweSecrets
	}

	accessToken, errToken := integrations.ResolveGitHubToken(ctx, userId, projectId)
	if errToken == nil && accessToken != "" {
		statePayload["repoAccessToken"] = accessToken
	}

	isReadOnly := false
	if db.UserProjectStoreInstance != nil {
		userProj, errUserProj := db.UserProjectStoreInstance.GetProjectAppSetting(ctx, organizationID, userId, projectId, "github")
		if errUserProj == nil && userProj != nil {
			isReadOnly = userProj.IsReadOnly
		}
	}
	statePayload["isReadOnly"] = isReadOnly

	if db.ProjectStoreInstance != nil {
		project, errProj := db.ProjectStoreInstance.GetProject(ctx, organizationID, projectId)
		if errProj != nil {
			return nil, fmt.Errorf("failed to get project: %w", errProj)
		}

		baseBranch := "main"
		if project.BaseBranch != nil && *project.BaseBranch != "" {
			baseBranch = *project.BaseBranch
		}
		statePayload["baseBranch"] = baseBranch
		statePayload["specBranchPrefix"] = project.GetSpecBranchPrefix()
		statePayload["gitEmail"] = project.GetGitHubEmail()

		statePayload["planningArtifactsLocation"] = project.GetPlanningArtifactsLocation()
		statePayload["specLocation"] = project.GetSpecLocation()
		statePayload["implementationArtifactsLocation"] = project.GetImplementationArtifactsLocation()
		statePayload["testingArtifactsLocation"] = project.GetTestingArtifactsLocation()

		projName := ""
		if project.Name != nil {
			projName = *project.Name
		}
		statePayload["projectName"] = projName
	}

	return statePayload, nil
}

// GetSession checks if a session exists with the agent.
func (p *AgentProxy) GetSession(ctx context.Context, appName, sessionId, userId string) (bool, error) {
	u := fmt.Sprintf("%s/api/apps/%s/users/%s/sessions/%s", p.BaseURL, url.PathEscape(appName), url.PathEscape(userId), url.PathEscape(sessionId))
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response body: %w", err)
	}
	if bytes.Contains(bodyBytes, []byte(fmt.Sprintf("session %s not found", sessionId))) {
		return false, nil
	}
	if bytes.Contains(bodyBytes, []byte("record not found")) {
		return false, nil
	}
	slog.Error("Unexpected status code", "code", resp.StatusCode, "body", string(bodyBytes))
	return false, fmt.Errorf("unexpected error getting session: %d, Body: %s", resp.StatusCode, string(bodyBytes))
}

// RunAgent sends a message to the agent and returns the full response.
func (p *AgentProxy) RunAgent(ctx context.Context, appName, sessionId, projectId, userId, userMessage string) ([]byte, error) {
	runPayload := map[string]interface{}{
		"appName":   appName,
		"userId":    userId,
		"sessionId": sessionId,
		"newMessage": map[string]interface{}{
			"role": "user",
			"parts": []map[string]interface{}{
				{"text": userMessage},
			},
		},
	}

	if err := p.populateAuthAndSecrets(ctx, runPayload, projectId, userId); err != nil {
		return nil, fmt.Errorf("failed to populate auth and secrets: %w", err)
	}

	runBytes, err := json.Marshal(runPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal run payload: %w", err)
	}

	agentURL := fmt.Sprintf("%s/api/run", p.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", agentURL, bytes.NewReader(runBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		slog.Error("Unexpected status code in RunAgent", "code", resp.StatusCode, "body", string(bodyBytes))
		if len(bodyBytes) > 0 {
			return nil, fmt.Errorf("%s", string(bodyBytes))
		}
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

type ProxyInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type ProxyMessagePart struct {
	Text             string               `json:"text,omitempty"`
	InlineData       *ProxyInlineData     `json:"inlineData,omitempty"`
	FunctionResponse *db.FunctionResponse `json:"functionResponse,omitempty"`
}

type ProxyNewMessage struct {
	Role  string             `json:"role"`
	Parts []ProxyMessagePart `json:"parts"`
}

func (p *AgentProxy) runStreamingCommon(ctx context.Context, endpoint string, appName, sessionId, projectId, userId string, newMessage ProxyNewMessage) (io.ReadCloser, error) {
	organizationID, errOrg := core.GetOrganizationID(ctx)
	if errOrg != nil {
		return nil, fmt.Errorf("failed to get organization ID: %w", errOrg)
	}

	statePayload, err := p.buildSessionStatePayload(ctx, projectId, userId, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to build session state payload: %w", err)
	}

	runPayload := map[string]interface{}{
		"appName":    appName,
		"userId":     userId,
		"sessionId":  sessionId,
		"newMessage": newMessage,
		"state":      statePayload,
	}

	if err := p.populateAuthAndSecrets(ctx, runPayload, projectId, userId); err != nil {
		return nil, fmt.Errorf("failed to populate auth and secrets: %w", err)
	}

	runBytes, err := json.Marshal(runPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal run payload: %w", err)
	}

	agentURL := fmt.Sprintf("%s%s", p.BaseURL, endpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", agentURL, bytes.NewReader(runBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(respBytes))
	}

	return resp.Body, nil
}

// RunAgentStreamingWithParts sends a multipart message to the agent and returns a reader for the stream.
func (p *AgentProxy) RunAgentStreamingWithParts(ctx context.Context, appName, sessionId, projectId, userId string, parts []ProxyMessagePart) (io.ReadCloser, error) {
	if len(parts) == 0 {
		parts = []ProxyMessagePart{{Text: ""}}
	}
	msg := ProxyNewMessage{
		Role:  "user",
		Parts: parts,
	}
	return p.runStreamingCommon(ctx, "/api/run_sse", appName, sessionId, projectId, userId, msg)
}

// RunAgentStreaming sends a message to the agent and returns a reader for the stream.
func (p *AgentProxy) RunAgentStreaming(ctx context.Context, appName, sessionId, projectId, userId, userMessage string) (io.ReadCloser, error) {
	return p.RunAgentStreamingWithParts(ctx, appName, sessionId, projectId, userId, []ProxyMessagePart{
		{Text: userMessage},
	})
}

