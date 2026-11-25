package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/rubicahq/pikko/services/api/internal/commerce"
	"github.com/rubicahq/pikko/services/api/internal/entitlements"
)

func (h *CommerceHandlers) mapLiveErr(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, commerce.ErrNotFound):
		writeError(w, r, http.StatusNotFound, "not_found", "Live session or resource not found.")
	case errors.Is(err, commerce.ErrInvalidInput):
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid live shopping input.")
	case errors.Is(err, commerce.ErrConflict):
		writeError(w, r, http.StatusConflict, "conflict", "Live shopping conflict.")
	case errors.Is(err, commerce.ErrLiveSessionClosed):
		writeError(w, r, http.StatusConflict, "session_closed", "Live session is not accepting orders.")
	case errors.Is(err, commerce.ErrLiveQuickAddDisabled):
		writeError(w, r, http.StatusConflict, "quick_add_disabled", "Quick-add is disabled for this live session.")
	case errors.Is(err, commerce.ErrLiveQuotaExceeded):
		writeError(w, r, http.StatusConflict, "quota_exceeded", "Campaign quota exceeded for this product.")
	case errors.Is(err, commerce.ErrInsufficientStock):
		writeError(w, r, http.StatusConflict, "insufficient_stock", "Insufficient stock or product unavailable.")
	case errors.Is(err, commerce.ErrInvalidTransition):
		writeError(w, r, http.StatusConflict, "invalid_transition", "Order cannot accept live quick-add.")
	default:
		writeError(w, r, http.StatusInternalServerError, "live_failed", "Unable to process live shopping request.")
	}
}

func (h *CommerceHandlers) requireLiveShopping(w http.ResponseWriter, r *http.Request) bool {
	if h.Ent == nil {
		return true
	}
	if err := h.Ent.RequireFeature(r.Context(), StoreID(r), entitlements.FeatureLiveShopping); err != nil {
		writeEntitlementErr(w, r, err)
		return false
	}
	return true
}

func (h *CommerceHandlers) ListLiveSessions(w http.ResponseWriter, r *http.Request) {
	if !h.requireLiveShopping(w, r) {
		return
	}
	list, err := h.Commerce.ListLiveSessions(r.Context(), StoreID(r))
	if err != nil {
		h.mapLiveErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": list})
}

func (h *CommerceHandlers) GetLiveSession(w http.ResponseWriter, r *http.Request) {
	if !h.requireLiveShopping(w, r) {
		return
	}
	id, err := uuid.Parse(r.PathValue("campaignID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	sess, err := h.Commerce.GetLiveSession(r.Context(), StoreID(r), id, true)
	if err != nil {
		h.mapLiveErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"session": sess})
}

func (h *CommerceHandlers) LiveOrderFeed(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("campaignID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	var since *time.Time
	if raw := r.URL.Query().Get("since"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid since timestamp.")
			return
		}
		since = &t
	}
	feed, err := h.Commerce.LiveOrderFeed(r.Context(), StoreID(r), id, since, 40)
	if err != nil {
		h.mapLiveErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"feed": feed})
}

func (h *CommerceHandlers) SearchLiveOrders(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("campaignID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	q := r.URL.Query().Get("q")
	list, err := h.Commerce.SearchLiveOrders(r.Context(), StoreID(r), id, q, 12)
	if err != nil {
		h.mapLiveErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"orders": list})
}

func (h *CommerceHandlers) LiveQuickAdd(w http.ResponseWriter, r *http.Request) {
	if !h.requireLiveShopping(w, r) {
		return
	}
	id, err := uuid.Parse(r.PathValue("campaignID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	var in commerce.LiveQuickAddInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	res, err := h.Commerce.LiveQuickAdd(r.Context(), StoreID(r), id, UserID(r), in)
	if err != nil {
		h.mapLiveErr(w, r, err)
		return
	}
	status := http.StatusOK
	if res.CreatedOrder {
		status = http.StatusCreated
	}
	writeJSON(w, status, res)
}

func (h *CommerceHandlers) PublicLiveCampaigns(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	list, err := h.Commerce.ListPublicLiveCampaigns(r.Context(), slug)
	if err != nil {
		if errors.Is(err, commerce.ErrNotFound) || errors.Is(err, commerce.ErrStoreUnavailable) {
			writeError(w, r, http.StatusNotFound, "not_found", "Store not found.")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "live_failed", "Unable to list live sessions.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"lives": list})
}

func (h *CommerceHandlers) PublicLiveCampaign(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	liveSlug := r.PathValue("liveSlug")
	sess, err := h.Commerce.GetPublicLiveCampaign(r.Context(), slug, liveSlug)
	if err != nil {
		if errors.Is(err, commerce.ErrNotFound) || errors.Is(err, commerce.ErrStoreUnavailable) {
			writeError(w, r, http.StatusNotFound, "not_found", "Live session not found.")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "live_failed", "Unable to load live session.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"live": sess})
}
