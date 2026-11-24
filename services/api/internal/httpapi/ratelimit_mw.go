package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rubicahq/pikko/services/api/internal/config"
)

// IPRateLimitMiddleware applies a coarse per-IP hourly ceiling using public_rate_limits.
// Health checks and OPTIONS are skipped. Auth magic-link has its own tighter limits.
func IPRateLimitMiddleware(db *pgxpool.Pool, cfg *config.Config) func(http.Handler) http.Handler {
	max := cfg.APIRateLimitIPHour
	if max <= 0 {
		max = 1200
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}
			path := r.URL.Path
			if path == "/healthz" || path == "/readyz" || path == "/livez" {
				next.ServeHTTP(w, r)
				return
			}
			ip := ClientIP(r)
			if ip == "" {
				ip = "unknown"
			}
			ok, err := allowIPRate(r.Context(), db, "api_ip:"+ip, max)
			if err != nil {
				// Production: fail closed so rate limits cannot be bypassed by DB blips (SEC-07).
				// Non-production keeps fail-open for local/dev resilience; auth magic-link
				// still has its own tighter email/IP limits.
				if cfg != nil && (cfg.AppEnv == "production" || cfg.AppEnv == "staging") {
					writeError(w, r, http.StatusServiceUnavailable, "unavailable", "Temporarily unavailable. Try again shortly.")
					return
				}
				next.ServeHTTP(w, r)
				return
			}
			if !ok {
				w.Header().Set("Retry-After", "3600")
				writeError(w, r, http.StatusTooManyRequests, "rate_limited", "Too many requests. Try again later.")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func allowIPRate(ctx context.Context, db *pgxpool.Pool, bucket string, maxPerHour int) (bool, error) {
	window := time.Now().UTC().Truncate(time.Hour)
	var count int
	err := db.QueryRow(ctx, `
		INSERT INTO public_rate_limits (bucket_key, window_start, count)
		VALUES ($1, $2, 1)
		ON CONFLICT (bucket_key, window_start)
		DO UPDATE SET count = public_rate_limits.count + 1
		RETURNING count
	`, bucket, window).Scan(&count)
	if err != nil {
		return false, err
	}
	return count <= maxPerHour, nil
}
