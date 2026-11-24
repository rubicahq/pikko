package auth_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rubicahq/pikko/services/api/internal/auth"
	"github.com/rubicahq/pikko/services/api/internal/config"
	"github.com/rubicahq/pikko/services/api/internal/email"
	"github.com/rubicahq/pikko/services/api/internal/urls"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = "postgres://pikko:pikko@localhost:5432/pikko_test?sslmode=disable"
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Skipf("no test db: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("no test db: %v", err)
	}
	return pool
}

func resetAuthTables(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		TRUNCATE audit_logs, email_jobs, auth_rate_limits, onboarding_drafts,
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

func newAuthService(t *testing.T, pool *pgxpool.Pool) *auth.Service {
	t.Helper()
	cfg := &config.Config{
		AppEnv:             "test",
		EmailMode:          "log",
		EmailFromName:      "Pikko",
		EmailAccountFrom:   "account@pikko.site",
		EmailReplyTo:       "support@pikko.site",
		MagicLinkTTL:       15 * time.Minute,
		ResendCooldown:     60 * time.Second,
		SessionTTL:         30 * 24 * time.Hour,
		RateLimitEmailHour: 50,
		RateLimitIPHour:    100,
		DefaultLocale:      "id-ID",
		DefaultTZ:          "Asia/Jakarta",
		CookieName:         "pikko_session",
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mail := &email.Service{DB: pool, Sender: &email.LogSender{Logger: logger}, Logger: logger}
	return &auth.Service{
		DB:     pool,
		Cfg:    cfg,
		Email:  mail,
		URLs:   urls.New("http://localhost:3000", "http://localhost:3000", "http://localhost:3000", "http://localhost:8080"),
		Logger: logger,
	}
}

func TestMagicLinkPeekDoesNotConsume(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	resetAuthTables(t, pool)
	svc := newAuthService(t, pool)
	ctx := context.Background()

	res, err := svc.RequestMagicLink(ctx, "merchant@example.com", "127.0.0.1", "test")
	if err != nil {
		t.Fatal(err)
	}
	if res.DevToken == "" {
		t.Fatal("expected dev token")
	}

	_, ok := svc.PeekToken(ctx, res.DevToken)
	if !ok {
		t.Fatal("peek should succeed")
	}
	_, ok = svc.PeekToken(ctx, res.DevToken)
	if !ok {
		t.Fatal("peek must not consume token")
	}

	verify, err := svc.VerifyAndConsume(ctx, res.DevToken, "127.0.0.1", "test")
	if err != nil {
		t.Fatal(err)
	}
	if !verify.IsNewAccount {
		t.Fatal("expected new account")
	}
	if verify.RedirectPath != "/onboarding" {
		t.Fatalf("redirect %s", verify.RedirectPath)
	}

	_, ok = svc.PeekToken(ctx, res.DevToken)
	if ok {
		t.Fatal("consumed token must be invalid")
	}
	_, err = svc.VerifyAndConsume(ctx, res.DevToken, "127.0.0.1", "test")
	if err == nil {
		t.Fatal("reuse must fail")
	}
}

func TestConcurrentVerifyConsumesOnce(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	resetAuthTables(t, pool)
	svc := newAuthService(t, pool)
	ctx := context.Background()

	res, err := svc.RequestMagicLink(ctx, "race@example.com", "127.0.0.1", "test")
	if err != nil {
		t.Fatal(err)
	}

	type result struct {
		err error
	}
	ch := make(chan result, 2)
	for i := 0; i < 2; i++ {
		go func() {
			_, e := svc.VerifyAndConsume(ctx, res.DevToken, "127.0.0.1", "test")
			ch <- result{e}
		}()
	}
	var okCount, errCount int
	for i := 0; i < 2; i++ {
		r := <-ch
		if r.err == nil {
			okCount++
		} else {
			errCount++
		}
	}
	if okCount != 1 || errCount != 1 {
		t.Fatalf("ok=%d err=%d", okCount, errCount)
	}

	var userCount int
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE email_normalized='race@example.com'`).Scan(&userCount)
	if userCount != 1 {
		t.Fatalf("duplicate accounts: %d", userCount)
	}
}

func TestSessionRevocation(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	resetAuthTables(t, pool)
	svc := newAuthService(t, pool)
	ctx := context.Background()

	res, err := svc.RequestMagicLink(ctx, "sess@example.com", "127.0.0.1", "test")
	if err != nil {
		t.Fatal(err)
	}
	verify, err := svc.VerifyAndConsume(ctx, res.DevToken, "127.0.0.1", "test")
	if err != nil {
		t.Fatal(err)
	}
	user, _, err := svc.Authenticate(ctx, verify.SessionToken)
	if err != nil || user.Email != "sess@example.com" {
		t.Fatal("session should work")
	}
	_ = svc.Logout(ctx, verify.SessionToken)
	_, _, err = svc.Authenticate(ctx, verify.SessionToken)
	if err == nil {
		t.Fatal("revoked session must fail")
	}
	_ = uuid.Nil
}
