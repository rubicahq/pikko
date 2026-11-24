package flipbook_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rubicahq/pikko/services/api/internal/ai"
	"github.com/rubicahq/pikko/services/api/internal/auth"
	"github.com/rubicahq/pikko/services/api/internal/config"
	"github.com/rubicahq/pikko/services/api/internal/email"
	"github.com/rubicahq/pikko/services/api/internal/httpapi"
	"github.com/rubicahq/pikko/services/api/internal/pdfutil"
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
		ImportMaxPDFBytes: 25 << 20, ImportMaxPages: 50, ImportMaxItems: 500,
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

func TestFlipbookPublishAndPublicHotspots(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	reset(t, pool)

	dir := t.TempDir()
	st, err := storage.NewLocal(dir)
	if err != nil {
		t.Fatal(err)
	}
	router, worker := httpapi.NewRouter(httpapi.Deps{
		Cfg: cfg(), DB: pool, Logger: slog.Default(), Storage: st,
		AI: ai.NewMockExtractor(), Renderer: &pdfutil.PlaceholderRenderer{Width: 60, Height: 80},
	})

	uidA, tokA := signIn(t, pool, "flip-a@example.com")
	uidB, tokB := signIn(t, pool, "flip-b@example.com")
	storeSvc := &stores.Service{DB: pool, Cfg: cfg()}
	stA, err := storeSvc.CreateStore(context.Background(), uidA, stores.CreateDirectInput{
		Name: "Flip A", Slug: "flip-a", Category: "other", SellingModel: "ready_stock",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = storeSvc.CreateStore(context.Background(), uidB, stores.CreateDirectInput{
		Name: "Flip B", Slug: "flip-b", Category: "other", SellingModel: "ready_stock",
	})
	if err != nil {
		t.Fatal(err)
	}

	body, _ := json.Marshal(map[string]any{
		"product": map[string]any{"name": "Hotspot Mug", "base_price_idr": 99000, "status": "draft"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/stores/"+stA.ID.String()+"/products", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tokA)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != 201 {
		t.Fatalf("create product %d %s", rr.Code, rr.Body.String())
	}
	var created struct {
		Product struct {
			ID string `json:"id"`
		} `json:"product"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &created)

	statusBody, _ := json.Marshal(map[string]string{"status": "published"})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/stores/"+stA.ID.String()+"/products/"+created.Product.ID+"/status", bytes.NewReader(statusBody))
	req.Header.Set("Authorization", "Bearer "+tokA)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("publish product %d", rr.Code)
	}

	pdf := pdfutil.MinimalPDF()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("extract_images", "true")
	fw, _ := mw.CreateFormFile("file", "c.pdf")
	_, _ = fw.Write(pdf)
	_ = mw.Close()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/stores/"+stA.ID.String()+"/catalog-imports", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+tokA)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != 202 {
		t.Fatalf("import %d %s", rr.Code, rr.Body.String())
	}
	for i := 0; i < 5; i++ {
		ok, err := worker.ProcessOnce(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			break
		}
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/stores/"+stA.ID.String()+"/catalog-pages", nil)
	req.Header.Set("Authorization", "Bearer "+tokA)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	var pagesResp struct {
		Pages []struct {
			ID string `json:"id"`
		} `json:"pages"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &pagesResp)
	if len(pagesResp.Pages) < 1 {
		t.Fatal("expected catalog pages from import")
	}
	pageID := pagesResp.Pages[0].ID

	req = httptest.NewRequest(http.MethodGet, "/api/v1/stores/"+stA.ID.String()+"/catalog-pages", nil)
	req.Header.Set("Authorization", "Bearer "+tokB)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != 404 {
		t.Fatalf("cross-store pages %d", rr.Code)
	}

	hsBody, _ := json.Marshal(map[string]any{
		"product_id": created.Product.ID,
		"x_pct":      10, "y_pct": 10, "width_pct": 20, "height_pct": 15,
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/stores/"+stA.ID.String()+"/catalog-pages/"+pageID+"/hotspots", bytes.NewReader(hsBody))
	req.Header.Set("Authorization", "Bearer "+tokA)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != 201 {
		t.Fatalf("create hotspot %d %s", rr.Code, rr.Body.String())
	}

	pubBody, _ := json.Marshal(map[string]any{"status": "published"})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/stores/"+stA.ID.String()+"/catalog-pages/status", bytes.NewReader(pubBody))
	req.Header.Set("Authorization", "Bearer "+tokA)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("publish pages %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/stores/"+stA.ID.String()+"/storefront/publish", nil)
	req.Header.Set("Authorization", "Bearer "+tokA)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("publish storefront %d %s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/public/stores/flip-a/catalog", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("public catalog %d %s", rr.Code, rr.Body.String())
	}
	var pub struct {
		Pages []struct {
			Hotspots []struct {
				ProductID string `json:"product_id"`
			} `json:"hotspots"`
		} `json:"pages"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &pub)
	if len(pub.Pages) < 1 || len(pub.Pages[0].Hotspots) < 1 {
		t.Fatalf("expected public hotspot, got %+v", pub)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/public/stores/flip-a/catalog/products/"+created.Product.ID, nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("public hotspot product %d", rr.Code)
	}
}
