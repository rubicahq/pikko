-- Align subscription limits/features to locked user matrix (follow-up to 000022).
-- Idempotent upserts — safe if 000022 already applied.

UPDATE subscription_plans SET
    description = 'Full sell and fulfill journey for solo sellers.',
    updated_at = now()
WHERE code = 'basic';

UPDATE subscription_plans SET
    description = 'Team collab, remove branding, bulk import, and ops efficiency.',
    updated_at = now()
WHERE code = 'standard';

UPDATE subscription_plans SET
    description = 'Higher limits plus store credit and advanced controls.',
    updated_at = now()
WHERE code = 'premium';

UPDATE subscription_plans SET
    description = 'Highest usage limits for growing multi-person stores.',
    updated_at = now()
WHERE code = 'ultimate';

-- Usage limits (exact lock). Keys catalogs/campaigns = active catalogs/campaigns.
INSERT INTO subscription_plan_limits (plan_code, limit_key, limit_value) VALUES
    ('basic', 'orders_per_month', 100),
    ('basic', 'active_products', 200),
    ('basic', 'catalogs', 3),
    ('basic', 'campaigns', 2),
    ('basic', 'team_members', 1),
    ('basic', 'storage_gb', 1),
    ('basic', 'ai_pdf_pages_per_month', 10),

    ('standard', 'orders_per_month', 500),
    ('standard', 'active_products', 1000),
    ('standard', 'catalogs', 15),
    ('standard', 'campaigns', 10),
    ('standard', 'team_members', 3),
    ('standard', 'storage_gb', 3),
    ('standard', 'ai_pdf_pages_per_month', 50),

    ('premium', 'orders_per_month', 2000),
    ('premium', 'active_products', 5000),
    ('premium', 'catalogs', 50),
    ('premium', 'campaigns', 30),
    ('premium', 'team_members', 5),
    ('premium', 'storage_gb', 10),
    ('premium', 'ai_pdf_pages_per_month', 200),

    ('ultimate', 'orders_per_month', 10000),
    ('ultimate', 'active_products', 20000),
    ('ultimate', 'catalogs', 200),
    ('ultimate', 'campaigns', 100),
    ('ultimate', 'team_members', 15),
    ('ultimate', 'storage_gb', 30),
    ('ultimate', 'ai_pdf_pages_per_month', 500)
ON CONFLICT (plan_code, limit_key) DO UPDATE
SET limit_value = EXCLUDED.limit_value;

-- Feature matrix: higher inherits lower. Keys aligned to released product where possible;
-- unreleased keys are seeded for plan config but stay ungated in UI (see docs).
WITH feature_matrix(feature_key, basic_on, standard_on, premium_on, ultimate_on) AS (
    VALUES
        -- Basic = full sell + fulfill
        ('storefront', true, true, true, true),
        ('products', true, true, true, true),
        ('product_attributes', true, true, true, true),
        ('catalogs', true, true, true, true),
        ('catalog_flipbook', true, true, true, true),
        ('ai_pdf_import', true, true, true, true),
        ('campaigns', true, true, true, true),
        ('preorders', true, true, true, true),
        ('live_shopping', true, true, true, true),
        ('orders', true, true, true, true),
        ('checkout', true, true, true, true),
        ('payments', true, true, true, true),
        ('finance', true, true, true, true),
        ('refunds', true, true, true, true),
        ('purchasing', true, true, true, true),
        ('receiving', true, true, true, true),
        ('vendors', true, true, true, true),
        ('packing', true, true, true, true),
        ('shipments', true, true, true, true),
        ('customer_portal', true, true, true, true),
        ('themes', true, true, true, true),
        ('branding', true, true, true, true),
        ('multi_store', true, true, true, true),
        ('notifications', true, true, true, true),
        ('whatsapp_templates', true, true, true, true),
        ('business_rules', true, true, true, true),
        ('analytics', true, true, true, true),

        -- Standard+
        ('remove_branding', false, true, true, true),
        ('quick_order', false, true, true, true),
        ('bulk_import', false, true, true, true),
        ('packing_by_customer', false, true, true, true),
        ('consolidated_payments', false, true, true, true),
        ('whatsapp_templates_custom', false, true, true, true), -- unreleased UI
        ('team', false, true, true, true),

        -- Premium+
        ('custom_domain', false, false, true, true), -- unreleased
        ('bulk_ops', false, false, true, true),
        ('advanced_reconciliation', false, false, true, true), -- unreleased differentiation
        ('store_credit', false, false, true, true),
        ('advanced_rules', false, false, true, true), -- unreleased differentiation
        ('advanced_analytics', false, false, true, true), -- unreleased differentiation
        ('granular_permissions', false, false, true, true),

        -- Ultimate+
        ('merchant_audit_log', false, false, false, true), -- no merchant UI yet
        ('priority_support', false, false, false, true) -- ops flag only
),
plans(plan_code, rank) AS (
    VALUES ('basic', 1), ('standard', 2), ('premium', 3), ('ultimate', 4)
)
INSERT INTO subscription_plan_features (plan_code, feature_key, enabled)
SELECT p.plan_code, f.feature_key,
    CASE p.plan_code
        WHEN 'basic' THEN f.basic_on
        WHEN 'standard' THEN f.standard_on
        WHEN 'premium' THEN f.premium_on
        WHEN 'ultimate' THEN f.ultimate_on
    END
FROM feature_matrix f
CROSS JOIN plans p
ON CONFLICT (plan_code, feature_key) DO UPDATE
SET enabled = EXCLUDED.enabled;
