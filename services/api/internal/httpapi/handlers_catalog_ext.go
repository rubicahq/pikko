package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/rubicahq/pikko/services/api/internal/catalog"
	"github.com/rubicahq/pikko/services/api/internal/entitlements"
)

// --- Brands ---

func (h *Phase2Handlers) ListBrands(w http.ResponseWriter, r *http.Request) {
	include := r.URL.Query().Get("include_archived") == "1"
	list, err := h.Catalog.ListBrands(r.Context(), StoreID(r), include)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "list_failed", "Unable to list brands.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"brands": list})
}

func (h *Phase2Handlers) CreateBrand(w http.ResponseWriter, r *http.Request) {
	var in catalog.BrandInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	b, err := h.Catalog.CreateBrand(r.Context(), StoreID(r), in)
	if err != nil {
		h.writeCatalogErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"brand": b})
}

func (h *Phase2Handlers) UpdateBrand(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("brandID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	var in catalog.BrandInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	b, err := h.Catalog.UpdateBrand(r.Context(), StoreID(r), id, in)
	if err != nil {
		h.writeCatalogErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"brand": b})
}

// --- Vendors (foundation; Agent 06 expands purchasing) ---

func (h *Phase2Handlers) ListVendors(w http.ResponseWriter, r *http.Request) {
	include := r.URL.Query().Get("include_archived") == "1"
	list, err := h.Catalog.ListVendors(r.Context(), StoreID(r), include)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "list_failed", "Unable to list vendors.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"vendors": list})
}

func (h *Phase2Handlers) CreateVendor(w http.ResponseWriter, r *http.Request) {
	var in catalog.VendorInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	v, err := h.Catalog.CreateVendor(r.Context(), StoreID(r), in)
	if err != nil {
		h.writeCatalogErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"vendor": v})
}

func (h *Phase2Handlers) UpdateVendor(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("vendorID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	var in catalog.VendorInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	v, err := h.Catalog.UpdateVendor(r.Context(), StoreID(r), id, in)
	if err != nil {
		h.writeCatalogErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"vendor": v})
}

func (h *Phase2Handlers) ListProductVendors(w http.ResponseWriter, r *http.Request) {
	productID, err := uuid.Parse(r.PathValue("productID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	list, err := h.Catalog.ListProductVendors(r.Context(), StoreID(r), productID)
	if err != nil {
		h.writeCatalogErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"vendors": list})
}

func (h *Phase2Handlers) LinkProductVendor(w http.ResponseWriter, r *http.Request) {
	productID, err := uuid.Parse(r.PathValue("productID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	var in catalog.LinkVendorInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	link, err := h.Catalog.LinkProductVendor(r.Context(), StoreID(r), productID, in)
	if err != nil {
		h.writeCatalogErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"link": link})
}

func (h *Phase2Handlers) UnlinkProductVendor(w http.ResponseWriter, r *http.Request) {
	productID, err := uuid.Parse(r.PathValue("productID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	vendorID, err := uuid.Parse(r.PathValue("vendorID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	if err := h.Catalog.UnlinkProductVendor(r.Context(), StoreID(r), productID, vendorID); err != nil {
		h.writeCatalogErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// --- Custom attributes ---

func (h *Phase2Handlers) ListAttributeDefinitions(w http.ResponseWriter, r *http.Request) {
	list, err := h.Catalog.ListAttributeDefinitions(r.Context(), StoreID(r))
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "list_failed", "Unable to list attributes.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"attributes": list})
}

func (h *Phase2Handlers) CreateAttributeDefinition(w http.ResponseWriter, r *http.Request) {
	if h.Ent != nil {
		if err := h.Ent.RequireFeature(r.Context(), StoreID(r), entitlements.FeatureProductAttributes); err != nil {
			writeEntitlementErr(w, r, err)
			return
		}
	}
	var in catalog.AttributeDefinitionInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	a, err := h.Catalog.CreateAttributeDefinition(r.Context(), StoreID(r), in)
	if err != nil {
		h.writeCatalogErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"attribute": a})
}

func (h *Phase2Handlers) UpdateAttributeDefinition(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("attributeID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	var in catalog.AttributeDefinitionInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	a, err := h.Catalog.UpdateAttributeDefinition(r.Context(), StoreID(r), id, in)
	if err != nil {
		h.writeCatalogErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"attribute": a})
}

func (h *Phase2Handlers) DeleteAttributeDefinition(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("attributeID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	if err := h.Catalog.DeleteAttributeDefinition(r.Context(), StoreID(r), id); err != nil {
		h.writeCatalogErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Phase2Handlers) SetProductAttributesBulk(w http.ResponseWriter, r *http.Request) {
	productID, err := uuid.Parse(r.PathValue("productID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	var body struct {
		Values []catalog.SetAttributeValueInput `json:"values"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	if err := h.Catalog.SetProductAttributesBulk(r.Context(), StoreID(r), productID, body.Values); err != nil {
		h.writeCatalogErr(w, r, err)
		return
	}
	list, err := h.Catalog.ListProductAttributes(r.Context(), StoreID(r), productID)
	if err != nil {
		h.writeCatalogErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"values": list})
}

func (h *Phase2Handlers) ListProductAttributes(w http.ResponseWriter, r *http.Request) {
	productID, err := uuid.Parse(r.PathValue("productID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	list, err := h.Catalog.ListProductAttributes(r.Context(), StoreID(r), productID)
	if err != nil {
		h.writeCatalogErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"values": list})
}

func (h *Phase2Handlers) SetProductAttribute(w http.ResponseWriter, r *http.Request) {
	productID, err := uuid.Parse(r.PathValue("productID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	var in catalog.SetAttributeValueInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	v, err := h.Catalog.SetProductAttribute(r.Context(), StoreID(r), productID, in)
	if err != nil {
		h.writeCatalogErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"value": v})
}

func (h *Phase2Handlers) DeleteProductAttribute(w http.ResponseWriter, r *http.Request) {
	productID, err := uuid.Parse(r.PathValue("productID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	attrID, err := uuid.Parse(r.PathValue("attributeID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	if err := h.Catalog.DeleteProductAttribute(r.Context(), StoreID(r), productID, attrID); err != nil {
		h.writeCatalogErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
