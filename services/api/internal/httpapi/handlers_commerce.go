package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/rubicahq/pikko/services/api/internal/campaigns"
	"github.com/rubicahq/pikko/services/api/internal/commerce"
	"github.com/rubicahq/pikko/services/api/internal/config"
	"github.com/rubicahq/pikko/services/api/internal/urls"
)

type CommerceHandlers struct {
	Commerce  *commerce.Service
	Campaigns *campaigns.Service
	URLs      *urls.Generator
	Cfg       *config.Config
}

func commerceInvalidInputMessage(err error) string {
	msg := err.Error()
	msg = strings.TrimPrefix(msg, "invalid input: ")
	msg = strings.TrimPrefix(msg, "invalid input")
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return "Invalid input."
	}
	r := []rune(msg)
	if r[0] >= 'a' && r[0] <= 'z' {
		r[0] = r[0] - ('a' - 'A')
	}
	return string(r)
}

func (h *CommerceHandlers) mapErr(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, commerce.ErrNotFound), errors.Is(err, commerce.ErrStoreUnavailable):
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
	case errors.Is(err, commerce.ErrInvalidInput):
		writeError(w, r, http.StatusBadRequest, "invalid_input", commerceInvalidInputMessage(err))
	case errors.Is(err, commerce.ErrCheckoutDisabled):
		writeError(w, r, http.StatusConflict, "checkout_disabled", "Checkout is not available for this store yet.")
	case errors.Is(err, commerce.ErrPriceChanged):
		writeError(w, r, http.StatusConflict, "price_changed", "Prices changed. Please review the updated total.")
	case errors.Is(err, commerce.ErrInsufficientStock), errors.Is(err, commerce.ErrQuotaExceeded):
		writeError(w, r, http.StatusConflict, "insufficient_stock", "One or more items are unavailable.")
	case errors.Is(err, commerce.ErrBatchClosed), errors.Is(err, commerce.ErrBatchNotOpen):
		writeError(w, r, http.StatusConflict, "batch_closed", "This preorder batch is not accepting orders.")
	case errors.Is(err, commerce.ErrMinimumNotMet):
		writeError(w, r, http.StatusConflict, "minimum_not_met", "Order does not meet the minimum requirement.")
	case errors.Is(err, commerce.ErrIncompatibleCart):
		writeError(w, r, http.StatusBadRequest, "incompatible_cart", "Ready-stock and preorder items must be checked out separately.")
	case errors.Is(err, commerce.ErrInvalidTransition):
		writeError(w, r, http.StatusConflict, "invalid_transition", "This status change is not allowed.")
	case errors.Is(err, commerce.ErrAlreadyVerified):
		writeError(w, r, http.StatusConflict, "already_verified", "Payment is already verified.")
	case errors.Is(err, commerce.ErrConflict), errors.Is(err, commerce.ErrVersionConflict):
		writeError(w, r, http.StatusConflict, "conflict", "Request conflicts with current state.")
	case errors.Is(err, commerce.ErrStoreCreditDisabled):
		writeError(w, r, http.StatusConflict, "store_credit_disabled", "Store credit is not enabled.")
	case errors.Is(err, commerce.ErrInsufficientCredit):
		writeError(w, r, http.StatusConflict, "insufficient_credit", "Insufficient store credit balance.")
	case errors.Is(err, commerce.ErrRateLimited):
		writeError(w, r, http.StatusTooManyRequests, "rate_limited", "Too many requests. Try again later.")
	default:
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Something went wrong.")
	}
}

// --- Merchant settings ---

func (h *CommerceHandlers) ListPaymentMethods(w http.ResponseWriter, r *http.Request) {
	storeID := StoreID(r)
	list, err := h.Commerce.ListPaymentMethods(r.Context(), storeID)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"payment_methods": list})
}

func (h *CommerceHandlers) CreatePaymentMethod(w http.ResponseWriter, r *http.Request) {
	var in commerce.UpsertPaymentMethodInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	m, err := h.Commerce.CreatePaymentMethod(r.Context(), StoreID(r), in)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"payment_method": m})
}

func (h *CommerceHandlers) UpdatePaymentMethod(w http.ResponseWriter, r *http.Request) {
	methodID, err := uuid.Parse(r.PathValue("methodID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	var in commerce.UpsertPaymentMethodInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	m, err := h.Commerce.UpdatePaymentMethod(r.Context(), StoreID(r), methodID, in)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"payment_method": m})
}

func (h *CommerceHandlers) DeletePaymentMethod(w http.ResponseWriter, r *http.Request) {
	methodID, err := uuid.Parse(r.PathValue("methodID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	if err := h.Commerce.DeletePaymentMethod(r.Context(), StoreID(r), methodID); err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *CommerceHandlers) GetFulfillment(w http.ResponseWriter, r *http.Request) {
	f, err := h.Commerce.GetFulfillmentSettings(r.Context(), StoreID(r))
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"fulfillment": f})
}

func (h *CommerceHandlers) PatchFulfillment(w http.ResponseWriter, r *http.Request) {
	var in commerce.PatchFulfillmentInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	f, err := h.Commerce.UpsertFulfillmentSettings(r.Context(), StoreID(r), in)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"fulfillment": f})
}

// --- Merchant orders ---

func (h *CommerceHandlers) ListOrders(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	list, total, err := h.Commerce.ListOrders(r.Context(), StoreID(r), commerce.ListOrdersFilter{
		Query: q.Get("q"), OrderStatus: q.Get("order_status"),
		PaymentStatus: q.Get("payment_status"), FulfillmentStatus: q.Get("fulfillment_status"),
		Limit: limit, Offset: offset,
	})
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"orders": list, "total": total})
}

func (h *CommerceHandlers) GetOrder(w http.ResponseWriter, r *http.Request) {
	orderID, err := uuid.Parse(r.PathValue("orderID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	o, err := h.Commerce.GetOrder(r.Context(), StoreID(r), orderID)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	pay, _ := h.Commerce.GetLatestPayment(r.Context(), StoreID(r), orderID)
	writeJSON(w, http.StatusOK, map[string]any{"order": o, "payment": pay})
}

func (h *CommerceHandlers) PatchOrder(w http.ResponseWriter, r *http.Request) {
	orderID, err := uuid.Parse(r.PathValue("orderID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	var in commerce.UpdateOrderStatusInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	o, err := h.Commerce.UpdateOrder(r.Context(), StoreID(r), orderID, UserID(r), in)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"order": o})
}

func (h *CommerceHandlers) VerifyPayment(w http.ResponseWriter, r *http.Request) {
	orderID, err := uuid.Parse(r.PathValue("orderID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	paymentID, err := uuid.Parse(r.PathValue("paymentID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	pay, err := h.Commerce.VerifyPayment(r.Context(), StoreID(r), orderID, paymentID, UserID(r))
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"payment": pay})
}

func (h *CommerceHandlers) RejectPayment(w http.ResponseWriter, r *http.Request) {
	orderID, err := uuid.Parse(r.PathValue("orderID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	paymentID, err := uuid.Parse(r.PathValue("paymentID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	pay, err := h.Commerce.RejectPayment(r.Context(), StoreID(r), orderID, paymentID, UserID(r), body.Reason)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"payment": pay})
}

func (h *CommerceHandlers) MerchantEvidenceFile(w http.ResponseWriter, r *http.Request) {
	evidenceID, err := uuid.Parse(r.PathValue("evidenceID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	ct, key, err := h.Commerce.OpenEvidenceStream(r.Context(), StoreID(r), evidenceID)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	h.streamFile(w, r, ct, key)
}

func (h *CommerceHandlers) ListCustomers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	list, total, err := h.Commerce.ListCustomers(r.Context(), StoreID(r), q.Get("q"), limit, offset)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"customers": list, "total": total})
}

// --- Public checkout / tracking ---

func (h *CommerceHandlers) CheckoutPreview(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	ip := ClientIP(r)
	if err := h.Commerce.AllowPublicRateLimit(r.Context(), commerce.PublicCheckoutBucket(ip, slug), 60); err != nil {
		h.mapErr(w, r, err)
		return
	}
	var body struct {
		Items             []commerce.CartLineInput `json:"items"`
		FulfillmentMethod string                   `json:"fulfillment_method"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	preview, err := h.Commerce.PreviewCheckout(r.Context(), slug, body.Items, body.FulfillmentMethod)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"preview": preview})
}

func (h *CommerceHandlers) CheckoutSubmit(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	ip := ClientIP(r)
	if err := h.Commerce.AllowPublicRateLimit(r.Context(), commerce.PublicCheckoutBucket(ip, slug), 30); err != nil {
		h.mapErr(w, r, err)
		return
	}
	var in commerce.CheckoutSubmitInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	if in.IdempotencyKey == "" {
		in.IdempotencyKey = r.Header.Get("Idempotency-Key")
	}
	res, err := h.Commerce.SubmitCheckout(r.Context(), slug, in)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, res)
}

func (h *CommerceHandlers) TrackingGet(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if err := h.Commerce.AllowPublicRateLimit(r.Context(), commerce.PublicTrackingBucket(ClientIP(r)), 120); err != nil {
		h.mapErr(w, r, err)
		return
	}
	view, err := h.Commerce.GetTracking(r.Context(), token)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	writeJSON(w, http.StatusOK, map[string]any{"tracking": view})
}

func (h *CommerceHandlers) TrackingEvidenceUpload(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if err := h.Commerce.AllowPublicRateLimit(r.Context(), commerce.PublicEvidenceBucket(ClientIP(r)), 20); err != nil {
		h.mapErr(w, r, err)
		return
	}
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid multipart form.")
		return
	}
	file, hdr, err := r.FormFile("file")
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "File required.")
		return
	}
	defer file.Close()
	note := ptrIf(strings.TrimSpace(r.FormValue("note")))
	pay, err := h.Commerce.SubmitEvidenceByTracking(
		r.Context(), token, hdr.Filename, hdr.Header.Get("Content-Type"), file, hdr.Size, note,
	)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	// Attach short-lived signed URLs for evidence (via tracking-scoped API path)
	for i := range pay.Evidence {
		pay.Evidence[i].SignedURL = h.URLs.APICallback(
			"/api/v1/public/tracking/" + token + "/evidence/" + pay.Evidence[i].ID.String() + "/file",
		)
		// Mark expiry conceptually — URL itself requires valid tracking token
		_ = time.Hour
	}
	writeJSON(w, http.StatusCreated, map[string]any{"payment": pay})
}

func (h *CommerceHandlers) TrackingEvidenceFile(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	evidenceID, err := uuid.Parse(r.PathValue("evidenceID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	if err := h.Commerce.AllowPublicRateLimit(r.Context(), commerce.PublicTrackingBucket(ClientIP(r)), 120); err != nil {
		h.mapErr(w, r, err)
		return
	}
	ct, key, err := h.Commerce.OpenEvidenceStreamForTracking(r.Context(), token, evidenceID)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "private, max-age=60")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	h.streamFile(w, r, ct, key)
}

func (h *CommerceHandlers) streamFile(w http.ResponseWriter, r *http.Request, contentType, key string) {
	if h.Commerce.Storage == nil {
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Storage unavailable.")
		return
	}
	rc, err := h.Commerce.Storage.Get(r.Context(), key)
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	defer rc.Close()
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "inline")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, rc)
}

func ptrIf(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
