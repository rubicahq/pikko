package syshealth

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

var (
	reUUID    = regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`)
	reDigits  = regexp.MustCompile(`\b\d{4,}\b`)
	reLineNum = regexp.MustCompile(`:\d+`)
)

// Fingerprint builds a stable issue key from source+module+code+normalized message/stack.
// Excludes request IDs and timestamps by construction.
func Fingerprint(source, module, code, exception, stack string) string {
	parts := []string{
		strings.ToLower(strings.TrimSpace(source)),
		strings.ToLower(strings.TrimSpace(module)),
		strings.ToLower(strings.TrimSpace(code)),
		normalizeNoise(exception),
		normalizeStack(stack),
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:16])
}

func normalizeNoise(s string) string {
	s = strings.TrimSpace(s)
	s = SanitizeText(s)
	s = reUUID.ReplaceAllString(s, "<id>")
	s = reDigits.ReplaceAllString(s, "<n>")
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 400 {
		s = s[:400]
	}
	return strings.ToLower(s)
}

func normalizeStack(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	lines := strings.Split(s, "\n")
	kept := make([]string, 0, 8)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = reLineNum.ReplaceAllString(line, "")
		line = reUUID.ReplaceAllString(line, "<id>")
		kept = append(kept, strings.ToLower(line))
		if len(kept) >= 8 {
			break
		}
	}
	return strings.Join(kept, ";")
}
