package validate

import (
	"strings"

	"github.com/rubicahq/pikko/services/api/internal/whatsapp"
)

// NormalizeIndonesianPhone validates and normalizes ID mobile / WhatsApp numbers.
// Accepts 08…, +62…, 62…, and local 8… forms used elsewhere in the repo.
// Returns digits suitable for storage and wa.me links (e.g. 6281234567890).
func NormalizeIndonesianPhone(raw string) (normalized string, ok bool) {
	digits := whatsapp.NormalizePhoneDigits(strings.TrimSpace(raw))
	if digits == "" {
		return "", false
	}
	if !strings.HasPrefix(digits, "62") {
		return "", false
	}
	national := digits[2:]
	// Indonesian mobiles: national significant number starts with 8, length 9–13.
	if !strings.HasPrefix(national, "8") {
		return "", false
	}
	if len(national) < 9 || len(national) > 13 {
		return "", false
	}
	return digits, true
}
