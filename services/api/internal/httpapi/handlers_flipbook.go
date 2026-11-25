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
	// Soft-deprecate product hotspots on merchant list for etalase-only flipbooks.
	for i := range pages {
		pages[i].Hotspots = nil
		pages[i].Items = nil
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
	writeError(w, r, http.StatusGone, "catalog_hotspots_disabled",
		"Catalog hotspots are disabled. Flipbooks are display-only etalase.")
}

func (h *FlipbookHandlers) UpdateHotspot(w http.ResponseWriter, r *http.Request) {
	writeError(w, r, http.StatusGone, "catalog_hotspots_disabled",
		"Catalog hotspots are disabled. Flipbooks are display-only etalase.")
}

func (h *FlipbookHandlers) DeleteHotspot(w http.ResponseWriter, r *http.Request) {
	writeError(w, r, http.StatusGone, "catalog_hotspots_disabled",
		"Catalog hotspots are disabled. Flipbooks are display-only etalase.")
}

func (h *FlipbookHandlers) PublicCatalog(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	store, pages, err := h.Flipbook.PublicByStoreSlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, flipbook.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "Catalog not found.")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "catalog_failed", "Unable to load catalog.")
		return
	}
	// Hotspots/items stay in public JSON for scheduled-live wishlist hover on storefront.
	// Cart / PublicHotspotProduct remain retired; AI import stays disabled.
	writeJSON(w, http.StatusOK, map[string]any{
		"store": map[string]any{
			"id":          store.ID,
			"name":        store.Name,
			"slug":        slug,
			"whatsapp":    store.WhatsApp,
			"catalog_url": h.URLs.Storefront(slug) + "/catalog",
		},
		"pages": pages,
	})
}

// PublicCatalogDoc returns the flipbook for one published catalog document.
func (h *FlipbookHandlers) PublicCatalogDoc(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	catalogID, err := uuid.Parse(r.PathValue("catalogID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Catalog not found.")
		return
	}
	store, pages, err := h.Flipbook.PublicByCatalog(r.Context(), slug, catalogID)
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
			"id":          store.ID,
			"name":        store.Name,
			"slug":        slug,
			"whatsapp":    store.WhatsApp,
			"catalog_url": h.URLs.Storefront(slug) + "/catalog/" + catalogID.String(),
		},
		"catalog_id": catalogID,
		"pages":      pages,
	})
}

// PublicHotspotProduct stays retired (no cart / contact checkout hook).
// Live wishlist uses hotspot product_id from the public pages payload instead.
func (h *FlipbookHandlers) PublicHotspotProduct(w http.ResponseWriter, r *http.Request) {
	writeError(w, r, http.StatusGone, "catalog_hotspots_disabled",
		"Catalog product cart actions are disabled. Use live wishlist from hotspot data.")
}
