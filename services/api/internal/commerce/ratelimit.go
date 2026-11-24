package commerce

import (
	"context"
	"fmt"
	"time"
)

// AllowPublicRateLimit increments a per-bucket hourly counter.
// Returns ErrRateLimited when count exceeds maxPerHour.
func (s *Service) AllowPublicRateLimit(ctx context.Context, bucket string, maxPerHour int) error {
	if maxPerHour <= 0 {
		maxPerHour = 30
	}
	window := time.Now().UTC().Truncate(time.Hour)
	var count int
	err := s.DB.QueryRow(ctx, `
		INSERT INTO public_rate_limits (bucket_key, window_start, count)
		VALUES ($1,$2,1)
		ON CONFLICT (bucket_key, window_start)
		DO UPDATE SET count = public_rate_limits.count + 1
		RETURNING count
	`, bucket, window).Scan(&count)
	if err != nil {
		return err
	}
	if count > maxPerHour {
		return ErrRateLimited
	}
	return nil
}

func PublicCheckoutBucket(ip, slug string) string {
	return fmt.Sprintf("checkout:%s:%s", slug, ip)
}

func PublicTrackingBucket(ip string) string {
	return fmt.Sprintf("tracking:%s", ip)
}

func PublicEvidenceBucket(ip string) string {
	return fmt.Sprintf("evidence:%s", ip)
}
