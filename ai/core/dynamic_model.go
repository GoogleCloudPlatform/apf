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
	"fmt"
	"iter"
	"log/slog"
	"strings"
	"sync"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/genai"
)

// DynamicModel wraps LLM model creation to dynamically resolve the request-scoped
// Gemini API key from userSecretsEncrypted (in session state) per invocation.
type DynamicModel struct {
	modelName string
	cfg       *Configuration
	mu        sync.Mutex
	cache     map[string]model.LLM
}

func NewDynamicModel(cfg *Configuration, modelName string) model.LLM {
	return &DynamicModel{
		modelName: modelName,
		cfg:       cfg,
		cache:     make(map[string]model.LLM),
	}
}

func (m *DynamicModel) Name() string {
	return m.modelName
}

func (m *DynamicModel) getModelForContext(ctx context.Context) (model.LLM, error) {
	var jweEncrypted string

	if invCtx, ok := ctx.(agent.InvocationContext); ok && invCtx.Session() != nil && invCtx.Session().State() != nil {
		if val, err := invCtx.Session().State().Get("userSecretsEncrypted"); err == nil && val != nil {
			if s, ok := val.(string); ok {
				jweEncrypted = s
			}
		}
		if jweEncrypted == "" {
			for k, v := range invCtx.Session().State().All() {
				if strings.EqualFold(k, "userSecretsEncrypted") {
					if s, ok := v.(string); ok {
						jweEncrypted = s
						break
					}
				}
			}
		}
	}

	apiKey := m.cfg.ResolveGeminiAPIKey(jweEncrypted)
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY_REQUIRED: Gemini API key is required. Please configure your key in User Secrets Settings or Project Settings.")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	llmInst, exists := m.cache[apiKey]
	if exists {
		return llmInst, nil
	}

	slog.Debug("Creating request-scoped Gemini model instance", "model", m.modelName, "hasJWE", jweEncrypted != "")
	llmInst, err := gemini.NewModel(context.Background(), m.modelName, &genai.ClientConfig{
		Backend: genai.BackendGeminiAPI,
		APIKey:  apiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create model for key: %w", err)
	}

	m.cache[apiKey] = llmInst
	return llmInst, nil
}

func (m *DynamicModel) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	llmInst, err := m.getModelForContext(ctx)
	if err != nil {
		return func(yield func(*model.LLMResponse, error) bool) {
			yield(nil, err)
		}
	}
	return llmInst.GenerateContent(ctx, req, stream)
}
