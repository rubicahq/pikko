package entitlements

// Feature keys — released product capabilities plus reserved keys for unreleased upsells.
// Gate only where product exists; reserved keys may be seeded enabled on higher plans without UI.
const (
	FeatureStorefront        = "storefront"
	FeatureProducts          = "products"
	FeatureCatalogs          = "catalogs"
	FeatureCatalogFlipbook   = "catalog_flipbook"
	FeatureOrders            = "orders"
	FeatureCheckout          = "checkout"
	FeaturePayments          = "payments"
	FeatureFinance           = "finance"
	FeatureRefunds           = "refunds"
	FeaturePacking           = "packing"
	FeatureShipments         = "shipments"
	FeatureNotifications     = "notifications"
	FeatureCustomerPortal    = "customer_portal"
	FeatureThemes            = "themes"
	FeatureBranding          = "branding"
	FeatureMultiStore        = "multi_store"
	FeatureCampaigns         = "campaigns"
	FeaturePreorders         = "preorders"
	FeatureBusinessRules     = "business_rules"
	FeatureAIPDFImport       = "ai_pdf_import"
	FeatureAnalytics         = "analytics"
	FeatureTeam              = "team"
	FeatureWhatsAppTemplates = "whatsapp_templates"
	FeaturePurchasing        = "purchasing"
	FeatureReceiving         = "receiving"
	FeatureVendors           = "vendors"
	FeatureLiveShopping      = "live_shopping"
	FeatureProductAttributes = "product_attributes"

	// Standard+
	FeatureRemoveBranding          = "remove_branding"
	FeatureQuickOrder              = "quick_order"
	FeatureBulkImport              = "bulk_import"
	FeaturePackingByCustomer       = "packing_by_customer"
	FeatureConsolidatedPayments    = "consolidated_payments"
	FeatureWhatsAppTemplatesCustom = "whatsapp_templates_custom" // reserved — no custom-template UI yet

	// Premium+
	FeatureCustomDomain            = "custom_domain" // reserved — post-MVP
	FeatureBulkOps                 = "bulk_ops"
	FeatureAdvancedReconciliation  = "advanced_reconciliation" // reserved differentiation
	FeatureStoreCredit             = "store_credit"
	FeatureAdvancedRules           = "advanced_rules"     // reserved differentiation
	FeatureAdvancedAnalytics       = "advanced_analytics" // reserved differentiation
	FeatureGranularPermissions     = "granular_permissions"

	// Ultimate+
	FeatureMerchantAuditLog = "merchant_audit_log" // reserved — no merchant UI yet
	FeaturePrioritySupport  = "priority_support"   // ops / support flag only
)

// Limit keys — Jakarta calendar-month usage where noted.
const (
	LimitOrdersPerMonth     = "orders_per_month"
	LimitActiveProducts     = "active_products"
	LimitCatalogs           = "catalogs" // active catalogs
	LimitCampaigns          = "campaigns" // active campaigns
	LimitTeamMembers        = "team_members"
	LimitStorageGB          = "storage_gb"
	LimitAIPDFPagesPerMonth = "ai_pdf_pages_per_month"
)

const Unlimited int64 = -1

var AllFeatureKeys = []string{
	FeatureStorefront, FeatureProducts, FeatureCatalogs, FeatureCatalogFlipbook,
	FeatureOrders, FeatureCheckout, FeaturePayments, FeatureFinance, FeatureRefunds,
	FeaturePacking, FeatureShipments, FeatureNotifications, FeatureCustomerPortal,
	FeatureThemes, FeatureBranding, FeatureMultiStore,
	FeatureCampaigns, FeaturePreorders, FeatureBusinessRules, FeatureAIPDFImport,
	FeatureAnalytics, FeatureTeam, FeatureWhatsAppTemplates, FeaturePurchasing,
	FeatureReceiving, FeatureVendors, FeatureLiveShopping, FeatureProductAttributes,
	FeatureRemoveBranding, FeatureQuickOrder, FeatureBulkImport, FeaturePackingByCustomer,
	FeatureConsolidatedPayments, FeatureWhatsAppTemplatesCustom,
	FeatureCustomDomain, FeatureBulkOps, FeatureAdvancedReconciliation, FeatureStoreCredit,
	FeatureAdvancedRules, FeatureAdvancedAnalytics, FeatureGranularPermissions,
	FeatureMerchantAuditLog, FeaturePrioritySupport,
}

var AllLimitKeys = []string{
	LimitOrdersPerMonth, LimitActiveProducts, LimitCatalogs, LimitCampaigns,
	LimitTeamMembers, LimitStorageGB, LimitAIPDFPagesPerMonth,
}

var PlanOrder = []string{"basic", "standard", "premium", "ultimate"}

// ReleasedMarketingFeatures are safe to list on public /pricing (product exists).
var ReleasedMarketingFeatures = []string{
	FeatureStorefront, FeatureCatalogs, FeatureCatalogFlipbook, FeatureAIPDFImport,
	FeatureCampaigns, FeaturePreorders, FeatureLiveShopping, FeatureOrders,
	FeaturePurchasing, FeatureReceiving, FeaturePacking, FeatureShipments,
	FeatureCustomerPortal, FeatureWhatsAppTemplates, FeatureBusinessRules, FeatureAnalytics,
	FeatureProductAttributes, FeatureRemoveBranding, FeatureQuickOrder, FeatureBulkImport,
	FeaturePackingByCustomer, FeatureConsolidatedPayments, FeatureTeam, FeatureBulkOps,
	FeatureStoreCredit,
}
