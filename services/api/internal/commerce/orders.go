package commerce

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rubicahq/pikko/services/api/internal/analytics"
)

type ListOrdersFilter struct {
	Query             string
	OrderStatus       string
	PaymentStatus     string
	FulfillmentStatus string
	CustomerID        *uuid.UUID
	Limit             int
	Offset            int
}

func (s *Service) ListOrders(ctx context.Context, storeID uuid.UUID, f ListOrdersFilter) ([]Order, int, error) {
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 25
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	args := []any{storeID}
	where := []string{"store_id=$1"}
	n := 2
	if f.OrderStatus != "" {
		where = append(where, fmt.Sprintf("order_status=$%d", n))
		args = append(args, f.OrderStatus)
		n++
	}
	if f.PaymentStatus != "" {
		where = append(where, fmt.Sprintf("payment_status=$%d", n))
		args = append(args, f.PaymentStatus)
		n++
	}
	if f.FulfillmentStatus != "" {
		where = append(where, fmt.Sprintf("fulfillment_status=$%d", n))
		args = append(args, f.FulfillmentStatus)
		n++
	}
	if f.CustomerID != nil {
		where = append(where, fmt.Sprintf("customer_id=$%d", n))
		args = append(args, *f.CustomerID)
		n++
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		where = append(where, fmt.Sprintf(
			"(order_number ILIKE $%d OR customer_name ILIKE $%d OR customer_email ILIKE $%d)", n, n, n,
		))
		args = append(args, "%"+q+"%")
		n++
	}
	w := strings.Join(where, " AND ")
	var total int
	if err := s.DB.QueryRow(ctx, `SELECT COUNT(*) FROM orders WHERE `+w, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, f.Limit, f.Offset)
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
		       preorder_batch_id, batch_name_snapshot, batch_slug_snapshot, requires_approval,
		       source, created_by_user_id, campaign_id
		FROM orders WHERE `+w+`
		ORDER BY created_at DESC
		LIMIT $`+fmt.Sprint(n)+` OFFSET $`+fmt.Sprint(n+1), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []Order
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *o)
	}
	return out, total, rows.Err()
}

type scannable interface {
	Scan(dest ...any) error
}

func scanOrder(row scannable) (*Order, error) {
	var o Order
	err := row.Scan(
		&o.ID, &o.StoreID, &o.CustomerID, &o.OrderNumber, &o.SellingModel,
		&o.OrderStatus, &o.PaymentStatus, &o.FulfillmentStatus, &o.Currency,
		&o.SubtotalIDR, &o.ShippingFeeIDR, &o.HandlingFeeIDR, &o.TotalIDR,
		&o.FulfillmentMethod, &o.CustomerName, &o.CustomerEmail, &o.CustomerPhone,
		&o.ShippingRecipient, &o.ShippingPhone, &o.ShippingLine1, &o.ShippingLine2,
		&o.ShippingCity, &o.ShippingProvince, &o.ShippingPostalCode, &o.ShippingCountry,
		&o.CustomerNotes, &o.InternalNotes, &o.CancelReason,
		&o.PaymentDeadlineAt, &o.ConfirmedAt, &o.CompletedAt, &o.CancelledAt,
		&o.CreatedAt, &o.UpdatedAt,
		&o.PreorderBatchID, &o.BatchNameSnapshot, &o.BatchSlugSnapshot, &o.RequiresApproval,
		&o.Source, &o.CreatedByUserID, &o.CampaignID,
	)
	if err != nil {
		return nil, err
	}
	if o.Source == "" {
		o.Source = OrderSourceCustomer
	}
	return &o, nil
}

// scanOrderExtended is an alias used by batch order lists.
func scanOrderExtended(row scannable) (*Order, error) {
	return scanOrder(row)
}

func (s *Service) GetOrder(ctx context.Context, storeID, orderID uuid.UUID) (*Order, error) {
	row := s.DB.QueryRow(ctx, `
		SELECT id, store_id, customer_id, order_number, selling_model,
		       order_status, payment_status, fulfillment_status, currency,
		       subtotal_idr, shipping_fee_idr, handling_fee_idr, total_idr,
		       fulfillment_method, customer_name, customer_email, customer_phone,
		       shipping_recipient, shipping_phone, shipping_line1, shipping_line2,
		       shipping_city, shipping_province, shipping_postal_code, shipping_country,
		       customer_notes, internal_notes, cancel_reason,
		       payment_deadline_at, confirmed_at, completed_at, cancelled_at,
		       created_at, updated_at,
		       preorder_batch_id, batch_name_snapshot, batch_slug_snapshot, requires_approval,
		       source, created_by_user_id, campaign_id
		FROM orders WHERE id=$1 AND store_id=$2
	`, orderID, storeID)
	o, err := scanOrder(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	items, err := s.listOrderItems(ctx, storeID, orderID)
	if err != nil {
		return nil, err
	}
	o.Items = items
	return o, nil
}

func (s *Service) listOrderItems(ctx context.Context, storeID, orderID uuid.UUID) ([]OrderItem, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT id, product_id, variant_id, batch_id, product_name, product_slug, sku, variant_label,
		       unit_price_idr, quantity, COALESCE(original_quantity, quantity), line_total_idr,
		       COALESCE(availability_status, 'available'), COALESCE(allocated_qty, 0), COALESCE(packed_qty, 0),
		       adjusted_line_total_idr
		FROM order_items WHERE order_id=$1 AND store_id=$2 ORDER BY created_at
	`, orderID, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OrderItem
	for rows.Next() {
		var it OrderItem
		if err := rows.Scan(
			&it.ID, &it.ProductID, &it.VariantID, &it.BatchID, &it.ProductName, &it.ProductSlug, &it.SKU, &it.VariantLabel,
			&it.UnitPriceIDR, &it.Quantity, &it.OriginalQuantity, &it.LineTotalIDR,
			&it.AvailabilityStatus, &it.AllocatedQty, &it.PackedQty, &it.AdjustedLineTotalIDR,
		); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

type UpdateOrderStatusInput struct {
	OrderStatus       *string `json:"order_status,omitempty"`
	FulfillmentStatus *string `json:"fulfillment_status,omitempty"`
	InternalNotes     *string `json:"internal_notes,omitempty"`
	CancelReason      *string `json:"cancel_reason,omitempty"`
	// Optional relation edits: omit to leave unchanged; empty string clears; UUID sets.
	CampaignID      *string `json:"campaign_id,omitempty"`
	PreorderBatchID *string `json:"preorder_batch_id,omitempty"`
}

func (s *Service) UpdateOrder(ctx context.Context, storeID, orderID, actorID uuid.UUID, in UpdateOrderStatusInput) (*Order, error) {
	o, err := s.GetOrder(ctx, storeID, orderID)
	if err != nil {
		return nil, err
	}
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if in.OrderStatus != nil && *in.OrderStatus != o.OrderStatus {
		if !canTransitionOrder(o.OrderStatus, *in.OrderStatus) {
			return nil, ErrInvalidTransition
		}
		if *in.OrderStatus == OrderCancelled {
			if o.PaymentStatus == PaymentPaid {
				return nil, ErrInvalidTransition // paid cancel needs refund resolution
			}
			_, err = tx.Exec(ctx, `
				UPDATE orders SET order_status='cancelled', fulfillment_status='cancelled',
					cancel_reason=$3, cancelled_at=now(), updated_at=now()
				WHERE id=$1 AND store_id=$2
			`, orderID, storeID, in.CancelReason)
			if err != nil {
				return nil, err
			}
			_, _ = tx.Exec(ctx, `
				INSERT INTO order_status_history (store_id, order_id, dimension, from_status, to_status, actor_user_id, reason)
				VALUES ($1,$2,'order',$3,'cancelled',$4,$5),
				       ($1,$2,'fulfillment',$6,'cancelled',$4,$5)
			`, storeID, orderID, o.OrderStatus, actorID, in.CancelReason, o.FulfillmentStatus)
			_, _ = tx.Exec(ctx, `
				UPDATE inventory_reservations SET status='released', released_at=now()
				WHERE order_id=$1 AND store_id=$2 AND status='active'
			`, orderID, storeID)
			o.OrderStatus = OrderCancelled
			o.FulfillmentStatus = FulfillmentCancelled
		} else if *in.OrderStatus == OrderCompleted {
			if o.PaymentStatus != PaymentPaid {
				return nil, ErrInvalidTransition
			}
			if o.FulfillmentStatus != FulfillmentDelivered && o.FulfillmentStatus != FulfillmentCollected {
				return nil, ErrInvalidTransition
			}
			_, err = tx.Exec(ctx, `
				UPDATE orders SET order_status='completed', completed_at=now(), updated_at=now()
				WHERE id=$1 AND store_id=$2
			`, orderID, storeID)
			if err != nil {
				return nil, err
			}
			_, _ = tx.Exec(ctx, `
				INSERT INTO order_status_history (store_id, order_id, dimension, from_status, to_status, actor_user_id)
				VALUES ($1,$2,'order',$3,'completed',$4)
			`, storeID, orderID, o.OrderStatus, actorID)
			_, _ = tx.Exec(ctx, `
				UPDATE inventory_reservations SET status='consumed'
				WHERE order_id=$1 AND store_id=$2 AND status='active'
			`, orderID, storeID)
			// decrement finite stock
			_, _ = tx.Exec(ctx, `
				UPDATE products p SET stock_quantity = GREATEST(0, stock_quantity - r.quantity), updated_at=now()
				FROM inventory_reservations r
				WHERE r.order_id=$1 AND r.store_id=$2 AND r.status='consumed' AND r.variant_id IS NULL
				  AND p.id=r.product_id AND p.inventory_mode='finite'
			`, orderID, storeID)
			o.OrderStatus = OrderCompleted
		} else if *in.OrderStatus == OrderConfirmed {
			if o.OrderStatus != OrderPendingConfirmation {
				return nil, ErrInvalidTransition
			}
			_, err = tx.Exec(ctx, `
				UPDATE orders SET order_status='confirmed', confirmed_at=now(), updated_at=now()
				WHERE id=$1 AND store_id=$2
			`, orderID, storeID)
			if err != nil {
				return nil, err
			}
			_, _ = tx.Exec(ctx, `
				INSERT INTO order_status_history (store_id, order_id, dimension, from_status, to_status, actor_user_id)
				VALUES ($1,$2,'order',$3,'confirmed',$4)
			`, storeID, orderID, o.OrderStatus, actorID)
			o.OrderStatus = OrderConfirmed
		} else {
			return nil, ErrInvalidTransition
		}
	}

	fulfillmentChanged := false
	if in.FulfillmentStatus != nil && *in.FulfillmentStatus != o.FulfillmentStatus {
		if o.OrderStatus == OrderCancelled {
			return nil, ErrInvalidTransition
		}
		if !canTransitionFulfillment(o.FulfillmentStatus, *in.FulfillmentStatus, o.FulfillmentMethod) {
			return nil, ErrInvalidTransition
		}
		if o.PaymentStatus != PaymentPaid && (*in.FulfillmentStatus == FulfillmentShipped ||
			*in.FulfillmentStatus == FulfillmentDelivered || *in.FulfillmentStatus == FulfillmentCollected) {
			return nil, ErrInvalidTransition
		}
		_, err = tx.Exec(ctx, `
			UPDATE orders SET fulfillment_status=$3, updated_at=now() WHERE id=$1 AND store_id=$2
		`, orderID, storeID, *in.FulfillmentStatus)
		if err != nil {
			return nil, err
		}
		_, _ = tx.Exec(ctx, `
			INSERT INTO order_status_history (store_id, order_id, dimension, from_status, to_status, actor_user_id)
			VALUES ($1,$2,'fulfillment',$3,$4,$5)
		`, storeID, orderID, o.FulfillmentStatus, *in.FulfillmentStatus, actorID)
		o.FulfillmentStatus = *in.FulfillmentStatus
		fulfillmentChanged = true
	}

	if in.InternalNotes != nil {
		_, err = tx.Exec(ctx, `
			UPDATE orders SET internal_notes=$3, updated_at=now() WHERE id=$1 AND store_id=$2
		`, orderID, storeID, in.InternalNotes)
		if err != nil {
			return nil, err
		}
	}

	if in.CampaignID != nil || in.PreorderBatchID != nil {
		var inCamp, inBatch *uuid.UUID
		if in.CampaignID != nil {
			raw := strings.TrimSpace(*in.CampaignID)
			if raw != "" {
				id, perr := uuid.Parse(raw)
				if perr != nil {
					return nil, fmt.Errorf("%w: campaign_id must be a valid UUID or empty", ErrInvalidInput)
				}
				inCamp = &id
			}
		} else {
			inCamp = o.CampaignID
		}
		if in.PreorderBatchID != nil {
			raw := strings.TrimSpace(*in.PreorderBatchID)
			if raw != "" {
				id, perr := uuid.Parse(raw)
				if perr != nil {
					return nil, fmt.Errorf("%w: preorder_batch_id must be a valid UUID or empty", ErrInvalidInput)
				}
				inBatch = &id
			}
		} else {
			inBatch = o.PreorderBatchID
		}
		rels, rerr := s.resolveAdminOrderRelations(ctx, storeID, inCamp, inBatch, nil)
		if rerr != nil {
			return nil, rerr
		}
		// When caller clears one side explicitly, do not auto-fill it back from the other.
		if in.CampaignID != nil && strings.TrimSpace(*in.CampaignID) == "" {
			rels.CampaignID = nil
		}
		if in.PreorderBatchID != nil && strings.TrimSpace(*in.PreorderBatchID) == "" {
			rels.BatchID = nil
			rels.BatchName = nil
			rels.BatchSlug = nil
			rels.ETAStart = nil
			rels.ETAEnd = nil
		}
		_, err = tx.Exec(ctx, `
			UPDATE orders SET
				campaign_id=$3,
				preorder_batch_id=$4,
				batch_name_snapshot=$5,
				batch_slug_snapshot=$6,
				batch_eta_start_snapshot=$7,
				batch_eta_end_snapshot=$8,
				updated_at=now()
			WHERE id=$1 AND store_id=$2
		`, orderID, storeID, rels.CampaignID, rels.BatchID, rels.BatchName, rels.BatchSlug, rels.ETAStart, rels.ETAEnd)
		if err != nil {
			return nil, err
		}
		revNum, rerr := s.nextRevisionNumber(ctx, tx, orderID)
		if rerr == nil {
			_, _ = tx.Exec(ctx, `
				INSERT INTO order_revisions (
					store_id, order_id, revision_number, change_type, summary, actor_user_id
				) VALUES ($1,$2,$3,'notes','Updated campaign / preorder relation',$4)
			`, storeID, orderID, revNum, actorID)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	if fulfillmentChanged {
		switch o.FulfillmentStatus {
		case FulfillmentShipped, FulfillmentDelivered, FulfillmentCollected:
			analytics.Emit(ctx, s.DB, storeID, "fulfillment."+o.FulfillmentStatus, &actorID, map[string]any{
				"order_id": orderID.String(),
			})
		}
	}
	return s.GetOrder(ctx, storeID, orderID)
}

func canTransitionOrder(from, to string) bool {
	switch from {
	case OrderPendingConfirmation:
		return to == OrderConfirmed || to == OrderCancelled
	case OrderConfirmed:
		return to == OrderCompleted || to == OrderCancelled
	default:
		return false
	}
}

func canTransitionFulfillment(from, to, method string) bool {
	allowed := map[string][]string{
		FulfillmentUnfulfilled: {
			FulfillmentPurchasing, FulfillmentReadyToShip, FulfillmentReadyForPickup, FulfillmentCancelled,
		},
		FulfillmentPurchasing:      {FulfillmentAwaitingArrival, FulfillmentCancelled},
		FulfillmentAwaitingArrival: {FulfillmentReadyToShip, FulfillmentReadyForPickup, FulfillmentCancelled},
		FulfillmentReadyToShip:     {FulfillmentShipped, FulfillmentCancelled},
		FulfillmentReadyForPickup:  {FulfillmentCollected, FulfillmentCancelled},
		FulfillmentShipped:         {FulfillmentDelivered, FulfillmentCancelled},
	}
	for _, a := range allowed[from] {
		if a == to {
			if to == FulfillmentReadyForPickup && method != FulfillmentPickup {
				return false
			}
			if to == FulfillmentReadyToShip && method == FulfillmentPickup {
				return false
			}
			return true
		}
	}
	return false
}

func (s *Service) ListCustomers(ctx context.Context, storeID uuid.UUID, q string, limit, offset int) ([]Customer, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	args := []any{storeID}
	where := "store_id=$1"
	if strings.TrimSpace(q) != "" {
		where += " AND (name ILIKE $2 OR email ILIKE $2 OR COALESCE(phone,'') ILIKE $2)"
		args = append(args, "%"+strings.TrimSpace(q)+"%")
	}
	var total int
	if err := s.DB.QueryRow(ctx, `SELECT COUNT(*) FROM customers WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	limPos := len(args) - 1
	offPos := len(args)
	rows, err := s.DB.Query(ctx, fmt.Sprintf(`
		SELECT id, store_id, name, email, phone, first_order_at, last_order_at,
		       total_orders, completed_order_value_idr, internal_notes, created_at, updated_at
		FROM customers WHERE %s
		ORDER BY last_order_at DESC NULLS LAST
		LIMIT $%d OFFSET $%d
	`, where, limPos, offPos), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []Customer
	for rows.Next() {
		var c Customer
		if err := rows.Scan(
			&c.ID, &c.StoreID, &c.Name, &c.Email, &c.Phone, &c.FirstOrderAt, &c.LastOrderAt,
			&c.TotalOrders, &c.CompletedOrderValueIDR, &c.InternalNotes, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, c)
	}
	return out, total, rows.Err()
}
