package syshealth

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	DB            *pgxpool.Pool
	Logger        *slog.Logger
	DeployVersion string
	Environment   string
	StuckAfter    time.Duration // default 30m for processing jobs
}

func (s *Service) env() string {
	if s.Environment == "" {
		return "production"
	}
	return s.Environment
}

func (s *Service) stuckAfter() time.Duration {
	if s.StuckAfter <= 0 {
		return 30 * time.Minute
	}
	return s.StuckAfter
}

func (s *Service) log() *slog.Logger {
	if s.Logger == nil {
		return slog.Default()
	}
	return s.Logger
}

// Record persists an error occurrence and upserts the issue. Failures are logged
// and never returned to callers — logging must not break merchant ops.
func (s *Service) Record(ctx context.Context, ev Event) {
	defer func() {
		if r := recover(); r != nil {
			s.log().Error("syshealth_record_panic", "recover", r)
		}
	}()
	if s.DB == nil {
		return
	}
	if ev.OccurredAt.IsZero() {
		ev.OccurredAt = time.Now().UTC()
	}
	if ev.Environment == "" {
		ev.Environment = s.env()
	}
	if ev.DeployVersion == "" {
		ev.DeployVersion = s.DeployVersion
	}
	if ev.Severity == "" {
		ev.Severity = SeverityError
	}
	if ev.Source == "" {
		ev.Source = "api"
	}
	msg := SanitizeText(strings.TrimSpace(ev.Message))
	if msg == "" {
		msg = "unknown error"
	}
	stack := SanitizeText(ev.Stack)
	fp := Fingerprint(ev.Source, ev.Module, ev.ErrorCode, msg, stack)
	title := msg
	if len(title) > 160 {
		title = title[:160]
	}
	ctxJSON := ContextJSON(ev.Context)

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		s.log().Error("syshealth_record_begin", "error", err)
		return
	}
	defer tx.Rollback(ctx)

	var issueID uuid.UUID
	var status string
	err = tx.QueryRow(ctx, `
		SELECT id, status FROM system_error_issues WHERE fingerprint=$1 FOR UPDATE
	`, fp).Scan(&issueID, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `
			INSERT INTO system_error_issues (
				fingerprint, title, severity, source, module, error_code,
				status, first_seen_at, last_seen_at, occurrence_count, deploy_version
			) VALUES ($1,$2,$3,$4,$5,$6,'unresolved',$7,$7,1,NULLIF($8,''))
			RETURNING id
		`, fp, title, ev.Severity, ev.Source, ev.Module, ev.ErrorCode, ev.OccurredAt, ev.DeployVersion).Scan(&issueID)
		if err != nil {
			s.log().Error("syshealth_issue_insert", "error", err)
			return
		}
	} else if err != nil {
		s.log().Error("syshealth_issue_select", "error", err)
		return
	} else {
		// Reopen resolved/ignored on recurrence
		newStatus := status
		if status == StatusResolved || status == StatusIgnored {
			newStatus = StatusUnresolved
		}
		_, err = tx.Exec(ctx, `
			UPDATE system_error_issues SET
				last_seen_at=$2,
				occurrence_count=occurrence_count+1,
				severity=CASE
					WHEN $3='critical' THEN 'critical'
					WHEN severity='critical' THEN 'critical'
					WHEN $3='error' AND severity IN ('warning','info') THEN 'error'
					ELSE severity END,
				title=$4,
				deploy_version=COALESCE(NULLIF($5,''), deploy_version),
				status=$6,
				resolved_at=CASE WHEN $6='unresolved' THEN NULL ELSE resolved_at END,
				updated_at=now()
			WHERE id=$1
		`, issueID, ev.OccurredAt, ev.Severity, title, ev.DeployVersion, newStatus)
		if err != nil {
			s.log().Error("syshealth_issue_update", "error", err)
			return
		}
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO system_error_occurrences (
			issue_id, occurred_at, environment, source, severity, module, error_code,
			message, store_id, user_id, request_id, endpoint, http_status, duration_ms,
			stack, deploy_version, context
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NULLIF($11,''),NULLIF($12,''),$13,$14,
			NULLIF($15,''),NULLIF($16,''),$17::jsonb
		)
	`, issueID, ev.OccurredAt, ev.Environment, ev.Source, ev.Severity, ev.Module, ev.ErrorCode,
		msg, ev.StoreID, ev.UserID, ev.RequestID, ev.Endpoint, ev.HTTPStatus, ev.DurationMs,
		stack, ev.DeployVersion, string(ctxJSON))
	if err != nil {
		s.log().Error("syshealth_occurrence_insert", "error", err)
		return
	}

	_, _ = tx.Exec(ctx, `
		UPDATE system_error_issues i SET
			affected_store_count = (
				SELECT COUNT(DISTINCT store_id) FROM system_error_occurrences
				WHERE issue_id=i.id AND store_id IS NOT NULL
			),
			updated_at=now()
		WHERE i.id=$1
	`, issueID)

	if err := tx.Commit(ctx); err != nil {
		s.log().Error("syshealth_record_commit", "error", err)
	}
}

// RecordRequestMetric updates minute buckets + latency samples. Best-effort.
func (s *Service) RecordRequestMetric(ctx context.Context, status, durationMs int) {
	defer func() {
		if r := recover(); r != nil {
			s.log().Error("syshealth_metric_panic", "recover", r)
		}
	}()
	if s.DB == nil || durationMs < 0 {
		return
	}
	bucket := time.Now().UTC().Truncate(time.Minute)
	success, clientErr, serverErr := 0, 0, 0
	switch {
	case status >= 500:
		serverErr = 1
	case status >= 400:
		clientErr = 1
	default:
		success = 1
	}
	_, err := s.DB.Exec(ctx, `
		INSERT INTO api_metric_minutes (
			bucket_ts, request_count, success_count, client_error_count, server_error_count,
			latency_sum_ms, latency_max_ms
		) VALUES ($1,1,$2,$3,$4,$5,$6)
		ON CONFLICT (bucket_ts) DO UPDATE SET
			request_count = api_metric_minutes.request_count + 1,
			success_count = api_metric_minutes.success_count + EXCLUDED.success_count,
			client_error_count = api_metric_minutes.client_error_count + EXCLUDED.client_error_count,
			server_error_count = api_metric_minutes.server_error_count + EXCLUDED.server_error_count,
			latency_sum_ms = api_metric_minutes.latency_sum_ms + EXCLUDED.latency_sum_ms,
			latency_max_ms = GREATEST(api_metric_minutes.latency_max_ms, EXCLUDED.latency_max_ms)
	`, bucket, success, clientErr, serverErr, durationMs, durationMs)
	if err != nil {
		s.log().Error("syshealth_metric_upsert", "error", err)
	}
	// Sample every request under light load; still fine for pilot volumes.
	_, err = s.DB.Exec(ctx, `
		INSERT INTO api_latency_samples (sampled_at, duration_ms, status) VALUES ($1,$2,$3)
	`, time.Now().UTC(), durationMs, status)
	if err != nil {
		s.log().Error("syshealth_latency_sample", "error", err)
	}
}

// RecordAIUsage stores measured AI call stats. Token fields nil when unknown.
func (s *Service) RecordAIUsage(ctx context.Context, storeID *uuid.UUID, provider, model, op string, success bool, promptTok, completionTok, totalTok *int, durationMs *int, errCode string) {
	defer func() {
		if r := recover(); r != nil {
			s.log().Error("syshealth_ai_usage_panic", "recover", r)
		}
	}()
	if s.DB == nil {
		return
	}
	_, err := s.DB.Exec(ctx, `
		INSERT INTO ai_usage_events (
			store_id, provider, model, operation, success,
			prompt_tokens, completion_tokens, total_tokens, duration_ms, error_code
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NULLIF($10,''))
	`, storeID, provider, model, op, success, promptTok, completionTok, totalTok, durationMs, errCode)
	if err != nil {
		s.log().Error("syshealth_ai_usage_insert", "error", err)
	}
}

func (s *Service) IsHQAdmin(ctx context.Context, email string) (bool, error) {
	var exists bool
	norm := strings.ToLower(strings.TrimSpace(email))
	err := s.DB.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM hq_admins WHERE email_normalized=$1)
	`, norm).Scan(&exists)
	return exists, err
}

func (s *Service) LinkHQAdminUser(ctx context.Context, emailNormalized string, userID uuid.UUID) {
	_, _ = s.DB.Exec(ctx, `
		UPDATE hq_admins SET user_id=$2, updated_at=now()
		WHERE email_normalized=$1 AND (user_id IS NULL OR user_id <> $2)
	`, strings.ToLower(emailNormalized), userID)
}

func (s *Service) Audit(ctx context.Context, actorUserID *uuid.UUID, actorEmail, action, subjectType, subjectID string, meta map[string]any) {
	b, _ := json.Marshal(SanitizeContext(meta))
	_, err := s.DB.Exec(ctx, `
		INSERT INTO hq_admin_audit (actor_user_id, actor_email, action, subject_type, subject_id, meta)
		VALUES ($1,$2,$3,$4,$5,$6::jsonb)
	`, actorUserID, actorEmail, action, nullEmpty(subjectType), nullEmpty(subjectID), string(b))
	if err != nil {
		s.log().Error("syshealth_audit_insert", "error", err)
	}
}

func nullEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// CleanupRetention deletes old rows per lock (occurrences 30d, samples 24h, jobs left to callers).
func (s *Service) CleanupRetention(ctx context.Context) {
	if s.DB == nil {
		return
	}
	_, err := s.DB.Exec(ctx, `DELETE FROM system_error_occurrences WHERE occurred_at < now() - interval '30 days'`)
	if err != nil {
		s.log().Error("syshealth_cleanup_occurrences", "error", err)
	}
	_, err = s.DB.Exec(ctx, `DELETE FROM system_error_issues WHERE last_seen_at < now() - interval '90 days' AND status IN ('resolved','ignored')`)
	if err != nil {
		s.log().Error("syshealth_cleanup_issues", "error", err)
	}
	_, err = s.DB.Exec(ctx, `DELETE FROM api_latency_samples WHERE sampled_at < now() - interval '24 hours'`)
	if err != nil {
		s.log().Error("syshealth_cleanup_samples", "error", err)
	}
	_, err = s.DB.Exec(ctx, `DELETE FROM api_metric_minutes WHERE bucket_ts < now() - interval '90 days'`)
	if err != nil {
		s.log().Error("syshealth_cleanup_metrics", "error", err)
	}
	_, err = s.DB.Exec(ctx, `DELETE FROM ai_usage_events WHERE occurred_at < now() - interval '90 days'`)
	if err != nil {
		s.log().Error("syshealth_cleanup_ai", "error", err)
	}
}
