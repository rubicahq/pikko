package syshealth

import (
	"context"
	"time"

	"github.com/google/uuid"
)

func (s *Service) Overview(ctx context.Context, r Range) (*Overview, error) {
	out := &Overview{
		Range:       r,
		GeneratedAt: time.Now().UTC(),
		Unavailable: []string{},
	}
	if s.DeployVersion != "" {
		v := s.DeployVersion
		out.DeployVersion = &v
	} else {
		out.Unavailable = append(out.Unavailable, "deploy_version")
	}

	services, err := s.ListServiceChecks(ctx)
	if err != nil {
		return nil, err
	}
	out.ServiceHealth = services

	var unresolved, critical int
	err = s.DB.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE status IN ('unresolved','investigating')),
			COUNT(*) FILTER (WHERE status IN ('unresolved','investigating') AND severity='critical')
		FROM system_error_issues
		WHERE last_seen_at >= $1 AND last_seen_at <= $2
	`, r.From, r.To).Scan(&unresolved, &critical)
	if err != nil {
		return nil, err
	}
	out.UnresolvedIssues = &unresolved
	out.CriticalIssues = &critical

	var affected int
	err = s.DB.QueryRow(ctx, `
		SELECT COUNT(DISTINCT store_id) FROM system_error_occurrences
		WHERE occurred_at >= $1 AND occurred_at <= $2 AND store_id IS NOT NULL
	`, r.From, r.To).Scan(&affected)
	if err != nil {
		return nil, err
	}
	out.AffectedStores = &affected

	var reqCount, successCount, serverErr int
	err = s.DB.QueryRow(ctx, `
		SELECT COALESCE(SUM(request_count),0), COALESCE(SUM(success_count),0), COALESCE(SUM(server_error_count),0)
		FROM api_metric_minutes WHERE bucket_ts >= $1 AND bucket_ts <= $2
	`, r.From, r.To).Scan(&reqCount, &successCount, &serverErr)
	if err != nil {
		return nil, err
	}
	if reqCount == 0 {
		out.Unavailable = append(out.Unavailable, "api_success_rate", "api_request_count")
	} else {
		out.APIRequestCount = &reqCount
		// Success rate = non-5xx / total (4xx are client, still "handled")
		rate := 100.0 * float64(reqCount-serverErr) / float64(reqCount)
		out.APISuccessRate = &rate
	}

	p95, ok, err := s.p95Latency(ctx, r.From, r.To)
	if err != nil {
		return nil, err
	}
	if ok {
		out.P95LatencyMs = &p95
	} else {
		out.Unavailable = append(out.Unavailable, "p95_latency_ms")
	}

	failed, stuck, err := s.jobFailureCounts(ctx, r)
	if err != nil {
		return nil, err
	}
	out.FailedJobs = &failed
	out.StuckJobs = &stuck

	charts, err := s.overviewCharts(ctx, r)
	if err != nil {
		return nil, err
	}
	out.Charts = charts

	out.RecentCritical, err = s.listIssues(ctx, r, "critical", StatusUnresolved, 8)
	if err != nil {
		return nil, err
	}
	out.RecentFailedJobs, err = s.recentFailedJobs(ctx, 8)
	if err != nil {
		return nil, err
	}
	out.MostAffectedStores, err = s.mostAffectedStores(ctx, r, 8)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Service) p95Latency(ctx context.Context, from, to time.Time) (float64, bool, error) {
	var p95 *float64
	err := s.DB.QueryRow(ctx, `
		SELECT percentile_cont(0.95) WITHIN GROUP (ORDER BY duration_ms)::float8
		FROM api_latency_samples
		WHERE sampled_at >= $1 AND sampled_at <= $2
	`, from, to).Scan(&p95)
	if err != nil {
		return 0, false, err
	}
	if p95 == nil {
		return 0, false, nil
	}
	return *p95, true, nil
}

func (s *Service) overviewCharts(ctx context.Context, r Range) (OverviewCharts, error) {
	var c OverviewCharts
	step := chartStep(r)

	rows, err := s.DB.Query(ctx, `
		SELECT date_trunc('hour', occurred_at) AS t, COUNT(*)
		FROM system_error_occurrences
		WHERE occurred_at >= $1 AND occurred_at <= $2
		GROUP BY 1 ORDER BY 1
	`, r.From, r.To)
	if err != nil {
		return c, err
	}
	defer rows.Close()
	for rows.Next() {
		var t time.Time
		var n int
		if err := rows.Scan(&t, &n); err != nil {
			return c, err
		}
		c.ErrorOccurrences = append(c.ErrorOccurrences, TimeCount{T: t.UTC(), Count: n})
	}

	mrows, err := s.DB.Query(ctx, `
		SELECT bucket_ts, request_count, server_error_count, latency_sum_ms, latency_max_ms
		FROM api_metric_minutes
		WHERE bucket_ts >= $1 AND bucket_ts <= $2
		ORDER BY bucket_ts
	`, r.From, r.To)
	if err != nil {
		return c, err
	}
	defer mrows.Close()
	for mrows.Next() {
		var t time.Time
		var req, serr, maxMs int
		var sumMs int64
		if err := mrows.Scan(&t, &req, &serr, &sumMs, &maxMs); err != nil {
			return c, err
		}
		tr := TimeRate{T: t.UTC(), Errors: serr, Requests: req}
		if req > 0 {
			rate := 100.0 * float64(serr) / float64(req)
			tr.Rate = &rate
			avg := float64(sumMs) / float64(req)
			c.APILatency = append(c.APILatency, TimeLatency{T: t.UTC(), AvgMs: &avg, MaxMs: &maxMs})
		} else {
			c.APILatency = append(c.APILatency, TimeLatency{T: t.UTC()})
		}
		c.APIErrorRate = append(c.APIErrorRate, tr)
	}
	_ = step

	modRows, err := s.DB.Query(ctx, `
		SELECT COALESCE(NULLIF(module,''), source) AS m, COUNT(*)
		FROM system_error_occurrences
		WHERE occurred_at >= $1 AND occurred_at <= $2
		GROUP BY 1 ORDER BY 2 DESC LIMIT 12
	`, r.From, r.To)
	if err != nil {
		return c, err
	}
	defer modRows.Close()
	for modRows.Next() {
		var name string
		var n int
		if err := modRows.Scan(&name, &n); err != nil {
			return c, err
		}
		c.ErrorsByModule = append(c.ErrorsByModule, NamedCount{Name: name, Count: n})
	}

	sevRows, err := s.DB.Query(ctx, `
		SELECT severity, COUNT(*) FROM system_error_occurrences
		WHERE occurred_at >= $1 AND occurred_at <= $2
		GROUP BY 1
	`, r.From, r.To)
	if err != nil {
		return c, err
	}
	defer sevRows.Close()
	for sevRows.Next() {
		var name string
		var n int
		if err := sevRows.Scan(&name, &n); err != nil {
			return c, err
		}
		c.Severity = append(c.Severity, NamedCount{Name: name, Count: n})
	}

	// Attach window P95 onto last latency point when samples exist (chart avg/max are measured).
	if p95, ok, err := s.p95Latency(ctx, r.From, r.To); err == nil && ok && len(c.APILatency) > 0 {
		last := c.APILatency[len(c.APILatency)-1]
		last.P95Ms = &p95
		c.APILatency[len(c.APILatency)-1] = last
	}
	return c, nil
}

func (s *Service) mostAffectedStores(ctx context.Context, r Range, limit int) ([]StoreImpact, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT o.store_id, s.name, s.slug, COUNT(*) AS c
		FROM system_error_occurrences o
		LEFT JOIN stores s ON s.id = o.store_id
		WHERE o.occurred_at >= $1 AND o.occurred_at <= $2 AND o.store_id IS NOT NULL
		GROUP BY o.store_id, s.name, s.slug
		ORDER BY c DESC
		LIMIT $3
	`, r.From, r.To, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StoreImpact
	for rows.Next() {
		var si StoreImpact
		var name, slug *string
		if err := rows.Scan(&si.StoreID, &name, &slug, &si.Count); err != nil {
			return nil, err
		}
		si.StoreName = name
		si.StoreSlug = slug
		out = append(out, si)
	}
	if out == nil {
		out = []StoreImpact{}
	}
	return out, nil
}

func ptrUUID(id uuid.UUID) *uuid.UUID { return &id }
