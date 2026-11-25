package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/rubicahq/pikko/services/api/internal/commerce"
	"github.com/rubicahq/pikko/services/api/internal/entitlements"
)

func (h *CommerceHandlers) PackingQueue(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	groupBy := strings.TrimSpace(strings.ToLower(q.Get("group_by")))
	if groupBy == "customer" && h.Ent != nil {
		if err := h.Ent.RequireFeature(r.Context(), StoreID(r), entitlements.FeaturePackingByCustomer); err != nil {
			writeEntitlementErr(w, r, err)
			return
		}
	}
	if h.Ent != nil {
		if err := h.Ent.RequireFeature(r.Context(), StoreID(r), entitlements.FeaturePacking); err != nil {
			writeEntitlementErr(w, r, err)
			return
		}
	}
	out, err := h.Commerce.ListPackingQueue(r.Context(), StoreID(r), groupBy, q.Get("status"))
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *CommerceHandlers) PackingChecklist(w http.ResponseWriter, r *http.Request) {
	orderID, err := uuid.Parse(r.PathValue("orderID"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid order id.")
		return
	}
	out, err := h.Commerce.GetPackingChecklist(r.Context(), StoreID(r), orderID)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"packing": out})
}

func (h *CommerceHandlers) UpdatePackingChecklist(w http.ResponseWriter, r *http.Request) {
	orderID, err := uuid.Parse(r.PathValue("orderID"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid order id.")
		return
	}
	var in commerce.UpdatePackingChecklistInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	out, err := h.Commerce.UpdatePackingChecklist(r.Context(), StoreID(r), orderID, UserID(r), in)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"packing": out})
}

func (h *CommerceHandlers) CompletePacking(w http.ResponseWriter, r *http.Request) {
	orderID, err := uuid.Parse(r.PathValue("orderID"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid order id.")
		return
	}
	var in commerce.CompletePackingInput
	_ = json.NewDecoder(r.Body).Decode(&in)
	out, err := h.Commerce.CompletePacking(r.Context(), StoreID(r), orderID, UserID(r), in)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"packing": out})
}

func (h *CommerceHandlers) ListShipments(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var orderID *uuid.UUID
	if v := q.Get("order_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid order_id.")
			return
		}
		orderID = &id
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	list, total, err := h.Commerce.ListShipments(r.Context(), StoreID(r), orderID, limit, offset)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"shipments": list, "total": total})
}

func (h *CommerceHandlers) CreateShipment(w http.ResponseWriter, r *http.Request) {
	var in commerce.CreateShipmentInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	sh, err := h.Commerce.CreateShipment(r.Context(), StoreID(r), UserID(r), in)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"shipment": sh})
}

func (h *CommerceHandlers) GetShipment(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("shipmentID"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid shipment id.")
		return
	}
	sh, err := h.Commerce.GetShipment(r.Context(), StoreID(r), id)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"shipment": sh})
}

func (h *CommerceHandlers) ListOrderShipments(w http.ResponseWriter, r *http.Request) {
	orderID, err := uuid.Parse(r.PathValue("orderID"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid order id.")
		return
	}
	list, err := h.Commerce.ListOrderShipments(r.Context(), StoreID(r), orderID)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"shipments": list})
}
