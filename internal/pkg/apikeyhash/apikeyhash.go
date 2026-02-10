package apikeyhash

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"
)

const prefix = "sha256:"

// Hash returns the canonical hash representation for an API key.
func Hash(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return prefix + hex.EncodeToString(sum[:])
}

// Verify checks whether plaintext matches the stored hash.
func Verify(storedHash, plaintext string) bool {
	if !strings.HasPrefix(storedHash, prefix) {
		return false
	}
	expected := Hash(plaintext)
	return subtle.ConstantTimeCompare([]byte(storedHash), []byte(expected)) == 1
}
