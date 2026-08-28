package imageshare

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// GenerateKey returns (plainKey, hash, prefix). Plain key is "sk-img-" + 64 hex chars.
func GenerateKey() (string, string, string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", "", fmt.Errorf("imageshare: generate key: %w", err)
	}
	plain := "sk-img-" + hex.EncodeToString(b)
	hash := HashKey(plain)
	prefix := plain[:14]
	return plain, hash, prefix, nil
}

// HashKey returns the SHA-256 hex hash of a plain key.
func HashKey(plain string) string {
	h := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(h[:])
}
