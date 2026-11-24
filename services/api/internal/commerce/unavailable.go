package commerce

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

type UnavailableImpactFilter struct {
	BatchID        uuid.UUID  `json:"batch_id"`
	ProductID      uuid.UUID  `json:"product_id"`
	VariantID      *uuid.UUID `json:"variant_id,omitempty"`
	UnavailableQty int        `json:"unavailable_qty"`
}

type UnavailableImpactItem struct {
	OrderID              uuid.UUID `json:"order_id"`
	OrderNumber          string    `json:"order_number"`
	OrderItemID          uuid.UUID `json:"order_item_id"`
	CustomerName         string    `json:"customer_name"`
	OrderedQty           int       `json:"ordered_qty"`
	AlreadyAllocated     int       `json:"already_allocated"`
	ProposedUnavailable  int       `json:"proposed_unavailable"`
	ProposedAllocated    int       `json:"proposed_allocated"`
	UnitPriceIDR         int64     `json:"unit_price_idr"`
	CurrentLineTotalIDR  int64     `json:"current_line_total_idr"`
	ProposedLineTotalIDR int64     `json:"proposed_line_total_idr"`
	CurrentOrderTotalIDR int64     `json:"current_order_total_idr"`
	ProposedOrderTotalIDR int64    `json:"proposed_order_total_idr"`
}

type UnavailableImpact struct {
	ProductID          uuid.UUID              `json:"product_id"`
	VariantID          *uuid.UUID             `json:"variant_id,omitempty"`
	BatchID            uuid.UUID              `json:"batch_id"`
	UnavailableQty     int                    `json:"unavailable_qty"`
	RequiredQty        int                    `json:"required_qty"`
	PurchasedQty       int                    `json:"purchased_qty"`
	Items              []UnavailableImpactItem `json:"items"`
	TotalDeltaIDR      int64                  `json:"total_delta_idr"`
}

type MarkUnavailableInput struct {
	UnavailableQty  int     `json:"unavailable_qty"`
	ApplyAdjustment bool    `json:"apply_adjustment"`
	Reason          *string `json:"reason,omitempty"`
}

// AnalyzeUnavailableImpact proposes how shortfall would hit order items (FIFO by order created_at)
// and the resulting payable deltas without mutating data (UNAV-01 hook for Agent 07 finance).
func (s *Service) AnalyzeUnavailableImpact(ctx context.Context, storeID uuid.UUID, f UnavailableImpactFilter) (*UnavailableImpact, error) {
	if f.BatchID == uuid.Nil || f.ProductID == uuid.Nil || f.UnavailableQty < 0 {
		return nil, ErrInvalidInput
	}
	if _, err := s.GetBatch(ctx, storeID, f.BatchID, false); err != nil {
		return nil, err
	}

	out := &UnavailableImpact{
		ProductID:      f.ProductID,
		VariantID:      f.VariantID,
		BatchID:        f.BatchID,
		UnavailableQty: f.UnavailableQty,
		Items:          []UnavailableImpactItem{},
	}

	_ = s.DB.QueryRow(ctx, `
		SELECT COALESCE(SUM(oi.quantity) FILTER (
		  WHERE o.order_status IN ('confirmed','completed') AND o.payment_status='paid'
		),0)
		FROM order_items oi
		JOIN orders o ON o.id=oi.order_id AND o.store_id=oi.store_id
		WHERE oi.store_id=$1 AND o.preorder_batch_id=$2 AND oi.product_id=$3
		  AND (($4::uuid IS NULL AND oi.variant_id IS NULL) OR oi.variant_id=$4)
		  AND o.order_status NOT IN ('cancelled')
	`, storeID, f.BatchID, f.ProductID, f.VariantID).Scan(&out.RequiredQty)

	_ = s.DB.QueryRow(ctx, `
		SELECT COALESCE(SUM(purchased_qty),0) FROM preorder_purchasing_records
		WHERE store_id=$1 AND batch_id=$2 AND product_id=$3
		  AND (($4::uuid IS NULL AND variant_id IS NULL) OR variant_id=$4)
	`, storeID, f.BatchID, f.ProductID, f.VariantID).Scan(&out.PurchasedQty)

	rows, err := s.DB.Query(ctx, `
		SELECT oi.id, oi.order_id, o.order_number, o.customer_name, oi.quantity,
		       oi.unit_price_idr, oi.line_total_idr, o.total_idr,
		       COALESCE(a.allocated_qty, oi.allocated_qty, 0)
		FROM order_items oi
		JOIN orders o ON o.id=oi.order_id AND o.store_id=oi.store_id
		LEFT JOIN preorder_item_allocations a ON a.order_item_id=oi.id
		WHERE oi.store_id=$1 AND o.preorder_batch_id=$2 AND oi.product_id=$3
		  AND (($4::uuid IS NULL AND oi.variant_id IS NULL) OR oi.variant_id=$4)
		  AND o.order_status IN ('confirmed','completed','pending_confirmation')
		  AND o.payment_status IN ('paid','awaiting_verification','unpaid')
		ORDER BY
		  CASE WHEN o.payment_status='paid' THEN 0 ELSE 1 END,
		  o.created_at ASC
	`, storeID, f.BatchID, f.ProductID, f.VariantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	remaining := f.UnavailableQty
	for rows.Next() {
		var it UnavailableImpactItem
		if err := rows.Scan(
			&it.OrderItemID, &it.OrderID, &it.OrderNumber, &it.CustomerName,
			&it.OrderedQty, &it.UnitPriceIDR, &it.CurrentLineTotalIDR, &it.CurrentOrderTotalIDR,
			&it.AlreadyAllocated,
		); err != nil {
			return nil, err
		}
		proposeUnavail := 0
		if remaining > 0 {
			room := it.OrderedQty - it.AlreadyAllocated
			if room < 0 {
				room = 0
			}
			proposeUnavail = remaining
			if proposeUnavail > room {
				proposeUnavail = room
			}
			remaining -= proposeUnavail
		}
		it.ProposedUnavailable = proposeUnavail
		it.ProposedAllocated = it.OrderedQty - proposeUnavail
		if it.ProposedAllocated < 0 {
			it.ProposedAllocated = 0
		}
		availQty := it.OrderedQty - proposeUnavail
		it.ProposedLineTotalIDR = it.UnitPriceIDR * int64(availQty)
		delta := it.CurrentLineTotalIDR - it.ProposedLineTotalIDR
		it.ProposedOrderTotalIDR = it.CurrentOrderTotalIDR - delta
		out.TotalDeltaIDR += delta
		out.Items = append(out.Items, it)
	}
	return out, rows.Err()
}

// MarkOrderItemUnavailable adjusts availability and optionally payable while preserving
// original_quantity / snapshots via append-only revisions (UNAV-01).
func (s *Service) MarkOrderItemUnavailable(
	ctx context.Context, storeID, orderID, itemID, actorID uuid.UUID, in MarkUnavailableInput,
) (*Order, error) {
	if in.UnavailableQty < 0 {
		return nil, ErrInvalidInput
	}
	o, err := s.GetOrder(ctx, storeID, orderID)
	if err != nil {
		return nil, err
	}
	if o.OrderStatus == OrderCancelled || o.OrderStatus == OrderCompleted {
		return nil, ErrInvalidTransition
	}
	var item OrderItem
	found := false
	for _, it := range o.Items {
		if it.ID == itemID {
			item = it
			found = true
			break
		}
	}
	if !found {
		return nil, ErrNotFound
	}
	if in.UnavailableQty > item.Quantity {
		return nil, ErrInvalidInput
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	newAvail := "available"
	if in.UnavailableQty >= item.Quantity {
		newAvail = "unavailable"
	} else if in.UnavailableQty > 0 {
		newAvail = "partially_unavailable"
	}
	availQty := item.Quantity - in.UnavailableQty
	adjustedLine := item.UnitPriceIDR * int64(availQty)

	_, err = tx.Exec(ctx, `
		UPDATE order_items SET
			availability_status=$4,
			adjusted_line_total_idr=CASE WHEN $5 THEN $6 ELSE adjusted_line_total_idr END
		WHERE id=$1 AND order_id=$2 AND store_id=$3
	`, itemID, orderID, storeID, newAvail, in.ApplyAdjustment, adjustedLine)
	if err != nil {
		return nil, err
	}

	// Mirror into allocation row when batch-linked and product is known.
	if o.PreorderBatchID != nil && item.ProductID != nil {
		allocQty := availQty
		_, err = tx.Exec(ctx, `
			INSERT INTO preorder_item_allocations (
				store_id, batch_id, order_id, order_item_id, product_id, variant_id,
				allocated_qty, unavailable_qty, resolution
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
			ON CONFLICT (order_item_id) DO UPDATE SET
				allocated_qty=EXCLUDED.allocated_qty,
				unavailable_qty=EXCLUDED.unavailable_qty,
				resolution=COALESCE(EXCLUDED.resolution, preorder_item_allocations.resolution),
				updated_at=now()
		`, storeID, *o.PreorderBatchID, orderID, itemID, *item.ProductID, item.VariantID,
			allocQty, in.UnavailableQty, in.Reason)
		if err != nil {
			return nil, err
		}
		_, _ = tx.Exec(ctx, `
			UPDATE order_items SET allocated_qty=$4
			WHERE id=$1 AND order_id=$2 AND store_id=$3
		`, itemID, orderID, storeID, allocQty)
	}

	var newSubtotal, newTotal int64
	newSubtotal = o.SubtotalIDR
	newTotal = o.TotalIDR
	var financeDeltaIDR int64
	if in.ApplyAdjustment {
		// Preserve immutable original before shrinking totals (Agent 07 ledger).
		_, err = tx.Exec(ctx, `
			UPDATE orders SET original_total_idr = COALESCE(original_total_idr, total_idr)
			WHERE id=$1 AND store_id=$2
		`, orderID, storeID)
		if err != nil {
			return nil, err
		}
		err = tx.QueryRow(ctx, `
			SELECT COALESCE(SUM(COALESCE(adjusted_line_total_idr, line_total_idr)),0)
			FROM order_items WHERE order_id=$1 AND store_id=$2
		`, orderID, storeID).Scan(&newSubtotal)
		if err != nil {
			return nil, err
		}
		newTotal = newSubtotal + o.ShippingFeeIDR + o.HandlingFeeIDR
		financeDeltaIDR = o.TotalIDR - newTotal
		if financeDeltaIDR < 0 {
			financeDeltaIDR = 0
		}
		_, err = tx.Exec(ctx, `
			UPDATE orders SET subtotal_idr=$3, total_idr=$4, updated_at=now()
			WHERE id=$1 AND store_id=$2
		`, orderID, storeID, newSubtotal, newTotal)
		if err != nil {
			return nil, err
		}
	}

	revNum, err := s.nextRevisionNumber(ctx, tx, orderID)
	if err != nil {
		return nil, err
	}
	before, _ := json.Marshal(map[string]any{
		"item_id": itemID.String(), "availability": item.AvailabilityStatus,
		"quantity": item.Quantity, "original_quantity": item.OriginalQuantity,
		"line_total_idr": item.LineTotalIDR, "subtotal_idr": o.SubtotalIDR, "total_idr": o.TotalIDR,
	})
	after, _ := json.Marshal(map[string]any{
		"item_id": itemID.String(), "availability": newAvail,
		"quantity": item.Quantity, "original_quantity": item.OriginalQuantity,
		"unavailable_qty": in.UnavailableQty,
		"adjusted_line_total_idr": adjustedLine,
		"subtotal_idr": newSubtotal, "total_idr": newTotal,
	})
	summary := fmt.Sprintf("Marked %s unavailable × %d (history preserved)", item.ProductName, in.UnavailableQty)
	var revID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO order_revisions (
			store_id, order_id, revision_number, change_type, summary,
			before_snapshot, after_snapshot, actor_user_id, reason
		) VALUES ($1,$2,$3,'item_unavailable',$4,$5,$6,$7,$8)
		RETURNING id
	`, storeID, orderID, revNum, summary, before, after, actorID, in.Reason).Scan(&revID)
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO order_item_events (
			store_id, order_id, order_item_id, revision_id, event_type,
			from_availability, to_availability, meta, actor_user_id, reason
		) VALUES ($1,$2,$3,$4,'availability_changed',$5,$6,$7,$8,$9)
	`, storeID, orderID, itemID, revID, item.AvailabilityStatus, newAvail,
		mustJSON(map[string]any{
			"unavailable_qty": in.UnavailableQty,
			"apply_adjustment": in.ApplyAdjustment,
			"adjusted_line_total_idr": adjustedLine,
		}),
		actorID, in.Reason)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	if in.ApplyAdjustment && financeDeltaIDR > 0 {
		idem := fmt.Sprintf("unavail:%s:%s:%d", orderID.String(), itemID.String(), in.UnavailableQty)
		_, _ = s.ApplyUnavailableFinanceAdjustment(ctx, storeID, orderID, itemID, actorID, financeDeltaIDR, &revID, in.Reason, idem)
	}
	if in.UnavailableQty > 0 {
		s.notifyUnavailableAllocation(ctx, storeID, orderID, item.ProductName, in.UnavailableQty)
	}
	return s.GetOrder(ctx, storeID, orderID)
}
