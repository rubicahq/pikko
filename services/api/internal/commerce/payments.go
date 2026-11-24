package commerce

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rubicahq/pikko/services/api/internal/email"
	"github.com/rubicahq/pikko/services/api/internal/money"
)

func (s *Service) GetLatestPayment(ctx context.Context, storeID, orderID uuid.UUID) (*Payment, error) {
	var p Payment
	err := s.DB.QueryRow(ctx, `
		SELECT id, store_id, order_id, payment_method_id, method_type, amount_idr, status,
		       bank_name_snapshot, account_holder_snapshot, account_number_snapshot, instructions_snapshot,
		       verified_at, rejected_at, reject_reason, created_at, updated_at
		FROM payments WHERE store_id=$1 AND order_id=$2
		ORDER BY created_at DESC LIMIT 1
	`, storeID, orderID).Scan(
		&p.ID, &p.StoreID, &p.OrderID, &p.PaymentMethodID, &p.MethodType, &p.AmountIDR, &p.Status,
		&p.BankNameSnapshot, &p.AccountHolderSnapshot, &p.AccountNumberSnapshot, &p.InstructionsSnapshot,
		&p.VerifiedAt, &p.RejectedAt, &p.RejectReason, &p.CreatedAt, &p.UpdatedAt,
	)
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
	return &p, nil
}

func (s *Service) listEvidence(ctx context.Context, storeID, paymentID uuid.UUID) ([]Evidence, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT e.id, e.file_id, e.submitted_at, e.note, f.content_type
		FROM payment_evidence e
		JOIN stored_files f ON f.id=e.file_id AND f.store_id=e.store_id
		WHERE e.payment_id=$1 AND e.store_id=$2
		ORDER BY e.submitted_at DESC
	`, paymentID, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Evidence
	for rows.Next() {
		var e Evidence
		if err := rows.Scan(&e.ID, &e.FileID, &e.SubmittedAt, &e.Note, &e.ContentType); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Service) SubmitEvidenceByTracking(
	ctx context.Context, rawToken string, filename, contentType string, r io.Reader, size int64, note *string,
) (*Payment, error) {
	view, orderID, storeID, err := s.resolveTracking(ctx, rawToken)
	if err != nil {
		return nil, err
	}
	_ = view
	o, err := s.GetOrder(ctx, storeID, orderID)
	if err != nil {
		return nil, err
	}
	if o.OrderStatus == OrderCancelled {
		return nil, ErrConflict
	}
	if o.PaymentStatus == PaymentPaid {
		return nil, ErrAlreadyVerified
	}
	pay, err := s.GetLatestPayment(ctx, storeID, orderID)
	if err != nil {
		return nil, err
	}
	if pay.Status == PaymentRecordVerified {
		return nil, ErrAlreadyVerified
	}

	if s.Files == nil {
		return nil, errors.New("files service unavailable")
	}
	f, err := s.Files.Upload(ctx, storeID, uuid.Nil, "payment_evidence", filename, contentType, r, size)
	if err != nil {
		return nil, err
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO payment_evidence (store_id, payment_id, order_id, file_id, note)
		VALUES ($1,$2,$3,$4,$5)
	`, storeID, pay.ID, orderID, f.ID, note)
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `
		UPDATE payments SET status='awaiting_verification', updated_at=now()
		WHERE id=$1 AND store_id=$2 AND status IN ('pending','awaiting_verification','rejected')
	`, pay.ID, storeID)
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `
		UPDATE orders SET payment_status='awaiting_verification', updated_at=now()
		WHERE id=$1 AND store_id=$2 AND payment_status IN ('unpaid','awaiting_verification','failed')
	`, orderID, storeID)
	if err != nil {
		return nil, err
	}
	_, _ = tx.Exec(ctx, `
		INSERT INTO order_status_history (store_id, order_id, dimension, from_status, to_status, reason)
		VALUES ($1,$2,'payment',$3,'awaiting_verification','evidence_submitted')
	`, storeID, orderID, o.PaymentStatus)

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	if s.Outbox != nil {
		_ = s.Outbox.Write(ctx, &storeID, "payment.payment_evidence_submitted", map[string]any{
			"order_id": orderID.String(), "payment_id": pay.ID.String(),
		}, "evidence:"+pay.ID.String()+":"+f.ID.String())
	}
	return s.GetLatestPayment(ctx, storeID, orderID)
}

func (s *Service) VerifyPayment(ctx context.Context, storeID, orderID, paymentID, actorID uuid.UUID) (*Payment, error) {
	o, err := s.GetOrder(ctx, storeID, orderID)
	if err != nil {
		return nil, err
	}
	if o.OrderStatus == OrderCancelled {
		return nil, ErrInvalidTransition
	}
	pay, err := s.GetLatestPayment(ctx, storeID, orderID)
	if err != nil {
		return nil, err
	}
	if pay.ID != paymentID {
		return nil, ErrNotFound
	}
	if pay.Status == PaymentRecordVerified {
		return nil, ErrAlreadyVerified
	}
	if pay.Status != PaymentRecordAwaitingVerification && pay.Status != PaymentRecordPending {
		return nil, ErrInvalidTransition
	}
	ev, _ := s.listEvidence(ctx, storeID, pay.ID)
	if len(ev) == 0 {
		return nil, ErrInvalidInput
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	ct, err := tx.Exec(ctx, `
		UPDATE payments SET status='verified', verified_at=now(), verified_by=$3, updated_at=now(),
			rejected_at=NULL, rejected_by=NULL, reject_reason=NULL
		WHERE id=$1 AND store_id=$2 AND status IN ('awaiting_verification','pending')
	`, paymentID, storeID, actorID)
	if err != nil {
		return nil, err
	}
	if ct.RowsAffected() == 0 {
		return nil, ErrAlreadyVerified
	}
	_, err = tx.Exec(ctx, `
		UPDATE orders SET payment_status='paid', updated_at=now() WHERE id=$1 AND store_id=$2
	`, orderID, storeID)
	if err != nil {
		return nil, err
	}
	_, _ = tx.Exec(ctx, `
		INSERT INTO order_status_history (store_id, order_id, dimension, from_status, to_status, actor_user_id, reason)
		VALUES ($1,$2,'payment',$3,'paid',$4,'verified')
	`, storeID, orderID, o.PaymentStatus, actorID)

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	if s.Outbox != nil {
		_ = s.Outbox.Write(ctx, &storeID, "payment.payment_verified", map[string]any{
			"order_id": orderID.String(), "payment_id": paymentID.String(),
		}, "payment_verified:"+paymentID.String())
	}

	s.notifyPaymentVerified(ctx, storeID, o)

	return s.GetLatestPayment(ctx, storeID, orderID)
}

func (s *Service) RejectPayment(ctx context.Context, storeID, orderID, paymentID, actorID uuid.UUID, reason string) (*Payment, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, ErrInvalidInput
	}
	o, err := s.GetOrder(ctx, storeID, orderID)
	if err != nil {
		return nil, err
	}
	pay, err := s.GetLatestPayment(ctx, storeID, orderID)
	if err != nil {
		return nil, err
	}
	if pay.ID != paymentID {
		return nil, ErrNotFound
	}
	if pay.Status == PaymentRecordVerified {
		return nil, ErrAlreadyVerified
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE payments SET status='rejected', rejected_at=now(), rejected_by=$3, reject_reason=$4, updated_at=now()
		WHERE id=$1 AND store_id=$2 AND status IN ('awaiting_verification','pending')
	`, paymentID, storeID, actorID, reason)
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `
		UPDATE orders SET payment_status='unpaid', updated_at=now() WHERE id=$1 AND store_id=$2
	`, orderID, storeID)
	if err != nil {
		return nil, err
	}
	_, _ = tx.Exec(ctx, `
		INSERT INTO order_status_history (store_id, order_id, dimension, from_status, to_status, actor_user_id, reason)
		VALUES ($1,$2,'payment',$3,'unpaid',$4,$5)
	`, storeID, orderID, o.PaymentStatus, actorID, reason)

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	if s.Outbox != nil {
		_ = s.Outbox.Write(ctx, &storeID, "payment.payment_rejected", map[string]any{
			"order_id": orderID.String(), "payment_id": paymentID.String(),
		}, "payment_rejected:"+paymentID.String()+":"+time.Now().UTC().Format(time.RFC3339Nano))
	}

	s.notifyPaymentRejected(ctx, storeID, o, reason)
	return s.GetLatestPayment(ctx, storeID, orderID)
}

func (s *Service) notifyPaymentVerified(ctx context.Context, storeID uuid.UUID, o *Order) {
	if s.Email == nil || s.Cfg == nil {
		return
	}
	var storeName string
	_ = s.DB.QueryRow(ctx, `SELECT name FROM stores WHERE id=$1`, storeID).Scan(&storeName)
	trackURL := ""
	if tok, err := s.rawTrackingTokenForOrder(ctx, storeID, o.ID); err == nil && s.URLs != nil {
		trackURL = s.URLs.Tracking(tok)
	}
	subj, html, text := email.PaymentVerifiedContent(storeName, o.OrderNumber, money.FormatIDR(o.TotalIDR), trackURL)
	_ = s.Email.Enqueue(ctx, email.Message{
		TemplateKey: "payment_verified_customer", To: o.CustomerEmail,
		From: s.Cfg.EmailNotificationFrom, FromName: storeName, ReplyTo: s.Cfg.EmailReplyTo,
		Subject: subj, HTML: html, Text: text,
		IdempotencyKey: "email:payment_verified:" + o.ID.String(),
	})
}

func (s *Service) notifyPaymentRejected(ctx context.Context, storeID uuid.UUID, o *Order, reason string) {
	if s.Email == nil || s.Cfg == nil {
		return
	}
	var storeName string
	_ = s.DB.QueryRow(ctx, `SELECT name FROM stores WHERE id=$1`, storeID).Scan(&storeName)
	trackURL := ""
	if tok, err := s.rawTrackingTokenForOrder(ctx, storeID, o.ID); err == nil && s.URLs != nil {
		trackURL = s.URLs.Tracking(tok)
	}
	subj, html, text := email.PaymentRejectedContent(storeName, o.OrderNumber, reason, trackURL)
	_ = s.Email.Enqueue(ctx, email.Message{
		TemplateKey: "payment_rejected_customer", To: o.CustomerEmail,
		From: s.Cfg.EmailNotificationFrom, FromName: storeName, ReplyTo: s.Cfg.EmailReplyTo,
		Subject: subj, HTML: html, Text: text,
		IdempotencyKey: "email:payment_rejected:" + o.ID.String() + ":" + time.Now().UTC().Format("20060102150405"),
	})
}

// rawTrackingTokenForOrder cannot recover the raw token (only hash stored).
// For follow-up emails we generate a fresh token if needed.
func (s *Service) rawTrackingTokenForOrder(ctx context.Context, storeID, orderID uuid.UUID) (string, error) {
	return s.EnsureTrackingToken(ctx, storeID, orderID)
}
