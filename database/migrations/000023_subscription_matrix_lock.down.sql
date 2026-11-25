-- Revert feature/limit seeds to 000022 progressive defaults (does not drop tables).

UPDATE subscription_plans SET
    description = 'Full sell and fulfill journey for solo sellers.',
    updated_at = now()
WHERE code = 'basic';
UPDATE subscription_plans SET
    description = 'Campaigns, team, AI catalog import, and analytics.',
    updated_at = now()
WHERE code = 'standard';
UPDATE subscription_plans SET
    description = 'Live ops, purchasing, WhatsApp templates, and higher limits.',
    updated_at = now()
WHERE code = 'premium';
UPDATE subscription_plans SET
    description = 'Highest limits for growing multi-person stores.',
    updated_at = now()
WHERE code = 'ultimate';

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
    ('ultimate', 'ai_pdf_pages_per_month', 5000)
ON CONFLICT (plan_code, limit_key) DO UPDATE
SET limit_value = EXCLUDED.limit_value;

-- Restore 000022 feature enablement for original keys; leave newer keys as-is (disabled on basic).
UPDATE subscription_plan_features SET enabled = false
WHERE plan_code = 'basic'
  AND feature_key IN (
    'campaigns', 'preorders', 'business_rules', 'ai_pdf_import', 'analytics', 'team',
    'whatsapp_templates', 'purchasing', 'receiving', 'vendors', 'live_shopping', 'product_attributes',
    'remove_branding', 'quick_order', 'bulk_import', 'packing_by_customer', 'consolidated_payments',
    'whatsapp_templates_custom', 'custom_domain', 'bulk_ops', 'advanced_reconciliation',
    'store_credit', 'advanced_rules', 'advanced_analytics', 'granular_permissions',
    'merchant_audit_log', 'priority_support'
  );

UPDATE subscription_plan_features f SET enabled = true
FROM subscription_plan_features b
WHERE f.plan_code = 'standard' AND b.plan_code = 'basic' AND f.feature_key = b.feature_key
  AND f.feature_key IN (
    'storefront', 'products', 'catalogs', 'catalog_flipbook', 'orders', 'checkout', 'payments',
    'finance', 'refunds', 'packing', 'shipments', 'notifications', 'customer_portal', 'themes',
    'branding', 'multi_store', 'campaigns', 'preorders', 'business_rules', 'ai_pdf_import',
    'analytics', 'team'
  );

UPDATE subscription_plan_features SET enabled = true
WHERE plan_code = 'standard'
  AND feature_key IN (
    'campaigns', 'preorders', 'business_rules', 'ai_pdf_import', 'analytics', 'team'
  );

UPDATE subscription_plan_features SET enabled = false
WHERE plan_code = 'standard'
  AND feature_key IN (
    'whatsapp_templates', 'purchasing', 'receiving', 'vendors', 'live_shopping', 'product_attributes'
  );

UPDATE subscription_plan_features SET enabled = true
WHERE plan_code IN ('premium', 'ultimate')
  AND feature_key IN (
    'storefront', 'products', 'catalogs', 'catalog_flipbook', 'orders', 'checkout', 'payments',
    'finance', 'refunds', 'packing', 'shipments', 'notifications', 'customer_portal', 'themes',
    'branding', 'multi_store', 'campaigns', 'preorders', 'business_rules', 'ai_pdf_import',
    'analytics', 'team', 'whatsapp_templates', 'purchasing', 'receiving', 'vendors',
    'live_shopping', 'product_attributes'
  );
