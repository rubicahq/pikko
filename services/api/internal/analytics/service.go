package analytics

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Default store timezone for merchant analytics (PRD §12 / Asia/Jakarta UI).
const DefaultTZ = "Asia/Jakarta"

// Successfully fulfilled fulfillment statuses (AN-01).
var fulfilledStatuses = []string{"shipped", "delivered", "collected"}

type Overview struct {
	Timezone string `json:"timezone"`
	From     string `json:"from"` // YYYY-MM-DD in store TZ
	To       string `json:"to"`

	// Legacy order-level sales (preserved; unpaid never labeled revenue).
	TotalOrders              int     `json:"total_orders"`
	PaidOrders               int     `json:"paid_orders"`
	VerifiedPaymentValueIDR  int64   `json:"verified_payment_value_idr"`
	UnpaidOrderValueIDR      int64   `json:"unpaid_order_value_idr"`
	TotalOrderValueIDR       int64   `json:"total_order_value_idr"`
	UniqueCustomers          int     `json:"unique_customers"`
	ConversionRate           float64 `json:"conversion_rate"` // paid / total orders in range

	// AN-01 North star: successfully fulfilled order item quantities.
	FulfilledOrderItems         int   `json:"fulfilled_order_items"`
	FulfilledOrderItemValueIDR  int64 `json:"fulfilled_order_item_value_idr"`
	FulfilledOrderItemsAllTime  int   `json:"fulfilled_order_items_all_time"`

	Daily       []DailyPoint `json:"daily"`
	TopProducts []TopProduct `json:"top_products"`

	// AN-02 practical MVP depth
	Activation  ActivationMetrics  `json:"activation"`
	Engagement  EngagementMetrics  `json:"engagement"`
	Operational OperationalMetrics `json:"operational"`
	Retention   RetentionMetrics   `json:"retention"`

	// AN-03 event sink summary (real rows only; empty when none)
	Events EventSummary `json:"events"`
}

type DailyPoint struct {
	Date              string `json:"date"`
	Orders            int    `json:"orders"`
	PaidOrders        int    `json:"paid_orders"`
	PaidValueIDR      int64  `json:"paid_value_idr"`
	OrderValueIDR     int64  `json:"order_value_idr"`
	FulfilledItems    int    `json:"fulfilled_items"`
	FulfilledValueIDR int64  `json:"fulfilled_value_idr"`
}

type TopProduct struct {
	ProductID   uuid.UUID `json:"product_id"`
	ProductName string    `json:"product_name"`
	Quantity    int       `json:"quantity"`
	RevenueIDR  int64     `json:"revenue_idr"`
}

type ActivationMetrics struct {
	HasProduct           bool       `json:"has_product"`
	HasPublishedProduct  bool       `json:"has_published_product"`
	HasPaymentMethod     bool       `json:"has_payment_method"`
	HasFulfillmentConfig bool       `json:"has_fulfillment_config"`
	IsPublished          bool       `json:"is_published"`
	HasOrder             bool       `json:"has_order"`
	HasPaidOrder         bool       `json:"has_paid_order"`
	HasFulfilledItem     bool       `json:"has_fulfilled_item"`
	FirstOrderAt         *time.Time `json:"first_order_at,omitempty"`
	ActivatedAt          *time.Time `json:"activated_at,omitempty"` // first paid order
	ChecklistComplete    int        `json:"checklist_complete"`
	ChecklistTotal       int        `json:"checklist_total"`
}

type EngagementMetrics struct {
	OrdersInRange       int `json:"orders_in_range"`
	PaidOrdersInRange   int `json:"paid_orders_in_range"`
	LiveSessionsInRange int `json:"live_sessions_in_range"`
	LiveQuickAddsInRange int `json:"live_quick_adds_in_range"`
	ActiveCampaigns     int `json:"active_campaigns"`
	ProductsSoldSKU     int `json:"products_sold_sku"`
}

type OperationalMetrics struct {
	AwaitingPaymentVerification int   `json:"awaiting_payment_verification"`
	PendingConfirmation         int   `json:"pending_confirmation"`
	ReadyToShip                 int   `json:"ready_to_ship"`
	ReadyForPickup              int   `json:"ready_for_pickup"`
	PackingQueueOrders          int   `json:"packing_queue_orders"`
	OpenRefunds                 int   `json:"open_refunds"`
	UnpaidOrderValueIDR         int64 `json:"unpaid_order_value_idr"`
	LiveSessionsOpen            int   `json:"live_sessions_open"`
}

type RetentionMetrics struct {
	NewCustomers       int     `json:"new_customers"`
	ReturningCustomers int     `json:"returning_customers"`
	RepeatOrderRate    float64 `json:"repeat_order_rate"` // returning / (new+returning) when >0
	CustomersWith2Plus int     `json:"customers_with_2_plus_orders"`
}

type EventSummary struct {
	TotalInRange int            `json:"total_in_range"`
	ByName       []EventCount   `json:"by_name"`
}

type EventCount struct {
	EventName string `json:"event_name"`
	Count     int    `json:"count"`
}

type EventInput struct {
	EventName  string         `json:"event_name"`
	SessionID  string         `json:"session_id,omitempty"`
	UserID     *uuid.UUID     `json:"user_id,omitempty"`
	Properties map[string]any `json:"properties,omitempty"`
	OccurredAt *time.Time     `json:"occurred_at,omitempty"`
}

type Service struct {
	DB *pgxpool.Pool
}

func loadLocation(tz string) *time.Location {
	if tz == "" {
		tz = DefaultTZ
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc, _ = time.LoadLocation(DefaultTZ)
	}
	return loc
}

// rangeBounds returns UTC [start, endExclusive) covering inclusive local dates from/to.
func rangeBounds(from, to string, loc *time.Location) (time.Time, time.Time, string, string, error) {
	nowLocal := time.Now().In(loc)
	if to == "" {
		to = nowLocal.Format("2006-01-02")
	}
	if from == "" {
		from = nowLocal.AddDate(0, 0, -29).Format("2006-01-02")
	}
	startLocal, err := time.ParseInLocation("2006-01-02", from, loc)
	if err != nil {
		return time.Time{}, time.Time{}, "", "", err
	}
	endLocal, err := time.ParseInLocation("2006-01-02", to, loc)
	if err != nil {
		return time.Time{}, time.Time{}, "", "", err
	}
	endExclusive := endLocal.AddDate(0, 0, 1)
	return startLocal.UTC(), endExclusive.UTC(), from, to, nil
}

// RecordEvent inserts one analytics_events row (AN-03). Never fabricates metrics.
func (s *Service) RecordEvent(ctx context.Context, storeID uuid.UUID, in EventInput) error {
	name := in.EventName
	if name == "" || len(name) > 120 {
		return errInvalidEvent
	}
	props := in.Properties
	if props == nil {
		props = map[string]any{}
	}
	raw, err := json.Marshal(props)
	if err != nil {
		return err
	}
	occurred := time.Now().UTC()
	if in.OccurredAt != nil {
		occurred = in.OccurredAt.UTC()
	}
	var session any
	if in.SessionID != "" {
		session = in.SessionID
	}
	_, err = s.DB.Exec(ctx, `
		INSERT INTO analytics_events (store_id, event_name, session_id, user_id, properties, occurred_at)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6)
	`, storeID, name, session, in.UserID, string(raw), occurred)
	return err
}

// Emit is a package-level helper for other packages (commerce) to sink events without fabricating.
func Emit(ctx context.Context, db *pgxpool.Pool, storeID uuid.UUID, eventName string, userID *uuid.UUID, props map[string]any) {
	if db == nil || eventName == "" {
		return
	}
	svc := &Service{DB: db}
	_ = svc.RecordEvent(ctx, storeID, EventInput{
		EventName:  eventName,
		UserID:     userID,
		Properties: props,
	})
}

var errInvalidEvent = errString("invalid analytics event")

type errString string

func (e errString) Error() string { return string(e) }

func (s *Service) Overview(ctx context.Context, storeID uuid.UUID, from, to, tz string) (*Overview, error) {
	if tz == "" {
		_ = s.DB.QueryRow(ctx, `SELECT COALESCE(NULLIF(timezone,''), $2) FROM stores WHERE id=$1`, storeID, DefaultTZ).Scan(&tz)
		if tz == "" {
			tz = DefaultTZ
		}
	}
	loc := loadLocation(tz)
	startUTC, endUTC, fromS, toS, err := rangeBounds(from, to, loc)
	if err != nil {
		return nil, err
	}

	out := &Overview{Timezone: tz, From: fromS, To: toS}

	err = s.DB.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE order_status NOT IN ('cancelled')),
			COUNT(*) FILTER (WHERE payment_status='paid' AND order_status NOT IN ('cancelled')),
			COALESCE(SUM(total_idr) FILTER (WHERE payment_status='paid' AND order_status NOT IN ('cancelled')),0),
			COALESCE(SUM(total_idr) FILTER (
				WHERE payment_status IN ('unpaid','awaiting_verification')
				  AND order_status NOT IN ('cancelled')
			),0),
			COALESCE(SUM(total_idr) FILTER (WHERE order_status NOT IN ('cancelled')),0),
			COUNT(DISTINCT customer_id) FILTER (WHERE order_status NOT IN ('cancelled') AND customer_id IS NOT NULL)
		FROM orders
		WHERE store_id=$1 AND created_at >= $2 AND created_at < $3
	`, storeID, startUTC, endUTC).Scan(
		&out.TotalOrders, &out.PaidOrders, &out.VerifiedPaymentValueIDR,
		&out.UnpaidOrderValueIDR, &out.TotalOrderValueIDR, &out.UniqueCustomers,
	)
	if err != nil {
		return nil, err
	}
	if out.TotalOrders > 0 {
		out.ConversionRate = float64(out.PaidOrders) / float64(out.TotalOrders)
	}
	out.Operational.UnpaidOrderValueIDR = out.UnpaidOrderValueIDR

	// AN-01: fulfilled item qty by fulfillment transition time in range (never fabricate).
	err = s.DB.QueryRow(ctx, `
		WITH fulfilled_at AS (
			SELECT o.id AS order_id,
			       COALESCE(
			         (SELECT MIN(h.created_at) FROM order_status_history h
			          WHERE h.order_id=o.id AND h.store_id=o.store_id
			            AND h.dimension='fulfillment'
			            AND h.to_status = ANY($4::text[])),
			         o.completed_at,
			         o.updated_at
			       ) AS at
			FROM orders o
			WHERE o.store_id=$1
			  AND o.order_status NOT IN ('cancelled')
			  AND o.payment_status='paid'
			  AND o.fulfillment_status = ANY($4::text[])
		)
		SELECT COALESCE(SUM(oi.quantity),0), COALESCE(SUM(oi.line_total_idr),0)
		FROM order_items oi
		JOIN fulfilled_at f ON f.order_id=oi.order_id
		WHERE oi.store_id=$1
		  AND f.at >= $2 AND f.at < $3
	`, storeID, startUTC, endUTC, fulfilledStatuses).Scan(
		&out.FulfilledOrderItems, &out.FulfilledOrderItemValueIDR,
	)
	if err != nil {
		return nil, err
	}

	err = s.DB.QueryRow(ctx, `
		SELECT COALESCE(SUM(oi.quantity),0)
		FROM order_items oi
		JOIN orders o ON o.id=oi.order_id AND o.store_id=oi.store_id
		WHERE oi.store_id=$1
		  AND o.order_status NOT IN ('cancelled')
		  AND o.payment_status='paid'
		  AND o.fulfillment_status = ANY($2::text[])
	`, storeID, fulfilledStatuses).Scan(&out.FulfilledOrderItemsAllTime)
	if err != nil {
		return nil, err
	}

	if err := s.fillDaily(ctx, out, storeID, startUTC, endUTC, tz); err != nil {
		return nil, err
	}
	if err := s.fillTopProducts(ctx, out, storeID, startUTC, endUTC); err != nil {
		return nil, err
	}
	if err := s.fillActivation(ctx, out, storeID); err != nil {
		return nil, err
	}
	if err := s.fillEngagement(ctx, out, storeID, startUTC, endUTC); err != nil {
		return nil, err
	}
	if err := s.fillOperational(ctx, out, storeID); err != nil {
		return nil, err
	}
	if err := s.fillRetention(ctx, out, storeID, startUTC, endUTC); err != nil {
		return nil, err
	}
	if err := s.fillEvents(ctx, out, storeID, startUTC, endUTC); err != nil {
		return nil, err
	}

	if out.Daily == nil {
		out.Daily = []DailyPoint{}
	}
	if out.TopProducts == nil {
		out.TopProducts = []TopProduct{}
	}
	if out.Events.ByName == nil {
		out.Events.ByName = []EventCount{}
	}
	return out, nil
}

func (s *Service) fillDaily(ctx context.Context, out *Overview, storeID uuid.UUID, startUTC, endUTC time.Time, tz string) error {
	rows, err := s.DB.Query(ctx, `
		WITH order_days AS (
			SELECT to_char(timezone($4, created_at), 'YYYY-MM-DD') AS d,
			       COUNT(*) FILTER (WHERE order_status NOT IN ('cancelled')) AS orders,
			       COUNT(*) FILTER (WHERE payment_status='paid' AND order_status NOT IN ('cancelled')) AS paid_orders,
			       COALESCE(SUM(total_idr) FILTER (WHERE payment_status='paid' AND order_status NOT IN ('cancelled')),0) AS paid_value,
			       COALESCE(SUM(total_idr) FILTER (WHERE order_status NOT IN ('cancelled')),0) AS order_value
			FROM orders
			WHERE store_id=$1 AND created_at >= $2 AND created_at < $3
			GROUP BY 1
		),
		fulfilled_at AS (
			SELECT o.id AS order_id,
			       COALESCE(
			         (SELECT MIN(h.created_at) FROM order_status_history h
			          WHERE h.order_id=o.id AND h.store_id=o.store_id
			            AND h.dimension='fulfillment'
			            AND h.to_status = ANY($5::text[])),
			         o.completed_at,
			         o.updated_at
			       ) AS at
			FROM orders o
			WHERE o.store_id=$1
			  AND o.order_status NOT IN ('cancelled')
			  AND o.payment_status='paid'
			  AND o.fulfillment_status = ANY($5::text[])
		),
		fulfill_days AS (
			SELECT to_char(timezone($4, f.at), 'YYYY-MM-DD') AS d,
			       COALESCE(SUM(oi.quantity),0) AS fulfilled_items,
			       COALESCE(SUM(oi.line_total_idr),0) AS fulfilled_value
			FROM order_items oi
			JOIN fulfilled_at f ON f.order_id=oi.order_id
			WHERE oi.store_id=$1 AND f.at >= $2 AND f.at < $3
			GROUP BY 1
		)
		SELECT COALESCE(od.d, fd.d),
		       COALESCE(od.orders,0), COALESCE(od.paid_orders,0),
		       COALESCE(od.paid_value,0), COALESCE(od.order_value,0),
		       COALESCE(fd.fulfilled_items,0), COALESCE(fd.fulfilled_value,0)
		FROM order_days od
		FULL OUTER JOIN fulfill_days fd ON od.d = fd.d
		ORDER BY 1
	`, storeID, startUTC, endUTC, tz, fulfilledStatuses)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var p DailyPoint
		if err := rows.Scan(&p.Date, &p.Orders, &p.PaidOrders, &p.PaidValueIDR, &p.OrderValueIDR, &p.FulfilledItems, &p.FulfilledValueIDR); err != nil {
			return err
		}
		out.Daily = append(out.Daily, p)
	}
	return rows.Err()
}

func (s *Service) fillTopProducts(ctx context.Context, out *Overview, storeID uuid.UUID, startUTC, endUTC time.Time) error {
	prows, err := s.DB.Query(ctx, `
		SELECT oi.product_id, MAX(oi.product_name),
		       COALESCE(SUM(oi.quantity),0),
		       COALESCE(SUM(oi.line_total_idr),0)
		FROM order_items oi
		JOIN orders o ON o.id=oi.order_id AND o.store_id=oi.store_id
		WHERE oi.store_id=$1
		  AND o.created_at >= $2 AND o.created_at < $3
		  AND o.order_status NOT IN ('cancelled')
		  AND o.payment_status='paid'
		  AND oi.product_id IS NOT NULL
		GROUP BY oi.product_id
		ORDER BY SUM(oi.line_total_idr) DESC
		LIMIT 10
	`, storeID, startUTC, endUTC)
	if err != nil {
		return err
	}
	defer prows.Close()
	for prows.Next() {
		var tp TopProduct
		if err := prows.Scan(&tp.ProductID, &tp.ProductName, &tp.Quantity, &tp.RevenueIDR); err != nil {
			return err
		}
		out.TopProducts = append(out.TopProducts, tp)
	}
	return prows.Err()
}

func (s *Service) fillActivation(ctx context.Context, out *Overview, storeID uuid.UUID) error {
	a := &out.Activation
	a.ChecklistTotal = 7
	_ = s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM products WHERE store_id=$1)`, storeID).Scan(&a.HasProduct)
	_ = s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM products WHERE store_id=$1 AND status='published')`, storeID).Scan(&a.HasPublishedProduct)
	_ = s.DB.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM store_payment_methods WHERE store_id=$1 AND is_enabled=true)
	`, storeID).Scan(&a.HasPaymentMethod)
	_ = s.DB.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM store_fulfillment_settings
			WHERE store_id=$1 AND (shipping_enabled OR free_shipping_enabled OR pickup_enabled)
		)
	`, storeID).Scan(&a.HasFulfillmentConfig)
	_ = s.DB.QueryRow(ctx, `SELECT publication_status='published' FROM stores WHERE id=$1`, storeID).Scan(&a.IsPublished)
	_ = s.DB.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM orders WHERE store_id=$1 AND order_status NOT IN ('cancelled'))
	`, storeID).Scan(&a.HasOrder)
	_ = s.DB.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM orders WHERE store_id=$1 AND payment_status='paid' AND order_status NOT IN ('cancelled'))
	`, storeID).Scan(&a.HasPaidOrder)
	a.HasFulfilledItem = out.FulfilledOrderItemsAllTime > 0

	var firstOrder, firstPaid *time.Time
	_ = s.DB.QueryRow(ctx, `
		SELECT MIN(created_at) FROM orders
		WHERE store_id=$1 AND order_status NOT IN ('cancelled')
	`, storeID).Scan(&firstOrder)
	_ = s.DB.QueryRow(ctx, `
		SELECT MIN(created_at) FROM orders
		WHERE store_id=$1 AND payment_status='paid' AND order_status NOT IN ('cancelled')
	`, storeID).Scan(&firstPaid)
	a.FirstOrderAt = firstOrder
	a.ActivatedAt = firstPaid

	n := 0
	if a.HasProduct {
		n++
	}
	if a.HasPublishedProduct {
		n++
	}
	if a.HasPaymentMethod {
		n++
	}
	if a.HasFulfillmentConfig {
		n++
	}
	if a.IsPublished {
		n++
	}
	if a.HasOrder {
		n++
	}
	if a.HasFulfilledItem {
		n++
	}
	a.ChecklistComplete = n
	return nil
}

func (s *Service) fillEngagement(ctx context.Context, out *Overview, storeID uuid.UUID, startUTC, endUTC time.Time) error {
	e := &out.Engagement
	e.OrdersInRange = out.TotalOrders
	e.PaidOrdersInRange = out.PaidOrders
	_ = s.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM campaigns
		WHERE store_id=$1 AND campaign_type='live'
		  AND created_at >= $2 AND created_at < $3
	`, storeID, startUTC, endUTC).Scan(&e.LiveSessionsInRange)
	_ = s.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM live_quick_adds
		WHERE store_id=$1 AND created_at >= $2 AND created_at < $3
	`, storeID, startUTC, endUTC).Scan(&e.LiveQuickAddsInRange)
	_ = s.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM campaigns
		WHERE store_id=$1
		  AND publication_status='published'
		  AND operational_status IN ('open','scheduled')
		  AND (closes_at IS NULL OR closes_at > now())
	`, storeID).Scan(&e.ActiveCampaigns)
	_ = s.DB.QueryRow(ctx, `
		SELECT COUNT(DISTINCT oi.product_id)
		FROM order_items oi
		JOIN orders o ON o.id=oi.order_id AND o.store_id=oi.store_id
		WHERE oi.store_id=$1
		  AND o.created_at >= $2 AND o.created_at < $3
		  AND o.payment_status='paid' AND o.order_status NOT IN ('cancelled')
		  AND oi.product_id IS NOT NULL
	`, storeID, startUTC, endUTC).Scan(&e.ProductsSoldSKU)
	return nil
}

func (s *Service) fillOperational(ctx context.Context, out *Overview, storeID uuid.UUID) error {
	op := &out.Operational
	_ = s.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM orders
		WHERE store_id=$1 AND order_status NOT IN ('cancelled')
		  AND payment_status='awaiting_verification'
	`, storeID).Scan(&op.AwaitingPaymentVerification)
	_ = s.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM orders
		WHERE store_id=$1 AND order_status='pending_confirmation'
	`, storeID).Scan(&op.PendingConfirmation)
	_ = s.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM orders
		WHERE store_id=$1 AND order_status NOT IN ('cancelled')
		  AND fulfillment_status='ready_to_ship'
	`, storeID).Scan(&op.ReadyToShip)
	_ = s.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM orders
		WHERE store_id=$1 AND order_status NOT IN ('cancelled')
		  AND fulfillment_status='ready_for_pickup'
	`, storeID).Scan(&op.ReadyForPickup)
	_ = s.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM orders
		WHERE store_id=$1 AND order_status NOT IN ('cancelled','completed')
		  AND payment_status='paid'
		  AND fulfillment_status IN ('unfulfilled','awaiting_arrival','ready_to_ship','ready_for_pickup','purchasing')
	`, storeID).Scan(&op.PackingQueueOrders)
	_ = s.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM refunds
		WHERE store_id=$1 AND status IN ('pending','processing')
	`, storeID).Scan(&op.OpenRefunds)
	_ = s.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM campaigns
		WHERE store_id=$1 AND campaign_type='live'
		  AND publication_status='published'
		  AND operational_status='open'
		  AND opens_at <= now() AND (closes_at IS NULL OR closes_at > now())
	`, storeID).Scan(&op.LiveSessionsOpen)
	return nil
}

func (s *Service) fillRetention(ctx context.Context, out *Overview, storeID uuid.UUID, startUTC, endUTC time.Time) error {
	r := &out.Retention
	_ = s.DB.QueryRow(ctx, `
		WITH first_orders AS (
			SELECT customer_id, MIN(created_at) AS first_at
			FROM orders
			WHERE store_id=$1 AND order_status NOT IN ('cancelled') AND customer_id IS NOT NULL
			GROUP BY customer_id
		),
		in_range AS (
			SELECT DISTINCT customer_id
			FROM orders
			WHERE store_id=$1 AND order_status NOT IN ('cancelled') AND customer_id IS NOT NULL
			  AND created_at >= $2 AND created_at < $3
		)
		SELECT
			COUNT(*) FILTER (WHERE f.first_at >= $2 AND f.first_at < $3),
			COUNT(*) FILTER (WHERE f.first_at < $2)
		FROM in_range i
		JOIN first_orders f ON f.customer_id=i.customer_id
	`, storeID, startUTC, endUTC).Scan(&r.NewCustomers, &r.ReturningCustomers)
	total := r.NewCustomers + r.ReturningCustomers
	if total > 0 {
		r.RepeatOrderRate = float64(r.ReturningCustomers) / float64(total)
	}
	_ = s.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM (
			SELECT customer_id
			FROM orders
			WHERE store_id=$1 AND order_status NOT IN ('cancelled') AND customer_id IS NOT NULL
			GROUP BY customer_id
			HAVING COUNT(*) >= 2
		) t
	`, storeID).Scan(&r.CustomersWith2Plus)
	return nil
}

func (s *Service) fillEvents(ctx context.Context, out *Overview, storeID uuid.UUID, startUTC, endUTC time.Time) error {
	_ = s.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM analytics_events
		WHERE store_id=$1 AND occurred_at >= $2 AND occurred_at < $3
	`, storeID, startUTC, endUTC).Scan(&out.Events.TotalInRange)
	rows, err := s.DB.Query(ctx, `
		SELECT event_name, COUNT(*)
		FROM analytics_events
		WHERE store_id=$1 AND occurred_at >= $2 AND occurred_at < $3
		GROUP BY event_name
		ORDER BY COUNT(*) DESC
		LIMIT 20
	`, storeID, startUTC, endUTC)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var ec EventCount
		if err := rows.Scan(&ec.EventName, &ec.Count); err != nil {
			return err
		}
		out.Events.ByName = append(out.Events.ByName, ec)
	}
	return rows.Err()
}
