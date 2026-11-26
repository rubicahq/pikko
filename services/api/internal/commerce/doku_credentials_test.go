package commerce

import (
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/rubicahq/pikko/services/api/internal/config"
)

func TestDOKUCredentialEncryptionRoundTrip(t *testing.T) {
	key := bytes.Repeat([]byte{0x2a}, 32)
	s := &Service{Cfg: &config.Config{DOKUCredentialEncryptionKey: base64.StdEncoding.EncodeToString(key)}}
	ciphertext, err := s.encryptCredential("merchant-secret", "store-a:checkout")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(ciphertext, []byte("merchant-secret")) {
		t.Fatal("secret stored in plaintext")
	}
	plain, err := s.decryptCredential(ciphertext, "store-a:checkout")
	if err != nil {
		t.Fatal(err)
	}
	if plain != "merchant-secret" {
		t.Fatalf("plain=%q", plain)
	}
	if _, err := s.decryptCredential(ciphertext, "store-b:checkout"); err == nil {
		t.Fatal("ciphertext must be bound to its store and credential type")
	}
}

func TestDOKUCredentialEncryptionUsesRandomNonce(t *testing.T) {
	key := bytes.Repeat([]byte{0x2b}, 32)
	s := &Service{Cfg: &config.Config{DOKUCredentialEncryptionKey: base64.StdEncoding.EncodeToString(key)}}
	a, _ := s.encryptCredential("same-secret", "store-a:mcp")
	b, _ := s.encryptCredential("same-secret", "store-a:mcp")
	if bytes.Equal(a, b) {
		t.Fatal("ciphertexts must differ")
	}
}
