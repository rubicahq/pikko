package entitlements

// Feature keys — only released product capabilities. Do not add unreleased features.
const (
	FeatureStorefront         = "storefront"
	FeatureProducts           = "products"
	FeatureCatalogs           = "catalogs"
	FeatureCatalogFlipbook    = "catalog_flipbook"
	FeatureOrders             = "orders"
	FeatureCheckout           = "checkout"
	FeaturePayments           = "payments"
	FeatureFinance            = "finance"
	FeatureRefunds            = "refunds"
	FeaturePacking            = "packing"
	FeatureShipments          = "shipments"
	FeatureNotifications      = "notifications"
	FeatureCustomerPortal     = "customer_portal"
	FeatureThemes             = "themes"
	FeatureBranding           = "branding"
	FeatureMultiStore         = "multi_store"
	FeatureCampaigns          = "campaigns"
	FeaturePreorders          = "preorders"
	FeatureBusinessRules      = "business_rules"
	FeatureAIPDFImport        = "ai_pdf_import"
	FeatureAnalytics          = "analytics"
	FeatureTeam               = "team"
	FeatureWhatsAppTemplates  = "whatsapp_templates"
	FeaturePurchasing         = "purchasing"
	FeatureReceiving          = "receiving"
	FeatureVendors            = "vendors"
	FeatureLiveShopping       = "live_shopping"
	FeatureProductAttributes  = "product_attributes"
)

// Limit keys — Jakarta calendar-month usage where noted.
const (
	LimitOrdersPerMonth       = "orders_per_month"
	LimitActiveProducts       = "active_products"
	LimitCatalogs             = "catalogs"
	LimitCampaigns            = "campaigns"
	LimitTeamMembers          = "team_members"
	LimitStorageGB            = "storage_gb"
	LimitAIPDFPagesPerMonth   = "ai_pdf_pages_per_month"
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
}

var AllLimitKeys = []string{
	LimitOrdersPerMonth, LimitActiveProducts, LimitCatalogs, LimitCampaigns,
	LimitTeamMembers, LimitStorageGB, LimitAIPDFPagesPerMonth,
}

var PlanOrder = []string{"basic", "standard", "premium", "ultimate"}
