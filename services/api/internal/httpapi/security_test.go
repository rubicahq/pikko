package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rubicahq/pikko/services/api/internal/config"
)

func TestSecurityHeadersIncludeHardening(t *testing.T) {
	cfg := &config.Config{AppEnv: "production"}
	h := SecurityHeaders(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	res := rec.Result()
	defer res.Body.Close()
	if res.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("missing nosniff")
	}
	if res.Header.Get("X-Frame-Options") != "DENY" {
		t.Fatalf("missing frame deny")
	}
	if res.Header.Get("Strict-Transport-Security") == "" {
		t.Fatalf("expected HSTS in production")
	}
	if res.Header.Get("Content-Security-Policy") == "" {
		t.Fatalf("expected CSP")
	}
}

func TestAccessLogDoesNotPanic(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := AccessLogMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	req := httptest.NewRequest(http.MethodGet, "/livez", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
}
