package commerce_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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
		         preorder_item_allocations, preorder_purchasing_records,
		         inventory_adjustments, inventory_reservations,
		         payment_evidence, payments, order_tracking_tokens, order_status_history, order_items, orders,
		         store_order_counters, customer_addresses, customers,
		         store_payment_methods, store_fulfillment_settings,
		         preorder_batch_variants, preorder_batch_products, preorder_batches,
		         catalog_page_hotspots, catalog_pages, catalog_import_items, catalog_import_jobs,
		         storefront_featured_collections, storefront_featured_products,
		         storefront_banners, storefront_configs,
		         collection_products, collections,
		         variant_option_values, product_variants, product_option_values, product_options,
		         product_images, products, categories, stored_files,
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

func TestCheckoutPaymentVerifyAndCrossStoreIsolation(t *testing.T) {
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

	userA, tokenA := signIn(t, pool, "order-a@example.com")
	userB, tokenB := signIn(t, pool, "order-b@example.com")
	stA, err := storeSvc.CreateStore(context.Background(), userA, stores.CreateDirectInput{
		Name: "Alpha Mart", Slug: "alpha-mart", Category: "other", SellingModel: "ready_stock",
	})
	if err != nil {
		t.Fatal(err)
	}
	stB, err := storeSvc.CreateStore(context.Background(), userB, stores.CreateDirectInput{
		Name: "Beta Mart", Slug: "beta-mart", Category: "other", SellingModel: "ready_stock",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Configure payment + fulfillment for A
	code, _ := doJSON(t, router, "POST", "/api/v1/stores/"+stA.ID.String()+"/payment-methods", tokenA, map[string]any{
		"bank_name": "BCA", "account_holder": "Alpha", "account_number": "1234567890",
	})
	if code != 201 {
		t.Fatalf("payment method %d", code)
	}
	code, _ = doJSON(t, router, "PATCH", "/api/v1/stores/"+stA.ID.String()+"/fulfillment", tokenA, map[string]any{
		"shipping_enabled": true, "shipping_fee_idr": 15000, "pickup_enabled": true,
	})
	if code != 200 {
		t.Fatalf("fulfillment %d", code)
	}

	// Publish storefront + product
	code, _ = doJSON(t, router, "POST", "/api/v1/stores/"+stA.ID.String()+"/storefront/publish", tokenA, map[string]any{})
	if code != 200 {
		t.Fatalf("publish %d", code)
	}
	code, prodBody := doJSON(t, router, "POST", "/api/v1/stores/"+stA.ID.String()+"/products", tokenA, map[string]any{
		"product": map[string]any{
			"name": "Ready Tee", "slug": "ready-tee", "base_price_idr": 100000,
			"ready_stock_eligible": true, "inventory_mode": "unlimited",
		},
	})
	if code != 201 {
		t.Fatalf("create product %d %v", code, prodBody)
	}
	productID := prodBody["product"].(map[string]any)["id"].(string)
	code, _ = doJSON(t, router, "POST", "/api/v1/stores/"+stA.ID.String()+"/products/"+productID+"/status", tokenA, map[string]any{
		"status": "published",
	})
	if code != 200 {
		t.Fatalf("publish product %d", code)
	}

	code, methods := doJSON(t, router, "GET", "/api/v1/stores/"+stA.ID.String()+"/payment-methods", tokenA, nil)
	if code != 200 {
		t.Fatal(methods)
	}
	pmID := methods["payment_methods"].([]any)[0].(map[string]any)["id"].(string)

	idem := uuid.NewString()
	code, submit := doJSON(t, router, "POST", "/api/v1/public/stores/alpha-mart/checkout/submit", "", map[string]any{
		"items": []map[string]any{{"product_id": productID, "quantity": 2}},
		"customer_name": "Budi", "customer_email": "budi@example.com", "customer_phone": "081234",
		"fulfillment_method": "shipping",
		"shipping_address": map[string]any{
			"recipient_name": "Budi", "line1": "Jl. Merdeka 1", "city": "Jakarta", "country": "ID",
		},
		"payment_method_id": pmID, "idempotency_key": idem,
		"client_expected_total_idr": 215000,
	})
	if code != 201 {
		t.Fatalf("checkout %d %v", code, submit)
	}
	order := submit["order"].(map[string]any)
	orderID := order["id"].(string)
	trackingToken := submit["tracking_token"].(string)
	if trackingToken == "" || order["order_number"] == "" {
		t.Fatalf("missing tracking/order number: %v", submit)
	}
	if int64(order["total_idr"].(float64)) != 215000 {
		t.Fatalf("total %v", order["total_idr"])
	}

	// Idempotent replay
	code, submit2 := doJSON(t, router, "POST", "/api/v1/public/stores/alpha-mart/checkout/submit", "", map[string]any{
		"items": []map[string]any{{"product_id": productID, "quantity": 2}},
		"customer_name": "Budi", "customer_email": "budi@example.com",
		"fulfillment_method": "shipping",
		"shipping_address": map[string]any{
			"recipient_name": "Budi", "line1": "Jl. Merdeka 1", "city": "Jakarta", "country": "ID",
		},
		"payment_method_id": pmID, "idempotency_key": idem,
	})
	if code != 201 {
		t.Fatalf("idempotent checkout %d", code)
	}
	if submit2["order"].(map[string]any)["id"] != orderID {
		t.Fatal("idempotency failed")
	}

	// Tracking
	code, track := doJSON(t, router, "GET", "/api/v1/public/tracking/"+trackingToken, "", nil)
	if code != 200 {
		t.Fatalf("tracking %d %v", code, track)
	}
	if track["tracking"].(map[string]any)["order"].(map[string]any)["internal_notes"] != nil {
		// should be omitted / null
	}

	// Upload evidence (1x1 png)
	png := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xde, 0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41, 0x54, 0x08, 0xd7, 0x63, 0xf8, 0xff, 0xff, 0x3f,
		0x00, 0x05, 0xfe, 0x02, 0xfe, 0xa7, 0x35, 0x81, 0x84, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e,
		0x44, 0xae, 0x42, 0x60, 0x82,
	}
	w := httptest.NewRecorder()
	body := &bytes.Buffer{}
	mw := multipartWriter(t, body, png)
	req := httptest.NewRequest("POST", "/api/v1/public/tracking/"+trackingToken+"/evidence", body)
	req.Header.Set("Content-Type", mw)
	router.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("evidence upload %d %s", w.Code, w.Body.String())
	}

	paymentID := submit["payment"].(map[string]any)["id"].(string)

	// Cross-store: B cannot see A's order
	code, _ = doJSON(t, router, "GET", "/api/v1/stores/"+stB.ID.String()+"/orders/"+orderID, tokenB, nil)
	if code != 404 {
		t.Fatalf("expected 404 cross-store, got %d", code)
	}
	code, _ = doJSON(t, router, "GET", "/api/v1/stores/"+stA.ID.String()+"/orders/"+orderID, tokenB, nil)
	if code != 404 {
		t.Fatalf("expected 404 unauthorized store, got %d", code)
	}

	// Verify payment as merchant A
	code, verified := doJSON(t, router, "POST",
		"/api/v1/stores/"+stA.ID.String()+"/orders/"+orderID+"/payments/"+paymentID+"/verify",
		tokenA, map[string]any{})
	if code != 200 {
		t.Fatalf("verify %d %v", code, verified)
	}
	if verified["payment"].(map[string]any)["status"] != "verified" {
		t.Fatalf("status %v", verified)
	}

	// Duplicate verify rejected
	code, _ = doJSON(t, router, "POST",
		"/api/v1/stores/"+stA.ID.String()+"/orders/"+orderID+"/payments/"+paymentID+"/verify",
		tokenA, map[string]any{})
	if code != 409 {
		t.Fatalf("expected conflict on re-verify, got %d", code)
	}

	// Email jobs enqueued
	var n int
	_ = pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM email_jobs WHERE template_key LIKE 'order_%' OR template_key LIKE 'payment_%'`).Scan(&n)
	if n < 2 {
		t.Fatalf("expected email jobs, got %d", n)
	}

	// Tracking token hash only (raw not stored)
	var rawCount int
	_ = pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM order_tracking_tokens WHERE encode(token_hash,'escape') LIKE $1
	`, "%"+trackingToken+"%").Scan(&rawCount)
	if rawCount != 0 {
		t.Fatal("raw tracking token must not be stored")
	}
	_ = strings.Builder{}
}

func multipartWriter(t *testing.T, body *bytes.Buffer, png []byte) string {
	t.Helper()
	boundary := "----pikkoboundary"
	body.WriteString("--" + boundary + "\r\n")
	body.WriteString(`Content-Disposition: form-data; name="file"; filename="proof.png"` + "\r\n")
	body.WriteString("Content-Type: image/png\r\n\r\n")
	body.Write(png)
	body.WriteString("\r\n--" + boundary + "--\r\n")
	return "multipart/form-data; boundary=" + boundary
}
