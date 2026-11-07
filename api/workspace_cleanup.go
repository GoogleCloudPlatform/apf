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

package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
)

func deleteProjectWorkspaceFolders(organizationID string, p *db.Project, extraUserIDs ...string) {
	if p == nil || p.ID == "" || p.ID == "." || p.ID == ".." || strings.Contains(p.ID, "/") || strings.Contains(p.ID, "\\") {
		return
	}
	if organizationID == "" || organizationID == "." || organizationID == ".." || strings.Contains(organizationID, "/") || strings.Contains(organizationID, "\\") {
		return
	}

	var mountPath string
	if appConfig != nil {
		mountPath = appConfig.GetWorkspaceMountPath()
	} else {
		mountPath = core.NewConfig().GetWorkspaceMountPath()
	}
	if strings.HasPrefix(mountPath, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			mountPath = filepath.Join(home, mountPath[2:])
		}
	}
	cleanMount := filepath.Clean(mountPath)

	userMap := make(map[string]bool)
	if p.OwnerId != nil && *p.OwnerId != "" {
		userMap[*p.OwnerId] = true
	}
	if p.Members != nil {
		for memberID := range *p.Members {
			if memberID != "" {
				userMap[memberID] = true
			}
		}
	}
	for _, uID := range extraUserIDs {
		if uID != "" {
			userMap[uID] = true
		}
	}

	baseDirs := []string{cleanMount, filepath.Join(cleanMount, "organizations")}
	for _, bDir := range baseDirs {
		orgDir := filepath.Join(bDir, organizationID)
		if userEntries, err := os.ReadDir(orgDir); err == nil {
			for _, uEntry := range userEntries {
				if uEntry.IsDir() && !strings.HasPrefix(uEntry.Name(), ".") {
					userMap[uEntry.Name()] = true
				}
			}
		}
	}

	for userID := range userMap {
		if userID == "" || userID == "." || userID == ".." || strings.Contains(userID, "/") || strings.Contains(userID, "\\") {
			continue
		}
		for _, bDir := range baseDirs {
			targetPath := filepath.Join(bDir, organizationID, userID, p.ID)
			if err := os.RemoveAll(targetPath); err != nil {
				log.Printf("Warning: Failed to clean up workspace path %s: %v", targetPath, err)
			} else {
				log.Printf("Successfully cleaned up workspace path %s", targetPath)
			}

			lockPath := targetPath + ".lock"
			_ = os.RemoveAll(lockPath)
		}
	}
}
