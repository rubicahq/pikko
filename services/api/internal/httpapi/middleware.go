package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/rubicahq/pikko/services/api/internal/auth"
	"github.com/rubicahq/pikko/services/api/internal/config"
	"github.com/rubicahq/pikko/services/api/internal/rbac"
	"github.com/rubicahq/pikko/services/api/internal/stores"
)

func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = uuid.NewString()
		}
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), ctxKeyRequestID, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func CORSMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	allowed := map[string]struct{}{}
	for _, o := range cfg.CORSOrigins {
		allowed[o] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if _, ok := allowed[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID, X-CSRF-Token")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func SecurityHeaders(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", "no-referrer")
			w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
			w.Header().Set("Cross-Origin-Resource-Policy", "same-site")
			w.Header().Set("X-Permitted-Cross-Domain-Policies", "none")
			// JSON APIs: deny framing + restrict sniffing; CSP is conservative for API responses.
			w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'")
			if cfg != nil && cfg.AppEnv == "production" {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}
			next.ServeHTTP(w, r)
		})
	}
}

type AuthMiddleware struct {
	Auth *auth.Service
	Cfg  *config.Config
}

func (m *AuthMiddleware) sessionRaw(r *http.Request) string {
	if c, err := r.Cookie(m.Cfg.CookieName); err == nil && c.Value != "" {
		return c.Value
	}
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(h), "bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

func (m *AuthMiddleware) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := m.sessionRaw(r)
		user, _, err := m.Auth.Authenticate(r.Context(), raw)
		if err != nil {
			writeError(w, r, http.StatusUnauthorized, "unauthorized", "Authentication required.")
			return
		}
		ctx := context.WithValue(r.Context(), ctxKeyUserID, user.ID)
		ctx = context.WithValue(ctx, ctxKeySession, raw)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserID(r *http.Request) uuid.UUID {
	id, _ := r.Context().Value(ctxKeyUserID).(uuid.UUID)
	return id
}

func SessionRaw(r *http.Request) string {
	s, _ := r.Context().Value(ctxKeySession).(string)
	return s
}

type StoreAuthMiddleware struct {
	Stores *stores.Service
}

func (m *StoreAuthMiddleware) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		storeIDStr := r.PathValue("storeID")
		storeID, err := uuid.Parse(storeIDStr)
		if err != nil {
			writeError(w, r, http.StatusNotFound, "not_found", "Store not found.")
			return
		}
		userID := UserID(r)
		st, err := m.Stores.GetAuthorized(r.Context(), userID, storeID)
		if err != nil {
			// Indistinguishable not-found vs forbidden for cross-store probes
			writeError(w, r, http.StatusNotFound, "not_found", "Store not found.")
			return
		}
		ctx := context.WithValue(r.Context(), ctxKeyStoreID, storeID)
		ctx = context.WithValue(ctx, ctxKeyMembershipRole, st.Role)
		_ = m.Stores.TouchLastAccessed(r.Context(), userID, storeID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequirePermission ensures the active membership grants perm (after membership check).
func (m *StoreAuthMiddleware) RequirePermission(perm rbac.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return m.Require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := MembershipRole(r)
			if !rbac.HasPermission(role, perm) {
				writeError(w, r, http.StatusForbidden, "forbidden", "You do not have permission for this action.")
				return
			}
			next.ServeHTTP(w, r)
		}))
	}
}

func StoreID(r *http.Request) uuid.UUID {
	id, _ := r.Context().Value(ctxKeyStoreID).(uuid.UUID)
	return id
}

func MembershipRole(r *http.Request) string {
	role, _ := r.Context().Value(ctxKeyMembershipRole).(string)
	return role
}

func ClientIP(r *http.Request) string {
	// Prefer X-Real-IP set by our nginx from $remote_addr (not client-spoofable).
	// Never trust the leftmost X-Forwarded-For hop — clients can prepend forged IPs
	// when the proxy uses $proxy_add_x_forwarded_for (SEC-07).
	if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
		return xri
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		// Rightmost hop is closest to us when a single trusted proxy appends.
		return strings.TrimSpace(parts[len(parts)-1])
	}
	host := r.RemoteAddr
	if i := strings.LastIndex(host, ":"); i >= 0 {
		return host[:i]
	}
	return host
}
