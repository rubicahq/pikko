package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/rubicahq/pikko/services/api/internal/auth"
	"github.com/rubicahq/pikko/services/api/internal/config"
	"github.com/rubicahq/pikko/services/api/internal/syshealth"
	"github.com/rubicahq/pikko/services/api/internal/validate"
)

type HQHandlers struct {
	Auth    *auth.Service
	Health  *syshealth.Service
	Checker *syshealth.Checker
	Alerter *syshealth.Alerter
	Cfg     *config.Config
}

type HQAuthMiddleware struct {
	Auth   *auth.Service
	Health *syshealth.Service
	Cfg    *config.Config
}

const ctxKeyHQEmail ctxKey = "hq_email"

func (m *HQAuthMiddleware) sessionRaw(r *http.Request) string {
	if c, err := r.Cookie(m.Cfg.CookieName); err == nil && c.Value != "" {
		return c.Value
	}
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(h), "bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

func (m *HQAuthMiddleware) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := m.sessionRaw(r)
		user, _, err := m.Auth.Authenticate(r.Context(), raw)
		if err != nil {
			writeError(w, r, http.StatusUnauthorized, "unauthorized", "Authentication required.")
			return
		}
		ok, err := m.Health.IsHQAdmin(r.Context(), strings.ToLower(strings.TrimSpace(user.Email)))
		if err != nil || !ok {
			writeError(w, r, http.StatusForbidden, "forbidden", "HQ access required.")
			return
		}
		ctx := context.WithValue(r.Context(), ctxKeyUserID, user.ID)
		ctx = context.WithValue(ctx, ctxKeySession, raw)
		ctx = context.WithValue(ctx, ctxKeyHQEmail, user.Email)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func hqEmail(r *http.Request) string {
	s, _ := r.Context().Value(ctxKeyHQEmail).(string)
	return s
}

func (h *HQHandlers) setSessionCookie(w http.ResponseWriter, token string, expires time.Time) {
	sameSite := http.SameSiteLaxMode
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

func (h *HQHandlers) clearSessionCookie(w http.ResponseWriter) {
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

// RequestMagicLink only emails allowlisted HQ admins; always returns a generic success body.
func (h *HQHandlers) RequestMagicLink(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	display, normalized, ok := validate.NormalizeEmail(body.Email)
	if !ok {
		writeError(w, r, http.StatusBadRequest, "invalid_email", "Enter a valid email address.")
		return
	}
	allowed, err := h.Health.IsHQAdmin(r.Context(), normalized)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "hq_auth_failed", "Unable to process sign-in.")
		return
	}
	cooldown := int(h.Cfg.ResendCooldown.Seconds())
	if allowed {
		res, err := h.Auth.RequestMagicLinkHQ(r.Context(), display, ClientIP(r), r.UserAgent())
		if err != nil {
			switch {
			case errors.Is(err, auth.ErrCooldown):
				writeError(w, r, http.StatusTooManyRequests, "cooldown", "Please wait before requesting another link.")
			case errors.Is(err, auth.ErrRateLimited):
				writeError(w, r, http.StatusTooManyRequests, "rate_limited", "Too many requests. Try again later.")
			default:
				writeError(w, r, http.StatusInternalServerError, "email_failed", "Unable to send sign-in link.")
			}
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok": true, "email": res.Email, "cooldown_seconds": res.CooldownSeconds,
			"message": "If this email can receive mail, a sign-in link is on the way.",
			"dev_token": res.DevToken,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "email": display, "cooldown_seconds": cooldown,
		"message": "If this email can receive mail, a sign-in link is on the way.",
	})
}

func (h *HQHandlers) PeekMagicLink(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	email, ok := h.Auth.PeekToken(r.Context(), token)
	if !ok {
		writeError(w, r, http.StatusGone, "invalid_token", "This link has expired.")
		return
	}
	allowed, err := h.Health.IsHQAdmin(r.Context(), email)
	if err != nil || !allowed {
		writeError(w, r, http.StatusGone, "invalid_token", "This link has expired.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "valid": true})
}

func (h *HQHandlers) VerifyMagicLink(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	// Peek first so we never create a session for non-HQ emails.
	email, ok := h.Auth.PeekToken(r.Context(), body.Token)
	if !ok {
		writeError(w, r, http.StatusGone, "invalid_token", "This link has expired.")
		return
	}
	allowed, err := h.Health.IsHQAdmin(r.Context(), email)
	if err != nil || !allowed {
		writeError(w, r, http.StatusForbidden, "forbidden", "HQ access required.")
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
	h.Health.LinkHQAdminUser(r.Context(), email, res.User.ID)
	expires := time.Now().UTC().Add(h.Cfg.SessionTTL)
	h.setSessionCookie(w, res.SessionToken, expires)
	h.Health.Audit(r.Context(), &res.User.ID, res.User.Email, "hq.login", "user", res.User.ID.String(), nil)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "user": res.User, "redirect_path": "/system-health",
	})
}

func (h *HQHandlers) Session(w http.ResponseWriter, r *http.Request) {
	raw := ""
	if c, err := r.Cookie(h.Cfg.CookieName); err == nil {
		raw = c.Value
	}
	user, _, err := h.Auth.Authenticate(r.Context(), raw)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, "unauthorized", "Authentication required.")
		return
	}
	ok, err := h.Health.IsHQAdmin(r.Context(), strings.ToLower(user.Email))
	if err != nil || !ok {
		writeError(w, r, http.StatusForbidden, "forbidden", "HQ access required.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "user": user})
}

func (h *HQHandlers) Logout(w http.ResponseWriter, r *http.Request) {
	raw := ""
	if c, err := r.Cookie(h.Cfg.CookieName); err == nil {
		raw = c.Value
	}
	if raw != "" {
		_ = h.Auth.Logout(r.Context(), raw)
	}
	h.clearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *HQHandlers) parseRange(r *http.Request) (syshealth.Range, error) {
	return syshealth.ParseRange(r.URL.Query().Get("range"), r.URL.Query().Get("from"), r.URL.Query().Get("to"))
}

func (h *HQHandlers) Overview(w http.ResponseWriter, r *http.Request) {
	rng, err := h.parseRange(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_range", err.Error())
		return
	}
	out, err := h.Health.Overview(r.Context(), rng)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "overview_failed", "Unable to load overview.")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *HQHandlers) ListIssues(w http.ResponseWriter, r *http.Request) {
	rng, err := h.parseRange(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_range", err.Error())
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	items, total, err := h.Health.ListIssues(r.Context(), syshealth.IssueFilter{
		Range: rng, Severity: r.URL.Query().Get("severity"), Status: r.URL.Query().Get("status"),
		Source: r.URL.Query().Get("source"), Module: r.URL.Query().Get("module"),
		Query: r.URL.Query().Get("q"), Limit: limit, Offset: offset,
	})
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "issues_failed", "Unable to list issues.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": total, "range": rng})
}

func (h *HQHandlers) GetIssue(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("issueID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Issue not found.")
		return
	}
	issue, occ, err := h.Health.GetIssue(r.Context(), id)
	if errors.Is(err, syshealth.ErrNotFound) {
		writeError(w, r, http.StatusNotFound, "not_found", "Issue not found.")
		return
	}
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "issue_failed", "Unable to load issue.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"issue": issue, "occurrences": occ})
}

func (h *HQHandlers) PatchIssue(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("issueID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Issue not found.")
		return
	}
	var body struct {
		Status *string `json:"status"`
		Notes  *string `json:"admin_notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	issue, err := h.Health.PatchIssue(r.Context(), id, body.Status, body.Notes, UserID(r))
	if errors.Is(err, syshealth.ErrNotFound) {
		writeError(w, r, http.StatusNotFound, "not_found", "Issue not found.")
		return
	}
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "patch_failed", err.Error())
		return
	}
	uid := UserID(r)
	h.Health.Audit(r.Context(), &uid, hqEmail(r), "hq.issue.patch", "issue", id.String(), map[string]any{
		"status": body.Status, "notes": body.Notes != nil,
	})
	writeJSON(w, http.StatusOK, map[string]any{"issue": issue})
}

func (h *HQHandlers) JobStats(w http.ResponseWriter, r *http.Request) {
	rng, err := h.parseRange(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_range", err.Error())
		return
	}
	st, err := h.Health.JobStats(r.Context(), rng)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "jobs_failed", "Unable to load job stats.")
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func (h *HQHandlers) ListJobs(w http.ResponseWriter, r *http.Request) {
	rng, err := h.parseRange(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_range", err.Error())
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.Health.ListJobs(r.Context(), rng, r.URL.Query().Get("status"), r.URL.Query().Get("queue"), limit)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "jobs_failed", "Unable to list jobs.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "range": rng})
}

func (h *HQHandlers) GetJob(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("jobID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Job not found.")
		return
	}
	job, err := h.Health.GetJob(r.Context(), r.URL.Query().Get("queue"), id)
	if errors.Is(err, syshealth.ErrNotFound) {
		writeError(w, r, http.StatusNotFound, "not_found", "Job not found.")
		return
	}
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "job_failed", "Unable to load job.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"job": job})
}

func (h *HQHandlers) RetryJob(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("jobID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Job not found.")
		return
	}
	job, err := h.Health.RetryJob(r.Context(), id)
	if errors.Is(err, syshealth.ErrNotFound) {
		writeError(w, r, http.StatusNotFound, "not_found", "Job not found.")
		return
	}
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "retry_failed", err.Error())
		return
	}
	uid := UserID(r)
	h.Health.Audit(r.Context(), &uid, hqEmail(r), "hq.job.retry", "job", id.String(), nil)
	writeJSON(w, http.StatusOK, map[string]any{"job": job})
}

func (h *HQHandlers) ListServices(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("refresh") == "1" && h.Checker != nil {
		_, _ = h.Checker.RunChecks(r.Context())
	}
	items, err := h.Health.ListServiceChecks(r.Context())
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "services_failed", "Unable to list services.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *HQHandlers) Usage(w http.ResponseWriter, r *http.Request) {
	rng, err := h.parseRange(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_range", err.Error())
		return
	}
	out, err := h.Health.Usage(r.Context(), rng)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "usage_failed", "Unable to load usage.")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *HQHandlers) ListAlertRules(w http.ResponseWriter, r *http.Request) {
	items, err := h.Health.ListAlertRules(r.Context())
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "alerts_failed", "Unable to list alert rules.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *HQHandlers) PatchAlertRule(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("ruleID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Rule not found.")
		return
	}
	var body struct {
		Enabled         *bool    `json:"enabled"`
		Threshold       *int     `json:"threshold"`
		WindowMinutes   *int     `json:"window_minutes"`
		CooldownMinutes *int     `json:"cooldown_minutes"`
		NotifyEmails    []string `json:"notify_emails"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	rule, err := h.Health.PatchAlertRule(r.Context(), id, body.Enabled, body.Threshold, body.WindowMinutes, body.CooldownMinutes, body.NotifyEmails)
	if errors.Is(err, syshealth.ErrNotFound) {
		writeError(w, r, http.StatusNotFound, "not_found", "Rule not found.")
		return
	}
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "patch_failed", "Unable to update rule.")
		return
	}
	uid := UserID(r)
	h.Health.Audit(r.Context(), &uid, hqEmail(r), "hq.alert_rule.patch", "alert_rule", id.String(), nil)
	writeJSON(w, http.StatusOK, map[string]any{"rule": rule})
}

func (h *HQHandlers) ListAlertEvents(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.Health.ListAlertEvents(r.Context(), limit)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "alerts_failed", "Unable to list alert events.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *HQHandlers) EvaluateAlerts(w http.ResponseWriter, r *http.Request) {
	if h.Alerter == nil {
		writeError(w, r, http.StatusServiceUnavailable, "unavailable", "Alerter not configured.")
		return
	}
	if err := h.Alerter.EvaluateRules(r.Context()); err != nil {
		writeError(w, r, http.StatusInternalServerError, "evaluate_failed", "Unable to evaluate alerts.")
		return
	}
	uid := UserID(r)
	h.Health.Audit(r.Context(), &uid, hqEmail(r), "hq.alerts.evaluate", "", "", nil)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
