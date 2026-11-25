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

	"github.com/rubicahq/pikko/services/api/internal/commerce"
	"github.com/rubicahq/pikko/services/api/internal/config"
	"github.com/rubicahq/pikko/services/api/internal/portal"
)

type PortalHandlers struct {
	Portal *portal.Service
	Cfg    *config.Config
}

func (h *PortalHandlers) portalCookieName() string {
	if h.Cfg.PortalCookieName != "" {
		return h.Cfg.PortalCookieName
	}
	return "pikko_portal_session"
}

func (h *PortalHandlers) setPortalCookie(w http.ResponseWriter, token string, expires time.Time) {
	sameSite := http.SameSiteLaxMode
	switch h.Cfg.CookieSameSite {
	case "strict":
		sameSite = http.SameSiteStrictMode
	case "none":
		sameSite = http.SameSiteNoneMode
	}
	http.SetCookie(w, &http.Cookie{
		Name:     h.portalCookieName(),
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

func (h *PortalHandlers) clearPortalCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.portalCookieName(),
		Value:    "",
		Path:     "/",
		Domain:   h.Cfg.CookieDomain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.Cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *PortalHandlers) portalSessionRaw(r *http.Request) string {
	if c, err := r.Cookie(h.portalCookieName()); err == nil && c.Value != "" {
		return c.Value
	}
	authz := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(authz), "bearer ") {
		return strings.TrimSpace(authz[7:])
	}
	return ""
}

func PortalCustomerID(r *http.Request) uuid.UUID {
	id, _ := r.Context().Value(ctxKeyPortalCustomerID).(uuid.UUID)
	return id
}

func PortalStoreID(r *http.Request) uuid.UUID {
	id, _ := r.Context().Value(ctxKeyPortalStoreID).(uuid.UUID)
	return id
}

func (h *PortalHandlers) mapErr(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, portal.ErrInvalidEmail):
		writeError(w, r, http.StatusBadRequest, "invalid_email", "Enter a valid email address.")
	case errors.Is(err, portal.ErrAccessLinkRequired):
		writeError(w, r, http.StatusGone, "access_link_required", "Use your personal customer portal link from the store.")
	case errors.Is(err, portal.ErrCooldown):
		writeError(w, r, http.StatusTooManyRequests, "cooldown", "Please wait before requesting another link.")
	case errors.Is(err, portal.ErrRateLimited):
		writeError(w, r, http.StatusTooManyRequests, "rate_limited", "Too many requests. Try again later.")
	case errors.Is(err, portal.ErrInvalidToken):
		writeError(w, r, http.StatusGone, "invalid_token", "This portal link is invalid or no longer works.")
	case errors.Is(err, portal.ErrUnauthorized):
		writeError(w, r, http.StatusUnauthorized, "unauthorized", "Not signed in.")
	case errors.Is(err, portal.ErrNotFound), errors.Is(err, commerce.ErrNotFound):
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
	case errors.Is(err, portal.ErrInvalidInput):
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid input.")
	case errors.Is(err, portal.ErrConflict):
		writeError(w, r, http.StatusConflict, "conflict", "Request conflicts with current state.")
	case errors.Is(err, portal.ErrForbiddenState):
		writeError(w, r, http.StatusConflict, "forbidden_state", "This action is not available for the current order state.")
	case errors.Is(err, portal.ErrStoreCreditOff):
		writeError(w, r, http.StatusConflict, "store_credit_disabled", "Store credit is not enabled.")
	default:
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Something went wrong.")
	}
}

func (h *PortalHandlers) Require(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("storeSlug")
		raw := h.portalSessionRaw(r)
		customer, _, store, err := h.Portal.AuthenticateForStore(r.Context(), slug, raw)
		if err != nil {
			writeError(w, r, http.StatusUnauthorized, "unauthorized", "Authentication required.")
			return
		}
		ctx := context.WithValue(r.Context(), ctxKeyPortalCustomerID, customer.ID)
		ctx = context.WithValue(ctx, ctxKeyPortalStoreID, store.ID)
		ctx = context.WithValue(ctx, ctxKeyPortalSession, raw)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *PortalHandlers) RequestMagicLink(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("storeSlug")
	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	_, err := h.Portal.RequestMagicLink(r.Context(), slug, body.Email, ClientIP(r), r.UserAgent())
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeError(w, r, http.StatusGone, "access_link_required", "Use your personal customer portal link.")
}

func (h *PortalHandlers) AuthenticateAccessKey(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("storeSlug")
	var body struct {
		AccessKey string `json:"access_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	key := strings.TrimSpace(body.AccessKey)
	if key == "" {
		key = strings.TrimSpace(r.URL.Query().Get("access_key"))
	}
	res, err := h.Portal.AuthenticateAccessKey(r.Context(), slug, key, ClientIP(r), r.UserAgent())
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	expires := time.Now().UTC().Add(h.Cfg.SessionTTL)
	h.setPortalCookie(w, res.SessionToken, expires)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":            true,
		"customer":      res.Customer,
		"store":         res.Store,
		"redirect_path": res.RedirectPath,
		"session_token": res.SessionToken,
	})
}

func (h *PortalHandlers) PeekMagicLink(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("storeSlug")
	token := r.URL.Query().Get("token")
	if !h.Portal.PeekToken(r.Context(), slug, token) {
		writeError(w, r, http.StatusGone, "invalid_token", "This link has expired.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "valid": true})
}

func (h *PortalHandlers) VerifyMagicLink(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("storeSlug")
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	res, err := h.Portal.VerifyAndConsume(r.Context(), slug, body.Token, ClientIP(r), r.UserAgent())
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	expires := time.Now().UTC().Add(h.Cfg.SessionTTL)
	h.setPortalCookie(w, res.SessionToken, expires)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":            true,
		"customer":      res.Customer,
		"store":         res.Store,
		"redirect_path": res.RedirectPath,
		"session_token": res.SessionToken, // tests / bearer clients
	})
}

func (h *PortalHandlers) Session(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("storeSlug")
	customer, sess, store, err := h.Portal.AuthenticateForStore(r.Context(), slug, h.portalSessionRaw(r))
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, "unauthorized", "Not signed in.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"customer": customer,
		"session":  sess,
		"store":    store,
	})
}

func (h *PortalHandlers) Logout(w http.ResponseWriter, r *http.Request) {
	_ = h.Portal.Logout(r.Context(), h.portalSessionRaw(r))
	h.clearPortalCookie(w)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *PortalHandlers) Dashboard(w http.ResponseWriter, r *http.Request) {
	dash, err := h.Portal.GetDashboard(r.Context(), PortalStoreID(r), PortalCustomerID(r))
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"dashboard": dash})
}

func (h *PortalHandlers) ListWishlist(w http.ResponseWriter, r *http.Request) {
	items, err := h.Portal.Commerce.ListCustomerWishlist(r.Context(), PortalStoreID(r), PortalCustomerID(r))
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *PortalHandlers) ListOrders(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	list, total, err := h.Portal.ListOrders(r.Context(), PortalStoreID(r), PortalCustomerID(r), limit, offset)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"orders": list, "total": total})
}

func (h *PortalHandlers) GetOrder(w http.ResponseWriter, r *http.Request) {
	orderID, err := uuid.Parse(r.PathValue("orderID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	detail, err := h.Portal.GetOrderDetail(r.Context(), PortalStoreID(r), PortalCustomerID(r), orderID)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"detail": detail})
}

func (h *PortalHandlers) GetRefundPreference(w http.ResponseWriter, r *http.Request) {
	pref, err := h.Portal.GetRefundPreference(r.Context(), PortalStoreID(r), PortalCustomerID(r))
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"refund_preference": pref})
}

func (h *PortalHandlers) PutRefundPreference(w http.ResponseWriter, r *http.Request) {
	var body portal.UpsertRefundPreferenceInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	pref, err := h.Portal.UpsertRefundPreference(r.Context(), PortalStoreID(r), PortalCustomerID(r), body)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"refund_preference": pref})
}

func (h *PortalHandlers) GetCredit(w http.ResponseWriter, r *http.Request) {
	if h.Portal.Commerce == nil {
		writeError(w, r, http.StatusConflict, "store_credit_disabled", "Store credit is not enabled.")
		return
	}
	bal, err := h.Portal.Commerce.GetStoreCredit(r.Context(), PortalStoreID(r), PortalCustomerID(r))
	if err != nil {
		if errors.Is(err, commerce.ErrStoreCreditDisabled) {
			writeError(w, r, http.StatusConflict, "store_credit_disabled", "Store credit is not enabled.")
			return
		}
		h.mapErr(w, r, err)
		return
	}
	ledger, err := h.Portal.Commerce.ListStoreCreditLedger(r.Context(), PortalStoreID(r), PortalCustomerID(r), 50)
	if err != nil {
		if errors.Is(err, commerce.ErrStoreCreditDisabled) {
			writeError(w, r, http.StatusConflict, "store_credit_disabled", "Store credit is not enabled.")
			return
		}
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"credit": bal, "ledger": ledger})
}

func (h *PortalHandlers) ListAddresses(w http.ResponseWriter, r *http.Request) {
	list, err := h.Portal.ListAddresses(r.Context(), PortalStoreID(r), PortalCustomerID(r))
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"addresses": list})
}

func (h *PortalHandlers) CreateAddress(w http.ResponseWriter, r *http.Request) {
	var body portal.AddressInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	addr, err := h.Portal.CreateAddress(r.Context(), PortalStoreID(r), PortalCustomerID(r), body)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"address": addr})
}

func (h *PortalHandlers) UpdateAddress(w http.ResponseWriter, r *http.Request) {
	addressID, err := uuid.Parse(r.PathValue("addressID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	var body portal.AddressInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	addr, err := h.Portal.UpdateAddress(r.Context(), PortalStoreID(r), PortalCustomerID(r), addressID, body)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"address": addr})
}

func (h *PortalHandlers) DeleteAddress(w http.ResponseWriter, r *http.Request) {
	addressID, err := uuid.Parse(r.PathValue("addressID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	if err := h.Portal.DeleteAddress(r.Context(), PortalStoreID(r), PortalCustomerID(r), addressID); err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *PortalHandlers) RequestAddressChange(w http.ResponseWriter, r *http.Request) {
	orderID, err := uuid.Parse(r.PathValue("orderID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	var body portal.CreateAddressChangeInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	req, err := h.Portal.RequestAddressChange(r.Context(), PortalStoreID(r), PortalCustomerID(r), orderID, body)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"address_change_request": req})
}

func (h *PortalHandlers) ListAddressChanges(w http.ResponseWriter, r *http.Request) {
	list, err := h.Portal.ListCustomerAddressChanges(r.Context(), PortalStoreID(r), PortalCustomerID(r))
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"address_change_requests": list})
}

// Merchant-facing address change queue
func (h *PortalHandlers) MerchantListAddressChanges(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	list, err := h.Portal.ListStoreAddressChanges(r.Context(), StoreID(r), status, limit)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"address_change_requests": list})
}

func (h *PortalHandlers) MerchantReviewAddressChange(w http.ResponseWriter, r *http.Request) {
	requestID, err := uuid.Parse(r.PathValue("requestID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	var body portal.ReviewAddressChangeInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	req, err := h.Portal.ReviewAddressChange(r.Context(), StoreID(r), requestID, UserID(r), body)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"address_change_request": req})
}
