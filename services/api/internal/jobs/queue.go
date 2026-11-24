package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound = errors.New("job not found")
	ErrConflict = errors.New("job conflict")
)

const (
	TypeCatalogPDFImport = "catalog_pdf_import"
)

type Job struct {
	ID             uuid.UUID
	StoreID        *uuid.UUID
	JobType        string
	Payload        json.RawMessage
	Status         string
	IdempotencyKey string
	Attempts       int
	MaxAttempts    int
	LastError      *string
	AvailableAt    time.Time
	LockedAt       *time.Time
	LockedBy       *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	CompletedAt    *time.Time
}

type Queue struct {
	DB *pgxpool.Pool
}

// Enqueue inserts a job; duplicate idempotency_key is a no-op that returns the existing job.
func (q *Queue) Enqueue(ctx context.Context, storeID *uuid.UUID, jobType string, payload any, idempotencyKey string, maxAttempts int) (*Job, error) {
	if idempotencyKey == "" {
		idempotencyKey = uuid.NewString()
	}
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	var j Job
	err = q.DB.QueryRow(ctx, `
		INSERT INTO job_queue (store_id, job_type, payload, idempotency_key, max_attempts)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (idempotency_key) DO UPDATE SET updated_at = job_queue.updated_at
		RETURNING id, store_id, job_type, payload, status, idempotency_key, attempts, max_attempts,
		          last_error, available_at, locked_at, locked_by, created_at, updated_at, completed_at
	`, storeID, jobType, raw, idempotencyKey, maxAttempts).Scan(
		&j.ID, &j.StoreID, &j.JobType, &j.Payload, &j.Status, &j.IdempotencyKey, &j.Attempts, &j.MaxAttempts,
		&j.LastError, &j.AvailableAt, &j.LockedAt, &j.LockedBy, &j.CreatedAt, &j.UpdatedAt, &j.CompletedAt,
	)
	if err != nil {
		return nil, err
	}
	return &j, nil
}

// ClaimNext locks the next available job of the given types using SKIP LOCKED.
func (q *Queue) ClaimNext(ctx context.Context, workerID string, types []string) (*Job, error) {
	tx, err := q.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var j Job
	err = tx.QueryRow(ctx, `
		SELECT id, store_id, job_type, payload, status, idempotency_key, attempts, max_attempts,
		       last_error, available_at, locked_at, locked_by, created_at, updated_at, completed_at
		FROM job_queue
		WHERE status IN ('pending', 'failed')
		  AND available_at <= now()
		  AND job_type = ANY($1)
		ORDER BY available_at ASC
		FOR UPDATE SKIP LOCKED
		LIMIT 1
	`, types).Scan(
		&j.ID, &j.StoreID, &j.JobType, &j.Payload, &j.Status, &j.IdempotencyKey, &j.Attempts, &j.MaxAttempts,
		&j.LastError, &j.AvailableAt, &j.LockedAt, &j.LockedBy, &j.CreatedAt, &j.UpdatedAt, &j.CompletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	err = tx.QueryRow(ctx, `
		UPDATE job_queue
		SET status='processing', attempts=attempts+1, locked_at=now(), locked_by=$2, updated_at=now(), last_error=NULL
		WHERE id=$1
		RETURNING id, store_id, job_type, payload, status, idempotency_key, attempts, max_attempts,
		          last_error, available_at, locked_at, locked_by, created_at, updated_at, completed_at
	`, j.ID, workerID).Scan(
		&j.ID, &j.StoreID, &j.JobType, &j.Payload, &j.Status, &j.IdempotencyKey, &j.Attempts, &j.MaxAttempts,
		&j.LastError, &j.AvailableAt, &j.LockedAt, &j.LockedBy, &j.CreatedAt, &j.UpdatedAt, &j.CompletedAt,
	)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &j, nil
}

func (q *Queue) Complete(ctx context.Context, id uuid.UUID) error {
	ct, err := q.DB.Exec(ctx, `
		UPDATE job_queue SET status='completed', completed_at=now(), updated_at=now(), locked_at=NULL, locked_by=NULL
		WHERE id=$1
	`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (q *Queue) Fail(ctx context.Context, id uuid.UUID, errMsg string, retryAfter time.Duration) error {
	var attempts, maxAttempts int
	err := q.DB.QueryRow(ctx, `SELECT attempts, max_attempts FROM job_queue WHERE id=$1`, id).Scan(&attempts, &maxAttempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	status := "failed"
	if attempts >= maxAttempts {
		status = "dead"
	}
	avail := time.Now().UTC().Add(retryAfter)
	_, err = q.DB.Exec(ctx, `
		UPDATE job_queue
		SET status=$2, last_error=$3, available_at=$4, updated_at=now(), locked_at=NULL, locked_by=NULL,
		    completed_at=CASE WHEN $2='dead' THEN now() ELSE NULL END
		WHERE id=$1
	`, id, status, errMsg, avail)
	return err
}

type Outbox struct {
	DB *pgxpool.Pool
}

func (o *Outbox) Write(ctx context.Context, storeID *uuid.UUID, eventType string, payload any, idempotencyKey string) error {
	if idempotencyKey == "" {
		idempotencyKey = uuid.NewString()
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = o.DB.Exec(ctx, `
		INSERT INTO outbox_events (store_id, event_type, payload, idempotency_key)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (idempotency_key) DO NOTHING
	`, storeID, eventType, raw, idempotencyKey)
	return err
}

// MarkPublished marks pending outbox rows as published (simple drain for MVP).
func (o *Outbox) MarkPublished(ctx context.Context, limit int) (int64, error) {
	if limit <= 0 {
		limit = 50
	}
	ct, err := o.DB.Exec(ctx, `
		UPDATE outbox_events SET published_at=now()
		WHERE id IN (
			SELECT id FROM outbox_events WHERE published_at IS NULL ORDER BY created_at LIMIT $1
		)
	`, limit)
	if err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}
