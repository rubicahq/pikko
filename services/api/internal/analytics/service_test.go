package analytics_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rubicahq/pikko/services/api/internal/analytics"
	"github.com/rubicahq/pikko/services/api/internal/auth"
	"github.com/rubicahq/pikko/services/api/internal/config"
	"github.com/rubicahq/pikko/services/api/internal/email"
	"github.com/rubicahq/pikko/services/api/internal/stores"
	"github.com/rubicahq/pikko/services/api/internal/urls"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = "postgres://pikko:pikko@localhost:5432/pikko_test?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Skip(err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		t.Skip(err)
	}
	return pool
}

func reset(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		TRUNCATE audit_logs, email_jobs, auth_rate_limits, public_rate_limits, onboarding_drafts, team_invites,
		         analytics_events, notifications, notification_preferences,
		         outbox_events, job_queue,
		         order_rule_applications, business_rule_versions, business_rules,
		         receiving_lines, receiving_records,
		         shipment_items, shipments, packing_task_items, packing_tasks,
		         preorder_item_allocations, preorder_purchasing_records,
		         inventory_adjustments, inventory_reservations,
		         store_credit_ledger, store_credits, refunds, financial_adjustments, financial_events,
		         payment_allocations, payment_evidence, payments, store_deposit_policies,
		         order_tracking_tokens, order_status_history, order_item_events, order_revisions, live_quick_adds, admin_order_drafts, order_items, orders,
		         store_order_counters, customer_addresses, customers,
		         store_payment_methods, store_fulfillment_settings,
		         campaign_products, campaigns,
		         preorder_batch_variants, preorder_batch_products, preorder_batches,
		         catalog_page_hotspots, catalog_pages, catalog_import_items, catalog_import_jobs,
		         storefront_featured_collections, storefront_featured_products,
		         storefront_banners, storefront_configs,
		         product_attribute_values, attribute_definitions,
		         product_vendors, vendors,
		         collection_products, collections,
		         variant_option_values, product_variants, product_option_values, product_options,
		         product_images, products, brands, categories, stored_files,
		         store_slug_aliases, store_memberships, stores, user_preferences,
		         sessions, authentication_tokens, users CASCADE
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func cfg() *config.Config {
	return &config.Config{
		AppEnv: "test", EmailMode: "log", EmailFromName: "Pikko",
		EmailAccountFrom: "account@pikko.site", EmailNotificationFrom: "notifications@pikko.site",
		EmailReplyTo: "support@pikko.site",
		MagicLinkTTL: 15 * time.Minute, ResendCooldown: time.Second,
		SessionTTL: 24 * time.Hour, RateLimitEmailHour: 100, RateLimitIPHour: 200,
		DefaultLocale: "id-ID", DefaultTZ: "Asia/Jakarta", DefaultCurr: "IDR",
		CookieName: "pikko_session", CORSOrigins: []string{"http://localhost:3000"},
		MarketingURL: "http://localhost:3000", DashboardURL: "http://localhost:3000",
		StorefrontURL: "http://localhost:3000", APIBaseURL: "http://localhost:8080",
	}
}

func signIn(t *testing.T, pool *pgxpool.Pool, emailAddr string) (uuid.UUID, string) {
	t.Helper()
	c := cfg()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mail := &email.Service{DB: pool, Sender: &email.LogSender{Logger: logger}, Logger: logger}
	svc := &auth.Service{DB: pool, Cfg: c, Email: mail, URLs: urls.New(c.MarketingURL, c.DashboardURL, c.StorefrontURL, c.APIBaseURL), Logger: logger}
	res, err := svc.RequestMagicLink(context.Background(), emailAddr, "127.0.0.1", "test")
	if err != nil {
		t.Fatal(err)
	}
	v, err := svc.VerifyAndConsume(context.Background(), res.DevToken, "127.0.0.1", "test")
	if err != nil {
		t.Fatal(err)
	}
	return v.User.ID, v.SessionToken
}

// TestAnalyticsNorthStarAndEmptyStates covers AN-01–04: fulfilled items from real rows, empty when none.
func TestAnalyticsNorthStarAndEmptyStates(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	reset(t, pool)

	c := cfg()
	storeSvc := &stores.Service{DB: pool, Cfg: c}
	svc := &analytics.Service{DB: pool}

	userID, _ := signIn(t, pool, "analytics-ops@example.com")
	st, err := storeSvc.CreateStore(context.Background(), userID, stores.CreateDirectInput{
		Name: "Analytics Mart", Slug: "analytics-mart", Category: "other", SellingModel: "both",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Empty store → zero north-star, empty arrays (AN-04)
	ov, err := svc.Overview(context.Background(), st.ID, "", "", "Asia/Jakarta")
	if err != nil {
		t.Fatal(err)
	}
	if ov.FulfilledOrderItems != 0 || ov.FulfilledOrderItemsAllTime != 0 {
		t.Fatalf("expected zero fulfilled items, got %#v", ov)
	}
	if ov.Daily == nil || ov.TopProducts == nil || ov.Events.ByName == nil {
		t.Fatalf("expected non-nil empty slices, got daily=%v products=%v events=%v", ov.Daily, ov.TopProducts, ov.Events.ByName)
	}
	if ov.TotalOrders != 0 || ov.VerifiedPaymentValueIDR != 0 {
		t.Fatalf("fabricated sales? %#v", ov)
	}

	custID := uuid.New()
	_, err = pool.Exec(context.Background(), `
		INSERT INTO customers (id, store_id, name, email) VALUES ($1,$2,'Buyer','buyer@example.com')
	`, custID, st.ID)
	if err != nil {
		t.Fatal(err)
	}

	orderID := uuid.New()
	itemID := uuid.New()
	now := time.Now().UTC()
	_, err = pool.Exec(context.Background(), `
		INSERT INTO orders (
			id, store_id, customer_id, order_number, idempotency_key, selling_model,
			order_status, payment_status, fulfillment_status, currency,
			subtotal_idr, total_idr, fulfillment_method,
			customer_name, customer_email, created_at, updated_at
		) VALUES (
			$1,$2,$3,'A-1001',$4,'ready_stock',
			'confirmed','paid','shipped','IDR',
			150000,150000,'shipping',
			'Buyer','buyer@example.com',$5,$5
		)
	`, orderID, st.ID, custID, "idem-analytics-1", now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(context.Background(), `
		INSERT INTO order_items (
			id, store_id, order_id, product_name, unit_price_idr, quantity, line_total_idr, created_at
		) VALUES ($1,$2,$3,'Fulfilled Tee',75000,2,150000,$4)
	`, itemID, st.ID, orderID, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(context.Background(), `
		INSERT INTO order_status_history (store_id, order_id, dimension, from_status, to_status, created_at)
		VALUES ($1,$2,'fulfillment','ready_to_ship','shipped',$3)
	`, st.ID, orderID, now)
	if err != nil {
		t.Fatal(err)
	}

	// Unpaid / unfulfilled order must not count toward north star
	unpaidID := uuid.New()
	_, err = pool.Exec(context.Background(), `
		INSERT INTO orders (
			id, store_id, customer_id, order_number, idempotency_key, selling_model,
			order_status, payment_status, fulfillment_status, currency,
			subtotal_idr, total_idr, fulfillment_method,
			customer_name, customer_email, created_at, updated_at
		) VALUES (
			$1,$2,$3,'A-1002',$4,'ready_stock',
			'confirmed','unpaid','unfulfilled','IDR',
			90000,90000,'shipping',
			'Buyer','buyer@example.com',$5,$5
		)
	`, unpaidID, st.ID, custID, "idem-analytics-2", now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(context.Background(), `
		INSERT INTO order_items (
			id, store_id, order_id, product_name, unit_price_idr, quantity, line_total_idr, created_at
		) VALUES ($1,$2,$3,'Pending Hat',90000,1,90000,$4)
	`, uuid.New(), st.ID, unpaidID, now)
	if err != nil {
		t.Fatal(err)
	}

	if err := svc.RecordEvent(context.Background(), st.ID, analytics.EventInput{
		EventName:  "payment.verified",
		Properties: map[string]any{"order_id": orderID.String()},
	}); err != nil {
		t.Fatal(err)
	}

	ov, err = svc.Overview(context.Background(), st.ID, "", "", "Asia/Jakarta")
	if err != nil {
		t.Fatal(err)
	}
	if ov.FulfilledOrderItems != 2 {
		t.Fatalf("AN-01 expected 2 fulfilled items, got %d", ov.FulfilledOrderItems)
	}
	if ov.FulfilledOrderItemValueIDR != 150000 {
		t.Fatalf("fulfilled value want 150000 got %d", ov.FulfilledOrderItemValueIDR)
	}
	if ov.FulfilledOrderItemsAllTime != 2 {
		t.Fatalf("all-time want 2 got %d", ov.FulfilledOrderItemsAllTime)
	}
	if ov.PaidOrders != 1 || ov.TotalOrders != 2 {
		t.Fatalf("orders paid=%d total=%d", ov.PaidOrders, ov.TotalOrders)
	}
	if ov.UnpaidOrderValueIDR != 90000 {
		t.Fatalf("unpaid value want 90000 got %d", ov.UnpaidOrderValueIDR)
	}
	if ov.Events.TotalInRange < 1 {
		t.Fatalf("expected recorded events, got %#v", ov.Events)
	}
	if !ov.Activation.HasOrder || !ov.Activation.HasPaidOrder || !ov.Activation.HasFulfilledItem {
		t.Fatalf("activation flags %#v", ov.Activation)
	}
	if ov.Retention.CustomersWith2Plus != 1 {
		t.Fatalf("want customers_with_2_plus=1 got %d", ov.Retention.CustomersWith2Plus)
	}

	// Cross-store isolation: other store stays empty
	other, err := storeSvc.CreateStore(context.Background(), userID, stores.CreateDirectInput{
		Name: "Other", Slug: "analytics-other", Category: "other", SellingModel: "both",
	})
	if err != nil {
		t.Fatal(err)
	}
	ov2, err := svc.Overview(context.Background(), other.ID, "", "", "Asia/Jakarta")
	if err != nil {
		t.Fatal(err)
	}
	if ov2.FulfilledOrderItems != 0 || ov2.TotalOrders != 0 {
		t.Fatalf("tenant leak into other store: %#v", ov2)
	}
}
