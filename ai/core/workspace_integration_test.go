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
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

// curl -X POST http://localhost:8080/api/apps/APF%20Analysis%20Agent/users/user/sessions/s_123 -H "Content-Type: application/json"

// curl -X POST http://localhost:8080/api/run \
// -H "Content-Type: application/json" \
// -d '{
// "appName": "APF Analysis Agent",
// "userId": "user",
// "sessionId": "s_123",
// "newMessage": {
//     "role": "user",
//     "parts": [{
//     "text": "Design Thinking"
//     }]
// }
// }'

func create_session(t *testing.T, url string, projectId string) (string, error) {

	data := map[string]interface{}{
		"state": map[string]interface{}{
			"projectId": projectId,
		},
	}
	// 3. Marshal the data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		t.Errorf("Error marshaling JSON: %v\n", err)
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		t.Errorf("Error creating request: %v\n", err)
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("Error sending request: %v\n", err)
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Error reading response body: %v\n", err)
		return "", err
	}

	// Parse the response body to get the session ID
	var sessionResponse map[string]interface{}
	if err := json.Unmarshal([]byte(body), &sessionResponse); err != nil {
		t.Errorf("Error unmarshaling response body: %v\n", err)
		return "", err
	}
	return sessionResponse["id"].(string), nil
}

func send_message(t *testing.T, url string, userId string, sessionId string, message string) (string, error) {

	data := map[string]interface{}{
		"appName":   "APF Analysis Agent",
		"userId":    userId,
		"sessionId": sessionId,
		"newMessage": map[string]interface{}{
			"role": "user",
			"parts": map[string]interface{}{
				"text": message,
			},
		},
	}
	// 3. Marshal the data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		t.Errorf("Error marshaling JSON: %v\n", err)
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		t.Errorf("Error creating request: %v\n", err)
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("Error sending request: %v\n", err)
		return "", err
	}
	defer resp.Body.Close()

	new_body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Error reading response body: %v\n", err)
		return "", err
	}

	return string(new_body), nil
}

func isServerOnline(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func TestCreateSession(t *testing.T) {
	if !isServerOnline("localhost:8089") {
		t.Skip("Skipping integration test: server at localhost:8089 is offline")
	}

	rootUrl := "http://localhost:8089/api"
	userId := "u_123"
	projectId := "pid_123"
	sessionId := uuid.New().String()
	appId := "APF%20Analysis%20Agent"
	url := rootUrl + "/apps/" + appId + "/users/" + userId + "/sessions/" + sessionId

	sessionId, err := create_session(t, url, projectId)
	if err != nil {
		t.Errorf("Error creating session: %v\n", err)
		return
	}

	t.Log("Session ID: ", sessionId)
}

func TestAssetSyncPluginBeforeAgentCallback(t *testing.T) {
	if !isServerOnline("localhost:8089") {
		t.Skip("Skipping integration test: server at localhost:8089 is offline")
	}

	rootUrl := "http://localhost:8089/api"
	userId := "u_123"
	projectId := "p_123"
	sessionId := uuid.New().String()
	appId := "APF%20Analysis%20Agent"
	url := rootUrl + "/apps/" + appId + "/users/" + userId + "/sessions/" + sessionId

	sessionId, err := create_session(t, url, projectId)
	if err != nil {
		t.Errorf("Error creating session: %v\n", err)
		return
	}

	resp, err := send_message(t, rootUrl+"/run", userId, sessionId, "Design Thinking")
	t.Log("Response 1: ", resp)

	resp, err = send_message(t, rootUrl+"/run", userId, sessionId, "Brainstorm")
	t.Log("Response 2: ", resp)

}
