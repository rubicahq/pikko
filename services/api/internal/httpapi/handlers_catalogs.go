package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/rubicahq/pikko/services/api/internal/catalog"
	"github.com/rubicahq/pikko/services/api/internal/catalogs"
)

type CatalogDocHandlers struct {
	Catalogs *catalogs.Service
}

func (h *CatalogDocHandlers) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.Catalogs.List(r.Context(), StoreID(r))
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "list_failed", "Unable to list catalogs.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"catalogs": list})
}

func (h *CatalogDocHandlers) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("catalogID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Catalog not found.")
		return
	}
	c, err := h.Catalogs.Get(r.Context(), StoreID(r), id)
	if err != nil {
		if errors.Is(err, catalogs.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "Catalog not found.")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "get_failed", "Unable to load catalog.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"catalog": c})
}

func (h *CatalogDocHandlers) Patch(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("catalogID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Catalog not found.")
		return
	}
	var in catalogs.PatchInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	c, err := h.Catalogs.Patch(r.Context(), StoreID(r), id, in)
	if err != nil {
		if errors.Is(err, catalogs.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "Catalog not found.")
			return
		}
		if errors.Is(err, catalogs.ErrInvalidInput) {
			writeError(w, r, http.StatusBadRequest, "invalid_input", err.Error())
			return
		}
		writeError(w, r, http.StatusInternalServerError, "patch_failed", "Unable to update catalog.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"catalog": c})
}

func (h *CatalogDocHandlers) AttachCampaign(w http.ResponseWriter, r *http.Request) {
	catalogID, err := uuid.Parse(r.PathValue("catalogID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Catalog not found.")
		return
	}
	var in struct {
		CampaignID uuid.UUID `json:"campaign_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.CampaignID == uuid.Nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "campaign_id required.")
		return
	}
	if err := h.Catalogs.AttachCampaign(r.Context(), StoreID(r), catalogID, in.CampaignID); err != nil {
		if errors.Is(err, catalogs.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "Catalog not found.")
			return
		}
		if errors.Is(err, catalogs.ErrInvalidInput) {
			writeError(w, r, http.StatusBadRequest, "invalid_input", err.Error())
			return
		}
		writeError(w, r, http.StatusInternalServerError, "attach_failed", "Unable to attach campaign.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *CatalogDocHandlers) DetachCampaign(w http.ResponseWriter, r *http.Request) {
	catalogID, err := uuid.Parse(r.PathValue("catalogID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Catalog not found.")
		return
	}
	campaignID, err := uuid.Parse(r.PathValue("campaignID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Campaign link not found.")
		return
	}
	if err := h.Catalogs.DetachCampaign(r.Context(), StoreID(r), catalogID, campaignID); err != nil {
		if errors.Is(err, catalogs.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "Campaign link not found.")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "detach_failed", "Unable to detach campaign.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *CatalogDocHandlers) AttachPreorder(w http.ResponseWriter, r *http.Request) {
	catalogID, err := uuid.Parse(r.PathValue("catalogID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Catalog not found.")
		return
	}
	var in struct {
		BatchID uuid.UUID `json:"batch_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.BatchID == uuid.Nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "batch_id required.")
		return
	}
	if err := h.Catalogs.AttachPreorder(r.Context(), StoreID(r), catalogID, in.BatchID); err != nil {
		if errors.Is(err, catalogs.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "Catalog not found.")
			return
		}
		if errors.Is(err, catalogs.ErrInvalidInput) {
			writeError(w, r, http.StatusBadRequest, "invalid_input", err.Error())
			return
		}
		writeError(w, r, http.StatusInternalServerError, "attach_failed", "Unable to attach preorder.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *CatalogDocHandlers) DetachPreorder(w http.ResponseWriter, r *http.Request) {
	catalogID, err := uuid.Parse(r.PathValue("catalogID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Catalog not found.")
		return
	}
	batchID, err := uuid.Parse(r.PathValue("batchID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Preorder link not found.")
		return
	}
	if err := h.Catalogs.DetachPreorder(r.Context(), StoreID(r), catalogID, batchID); err != nil {
		if errors.Is(err, catalogs.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "Preorder link not found.")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "detach_failed", "Unable to detach preorder.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *CatalogDocHandlers) ListForCampaign(w http.ResponseWriter, r *http.Request) {
	campaignID, err := uuid.Parse(r.PathValue("campaignID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Campaign not found.")
		return
	}
	list, err := h.Catalogs.ListForCampaign(r.Context(), StoreID(r), campaignID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "list_failed", "Unable to list catalogs.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"catalogs": list})
}

func (h *CatalogDocHandlers) ListForPreorder(w http.ResponseWriter, r *http.Request) {
	batchID, err := uuid.Parse(r.PathValue("batchID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Preorder batch not found.")
		return
	}
	list, err := h.Catalogs.ListForPreorder(r.Context(), StoreID(r), batchID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "list_failed", "Unable to list catalogs.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"catalogs": list})
}

type ProductRelationHandlers struct {
	Catalog *catalog.Service
}

func (h *ProductRelationHandlers) List(w http.ResponseWriter, r *http.Request) {
	productID, err := uuid.Parse(r.PathValue("productID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Product not found.")
		return
	}
	list, err := h.Catalog.ListProductRelations(r.Context(), StoreID(r), productID)
	if err != nil {
		if errors.Is(err, catalog.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "Product not found.")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "list_failed", "Unable to list relations.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"relations": list})
}

func (h *ProductRelationHandlers) Create(w http.ResponseWriter, r *http.Request) {
	productID, err := uuid.Parse(r.PathValue("productID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Product not found.")
		return
	}
	var in catalog.RelateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	var userID *uuid.UUID
	if uid := UserID(r); uid != uuid.Nil {
		userID = &uid
	}
	rel, err := h.Catalog.RelateProduct(r.Context(), StoreID(r), productID, userID, in)
	if err != nil {
		if errors.Is(err, catalog.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "Product not found.")
			return
		}
		if errors.Is(err, catalog.ErrInvalidInput) {
			writeError(w, r, http.StatusBadRequest, "invalid_input", err.Error())
			return
		}
		writeError(w, r, http.StatusInternalServerError, "relate_failed", "Unable to create relation.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"relation": rel})
}

func (h *ProductRelationHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	relationID, err := uuid.Parse(r.PathValue("relationID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Relation not found.")
		return
	}
	if err := h.Catalog.DeleteProductRelation(r.Context(), StoreID(r), relationID); err != nil {
		if errors.Is(err, catalog.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "Relation not found.")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "delete_failed", "Unable to delete relation.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
