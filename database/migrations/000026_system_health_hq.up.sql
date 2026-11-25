-- HQ BO allowlist + System Health persistence (internal Tim Pikko only)

CREATE TABLE hq_admins (
    email_normalized TEXT PRIMARY KEY,
    user_id          UUID REFERENCES users(id) ON DELETE SET NULL,
    display_email    TEXT NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO hq_admins (email_normalized, display_email)
VALUES ('rickyhilmi@gmail.com', 'rickyhilmi@gmail.com')
ON CONFLICT (email_normalized) DO NOTHING;

CREATE TABLE system_error_issues (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    fingerprint       TEXT NOT NULL,
    title             TEXT NOT NULL,
    severity          TEXT NOT NULL CHECK (severity IN ('critical', 'error', 'warning', 'info')),
    source            TEXT NOT NULL,
    module            TEXT NOT NULL DEFAULT '',
    error_code        TEXT NOT NULL DEFAULT '',
    status            TEXT NOT NULL DEFAULT 'unresolved'
        CHECK (status IN ('unresolved', 'investigating', 'resolved', 'ignored')),
    first_seen_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    occurrence_count  INT NOT NULL DEFAULT 1 CHECK (occurrence_count >= 0),
    affected_store_count INT NOT NULL DEFAULT 0 CHECK (affected_store_count >= 0),
    deploy_version    TEXT,
    admin_notes       TEXT,
    resolved_at       TIMESTAMPTZ,
    resolved_by       UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT system_error_issues_fingerprint_unique UNIQUE (fingerprint)
);

CREATE INDEX system_error_issues_status_last_idx
    ON system_error_issues (status, last_seen_at DESC);
CREATE INDEX system_error_issues_severity_last_idx
    ON system_error_issues (severity, last_seen_at DESC);

CREATE TABLE system_error_occurrences (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    issue_id        UUID NOT NULL REFERENCES system_error_issues(id) ON DELETE CASCADE,
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    environment     TEXT NOT NULL DEFAULT 'production',
    source          TEXT NOT NULL,
    severity        TEXT NOT NULL CHECK (severity IN ('critical', 'error', 'warning', 'info')),
    module          TEXT NOT NULL DEFAULT '',
    error_code      TEXT NOT NULL DEFAULT '',
    message         TEXT NOT NULL,
    store_id        UUID REFERENCES stores(id) ON DELETE SET NULL,
    user_id         UUID REFERENCES users(id) ON DELETE SET NULL,
    request_id      TEXT,
    endpoint        TEXT,
    http_status     INT,
    duration_ms     INT,
    stack           TEXT,
    deploy_version  TEXT,
    context         JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX system_error_occurrences_issue_time_idx
    ON system_error_occurrences (issue_id, occurred_at DESC);
CREATE INDEX system_error_occurrences_time_idx
    ON system_error_occurrences (occurred_at DESC);
CREATE INDEX system_error_occurrences_request_idx
    ON system_error_occurrences (request_id)
    WHERE request_id IS NOT NULL AND request_id <> '';
CREATE INDEX system_error_occurrences_store_idx
    ON system_error_occurrences (store_id, occurred_at DESC)
    WHERE store_id IS NOT NULL;

-- Per-minute API aggregates (never invent zeros for uninstrumented windows — UI uses measured buckets only)
CREATE TABLE api_metric_minutes (
    bucket_ts           TIMESTAMPTZ NOT NULL,
    request_count       INT NOT NULL DEFAULT 0 CHECK (request_count >= 0),
    success_count       INT NOT NULL DEFAULT 0 CHECK (success_count >= 0),
    client_error_count  INT NOT NULL DEFAULT 0 CHECK (client_error_count >= 0),
    server_error_count  INT NOT NULL DEFAULT 0 CHECK (server_error_count >= 0),
    latency_sum_ms      BIGINT NOT NULL DEFAULT 0 CHECK (latency_sum_ms >= 0),
    latency_max_ms      INT NOT NULL DEFAULT 0 CHECK (latency_max_ms >= 0),
    PRIMARY KEY (bucket_ts)
);

CREATE INDEX api_metric_minutes_ts_idx ON api_metric_minutes (bucket_ts DESC);

-- Latency samples for P50/P95 (short retention; cleanup job)
CREATE TABLE api_latency_samples (
    id          BIGSERIAL PRIMARY KEY,
    sampled_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    duration_ms INT NOT NULL CHECK (duration_ms >= 0),
    status      INT NOT NULL
);

CREATE INDEX api_latency_samples_time_idx ON api_latency_samples (sampled_at DESC);

CREATE TABLE system_service_checks (
    service_key     TEXT PRIMARY KEY,
    display_name    TEXT NOT NULL,
    status          TEXT NOT NULL CHECK (status IN ('operational', 'degraded', 'unavailable', 'unknown')),
    detail          TEXT,
    checked_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    latency_ms      INT,
    meta            JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE TABLE hq_alert_rules (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL,
    enabled         BOOLEAN NOT NULL DEFAULT true,
    rule_type       TEXT NOT NULL CHECK (rule_type IN (
        'critical_issue', 'error_spike', 'failed_jobs', 'stuck_jobs', 'service_down'
    )),
    threshold       INT NOT NULL DEFAULT 1 CHECK (threshold >= 0),
    window_minutes  INT NOT NULL DEFAULT 15 CHECK (window_minutes > 0),
    cooldown_minutes INT NOT NULL DEFAULT 60 CHECK (cooldown_minutes >= 0),
    notify_emails   TEXT[] NOT NULL DEFAULT ARRAY['rickyhilmi@gmail.com']::TEXT[],
    last_fired_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO hq_alert_rules (name, rule_type, threshold, window_minutes, cooldown_minutes)
VALUES
    ('Critical issue opened', 'critical_issue', 1, 15, 60),
    ('Failed jobs threshold', 'failed_jobs', 5, 60, 120),
    ('Stuck jobs threshold', 'stuck_jobs', 3, 60, 120),
    ('Service unavailable', 'service_down', 1, 15, 30);

CREATE TABLE hq_alert_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id         UUID REFERENCES hq_alert_rules(id) ON DELETE SET NULL,
    rule_type       TEXT NOT NULL,
    severity        TEXT NOT NULL DEFAULT 'error'
        CHECK (severity IN ('critical', 'error', 'warning', 'info')),
    title           TEXT NOT NULL,
    message         TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'open'
        CHECK (status IN ('open', 'acknowledged', 'resolved')),
    fired_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    recovered_at    TIMESTAMPTZ,
    notified_at     TIMESTAMPTZ,
    fingerprint     TEXT NOT NULL,
    meta            JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX hq_alert_events_open_fingerprint_unique
    ON hq_alert_events (fingerprint)
    WHERE status = 'open';

CREATE INDEX hq_alert_events_fired_idx ON hq_alert_events (fired_at DESC);

-- Optional AI usage counters (call counts only; cost may be unavailable)
CREATE TABLE ai_usage_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    store_id        UUID REFERENCES stores(id) ON DELETE SET NULL,
    provider        TEXT NOT NULL DEFAULT '',
    model           TEXT NOT NULL DEFAULT '',
    operation       TEXT NOT NULL DEFAULT 'catalog_extract',
    success         BOOLEAN NOT NULL DEFAULT true,
    prompt_tokens   INT,
    completion_tokens INT,
    total_tokens    INT,
    duration_ms     INT,
    error_code      TEXT,
    meta            JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX ai_usage_events_time_idx ON ai_usage_events (occurred_at DESC);
CREATE INDEX ai_usage_events_store_idx ON ai_usage_events (store_id, occurred_at DESC)
    WHERE store_id IS NOT NULL;

CREATE TABLE hq_admin_audit (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_user_id   UUID REFERENCES users(id) ON DELETE SET NULL,
    actor_email     TEXT NOT NULL,
    action          TEXT NOT NULL,
    subject_type    TEXT,
    subject_id      TEXT,
    meta            JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX hq_admin_audit_time_idx ON hq_admin_audit (created_at DESC);
