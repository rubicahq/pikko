package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv        string
	BaseDomain    string
	LegacyDomain  string // prior apex for cutover allowlists (e.g. pikko.site → pikko.id)
	MarketingURL  string
	DashboardURL  string
	StorefrontURL string
	APIBaseURL    string
	DefaultLocale string
	DefaultTZ     string
	DefaultCurr   string

	DatabaseURL string

	StorageDriver    string
	StorageEndpoint  string
	StorageBucket    string
	StorageAccessKey string
	StorageSecretKey string
	StorageLocalPath string

	EmailProvider          string
	EmailMode              string
	BrevoAPIKey            string
	EmailFromName          string
	EmailAccountFrom       string
	EmailNotificationFrom  string
	EmailReplyTo           string

	MagicLinkTTL       time.Duration
	ResendCooldown     time.Duration
	SessionTTL         time.Duration
	CookieName         string
	CookieDomain       string // empty = host-only (preferred when Next proxies /api)
	CookieSecure       bool
	CookieSameSite     string
	PortalCookieName   string // customer portal session (separate from merchant)
	RedirectAllowlist  []string
	RateLimitEmailHour int
	RateLimitIPHour    int
	APIRateLimitIPHour int

	APIAddr     string
	CORSOrigins []string

	// AI (SumoPod OpenAI-compatible). Keys must never be logged or committed.
	AIAPIKey  string
	AIBaseURL string
	AIModel   string

	ImportMaxPDFBytes int64
	ImportMaxPages    int
	ImportMaxItems    int

	// FeatureStoreCredit is the global kill switch for REF-02/03 store credit ledger.
	// Per-store enablement is stores.store_credit_enabled (both required).
	FeatureStoreCredit bool
}

func Load() (*Config, error) {
	baseDomain := getenv("APP_BASE_DOMAIN", "pikko.site")
	// APP_URL is an alias for MARKETING_URL (PRD §7 / launch cutover docs).
	marketing := firstNonEmpty(os.Getenv("MARKETING_URL"), os.Getenv("APP_URL"), "http://localhost:3000")
	emailDomain := baseDomain
	if emailDomain == "localhost" || strings.HasPrefix(emailDomain, "127.") {
		emailDomain = "pikko.site"
	}

	c := &Config{
		AppEnv:        getenv("APP_ENV", "development"),
		BaseDomain:    baseDomain,
		LegacyDomain:  getenv("LEGACY_BASE_DOMAIN", ""),
		MarketingURL:  marketing,
		DashboardURL:  getenv("DASHBOARD_URL", "http://localhost:3000"),
		StorefrontURL: getenv("STOREFRONT_URL", "http://localhost:3000"),
		APIBaseURL:    getenv("API_BASE_URL", "http://localhost:8080"),
		DefaultLocale: getenv("APP_DEFAULT_LOCALE", "id-ID"),
		DefaultTZ:     getenv("APP_DEFAULT_TIMEZONE", "Asia/Jakarta"),
		DefaultCurr:   getenv("APP_DEFAULT_CURRENCY", "IDR"),

		DatabaseURL: os.Getenv("DATABASE_URL"),

		StorageDriver:    getenv("STORAGE_DRIVER", "local"),
		StorageEndpoint:  os.Getenv("STORAGE_ENDPOINT"),
		StorageBucket:    getenv("STORAGE_BUCKET", "pikko"),
		StorageAccessKey: os.Getenv("STORAGE_ACCESS_KEY"),
		StorageSecretKey: os.Getenv("STORAGE_SECRET_KEY"),
		StorageLocalPath: getenv("STORAGE_LOCAL_PATH", "./.data/storage"),

		EmailProvider:         getenv("EMAIL_PROVIDER", "brevo"),
		EmailMode:             getenv("EMAIL_MODE", "log"),
		BrevoAPIKey:           os.Getenv("BREVO_API_KEY"),
		EmailFromName:         getenv("EMAIL_FROM_NAME", "Pikko"),
		EmailAccountFrom:      getenv("EMAIL_ACCOUNT_FROM", "account@"+emailDomain),
		EmailNotificationFrom: getenv("EMAIL_NOTIFICATION_FROM", "notifications@"+emailDomain),
		EmailReplyTo:          getenv("EMAIL_REPLY_TO", "support@"+emailDomain),

		CookieName:       getenv("AUTH_COOKIE_NAME", "pikko_session"),
		CookieDomain:     getenv("AUTH_COOKIE_DOMAIN", ""),
		CookieSecure:     getenvBool("AUTH_COOKIE_SECURE", false),
		CookieSameSite:   getenv("AUTH_COOKIE_SAMESITE", "lax"),
		PortalCookieName: getenv("PORTAL_COOKIE_NAME", "pikko_portal_session"),

		APIAddr: getenv("API_ADDR", ":8080"),
	}

	ttlMin, err := getenvInt("AUTH_MAGIC_LINK_TTL_MINUTES", 15)
	if err != nil {
		return nil, err
	}
	c.MagicLinkTTL = time.Duration(ttlMin) * time.Minute

	cooldown, err := getenvInt("AUTH_RESEND_COOLDOWN_SECONDS", 60)
	if err != nil {
		return nil, err
	}
	c.ResendCooldown = time.Duration(cooldown) * time.Second

	sessionDays, err := getenvInt("AUTH_SESSION_TTL_DAYS", 30)
	if err != nil {
		return nil, err
	}
	c.SessionTTL = time.Duration(sessionDays) * 24 * time.Hour

	c.RateLimitEmailHour, err = getenvInt("AUTH_RATE_LIMIT_EMAIL_PER_HOUR", 5)
	if err != nil {
		return nil, err
	}
	c.RateLimitIPHour, err = getenvInt("AUTH_RATE_LIMIT_IP_PER_HOUR", 20)
	if err != nil {
		return nil, err
	}
	c.APIRateLimitIPHour, err = getenvInt("API_RATE_LIMIT_IP_PER_HOUR", 1200)
	if err != nil {
		return nil, err
	}

	allowDefault := "localhost,127.0.0.1," + c.BaseDomain
	if c.LegacyDomain != "" && c.LegacyDomain != c.BaseDomain {
		allowDefault += "," + c.LegacyDomain
	}
	allow := getenv("AUTH_REDIRECT_ALLOWLIST", allowDefault)
	c.RedirectAllowlist = splitCSV(allow)

	corsDefault := "http://localhost:3000"
	if c.AppEnv == "production" || c.AppEnv == "staging" {
		corsDefault = strings.Join([]string{
			c.DashboardURL,
			c.MarketingURL,
			c.StorefrontURL,
		}, ",")
	}
	c.CORSOrigins = splitCSV(getenv("API_CORS_ORIGINS", corsDefault))

	c.AIAPIKey = os.Getenv("AI_API_KEY")
	c.AIBaseURL = getenv("AI_BASE_URL", "")
	c.AIModel = getenv("AI_MODEL", "gemini/gemini-3.5-flash")

	c.ImportMaxPDFBytes = int64(25 << 20)
	if n, err := getenvInt("IMPORT_MAX_PDF_BYTES", 25<<20); err == nil {
		c.ImportMaxPDFBytes = int64(n)
	}
	c.ImportMaxPages, err = getenvInt("IMPORT_MAX_PAGES", 50)
	if err != nil {
		return nil, err
	}
	c.ImportMaxItems, err = getenvInt("IMPORT_MAX_ITEMS", 500)
	if err != nil {
		return nil, err
	}

	c.FeatureStoreCredit = getenvBool("FEATURE_STORE_CREDIT", false)

	if c.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if c.EmailMode == "brevo" && c.BrevoAPIKey == "" {
		return nil, fmt.Errorf("BREVO_API_KEY is required when EMAIL_MODE=brevo")
	}
	return c, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func getenvInt(key string, fallback int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return n, nil
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
