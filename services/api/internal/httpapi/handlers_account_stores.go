package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/rubicahq/pikko/services/api/internal/account"
	"github.com/rubicahq/pikko/services/api/internal/config"
	"github.com/rubicahq/pikko/services/api/internal/rbac"
	"github.com/rubicahq/pikko/services/api/internal/stores"
	"github.com/rubicahq/pikko/services/api/internal/urls"
	"github.com/rubicahq/pikko/services/api/internal/validate"
)

type AccountHandlers struct {
	Account *account.Service
}

func (h *AccountHandlers) Get(w http.ResponseWriter, r *http.Request) {
	a, err := h.Account.Get(r.Context(), UserID(r))
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "account_failed", "Unable to load account.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"account": a})
}

func (h *AccountHandlers) Patch(w http.ResponseWriter, r *http.Request) {
	var in account.PatchInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	a, err := h.Account.Patch(r.Context(), UserID(r), in)
	if err != nil {
		if errors.Is(err, account.ErrInvalidInput) {
			writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid account fields.")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "account_failed", "Unable to update account.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"account": a})
}

type StoreHandlers struct {
	Stores *stores.Service
	URLs   *urls.Generator
	Cfg    *config.Config
}

func (h *StoreHandlers) decorate(st *stores.Store) {
	st.StorefrontURL = h.URLs.Storefront(st.Slug)
}

func (h *StoreHandlers) List(w http.ResponseWriter, r *http.Request) {
	includeArchived := r.URL.Query().Get("include_archived") == "true"
	list, err := h.Stores.ListForUser(r.Context(), UserID(r), includeArchived)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "list_failed", "Unable to list stores.")
		return
	}
	for i := range list {
		h.decorate(&list[i])
		list[i].Permissions = permissionStrings(list[i].Role)
	}
	writeJSON(w, http.StatusOK, map[string]any{"stores": list})
}

func (h *StoreHandlers) Create(w http.ResponseWriter, r *http.Request) {
	var in stores.CreateDirectInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	st, err := h.Stores.CreateStore(r.Context(), UserID(r), in)
	if err != nil {
		h.writeStoreErr(w, r, err)
		return
	}
	h.decorate(st)
	writeJSON(w, http.StatusCreated, map[string]any{"store": st})
}

func (h *StoreHandlers) Get(w http.ResponseWriter, r *http.Request) {
	st, err := h.Stores.GetAuthorized(r.Context(), UserID(r), StoreID(r))
	if err != nil {
		h.writeStoreErr(w, r, err)
		return
	}
	h.decorate(st)
	perms := permissionStrings(st.Role)
	st.Permissions = perms
	writeJSON(w, http.StatusOK, map[string]any{
		"store":       st,
		"permissions": perms,
	})
}

func permissionStrings(role string) []string {
	perms := rbac.PermissionsFor(role)
	out := make([]string, len(perms))
	for i, p := range perms {
		out[i] = string(p)
	}
	return out
}

func (h *StoreHandlers) Patch(w http.ResponseWriter, r *http.Request) {
	var in stores.PatchInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	st, err := h.Stores.Patch(r.Context(), UserID(r), StoreID(r), in)
	if err != nil {
		h.writeStoreErr(w, r, err)
		return
	}
	h.decorate(st)
	writeJSON(w, http.StatusOK, map[string]any{"store": st})
}

func (h *StoreHandlers) Archive(w http.ResponseWriter, r *http.Request) {
	st, err := h.Stores.Archive(r.Context(), UserID(r), StoreID(r))
	if err != nil {
		h.writeStoreErr(w, r, err)
		return
	}
	h.decorate(st)
	writeJSON(w, http.StatusOK, map[string]any{"store": st})
}

func (h *StoreHandlers) CheckSlug(w http.ResponseWriter, r *http.Request) {
	slug := r.URL.Query().Get("slug")
	ok, err := h.Stores.CheckSlugAvailable(r.Context(), slug)
	if err != nil {
		if errors.Is(err, stores.ErrInvalidInput) {
			writeJSON(w, http.StatusOK, map[string]any{"available": false, "reason": "invalid"})
			return
		}
		writeError(w, r, http.StatusInternalServerError, "slug_check_failed", "Unable to check slug.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"available": ok, "slug": slug})
}

func (h *StoreHandlers) SuggestSlug(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if !validate.ValidateStoreName(name) {
		suggested := validate.SuggestSlug(name)
		writeJSON(w, http.StatusOK, map[string]any{"slug": suggested})
		return
	}
	slug, err := h.Stores.SuggestSlug(r.Context(), name)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"slug": validate.SuggestSlug(name)})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"slug": slug})
}

func (h *StoreHandlers) GetDraft(w http.ResponseWriter, r *http.Request) {
	d, err := h.Stores.GetOrCreateDraft(r.Context(), UserID(r))
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "draft_failed", "Unable to load onboarding.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"draft": d})
}

func (h *StoreHandlers) SaveIdentity(w http.ResponseWriter, r *http.Request) {
	var in stores.IdentityInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	d, err := h.Stores.SaveIdentityStep(r.Context(), UserID(r), in)
	if err != nil {
		h.writeStoreErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"draft": d})
}

func (h *StoreHandlers) SaveSellingModel(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SellingModel string `json:"selling_model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	d, err := h.Stores.SaveSellingModelStep(r.Context(), UserID(r), body.SellingModel)
	if err != nil {
		h.writeStoreErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"draft": d})
}

func (h *StoreHandlers) CompleteOnboarding(w http.ResponseWriter, r *http.Request) {
	st, err := h.Stores.CreateStoreFromDraft(r.Context(), UserID(r))
	if err != nil {
		h.writeStoreErr(w, r, err)
		return
	}
	h.decorate(st)
	writeJSON(w, http.StatusCreated, map[string]any{"store": st})
}

func (h *StoreHandlers) Categories(w http.ResponseWriter, r *http.Request) {
	type item struct {
		Value string `json:"value"`
		Label string `json:"label"`
	}
	items := []item{
		{"personal_shopper_jastip", "Personal shopper / Jastip"},
		{"fashion_and_accessories", "Fashion and accessories"},
		{"beauty_and_skincare", "Beauty and skincare"},
		{"food_and_beverages", "Food and beverages"},
		{"home_and_living", "Home and living"},
		{"baby_and_kids", "Baby and kids"},
		{"electronics_and_gadgets", "Electronics and gadgets"},
		{"handmade_and_crafts", "Handmade and crafts"},
		{"collectibles_and_hobbies", "Collectibles and hobbies"},
		{"health_and_wellness", "Health and wellness"},
		{"merchandise", "Merchandise"},
		{"other", "Other"},
	}
	writeJSON(w, http.StatusOK, map[string]any{"categories": items})
}

func (h *StoreHandlers) writeStoreErr(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, stores.ErrForbidden), errors.Is(err, stores.ErrNotFound):
		writeError(w, r, http.StatusNotFound, "not_found", "Store not found.")
	case errors.Is(err, stores.ErrSlugTaken):
		writeError(w, r, http.StatusConflict, "slug_taken", "This store URL is already taken.")
	case errors.Is(err, stores.ErrInvalidInput), errors.Is(err, stores.ErrNoActiveDraft):
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Check your store details and try again.")
	case errors.Is(err, stores.ErrCannotArchive):
		writeError(w, r, http.StatusConflict, "cannot_archive", "Resolve outstanding transactions before archiving.")
	default:
		writeError(w, r, http.StatusInternalServerError, "store_failed", "Unable to complete store operation.")
	}
}

// Ensure uuid import used for potential future handlers
var _ = uuid.Nil
