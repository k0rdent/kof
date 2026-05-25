// Copyright 2025
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

package audit

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// Signer is a pluggable KMS-style interface for signing manifest bytes.
// The signature is base64-encoded and written to manifest.json.sig.
type Signer interface {
	// Sign returns the base64-encoded signature over data.
	Sign(ctx context.Context, data []byte) ([]byte, error)
	// KeyID returns the key reference embedded in the manifest.
	KeyID() string
	// Algorithm returns the signing algorithm name embedded in the manifest.
	Algorithm() string
}

// ---------------------------------------------------------------------------
// LocalSigner — HMAC-SHA256 with a static secret.
// Suitable for development and testing.  In production, replace with an
// AWS KMS, HashiCorp Vault, or PKCS#11-backed implementation.
// ---------------------------------------------------------------------------

// LocalSigner implements Signer using HMAC-SHA256 with a static key.
// The key is derived from the KMS key ID string (UTF-8 bytes).
// For stronger security, set KMS_KEY_ID to a base64-encoded 32-byte secret.
type LocalSigner struct {
	keyID string
	key   []byte
}

// NewLocalSigner creates a LocalSigner. The keyID is used both as the key
// identifier in the manifest and as the HMAC key material (UTF-8 bytes).
func NewLocalSigner(keyID string) *LocalSigner {
	// Attempt to base64-decode the keyID for proper key material.
	// Fall back to raw UTF-8 bytes so plain strings work in dev.
	keyMaterial, err := base64.StdEncoding.DecodeString(keyID)
	if err != nil || len(keyMaterial) == 0 {
		keyMaterial = []byte(keyID)
	}
	return &LocalSigner{keyID: keyID, key: keyMaterial}
}

func (s *LocalSigner) Sign(_ context.Context, data []byte) ([]byte, error) {
	mac := hmac.New(sha256.New, s.key)
	if _, err := mac.Write(data); err != nil {
		return nil, fmt.Errorf("HMAC write: %w", err)
	}
	sig := mac.Sum(nil)
	dst := make([]byte, base64.StdEncoding.EncodedLen(len(sig)))
	base64.StdEncoding.Encode(dst, sig)
	return dst, nil
}

func (s *LocalSigner) KeyID() string     { return s.keyID }
func (s *LocalSigner) Algorithm() string { return SigningAlgorithmHMAC }

// NewSigner returns the appropriate Signer based on configuration.
// Currently only LocalSigner is implemented; extend here to support AWS KMS.
func NewSigner(cfg *Config) (Signer, error) {
	// Future: if cfg.KMSProvider == "aws", return NewAWSKMSSigner(cfg)
	return NewLocalSigner(cfg.KMSKeyID), nil
}
