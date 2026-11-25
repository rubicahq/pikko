package syshealth

import (
	"context"
	"time"
)

func (s *Service) Usage(ctx context.Context, r Range) (*UsageSummary, error) {
	out := &UsageSummary{
		Range:       r,
		GeneratedAt: time.Now().UTC(),
		Unavailable: []string{},
		AICostNote:  "Cost estimate unavailable — no billing rates configured.",
		EmailCostNote: "Cost estimate unavailable — no email billing rates configured.",
	}

	var aiCalls, aiOK int
	var tokens *int64
	err := s.DB.QueryRow(ctx, `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE success),
		       CASE WHEN COUNT(total_tokens) = 0 THEN NULL
		            ELSE SUM(total_tokens)::bigint END
		FROM ai_usage_events
		WHERE occurred_at >= $1 AND occurred_at <= $2
	`, r.From, r.To).Scan(&aiCalls, &aiOK, &tokens)
	if err != nil {
		return nil, err
	}
	out.AICalls = &aiCalls
	out.AISuccessCalls = &aiOK
	if tokens == nil {
		out.Unavailable = append(out.Unavailable, "ai_tokens")
	} else {
		out.AITokens = tokens
	}
	out.Unavailable = append(out.Unavailable, "ai_cost_estimate")
	out.Unavailable = append(out.Unavailable, "email_cost_estimate")

	var bytes *int64
	var files int
	err = s.DB.QueryRow(ctx, `
		SELECT COALESCE(SUM(size_bytes),0), COUNT(*) FROM stored_files
	`).Scan(&bytes, &files)
	if err != nil {
		return nil, err
	}
	out.StorageBytes = bytes
	out.StorageFiles = &files

	var sent, failed int
	err = s.DB.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE status='sent'),
			COUNT(*) FILTER (WHERE status='failed')
		FROM email_jobs
		WHERE created_at >= $1 AND created_at <= $2
	`, r.From, r.To).Scan(&sent, &failed)
	if err != nil {
		return nil, err
	}
	out.EmailSent = &sent
	out.EmailFailed = &failed
	return out, nil
}
