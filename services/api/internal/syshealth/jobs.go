package syshealth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Service) jobFailureCounts(ctx context.Context, r Range) (failed, stuck int, err error) {
	err = s.DB.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM job_queue WHERE status IN ('failed','dead') AND updated_at >= $1 AND updated_at <= $2)
			+ (SELECT COUNT(*) FROM email_jobs WHERE status='failed' AND updated_at >= $1 AND updated_at <= $2)
			+ (SELECT COUNT(*) FROM catalog_import_jobs WHERE status='failed' AND updated_at >= $1 AND updated_at <= $2),
			(SELECT COUNT(*) FROM job_queue
			 WHERE status='processing' AND locked_at IS NOT NULL AND locked_at < now() - $3::interval)
	`, r.From, r.To, fmt.Sprintf("%f seconds", s.stuckAfter().Seconds())).Scan(&failed, &stuck)
	return
}

func (s *Service) JobStats(ctx context.Context, r Range) (*JobStats, error) {
	st := &JobStats{Range: r}
	err := s.DB.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM job_queue WHERE status='pending')
			+ (SELECT COUNT(*) FROM email_jobs WHERE status='pending'),
			(SELECT COUNT(*) FROM job_queue WHERE status='processing')
			+ (SELECT COUNT(*) FROM email_jobs WHERE status='sending')
			+ (SELECT COUNT(*) FROM catalog_import_jobs WHERE status='processing'),
			(SELECT COUNT(*) FROM job_queue WHERE status='completed' AND completed_at >= $1 AND completed_at <= $2)
			+ (SELECT COUNT(*) FROM email_jobs WHERE status='sent' AND sent_at >= $1 AND sent_at <= $2)
			+ (SELECT COUNT(*) FROM catalog_import_jobs WHERE status='completed' AND completed_at >= $1 AND completed_at <= $2),
			(SELECT COUNT(*) FROM job_queue WHERE status IN ('failed','dead') AND updated_at >= $1 AND updated_at <= $2)
			+ (SELECT COUNT(*) FROM email_jobs WHERE status='failed' AND updated_at >= $1 AND updated_at <= $2)
			+ (SELECT COUNT(*) FROM catalog_import_jobs WHERE status='failed' AND updated_at >= $1 AND updated_at <= $2),
			(SELECT COUNT(*) FROM job_queue WHERE status='processing' AND locked_at < now() - $3::interval),
			(SELECT COUNT(*) FROM job_queue WHERE status='failed' AND attempts < max_attempts AND available_at > now()),
			(SELECT COUNT(*) FROM job_queue WHERE status='dead' AND updated_at >= $1 AND updated_at <= $2)
	`, r.From, r.To, fmt.Sprintf("%f seconds", s.stuckAfter().Seconds())).Scan(
		&st.Queued, &st.Running, &st.Success, &st.Failed, &st.Stuck, &st.Retrying, &st.Dead,
	)
	if err != nil {
		return nil, err
	}
	var avg *float64
	err = s.DB.QueryRow(ctx, `
		SELECT AVG(EXTRACT(EPOCH FROM (completed_at - created_at)) * 1000)
		FROM job_queue
		WHERE status='completed' AND completed_at >= $1 AND completed_at <= $2 AND completed_at IS NOT NULL
	`, r.From, r.To).Scan(&avg)
	if err != nil {
		return nil, err
	}
	st.AvgDurMs = avg
	return st, nil
}

func (s *Service) ListJobs(ctx context.Context, r Range, status, queue string, limit int) ([]JobRow, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	switch queue {
	case "job_queue":
		return s.listJobQueue(ctx, r, status, limit)
	case "email_jobs":
		return s.listEmailJobs(ctx, r, status, limit)
	case "catalog_import":
		return s.listImportJobs(ctx, r, status, limit)
	}
	jobs, err := s.listJobQueue(ctx, r, status, limit)
	if err != nil {
		return nil, err
	}
	emails, err := s.listEmailJobs(ctx, r, status, limit)
	if err != nil {
		return nil, err
	}
	imps, err := s.listImportJobs(ctx, r, status, limit)
	if err != nil {
		return nil, err
	}
	out := append(jobs, emails...)
	out = append(out, imps...)
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *Service) listJobQueue(ctx context.Context, r Range, status string, limit int) ([]JobRow, error) {
	q := `
		SELECT id, job_type, status, store_id, attempts, max_attempts, last_error,
		       created_at, updated_at, completed_at, locked_at
		FROM job_queue
		WHERE created_at >= $1 AND created_at <= $2
	`
	args := []any{r.From, r.To}
	if status != "" {
		q += ` AND status=$3`
		args = append(args, status)
		q += ` ORDER BY updated_at DESC LIMIT $4`
		args = append(args, limit)
	} else {
		q += ` ORDER BY updated_at DESC LIMIT $3`
		args = append(args, limit)
	}
	rows, err := s.DB.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []JobRow
	cutoff := time.Now().UTC().Add(-s.stuckAfter())
	for rows.Next() {
		var j JobRow
		var lockedAt *time.Time
		var maxAttempts int
		j.Queue = "job_queue"
		if err := rows.Scan(&j.ID, &j.JobType, &j.Status, &j.StoreID, &j.Attempts, &maxAttempts, &j.LastError,
			&j.CreatedAt, &j.UpdatedAt, &j.CompletedAt, &lockedAt); err != nil {
			return nil, err
		}
		j.MaxAttempts = &maxAttempts
		if j.LastError != nil {
			san := SanitizeText(*j.LastError)
			j.LastError = &san
		}
		if j.Status == "processing" && lockedAt != nil && lockedAt.Before(cutoff) {
			j.Stuck = true
		}
		out = append(out, j)
	}
	if out == nil {
		out = []JobRow{}
	}
	return out, nil
}

func (s *Service) listEmailJobs(ctx context.Context, r Range, status string, limit int) ([]JobRow, error) {
	q := `
		SELECT id, template_key, status, attempts, last_error, created_at, updated_at, sent_at
		FROM email_jobs WHERE created_at >= $1 AND created_at <= $2
	`
	args := []any{r.From, r.To}
	if status != "" {
		q += ` AND status=$3 ORDER BY updated_at DESC LIMIT $4`
		args = append(args, status, limit)
	} else {
		q += ` ORDER BY updated_at DESC LIMIT $3`
		args = append(args, limit)
	}
	rows, err := s.DB.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []JobRow
	for rows.Next() {
		var j JobRow
		j.Queue = "email_jobs"
		if err := rows.Scan(&j.ID, &j.JobType, &j.Status, &j.Attempts, &j.LastError, &j.CreatedAt, &j.UpdatedAt, &j.CompletedAt); err != nil {
			return nil, err
		}
		if j.LastError != nil {
			san := SanitizeText(*j.LastError)
			j.LastError = &san
		}
		out = append(out, j)
	}
	if out == nil {
		out = []JobRow{}
	}
	return out, nil
}

func (s *Service) listImportJobs(ctx context.Context, r Range, status string, limit int) ([]JobRow, error) {
	q := `
		SELECT id, status, store_id, error_message, created_at, updated_at, completed_at
		FROM catalog_import_jobs WHERE created_at >= $1 AND created_at <= $2
	`
	args := []any{r.From, r.To}
	if status != "" {
		q += ` AND status=$3 ORDER BY updated_at DESC LIMIT $4`
		args = append(args, status, limit)
	} else {
		q += ` ORDER BY updated_at DESC LIMIT $3`
		args = append(args, limit)
	}
	rows, err := s.DB.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []JobRow
	for rows.Next() {
		var j JobRow
		j.Queue = "catalog_import"
		j.JobType = "catalog_pdf_import"
		if err := rows.Scan(&j.ID, &j.Status, &j.StoreID, &j.LastError, &j.CreatedAt, &j.UpdatedAt, &j.CompletedAt); err != nil {
			return nil, err
		}
		if j.LastError != nil {
			san := SanitizeText(*j.LastError)
			j.LastError = &san
		}
		out = append(out, j)
	}
	if out == nil {
		out = []JobRow{}
	}
	return out, nil
}

func (s *Service) recentFailedJobs(ctx context.Context, limit int) ([]JobRow, error) {
	r := Range{From: time.Now().UTC().Add(-7 * 24 * time.Hour), To: time.Now().UTC(), Key: "7d"}
	jq, err := s.listJobQueue(ctx, r, "failed", limit)
	if err != nil {
		return nil, err
	}
	dead, err := s.listJobQueue(ctx, r, "dead", limit)
	if err != nil {
		return nil, err
	}
	out := append(jq, dead...)
	emails, err := s.listEmailJobs(ctx, r, "failed", limit)
	if err != nil {
		return nil, err
	}
	out = append(out, emails...)
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// RetryJob safely requeues idempotent job_queue rows only (failed/dead → pending).
// Never retries email/notification side-effects outside job_queue.
func (s *Service) RetryJob(ctx context.Context, id uuid.UUID) (*JobRow, error) {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var status string
	err = tx.QueryRow(ctx, `
		SELECT status FROM job_queue WHERE id=$1 FOR UPDATE
	`, id).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if status != "failed" && status != "dead" {
		return nil, fmt.Errorf("only failed or dead jobs can be retried")
	}
	_, err = tx.Exec(ctx, `
		UPDATE job_queue SET
			status='pending', available_at=now(), last_error=NULL,
			locked_at=NULL, locked_by=NULL, updated_at=now(),
			attempts=CASE WHEN status='dead' THEN 0 ELSE attempts END,
			max_attempts=CASE WHEN status='dead' THEN GREATEST(max_attempts, attempts+3) ELSE max_attempts END
		WHERE id=$1
	`, id)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetJob(ctx, "job_queue", id)
}

func (s *Service) GetJob(ctx context.Context, queue string, id uuid.UUID) (*JobRow, error) {
	if queue != "" && queue != "job_queue" {
		return nil, ErrNotFound
	}
	var j JobRow
	var lockedAt *time.Time
	var maxA int
	j.Queue = "job_queue"
	err := s.DB.QueryRow(ctx, `
		SELECT id, job_type, status, store_id, attempts, max_attempts, last_error,
		       created_at, updated_at, completed_at, locked_at
		FROM job_queue WHERE id=$1
	`, id).Scan(&j.ID, &j.JobType, &j.Status, &j.StoreID, &j.Attempts, &maxA, &j.LastError,
		&j.CreatedAt, &j.UpdatedAt, &j.CompletedAt, &lockedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	j.MaxAttempts = &maxA
	if j.LastError != nil {
		san := SanitizeText(*j.LastError)
		j.LastError = &san
	}
	if j.Status == "processing" && lockedAt != nil && lockedAt.Before(time.Now().UTC().Add(-s.stuckAfter())) {
		j.Stuck = true
	}
	return &j, nil
}
