package commerce

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type PurchasingLine struct {
	ProductID      uuid.UUID  `json:"product_id"`
	VariantID      *uuid.UUID `json:"variant_id,omitempty"`
	ProductName    string     `json:"product_name"`
	SKU            *string    `json:"sku,omitempty"`
	VariantLabel   *string    `json:"variant_label,omitempty"`
	RequiredQty    int        `json:"required_qty"`    // confirmed + paid
	UnpaidDemand   int        `json:"unpaid_demand"`   // confirmed but unpaid
	PurchasedQty   int        `json:"purchased_qty"`
	UnavailableQty int        `json:"unavailable_qty"`
	AllocatedQty   int        `json:"allocated_qty"`
	OutstandingQty int        `json:"outstanding_qty"` // required - purchased (floored at 0)
}

type UpdatePurchasingInput struct {
	ProductID      uuid.UUID  `json:"product_id"`
	VariantID      *uuid.UUID `json:"variant_id,omitempty"`
	PurchasedQty   *int       `json:"purchased_qty,omitempty"`
	UnavailableQty *int       `json:"unavailable_qty,omitempty"`
	Notes          *string    `json:"notes,omitempty"`
}

type AllocateInput struct {
	OrderItemID    uuid.UUID `json:"order_item_id"`
	AllocatedQty   int       `json:"allocated_qty"`
	UnavailableQty int       `json:"unavailable_qty"`
	Resolution     *string   `json:"resolution,omitempty"`
}

func (s *Service) GetPurchasingList(ctx context.Context, storeID, batchID uuid.UUID) ([]PurchasingLine, error) {
	if _, err := s.GetBatch(ctx, storeID, batchID, false); err != nil {
		return nil, err
	}
	rows, err := s.DB.Query(ctx, `
		SELECT oi.product_id, oi.variant_id,
		       MAX(oi.product_name) AS product_name,
		       MAX(oi.sku) AS sku,
		       MAX(oi.variant_label) AS variant_label,
		       COALESCE(SUM(oi.quantity) FILTER (
		         WHERE o.order_status IN ('confirmed','completed')
		           AND o.payment_status='paid'
		       ),0) AS required_qty,
		       COALESCE(SUM(oi.quantity) FILTER (
		         WHERE o.order_status IN ('confirmed','pending_confirmation')
		           AND o.payment_status IN ('unpaid','awaiting_verification')
		       ),0) AS unpaid_demand
		FROM order_items oi
		JOIN orders o ON o.id=oi.order_id AND o.store_id=oi.store_id
		WHERE oi.store_id=$1 AND o.preorder_batch_id=$2
		  AND o.order_status NOT IN ('cancelled')
		GROUP BY oi.product_id, oi.variant_id
		ORDER BY product_name
	`, storeID, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PurchasingLine
	for rows.Next() {
		var line PurchasingLine
		if err := rows.Scan(
			&line.ProductID, &line.VariantID, &line.ProductName, &line.SKU, &line.VariantLabel,
			&line.RequiredQty, &line.UnpaidDemand,
		); err != nil {
			return nil, err
		}
		_ = s.DB.QueryRow(ctx, `
			SELECT purchased_qty, unavailable_qty FROM preorder_purchasing_records
			WHERE store_id=$1 AND batch_id=$2 AND product_id=$3
			  AND (($4::uuid IS NULL AND variant_id IS NULL) OR variant_id=$4)
		`, storeID, batchID, line.ProductID, line.VariantID).Scan(&line.PurchasedQty, &line.UnavailableQty)
		_ = s.DB.QueryRow(ctx, `
			SELECT COALESCE(SUM(allocated_qty),0) FROM preorder_item_allocations
			WHERE store_id=$1 AND batch_id=$2 AND product_id=$3
			  AND (($4::uuid IS NULL AND variant_id IS NULL) OR variant_id=$4)
		`, storeID, batchID, line.ProductID, line.VariantID).Scan(&line.AllocatedQty)
		outQty := line.RequiredQty - line.PurchasedQty
		if outQty < 0 {
			outQty = 0
		}
		line.OutstandingQty = outQty
		out = append(out, line)
	}
	return out, rows.Err()
}

func (s *Service) UpdatePurchasing(ctx context.Context, storeID, batchID, actorID uuid.UUID, in UpdatePurchasingInput) error {
	if _, err := s.GetBatch(ctx, storeID, batchID, false); err != nil {
		return err
	}
	purchased := 0
	unavailable := 0
	if in.PurchasedQty != nil {
		if *in.PurchasedQty < 0 {
			return ErrInvalidInput
		}
		purchased = *in.PurchasedQty
	}
	if in.UnavailableQty != nil {
		if *in.UnavailableQty < 0 {
			return ErrInvalidInput
		}
		unavailable = *in.UnavailableQty
	}
	_, err := s.DB.Exec(ctx, `
		INSERT INTO preorder_purchasing_records (
			store_id, batch_id, product_id, variant_id, purchased_qty, unavailable_qty, notes, updated_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (batch_id, product_id, variant_id) DO UPDATE SET
			purchased_qty=EXCLUDED.purchased_qty,
			unavailable_qty=EXCLUDED.unavailable_qty,
			notes=COALESCE(EXCLUDED.notes, preorder_purchasing_records.notes),
			updated_by=EXCLUDED.updated_by,
			updated_at=now()
	`, storeID, batchID, in.ProductID, in.VariantID, purchased, unavailable, in.Notes, actorID)
	return err
}

func (s *Service) AllocatePurchased(ctx context.Context, storeID, batchID uuid.UUID, in AllocateInput) error {
	if in.AllocatedQty < 0 || in.UnavailableQty < 0 {
		return ErrInvalidInput
	}
	var orderID, productID uuid.UUID
	var variantID *uuid.UUID
	var orderedQty int
	err := s.DB.QueryRow(ctx, `
		SELECT oi.order_id, oi.product_id, oi.variant_id, oi.quantity
		FROM order_items oi
		JOIN orders o ON o.id=oi.order_id AND o.store_id=oi.store_id
		WHERE oi.id=$1 AND oi.store_id=$2 AND o.preorder_batch_id=$3
	`, in.OrderItemID, storeID, batchID).Scan(&orderID, &productID, &variantID, &orderedQty)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if in.AllocatedQty+in.UnavailableQty > orderedQty {
		return ErrInvalidInput
	}
	var purchased int
	_ = s.DB.QueryRow(ctx, `
		SELECT purchased_qty FROM preorder_purchasing_records
		WHERE store_id=$1 AND batch_id=$2 AND product_id=$3
		  AND (($4::uuid IS NULL AND variant_id IS NULL) OR variant_id=$4)
	`, storeID, batchID, productID, variantID).Scan(&purchased)
	var alreadyAllocated int
	_ = s.DB.QueryRow(ctx, `
		SELECT COALESCE(SUM(allocated_qty),0) FROM preorder_item_allocations
		WHERE store_id=$1 AND batch_id=$2 AND product_id=$3
		  AND (($4::uuid IS NULL AND variant_id IS NULL) OR variant_id=$4)
		  AND order_item_id <> $5
	`, storeID, batchID, productID, variantID, in.OrderItemID).Scan(&alreadyAllocated)
	if alreadyAllocated+in.AllocatedQty > purchased {
		return ErrInvalidInput
	}
	_, err = s.DB.Exec(ctx, `
		INSERT INTO preorder_item_allocations (
			store_id, batch_id, order_id, order_item_id, product_id, variant_id,
			allocated_qty, unavailable_qty, resolution
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (order_item_id) DO UPDATE SET
			allocated_qty=EXCLUDED.allocated_qty,
			unavailable_qty=EXCLUDED.unavailable_qty,
			resolution=EXCLUDED.resolution,
			updated_at=now()
	`, storeID, batchID, orderID, in.OrderItemID, productID, variantID,
		in.AllocatedQty, in.UnavailableQty, in.Resolution)
	return err
}

func (s *Service) ListBatchOrders(ctx context.Context, storeID, batchID uuid.UUID, limit, offset int) ([]Order, int, error) {
	if _, err := s.GetBatch(ctx, storeID, batchID, false); err != nil {
		return nil, 0, err
	}
	f := ListOrdersFilter{Limit: limit, Offset: offset}
	// reuse list with batch filter via direct query
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 25
	}
	var total int
	if err := s.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM orders WHERE store_id=$1 AND preorder_batch_id=$2
	`, storeID, batchID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.DB.Query(ctx, `
		SELECT id, store_id, customer_id, order_number, selling_model,
		       order_status, payment_status, fulfillment_status, currency,
		       subtotal_idr, shipping_fee_idr, handling_fee_idr, total_idr,
		       fulfillment_method, customer_name, customer_email, customer_phone,
		       shipping_recipient, shipping_phone, shipping_line1, shipping_line2,
		       shipping_city, shipping_province, shipping_postal_code, shipping_country,
		       customer_notes, internal_notes, cancel_reason,
		       payment_deadline_at, confirmed_at, completed_at, cancelled_at,
		       created_at, updated_at,
		       preorder_batch_id, batch_name_snapshot, batch_slug_snapshot,
		       requires_approval
		FROM orders WHERE store_id=$1 AND preorder_batch_id=$2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`, storeID, batchID, f.Limit, f.Offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []Order
	for rows.Next() {
		o, err := scanOrderExtended(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *o)
	}
	return out, total, rows.Err()
}
