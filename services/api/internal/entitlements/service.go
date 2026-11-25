package entitlements

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rubicahq/pikko/services/api/internal/config"
)

type Service struct {
	DB  *pgxpool.Pool
	Cfg *config.Config
}

type Plan struct {
	Code         string           `json:"code"`
	Name         string           `json:"name"`
	Rank         int              `json:"rank"`
	Description  string           `json:"description"`
	PriceMonthly int64            `json:"price_monthly"`
	PriceAnnual  int64            `json:"price_annual"`
	Currency     string           `json:"currency"`
	Recommended  bool             `json:"recommended"`
	IsActive     bool             `json:"is_active"`
	Features     map[string]bool  `json:"features"`
	Limits       map[string]int64 `json:"limits"`
}

type Subscription struct {
	ID                   uuid.UUID  `json:"id"`
	StoreID              uuid.UUID  `json:"store_id"`
	PlanCode             string     `json:"plan_code"`
	BillingCycle         string     `json:"billing_cycle"`
	Status               string     `json:"status"`
	IsTest               bool       `json:"is_test"`
	ActivationMode       string     `json:"activation_mode"`
	ActivatedAt          time.Time  `json:"activated_at"`
	CurrentPeriodStart   time.Time  `json:"current_period_start"`
	CurrentPeriodEnd     *time.Time `json:"current_period_end,omitempty"`
	CanceledAt           *time.Time `json:"canceled_at,omitempty"`
	TrialEndsAt          *time.Time `json:"trial_ends_at,omitempty"`
	AutoRenew            bool       `json:"auto_renew"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
	PlanName             string     `json:"plan_name,omitempty"`
	PlanRank             int        `json:"plan_rank,omitempty"`
}

type UsageEntry struct {
	LimitKey string `json:"limit_key"`
	Used     int64  `json:"used"`
	Limit    int64  `json:"limit"`
	PeriodYM string `json:"period_ym"`
}

type EntitlementSnapshot struct {
	Subscription *Subscription    `json:"subscription"`
	Plan         *Plan            `json:"plan,omitempty"`
	Features     map[string]bool  `json:"features"`
	Limits       map[string]int64 `json:"limits"`
	Usage        []UsageEntry     `json:"usage"`
	Flags        Flags            `json:"flags"`
	CanCreate    bool             `json:"can_create"` // false when expired/missing (ops fulfill still ok)
	IsExpired    bool             `json:"is_expired"`
}

func jakartaNow() time.Time {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return time.Now().UTC()
	}
	return time.Now().In(loc)
}

func PeriodYM(t time.Time) string {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return t.UTC().Format("2006-01")
	}
	return t.In(loc).Format("2006-01")
}

func (s *Service) ListPlans(ctx context.Context, activeOnly bool) ([]Plan, error) {
	q := `
		SELECT code, name, rank, description, price_monthly, price_annual, currency, recommended, is_active
		FROM subscription_plans
	`
	if activeOnly {
		q += ` WHERE is_active=true`
	}
	q += ` ORDER BY rank ASC`
	rows, err := s.DB.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var plans []Plan
	for rows.Next() {
		var p Plan
		if err := rows.Scan(&p.Code, &p.Name, &p.Rank, &p.Description, &p.PriceMonthly, &p.PriceAnnual, &p.Currency, &p.Recommended, &p.IsActive); err != nil {
			return nil, err
		}
		p.Features = map[string]bool{}
		p.Limits = map[string]int64{}
		plans = append(plans, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range plans {
		feats, err := s.planFeatures(ctx, plans[i].Code)
		if err != nil {
			return nil, err
		}
		plans[i].Features = feats
		lims, err := s.planLimits(ctx, plans[i].Code)
		if err != nil {
			return nil, err
		}
		plans[i].Limits = lims
	}
	return plans, nil
}

func (s *Service) GetPlan(ctx context.Context, code string) (*Plan, error) {
	var p Plan
	err := s.DB.QueryRow(ctx, `
		SELECT code, name, rank, description, price_monthly, price_annual, currency, recommended, is_active
		FROM subscription_plans WHERE code=$1
	`, code).Scan(&p.Code, &p.Name, &p.Rank, &p.Description, &p.PriceMonthly, &p.PriceAnnual, &p.Currency, &p.Recommended, &p.IsActive)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	p.Features, err = s.planFeatures(ctx, code)
	if err != nil {
		return nil, err
	}
	p.Limits, err = s.planLimits(ctx, code)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Service) planFeatures(ctx context.Context, code string) (map[string]bool, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT feature_key, enabled FROM subscription_plan_features WHERE plan_code=$1
	`, code)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var k string
		var en bool
		if err := rows.Scan(&k, &en); err != nil {
			return nil, err
		}
		out[k] = en
	}
	return out, rows.Err()
}

func (s *Service) planLimits(ctx context.Context, code string) (map[string]int64, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT limit_key, limit_value FROM subscription_plan_limits WHERE plan_code=$1
	`, code)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var k string
		var v int64
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}

func (s *Service) GetSubscription(ctx context.Context, storeID uuid.UUID) (*Subscription, error) {
	var sub Subscription
	err := s.DB.QueryRow(ctx, `
		SELECT ss.id, ss.store_id, ss.plan_code, ss.billing_cycle, ss.status, ss.is_test, ss.activation_mode,
		       ss.activated_at, ss.current_period_start, ss.current_period_end, ss.canceled_at, ss.trial_ends_at,
		       ss.auto_renew, ss.created_at, ss.updated_at, p.name, p.rank
		FROM store_subscriptions ss
		JOIN subscription_plans p ON p.code = ss.plan_code
		WHERE ss.store_id=$1
	`, storeID).Scan(
		&sub.ID, &sub.StoreID, &sub.PlanCode, &sub.BillingCycle, &sub.Status, &sub.IsTest, &sub.ActivationMode,
		&sub.ActivatedAt, &sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &sub.CanceledAt, &sub.TrialEndsAt,
		&sub.AutoRenew, &sub.CreatedAt, &sub.UpdatedAt, &sub.PlanName, &sub.PlanRank,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

func (s *Service) Snapshot(ctx context.Context, storeID uuid.UUID) (*EntitlementSnapshot, error) {
	flags, err := s.LoadFlags(ctx)
	if err != nil {
		return nil, err
	}
	snap := &EntitlementSnapshot{
		Features:  map[string]bool{},
		Limits:    map[string]int64{},
		Usage:     []UsageEntry{},
		Flags:     flags,
		CanCreate: false,
	}
	sub, err := s.GetSubscription(ctx, storeID)
	if errors.Is(err, ErrNotFound) {
		return snap, nil
	}
	if err != nil {
		return nil, err
	}
	snap.Subscription = sub
	expired := sub.Status == "expired" || sub.Status == "canceled"
	if sub.CurrentPeriodEnd != nil && sub.CurrentPeriodEnd.Before(time.Now().UTC()) && sub.Status == "active" {
		expired = true
	}
	snap.IsExpired = expired
	snap.CanCreate = sub.Status == "active" || sub.Status == "trialing"
	if expired {
		snap.CanCreate = false
	}

	plan, err := s.GetPlan(ctx, sub.PlanCode)
	if err != nil {
		return nil, err
	}
	snap.Plan = plan

	feats := map[string]bool{}
	for k, v := range plan.Features {
		feats[k] = v
	}
	lims := map[string]int64{}
	for k, v := range plan.Limits {
		lims[k] = v
	}

	// Store overrides
	frows, err := s.DB.Query(ctx, `SELECT feature_key, enabled FROM store_feature_overrides WHERE store_id=$1`, storeID)
	if err != nil {
		return nil, err
	}
	for frows.Next() {
		var k string
		var en bool
		if err := frows.Scan(&k, &en); err != nil {
			frows.Close()
			return nil, err
		}
		feats[k] = en
	}
	frows.Close()

	lrows, err := s.DB.Query(ctx, `SELECT limit_key, limit_value FROM store_limit_overrides WHERE store_id=$1`, storeID)
	if err != nil {
		return nil, err
	}
	for lrows.Next() {
		var k string
		var v int64
		if err := lrows.Scan(&k, &v); err != nil {
			lrows.Close()
			return nil, err
		}
		lims[k] = v
	}
	lrows.Close()

	if !flags.SubscriptionsEnabled {
		// When globally off, treat as fully entitled (escape hatch).
		for _, k := range AllFeatureKeys {
			feats[k] = true
		}
		for _, k := range AllLimitKeys {
			lims[k] = Unlimited
		}
		snap.CanCreate = true
	}

	snap.Features = feats
	snap.Limits = lims

	period := PeriodYM(jakartaNow())
	usage, err := s.usageForPeriod(ctx, storeID, period, lims)
	if err != nil {
		return nil, err
	}
	snap.Usage = usage
	return snap, nil
}

func (s *Service) usageForPeriod(ctx context.Context, storeID uuid.UUID, period string, lims map[string]int64) ([]UsageEntry, error) {
	used := map[string]int64{}
	rows, err := s.DB.Query(ctx, `
		SELECT limit_key, used_value FROM store_usage_counters
		WHERE store_id=$1 AND period_ym=$2
	`, storeID, period)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var k string
		var v int64
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		used[k] = v
	}
	// Live gauges for non-monthly counters (active products, catalogs, campaigns, team, storage).
	live, err := s.liveGauges(ctx, storeID)
	if err != nil {
		return nil, err
	}
	for k, v := range live {
		used[k] = v
	}
	out := make([]UsageEntry, 0, len(AllLimitKeys))
	for _, k := range AllLimitKeys {
		lim := lims[k]
		out = append(out, UsageEntry{
			LimitKey: k,
			Used:     used[k],
			Limit:    lim,
			PeriodYM: period,
		})
	}
	return out, nil
}

func (s *Service) liveGauges(ctx context.Context, storeID uuid.UUID) (map[string]int64, error) {
	out := map[string]int64{}
	var n int64
	if err := s.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM products WHERE store_id=$1 AND status='published'
	`, storeID).Scan(&n); err != nil {
		return nil, err
	}
	out[LimitActiveProducts] = n

	if err := s.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM catalogs WHERE store_id=$1 AND status <> 'archived'
	`, storeID).Scan(&n); err != nil {
		return nil, err
	}
	out[LimitCatalogs] = n

	if err := s.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM campaigns
		WHERE store_id=$1 AND operational_status NOT IN ('cancelled', 'completed')
	`, storeID).Scan(&n); err != nil {
		return nil, err
	}
	out[LimitCampaigns] = n

	if err := s.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM store_memberships WHERE store_id=$1 AND status='active'
	`, storeID).Scan(&n); err != nil {
		return nil, err
	}
	out[LimitTeamMembers] = n

	if err := s.DB.QueryRow(ctx, `
		SELECT COALESCE(SUM(size_bytes),0) FROM stored_files WHERE store_id=$1
	`, storeID).Scan(&n); err != nil {
		return nil, err
	}
	gb := n / (1024 * 1024 * 1024)
	if n > 0 && n%(1024*1024*1024) != 0 {
		gb++
	}
	if n == 0 {
		gb = 0
	}
	out[LimitStorageGB] = gb
	return out, nil
}

// RequireFeature fails if the store cannot use the feature.
func (s *Service) RequireFeature(ctx context.Context, storeID uuid.UUID, featureKey string) error {
	snap, err := s.Snapshot(ctx, storeID)
	if err != nil {
		return err
	}
	if !snap.Flags.SubscriptionsEnabled {
		return nil
	}
	if snap.Subscription == nil {
		return ErrSubscriptionRequired
	}
	if !snap.CanCreate && !isOpsSafeFeature(featureKey) {
		return ErrSubscriptionRequired
	}
	if !snap.Features[featureKey] {
		return &FeatureError{FeatureKey: featureKey}
	}
	return nil
}

func isOpsSafeFeature(key string) bool {
	switch key {
	case FeatureOrders, FeaturePayments, FeatureFinance, FeatureRefunds,
		FeaturePacking, FeatureShipments, FeatureCustomerPortal, FeatureNotifications:
		return true
	default:
		return false
	}
}

// RequireLimit checks usage against limit; for monthly counters, current used must be < limit
// before increment. For gauges, used must be < limit before create.
func (s *Service) RequireLimit(ctx context.Context, storeID uuid.UUID, limitKey string, delta int64) error {
	snap, err := s.Snapshot(ctx, storeID)
	if err != nil {
		return err
	}
	if !snap.Flags.SubscriptionsEnabled || !snap.Flags.UsageEnforceEnabled {
		return nil
	}
	if snap.Subscription == nil || !snap.CanCreate {
		return ErrSubscriptionRequired
	}
	lim, ok := snap.Limits[limitKey]
	if !ok {
		return nil
	}
	if lim == Unlimited {
		return nil
	}
	var used int64
	for _, u := range snap.Usage {
		if u.LimitKey == limitKey {
			used = u.Used
			break
		}
	}
	if used+delta > lim {
		return &LimitError{
			LimitKey: limitKey,
			Used:     used,
			Limit:    lim,
			Message:  fmt.Sprintf("Plan limit reached for %s (%d / %d).", limitKey, used, lim),
		}
	}
	return nil
}

// IncrementUsage adds delta to a monthly counter (orders, AI pages).
func (s *Service) IncrementUsage(ctx context.Context, storeID uuid.UUID, limitKey string, delta int64) error {
	if delta == 0 {
		return nil
	}
	period := PeriodYM(jakartaNow())
	_, err := s.DB.Exec(ctx, `
		INSERT INTO store_usage_counters (store_id, period_ym, limit_key, used_value, updated_at)
		VALUES ($1,$2,$3,$4,now())
		ON CONFLICT (store_id, period_ym, limit_key)
		DO UPDATE SET used_value = store_usage_counters.used_value + EXCLUDED.used_value, updated_at=now()
	`, storeID, period, limitKey, delta)
	return err
}

type ActivateInput struct {
	PlanCode     string `json:"plan_code"`
	BillingCycle string `json:"billing_cycle"`
}

// ActivateTestBypass activates or replaces subscription with Rp0 simulated payment.
func (s *Service) ActivateTestBypass(ctx context.Context, storeID, actorID uuid.UUID, in ActivateInput) (*Subscription, error) {
	flags, err := s.LoadFlags(ctx)
	if err != nil {
		return nil, err
	}
	if !flags.SubscriptionsEnabled {
		return nil, ErrSubscriptionsOff
	}
	if !flags.TestBypassEnabled {
		return nil, ErrTestBypassDisabled
	}
	if in.PlanCode == "" || (in.BillingCycle != "monthly" && in.BillingCycle != "annual") {
		return nil, ErrInvalidInput
	}
	plan, err := s.GetPlan(ctx, in.PlanCode)
	if err != nil {
		return nil, err
	}
	if !plan.IsActive {
		return nil, ErrInvalidInput
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var existingID *uuid.UUID
	var fromPlan, fromCycle *string
	var eid uuid.UUID
	var fp, fc string
	err = tx.QueryRow(ctx, `
		SELECT id, plan_code, billing_cycle FROM store_subscriptions WHERE store_id=$1 FOR UPDATE
	`, storeID).Scan(&eid, &fp, &fc)
	if err == nil {
		existingID = &eid
		fromPlan = &fp
		fromCycle = &fc
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	now := time.Now().UTC()
	var periodEnd *time.Time
	if in.BillingCycle == "monthly" {
		t := now.AddDate(0, 1, 0)
		periodEnd = &t
	} else {
		t := now.AddDate(1, 0, 0)
		periodEnd = &t
	}

	var subID uuid.UUID
	if existingID != nil {
		err = tx.QueryRow(ctx, `
			UPDATE store_subscriptions SET
				plan_code=$2, billing_cycle=$3, status='active', is_test=true, activation_mode='test_bypass',
				activated_at=$4, current_period_start=$4, current_period_end=$5, canceled_at=NULL,
				auto_renew=false, updated_at=now()
			WHERE store_id=$1
			RETURNING id
		`, storeID, in.PlanCode, in.BillingCycle, now, periodEnd).Scan(&subID)
	} else {
		err = tx.QueryRow(ctx, `
			INSERT INTO store_subscriptions (
				store_id, plan_code, billing_cycle, status, is_test, activation_mode,
				activated_at, current_period_start, current_period_end, auto_renew
			) VALUES ($1,$2,$3,'active',true,'test_bypass',$4,$4,$5,false)
			RETURNING id
		`, storeID, in.PlanCode, in.BillingCycle, now, periodEnd).Scan(&subID)
	}
	if err != nil {
		return nil, err
	}

	meta, _ := json.Marshal(map[string]any{
		"mode": "test_bypass", "amount": 0, "is_test": true, "internal_simulation": true,
	})
	_, err = tx.Exec(ctx, `
		INSERT INTO store_subscription_payments (
			store_id, subscription_id, amount, currency, status, payment_kind, is_test, exclude_from_revenue, metadata
		) VALUES ($1,$2,0,'IDR','succeeded','internal_simulation',true,true,$3)
	`, storeID, subID, meta)
	if err != nil {
		return nil, err
	}

	eventType := "activated_test_bypass"
	if existingID != nil {
		eventType = "changed_test_bypass"
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO store_subscription_events (
			store_id, subscription_id, actor_user_id, event_type, from_plan, to_plan, from_cycle, to_cycle, metadata
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`, storeID, subID, actorID, eventType, fromPlan, in.PlanCode, fromCycle, in.BillingCycle, meta)
	if err != nil {
		return nil, err
	}

	_, _ = tx.Exec(ctx, `
		INSERT INTO audit_logs (actor_user_id, store_id, action, subject_type, subject_id, metadata)
		VALUES ($1,$2,'subscription_activated','subscription',$3,$4)
	`, actorID, storeID, subID.String(), meta)

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetSubscription(ctx, storeID)
}

// ChangePlan upgrades/downgrades with over-limit warning block for creates (no auto-delete).
func (s *Service) ChangePlan(ctx context.Context, storeID, actorID uuid.UUID, in ActivateInput, force bool) (*Subscription, []UsageEntry, error) {
	flags, err := s.LoadFlags(ctx)
	if err != nil {
		return nil, nil, err
	}
	if !flags.PlanChangeEnabled {
		return nil, nil, ErrPlanChangeDisabled
	}
	if flags.TestBypassEnabled {
		// In testing mode, change uses same zero-charge path but checks downgrade gauges.
		target, err := s.GetPlan(ctx, in.PlanCode)
		if err != nil {
			return nil, nil, err
		}
		snap, err := s.Snapshot(ctx, storeID)
		if err != nil {
			return nil, nil, err
		}
		over := []UsageEntry{}
		if snap.Subscription != nil && target.Rank < snap.Subscription.PlanRank {
			for _, u := range snap.Usage {
				lim := target.Limits[u.LimitKey]
				if lim == Unlimited {
					continue
				}
				if u.Used > lim {
					over = append(over, UsageEntry{
						LimitKey: u.LimitKey,
						Used:     u.Used,
						Limit:    lim,
						PeriodYM: u.PeriodYM,
					})
				}
			}
			if len(over) > 0 && !force {
				return nil, over, ErrDowngradeBlocked
			}
			// force=true: allow downgrade but over-limit blocks new creates (existing kept)
		}
		sub, err := s.ActivateTestBypass(ctx, storeID, actorID, in)
		return sub, over, err
	}
	if !flags.RealBillingEnabled {
		return nil, nil, ErrRealBillingDisabled
	}
	return nil, nil, ErrRealBillingDisabled
}

type AdminPlanPatch struct {
	PriceMonthly *int64  `json:"price_monthly"`
	PriceAnnual  *int64  `json:"price_annual"`
	Description  *string `json:"description"`
	Recommended  *bool   `json:"recommended"`
	IsActive     *bool   `json:"is_active"`
	Features     map[string]bool  `json:"features"`
	Limits       map[string]int64 `json:"limits"`
}

func (s *Service) AdminUpdatePlan(ctx context.Context, code string, patch AdminPlanPatch, actor *uuid.UUID) (*Plan, error) {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	sets := []string{"updated_at=now()"}
	args := []any{code}
	n := 2
	if patch.PriceMonthly != nil {
		sets = append(sets, fmt.Sprintf("price_monthly=$%d", n))
		args = append(args, *patch.PriceMonthly)
		n++
	}
	if patch.PriceAnnual != nil {
		sets = append(sets, fmt.Sprintf("price_annual=$%d", n))
		args = append(args, *patch.PriceAnnual)
		n++
	}
	if patch.Description != nil {
		sets = append(sets, fmt.Sprintf("description=$%d", n))
		args = append(args, *patch.Description)
		n++
	}
	if patch.Recommended != nil {
		sets = append(sets, fmt.Sprintf("recommended=$%d", n))
		args = append(args, *patch.Recommended)
		n++
	}
	if patch.IsActive != nil {
		sets = append(sets, fmt.Sprintf("is_active=$%d", n))
		args = append(args, *patch.IsActive)
		n++
	}
	q := fmt.Sprintf(`UPDATE subscription_plans SET %s WHERE code=$1`, joinComma(sets))
	tag, err := tx.Exec(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	for k, en := range patch.Features {
		_, err := tx.Exec(ctx, `
			INSERT INTO subscription_plan_features (plan_code, feature_key, enabled)
			VALUES ($1,$2,$3)
			ON CONFLICT (plan_code, feature_key) DO UPDATE SET enabled=EXCLUDED.enabled
		`, code, k, en)
		if err != nil {
			return nil, err
		}
	}
	for k, v := range patch.Limits {
		_, err := tx.Exec(ctx, `
			INSERT INTO subscription_plan_limits (plan_code, limit_key, limit_value)
			VALUES ($1,$2,$3)
			ON CONFLICT (plan_code, limit_key) DO UPDATE SET limit_value=EXCLUDED.limit_value
		`, code, k, v)
		if err != nil {
			return nil, err
		}
	}
	meta, _ := json.Marshal(patch)
	_, _ = tx.Exec(ctx, `
		INSERT INTO audit_logs (actor_user_id, store_id, action, subject_type, subject_id, metadata)
		VALUES ($1,NULL,'subscription_plan_updated','subscription_plan',$2,$3)
	`, actor, code, meta)

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetPlan(ctx, code)
}

func (s *Service) AdminSetFlags(ctx context.Context, flags Flags, actor *uuid.UUID) (Flags, error) {
	raw, err := json.Marshal(flags)
	if err != nil {
		return flags, err
	}
	_, err = s.DB.Exec(ctx, `
		INSERT INTO platform_subscription_config (key, value, updated_at, updated_by)
		VALUES ('flags', $1::jsonb, now(), $2)
		ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value, updated_at=now(), updated_by=EXCLUDED.updated_by
	`, raw, actor)
	if err != nil {
		return flags, err
	}
	return s.LoadFlags(ctx)
}

func (s *Service) SetStoreOverride(ctx context.Context, storeID uuid.UUID, featureKey string, enabled bool, reason string, actor *uuid.UUID) error {
	_, err := s.DB.Exec(ctx, `
		INSERT INTO store_feature_overrides (store_id, feature_key, enabled, reason, updated_by, updated_at)
		VALUES ($1,$2,$3,$4,$5,now())
		ON CONFLICT (store_id, feature_key) DO UPDATE
		  SET enabled=EXCLUDED.enabled, reason=EXCLUDED.reason, updated_by=EXCLUDED.updated_by, updated_at=now()
	`, storeID, featureKey, enabled, reason, actor)
	return err
}

func (s *Service) SetStoreLimitOverride(ctx context.Context, storeID uuid.UUID, limitKey string, value int64, reason string, actor *uuid.UUID) error {
	_, err := s.DB.Exec(ctx, `
		INSERT INTO store_limit_overrides (store_id, limit_key, limit_value, reason, updated_by, updated_at)
		VALUES ($1,$2,$3,$4,$5,now())
		ON CONFLICT (store_id, limit_key) DO UPDATE
		  SET limit_value=EXCLUDED.limit_value, reason=EXCLUDED.reason, updated_by=EXCLUDED.updated_by, updated_at=now()
	`, storeID, limitKey, value, reason, actor)
	return err
}

func joinComma(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += ", "
		}
		out += p
	}
	return out
}
