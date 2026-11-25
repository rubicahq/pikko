package syshealth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type IssueFilter struct {
	Range     Range
	Severity  string
	Status    string
	Source    string
	Module    string
	Query     string // request/issue/store/job id search
	Limit     int
	Offset    int
}

func (s *Service) ListIssues(ctx context.Context, f IssueFilter) ([]Issue, int, error) {
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 50
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	args := []any{f.Range.From, f.Range.To}
	where := []string{"last_seen_at >= $1 AND last_seen_at <= $2"}
	n := 3
	if f.Severity != "" {
		where = append(where, fmt.Sprintf("severity=$%d", n))
		args = append(args, f.Severity)
		n++
	}
	if f.Status != "" {
		where = append(where, fmt.Sprintf("status=$%d", n))
		args = append(args, f.Status)
		n++
	}
	if f.Source != "" {
		where = append(where, fmt.Sprintf("source=$%d", n))
		args = append(args, f.Source)
		n++
	}
	if f.Module != "" {
		where = append(where, fmt.Sprintf("module=$%d", n))
		args = append(args, f.Module)
		n++
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		where = append(where, fmt.Sprintf(`(
			id::text = $%d OR fingerprint = $%d OR title ILIKE $%d
			OR EXISTS (
				SELECT 1 FROM system_error_occurrences o
				WHERE o.issue_id = system_error_issues.id
				  AND (o.request_id = $%d OR o.store_id::text = $%d OR o.id::text = $%d)
			)
		)`, n, n, n+1, n, n, n))
		args = append(args, q, "%"+q+"%")
		n += 2
	}
	clause := strings.Join(where, " AND ")
	var total int
	if err := s.DB.QueryRow(ctx, `SELECT COUNT(*) FROM system_error_issues WHERE `+clause, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, f.Limit, f.Offset)
	rows, err := s.DB.Query(ctx, `
		SELECT id, fingerprint, title, severity, source, module, error_code, status,
		       first_seen_at, last_seen_at, occurrence_count, affected_store_count,
		       deploy_version, admin_notes, resolved_at
		FROM system_error_issues
		WHERE `+clause+`
		ORDER BY last_seen_at DESC
		LIMIT $`+fmt.Sprint(n)+` OFFSET $`+fmt.Sprint(n+1), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []Issue
	for rows.Next() {
		var it Issue
		if err := rows.Scan(
			&it.ID, &it.Fingerprint, &it.Title, &it.Severity, &it.Source, &it.Module, &it.ErrorCode, &it.Status,
			&it.FirstSeenAt, &it.LastSeenAt, &it.OccurrenceCount, &it.AffectedStoreCount,
			&it.DeployVersion, &it.AdminNotes, &it.ResolvedAt,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, it)
	}
	if out == nil {
		out = []Issue{}
	}
	return out, total, nil
}

func (s *Service) listIssues(ctx context.Context, r Range, severity, status string, limit int) ([]Issue, error) {
	items, _, err := s.ListIssues(ctx, IssueFilter{Range: r, Severity: severity, Status: status, Limit: limit})
	return items, err
}

func (s *Service) GetIssue(ctx context.Context, id uuid.UUID) (*Issue, []Occurrence, error) {
	var it Issue
	err := s.DB.QueryRow(ctx, `
		SELECT id, fingerprint, title, severity, source, module, error_code, status,
		       first_seen_at, last_seen_at, occurrence_count, affected_store_count,
		       deploy_version, admin_notes, resolved_at
		FROM system_error_issues WHERE id=$1
	`, id).Scan(
		&it.ID, &it.Fingerprint, &it.Title, &it.Severity, &it.Source, &it.Module, &it.ErrorCode, &it.Status,
		&it.FirstSeenAt, &it.LastSeenAt, &it.OccurrenceCount, &it.AffectedStoreCount,
		&it.DeployVersion, &it.AdminNotes, &it.ResolvedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	rows, err := s.DB.Query(ctx, `
		SELECT id, issue_id, occurred_at, environment, source, severity, module, error_code,
		       message, store_id, user_id, request_id, endpoint, http_status, duration_ms,
		       stack, deploy_version, context
		FROM system_error_occurrences WHERE issue_id=$1
		ORDER BY occurred_at DESC LIMIT 50
	`, id)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var occ []Occurrence
	for rows.Next() {
		var o Occurrence
		var ctxRaw []byte
		if err := rows.Scan(
			&o.ID, &o.IssueID, &o.OccurredAt, &o.Environment, &o.Source, &o.Severity, &o.Module, &o.ErrorCode,
			&o.Message, &o.StoreID, &o.UserID, &o.RequestID, &o.Endpoint, &o.HTTPStatus, &o.DurationMs,
			&o.Stack, &o.DeployVersion, &ctxRaw,
		); err != nil {
			return nil, nil, err
		}
		_ = json.Unmarshal(ctxRaw, &o.Context)
		if o.Context == nil {
			o.Context = map[string]any{}
		}
		occ = append(occ, o)
	}
	if occ == nil {
		occ = []Occurrence{}
	}
	return &it, occ, nil
}

func (s *Service) PatchIssue(ctx context.Context, id uuid.UUID, status *string, notes *string, actor uuid.UUID) (*Issue, error) {
	it, _, err := s.GetIssue(ctx, id)
	if err != nil {
		return nil, err
	}
	if status != nil {
		switch *status {
		case StatusUnresolved, StatusInvestigating, StatusResolved, StatusIgnored:
			it.Status = *status
		default:
			return nil, fmt.Errorf("invalid status")
		}
	}
	if notes != nil {
		n := SanitizeText(*notes)
		it.AdminNotes = &n
	}
	var resolvedAt any
	var resolvedBy any
	if status != nil && (*status == StatusResolved || *status == StatusIgnored) {
		now := time.Now().UTC()
		it.ResolvedAt = &now
		resolvedAt = now
		resolvedBy = actor
	} else if status != nil && *status == StatusUnresolved {
		it.ResolvedAt = nil
		resolvedAt = nil
		resolvedBy = nil
	} else {
		resolvedAt = it.ResolvedAt
		resolvedBy = nil
	}
	_, err = s.DB.Exec(ctx, `
		UPDATE system_error_issues SET
			status=$2, admin_notes=$3, resolved_at=$4,
			resolved_by=COALESCE($5, resolved_by), updated_at=now()
		WHERE id=$1
	`, id, it.Status, it.AdminNotes, resolvedAt, resolvedBy)
	if err != nil {
		return nil, err
	}
	return it, nil
}

var ErrNotFound = errors.New("not found")
