ALTER TABLE onboarding_drafts DROP COLUMN IF EXISTS billing_cycle;
ALTER TABLE onboarding_drafts DROP COLUMN IF EXISTS plan_code;

ALTER TABLE onboarding_drafts DROP CONSTRAINT IF EXISTS onboarding_drafts_step_check;
ALTER TABLE onboarding_drafts
    ADD CONSTRAINT onboarding_drafts_step_check CHECK (step IN (1, 2));

DROP TABLE IF EXISTS store_usage_counters;
DROP TABLE IF EXISTS store_limit_overrides;
DROP TABLE IF EXISTS store_feature_overrides;
DROP TABLE IF EXISTS store_subscription_events;
DROP TABLE IF EXISTS store_subscription_payments;
DROP TABLE IF EXISTS store_subscriptions;
DROP TABLE IF EXISTS platform_subscription_config;
DROP TABLE IF EXISTS subscription_plan_limits;
DROP TABLE IF EXISTS subscription_plan_features;
DROP TABLE IF EXISTS subscription_plans;
