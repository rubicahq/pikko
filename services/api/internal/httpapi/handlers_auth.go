package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/rubicahq/pikko/services/api/internal/auth"
	"github.com/rubicahq/pikko/services/api/internal/config"
)

type AuthHandlers struct {
	Auth *auth.Service
	Cfg  *config.Config
}

func (h *AuthHandlers) setSessionCookie(w http.ResponseWriter, token string, expires time.Time) {
	sameSite := http.SameSiteLaxMode
	switch h.Cfg.CookieSameSite {
	case "strict":
		sameSite = http.SameSiteStrictMode
	case "none":
		sameSite = http.SameSiteNoneMode
	}
	http.SetCookie(w, &http.Cookie{
		Name:     h.Cfg.CookieName,
		Value:    token,
		Path:     "/",
		Domain:   h.Cfg.CookieDomain,
		Expires:  expires,
		MaxAge:   int(time.Until(expires).Seconds()),
		HttpOnly: true,
		Secure:   h.Cfg.CookieSecure,
		SameSite: sameSite,
	})
}

func (h *AuthHandlers) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.Cfg.CookieName,
		Value:    "",
		Path:     "/",
		Domain:   h.Cfg.CookieDomain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.Cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *AuthHandlers) RequestMagicLink(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	res, err := h.Auth.RequestMagicLink(r.Context(), body.Email, ClientIP(r), r.UserAgent())
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidEmail):
			writeError(w, r, http.StatusBadRequest, "invalid_email", "Enter a valid email address.")
		case errors.Is(err, auth.ErrCooldown):
			writeError(w, r, http.StatusTooManyRequests, "cooldown", "Please wait before requesting another link.")
		case errors.Is(err, auth.ErrRateLimited):
			writeError(w, r, http.StatusTooManyRequests, "rate_limited", "Too many requests. Try again later.")
		default:
			writeError(w, r, http.StatusInternalServerError, "email_failed", "Unable to send sign-in link.")
		}
		return
	}
	// Generic success — do not reveal whether account exists
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":               true,
		"email":            res.Email,
		"cooldown_seconds": res.CooldownSeconds,
		"message":          "If this email can receive mail, a sign-in link is on the way.",
		"dev_token":        res.DevToken,
	})
}

func (h *AuthHandlers) PeekMagicLink(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	_, ok := h.Auth.PeekToken(r.Context(), token)
	if !ok {
		writeError(w, r, http.StatusGone, "invalid_token", "This link has expired.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "valid": true})
}

func (h *AuthHandlers) VerifyMagicLink(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	res, err := h.Auth.VerifyAndConsume(r.Context(), body.Token, ClientIP(r), r.UserAgent())
	if err != nil {
		if errors.Is(err, auth.ErrInvalidToken) {
			writeError(w, r, http.StatusGone, "invalid_token", "This link has expired.")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "verify_failed", "Unable to complete sign-in.")
		return
	}
	expires := time.Now().UTC().Add(h.Cfg.SessionTTL)
	h.setSessionCookie(w, res.SessionToken, expires)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":             true,
		"user":           res.User,
		"is_new_account": res.IsNewAccount,
		"redirect_path":  res.RedirectPath,
	})
}

func (h *AuthHandlers) Session(w http.ResponseWriter, r *http.Request) {
	raw := ""
	if c, err := r.Cookie(h.Cfg.CookieName); err == nil {
		raw = c.Value
	}
	user, sess, err := h.Auth.Authenticate(r.Context(), raw)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, "unauthorized", "Not signed in.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user":    user,
		"session": sess,
	})
}

func (h *AuthHandlers) Logout(w http.ResponseWriter, r *http.Request) {
	raw := ""
	if c, err := r.Cookie(h.Cfg.CookieName); err == nil {
		raw = c.Value
	}
	_ = h.Auth.Logout(r.Context(), raw)
	h.clearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *AuthHandlers) LogoutAll(w http.ResponseWriter, r *http.Request) {
	_ = h.Auth.LogoutAll(r.Context(), UserID(r))
	h.clearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *AuthHandlers) ListSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := h.Auth.ListSessions(r.Context(), UserID(r), SessionRaw(r))
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "list_failed", "Unable to list sessions.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": sessions})
}

func (h *AuthHandlers) RevokeSession(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("sessionID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Session not found.")
		return
	}
	if err := h.Auth.RevokeSession(r.Context(), UserID(r), id); err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Session not found.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
