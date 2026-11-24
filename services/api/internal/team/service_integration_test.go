package team_test

import (
	"bytes"
	"context"
	"encoding/json"
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
	"github.com/rubicahq/pikko/services/api/internal/rbac"
	"github.com/rubicahq/pikko/services/api/internal/storage"
	"github.com/rubicahq/pikko/services/api/internal/stores"
	"github.com/rubicahq/pikko/services/api/internal/team"
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

func TestInviteAcceptAndTenantIsolation(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	reset(t, pool)
	c := cfg()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mail := &email.Service{DB: pool, Sender: &email.LogSender{Logger: logger}, Logger: logger}
	urlGen := urls.New(c.MarketingURL, c.DashboardURL, c.StorefrontURL, c.APIBaseURL)
	storeSvc := &stores.Service{DB: pool, Cfg: c}
	teamSvc := &team.Service{DB: pool, Cfg: c, Email: mail, URLs: urlGen, Logger: logger}

	ownerA, tokenA := signIn(t, pool, "owner-a@example.com")
	ownerB, tokenB := signIn(t, pool, "owner-b@example.com")

	stA, err := storeSvc.CreateStore(context.Background(), ownerA, stores.CreateDirectInput{
		Name: "Store A", Slug: "store-a-rbac", Category: "other", SellingModel: "ready_stock",
	})
	if err != nil {
		t.Fatal(err)
	}
	stB, err := storeSvc.CreateStore(context.Background(), ownerB, stores.CreateDirectInput{
		Name: "Store B", Slug: "store-b-rbac", Category: "other", SellingModel: "ready_stock",
	})
	if err != nil {
		t.Fatal(err)
	}

	inv, err := teamSvc.CreateInvite(context.Background(), stA.ID, ownerA, team.CreateInviteInput{
		Email: "staff-a@example.com", Role: rbac.RoleCatalog,
	})
	if err != nil || inv.DevToken == "" {
		t.Fatalf("invite: %+v err=%v", inv, err)
	}

	accepted, err := teamSvc.AcceptInvite(context.Background(), inv.DevToken, "127.0.0.1", "test")
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Role != rbac.RoleCatalog || accepted.StoreID != stA.ID {
		t.Fatalf("accept result: %+v", accepted)
	}
	staffToken := accepted.SessionToken

	local, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	router, _ := httpapi.NewRouter(httpapi.Deps{Cfg: c, DB: pool, Logger: slog.Default(), Storage: local})

	// Staff of A can read catalog of A
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stores/"+stA.ID.String()+"/products", nil)
	req.AddCookie(&http.Cookie{Name: c.CookieName, Value: staffToken})
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("staff catalog A status=%d body=%s", rr.Code, rr.Body.String())
	}

	// Staff of A cannot access Store B (tenant isolation)
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/stores/"+stB.ID.String()+"/products", nil)
	req2.AddCookie(&http.Cookie{Name: c.CookieName, Value: staffToken})
	rr2 := httptest.NewRecorder()
	router.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusNotFound {
		t.Fatalf("staff cross-store expected 404 got %d body=%s", rr2.Code, rr2.Body.String())
	}

	// Catalog staff cannot invite
	body, _ := json.Marshal(map[string]string{"email": "x@example.com", "role": "cs"})
	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/stores/"+stA.ID.String()+"/team/invites", bytes.NewReader(body))
	req3.Header.Set("Content-Type", "application/json")
	req3.AddCookie(&http.Cookie{Name: c.CookieName, Value: staffToken})
	rr3 := httptest.NewRecorder()
	router.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusForbidden {
		t.Fatalf("catalog invite expected 403 got %d", rr3.Code)
	}

	// Catalog staff cannot archive
	req4 := httptest.NewRequest(http.MethodPost, "/api/v1/stores/"+stA.ID.String()+"/archive", nil)
	req4.AddCookie(&http.Cookie{Name: c.CookieName, Value: staffToken})
	rr4 := httptest.NewRecorder()
	router.ServeHTTP(rr4, req4)
	if rr4.Code != http.StatusForbidden {
		t.Fatalf("catalog archive expected 403 got %d", rr4.Code)
	}

	// Owner B still isolated from A
	req5 := httptest.NewRequest(http.MethodGet, "/api/v1/stores/"+stA.ID.String(), nil)
	req5.AddCookie(&http.Cookie{Name: c.CookieName, Value: tokenB})
	rr5 := httptest.NewRecorder()
	router.ServeHTTP(rr5, req5)
	if rr5.Code != http.StatusNotFound {
		t.Fatalf("owner B → store A expected 404 got %d", rr5.Code)
	}

	// Cancel + resend path for owner A
	inv2, err := teamSvc.CreateInvite(context.Background(), stA.ID, ownerA, team.CreateInviteInput{
		Email: "pending@example.com", Role: rbac.RoleCS,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := teamSvc.CancelInvite(context.Background(), stA.ID, ownerA, inv2.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := teamSvc.CancelInvite(context.Background(), stA.ID, ownerA, inv2.ID); err == nil {
		t.Fatal("second cancel should fail")
	}

	_ = tokenA
}

func TestDifferentRolesPerStore(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	reset(t, pool)
	c := cfg()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mail := &email.Service{DB: pool, Sender: &email.LogSender{Logger: logger}, Logger: logger}
	urlGen := urls.New(c.MarketingURL, c.DashboardURL, c.StorefrontURL, c.APIBaseURL)
	storeSvc := &stores.Service{DB: pool, Cfg: c}
	teamSvc := &team.Service{DB: pool, Cfg: c, Email: mail, URLs: urlGen, Logger: logger}

	owner1, _ := signIn(t, pool, "o1@example.com")
	owner2, _ := signIn(t, pool, "o2@example.com")
	st1, err := storeSvc.CreateStore(context.Background(), owner1, stores.CreateDirectInput{
		Name: "One", Slug: "role-one", Category: "other", SellingModel: "ready_stock",
	})
	if err != nil {
		t.Fatal(err)
	}
	st2, err := storeSvc.CreateStore(context.Background(), owner2, stores.CreateDirectInput{
		Name: "Two", Slug: "role-two", Category: "other", SellingModel: "ready_stock",
	})
	if err != nil {
		t.Fatal(err)
	}

	inv1, err := teamSvc.CreateInvite(context.Background(), st1.ID, owner1, team.CreateInviteInput{
		Email: "multi@example.com", Role: rbac.RoleCS,
	})
	if err != nil {
		t.Fatal(err)
	}
	a1, err := teamSvc.AcceptInvite(context.Background(), inv1.DevToken, "127.0.0.1", "test")
	if err != nil {
		t.Fatal(err)
	}

	inv2, err := teamSvc.CreateInvite(context.Background(), st2.ID, owner2, team.CreateInviteInput{
		Email: "multi@example.com", Role: rbac.RoleFinance,
	})
	if err != nil {
		t.Fatal(err)
	}
	a2, err := teamSvc.AcceptInvite(context.Background(), inv2.DevToken, "127.0.0.1", "test")
	if err != nil {
		t.Fatal(err)
	}
	if a1.User.ID != a2.User.ID {
		t.Fatal("same email must map to same user")
	}

	r1, err := teamSvc.MembershipRole(context.Background(), a1.User.ID, st1.ID)
	if err != nil || r1 != rbac.RoleCS {
		t.Fatalf("role store1=%s err=%v", r1, err)
	}
	r2, err := teamSvc.MembershipRole(context.Background(), a1.User.ID, st2.ID)
	if err != nil || r2 != rbac.RoleFinance {
		t.Fatalf("role store2=%s err=%v", r2, err)
	}

	list, err := storeSvc.ListForUser(context.Background(), a1.User.ID, false)
	if err != nil || len(list) != 2 {
		t.Fatalf("expected 2 memberships got %+v err=%v", list, err)
	}
}
