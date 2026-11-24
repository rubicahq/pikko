package whatsapp

import (
	"net/url"
	"regexp"
	"strings"
)

var varPattern = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_]+)\s*\}\}`)

// Render replaces {{var}} placeholders. Missing keys become empty string.
func Render(body string, vars map[string]string) string {
	return varPattern.ReplaceAllStringFunc(body, func(m string) string {
		sub := varPattern.FindStringSubmatch(m)
		if len(sub) < 2 {
			return ""
		}
		if vars == nil {
			return ""
		}
		return vars[sub[1]]
	})
}

// MissingVariables returns template variables that are empty after merge.
func MissingVariables(t Template, vars map[string]string) []string {
	missing := make([]string, 0)
	for _, key := range t.Variables {
		if strings.TrimSpace(vars[key]) == "" {
			missing = append(missing, key)
		}
	}
	return missing
}

// DeepLink builds https://wa.me/<phone>?text=<urlencoded body>.
// phone may be empty — returns link without phone path segment for chooser.
func DeepLink(phoneDigits, body string) string {
	text := url.QueryEscape(body)
	digits := NormalizePhoneDigits(phoneDigits)
	if digits == "" {
		return "https://wa.me/?text=" + text
	}
	return "https://wa.me/" + digits + "?text=" + text
}

// NormalizePhoneDigits strips non-digits and converts leading 0 Indonesian mobiles to 62.
func NormalizePhoneDigits(raw string) string {
	if raw == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	digits := b.String()
	if digits == "" {
		return ""
	}
	if strings.HasPrefix(digits, "0") {
		digits = "62" + digits[1:]
	}
	if strings.HasPrefix(digits, "8") && len(digits) >= 9 && len(digits) <= 13 {
		// Common local mobile without leading 0
		digits = "62" + digits
	}
	return digits
}
