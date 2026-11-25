package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/rubicahq/pikko/services/api/internal/commerce"
	"github.com/rubicahq/pikko/services/api/internal/entitlements"
)

func (h *CommerceHandlers) CreateAdminOrder(w http.ResponseWriter, r *http.Request) {
	if h.Ent != nil {
		if err := h.Ent.RequireFeature(r.Context(), StoreID(r), entitlements.FeatureOrders); err != nil {
			writeEntitlementErr(w, r, err)
			return
		}
		if err := h.Ent.RequireLimit(r.Context(), StoreID(r), entitlements.LimitOrdersPerMonth, 1); err != nil {
			writeEntitlementErr(w, r, err)
			return
		}
	}
	var in commerce.AdminCreateOrderInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	res, err := h.Commerce.CreateAdminOrder(r.Context(), StoreID(r), UserID(r), in)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	if h.Ent != nil {
		_ = h.Ent.IncrementUsage(r.Context(), StoreID(r), entitlements.LimitOrdersPerMonth, 1)
	}
	writeJSON(w, http.StatusCreated, res)
}

func (h *CommerceHandlers) ValidateOrderLines(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Items []commerce.CartLineInput `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	res, err := h.Commerce.BulkValidateLines(r.Context(), StoreID(r), body.Items)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *CommerceHandlers) SuggestOrderCatalog(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	list, err := h.Commerce.SuggestCatalog(r.Context(), StoreID(r), r.URL.Query().Get("q"), limit)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"suggestions": list})
}

func (h *CommerceHandlers) ListOrderDrafts(w http.ResponseWriter, r *http.Request) {
	list, err := h.Commerce.ListOrderDrafts(r.Context(), StoreID(r), UserID(r))
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"drafts": list})
}

func (h *CommerceHandlers) GetOrderDraft(w http.ResponseWriter, r *http.Request) {
	draftID, err := uuid.Parse(r.PathValue("draftID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	d, err := h.Commerce.GetOrderDraft(r.Context(), StoreID(r), UserID(r), draftID)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"draft": d})
}

func (h *CommerceHandlers) SaveOrderDraft(w http.ResponseWriter, r *http.Request) {
	var in commerce.SaveDraftInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	var draftID *uuid.UUID
	if raw := r.PathValue("draftID"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
			return
		}
		draftID = &id
	}
	d, err := h.Commerce.SaveOrderDraft(r.Context(), StoreID(r), UserID(r), draftID, in)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	status := http.StatusOK
	if draftID == nil {
		status = http.StatusCreated
	}
	writeJSON(w, status, map[string]any{"draft": d})
}

func (h *CommerceHandlers) ValidateOrderDraft(w http.ResponseWriter, r *http.Request) {
	draftID, err := uuid.Parse(r.PathValue("draftID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	d, vres, err := h.Commerce.ValidateOrderDraft(r.Context(), StoreID(r), UserID(r), draftID)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"draft": d, "validation": vres})
}

func (h *CommerceHandlers) DeleteOrderDraft(w http.ResponseWriter, r *http.Request) {
	draftID, err := uuid.Parse(r.PathValue("draftID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	if err := h.Commerce.DeleteOrderDraft(r.Context(), StoreID(r), UserID(r), draftID); err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *CommerceHandlers) OrderCSVTemplate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="pikko-order-import-template.csv"`)
	_, _ = w.Write([]byte(commerce.OrderCSVTemplate))
}

func (h *CommerceHandlers) PreviewOrderCSV(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(3 << 20); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Expected multipart form with file.")
		return
	}
	file, hdr, err := r.FormFile("file")
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "file is required.")
		return
	}
	defer file.Close()
	var colMap *commerce.OrderCSVColumnMap
	if raw := r.FormValue("column_map"); raw != "" {
		var cm commerce.OrderCSVColumnMap
		if err := json.Unmarshal([]byte(raw), &cm); err == nil {
			colMap = &cm
		}
	}
	preview, err := h.Commerce.PreviewOrderCSV(r.Context(), StoreID(r), file, hdr.Size, colMap)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

func (h *CommerceHandlers) ImportOrderCSV(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(3 << 20); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Expected multipart form with file.")
		return
	}
	file, hdr, err := r.FormFile("file")
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "file is required.")
		return
	}
	defer file.Close()
	var colMap *commerce.OrderCSVColumnMap
	if raw := r.FormValue("column_map"); raw != "" {
		var cm commerce.OrderCSVColumnMap
		if err := json.Unmarshal([]byte(raw), &cm); err == nil {
			colMap = &cm
		}
	}
	idem := r.FormValue("idempotency_key")
	res, err := h.Commerce.ImportOrderCSV(r.Context(), StoreID(r), UserID(r), file, hdr.Size, colMap, idem)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *CommerceHandlers) ListOrderRevisions(w http.ResponseWriter, r *http.Request) {
	orderID, err := uuid.Parse(r.PathValue("orderID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	list, err := h.Commerce.ListOrderRevisions(r.Context(), StoreID(r), orderID)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"revisions": list})
}

func (h *CommerceHandlers) ListOrderItemEvents(w http.ResponseWriter, r *http.Request) {
	orderID, err := uuid.Parse(r.PathValue("orderID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	list, err := h.Commerce.ListOrderItemEvents(r.Context(), StoreID(r), orderID)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": list})
}

func (h *CommerceHandlers) AdjustOrderItemQty(w http.ResponseWriter, r *http.Request) {
	orderID, err := uuid.Parse(r.PathValue("orderID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	itemID, err := uuid.Parse(r.PathValue("itemID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	var in commerce.AdjustItemQtyInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	o, err := h.Commerce.AdjustOrderItemQty(r.Context(), StoreID(r), orderID, itemID, UserID(r), in)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"order": o})
}
