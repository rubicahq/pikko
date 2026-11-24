package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/rubicahq/pikko/services/api/internal/commerce"
)

func (h *CommerceHandlers) PurchasingEligibility(w http.ResponseWriter, r *http.Request) {
	list, err := h.Commerce.ListPurchasingEligibility(r.Context(), StoreID(r))
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"eligibility": list})
}

func (h *CommerceHandlers) StorePurchasing(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := commerce.PurchasingFilter{GroupBy: q.Get("group_by")}
	if v := q.Get("batch_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid batch_id.")
			return
		}
		f.BatchID = &id
	}
	if v := q.Get("campaign_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid campaign_id.")
			return
		}
		f.CampaignID = &id
	}
	if v := q.Get("vendor_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid vendor_id.")
			return
		}
		f.VendorID = &id
	}
	if v := q.Get("brand_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid brand_id.")
			return
		}
		f.BrandID = &id
	}
	list, err := h.Commerce.AggregatePurchasing(r.Context(), StoreID(r), f)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"purchasing": list})
}

func (h *CommerceHandlers) UnavailableImpact(w http.ResponseWriter, r *http.Request) {
	var in commerce.UnavailableImpactFilter
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	impact, err := h.Commerce.AnalyzeUnavailableImpact(r.Context(), StoreID(r), in)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"impact": impact})
}

func (h *CommerceHandlers) MarkOrderItemUnavailable(w http.ResponseWriter, r *http.Request) {
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
	var in commerce.MarkUnavailableInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	order, err := h.Commerce.MarkOrderItemUnavailable(r.Context(), StoreID(r), orderID, itemID, UserID(r), in)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"order": order})
}

func (h *CommerceHandlers) ListReceiving(w http.ResponseWriter, r *http.Request) {
	var batchID *uuid.UUID
	if v := r.URL.Query().Get("batch_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid batch_id.")
			return
		}
		batchID = &id
	}
	list, err := h.Commerce.ListReceiving(r.Context(), StoreID(r), batchID)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"receiving": list})
}

func (h *CommerceHandlers) GetReceiving(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("receivingID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	rec, err := h.Commerce.GetReceiving(r.Context(), StoreID(r), id)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"receiving": rec})
}

func (h *CommerceHandlers) CreateReceiving(w http.ResponseWriter, r *http.Request) {
	var in commerce.CreateReceivingInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	rec, err := h.Commerce.CreateReceiving(r.Context(), StoreID(r), UserID(r), in)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"receiving": rec})
}

func (h *CommerceHandlers) UpdateReceivingLine(w http.ResponseWriter, r *http.Request) {
	receivingID, err := uuid.Parse(r.PathValue("receivingID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	lineID, err := uuid.Parse(r.PathValue("lineID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	var in commerce.UpdateReceivingLineInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	rec, err := h.Commerce.UpdateReceivingLine(r.Context(), StoreID(r), receivingID, lineID, in)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"receiving": rec})
}
