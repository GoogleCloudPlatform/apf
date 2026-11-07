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
	"errors"
	"time"
)

// UserClaims represents standard user info decode from token.
type UserClaims struct {
	Subject        string   `json:"sub"`
	Roles          []string `json:"roles"`
	Username       string   `json:"username"`
	Token          string   `json:"token,omitempty"`
	OrganizationID string   `json:"organization_id,omitempty"`
	Organizations  []string `json:"organizations,omitempty"`
	IsSuperAdmin   bool     `json:"is_super_admin"`
	Permissions    []string `json:"permissions"`
}

// ContextKey is a custom type for context keys to avoid collisions.
type ContextKey string

// UserClaimsKey is the key used to store/retrieve UserClaims in context.
const UserClaimsKey ContextKey = "userClaims"

// OrganizationIDKey is the key used to store/retrieve OrganizationID in context.
const OrganizationIDKey ContextKey = "organizationId"

// OrganizationIDHeader is the HTTP header used to pass the Organization ID.
const OrganizationIDHeader = "X-Organization-Id"

// WithOrganizationID injects the OrganizationID into context.
func WithOrganizationID(ctx context.Context, orgID string) context.Context {
	return context.WithValue(ctx, OrganizationIDKey, orgID)
}

// GetOrganizationID retrieves the OrganizationID from context.
func GetOrganizationID(ctx context.Context) (string, error) {
	if val, ok := ctx.Value(OrganizationIDKey).(string); ok && val != "" {
		return val, nil
	}
	if claims, ok := ctx.Value(UserClaimsKey).(*UserClaims); ok {
		if claims.OrganizationID != "" {
			return claims.OrganizationID, nil
		}
	}
	return "", errors.New("organization ID not found in context")
}

// HasAccessToOrg checks if the user claims grant access to the specified organization.
func (c *UserClaims) HasAccessToOrg(orgID string) bool {
	if c.IsSuperAdmin {
		return true
	}
	if c.OrganizationID != "" {
		return c.OrganizationID == orgID
	}
	for _, org := range c.Organizations {
		if org == orgID {
			return true
		}
	}
	return false
}

// NFSVolumeStats represents real-time NFS storage volume capacity and health.
type NFSVolumeStats struct {
	TotalBytes     int64  `json:"totalBytes" firestore:"totalBytes"`
	AvailableBytes int64  `json:"availableBytes" firestore:"availableBytes"`
	UsedBytes      int64  `json:"usedBytes" firestore:"usedBytes"`
	MountPath      string `json:"mountPath" firestore:"mountPath"`
	IsHealthy      bool   `json:"isHealthy" firestore:"isHealthy"`
}

// NFSFolderMetadata represents tenant folder-level storage quota, usage, and access flags.
type NFSFolderMetadata struct {
	FolderID        string    `json:"folderId" firestore:"folderId"`
	OrganizationID  string    `json:"organizationId" firestore:"organizationId"`
	ProjectID       string    `json:"projectId" firestore:"projectId"`
	WorkspacePath   string    `json:"workspacePath" firestore:"workspacePath"`
	QuotaSizeBytes  int64     `json:"quotaSizeBytes" firestore:"quotaSizeBytes"`
	UsedSizeBytes   int64     `json:"usedSizeBytes" firestore:"usedSizeBytes"`
	FileCount       int64     `json:"fileCount" firestore:"fileCount"`
	RetentionPolicy string    `json:"retentionPolicy" firestore:"retentionPolicy"`
	LastAuditedAt   time.Time `json:"lastAuditedAt" firestore:"lastAuditedAt"`
	CreatedAt       time.Time `json:"createdAt" firestore:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt" firestore:"updatedAt"`
}

// NFSFolderFilter specifies criteria for listing folder metadata records.
type NFSFolderFilter struct {
	OrganizationID string `json:"organizationId,omitempty"`
	ProjectID      string `json:"projectId,omitempty"`
}

// NFSFolderMetadataUpdate holds optional fields when updating folder metadata.
type NFSFolderMetadataUpdate struct {
	QuotaSizeBytes  *int64  `json:"quotaSizeBytes,omitempty"`
	RetentionPolicy *string `json:"retentionPolicy,omitempty"`
}

// NFS Storage Sentinel Errors
var (
	ErrNFSQuotaExceeded = errors.New("nfs storage quota exceeded")
)

