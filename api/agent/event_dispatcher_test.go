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
	"testing"
)

func TestEventDispatcher_CustomRegistration(t *testing.T) {
	dispatcher := NewEventDispatcher()

	customMatched := false
	customHandled := false

	customStrategy := FuncEventHandler{
		Name: "CustomTestEvent",
		MatchFunc: func(event ADKEvent) bool {
			customMatched = (event.Author == "CUSTOM_TEST_BOT")
			return customMatched
		},
		HandleFunc: func(ctx context.Context, event ADKEvent, eventCtx EventContext) bool {
			customHandled = true
			return true
		},
	}

	dispatcher.Register(customStrategy)

	testEvent := ADKEvent{
		Author: "CUSTOM_TEST_BOT",
	}

	eventCtx := EventContext{
		ProjectID: "test-proj",
	}

	handled := dispatcher.Dispatch(context.Background(), testEvent, eventCtx)

	if !handled {
		t.Errorf("Expected dispatcher to return handled=true for registered custom event")
	}
	if !customMatched {
		t.Errorf("Expected custom MatchFunc to be triggered")
	}
	if !customHandled {
		t.Errorf("Expected custom HandleFunc to be executed")
	}
}

func TestEventDispatcher_ShortCircuitPriority(t *testing.T) {
	dispatcher := NewEventDispatcher()

	firstCalled := false
	secondCalled := false

	dispatcher.Register(FuncEventHandler{
		Name: "FirstStopHandler",
		MatchFunc: func(event ADKEvent) bool {
			return true
		},
		HandleFunc: func(ctx context.Context, event ADKEvent, eventCtx EventContext) bool {
			firstCalled = true
			return true // stop propagation
		},
	})

	dispatcher.Register(FuncEventHandler{
		Name: "SecondHandler",
		MatchFunc: func(event ADKEvent) bool {
			return true
		},
		HandleFunc: func(ctx context.Context, event ADKEvent, eventCtx EventContext) bool {
			secondCalled = true
			return false
		},
	})

	dispatcher.Dispatch(context.Background(), ADKEvent{}, EventContext{})

	if !firstCalled {
		t.Errorf("Expected first handler to be called")
	}
	if secondCalled {
		t.Errorf("Expected second handler NOT to be called due to short-circuit return true")
	}
}
