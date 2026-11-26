package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rubicahq/pikko/services/api/internal/connie"
)

type ConnieHandlers struct {
	Connie *connie.Service
}

func (h *ConnieHandlers) ListTools(w http.ResponseWriter, r *http.Request) {
	tools := h.Connie.ListTools(MembershipRole(r))
	writeJSON(w, http.StatusOK, map[string]any{
		"tools":         tools,
		"count":         len(tools),
		"primary_cap":   31,
		"ai_configured": h.Connie.AIConfigured(),
	})
}

func (h *ConnieHandlers) CallTool(w http.ResponseWriter, r *http.Request) {
	var req connie.CallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	if req.Name == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Tool name is required.")
		return
	}
	cc := connie.CallContext{
		StoreID: StoreID(r),
		UserID:  UserID(r),
		Role:    MembershipRole(r),
	}
	res := h.Connie.Call(r.Context(), cc, req)
	status := http.StatusOK
	if !res.OK {
		switch res.Code {
		case "unknown_tool":
			status = http.StatusNotFound
		case "forbidden":
			status = http.StatusForbidden
		case "invalid_arguments":
			status = http.StatusBadRequest
		case "not_found":
			status = http.StatusNotFound
		case "conflict":
			status = http.StatusConflict
		default:
			status = http.StatusBadRequest
		}
	}
	writeJSON(w, status, res)
}

func (h *ConnieHandlers) Chat(w http.ResponseWriter, r *http.Request) {
	var req connie.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid JSON.")
		return
	}
	cc := connie.CallContext{
		StoreID: StoreID(r),
		UserID:  UserID(r),
		Role:    MembershipRole(r),
	}
	res, err := h.Connie.Chat(r.Context(), cc, req)
	if err != nil {
		if errors.Is(err, connie.ErrInvalidArgs) {
			writeError(w, r, http.StatusBadRequest, "invalid_input", "Chat messages are required.")
			return
		}
		if errors.Is(err, connie.ErrAIUnavailable) {
			writeError(w, r, http.StatusBadGateway, "ai_unavailable", "Connie AI provider is temporarily unavailable.")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Connie chat failed.")
		return
	}
	writeJSON(w, http.StatusOK, res)
}
