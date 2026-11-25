package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/rubicahq/pikko/services/api/internal/campaigns"
	"github.com/rubicahq/pikko/services/api/internal/entitlements"
)

type CampaignHandlers struct {
	Campaigns *campaigns.Service
	Ent       *entitlements.Service
}

func (h *CampaignHandlers) mapErr(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, campaigns.ErrNotFound):
		writeError(w, r, http.StatusNotFound, "not_found", "Campaign not found.")
	case errors.Is(err, campaigns.ErrInvalidInput):
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid campaign input.")
	case errors.Is(err, campaigns.ErrConflict):
		writeError(w, r, http.StatusConflict, "conflict", "Campaign conflict.")
	default:
		writeError(w, r, http.StatusInternalServerError, "campaign_failed", "Unable to process campaign request.")
	}
}

func (h *CampaignHandlers) List(w http.ResponseWriter, r *http.Request) {
	ctype := r.URL.Query().Get("type")
	list, err := h.Campaigns.List(r.Context(), StoreID(r), ctype)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"campaigns": list})
}

func (h *CampaignHandlers) Create(w http.ResponseWriter, r *http.Request) {
	if h.Ent != nil {
		if err := h.Ent.RequireFeature(r.Context(), StoreID(r), entitlements.FeatureCampaigns); err != nil {
			writeEntitlementErr(w, r, err)
			return
		}
		if err := h.Ent.RequireLimit(r.Context(), StoreID(r), entitlements.LimitCampaigns, 1); err != nil {
			writeEntitlementErr(w, r, err)
			return
		}
	}
	var in campaigns.CreateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	c, err := h.Campaigns.Create(r.Context(), StoreID(r), in)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"campaign": c})
}

func (h *CampaignHandlers) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("campaignID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	c, err := h.Campaigns.Get(r.Context(), StoreID(r), id, true)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"campaign": c})
}

func (h *CampaignHandlers) Patch(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("campaignID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	var in campaigns.PatchInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	c, err := h.Campaigns.Patch(r.Context(), StoreID(r), id, in)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"campaign": c})
}

func (h *CampaignHandlers) AssignProduct(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("campaignID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	var in campaigns.AssignProductInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	p, err := h.Campaigns.AssignProduct(r.Context(), StoreID(r), id, in)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"product": p})
}

func (h *CampaignHandlers) RemoveProduct(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("campaignID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	productID, err := uuid.Parse(r.PathValue("productID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	if err := h.Campaigns.RemoveProduct(r.Context(), StoreID(r), id, productID); err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
