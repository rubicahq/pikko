package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/rubicahq/pikko/services/api/internal/commerce"
)

// --- Preorder batches (merchant) ---

func (h *CommerceHandlers) ListBatches(w http.ResponseWriter, r *http.Request) {
	list, err := h.Commerce.ListBatches(r.Context(), StoreID(r))
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"batches": list})
}

func (h *CommerceHandlers) CreateBatch(w http.ResponseWriter, r *http.Request) {
	var in commerce.CreateBatchInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	b, err := h.Commerce.CreateBatch(r.Context(), StoreID(r), in)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	if h.Campaigns != nil {
		_ = h.Campaigns.EnsureLinkedFromBatch(r.Context(), StoreID(r), b.ID)
	}
	writeJSON(w, http.StatusCreated, map[string]any{"batch": b})
}

func (h *CommerceHandlers) GetBatch(w http.ResponseWriter, r *http.Request) {
	batchID, err := uuid.Parse(r.PathValue("batchID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	b, err := h.Commerce.GetBatch(r.Context(), StoreID(r), batchID, true)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"batch": b})
}

func (h *CommerceHandlers) PatchBatch(w http.ResponseWriter, r *http.Request) {
	batchID, err := uuid.Parse(r.PathValue("batchID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	var in commerce.PatchBatchInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	b, err := h.Commerce.UpdateBatch(r.Context(), StoreID(r), batchID, in)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	if h.Campaigns != nil {
		_ = h.Campaigns.EnsureLinkedFromBatch(r.Context(), StoreID(r), batchID)
		_ = h.Campaigns.SyncFromBatch(r.Context(), StoreID(r), batchID)
	}
	writeJSON(w, http.StatusOK, map[string]any{"batch": b})
}

func (h *CommerceHandlers) AssignBatchProduct(w http.ResponseWriter, r *http.Request) {
	batchID, err := uuid.Parse(r.PathValue("batchID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	var in commerce.AssignProductInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	p, err := h.Commerce.AssignBatchProduct(r.Context(), StoreID(r), batchID, in)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	if h.Campaigns != nil {
		_ = h.Campaigns.EnsureLinkedFromBatch(r.Context(), StoreID(r), batchID)
		_ = h.Campaigns.SyncBatchProduct(r.Context(), StoreID(r), batchID, in.ProductID)
	}
	writeJSON(w, http.StatusOK, map[string]any{"product": p})
}

func (h *CommerceHandlers) RemoveBatchProduct(w http.ResponseWriter, r *http.Request) {
	batchID, err := uuid.Parse(r.PathValue("batchID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	productID, err := uuid.Parse(r.PathValue("productID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	if err := h.Commerce.RemoveBatchProduct(r.Context(), StoreID(r), batchID, productID); err != nil {
		h.mapErr(w, r, err)
		return
	}
	if h.Campaigns != nil {
		_ = h.Campaigns.RemoveBatchProductMirror(r.Context(), StoreID(r), batchID, productID)
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *CommerceHandlers) BatchPurchasing(w http.ResponseWriter, r *http.Request) {
	batchID, err := uuid.Parse(r.PathValue("batchID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	list, err := h.Commerce.GetPurchasingList(r.Context(), StoreID(r), batchID)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"purchasing": list})
}

func (h *CommerceHandlers) UpdateBatchPurchasing(w http.ResponseWriter, r *http.Request) {
	batchID, err := uuid.Parse(r.PathValue("batchID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	var in commerce.UpdatePurchasingInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	if err := h.Commerce.UpdatePurchasing(r.Context(), StoreID(r), batchID, UserID(r), in); err != nil {
		h.mapErr(w, r, err)
		return
	}
	list, err := h.Commerce.GetPurchasingList(r.Context(), StoreID(r), batchID)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"purchasing": list})
}

func (h *CommerceHandlers) AllocateBatchItem(w http.ResponseWriter, r *http.Request) {
	batchID, err := uuid.Parse(r.PathValue("batchID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	var in commerce.AllocateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	if err := h.Commerce.AllocatePurchased(r.Context(), StoreID(r), batchID, in); err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *CommerceHandlers) ListBatchOrders(w http.ResponseWriter, r *http.Request) {
	batchID, err := uuid.Parse(r.PathValue("batchID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	list, total, err := h.Commerce.ListBatchOrders(r.Context(), StoreID(r), batchID, limit, offset)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"orders": list, "total": total})
}

// --- Business rules ---

func (h *CommerceHandlers) ListRules(w http.ResponseWriter, r *http.Request) {
	list, err := h.Commerce.ListBusinessRules(r.Context(), StoreID(r))
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rules": list})
}

func (h *CommerceHandlers) CreateRule(w http.ResponseWriter, r *http.Request) {
	var in commerce.UpsertRuleInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	rule, err := h.Commerce.CreateBusinessRule(r.Context(), StoreID(r), UserID(r), in)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"rule": rule})
}

func (h *CommerceHandlers) UpdateRule(w http.ResponseWriter, r *http.Request) {
	ruleID, err := uuid.Parse(r.PathValue("ruleID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	var in commerce.UpsertRuleInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	rule, err := h.Commerce.UpdateBusinessRule(r.Context(), StoreID(r), ruleID, UserID(r), in)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rule": rule})
}

func (h *CommerceHandlers) DeleteRule(w http.ResponseWriter, r *http.Request) {
	ruleID, err := uuid.Parse(r.PathValue("ruleID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	if err := h.Commerce.DeleteBusinessRule(r.Context(), StoreID(r), ruleID); err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *CommerceHandlers) PreviewRules(w http.ResponseWriter, r *http.Request) {
	var in commerce.RulePreviewInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	eval, err := h.Commerce.PreviewRules(r.Context(), StoreID(r), in)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"preview": eval})
}

// --- Public preorders ---

func (h *CommerceHandlers) PublicBatches(w http.ResponseWriter, r *http.Request) {
	list, err := h.Commerce.PublicBatches(r.Context(), r.PathValue("slug"))
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"batches": list})
}

func (h *CommerceHandlers) PublicBatch(w http.ResponseWriter, r *http.Request) {
	b, err := h.Commerce.PublicBatch(r.Context(), r.PathValue("slug"), r.PathValue("batchSlug"))
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"batch": b})
}

func (h *CommerceHandlers) PublicProductBatches(w http.ResponseWriter, r *http.Request) {
	batches, err := h.Commerce.PublicProductOpenBatches(r.Context(), r.PathValue("slug"), r.PathValue("productSlug"))
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"batches": batches})
}
