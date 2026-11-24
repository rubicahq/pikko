package whatsapp_test

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
		         whatsapp_template_sends,
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

func TestWhatsAppTemplatesFlow(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	reset(t, pool)

	dir := t.TempDir()
	objStore, err := storage.NewLocal(dir)
	if err != nil {
		t.Fatal(err)
	}
	c := cfg()
	router, _ := httpapi.NewRouter(httpapi.Deps{Cfg: c, DB: pool, Logger: slog.Default(), Storage: objStore})
	storeSvc := &stores.Service{DB: pool, Cfg: c}

	userID, token := signIn(t, pool, "wa-ops@example.com")
	st, err := storeSvc.CreateStore(context.Background(), userID, stores.CreateDirectInput{
		Name: "WA Mart", Slug: "wa-mart", Category: "other", SellingModel: "ready_stock",
	})
	if err != nil {
		t.Fatal(err)
	}
	sid := st.ID.String()

	code, list := doJSON(t, router, "GET", "/api/v1/stores/"+sid+"/whatsapp/templates", token, nil)
	if code != 200 {
		t.Fatalf("list %d %v", code, list)
	}
	if int(list["total"].(float64)) != 26 {
		t.Fatalf("total=%v", list["total"])
	}

	code, list = doJSON(t, router, "GET", "/api/v1/stores/"+sid+"/whatsapp/templates?audience=vendor", token, nil)
	if code != 200 || int(list["total"].(float64)) != 8 {
		t.Fatalf("vendor list %d %v", code, list)
	}

	code, _ = doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/payment-methods", token, map[string]any{
		"bank_name": "BCA", "account_holder": "WA", "account_number": "112233",
	})
	if code != 201 {
		t.Fatalf("payment %d", code)
	}
	code, _ = doJSON(t, router, "PATCH", "/api/v1/stores/"+sid+"/fulfillment", token, map[string]any{
		"shipping_enabled": true, "shipping_fee_idr": 10000,
	})
	if code != 200 {
		t.Fatalf("fulfillment %d", code)
	}
	code, prodBody := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/products", token, map[string]any{
		"product": map[string]any{
			"name": "WA Tee", "slug": "wa-tee", "sku": "WA-1", "base_price_idr": 50000,
			"ready_stock_eligible": true, "preorder_eligible": false, "inventory_mode": "unlimited",
		},
	})
	if code != 201 {
		t.Fatalf("product %d %v", code, prodBody)
	}
	productID := prodBody["product"].(map[string]any)["id"].(string)
	_, _ = doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/products/"+productID+"/status", token, map[string]any{"status": "published"})

	code, created := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/orders", token, map[string]any{
		"items":              []map[string]any{{"product_id": productID, "quantity": 1}},
		"customer_name":      "Budi WA",
		"customer_email":     "budi.wa@example.com",
		"customer_phone":     "081234567890",
		"fulfillment_method": "shipping",
		"shipping_address": map[string]any{
			"recipient_name": "Budi WA", "line1": "Jl. WA 1", "city": "Jakarta", "country": "ID",
		},
		"idempotency_key": uuid.NewString(),
	})
	if code != 201 {
		t.Fatalf("order %d %v", code, created)
	}
	orderID := created["order"].(map[string]any)["id"].(string)

	code, vendor := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/vendors", token, map[string]any{
		"name": "Vendor WA", "contact_name": "Kak Vendor", "contact_phone": "081111111111",
	})
	if code != 201 {
		t.Fatalf("vendor %d %v", code, vendor)
	}
	vendorID := vendor["vendor"].(map[string]any)["id"].(string)

	code, preview := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/whatsapp/templates/WA-C12/preview", token, map[string]any{
		"order_id": orderID,
		"overrides": map[string]string{
			"carrier": "JNE", "tracking_number": "JN123",
		},
	})
	if code != 200 {
		t.Fatalf("preview %d %v", code, preview)
	}
	body := preview["body"].(string)
	if !strings.Contains(body, "Budi WA") || !strings.Contains(body, "JN123") {
		t.Fatalf("body=%q", body)
	}
	link := preview["deep_link"].(string)
	if !strings.HasPrefix(link, "https://wa.me/6281234567890?text=") {
		t.Fatalf("link=%q", link)
	}

	code, open := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/whatsapp/templates/WA-V02/open", token, map[string]any{
		"vendor_id": vendorID,
		"action":    "open",
		"overrides": map[string]string{"item_summary": "- Tee x2", "amount_formatted": "Rp100.000", "batch_name": "Batch A"},
	})
	if code != 200 {
		t.Fatalf("open %d %v", code, open)
	}
	if !strings.Contains(open["deep_link"].(string), "6281111111111") {
		t.Fatalf("vendor link=%v", open["deep_link"])
	}

	code, sends := doJSON(t, router, "GET", "/api/v1/stores/"+sid+"/whatsapp/sends", token, nil)
	if code != 200 || int(sends["total"].(float64)) < 1 {
		t.Fatalf("sends %d %v", code, sends)
	}

	user2, token2 := signIn(t, pool, "wa-other@example.com")
	st2, err := storeSvc.CreateStore(context.Background(), user2, stores.CreateDirectInput{
		Name: "Other WA", Slug: "other-wa", Category: "other", SellingModel: "ready_stock",
	})
	if err != nil {
		t.Fatal(err)
	}
	code, bad := doJSON(t, router, "POST", "/api/v1/stores/"+st2.ID.String()+"/whatsapp/templates/WA-C01/preview", token2, map[string]any{
		"order_id": orderID,
	})
	if code != 404 {
		t.Fatalf("expected 404 cross-store, got %d %v", code, bad)
	}

	var emailJobs int
	if err := pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM email_jobs`).Scan(&emailJobs); err != nil {
		t.Fatal(err)
	}
}
