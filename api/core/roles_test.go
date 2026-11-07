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
	"testing"
)

func TestGetPermissionsForRole(t *testing.T) {
	ctx := context.Background()

	t.Run("Empty role returns empty list without error", func(t *testing.T) {
		perms, err := GetPermissionsForRole(ctx, "")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if len(perms) != 0 {
			t.Errorf("expected 0 permissions, got %d", len(perms))
		}
	})

	t.Run("Valid predefined roles return expected permissions", func(t *testing.T) {
		for role, expectedPerms := range PredefinedRolePermissions {
			perms, err := GetPermissionsForRole(ctx, role)
			if err != nil {
				t.Fatalf("unexpected error for role %s: %v", role, err)
			}
			if len(perms) != len(expectedPerms) {
				t.Errorf("role %s: expected %d permissions, got %d", role, len(expectedPerms), len(perms))
			}
		}
	})

	t.Run("Unknown role returns error", func(t *testing.T) {
		perms, err := GetPermissionsForRole(ctx, "nonexistent:role")
		if err == nil {
			t.Fatalf("expected error for unknown role, got nil")
		}
		if perms != nil {
			t.Errorf("expected nil permissions for unknown role, got: %v", perms)
		}
	})
}

func TestHasAdminPermissions(t *testing.T) {
	ctx := context.Background()

	t.Run("OrgOwner has admin permissions", func(t *testing.T) {
		if !HasAdminPermissions(ctx, RoleOrgOwner) {
			t.Errorf("expected RoleOrgOwner to have admin permissions")
		}
	})

	t.Run("OrgAdmin has admin permissions", func(t *testing.T) {
		if !HasAdminPermissions(ctx, RoleOrgAdmin) {
			t.Errorf("expected RoleOrgAdmin to have admin permissions")
		}
	})

	t.Run("OrgEditor has admin permissions", func(t *testing.T) {
		if !HasAdminPermissions(ctx, RoleOrgEditor) {
			t.Errorf("expected RoleOrgEditor to have admin permissions")
		}
	})

	t.Run("UserEditor lacks admin permissions because it lacks role:update", func(t *testing.T) {
		if HasAdminPermissions(ctx, RoleUserEditor) {
			t.Errorf("expected RoleUserEditor NOT to have admin permissions")
		}
	})

	t.Run("OrgViewer lacks admin permissions", func(t *testing.T) {
		if HasAdminPermissions(ctx, RoleOrgViewer) {
			t.Errorf("expected RoleOrgViewer NOT to have admin permissions")
		}
	})

	t.Run("ProjectAdmin lacks admin permissions", func(t *testing.T) {
		if HasAdminPermissions(ctx, RoleProjectAdmin) {
			t.Errorf("expected RoleProjectAdmin NOT to have admin permissions")
		}
	})

	t.Run("Unknown role returns false", func(t *testing.T) {
		if HasAdminPermissions(ctx, "invalid:role") {
			t.Errorf("expected invalid role NOT to have admin permissions")
		}
	})
}

func TestHasPermission(t *testing.T) {
	ctx := context.Background()

	t.Run("User with matching permission returns true", func(t *testing.T) {
		roles := []string{string(RoleProjectCreator)}
		if !HasPermission(ctx, roles, PermissionProjectCreate) {
			t.Errorf("expected RoleProjectCreator to have PermissionProjectCreate")
		}
	})

	t.Run("User with multiple roles returns true if one possesses permission", func(t *testing.T) {
		roles := []string{string(RoleProjectViewer), string(RoleProjectEditor)}
		if !HasPermission(ctx, roles, PermissionProjectUpdate) {
			t.Errorf("expected roles %v to possess PermissionProjectUpdate", roles)
		}
	})

	t.Run("User without required permission returns false", func(t *testing.T) {
		roles := []string{string(RoleProjectViewer)}
		if HasPermission(ctx, roles, PermissionProjectDelete) {
			t.Errorf("expected RoleProjectViewer NOT to have PermissionProjectDelete")
		}
	})

	t.Run("Empty roles list returns false", func(t *testing.T) {
		if HasPermission(ctx, []string{}, PermissionProjectRead) {
			t.Errorf("expected empty roles list to return false")
		}
	})

	t.Run("Invalid role strings are skipped gracefully", func(t *testing.T) {
		roles := []string{"invalid:role", string(RoleUserViewer)}
		if !HasPermission(ctx, roles, PermissionUserRead) {
			t.Errorf("expected valid role in list to grant permission despite invalid role")
		}
		if HasPermission(ctx, []string{"invalid:role"}, PermissionUserRead) {
			t.Errorf("expected invalid role alone to return false")
		}
	})
}
