package syshealth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rubicahq/pikko/services/api/internal/email"
)

type Alerter struct {
	Health *Service
	Email  *email.Service
	From   string
	FromName string
}

func (s *Service) ListAlertRules(ctx context.Context) ([]AlertRule, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT id, name, enabled, rule_type, threshold, window_minutes, cooldown_minutes,
		       notify_emails, last_fired_at
		FROM hq_alert_rules ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AlertRule
	for rows.Next() {
		var r AlertRule
		if err := rows.Scan(&r.ID, &r.Name, &r.Enabled, &r.RuleType, &r.Threshold, &r.WindowMinutes,
			&r.CooldownMinutes, &r.NotifyEmails, &r.LastFiredAt); err != nil {
			return nil, err
		}
		if r.NotifyEmails == nil {
			r.NotifyEmails = []string{}
		}
		out = append(out, r)
	}
	if out == nil {
		out = []AlertRule{}
	}
	return out, nil
}

func (s *Service) PatchAlertRule(ctx context.Context, id uuid.UUID, enabled *bool, threshold, window, cooldown *int, emails []string) (*AlertRule, error) {
	rules, err := s.ListAlertRules(ctx)
	if err != nil {
		return nil, err
	}
	var found *AlertRule
	for i := range rules {
		if rules[i].ID == id {
			found = &rules[i]
			break
		}
	}
	if found == nil {
		return nil, ErrNotFound
	}
	if enabled != nil {
		found.Enabled = *enabled
	}
	if threshold != nil {
		found.Threshold = *threshold
	}
	if window != nil {
		found.WindowMinutes = *window
	}
	if cooldown != nil {
		found.CooldownMinutes = *cooldown
	}
	if emails != nil {
		found.NotifyEmails = emails
	}
	_, err = s.DB.Exec(ctx, `
		UPDATE hq_alert_rules SET
			enabled=$2, threshold=$3, window_minutes=$4, cooldown_minutes=$5,
			notify_emails=$6, updated_at=now()
		WHERE id=$1
	`, id, found.Enabled, found.Threshold, found.WindowMinutes, found.CooldownMinutes, found.NotifyEmails)
	if err != nil {
		return nil, err
	}
	return found, nil
}

func (s *Service) ListAlertEvents(ctx context.Context, limit int) ([]AlertEvent, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.DB.Query(ctx, `
		SELECT id, rule_id, rule_type, severity, title, message, status,
		       fired_at, recovered_at, notified_at, fingerprint, meta
		FROM hq_alert_events ORDER BY fired_at DESC LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AlertEvent
	for rows.Next() {
		var e AlertEvent
		var meta []byte
		if err := rows.Scan(&e.ID, &e.RuleID, &e.RuleType, &e.Severity, &e.Title, &e.Message, &e.Status,
			&e.FiredAt, &e.RecoveredAt, &e.NotifiedAt, &e.Fingerprint, &meta); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(meta, &e.Meta)
		if e.Meta == nil {
			e.Meta = map[string]any{}
		}
		out = append(out, e)
	}
	if out == nil {
		out = []AlertEvent{}
	}
	return out, nil
}

// EvaluateRules checks thresholds and creates/dedupes alert events. Optional email notify.
func (a *Alerter) EvaluateRules(ctx context.Context) error {
	if a.Health == nil || a.Health.DB == nil {
		return nil
	}
	rules, err := a.Health.ListAlertRules(ctx)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		if rule.LastFiredAt != nil && rule.CooldownMinutes > 0 {
			if now.Sub(*rule.LastFiredAt) < time.Duration(rule.CooldownMinutes)*time.Minute {
				continue
			}
		}
		windowFrom := now.Add(-time.Duration(rule.WindowMinutes) * time.Minute)
		r := Range{From: windowFrom, To: now, Key: "custom"}
		count, title, msg, sev, fp, err := a.measure(ctx, rule, r)
		if err != nil {
			a.Health.log().Error("alert_measure", "rule", rule.RuleType, "error", err)
			continue
		}
		if count < rule.Threshold {
			// recovery: close open events for this fingerprint
			_, _ = a.Health.DB.Exec(ctx, `
				UPDATE hq_alert_events SET status='resolved', recovered_at=now()
				WHERE fingerprint=$1 AND status='open'
			`, fp)
			continue
		}
		var existing uuid.UUID
		err = a.Health.DB.QueryRow(ctx, `
			SELECT id FROM hq_alert_events WHERE fingerprint=$1 AND status='open'
		`, fp).Scan(&existing)
		if err == nil {
			continue // already open — dedupe
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		meta, _ := json.Marshal(map[string]any{"count": count, "threshold": rule.Threshold})
		var eventID uuid.UUID
		err = a.Health.DB.QueryRow(ctx, `
			INSERT INTO hq_alert_events (rule_id, rule_type, severity, title, message, fingerprint, meta)
			VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb) RETURNING id
		`, rule.ID, rule.RuleType, sev, title, msg, fp, string(meta)).Scan(&eventID)
		if err != nil {
			continue
		}
		_, _ = a.Health.DB.Exec(ctx, `UPDATE hq_alert_rules SET last_fired_at=now() WHERE id=$1`, rule.ID)
		a.notify(ctx, rule, title, msg, eventID)
	}
	return nil
}

func (a *Alerter) measure(ctx context.Context, rule AlertRule, r Range) (count int, title, msg, sev, fp string, err error) {
	sev = SeverityError
	switch rule.RuleType {
	case "critical_issue":
		err = a.Health.DB.QueryRow(ctx, `
			SELECT COUNT(*) FROM system_error_issues
			WHERE severity='critical' AND status IN ('unresolved','investigating')
			  AND last_seen_at >= $1 AND last_seen_at <= $2
		`, r.From, r.To).Scan(&count)
		title = "Critical issues"
		msg = fmt.Sprintf("%d critical issue(s) in window", count)
		sev = SeverityCritical
		fp = "critical_issue"
	case "failed_jobs":
		count, _, err = a.Health.jobFailureCounts(ctx, r)
		title = "Failed jobs"
		msg = fmt.Sprintf("%d failed job(s) in window", count)
		fp = "failed_jobs"
	case "stuck_jobs":
		_, count, err = a.Health.jobFailureCounts(ctx, r)
		title = "Stuck jobs"
		msg = fmt.Sprintf("%d stuck job(s)", count)
		fp = "stuck_jobs"
	case "service_down":
		err = a.Health.DB.QueryRow(ctx, `
			SELECT COUNT(*) FROM system_service_checks WHERE status='unavailable'
		`).Scan(&count)
		title = "Service unavailable"
		msg = fmt.Sprintf("%d service(s) unavailable", count)
		sev = SeverityCritical
		fp = "service_down"
	case "error_spike":
		err = a.Health.DB.QueryRow(ctx, `
			SELECT COUNT(*) FROM system_error_occurrences
			WHERE occurred_at >= $1 AND occurred_at <= $2 AND severity IN ('error','critical')
		`, r.From, r.To).Scan(&count)
		title = "Error spike"
		msg = fmt.Sprintf("%d error occurrence(s) in window", count)
		fp = "error_spike"
	default:
		err = fmt.Errorf("unknown rule type")
	}
	return
}

func (a *Alerter) notify(ctx context.Context, rule AlertRule, title, msg string, eventID uuid.UUID) {
	if a.Email == nil {
		return
	}
	for _, to := range rule.NotifyEmails {
		if to == "" {
			continue
		}
		m := email.Message{
			TemplateKey:    "hq_alert",
			To:             to,
			From:           a.From,
			FromName:       a.FromName,
			Subject:        "[Pikko HQ] " + title,
			HTML:           "<p>" + SanitizeText(msg) + "</p>",
			Text:           SanitizeText(msg),
			IdempotencyKey: "hq-alert:" + eventID.String() + ":" + to,
		}
		if err := a.Email.Enqueue(ctx, m); err != nil {
			a.Health.log().Error("hq_alert_email", "error", err)
			continue
		}
	}
	_, _ = a.Health.DB.Exec(ctx, `UPDATE hq_alert_events SET notified_at=now() WHERE id=$1`, eventID)
}
