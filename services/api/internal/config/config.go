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
	CookieSecure       bool
	CookieSameSite     string
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
}

func Load() (*Config, error) {
	c := &Config{
		AppEnv:        getenv("APP_ENV", "development"),
		BaseDomain:    getenv("APP_BASE_DOMAIN", "pikko.site"),
		MarketingURL:  getenv("MARKETING_URL", "http://localhost:3000"),
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
		EmailAccountFrom:      getenv("EMAIL_ACCOUNT_FROM", "account@pikko.site"),
		EmailNotificationFrom: getenv("EMAIL_NOTIFICATION_FROM", "notifications@pikko.site"),
		EmailReplyTo:          getenv("EMAIL_REPLY_TO", "support@pikko.site"),

		CookieName:     getenv("AUTH_COOKIE_NAME", "pikko_session"),
		CookieSecure:   getenvBool("AUTH_COOKIE_SECURE", false),
		CookieSameSite: getenv("AUTH_COOKIE_SAMESITE", "lax"),

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

	allow := getenv("AUTH_REDIRECT_ALLOWLIST", "localhost,127.0.0.1,pikko.site")
	c.RedirectAllowlist = splitCSV(allow)
	c.CORSOrigins = splitCSV(getenv("API_CORS_ORIGINS", "http://localhost:3000"))

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
