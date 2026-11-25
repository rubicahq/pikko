package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"

	"github.com/rubicahq/pikko/services/api/internal/entitlements"
)

type BillingHandlers struct {
	Ent *entitlements.Service
}

func (h *BillingHandlers) ListPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := h.Ent.ListPlans(r.Context(), true)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Unable to list plans.")
		return
	}
	// Public pricing: hide internal feature keys that are false-only noise? Keep full matrix for honesty.
	writeJSON(w, http.StatusOK, map[string]any{
		"plans": plans,
		"flags": mustFlags(h, r),
		"testing_mode": mustFlags(h, r).TestBypassEnabled,
		"currency":     "IDR",
		"note":         "Annual billing includes 2 months free. No platform transaction fee.",
	})
}

func mustFlags(h *BillingHandlers, r *http.Request) entitlements.Flags {
	f, err := h.Ent.LoadFlags(r.Context())
	if err != nil {
		return entitlements.DefaultFlags()
	}
	return f
}

func (h *BillingHandlers) GetStoreSubscription(w http.ResponseWriter, r *http.Request) {
	storeID, err := uuid.Parse(r.PathValue("storeID"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid store id.")
		return
	}
	snap, err := h.Ent.Snapshot(r.Context(), storeID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Unable to load subscription.")
		return
	}
	writeJSON(w, http.StatusOK, snap)
}

func (h *BillingHandlers) Activate(w http.ResponseWriter, r *http.Request) {
	storeID, err := uuid.Parse(r.PathValue("storeID"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid store id.")
		return
	}
	userID := UserID(r)
	var in entitlements.ActivateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	sub, err := h.Ent.ActivateTestBypass(r.Context(), storeID, userID, in)
	if writeEntitlementErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"subscription": sub})
}

func (h *BillingHandlers) Change(w http.ResponseWriter, r *http.Request) {
	storeID, err := uuid.Parse(r.PathValue("storeID"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid store id.")
		return
	}
	userID := UserID(r)
	var body struct {
		entitlements.ActivateInput
		Force bool `json:"force"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	sub, over, err := h.Ent.ChangePlan(r.Context(), storeID, userID, body.ActivateInput, body.Force)
	if errors.Is(err, entitlements.ErrDowngradeBlocked) {
		writeJSON(w, http.StatusConflict, map[string]any{
			"error": map[string]any{
				"code":       "downgrade_over_limit",
				"message":    "Current usage exceeds the target plan limits. Existing data is kept; new creates will be blocked until usage is reduced. Pass force=true to confirm downgrade.",
				"request_id": RequestID(r),
				"details":    map[string]any{"over_limit": over},
			},
		})
		return
	}
	if writeEntitlementErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"subscription": sub, "over_limit": over})
}

func (h *BillingHandlers) Usage(w http.ResponseWriter, r *http.Request) {
	storeID, err := uuid.Parse(r.PathValue("storeID"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid store id.")
		return
	}
	snap, err := h.Ent.Snapshot(r.Context(), storeID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Unable to load usage.")
		return
	}
	period := ""
	if len(snap.Usage) > 0 {
		period = snap.Usage[0].PeriodYM
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"usage":  snap.Usage,
		"limits": snap.Limits,
		"period": period,
	})
}

func writeEntitlementErr(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	var lim *entitlements.LimitError
	if errors.As(err, &lim) {
		writeJSON(w, http.StatusPaymentRequired, map[string]any{
			"error": map[string]any{
				"code":       "plan_limit_exceeded",
				"message":    lim.Error(),
				"request_id": RequestID(r),
				"details": map[string]any{
					"limit_key": lim.LimitKey,
					"used":      lim.Used,
					"limit":     lim.Limit,
				},
			},
		})
		return true
	}
	var feat *entitlements.FeatureError
	if errors.As(err, &feat) {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"error": map[string]any{
				"code":       "feature_not_entitled",
				"message":    feat.Error(),
				"request_id": RequestID(r),
				"details":    map[string]any{"feature_key": feat.FeatureKey},
			},
		})
		return true
	}
	switch {
	case errors.Is(err, entitlements.ErrSubscriptionRequired):
		writeError(w, r, http.StatusPaymentRequired, "subscription_required", "An active subscription is required.")
	case errors.Is(err, entitlements.ErrTestBypassDisabled):
		writeError(w, r, http.StatusForbidden, "test_bypass_disabled", "Testing activation is disabled.")
	case errors.Is(err, entitlements.ErrPlanChangeDisabled):
		writeError(w, r, http.StatusForbidden, "plan_change_disabled", "Plan changes are disabled.")
	case errors.Is(err, entitlements.ErrRealBillingDisabled):
		writeError(w, r, http.StatusForbidden, "real_billing_disabled", "Paid billing is not enabled yet.")
	case errors.Is(err, entitlements.ErrTrialDisabled):
		writeError(w, r, http.StatusForbidden, "trial_disabled", "Commercial trial is not enabled.")
	case errors.Is(err, entitlements.ErrInvalidInput), errors.Is(err, entitlements.ErrNotFound):
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid plan or billing cycle.")
	case errors.Is(err, entitlements.ErrSubscriptionsOff):
		writeError(w, r, http.StatusServiceUnavailable, "subscriptions_disabled", "Subscriptions are temporarily disabled.")
	default:
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Something went wrong.")
	}
	return true
}

// Admin handlers (token-gated)

type AdminBillingHandlers struct {
	Ent *entitlements.Service
	Tok string
}

func (h *AdminBillingHandlers) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	tok := h.Tok
	if tok == "" {
		tok = os.Getenv("SUBSCRIPTION_ADMIN_TOKEN")
	}
	if tok == "" {
		writeError(w, r, http.StatusForbidden, "forbidden", "Admin token not configured.")
		return false
	}
	got := r.Header.Get("X-Admin-Token")
	if got == "" {
		got = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	}
	if got != tok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized", "Invalid admin token.")
		return false
	}
	return true
}

func (h *AdminBillingHandlers) ListPlans(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	plans, err := h.Ent.ListPlans(r.Context(), false)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Unable to list plans.")
		return
	}
	flags, _ := h.Ent.LoadFlags(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"plans": plans, "flags": flags})
}

func (h *AdminBillingHandlers) PatchPlan(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	code := r.PathValue("planCode")
	var patch entitlements.AdminPlanPatch
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	plan, err := h.Ent.AdminUpdatePlan(r.Context(), code, patch, nil)
	if writeEntitlementErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"plan": plan})
}

func (h *AdminBillingHandlers) PatchFlags(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	var flags entitlements.Flags
	if err := json.NewDecoder(r.Body).Decode(&flags); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	out, err := h.Ent.AdminSetFlags(r.Context(), flags, nil)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Unable to update flags.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"flags": out})
}

func (h *AdminBillingHandlers) SetStoreOverride(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	storeID, err := uuid.Parse(r.PathValue("storeID"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid store id.")
		return
	}
	var body struct {
		FeatureKey *string `json:"feature_key"`
		Enabled    *bool   `json:"enabled"`
		LimitKey   *string `json:"limit_key"`
		LimitValue *int64  `json:"limit_value"`
		Reason     string  `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	if body.FeatureKey != nil && body.Enabled != nil {
		if err := h.Ent.SetStoreOverride(r.Context(), storeID, *body.FeatureKey, *body.Enabled, body.Reason, nil); err != nil {
			writeError(w, r, http.StatusInternalServerError, "internal_error", "Unable to set override.")
			return
		}
	}
	if body.LimitKey != nil && body.LimitValue != nil {
		if err := h.Ent.SetStoreLimitOverride(r.Context(), storeID, *body.LimitKey, *body.LimitValue, body.Reason, nil); err != nil {
			writeError(w, r, http.StatusInternalServerError, "internal_error", "Unable to set limit override.")
			return
		}
	}
	snap, _ := h.Ent.Snapshot(r.Context(), storeID)
	writeJSON(w, http.StatusOK, snap)
}
