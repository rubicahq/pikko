package auth

import (
	"testing"
)

func TestNewTokenEntropyAndHash(t *testing.T) {
	raw, hash, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < 40 {
		t.Fatalf("token too short: %d", len(raw))
	}
	if len(hash) != 32 {
		t.Fatalf("hash length %d", len(hash))
	}
	if string(HashToken(raw)) != string(hash) {
		t.Fatal("hash mismatch")
	}
	raw2, _, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if raw == raw2 {
		t.Fatal("tokens should be unique")
	}
}
