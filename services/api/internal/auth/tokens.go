package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// NewToken generates a high-entropy token (>=256 bits) and its SHA-256 hash.
// Only the hash is stored; the raw token is sent to the user.
func NewToken() (raw string, hash []byte, err error) {
	buf := make([]byte, 32) // 256 bits
	if _, err = rand.Read(buf); err != nil {
		return "", nil, fmt.Errorf("generate token: %w", err)
	}
	raw = base64.RawURLEncoding.EncodeToString(buf)
	sum := sha256.Sum256([]byte(raw))
	return raw, sum[:], nil
}

func HashToken(raw string) []byte {
	sum := sha256.Sum256([]byte(raw))
	return sum[:]
}
