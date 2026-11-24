package config

import (
	"os"
	"strings"
	"testing"
)

func TestLoadDomainAgnosticDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://pikko:pikko@localhost:5432/pikko_test?sslmode=disable")
	t.Setenv("APP_BASE_DOMAIN", "pikko.id")
	t.Setenv("LEGACY_BASE_DOMAIN", "pikko.site")
	t.Setenv("APP_ENV", "development")
	os.Unsetenv("MARKETING_URL")
	os.Unsetenv("APP_URL")
	os.Unsetenv("EMAIL_ACCOUNT_FROM")
	os.Unsetenv("EMAIL_NOTIFICATION_FROM")
	os.Unsetenv("EMAIL_REPLY_TO")
	os.Unsetenv("AUTH_REDIRECT_ALLOWLIST")
	os.Unsetenv("API_CORS_ORIGINS")
	os.Unsetenv("AUTH_COOKIE_DOMAIN")

	t.Setenv("APP_URL", "https://pikko.id")

	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.BaseDomain != "pikko.id" {
		t.Fatalf("BaseDomain=%s", c.BaseDomain)
	}
	if c.LegacyDomain != "pikko.site" {
		t.Fatalf("LegacyDomain=%s", c.LegacyDomain)
	}
	if c.MarketingURL != "https://pikko.id" {
		t.Fatalf("MarketingURL from APP_URL=%s", c.MarketingURL)
	}
	if c.EmailAccountFrom != "account@pikko.id" {
		t.Fatalf("EmailAccountFrom=%s", c.EmailAccountFrom)
	}
	if c.EmailNotificationFrom != "notifications@pikko.id" {
		t.Fatalf("EmailNotificationFrom=%s", c.EmailNotificationFrom)
	}
	if c.EmailReplyTo != "support@pikko.id" {
		t.Fatalf("EmailReplyTo=%s", c.EmailReplyTo)
	}
	joined := strings.Join(c.RedirectAllowlist, ",")
	if !strings.Contains(joined, "pikko.id") || !strings.Contains(joined, "pikko.site") {
		t.Fatalf("allowlist missing domains: %s", joined)
	}
	if c.CookieDomain != "" {
		t.Fatalf("expected empty CookieDomain by default, got %q", c.CookieDomain)
	}
}

func TestLoadCookieDomain(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://pikko:pikko@localhost:5432/pikko_test?sslmode=disable")
	t.Setenv("AUTH_COOKIE_DOMAIN", ".pikko.id")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.CookieDomain != ".pikko.id" {
		t.Fatalf("CookieDomain=%s", c.CookieDomain)
	}
}
