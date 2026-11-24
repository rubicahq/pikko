package commerce

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rubicahq/pikko/services/api/internal/money"
)

// ListOrderPayments returns all payments for an order (supports deposit + final).
func (s *Service) ListOrderPayments(ctx context.Context, storeID, orderID uuid.UUID) ([]Payment, error) {
	if _, err := s.GetOrder(ctx, storeID, orderID); err != nil {
		return nil, err
	}
	rows, err := s.DB.Query(ctx, `
		SELECT id, store_id, order_id, payment_method_id, method_type, amount_idr, status,
		       bank_name_snapshot, account_holder_snapshot, account_number_snapshot, instructions_snapshot,
		       verified_at, rejected_at, reject_reason, created_at, updated_at,
		       payment_kind, idempotency_key, note, version
		FROM payments
		WHERE store_id=$1 AND (
		  order_id=$2 OR id IN (SELECT payment_id FROM payment_allocations WHERE order_id=$2 AND store_id=$1)
		)
		ORDER BY created_at ASC
	`, storeID, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Payment
	for rows.Next() {
		p, err := scanPaymentExtended(rows)
		if err != nil {
			return nil, err
		}
		ev, err := s.listEvidence(ctx, storeID, p.ID)
		if err != nil {
			return nil, err
		}
		p.Evidence = ev
		allocs, err := s.listPaymentAllocations(ctx, storeID, p.ID)
		if err != nil {
			return nil, err
		}
		p.Allocations = allocs
		out = append(out, *p)
	}
	return out, rows.Err()
}

func scanPaymentExtended(row scannable) (*Payment, error) {
	var p Payment
	var kind, idem, note *string
	var version int
	err := row.Scan(
		&p.ID, &p.StoreID, &p.OrderID, &p.PaymentMethodID, &p.MethodType, &p.AmountIDR, &p.Status,
		&p.BankNameSnapshot, &p.AccountHolderSnapshot, &p.AccountNumberSnapshot, &p.InstructionsSnapshot,
		&p.VerifiedAt, &p.RejectedAt, &p.RejectReason, &p.CreatedAt, &p.UpdatedAt,
		&kind, &idem, &note, &version,
	)
	if err != nil {
		return nil, err
	}
	if kind != nil {
		p.PaymentKind = *kind
	} else {
		p.PaymentKind = PaymentKindFull
	}
	p.IdempotencyKey = idem
	p.Note = note
	p.Version = version
	return &p, nil
}

func (s *Service) listPaymentAllocations(ctx context.Context, storeID, paymentID uuid.UUID) ([]PaymentAllocation, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT id, store_id, payment_id, order_id, amount_idr, created_at
		FROM payment_allocations WHERE store_id=$1 AND payment_id=$2
		ORDER BY created_at
	`, storeID, paymentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PaymentAllocation
	for rows.Next() {
		var a PaymentAllocation
		if err := rows.Scan(&a.ID, &a.StoreID, &a.PaymentID, &a.OrderID, &a.AmountIDR, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// RecordPayment creates a deposit/final/partial/consolidated payment with idempotency (FIN-07, ORD-10, E2E-H).
func (s *Service) RecordPayment(ctx context.Context, storeID, actorID uuid.UUID, in RecordPaymentInput) (*Payment, error) {
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	if in.IdempotencyKey == "" || in.OrderID == uuid.Nil || in.PaymentMethodID == uuid.Nil {
		return nil, ErrInvalidInput
	}
	if err := money.ValidateIDR(in.AmountIDR, false); err != nil {
		return nil, ErrInvalidInput
	}
	kind := strings.TrimSpace(in.PaymentKind)
	if kind == "" {
		kind = PaymentKindPartial
	}
	switch kind {
	case PaymentKindFull, PaymentKindDeposit, PaymentKindFinal, PaymentKindPartial, PaymentKindConsolidated:
	default:
		return nil, ErrInvalidInput
	}

	allocs := in.Allocations
	if len(allocs) == 0 {
		allocs = []PaymentAllocationInput{{OrderID: in.OrderID, AmountIDR: in.AmountIDR}}
	}
	var sum int64
	for _, a := range allocs {
		if a.OrderID == uuid.Nil || a.AmountIDR <= 0 {
			return nil, ErrInvalidInput
		}
		sum += a.AmountIDR
	}
	if sum != in.AmountIDR {
		return nil, ErrInvalidInput
	}
	if kind == PaymentKindConsolidated && len(allocs) < 2 {
		return nil, ErrInvalidInput
	}

	// Idempotent replay
	var existingID uuid.UUID
	err := s.DB.QueryRow(ctx, `
		SELECT id FROM payments WHERE store_id=$1 AND idempotency_key=$2
	`, storeID, in.IdempotencyKey).Scan(&existingID)
	if err == nil {
		return s.getPaymentByID(ctx, storeID, existingID)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	o, err := s.GetOrder(ctx, storeID, in.OrderID)
	if err != nil {
		return nil, err
	}
	if o.OrderStatus == OrderCancelled {
		return nil, ErrConflict
	}

	for _, a := range allocs {
		if a.OrderID == in.OrderID {
			continue
		}
		if _, err := s.GetOrder(ctx, storeID, a.OrderID); err != nil {
			return nil, err
		}
	}

	var method PaymentMethod
	err = s.DB.QueryRow(ctx, `
		SELECT id, bank_name, account_holder, account_number, instructions
		FROM store_payment_methods WHERE id=$1 AND store_id=$2 AND is_enabled=true
	`, in.PaymentMethodID, storeID).Scan(
		&method.ID, &method.BankName, &method.AccountHolder, &method.AccountNumber, &method.Instructions,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var payID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO payments (
			store_id, order_id, payment_method_id, method_type, amount_idr, status,
			bank_name_snapshot, account_holder_snapshot, account_number_snapshot, instructions_snapshot,
			payment_kind, idempotency_key, note
		) VALUES ($1,$2,$3,'bank_transfer',$4,'pending',$5,$6,$7,$8,$9,$10,$11)
		RETURNING id
	`, storeID, in.OrderID, method.ID, in.AmountIDR,
		method.BankName, method.AccountHolder, method.AccountNumber, method.Instructions,
		kind, in.IdempotencyKey, in.Note,
	).Scan(&payID)
	if isUniqueViolation(err) {
		_ = tx.Rollback(ctx)
		return s.getPaymentByID(ctx, storeID, existingOrLookup(ctx, s, storeID, in.IdempotencyKey))
	}
	if err != nil {
		return nil, err
	}

	for _, a := range allocs {
		_, err = tx.Exec(ctx, `
			INSERT INTO payment_allocations (store_id, payment_id, order_id, amount_idr)
			VALUES ($1,$2,$3,$4)
		`, storeID, payID, a.OrderID, a.AmountIDR)
		if err != nil {
			return nil, err
		}
	}

	actor := &actorID
	oid := &in.OrderID
	_, err = s.insertFinancialEventTx(ctx, tx, storeID, o.CustomerID, oid, &payID, nil,
		"payment_recorded", "credit", in.AmountIDR, "payment_recorded:"+in.IdempotencyKey,
		map[string]any{"payment_kind": kind, "allocations": len(allocs)}, actor)
	if err != nil && !errors.Is(err, ErrIdempotencyConflict) {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.getPaymentByID(ctx, storeID, payID)
}

func existingOrLookup(ctx context.Context, s *Service, storeID uuid.UUID, key string) uuid.UUID {
	var id uuid.UUID
	_ = s.DB.QueryRow(ctx, `SELECT id FROM payments WHERE store_id=$1 AND idempotency_key=$2`, storeID, key).Scan(&id)
	return id
}

func (s *Service) getPaymentByID(ctx context.Context, storeID, paymentID uuid.UUID) (*Payment, error) {
	row := s.DB.QueryRow(ctx, `
		SELECT id, store_id, order_id, payment_method_id, method_type, amount_idr, status,
		       bank_name_snapshot, account_holder_snapshot, account_number_snapshot, instructions_snapshot,
		       verified_at, rejected_at, reject_reason, created_at, updated_at,
		       payment_kind, idempotency_key, note, version
		FROM payments WHERE id=$1 AND store_id=$2
	`, paymentID, storeID)
	p, err := scanPaymentExtended(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	ev, err := s.listEvidence(ctx, storeID, p.ID)
	if err != nil {
		return nil, err
	}
	p.Evidence = ev
	allocs, err := s.listPaymentAllocations(ctx, storeID, p.ID)
	if err != nil {
		return nil, err
	}
	p.Allocations = allocs
	return p, nil
}

// VerifyPaymentExtended verifies a payment (including multi-allocation) and recomputes settlement.
// Preserves existing VerifyPayment for single-order flow; this is additive for finance workspace.
func (s *Service) VerifyPaymentExtended(ctx context.Context, storeID, orderID, paymentID, actorID uuid.UUID) (*Payment, error) {
	pay, err := s.getPaymentByID(ctx, storeID, paymentID)
	if err != nil {
		return nil, err
	}
	if pay.OrderID != orderID && !allocationContains(pay.Allocations, orderID) {
		return nil, ErrNotFound
	}
	if pay.Status == PaymentRecordVerified {
		return pay, ErrAlreadyVerified
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	ct, err := tx.Exec(ctx, `
		UPDATE payments SET status='verified', verified_at=now(), verified_by=$3, updated_at=now(),
			rejected_at=NULL, rejected_by=NULL, reject_reason=NULL, version=version+1
		WHERE id=$1 AND store_id=$2 AND status IN ('awaiting_verification','pending')
	`, paymentID, storeID, actorID)
	if err != nil {
		return nil, err
	}
	if ct.RowsAffected() == 0 {
		return nil, ErrAlreadyVerified
	}

	orders := uniqueOrderIDs(pay)
	for _, oid := range orders {
		if _, err := s.recomputeSettlementTx(ctx, tx, storeID, oid); err != nil {
			return nil, err
		}
		_, _ = tx.Exec(ctx, `
			INSERT INTO order_status_history (store_id, order_id, dimension, from_status, to_status, actor_user_id, reason)
			VALUES ($1,$2,'payment', (SELECT payment_status FROM orders WHERE id=$2), 'paid', $3, 'verified')
		`, storeID, oid, actorID)
	}

	oid := &pay.OrderID
	_, err = s.insertFinancialEventTx(ctx, tx, storeID, nil, oid, &paymentID, nil,
		"payment_verified", "credit", pay.AmountIDR, "payment_verified:"+paymentID.String(),
		map[string]any{"orders": len(orders)}, &actorID)
	if err != nil && !errors.Is(err, ErrIdempotencyConflict) {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	if s.Outbox != nil {
		_ = s.Outbox.Write(ctx, &storeID, "payment.payment_verified", map[string]any{
			"order_id": orderID.String(), "payment_id": paymentID.String(),
		}, "payment_verified:"+paymentID.String())
	}
	return s.getPaymentByID(ctx, storeID, paymentID)
}

func allocationContains(allocs []PaymentAllocation, orderID uuid.UUID) bool {
	for _, a := range allocs {
		if a.OrderID == orderID {
			return true
		}
	}
	return false
}

func uniqueOrderIDs(pay *Payment) []uuid.UUID {
	seen := map[uuid.UUID]struct{}{}
	var out []uuid.UUID
	if pay.OrderID != uuid.Nil {
		seen[pay.OrderID] = struct{}{}
		out = append(out, pay.OrderID)
	}
	for _, a := range pay.Allocations {
		if _, ok := seen[a.OrderID]; ok {
			continue
		}
		seen[a.OrderID] = struct{}{}
		out = append(out, a.OrderID)
	}
	return out
}
