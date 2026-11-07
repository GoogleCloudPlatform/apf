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
	"context"

	"google.com/apf/api/core"
	"google.com/apf/api/db"
	"google.com/apf/api/jobs"
)

// WorkspaceDirStats holds scanned disk usage metrics for a workspace directory.
type WorkspaceDirStats = jobs.WorkspaceDirStats

// ScanWorkspaceDirectory recursively calculates total size in bytes and file count for a directory.
func ScanWorkspaceDirectory(dirPath string) (totalBytes int64, fileCount int64, err error) {
	return jobs.ScanWorkspaceDirectory(dirPath)
}

// RunNFSAuditScan scans all registered or discovered organization workspace folders and updates Firestore metadata.
func RunNFSAuditScan(ctx context.Context, config *core.Configuration, nfsStore db.NFSStore) (int, error) {
	return jobs.RunNFSAuditScan(ctx, config, nfsStore)
}
