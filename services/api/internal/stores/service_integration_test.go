package stores_test

import (
	"context"
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
		         payment_evidence, payments, order_tracking_tokens, order_status_history, order_items, orders,
		         store_order_counters, customer_addresses, customers,
		         store_payment_methods, store_fulfillment_settings,
		         catalog_page_hotspots, catalog_pages, catalog_import_items, catalog_import_jobs,
		         storefront_featured_collections, storefront_featured_products,
		         storefront_banners, storefront_configs,
		         campaign_products, campaigns,
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

func TestCreateStoreAndCrossStoreDenied(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	reset(t, pool)
	c := cfg()
	storeSvc := &stores.Service{DB: pool, Cfg: c}

	userA, tokenA := signIn(t, pool, "a@example.com")
	userB, tokenB := signIn(t, pool, "b@example.com")

	stA, err := storeSvc.CreateStore(context.Background(), userA, stores.CreateDirectInput{
		Name: "Store A", Slug: "store-a", Category: "other", SellingModel: "ready_stock",
	})
	if err != nil {
		t.Fatal(err)
	}
	stB, err := storeSvc.CreateStore(context.Background(), userB, stores.CreateDirectInput{
		Name: "Store B", Slug: "store-b", Category: "other", SellingModel: "preorder",
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := storeSvc.AssertMember(context.Background(), userA, stB.ID); err == nil {
		t.Fatal("user A must not access store B")
	}
	if err := storeSvc.AssertMember(context.Background(), userB, stA.ID); err == nil {
		t.Fatal("user B must not access store A")
	}

	local, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	router, _ := httpapi.NewRouter(httpapi.Deps{Cfg: c, DB: pool, Logger: slog.Default(), Storage: local})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stores/"+stB.ID.String(), nil)
	req.AddCookie(&http.Cookie{Name: c.CookieName, Value: tokenA})
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("cross-store GET status=%d body=%s", rr.Code, rr.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/stores/"+stA.ID.String(), nil)
	req2.AddCookie(&http.Cookie{Name: c.CookieName, Value: tokenA})
	rr2 := httptest.NewRecorder()
	router.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("owner GET status=%d", rr2.Code)
	}

	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/stores/"+stA.ID.String()+"/archive", nil)
	req3.AddCookie(&http.Cookie{Name: c.CookieName, Value: tokenB})
	rr3 := httptest.NewRecorder()
	router.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusNotFound {
		t.Fatalf("cross-store archive status=%d", rr3.Code)
	}

	list, err := storeSvc.ListForUser(context.Background(), userA, false)
	if err != nil || len(list) != 1 || list[0].ID != stA.ID {
		t.Fatalf("list isolation failed: %+v err=%v", list, err)
	}
	_ = tokenB
}

func TestOnboardingResumeAndIdempotentCreate(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	reset(t, pool)
	c := cfg()
	storeSvc := &stores.Service{DB: pool, Cfg: c}
	userID, _ := signIn(t, pool, "onboard@example.com")

	d, err := storeSvc.GetOrCreateDraft(context.Background(), userID)
	if err != nil || d.Step != 1 {
		t.Fatalf("draft: %+v err=%v", d, err)
	}
	_, err = storeSvc.SaveIdentityStep(context.Background(), userID, stores.IdentityInput{
		Name: "Japan Picks", Slug: "japanpicks", Category: "personal_shopper_jastip",
	})
	if err != nil {
		t.Fatal(err)
	}
	d2, err := storeSvc.GetOrCreateDraft(context.Background(), userID)
	if err != nil || d2.Step != 2 || d2.StoreSlug == nil || *d2.StoreSlug != "japanpicks" {
		t.Fatalf("resume failed: %+v", d2)
	}
	_, err = storeSvc.SaveSellingModelStep(context.Background(), userID, "both")
	if err != nil {
		t.Fatal(err)
	}
	st, err := storeSvc.CreateStoreFromDraft(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = storeSvc.CreateStoreFromDraft(context.Background(), userID)
	if err == nil {
		t.Fatal("second create should fail without draft")
	}
	if st.Theme != "pikko_signature" || st.PublicationStatus != "draft" || st.CheckoutEnabled {
		t.Fatalf("defaults wrong: %+v", st)
	}
}
