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

// System Permission Constants
const (
	// Org Permissions
	PermissionOrgCreate = "org:create"
	PermissionOrgRead   = "org:read"
	PermissionOrgUpdate = "org:update"
	PermissionOrgDelete = "org:delete"
	PermissionOrgList   = "org:list"

	// Project Permissions
	PermissionProjectCreate = "project:create"
	PermissionProjectRead   = "project:read"
	PermissionProjectUpdate = "project:update"
	PermissionProjectDelete = "project:delete"
	PermissionProjectList   = "project:list"

	// User Permissions
	PermissionUserCreate = "user:create"
	PermissionUserRead   = "user:read"
	PermissionUserUpdate = "user:update"
	PermissionUserDelete = "user:delete"
	PermissionUserList   = "user:list"

	// Role Permissions
	PermissionRoleCreate = "role:create"
	PermissionRoleRead   = "role:read"
	PermissionRoleUpdate = "role:update"
	PermissionRoleDelete = "role:delete"
	PermissionRoleList   = "role:list"
)


// AllPermissions returns all 20 valid system permissions.
func AllPermissions() []string {
	return []string{
		PermissionOrgCreate, PermissionOrgRead, PermissionOrgUpdate, PermissionOrgDelete, PermissionOrgList,
		PermissionProjectCreate, PermissionProjectRead, PermissionProjectUpdate, PermissionProjectDelete, PermissionProjectList,
		PermissionUserCreate, PermissionUserRead, PermissionUserUpdate, PermissionUserDelete, PermissionUserList,
		PermissionRoleCreate, PermissionRoleRead, PermissionRoleUpdate, PermissionRoleDelete, PermissionRoleList,
	}
}

// CustomRolePermissions returns all permissions assignable to custom roles.
// Note: "org:create" and "org:delete" are excluded as they are Owner/SuperAdmin-only.
func CustomRolePermissions() []string {
	return []string{
		PermissionOrgRead, PermissionOrgUpdate, PermissionOrgList,
		PermissionProjectCreate, PermissionProjectRead, PermissionProjectUpdate, PermissionProjectDelete, PermissionProjectList,
		PermissionUserCreate, PermissionUserRead, PermissionUserUpdate, PermissionUserDelete, PermissionUserList,
		PermissionRoleCreate, PermissionRoleRead, PermissionRoleUpdate, PermissionRoleDelete, PermissionRoleList,
	}
}
