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

package aitesting

import (
	"context"
	"iter"
	"maps"
	"os"
	"time"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/memory"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/toolconfirmation"
	"google.golang.org/genai"
)

type StateInterface interface {
	Get(key string) (interface{}, error)
	Set(key string, value interface{}) error
}

// MockState implements StateInterface and session.State
type MockState struct {
	Data map[string]interface{}
}

func (m *MockState) Get(key string) (interface{}, error) {
	val, ok := m.Data[key]
	if !ok {
		return nil, os.ErrNotExist
	}
	return val, nil
}

func (m *MockState) Set(key string, value interface{}) error {
	if m.Data == nil {
		m.Data = make(map[string]interface{})
	}
	m.Data[key] = value
	return nil
}

func (m *MockState) All() iter.Seq2[string, any] {
	return maps.All(m.Data)
}

// MockCallbackContext implements agent.CallbackContext
type MockCallbackContext struct {
	UserId    string
	SessionId string
	StateObj  *MockState
}

func (m *MockCallbackContext) UserID() string {
	return m.UserId
}

func (m *MockCallbackContext) SessionID() string {
	return m.SessionId
}

func (m *MockCallbackContext) State() session.State {
	return m.StateObj
}

func (m *MockCallbackContext) AgentName() string {
	return "test-agent"
}

func (m *MockCallbackContext) AppName() string {
	return "test-app"
}

func (m *MockCallbackContext) Artifacts() agent.Artifacts {
	return nil
}

func (m *MockCallbackContext) Branch() string {
	return "main"
}

func (m *MockCallbackContext) Deadline() (deadline time.Time, ok bool) {
	return time.Time{}, false
}

func (m *MockCallbackContext) Done() <-chan struct{} {
	return nil
}

func (m *MockCallbackContext) Err() error {
	return nil
}

func (m *MockCallbackContext) Value(key interface{}) interface{} {
	return nil
}

func (m *MockCallbackContext) InvocationID() string {
	return "test-invocation-id"
}

func (m *MockCallbackContext) ReadonlyState() session.ReadonlyState {
	return nil
}

func (m *MockCallbackContext) UserContent() *genai.Content {
	return nil
}

var _ agent.CallbackContext = (*MockCallbackContext)(nil)

// MockToolContext implements tool.Context
type MockToolContext struct {
	*MockCallbackContext
}

func (m *MockToolContext) FunctionCallID() string {
	return "test-function-call-id"
}

func (m *MockToolContext) Actions() *session.EventActions {
	return nil
}

func (m *MockToolContext) SearchMemory(ctx context.Context, query string) (*memory.SearchResponse, error) {
	return nil, nil
}

func (m *MockToolContext) ToolConfirmation() *toolconfirmation.ToolConfirmation {
	return nil
}

func (m *MockToolContext) RequestConfirmation(hint string, payload any) error {
	return nil
}

var _ tool.Context = (*MockToolContext)(nil)
