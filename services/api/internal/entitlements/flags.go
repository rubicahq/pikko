package entitlements

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
)

type Flags struct {
	SubscriptionsEnabled   bool `json:"subscriptions_enabled"`
	PlanSelectEnabled      bool `json:"plan_select_enabled"`
	TestBypassEnabled      bool `json:"test_bypass_enabled"`
	UsageEnforceEnabled    bool `json:"usage_enforce_enabled"`
	PlanChangeEnabled      bool `json:"plan_change_enabled"`
	CommercialTrialEnabled bool `json:"commercial_trial_enabled"`
	RealBillingEnabled     bool `json:"real_billing_enabled"`
	AutoRenewEnabled       bool `json:"auto_renew_enabled"`
}

func DefaultFlags() Flags {
	return Flags{
		SubscriptionsEnabled:   true,
		PlanSelectEnabled:      true,
		TestBypassEnabled:      true,
		UsageEnforceEnabled:    true,
		PlanChangeEnabled:      true,
		CommercialTrialEnabled: false,
		RealBillingEnabled:     false,
		AutoRenewEnabled:       false,
	}
}

func (s *Service) LoadFlags(ctx context.Context) (Flags, error) {
	f := DefaultFlags()
	// Env overrides take precedence for kill switches.
	if s.Cfg != nil {
		f.SubscriptionsEnabled = s.Cfg.FeatureSubscriptions
		f.PlanSelectEnabled = s.Cfg.FeatureSubscriptionPlanSelect
		f.TestBypassEnabled = s.Cfg.FeatureSubscriptionTestBypass
		f.UsageEnforceEnabled = s.Cfg.FeatureSubscriptionUsageEnforce
		f.PlanChangeEnabled = s.Cfg.FeatureSubscriptionPlanChange
		f.CommercialTrialEnabled = s.Cfg.FeatureCommercialTrial
		f.RealBillingEnabled = s.Cfg.FeatureRealBilling
		f.AutoRenewEnabled = s.Cfg.FeatureSubscriptionAutoRenew
	}
	var raw []byte
	err := s.DB.QueryRow(ctx, `
		SELECT value FROM platform_subscription_config WHERE key='flags'
	`).Scan(&raw)
	if errorsIsNoRows(err) {
		return f, nil
	}
	if err != nil {
		return f, err
	}
	var db Flags
	if err := json.Unmarshal(raw, &db); err != nil {
		return f, nil
	}
	// Env false always wins as kill switch; env true can still be turned off in DB.
	merge := func(env, dbv bool) bool {
		if !env {
			return false
		}
		return dbv
	}
	if s.Cfg != nil {
		f.SubscriptionsEnabled = merge(s.Cfg.FeatureSubscriptions, db.SubscriptionsEnabled)
		f.PlanSelectEnabled = merge(s.Cfg.FeatureSubscriptionPlanSelect, db.PlanSelectEnabled)
		f.TestBypassEnabled = merge(s.Cfg.FeatureSubscriptionTestBypass, db.TestBypassEnabled)
		f.UsageEnforceEnabled = merge(s.Cfg.FeatureSubscriptionUsageEnforce, db.UsageEnforceEnabled)
		f.PlanChangeEnabled = merge(s.Cfg.FeatureSubscriptionPlanChange, db.PlanChangeEnabled)
		f.CommercialTrialEnabled = merge(s.Cfg.FeatureCommercialTrial, db.CommercialTrialEnabled)
		f.RealBillingEnabled = merge(s.Cfg.FeatureRealBilling, db.RealBillingEnabled)
		f.AutoRenewEnabled = merge(s.Cfg.FeatureSubscriptionAutoRenew, db.AutoRenewEnabled)
	} else {
		f = db
	}
	return f, nil
}

func errorsIsNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
