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

package core_test

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
	"io"
	"testing"

	"google.com/apf/ai/core"
)

func generateTestRSAKeyPair() (string, *rsa.PublicKey, error) {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", nil, err
	}
	privBytes := x509.MarshalPKCS1PrivateKey(privKey)
	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	})
	return string(privPEM), &privKey.PublicKey, nil
}

func encryptSecretsForTest(secrets *core.UserSecretsBridge, rsaPub *rsa.PublicKey) (string, error) {
	data, err := json.Marshal(secrets)
	if err != nil {
		return "", err
	}

	cek := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, cek); err != nil {
		return "", err
	}

	encryptedCEK, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, rsaPub, cek, nil)
	if err != nil {
		return "", err
	}

	aesBlock, err := aes.NewCipher(cek)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(aesBlock)
	if err != nil {
		return "", err
	}
	iv := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	header := []byte(`{"alg":"RSA-OAEP-256","enc":"A256GCM"}`)
	part1 := base64.RawURLEncoding.EncodeToString(header)
	part2 := base64.RawURLEncoding.EncodeToString(encryptedCEK)
	part3 := base64.RawURLEncoding.EncodeToString(iv)

	sealed := gcm.Seal(nil, iv, data, []byte(part1))
	tagSize := gcm.Overhead()
	actualCiphertext := sealed[:len(sealed)-tagSize]
	authTag := sealed[len(sealed)-tagSize:]

	part4 := base64.RawURLEncoding.EncodeToString(actualCiphertext)
	part5 := base64.RawURLEncoding.EncodeToString(authTag)

	return part1 + "." + part2 + "." + part3 + "." + part4 + "." + part5, nil
}

func TestSecretsDecryptor(t *testing.T) {
	privPEM, pubKey, err := generateTestRSAKeyPair()
	if err != nil {
		t.Fatalf("failed to generate test RSA key pair: %v", err)
	}
	t.Setenv("SECRET_ENCRYPTION_PRIVATE_KEY", privPEM)

	decryptor, err := core.NewSecretsDecryptor()
	if err != nil {
		t.Fatalf("failed to create SecretsDecryptor: %v", err)
	}

	origSecrets := &core.UserSecretsBridge{
		UserID:           "user-123",
		GeminiConfigured: true,
		GeminiApiKey:     "gemini_simulated_api_key_12345",
	}

	jwe, err := encryptSecretsForTest(origSecrets, pubKey)
	if err != nil {
		t.Fatalf("failed to encrypt test secrets: %v", err)
	}

	decrypted, err := decryptor.Decrypt(jwe)
	if err != nil {
		t.Fatalf("failed to decrypt JWE: %v", err)
	}

	if decrypted.UserID != origSecrets.UserID {
		t.Errorf("expected UserID %q, got %q", origSecrets.UserID, decrypted.UserID)
	}
	if !decrypted.GeminiConfigured {
		t.Errorf("expected GeminiConfigured true")
	}
	if decrypted.GeminiApiKey != origSecrets.GeminiApiKey {
		t.Errorf("expected GeminiApiKey %q, got %q", origSecrets.GeminiApiKey, decrypted.GeminiApiKey)
	}
}
