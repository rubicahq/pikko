package storage

import (
	"fmt"
	"path"
	"strings"

	"github.com/google/uuid"
)

// StoreScopedKey builds a tenant-scoped object key (PRD §31, §35).
// Pattern: stores/{storeID}/{purpose}/{yyyy}/{mm}/{uuid}{ext}
func StoreScopedKey(storeID uuid.UUID, purpose, filename string) string {
	ext := path.Ext(filename)
	ext = strings.ToLower(ext)
	if len(ext) > 8 {
		ext = ""
	}
	id := uuid.NewString()
	purpose = sanitizeSegment(purpose)
	return fmt.Sprintf("stores/%s/%s/%s%s", storeID.String(), purpose, id, ext)
}

func sanitizeSegment(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" {
		return "other"
	}
	return out
}

// AssertStoreScoped ensures a key belongs to the given store.
func AssertStoreScoped(storeID uuid.UUID, key string) error {
	prefix := "stores/" + storeID.String() + "/"
	if !strings.HasPrefix(key, prefix) {
		return fmt.Errorf("storage key is not scoped to store")
	}
	if strings.Contains(key, "..") {
		return fmt.Errorf("invalid storage key")
	}
	return nil
}
