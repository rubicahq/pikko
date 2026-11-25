package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultMaxTokens is capped for OpenAI-compatible gateways (SumoPod/Gemini).
// 65536 was rejected or truncated by some models and wasted retries.
const DefaultMaxTokens = 16384
const maxEmptyRetries = 3

// OpenAICompatClient talks to an OpenAI-compatible chat completions API (e.g. SumoPod).
// API keys must only come from env — never log or persist the key.
type OpenAICompatClient struct {
	BaseURL    string
	APIKey     string
	Model      string
	HTTPClient *http.Client
	MaxTokens  int
}

func NewOpenAICompat(baseURL, apiKey, model string) *OpenAICompatClient {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	return &OpenAICompatClient{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   model,
		HTTPClient: &http.Client{
			Timeout: 180 * time.Second,
		},
		MaxTokens: DefaultMaxTokens,
	}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *OpenAICompatClient) ExtractCatalog(ctx context.Context, req ExtractRequest) (*ExtractResult, error) {
	if c.APIKey == "" || c.BaseURL == "" || c.Model == "" {
		return nil, fmt.Errorf("AI client not configured")
	}
	maxTokens := c.MaxTokens
	if maxTokens <= 0 {
		maxTokens = DefaultMaxTokens
	}

	var all []ExtractedItem
	var lastRaw json.RawMessage

	for _, page := range req.Pages {
		if req.MaxItems > 0 && len(all) >= req.MaxItems {
			break
		}
		tiles, err := prepareVisionTiles(page)
		if err != nil {
			return nil, fmt.Errorf("prepare page %d for vision: %w", page.PageNumber, err)
		}
		for ti, tile := range tiles {
			tileMax := 80
			if req.MaxItems > 0 {
				tileMax = req.MaxItems - len(all)
				if tileMax <= 0 {
					break
				}
			}
			items, raw, err := c.extractOnce(ctx, tile.PageImage, tileMax, maxTokens)
			if err != nil {
				return nil, fmt.Errorf("AI extraction failed on page %d tile %d: %w", page.PageNumber, ti+1, err)
			}
			lastRaw = raw
			for i := range items {
				if items[i].PageNumber == nil {
					p := page.PageNumber
					items[i].PageNumber = &p
				}
				mapTileBBoxToPage(&items[i], tile)
				items[i].NormalizeConfidence()
				clampBBox(&items[i])
			}
			all = mergeExtractedItems(all, items)
			if req.MaxItems > 0 && len(all) > req.MaxItems {
				all = all[:req.MaxItems]
				break
			}
		}
	}

	return &ExtractResult{Items: all, Raw: lastRaw}, nil
}

func (c *OpenAICompatClient) extractOnce(ctx context.Context, page PageImage, maxItems, maxTokens int) ([]ExtractedItem, json.RawMessage, error) {
	parts := []any{
		map[string]any{"type": "text", "text": fmt.Sprintf(
			"Extract EVERY distinct sellable catalog item visible in this page image (up to %d). Dense jastip/poster grids often have 20–30 dashed cells — return one item per cell. Return JSON only.",
			maxItems,
		)},
		map[string]any{
			"type": "text",
			"text": fmt.Sprintf("Page %d:", page.PageNumber),
		},
		map[string]any{
			"type": "image_url",
			"image_url": map[string]string{
				"url": fmt.Sprintf("data:%s;base64,%s", mimeOrJPEG(page.MIMEType), page.Base64Data),
			},
		},
	}

	payload := chatRequest{
		Model: c.Model,
		Messages: []chatMessage{
			{Role: "system", Content: SystemPrompt},
			{Role: "user", Content: parts},
		},
		MaxTokens:   maxTokens,
		Temperature: 0.1,
	}

	var lastErr error
	var contentParts []string
	for attempt := 0; attempt < maxEmptyRetries; attempt++ {
		content, finish, err := c.chat(ctx, payload)
		if err != nil {
			lastErr = err
			continue
		}
		if strings.TrimSpace(content) == "" {
			lastErr = fmt.Errorf("empty AI response")
			continue
		}
		contentParts = append(contentParts, content)
		joined := strings.Join(contentParts, "")
		items, raw, err := parseExtraction(joined)
		if err != nil {
			// Truncated mid-JSON: ask the model to continue from the cut.
			if finish == "length" || finish == "max_tokens" {
				lastErr = err
				payload.Messages = append(payload.Messages,
					chatMessage{Role: "assistant", Content: content},
					chatMessage{Role: "user", Content: "Continue the JSON exactly where you left off. Output only the continuation, no markdown."},
				)
				continue
			}
			lastErr = err
			contentParts = contentParts[:0] // reset — non-truncation parse error
			continue
		}
		if maxItems > 0 && len(items) > maxItems {
			items = items[:maxItems]
		}
		for i := range items {
			items[i].NormalizeConfidence()
			clampBBox(&items[i])
		}
		return items, raw, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("AI extraction failed")
	}
	return nil, nil, lastErr
}

func mimeOrJPEG(mime string) string {
	if mime == "" {
		return "image/jpeg"
	}
	return mime
}

func (c *OpenAICompatClient) chat(ctx context.Context, payload chatRequest) (content string, finishReason string, err error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", "", err
	}
	endpoint := c.BaseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Accept", "application/json")

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 16<<20))
	if res.StatusCode >= 300 {
		// Never include Authorization or API key material in errors.
		return "", "", fmt.Errorf("AI provider status %d: %s", res.StatusCode, truncate(string(raw), 400))
	}
	var out chatResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", "", fmt.Errorf("invalid AI response JSON")
	}
	if out.Error != nil && out.Error.Message != "" {
		return "", "", fmt.Errorf("AI provider error: %s", truncate(out.Error.Message, 200))
	}
	if len(out.Choices) == 0 {
		return "", "", fmt.Errorf("empty AI response")
	}
	return out.Choices[0].Message.Content, out.Choices[0].FinishReason, nil
}

func parseExtraction(content string) ([]ExtractedItem, json.RawMessage, error) {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimPrefix(content, "```JSON")
		content = strings.TrimPrefix(content, "```")
		if i := strings.LastIndex(content, "```"); i >= 0 {
			content = content[:i]
		}
		content = strings.TrimSpace(content)
	}
	var envelope struct {
		Items []ExtractedItem `json:"items"`
	}
	if err := json.Unmarshal([]byte(content), &envelope); err == nil {
		raw := json.RawMessage(content)
		return envelope.Items, raw, nil
	}
	// Some models return a bare array.
	var arr []ExtractedItem
	if err := json.Unmarshal([]byte(content), &arr); err == nil {
		wrapped, _ := json.Marshal(map[string]any{"items": arr})
		return arr, wrapped, nil
	}
	return nil, nil, fmt.Errorf("AI JSON parse: expected {\"items\":[...]} or array")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func clampBBox(it *ExtractedItem) {
	clamp := func(p **float64) {
		if *p == nil {
			return
		}
		v := **p
		if v < 0 {
			v = 0
		}
		if v > 100 {
			v = 100
		}
		**p = v
	}
	clamp(&it.BBoxXPct)
	clamp(&it.BBoxYPct)
	clamp(&it.BBoxWidthPct)
	clamp(&it.BBoxHeightPct)
	if it.BBoxXPct != nil && it.BBoxWidthPct != nil && *it.BBoxXPct+*it.BBoxWidthPct > 100 {
		w := 100 - *it.BBoxXPct
		if w <= 0 {
			it.BBoxWidthPct = nil
			it.BBoxXPct = nil
			it.BBoxYPct = nil
			it.BBoxHeightPct = nil
		} else {
			it.BBoxWidthPct = &w
		}
	}
	if it.BBoxYPct != nil && it.BBoxHeightPct != nil && *it.BBoxYPct+*it.BBoxHeightPct > 100 {
		h := 100 - *it.BBoxYPct
		if h <= 0 {
			it.BBoxWidthPct = nil
			it.BBoxXPct = nil
			it.BBoxYPct = nil
			it.BBoxHeightPct = nil
		} else {
			it.BBoxHeightPct = &h
		}
	}
}

var _ CatalogExtractor = (*OpenAICompatClient)(nil)

// NewExtractor selects mock when no API key is configured (safe for CI).
func NewExtractor(apiKey, baseURL, model string) CatalogExtractor {
	if strings.TrimSpace(apiKey) == "" {
		return NewMockExtractor()
	}
	baseURL = strings.TrimSpace(baseURL)
	model = strings.TrimSpace(model)
	if baseURL == "" || model == "" {
		return NewMockExtractor()
	}
	return NewOpenAICompat(baseURL, apiKey, model)
}

// ProviderName reports which extractor implementation is active (for job metadata).
func ProviderName(ex CatalogExtractor) string {
	switch ex.(type) {
	case *MockExtractor:
		return "mock"
	case *OpenAICompatClient:
		return "openai_compat"
	default:
		return "unknown"
	}
}
