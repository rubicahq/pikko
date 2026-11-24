package commerce

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rubicahq/pikko/services/api/internal/money"
)

// ApplyFinancialAdjustment records an unavailable/manual delta and recomputes payable (FIN-04, UNAV-03).
// Does not mutate historical payment rows; appends financial_adjustments + financial_events.
func (s *Service) ApplyFinancialAdjustment(ctx context.Context, storeID, actorID uuid.UUID, in FinancialAdjustmentInput) (*OrderFinanceSummary, error) {
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	if in.IdempotencyKey == "" || in.OrderID == uuid.Nil {
		return nil, ErrInvalidInput
	}
	if err := money.ValidateIDR(in.AmountIDR, false); err != nil {
		return nil, ErrInvalidInput
	}
	switch in.AdjustmentType {
	case AdjustUnavailable, AdjustManual, AdjustFee, AdjustCredit:
	default:
		return nil, ErrInvalidInput
	}

	// Idempotent replay
	var existingOrder uuid.UUID
	err := s.DB.QueryRow(ctx, `
		SELECT order_id FROM financial_adjustments WHERE store_id=$1 AND idempotency_key=$2
	`, storeID, in.IdempotencyKey).Scan(&existingOrder)
	if err == nil {
		return s.GetOrderFinance(ctx, storeID, existingOrder)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	o, err := s.GetOrder(ctx, storeID, in.OrderID)
	if err != nil {
		return nil, err
	}
	if o.OrderStatus == OrderCancelled {
		return nil, ErrInvalidTransition
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Lock order and bump unavailable / credit deltas
	var original, unavail, credit int64
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(original_total_idr, total_idr), COALESCE(unavailable_delta_idr,0), COALESCE(credit_applied_idr,0)
		FROM orders WHERE id=$1 AND store_id=$2 FOR UPDATE
	`, in.OrderID, storeID).Scan(&original, &unavail, &credit)
	if err != nil {
		return nil, err
	}
	if original == 0 {
		original = o.TotalIDR
		_, _ = tx.Exec(ctx, `UPDATE orders SET original_total_idr=$3 WHERE id=$1 AND store_id=$2 AND original_total_idr IS NULL`,
			in.OrderID, storeID, original)
	}

	switch in.AdjustmentType {
	case AdjustUnavailable, AdjustManual, AdjustFee:
		unavail += in.AmountIDR
		_, err = tx.Exec(ctx, `
			UPDATE orders SET unavailable_delta_idr=$3, updated_at=now() WHERE id=$1 AND store_id=$2
		`, in.OrderID, storeID, unavail)
	case AdjustCredit:
		credit += in.AmountIDR
		_, err = tx.Exec(ctx, `
			UPDATE orders SET credit_applied_idr=$3, updated_at=now() WHERE id=$1 AND store_id=$2
		`, in.OrderID, storeID, credit)
	}
	if err != nil {
		return nil, err
	}

	eventType := "adjustment_manual"
	if in.AdjustmentType == AdjustUnavailable {
		eventType = "adjustment_unavailable"
	}
	oid := &in.OrderID
	evID, err := s.insertFinancialEventTx(ctx, tx, storeID, o.CustomerID, oid, nil, nil,
		eventType, "debit", in.AmountIDR, "adj:"+in.IdempotencyKey,
		map[string]any{"adjustment_type": in.AdjustmentType}, &actorID)
	if err != nil && !errors.Is(err, ErrIdempotencyConflict) {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO financial_adjustments (
			store_id, order_id, order_item_id, revision_id, financial_event_id,
			adjustment_type, amount_idr, reason, actor_user_id, idempotency_key
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (store_id, idempotency_key) DO NOTHING
	`, storeID, in.OrderID, in.OrderItemID, in.RevisionID, evID,
		in.AdjustmentType, in.AmountIDR, in.Reason, actorID, in.IdempotencyKey)
	if err != nil {
		return nil, err
	}

	summary, err := s.recomputeSettlementTx(ctx, tx, storeID, in.OrderID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	if s.Outbox != nil {
		_ = s.Outbox.Write(ctx, &storeID, "finance.adjustment_applied", map[string]any{
			"order_id": in.OrderID.String(), "amount_idr": in.AmountIDR, "type": in.AdjustmentType,
		}, "adj:"+in.IdempotencyKey)
	}
	full, err := s.GetOrderFinance(ctx, storeID, in.OrderID)
	if err != nil {
		return summary, nil
	}
	return full, nil
}

// ApplyUnavailableFinanceAdjustment is the Agent 06 hook: emit ledger from unavailable delta without rewriting snapshots.
func (s *Service) ApplyUnavailableFinanceAdjustment(
	ctx context.Context, storeID, orderID, itemID, actorID uuid.UUID, deltaIDR int64, revisionID *uuid.UUID, reason *string, idemKey string,
) (*OrderFinanceSummary, error) {
	if deltaIDR <= 0 {
		return s.GetOrderFinance(ctx, storeID, orderID)
	}
	if idemKey == "" {
		idemKey = "unavail:" + orderID.String() + ":" + itemID.String() + ":" + uuid.NewString()
	}
	return s.ApplyFinancialAdjustment(ctx, storeID, actorID, FinancialAdjustmentInput{
		OrderID:        orderID,
		OrderItemID:    &itemID,
		AdjustmentType: AdjustUnavailable,
		AmountIDR:      deltaIDR,
		Reason:         reason,
		IdempotencyKey: idemKey,
		RevisionID:     revisionID,
	})
}
