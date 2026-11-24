package commerce

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type PurchasingLine struct {
	ProductID      uuid.UUID  `json:"product_id"`
	VariantID      *uuid.UUID `json:"variant_id,omitempty"`
	ProductName    string     `json:"product_name"`
	SKU            *string    `json:"sku,omitempty"`
	VariantLabel   *string    `json:"variant_label,omitempty"`
	BrandID        *uuid.UUID `json:"brand_id,omitempty"`
	BrandName      *string    `json:"brand_name,omitempty"`
	VendorID       *uuid.UUID `json:"vendor_id,omitempty"`
	VendorName     *string    `json:"vendor_name,omitempty"`
	CampaignID     *uuid.UUID `json:"campaign_id,omitempty"`
	BatchID        *uuid.UUID `json:"batch_id,omitempty"`
	BatchName      *string    `json:"batch_name,omitempty"`
	RequiredQty    int        `json:"required_qty"`  // confirmed + paid
	UnpaidDemand   int        `json:"unpaid_demand"` // confirmed but unpaid
	PurchasedQty   int        `json:"purchased_qty"`
	UnavailableQty int        `json:"unavailable_qty"`
	AllocatedQty   int        `json:"allocated_qty"`
	ReceivedQty    int        `json:"received_qty"`
	OutstandingQty int        `json:"outstanding_qty"` // required - purchased (floored at 0)
	Version        int        `json:"version,omitempty"`
}

type UpdatePurchasingInput struct {
	ProductID      uuid.UUID  `json:"product_id"`
	VariantID      *uuid.UUID `json:"variant_id,omitempty"`
	VendorID       *uuid.UUID `json:"vendor_id,omitempty"`
	PurchasedQty   *int       `json:"purchased_qty,omitempty"`
	UnavailableQty *int       `json:"unavailable_qty,omitempty"`
	Notes          *string    `json:"notes,omitempty"`
	ExpectedVersion *int      `json:"expected_version,omitempty"`
}

type AllocateInput struct {
	OrderItemID     uuid.UUID `json:"order_item_id"`
	AllocatedQty    int       `json:"allocated_qty"`
	UnavailableQty  int       `json:"unavailable_qty"`
	Resolution      *string   `json:"resolution,omitempty"`
	ApplyAdjustment bool      `json:"apply_adjustment,omitempty"`
	ActorUserID     uuid.UUID `json:"-"`
}

func (s *Service) GetPurchasingList(ctx context.Context, storeID, batchID uuid.UUID) ([]PurchasingLine, error) {
	if _, err := s.GetBatch(ctx, storeID, batchID, false); err != nil {
		return nil, err
	}
	f := PurchasingFilter{BatchID: &batchID}
	return s.AggregatePurchasing(ctx, storeID, f)
}

func (s *Service) UpdatePurchasing(ctx context.Context, storeID, batchID, actorID uuid.UUID, in UpdatePurchasingInput) error {
	if _, err := s.GetBatch(ctx, storeID, batchID, false); err != nil {
		return err
	}
	if in.ProductID == uuid.Nil {
		return ErrInvalidInput
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
	if in.VendorID != nil {
		var ok bool
		err := s.DB.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM vendors WHERE id=$1 AND store_id=$2 AND status='active')
		`, *in.VendorID, storeID).Scan(&ok)
		if err != nil {
			return err
		}
		if !ok {
			return ErrInvalidInput
		}
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var existingID uuid.UUID
	var existingVersion int
	err = tx.QueryRow(ctx, `
		SELECT id, version FROM preorder_purchasing_records
		WHERE store_id=$1 AND batch_id=$2 AND product_id=$3
		  AND (($4::uuid IS NULL AND variant_id IS NULL) OR variant_id=$4)
		  AND (($5::uuid IS NULL AND vendor_id IS NULL) OR vendor_id=$5)
		FOR UPDATE
	`, storeID, batchID, in.ProductID, in.VariantID, in.VendorID).Scan(&existingID, &existingVersion)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if existingID != uuid.Nil {
		if in.ExpectedVersion != nil && *in.ExpectedVersion != existingVersion {
			return ErrConflict
		}
		_, err = tx.Exec(ctx, `
			UPDATE preorder_purchasing_records SET
				purchased_qty=$2,
				unavailable_qty=$3,
				notes=COALESCE($4, notes),
				updated_by=$5,
				updated_at=now(),
				version=version+1
			WHERE id=$1
		`, existingID, purchased, unavailable, in.Notes, actorID)
		if err != nil {
			return err
		}
	} else {
		_, err = tx.Exec(ctx, `
			INSERT INTO preorder_purchasing_records (
				store_id, batch_id, product_id, variant_id, vendor_id,
				purchased_qty, unavailable_qty, notes, updated_by
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		`, storeID, batchID, in.ProductID, in.VariantID, in.VendorID,
			purchased, unavailable, in.Notes, actorID)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// AllocatePurchased assigns purchased qty to an order item with row-level locking
// so concurrent allocators cannot oversell purchased stock.
func (s *Service) AllocatePurchased(ctx context.Context, storeID, batchID uuid.UUID, in AllocateInput) error {
	if in.AllocatedQty < 0 || in.UnavailableQty < 0 {
		return ErrInvalidInput
	}
	if in.AllocatedQty == 0 && in.UnavailableQty == 0 {
		return ErrInvalidInput
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var orderID, productID uuid.UUID
	var variantID *uuid.UUID
	var orderedQty int
	var unitPrice int64
	var productName string
	var availStatus string
	err = tx.QueryRow(ctx, `
		SELECT oi.order_id, oi.product_id, oi.variant_id, oi.quantity,
		       oi.unit_price_idr, oi.product_name, oi.availability_status
		FROM order_items oi
		JOIN orders o ON o.id=oi.order_id AND o.store_id=oi.store_id
		WHERE oi.id=$1 AND oi.store_id=$2 AND o.preorder_batch_id=$3
		FOR UPDATE OF oi
	`, in.OrderItemID, storeID, batchID).Scan(
		&orderID, &productID, &variantID, &orderedQty, &unitPrice, &productName, &availStatus,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if in.AllocatedQty+in.UnavailableQty > orderedQty {
		return ErrInvalidInput
	}

	// Lock all purchasing records for this exact variant (multi-vendor sum).
	rows, err := tx.Query(ctx, `
		SELECT id, purchased_qty FROM preorder_purchasing_records
		WHERE store_id=$1 AND batch_id=$2 AND product_id=$3
		  AND (($4::uuid IS NULL AND variant_id IS NULL) OR variant_id=$4)
		FOR UPDATE
	`, storeID, batchID, productID, variantID)
	if err != nil {
		return err
	}
	purchased := 0
	for rows.Next() {
		var id uuid.UUID
		var qty int
		if err := rows.Scan(&id, &qty); err != nil {
			rows.Close()
			return err
		}
		purchased += qty
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	// Lock sibling allocations for the same variant.
	allocRows, err := tx.Query(ctx, `
		SELECT id, order_item_id, allocated_qty FROM preorder_item_allocations
		WHERE store_id=$1 AND batch_id=$2 AND product_id=$3
		  AND (($4::uuid IS NULL AND variant_id IS NULL) OR variant_id=$4)
		FOR UPDATE
	`, storeID, batchID, productID, variantID)
	if err != nil {
		return err
	}
	alreadyAllocated := 0
	for allocRows.Next() {
		var id, itemID uuid.UUID
		var qty int
		if err := allocRows.Scan(&id, &itemID, &qty); err != nil {
			allocRows.Close()
			return err
		}
		if itemID != in.OrderItemID {
			alreadyAllocated += qty
		}
	}
	allocRows.Close()
	if err := allocRows.Err(); err != nil {
		return err
	}
	if alreadyAllocated+in.AllocatedQty > purchased {
		return ErrConflict
	}

	_, err = tx.Exec(ctx, `
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
	if err != nil {
		return err
	}

	newAvail := "available"
	if in.UnavailableQty >= orderedQty {
		newAvail = "unavailable"
	} else if in.UnavailableQty > 0 {
		newAvail = "partially_unavailable"
	}

	availableQty := orderedQty - in.UnavailableQty
	if availableQty < 0 {
		availableQty = 0
	}
	adjustedLine := unitPrice * int64(availableQty)

	_, err = tx.Exec(ctx, `
		UPDATE order_items SET
			allocated_qty=$4,
			availability_status=$5,
			adjusted_line_total_idr=CASE WHEN $6 THEN $7 ELSE adjusted_line_total_idr END
		WHERE id=$1 AND order_id=$2 AND store_id=$3
	`, in.OrderItemID, orderID, storeID, in.AllocatedQty, newAvail, in.ApplyAdjustment, adjustedLine)
	if err != nil {
		return err
	}

	if in.ApplyAdjustment && in.UnavailableQty > 0 {
		var order SubtotalSnapshot
		err = tx.QueryRow(ctx, `
			SELECT subtotal_idr, shipping_fee_idr, handling_fee_idr, total_idr
			FROM orders WHERE id=$1 AND store_id=$2 FOR UPDATE
		`, orderID, storeID).Scan(&order.Subtotal, &order.Shipping, &order.Handling, &order.Total)
		if err != nil {
			return err
		}
		var newSubtotal int64
		err = tx.QueryRow(ctx, `
			SELECT COALESCE(SUM(COALESCE(adjusted_line_total_idr, line_total_idr)),0)
			FROM order_items WHERE order_id=$1 AND store_id=$2
		`, orderID, storeID).Scan(&newSubtotal)
		if err != nil {
			return err
		}
		newTotal := newSubtotal + order.Shipping + order.Handling
		_, err = tx.Exec(ctx, `
			UPDATE orders SET subtotal_idr=$3, total_idr=$4, updated_at=now()
			WHERE id=$1 AND store_id=$2
		`, orderID, storeID, newSubtotal, newTotal)
		if err != nil {
			return err
		}

		revNum, err := s.nextRevisionNumber(ctx, tx, orderID)
		if err != nil {
			return err
		}
		before, _ := json.Marshal(map[string]any{
			"item_id": in.OrderItemID.String(), "availability": availStatus,
			"subtotal_idr": order.Subtotal, "total_idr": order.Total,
			"unavailable_qty": 0, "allocated_qty": 0,
		})
		after, _ := json.Marshal(map[string]any{
			"item_id": in.OrderItemID.String(), "availability": newAvail,
			"subtotal_idr": newSubtotal, "total_idr": newTotal,
			"unavailable_qty": in.UnavailableQty, "allocated_qty": in.AllocatedQty,
			"adjusted_line_total_idr": adjustedLine,
		})
		summary := fmt.Sprintf("Unavailable adjustment for %s: %d of %d", productName, in.UnavailableQty, orderedQty)
		var revID uuid.UUID
		err = tx.QueryRow(ctx, `
			INSERT INTO order_revisions (
				store_id, order_id, revision_number, change_type, summary,
				before_snapshot, after_snapshot, actor_user_id, reason
			) VALUES ($1,$2,$3,'item_unavailable',$4,$5,$6,$7,$8)
			RETURNING id
		`, storeID, orderID, revNum, summary, before, after, nilUUID(in.ActorUserID), in.Resolution).Scan(&revID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO order_item_events (
				store_id, order_id, order_item_id, revision_id, event_type,
				from_availability, to_availability, meta, actor_user_id, reason
			) VALUES ($1,$2,$3,$4,'availability_changed',$5,$6,$7,$8,$9)
		`, storeID, orderID, in.OrderItemID, revID, availStatus, newAvail,
			mustJSON(map[string]any{
				"allocated_qty": in.AllocatedQty, "unavailable_qty": in.UnavailableQty,
				"adjusted_line_total_idr": adjustedLine,
			}),
			nilUUID(in.ActorUserID), in.Resolution)
		if err != nil {
			return err
		}
	} else {
		_, err = tx.Exec(ctx, `
			INSERT INTO order_item_events (
				store_id, order_id, order_item_id, event_type, meta, actor_user_id, reason
			) VALUES ($1,$2,$3,'allocated',$4,$5,$6)
		`, storeID, orderID, in.OrderItemID,
			mustJSON(map[string]any{
				"allocated_qty": in.AllocatedQty, "unavailable_qty": in.UnavailableQty,
			}),
			nilUUID(in.ActorUserID), in.Resolution)
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	if in.UnavailableQty > 0 {
		s.notifyUnavailableAllocation(ctx, storeID, orderID, productName, in.UnavailableQty)
	}
	return nil
}

type SubtotalSnapshot struct {
	Subtotal int64
	Shipping int64
	Handling int64
	Total    int64
}

func nilUUID(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func (s *Service) ListBatchOrders(ctx context.Context, storeID, batchID uuid.UUID, limit, offset int) ([]Order, int, error) {
	if _, err := s.GetBatch(ctx, storeID, batchID, false); err != nil {
		return nil, 0, err
	}
	f := ListOrdersFilter{Limit: limit, Offset: offset}
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
		       requires_approval, source, created_by_user_id, campaign_id
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
