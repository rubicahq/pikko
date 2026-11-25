package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/rubicahq/pikko/services/api/internal/analytics"
	"github.com/rubicahq/pikko/services/api/internal/commerce"
	"github.com/rubicahq/pikko/services/api/internal/notifications"
	"github.com/rubicahq/pikko/services/api/internal/portal"
)

type Phase5Handlers struct {
	Notifications *notifications.Service
	Analytics     *analytics.Service
	Commerce      *commerce.Service
	Portal        *portal.Service
}

func (h *Phase5Handlers) ListNotifications(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	unreadOnly := q.Get("unread") == "1" || q.Get("unread") == "true"
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	list, total, err := h.Notifications.List(r.Context(), StoreID(r), unreadOnly, limit, offset)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "list_failed", "Unable to list notifications.")
		return
	}
	unread, _ := h.Notifications.UnreadCount(r.Context(), StoreID(r))
	writeJSON(w, http.StatusOK, map[string]any{
		"notifications": list, "total": total, "unread_count": unread,
	})
}

func (h *Phase5Handlers) UnreadCount(w http.ResponseWriter, r *http.Request) {
	n, err := h.Notifications.UnreadCount(r.Context(), StoreID(r))
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "count_failed", "Unable to count notifications.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"unread_count": n})
}

func (h *Phase5Handlers) MarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("notificationID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Notification not found.")
		return
	}
	if err := h.Notifications.MarkRead(r.Context(), StoreID(r), id); err != nil {
		if errors.Is(err, notifications.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "Notification not found.")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "update_failed", "Unable to mark notification read.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Phase5Handlers) MarkAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	n, err := h.Notifications.MarkAllRead(r.Context(), StoreID(r))
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "update_failed", "Unable to mark notifications read.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"updated": n})
}

func (h *Phase5Handlers) GetNotificationPreferences(w http.ResponseWriter, r *http.Request) {
	p, err := h.Notifications.GetPreferences(r.Context(), StoreID(r))
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "get_failed", "Unable to load preferences.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"preferences": p})
}

func (h *Phase5Handlers) PatchNotificationPreferences(w http.ResponseWriter, r *http.Request) {
	var in notifications.UpdatePreferencesInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid JSON body.")
		return
	}
	p, err := h.Notifications.UpdatePreferences(r.Context(), StoreID(r), in)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "update_failed", "Unable to update preferences.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"preferences": p})
}

func (h *Phase5Handlers) AnalyticsOverview(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	ov, err := h.Analytics.Overview(r.Context(), StoreID(r), q.Get("from"), q.Get("to"), q.Get("tz"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Unable to load analytics for the given range.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"analytics": ov})
}

func (h *Phase5Handlers) RecordAnalyticsEvent(w http.ResponseWriter, r *http.Request) {
	var in analytics.EventInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid JSON body.")
		return
	}
	if in.EventName == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "event_name is required.")
		return
	}
	if u := UserID(r); u != uuid.Nil {
		uid := u
		in.UserID = &uid
	}
	if err := h.Analytics.RecordEvent(r.Context(), StoreID(r), in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Unable to record analytics event.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true})
}

func (h *Phase5Handlers) CreateCustomer(w http.ResponseWriter, r *http.Request) {
	var in commerce.CreateCustomerInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid JSON body.")
		return
	}
	res, err := h.Commerce.CreateCustomer(r.Context(), StoreID(r), in)
	if err != nil {
		if errors.Is(err, commerce.ErrInvalidInput) {
			writeError(w, r, http.StatusBadRequest, "invalid_input", "Name and a valid Indonesian WhatsApp/phone are required.")
			return
		}
		if errors.Is(err, commerce.ErrConflict) {
			writeError(w, r, http.StatusConflict, "conflict", "A customer with this phone already exists in this store.")
			return
		}
		if errors.Is(err, commerce.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "Store not found.")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "create_failed", "Unable to create customer.")
		return
	}
	writeJSON(w, http.StatusCreated, res)
}

func (h *Phase5Handlers) GetCustomer(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("customerID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Customer not found.")
		return
	}
	c, err := h.Commerce.GetCustomer(r.Context(), StoreID(r), id)
	if err != nil {
		if errors.Is(err, commerce.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "Customer not found.")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "get_failed", "Unable to load customer.")
		return
	}
	out := map[string]any{"customer": c}
	if h.Commerce != nil {
		if portalURL, slug, urlErr := h.Commerce.CustomerPortalURL(r.Context(), StoreID(r), id); urlErr == nil {
			if slug != "" {
				out["store_slug"] = slug
			}
			if portalURL != "" {
				out["portal_url"] = portalURL
			}
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Phase5Handlers) RotateCustomerPortalAccess(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("customerID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Customer not found.")
		return
	}
	portalURL, slug, err := h.Commerce.RotatePortalAccessSecret(r.Context(), StoreID(r), id)
	if err != nil {
		if errors.Is(err, commerce.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "Customer not found.")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "rotate_failed", "Unable to rotate portal link.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":         true,
		"portal_url": portalURL,
		"store_slug": slug,
	})
}

func (h *Phase5Handlers) PatchCustomer(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("customerID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Customer not found.")
		return
	}
	var in commerce.UpdateCustomerInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid JSON body.")
		return
	}
	c, err := h.Commerce.UpdateCustomer(r.Context(), StoreID(r), id, in)
	if err != nil {
		if errors.Is(err, commerce.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "Customer not found.")
			return
		}
		if errors.Is(err, commerce.ErrInvalidInput) {
			writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid customer update.")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "update_failed", "Unable to update customer.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"customer": c})
}

func (h *Phase5Handlers) ListBatchVariants(w http.ResponseWriter, r *http.Request) {
	batchID, err := uuid.Parse(r.PathValue("batchID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Batch not found.")
		return
	}
	productID, err := uuid.Parse(r.PathValue("productID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Product not found.")
		return
	}
	list, err := h.Commerce.ListBatchVariants(r.Context(), StoreID(r), batchID, productID)
	if err != nil {
		if errors.Is(err, commerce.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "Batch not found.")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "list_failed", "Unable to list batch variants.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"variants": list})
}

func (h *Phase5Handlers) AssignBatchVariant(w http.ResponseWriter, r *http.Request) {
	batchID, err := uuid.Parse(r.PathValue("batchID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Batch not found.")
		return
	}
	productID, err := uuid.Parse(r.PathValue("productID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Product not found.")
		return
	}
	var in commerce.AssignVariantInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid JSON body.")
		return
	}
	v, err := h.Commerce.AssignBatchVariant(r.Context(), StoreID(r), batchID, productID, in)
	if err != nil {
		if errors.Is(err, commerce.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "Product or variant not found on batch.")
			return
		}
		if errors.Is(err, commerce.ErrInvalidInput) {
			writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid variant assignment.")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "assign_failed", "Unable to assign variant.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"variant": v})
}

func (h *Phase5Handlers) RemoveBatchVariant(w http.ResponseWriter, r *http.Request) {
	batchID, err := uuid.Parse(r.PathValue("batchID"))
	productID, err2 := uuid.Parse(r.PathValue("productID"))
	variantID, err3 := uuid.Parse(r.PathValue("variantID"))
	if err != nil || err2 != nil || err3 != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Variant not found.")
		return
	}
	if err := h.Commerce.RemoveBatchVariant(r.Context(), StoreID(r), batchID, productID, variantID); err != nil {
		if errors.Is(err, commerce.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "Variant not found.")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "delete_failed", "Unable to remove variant.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
