package jobs

import (
	"context"
	"log/slog"
	"time"
)

type Handler func(ctx context.Context, job *Job) error

type EmailDrainer interface {
	DrainPending(ctx context.Context, limit int) (int, error)
}

type Worker struct {
	Queue    *Queue
	Outbox   *Outbox
	Email    EmailDrainer
	Logger   *slog.Logger
	WorkerID string
	Types    []string
	Handlers map[string]Handler
	Interval time.Duration
}

func (w *Worker) Run(ctx context.Context) {
	if w.Interval <= 0 {
		w.Interval = 500 * time.Millisecond
	}
	if w.WorkerID == "" {
		w.WorkerID = "api-worker"
	}
	ticker := time.NewTicker(w.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *Worker) tick(ctx context.Context) {
	if w.Outbox != nil {
		_, _ = w.Outbox.MarkPublished(ctx, 50)
	}
	if w.Email != nil {
		_, _ = w.Email.DrainPending(ctx, 20)
	}
	job, err := w.Queue.ClaimNext(ctx, w.WorkerID, w.Types)
	if err != nil {
		if w.Logger != nil {
			w.Logger.Error("job_claim_failed", "error", err)
		}
		return
	}
	if job == nil {
		return
	}
	h := w.Handlers[job.JobType]
	if h == nil {
		_ = w.Queue.Fail(ctx, job.ID, "no handler for job type", time.Minute)
		return
	}
	if err := h(ctx, job); err != nil {
		if w.Logger != nil {
			w.Logger.Error("job_failed", "job_id", job.ID, "type", job.JobType, "error", err)
		}
		_ = w.Queue.Fail(ctx, job.ID, err.Error(), 10*time.Second)
		return
	}
	_ = w.Queue.Complete(ctx, job.ID)
}

// ProcessOnce claims and runs a single job (tests).
func (w *Worker) ProcessOnce(ctx context.Context) (bool, error) {
	job, err := w.Queue.ClaimNext(ctx, w.WorkerID, w.Types)
	if err != nil {
		return false, err
	}
	if job == nil {
		return false, nil
	}
	h := w.Handlers[job.JobType]
	if h == nil {
		_ = w.Queue.Fail(ctx, job.ID, "no handler", time.Minute)
		return true, nil
	}
	if err := h(ctx, job); err != nil {
		_ = w.Queue.Fail(ctx, job.ID, err.Error(), time.Second)
		return true, err
	}
	return true, w.Queue.Complete(ctx, job.ID)
}
