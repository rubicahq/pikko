package commerce

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ReceivingRecord struct {
	ID         uuid.UUID     `json:"id"`
	StoreID    uuid.UUID     `json:"store_id"`
	BatchID    *uuid.UUID    `json:"batch_id,omitempty"`
	VendorID   *uuid.UUID    `json:"vendor_id,omitempty"`
	VendorName *string       `json:"vendor_name,omitempty"`
	BatchName  *string       `json:"batch_name,omitempty"`
	Status     string        `json:"status"`
	Notes      *string       `json:"notes,omitempty"`
	ReceivedAt *time.Time    `json:"received_at,omitempty"`
	CreatedBy  *uuid.UUID    `json:"created_by,omitempty"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
	Lines      []ReceivingLine `json:"lines,omitempty"`
}

type ReceivingLine struct {
	ID                 uuid.UUID  `json:"id"`
	StoreID            uuid.UUID  `json:"store_id"`
	ReceivingID        uuid.UUID  `json:"receiving_id"`
	ProductID          uuid.UUID  `json:"product_id"`
	VariantID          *uuid.UUID `json:"variant_id,omitempty"`
	PurchasingRecordID *uuid.UUID `json:"purchasing_record_id,omitempty"`
	ProductName        *string    `json:"product_name,omitempty"`
	VariantLabel       *string    `json:"variant_label,omitempty"`
	ExpectedQty        int        `json:"expected_qty"`
	ReceivedQty        int        `json:"received_qty"`
	DiscrepancyNotes   *string    `json:"discrepancy_notes,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type CreateReceivingInput struct {
	BatchID  *uuid.UUID             `json:"batch_id,omitempty"`
	VendorID *uuid.UUID             `json:"vendor_id,omitempty"`
	Notes    *string                `json:"notes,omitempty"`
	Lines    []CreateReceivingLine  `json:"lines,omitempty"`
}

type CreateReceivingLine struct {
	ProductID          uuid.UUID  `json:"product_id"`
	VariantID          *uuid.UUID `json:"variant_id,omitempty"`
	PurchasingRecordID *uuid.UUID `json:"purchasing_record_id,omitempty"`
	ExpectedQty        int        `json:"expected_qty"`
	ReceivedQty        int        `json:"received_qty"`
	DiscrepancyNotes   *string    `json:"discrepancy_notes,omitempty"`
}

type UpdateReceivingLineInput struct {
	ReceivedQty      *int    `json:"received_qty,omitempty"`
	DiscrepancyNotes *string `json:"discrepancy_notes,omitempty"`
}

func (s *Service) ListReceiving(ctx context.Context, storeID uuid.UUID, batchID *uuid.UUID) ([]ReceivingRecord, error) {
	args := []any{storeID}
	where := "r.store_id=$1 AND r.status <> 'cancelled'"
	if batchID != nil {
		where += " AND r.batch_id=$2"
		args = append(args, *batchID)
	}
	rows, err := s.DB.Query(ctx, `
		SELECT r.id, r.store_id, r.batch_id, r.vendor_id, v.name, b.name,
		       r.status, r.notes, r.received_at, r.created_by, r.created_at, r.updated_at
		FROM receiving_records r
		LEFT JOIN vendors v ON v.id=r.vendor_id AND v.store_id=r.store_id
		LEFT JOIN preorder_batches b ON b.id=r.batch_id AND b.store_id=r.store_id
		WHERE `+where+`
		ORDER BY r.created_at DESC
		LIMIT 100
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ReceivingRecord
	for rows.Next() {
		var r ReceivingRecord
		if err := rows.Scan(
			&r.ID, &r.StoreID, &r.BatchID, &r.VendorID, &r.VendorName, &r.BatchName,
			&r.Status, &r.Notes, &r.ReceivedAt, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Service) GetReceiving(ctx context.Context, storeID, receivingID uuid.UUID) (*ReceivingRecord, error) {
	var r ReceivingRecord
	err := s.DB.QueryRow(ctx, `
		SELECT r.id, r.store_id, r.batch_id, r.vendor_id, v.name, b.name,
		       r.status, r.notes, r.received_at, r.created_by, r.created_at, r.updated_at
		FROM receiving_records r
		LEFT JOIN vendors v ON v.id=r.vendor_id AND v.store_id=r.store_id
		LEFT JOIN preorder_batches b ON b.id=r.batch_id AND b.store_id=r.store_id
		WHERE r.id=$1 AND r.store_id=$2
	`, receivingID, storeID).Scan(
		&r.ID, &r.StoreID, &r.BatchID, &r.VendorID, &r.VendorName, &r.BatchName,
		&r.Status, &r.Notes, &r.ReceivedAt, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	lines, err := s.listReceivingLines(ctx, storeID, receivingID)
	if err != nil {
		return nil, err
	}
	r.Lines = lines
	return &r, nil
}

func (s *Service) listReceivingLines(ctx context.Context, storeID, receivingID uuid.UUID) ([]ReceivingLine, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT rl.id, rl.store_id, rl.receiving_id, rl.product_id, rl.variant_id,
		       rl.purchasing_record_id, p.name, pv.option_key,
		       rl.expected_qty, rl.received_qty, rl.discrepancy_notes,
		       rl.created_at, rl.updated_at
		FROM receiving_lines rl
		LEFT JOIN products p ON p.id=rl.product_id
		LEFT JOIN product_variants pv ON pv.id=rl.variant_id
		WHERE rl.store_id=$1 AND rl.receiving_id=$2
		ORDER BY p.name NULLS LAST, pv.option_key NULLS FIRST
	`, storeID, receivingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ReceivingLine
	for rows.Next() {
		var l ReceivingLine
		if err := rows.Scan(
			&l.ID, &l.StoreID, &l.ReceivingID, &l.ProductID, &l.VariantID,
			&l.PurchasingRecordID, &l.ProductName, &l.VariantLabel,
			&l.ExpectedQty, &l.ReceivedQty, &l.DiscrepancyNotes,
			&l.CreatedAt, &l.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (s *Service) CreateReceiving(ctx context.Context, storeID, actorID uuid.UUID, in CreateReceivingInput) (*ReceivingRecord, error) {
	if in.BatchID != nil {
		if _, err := s.GetBatch(ctx, storeID, *in.BatchID, false); err != nil {
			return nil, err
		}
	}
	if in.VendorID != nil {
		var ok bool
		err := s.DB.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM vendors WHERE id=$1 AND store_id=$2)
		`, *in.VendorID, storeID).Scan(&ok)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrInvalidInput
		}
	}
	for _, line := range in.Lines {
		if line.ProductID == uuid.Nil || line.ExpectedQty < 0 || line.ReceivedQty < 0 {
			return nil, ErrInvalidInput
		}
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var id uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO receiving_records (store_id, batch_id, vendor_id, notes, created_by, status)
		VALUES ($1,$2,$3,$4,$5,'draft')
		RETURNING id
	`, storeID, in.BatchID, in.VendorID, in.Notes, actorID).Scan(&id)
	if err != nil {
		return nil, err
	}

	for _, line := range in.Lines {
		_, err = tx.Exec(ctx, `
			INSERT INTO receiving_lines (
				store_id, receiving_id, product_id, variant_id, purchasing_record_id,
				expected_qty, received_qty, discrepancy_notes
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		`, storeID, id, line.ProductID, line.VariantID, line.PurchasingRecordID,
			line.ExpectedQty, line.ReceivedQty, line.DiscrepancyNotes)
		if err != nil {
			return nil, err
		}
	}

	status := deriveReceivingStatus(in.Lines)
	if status != "draft" {
		now := time.Now().UTC()
		_, err = tx.Exec(ctx, `
			UPDATE receiving_records SET status=$2, received_at=$3, updated_at=now() WHERE id=$1
		`, id, status, now)
		if err != nil {
			return nil, err
		}
		if err := s.applyReceivingToPurchasing(ctx, tx, storeID, in.Lines); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetReceiving(ctx, storeID, id)
}

func deriveReceivingStatus(lines []CreateReceivingLine) string {
	if len(lines) == 0 {
		return "draft"
	}
	anyRecv := false
	allComplete := true
	for _, l := range lines {
		if l.ReceivedQty > 0 {
			anyRecv = true
		}
		if l.ReceivedQty < l.ExpectedQty {
			allComplete = false
		}
	}
	if !anyRecv {
		return "draft"
	}
	if allComplete {
		return "complete"
	}
	return "partial"
}

func (s *Service) applyReceivingToPurchasing(ctx context.Context, tx pgx.Tx, storeID uuid.UUID, lines []CreateReceivingLine) error {
	for _, line := range lines {
		if line.ReceivedQty <= 0 {
			continue
		}
		if line.PurchasingRecordID != nil {
			_, err := tx.Exec(ctx, `
				UPDATE preorder_purchasing_records
				SET received_qty = received_qty + $2, updated_at=now(), version=version+1
				WHERE id=$1 AND store_id=$3
			`, *line.PurchasingRecordID, line.ReceivedQty, storeID)
			if err != nil {
				return err
			}
			continue
		}
		// Best-effort match unassigned purchasing record for product/variant when batch known via record.
		_, _ = tx.Exec(ctx, `
			UPDATE preorder_purchasing_records
			SET received_qty = received_qty + $3, updated_at=now(), version=version+1
			WHERE store_id=$1 AND product_id=$2
			  AND (($4::uuid IS NULL AND variant_id IS NULL) OR variant_id=$4)
			  AND id = (
			    SELECT id FROM preorder_purchasing_records
			    WHERE store_id=$1 AND product_id=$2
			      AND (($4::uuid IS NULL AND variant_id IS NULL) OR variant_id=$4)
			    ORDER BY updated_at DESC LIMIT 1
			  )
		`, storeID, line.ProductID, line.ReceivedQty, line.VariantID)
	}
	return nil
}

func (s *Service) UpdateReceivingLine(
	ctx context.Context, storeID, receivingID, lineID uuid.UUID, in UpdateReceivingLineInput,
) (*ReceivingRecord, error) {
	rec, err := s.GetReceiving(ctx, storeID, receivingID)
	if err != nil {
		return nil, err
	}
	if rec.Status == "cancelled" {
		return nil, ErrInvalidTransition
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var cur ReceivingLine
	err = tx.QueryRow(ctx, `
		SELECT id, product_id, variant_id, purchasing_record_id, expected_qty, received_qty
		FROM receiving_lines
		WHERE id=$1 AND receiving_id=$2 AND store_id=$3
		FOR UPDATE
	`, lineID, receivingID, storeID).Scan(
		&cur.ID, &cur.ProductID, &cur.VariantID, &cur.PurchasingRecordID, &cur.ExpectedQty, &cur.ReceivedQty,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	newRecv := cur.ReceivedQty
	if in.ReceivedQty != nil {
		if *in.ReceivedQty < 0 {
			return nil, ErrInvalidInput
		}
		newRecv = *in.ReceivedQty
	}
	delta := newRecv - cur.ReceivedQty

	_, err = tx.Exec(ctx, `
		UPDATE receiving_lines SET
			received_qty=$4,
			discrepancy_notes=COALESCE($5, discrepancy_notes),
			updated_at=now()
		WHERE id=$1 AND receiving_id=$2 AND store_id=$3
	`, lineID, receivingID, storeID, newRecv, in.DiscrepancyNotes)
	if err != nil {
		return nil, err
	}

	if delta != 0 && cur.PurchasingRecordID != nil {
		_, err = tx.Exec(ctx, `
			UPDATE preorder_purchasing_records
			SET received_qty = GREATEST(0, received_qty + $2), updated_at=now(), version=version+1
			WHERE id=$1 AND store_id=$3
		`, *cur.PurchasingRecordID, delta, storeID)
		if err != nil {
			return nil, err
		}
	}

	// Recompute header status
	var expSum, recvSum int
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(expected_qty),0), COALESCE(SUM(received_qty),0)
		FROM receiving_lines WHERE receiving_id=$1 AND store_id=$2
	`, receivingID, storeID).Scan(&expSum, &recvSum)
	if err != nil {
		return nil, err
	}
	status := "draft"
	var receivedAt any
	if recvSum > 0 && recvSum < expSum {
		status = "partial"
		receivedAt = time.Now().UTC()
	} else if recvSum > 0 && recvSum >= expSum {
		status = "complete"
		receivedAt = time.Now().UTC()
	}
	_, err = tx.Exec(ctx, `
		UPDATE receiving_records SET status=$3, received_at=COALESCE(received_at, $4::timestamptz), updated_at=now()
		WHERE id=$1 AND store_id=$2
	`, receivingID, storeID, status, receivedAt)
	if err != nil {
		return nil, err
	}

	// RCV-03: when complete against a batch, nudge fulfillment toward packing eligibility.
	if status == "complete" && rec.BatchID != nil {
		_, _ = tx.Exec(ctx, `
			UPDATE orders SET fulfillment_status='awaiting_arrival', updated_at=now()
			WHERE store_id=$1 AND preorder_batch_id=$2
			  AND fulfillment_status='purchasing'
			  AND order_status NOT IN ('cancelled')
		`, storeID, *rec.BatchID)
		_, _ = tx.Exec(ctx, `
			UPDATE preorder_batches SET operational_status='arrived', updated_at=now()
			WHERE id=$1 AND store_id=$2
			  AND operational_status IN ('purchasing','awaiting_arrival')
		`, *rec.BatchID, storeID)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetReceiving(ctx, storeID, receivingID)
}
