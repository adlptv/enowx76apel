package sync

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"

	"golang.org/x/crypto/hkdf"
)

// deriveKey turns the server-issued sync_key + the per-user kdf_salt into a
// 32-byte AES key (HKDF-SHA256). Both come from the cached /me. The key is used
// to seal credential sync items; the server only ever sees ciphertext.
func deriveKey(syncKey, kdfSalt string) ([]byte, error) {
	if syncKey == "" {
		return nil, errors.New("no sync key")
	}
	r := hkdf.New(sha256.New, []byte(syncKey), []byte(kdfSalt), []byte("enowx-sync-cred-v1"))
	key := make([]byte, 32)
	if _, err := io.ReadFull(r, key); err != nil {
		return nil, err
	}
	return key, nil
}

// seal encrypts plaintext with AES-256-GCM, returning base64 payload + nonce.
func seal(key, plaintext []byte) (payload, nonce string, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", err
	}
	n := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, n); err != nil {
		return "", "", err
	}
	ct := gcm.Seal(nil, n, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ct), base64.StdEncoding.EncodeToString(n), nil
}

// open reverses seal.
func open(key []byte, payload, nonce string) ([]byte, error) {
	ct, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, err
	}
	n, err := base64.StdEncoding.DecodeString(nonce)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(n) != gcm.NonceSize() {
		return nil, errors.New("bad nonce")
	}
	return gcm.Open(nil, n, ct, nil)
}
