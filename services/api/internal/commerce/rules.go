package commerce

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rubicahq/pikko/services/api/internal/money"
)

const (
	RuleHandlingFee   = "handling_fee"
	RuleMinimumOrder  = "minimum_order"
	RuleOrderApproval = "order_approval"

	ScopeStore         = "store"
	ScopeSellingModel  = "selling_model"
	ScopeBatch         = "batch"

	FeeModeFixed       = "fixed"
	FeeModePercent     = "percent"
	FeeModeConditional = "conditional" // ordered subtotal tiers; first match wins
)

var (
	ErrMinimumNotMet     = errors.New("minimum order not met")
	ErrBatchClosed       = errors.New("preorder batch closed")
	ErrBatchNotOpen      = errors.New("preorder batch not open")
	ErrQuotaExceeded     = errors.New("preorder quota exceeded")
	ErrIncompatibleCart  = errors.New("incompatible cart group")
	ErrApprovalRequired  = errors.New("approval required")
)

// HandlingFeeTier is one conditional handling-fee row (spreadsheet-style).
// Match when product subtotal > MinSubtotalIDR; first matching tier in list order wins.
type HandlingFeeTier struct {
	MinSubtotalIDR int64 `json:"min_subtotal_idr"`
	FeeFixedIDR    int64 `json:"fee_fixed_idr"`
}

type RuleConfig struct {
	FeeMode         string             `json:"fee_mode,omitempty"` // fixed|percent|conditional
	FeeFixedIDR     *int64             `json:"fee_fixed_idr,omitempty"`
	FeePercent      *float64           `json:"fee_percent,omitempty"`
	Tiers           []HandlingFeeTier  `json:"tiers,omitempty"` // conditional mode only; ordered
	MinSubtotalIDR  *int64             `json:"min_subtotal_idr,omitempty"`
	MinQuantity     *int               `json:"min_quantity,omitempty"`
	RequireApproval *bool              `json:"require_approval,omitempty"`
}

type BusinessRule struct {
	ID                uuid.UUID       `json:"id"`
	StoreID           uuid.UUID       `json:"store_id"`
	RuleType          string          `json:"rule_type"`
	Name              string          `json:"name"`
	ScopeType         string          `json:"scope_type"`
	ScopeSellingModel *string         `json:"scope_selling_model,omitempty"`
	ScopeBatchID      *uuid.UUID      `json:"scope_batch_id,omitempty"`
	IsEnabled         bool            `json:"is_enabled"`
	CurrentVersion    int             `json:"current_version"`
	Config            RuleConfig      `json:"config"`
	VersionID         *uuid.UUID      `json:"version_id,omitempty"`
	CreatedAt         string          `json:"created_at,omitempty"`
	UpdatedAt         string          `json:"updated_at,omitempty"`
}

type UpsertRuleInput struct {
	RuleType          string     `json:"rule_type"`
	Name              string     `json:"name"`
	ScopeType         string     `json:"scope_type"`
	ScopeSellingModel *string    `json:"scope_selling_model,omitempty"`
	ScopeBatchID      *uuid.UUID `json:"scope_batch_id,omitempty"`
	IsEnabled         *bool      `json:"is_enabled,omitempty"`
	Config            RuleConfig `json:"config"`
}

type AppliedRule struct {
	RuleID      uuid.UUID       `json:"rule_id"`
	VersionID   uuid.UUID       `json:"version_id"`
	RuleType    string          `json:"rule_type"`
	ScopeType   string          `json:"scope_type"`
	Config      RuleConfig      `json:"config"`
	Result      json.RawMessage `json:"result"`
}

type RulesEvaluation struct {
	HandlingFeeIDR    int64         `json:"handling_fee_idr"`
	RequiresApproval  bool          `json:"requires_approval"`
	MinimumOK         bool          `json:"minimum_ok"`
	MinimumMessage    string        `json:"minimum_message,omitempty"`
	Applied           []AppliedRule `json:"applied"`
	ShippingFeeIDR    int64         `json:"shipping_fee_idr"`
	SubtotalIDR       int64         `json:"subtotal_idr"`
	TotalIDR          int64         `json:"total_idr"`
	ItemQuantity      int           `json:"item_quantity"`
	SellingModel      string        `json:"selling_model"`
	BatchID           *uuid.UUID    `json:"batch_id,omitempty"`
}

type RulePreviewInput struct {
	SellingModel      string     `json:"selling_model"`
	BatchID           *uuid.UUID `json:"batch_id,omitempty"`
	SubtotalIDR       int64      `json:"subtotal_idr"`
	ItemQuantity      int        `json:"item_quantity"`
	FulfillmentMethod string     `json:"fulfillment_method"`
}

// scopePrecedence returns higher number for more specific scopes (PRD §27).
func scopePrecedence(scopeType string) int {
	switch scopeType {
	case ScopeBatch:
		return 3
	case ScopeSellingModel:
		return 2
	case ScopeStore:
		return 1
	default:
		return 0
	}
}

// pickMostSpecific chooses the winning rule among candidates of the same type.
func pickMostSpecific(candidates []BusinessRule) *BusinessRule {
	if len(candidates) == 0 {
		return nil
	}
	best := candidates[0]
	for _, c := range candidates[1:] {
		if scopePrecedence(c.ScopeType) > scopePrecedence(best.ScopeType) {
			best = c
		}
	}
	return &best
}

func validateRuleConfig(ruleType string, cfg RuleConfig) error {
	switch ruleType {
	case RuleHandlingFee:
		switch cfg.FeeMode {
		case FeeModeFixed:
			if cfg.FeeFixedIDR == nil || *cfg.FeeFixedIDR < 0 {
				return ErrInvalidInput
			}
		case FeeModePercent:
			if cfg.FeePercent == nil || *cfg.FeePercent < 0 || *cfg.FeePercent > 100 {
				return ErrInvalidInput
			}
		case FeeModeConditional:
			if len(cfg.Tiers) == 0 {
				return ErrInvalidInput
			}
			for _, tier := range cfg.Tiers {
				if tier.MinSubtotalIDR < 0 || tier.FeeFixedIDR < 0 {
					return ErrInvalidInput
				}
			}
		default:
			return ErrInvalidInput
		}
	case RuleMinimumOrder:
		if (cfg.MinSubtotalIDR == nil || *cfg.MinSubtotalIDR <= 0) &&
			(cfg.MinQuantity == nil || *cfg.MinQuantity <= 0) {
			return ErrInvalidInput
		}
	case RuleOrderApproval:
		// presence means approval required when enabled
	default:
		return ErrInvalidInput
	}
	return nil
}

func validateRuleScope(in UpsertRuleInput) error {
	switch in.ScopeType {
	case ScopeStore:
		if in.ScopeBatchID != nil || in.ScopeSellingModel != nil {
			return ErrInvalidInput
		}
	case ScopeSellingModel:
		if in.ScopeSellingModel == nil || in.ScopeBatchID != nil {
			return ErrInvalidInput
		}
		sm := *in.ScopeSellingModel
		if sm != "ready_stock" && sm != "preorder" {
			return ErrInvalidInput
		}
	case ScopeBatch:
		if in.ScopeBatchID == nil || in.ScopeSellingModel != nil {
			return ErrInvalidInput
		}
	default:
		return ErrInvalidInput
	}
	return nil
}

func (s *Service) ListBusinessRules(ctx context.Context, storeID uuid.UUID) ([]BusinessRule, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT r.id, r.store_id, r.rule_type, r.name, r.scope_type, r.scope_selling_model,
		       r.scope_batch_id, r.is_enabled, r.current_version, r.created_at, r.updated_at,
		       v.id, v.config
		FROM business_rules r
		LEFT JOIN business_rule_versions v
		  ON v.rule_id=r.id AND v.version=r.current_version
		WHERE r.store_id=$1
		ORDER BY r.rule_type, r.scope_type, r.created_at
	`, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BusinessRule
	for rows.Next() {
		var r BusinessRule
		var cfgRaw []byte
		var versionID *uuid.UUID
		var createdAt, updatedAt interface{}
		if err := rows.Scan(
			&r.ID, &r.StoreID, &r.RuleType, &r.Name, &r.ScopeType, &r.ScopeSellingModel,
			&r.ScopeBatchID, &r.IsEnabled, &r.CurrentVersion, &createdAt, &updatedAt,
			&versionID, &cfgRaw,
		); err != nil {
			return nil, err
		}
		r.VersionID = versionID
		if len(cfgRaw) > 0 {
			_ = json.Unmarshal(cfgRaw, &r.Config)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Service) CreateBusinessRule(ctx context.Context, storeID, actorID uuid.UUID, in UpsertRuleInput) (*BusinessRule, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || len(in.Name) > 120 {
		return nil, ErrInvalidInput
	}
	if err := validateRuleScope(in); err != nil {
		return nil, err
	}
	if err := validateRuleConfig(in.RuleType, in.Config); err != nil {
		return nil, err
	}
	if in.ScopeType == ScopeBatch && in.ScopeBatchID != nil {
		var ok bool
		err := s.DB.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM preorder_batches WHERE id=$1 AND store_id=$2)
		`, *in.ScopeBatchID, storeID).Scan(&ok)
		if err != nil || !ok {
			return nil, ErrInvalidInput
		}
	}
	enabled := true
	if in.IsEnabled != nil {
		enabled = *in.IsEnabled
	}
	cfgRaw, err := json.Marshal(in.Config)
	if err != nil {
		return nil, err
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var ruleID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO business_rules (
			store_id, rule_type, name, scope_type, scope_selling_model, scope_batch_id,
			is_enabled, current_version
		) VALUES ($1,$2,$3,$4,$5,$6,$7,1)
		RETURNING id
	`, storeID, in.RuleType, in.Name, in.ScopeType, in.ScopeSellingModel, in.ScopeBatchID, enabled,
	).Scan(&ruleID)
	if err != nil {
		if strings.Contains(err.Error(), "business_rules_scope_unique") {
			return nil, ErrConflict
		}
		return nil, err
	}
	var versionID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO business_rule_versions (store_id, rule_id, version, config, published_by)
		VALUES ($1,$2,1,$3,$4) RETURNING id
	`, storeID, ruleID, cfgRaw, actorID).Scan(&versionID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	list, err := s.ListBusinessRules(ctx, storeID)
	if err != nil {
		return nil, err
	}
	for i := range list {
		if list[i].ID == ruleID {
			return &list[i], nil
		}
	}
	return nil, ErrNotFound
}

func (s *Service) UpdateBusinessRule(ctx context.Context, storeID, ruleID, actorID uuid.UUID, in UpsertRuleInput) (*BusinessRule, error) {
	var existing BusinessRule
	err := s.DB.QueryRow(ctx, `
		SELECT id, rule_type, scope_type, scope_selling_model, scope_batch_id, is_enabled, current_version, name
		FROM business_rules WHERE id=$1 AND store_id=$2
	`, ruleID, storeID).Scan(
		&existing.ID, &existing.RuleType, &existing.ScopeType, &existing.ScopeSellingModel,
		&existing.ScopeBatchID, &existing.IsEnabled, &existing.CurrentVersion, &existing.Name,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = existing.Name
	}
	ruleType := existing.RuleType
	if in.RuleType != "" && in.RuleType != ruleType {
		return nil, ErrInvalidInput // type immutable
	}
	scopeType := existing.ScopeType
	scopeModel := existing.ScopeSellingModel
	scopeBatch := existing.ScopeBatchID
	if in.ScopeType != "" {
		scopeType = in.ScopeType
		scopeModel = in.ScopeSellingModel
		scopeBatch = in.ScopeBatchID
		tmp := UpsertRuleInput{ScopeType: scopeType, ScopeSellingModel: scopeModel, ScopeBatchID: scopeBatch}
		if err := validateRuleScope(tmp); err != nil {
			return nil, err
		}
	}
	if err := validateRuleConfig(ruleType, in.Config); err != nil {
		return nil, err
	}
	enabled := existing.IsEnabled
	if in.IsEnabled != nil {
		enabled = *in.IsEnabled
	}
	cfgRaw, err := json.Marshal(in.Config)
	if err != nil {
		return nil, err
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	newVersion := existing.CurrentVersion + 1
	_, err = tx.Exec(ctx, `
		UPDATE business_rules SET
			name=$3, scope_type=$4, scope_selling_model=$5, scope_batch_id=$6,
			is_enabled=$7, current_version=$8, updated_at=now()
		WHERE id=$1 AND store_id=$2
	`, ruleID, storeID, name, scopeType, scopeModel, scopeBatch, enabled, newVersion)
	if err != nil {
		if strings.Contains(err.Error(), "business_rules_scope_unique") {
			return nil, ErrConflict
		}
		return nil, err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO business_rule_versions (store_id, rule_id, version, config, published_by)
		VALUES ($1,$2,$3,$4,$5)
	`, storeID, ruleID, newVersion, cfgRaw, actorID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	list, err := s.ListBusinessRules(ctx, storeID)
	if err != nil {
		return nil, err
	}
	for i := range list {
		if list[i].ID == ruleID {
			return &list[i], nil
		}
	}
	return nil, ErrNotFound
}

func (s *Service) DeleteBusinessRule(ctx context.Context, storeID, ruleID uuid.UUID) error {
	tag, err := s.DB.Exec(ctx, `
		DELETE FROM business_rules WHERE id=$1 AND store_id=$2
	`, ruleID, storeID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) loadEnabledRules(ctx context.Context, storeID uuid.UUID) ([]BusinessRule, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT r.id, r.store_id, r.rule_type, r.name, r.scope_type, r.scope_selling_model,
		       r.scope_batch_id, r.is_enabled, r.current_version, v.id, v.config
		FROM business_rules r
		JOIN business_rule_versions v ON v.rule_id=r.id AND v.version=r.current_version
		WHERE r.store_id=$1 AND r.is_enabled=true AND r.current_version > 0
	`, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BusinessRule
	for rows.Next() {
		var r BusinessRule
		var cfgRaw []byte
		var versionID uuid.UUID
		if err := rows.Scan(
			&r.ID, &r.StoreID, &r.RuleType, &r.Name, &r.ScopeType, &r.ScopeSellingModel,
			&r.ScopeBatchID, &r.IsEnabled, &r.CurrentVersion, &versionID, &cfgRaw,
		); err != nil {
			return nil, err
		}
		r.VersionID = &versionID
		_ = json.Unmarshal(cfgRaw, &r.Config)
		out = append(out, r)
	}
	return out, rows.Err()
}

func ruleMatchesScope(r BusinessRule, sellingModel string, batchID *uuid.UUID) bool {
	switch r.ScopeType {
	case ScopeStore:
		return true
	case ScopeSellingModel:
		return r.ScopeSellingModel != nil && *r.ScopeSellingModel == sellingModel
	case ScopeBatch:
		return batchID != nil && r.ScopeBatchID != nil && *r.ScopeBatchID == *batchID
	default:
		return false
	}
}

// computeHandlingFee resolves fixed, percent, or conditional (first-match) tiers.
// Conditional tiers match when subtotal > MinSubtotalIDR; list order is priority.
func computeHandlingFee(cfg RuleConfig, subtotalIDR int64) int64 {
	switch cfg.FeeMode {
	case FeeModeFixed:
		if cfg.FeeFixedIDR != nil {
			return *cfg.FeeFixedIDR
		}
	case FeeModePercent:
		pct := 0.0
		if cfg.FeePercent != nil {
			pct = *cfg.FeePercent
		}
		return money.PercentOfIDR(subtotalIDR, pct)
	case FeeModeConditional:
		for _, tier := range cfg.Tiers {
			if subtotalIDR > tier.MinSubtotalIDR {
				return tier.FeeFixedIDR
			}
		}
	}
	return 0
}

// EvaluateRules applies precedence and computes fees / minimum / approval (PRD §27).
func EvaluateRules(
	all []BusinessRule,
	sellingModel string,
	batchID *uuid.UUID,
	subtotalIDR int64,
	itemQty int,
	shippingFeeIDR int64,
) (*RulesEvaluation, error) {
	byType := map[string][]BusinessRule{}
	for _, r := range all {
		if !ruleMatchesScope(r, sellingModel, batchID) {
			continue
		}
		byType[r.RuleType] = append(byType[r.RuleType], r)
	}

	eval := &RulesEvaluation{
		SubtotalIDR:    subtotalIDR,
		ShippingFeeIDR: shippingFeeIDR,
		ItemQuantity:   itemQty,
		SellingModel:   sellingModel,
		BatchID:        batchID,
		MinimumOK:      true,
	}

	if feeRule := pickMostSpecific(byType[RuleHandlingFee]); feeRule != nil && feeRule.VersionID != nil {
		fee := computeHandlingFee(feeRule.Config, subtotalIDR)
		eval.HandlingFeeIDR = fee
		result, _ := json.Marshal(map[string]any{"handling_fee_idr": fee})
		eval.Applied = append(eval.Applied, AppliedRule{
			RuleID: feeRule.ID, VersionID: *feeRule.VersionID, RuleType: RuleHandlingFee,
			ScopeType: feeRule.ScopeType, Config: feeRule.Config, Result: result,
		})
	}

	if minRule := pickMostSpecific(byType[RuleMinimumOrder]); minRule != nil && minRule.VersionID != nil {
		ok := true
		msg := ""
		if minRule.Config.MinSubtotalIDR != nil && subtotalIDR < *minRule.Config.MinSubtotalIDR {
			ok = false
			msg = "Product subtotal is below the minimum order."
		}
		if minRule.Config.MinQuantity != nil && itemQty < *minRule.Config.MinQuantity {
			ok = false
			if msg == "" {
				msg = "Item quantity is below the minimum order."
			}
		}
		eval.MinimumOK = ok
		eval.MinimumMessage = msg
		result, _ := json.Marshal(map[string]any{"ok": ok, "message": msg})
		eval.Applied = append(eval.Applied, AppliedRule{
			RuleID: minRule.ID, VersionID: *minRule.VersionID, RuleType: RuleMinimumOrder,
			ScopeType: minRule.ScopeType, Config: minRule.Config, Result: result,
		})
		if !ok {
			eval.TotalIDR = money.SumIDR(subtotalIDR, eval.HandlingFeeIDR, shippingFeeIDR)
			return eval, ErrMinimumNotMet
		}
	}

	if appr := pickMostSpecific(byType[RuleOrderApproval]); appr != nil && appr.VersionID != nil {
		eval.RequiresApproval = true
		result, _ := json.Marshal(map[string]any{"requires_approval": true})
		eval.Applied = append(eval.Applied, AppliedRule{
			RuleID: appr.ID, VersionID: *appr.VersionID, RuleType: RuleOrderApproval,
			ScopeType: appr.ScopeType, Config: appr.Config, Result: result,
		})
	}

	eval.TotalIDR = money.SumIDR(subtotalIDR, eval.HandlingFeeIDR, shippingFeeIDR)
	return eval, nil
}

func (s *Service) EvaluateCheckoutRules(
	ctx context.Context,
	storeID uuid.UUID,
	sellingModel string,
	batchID *uuid.UUID,
	subtotalIDR int64,
	itemQty int,
	shippingFeeIDR int64,
) (*RulesEvaluation, error) {
	all, err := s.loadEnabledRules(ctx, storeID)
	if err != nil {
		return nil, err
	}
	return EvaluateRules(all, sellingModel, batchID, subtotalIDR, itemQty, shippingFeeIDR)
}

func (s *Service) PreviewRules(ctx context.Context, storeID uuid.UUID, in RulePreviewInput) (*RulesEvaluation, error) {
	if in.SellingModel != "ready_stock" && in.SellingModel != "preorder" {
		return nil, ErrInvalidInput
	}
	if in.SubtotalIDR < 0 || in.ItemQuantity < 0 {
		return nil, ErrInvalidInput
	}
	shipFee := int64(0)
	if in.FulfillmentMethod != "" {
		var err error
		shipFee, err = s.resolveShipping(ctx, storeID, in.FulfillmentMethod)
		if err != nil {
			return nil, err
		}
	}
	eval, err := s.EvaluateCheckoutRules(ctx, storeID, in.SellingModel, in.BatchID, in.SubtotalIDR, in.ItemQuantity, shipFee)
	if err != nil && !errors.Is(err, ErrMinimumNotMet) {
		return nil, err
	}
	return eval, nil
}
