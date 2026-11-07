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

package db

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"google.com/apf/api/core"
)

const keyVersionHeader = "v1:"

var encryptionKey []byte

func init() {
	cfg := core.NewConfig()
	keyStr := cfg.SecretsEncryptionKey
	if keyStr == "" {
		keyStr = os.Getenv("SECRETS_ENCRYPTION_KEY")
	}
	if keyStr == "" {
		isTestRun := false
		for _, arg := range os.Args {
			if strings.Contains(arg, "-test.run") || strings.HasSuffix(arg, ".test") || strings.Contains(arg, "/_test/") {
				isTestRun = true
				break
			}
		}
		if isTestRun || flag.Lookup("test.v") != nil {
			keyStr = "dev-32-bytes-secrets-key-apfninj" // Default for isolated unit tests
		} else {
			panic("SECRETS_ENCRYPTION_KEY environment variable is required")
		}
	}


	var keyBytes []byte
	if b, err := hex.DecodeString(keyStr); err == nil && len(b) == 32 {
		keyBytes = b
	} else if b, err := base64.StdEncoding.DecodeString(keyStr); err == nil && len(b) == 32 {
		keyBytes = b
	} else if len(keyStr) == 32 {
		keyBytes = []byte(keyStr)
	} else {
		panic("SECRETS_ENCRYPTION_KEY must be exactly 32 bytes (or valid 32-byte hex/base64 encoded string)")
	}
	encryptionKey = keyBytes
}

// EncryptSecret encrypts a plaintext secret using AES-GCM and prepends the key version header.
func EncryptSecret(plaintext string) (string, error) {
	return encryptSecret(plaintext)
}

func encryptSecret(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return keyVersionHeader + base64.StdEncoding.EncodeToString(ciphertext), nil
}

func decryptSecret(cryptoText string) (string, error) {
	if cryptoText == "" {
		return "", nil
	}
	if !strings.HasPrefix(cryptoText, keyVersionHeader) {
		return "", errors.New("missing key version header")
	}
	actualCryptoText := strings.TrimPrefix(cryptoText, keyVersionHeader)
	ciphertext, err := base64.StdEncoding.DecodeString(actualCryptoText)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, actualCiphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, actualCiphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
