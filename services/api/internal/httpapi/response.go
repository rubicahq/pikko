package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

type ErrorBody struct {
	Error struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		RequestID string `json:"request_id"`
	} `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	var body ErrorBody
	body.Error.Code = code
	body.Error.Message = message
	body.Error.RequestID = RequestID(r)
	writeJSON(w, status, body)
}

func RequestID(r *http.Request) string {
	if id := r.Header.Get("X-Request-ID"); id != "" {
		return id
	}
	if id, ok := r.Context().Value(ctxKeyRequestID).(string); ok {
		return id
	}
	return uuid.NewString()
}

type ctxKey string

const (
	ctxKeyRequestID        ctxKey = "request_id"
	ctxKeyUserID           ctxKey = "user_id"
	ctxKeySession          ctxKey = "session_raw"
	ctxKeyStoreID          ctxKey = "store_id"
	ctxKeyMembershipRole   ctxKey = "membership_role"
	ctxKeyPortalCustomerID ctxKey = "portal_customer_id"
	ctxKeyPortalSession    ctxKey = "portal_session_raw"
	ctxKeyPortalStoreID    ctxKey = "portal_store_id"
)
