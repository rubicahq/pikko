package syshealth

import (
	"time"

	"github.com/google/uuid"
)

const (
	SeverityCritical = "critical"
	SeverityError    = "error"
	SeverityWarning  = "warning"
	SeverityInfo     = "info"

	StatusUnresolved    = "unresolved"
	StatusInvestigating = "investigating"
	StatusResolved      = "resolved"
	StatusIgnored       = "ignored"

	ServiceOperational = "operational"
	ServiceDegraded    = "degraded"
	ServiceUnavailable = "unavailable"
	ServiceUnknown     = "unknown"
)

type Event struct {
	OccurredAt    time.Time
	Environment   string
	Source        string
	Severity      string
	Module        string
	ErrorCode     string
	Message       string
	StoreID       *uuid.UUID
	UserID        *uuid.UUID
	RequestID     string
	Endpoint      string
	HTTPStatus    *int
	DurationMs    *int
	Stack         string
	DeployVersion string
	Context       map[string]any
}

type Issue struct {
	ID                 uuid.UUID  `json:"id"`
	Fingerprint        string     `json:"fingerprint"`
	Title              string     `json:"title"`
	Severity           string     `json:"severity"`
	Source             string     `json:"source"`
	Module             string     `json:"module"`
	ErrorCode          string     `json:"error_code"`
	Status             string     `json:"status"`
	FirstSeenAt        time.Time  `json:"first_seen_at"`
	LastSeenAt         time.Time  `json:"last_seen_at"`
	OccurrenceCount    int        `json:"occurrence_count"`
	AffectedStoreCount int        `json:"affected_store_count"`
	DeployVersion      *string    `json:"deploy_version,omitempty"`
	AdminNotes         *string    `json:"admin_notes,omitempty"`
	ResolvedAt         *time.Time `json:"resolved_at,omitempty"`
}

type Occurrence struct {
	ID            uuid.UUID      `json:"id"`
	IssueID       uuid.UUID      `json:"issue_id"`
	OccurredAt    time.Time      `json:"occurred_at"`
	Environment   string         `json:"environment"`
	Source        string         `json:"source"`
	Severity      string         `json:"severity"`
	Module        string         `json:"module"`
	ErrorCode     string         `json:"error_code"`
	Message       string         `json:"message"`
	StoreID       *uuid.UUID     `json:"store_id,omitempty"`
	UserID        *uuid.UUID     `json:"user_id,omitempty"`
	RequestID     *string        `json:"request_id,omitempty"`
	Endpoint      *string        `json:"endpoint,omitempty"`
	HTTPStatus    *int           `json:"http_status,omitempty"`
	DurationMs    *int           `json:"duration_ms,omitempty"`
	Stack         *string        `json:"stack,omitempty"`
	DeployVersion *string        `json:"deploy_version,omitempty"`
	Context       map[string]any `json:"context"`
}

type Range struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
	Key  string    `json:"key"` // 1h|24h|7d|30d|custom
}

type Overview struct {
	Range              Range          `json:"range"`
	GeneratedAt        time.Time      `json:"generated_at"`
	DeployVersion      *string        `json:"deploy_version"`
	ServiceHealth      []ServiceRow   `json:"service_health"`
	UnresolvedIssues   *int           `json:"unresolved_issues"`
	CriticalIssues     *int           `json:"critical_issues"`
	AffectedStores     *int           `json:"affected_stores"`
	APISuccessRate     *float64       `json:"api_success_rate"` // 0-100, nil if no measured requests
	APIRequestCount    *int           `json:"api_request_count"`
	P95LatencyMs       *float64       `json:"p95_latency_ms"`
	FailedJobs         *int           `json:"failed_jobs"`
	StuckJobs          *int           `json:"stuck_jobs"`
	Charts             OverviewCharts `json:"charts"`
	RecentCritical     []Issue        `json:"recent_critical_issues"`
	RecentFailedJobs   []JobRow       `json:"recent_failed_jobs"`
	MostAffectedStores []StoreImpact  `json:"most_affected_stores"`
	Unavailable        []string       `json:"unavailable"` // KPI keys that could not be measured
}

type OverviewCharts struct {
	ErrorOccurrences []TimeCount   `json:"error_occurrences"`
	APIErrorRate     []TimeRate    `json:"api_error_rate"`
	APILatency       []TimeLatency `json:"api_latency"`
	ErrorsByModule   []NamedCount  `json:"errors_by_module"`
	Severity         []NamedCount  `json:"severity"`
}

type TimeCount struct {
	T     time.Time `json:"t"`
	Count int       `json:"count"`
}

type TimeRate struct {
	T          time.Time `json:"t"`
	Rate       *float64  `json:"rate"` // percent; nil if no requests in bucket
	Errors     int       `json:"errors"`
	Requests   int       `json:"requests"`
}

type TimeLatency struct {
	T      time.Time `json:"t"`
	AvgMs  *float64  `json:"avg_ms"`
	MaxMs  *int      `json:"max_ms"`
	P95Ms  *float64  `json:"p95_ms"`
}

type NamedCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type StoreImpact struct {
	StoreID   uuid.UUID `json:"store_id"`
	StoreName *string   `json:"store_name,omitempty"`
	StoreSlug *string   `json:"store_slug,omitempty"`
	Count     int       `json:"count"`
}

type ServiceRow struct {
	Key        string         `json:"key"`
	Name       string         `json:"name"`
	Status     string         `json:"status"`
	Detail     *string        `json:"detail,omitempty"`
	CheckedAt  *time.Time     `json:"checked_at,omitempty"`
	LatencyMs  *int           `json:"latency_ms,omitempty"`
	Meta       map[string]any `json:"meta,omitempty"`
}

type JobRow struct {
	ID          uuid.UUID  `json:"id"`
	Queue       string     `json:"queue"` // job_queue | email_jobs | catalog_import
	JobType     string     `json:"job_type"`
	Status      string     `json:"status"`
	StoreID     *uuid.UUID `json:"store_id,omitempty"`
	Attempts    int        `json:"attempts"`
	MaxAttempts *int       `json:"max_attempts,omitempty"`
	LastError   *string    `json:"last_error,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Stuck       bool       `json:"stuck"`
}

type JobStats struct {
	Queued    int      `json:"queued"`
	Running   int      `json:"running"`
	Success   int      `json:"success"`
	Failed    int      `json:"failed"`
	Stuck     int      `json:"stuck"`
	Retrying  int      `json:"retrying"`
	Dead      int      `json:"dead"`
	AvgDurMs  *float64 `json:"avg_duration_ms"`
	Range     Range    `json:"range"`
}

type UsageSummary struct {
	Range            Range          `json:"range"`
	GeneratedAt      time.Time      `json:"generated_at"`
	AICalls          *int           `json:"ai_calls"`
	AISuccessCalls   *int           `json:"ai_success_calls"`
	AITokens         *int64         `json:"ai_tokens"` // nil if tokens never recorded
	AICostEstimate   *float64       `json:"ai_cost_estimate"`
	AICostNote       string         `json:"ai_cost_note"`
	StorageBytes     *int64         `json:"storage_bytes"`
	StorageFiles     *int           `json:"storage_files"`
	EmailSent        *int           `json:"email_sent"`
	EmailFailed      *int           `json:"email_failed"`
	EmailCostEstimate *float64      `json:"email_cost_estimate"`
	EmailCostNote    string         `json:"email_cost_note"`
	Unavailable      []string       `json:"unavailable"`
}

type AlertRule struct {
	ID               uuid.UUID  `json:"id"`
	Name             string     `json:"name"`
	Enabled          bool       `json:"enabled"`
	RuleType         string     `json:"rule_type"`
	Threshold        int        `json:"threshold"`
	WindowMinutes    int        `json:"window_minutes"`
	CooldownMinutes   int        `json:"cooldown_minutes"`
	NotifyEmails     []string   `json:"notify_emails"`
	LastFiredAt      *time.Time `json:"last_fired_at,omitempty"`
}

type AlertEvent struct {
	ID          uuid.UUID      `json:"id"`
	RuleID      *uuid.UUID     `json:"rule_id,omitempty"`
	RuleType    string         `json:"rule_type"`
	Severity    string         `json:"severity"`
	Title       string         `json:"title"`
	Message     string         `json:"message"`
	Status      string         `json:"status"`
	FiredAt     time.Time      `json:"fired_at"`
	RecoveredAt *time.Time     `json:"recovered_at,omitempty"`
	NotifiedAt  *time.Time     `json:"notified_at,omitempty"`
	Fingerprint string         `json:"fingerprint"`
	Meta        map[string]any `json:"meta"`
}
