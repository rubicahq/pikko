package connie

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/rubicahq/pikko/services/api/internal/catalog"
	"github.com/rubicahq/pikko/services/api/internal/commerce"
	"github.com/rubicahq/pikko/services/api/internal/rbac"
)

// buildTools registers the primary store-ops MCP tools.
func (s *Service) buildTools() []ToolMeta {
	return []ToolMeta{
		s.tool("get_store", "Get the current store profile (name, slug, selling model, publication status).",
			emptyObjectSchema(), rbac.PermStoresRead, s.getStore),
		s.tool("get_setup_checklist", "Get store setup checklist completion flags (products, storefront, payments, etc.).",
			emptyObjectSchema(), rbac.PermStoresRead, s.getSetupChecklist),
		s.tool("get_storefront_status", "Get draft and published storefront status for this store.",
			emptyObjectSchema(), rbac.PermStorefrontRead, s.getStorefrontStatus),
		s.tool("list_products", "List products in the store catalog. Optional query, status, limit, offset.",
			schema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query":  propString("Search by name or SKU"),
					"status": propString("draft | published | archived"),
					"limit":  propInt("Max results (default 25, max 100)"),
					"offset": propInt("Pagination offset"),
				},
			}), rbac.PermCatalogRead, s.listProducts),
		s.tool("get_product", "Get a single product by id.",
			schema(map[string]any{
				"type":                 "object",
				"required":             []string{"product_id"},
				"properties":           map[string]any{"product_id": propString("Product UUID")},
				"additionalProperties": false,
			}), rbac.PermCatalogRead, s.getProduct),
		s.tool("list_catalogs", "List PDF catalogs for the store.",
			emptyObjectSchema(), rbac.PermCatalogRead, s.listCatalogs),
		s.tool("get_catalog", "Get a PDF catalog by id including campaign/preorder links.",
			schema(map[string]any{
				"type":                 "object",
				"required":             []string{"catalog_id"},
				"properties":           map[string]any{"catalog_id": propString("Catalog UUID")},
				"additionalProperties": false,
			}), rbac.PermCatalogRead, s.getCatalog),
		s.tool("list_campaigns", "List campaigns (live / preorder / promo).",
			schema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"type": propString("Optional campaign type filter"),
				},
			}), rbac.PermCampaignRead, s.listCampaigns),
		s.tool("get_campaign", "Get a campaign by id with assigned products.",
			schema(map[string]any{
				"type":                 "object",
				"required":             []string{"campaign_id"},
				"properties":           map[string]any{"campaign_id": propString("Campaign UUID")},
				"additionalProperties": false,
			}), rbac.PermCampaignRead, s.getCampaign),
		s.tool("list_orders", "Search and list orders. Filter by status, payment, fulfillment, or free-text query.",
			schema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query":              propString("Order number, customer name, or email"),
					"order_status":       propString("Order status filter"),
					"payment_status":     propString("Payment status filter"),
					"fulfillment_status": propString("Fulfillment status filter"),
					"limit":              propInt("Max results (default 25)"),
					"offset":             propInt("Pagination offset"),
				},
			}), rbac.PermOrdersRead, s.listOrders),
		s.tool("get_order", "Get a single order with line items.",
			schema(map[string]any{
				"type":                 "object",
				"required":             []string{"order_id"},
				"properties":           map[string]any{"order_id": propString("Order UUID")},
				"additionalProperties": false,
			}), rbac.PermOrdersRead, s.getOrder),
		s.tool("list_order_payments", "List payments recorded against an order.",
			schema(map[string]any{
				"type":                 "object",
				"required":             []string{"order_id"},
				"properties":           map[string]any{"order_id": propString("Order UUID")},
				"additionalProperties": false,
			}), rbac.PermOrdersPayments, s.listOrderPayments),
		s.tool("check_doku_payment", "Read-only payment inquiry for one Pikko order number or UUID via its stored DOKU invoice.",
			schema(map[string]any{
				"type": "object", "required": []string{"order_identifier"},
				"properties":           map[string]any{"order_identifier": propString("Pikko order number or UUID; never a DOKU invoice")},
				"additionalProperties": false,
			}), rbac.PermOrdersPayments, s.checkDOKUPayment),
		s.tool("list_customers", "List customers. Optional name/phone/email query.",
			schema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query":  propString("Search customers"),
					"limit":  propInt("Max results (default 25)"),
					"offset": propInt("Pagination offset"),
				},
			}), rbac.PermCustomersRead, s.listCustomers),
		s.tool("get_customer", "Get a customer profile and recent orders.",
			schema(map[string]any{
				"type":                 "object",
				"required":             []string{"customer_id"},
				"properties":           map[string]any{"customer_id": propString("Customer UUID")},
				"additionalProperties": false,
			}), rbac.PermCustomersRead, s.getCustomer),
		s.tool("create_customer", "Create a customer with name and Indonesian phone (no order required). Returns portal URL.",
			schema(map[string]any{
				"type":     "object",
				"required": []string{"name", "phone"},
				"properties": map[string]any{
					"name":  propString("Customer display name"),
					"phone": propString("WhatsApp / phone (ID format)"),
				},
				"additionalProperties": false,
			}), rbac.PermCustomersWrite, s.createCustomer),
		s.tool("list_preorders", "List preorder batches for the store.",
			emptyObjectSchema(), rbac.PermCampaignRead, s.listPreorders),
		s.tool("get_preorder", "Get a preorder batch by id with products.",
			schema(map[string]any{
				"type":                 "object",
				"required":             []string{"batch_id"},
				"properties":           map[string]any{"batch_id": propString("Preorder batch UUID")},
				"additionalProperties": false,
			}), rbac.PermCampaignRead, s.getPreorder),
		s.tool("get_purchasing_summary", "Get aggregated purchasing needs across open preorders/campaigns.",
			schema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"batch_id":    propString("Optional preorder batch UUID filter"),
					"campaign_id": propString("Optional campaign UUID filter"),
					"group_by":    propString("variant | vendor | brand | campaign"),
				},
			}), rbac.PermPurchasingRead, s.getPurchasingSummary),
		s.tool("get_packing_queue", "Get packing queue of orders ready to pack/ship.",
			schema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"group_by": propString("Optional grouping"),
					"status":   propString("Optional packing status filter"),
				},
			}), rbac.PermPackingRead, s.getPackingQueue),
		s.tool("list_shipments", "List shipments. Optionally filter by order_id.",
			schema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"order_id": propString("Optional order UUID"),
					"limit":    propInt("Max results (default 25)"),
					"offset":   propInt("Pagination offset"),
				},
			}), rbac.PermPackingRead, s.listShipments),
		s.tool("get_finance_workspace", "List order finance / settlement rows for reconciliation.",
			schema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"settlement_status": propString("Optional settlement filter"),
					"limit":             propInt("Max results (default 25)"),
					"offset":            propInt("Pagination offset"),
				},
			}), rbac.PermFinanceRead, s.getFinanceWorkspace),
		s.tool("get_order_finance", "Get finance summary for one order (payments, settlement, refunds).",
			schema(map[string]any{
				"type":                 "object",
				"required":             []string{"order_id"},
				"properties":           map[string]any{"order_id": propString("Order UUID")},
				"additionalProperties": false,
			}), rbac.PermFinanceRead, s.getOrderFinance),
		s.tool("get_subscription", "Get store subscription plan, features, and limits (read-only).",
			emptyObjectSchema(), rbac.PermStoresSettings, s.getSubscription),
		s.tool("get_subscription_usage", "Get current-period usage vs plan limits (read-only).",
			emptyObjectSchema(), rbac.PermStoresRead, s.getSubscriptionUsage),
		s.tool("get_analytics_overview", "Get analytics overview for a date range (defaults to last 30 days Jakarta).",
			schema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"from": propString("Start date YYYY-MM-DD"),
					"to":   propString("End date YYYY-MM-DD"),
				},
			}), rbac.PermAnalyticsRead, s.getAnalyticsOverview),
		s.tool("list_notifications", "List recent in-app notifications.",
			schema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"unread_only": map[string]any{"type": "boolean", "description": "Only unread"},
					"limit":       propInt("Max results (default 20)"),
					"offset":      propInt("Pagination offset"),
				},
			}), rbac.PermNotificationsRead, s.listNotifications),
		s.tool("get_fulfillment_settings", "Get store fulfillment / shipping settings.",
			emptyObjectSchema(), rbac.PermStoresSettings, s.getFulfillmentSettings),
		s.tool("list_payment_methods", "List configured merchant payment methods.",
			emptyObjectSchema(), rbac.PermStoresSettings, s.listPaymentMethods),
		s.tool("list_team_members", "List store team members and roles.",
			emptyObjectSchema(), rbac.PermTeamRead, s.listTeamMembers),
		s.tool("list_vendors", "List purchasing vendors linked to the store.",
			emptyObjectSchema(), rbac.PermPurchasingRead, s.listVendors),
	}
}

type dokuPaymentCheck struct {
	OrderID      string `json:"order_id"`
	OrderNumber  string `json:"order_number"`
	OrderURL     string `json:"order_url"`
	PikkoStatus  string `json:"pikko_status"`
	DOKUStatus   string `json:"doku_status,omitempty"`
	CheckedAt    string `json:"checked_at"`
	Verification string `json:"verification"`
	Explanation  string `json:"explanation"`
	InvoiceMatch bool   `json:"invoice_match"`
	AmountMatch  bool   `json:"amount_match"`
}

func (s *Service) checkDOKUPayment(ctx context.Context, cc CallContext, args json.RawMessage) (any, error) {
	in, err := decodeArgs[struct {
		OrderIdentifier string `json:"order_identifier"`
	}](args)
	if err != nil {
		return nil, err
	}
	order, err := s.Commerce.GetOrderByIdentifier(ctx, cc.StoreID, in.OrderIdentifier)
	if err != nil {
		return nil, err
	}
	out := dokuPaymentCheck{OrderID: order.ID.String(), OrderNumber: order.OrderNumber, OrderURL: "/stores/" + cc.StoreID.String() + "/orders/" + order.ID.String(), PikkoStatus: order.PaymentStatus, CheckedAt: time.Now().UTC().Format(time.RFC3339), Verification: "unverified"}
	attempt, err := s.Commerce.GetLatestPaymentAttempt(ctx, cc.StoreID, order.ID)
	if errors.Is(err, commerce.ErrNotFound) {
		out.Explanation = "Order ini belum memiliki payment attempt atau invoice DOKU."
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(attempt.Provider, "doku") {
		out.Explanation = "Pembayaran order ini menggunakan provider lain, bukan DOKU."
		return out, nil
	}
	if attempt.ProviderInvoiceNumber == nil || strings.TrimSpace(*attempt.ProviderInvoiceNumber) == "" {
		out.Explanation = "Payment attempt DOKU belum memiliki invoice yang dapat diperiksa."
		return out, nil
	}
	credentials, credentialErr := s.Commerce.LoadDOKUCredentials(ctx, cc.StoreID)
	if credentialErr != nil {
		out.Explanation = "Credential DOKU Sandbox belum dikonfigurasi untuk toko ini."
		return out, nil
	}
	if s.Cfg == nil {
		out.Explanation = "Konfigurasi DOKU MCP Sandbox belum tersedia; status DOKU belum terverifikasi."
		return out, nil
	}
	client := s.DOKU
	if client == nil {
		client = &DOKUMCPClient{Endpoint: s.Cfg.DOKUMCPEndpoint, ClientID: credentials.MCPClientID, APIKey: credentials.MCPAPIKey, Timeout: s.Cfg.DOKUMCPTimeout}
	}
	checkCtx, cancel := context.WithTimeout(ctx, s.Cfg.DOKUMCPTimeout)
	defer cancel()
	inq, err := client.GetTransactionByInvoiceNumber(checkCtx, *attempt.ProviderInvoiceNumber)
	if err != nil {
		out.Explanation = "DOKU tidak dapat diverifikasi saat ini; gunakan status terakhir di Pikko."
		return out, nil
	}
	out.DOKUStatus = inq.Status
	out.InvoiceMatch = strings.EqualFold(strings.TrimSpace(inq.InvoiceNumber), strings.TrimSpace(*attempt.ProviderInvoiceNumber))
	out.AmountMatch = inq.AmountIDR == attempt.AmountIDR && attempt.AmountIDR == order.TotalIDR
	paid := strings.EqualFold(inq.Status, "success") || strings.EqualFold(inq.Status, "paid")
	if out.InvoiceMatch && out.AmountMatch && paid {
		out.Verification = "confirmed"
		if order.PaymentStatus == commerce.PaymentPaid {
			out.Explanation = "Pembayaran terkonfirmasi di DOKU dan Pikko."
		} else {
			out.Explanation = "DOKU mengonfirmasi pembayaran, tetapi Pikko masih " + order.PaymentStatus + ". Webhook/rekonsiliasi tetap menangani pembaruan status."
		}
	} else {
		out.Explanation = "Hasil DOKU belum dapat mengonfirmasi pembayaran karena status, invoice, atau nominal tidak cocok."
	}
	return out, nil
}

func (s *Service) tool(name, desc string, schema json.RawMessage, perm rbac.Permission, h ToolHandler) ToolMeta {
	return ToolMeta{
		Tool: Tool{
			Name:        name,
			Description: desc,
			InputSchema: schema,
		},
		Permission: perm,
		Handler:    h,
	}
}

func decodeArgs[T any](raw json.RawMessage) (T, error) {
	var v T
	if len(raw) == 0 || string(raw) == "null" {
		return v, nil
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return v, ErrInvalidArgs
	}
	return v, nil
}

func (s *Service) getStore(ctx context.Context, cc CallContext, _ json.RawMessage) (any, error) {
	st, err := s.Stores.GetAuthorized(ctx, cc.UserID, cc.StoreID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"store": st}, nil
}

func (s *Service) getSetupChecklist(ctx context.Context, cc CallContext, _ json.RawMessage) (any, error) {
	cl, err := s.Catalog.SetupChecklist(ctx, cc.StoreID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"checklist": cl}, nil
}

func (s *Service) getStorefrontStatus(ctx context.Context, cc CallContext, _ json.RawMessage) (any, error) {
	draft, err := s.Storefront.Get(ctx, cc.StoreID, "draft")
	if err != nil {
		return nil, err
	}
	published, err := s.Storefront.Get(ctx, cc.StoreID, "published")
	out := map[string]any{"draft": draft, "has_published": err == nil}
	if err == nil {
		out["published"] = published
	}
	return out, nil
}

func (s *Service) listProducts(ctx context.Context, cc CallContext, args json.RawMessage) (any, error) {
	in, err := decodeArgs[struct {
		Query  string `json:"query"`
		Status string `json:"status"`
		Limit  int    `json:"limit"`
		Offset int    `json:"offset"`
	}](args)
	if err != nil {
		return nil, err
	}
	list, total, err := s.Catalog.ListProducts(ctx, cc.StoreID, catalog.ProductListFilter{
		Query:  in.Query,
		Status: in.Status,
		Limit:  clampLimit(in.Limit, 25, 100),
		Offset: in.Offset,
	}, false)
	if err != nil {
		return nil, err
	}
	return map[string]any{"products": list, "total": total}, nil
}

func (s *Service) getProduct(ctx context.Context, cc CallContext, args json.RawMessage) (any, error) {
	in, err := decodeArgs[struct {
		ProductID string `json:"product_id"`
	}](args)
	if err != nil {
		return nil, err
	}
	id, err := parseUUIDArg(in.ProductID, "product_id")
	if err != nil {
		return nil, err
	}
	p, err := s.Catalog.GetProduct(ctx, cc.StoreID, id, false)
	if err != nil {
		return nil, err
	}
	return map[string]any{"product": p}, nil
}

func (s *Service) listCatalogs(ctx context.Context, cc CallContext, _ json.RawMessage) (any, error) {
	list, err := s.Catalogs.List(ctx, cc.StoreID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"catalogs": list}, nil
}

func (s *Service) getCatalog(ctx context.Context, cc CallContext, args json.RawMessage) (any, error) {
	in, err := decodeArgs[struct {
		CatalogID string `json:"catalog_id"`
	}](args)
	if err != nil {
		return nil, err
	}
	id, err := parseUUIDArg(in.CatalogID, "catalog_id")
	if err != nil {
		return nil, err
	}
	c, err := s.Catalogs.Get(ctx, cc.StoreID, id)
	if err != nil {
		return nil, err
	}
	return map[string]any{"catalog": c}, nil
}

func (s *Service) listCampaigns(ctx context.Context, cc CallContext, args json.RawMessage) (any, error) {
	in, err := decodeArgs[struct {
		Type string `json:"type"`
	}](args)
	if err != nil {
		return nil, err
	}
	list, err := s.Campaigns.List(ctx, cc.StoreID, in.Type)
	if err != nil {
		return nil, err
	}
	return map[string]any{"campaigns": list}, nil
}

func (s *Service) getCampaign(ctx context.Context, cc CallContext, args json.RawMessage) (any, error) {
	in, err := decodeArgs[struct {
		CampaignID string `json:"campaign_id"`
	}](args)
	if err != nil {
		return nil, err
	}
	id, err := parseUUIDArg(in.CampaignID, "campaign_id")
	if err != nil {
		return nil, err
	}
	c, err := s.Campaigns.Get(ctx, cc.StoreID, id, true)
	if err != nil {
		return nil, err
	}
	return map[string]any{"campaign": c}, nil
}

func (s *Service) listOrders(ctx context.Context, cc CallContext, args json.RawMessage) (any, error) {
	in, err := decodeArgs[struct {
		Query             string `json:"query"`
		OrderStatus       string `json:"order_status"`
		PaymentStatus     string `json:"payment_status"`
		FulfillmentStatus string `json:"fulfillment_status"`
		Limit             int    `json:"limit"`
		Offset            int    `json:"offset"`
	}](args)
	if err != nil {
		return nil, err
	}
	list, total, err := s.Commerce.ListOrders(ctx, cc.StoreID, commerce.ListOrdersFilter{
		Query:             in.Query,
		OrderStatus:       in.OrderStatus,
		PaymentStatus:     in.PaymentStatus,
		FulfillmentStatus: in.FulfillmentStatus,
		Limit:             clampLimit(in.Limit, 25, 100),
		Offset:            in.Offset,
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"orders": list, "total": total}, nil
}

func (s *Service) getOrder(ctx context.Context, cc CallContext, args json.RawMessage) (any, error) {
	in, err := decodeArgs[struct {
		OrderID string `json:"order_id"`
	}](args)
	if err != nil {
		return nil, err
	}
	id, err := parseUUIDArg(in.OrderID, "order_id")
	if err != nil {
		return nil, err
	}
	o, err := s.Commerce.GetOrder(ctx, cc.StoreID, id)
	if err != nil {
		return nil, err
	}
	return map[string]any{"order": o}, nil
}

func (s *Service) listOrderPayments(ctx context.Context, cc CallContext, args json.RawMessage) (any, error) {
	in, err := decodeArgs[struct {
		OrderID string `json:"order_id"`
	}](args)
	if err != nil {
		return nil, err
	}
	id, err := parseUUIDArg(in.OrderID, "order_id")
	if err != nil {
		return nil, err
	}
	list, err := s.Commerce.ListOrderPayments(ctx, cc.StoreID, id)
	if err != nil {
		return nil, err
	}
	return map[string]any{"payments": list}, nil
}

func (s *Service) listCustomers(ctx context.Context, cc CallContext, args json.RawMessage) (any, error) {
	in, err := decodeArgs[struct {
		Query  string `json:"query"`
		Limit  int    `json:"limit"`
		Offset int    `json:"offset"`
	}](args)
	if err != nil {
		return nil, err
	}
	list, total, err := s.Commerce.ListCustomers(ctx, cc.StoreID, in.Query, clampLimit(in.Limit, 25, 100), in.Offset)
	if err != nil {
		return nil, err
	}
	return map[string]any{"customers": list, "total": total}, nil
}

func (s *Service) getCustomer(ctx context.Context, cc CallContext, args json.RawMessage) (any, error) {
	in, err := decodeArgs[struct {
		CustomerID string `json:"customer_id"`
	}](args)
	if err != nil {
		return nil, err
	}
	id, err := parseUUIDArg(in.CustomerID, "customer_id")
	if err != nil {
		return nil, err
	}
	c, err := s.Commerce.GetCustomer(ctx, cc.StoreID, id)
	if err != nil {
		return nil, err
	}
	return map[string]any{"customer": c}, nil
}

func (s *Service) createCustomer(ctx context.Context, cc CallContext, args json.RawMessage) (any, error) {
	in, err := decodeArgs[commerce.CreateCustomerInput](args)
	if err != nil {
		return nil, err
	}
	res, err := s.Commerce.CreateCustomer(ctx, cc.StoreID, in)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *Service) listPreorders(ctx context.Context, cc CallContext, _ json.RawMessage) (any, error) {
	list, err := s.Commerce.ListBatches(ctx, cc.StoreID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"preorders": list}, nil
}

func (s *Service) getPreorder(ctx context.Context, cc CallContext, args json.RawMessage) (any, error) {
	in, err := decodeArgs[struct {
		BatchID string `json:"batch_id"`
	}](args)
	if err != nil {
		return nil, err
	}
	id, err := parseUUIDArg(in.BatchID, "batch_id")
	if err != nil {
		return nil, err
	}
	b, err := s.Commerce.GetBatch(ctx, cc.StoreID, id, true)
	if err != nil {
		return nil, err
	}
	return map[string]any{"preorder": b}, nil
}

func (s *Service) getPurchasingSummary(ctx context.Context, cc CallContext, args json.RawMessage) (any, error) {
	in, err := decodeArgs[struct {
		BatchID    string `json:"batch_id"`
		CampaignID string `json:"campaign_id"`
		GroupBy    string `json:"group_by"`
	}](args)
	if err != nil {
		return nil, err
	}
	f := commerce.PurchasingFilter{GroupBy: in.GroupBy}
	if in.BatchID != "" {
		id, err := parseUUIDArg(in.BatchID, "batch_id")
		if err != nil {
			return nil, err
		}
		f.BatchID = &id
	}
	if in.CampaignID != "" {
		id, err := parseUUIDArg(in.CampaignID, "campaign_id")
		if err != nil {
			return nil, err
		}
		f.CampaignID = &id
	}
	list, err := s.Commerce.AggregatePurchasing(ctx, cc.StoreID, f)
	if err != nil {
		return nil, err
	}
	return map[string]any{"lines": list}, nil
}

func (s *Service) getPackingQueue(ctx context.Context, cc CallContext, args json.RawMessage) (any, error) {
	in, err := decodeArgs[struct {
		GroupBy string `json:"group_by"`
		Status  string `json:"status"`
	}](args)
	if err != nil {
		return nil, err
	}
	out, err := s.Commerce.ListPackingQueue(ctx, cc.StoreID, in.GroupBy, in.Status)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Service) listShipments(ctx context.Context, cc CallContext, args json.RawMessage) (any, error) {
	in, err := decodeArgs[struct {
		OrderID string `json:"order_id"`
		Limit   int    `json:"limit"`
		Offset  int    `json:"offset"`
	}](args)
	if err != nil {
		return nil, err
	}
	var orderID *uuid.UUID
	if in.OrderID != "" {
		id, err := parseUUIDArg(in.OrderID, "order_id")
		if err != nil {
			return nil, err
		}
		orderID = &id
	}
	list, total, err := s.Commerce.ListShipments(ctx, cc.StoreID, orderID, clampLimit(in.Limit, 25, 100), in.Offset)
	if err != nil {
		return nil, err
	}
	return map[string]any{"shipments": list, "total": total}, nil
}

func (s *Service) getFinanceWorkspace(ctx context.Context, cc CallContext, args json.RawMessage) (any, error) {
	in, err := decodeArgs[struct {
		SettlementStatus string `json:"settlement_status"`
		Limit            int    `json:"limit"`
		Offset           int    `json:"offset"`
	}](args)
	if err != nil {
		return nil, err
	}
	list, total, err := s.Commerce.ListFinanceWorkspace(ctx, cc.StoreID, in.SettlementStatus, clampLimit(in.Limit, 25, 100), in.Offset)
	if err != nil {
		return nil, err
	}
	return map[string]any{"orders": list, "total": total}, nil
}

func (s *Service) getOrderFinance(ctx context.Context, cc CallContext, args json.RawMessage) (any, error) {
	in, err := decodeArgs[struct {
		OrderID string `json:"order_id"`
	}](args)
	if err != nil {
		return nil, err
	}
	id, err := parseUUIDArg(in.OrderID, "order_id")
	if err != nil {
		return nil, err
	}
	sum, err := s.Commerce.GetOrderFinance(ctx, cc.StoreID, id)
	if err != nil {
		return nil, err
	}
	return map[string]any{"finance": sum}, nil
}

func (s *Service) getSubscription(ctx context.Context, cc CallContext, _ json.RawMessage) (any, error) {
	snap, err := s.Ent.Snapshot(ctx, cc.StoreID)
	if err != nil {
		return nil, err
	}
	return snap, nil
}

func (s *Service) getSubscriptionUsage(ctx context.Context, cc CallContext, _ json.RawMessage) (any, error) {
	snap, err := s.Ent.Snapshot(ctx, cc.StoreID)
	if err != nil {
		return nil, err
	}
	period := ""
	if len(snap.Usage) > 0 {
		period = snap.Usage[0].PeriodYM
	}
	return map[string]any{
		"usage":  snap.Usage,
		"limits": snap.Limits,
		"period": period,
	}, nil
}

func (s *Service) getAnalyticsOverview(ctx context.Context, cc CallContext, args json.RawMessage) (any, error) {
	in, err := decodeArgs[struct {
		From string `json:"from"`
		To   string `json:"to"`
	}](args)
	if err != nil {
		return nil, err
	}
	tz := "Asia/Jakarta"
	if s.Cfg != nil && s.Cfg.DefaultTZ != "" {
		tz = s.Cfg.DefaultTZ
	}
	ov, err := s.Analytics.Overview(ctx, cc.StoreID, in.From, in.To, tz)
	if err != nil {
		return nil, err
	}
	return map[string]any{"analytics": ov}, nil
}

func (s *Service) listNotifications(ctx context.Context, cc CallContext, args json.RawMessage) (any, error) {
	in, err := decodeArgs[struct {
		UnreadOnly bool `json:"unread_only"`
		Limit      int  `json:"limit"`
		Offset     int  `json:"offset"`
	}](args)
	if err != nil {
		return nil, err
	}
	list, total, err := s.Notifications.List(ctx, cc.StoreID, in.UnreadOnly, clampLimit(in.Limit, 20, 100), in.Offset)
	if err != nil {
		return nil, err
	}
	return map[string]any{"notifications": list, "total": total}, nil
}

func (s *Service) getFulfillmentSettings(ctx context.Context, cc CallContext, _ json.RawMessage) (any, error) {
	fs, err := s.Commerce.GetFulfillmentSettings(ctx, cc.StoreID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"fulfillment": fs}, nil
}

func (s *Service) listPaymentMethods(ctx context.Context, cc CallContext, _ json.RawMessage) (any, error) {
	list, err := s.Commerce.ListPaymentMethods(ctx, cc.StoreID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"payment_methods": list}, nil
}

func (s *Service) listTeamMembers(ctx context.Context, cc CallContext, _ json.RawMessage) (any, error) {
	list, err := s.Team.ListMembers(ctx, cc.StoreID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"members": list}, nil
}

func (s *Service) listVendors(ctx context.Context, cc CallContext, _ json.RawMessage) (any, error) {
	list, err := s.Catalog.ListVendors(ctx, cc.StoreID, false)
	if err != nil {
		return nil, err
	}
	return map[string]any{"vendors": list}, nil
}
