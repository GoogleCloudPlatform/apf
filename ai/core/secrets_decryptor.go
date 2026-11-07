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
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"
)

type UserSecretsBridge struct {
	UserID               string `json:"userId"`
	FigmaConfigured      bool   `json:"figmaConfigured"`
	LucidchartConfigured bool   `json:"lucidchartConfigured"`
	StitchConfigured     bool   `json:"stitchConfigured"`
	DrawioConfigured     bool   `json:"drawioConfigured"`
	GeminiConfigured     bool   `json:"geminiConfigured"`
	FigmaApiKey          string `json:"figmaApiKey"`
	LucidchartApiKey     string `json:"lucidchartApiKey"`
	StitchApiKey         string `json:"stitchApiKey"`
	DrawioApiKey         string `json:"drawioApiKey"`
	GeminiApiKey         string `json:"geminiApiKey"`
}

type SecretsDecryptor struct {
	privateKey *rsa.PrivateKey
}

func NewSecretsDecryptor() (*SecretsDecryptor, error) {
	privPEM := os.Getenv("SECRET_ENCRYPTION_PRIVATE_KEY")
	if strings.TrimSpace(privPEM) == "" {
		return nil, errors.New("SECRET_ENCRYPTION_PRIVATE_KEY environment variable is required")
	}
	privPEM = strings.ReplaceAll(privPEM, `\n`, "\n")
	block, _ := pem.Decode([]byte(privPEM))
	if block == nil {
		return nil, errors.New("failed to decode private key PEM")
	}
	privKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		pkcs8Key, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		var ok bool
		privKey, ok = pkcs8Key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("private key is not an RSA key")
		}
	}
	return &SecretsDecryptor{privateKey: privKey}, nil
}

func (d *SecretsDecryptor) Decrypt(jweCompact string) (*UserSecretsBridge, error) {
	if jweCompact == "" {
		return nil, nil
	}
	parts := strings.Split(jweCompact, ".")
	if len(parts) != 5 {
		return nil, fmt.Errorf("invalid JWE compact format: expected 5 parts, got %d", len(parts))
	}

	part1 := parts[0]
	encryptedCEKBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid JWE part2 base64url: %w", err)
	}

	ivBytes, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid JWE part3 base64url: %w", err)
	}

	ciphertextBytes, err := base64.RawURLEncoding.DecodeString(parts[3])
	if err != nil {
		return nil, fmt.Errorf("invalid JWE part4 base64url: %w", err)
	}

	tagBytes, err := base64.RawURLEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, fmt.Errorf("invalid JWE part5 base64url: %w", err)
	}

	// 1. Decrypt CEK with RSA-OAEP
	cek, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, d.privateKey, encryptedCEKBytes, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt JWE CEK with RSA-OAEP: %w", err)
	}

	// 2. Decrypt AES-GCM payload
	block, err := aes.NewCipher(cek)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES GCM: %w", err)
	}

	sealed := append(ciphertextBytes, tagBytes...)
	plaintext, err := gcm.Open(nil, ivBytes, sealed, []byte(part1))
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt JWE payload with AES GCM: %w", err)
	}

	var secrets UserSecretsBridge
	if err := json.Unmarshal(plaintext, &secrets); err != nil {
		return nil, fmt.Errorf("failed to unmarshal secrets JSON: %w", err)
	}

	return &secrets, nil
}
