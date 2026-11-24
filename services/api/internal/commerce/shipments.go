package commerce

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rubicahq/pikko/services/api/internal/email"
	"github.com/rubicahq/pikko/services/api/internal/money"
)

const (
	ShipmentLabelCreated = "label_created"
	ShipmentShipped      = "shipped"
	ShipmentInTransit    = "in_transit"
	ShipmentDelivered    = "delivered"
	ShipmentCancelled    = "cancelled"
)

type ShipmentItem struct {
	ID            uuid.UUID `json:"id"`
	OrderItemID   uuid.UUID `json:"order_item_id"`
	ProductName   string    `json:"product_name,omitempty"`
	SKU           *string   `json:"sku,omitempty"`
	VariantLabel  *string   `json:"variant_label,omitempty"`
	Quantity      int       `json:"quantity"`
}

type Shipment struct {
	ID              uuid.UUID      `json:"id"`
	StoreID         uuid.UUID      `json:"store_id"`
	OrderID         uuid.UUID      `json:"order_id"`
	OrderNumber     string         `json:"order_number,omitempty"`
	CustomerName    string         `json:"customer_name,omitempty"`
	CustomerEmail   string         `json:"customer_email,omitempty"`
	Carrier         string         `json:"carrier"`
	TrackingNumber  string         `json:"tracking_number"`
	Status          string         `json:"status"`
	Notes           *string        `json:"notes,omitempty"`
	ShippedAt       *time.Time     `json:"shipped_at,omitempty"`
	CreatedBy       *uuid.UUID     `json:"created_by,omitempty"`
	NotifyCustomer  bool           `json:"notify_customer"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	Items           []ShipmentItem `json:"items,omitempty"`
}

type CreateShipmentItemInput struct {
	OrderItemID uuid.UUID `json:"order_item_id"`
	Quantity    int       `json:"quantity"`
}

type CreateShipmentInput struct {
	OrderID         uuid.UUID                 `json:"order_id"`
	Carrier         string                    `json:"carrier"`
	TrackingNumber  string                    `json:"tracking_number"`
	Notes           *string                   `json:"notes,omitempty"`
	NotifyCustomer  *bool                     `json:"notify_customer,omitempty"`
	Items           []CreateShipmentItemInput `json:"items"`
}

func (s *Service) ListShipments(ctx context.Context, storeID uuid.UUID, orderID *uuid.UUID, limit, offset int) ([]Shipment, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	args := []any{storeID}
	where := "sh.store_id=$1 AND sh.status <> 'cancelled'"
	if orderID != nil {
		where += " AND sh.order_id=$2"
		args = append(args, *orderID)
	}
	var total int
	if err := s.DB.QueryRow(ctx, `SELECT COUNT(*) FROM shipments sh WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	limPos := len(args) - 1
	offPos := len(args)
	rows, err := s.DB.Query(ctx, fmt.Sprintf(`
		SELECT sh.id, sh.store_id, sh.order_id, o.order_number, o.customer_name, o.customer_email,
		       sh.carrier, sh.tracking_number, sh.status, sh.notes, sh.shipped_at,
		       sh.created_by, sh.notify_customer, sh.created_at, sh.updated_at
		FROM shipments sh
		JOIN orders o ON o.id=sh.order_id AND o.store_id=sh.store_id
		WHERE %s
		ORDER BY sh.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, limPos, offPos), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []Shipment
	for rows.Next() {
		var sh Shipment
		if err := rows.Scan(
			&sh.ID, &sh.StoreID, &sh.OrderID, &sh.OrderNumber, &sh.CustomerName, &sh.CustomerEmail,
			&sh.Carrier, &sh.TrackingNumber, &sh.Status, &sh.Notes, &sh.ShippedAt,
			&sh.CreatedBy, &sh.NotifyCustomer, &sh.CreatedAt, &sh.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, sh)
	}
	return out, total, rows.Err()
}

func (s *Service) GetShipment(ctx context.Context, storeID, shipmentID uuid.UUID) (*Shipment, error) {
	var sh Shipment
	err := s.DB.QueryRow(ctx, `
		SELECT sh.id, sh.store_id, sh.order_id, o.order_number, o.customer_name, o.customer_email,
		       sh.carrier, sh.tracking_number, sh.status, sh.notes, sh.shipped_at,
		       sh.created_by, sh.notify_customer, sh.created_at, sh.updated_at
		FROM shipments sh
		JOIN orders o ON o.id=sh.order_id AND o.store_id=sh.store_id
		WHERE sh.id=$1 AND sh.store_id=$2
	`, shipmentID, storeID).Scan(
		&sh.ID, &sh.StoreID, &sh.OrderID, &sh.OrderNumber, &sh.CustomerName, &sh.CustomerEmail,
		&sh.Carrier, &sh.TrackingNumber, &sh.Status, &sh.Notes, &sh.ShippedAt,
		&sh.CreatedBy, &sh.NotifyCustomer, &sh.CreatedAt, &sh.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	items, err := s.listShipmentItems(ctx, storeID, shipmentID)
	if err != nil {
		return nil, err
	}
	sh.Items = items
	return &sh, nil
}

func (s *Service) listShipmentItems(ctx context.Context, storeID, shipmentID uuid.UUID) ([]ShipmentItem, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT si.id, si.order_item_id, oi.product_name, oi.sku, oi.variant_label, si.quantity
		FROM shipment_items si
		JOIN order_items oi ON oi.id=si.order_item_id AND oi.store_id=si.store_id
		WHERE si.shipment_id=$1 AND si.store_id=$2
		ORDER BY oi.product_name
	`, shipmentID, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ShipmentItem
	for rows.Next() {
		var it ShipmentItem
		if err := rows.Scan(&it.ID, &it.OrderItemID, &it.ProductName, &it.SKU, &it.VariantLabel, &it.Quantity); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (s *Service) ListOrderShipments(ctx context.Context, storeID, orderID uuid.UUID) ([]Shipment, error) {
	list, _, err := s.ListShipments(ctx, storeID, &orderID, 100, 0)
	if err != nil {
		return nil, err
	}
	for i := range list {
		items, err := s.listShipmentItems(ctx, storeID, list[i].ID)
		if err != nil {
			return nil, err
		}
		list[i].Items = items
	}
	return list, nil
}

func (s *Service) CreateShipment(ctx context.Context, storeID, actorID uuid.UUID, in CreateShipmentInput) (*Shipment, error) {
	carrier := strings.TrimSpace(in.Carrier)
	tracking := strings.TrimSpace(in.TrackingNumber)
	if in.OrderID == uuid.Nil || carrier == "" || tracking == "" || len(in.Items) == 0 {
		return nil, ErrInvalidInput
	}
	for _, it := range in.Items {
		if it.OrderItemID == uuid.Nil || it.Quantity <= 0 {
			return nil, ErrInvalidInput
		}
	}

	o, err := s.GetOrder(ctx, storeID, in.OrderID)
	if err != nil {
		return nil, err
	}
	if o.OrderStatus == OrderCancelled {
		return nil, ErrInvalidTransition
	}
	if o.FulfillmentMethod == FulfillmentPickup {
		return nil, ErrConflict // pickup uses collect flow, not carrier shipment
	}
	if o.PaymentStatus != PaymentPaid {
		return nil, ErrInvalidTransition
	}
	switch o.FulfillmentStatus {
	case FulfillmentReadyToShip, FulfillmentShipped:
		// ok — partial multi-parcel while already shipped, or first ship from ready
	default:
		return nil, ErrInvalidTransition
	}

	itemByID := map[uuid.UUID]OrderItem{}
	for _, it := range o.Items {
		itemByID[it.ID] = it
	}
	alreadyShipped, err := s.shippedQtyByItem(ctx, storeID, in.OrderID)
	if err != nil {
		return nil, err
	}

	notify := true
	if in.NotifyCustomer != nil {
		notify = *in.NotifyCustomer
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	now := s.now()
	var shipmentID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO shipments (
			store_id, order_id, carrier, tracking_number, status, notes,
			shipped_at, created_by, notify_customer
		) VALUES ($1,$2,$3,$4,'shipped',$5,$6,$7,$8)
		RETURNING id
	`, storeID, in.OrderID, carrier, tracking, in.Notes, now, actorID, notify).Scan(&shipmentID)
	if err != nil {
		return nil, err
	}

	for _, line := range in.Items {
		it, ok := itemByID[line.OrderItemID]
		if !ok {
			return nil, ErrInvalidInput
		}
		shippedSoFar := alreadyShipped[line.OrderItemID]
		remain := it.PackedQty - shippedSoFar
		if remain < 0 {
			remain = 0
		}
		if line.Quantity > remain {
			return nil, ErrConflict // cannot ship more than packed remaining
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO shipment_items (store_id, shipment_id, order_item_id, quantity)
			VALUES ($1,$2,$3,$4)
		`, storeID, shipmentID, line.OrderItemID, line.Quantity)
		if err != nil {
			return nil, err
		}
		alreadyShipped[line.OrderItemID] = shippedSoFar + line.Quantity
	}

	// Determine if all packed qty is now shipped → fulfillment shipped.
	fullyShipped := true
	for _, it := range o.Items {
		if it.PackedQty <= 0 {
			continue
		}
		if alreadyShipped[it.ID] < it.PackedQty {
			fullyShipped = false
			break
		}
	}
	if fullyShipped && o.FulfillmentStatus != FulfillmentShipped {
		if canTransitionFulfillment(o.FulfillmentStatus, FulfillmentShipped, o.FulfillmentMethod) ||
			o.FulfillmentStatus == FulfillmentReadyToShip {
			_, err = tx.Exec(ctx, `
				UPDATE orders SET fulfillment_status='shipped', updated_at=now()
				WHERE id=$1 AND store_id=$2
			`, in.OrderID, storeID)
			if err != nil {
				return nil, err
			}
			_, _ = tx.Exec(ctx, `
				INSERT INTO order_status_history (store_id, order_id, dimension, from_status, to_status, actor_user_id, reason)
				VALUES ($1,$2,'fulfillment',$3,'shipped',$4,'shipment_created')
			`, storeID, in.OrderID, o.FulfillmentStatus, actorID)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	if s.Outbox != nil {
		sid := storeID
		_ = s.Outbox.Write(ctx, &sid, "shipment.created", map[string]any{
			"shipment_id": shipmentID.String(),
			"order_id":    in.OrderID.String(),
			"carrier":     carrier,
			"tracking":    tracking,
		}, fmt.Sprintf("shipment.created:%s", shipmentID.String()))
	}

	sh, err := s.GetShipment(ctx, storeID, shipmentID)
	if err != nil {
		return nil, err
	}

	if notify {
		s.notifyShipmentCustomer(ctx, storeID, o, sh)
	}
	oid := in.OrderID
	s.notifyInApp(ctx, storeID, "order_shipped",
		fmt.Sprintf("Shipped %s", o.OrderNumber),
		fmt.Sprintf("%s · %s", carrier, tracking),
		"order", &oid,
	)

	return sh, nil
}

func (s *Service) notifyShipmentCustomer(ctx context.Context, storeID uuid.UUID, o *Order, sh *Shipment) {
	if s.Email == nil || s.Cfg == nil || strings.TrimSpace(o.CustomerEmail) == "" {
		return
	}
	trackURL := ""
	if tok, err := s.EnsureTrackingToken(ctx, storeID, o.ID); err == nil && s.URLs != nil {
		trackURL = s.URLs.Tracking(tok)
	}
	storeName := ""
	_ = s.DB.QueryRow(ctx, `SELECT name FROM stores WHERE id=$1`, storeID).Scan(&storeName)
	if storeName == "" {
		storeName = "Store"
	}
	subject, html, text := email.ShipmentCreatedContent(
		storeName, o.OrderNumber, sh.Carrier, sh.TrackingNumber,
		money.FormatIDR(o.TotalIDR), trackURL,
	)
	_ = s.Email.Enqueue(ctx, email.Message{
		TemplateKey: "order_shipped_customer", To: o.CustomerEmail,
		From: s.Cfg.EmailNotificationFrom, FromName: storeName, ReplyTo: s.Cfg.EmailReplyTo,
		Subject: subject, HTML: html, Text: text,
		IdempotencyKey: fmt.Sprintf("email:shipped:%s", sh.ID.String()),
	})
}
