// Package connie implements Ask Connie's store-ops MCP tool surface.
// Tools are MCP-shaped (name, description, inputSchema) and execute against
// existing Pikko services with store-scoped auth + RBAC.
package connie

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"

	"github.com/rubicahq/pikko/services/api/internal/rbac"
)

var (
	ErrUnknownTool   = errors.New("unknown tool")
	ErrForbiddenTool = errors.New("forbidden tool")
	ErrInvalidArgs   = errors.New("invalid tool arguments")
	ErrNotFound      = errors.New("not found")
	ErrAIUnavailable = errors.New("ai unavailable")
)

// Tool is an MCP tools/list entry.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// ToolMeta adds server-side RBAC for filtering and call enforcement.
type ToolMeta struct {
	Tool
	Permission rbac.Permission
	Handler    ToolHandler
}

// CallContext is injected by the HTTP layer after membership + session checks.
// StoreID always comes from the authenticated path — never trust client args for tenancy.
type CallContext struct {
	StoreID uuid.UUID
	UserID  uuid.UUID
	Role    string
}

// ToolHandler executes one tool. Arguments are already JSON-decoded object bytes.
type ToolHandler func(ctx context.Context, cc CallContext, args json.RawMessage) (any, error)

// CallRequest is MCP tools/call style.
type CallRequest struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// CallResult is returned to Connie / API clients.
type CallResult struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Result  any    `json:"result,omitempty"`
	Error   string `json:"error,omitempty"`
	Code    string `json:"code,omitempty"`
	Content []ContentBlock `json:"content"`
	IsError bool   `json:"isError"`
}

// ContentBlock mirrors MCP text content for tool results.
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ChatMessage is a single turn in Connie chat.
type ChatMessage struct {
	Role       string     `json:"role"` // user | assistant | tool | system
	Content    string     `json:"content,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall is an assistant-requested tool invocation.
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"` // function
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// ChatRequest is POST /connie/chat body.
type ChatRequest struct {
	Messages []ChatMessage `json:"messages"`
}

// ChatResponse is returned after the tool loop.
type ChatResponse struct {
	Message      ChatMessage  `json:"message"`
	ToolResults  []CallResult `json:"tool_results,omitempty"`
	ToolsUsed    []string     `json:"tools_used,omitempty"`
	AIConfigured bool         `json:"ai_configured"`
	Mode         string       `json:"mode"` // ai | heuristic
}
