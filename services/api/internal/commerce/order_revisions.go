package commerce

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type AdjustItemQtyInput struct {
	Quantity int     `json:"quantity"`
	Reason   *string `json:"reason,omitempty"`
}

func (s *Service) ListOrderRevisions(ctx context.Context, storeID, orderID uuid.UUID) ([]OrderRevision, error) {
	if _, err := s.GetOrder(ctx, storeID, orderID); err != nil {
		return nil, err
	}
	rows, err := s.DB.Query(ctx, `
		SELECT id, store_id, order_id, revision_number, change_type, summary,
		       before_snapshot, after_snapshot, actor_user_id, reason, created_at
		FROM order_revisions
		WHERE store_id=$1 AND order_id=$2
		ORDER BY revision_number DESC
	`, storeID, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OrderRevision
	for rows.Next() {
		var r OrderRevision
		if err := rows.Scan(
			&r.ID, &r.StoreID, &r.OrderID, &r.RevisionNumber, &r.ChangeType, &r.Summary,
			&r.BeforeSnapshot, &r.AfterSnapshot, &r.ActorUserID, &r.Reason, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Service) ListOrderItemEvents(ctx context.Context, storeID, orderID uuid.UUID) ([]OrderItemEvent, error) {
	if _, err := s.GetOrder(ctx, storeID, orderID); err != nil {
		return nil, err
	}
	rows, err := s.DB.Query(ctx, `
		SELECT id, store_id, order_id, order_item_id, revision_id, event_type,
		       from_quantity, to_quantity, from_availability, to_availability,
		       meta, actor_user_id, reason, created_at
		FROM order_item_events
		WHERE store_id=$1 AND order_id=$2
		ORDER BY created_at ASC
	`, storeID, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OrderItemEvent
	for rows.Next() {
		var e OrderItemEvent
		if err := rows.Scan(
			&e.ID, &e.StoreID, &e.OrderID, &e.OrderItemID, &e.RevisionID, &e.EventType,
			&e.FromQuantity, &e.ToQuantity, &e.FromAvailability, &e.ToAvailability,
			&e.Meta, &e.ActorUserID, &e.Reason, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// AdjustOrderItemQty non-destructively adjusts quantity and appends revision + item event.
func (s *Service) AdjustOrderItemQty(ctx context.Context, storeID, orderID, itemID, actorID uuid.UUID, in AdjustItemQtyInput) (*Order, error) {
	if in.Quantity <= 0 {
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
	if item.Quantity == in.Quantity {
		return o, nil
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	newLine := item.UnitPriceIDR * int64(in.Quantity)
	_, err = tx.Exec(ctx, `
		UPDATE order_items
		SET quantity=$4, line_total_idr=$5, adjusted_line_total_idr=$5
		WHERE id=$1 AND order_id=$2 AND store_id=$3
	`, itemID, orderID, storeID, in.Quantity, newLine)
	if err != nil {
		return nil, err
	}

	// Recompute order totals from items
	var subtotal int64
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(line_total_idr),0) FROM order_items WHERE order_id=$1 AND store_id=$2
	`, orderID, storeID).Scan(&subtotal)
	if err != nil {
		return nil, err
	}
	total := subtotal + o.ShippingFeeIDR + o.HandlingFeeIDR
	_, err = tx.Exec(ctx, `
		UPDATE orders SET subtotal_idr=$3, total_idr=$4, updated_at=now()
		WHERE id=$1 AND store_id=$2
	`, orderID, storeID, subtotal, total)
	if err != nil {
		return nil, err
	}

	// Keep active reservation qty in sync when present
	_, _ = tx.Exec(ctx, `
		UPDATE inventory_reservations SET quantity=$4
		WHERE order_item_id=$1 AND order_id=$2 AND store_id=$3 AND status='active'
	`, itemID, orderID, storeID, in.Quantity)

	revNum, err := s.nextRevisionNumber(ctx, tx, orderID)
	if err != nil {
		return nil, err
	}
	before, _ := json.Marshal(map[string]any{
		"item_id": itemID.String(), "quantity": item.Quantity, "line_total_idr": item.LineTotalIDR,
		"subtotal_idr": o.SubtotalIDR, "total_idr": o.TotalIDR,
	})
	after, _ := json.Marshal(map[string]any{
		"item_id": itemID.String(), "quantity": in.Quantity, "line_total_idr": newLine,
		"subtotal_idr": subtotal, "total_idr": total,
	})
	summary := fmt.Sprintf("Adjusted qty for %s: %d → %d", item.ProductName, item.Quantity, in.Quantity)
	var revID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO order_revisions (
			store_id, order_id, revision_number, change_type, summary,
			before_snapshot, after_snapshot, actor_user_id, reason
		) VALUES ($1,$2,$3,'item_qty',$4,$5,$6,$7,$8)
		RETURNING id
	`, storeID, orderID, revNum, summary, before, after, actorID, in.Reason).Scan(&revID)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO order_item_events (
			store_id, order_id, order_item_id, revision_id, event_type,
			from_quantity, to_quantity, actor_user_id, reason
		) VALUES ($1,$2,$3,$4,'qty_adjusted',$5,$6,$7,$8)
	`, storeID, orderID, itemID, revID, item.Quantity, in.Quantity, actorID, in.Reason)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetOrder(ctx, storeID, orderID)
}

func (s *Service) nextRevisionNumber(ctx context.Context, tx pgx.Tx, orderID uuid.UUID) (int, error) {
	var n int
	err := tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(revision_number),0)+1 FROM order_revisions WHERE order_id=$1
	`, orderID).Scan(&n)
	return n, err
}
