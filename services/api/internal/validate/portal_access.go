package validate

import (
	"crypto/rand"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

const portalAccessSecretLen = 4

var portalAccessSecretRe = regexp.MustCompile(`^[a-z0-9]{4}$`)

// NewPortalAccessSecret mints a 4-char lowercase alphanumeric secret for portal URLs.
func NewPortalAccessSecret() (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	buf := make([]byte, portalAccessSecretLen)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate portal access secret: %w", err)
	}
	out := make([]byte, portalAccessSecretLen)
	for i := 0; i < portalAccessSecretLen; i++ {
		out[i] = alphabet[int(buf[i])%len(alphabet)]
	}
	return string(out), nil
}

func ValidPortalAccessSecret(secret string) bool {
	return portalAccessSecretRe.MatchString(secret)
}

// ParsePortalAccessKey splits "{secret}-{phone}" where secret is exactly 4 lowercase alnum chars.
// Phone may contain digits only after normalization by the caller; raw path phone is digits (and optionally +).
func ParsePortalAccessKey(raw string) (secret, phone string, ok bool) {
	raw = strings.TrimSpace(raw)
	if len(raw) < portalAccessSecretLen+2 {
		return "", "", false
	}
	secret = raw[:portalAccessSecretLen]
	if !ValidPortalAccessSecret(secret) {
		return "", "", false
	}
	if raw[portalAccessSecretLen] != '-' {
		return "", "", false
	}
	phone = strings.TrimSpace(raw[portalAccessSecretLen+1:])
	if phone == "" {
		return "", "", false
	}
	// Reject unexpected separators in the phone segment (normalized form is digits).
	for _, r := range phone {
		if !unicode.IsDigit(r) && r != '+' {
			return "", "", false
		}
	}
	return secret, phone, true
}

func FormatPortalAccessKey(secret, phone string) string {
	return secret + "-" + phone
}
