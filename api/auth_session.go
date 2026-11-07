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
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// OrgSessionClaims defines the claims in our secure, backend-issued JWT token.
type OrgSessionClaims struct {
	jwt.RegisteredClaims
	OrgID        string   `json:"org_id"`
	Roles        []string `json:"roles"`
	Permissions  []string `json:"permissions"`
	IsSuperAdmin bool     `json:"is_super_admin,omitempty"`
}

// MintOrgSessionToken generates a signed JWT token for the given user and organization.
func MintOrgSessionToken(subject string, orgID string, roles []string, permissions []string, isSuperAdmin bool, secret string) (string, error) {
	claims := OrgSessionClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		OrgID:        orgID,
		Roles:        roles,
		Permissions:  permissions,
		IsSuperAdmin: isSuperAdmin,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// VerifyOrgSessionToken parses and validates a signed Org Session JWT token.
func VerifyOrgSessionToken(tokenStr string, secret string) (*OrgSessionClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &OrgSessionClaims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("token signature is invalid")
	}
	claims, ok := token.Claims.(*OrgSessionClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims format")
	}
	return claims, nil
}
