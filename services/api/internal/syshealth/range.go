package syshealth

import (
	"fmt"
	"time"
)

// ParseRange maps lock presets (1h/24h/7d/30d/custom). Default 24h.
func ParseRange(key, fromRaw, toRaw string) (Range, error) {
	now := time.Now().UTC()
	key = normalizeKey(key)
	switch key {
	case "1h":
		return Range{From: now.Add(-1 * time.Hour), To: now, Key: "1h"}, nil
	case "7d":
		return Range{From: now.Add(-7 * 24 * time.Hour), To: now, Key: "7d"}, nil
	case "30d":
		return Range{From: now.Add(-30 * 24 * time.Hour), To: now, Key: "30d"}, nil
	case "custom":
		if fromRaw == "" || toRaw == "" {
			return Range{}, fmt.Errorf("custom range requires from and to")
		}
		from, err := time.Parse(time.RFC3339, fromRaw)
		if err != nil {
			return Range{}, fmt.Errorf("invalid from")
		}
		to, err := time.Parse(time.RFC3339, toRaw)
		if err != nil {
			return Range{}, fmt.Errorf("invalid to")
		}
		if !to.After(from) {
			return Range{}, fmt.Errorf("to must be after from")
		}
		if to.Sub(from) > 90*24*time.Hour {
			return Range{}, fmt.Errorf("custom range max 90d")
		}
		return Range{From: from.UTC(), To: to.UTC(), Key: "custom"}, nil
	default: // 24h
		return Range{From: now.Add(-24 * time.Hour), To: now, Key: "24h"}, nil
	}
}

func normalizeKey(k string) string {
	switch k {
	case "1h", "24h", "7d", "30d", "custom":
		return k
	default:
		return "24h"
	}
}

func chartStep(r Range) time.Duration {
	d := r.To.Sub(r.From)
	switch {
	case d <= 2*time.Hour:
		return 5 * time.Minute
	case d <= 48*time.Hour:
		return time.Hour
	case d <= 14*24*time.Hour:
		return 6 * time.Hour
	default:
		return 24 * time.Hour
	}
}
