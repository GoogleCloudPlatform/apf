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
	"context"
	"sync"
)

// ADKEventHandler defines the strategy interface for matching and handling ADK events.
type ADKEventHandler interface {
	Match(event ADKEvent) bool
	Handle(ctx context.Context, event ADKEvent, eventCtx EventContext) bool
}

// FuncEventHandler is a convenient struct implementation of ADKEventHandler using function hooks.
type FuncEventHandler struct {
	Name       string
	MatchFunc  func(event ADKEvent) bool
	HandleFunc func(ctx context.Context, event ADKEvent, eventCtx EventContext) bool
}

// Match returns true if the event satisfies the handler's MatchFunc.
func (h FuncEventHandler) Match(event ADKEvent) bool {
	return h.MatchFunc != nil && h.MatchFunc(event)
}

// Handle executes the handler's HandleFunc and returns true if propagation should stop.
func (h FuncEventHandler) Handle(ctx context.Context, event ADKEvent, eventCtx EventContext) bool {
	if h.HandleFunc != nil {
		return h.HandleFunc(ctx, event, eventCtx)
	}
	return false
}

// EventDispatcher manages a registered list of ADKEventHandlers.
type EventDispatcher struct {
	mu       sync.RWMutex
	handlers []ADKEventHandler
}

// NewEventDispatcher creates an empty EventDispatcher.
func NewEventDispatcher() *EventDispatcher {
	return &EventDispatcher{
		handlers: make([]ADKEventHandler, 0),
	}
}

// Register appends a new handler strategy to the dispatcher.
func (d *EventDispatcher) Register(handler ADKEventHandler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handlers = append(d.handlers, handler)
}

// Dispatch executes matching handlers for the given event.
// Returns true if at least one handler matched.
func (d *EventDispatcher) Dispatch(ctx context.Context, event ADKEvent, eventCtx EventContext) bool {
	d.mu.RLock()
	handlers := d.handlers
	d.mu.RUnlock()

	handled := false
	for _, h := range handlers {
		if h.Match(event) {
			handled = true
			stop := h.Handle(ctx, event, eventCtx)
			if stop {
				break
			}
		}
	}
	return handled
}

// defaultEventDispatcher creates an EventDispatcher preconfigured with standard ADK event strategies.
func defaultEventDispatcher() *EventDispatcher {
	d := NewEventDispatcher()

	d.Register(FuncEventHandler{
		Name: "UserInput",
		MatchFunc: func(event ADKEvent) bool {
			return event.Author == SenderUser
		},
		HandleFunc: func(ctx context.Context, event ADKEvent, eventCtx EventContext) bool {
			handleUserInput(ctx, event, eventCtx)
			return true
		},
	})

	d.Register(FuncEventHandler{
		Name: "FinalTextResponse",
		MatchFunc: func(event ADKEvent) bool {
			return event.Author != SenderUser && !event.Partial && hasText(event)
		},
		HandleFunc: func(ctx context.Context, event ADKEvent, eventCtx EventContext) bool {
			handleAgentFinalTextResponse(ctx, event, eventCtx)
			return false
		},
	})

	d.Register(FuncEventHandler{
		Name: "StreamingTextResponse",
		MatchFunc: func(event ADKEvent) bool {
			return event.Author != SenderUser && event.Partial && !event.TurnComplete && hasText(event)
		},
		HandleFunc: func(ctx context.Context, event ADKEvent, eventCtx EventContext) bool {
			handleAgentStreamingTextResponse(ctx, event, eventCtx)
			return false
		},
	})

	d.Register(FuncEventHandler{
		Name: "StreamingThoughtResponse",
		MatchFunc: func(event ADKEvent) bool {
			return event.Author != SenderUser && event.Partial && !event.TurnComplete && hasThought(event)
		},
		HandleFunc: func(ctx context.Context, event ADKEvent, eventCtx EventContext) bool {
			handleAgentStreamingThoughtResponse(ctx, event, eventCtx)
			return false
		},
	})

	d.Register(FuncEventHandler{
		Name: "FileWriteNotification",
		MatchFunc: func(event ADKEvent) bool {
			return event.Author != SenderUser && hasFileWriteNotification(event)
		},
		HandleFunc: func(ctx context.Context, event ADKEvent, eventCtx EventContext) bool {
			handleFileWriteNotification(ctx, event, eventCtx)
			return false
		},
	})

	d.Register(FuncEventHandler{
		Name: "ToolCall",
		MatchFunc: func(event ADKEvent) bool {
			return event.Author != SenderUser && hasFunctionCall(event)
		},
		HandleFunc: func(ctx context.Context, event ADKEvent, eventCtx EventContext) bool {
			handleToolCallRequest(ctx, event, eventCtx)
			return true
		},
	})

	d.Register(FuncEventHandler{
		Name: "ToolResult",
		MatchFunc: func(event ADKEvent) bool {
			return event.Author != SenderUser && !hasFunctionCall(event) && hasFunctionResponse(event)
		},
		HandleFunc: func(ctx context.Context, event ADKEvent, eventCtx EventContext) bool {
			handleToolResultProvided(ctx, event, eventCtx)
			return true
		},
	})

	d.Register(FuncEventHandler{
		Name: "StateArtifactUpdate",
		MatchFunc: func(event ADKEvent) bool {
			return event.Author != SenderUser && !hasFunctionCall(event) && !hasFunctionResponse(event) &&
				hasStateOrArtifactDelta(event) && event.Content == nil
		},
		HandleFunc: func(ctx context.Context, event ADKEvent, eventCtx EventContext) bool {
			handleStateArtifactUpdateOnly(ctx, event, eventCtx)
			return true
		},
	})

	d.Register(FuncEventHandler{
		Name: "TransferSignal",
		MatchFunc: func(event ADKEvent) bool {
			return event.Author != SenderUser && !hasFunctionCall(event) && !hasFunctionResponse(event) &&
				!(hasStateOrArtifactDelta(event) && event.Content == nil) && hasTransferSignal(event)
		},
		HandleFunc: func(ctx context.Context, event ADKEvent, eventCtx EventContext) bool {
			handleAgentTransferSignal(ctx, event, eventCtx)
			return true
		},
	})

	d.Register(FuncEventHandler{
		Name: "EscalationSignal",
		MatchFunc: func(event ADKEvent) bool {
			return event.Author != SenderUser && !hasFunctionCall(event) && !hasFunctionResponse(event) &&
				!(hasStateOrArtifactDelta(event) && event.Content == nil) && !hasTransferSignal(event) &&
				hasEscalationSignal(event)
		},
		HandleFunc: func(ctx context.Context, event ADKEvent, eventCtx EventContext) bool {
			handleLoopEscalationSignal(ctx, event, eventCtx)
			return true
		},
	})

	return d
}

// DefaultDispatcher is the standard package-level event dispatcher.
var DefaultDispatcher = defaultEventDispatcher()
