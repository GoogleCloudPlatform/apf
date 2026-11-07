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
)

// UserRole defines the role of a user in an organization
type UserRole string

const (
	RoleOrgOwner      UserRole = "org:owner"
	RoleOrgAdmin      UserRole = "org:admin"
	RoleOrgEditor     UserRole = "org:editor"
	RoleOrgViewer     UserRole = "org:viewer"

	RoleProjectAdmin  UserRole = "project:admin"
	RoleProjectEditor UserRole = "project:editor"
	RoleProjectViewer UserRole = "project:viewer"
	RoleProjectCreator UserRole = "project:creator"

	RoleUserAdmin     UserRole = "user:admin"
	RoleUserEditor    UserRole = "user:editor"
	RoleUserViewer    UserRole = "user:viewer"

)

// PredefinedRolePermissions maps predefined roles to their granular system permissions.
var PredefinedRolePermissions = map[UserRole][]string{
	// Org Owner has all 20 permissions
	RoleOrgOwner: AllPermissions(),

	// Org Admin has all permissions EXCEPT org:delete (19 permissions)
	RoleOrgAdmin: {
		PermissionOrgCreate, PermissionOrgRead, PermissionOrgUpdate, PermissionOrgList,
		PermissionProjectCreate, PermissionProjectRead, PermissionProjectUpdate, PermissionProjectDelete, PermissionProjectList,
		PermissionUserCreate, PermissionUserRead, PermissionUserUpdate, PermissionUserDelete, PermissionUserList,
		PermissionRoleCreate, PermissionRoleRead, PermissionRoleUpdate, PermissionRoleDelete, PermissionRoleList,
	},

	// Org Editor has read/update/list for org, plus all project/user/role editor or read permissions
	RoleOrgEditor: {
		PermissionOrgRead, PermissionOrgUpdate, PermissionOrgList,
		PermissionProjectRead, PermissionProjectUpdate, PermissionProjectList,
		PermissionUserRead, PermissionUserUpdate, PermissionUserList,
		PermissionRoleRead, PermissionRoleUpdate, PermissionRoleList,
	},

	// Org Viewer has read/list across all resources
	RoleOrgViewer: {
		PermissionOrgRead, PermissionOrgList,
		PermissionProjectRead, PermissionProjectList,
		PermissionUserRead, PermissionUserList,
		PermissionRoleRead, PermissionRoleList,
	},

	// Project Roles
	RoleProjectAdmin: {
		PermissionProjectCreate, PermissionProjectRead, PermissionProjectUpdate, PermissionProjectDelete, PermissionProjectList,
	},
	RoleProjectEditor: {
		PermissionProjectRead, PermissionProjectUpdate, PermissionProjectList,
	},
	RoleProjectViewer: {
		PermissionProjectRead, PermissionProjectList,
	},
	RoleProjectCreator: {
		PermissionProjectCreate, PermissionProjectRead, PermissionProjectList,
	},

	// User Roles
	RoleUserAdmin: {
		PermissionUserCreate, PermissionUserRead, PermissionUserUpdate, PermissionUserDelete, PermissionUserList,
	},
	RoleUserEditor: {
		PermissionUserRead, PermissionUserUpdate, PermissionUserList,
	},
	RoleUserViewer: {
		PermissionUserRead, PermissionUserList,
	},

}

// PredefinedRoleDescriptions maps predefined roles to their human-readable descriptions.
var PredefinedRoleDescriptions = map[UserRole]string{
	RoleOrgOwner:      "Full administrative access and organization ownership.",
	RoleOrgAdmin:      "Administrative access to manage users, roles, and settings.",
	RoleOrgEditor:     "Can view and modify organization settings, but cannot delete or invite owners.",
	RoleOrgViewer:     "Read-only access to view organization details.",
	RoleProjectAdmin:  "Full administrative control over organization projects.",
	RoleProjectEditor: "Can create, view, and modify project details and files.",
	RoleProjectViewer: "Read-only access to projects and workspaces.",
	RoleProjectCreator: "Can create, view, and list projects within the organization.",
	RoleUserAdmin:     "Can invite, manage, and remove organization users.",
	RoleUserEditor:    "Can view and update user profile metadata.",
	RoleUserViewer:    "Read-only access to the organization user directory.",

}

// GetPermissionsForRole retrieves permissions for a given role.
// It performs a fast, in-memory lookup from the statically defined predefined roles.
func GetPermissionsForRole(ctx context.Context, role UserRole) ([]string, error) {
	if role == "" {
		return []string{}, nil
	}
	if permissions, ok := PredefinedRolePermissions[role]; ok {
		return permissions, nil
	}
	return nil, fmt.Errorf("unknown role: %s", role)
}

// HasAdminPermissions checks if a role has critical administrative permissions (user:update and role:update).
func HasAdminPermissions(ctx context.Context, role UserRole) bool {
	permissions, err := GetPermissionsForRole(ctx, role)
	if err != nil {
		return false
	}
	hasUserUpdate := false
	hasRoleUpdate := false
	for _, p := range permissions {
		if p == PermissionUserUpdate {
			hasUserUpdate = true
		}
		if p == PermissionRoleUpdate {
			hasRoleUpdate = true
		}
	}
	return hasUserUpdate && hasRoleUpdate
}

// HasPermission checks if any of the given roles possesses the required permission.
func HasPermission(ctx context.Context, roles []string, requiredPermission string) bool {
	for _, roleStr := range roles {
		role := UserRole(roleStr)
		perms, err := GetPermissionsForRole(ctx, role)
		if err != nil {
			continue
		}
		for _, p := range perms {
			if p == requiredPermission {
				return true
			}
		}
	}
	return false
}
