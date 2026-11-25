package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/rubicahq/pikko/services/api/internal/commerce"
	"github.com/rubicahq/pikko/services/api/internal/entitlements"
)

func (h *CommerceHandlers) mapFinanceErr(w http.ResponseWriter, r *http.Request, err error) bool {
	switch {
	case errors.Is(err, commerce.ErrStoreCreditDisabled):
		writeError(w, r, http.StatusConflict, "store_credit_disabled", "Store credit is not enabled.")
		return true
	case errors.Is(err, commerce.ErrInsufficientCredit):
		writeError(w, r, http.StatusConflict, "insufficient_credit", "Insufficient store credit balance.")
		return true
	case errors.Is(err, commerce.ErrVersionConflict):
		writeError(w, r, http.StatusConflict, "version_conflict", "Resource was modified concurrently. Retry.")
		return true
	case errors.Is(err, commerce.ErrIdempotencyConflict):
		writeError(w, r, http.StatusOK, "idempotent_replay", "Request already processed.")
		return true
	case errors.Is(err, commerce.ErrAlreadyVerified):
		writeError(w, r, http.StatusConflict, "already_verified", "Payment is already verified.")
		return true
	default:
		return false
	}
}

func (h *CommerceHandlers) FinanceWorkspace(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	settlement := r.URL.Query().Get("settlement_status")
	list, total, err := h.Commerce.ListFinanceWorkspace(r.Context(), StoreID(r), settlement, limit, offset)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"orders": list, "total": total,
		"store_credit_globally_enabled": h.Commerce.StoreCreditGloballyEnabled(),
	})
}

func (h *CommerceHandlers) OrderFinance(w http.ResponseWriter, r *http.Request) {
	orderID, err := uuid.Parse(r.PathValue("orderID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	fin, err := h.Commerce.GetOrderFinance(r.Context(), StoreID(r), orderID)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	payments, _ := h.Commerce.ListOrderPayments(r.Context(), StoreID(r), orderID)
	refunds, _ := h.Commerce.ListOrderRefunds(r.Context(), StoreID(r), orderID)
	writeJSON(w, http.StatusOK, map[string]any{
		"finance": fin, "payments": payments, "refunds": refunds,
	})
}

func (h *CommerceHandlers) ListOrderPayments(w http.ResponseWriter, r *http.Request) {
	orderID, err := uuid.Parse(r.PathValue("orderID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	list, err := h.Commerce.ListOrderPayments(r.Context(), StoreID(r), orderID)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"payments": list})
}

func (h *CommerceHandlers) RecordPayment(w http.ResponseWriter, r *http.Request) {
	var in commerce.RecordPaymentInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	if in.OrderID == uuid.Nil {
		if orderID, err := uuid.Parse(r.PathValue("orderID")); err == nil {
			in.OrderID = orderID
		}
	}
	if h.Ent != nil && strings.EqualFold(strings.TrimSpace(in.PaymentKind), commerce.PaymentKindConsolidated) {
		if err := h.Ent.RequireFeature(r.Context(), StoreID(r), entitlements.FeatureConsolidatedPayments); err != nil {
			writeEntitlementErr(w, r, err)
			return
		}
	}
	pay, err := h.Commerce.RecordPayment(r.Context(), StoreID(r), UserID(r), in)
	if err != nil {
		if h.mapFinanceErr(w, r, err) {
			return
		}
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"payment": pay})
}

func (h *CommerceHandlers) VerifyPaymentExtended(w http.ResponseWriter, r *http.Request) {
	orderID, err := uuid.Parse(r.PathValue("orderID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	paymentID, err := uuid.Parse(r.PathValue("paymentID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	pay, err := h.Commerce.VerifyPaymentExtended(r.Context(), StoreID(r), orderID, paymentID, UserID(r))
	if err != nil {
		if errors.Is(err, commerce.ErrAlreadyVerified) && pay != nil {
			writeJSON(w, http.StatusOK, map[string]any{"payment": pay, "already_verified": true})
			return
		}
		if h.mapFinanceErr(w, r, err) {
			return
		}
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"payment": pay})
}

func (h *CommerceHandlers) ApplyFinancialAdjustment(w http.ResponseWriter, r *http.Request) {
	var in commerce.FinancialAdjustmentInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	fin, err := h.Commerce.ApplyFinancialAdjustment(r.Context(), StoreID(r), UserID(r), in)
	if err != nil {
		if h.mapFinanceErr(w, r, err) {
			return
		}
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"finance": fin})
}

func (h *CommerceHandlers) CreateRefund(w http.ResponseWriter, r *http.Request) {
	var in commerce.CreateRefundInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	if in.OrderID == uuid.Nil {
		if orderID, err := uuid.Parse(r.PathValue("orderID")); err == nil {
			in.OrderID = orderID
		}
	}
	ref, err := h.Commerce.CreateRefund(r.Context(), StoreID(r), UserID(r), in)
	if err != nil {
		if h.mapFinanceErr(w, r, err) {
			return
		}
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"refund": ref})
}

func (h *CommerceHandlers) ProcessRefund(w http.ResponseWriter, r *http.Request) {
	refundID, err := uuid.Parse(r.PathValue("refundID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	var in commerce.ProcessRefundInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	ref, err := h.Commerce.ProcessRefund(r.Context(), StoreID(r), refundID, UserID(r), in)
	if err != nil {
		if h.mapFinanceErr(w, r, err) {
			return
		}
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"refund": ref})
}

func (h *CommerceHandlers) ListDepositPolicies(w http.ResponseWriter, r *http.Request) {
	list, err := h.Commerce.ListDepositPolicies(r.Context(), StoreID(r))
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deposit_policies": list})
}

func (h *CommerceHandlers) CreateDepositPolicy(w http.ResponseWriter, r *http.Request) {
	var in commerce.UpsertDepositPolicyInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	p, err := h.Commerce.CreateDepositPolicy(r.Context(), StoreID(r), in)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"deposit_policy": p})
}

func (h *CommerceHandlers) UpdateDepositPolicy(w http.ResponseWriter, r *http.Request) {
	policyID, err := uuid.Parse(r.PathValue("policyID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	var in commerce.UpsertDepositPolicyInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	p, err := h.Commerce.UpdateDepositPolicy(r.Context(), StoreID(r), policyID, in)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deposit_policy": p})
}

func (h *CommerceHandlers) GetStoreCredit(w http.ResponseWriter, r *http.Request) {
	customerID, err := uuid.Parse(r.PathValue("customerID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
		return
	}
	bal, err := h.Commerce.GetStoreCredit(r.Context(), StoreID(r), customerID)
	if err != nil {
		if h.mapFinanceErr(w, r, err) {
			return
		}
		h.mapErr(w, r, err)
		return
	}
	ledger, _ := h.Commerce.ListStoreCreditLedger(r.Context(), StoreID(r), customerID, 50)
	writeJSON(w, http.StatusOK, map[string]any{"credit": bal, "ledger": ledger})
}

func (h *CommerceHandlers) GrantStoreCredit(w http.ResponseWriter, r *http.Request) {
	if h.Ent != nil {
		if err := h.Ent.RequireFeature(r.Context(), StoreID(r), entitlements.FeatureStoreCredit); err != nil {
			writeEntitlementErr(w, r, err)
			return
		}
	}
	var in commerce.GrantStoreCreditInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	if in.CustomerID == uuid.Nil {
		if customerID, err := uuid.Parse(r.PathValue("customerID")); err == nil {
			in.CustomerID = customerID
		}
	}
	bal, err := h.Commerce.GrantStoreCredit(r.Context(), StoreID(r), UserID(r), in)
	if err != nil {
		if h.mapFinanceErr(w, r, err) {
			return
		}
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"credit": bal})
}

func (h *CommerceHandlers) ApplyStoreCredit(w http.ResponseWriter, r *http.Request) {
	var in commerce.ApplyStoreCreditInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	fin, err := h.Commerce.ApplyStoreCreditToOrder(r.Context(), StoreID(r), UserID(r), in)
	if err != nil {
		if h.mapFinanceErr(w, r, err) {
			return
		}
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"finance": fin})
}

func (h *CommerceHandlers) PatchStoreCreditSettings(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	if err := h.Commerce.SetStoreCreditEnabled(r.Context(), StoreID(r), in.Enabled); err != nil {
		if h.mapFinanceErr(w, r, err) {
			return
		}
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"store_credit_enabled": in.Enabled,
		"globally_enabled":     h.Commerce.StoreCreditGloballyEnabled(),
	})
}
