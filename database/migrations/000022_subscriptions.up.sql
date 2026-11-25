-- Store subscriptions, plan catalog, entitlements, usage, overrides (testing mode).

CREATE TABLE subscription_plans (
    code            TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    rank            INT NOT NULL UNIQUE,
    description     TEXT NOT NULL DEFAULT '',
    price_monthly   BIGINT NOT NULL CHECK (price_monthly >= 0),
    price_annual    BIGINT NOT NULL CHECK (price_annual >= 0),
    currency        TEXT NOT NULL DEFAULT 'IDR',
    recommended     BOOLEAN NOT NULL DEFAULT false,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE subscription_plan_features (
    plan_code       TEXT NOT NULL REFERENCES subscription_plans(code) ON DELETE CASCADE,
    feature_key     TEXT NOT NULL,
    enabled         BOOLEAN NOT NULL DEFAULT true,
    PRIMARY KEY (plan_code, feature_key)
);

CREATE TABLE subscription_plan_limits (
    plan_code       TEXT NOT NULL REFERENCES subscription_plans(code) ON DELETE CASCADE,
    limit_key       TEXT NOT NULL,
    limit_value     BIGINT NOT NULL, -- -1 = unlimited
    PRIMARY KEY (plan_code, limit_key)
);

CREATE TABLE store_subscriptions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id        UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    plan_code       TEXT NOT NULL REFERENCES subscription_plans(code),
    billing_cycle   TEXT NOT NULL CHECK (billing_cycle IN ('monthly', 'annual')),
    status          TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'past_due', 'expired', 'canceled', 'trialing')),
    is_test         BOOLEAN NOT NULL DEFAULT false,
    activation_mode TEXT NOT NULL DEFAULT 'test_bypass'
        CHECK (activation_mode IN ('test_bypass', 'trial', 'paid', 'admin_grant', 'migration')),
    activated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    current_period_start TIMESTAMPTZ NOT NULL DEFAULT now(),
    current_period_end   TIMESTAMPTZ,
    canceled_at     TIMESTAMPTZ,
    trial_ends_at   TIMESTAMPTZ,
    auto_renew      BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT store_subscriptions_store_unique UNIQUE (store_id)
);

CREATE INDEX store_subscriptions_plan_idx ON store_subscriptions (plan_code);
CREATE INDEX store_subscriptions_status_idx ON store_subscriptions (status);

CREATE TABLE store_subscription_payments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id        UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    subscription_id UUID NOT NULL REFERENCES store_subscriptions(id) ON DELETE CASCADE,
    amount          BIGINT NOT NULL DEFAULT 0,
    currency        TEXT NOT NULL DEFAULT 'IDR',
    status          TEXT NOT NULL DEFAULT 'succeeded'
        CHECK (status IN ('succeeded', 'failed', 'pending')),
    payment_kind    TEXT NOT NULL DEFAULT 'internal_simulation'
        CHECK (payment_kind IN ('internal_simulation', 'gateway', 'manual')),
    is_test         BOOLEAN NOT NULL DEFAULT true,
    exclude_from_revenue BOOLEAN NOT NULL DEFAULT true,
    provider_ref    TEXT,
    metadata        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX store_subscription_payments_store_idx ON store_subscription_payments (store_id, created_at DESC);

CREATE TABLE store_subscription_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id        UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    subscription_id UUID REFERENCES store_subscriptions(id) ON DELETE SET NULL,
    actor_user_id   UUID REFERENCES users(id) ON DELETE SET NULL,
    event_type      TEXT NOT NULL,
    from_plan       TEXT,
    to_plan         TEXT,
    from_cycle      TEXT,
    to_cycle        TEXT,
    metadata        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX store_subscription_events_store_idx ON store_subscription_events (store_id, created_at DESC);

CREATE TABLE store_feature_overrides (
    store_id        UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    feature_key     TEXT NOT NULL,
    enabled         BOOLEAN NOT NULL,
    reason          TEXT,
    updated_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (store_id, feature_key)
);

CREATE TABLE store_limit_overrides (
    store_id        UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    limit_key       TEXT NOT NULL,
    limit_value     BIGINT NOT NULL,
    reason          TEXT,
    updated_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (store_id, limit_key)
);

-- Jakarta calendar month period as YYYY-MM string in Asia/Jakarta.
CREATE TABLE store_usage_counters (
    store_id        UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    period_ym       TEXT NOT NULL,
    limit_key       TEXT NOT NULL,
    used_value      BIGINT NOT NULL DEFAULT 0 CHECK (used_value >= 0),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (store_id, period_ym, limit_key)
);

CREATE TABLE platform_subscription_config (
    key             TEXT PRIMARY KEY,
    value           JSONB NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by      UUID REFERENCES users(id) ON DELETE SET NULL
);

ALTER TABLE onboarding_drafts
    DROP CONSTRAINT IF EXISTS onboarding_drafts_step_check;

ALTER TABLE onboarding_drafts
    ADD CONSTRAINT onboarding_drafts_step_check CHECK (step IN (1, 2, 3, 4));

ALTER TABLE onboarding_drafts
    ADD COLUMN IF NOT EXISTS plan_code TEXT REFERENCES subscription_plans(code),
    ADD COLUMN IF NOT EXISTS billing_cycle TEXT
        CHECK (billing_cycle IS NULL OR billing_cycle IN ('monthly', 'annual'));

-- Seed plans
INSERT INTO subscription_plans (code, name, rank, description, price_monthly, price_annual, recommended) VALUES
    ('basic', 'Basic', 1, 'Full sell and fulfill journey for solo sellers.', 49000, 490000, false),
    ('standard', 'Standard', 2, 'Campaigns, team, AI catalog import, and analytics.', 99000, 990000, false),
    ('premium', 'Premium', 3, 'Live ops, purchasing, WhatsApp templates, and higher limits.', 199000, 1990000, true),
    ('ultimate', 'Ultimate', 4, 'Highest limits for growing multi-person stores.', 399000, 3990000, false);

-- Feature matrix (higher inherits lower; only released product features)
-- Basic: sell + fulfill core
INSERT INTO subscription_plan_features (plan_code, feature_key, enabled) VALUES
    ('basic', 'storefront', true),
    ('basic', 'products', true),
    ('basic', 'catalogs', true),
    ('basic', 'catalog_flipbook', true),
    ('basic', 'orders', true),
    ('basic', 'checkout', true),
    ('basic', 'payments', true),
    ('basic', 'finance', true),
    ('basic', 'refunds', true),
    ('basic', 'packing', true),
    ('basic', 'shipments', true),
    ('basic', 'notifications', true),
    ('basic', 'customer_portal', true),
    ('basic', 'themes', true),
    ('basic', 'branding', true),
    ('basic', 'multi_store', true),
    ('basic', 'campaigns', false),
    ('basic', 'preorders', false),
    ('basic', 'business_rules', false),
    ('basic', 'ai_pdf_import', false),
    ('basic', 'analytics', false),
    ('basic', 'team', false),
    ('basic', 'whatsapp_templates', false),
    ('basic', 'purchasing', false),
    ('basic', 'receiving', false),
    ('basic', 'vendors', false),
    ('basic', 'live_shopping', false),
    ('basic', 'product_attributes', false);

INSERT INTO subscription_plan_features (plan_code, feature_key, enabled)
SELECT 'standard', feature_key,
    CASE feature_key
        WHEN 'campaigns' THEN true
        WHEN 'preorders' THEN true
        WHEN 'business_rules' THEN true
        WHEN 'ai_pdf_import' THEN true
        WHEN 'analytics' THEN true
        WHEN 'team' THEN true
        ELSE enabled
    END
FROM subscription_plan_features WHERE plan_code = 'basic';

INSERT INTO subscription_plan_features (plan_code, feature_key, enabled)
SELECT 'premium', feature_key,
    CASE feature_key
        WHEN 'whatsapp_templates' THEN true
        WHEN 'purchasing' THEN true
        WHEN 'receiving' THEN true
        WHEN 'vendors' THEN true
        WHEN 'live_shopping' THEN true
        WHEN 'product_attributes' THEN true
        ELSE enabled
    END
FROM subscription_plan_features WHERE plan_code = 'standard';

INSERT INTO subscription_plan_features (plan_code, feature_key, enabled)
SELECT 'ultimate', feature_key, enabled
FROM subscription_plan_features WHERE plan_code = 'premium';

-- Usage limits (-1 = unlimited)
INSERT INTO subscription_plan_limits (plan_code, limit_key, limit_value) VALUES
    ('basic', 'orders_per_month', 100),
    ('basic', 'active_products', 50),
    ('basic', 'catalogs', 2),
    ('basic', 'campaigns', 0),
    ('basic', 'team_members', 1),
    ('basic', 'storage_gb', 1),
    ('basic', 'ai_pdf_pages_per_month', 0),

    ('standard', 'orders_per_month', 500),
    ('standard', 'active_products', 250),
    ('standard', 'catalogs', 10),
    ('standard', 'campaigns', 5),
    ('standard', 'team_members', 3),
    ('standard', 'storage_gb', 5),
    ('standard', 'ai_pdf_pages_per_month', 100),

    ('premium', 'orders_per_month', 2000),
    ('premium', 'active_products', 1000),
    ('premium', 'catalogs', 50),
    ('premium', 'campaigns', 25),
    ('premium', 'team_members', 10),
    ('premium', 'storage_gb', 25),
    ('premium', 'ai_pdf_pages_per_month', 500),

    ('ultimate', 'orders_per_month', -1),
    ('ultimate', 'active_products', -1),
    ('ultimate', 'catalogs', -1),
    ('ultimate', 'campaigns', -1),
    ('ultimate', 'team_members', 50),
    ('ultimate', 'storage_gb', 100),
    ('ultimate', 'ai_pdf_pages_per_month', 5000);

INSERT INTO platform_subscription_config (key, value) VALUES
    ('flags', jsonb_build_object(
        'subscriptions_enabled', true,
        'plan_select_enabled', true,
        'test_bypass_enabled', true,
        'usage_enforce_enabled', true,
        'plan_change_enabled', true,
        'commercial_trial_enabled', false,
        'real_billing_enabled', false,
        'auto_renew_enabled', false
    )),
    ('trial', jsonb_build_object(
        'days', 30,
        'one_per_store', true,
        'all_plans', true
    ));

-- Existing stores without subscription → Ultimate testing activation
INSERT INTO store_subscriptions (
    store_id, plan_code, billing_cycle, status, is_test, activation_mode,
    activated_at, current_period_start, auto_renew
)
SELECT s.id, 'ultimate', 'monthly', 'active', true, 'migration',
       now(), now(), false
FROM stores s
WHERE NOT EXISTS (
    SELECT 1 FROM store_subscriptions ss WHERE ss.store_id = s.id
);

INSERT INTO store_subscription_events (store_id, subscription_id, event_type, to_plan, to_cycle, metadata)
SELECT ss.store_id, ss.id, 'migrated_to_ultimate_testing', 'ultimate', 'monthly',
       jsonb_build_object('source', '000022_subscriptions', 'is_test', true)
FROM store_subscriptions ss
WHERE ss.activation_mode = 'migration';

INSERT INTO store_subscription_payments (
    store_id, subscription_id, amount, currency, status, payment_kind, is_test, exclude_from_revenue, metadata
)
SELECT ss.store_id, ss.id, 0, 'IDR', 'succeeded', 'internal_simulation', true, true,
       jsonb_build_object('source', 'migration', 'mode', 'test_bypass')
FROM store_subscriptions ss
WHERE ss.activation_mode = 'migration';
