package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/rubicahq/pikko/services/api/internal/syshealth"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

// AccessLogMiddleware emits structured request logs without sensitive payloads.
// When health is set, also records measured API metrics and unexpected 5xx errors.
func AccessLogMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return AccessLogMiddlewareWithHealth(logger, nil)
}

func AccessLogMiddlewareWithHealth(logger *slog.Logger, health *syshealth.Service) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: 0}
			next.ServeHTTP(rec, r)
			status := rec.status
			if status == 0 {
				status = http.StatusOK
			}
			dur := int(time.Since(start).Milliseconds())
			path := r.URL.Path
			logger.Info("http_request",
				"method", r.Method,
				"path", path,
				"status", status,
				"bytes", rec.bytes,
				"duration_ms", dur,
				"request_id", RequestID(r),
				"ip", ClientIP(r),
			)
			if health == nil {
				return
			}
			// Skip health probes from metric pollution.
			if path == "/healthz" || path == "/readyz" || path == "/livez" {
				return
			}
			bg := context.Background()
			if strings.HasPrefix(path, "/api/") {
				go health.RecordRequestMetric(bg, status, dur)
			}
			// Do not classify normal business validation (4xx) as unexpected errors.
			if status >= 500 {
				st := status
				d := dur
				rid := RequestID(r)
				mod := moduleFromPath(path)
				msg := "HTTP " + http.StatusText(status) + " on " + r.Method + " " + path
				ep := r.Method + " " + path
				go health.Record(bg, syshealth.Event{
					Source:     "api",
					Severity:   syshealth.SeverityError,
					Module:     mod,
					ErrorCode:  "http_5xx",
					Message:    msg,
					RequestID:  rid,
					Endpoint:   ep,
					HTTPStatus: &st,
					DurationMs: &d,
					Context:    map[string]any{"path": path},
				})
			}
		})
	}
}

func moduleFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 3 && parts[0] == "api" && parts[1] == "v1" {
		return parts[2]
	}
	if len(parts) > 0 {
		return parts[0]
	}
	return "api"
}
