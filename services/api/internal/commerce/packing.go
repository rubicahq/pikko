package commerce

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	PackingPending    = "pending"
	PackingInProgress = "in_progress"
	PackingPacked     = "packed"
	PackingCancelled  = "cancelled"
)

type PackingQueueItem struct {
	OrderID           uuid.UUID  `json:"order_id"`
	OrderNumber       string     `json:"order_number"`
	CustomerID        *uuid.UUID `json:"customer_id,omitempty"`
	CustomerName      string     `json:"customer_name"`
	CustomerEmail     string     `json:"customer_email"`
	FulfillmentMethod string     `json:"fulfillment_method"`
	FulfillmentStatus string     `json:"fulfillment_status"`
	PaymentStatus     string     `json:"payment_status"`
	SellingModel      string     `json:"selling_model"`
	ItemCount         int        `json:"item_count"`
	PackableQty       int        `json:"packable_qty"`
	PackedQty         int        `json:"packed_qty"`
	RemainingQty      int        `json:"remaining_qty"`
	PackingStatus     *string    `json:"packing_status,omitempty"`
	PackingTaskID     *uuid.UUID `json:"packing_task_id,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type PackingCustomerGroup struct {
	CustomerKey   string             `json:"customer_key"`
	CustomerID    *uuid.UUID         `json:"customer_id,omitempty"`
	CustomerName  string             `json:"customer_name"`
	CustomerEmail string             `json:"customer_email"`
	OrderCount    int                `json:"order_count"`
	PackableQty   int                `json:"packable_qty"`
	RemainingQty  int                `json:"remaining_qty"`
	Orders        []PackingQueueItem `json:"orders"`
}

type PackingQueueResponse struct {
	Orders    []PackingQueueItem     `json:"orders,omitempty"`
	Customers []PackingCustomerGroup `json:"customers,omitempty"`
	GroupBy   string                 `json:"group_by"`
}

type PackingChecklistItem struct {
	OrderItemID        uuid.UUID  `json:"order_item_id"`
	ProductName        string     `json:"product_name"`
	SKU                *string    `json:"sku,omitempty"`
	VariantLabel       *string    `json:"variant_label,omitempty"`
	Quantity           int        `json:"quantity"`
	AvailabilityStatus string     `json:"availability_status"`
	AllocatedQty       int        `json:"allocated_qty"`
	AvailableQty       int        `json:"available_qty"`
	PackedQty          int        `json:"packed_qty"`
	ShippedQty         int        `json:"shipped_qty"`
	RemainingToPack    int        `json:"remaining_to_pack"`
	Packable           bool       `json:"packable"`
}

type PackingChecklist struct {
	OrderID           uuid.UUID              `json:"order_id"`
	OrderNumber       string                 `json:"order_number"`
	CustomerName      string                 `json:"customer_name"`
	CustomerEmail     string                 `json:"customer_email"`
	FulfillmentMethod string                 `json:"fulfillment_method"`
	FulfillmentStatus string                 `json:"fulfillment_status"`
	PaymentStatus     string                 `json:"payment_status"`
	PackingEligible   bool                   `json:"packing_eligible"`
	PackingTaskID     *uuid.UUID             `json:"packing_task_id,omitempty"`
	PackingStatus     *string                `json:"packing_status,omitempty"`
	Items             []PackingChecklistItem `json:"items"`
	AllPacked         bool                   `json:"all_packed"`
}

type UpdatePackingChecklistInput struct {
	Items []UpdatePackingItem `json:"items"`
	Notes *string             `json:"notes,omitempty"`
}

type UpdatePackingItem struct {
	OrderItemID uuid.UUID `json:"order_item_id"`
	QtyPacked   int       `json:"qty_packed"`
}

type CompletePackingInput struct {
	Notes *string `json:"notes,omitempty"`
}

// availableToPack returns qty eligible for packing (gates on allocation / availability).
func availableToPack(it OrderItem) int {
	if it.AvailabilityStatus == "unavailable" {
		return 0
	}
	if it.BatchID != nil {
		// Preorder: only allocated units may be packed (RCV-03 / PACK-02 gate).
		return it.AllocatedQty
	}
	if it.AllocatedQty > 0 {
		return it.AllocatedQty
	}
	if it.AvailabilityStatus == "partially_unavailable" {
		// Without allocation snapshot, fall back to remaining positive quantity.
		if it.Quantity < 0 {
			return 0
		}
		return it.Quantity
	}
	return it.Quantity
}

func (s *Service) ListPackingQueue(ctx context.Context, storeID uuid.UUID, groupBy, statusFilter string) (*PackingQueueResponse, error) {
	groupBy = strings.TrimSpace(strings.ToLower(groupBy))
	if groupBy == "" {
		groupBy = "order"
	}
	if groupBy != "order" && groupBy != "customer" {
		return nil, ErrInvalidInput
	}
	statusFilter = strings.TrimSpace(strings.ToLower(statusFilter))

	rows, err := s.DB.Query(ctx, `
		SELECT o.id, o.order_number, o.customer_id, o.customer_name, o.customer_email,
		       o.fulfillment_method, o.fulfillment_status, o.payment_status, o.selling_model,
		       o.created_at, o.updated_at,
		       COALESCE((
		         SELECT COUNT(*) FROM order_items oi WHERE oi.order_id=o.id AND oi.store_id=o.store_id
		       ),0),
		       COALESCE((
		         SELECT SUM(
		           CASE
		             WHEN oi.availability_status='unavailable' THEN 0
		             WHEN oi.batch_id IS NOT NULL THEN oi.allocated_qty
		             WHEN oi.allocated_qty > 0 THEN oi.allocated_qty
		             ELSE oi.quantity
		           END
		         ) FROM order_items oi WHERE oi.order_id=o.id AND oi.store_id=o.store_id
		       ),0),
		       COALESCE((
		         SELECT SUM(oi.packed_qty) FROM order_items oi WHERE oi.order_id=o.id AND oi.store_id=o.store_id
		       ),0),
		       pt.id, pt.status
		FROM orders o
		LEFT JOIN packing_tasks pt ON pt.order_id=o.id AND pt.store_id=o.store_id AND pt.status <> 'cancelled'
		WHERE o.store_id=$1
		  AND o.order_status <> 'cancelled'
		  AND o.fulfillment_status IN (
		    'unfulfilled','awaiting_arrival','ready_to_ship','ready_for_pickup','purchasing'
		  )
		ORDER BY o.customer_email ASC, o.created_at ASC
		LIMIT 200
	`, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var all []PackingQueueItem
	for rows.Next() {
		var q PackingQueueItem
		var packable, packed int
		if err := rows.Scan(
			&q.OrderID, &q.OrderNumber, &q.CustomerID, &q.CustomerName, &q.CustomerEmail,
			&q.FulfillmentMethod, &q.FulfillmentStatus, &q.PaymentStatus, &q.SellingModel,
			&q.CreatedAt, &q.UpdatedAt,
			&q.ItemCount, &packable, &packed,
			&q.PackingTaskID, &q.PackingStatus,
		); err != nil {
			return nil, err
		}
		q.PackableQty = packable
		q.PackedQty = packed
		q.RemainingQty = packable - packed
		if q.RemainingQty < 0 {
			q.RemainingQty = 0
		}
		// Eligible when something packable remains, or task in progress / packed for visibility.
		eligible := q.PackableQty > 0 && (q.FulfillmentStatus == FulfillmentAwaitingArrival ||
			q.FulfillmentStatus == FulfillmentUnfulfilled ||
			q.FulfillmentStatus == FulfillmentReadyToShip ||
			q.FulfillmentStatus == FulfillmentReadyForPickup ||
			(q.FulfillmentStatus == FulfillmentPurchasing && q.PackableQty > 0))
		if !eligible && (q.PackingStatus == nil || *q.PackingStatus == PackingPending) {
			continue
		}
		switch statusFilter {
		case "eligible":
			if q.RemainingQty <= 0 {
				continue
			}
		case "in_progress":
			if q.PackingStatus == nil || *q.PackingStatus != PackingInProgress {
				continue
			}
		case "packed":
			if q.PackingStatus == nil || *q.PackingStatus != PackingPacked {
				continue
			}
		}
		all = append(all, q)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := &PackingQueueResponse{GroupBy: groupBy}
	if groupBy == "customer" {
		byKey := map[string]*PackingCustomerGroup{}
		var orderKeys []string
		for _, o := range all {
			key := strings.ToLower(strings.TrimSpace(o.CustomerEmail))
			if key == "" {
				key = o.OrderID.String()
			}
			g, ok := byKey[key]
			if !ok {
				g = &PackingCustomerGroup{
					CustomerKey: key, CustomerID: o.CustomerID,
					CustomerName: o.CustomerName, CustomerEmail: o.CustomerEmail,
				}
				byKey[key] = g
				orderKeys = append(orderKeys, key)
			}
			g.Orders = append(g.Orders, o)
			g.OrderCount++
			g.PackableQty += o.PackableQty
			g.RemainingQty += o.RemainingQty
		}
		for _, k := range orderKeys {
			out.Customers = append(out.Customers, *byKey[k])
		}
	} else {
		out.Orders = all
	}
	return out, nil
}

func (s *Service) GetPackingChecklist(ctx context.Context, storeID, orderID uuid.UUID) (*PackingChecklist, error) {
	o, err := s.GetOrder(ctx, storeID, orderID)
	if err != nil {
		return nil, err
	}
	shipped, err := s.shippedQtyByItem(ctx, storeID, orderID)
	if err != nil {
		return nil, err
	}
	var taskID *uuid.UUID
	var taskStatus *string
	err = s.DB.QueryRow(ctx, `
		SELECT id, status FROM packing_tasks
		WHERE store_id=$1 AND order_id=$2 AND status <> 'cancelled'
		ORDER BY created_at DESC LIMIT 1
	`, storeID, orderID).Scan(&taskID, &taskStatus)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	items := make([]PackingChecklistItem, 0, len(o.Items))
	allPacked := true
	anyPackable := false
	for _, it := range o.Items {
		avail := availableToPack(it)
		shipQty := shipped[it.ID]
		remaining := avail - it.PackedQty
		if remaining < 0 {
			remaining = 0
		}
		packable := avail > 0
		if packable {
			anyPackable = true
			if remaining > 0 {
				allPacked = false
			}
		}
		items = append(items, PackingChecklistItem{
			OrderItemID: it.ID, ProductName: it.ProductName, SKU: it.SKU, VariantLabel: it.VariantLabel,
			Quantity: it.Quantity, AvailabilityStatus: it.AvailabilityStatus,
			AllocatedQty: it.AllocatedQty, AvailableQty: avail, PackedQty: it.PackedQty,
			ShippedQty: shipQty, RemainingToPack: remaining, Packable: packable,
		})
	}
	if !anyPackable {
		allPacked = false
	}

	eligible := anyPackable && o.OrderStatus != OrderCancelled &&
		(o.FulfillmentStatus == FulfillmentAwaitingArrival ||
			o.FulfillmentStatus == FulfillmentUnfulfilled ||
			o.FulfillmentStatus == FulfillmentReadyToShip ||
			o.FulfillmentStatus == FulfillmentReadyForPickup ||
			o.FulfillmentStatus == FulfillmentPurchasing)

	return &PackingChecklist{
		OrderID: o.ID, OrderNumber: o.OrderNumber,
		CustomerName: o.CustomerName, CustomerEmail: o.CustomerEmail,
		FulfillmentMethod: o.FulfillmentMethod, FulfillmentStatus: o.FulfillmentStatus,
		PaymentStatus: o.PaymentStatus, PackingEligible: eligible,
		PackingTaskID: taskID, PackingStatus: taskStatus,
		Items: items, AllPacked: allPacked && anyPackable,
	}, nil
}

func (s *Service) UpdatePackingChecklist(ctx context.Context, storeID, orderID, actorID uuid.UUID, in UpdatePackingChecklistInput) (*PackingChecklist, error) {
	if len(in.Items) == 0 {
		return nil, ErrInvalidInput
	}
	o, err := s.GetOrder(ctx, storeID, orderID)
	if err != nil {
		return nil, err
	}
	if o.OrderStatus == OrderCancelled {
		return nil, ErrInvalidTransition
	}

	itemByID := map[uuid.UUID]OrderItem{}
	for _, it := range o.Items {
		itemByID[it.ID] = it
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	taskID, err := s.ensurePackingTaskTx(ctx, tx, storeID, orderID, actorID, in.Notes)
	if err != nil {
		return nil, err
	}

	anyPacked := false
	allDone := true
	for _, upd := range in.Items {
		it, ok := itemByID[upd.OrderItemID]
		if !ok {
			return nil, ErrInvalidInput
		}
		avail := availableToPack(it)
		if upd.QtyPacked < 0 || upd.QtyPacked > avail {
			return nil, ErrInvalidInput
		}
		if upd.QtyPacked > 0 {
			anyPacked = true
		}
		if avail > 0 && upd.QtyPacked < avail {
			allDone = false
		}
		_, err = tx.Exec(ctx, `
			UPDATE order_items SET packed_qty=$3 WHERE id=$1 AND store_id=$2 AND order_id=$4
		`, upd.OrderItemID, storeID, upd.QtyPacked, orderID)
		if err != nil {
			return nil, err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO packing_task_items (store_id, packing_task_id, order_item_id, qty_to_pack, qty_packed)
			VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT (packing_task_id, order_item_id) DO UPDATE
			  SET qty_to_pack=EXCLUDED.qty_to_pack, qty_packed=EXCLUDED.qty_packed, updated_at=now()
		`, storeID, taskID, upd.OrderItemID, avail, upd.QtyPacked)
		if err != nil {
			return nil, err
		}
	}

	status := PackingInProgress
	if allDone && anyPacked {
		status = PackingInProgress // complete endpoint flips to packed + fulfillment
	} else if !anyPacked {
		status = PackingPending
	}
	_, err = tx.Exec(ctx, `
		UPDATE packing_tasks SET status=$3, notes=COALESCE($4, notes), updated_at=now()
		WHERE id=$1 AND store_id=$2
	`, taskID, storeID, status, in.Notes)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetPackingChecklist(ctx, storeID, orderID)
}

func (s *Service) CompletePacking(ctx context.Context, storeID, orderID, actorID uuid.UUID, in CompletePackingInput) (*PackingChecklist, error) {
	o, err := s.GetOrder(ctx, storeID, orderID)
	if err != nil {
		return nil, err
	}
	if o.OrderStatus == OrderCancelled {
		return nil, ErrInvalidTransition
	}

	shipped, err := s.shippedQtyByItem(ctx, storeID, orderID)
	if err != nil {
		return nil, err
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	taskID, err := s.ensurePackingTaskTx(ctx, tx, storeID, orderID, actorID, in.Notes)
	if err != nil {
		return nil, err
	}

	anyPackable := false
	for _, it := range o.Items {
		avail := availableToPack(it)
		if avail <= 0 {
			continue
		}
		anyPackable = true
		packed := it.PackedQty
		if packed < avail {
			// Auto-fill remaining checklist on complete if merchant marks packed without saving each line.
			packed = avail
			_, err = tx.Exec(ctx, `
				UPDATE order_items SET packed_qty=$3 WHERE id=$1 AND store_id=$2 AND order_id=$4
			`, it.ID, storeID, packed, orderID)
			if err != nil {
				return nil, err
			}
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO packing_task_items (store_id, packing_task_id, order_item_id, qty_to_pack, qty_packed)
			VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT (packing_task_id, order_item_id) DO UPDATE
			  SET qty_to_pack=EXCLUDED.qty_to_pack, qty_packed=EXCLUDED.qty_packed, updated_at=now()
		`, storeID, taskID, it.ID, avail, packed)
		if err != nil {
			return nil, err
		}
		_ = shipped // reserved for future gate: cannot complete below already-shipped
	}
	if !anyPackable {
		return nil, ErrConflict
	}

	nextFulfillment := FulfillmentReadyToShip
	if o.FulfillmentMethod == FulfillmentPickup {
		nextFulfillment = FulfillmentReadyForPickup
	}
	fromStatus := o.FulfillmentStatus
	if fromStatus != nextFulfillment {
		if fromStatus == FulfillmentReadyToShip || fromStatus == FulfillmentReadyForPickup {
			// already ready — keep
			nextFulfillment = fromStatus
		} else if !canTransitionFulfillment(fromStatus, nextFulfillment, o.FulfillmentMethod) {
			// Allow awaiting_arrival / unfulfilled / purchasing → ready_*
			if fromStatus == FulfillmentPurchasing {
				// purchasing → awaiting_arrival then ready is two hops; permit direct ready for packing complete.
				_, err = tx.Exec(ctx, `
					UPDATE orders SET fulfillment_status=$3, updated_at=now() WHERE id=$1 AND store_id=$2
				`, orderID, storeID, nextFulfillment)
				if err != nil {
					return nil, err
				}
				_, _ = tx.Exec(ctx, `
					INSERT INTO order_status_history (store_id, order_id, dimension, from_status, to_status, actor_user_id, reason)
					VALUES ($1,$2,'fulfillment',$3,$4,$5,'packing_complete')
				`, storeID, orderID, fromStatus, nextFulfillment, actorID)
			} else {
				return nil, ErrInvalidTransition
			}
		} else {
			_, err = tx.Exec(ctx, `
				UPDATE orders SET fulfillment_status=$3, updated_at=now() WHERE id=$1 AND store_id=$2
			`, orderID, storeID, nextFulfillment)
			if err != nil {
				return nil, err
			}
			_, _ = tx.Exec(ctx, `
				INSERT INTO order_status_history (store_id, order_id, dimension, from_status, to_status, actor_user_id, reason)
				VALUES ($1,$2,'fulfillment',$3,$4,$5,'packing_complete')
			`, storeID, orderID, fromStatus, nextFulfillment, actorID)
		}
	}

	_, err = tx.Exec(ctx, `
		UPDATE packing_tasks
		SET status=$3, packed_at=now(), notes=COALESCE($4, notes), updated_at=now()
		WHERE id=$1 AND store_id=$2
	`, taskID, storeID, PackingPacked, in.Notes)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	if s.Outbox != nil {
		sid := storeID
		_ = s.Outbox.Write(ctx, &sid, "packing.completed", map[string]any{
			"order_id": orderID.String(), "packing_task_id": taskID.String(),
		}, fmt.Sprintf("packing.completed:%s", orderID.String()))
	}

	oid := orderID
	s.notifyInApp(ctx, storeID, "packing_completed",
		fmt.Sprintf("Order packed: %s", o.OrderNumber),
		fmt.Sprintf("Ready for %s", strings.ReplaceAll(nextFulfillment, "_", " ")),
		"order", &oid,
	)

	return s.GetPackingChecklist(ctx, storeID, orderID)
}

func (s *Service) ensurePackingTaskTx(ctx context.Context, tx pgx.Tx, storeID, orderID, actorID uuid.UUID, notes *string) (uuid.UUID, error) {
	var id uuid.UUID
	err := tx.QueryRow(ctx, `
		SELECT id FROM packing_tasks
		WHERE store_id=$1 AND order_id=$2 AND status <> 'cancelled'
		ORDER BY created_at DESC LIMIT 1
	`, storeID, orderID).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, err
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO packing_tasks (store_id, order_id, status, assigned_user_id, notes, created_by)
		VALUES ($1,$2,'pending',$3,$4,$3)
		RETURNING id
	`, storeID, orderID, actorID, notes).Scan(&id)
	return id, err
}

func (s *Service) shippedQtyByItem(ctx context.Context, storeID, orderID uuid.UUID) (map[uuid.UUID]int, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT si.order_item_id, COALESCE(SUM(si.quantity),0)
		FROM shipment_items si
		JOIN shipments sh ON sh.id=si.shipment_id AND sh.store_id=si.store_id
		WHERE si.store_id=$1 AND sh.order_id=$2 AND sh.status <> 'cancelled'
		GROUP BY si.order_item_id
	`, storeID, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[uuid.UUID]int{}
	for rows.Next() {
		var id uuid.UUID
		var qty int
		if err := rows.Scan(&id, &qty); err != nil {
			return nil, err
		}
		out[id] = qty
	}
	return out, rows.Err()
}
