package commerce

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rubicahq/pikko/services/api/internal/money"
)

// GetStoreCredit returns customer credit balance when feature flag is on (REF-02).
func (s *Service) GetStoreCredit(ctx context.Context, storeID, customerID uuid.UUID) (*StoreCreditBalance, error) {
	ok, err := s.storeCreditEnabledForStore(ctx, storeID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrStoreCreditDisabled
	}
	var bal StoreCreditBalance
	err = s.DB.QueryRow(ctx, `
		SELECT store_id, customer_id, balance_idr, version, updated_at
		FROM store_credits WHERE store_id=$1 AND customer_id=$2
	`, storeID, customerID).Scan(&bal.StoreID, &bal.CustomerID, &bal.BalanceIDR, &bal.Version, &bal.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return &StoreCreditBalance{StoreID: storeID, CustomerID: customerID, BalanceIDR: 0, Version: 0}, nil
	}
	if err != nil {
		return nil, err
	}
	return &bal, nil
}

// ListStoreCreditLedger returns append-only credit ledger entries.
func (s *Service) ListStoreCreditLedger(ctx context.Context, storeID, customerID uuid.UUID, limit int) ([]StoreCreditLedgerEntry, error) {
	ok, err := s.storeCreditEnabledForStore(ctx, storeID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrStoreCreditDisabled
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.DB.Query(ctx, `
		SELECT id, store_id, customer_id, entry_type, amount_idr, balance_after_idr,
		       order_id, refund_id, financial_event_id, idempotency_key, note, actor_user_id, created_at
		FROM store_credit_ledger
		WHERE store_id=$1 AND customer_id=$2
		ORDER BY created_at DESC
		LIMIT $3
	`, storeID, customerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StoreCreditLedgerEntry
	for rows.Next() {
		var e StoreCreditLedgerEntry
		if err := rows.Scan(
			&e.ID, &e.StoreID, &e.CustomerID, &e.EntryType, &e.AmountIDR, &e.BalanceAfterIDR,
			&e.OrderID, &e.RefundID, &e.FinancialEventID, &e.IdempotencyKey, &e.Note, &e.ActorUserID, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// GrantStoreCredit credits a customer ledger (feature-flagged).
func (s *Service) GrantStoreCredit(ctx context.Context, storeID, actorID uuid.UUID, in GrantStoreCreditInput) (*StoreCreditBalance, error) {
	ok, err := s.storeCreditEnabledForStore(ctx, storeID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrStoreCreditDisabled
	}
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	if in.IdempotencyKey == "" || in.CustomerID == uuid.Nil {
		return nil, ErrInvalidInput
	}
	if err := money.ValidateIDR(in.AmountIDR, false); err != nil {
		return nil, ErrInvalidInput
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if err := s.grantStoreCreditTx(ctx, tx, storeID, in.CustomerID, in.AmountIDR, actorID,
		in.IdempotencyKey, in.OrderID, in.RefundID, in.Note); err != nil {
		if errors.Is(err, ErrIdempotencyConflict) {
			_ = tx.Rollback(ctx)
			return s.GetStoreCredit(ctx, storeID, in.CustomerID)
		}
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetStoreCredit(ctx, storeID, in.CustomerID)
}

func (s *Service) grantStoreCreditTx(
	ctx context.Context, tx pgx.Tx, storeID, customerID uuid.UUID, amountIDR int64, actorID uuid.UUID,
	idemKey string, orderID, refundID *uuid.UUID, note *string,
) error {
	var bal int64
	var ver int
	err := tx.QueryRow(ctx, `
		SELECT balance_idr, version FROM store_credits
		WHERE store_id=$1 AND customer_id=$2 FOR UPDATE
	`, storeID, customerID).Scan(&bal, &ver)
	if errors.Is(err, pgx.ErrNoRows) {
		_, err = tx.Exec(ctx, `
			INSERT INTO store_credits (store_id, customer_id, balance_idr, version)
			VALUES ($1,$2,0,1)
		`, storeID, customerID)
		if err != nil && !isUniqueViolation(err) {
			return err
		}
		err = tx.QueryRow(ctx, `
			SELECT balance_idr, version FROM store_credits
			WHERE store_id=$1 AND customer_id=$2 FOR UPDATE
		`, storeID, customerID).Scan(&bal, &ver)
	}
	if err != nil {
		return err
	}

	newBal := bal + amountIDR
	ct, err := tx.Exec(ctx, `
		UPDATE store_credits SET balance_idr=$3, version=version+1, updated_at=now()
		WHERE store_id=$1 AND customer_id=$2 AND version=$4
	`, storeID, customerID, newBal, ver)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrVersionConflict
	}

	evID, err := s.insertFinancialEventTx(ctx, tx, storeID, &customerID, orderID, nil, refundID,
		"credit_granted", "credit", amountIDR, "credit_granted:"+idemKey,
		map[string]any{"balance_after": newBal}, &actorID)
	if err != nil && !errors.Is(err, ErrIdempotencyConflict) {
		return err
	}
	if errors.Is(err, ErrIdempotencyConflict) {
		return ErrIdempotencyConflict
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO store_credit_ledger (
			store_id, customer_id, entry_type, amount_idr, balance_after_idr,
			order_id, refund_id, financial_event_id, idempotency_key, note, actor_user_id
		) VALUES ($1,$2,'credit',$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (store_id, idempotency_key) DO NOTHING
	`, storeID, customerID, amountIDR, newBal, orderID, refundID, evID, idemKey, note, actorID)
	return err
}

// ApplyStoreCreditToOrder reduces payable via credit_applied (REF-03 / E2E-G). Feature-flagged.
func (s *Service) ApplyStoreCreditToOrder(ctx context.Context, storeID, actorID uuid.UUID, in ApplyStoreCreditInput) (*OrderFinanceSummary, error) {
	ok, err := s.storeCreditEnabledForStore(ctx, storeID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrStoreCreditDisabled
	}
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	if in.IdempotencyKey == "" || in.OrderID == uuid.Nil {
		return nil, ErrInvalidInput
	}
	if err := money.ValidateIDR(in.AmountIDR, false); err != nil {
		return nil, ErrInvalidInput
	}

	o, err := s.GetOrder(ctx, storeID, in.OrderID)
	if err != nil {
		return nil, err
	}
	if o.CustomerID == nil {
		return nil, ErrInvalidInput
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Cap credit against remaining due so payable cannot be driven artificially
	// into refund entitlement (REF-03 / money integrity).
	fin, err := s.recomputeSettlementTx(ctx, tx, storeID, in.OrderID)
	if err != nil {
		return nil, err
	}
	if in.AmountIDR > fin.RemainingDueIDR {
		return nil, ErrInvalidInput
	}

	var bal int64
	var ver int
	err = tx.QueryRow(ctx, `
		SELECT balance_idr, version FROM store_credits
		WHERE store_id=$1 AND customer_id=$2 FOR UPDATE
	`, storeID, *o.CustomerID).Scan(&bal, &ver)
	if errors.Is(err, pgx.ErrNoRows) || bal < in.AmountIDR {
		return nil, ErrInsufficientCredit
	}
	if err != nil {
		return nil, err
	}

	newBal := bal - in.AmountIDR
	ct, err := tx.Exec(ctx, `
		UPDATE store_credits SET balance_idr=$3, version=version+1, updated_at=now()
		WHERE store_id=$1 AND customer_id=$2 AND version=$4
	`, storeID, *o.CustomerID, newBal, ver)
	if err != nil {
		return nil, err
	}
	if ct.RowsAffected() == 0 {
		return nil, ErrVersionConflict
	}

	_, err = tx.Exec(ctx, `
		UPDATE orders SET credit_applied_idr=COALESCE(credit_applied_idr,0)+$3, updated_at=now()
		WHERE id=$1 AND store_id=$2
	`, in.OrderID, storeID, in.AmountIDR)
	if err != nil {
		return nil, err
	}

	oid := &in.OrderID
	evID, err := s.insertFinancialEventTx(ctx, tx, storeID, o.CustomerID, oid, nil, nil,
		"credit_applied", "debit", in.AmountIDR, "credit_applied:"+in.IdempotencyKey,
		map[string]any{"balance_after": newBal}, &actorID)
	if err != nil {
		if errors.Is(err, ErrIdempotencyConflict) {
			_ = tx.Rollback(ctx)
			return s.GetOrderFinance(ctx, storeID, in.OrderID)
		}
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO store_credit_ledger (
			store_id, customer_id, entry_type, amount_idr, balance_after_idr,
			order_id, financial_event_id, idempotency_key, note, actor_user_id
		) VALUES ($1,$2,'debit',$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (store_id, idempotency_key) DO NOTHING
	`, storeID, *o.CustomerID, in.AmountIDR, newBal, in.OrderID, evID, in.IdempotencyKey, in.Note, actorID)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO financial_adjustments (
			store_id, order_id, financial_event_id, adjustment_type, amount_idr, reason, actor_user_id, idempotency_key
		) VALUES ($1,$2,$3,'credit_applied',$4,$5,$6,$7)
		ON CONFLICT (store_id, idempotency_key) DO NOTHING
	`, storeID, in.OrderID, evID, in.AmountIDR, in.Note, actorID, "adj:"+in.IdempotencyKey)
	if err != nil {
		return nil, err
	}

	if _, err := s.recomputeSettlementTx(ctx, tx, storeID, in.OrderID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetOrderFinance(ctx, storeID, in.OrderID)
}

// SetStoreCreditEnabled toggles per-store store credit (requires global FEATURE_STORE_CREDIT).
func (s *Service) SetStoreCreditEnabled(ctx context.Context, storeID uuid.UUID, enabled bool) error {
	if enabled && !s.StoreCreditGloballyEnabled() {
		return ErrStoreCreditDisabled
	}
	_, err := s.DB.Exec(ctx, `
		UPDATE stores SET store_credit_enabled=$2, updated_at=now() WHERE id=$1
	`, storeID, enabled)
	return err
}
