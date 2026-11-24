package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/rubicahq/pikko/services/api/internal/whatsapp"
)

type WhatsAppHandlers struct {
	WA *whatsapp.Service
}

func (h *WhatsAppHandlers) ListTemplates(w http.ResponseWriter, r *http.Request) {
	audience := r.URL.Query().Get("audience")
	templates := h.WA.ListTemplates(audience)
	writeJSON(w, http.StatusOK, map[string]any{
		"templates": templates,
		"total":     len(templates),
	})
}

func (h *WhatsAppHandlers) GetTemplate(w http.ResponseWriter, r *http.Request) {
	tmpl, err := h.WA.GetTemplate(r.PathValue("code"))
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"template": tmpl})
}

func (h *WhatsAppHandlers) Preview(w http.ResponseWriter, r *http.Request) {
	var in struct {
		OrderID    *uuid.UUID         `json:"order_id"`
		CustomerID *uuid.UUID         `json:"customer_id"`
		VendorID   *uuid.UUID         `json:"vendor_id"`
		Overrides  map[string]string  `json:"overrides"`
		Variables  map[string]string  `json:"variables"` // alias
	}
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
			return
		}
	}
	overrides := in.Overrides
	if overrides == nil {
		overrides = in.Variables
	}
	out, err := h.WA.Resolve(r.Context(), StoreID(r), r.PathValue("code"), whatsapp.ContextInput{
		OrderID: in.OrderID, CustomerID: in.CustomerID, VendorID: in.VendorID, Overrides: overrides,
	})
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *WhatsAppHandlers) Open(w http.ResponseWriter, r *http.Request) {
	var in struct {
		OrderID    *uuid.UUID        `json:"order_id"`
		CustomerID *uuid.UUID        `json:"customer_id"`
		VendorID   *uuid.UUID        `json:"vendor_id"`
		Overrides  map[string]string `json:"overrides"`
		Action     string            `json:"action"`
		Phone      string            `json:"phone"`
		Body       string            `json:"body"`
	}
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
			return
		}
	}
	resolved, err := h.WA.Resolve(r.Context(), StoreID(r), r.PathValue("code"), whatsapp.ContextInput{
		OrderID: in.OrderID, CustomerID: in.CustomerID, VendorID: in.VendorID, Overrides: in.Overrides,
	})
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	body := resolved.Body
	if in.Body != "" {
		body = in.Body
	}
	phone := resolved.Phone
	if in.Phone != "" {
		phone = in.Phone
	}
	link := whatsapp.DeepLink(phone, body)
	action := in.Action
	if action == "" {
		action = "open"
	}
	log, err := h.WA.LogSend(r.Context(), StoreID(r), whatsapp.LogInput{
		TemplateCode: resolved.Template.Code,
		Action:       action,
		Phone:        phone,
		Body:         body,
		DeepLink:     link,
		OrderID:      resolved.OrderID,
		CustomerID:   resolved.CustomerID,
		VendorID:     resolved.VendorID,
		OpenedBy:     UserID(r),
	})
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"deep_link":  link,
		"body":       body,
		"phone":      phone,
		"phone_e164": whatsapp.NormalizePhoneDigits(phone),
		"template":   resolved.Template,
		"missing":    resolved.Missing,
		"send":       log,
	})
}

func (h *WhatsAppHandlers) ListSends(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	out, err := h.WA.ListSends(r.Context(), StoreID(r), limit)
	if err != nil {
		h.mapErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sends": out, "total": len(out)})
}

func (h *WhatsAppHandlers) mapErr(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, whatsapp.ErrNotFound):
		writeError(w, r, http.StatusNotFound, "not_found", "WhatsApp template or related resource not found.")
	case errors.Is(err, whatsapp.ErrInvalid):
		writeError(w, r, http.StatusBadRequest, "invalid_input", err.Error())
	default:
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Something went wrong.")
	}
}
