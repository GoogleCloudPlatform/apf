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

package bmad

import (
	"fmt"
	"path/filepath"
	"strings"

	"google.com/apf/ai/core"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/plugin"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"
)

var notificationCounter uint64

type SubInterceptor interface {
	Name() string
	Before(ctx tool.Context, tool tool.Tool, args map[string]any) (map[string]any, error)
	After(ctx tool.Context, tool tool.Tool, args, result map[string]any, err error) (map[string]any, error)
	BeforeRun(ctx agent.InvocationContext) error
	AfterRun(ctx agent.InvocationContext)
}

type interceptorContext interface {
	UserID() string
	State() session.State
}

type BaseInterceptor struct{}

func (b *BaseInterceptor) Before(ctx tool.Context, tool tool.Tool, args map[string]any) (map[string]any, error) {
	return nil, nil
}

func (b *BaseInterceptor) After(ctx tool.Context, tool tool.Tool, args, result map[string]any, err error) (map[string]any, error) {
	return result, err
}

func (b *BaseInterceptor) BeforeRun(ctx agent.InvocationContext) error {
	return nil
}

func (b *BaseInterceptor) AfterRun(ctx agent.InvocationContext) {}

type BmadInterceptorPlugin struct {
	cfg          *core.Configuration
	interceptors []SubInterceptor
}

func (p *BmadInterceptorPlugin) ensureInterceptors() {
	if p.interceptors == nil {
		p.interceptors = []SubInterceptor{
			&PathSafetyInterceptor{cfg: p.cfg},
			&ContentSanitizerInterceptor{cfg: p.cfg},
			&FileWriteNotifierInterceptor{cfg: p.cfg},
		}
	}
}

func (p *BmadInterceptorPlugin) BeforeToolCallback(ctx tool.Context, tool tool.Tool, args map[string]any) (map[string]any, error) {
	p.ensureInterceptors()
	for _, interceptor := range p.interceptors {
		res, err := interceptor.Before(ctx, tool, args)
		if err != nil {
			return nil, err
		}
		if res != nil {
			args = res
		}
	}
	return nil, nil
}

func (p *BmadInterceptorPlugin) AfterToolCallback(ctx tool.Context, tool tool.Tool, args, result map[string]any, err error) (map[string]any, error) {
	p.ensureInterceptors()
	var currentResult map[string]any = result
	var currentErr error = err
	for _, interceptor := range p.interceptors {
		res, newErr := interceptor.After(ctx, tool, args, currentResult, currentErr)
		currentResult = res
		currentErr = newErr
	}
	return currentResult, currentErr
}

func (p *BmadInterceptorPlugin) BeforeRunCallback(ctx agent.InvocationContext) (*genai.Content, error) {
	p.ensureInterceptors()
	for _, interceptor := range p.interceptors {
		if err := interceptor.BeforeRun(ctx); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (p *BmadInterceptorPlugin) AfterRunCallback(ctx agent.InvocationContext) {
	p.ensureInterceptors()
	for _, interceptor := range p.interceptors {
		interceptor.AfterRun(ctx)
	}
}

func NewBmadInterceptorPlugin(cfg *core.Configuration) (*plugin.Plugin, error) {
	p := &BmadInterceptorPlugin{
		cfg: cfg,
		interceptors: []SubInterceptor{
			&PathSafetyInterceptor{cfg: cfg},
			&ContentSanitizerInterceptor{cfg: cfg},
			&FileWriteNotifierInterceptor{cfg: cfg},
		},
	}
	return plugin.New(plugin.Config{
		Name:               "bmad_interceptor",
		BeforeRunCallback:  p.BeforeRunCallback,
		BeforeToolCallback: p.BeforeToolCallback,
		AfterToolCallback:  p.AfterToolCallback,
		AfterRunCallback:   p.AfterRunCallback,
	})
}

func (p *BmadInterceptorPlugin) executeWriteIntercept(ctx interceptorContext, repoCfg *core.RepoConfiguration, result map[string]any) (map[string]any, error) {
	p.ensureInterceptors()
	for _, interceptor := range p.interceptors {
		if r, ok := interceptor.(*FileWriteNotifierInterceptor); ok {
			return r.executeWriteIntercept(ctx, repoCfg, result)
		}
	}
	return nil, fmt.Errorf("FileWriteNotifierInterceptor not registered")
}

// ==========================================
// Shared Helpers
// ==========================================
func isPathUnderAllowedDirs(relPath string, dirs ...string) bool {
	cleanedRel := filepath.Clean(relPath)
	for _, d := range dirs {
		if d == "" {
			continue
		}
		cleanDir := filepath.Clean(d)
		if cleanedRel == cleanDir || strings.HasPrefix(cleanedRel, cleanDir+string(filepath.Separator)) {
			return true
		}
	}
	return false
}
