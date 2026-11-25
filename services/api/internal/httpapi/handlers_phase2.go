package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/rubicahq/pikko/services/api/internal/catalog"
	"github.com/rubicahq/pikko/services/api/internal/entitlements"
	"github.com/rubicahq/pikko/services/api/internal/files"
	"github.com/rubicahq/pikko/services/api/internal/rbac"
	"github.com/rubicahq/pikko/services/api/internal/storage"
	"github.com/rubicahq/pikko/services/api/internal/storefront"
	"github.com/rubicahq/pikko/services/api/internal/themes"
	"github.com/rubicahq/pikko/services/api/internal/urls"
)

type Phase2Handlers struct {
	Storefront *storefront.Service
	Catalog    *catalog.Service
	Files      *files.Service
	URLs       *urls.Generator
	Ent        *entitlements.Service
}

func (h *Phase2Handlers) ListThemes(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"themes": themes.All()})
}

func (h *Phase2Handlers) GetStorefront(w http.ResponseWriter, r *http.Request) {
	channel := r.URL.Query().Get("channel")
	if channel == "" {
		channel = "draft"
	}
	cfg, err := h.Storefront.Get(r.Context(), StoreID(r), channel)
	if err != nil {
		if errors.Is(err, storefront.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "Storefront configuration not found.")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "storefront_failed", "Unable to load storefront.")
		return
	}
	featured, _ := h.Storefront.ListFeaturedProductIDs(r.Context(), StoreID(r), channel)
	writeJSON(w, http.StatusOK, map[string]any{"storefront": cfg, "featured_product_ids": featured})
}

func (h *Phase2Handlers) PatchStorefront(w http.ResponseWriter, r *http.Request) {
	var in storefront.PatchInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	cfg, err := h.Storefront.PatchDraft(r.Context(), StoreID(r), in)
	if err != nil {
		if errors.Is(err, storefront.ErrInvalidInput) {
			writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid storefront fields.")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "storefront_failed", "Unable to update storefront.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"storefront": cfg})
}

func (h *Phase2Handlers) PublishStorefront(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.Storefront.Publish(r.Context(), StoreID(r), UserID(r))
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "publish_failed", "Unable to publish storefront.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"storefront": cfg})
}

func (h *Phase2Handlers) DiscardStorefront(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.Storefront.DiscardDraft(r.Context(), StoreID(r))
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "discard_failed", "Unable to discard draft.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"storefront": cfg})
}

func (h *Phase2Handlers) SetFeatured(w http.ResponseWriter, r *http.Request) {
	var in storefront.FeaturedUpdate
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	if err := h.Storefront.SetFeatured(r.Context(), StoreID(r), in); err != nil {
		if errors.Is(err, storefront.ErrInvalidInput) {
			writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid featured selections.")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "featured_failed", "Unable to update featured items.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Phase2Handlers) UploadFile(w http.ResponseWriter, r *http.Request) {
	if h.Ent != nil {
		// Block when already at/over storage capacity (gauge).
		if err := h.Ent.RequireLimit(r.Context(), StoreID(r), entitlements.LimitStorageGB, 1); err != nil {
			writeEntitlementErr(w, r, err)
			return
		}
	}
	const maxMem = 8 << 20
	if err := r.ParseMultipartForm(maxMem); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid multipart upload.")
		return
	}
	purpose := r.FormValue("purpose")
	// Payment/refund evidence must not be planted via catalog.write upload (SEC-02/04).
	if storage.IsPrivateUploadPurpose(purpose) {
		writeError(w, r, http.StatusForbidden, "forbidden", "Use the dedicated evidence upload path for this file type.")
		return
	}
	file, hdr, err := r.FormFile("file")
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "File is required.")
		return
	}
	defer file.Close()
	// Storage GB is a gauge; RequireLimit(..., 0) ensures used <= limit before upload.
	// After upload the next create/upload will re-check.
	f, err := h.Files.Upload(r.Context(), StoreID(r), UserID(r), purpose, hdr.Filename, hdr.Header.Get("Content-Type"), file, hdr.Size)
	if err != nil {
		if errors.Is(err, files.ErrInvalidInput) || strings.Contains(err.Error(), "invalid input") {
			writeError(w, r, http.StatusBadRequest, "invalid_input", err.Error())
			return
		}
		writeError(w, r, http.StatusInternalServerError, "upload_failed", "Unable to upload file.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"file": f})
}

func (h *Phase2Handlers) GetFile(w http.ResponseWriter, r *http.Request) {
	fileID, err := uuid.Parse(r.PathValue("fileID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "File not found.")
		return
	}
	storeID := StoreID(r)
	// Merchant route always has store auth. Public file serving uses public handler.
	f, rc, err := h.Files.Open(r.Context(), storeID, fileID)
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "File not found.")
		return
	}
	defer rc.Close()
	role := MembershipRole(r)
	if !merchantMayReadFilePurpose(role, f.Purpose) {
		writeError(w, r, http.StatusForbidden, "forbidden", "You do not have permission for this file.")
		return
	}
	w.Header().Set("Content-Type", f.ContentType)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	_, _ = io.Copy(w, rc)
}

// merchantMayReadFilePurpose gates private purposes behind orders/finance/catalog perms (SEC-02/04).
func merchantMayReadFilePurpose(role, purpose string) bool {
	switch purpose {
	case "payment_evidence", "refund_evidence":
		return rbac.HasPermission(role, rbac.PermOrdersPayments) ||
			rbac.HasPermission(role, rbac.PermOrdersRead) ||
			rbac.HasPermission(role, rbac.PermFinanceRead)
	case "catalog_pdf", "catalog_csv":
		return rbac.HasPermission(role, rbac.PermCatalogRead)
	default:
		// Branding / product assets: any store member already passed membership check.
		return true
	}
}

func (h *Phase2Handlers) ListCategories(w http.ResponseWriter, r *http.Request) {
	list, err := h.Catalog.ListCategories(r.Context(), StoreID(r))
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "list_failed", "Unable to list categories.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"categories": list})
}

func (h *Phase2Handlers) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var in catalog.CategoryInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	c, err := h.Catalog.CreateCategory(r.Context(), StoreID(r), in)
	if err != nil {
		h.writeCatalogErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"category": c})
}

func (h *Phase2Handlers) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("categoryID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Category not found.")
		return
	}
	var in catalog.CategoryInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	c, err := h.Catalog.UpdateCategory(r.Context(), StoreID(r), id, in)
	if err != nil {
		h.writeCatalogErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"category": c})
}

func (h *Phase2Handlers) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("categoryID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Category not found.")
		return
	}
	if err := h.Catalog.DeleteCategory(r.Context(), StoreID(r), id); err != nil {
		h.writeCatalogErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Phase2Handlers) ListProducts(w http.ResponseWriter, r *http.Request) {
	f := catalog.ProductListFilter{
		Status: r.URL.Query().Get("status"),
		Query:  r.URL.Query().Get("q"),
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		f.Limit, _ = strconv.Atoi(v)
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		f.Offset, _ = strconv.Atoi(v)
	}
	if v := r.URL.Query().Get("category_id"); v != "" {
		id, err := uuid.Parse(v)
		if err == nil {
			f.CategoryID = &id
		}
	}
	list, total, err := h.Catalog.ListProducts(r.Context(), StoreID(r), f, false)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "list_failed", "Unable to list products.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"products": list, "total": total})
}

func (h *Phase2Handlers) CreateProduct(w http.ResponseWriter, r *http.Request) {
	if h.Ent != nil {
		if err := h.Ent.RequireFeature(r.Context(), StoreID(r), entitlements.FeatureProducts); err != nil {
			writeEntitlementErr(w, r, err)
			return
		}
		if err := h.Ent.RequireLimit(r.Context(), StoreID(r), entitlements.LimitActiveProducts, 1); err != nil {
			writeEntitlementErr(w, r, err)
			return
		}
	}
	var in catalog.ProductWriteInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	p, err := h.Catalog.CreateProduct(r.Context(), StoreID(r), in)
	if err != nil {
		h.writeCatalogErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"product": p})
}

func (h *Phase2Handlers) GetProduct(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("productID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Product not found.")
		return
	}
	p, err := h.Catalog.GetProduct(r.Context(), StoreID(r), id, false)
	if err != nil {
		h.writeCatalogErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"product": p})
}

func (h *Phase2Handlers) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("productID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Product not found.")
		return
	}
	var in catalog.ProductWriteInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	p, err := h.Catalog.UpdateProduct(r.Context(), StoreID(r), id, in)
	if err != nil {
		h.writeCatalogErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"product": p})
}

func (h *Phase2Handlers) PatchProductStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("productID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Product not found.")
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	p, err := h.Catalog.SetProductStatus(r.Context(), StoreID(r), id, body.Status)
	if err != nil {
		h.writeCatalogErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"product": p})
}

func (h *Phase2Handlers) ArchiveProduct(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("productID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Product not found.")
		return
	}
	if err := h.Catalog.DeleteProduct(r.Context(), StoreID(r), id); err != nil {
		h.writeCatalogErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Phase2Handlers) ListCollections(w http.ResponseWriter, r *http.Request) {
	list, err := h.Catalog.ListCollections(r.Context(), StoreID(r), false)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "list_failed", "Unable to list collections.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"collections": list})
}

func (h *Phase2Handlers) CreateCollection(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string      `json:"name"`
		Slug        string      `json:"slug"`
		Description *string     `json:"description"`
		ProductIDs  []uuid.UUID `json:"product_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	c, err := h.Catalog.CreateCollection(r.Context(), StoreID(r), body.Name, body.Slug, body.Description, body.ProductIDs)
	if err != nil {
		h.writeCatalogErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"collection": c})
}

func (h *Phase2Handlers) SetupChecklist(w http.ResponseWriter, r *http.Request) {
	items, err := h.Catalog.SetupChecklist(r.Context(), StoreID(r))
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "checklist_failed", "Unable to load checklist.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"checklist": items})
}

func (h *Phase2Handlers) writeCatalogErr(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, catalog.ErrNotFound):
		writeError(w, r, http.StatusNotFound, "not_found", "Resource not found.")
	case errors.Is(err, catalog.ErrInvalidInput):
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid input.")
	case errors.Is(err, catalog.ErrConflict):
		writeError(w, r, http.StatusConflict, "conflict", "Resource already exists.")
	default:
		writeError(w, r, http.StatusInternalServerError, "catalog_failed", "Catalog operation failed.")
	}
}

// --- Public endpoints ---

func (h *Phase2Handlers) PublicStore(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	ps, err := h.Storefront.PublicBySlug(r.Context(), slug)
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Store not found.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"store":          ps,
		"storefront_url": h.URLs.Storefront(ps.Slug),
	})
}

func (h *Phase2Handlers) PublicProducts(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	ps, err := h.Storefront.PublicBySlug(r.Context(), slug)
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Store not found.")
		return
	}
	f := catalog.ProductListFilter{Query: r.URL.Query().Get("q"), Limit: 48}
	if v := r.URL.Query().Get("category_id"); v != "" {
		id, err := uuid.Parse(v)
		if err == nil {
			f.CategoryID = &id
		}
	}
	list, total, err := h.Catalog.ListProducts(r.Context(), ps.ID, f, true)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "list_failed", "Unable to list products.")
		return
	}
	for i := range list {
		catalog.StripPrivate(&list[i])
	}
	writeJSON(w, http.StatusOK, map[string]any{"products": list, "total": total})
}

func (h *Phase2Handlers) PublicProduct(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	productSlug := r.PathValue("productSlug")
	ps, err := h.Storefront.PublicBySlug(r.Context(), slug)
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Store not found.")
		return
	}
	p, err := h.Catalog.GetProductBySlug(r.Context(), ps.ID, productSlug, true)
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Product not found.")
		return
	}
	catalog.StripPrivate(p)
	writeJSON(w, http.StatusOK, map[string]any{
		"product":     p,
		"product_url": h.URLs.Product(ps.Slug, p.Slug),
		// link_ref: stable fields for flipbook hotspots / catalog page anchors
		"link_ref": p.LinkRef(),
	})
}

func (h *Phase2Handlers) PublicCategories(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	ps, err := h.Storefront.PublicBySlug(r.Context(), slug)
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Store not found.")
		return
	}
	list, err := h.Catalog.ListCategories(r.Context(), ps.ID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "list_failed", "Unable to list categories.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"categories": list})
}

func (h *Phase2Handlers) PublicFile(w http.ResponseWriter, r *http.Request) {
	// GET /api/v1/public/files/{storeID}/{fileID} — published store assets / product images only (SEC-04).
	storeID, err := uuid.Parse(r.PathValue("storeID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "File not found.")
		return
	}
	fileID, err := uuid.Parse(r.PathValue("fileID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "File not found.")
		return
	}
	var pub bool
	err = h.Catalog.DB.QueryRow(r.Context(), `
		SELECT publication_status='published' AND archived_at IS NULL FROM stores WHERE id=$1
	`, storeID).Scan(&pub)
	if err != nil || !pub {
		writeError(w, r, http.StatusNotFound, "not_found", "File not found.")
		return
	}
	f, rc, err := h.Files.Open(r.Context(), storeID, fileID)
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "File not found.")
		return
	}
	defer rc.Close()
	if !storage.IsPublicPurpose(f.Purpose) {
		// Indistinguishable 404 — do not confirm private evidence/PDF exists.
		writeError(w, r, http.StatusNotFound, "not_found", "File not found.")
		return
	}
	w.Header().Set("Content-Type", f.ContentType)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = io.Copy(w, rc)
}
