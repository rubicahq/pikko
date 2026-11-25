package connie

import (
	"encoding/json"
	"testing"

	"github.com/rubicahq/pikko/services/api/internal/rbac"
)

func TestTop30ToolsRegistered(t *testing.T) {
	s := &Service{}
	tools := s.buildTools()
	if len(tools) != 30 {
		t.Fatalf("want 30 primary tools, got %d", len(tools))
	}
	seen := map[string]struct{}{}
	for _, tool := range tools {
		if tool.Name == "" {
			t.Fatal("empty tool name")
		}
		if tool.Description == "" {
			t.Fatalf("%s missing description", tool.Name)
		}
		if len(tool.InputSchema) == 0 {
			t.Fatalf("%s missing inputSchema", tool.Name)
		}
		var schema map[string]any
		if err := json.Unmarshal(tool.InputSchema, &schema); err != nil {
			t.Fatalf("%s invalid schema: %v", tool.Name, err)
		}
		if schema["type"] != "object" {
			t.Fatalf("%s schema type want object", tool.Name)
		}
		if tool.Handler == nil {
			t.Fatalf("%s missing handler", tool.Name)
		}
		if tool.Permission == "" {
			t.Fatalf("%s missing permission", tool.Name)
		}
		if _, ok := seen[tool.Name]; ok {
			t.Fatalf("duplicate tool %s", tool.Name)
		}
		seen[tool.Name] = struct{}{}
	}
	required := []string{
		"get_store", "list_orders", "get_order", "list_products", "list_catalogs",
		"list_customers", "create_customer", "list_campaigns", "get_subscription",
		"get_storefront_status", "get_packing_queue", "get_finance_workspace",
	}
	for _, name := range required {
		if _, ok := seen[name]; !ok {
			t.Fatalf("missing required tool %s", name)
		}
	}
}

func TestListToolsFiltersByRole(t *testing.T) {
	s := &Service{}
	owner := s.ListTools(rbac.RoleOwner)
	if len(owner) != 30 {
		t.Fatalf("owner should see all 30, got %d", len(owner))
	}
	packing := s.ListTools(rbac.RolePacking)
	names := map[string]bool{}
	for _, ttool := range packing {
		names[ttool.Name] = true
	}
	if names["list_orders"] != true {
		t.Fatal("packing should list orders")
	}
	if names["create_customer"] {
		t.Fatal("packing must not create customers")
	}
	if names["get_subscription"] {
		t.Fatal("packing must not read subscription settings")
	}
	if !names["get_packing_queue"] {
		t.Fatal("packing should see packing queue")
	}
}

func TestMatchHeuristicTool(t *testing.T) {
	name, _ := matchHeuristicTool("list pending orders")
	if name != "list_orders" {
		t.Fatalf("got %s", name)
	}
	name, _ = matchHeuristicTool("berapa usage subscription?")
	if name != "get_subscription_usage" {
		t.Fatalf("got %s", name)
	}
	name, _ = matchHeuristicTool("xyz nonsense")
	if name != "" {
		t.Fatalf("expected empty, got %s", name)
	}
}
