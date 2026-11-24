package campaigns_test

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
		         preorder_item_allocations, preorder_purchasing_records,
		         inventory_adjustments, inventory_reservations,
		         payment_evidence, payments, order_tracking_tokens, order_status_history, order_item_events, order_revisions, admin_order_drafts, order_items, orders,
		         store_order_counters, customer_addresses, customers,
		         store_payment_methods, store_fulfillment_settings,
		         campaign_products, campaigns,
		         preorder_batch_variants, preorder_batch_products, preorder_batches,
		         catalog_page_hotspots, catalog_pages, catalog_import_items, catalog_import_jobs,
		         storefront_featured_collections, storefront_featured_products,
		         storefront_banners, storefront_configs,
		         product_attribute_values, attribute_definitions,
		         receiving_lines, receiving_records,
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
		EmailAccountFrom: "account@pikko.site", EmailReplyTo: "support@pikko.site",
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
		raw, _ := json.Marshal(body)
		rdr = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, rdr)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func TestCampaignPreorderUnificationAndCatalogFoundations(t *testing.T) {
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

	userA, tokenA := signIn(t, pool, "camp-a@example.com")
	userB, tokenB := signIn(t, pool, "camp-b@example.com")
	stA, err := storeSvc.CreateStore(context.Background(), userA, stores.CreateDirectInput{
		Name: "Camp A", Slug: "camp-a", Category: "other", SellingModel: "both",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = storeSvc.CreateStore(context.Background(), userB, stores.CreateDirectInput{
		Name: "Camp B", Slug: "camp-b", Category: "other", SellingModel: "preorder",
	})
	if err != nil {
		t.Fatal(err)
	}

	opens := time.Now().UTC().Add(-time.Hour)
	closes := time.Now().UTC().Add(48 * time.Hour)
	code, campBody := doJSON(t, router, "POST", "/api/v1/stores/"+stA.ID.String()+"/campaigns", tokenA, map[string]any{
		"campaign_type": "preorder", "name": "Japan Drop", "slug": "japan-drop",
		"opens_at": opens.Format(time.RFC3339), "closes_at": closes.Format(time.RFC3339),
		"publication_status": "published", "eta_start": "2026-12-01", "eta_end": "2026-12-15",
	})
	if code != 201 {
		t.Fatalf("create campaign %d %v", code, campBody)
	}
	camp := campBody["campaign"].(map[string]any)
	if camp["campaign_type"] != "preorder" {
		t.Fatalf("type %#v", camp["campaign_type"])
	}
	batchID, _ := camp["preorder_batch_id"].(string)
	if batchID == "" {
		t.Fatal("expected linked preorder_batch_id")
	}
	campaignID := camp["id"].(string)

	// Legacy preorder list sees the batch
	code, batches := doJSON(t, router, "GET", "/api/v1/stores/"+stA.ID.String()+"/preorders", tokenA, nil)
	if code != 200 {
		t.Fatalf("list batches %d", code)
	}
	list := batches["batches"].([]any)
	if len(list) != 1 {
		t.Fatalf("expected 1 batch, got %d", len(list))
	}

	// Cross-store isolation
	code, _ = doJSON(t, router, "GET", "/api/v1/stores/"+stA.ID.String()+"/campaigns/"+campaignID, tokenB, nil)
	if code != 403 && code != 404 {
		t.Fatalf("cross-store get want 403/404 got %d", code)
	}

	// Live campaign shell
	code, liveBody := doJSON(t, router, "POST", "/api/v1/stores/"+stA.ID.String()+"/campaigns", tokenA, map[string]any{
		"campaign_type": "live", "name": "Friday Live", "slug": "friday-live",
		"opens_at": opens.Format(time.RFC3339), "closes_at": closes.Format(time.RFC3339),
		"publication_status": "draft", "live_quick_add_enabled": true,
	})
	if code != 201 {
		t.Fatalf("live campaign %d %v", code, liveBody)
	}
	live := liveBody["campaign"].(map[string]any)
	if live["preorder_batch_id"] != nil {
		t.Fatal("live must not link batch")
	}
	if live["live_quick_add_enabled"] != true {
		t.Fatal("quick add flag")
	}

	// Brand + vendor + attribute foundations
	code, brandBody := doJSON(t, router, "POST", "/api/v1/stores/"+stA.ID.String()+"/brands", tokenA, map[string]any{
		"name": "Muji", "slug": "muji",
	})
	if code != 201 {
		t.Fatalf("brand %d %v", code, brandBody)
	}
	brandID := brandBody["brand"].(map[string]any)["id"].(string)

	code, vendorBody := doJSON(t, router, "POST", "/api/v1/stores/"+stA.ID.String()+"/vendors", tokenA, map[string]any{
		"name": "Tokyo Supplier", "contact_email": "buy@tokyo.example",
	})
	if code != 201 {
		t.Fatalf("vendor %d %v", code, vendorBody)
	}
	vendorID := vendorBody["vendor"].(map[string]any)["id"].(string)

	code, attrBody := doJSON(t, router, "POST", "/api/v1/stores/"+stA.ID.String()+"/attributes", tokenA, map[string]any{
		"key": "material", "label": "Material", "data_type": "text",
	})
	if code != 201 {
		t.Fatalf("attr %d %v", code, attrBody)
	}
	attrID := attrBody["attribute"].(map[string]any)["id"].(string)

	code, prodBody := doJSON(t, router, "POST", "/api/v1/stores/"+stA.ID.String()+"/products", tokenA, map[string]any{
		"product": map[string]any{
			"name": "Canvas Tote", "base_price_idr": 150000, "brand_id": brandID,
			"status": "published", "preorder_eligible": true,
		},
	})
	if code != 201 {
		t.Fatalf("product %d %v", code, prodBody)
	}
	productID := prodBody["product"].(map[string]any)["id"].(string)

	code, _ = doJSON(t, router, "POST", "/api/v1/stores/"+stA.ID.String()+"/products/"+productID+"/vendors", tokenA, map[string]any{
		"vendor_id": vendorID, "is_preferred": true,
	})
	if code != 200 {
		t.Fatalf("link vendor %d", code)
	}
	code, _ = doJSON(t, router, "PUT", "/api/v1/stores/"+stA.ID.String()+"/products/"+productID+"/attributes", tokenA, map[string]any{
		"attribute_id": attrID, "value_text": "Canvas",
	})
	if code != 200 {
		t.Fatalf("set attr %d", code)
	}

	code, assignBody := doJSON(t, router, "POST", "/api/v1/stores/"+stA.ID.String()+"/campaigns/"+campaignID+"/products", tokenA, map[string]any{
		"product_id": productID, "price_override_idr": 140000, "availability": "available",
		"selling_phase": "preorder", "display_order": 1,
	})
	if code != 200 {
		t.Fatalf("assign %d %v", code, assignBody)
	}

	// Mirrored onto preorder_batch_products
	var mirrored int
	err = pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM preorder_batch_products WHERE batch_id=$1 AND product_id=$2
	`, batchID, productID).Scan(&mirrored)
	if err != nil || mirrored != 1 {
		t.Fatalf("mirror batch product: err=%v count=%d", err, mirrored)
	}

	// Legacy CreateBatch also links a campaign
	code, legacy := doJSON(t, router, "POST", "/api/v1/stores/"+stA.ID.String()+"/preorders", tokenA, map[string]any{
		"name": "Legacy Batch", "slug": "legacy-batch",
		"opens_at": opens.Format(time.RFC3339), "closes_at": closes.Format(time.RFC3339),
		"publication_status": "draft",
	})
	if code != 201 {
		t.Fatalf("legacy batch %d %v", code, legacy)
	}
	legacyBatchID := legacy["batch"].(map[string]any)["id"].(string)
	var linked uuid.UUID
	err = pool.QueryRow(context.Background(), `
		SELECT campaign_id FROM preorder_batches WHERE id=$1
	`, legacyBatchID).Scan(&linked)
	if err != nil {
		t.Fatalf("legacy campaign link: %v", err)
	}

	code, all := doJSON(t, router, "GET", "/api/v1/stores/"+stA.ID.String()+"/campaigns", tokenA, nil)
	if code != 200 {
		t.Fatalf("list campaigns %d", code)
	}
	camps := all["campaigns"].([]any)
	if len(camps) < 3 {
		t.Fatalf("expected >=3 campaigns, got %d", len(camps))
	}
}
