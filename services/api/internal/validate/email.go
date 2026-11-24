package validate

import (
	"net/mail"
	"strings"
	"unicode"
)

// NormalizeEmail lowercases and trims; returns empty if invalid.
func NormalizeEmail(raw string) (display string, normalized string, ok bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > 254 {
		return "", "", false
	}
	addr, err := mail.ParseAddress(raw)
	if err != nil {
		return "", "", false
	}
	email := strings.TrimSpace(addr.Address)
	if email == "" || !strings.Contains(email, "@") {
		return "", "", false
	}
	parts := strings.SplitN(email, "@", 2)
	local, domain := parts[0], parts[1]
	if local == "" || domain == "" || !strings.Contains(domain, ".") {
		return "", "", false
	}
	for _, r := range email {
		if unicode.IsSpace(r) {
			return "", "", false
		}
	}
	normalized = strings.ToLower(email)
	return email, normalized, true
}
