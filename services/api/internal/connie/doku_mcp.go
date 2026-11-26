package connie

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var ErrProviderUnavailable = errors.New("provider unavailable")

type DOKUInquiry struct {
	InvoiceNumber string
	Status        string
	AmountIDR     int64
}

type DOKUInquirer interface {
	GetTransactionByInvoiceNumber(context.Context, string) (*DOKUInquiry, error)
}

type DOKUMCPClient struct {
	Endpoint, ClientID, APIKey string
	Timeout                    time.Duration
	HTTPClient                 *http.Client
}

type rpcResponse struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (c *DOKUMCPClient) call(ctx context.Context, sessionID, method string, params any, id int) (rpcResponse, string, error) {
	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(body))
	if err != nil {
		return rpcResponse{}, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Client-Id", c.ClientID)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(c.APIKey+":")))
	if sessionID != "" {
		req.Header.Set("Mcp-Session-Id", sessionID)
	}
	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: c.Timeout}
	}
	res, err := client.Do(req)
	if err != nil {
		return rpcResponse{}, "", fmt.Errorf("%w: request failed", ErrProviderUnavailable)
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return rpcResponse{}, "", fmt.Errorf("%w: status %d", ErrProviderUnavailable, res.StatusCode)
	}
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if strings.Contains(res.Header.Get("Content-Type"), "text/event-stream") {
		raw = lastSSEData(raw)
	}
	var out rpcResponse
	if json.Unmarshal(raw, &out) != nil {
		return out, "", fmt.Errorf("%w: invalid response", ErrProviderUnavailable)
	}
	if out.Error != nil {
		return out, "", fmt.Errorf("%w: rpc error %d", ErrProviderUnavailable, out.Error.Code)
	}
	return out, res.Header.Get("Mcp-Session-Id"), nil
}

func (c *DOKUMCPClient) initialized(ctx context.Context, sessionID string) error {
	body := []byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Client-Id", c.ClientID)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(c.APIKey+":")))
	req.Header.Set("Mcp-Session-Id", sessionID)
	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: c.Timeout}
	}
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: initialization failed", ErrProviderUnavailable)
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("%w: initialization status %d", ErrProviderUnavailable, res.StatusCode)
	}
	return nil
}

func lastSSEData(raw []byte) []byte {
	var data []byte
	sc := bufio.NewScanner(bytes.NewReader(raw))
	for sc.Scan() {
		if strings.HasPrefix(sc.Text(), "data:") {
			data = []byte(strings.TrimSpace(strings.TrimPrefix(sc.Text(), "data:")))
		}
	}
	return data
}

func (c *DOKUMCPClient) GetTransactionByInvoiceNumber(ctx context.Context, invoice string) (*DOKUInquiry, error) {
	init, session, err := c.call(ctx, "", "initialize", map[string]any{"protocolVersion": "2025-03-26", "capabilities": map[string]any{}, "clientInfo": map[string]any{"name": "pikko-ask-connie", "version": "1.0"}}, 1)
	_ = init
	if err != nil {
		return nil, err
	}
	if err := c.initialized(ctx, session); err != nil {
		return nil, err
	}
	listed, _, err := c.call(ctx, session, "tools/list", map[string]any{}, 2)
	if err != nil {
		return nil, err
	}
	var tools struct {
		Tools []struct {
			Name        string `json:"name"`
			InputSchema struct {
				Required   []string                   `json:"required"`
				Properties map[string]json.RawMessage `json:"properties"`
			} `json:"inputSchema"`
		} `json:"tools"`
	}
	if json.Unmarshal(listed.Result, &tools) != nil {
		return nil, fmt.Errorf("%w: invalid tools list", ErrProviderUnavailable)
	}
	argName := ""
	for _, tool := range tools.Tools {
		if tool.Name == "get_transaction_by_invoice_number" {
			for _, n := range tool.InputSchema.Required {
				if invoiceKey(n) {
					argName = n
					break
				}
			}
			if argName == "" {
				for n := range tool.InputSchema.Properties {
					if invoiceKey(n) {
						argName = n
						break
					}
				}
			}
		}
	}
	if argName == "" {
		return nil, fmt.Errorf("%w: inquiry tool schema unavailable", ErrProviderUnavailable)
	}
	called, _, err := c.call(ctx, session, "tools/call", map[string]any{"name": "get_transaction_by_invoice_number", "arguments": map[string]any{argName: invoice}}, 3)
	if err != nil {
		return nil, err
	}
	var result any
	if json.Unmarshal(called.Result, &result) != nil {
		return nil, fmt.Errorf("%w: invalid tool result", ErrProviderUnavailable)
	}
	flat := flattenResult(result)
	inv := firstString(flat, "invoice_number", "invoiceNumber", "invoice")
	status := firstString(flat, "status", "transaction_status", "transactionStatus")
	amount := firstInt64(flat, "amount", "amount_idr", "amountIDR")
	if inv == "" {
		inv = invoice
	}
	if status == "" {
		return nil, fmt.Errorf("%w: incomplete result", ErrProviderUnavailable)
	}
	return &DOKUInquiry{InvoiceNumber: inv, Status: status, AmountIDR: amount}, nil
}

func invoiceKey(s string) bool {
	n := strings.ToLower(strings.ReplaceAll(s, "_", ""))
	return n == "invoicenumber" || n == "invoice"
}
func flattenResult(v any) map[string]any {
	out := map[string]any{}
	var walk func(any)
	walk = func(x any) {
		switch y := x.(type) {
		case map[string]any:
			for k, z := range y {
				out[k] = z
				walk(z)
			}
		case []any:
			for _, z := range y {
				walk(z)
			}
		case string:
			var z any
			if json.Unmarshal([]byte(y), &z) == nil {
				walk(z)
			}
		}
	}
	walk(v)
	return out
}
func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok {
			return v
		}
	}
	return ""
}
func firstInt64(m map[string]any, keys ...string) int64 {
	for _, k := range keys {
		switch v := m[k].(type) {
		case float64:
			return int64(v)
		case string:
			var n int64
			if _, e := fmt.Sscan(v, &n); e == nil {
				return n
			}
		}
	}
	return 0
}
