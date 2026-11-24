package commerce

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rubicahq/pikko/services/api/internal/money"
)

// CreateRefund records a refund entitlement / request (REF-01). Amount must not exceed entitled.
func (s *Service) CreateRefund(ctx context.Context, storeID, actorID uuid.UUID, in CreateRefundInput) (*Refund, error) {
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	if in.IdempotencyKey == "" || in.OrderID == uuid.Nil {
		return nil, ErrInvalidInput
	}
	if err := money.ValidateIDR(in.AmountIDR, false); err != nil {
		return nil, ErrInvalidInput
	}
	method := strings.TrimSpace(in.Method)
	if method == "" {
		method = RefundMethodBankTransfer
	}
	if method != RefundMethodBankTransfer && method != RefundMethodStoreCredit {
		return nil, ErrInvalidInput
	}
	if method == RefundMethodStoreCredit {
		ok, err := s.storeCreditEnabledForStore(ctx, storeID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrStoreCreditDisabled
		}
	}

	var existing uuid.UUID
	err := s.DB.QueryRow(ctx, `
		SELECT id FROM refunds WHERE store_id=$1 AND idempotency_key=$2
	`, storeID, in.IdempotencyKey).Scan(&existing)
	if err == nil {
		return s.GetRefund(ctx, storeID, existing)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	fin, err := s.GetOrderFinance(ctx, storeID, in.OrderID)
	if err != nil {
		return nil, err
	}
	outstandingEntitled := fin.RefundEntitledIDR - fin.RefundedIDR
	if outstandingEntitled < 0 {
		outstandingEntitled = 0
	}
	if in.AmountIDR > outstandingEntitled {
		return nil, ErrInvalidInput
	}

	o, err := s.GetOrder(ctx, storeID, in.OrderID)
	if err != nil {
		return nil, err
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var refundID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO refunds (
			store_id, order_id, customer_id, amount_idr, method, status, reason,
			evidence_file_id, idempotency_key
		) VALUES ($1,$2,$3,$4,$5,'pending',$6,$7,$8)
		RETURNING id
	`, storeID, in.OrderID, o.CustomerID, in.AmountIDR, method, in.Reason, in.EvidenceFileID, in.IdempotencyKey).Scan(&refundID)
	if isUniqueViolation(err) {
		_ = tx.Rollback(ctx)
		var id uuid.UUID
		_ = s.DB.QueryRow(ctx, `SELECT id FROM refunds WHERE store_id=$1 AND idempotency_key=$2`, storeID, in.IdempotencyKey).Scan(&id)
		return s.GetRefund(ctx, storeID, id)
	}
	if err != nil {
		return nil, err
	}

	oid := &in.OrderID
	_, err = s.insertFinancialEventTx(ctx, tx, storeID, o.CustomerID, oid, nil, &refundID,
		"refund_entitled", "debit", in.AmountIDR, "refund_entitled:"+in.IdempotencyKey,
		map[string]any{"method": method}, &actorID)
	if err != nil && !errors.Is(err, ErrIdempotencyConflict) {
		return nil, err
	}

	_, _ = tx.Exec(ctx, `
		UPDATE orders SET settlement_status='refund_due', updated_at=now()
		WHERE id=$1 AND store_id=$2 AND settlement_status IN ('settled','refund_due','partially_settled','open')
	`, in.OrderID, storeID)

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetRefund(ctx, storeID, refundID)
}

func (s *Service) GetRefund(ctx context.Context, storeID, refundID uuid.UUID) (*Refund, error) {
	var r Refund
	err := s.DB.QueryRow(ctx, `
		SELECT id, store_id, order_id, customer_id, amount_idr, method, status, reason,
		       evidence_file_id, financial_event_id, idempotency_key, version,
		       processed_at, processed_by, created_at, updated_at
		FROM refunds WHERE id=$1 AND store_id=$2
	`, refundID, storeID).Scan(
		&r.ID, &r.StoreID, &r.OrderID, &r.CustomerID, &r.AmountIDR, &r.Method, &r.Status, &r.Reason,
		&r.EvidenceFileID, &r.FinancialEventID, &r.IdempotencyKey, &r.Version,
		&r.ProcessedAt, &r.ProcessedBy, &r.CreatedAt, &r.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Service) ListOrderRefunds(ctx context.Context, storeID, orderID uuid.UUID) ([]Refund, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT id, store_id, order_id, customer_id, amount_idr, method, status, reason,
		       evidence_file_id, financial_event_id, idempotency_key, version,
		       processed_at, processed_by, created_at, updated_at
		FROM refunds WHERE store_id=$1 AND order_id=$2 ORDER BY created_at ASC
	`, storeID, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Refund
	for rows.Next() {
		var r Refund
		if err := rows.Scan(
			&r.ID, &r.StoreID, &r.OrderID, &r.CustomerID, &r.AmountIDR, &r.Method, &r.Status, &r.Reason,
			&r.EvidenceFileID, &r.FinancialEventID, &r.IdempotencyKey, &r.Version,
			&r.ProcessedAt, &r.ProcessedBy, &r.CreatedAt, &r.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ProcessRefund completes/fails/cancels a refund with optimistic version (FIN-07 concurrency).
func (s *Service) ProcessRefund(ctx context.Context, storeID, refundID, actorID uuid.UUID, in ProcessRefundInput) (*Refund, error) {
	switch in.Status {
	case RefundCompleted, RefundFailed, RefundCancelled:
	default:
		return nil, ErrInvalidInput
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var r Refund
	err = tx.QueryRow(ctx, `
		SELECT id, store_id, order_id, customer_id, amount_idr, method, status, reason,
		       evidence_file_id, financial_event_id, idempotency_key, version
		FROM refunds WHERE id=$1 AND store_id=$2 FOR UPDATE
	`, refundID, storeID).Scan(
		&r.ID, &r.StoreID, &r.OrderID, &r.CustomerID, &r.AmountIDR, &r.Method, &r.Status, &r.Reason,
		&r.EvidenceFileID, &r.FinancialEventID, &r.IdempotencyKey, &r.Version,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if in.ExpectedVersion > 0 && r.Version != in.ExpectedVersion {
		return nil, ErrVersionConflict
	}
	if r.Status == RefundCompleted || r.Status == RefundCancelled {
		_ = tx.Rollback(ctx)
		return s.GetRefund(ctx, storeID, refundID)
	}

	ct, err := tx.Exec(ctx, `
		UPDATE refunds SET
			status=$4, processed_at=now(), processed_by=$3, version=version+1,
			reason=COALESCE($5, reason), updated_at=now()
		WHERE id=$1 AND store_id=$2 AND version=$6 AND status IN ('pending','processing','failed')
	`, refundID, storeID, actorID, in.Status, in.Reason, r.Version)
	if err != nil {
		return nil, err
	}
	if ct.RowsAffected() == 0 {
		return nil, ErrVersionConflict
	}

	oid := &r.OrderID
	eventType := "refund_issued"
	if in.Status == RefundFailed {
		eventType = "refund_failed"
	}
	if in.Status == RefundCompleted {
		evID, err := s.insertFinancialEventTx(ctx, tx, storeID, r.CustomerID, oid, nil, &refundID,
			eventType, "debit", r.AmountIDR, "refund_issued:"+refundID.String(),
			map[string]any{"method": r.Method}, &actorID)
		if err != nil && !errors.Is(err, ErrIdempotencyConflict) {
			return nil, err
		}
		_, _ = tx.Exec(ctx, `UPDATE refunds SET financial_event_id=$3 WHERE id=$1 AND store_id=$2`, refundID, storeID, evID)

		if r.Method == RefundMethodStoreCredit && r.CustomerID != nil {
			if err := s.grantStoreCreditTx(ctx, tx, storeID, *r.CustomerID, r.AmountIDR, actorID,
				"refund_credit:"+refundID.String(), oid, &refundID, ptrStr("Refund to store credit")); err != nil {
				return nil, err
			}
		}

		// Update payment_status based on remaining entitled after this refund
		var refunded int64
		_ = tx.QueryRow(ctx, `
			SELECT COALESCE(SUM(amount_idr),0) FROM refunds
			WHERE order_id=$1 AND store_id=$2 AND status='completed'
		`, r.OrderID, storeID).Scan(&refunded)

		fin, err := s.recomputeSettlementTx(ctx, tx, storeID, r.OrderID)
		if err != nil {
			return nil, err
		}
		payStatus := PaymentPartiallyRefunded
		settle := money.SettlementRefundDue
		if refunded >= fin.PaidIDR && fin.PaidIDR > 0 {
			payStatus = PaymentRefunded
			settle = money.SettlementRefunded
		} else if fin.RefundEntitledIDR <= refunded {
			settle = money.SettlementRefunded
			if refunded < fin.PaidIDR {
				payStatus = PaymentPartiallyRefunded
			}
		}
		_, _ = tx.Exec(ctx, `
			UPDATE orders SET payment_status=$3, settlement_status=$4, updated_at=now()
			WHERE id=$1 AND store_id=$2
		`, r.OrderID, storeID, payStatus, settle)
		_, _ = tx.Exec(ctx, `
			INSERT INTO order_status_history (store_id, order_id, dimension, from_status, to_status, actor_user_id, reason)
			VALUES ($1,$2,'payment', (SELECT payment_status FROM orders WHERE id=$2 AND store_id=$1), $3, $4, 'refund_completed')
		`, storeID, r.OrderID, payStatus, actorID)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	if s.Outbox != nil {
		_ = s.Outbox.Write(ctx, &storeID, "finance.refund_"+in.Status, map[string]any{
			"refund_id": refundID.String(), "order_id": r.OrderID.String(),
		}, "refund:"+refundID.String()+":"+in.Status)
	}
	return s.GetRefund(ctx, storeID, refundID)
}
