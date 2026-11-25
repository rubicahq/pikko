package syshealth

import (
	"encoding/json"
	"regexp"
	"strings"
)

var (
	reBearer     = regexp.MustCompile(`(?i)(bearer\s+)[a-z0-9\-._~+/]+=*`)
	reTokenQuery = regexp.MustCompile(`(?i)([?&](?:token|access_token|refresh_token|session|magic|key|secret|password|passwd|authorization)=)([^&\s]+)`)
	reEmail      = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
	reLongHex    = regexp.MustCompile(`(?i)\b[0-9a-f]{32,}\b`)
	reJWT        = regexp.MustCompile(`\beyJ[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+\b`)
	reCookieHdr  = regexp.MustCompile(`(?i)(cookie\s*[:=]\s*)([^\n;]+)`)
	reCardish    = regexp.MustCompile(`\b(?:\d[ -]*?){13,19}\b`)
)

// SanitizeText redacts secrets, tokens, and obvious PII from free-form text.
func SanitizeText(s string) string {
	if s == "" {
		return s
	}
	out := s
	out = reBearer.ReplaceAllString(out, `${1}[REDACTED]`)
	out = reTokenQuery.ReplaceAllString(out, `${1}[REDACTED]`)
	out = reJWT.ReplaceAllString(out, `[REDACTED_JWT]`)
	out = reCookieHdr.ReplaceAllString(out, `${1}[REDACTED]`)
	out = reLongHex.ReplaceAllString(out, `[REDACTED]`)
	out = reCardish.ReplaceAllString(out, `[REDACTED]`)
	out = reEmail.ReplaceAllString(out, `[REDACTED_EMAIL]`)
	return out
}

// SanitizeContext drops sensitive keys and sanitizes string values.
func SanitizeContext(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		lk := strings.ToLower(k)
		if isSensitiveKey(lk) {
			out[k] = "[REDACTED]"
			continue
		}
		switch t := v.(type) {
		case string:
			out[k] = SanitizeText(t)
		case map[string]any:
			out[k] = SanitizeContext(t)
		case []any:
			out[k] = sanitizeSlice(t)
		default:
			out[k] = v
		}
	}
	return out
}

func sanitizeSlice(in []any) []any {
	out := make([]any, 0, len(in))
	for _, v := range in {
		switch t := v.(type) {
		case string:
			out = append(out, SanitizeText(t))
		case map[string]any:
			out = append(out, SanitizeContext(t))
		default:
			out = append(out, t)
		}
	}
	return out
}

func isSensitiveKey(k string) bool {
	deny := []string{
		"password", "passwd", "secret", "token", "api_key", "apikey", "authorization",
		"cookie", "set-cookie", "magic_link", "session", "bearer", "private_key",
		"access_key", "secret_key", "account_number", "bank", "card", "cvv", "ssn",
		"address", "phone", "email", "body", "payload", "raw", "file", "upload",
	}
	for _, d := range deny {
		if k == d || strings.Contains(k, d) {
			return true
		}
	}
	return false
}

// ContextJSON returns sanitized JSON bytes (never fails; falls back to {}).
func ContextJSON(in map[string]any) []byte {
	b, err := json.Marshal(SanitizeContext(in))
	if err != nil {
		return []byte(`{}`)
	}
	return b
}
