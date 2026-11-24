package commerce

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/rubicahq/pikko/services/api/internal/money"
)

var (
	ErrStoreCreditDisabled = errors.New("store credit disabled")
	ErrIdempotencyConflict = errors.New("idempotency conflict")
	ErrInsufficientCredit  = errors.New("insufficient store credit")
	ErrVersionConflict     = errors.New("version conflict")
)

// StoreCreditGloballyEnabled reports the FEATURE_STORE_CREDIT kill switch.
func (s *Service) StoreCreditGloballyEnabled() bool {
	if s.Cfg == nil {
		return false
	}
	return s.Cfg.FeatureStoreCredit
}

func (s *Service) storeCreditEnabledForStore(ctx context.Context, storeID uuid.UUID) (bool, error) {
	if !s.StoreCreditGloballyEnabled() {
		return false, nil
	}
	var enabled bool
	err := s.DB.QueryRow(ctx, `SELECT store_credit_enabled FROM stores WHERE id=$1`, storeID).Scan(&enabled)
	if err != nil {
		return false, err
	}
	return enabled, nil
}

// ListFinanceWorkspace returns reconciliation summaries (FIN-05).
func (s *Service) ListFinanceWorkspace(ctx context.Context, storeID uuid.UUID, settlementFilter string, limit, offset int) ([]OrderFinanceSummary, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	if offset < 0 {
		offset = 0
	}
	args := []any{storeID}
	where := "o.store_id=$1"
	if settlementFilter != "" {
		args = append(args, settlementFilter)
		where += " AND o.settlement_status=$2"
	}
	var total int
	if err := s.DB.QueryRow(ctx, `SELECT COUNT(*) FROM orders o WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	limPH := len(args) + 1
	offPH := len(args) + 2
	args = append(args, limit, offset)

	q := fmt.Sprintf(`
		SELECT o.id, o.order_number, o.customer_id, o.customer_name, o.customer_email,
		       COALESCE(o.original_total_idr, o.total_idr),
		       COALESCE(o.unavailable_delta_idr, 0),
		       COALESCE(o.credit_applied_idr, 0),
		       COALESCE(o.payable_idr, o.total_idr),
		       o.payment_status, o.settlement_status,
		       COALESCE((
		         SELECT SUM(pa.amount_idr) FROM payment_allocations pa
		         JOIN payments p ON p.id=pa.payment_id
		         WHERE pa.order_id=o.id AND pa.store_id=o.store_id AND p.status='verified'
		       ), (
		         SELECT COALESCE(SUM(p2.amount_idr),0) FROM payments p2
		         WHERE p2.order_id=o.id AND p2.store_id=o.store_id AND p2.status='verified'
		           AND NOT EXISTS (SELECT 1 FROM payment_allocations pa2 WHERE pa2.payment_id=p2.id)
		       ), 0),
		       COALESCE((
		         SELECT SUM(r.amount_idr) FROM refunds r
		         WHERE r.order_id=o.id AND r.store_id=o.store_id AND r.status='completed'
		       ), 0)
		FROM orders o
		WHERE %s
		ORDER BY o.updated_at DESC
		LIMIT $%d OFFSET $%d`, where, limPH, offPH)
	rows, err := s.DB.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []OrderFinanceSummary
	for rows.Next() {
		var sm OrderFinanceSummary
		if err := rows.Scan(
			&sm.OrderID, &sm.OrderNumber, &sm.CustomerID, &sm.CustomerName, &sm.CustomerEmail,
			&sm.OriginalTotalIDR, &sm.UnavailableDeltaIDR, &sm.CreditAppliedIDR, &sm.PayableIDR,
			&sm.PaymentStatus, &sm.SettlementStatus, &sm.PaidIDR, &sm.RefundedIDR,
		); err != nil {
			return nil, 0, err
		}
		sm.Money = money.Reconcile(money.ReconciliationInput{
			OriginalTotalIDR:    sm.OriginalTotalIDR,
			UnavailableDeltaIDR: sm.UnavailableDeltaIDR,
			CreditAppliedIDR:    sm.CreditAppliedIDR,
			PaidIDR:             sm.PaidIDR,
		})
		sm.PayableIDR = sm.Money.PayableIDR
		sm.RefundEntitledIDR = sm.Money.RefundEntitledIDR
		sm.RemainingDueIDR = sm.Money.RemainingDueIDR
		out = append(out, sm)
	}
	return out, total, rows.Err()
}

// GetOrderFinance returns reconciliation for a single order (FIN-05/06).
func (s *Service) GetOrderFinance(ctx context.Context, storeID, orderID uuid.UUID) (*OrderFinanceSummary, error) {
	var sm OrderFinanceSummary
	err := s.DB.QueryRow(ctx, `
		SELECT o.id, o.order_number, o.customer_id, o.customer_name, o.customer_email,
		       COALESCE(o.original_total_idr, o.total_idr),
		       COALESCE(o.unavailable_delta_idr, 0),
		       COALESCE(o.credit_applied_idr, 0),
		       COALESCE(o.payable_idr, o.total_idr),
		       o.payment_status, o.settlement_status,
		       COALESCE((
		         SELECT SUM(pa.amount_idr) FROM payment_allocations pa
		         JOIN payments p ON p.id=pa.payment_id
		         WHERE pa.order_id=o.id AND pa.store_id=o.store_id AND p.status='verified'
		       ), (
		         SELECT COALESCE(SUM(p2.amount_idr),0) FROM payments p2
		         WHERE p2.order_id=o.id AND p2.store_id=o.store_id AND p2.status='verified'
		           AND NOT EXISTS (SELECT 1 FROM payment_allocations pa2 WHERE pa2.payment_id=p2.id)
		       ), 0),
		       COALESCE((
		         SELECT SUM(r.amount_idr) FROM refunds r
		         WHERE r.order_id=o.id AND r.store_id=o.store_id AND r.status='completed'
		       ), 0)
		FROM orders o WHERE o.id=$1 AND o.store_id=$2
	`, orderID, storeID).Scan(
		&sm.OrderID, &sm.OrderNumber, &sm.CustomerID, &sm.CustomerName, &sm.CustomerEmail,
		&sm.OriginalTotalIDR, &sm.UnavailableDeltaIDR, &sm.CreditAppliedIDR, &sm.PayableIDR,
		&sm.PaymentStatus, &sm.SettlementStatus, &sm.PaidIDR, &sm.RefundedIDR,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	sm.Money = money.Reconcile(money.ReconciliationInput{
		OriginalTotalIDR:    sm.OriginalTotalIDR,
		UnavailableDeltaIDR: sm.UnavailableDeltaIDR,
		CreditAppliedIDR:    sm.CreditAppliedIDR,
		PaidIDR:             sm.PaidIDR,
	})
	sm.PayableIDR = sm.Money.PayableIDR
	sm.RefundEntitledIDR = sm.Money.RefundEntitledIDR
	sm.RemainingDueIDR = sm.Money.RemainingDueIDR
	return &sm, nil
}

// RecomputeAndPersistSettlement recalculates payable/settlement for an order under row lock.
func (s *Service) recomputeSettlementTx(ctx context.Context, tx pgx.Tx, storeID, orderID uuid.UUID) (*OrderFinanceSummary, error) {
	var original, unavail, credit int64
	var paymentStatus string
	err := tx.QueryRow(ctx, `
		SELECT COALESCE(original_total_idr, total_idr), COALESCE(unavailable_delta_idr,0),
		       COALESCE(credit_applied_idr,0), payment_status
		FROM orders WHERE id=$1 AND store_id=$2 FOR UPDATE
	`, orderID, storeID).Scan(&original, &unavail, &credit, &paymentStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	var paid int64
	err = tx.QueryRow(ctx, `
		SELECT COALESCE((
		  SELECT SUM(pa.amount_idr) FROM payment_allocations pa
		  JOIN payments p ON p.id=pa.payment_id
		  WHERE pa.order_id=$1 AND pa.store_id=$2 AND p.status='verified'
		), (
		  SELECT COALESCE(SUM(p2.amount_idr),0) FROM payments p2
		  WHERE p2.order_id=$1 AND p2.store_id=$2 AND p2.status='verified'
		    AND NOT EXISTS (SELECT 1 FROM payment_allocations pa2 WHERE pa2.payment_id=p2.id)
		), 0)
	`, orderID, storeID).Scan(&paid)
	if err != nil {
		return nil, err
	}

	res := money.Reconcile(money.ReconciliationInput{
		OriginalTotalIDR:    original,
		UnavailableDeltaIDR: unavail,
		CreditAppliedIDR:    credit,
		PaidIDR:             paid,
	})

	newPayStatus := paymentStatus
	switch {
	case res.RefundEntitledIDR > 0 && paid > 0 && res.RemainingDueIDR == 0:
		// keep paid until refunds complete; settlement_status carries refund_due
	case res.RemainingDueIDR == 0 && paid > 0:
		newPayStatus = PaymentPaid
	case paid > 0 && res.RemainingDueIDR > 0:
		// partial — still unpaid for remainder; leave payment_status as-is unless verified deposit
		if paymentStatus == PaymentUnpaid {
			newPayStatus = PaymentUnpaid
		}
	}

	_, err = tx.Exec(ctx, `
		UPDATE orders SET
			payable_idr=$3, settlement_status=$4, payment_status=$5,
			total_idr=$3, updated_at=now()
		WHERE id=$1 AND store_id=$2
	`, orderID, storeID, res.PayableIDR, res.SettlementStatus, newPayStatus)
	if err != nil {
		return nil, err
	}

	return &OrderFinanceSummary{
		OrderID:             orderID,
		OriginalTotalIDR:    original,
		UnavailableDeltaIDR: unavail,
		CreditAppliedIDR:    credit,
		PayableIDR:          res.PayableIDR,
		PaidIDR:             paid,
		RefundEntitledIDR:   res.RefundEntitledIDR,
		RemainingDueIDR:     res.RemainingDueIDR,
		PaymentStatus:       newPayStatus,
		SettlementStatus:    res.SettlementStatus,
		Money:               res,
	}, nil
}

func (s *Service) insertFinancialEventTx(
	ctx context.Context, tx pgx.Tx,
	storeID uuid.UUID, customerID, orderID, paymentID, refundID *uuid.UUID,
	eventType, direction string, amountIDR int64, idempotencyKey string,
	meta map[string]any, actorID *uuid.UUID,
) (uuid.UUID, error) {
	raw, _ := json.Marshal(meta)
	if raw == nil {
		raw = []byte("{}")
	}
	var id uuid.UUID
	err := tx.QueryRow(ctx, `
		INSERT INTO financial_events (
			store_id, customer_id, order_id, payment_id, refund_id,
			event_type, direction, amount_idr, idempotency_key, meta, actor_user_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (store_id, idempotency_key) DO NOTHING
		RETURNING id
	`, storeID, customerID, orderID, paymentID, refundID, eventType, direction, amountIDR, idempotencyKey, raw, actorID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		// idempotent replay
		err = tx.QueryRow(ctx, `
			SELECT id FROM financial_events WHERE store_id=$1 AND idempotency_key=$2
		`, storeID, idempotencyKey).Scan(&id)
		if err != nil {
			return uuid.Nil, err
		}
		return id, ErrIdempotencyConflict
	}
	return id, err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
