package connie

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"github.com/rubicahq/pikko/services/api/internal/analytics"
	"github.com/rubicahq/pikko/services/api/internal/campaigns"
	"github.com/rubicahq/pikko/services/api/internal/catalog"
	"github.com/rubicahq/pikko/services/api/internal/catalogs"
	"github.com/rubicahq/pikko/services/api/internal/commerce"
	"github.com/rubicahq/pikko/services/api/internal/config"
	"github.com/rubicahq/pikko/services/api/internal/entitlements"
	"github.com/rubicahq/pikko/services/api/internal/notifications"
	"github.com/rubicahq/pikko/services/api/internal/rbac"
	"github.com/rubicahq/pikko/services/api/internal/storefront"
	"github.com/rubicahq/pikko/services/api/internal/stores"
	"github.com/rubicahq/pikko/services/api/internal/team"
)

// Service hosts the Connie MCP tool registry and chat loop.
type Service struct {
	Cfg           *config.Config
	Logger        *slog.Logger
	Stores        *stores.Service
	Catalog       *catalog.Service
	Catalogs      *catalogs.Service
	Campaigns     *campaigns.Service
	Commerce      *commerce.Service
	Storefront    *storefront.Service
	Analytics     *analytics.Service
	Notifications *notifications.Service
	Team          *team.Service
	Ent           *entitlements.Service

	tools []ToolMeta
}

func (s *Service) ensureTools() {
	if len(s.tools) > 0 {
		return
	}
	s.tools = s.buildTools()
}

// ListTools returns MCP tools the membership role may call.
func (s *Service) ListTools(role string) []Tool {
	s.ensureTools()
	out := make([]Tool, 0, len(s.tools))
	for _, t := range s.tools {
		if rbac.HasPermission(role, t.Permission) {
			out = append(out, t.Tool)
		}
	}
	return out
}

// ToolCount returns total registered primary tools (unfiltered).
func (s *Service) ToolCount() int {
	s.ensureTools()
	return len(s.tools)
}

// Call executes one tool with RBAC + tenant isolation.
func (s *Service) Call(ctx context.Context, cc CallContext, req CallRequest) CallResult {
	s.ensureTools()
	name := strings.TrimSpace(req.Name)
	var meta *ToolMeta
	for i := range s.tools {
		if s.tools[i].Name == name {
			meta = &s.tools[i]
			break
		}
	}
	if meta == nil {
		return errResult(name, "unknown_tool", ErrUnknownTool.Error())
	}
	if !rbac.HasPermission(cc.Role, meta.Permission) {
		return errResult(name, "forbidden", ErrForbiddenTool.Error())
	}
	args := req.Arguments
	if len(args) == 0 || string(args) == "null" {
		args = json.RawMessage(`{}`)
	}
	if !json.Valid(args) {
		return errResult(name, "invalid_arguments", ErrInvalidArgs.Error())
	}
	res, err := meta.Handler(ctx, cc, args)
	if err != nil {
		code, msg := mapToolErr(err)
		return errResult(name, code, msg)
	}
	text, _ := json.MarshalIndent(res, "", "  ")
	return CallResult{
		Name:    name,
		OK:      true,
		Result:  res,
		Content: []ContentBlock{{Type: "text", Text: string(text)}},
		IsError: false,
	}
}

func errResult(name, code, msg string) CallResult {
	return CallResult{
		Name:    name,
		OK:      false,
		Error:   msg,
		Code:    code,
		Content: []ContentBlock{{Type: "text", Text: msg}},
		IsError: true,
	}
}

func mapToolErr(err error) (code, msg string) {
	switch {
	case errors.Is(err, ErrInvalidArgs):
		return "invalid_arguments", err.Error()
	case errors.Is(err, ErrNotFound):
		return "not_found", "Resource not found in this store."
	case errors.Is(err, commerce.ErrNotFound), errors.Is(err, catalog.ErrNotFound),
		errors.Is(err, catalogs.ErrNotFound), errors.Is(err, campaigns.ErrNotFound),
		errors.Is(err, storefront.ErrNotFound), errors.Is(err, stores.ErrNotFound):
		return "not_found", "Resource not found in this store."
	case errors.Is(err, commerce.ErrInvalidInput), errors.Is(err, catalog.ErrInvalidInput):
		return "invalid_arguments", "Invalid tool arguments."
	case errors.Is(err, commerce.ErrConflict):
		return "conflict", err.Error()
	default:
		return "tool_error", fmt.Sprintf("Tool failed: %v", err)
	}
}

func parseUUIDArg(raw string, field string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: %s", ErrInvalidArgs, field)
	}
	return id, nil
}

func clampLimit(n, def, max int) int {
	if n <= 0 {
		return def
	}
	if n > max {
		return max
	}
	return n
}
