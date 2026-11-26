package connie

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDOKUMCPListsSchemaBeforeInquiry(t *testing.T) {
	var methods []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Basic c2VjcmV0Og==" {
			t.Errorf("authorization = %q", got)
		}
		var req struct {
			ID     int             `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		methods = append(methods, req.Method)
		w.Header().Set("Content-Type", "application/json")
		if req.Method == "initialize" {
			w.Header().Set("Mcp-Session-Id", "session-1")
		}
		var result any
		switch req.Method {
		case "initialize":
			result = map[string]any{"protocolVersion": "2025-03-26"}
		case "tools/list":
			result = map[string]any{"tools": []any{map[string]any{"name": "get_transaction_by_invoice_number", "inputSchema": map[string]any{"type": "object", "required": []string{"invoiceNumber"}, "properties": map[string]any{"invoiceNumber": map[string]any{"type": "string"}}}}}}
		case "tools/call":
			var p struct {
				Arguments map[string]string `json:"arguments"`
			}
			_ = json.Unmarshal(req.Params, &p)
			if p.Arguments["invoiceNumber"] != "INV-1" {
				t.Errorf("arguments = %#v", p.Arguments)
			}
			result = map[string]any{"structuredContent": map[string]any{"invoice_number": "INV-1", "status": "SUCCESS", "amount": 125000}}
		}
		if req.Method == "notifications/initialized" {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
	}))
	defer srv.Close()
	c := &DOKUMCPClient{Endpoint: srv.URL, ClientID: "client", APIKey: "secret", Timeout: time.Second}
	got, err := c.GetTransactionByInvoiceNumber(context.Background(), "INV-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "SUCCESS" || got.AmountIDR != 125000 || got.InvoiceNumber != "INV-1" {
		t.Fatalf("got %#v", got)
	}
	if len(methods) != 4 || methods[1] != "notifications/initialized" || methods[2] != "tools/list" || methods[3] != "tools/call" {
		t.Fatalf("methods %#v", methods)
	}
}

func TestDOKUMCPTimeoutIsProviderUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { time.Sleep(50 * time.Millisecond) }))
	defer srv.Close()
	c := &DOKUMCPClient{Endpoint: srv.URL, ClientID: "client", APIKey: "secret", Timeout: 5 * time.Millisecond}
	if _, err := c.GetTransactionByInvoiceNumber(context.Background(), "INV-1"); err == nil {
		t.Fatal("expected timeout")
	}
}
