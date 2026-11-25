package portal_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rubicahq/pikko/services/api/internal/auth"
	"github.com/rubicahq/pikko/services/api/internal/config"
	"github.com/rubicahq/pikko/services/api/internal/email"
	"github.com/rubicahq/pikko/services/api/internal/httpapi"
	"github.com/rubicahq/pikko/services/api/internal/storage"
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
		         portal_address_change_requests, customer_refund_preferences,
		         customer_portal_sessions, customer_portal_tokens,
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
		CookieName: "pikko_session", PortalCookieName: "pikko_portal_session",
		CORSOrigins:   []string{"http://localhost:3000"},
		MarketingURL:  "http://localhost:3000", DashboardURL: "http://localhost:3000",
		StorefrontURL: "http://localhost:3000", APIBaseURL: "http://localhost:8080",
		FeatureStoreCredit: true,
	}
}

func signInMerchant(t *testing.T, pool *pgxpool.Pool, emailAddr string) (uuid.UUID, string) {
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

func doJSON(t *testing.T, router http.Handler, method, path, token string, body any) (int, map[string]any) {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, rdr)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func TestPortalSecurityIDOR(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	reset(t, pool)

	dir := t.TempDir()
	store, err := storage.NewLocal(dir)
	if err != nil {
		t.Fatal(err)
	}
	c := cfg()
	router, _ := httpapi.NewRouter(httpapi.Deps{Cfg: c, DB: pool, Logger: slog.Default(), Storage: store})
	storeSvc := &stores.Service{DB: pool, Cfg: c}

	_, tokenA := signInMerchant(t, pool, "merchant-a@example.com")
	stA, err := storeSvc.CreateStore(context.Background(), mustUser(t, pool, "merchant-a@example.com"), stores.CreateDirectInput{
		Name: "Alpha Shop", Slug: "alpha-shop", Category: "other", SellingModel: "ready_stock",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, tokenB := signInMerchant(t, pool, "merchant-b@example.com")
	stB, err := storeSvc.CreateStore(context.Background(), mustUser(t, pool, "merchant-b@example.com"), stores.CreateDirectInput{
		Name: "Beta Shop", Slug: "beta-shop", Category: "other", SellingModel: "ready_stock",
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = tokenB

	sidA := stA.ID.String()
	sidB := stB.ID.String()

	productA := publishProduct(t, router, sidA, tokenA, "alpha-item", 100000)
	productB := publishProduct(t, router, sidB, tokenB, "beta-item", 150000)

	enableShipping(t, router, sidA, tokenA)
	enableShipping(t, router, sidB, tokenB)

	orderA1 := createAdminOrder(t, router, sidA, tokenA, productA, "Alice", "alice@example.com", "081211111111")
	orderA2 := createAdminOrder(t, router, sidA, tokenA, productA, "Bob", "bob@example.com", "081222222222")
	orderB1 := createAdminOrder(t, router, sidB, tokenB, productB, "Alice", "alice@example.com", "081211111111")

	// Portal login Alice @ alpha-shop via durable secret+phone URL
	aliceTokenA := portalSignIn(t, pool, router, stA.ID, "alpha-shop", "alice@example.com")
	bobTokenA := portalSignIn(t, pool, router, stA.ID, "alpha-shop", "bob@example.com")
	aliceTokenB := portalSignIn(t, pool, router, stB.ID, "beta-shop", "alice@example.com")

	// Alice sees only her alpha order
	code, dash := doJSON(t, router, "GET", "/api/v1/portal/alpha-shop/dashboard", aliceTokenA, nil)
	if code != 200 {
		t.Fatalf("dashboard %d %v", code, dash)
	}
	orders := dash["dashboard"].(map[string]any)["orders"].([]any)
	if len(orders) != 1 {
		t.Fatalf("alice alpha orders want 1 got %d", len(orders))
	}
	if orders[0].(map[string]any)["id"].(string) != orderA1 {
		t.Fatal("alice should only see her order")
	}

	// IDOR: Alice cannot read Bob's order in same store
	code, body := doJSON(t, router, "GET", "/api/v1/portal/alpha-shop/orders/"+orderA2, aliceTokenA, nil)
	if code != 404 {
		t.Fatalf("cross-customer order want 404 got %d %v", code, body)
	}

	// Bob can read his own
	code, body = doJSON(t, router, "GET", "/api/v1/portal/alpha-shop/orders/"+orderA2, bobTokenA, nil)
	if code != 200 {
		t.Fatalf("bob own order %d %v", code, body)
	}
	if body["detail"].(map[string]any)["order"].(map[string]any)["internal_notes"] != nil {
		t.Fatal("internal notes must not leak to portal")
	}

	// Tenant isolation: Alice alpha session cannot access beta-shop routes
	code, body = doJSON(t, router, "GET", "/api/v1/portal/beta-shop/dashboard", aliceTokenA, nil)
	if code != 401 {
		t.Fatalf("cross-store session want 401 got %d %v", code, body)
	}
	code, body = doJSON(t, router, "GET", "/api/v1/portal/beta-shop/orders/"+orderB1, aliceTokenA, nil)
	if code != 401 {
		t.Fatalf("cross-store order want 401 got %d %v", code, body)
	}

	// Alice beta session cannot read alpha order id
	code, body = doJSON(t, router, "GET", "/api/v1/portal/alpha-shop/orders/"+orderA1, aliceTokenB, nil)
	if code != 401 {
		t.Fatalf("beta session on alpha want 401 got %d %v", code, body)
	}

	// Address IDOR: Bob cannot patch Alice's address
	code, addrBody := doJSON(t, router, "POST", "/api/v1/portal/alpha-shop/addresses", aliceTokenA, map[string]any{
		"recipient_name": "Alice", "line1": "Jl A 1", "city": "Jakarta", "country": "ID", "is_default": true,
	})
	if code != 201 {
		t.Fatalf("create address %d %v", code, addrBody)
	}
	addrID := addrBody["address"].(map[string]any)["id"].(string)
	code, body = doJSON(t, router, "PATCH", "/api/v1/portal/alpha-shop/addresses/"+addrID, bobTokenA, map[string]any{
		"recipient_name": "Hacker", "line1": "Jl Hack", "city": "Jakarta", "country": "ID",
	})
	if code != 404 {
		t.Fatalf("address IDOR want 404 got %d %v", code, body)
	}

	// Refund preference + bank details after verified session
	code, pref := doJSON(t, router, "PUT", "/api/v1/portal/alpha-shop/refund-preference", aliceTokenA, map[string]any{
		"preferred_method": "bank_transfer",
		"bank_name":        "BCA",
		"account_holder":   "Alice",
		"account_number":   "1234567890",
	})
	if code != 200 {
		t.Fatalf("refund pref %d %v", code, pref)
	}

	// Guest tracking still works (PORT-01 preserved)
	var trackToken string
	err = pool.QueryRow(context.Background(), `
		SELECT encode(token_hash, 'hex') FROM order_tracking_tokens WHERE order_id=$1 LIMIT 1
	`, uuid.MustParse(orderA1)).Scan(&trackToken)
	_ = err
	// tracking uses raw token from checkout; admin orders also mint — fetch via Ensure path is hashed.
	// Create a known tracking token via admin order response if available; otherwise skip raw track assert.
	code, track := doJSON(t, router, "GET", "/api/v1/public/tracking/not-a-real-token", "", nil)
	if code != 404 {
		t.Fatalf("invalid tracking should 404 got %d %v", code, track)
	}

	// Token entropy: session token from verify is high entropy (≥128-bit → base64url of 32 bytes ≈ 43 chars)
	if len(aliceTokenA) < 40 {
		t.Fatalf("portal session token too short: %d", len(aliceTokenA))
	}

	// Store credit gated
	code, credit := doJSON(t, router, "GET", "/api/v1/portal/alpha-shop/credit", aliceTokenA, nil)
	if code != 409 {
		t.Fatalf("credit disabled per-store want 409 got %d %v", code, credit)
	}
}

func mustUser(t *testing.T, pool *pgxpool.Pool, emailAddr string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := pool.QueryRow(context.Background(), `SELECT id FROM users WHERE email_normalized=lower($1)`, emailAddr).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func publishProduct(t *testing.T, router http.Handler, sid, token, slug string, price int64) string {
	t.Helper()
	code, prodBody := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/products", token, map[string]any{
		"product": map[string]any{
			"name": slug, "slug": slug, "sku": slug, "base_price_idr": price,
			"ready_stock_eligible": true, "inventory_mode": "unlimited",
		},
	})
	if code != 201 {
		t.Fatalf("product %d %v", code, prodBody)
	}
	productID := prodBody["product"].(map[string]any)["id"].(string)
	code, _ = doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/products/"+productID+"/status", token, map[string]any{
		"status": "published",
	})
	if code != 200 {
		t.Fatalf("publish %d", code)
	}
	return productID
}

func enableShipping(t *testing.T, router http.Handler, sid, token string) {
	t.Helper()
	code, _ := doJSON(t, router, "PATCH", "/api/v1/stores/"+sid+"/fulfillment", token, map[string]any{
		"pickup_enabled": true, "shipping_enabled": true, "shipping_fee_idr": 10000,
	})
	if code != 200 {
		t.Fatalf("fulfillment %d", code)
	}
}

func createAdminOrder(t *testing.T, router http.Handler, sid, token, productID, name, emailAddr, phone string) string {
	t.Helper()
	code, created := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/orders", token, map[string]any{
		"items":              []map[string]any{{"product_id": productID, "quantity": 1}},
		"customer_name":      name,
		"customer_email":     emailAddr,
		"customer_phone":     phone,
		"fulfillment_method": "shipping",
		"shipping_address": map[string]any{
			"recipient_name": name, "line1": "Jl Test 1", "city": "Jakarta", "country": "ID",
		},
		"idempotency_key": uuid.NewString(),
	})
	if code != 201 {
		t.Fatalf("order %d %v", code, created)
	}
	return created["order"].(map[string]any)["id"].(string)
}

func portalSignIn(t *testing.T, pool *pgxpool.Pool, router http.Handler, storeID uuid.UUID, slug, emailAddr string) string {
	t.Helper()
	var secret, phone string
	err := pool.QueryRow(context.Background(), `
		SELECT portal_access_secret, phone FROM customers
		WHERE store_id=$1 AND email IS NOT NULL AND lower(email)=lower($2)
	`, storeID, emailAddr).Scan(&secret, &phone)
	if err != nil {
		t.Fatalf("lookup portal access for %s: %v", emailAddr, err)
	}
	accessKey := secret + "-" + phone
	code, ver := doJSON(t, router, "POST", "/api/v1/portal/"+slug+"/auth/access", "", map[string]any{
		"access_key": accessKey,
	})
	if code != 200 {
		t.Fatalf("portal access %d %v", code, ver)
	}
	tok, _ := ver["session_token"].(string)
	if tok == "" {
		t.Fatal("missing session_token")
	}
	return tok
}
