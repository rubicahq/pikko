package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/rubicahq/pikko/services/api/internal/catalog"
	"github.com/rubicahq/pikko/services/api/internal/flipbook"
	"github.com/rubicahq/pikko/services/api/internal/urls"
)

type FlipbookHandlers struct {
	Flipbook *flipbook.Service
	Catalog  *catalog.Service
	URLs     *urls.Generator
}

func (h *FlipbookHandlers) ListPages(w http.ResponseWriter, r *http.Request) {
	pages, err := h.Flipbook.MerchantList(r.Context(), StoreID(r))
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "list_failed", "Unable to list catalog pages.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"pages": pages})
}

func (h *FlipbookHandlers) Reorder(w http.ResponseWriter, r *http.Request) {
	var in flipbook.ReorderInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	if err := h.Flipbook.Reorder(r.Context(), StoreID(r), in); err != nil {
		if errors.Is(err, flipbook.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "Catalog page not found.")
			return
		}
		if errors.Is(err, flipbook.ErrInvalidInput) {
			writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid page order.")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "reorder_failed", "Unable to reorder pages.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *FlipbookHandlers) Publish(w http.ResponseWriter, r *http.Request) {
	var in struct {
		PageIDs []uuid.UUID `json:"page_ids"`
		Status  string      `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	if in.Status == "" {
		in.Status = "published"
	}
	if err := h.Flipbook.SetPagesStatus(r.Context(), StoreID(r), in.PageIDs, in.Status); err != nil {
		if errors.Is(err, flipbook.ErrInvalidInput) {
			writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid status.")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "publish_failed", "Unable to update page status.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": in.Status})
}

func (h *FlipbookHandlers) CreateHotspot(w http.ResponseWriter, r *http.Request) {
	pageID, err := uuid.Parse(r.PathValue("pageID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Catalog page not found.")
		return
	}
	var in flipbook.HotspotInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	hs, err := h.Flipbook.CreateHotspot(r.Context(), StoreID(r), pageID, in)
	if err != nil {
		if errors.Is(err, flipbook.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "Catalog page not found.")
			return
		}
		if errors.Is(err, flipbook.ErrInvalidInput) {
			writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid hotspot.")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "create_failed", "Unable to create hotspot.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"hotspot": hs})
}

func (h *FlipbookHandlers) UpdateHotspot(w http.ResponseWriter, r *http.Request) {
	hotspotID, err := uuid.Parse(r.PathValue("hotspotID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Hotspot not found.")
		return
	}
	var in flipbook.HotspotInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	hs, err := h.Flipbook.UpdateHotspot(r.Context(), StoreID(r), hotspotID, in)
	if err != nil {
		if errors.Is(err, flipbook.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "Hotspot not found.")
			return
		}
		if errors.Is(err, flipbook.ErrInvalidInput) {
			writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid hotspot.")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "update_failed", "Unable to update hotspot.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"hotspot": hs})
}

func (h *FlipbookHandlers) DeleteHotspot(w http.ResponseWriter, r *http.Request) {
	hotspotID, err := uuid.Parse(r.PathValue("hotspotID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Hotspot not found.")
		return
	}
	if err := h.Flipbook.DeleteHotspot(r.Context(), StoreID(r), hotspotID); err != nil {
		if errors.Is(err, flipbook.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "Hotspot not found.")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "delete_failed", "Unable to delete hotspot.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *FlipbookHandlers) PublicCatalog(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	storeID, storeName, pages, err := h.Flipbook.PublicByStoreSlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, flipbook.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "Catalog not found.")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "catalog_failed", "Unable to load catalog.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"store": map[string]any{
			"id":   storeID,
			"name": storeName,
			"slug": slug,
			"catalog_url": h.URLs.Storefront(slug) + "/catalog",
		},
		"pages": pages,
	})
}

// PublicHotspotProduct returns a published product card payload for flipbook checkout hooks.
func (h *FlipbookHandlers) PublicHotspotProduct(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	productID, err := uuid.Parse(r.PathValue("productID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Product not found.")
		return
	}
	storeID, _, _, err := h.Flipbook.PublicByStoreSlug(r.Context(), slug)
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Store not found.")
		return
	}
	p, err := h.Catalog.GetProduct(r.Context(), storeID, productID, true)
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Product not found.")
		return
	}
	// Strip cost
	p.CostPriceIDR = nil
	writeJSON(w, http.StatusOK, map[string]any{
		"product": p,
		"link_ref": map[string]any{
			"product_id":   p.ID,
			"product_slug": p.Slug,
			"product_url":  h.URLs.Product(slug, p.Slug),
		},
		// Cart is not fully implemented yet — storefront uses this hook + product_url.
		"checkout_hook": "add_to_cart_or_product_link",
	})
}
