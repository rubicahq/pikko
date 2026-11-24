package commerce

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rubicahq/pikko/services/api/internal/money"
)

func (s *Service) ListDepositPolicies(ctx context.Context, storeID uuid.UUID) ([]DepositPolicy, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT id, store_id, name, applies_to, deposit_type, percent_bps, fixed_amount_idr,
		       is_enabled, created_at, updated_at
		FROM store_deposit_policies WHERE store_id=$1
		ORDER BY created_at ASC
	`, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DepositPolicy
	for rows.Next() {
		var p DepositPolicy
		if err := rows.Scan(
			&p.ID, &p.StoreID, &p.Name, &p.AppliesTo, &p.DepositType, &p.PercentBPS, &p.FixedAmountIDR,
			&p.IsEnabled, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Service) CreateDepositPolicy(ctx context.Context, storeID uuid.UUID, in UpsertDepositPolicyInput) (*DepositPolicy, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = "Default deposit"
	}
	applies := strings.TrimSpace(in.AppliesTo)
	if applies == "" {
		applies = "preorder"
	}
	switch applies {
	case "all", "preorder", "ready_stock":
	default:
		return nil, ErrInvalidInput
	}
	dtype := strings.TrimSpace(in.DepositType)
	switch dtype {
	case "percent":
		if in.PercentBPS == nil || *in.PercentBPS < 0 || *in.PercentBPS > 10000 {
			return nil, ErrInvalidInput
		}
	case "fixed":
		if in.FixedAmountIDR == nil || *in.FixedAmountIDR < 0 {
			return nil, ErrInvalidInput
		}
	default:
		return nil, ErrInvalidInput
	}
	enabled := true
	if in.IsEnabled != nil {
		enabled = *in.IsEnabled
	}

	var p DepositPolicy
	err := s.DB.QueryRow(ctx, `
		INSERT INTO store_deposit_policies (
			store_id, name, applies_to, deposit_type, percent_bps, fixed_amount_idr, is_enabled
		) VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, store_id, name, applies_to, deposit_type, percent_bps, fixed_amount_idr,
		          is_enabled, created_at, updated_at
	`, storeID, name, applies, dtype, in.PercentBPS, in.FixedAmountIDR, enabled).Scan(
		&p.ID, &p.StoreID, &p.Name, &p.AppliesTo, &p.DepositType, &p.PercentBPS, &p.FixedAmountIDR,
		&p.IsEnabled, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Service) UpdateDepositPolicy(ctx context.Context, storeID, policyID uuid.UUID, in UpsertDepositPolicyInput) (*DepositPolicy, error) {
	existing, err := s.getDepositPolicy(ctx, storeID, policyID)
	if err != nil {
		return nil, err
	}
	name := existing.Name
	if strings.TrimSpace(in.Name) != "" {
		name = strings.TrimSpace(in.Name)
	}
	applies := existing.AppliesTo
	if strings.TrimSpace(in.AppliesTo) != "" {
		applies = strings.TrimSpace(in.AppliesTo)
	}
	dtype := existing.DepositType
	if strings.TrimSpace(in.DepositType) != "" {
		dtype = strings.TrimSpace(in.DepositType)
	}
	percent := existing.PercentBPS
	fixed := existing.FixedAmountIDR
	if in.PercentBPS != nil {
		percent = in.PercentBPS
	}
	if in.FixedAmountIDR != nil {
		fixed = in.FixedAmountIDR
	}
	enabled := existing.IsEnabled
	if in.IsEnabled != nil {
		enabled = *in.IsEnabled
	}

	var p DepositPolicy
	err = s.DB.QueryRow(ctx, `
		UPDATE store_deposit_policies SET
			name=$3, applies_to=$4, deposit_type=$5, percent_bps=$6, fixed_amount_idr=$7,
			is_enabled=$8, updated_at=now()
		WHERE id=$1 AND store_id=$2
		RETURNING id, store_id, name, applies_to, deposit_type, percent_bps, fixed_amount_idr,
		          is_enabled, created_at, updated_at
	`, policyID, storeID, name, applies, dtype, percent, fixed, enabled).Scan(
		&p.ID, &p.StoreID, &p.Name, &p.AppliesTo, &p.DepositType, &p.PercentBPS, &p.FixedAmountIDR,
		&p.IsEnabled, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Service) getDepositPolicy(ctx context.Context, storeID, policyID uuid.UUID) (*DepositPolicy, error) {
	var p DepositPolicy
	err := s.DB.QueryRow(ctx, `
		SELECT id, store_id, name, applies_to, deposit_type, percent_bps, fixed_amount_idr,
		       is_enabled, created_at, updated_at
		FROM store_deposit_policies WHERE id=$1 AND store_id=$2
	`, policyID, storeID).Scan(
		&p.ID, &p.StoreID, &p.Name, &p.AppliesTo, &p.DepositType, &p.PercentBPS, &p.FixedAmountIDR,
		&p.IsEnabled, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// SuggestDepositAmount returns the deposit for an order total given the active policy (ORD-10).
func (s *Service) SuggestDepositAmount(ctx context.Context, storeID uuid.UUID, sellingModel string, orderTotalIDR int64) (int64, *DepositPolicy, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT id, store_id, name, applies_to, deposit_type, percent_bps, fixed_amount_idr,
		       is_enabled, created_at, updated_at
		FROM store_deposit_policies
		WHERE store_id=$1 AND is_enabled=true
		ORDER BY
		  CASE WHEN applies_to=$2 THEN 0 WHEN applies_to='all' THEN 1 ELSE 2 END,
		  created_at ASC
		LIMIT 1
	`, storeID, sellingModel)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return 0, nil, nil
	}
	var p DepositPolicy
	if err := rows.Scan(
		&p.ID, &p.StoreID, &p.Name, &p.AppliesTo, &p.DepositType, &p.PercentBPS, &p.FixedAmountIDR,
		&p.IsEnabled, &p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		return 0, nil, err
	}
	bps := 0
	if p.PercentBPS != nil {
		bps = *p.PercentBPS
	}
	fixed := int64(0)
	if p.FixedAmountIDR != nil {
		fixed = *p.FixedAmountIDR
	}
	amt := money.DepositAmountIDR(orderTotalIDR, p.DepositType, bps, fixed)
	return amt, &p, nil
}
