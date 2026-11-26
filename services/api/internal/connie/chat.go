package connie

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	maxChatToolRounds = 4
	maxChatMessages   = 24
	systemPrompt      = `You are Connie, a helpful store operations teammate for Indonesian merchants on Pikko.
You speak clearly in the merchant's language (Bahasa Indonesia or English — match the user).
Use tools to look up real store data before answering. Never invent order numbers, prices, or customer details.
Do not request credentials, secrets, or data about other stores.
Prefer concise, actionable answers. When you use tools, summarize the result for the merchant.`
)

// AIConfigured reports whether chat completions can reach a provider.
func (s *Service) AIConfigured() bool {
	if s.Cfg == nil {
		return false
	}
	return strings.TrimSpace(s.Cfg.AIAPIKey) != "" &&
		strings.TrimSpace(s.Cfg.AIBaseURL) != "" &&
		strings.TrimSpace(s.Cfg.AIModel) != ""
}

// Chat runs an assistant turn with tool calling (AI) or heuristic tool routing.
func (s *Service) Chat(ctx context.Context, cc CallContext, req ChatRequest) (*ChatResponse, error) {
	msgs := trimMessages(req.Messages)
	if len(msgs) == 0 {
		return nil, ErrInvalidArgs
	}
	if s.AIConfigured() {
		return s.chatWithAI(ctx, cc, msgs)
	}
	return s.chatHeuristic(ctx, cc, msgs)
}

func trimMessages(in []ChatMessage) []ChatMessage {
	out := make([]ChatMessage, 0, len(in))
	for _, m := range in {
		role := strings.TrimSpace(m.Role)
		if role != "user" && role != "assistant" && role != "tool" && role != "system" {
			continue
		}
		out = append(out, m)
	}
	if len(out) > maxChatMessages {
		out = out[len(out)-maxChatMessages:]
	}
	return out
}

func (s *Service) chatHeuristic(ctx context.Context, cc CallContext, msgs []ChatMessage) (*ChatResponse, error) {
	last := msgs[len(msgs)-1]
	if last.Role != "user" {
		return nil, ErrInvalidArgs
	}
	q := strings.ToLower(strings.TrimSpace(last.Content))
	name, args := matchHeuristicTool(q)
	if name == "" {
		tools := s.ListTools(cc.Role)
		names := make([]string, 0, len(tools))
		for _, t := range tools {
			names = append(names, t.Name)
		}
		text := "Saya Connie — asisten operasi toko Pikko. Tanya tentang orders, products, catalogs, customers, campaigns, packing, finance, atau subscription.\n\nContoh: \"berapa order pending?\", \"list products\", \"status storefront\", \"subscription usage\".\n\nAI chat belum terhubung di environment ini; saya tetap bisa menjalankan tool secara langsung jika perintahnya jelas."
		if len(names) > 0 {
			text += "\n\nTool tersedia: " + strings.Join(names, ", ") + "."
		}
		return &ChatResponse{
			Message:      ChatMessage{Role: "assistant", Content: text},
			AIConfigured: false,
			Mode:         "heuristic",
		}, nil
	}
	res := s.Call(ctx, cc, CallRequest{Name: name, Arguments: args})
	summary := summarizeToolResult(name, res)
	return &ChatResponse{
		Message:      ChatMessage{Role: "assistant", Content: summary},
		ToolResults:  []CallResult{res},
		ToolsUsed:    []string{name},
		AIConfigured: false,
		Mode:         "heuristic",
	}, nil
}

func matchHeuristicTool(q string) (string, json.RawMessage) {
	empty := json.RawMessage(`{}`)
	switch {
	case (strings.Contains(q, "doku") || strings.Contains(q, "cek pembayaran")) && (strings.Contains(q, "order") || strings.Contains(q, "pesanan")):
		identifier := regexp.MustCompile(`(?i)\b(?:PK-[A-Z0-9-]+|[0-9a-f]{8}-[0-9a-f-]{27,})\b`).FindString(q)
		if identifier == "" {
			return "", nil
		}
		args, _ := json.Marshal(map[string]string{"order_identifier": identifier})
		return "check_doku_payment", args
	case strings.Contains(q, "subscription") && strings.Contains(q, "usage"),
		strings.Contains(q, "usage") && (strings.Contains(q, "plan") || strings.Contains(q, "limit")):
		return "get_subscription_usage", empty
	case strings.Contains(q, "subscription"), strings.Contains(q, "billing"), strings.Contains(q, "plan"):
		return "get_subscription", empty
	case strings.Contains(q, "storefront"), strings.Contains(q, "publish"):
		return "get_storefront_status", empty
	case strings.Contains(q, "checklist"), strings.Contains(q, "setup"):
		return "get_setup_checklist", empty
	case strings.Contains(q, "packing"):
		return "get_packing_queue", empty
	case strings.Contains(q, "shipment"), strings.Contains(q, "pengiriman"):
		return "list_shipments", empty
	case strings.Contains(q, "purchasing"), strings.Contains(q, "pembelian"):
		return "get_purchasing_summary", empty
	case strings.Contains(q, "vendor"):
		return "list_vendors", empty
	case strings.Contains(q, "finance"), strings.Contains(q, "reconciliation"), strings.Contains(q, "settlement"):
		return "get_finance_workspace", empty
	case strings.Contains(q, "analytics"), strings.Contains(q, "penjualan"), strings.Contains(q, "sales"):
		return "get_analytics_overview", empty
	case strings.Contains(q, "notification"):
		return "list_notifications", empty
	case strings.Contains(q, "payment method"), strings.Contains(q, "metode bayar"):
		return "list_payment_methods", empty
	case strings.Contains(q, "fulfillment"), strings.Contains(q, "ongkir"):
		return "get_fulfillment_settings", empty
	case strings.Contains(q, "team"), strings.Contains(q, "member"):
		return "list_team_members", empty
	case strings.Contains(q, "customer"), strings.Contains(q, "pelanggan"):
		return "list_customers", empty
	case strings.Contains(q, "catalog"), strings.Contains(q, "katalog"):
		return "list_catalogs", empty
	case strings.Contains(q, "campaign"), strings.Contains(q, "kampanye"):
		return "list_campaigns", empty
	case strings.Contains(q, "preorder"):
		return "list_preorders", empty
	case strings.Contains(q, "product"), strings.Contains(q, "produk"), strings.Contains(q, "barang"):
		return "list_products", empty
	case strings.Contains(q, "order"), strings.Contains(q, "pesanan"):
		args := empty
		if strings.Contains(q, "pending") || strings.Contains(q, "unpaid") || strings.Contains(q, "belum bayar") {
			args = json.RawMessage(`{"payment_status":"unpaid","limit":25}`)
		}
		return "list_orders", args
	case strings.Contains(q, "store"), strings.Contains(q, "toko"):
		return "get_store", empty
	default:
		return "", nil
	}
}

func summarizeToolResult(name string, res CallResult) string {
	if !res.OK {
		return fmt.Sprintf("Gagal menjalankan %s: %s", name, res.Error)
	}
	b, _ := json.MarshalIndent(res.Result, "", "  ")
	text := string(b)
	if len(text) > 3500 {
		text = text[:3500] + "…"
	}
	return fmt.Sprintf("Hasil %s:\n```json\n%s\n```", name, text)
}

func (s *Service) chatWithAI(ctx context.Context, cc CallContext, msgs []ChatMessage) (*ChatResponse, error) {
	tools := s.ListTools(cc.Role)
	oaTools := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		var params any
		_ = json.Unmarshal(t.InputSchema, &params)
		oaTools = append(oaTools, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  params,
			},
		})
	}

	history := make([]ChatMessage, 0, len(msgs)+8)
	history = append(history, ChatMessage{Role: "system", Content: systemPrompt})
	history = append(history, msgs...)

	var used []string
	var results []CallResult

	for round := 0; round < maxChatToolRounds; round++ {
		asst, err := s.completeChat(ctx, history, oaTools)
		if err != nil {
			return nil, err
		}
		if len(asst.ToolCalls) == 0 {
			return &ChatResponse{
				Message:      ChatMessage{Role: "assistant", Content: strings.TrimSpace(asst.Content)},
				ToolResults:  results,
				ToolsUsed:    uniqueStrings(used),
				AIConfigured: true,
				Mode:         "ai",
			}, nil
		}
		history = append(history, *asst)
		for _, tc := range asst.ToolCalls {
			args := json.RawMessage(tc.Function.Arguments)
			if !json.Valid(args) {
				args = json.RawMessage(`{}`)
			}
			res := s.Call(ctx, cc, CallRequest{Name: tc.Function.Name, Arguments: args})
			used = append(used, tc.Function.Name)
			results = append(results, res)
			payload, _ := json.Marshal(res.Result)
			if !res.OK {
				payload, _ = json.Marshal(map[string]any{"error": res.Error, "code": res.Code})
			}
			history = append(history, ChatMessage{
				Role:       "tool",
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
				Content:    string(payload),
			})
		}
	}
	return &ChatResponse{
		Message: ChatMessage{
			Role:    "assistant",
			Content: "Saya sudah menjalankan beberapa tool tetapi masih butuh klarifikasi. Coba tanya lebih spesifik (mis. nomor order atau nama produk).",
		},
		ToolResults:  results,
		ToolsUsed:    uniqueStrings(used),
		AIConfigured: true,
		Mode:         "ai",
	}, nil
}

func uniqueStrings(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func (s *Service) completeChat(ctx context.Context, msgs []ChatMessage, tools []map[string]any) (*ChatMessage, error) {
	type oaMsg struct {
		Role       string     `json:"role"`
		Content    any        `json:"content,omitempty"`
		ToolCallID string     `json:"tool_call_id,omitempty"`
		Name       string     `json:"name,omitempty"`
		ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	}
	oaMsgs := make([]oaMsg, 0, len(msgs))
	for _, m := range msgs {
		om := oaMsg{Role: m.Role, ToolCallID: m.ToolCallID, Name: m.Name}
		if len(m.ToolCalls) > 0 {
			om.ToolCalls = m.ToolCalls
		}
		if m.Content != "" || m.Role == "tool" {
			om.Content = m.Content
		}
		oaMsgs = append(oaMsgs, om)
	}
	body := map[string]any{
		"model":       s.Cfg.AIModel,
		"messages":    oaMsgs,
		"temperature": 0.2,
		"max_tokens":  2048,
	}
	if len(tools) > 0 {
		body["tools"] = tools
		body["tool_choice"] = "auto"
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	endpoint := strings.TrimRight(s.Cfg.AIBaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.Cfg.AIAPIKey)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 90 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAIUnavailable, err)
	}
	defer res.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: provider status %d", ErrAIUnavailable, res.StatusCode)
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Role      string     `json:"role"`
				Content   string     `json:"content"`
				ToolCalls []ToolCall `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("%w: invalid AI JSON", ErrAIUnavailable)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return nil, fmt.Errorf("%w: %s", ErrAIUnavailable, parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("%w: empty choices", ErrAIUnavailable)
	}
	msg := parsed.Choices[0].Message
	for i := range msg.ToolCalls {
		if msg.ToolCalls[i].ID == "" {
			msg.ToolCalls[i].ID = "call_" + uuid.NewString()
		}
		if msg.ToolCalls[i].Type == "" {
			msg.ToolCalls[i].Type = "function"
		}
	}
	return &ChatMessage{
		Role:      "assistant",
		Content:   msg.Content,
		ToolCalls: msg.ToolCalls,
	}, nil
}
