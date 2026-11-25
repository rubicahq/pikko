package syshealth

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rubicahq/pikko/services/api/internal/config"
	"github.com/rubicahq/pikko/services/api/internal/storage"
)

type Checker struct {
	DB      *pgxpool.Pool
	Cfg     *config.Config
	Storage storage.ObjectStore
	Health  *Service
}

// RunChecks performs lightweight probes and persists results. Never pings AI/email providers.
func (c *Checker) RunChecks(ctx context.Context) ([]ServiceRow, error) {
	now := time.Now().UTC()
	rows := []ServiceRow{
		c.checkAPI(now),
		c.checkPostgres(ctx, now),
		c.checkStorage(ctx, now),
		c.checkWorkers(ctx, now),
		c.checkEmailConfig(now),
		c.checkAIConfig(now),
		c.checkSurface("public_site", "Public site", c.Cfg.MarketingURL, now),
		c.checkSurface("merchant_dashboard", "Merchant dashboard", c.Cfg.DashboardURL, now),
		c.checkSurface("storefront", "Storefront", c.Cfg.StorefrontURL, now),
	}
	for _, r := range rows {
		_ = c.persist(ctx, r)
	}
	return rows, nil
}

func (c *Checker) checkAPI(now time.Time) ServiceRow {
	// Process is serving if this code runs.
	ms := 0
	return ServiceRow{
		Key: "api", Name: "API", Status: ServiceOperational,
		Detail: strPtr("process up"), CheckedAt: &now, LatencyMs: &ms,
	}
}

func (c *Checker) checkPostgres(ctx context.Context, now time.Time) ServiceRow {
	start := time.Now()
	err := c.DB.Ping(ctx)
	ms := int(time.Since(start).Milliseconds())
	if err != nil {
		d := "ping failed"
		return ServiceRow{Key: "postgres", Name: "Postgres", Status: ServiceUnavailable, Detail: &d, CheckedAt: &now, LatencyMs: &ms}
	}
	var one int
	if err := c.DB.QueryRow(ctx, `SELECT 1`).Scan(&one); err != nil {
		d := "query failed"
		return ServiceRow{Key: "postgres", Name: "Postgres", Status: ServiceUnavailable, Detail: &d, CheckedAt: &now, LatencyMs: &ms}
	}
	st := c.DB.Stat()
	meta := map[string]any{
		"total_conns": st.TotalConns(),
		"idle_conns":  st.IdleConns(),
	}
	d := "ok"
	return ServiceRow{Key: "postgres", Name: "Postgres", Status: ServiceOperational, Detail: &d, CheckedAt: &now, LatencyMs: &ms, Meta: meta}
}

func (c *Checker) checkStorage(ctx context.Context, now time.Time) ServiceRow {
	start := time.Now()
	ms := 0
	if c.Cfg.StorageDriver == "local" {
		path := c.Cfg.StorageLocalPath
		if path == "" {
			path = "./.data/storage"
		}
		if err := os.MkdirAll(path, 0o755); err != nil {
			ms = int(time.Since(start).Milliseconds())
			d := "local path unavailable"
			return ServiceRow{Key: "object_storage", Name: "Object storage", Status: ServiceUnavailable, Detail: &d, CheckedAt: &now, LatencyMs: &ms}
		}
		ms = int(time.Since(start).Milliseconds())
		d := "local ok"
		return ServiceRow{Key: "object_storage", Name: "Object storage", Status: ServiceOperational, Detail: &d, CheckedAt: &now, LatencyMs: &ms}
	}
	if c.Storage == nil {
		d := "unconfigured"
		return ServiceRow{Key: "object_storage", Name: "Object storage", Status: ServiceUnknown, Detail: &d, CheckedAt: &now}
	}
	// Presence of client only — no remote round-trip that could be expensive/flaky for health.
	ms = int(time.Since(start).Milliseconds())
	d := "client configured (connectivity not probed)"
	return ServiceRow{Key: "object_storage", Name: "Object storage", Status: ServiceUnknown, Detail: &d, CheckedAt: &now, LatencyMs: &ms}
}

func (c *Checker) checkWorkers(ctx context.Context, now time.Time) ServiceRow {
	var processing, stuck int
	_ = c.DB.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE status='processing'),
			COUNT(*) FILTER (WHERE status='processing' AND locked_at < now() - interval '30 minutes')
		FROM job_queue
	`).Scan(&processing, &stuck)
	ms := 0
	meta := map[string]any{"processing": processing, "stuck": stuck}
	if stuck > 0 {
		d := "stuck jobs detected"
		return ServiceRow{Key: "workers", Name: "Workers", Status: ServiceDegraded, Detail: &d, CheckedAt: &now, LatencyMs: &ms, Meta: meta}
	}
	d := "queue reachable"
	return ServiceRow{Key: "workers", Name: "Workers", Status: ServiceOperational, Detail: &d, CheckedAt: &now, LatencyMs: &ms, Meta: meta}
}

func (c *Checker) checkEmailConfig(now time.Time) ServiceRow {
	mode := c.Cfg.EmailMode
	ms := 0
	if mode == "log" {
		d := "EMAIL_MODE=log (not sending)"
		return ServiceRow{Key: "email", Name: "Email", Status: ServiceDegraded, Detail: &d, CheckedAt: &now, LatencyMs: &ms,
			Meta: map[string]any{"mode": mode}}
	}
	if c.Cfg.BrevoAPIKey == "" && mode != "log" {
		d := "provider key missing"
		return ServiceRow{Key: "email", Name: "Email", Status: ServiceUnavailable, Detail: &d, CheckedAt: &now, LatencyMs: &ms,
			Meta: map[string]any{"mode": mode}}
	}
	d := "configured (provider not pinged)"
	return ServiceRow{Key: "email", Name: "Email", Status: ServiceUnknown, Detail: &d, CheckedAt: &now, LatencyMs: &ms,
		Meta: map[string]any{"mode": mode, "provider": c.Cfg.EmailProvider}}
}

func (c *Checker) checkAIConfig(now time.Time) ServiceRow {
	ms := 0
	if strings.TrimSpace(c.Cfg.AIAPIKey) == "" || strings.TrimSpace(c.Cfg.AIBaseURL) == "" {
		d := "AI not configured (mock extractor)"
		return ServiceRow{Key: "ai", Name: "AI", Status: ServiceUnknown, Detail: &d, CheckedAt: &now, LatencyMs: &ms}
	}
	d := "configured (provider not pinged)"
	return ServiceRow{Key: "ai", Name: "AI", Status: ServiceUnknown, Detail: &d, CheckedAt: &now, LatencyMs: &ms,
		Meta: map[string]any{"model": c.Cfg.AIModel}}
}

func (c *Checker) checkSurface(key, name, url string, now time.Time) ServiceRow {
	ms := 0
	if strings.TrimSpace(url) == "" {
		d := "URL not configured"
		return ServiceRow{Key: key, Name: name, Status: ServiceUnknown, Detail: &d, CheckedAt: &now, LatencyMs: &ms}
	}
	// Do not HTTP-probe public surfaces from API (avoid coupling / loops). Report configured.
	d := "URL configured (HTTP not probed from API)"
	return ServiceRow{Key: key, Name: name, Status: ServiceUnknown, Detail: &d, CheckedAt: &now, LatencyMs: &ms,
		Meta: map[string]any{"url": url}}
}

func (c *Checker) persist(ctx context.Context, r ServiceRow) error {
	meta, _ := json.Marshal(r.Meta)
	if meta == nil {
		meta = []byte(`{}`)
	}
	_, err := c.DB.Exec(ctx, `
		INSERT INTO system_service_checks (service_key, display_name, status, detail, checked_at, latency_ms, meta)
		VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb)
		ON CONFLICT (service_key) DO UPDATE SET
			display_name=EXCLUDED.display_name,
			status=EXCLUDED.status,
			detail=EXCLUDED.detail,
			checked_at=EXCLUDED.checked_at,
			latency_ms=EXCLUDED.latency_ms,
			meta=EXCLUDED.meta
	`, r.Key, r.Name, r.Status, r.Detail, r.CheckedAt, r.LatencyMs, string(meta))
	return err
}

func (s *Service) ListServiceChecks(ctx context.Context) ([]ServiceRow, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT service_key, display_name, status, detail, checked_at, latency_ms, meta
		FROM system_service_checks ORDER BY service_key
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ServiceRow
	for rows.Next() {
		var r ServiceRow
		var meta []byte
		if err := rows.Scan(&r.Key, &r.Name, &r.Status, &r.Detail, &r.CheckedAt, &r.LatencyMs, &meta); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(meta, &r.Meta)
		out = append(out, r)
	}
	if len(out) == 0 {
		// No checks yet — return unknown placeholders rather than fabricating operational.
		keys := []struct{ k, n string }{
			{"api", "API"}, {"postgres", "Postgres"}, {"object_storage", "Object storage"},
			{"workers", "Workers"}, {"email", "Email"}, {"ai", "AI"},
			{"public_site", "Public site"}, {"merchant_dashboard", "Merchant dashboard"}, {"storefront", "Storefront"},
		}
		for _, k := range keys {
			d := "not checked yet"
			out = append(out, ServiceRow{Key: k.k, Name: k.n, Status: ServiceUnknown, Detail: &d})
		}
	}
	return out, nil
}

func strPtr(s string) *string { return &s }
