package commerce

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rubicahq/pikko/services/api/internal/analytics"
	"github.com/rubicahq/pikko/services/api/internal/validate"
)

// OrderSourceLiveQuickAdd marks merchant-assisted lines captured during a live session.
const OrderSourceLiveQuickAdd = "live_quick_add"

var ErrLiveSessionClosed = errors.New("live session closed")
var ErrLiveQuickAddDisabled = errors.New("live quick-add disabled")
var ErrLiveQuotaExceeded = errors.New("live quota exceeded")

type LiveProductStock struct {
	ID                uuid.UUID     `json:"id"`
	CampaignID        uuid.UUID     `json:"campaign_id"`
	ProductID         uuid.UUID     `json:"product_id"`
	ProductName       string        `json:"product_name"`
	ProductSlug       string        `json:"product_slug"`
	BasePriceIDR      int64         `json:"base_price_idr"`
	PriceOverrideIDR  *int64        `json:"price_override_idr,omitempty"`
	EffectivePriceIDR int64         `json:"effective_price_idr"`
	QuantityLimit     *int          `json:"quantity_limit,omitempty"`
	SoldQty           int           `json:"sold_qty"`
	RemainingQty      *int          `json:"remaining_qty,omitempty"`
	Availability      string        `json:"availability"`
	Featured          bool          `json:"featured"`
	DisplayOrder      int           `json:"display_order"`
	InventoryMode     string        `json:"inventory_mode,omitempty"`
	StockAvailable    *int          `json:"stock_available,omitempty"`
	Variants          []LiveVariant `json:"variants,omitempty"`
}

type LiveVariant struct {
	ID                uuid.UUID `json:"id"`
	OptionKey         string    `json:"option_key"`
	SKU               *string   `json:"sku,omitempty"`
	PriceOverrideIDR  *int64    `json:"price_override_idr,omitempty"`
	EffectivePriceIDR int64     `json:"effective_price_idr"`
	StockQuantity     *int      `json:"stock_quantity,omitempty"`
	Availability      string    `json:"availability"`
}

type LiveSession struct {
	ID                  uuid.UUID            `json:"id"`
	StoreID             uuid.UUID            `json:"store_id"`
	Name                string               `json:"name"`
	Slug                string               `json:"slug"`
	Description         *string              `json:"description,omitempty"`
	OpensAt             time.Time            `json:"opens_at"`
	ClosesAt            time.Time            `json:"closes_at"`
	PublicationStatus   string               `json:"publication_status"`
	OperationalStatus   string               `json:"operational_status"`
	LiveQuickAddEnabled bool                 `json:"live_quick_add_enabled"`
	LiveShowOrderFeed   bool                 `json:"live_show_order_feed"`
	IsAcceptingOrders   bool                 `json:"is_accepting_orders"`
	WishlistEnabled     bool                 `json:"wishlist_enabled"`
	OrderCount          int                  `json:"order_count"`
	ItemCount           int                  `json:"item_count"`
	GMVIDR              int64                `json:"gmv_idr"`
	Products            []LiveProductStock   `json:"products,omitempty"`
	Catalogs            []LiveCatalogSummary `json:"catalogs,omitempty"`
}

type LiveFeedEntry struct {
	OrderID       uuid.UUID `json:"order_id"`
	OrderNumber   string    `json:"order_number"`
	CustomerName  string    `json:"customer_name"`
	CustomerEmail string    `json:"customer_email"`
	CustomerPhone *string   `json:"customer_phone,omitempty"`
	OrderStatus   string    `json:"order_status"`
	PaymentStatus string    `json:"payment_status"`
	Source        string    `json:"source"`
	TotalIDR      int64     `json:"total_idr"`
	ItemCount     int       `json:"item_count"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	LatestProduct *string   `json:"latest_product,omitempty"`
	LatestQty     *int      `json:"latest_qty,omitempty"`
}

type LiveQuickAddInput struct {
	OrderID           *uuid.UUID `json:"order_id,omitempty"`
	CustomerID        *uuid.UUID `json:"customer_id,omitempty"`
	CustomerName      string     `json:"customer_name,omitempty"`
	CustomerEmail     string     `json:"customer_email,omitempty"`
	CustomerPhone     *string    `json:"customer_phone,omitempty"`
	ProductID         uuid.UUID  `json:"product_id"`
	VariantID         *uuid.UUID `json:"variant_id,omitempty"`
	Quantity          int        `json:"quantity"`
	IdempotencyKey    string     `json:"idempotency_key"`
	FulfillmentMethod string     `json:"fulfillment_method,omitempty"`
	InternalNotes     *string    `json:"internal_notes,omitempty"`
}

type LiveQuickAddResult struct {
	Order         *Order       `json:"order"`
	AddedItemID   uuid.UUID    `json:"added_item_id"`
	CreatedOrder  bool         `json:"created_order"`
	QuickAddID    uuid.UUID    `json:"quick_add_id"`
	TrackingToken string       `json:"tracking_token,omitempty"`
	TrackingURL   string       `json:"tracking_url,omitempty"`
	Session       *LiveSession `json:"session,omitempty"`
}

type PublicLiveCampaign struct {
	ID                string     `json:"id"`
	Name              string     `json:"name"`
	Slug              string     `json:"slug"`
	Description       *string    `json:"description,omitempty"`
	OpensAt           time.Time  `json:"opens_at"`
	ClosesAt          time.Time  `json:"closes_at"`
	OperationalStatus string     `json:"operational_status"`
	ProductCount      int        `json:"product_count"`
	CoverFileID       *uuid.UUID `json:"cover_file_id,omitempty"`
}

func (s *Service) assertLiveOpen(ctx context.Context, storeID, campaignID uuid.UUID) (LiveSession, error) {
	var sess LiveSession
	var closedEarly *time.Time
	err := s.DB.QueryRow(ctx, `
		SELECT id, store_id, name, slug, description, opens_at, closes_at,
		       publication_status, operational_status,
		       live_quick_add_enabled, live_show_order_feed, closed_early_at
		FROM campaigns
		WHERE id=$1 AND store_id=$2 AND campaign_type='live'
	`, campaignID, storeID).Scan(
		&sess.ID, &sess.StoreID, &sess.Name, &sess.Slug, &sess.Description,
		&sess.OpensAt, &sess.ClosesAt, &sess.PublicationStatus, &sess.OperationalStatus,
		&sess.LiveQuickAddEnabled, &sess.LiveShowOrderFeed, &closedEarly,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return sess, ErrNotFound
	}
	if err != nil {
		return sess, err
	}

	now := s.now().UTC()
	// Refresh early lifecycle from absolute window (UTC).
	if sess.PublicationStatus == "published" &&
		(sess.OperationalStatus == "draft" || sess.OperationalStatus == "scheduled" ||
			sess.OperationalStatus == "open" || sess.OperationalStatus == "closed") &&
		closedEarly == nil {
		derived := "scheduled"
		if !now.Before(sess.OpensAt) && now.Before(sess.ClosesAt) {
			derived = "open"
		} else if !now.Before(sess.ClosesAt) {
			derived = "closed"
		}
		if derived != sess.OperationalStatus {
			_, _ = s.DB.Exec(ctx, `
				UPDATE campaigns SET operational_status=$3, updated_at=now()
				WHERE id=$1 AND store_id=$2
			`, campaignID, storeID, derived)
			sess.OperationalStatus = derived
		}
	}

	sess.IsAcceptingOrders = sess.PublicationStatus == "published" &&
		sess.OperationalStatus == "open" &&
		!now.Before(sess.OpensAt) && now.Before(sess.ClosesAt) &&
		closedEarly == nil
	sess.WishlistEnabled = sess.PublicationStatus == "published" &&
		sess.OperationalStatus == "scheduled"
	return sess, nil
}

func (s *Service) GetLiveSession(ctx context.Context, storeID, campaignID uuid.UUID, withProducts bool) (*LiveSession, error) {
	sess, err := s.assertLiveOpen(ctx, storeID, campaignID)
	if err != nil {
		return nil, err
	}

	_ = s.DB.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(SUM(total_idr),0),
		       COALESCE((
		         SELECT SUM(oi.quantity) FROM order_items oi
		         JOIN orders o2 ON o2.id=oi.order_id
		         WHERE o2.store_id=$1 AND o2.campaign_id=$2 AND o2.order_status <> 'cancelled'
		       ),0)
		FROM orders
		WHERE store_id=$1 AND campaign_id=$2 AND order_status <> 'cancelled'
	`, storeID, campaignID).Scan(&sess.OrderCount, &sess.GMVIDR, &sess.ItemCount)

	if withProducts {
		products, err := s.listLiveProducts(ctx, storeID, campaignID)
		if err != nil {
			return nil, err
		}
		sess.Products = products
		catalogs, err := s.ListLiveCatalogs(ctx, storeID, campaignID)
		if err != nil {
			return nil, err
		}
		sess.Catalogs = catalogs
	}
	return &sess, nil
}

func (s *Service) ListLiveSessions(ctx context.Context, storeID uuid.UUID) ([]LiveSession, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT id FROM campaigns
		WHERE store_id=$1 AND campaign_type='live'
		ORDER BY opens_at DESC
	`, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]LiveSession, 0, len(ids))
	for _, id := range ids {
		sess, err := s.GetLiveSession(ctx, storeID, id, false)
		if err != nil {
			return nil, err
		}
		out = append(out, *sess)
	}
	return out, nil
}

func (s *Service) listLiveProducts(ctx context.Context, storeID, campaignID uuid.UUID) ([]LiveProductStock, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT cp.id, cp.campaign_id, cp.product_id, p.name, p.slug, p.base_price_idr,
		       cp.price_override_idr, cp.quantity_limit, cp.availability, cp.featured, cp.display_order,
		       p.inventory_mode, p.stock_quantity,
		       COALESCE((
		         SELECT SUM(oi.quantity) FROM order_items oi
		         JOIN orders o ON o.id=oi.order_id AND o.store_id=oi.store_id
		         WHERE o.store_id=$2 AND o.campaign_id=$1 AND o.order_status <> 'cancelled'
		           AND oi.product_id=cp.product_id
		       ),0)
		FROM campaign_products cp
		JOIN products p ON p.id=cp.product_id AND p.store_id=cp.store_id
		WHERE cp.campaign_id=$1 AND cp.store_id=$2
		ORDER BY cp.featured DESC, cp.display_order, p.name
	`, campaignID, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []LiveProductStock
	for rows.Next() {
		var p LiveProductStock
		var stock *int
		if err := rows.Scan(
			&p.ID, &p.CampaignID, &p.ProductID, &p.ProductName, &p.ProductSlug, &p.BasePriceIDR,
			&p.PriceOverrideIDR, &p.QuantityLimit, &p.Availability, &p.Featured, &p.DisplayOrder,
			&p.InventoryMode, &stock, &p.SoldQty,
		); err != nil {
			return nil, err
		}
		p.EffectivePriceIDR = p.BasePriceIDR
		if p.PriceOverrideIDR != nil {
			p.EffectivePriceIDR = *p.PriceOverrideIDR
		}
		if p.QuantityLimit != nil {
			rem := *p.QuantityLimit - p.SoldQty
			if rem < 0 {
				rem = 0
			}
			p.RemainingQty = &rem
		}
		if p.InventoryMode == "finite" && stock != nil {
			var reserved int
			_ = s.DB.QueryRow(ctx, `
				SELECT COALESCE(SUM(quantity),0) FROM inventory_reservations
				WHERE store_id=$1 AND product_id=$2 AND status='active' AND batch_id IS NULL AND variant_id IS NULL
			`, storeID, p.ProductID).Scan(&reserved)
			avail := *stock - reserved
			if avail < 0 {
				avail = 0
			}
			p.StockAvailable = &avail
		}
		variants, _ := s.listLiveVariants(ctx, storeID, p.ProductID, p.EffectivePriceIDR)
		p.Variants = variants
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Service) listLiveVariants(ctx context.Context, storeID, productID uuid.UUID, baseEffective int64) ([]LiveVariant, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT id, COALESCE(option_key,''), NULLIF(sku,''), price_override_idr, stock_quantity, availability
		FROM product_variants
		WHERE store_id=$1 AND product_id=$2
		ORDER BY option_key
	`, storeID, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LiveVariant
	for rows.Next() {
		var v LiveVariant
		if err := rows.Scan(&v.ID, &v.OptionKey, &v.SKU, &v.PriceOverrideIDR, &v.StockQuantity, &v.Availability); err != nil {
			return nil, err
		}
		v.EffectivePriceIDR = baseEffective
		if v.PriceOverrideIDR != nil {
			v.EffectivePriceIDR = *v.PriceOverrideIDR
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Service) LiveOrderFeed(ctx context.Context, storeID, campaignID uuid.UUID, since *time.Time, limit int) ([]LiveFeedEntry, error) {
	sess, err := s.assertLiveOpen(ctx, storeID, campaignID)
	if err != nil {
		return nil, err
	}
	if !sess.LiveShowOrderFeed {
		return []LiveFeedEntry{}, nil
	}
	if limit <= 0 || limit > 100 {
		limit = 40
	}
	args := []any{storeID, campaignID}
	q := `
		SELECT o.id, o.order_number, o.customer_name, o.customer_email, o.customer_phone,
		       o.order_status, o.payment_status, o.source, o.total_idr, o.created_at, o.updated_at,
		       COALESCE((SELECT COUNT(*) FROM order_items oi WHERE oi.order_id=o.id),0),
		       (SELECT oi.product_name FROM order_items oi WHERE oi.order_id=o.id ORDER BY oi.created_at DESC LIMIT 1),
		       (SELECT oi.quantity FROM order_items oi WHERE oi.order_id=o.id ORDER BY oi.created_at DESC LIMIT 1)
		FROM orders o
		WHERE o.store_id=$1 AND o.campaign_id=$2 AND o.order_status <> 'cancelled'
	`
	if since != nil {
		args = append(args, since.UTC())
		q += fmt.Sprintf(` AND o.updated_at >= $%d`, len(args))
	}
	args = append(args, limit)
	q += fmt.Sprintf(` ORDER BY o.updated_at DESC LIMIT $%d`, len(args))

	rows, err := s.DB.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []LiveFeedEntry{}
	for rows.Next() {
		var e LiveFeedEntry
		if err := rows.Scan(
			&e.OrderID, &e.OrderNumber, &e.CustomerName, &e.CustomerEmail, &e.CustomerPhone,
			&e.OrderStatus, &e.PaymentStatus, &e.Source, &e.TotalIDR, &e.CreatedAt, &e.UpdatedAt,
			&e.ItemCount, &e.LatestProduct, &e.LatestQty,
		); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// LiveQuickAdd adds a campaign SKU to an existing order or creates a new live order (LIVE-02).
// Concurrency-safe: locks campaign_products + inventory rows; enforces session quota.
func (s *Service) LiveQuickAdd(ctx context.Context, storeID, campaignID, actorID uuid.UUID, in LiveQuickAddInput) (*LiveQuickAddResult, error) {
	if in.Quantity <= 0 || in.Quantity > 999 {
		return nil, ErrInvalidInput
	}
	if strings.TrimSpace(in.IdempotencyKey) == "" || in.ProductID == uuid.Nil {
		return nil, ErrInvalidInput
	}

	// Idempotent replay
	var existingQA uuid.UUID
	var replayOrderID, replayItemID uuid.UUID
	err := s.DB.QueryRow(ctx, `
		SELECT id, order_id, order_item_id FROM live_quick_adds
		WHERE store_id=$1 AND idempotency_key=$2
	`, storeID, in.IdempotencyKey).Scan(&existingQA, &replayOrderID, &replayItemID)
	if err == nil {
		o, err := s.GetOrder(ctx, storeID, replayOrderID)
		if err != nil {
			return nil, err
		}
		sess, _ := s.GetLiveSession(ctx, storeID, campaignID, true)
		return &LiveQuickAddResult{
			Order: o, AddedItemID: replayItemID, QuickAddID: existingQA, Session: sess,
		}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	sess, err := s.assertLiveOpen(ctx, storeID, campaignID)
	if err != nil {
		return nil, err
	}
	if !sess.IsAcceptingOrders {
		return nil, ErrLiveSessionClosed
	}
	if !sess.LiveQuickAddEnabled {
		return nil, ErrLiveQuickAddDisabled
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var cpAvail string
	var priceOverride *int64
	var qtyLimit *int
	var featured bool
	err = tx.QueryRow(ctx, `
		SELECT availability, price_override_idr, quantity_limit, featured
		FROM campaign_products
		WHERE store_id=$1 AND campaign_id=$2 AND product_id=$3
		FOR UPDATE
	`, storeID, campaignID, in.ProductID).Scan(&cpAvail, &priceOverride, &qtyLimit, &featured)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if cpAvail != "available" {
		return nil, ErrInsufficientStock
	}

	var name, slug, status, availability, invMode string
	var productSKU *string
	var basePrice int64
	var stock *int
	err = tx.QueryRow(ctx, `
		SELECT name, slug, NULLIF(sku,''), base_price_idr, status, availability, inventory_mode, stock_quantity
		FROM products WHERE id=$1 AND store_id=$2 FOR UPDATE
	`, in.ProductID, storeID).Scan(&name, &slug, &productSKU, &basePrice, &status, &availability, &invMode, &stock)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if status == "archived" || availability == "unavailable" {
		return nil, ErrInsufficientStock
	}

	unitPrice := basePrice
	if priceOverride != nil {
		unitPrice = *priceOverride
	}
	var variantLabel *string
	sku := productSKU
	var variantID *uuid.UUID
	if in.VariantID != nil {
		var vAvail string
		var vPrice *int64
		var vStock *int
		var optKey string
		var vSKU *string
		err = tx.QueryRow(ctx, `
			SELECT availability, price_override_idr, stock_quantity, option_key, NULLIF(sku,'')
			FROM product_variants WHERE id=$1 AND product_id=$2 AND store_id=$3 FOR UPDATE
		`, *in.VariantID, in.ProductID, storeID).Scan(&vAvail, &vPrice, &vStock, &optKey, &vSKU)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		if err != nil {
			return nil, err
		}
		if vAvail != "available" {
			return nil, ErrInsufficientStock
		}
		if vPrice != nil {
			unitPrice = *vPrice
		}
		variantLabel = &optKey
		if vSKU != nil {
			sku = vSKU
		}
		variantID = in.VariantID
		if vStock != nil {
			stock = vStock
		}
	}

	// Campaign session quota (sold across all non-cancelled orders for this campaign product).
	var sold int
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(oi.quantity),0)
		FROM order_items oi
		JOIN orders o ON o.id=oi.order_id AND o.store_id=oi.store_id
		WHERE o.store_id=$1 AND o.campaign_id=$2 AND o.order_status <> 'cancelled'
		  AND oi.product_id=$3
	`, storeID, campaignID, in.ProductID).Scan(&sold)
	if err != nil {
		return nil, err
	}
	if qtyLimit != nil && sold+in.Quantity > *qtyLimit {
		return nil, ErrLiveQuotaExceeded
	}

	if invMode == "finite" {
		stockQty := 0
		if stock != nil {
			stockQty = *stock
		}
		var reserved int
		err = tx.QueryRow(ctx, `
			SELECT COALESCE(SUM(quantity),0) FROM inventory_reservations
			WHERE store_id=$1 AND product_id=$2 AND status='active' AND batch_id IS NULL
			  AND (($3::uuid IS NULL AND variant_id IS NULL) OR variant_id=$3)
		`, storeID, in.ProductID, variantID).Scan(&reserved)
		if err != nil {
			return nil, err
		}
		if stockQty-reserved < in.Quantity {
			return nil, ErrInsufficientStock
		}
	}

	createdOrder := false
	var orderID uuid.UUID
	var order *Order

	if in.OrderID != nil {
		orderID = *in.OrderID
		var oCampaign *uuid.UUID
		var oStatus, oPay, oFul string
		var oSource string
		err = tx.QueryRow(ctx, `
			SELECT campaign_id, order_status, payment_status, fulfillment_status, source
			FROM orders WHERE id=$1 AND store_id=$2 FOR UPDATE
		`, orderID, storeID).Scan(&oCampaign, &oStatus, &oPay, &oFul, &oSource)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		if err != nil {
			return nil, err
		}
		if oStatus == OrderCancelled || oStatus == OrderCompleted {
			return nil, ErrInvalidTransition
		}
		if oCampaign != nil && *oCampaign != campaignID {
			return nil, ErrConflict
		}
		if oCampaign == nil {
			_, err = tx.Exec(ctx, `
				UPDATE orders SET campaign_id=$3, updated_at=now() WHERE id=$1 AND store_id=$2
			`, orderID, storeID, campaignID)
			if err != nil {
				return nil, err
			}
		}
		_ = oPay
		_ = oFul
		_ = oSource
	} else {
		// Create a new order for the customer.
		nameCust := strings.TrimSpace(in.CustomerName)
		_, emailAddr, emailOK := validate.NormalizeEmail(in.CustomerEmail)
		if len(nameCust) < 2 || !emailOK {
			return nil, ErrInvalidInput
		}
		ful := strings.TrimSpace(in.FulfillmentMethod)
		if ful == "" {
			ful = FulfillmentPickup
		}
		if ful != FulfillmentPickup && ful != FulfillmentShipping && ful != FulfillmentFreeShipping {
			return nil, ErrInvalidInput
		}

		customerID, err := upsertCustomerTx(ctx, tx, storeID, nameCust, emailAddr, in.CustomerPhone)
		if err != nil {
			return nil, err
		}
		if in.CustomerID != nil {
			_ = in.CustomerID
		}
		orderNumber, err := nextOrderNumberTx(ctx, tx, storeID)
		if err != nil {
			return nil, err
		}
		var currency string
		err = tx.QueryRow(ctx, `SELECT currency FROM stores WHERE id=$1`, storeID).Scan(&currency)
		if err != nil {
			return nil, err
		}
		lineTotal := unitPrice * int64(in.Quantity)
		now := s.now()
		idem := "live:" + in.IdempotencyKey
		err = tx.QueryRow(ctx, `
			INSERT INTO orders (
				store_id, customer_id, order_number, idempotency_key, selling_model,
				order_status, payment_status, fulfillment_status, currency,
				subtotal_idr, shipping_fee_idr, handling_fee_idr, total_idr,
				original_total_idr, payable_idr,
				fulfillment_method, customer_name, customer_email, customer_phone,
				internal_notes, confirmed_at, requires_approval,
				source, created_by_user_id, campaign_id
			) VALUES (
				$1,$2,$3,$4,'ready_stock',
				'confirmed','unpaid','unfulfilled',$5,
				$6,0,0,$6,$6,$6,
				$7,$8,$9,$10,$11,$12,false,
				$13,$14,$15
			) RETURNING id
		`, storeID, customerID, orderNumber, idem, currency,
			lineTotal, ful, nameCust, emailAddr, in.CustomerPhone,
			in.InternalNotes, now, OrderSourceLiveQuickAdd, actorID, campaignID,
		).Scan(&orderID)
		if err != nil {
			return nil, err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO order_status_history (store_id, order_id, dimension, from_status, to_status, actor_user_id, reason)
			VALUES ($1,$2,'order',NULL,'confirmed',$3,'live_quick_add'),
			       ($1,$2,'payment',NULL,'unpaid',$3,'live_quick_add'),
			       ($1,$2,'fulfillment',NULL,'unfulfilled',$3,'live_quick_add')
		`, storeID, orderID, actorID)
		if err != nil {
			return nil, err
		}
		createdOrder = true
	}

	// Merge into existing line when same product+variant already on order.
	var existingItemID uuid.UUID
	var existingQty int
	var existingUnit int64
	mergeErr := tx.QueryRow(ctx, `
		SELECT id, quantity, unit_price_idr FROM order_items
		WHERE order_id=$1 AND store_id=$2 AND product_id=$3
		  AND (($4::uuid IS NULL AND variant_id IS NULL) OR variant_id=$4)
		FOR UPDATE
	`, orderID, storeID, in.ProductID, variantID).Scan(&existingItemID, &existingQty, &existingUnit)

	var itemID uuid.UUID
	lineTotal := unitPrice * int64(in.Quantity)
	if mergeErr == nil {
		newQty := existingQty + in.Quantity
		newLine := existingUnit * int64(newQty)
		_, err = tx.Exec(ctx, `
			UPDATE order_items
			SET quantity=$4, line_total_idr=$5, adjusted_line_total_idr=$5
			WHERE id=$1 AND order_id=$2 AND store_id=$3
		`, existingItemID, orderID, storeID, newQty, newLine)
		if err != nil {
			return nil, err
		}
		itemID = existingItemID
		_, _ = tx.Exec(ctx, `
			UPDATE inventory_reservations SET quantity=$4
			WHERE order_item_id=$1 AND order_id=$2 AND store_id=$3 AND status='active'
		`, itemID, orderID, storeID, newQty)
		revNum, err := s.nextRevisionNumber(ctx, tx, orderID)
		if err != nil {
			return nil, err
		}
		summary := fmt.Sprintf("Live quick-add: %s qty %d → %d", name, existingQty, newQty)
		before, _ := json.Marshal(map[string]any{"item_id": itemID.String(), "quantity": existingQty})
		after, _ := json.Marshal(map[string]any{"item_id": itemID.String(), "quantity": newQty})
		var revID uuid.UUID
		err = tx.QueryRow(ctx, `
			INSERT INTO order_revisions (
				store_id, order_id, revision_number, change_type, summary,
				before_snapshot, after_snapshot, actor_user_id, reason
			) VALUES ($1,$2,$3,'live_quick_add',$4,$5,$6,$7,'live_quick_add')
			RETURNING id
		`, storeID, orderID, revNum, summary, before, after, actorID).Scan(&revID)
		if err != nil {
			return nil, err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO order_item_events (
				store_id, order_id, order_item_id, revision_id, event_type,
				from_quantity, to_quantity, actor_user_id, reason, meta
			) VALUES ($1,$2,$3,$4,'live_added',$5,$6,$7,'live_quick_add',$8)
		`, storeID, orderID, itemID, revID, existingQty, newQty, actorID,
			liveMetaJSON(map[string]any{"campaign_id": campaignID.String(), "featured": featured}))
		if err != nil {
			return nil, err
		}
	} else if errors.Is(mergeErr, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `
			INSERT INTO order_items (
				store_id, order_id, product_id, variant_id, product_name, product_slug,
				sku, variant_label, unit_price_idr, quantity, original_quantity, line_total_idr,
				availability_status, allocated_qty, packed_qty
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$10,$11,'available',0,0)
			RETURNING id
		`, storeID, orderID, in.ProductID, variantID, name, slug,
			sku, variantLabel, unitPrice, in.Quantity, lineTotal,
		).Scan(&itemID)
		if err != nil {
			return nil, err
		}
		if invMode == "finite" {
			_, err = tx.Exec(ctx, `
				INSERT INTO inventory_reservations (
					store_id, product_id, variant_id, order_id, order_item_id, quantity, status
				) VALUES ($1,$2,$3,$4,$5,$6,'active')
			`, storeID, in.ProductID, variantID, orderID, itemID, in.Quantity)
			if err != nil {
				return nil, err
			}
		}
		revNum, err := s.nextRevisionNumber(ctx, tx, orderID)
		if err != nil {
			return nil, err
		}
		summary := fmt.Sprintf("Live quick-add: %s × %d", name, in.Quantity)
		after, _ := json.Marshal(map[string]any{
			"item_id": itemID.String(), "product_id": in.ProductID.String(), "quantity": in.Quantity,
		})
		var revID uuid.UUID
		err = tx.QueryRow(ctx, `
			INSERT INTO order_revisions (
				store_id, order_id, revision_number, change_type, summary,
				before_snapshot, after_snapshot, actor_user_id, reason
			) VALUES ($1,$2,$3,'live_quick_add',$4,NULL,$5,$6,'live_quick_add')
			RETURNING id
		`, storeID, orderID, revNum, summary, after, actorID).Scan(&revID)
		if err != nil {
			return nil, err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO order_item_events (
				store_id, order_id, order_item_id, revision_id, event_type,
				from_quantity, to_quantity, actor_user_id, reason, meta
			) VALUES ($1,$2,$3,$4,'live_added',NULL,$5,$6,'live_quick_add',$7)
		`, storeID, orderID, itemID, revID, in.Quantity, actorID,
			liveMetaJSON(map[string]any{"campaign_id": campaignID.String()}))
		if err != nil {
			return nil, err
		}
	} else {
		return nil, mergeErr
	}

	// Recompute order totals from items.
	var subtotal int64
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(line_total_idr),0) FROM order_items WHERE order_id=$1 AND store_id=$2
	`, orderID, storeID).Scan(&subtotal)
	if err != nil {
		return nil, err
	}
	var shipFee, handleFee int64
	err = tx.QueryRow(ctx, `
		SELECT shipping_fee_idr, handling_fee_idr FROM orders WHERE id=$1 AND store_id=$2
	`, orderID, storeID).Scan(&shipFee, &handleFee)
	if err != nil {
		return nil, err
	}
	total := subtotal + shipFee + handleFee
	_, err = tx.Exec(ctx, `
		UPDATE orders SET subtotal_idr=$3, total_idr=$4, payable_idr=$4, updated_at=now()
		WHERE id=$1 AND store_id=$2
	`, orderID, storeID, subtotal, total)
	if err != nil {
		return nil, err
	}

	var qaID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO live_quick_adds (
			store_id, campaign_id, order_id, order_item_id, product_id, variant_id,
			quantity, unit_price_idr, actor_user_id, idempotency_key
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id
	`, storeID, campaignID, orderID, itemID, in.ProductID, variantID,
		in.Quantity, unitPrice, actorID, in.IdempotencyKey,
	).Scan(&qaID)
	if err != nil {
		if strings.Contains(err.Error(), "live_quick_adds_idempotency_unique") {
			return nil, ErrConflict
		}
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	order, err = s.GetOrder(ctx, storeID, orderID)
	if err != nil {
		return nil, err
	}
	updated, _ := s.GetLiveSession(ctx, storeID, campaignID, true)

	res := &LiveQuickAddResult{
		Order: order, AddedItemID: itemID, CreatedOrder: createdOrder,
		QuickAddID: qaID, Session: updated,
	}
	if createdOrder && s.URLs != nil {
		token, terr := s.EnsureTrackingToken(ctx, storeID, orderID)
		if terr == nil {
			res.TrackingToken = token
			res.TrackingURL = s.URLs.Tracking(token)
			order.TrackingToken = token
			order.TrackingURL = res.TrackingURL
		}
	}
	analytics.Emit(ctx, s.DB, storeID, "live.quick_add", &actorID, map[string]any{
		"campaign_id":   campaignID.String(),
		"order_id":      orderID.String(),
		"quantity":      in.Quantity,
		"created_order": createdOrder,
	})
	return res, nil
}

func liveMetaJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func (s *Service) ListPublicLiveCampaigns(ctx context.Context, storeSlug string) ([]PublicLiveCampaign, error) {
	st, err := s.loadPublicStore(ctx, storeSlug)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	rows, err := s.DB.Query(ctx, `
		SELECT c.id, c.name, c.slug, c.description, c.opens_at, c.closes_at,
		       c.operational_status, c.cover_file_id,
		       COALESCE((
                 SELECT COUNT(DISTINCT product_id) FROM (
                   SELECT cp.product_id FROM campaign_products cp
                   WHERE cp.campaign_id=c.id
                   UNION
                   SELECT h.product_id FROM catalog_campaigns cc
                   JOIN catalogs cat ON cat.id=cc.catalog_id AND cat.status='published'
                   JOIN catalog_pages pg ON pg.catalog_id=cat.id AND pg.status='published'
                   JOIN catalog_page_hotspots h ON h.catalog_page_id=pg.id AND h.product_id IS NOT NULL
                   JOIN products p ON p.id=h.product_id AND p.store_id=c.store_id AND p.status='published'
                   WHERE cc.campaign_id=c.id AND cc.store_id=c.store_id
                 ) linked
               ),0)
		FROM campaigns c
		WHERE c.store_id=$1 AND c.campaign_type='live' AND c.publication_status='published'
		  AND c.operational_status IN ('scheduled','open','closed')
		ORDER BY
		  CASE WHEN c.operational_status='open' THEN 0
		       WHEN c.operational_status='scheduled' THEN 1
		       ELSE 2 END,
		  c.opens_at DESC
	`, st.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PublicLiveCampaign{}
	for rows.Next() {
		var c PublicLiveCampaign
		var id uuid.UUID
		if err := rows.Scan(
			&id, &c.Name, &c.Slug, &c.Description, &c.OpensAt, &c.ClosesAt,
			&c.OperationalStatus, &c.CoverFileID, &c.ProductCount,
		); err != nil {
			return nil, err
		}
		c.ID = id.String()
		// Derive open/closed for display if timestamps say so.
		if c.OperationalStatus != "closed" {
			if !now.Before(c.OpensAt) && now.Before(c.ClosesAt) {
				c.OperationalStatus = "open"
			} else if now.Before(c.OpensAt) {
				c.OperationalStatus = "scheduled"
			} else {
				c.OperationalStatus = "closed"
			}
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Service) GetPublicLiveCampaign(ctx context.Context, storeSlug, campaignSlug string) (*LiveSession, error) {
	st, err := s.loadPublicStore(ctx, storeSlug)
	if err != nil {
		return nil, err
	}
	var campaignID uuid.UUID
	err = s.DB.QueryRow(ctx, `
		SELECT id FROM campaigns
		WHERE store_id=$1 AND slug=$2 AND campaign_type='live' AND publication_status='published'
	`, st.ID, campaignSlug).Scan(&campaignID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	sess, err := s.GetLiveSession(ctx, st.ID, campaignID, true)
	if err != nil {
		return nil, err
	}
	// Public payload: only available products.
	filtered := make([]LiveProductStock, 0, len(sess.Products))
	for _, p := range sess.Products {
		if p.Availability == "available" {
			filtered = append(filtered, p)
		}
	}
	if sess.WishlistEnabled {
		linked, err := s.listLinkedCatalogWishlistProducts(ctx, st.ID, campaignID)
		if err != nil {
			return nil, err
		}
		seen := make(map[uuid.UUID]bool, len(filtered))
		for _, p := range filtered {
			seen[p.ProductID] = true
		}
		for _, p := range linked {
			if !seen[p.ProductID] {
				filtered = append(filtered, p)
				seen[p.ProductID] = true
			}
		}
	}
	sess.Products = filtered
	return sess, nil
}

// listLinkedCatalogWishlistProducts exposes published catalog items during a scheduled live.
func (s *Service) listLinkedCatalogWishlistProducts(ctx context.Context, storeID, campaignID uuid.UUID) ([]LiveProductStock, error) {
	rows, err := s.DB.Query(ctx, `
        SELECT DISTINCT p.id, p.name, p.slug, p.base_price_idr
        FROM catalog_campaigns cc
        JOIN catalogs cat ON cat.id=cc.catalog_id AND cat.status='published'
        JOIN catalog_pages pg ON pg.catalog_id=cat.id AND pg.status='published'
        JOIN catalog_page_hotspots h ON h.catalog_page_id=pg.id AND h.product_id IS NOT NULL
        JOIN products p ON p.id=h.product_id AND p.store_id=$1 AND p.status='published'
        WHERE cc.store_id=$1 AND cc.campaign_id=$2
        ORDER BY p.name
    `, storeID, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []LiveProductStock{}
	for rows.Next() {
		p := LiveProductStock{CampaignID: campaignID, Availability: "available"}
		if err := rows.Scan(&p.ProductID, &p.ProductName, &p.ProductSlug, &p.BasePriceIDR); err != nil {
			return nil, err
		}
		p.EffectivePriceIDR = p.BasePriceIDR
		out = append(out, p)
	}
	return out, rows.Err()
}

// SearchLiveOrders finds recent orders for quick-add target selection (by email/phone/number).
func (s *Service) SearchLiveOrders(ctx context.Context, storeID, campaignID uuid.UUID, q string, limit int) ([]LiveFeedEntry, error) {
	if _, err := s.assertLiveOpen(ctx, storeID, campaignID); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 25 {
		limit = 12
	}
	q = strings.TrimSpace(q)
	like := "%" + q + "%"
	rows, err := s.DB.Query(ctx, `
		SELECT o.id, o.order_number, o.customer_name, o.customer_email, o.customer_phone,
		       o.order_status, o.payment_status, o.source, o.total_idr, o.created_at, o.updated_at,
		       COALESCE((SELECT COUNT(*) FROM order_items oi WHERE oi.order_id=o.id),0),
		       NULL::text, NULL::int
		FROM orders o
		WHERE o.store_id=$1
		  AND o.order_status NOT IN ('cancelled','completed')
		  AND (
		    $3 = '' OR o.order_number ILIKE $3 OR o.customer_email ILIKE $3
		    OR o.customer_name ILIKE $3 OR COALESCE(o.customer_phone,'') ILIKE $3
		  )
		  AND (o.campaign_id IS NULL OR o.campaign_id=$2)
		ORDER BY
		  CASE WHEN o.campaign_id=$2 THEN 0 ELSE 1 END,
		  o.updated_at DESC
		LIMIT $4
	`, storeID, campaignID, like, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []LiveFeedEntry{}
	for rows.Next() {
		var e LiveFeedEntry
		if err := rows.Scan(
			&e.OrderID, &e.OrderNumber, &e.CustomerName, &e.CustomerEmail, &e.CustomerPhone,
			&e.OrderStatus, &e.PaymentStatus, &e.Source, &e.TotalIDR, &e.CreatedAt, &e.UpdatedAt,
			&e.ItemCount, &e.LatestProduct, &e.LatestQty,
		); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
