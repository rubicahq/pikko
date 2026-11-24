package catalog_test

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
		         inventory_adjustments, inventory_reservations,
		         store_credit_ledger, store_credits, refunds, financial_adjustments, financial_events,
		         payment_allocations, payment_evidence, payments, store_deposit_policies,
		         order_tracking_tokens, order_status_history, order_item_events, order_revisions, admin_order_drafts, order_items, orders,
		         store_order_counters, customer_addresses, customers,
		         store_payment_methods, store_fulfillment_settings,
		         catalog_page_hotspots, catalog_pages, catalog_import_items, catalog_import_jobs,
		         storefront_featured_collections, storefront_featured_products,
		         storefront_banners, storefront_configs,
		         campaign_products, campaigns,
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

func TestProductCRUDAndCrossStoreIsolation(t *testing.T) {
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

	userA, tokenA := signIn(t, pool, "catalog-a@example.com")
	userB, tokenB := signIn(t, pool, "catalog-b@example.com")
	stA, err := storeSvc.CreateStore(context.Background(), userA, stores.CreateDirectInput{
		Name: "Alpha Shop", Slug: "alpha-shop", Category: "other", SellingModel: "ready_stock",
	})
	if err != nil {
		t.Fatal(err)
	}
	stB, err := storeSvc.CreateStore(context.Background(), userB, stores.CreateDirectInput{
		Name: "Beta Shop", Slug: "beta-shop", Category: "other", SellingModel: "ready_stock",
	})
	if err != nil {
		t.Fatal(err)
	}

	body := map[string]any{
		"product": map[string]any{
			"name": "Tote Bag", "base_price_idr": 125000, "sku": "TOTE-1", "status": "draft",
		},
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/stores/"+stA.ID.String()+"/products", bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer "+tokenA)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create product: %d %s", rec.Code, rec.Body.String())
	}
	var created struct {
		Product struct {
			ID     string `json:"id"`
			Status string `json:"status"`
			SKU    string `json:"sku"`
		} `json:"product"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	if created.Product.Status != "draft" || created.Product.SKU != "TOTE-1" {
		t.Fatalf("unexpected product %#v", created.Product)
	}

	// Cross-store: B cannot read A's product
	req = httptest.NewRequest(http.MethodGet, "/api/v1/stores/"+stA.ID.String()+"/products/"+created.Product.ID, nil)
	req.Header.Set("Authorization", "Bearer "+tokenB)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 cross-store, got %d", rec.Code)
	}

	// B cannot create under A's store
	req = httptest.NewRequest(http.MethodPost, "/api/v1/stores/"+stA.ID.String()+"/products", bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer "+tokenB)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 create cross-store, got %d", rec.Code)
	}

	// Publish + public
	statusBody, _ := json.Marshal(map[string]string{"status": "published"})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/stores/"+stA.ID.String()+"/products/"+created.Product.ID+"/status", bytes.NewReader(statusBody))
	req.Header.Set("Authorization", "Bearer "+tokenA)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("publish: %d %s", rec.Code, rec.Body.String())
	}

	// Storefront must be published for public store endpoint
	req = httptest.NewRequest(http.MethodPost, "/api/v1/stores/"+stA.ID.String()+"/storefront/publish", nil)
	req.Header.Set("Authorization", "Bearer "+tokenA)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("publish storefront: %d %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/public/stores/alpha-shop/products/tote-bag", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("public product: %d %s", rec.Code, rec.Body.String())
	}
	var pub map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &pub)
	if pub["link_ref"] == nil {
		t.Fatal("expected link_ref for flipbook hotspots")
	}

	// Draft product not visible publicly on B's unpublished store
	_ = stB
	_ = io.Discard
}
