package validate

import (
	"regexp"
	"strings"
	"unicode"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// ReservedSlugs are prohibited store slugs (PRD §10). Centralized list.
var ReservedSlugs = map[string]struct{}{
	"admin": {}, "api": {}, "app": {}, "support": {}, "login": {},
	"checkout": {}, "settings": {}, "www": {}, "mail": {}, "static": {},
	"assets": {}, "cdn": {}, "help": {}, "status": {}, "docs": {},
	"auth": {}, "account": {}, "stores": {}, "store": {}, "shop": {},
	"pikko": {}, "null": {}, "undefined": {}, "new": {}, "create": {},
	"dashboard": {}, "billing": {}, "pricing": {}, "legal": {}, "privacy": {},
	"terms": {}, "track": {}, "tracking": {}, "webhook": {}, "webhooks": {},
	"public": {}, "internal": {}, "system": {}, "root": {}, "me": {},
}

func IsReservedSlug(slug string) bool {
	_, ok := ReservedSlugs[strings.ToLower(slug)]
	return ok
}

func ValidateSlug(slug string) bool {
	if len(slug) < 3 || len(slug) > 50 {
		return false
	}
	if strings.HasPrefix(slug, "-") || strings.HasSuffix(slug, "-") {
		return false
	}
	if !slugPattern.MatchString(slug) {
		return false
	}
	return !IsReservedSlug(slug)
}

func SuggestSlug(name string) string {
	var b strings.Builder
	prevHyphen := false
	for _, r := range strings.ToLower(name) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			// Keep ASCII alphanumerics; map common latin.
			if r < unicode.MaxASCII {
				b.WriteRune(r)
				prevHyphen = false
			}
		case r == ' ' || r == '_' || r == '-' || r == '.':
			if b.Len() > 0 && !prevHyphen {
				b.WriteByte('-')
				prevHyphen = true
			}
		}
	}
	s := strings.Trim(b.String(), "-")
	if len(s) > 50 {
		s = strings.Trim(s[:50], "-")
	}
	if len(s) < 3 {
		s = "store"
	}
	if IsReservedSlug(s) {
		s = s + "-shop"
	}
	return s
}

func ValidateStoreName(name string) bool {
	n := strings.TrimSpace(name)
	runes := []rune(n)
	return len(runes) >= 2 && len(runes) <= 80
}

var ValidCategories = map[string]struct{}{
	"personal_shopper_jastip": {},
	"fashion_and_accessories": {},
	"beauty_and_skincare":     {},
	"food_and_beverages":      {},
	"home_and_living":         {},
	"baby_and_kids":           {},
	"electronics_and_gadgets": {},
	"handmade_and_crafts":     {},
	"collectibles_and_hobbies": {},
	"health_and_wellness":     {},
	"merchandise":             {},
	"other":                   {},
}

var ValidSellingModels = map[string]struct{}{
	"ready_stock": {},
	"preorder":    {},
	"both":        {},
}
