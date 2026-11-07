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

package workspace

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

const (
	BackupRefPrefix = "refs/foundry/backups/"
	ShadowRefPrefix = "refs/foundry/shadow-rebase/"
)

type GitSnapshot struct {
	ID          string    `json:"id"`
	RefName     string    `json:"refName"`
	Timestamp   time.Time `json:"timestamp"`
	CommitHash  string    `json:"commitHash"`
	Description string    `json:"description"`
}

// CreateSnapshot creates a backup ref under refs/foundry/backups/ to lock in the safe workspace state before mutations.
func (c *GitClient) CreateSnapshot(ctx context.Context, organizationID string, repoCfg *GitRepoConfig, description string) (*GitSnapshot, error) {
	lock, err := c.acquireLock(ctx, organizationID, repoCfg)
	if err != nil {
		return nil, err
	}
	defer lock.Unlock()

	clonePath := c.getClonePath(organizationID, repoCfg)
	headHash, err := c.runGit(ctx, clonePath, "", "rev-parse", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("failed to resolve HEAD for snapshot: %w", err)
	}

	timestampStr := time.Now().Format("20060102-150405")
	shortHash := headHash
	if len(shortHash) > 7 {
		shortHash = shortHash[:7]
	}
	refName := fmt.Sprintf("%s%s-%s", BackupRefPrefix, timestampStr, shortHash)

	if _, err := c.runGit(ctx, clonePath, "", "update-ref", refName, headHash); err != nil {
		return nil, fmt.Errorf("failed to store snapshot reference %s: %w", refName, err)
	}

	snapshot := &GitSnapshot{
		ID:          refName,
		RefName:     refName,
		Timestamp:   time.Now(),
		CommitHash:  headHash,
		Description: description,
	}

	slog.Info("Successfully created Git workspace snapshot", "ref", refName, "commit", shortHash, "desc", description)
	return snapshot, nil
}

// RestoreSnapshot rolls back the active workspace HEAD and worktree to a previously recorded snapshot ref.
func (c *GitClient) RestoreSnapshot(ctx context.Context, organizationID string, repoCfg *GitRepoConfig, snapshotID string) error {
	lock, err := c.acquireLock(ctx, organizationID, repoCfg)
	if err != nil {
		return err
	}
	defer lock.Unlock()

	clonePath := c.getClonePath(organizationID, repoCfg)
	if _, err := c.runGit(ctx, clonePath, "", "reset", "--hard", snapshotID); err != nil {
		return fmt.Errorf("failed to execute hard reset to snapshot %s: %w", snapshotID, err)
	}

	slog.Info("Successfully restored Git workspace to snapshot", "ref", snapshotID)
	return nil
}

// ListSnapshots returns all backup refs created for the workspace, ordered by timestamp descending.
func (c *GitClient) ListSnapshots(ctx context.Context, organizationID string, repoCfg *GitRepoConfig) ([]*GitSnapshot, error) {
	clonePath := c.getClonePath(organizationID, repoCfg)
	out, err := c.runGit(ctx, clonePath, "", "for-each-ref", "--format=%(refname)|%(objectname)|%(committerdate:iso-strict)|%(subject)", BackupRefPrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshot references: %w", err)
	}

	var snapshots []*GitSnapshot
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 3 {
			continue
		}
		refName := parts[0]
		commitHash := parts[1]
		var timestamp time.Time
		if t, errParse := time.Parse(time.RFC3339, parts[2]); errParse == nil {
			timestamp = t
		} else {
			timestamp = time.Now()
		}
		desc := ""
		if len(parts) == 4 {
			desc = strings.TrimSpace(parts[3])
		}

		snapshots = append(snapshots, &GitSnapshot{
			ID:          refName,
			RefName:     refName,
			Timestamp:   timestamp,
			CommitHash:  commitHash,
			Description: desc,
		})
	}

	return snapshots, nil
}
