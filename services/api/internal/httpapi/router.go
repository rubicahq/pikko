package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rubicahq/pikko/services/api/internal/account"
	"github.com/rubicahq/pikko/services/api/internal/ai"
	"github.com/rubicahq/pikko/services/api/internal/analytics"
	"github.com/rubicahq/pikko/services/api/internal/auth"
	"github.com/rubicahq/pikko/services/api/internal/campaigns"
	"github.com/rubicahq/pikko/services/api/internal/catalog"
	"github.com/rubicahq/pikko/services/api/internal/catalogs"
	"github.com/rubicahq/pikko/services/api/internal/commerce"
	"github.com/rubicahq/pikko/services/api/internal/config"
	"github.com/rubicahq/pikko/services/api/internal/connie"
	"github.com/rubicahq/pikko/services/api/internal/email"
	"github.com/rubicahq/pikko/services/api/internal/entitlements"
	"github.com/rubicahq/pikko/services/api/internal/files"
	"github.com/rubicahq/pikko/services/api/internal/flipbook"
	"github.com/rubicahq/pikko/services/api/internal/imports"
	"github.com/rubicahq/pikko/services/api/internal/jobs"
	"github.com/rubicahq/pikko/services/api/internal/notifications"
	"github.com/rubicahq/pikko/services/api/internal/pdfutil"
	"github.com/rubicahq/pikko/services/api/internal/portal"
	"github.com/rubicahq/pikko/services/api/internal/rbac"
	"github.com/rubicahq/pikko/services/api/internal/storage"
	"github.com/rubicahq/pikko/services/api/internal/storefront"
	"github.com/rubicahq/pikko/services/api/internal/stores"
	"github.com/rubicahq/pikko/services/api/internal/syshealth"
	"github.com/rubicahq/pikko/services/api/internal/team"
	"github.com/rubicahq/pikko/services/api/internal/urls"
	"github.com/rubicahq/pikko/services/api/internal/whatsapp"
)

type Deps struct {
	Cfg     *config.Config
	DB      *pgxpool.Pool
	Logger  *slog.Logger
	Storage storage.ObjectStore
	// Optional overrides for tests
	AI       ai.CatalogExtractor
	Renderer pdfutil.Renderer
}

// RouterExtras exposes background helpers started by cmd/api (not by tests).
type RouterExtras struct {
	Health  *syshealth.Service
	Checker *syshealth.Checker
	Alerter *syshealth.Alerter
}

// NewRouter builds the HTTP handler and an optional background job worker.
func NewRouter(d Deps) (http.Handler, *jobs.Worker) {
	h, w, _ := NewRouterWithExtras(d)
	return h, w
}

func NewRouterWithExtras(d Deps) (http.Handler, *jobs.Worker, RouterExtras) {
	urlGen := urls.New(d.Cfg.MarketingURL, d.Cfg.DashboardURL, d.Cfg.StorefrontURL, d.Cfg.APIBaseURL).WithHQ(d.Cfg.HQURL)
	mailSender := email.NewSender(d.Cfg.EmailMode, d.Cfg.BrevoAPIKey, d.Logger)
	mailSvc := &email.Service{DB: d.DB, Sender: mailSender, Logger: d.Logger, DefaultFromName: d.Cfg.EmailFromName}
	authSvc := &auth.Service{DB: d.DB, Cfg: d.Cfg, Email: mailSvc, URLs: urlGen, Logger: d.Logger}
	healthSvc := &syshealth.Service{
		DB: d.DB, Logger: d.Logger, DeployVersion: d.Cfg.DeployVersion, Environment: d.Cfg.AppEnv,
	}
	healthChecker := &syshealth.Checker{DB: d.DB, Cfg: d.Cfg, Storage: d.Storage, Health: healthSvc}
	healthAlerter := &syshealth.Alerter{
		Health: healthSvc, Email: mailSvc,
		From: d.Cfg.EmailNotificationFrom, FromName: d.Cfg.EmailFromName,
	}
	hqH := &HQHandlers{Auth: authSvc, Health: healthSvc, Checker: healthChecker, Alerter: healthAlerter, Cfg: d.Cfg}
	hqMW := &HQAuthMiddleware{Auth: authSvc, Health: healthSvc, Cfg: d.Cfg}
	storeSvc := &stores.Service{DB: d.DB, Cfg: d.Cfg}
	accountSvc := &account.Service{DB: d.DB}
	storefrontSvc := &storefront.Service{DB: d.DB}
	catalogSvc := &catalog.Service{DB: d.DB}
	filesSvc := &files.Service{DB: d.DB, Storage: d.Storage}
	queue := &jobs.Queue{DB: d.DB}
	outbox := &jobs.Outbox{DB: d.DB}
	extractor := d.AI
	if extractor == nil {
		extractor = ai.NewExtractor(d.Cfg.AIAPIKey, d.Cfg.AIBaseURL, d.Cfg.AIModel)
	}
	renderer := d.Renderer
	if renderer == nil {
		renderer = pdfutil.NewRenderer()
	}
	entSvc := &entitlements.Service{DB: d.DB, Cfg: d.Cfg}
	importSvc := &imports.Service{
		DB: d.DB, Files: filesSvc, Catalog: catalogSvc, Queue: queue, Outbox: outbox,
		AI: extractor, Renderer: renderer, Storage: d.Storage, Logger: d.Logger,
		Ent: entSvc,
		Limits: imports.Limits{
			MaxPDFBytes: d.Cfg.ImportMaxPDFBytes,
			MaxPages:    d.Cfg.ImportMaxPages,
			MaxItems:    d.Cfg.ImportMaxItems,
		},
	}
	if importSvc.Limits.MaxPDFBytes == 0 {
		importSvc.Limits = imports.DefaultLimits()
	}
	billingH := &BillingHandlers{Ent: entSvc}
	adminBillingH := &AdminBillingHandlers{Ent: entSvc, Tok: d.Cfg.SubscriptionAdminToken}

	authH := &AuthHandlers{Auth: authSvc, Cfg: d.Cfg}
	accountH := &AccountHandlers{Account: accountSvc}
	storeH := &StoreHandlers{Stores: storeSvc, URLs: urlGen, Cfg: d.Cfg, Ent: entSvc}
	teamSvc := &team.Service{DB: d.DB, Cfg: d.Cfg, Email: mailSvc, URLs: urlGen, Logger: d.Logger}
	teamH := &TeamHandlers{Team: teamSvc, Cfg: d.Cfg, Ent: entSvc}
	p2 := &Phase2Handlers{Storefront: storefrontSvc, Catalog: catalogSvc, Files: filesSvc, URLs: urlGen, Ent: entSvc}
	impH := &ImportHandlers{Imports: importSvc, Ent: entSvc}
	flipSvc := &flipbook.Service{DB: d.DB}
	flipH := &FlipbookHandlers{Flipbook: flipSvc, Catalog: catalogSvc, URLs: urlGen}
	catalogDocSvc := &catalogs.Service{DB: d.DB}
	catalogDocH := &CatalogDocHandlers{Catalogs: catalogDocSvc, URLs: urlGen}
	relH := &ProductRelationHandlers{Catalog: catalogSvc}
	notifSvc := &notifications.Service{DB: d.DB}
	analyticsSvc := &analytics.Service{DB: d.DB}
	commerceSvc := &commerce.Service{
		DB: d.DB, Cfg: d.Cfg, Files: filesSvc, Storage: d.Storage,
		Email: mailSvc, Outbox: outbox, URLs: urlGen, Logger: d.Logger,
		Notifier: notifSvc,
	}
	campaignSvc := &campaigns.Service{DB: d.DB}
	campaignH := &CampaignHandlers{Campaigns: campaignSvc, Ent: entSvc}
	commerceH := &CommerceHandlers{Commerce: commerceSvc, Campaigns: campaignSvc, URLs: urlGen, Cfg: d.Cfg, Ent: entSvc}
	portalSvc := &portal.Service{
		DB: d.DB, Cfg: d.Cfg, Email: mailSvc, URLs: urlGen, Logger: d.Logger, Commerce: commerceSvc,
	}
	p5 := &Phase5Handlers{Notifications: notifSvc, Analytics: analyticsSvc, Commerce: commerceSvc, Portal: portalSvc, Ent: entSvc}
	portalH := &PortalHandlers{Portal: portalSvc, Cfg: d.Cfg}
	waSvc := &whatsapp.Service{DB: d.DB, URLs: urlGen}
	waH := &WhatsAppHandlers{WA: waSvc}

	connieSvc := &connie.Service{
		Cfg: d.Cfg, Logger: d.Logger,
		Stores: storeSvc, Catalog: catalogSvc, Catalogs: catalogDocSvc,
		Campaigns: campaignSvc, Commerce: commerceSvc, Storefront: storefrontSvc,
		Analytics: analyticsSvc, Notifications: notifSvc, Team: teamSvc, Ent: entSvc,
	}
	connieH := &ConnieHandlers{Connie: connieSvc}

	authMW := &AuthMiddleware{Auth: authSvc, Cfg: d.Cfg}
	storeMW := &StoreAuthMiddleware{Stores: storeSvc}

	mux := http.NewServeMux()

	registerHealth(mux, healthDeps{DB: d.DB, Cfg: d.Cfg, Storage: d.Storage})

	// Auth (public)
	mux.HandleFunc("POST /api/v1/auth/magic-link/request", authH.RequestMagicLink)
	mux.HandleFunc("GET /api/v1/auth/magic-link/peek", authH.PeekMagicLink)
	mux.HandleFunc("POST /api/v1/auth/magic-link/verify", authH.VerifyMagicLink)
	mux.HandleFunc("GET /api/v1/auth/session", authH.Session)
	mux.HandleFunc("POST /api/v1/auth/logout", authH.Logout)

	// HQ BO (Tim Pikko) — separate from merchant dashboard
	mux.HandleFunc("POST /api/v1/hq/auth/magic-link/request", hqH.RequestMagicLink)
	mux.HandleFunc("GET /api/v1/hq/auth/magic-link/peek", hqH.PeekMagicLink)
	mux.HandleFunc("POST /api/v1/hq/auth/magic-link/verify", hqH.VerifyMagicLink)
	mux.HandleFunc("GET /api/v1/hq/auth/session", hqH.Session)
	mux.HandleFunc("POST /api/v1/hq/auth/logout", hqH.Logout)
	hqAuthed := func(fn http.HandlerFunc) http.Handler { return hqMW.Require(fn) }
	mux.Handle("GET /api/v1/hq/system-health/overview", hqAuthed(hqH.Overview))
	mux.Handle("GET /api/v1/hq/system-health/issues", hqAuthed(hqH.ListIssues))
	mux.Handle("GET /api/v1/hq/system-health/issues/{issueID}", hqAuthed(hqH.GetIssue))
	mux.Handle("PATCH /api/v1/hq/system-health/issues/{issueID}", hqAuthed(hqH.PatchIssue))
	mux.Handle("GET /api/v1/hq/system-health/jobs/stats", hqAuthed(hqH.JobStats))
	mux.Handle("GET /api/v1/hq/system-health/jobs", hqAuthed(hqH.ListJobs))
	mux.Handle("GET /api/v1/hq/system-health/jobs/{jobID}", hqAuthed(hqH.GetJob))
	mux.Handle("POST /api/v1/hq/system-health/jobs/{jobID}/retry", hqAuthed(hqH.RetryJob))
	mux.Handle("GET /api/v1/hq/system-health/services", hqAuthed(hqH.ListServices))
	mux.Handle("GET /api/v1/hq/system-health/usage", hqAuthed(hqH.Usage))
	mux.Handle("GET /api/v1/hq/system-health/alerts/rules", hqAuthed(hqH.ListAlertRules))
	mux.Handle("PATCH /api/v1/hq/system-health/alerts/rules/{ruleID}", hqAuthed(hqH.PatchAlertRule))
	mux.Handle("GET /api/v1/hq/system-health/alerts/events", hqAuthed(hqH.ListAlertEvents))
	mux.Handle("POST /api/v1/hq/system-health/alerts/evaluate", hqAuthed(hqH.EvaluateAlerts))

	// Authenticated
	mux.Handle("POST /api/v1/auth/logout-all", authMW.Require(http.HandlerFunc(authH.LogoutAll)))
	mux.Handle("GET /api/v1/auth/sessions", authMW.Require(http.HandlerFunc(authH.ListSessions)))
	mux.Handle("DELETE /api/v1/auth/sessions/{sessionID}", authMW.Require(http.HandlerFunc(authH.RevokeSession)))

	mux.Handle("GET /api/v1/account", authMW.Require(http.HandlerFunc(accountH.Get)))
	mux.Handle("PATCH /api/v1/account", authMW.Require(http.HandlerFunc(accountH.Patch)))

	mux.Handle("GET /api/v1/meta/categories", authMW.Require(http.HandlerFunc(storeH.Categories)))
	mux.Handle("GET /api/v1/stores", authMW.Require(http.HandlerFunc(storeH.List)))
	mux.Handle("POST /api/v1/stores", authMW.Require(http.HandlerFunc(storeH.Create)))
	mux.Handle("GET /api/v1/stores/slug-available", authMW.Require(http.HandlerFunc(storeH.CheckSlug)))
	mux.Handle("GET /api/v1/stores/suggest-slug", authMW.Require(http.HandlerFunc(storeH.SuggestSlug)))

	mux.Handle("GET /api/v1/onboarding/draft", authMW.Require(http.HandlerFunc(storeH.GetDraft)))
	mux.Handle("POST /api/v1/onboarding/identity", authMW.Require(http.HandlerFunc(storeH.SaveIdentity)))
	mux.Handle("POST /api/v1/onboarding/selling-model", authMW.Require(http.HandlerFunc(storeH.SaveSellingModel)))
	mux.Handle("POST /api/v1/onboarding/plan", authMW.Require(http.HandlerFunc(storeH.SavePlan)))
	mux.Handle("POST /api/v1/onboarding/complete", authMW.Require(http.HandlerFunc(storeH.CompleteOnboarding)))

	mux.HandleFunc("GET /api/v1/billing/plans", billingH.ListPlans)
	mux.HandleFunc("GET /api/v1/admin/subscription/plans", adminBillingH.ListPlans)
	mux.HandleFunc("PATCH /api/v1/admin/subscription/plans/{planCode}", adminBillingH.PatchPlan)
	mux.HandleFunc("PUT /api/v1/admin/subscription/flags", adminBillingH.PatchFlags)
	mux.HandleFunc("POST /api/v1/admin/stores/{storeID}/subscription/overrides", adminBillingH.SetStoreOverride)

	// Explicit store-scoped routes with membership + optional permission checks
	storeAuthed := func(h http.HandlerFunc) http.Handler {
		return authMW.Require(storeMW.Require(h))
	}
	storePerm := func(perm rbac.Permission, h http.HandlerFunc) http.Handler {
		return authMW.Require(storeMW.RequirePermission(perm)(h))
	}

	mux.Handle("GET /api/v1/stores/{storeID}/subscription", storePerm(rbac.PermStoresSettings, billingH.GetStoreSubscription))
	mux.Handle("POST /api/v1/stores/{storeID}/subscription/activate", storePerm(rbac.PermStoresSettings, billingH.Activate))
	mux.Handle("POST /api/v1/stores/{storeID}/subscription/change", storePerm(rbac.PermStoresSettings, billingH.Change))
	mux.Handle("GET /api/v1/stores/{storeID}/subscription/usage", storePerm(rbac.PermStoresRead, billingH.Usage))

	mux.Handle("GET /api/v1/stores/{storeID}", storePerm(rbac.PermStoresRead, storeH.Get))
	mux.Handle("PATCH /api/v1/stores/{storeID}", storePerm(rbac.PermStoresSettings, storeH.Patch))
	mux.Handle("POST /api/v1/stores/{storeID}/archive", storePerm(rbac.PermStoresArchive, storeH.Archive))

	// Team RBAC (Agent 03)
	mux.HandleFunc("GET /api/v1/team/invites/peek", teamH.PeekInvite)
	mux.HandleFunc("POST /api/v1/team/invites/accept", teamH.AcceptInvite)
	mux.Handle("GET /api/v1/stores/{storeID}/team/me", storePerm(rbac.PermStoresRead, teamH.Me))
	mux.Handle("GET /api/v1/stores/{storeID}/team/members", storePerm(rbac.PermTeamRead, teamH.ListMembers))
	mux.Handle("GET /api/v1/stores/{storeID}/team/invites", storePerm(rbac.PermTeamRead, teamH.ListInvites))
	mux.Handle("POST /api/v1/stores/{storeID}/team/invites", storePerm(rbac.PermTeamInvite, teamH.CreateInvite))
	mux.Handle("POST /api/v1/stores/{storeID}/team/invites/{inviteID}/resend", storePerm(rbac.PermTeamInvite, teamH.ResendInvite))
	mux.Handle("POST /api/v1/stores/{storeID}/team/invites/{inviteID}/cancel", storePerm(rbac.PermTeamManage, teamH.CancelInvite))
	mux.Handle("POST /api/v1/stores/{storeID}/team/members/{membershipID}/revoke", storePerm(rbac.PermTeamManage, teamH.RevokeMember))
	mux.Handle("PATCH /api/v1/stores/{storeID}/team/members/{membershipID}", storePerm(rbac.PermTeamManage, teamH.PatchMember))

	// Phase 2: storefront, branding files, catalog
	mux.Handle("GET /api/v1/meta/themes", authMW.Require(http.HandlerFunc(p2.ListThemes)))
	mux.Handle("GET /api/v1/stores/{storeID}/storefront", storePerm(rbac.PermStorefrontRead, p2.GetStorefront))
	mux.Handle("PATCH /api/v1/stores/{storeID}/storefront", storePerm(rbac.PermStorefrontWrite, p2.PatchStorefront))
	mux.Handle("POST /api/v1/stores/{storeID}/storefront/publish", storePerm(rbac.PermStorefrontWrite, p2.PublishStorefront))
	mux.Handle("POST /api/v1/stores/{storeID}/storefront/discard", storePerm(rbac.PermStorefrontWrite, p2.DiscardStorefront))
	mux.Handle("PUT /api/v1/stores/{storeID}/storefront/featured", storePerm(rbac.PermStorefrontWrite, p2.SetFeatured))
	mux.Handle("GET /api/v1/stores/{storeID}/setup-checklist", storePerm(rbac.PermStoresRead, p2.SetupChecklist))

	mux.Handle("POST /api/v1/stores/{storeID}/files", storePerm(rbac.PermCatalogWrite, p2.UploadFile))
	mux.Handle("GET /api/v1/stores/{storeID}/files/{fileID}", storeAuthed(p2.GetFile))

	mux.Handle("GET /api/v1/stores/{storeID}/categories", storePerm(rbac.PermCatalogRead, p2.ListCategories))
	mux.Handle("POST /api/v1/stores/{storeID}/categories", storePerm(rbac.PermCatalogWrite, p2.CreateCategory))
	mux.Handle("PATCH /api/v1/stores/{storeID}/categories/{categoryID}", storePerm(rbac.PermCatalogWrite, p2.UpdateCategory))
	mux.Handle("DELETE /api/v1/stores/{storeID}/categories/{categoryID}", storePerm(rbac.PermCatalogWrite, p2.DeleteCategory))

	mux.Handle("GET /api/v1/stores/{storeID}/products", storePerm(rbac.PermCatalogRead, p2.ListProducts))
	mux.Handle("POST /api/v1/stores/{storeID}/products", storePerm(rbac.PermCatalogWrite, p2.CreateProduct))
	mux.Handle("GET /api/v1/stores/{storeID}/products/{productID}", storePerm(rbac.PermCatalogRead, p2.GetProduct))
	mux.Handle("PATCH /api/v1/stores/{storeID}/products/{productID}", storePerm(rbac.PermCatalogWrite, p2.UpdateProduct))
	mux.Handle("POST /api/v1/stores/{storeID}/products/{productID}/status", storePerm(rbac.PermCatalogWrite, p2.PatchProductStatus))
	mux.Handle("POST /api/v1/stores/{storeID}/products/{productID}/archive", storePerm(rbac.PermCatalogWrite, p2.ArchiveProduct))

	mux.Handle("GET /api/v1/stores/{storeID}/collections", storePerm(rbac.PermCatalogRead, p2.ListCollections))
	mux.Handle("POST /api/v1/stores/{storeID}/collections", storePerm(rbac.PermCatalogWrite, p2.CreateCollection))

	// Brands / vendors / custom attributes (Agent 04 catalog foundations)
	mux.Handle("GET /api/v1/stores/{storeID}/brands", storePerm(rbac.PermCatalogRead, p2.ListBrands))
	mux.Handle("POST /api/v1/stores/{storeID}/brands", storePerm(rbac.PermCatalogWrite, p2.CreateBrand))
	mux.Handle("PATCH /api/v1/stores/{storeID}/brands/{brandID}", storePerm(rbac.PermCatalogWrite, p2.UpdateBrand))
	mux.Handle("GET /api/v1/stores/{storeID}/vendors", storePerm(rbac.PermPurchasingRead, p2.ListVendors))
	mux.Handle("POST /api/v1/stores/{storeID}/vendors", storePerm(rbac.PermPurchasingWrite, p2.CreateVendor))
	mux.Handle("PATCH /api/v1/stores/{storeID}/vendors/{vendorID}", storePerm(rbac.PermPurchasingWrite, p2.UpdateVendor))
	mux.Handle("GET /api/v1/stores/{storeID}/products/{productID}/vendors", storePerm(rbac.PermCatalogRead, p2.ListProductVendors))
	mux.Handle("POST /api/v1/stores/{storeID}/products/{productID}/vendors", storePerm(rbac.PermCatalogWrite, p2.LinkProductVendor))
	mux.Handle("DELETE /api/v1/stores/{storeID}/products/{productID}/vendors/{vendorID}", storePerm(rbac.PermCatalogWrite, p2.UnlinkProductVendor))
	mux.Handle("GET /api/v1/stores/{storeID}/attributes", storePerm(rbac.PermCatalogRead, p2.ListAttributeDefinitions))
	mux.Handle("POST /api/v1/stores/{storeID}/attributes", storePerm(rbac.PermCatalogWrite, p2.CreateAttributeDefinition))
	mux.Handle("PATCH /api/v1/stores/{storeID}/attributes/{attributeID}", storePerm(rbac.PermCatalogWrite, p2.UpdateAttributeDefinition))
	mux.Handle("DELETE /api/v1/stores/{storeID}/attributes/{attributeID}", storePerm(rbac.PermCatalogWrite, p2.DeleteAttributeDefinition))
	mux.Handle("GET /api/v1/stores/{storeID}/products/{productID}/attributes", storePerm(rbac.PermCatalogRead, p2.ListProductAttributes))
	mux.Handle("PUT /api/v1/stores/{storeID}/products/{productID}/attributes", storePerm(rbac.PermCatalogWrite, p2.SetProductAttribute))
	mux.Handle("PUT /api/v1/stores/{storeID}/products/{productID}/attributes/bulk", storePerm(rbac.PermCatalogWrite, p2.SetProductAttributesBulk))
	mux.Handle("DELETE /api/v1/stores/{storeID}/products/{productID}/attributes/{attributeID}", storePerm(rbac.PermCatalogWrite, p2.DeleteProductAttribute))

	// Campaigns (Agent 04 — CAMP-05 unification)
	mux.Handle("GET /api/v1/stores/{storeID}/campaigns", storePerm(rbac.PermCampaignRead, campaignH.List))
	mux.Handle("POST /api/v1/stores/{storeID}/campaigns", storePerm(rbac.PermCampaignWrite, campaignH.Create))
	mux.Handle("GET /api/v1/stores/{storeID}/campaigns/{campaignID}", storePerm(rbac.PermCampaignRead, campaignH.Get))
	mux.Handle("PATCH /api/v1/stores/{storeID}/campaigns/{campaignID}", storePerm(rbac.PermCampaignWrite, campaignH.Patch))
	mux.Handle("POST /api/v1/stores/{storeID}/campaigns/{campaignID}/products", storePerm(rbac.PermCampaignWrite, campaignH.AssignProduct))
	mux.Handle("DELETE /api/v1/stores/{storeID}/campaigns/{campaignID}/products/{productID}", storePerm(rbac.PermCampaignWrite, campaignH.RemoveProduct))
	mux.Handle("GET /api/v1/stores/{storeID}/campaigns/{campaignID}/catalogs", storePerm(rbac.PermCampaignRead, catalogDocH.ListForCampaign))

	// Live shopping ops (Agent 12 — LIVE-01–03; not video)
	mux.Handle("GET /api/v1/stores/{storeID}/live", storePerm(rbac.PermCampaignRead, commerceH.ListLiveSessions))
	mux.Handle("GET /api/v1/stores/{storeID}/live/{campaignID}", storePerm(rbac.PermCampaignRead, commerceH.GetLiveSession))
	mux.Handle("GET /api/v1/stores/{storeID}/live/{campaignID}/feed", storePerm(rbac.PermOrdersRead, commerceH.LiveOrderFeed))
	mux.Handle("GET /api/v1/stores/{storeID}/live/{campaignID}/orders/search", storePerm(rbac.PermOrdersRead, commerceH.SearchLiveOrders))
	mux.Handle("POST /api/v1/stores/{storeID}/live/{campaignID}/quick-add", storePerm(rbac.PermOrdersWrite, commerceH.LiveQuickAdd))

	// Catalog import PDF + CSV (PRD §18)
	mux.Handle("POST /api/v1/stores/{storeID}/catalog-imports", storePerm(rbac.PermCatalogWrite, impH.Start))
	mux.Handle("GET /api/v1/stores/{storeID}/catalog-imports", storePerm(rbac.PermCatalogRead, impH.List))
	mux.Handle("GET /api/v1/stores/{storeID}/catalog-imports/csv-template", storePerm(rbac.PermCatalogRead, impH.CSVTemplate))
	mux.Handle("GET /api/v1/stores/{storeID}/catalog-imports/{jobID}", storePerm(rbac.PermCatalogRead, impH.Get))
	mux.Handle("GET /api/v1/stores/{storeID}/catalog-imports/{jobID}/items", storePerm(rbac.PermCatalogRead, impH.ListItems))
	mux.Handle("PATCH /api/v1/stores/{storeID}/catalog-imports/{jobID}/items/{itemID}", storePerm(rbac.PermCatalogWrite, impH.PatchItem))
	mux.Handle("POST /api/v1/stores/{storeID}/catalog-imports/{jobID}/items/bulk", storePerm(rbac.PermCatalogWrite, impH.BulkItems))
	mux.Handle("DELETE /api/v1/stores/{storeID}/catalog-imports/{jobID}/items/{itemID}", storePerm(rbac.PermCatalogWrite, impH.DeleteItem))
	mux.Handle("POST /api/v1/stores/{storeID}/catalog-imports/{jobID}/confirm", storePerm(rbac.PermCatalogWrite, impH.Confirm))
	mux.Handle("POST /api/v1/stores/{storeID}/catalog-imports/{jobID}/retry", storePerm(rbac.PermCatalogWrite, impH.Retry))

	// Flipbook catalog pages / hotspots
	mux.Handle("GET /api/v1/stores/{storeID}/catalog-pages", storePerm(rbac.PermCatalogRead, flipH.ListPages))
	mux.Handle("PUT /api/v1/stores/{storeID}/catalog-pages/order", storePerm(rbac.PermCatalogWrite, flipH.Reorder))
	mux.Handle("POST /api/v1/stores/{storeID}/catalog-pages/status", storePerm(rbac.PermCatalogWrite, flipH.Publish))
	mux.Handle("POST /api/v1/stores/{storeID}/catalog-pages/{pageID}/hotspots", storePerm(rbac.PermCatalogWrite, flipH.CreateHotspot))
	mux.Handle("PATCH /api/v1/stores/{storeID}/catalog-pages/hotspots/{hotspotID}", storePerm(rbac.PermCatalogWrite, flipH.UpdateHotspot))
	mux.Handle("DELETE /api/v1/stores/{storeID}/catalog-pages/hotspots/{hotspotID}", storePerm(rbac.PermCatalogWrite, flipH.DeleteHotspot))

	// PDF catalog documents + flexible campaign/preorder links
	mux.Handle("GET /api/v1/stores/{storeID}/catalogs", storePerm(rbac.PermCatalogRead, catalogDocH.List))
	mux.Handle("GET /api/v1/stores/{storeID}/catalogs/{catalogID}", storePerm(rbac.PermCatalogRead, catalogDocH.Get))
	mux.Handle("PATCH /api/v1/stores/{storeID}/catalogs/{catalogID}", storePerm(rbac.PermCatalogWrite, catalogDocH.Patch))
	mux.Handle("POST /api/v1/stores/{storeID}/catalogs/{catalogID}/campaigns", storePerm(rbac.PermCatalogWrite, catalogDocH.AttachCampaign))
	mux.Handle("DELETE /api/v1/stores/{storeID}/catalogs/{catalogID}/campaigns/{campaignID}", storePerm(rbac.PermCatalogWrite, catalogDocH.DetachCampaign))
	mux.Handle("POST /api/v1/stores/{storeID}/catalogs/{catalogID}/preorders", storePerm(rbac.PermCatalogWrite, catalogDocH.AttachPreorder))
	mux.Handle("DELETE /api/v1/stores/{storeID}/catalogs/{catalogID}/preorders/{batchID}", storePerm(rbac.PermCatalogWrite, catalogDocH.DetachPreorder))

	// Soft product relate / merge across catalogs
	mux.Handle("GET /api/v1/stores/{storeID}/products/{productID}/relations", storePerm(rbac.PermCatalogRead, relH.List))
	mux.Handle("POST /api/v1/stores/{storeID}/products/{productID}/relations", storePerm(rbac.PermCatalogWrite, relH.Create))
	mux.Handle("DELETE /api/v1/stores/{storeID}/products/{productID}/relations/{relationID}", storePerm(rbac.PermCatalogWrite, relH.Delete))

	// Phase 3: merchant commerce settings + orders
	mux.Handle("GET /api/v1/stores/{storeID}/payment-methods", storePerm(rbac.PermStoresSettings, commerceH.ListPaymentMethods))
	mux.Handle("POST /api/v1/stores/{storeID}/payment-methods", storePerm(rbac.PermStoresSettings, commerceH.CreatePaymentMethod))
	mux.Handle("PATCH /api/v1/stores/{storeID}/payment-methods/{methodID}", storePerm(rbac.PermStoresSettings, commerceH.UpdatePaymentMethod))
	mux.Handle("DELETE /api/v1/stores/{storeID}/payment-methods/{methodID}", storePerm(rbac.PermStoresSettings, commerceH.DeletePaymentMethod))
	mux.Handle("GET /api/v1/stores/{storeID}/fulfillment", storePerm(rbac.PermStoresSettings, commerceH.GetFulfillment))
	mux.Handle("PATCH /api/v1/stores/{storeID}/fulfillment", storePerm(rbac.PermStoresSettings, commerceH.PatchFulfillment))
	mux.Handle("GET /api/v1/stores/{storeID}/orders", storePerm(rbac.PermOrdersRead, commerceH.ListOrders))
	mux.Handle("POST /api/v1/stores/{storeID}/orders", storePerm(rbac.PermOrdersWrite, commerceH.CreateAdminOrder))
	// Admin entry helpers live under /order-entry to avoid ServeMux conflicts with /orders/{orderID}/…
	mux.Handle("POST /api/v1/stores/{storeID}/order-entry/validate", storePerm(rbac.PermOrdersWrite, commerceH.ValidateOrderLines))
	mux.Handle("GET /api/v1/stores/{storeID}/order-entry/catalog-suggest", storePerm(rbac.PermOrdersRead, commerceH.SuggestOrderCatalog))
	mux.Handle("GET /api/v1/stores/{storeID}/order-entry/drafts", storePerm(rbac.PermOrdersWrite, commerceH.ListOrderDrafts))
	mux.Handle("POST /api/v1/stores/{storeID}/order-entry/drafts", storePerm(rbac.PermOrdersWrite, commerceH.SaveOrderDraft))
	mux.Handle("GET /api/v1/stores/{storeID}/order-entry/drafts/{draftID}", storePerm(rbac.PermOrdersWrite, commerceH.GetOrderDraft))
	mux.Handle("PUT /api/v1/stores/{storeID}/order-entry/drafts/{draftID}", storePerm(rbac.PermOrdersWrite, commerceH.SaveOrderDraft))
	mux.Handle("POST /api/v1/stores/{storeID}/order-entry/drafts/{draftID}/validate", storePerm(rbac.PermOrdersWrite, commerceH.ValidateOrderDraft))
	mux.Handle("DELETE /api/v1/stores/{storeID}/order-entry/drafts/{draftID}", storePerm(rbac.PermOrdersWrite, commerceH.DeleteOrderDraft))
	mux.Handle("GET /api/v1/stores/{storeID}/order-entry/csv-template", storePerm(rbac.PermOrdersRead, commerceH.OrderCSVTemplate))
	mux.Handle("POST /api/v1/stores/{storeID}/order-entry/csv-preview", storePerm(rbac.PermOrdersWrite, commerceH.PreviewOrderCSV))
	mux.Handle("POST /api/v1/stores/{storeID}/order-entry/csv-import", storePerm(rbac.PermOrdersWrite, commerceH.ImportOrderCSV))
	mux.Handle("GET /api/v1/stores/{storeID}/orders/evidence/{evidenceID}/file", storePerm(rbac.PermOrdersRead, commerceH.MerchantEvidenceFile))
	mux.Handle("GET /api/v1/stores/{storeID}/orders/{orderID}", storePerm(rbac.PermOrdersRead, commerceH.GetOrder))
	mux.Handle("PATCH /api/v1/stores/{storeID}/orders/{orderID}", storePerm(rbac.PermOrdersWrite, commerceH.PatchOrder))
	mux.Handle("GET /api/v1/stores/{storeID}/orders/{orderID}/revisions", storePerm(rbac.PermOrdersRead, commerceH.ListOrderRevisions))
	mux.Handle("GET /api/v1/stores/{storeID}/orders/{orderID}/item-events", storePerm(rbac.PermOrdersRead, commerceH.ListOrderItemEvents))
	mux.Handle("PATCH /api/v1/stores/{storeID}/orders/{orderID}/items/{itemID}", storePerm(rbac.PermOrdersWrite, commerceH.AdjustOrderItemQty))
	mux.Handle("POST /api/v1/stores/{storeID}/orders/{orderID}/payments/paid-by-admin", storePerm(rbac.PermOrdersPayments, commerceH.MarkPaidByAdmin))
	mux.Handle("POST /api/v1/stores/{storeID}/orders/{orderID}/payments/{paymentID}/verify", storePerm(rbac.PermOrdersPayments, commerceH.VerifyPayment))
	mux.Handle("POST /api/v1/stores/{storeID}/orders/{orderID}/payments/{paymentID}/reject", storePerm(rbac.PermOrdersPayments, commerceH.RejectPayment))
	mux.Handle("GET /api/v1/stores/{storeID}/orders/{orderID}/payments", storePerm(rbac.PermOrdersPayments, commerceH.ListOrderPayments))
	mux.Handle("POST /api/v1/stores/{storeID}/orders/{orderID}/payments", storePerm(rbac.PermFinanceWrite, commerceH.RecordPayment))
	mux.Handle("POST /api/v1/stores/{storeID}/orders/{orderID}/payments/{paymentID}/verify-settlement", storePerm(rbac.PermFinanceWrite, commerceH.VerifyPaymentExtended))

	// Agent 07: finance reconciliation, refunds, store credit, deposits
	mux.Handle("GET /api/v1/stores/{storeID}/finance", storePerm(rbac.PermFinanceRead, commerceH.FinanceWorkspace))
	mux.Handle("GET /api/v1/stores/{storeID}/finance/orders/{orderID}", storePerm(rbac.PermFinanceRead, commerceH.OrderFinance))
	mux.Handle("POST /api/v1/stores/{storeID}/finance/adjustments", storePerm(rbac.PermFinanceWrite, commerceH.ApplyFinancialAdjustment))
	mux.Handle("POST /api/v1/stores/{storeID}/finance/refunds", storePerm(rbac.PermFinanceWrite, commerceH.CreateRefund))
	mux.Handle("POST /api/v1/stores/{storeID}/orders/{orderID}/refunds", storePerm(rbac.PermFinanceWrite, commerceH.CreateRefund))
	mux.Handle("POST /api/v1/stores/{storeID}/finance/refunds/{refundID}/process", storePerm(rbac.PermFinanceWrite, commerceH.ProcessRefund))
	mux.Handle("GET /api/v1/stores/{storeID}/finance/deposit-policies", storePerm(rbac.PermFinanceRead, commerceH.ListDepositPolicies))
	mux.Handle("POST /api/v1/stores/{storeID}/finance/deposit-policies", storePerm(rbac.PermFinanceWrite, commerceH.CreateDepositPolicy))
	mux.Handle("PATCH /api/v1/stores/{storeID}/finance/deposit-policies/{policyID}", storePerm(rbac.PermFinanceWrite, commerceH.UpdateDepositPolicy))
	mux.Handle("GET /api/v1/stores/{storeID}/finance/customers/{customerID}/credit", storePerm(rbac.PermFinanceRead, commerceH.GetStoreCredit))
	mux.Handle("POST /api/v1/stores/{storeID}/finance/customers/{customerID}/credit", storePerm(rbac.PermFinanceWrite, commerceH.GrantStoreCredit))
	mux.Handle("POST /api/v1/stores/{storeID}/finance/credit/apply", storePerm(rbac.PermFinanceWrite, commerceH.ApplyStoreCredit))
	mux.Handle("PATCH /api/v1/stores/{storeID}/finance/store-credit-settings", storePerm(rbac.PermStoresSettings, commerceH.PatchStoreCreditSettings))

	mux.Handle("GET /api/v1/stores/{storeID}/customers", storePerm(rbac.PermCustomersRead, commerceH.ListCustomers))
	mux.Handle("POST /api/v1/stores/{storeID}/customers", storePerm(rbac.PermCustomersWrite, p5.CreateCustomer))
	mux.Handle("GET /api/v1/stores/{storeID}/customers/{customerID}", storePerm(rbac.PermCustomersRead, p5.GetCustomer))
	mux.Handle("GET /api/v1/stores/{storeID}/customers/{customerID}/wishlist", storePerm(rbac.PermCustomersRead, commerceH.ListCustomerWishlist))
	mux.Handle("PATCH /api/v1/stores/{storeID}/customers/{customerID}", storePerm(rbac.PermCustomersWrite, p5.PatchCustomer))
	mux.Handle("POST /api/v1/stores/{storeID}/customers/{customerID}/portal-access/rotate", storePerm(rbac.PermCustomersWrite, p5.RotateCustomerPortalAccess))
	mux.Handle("GET /api/v1/stores/{storeID}/address-change-requests", storePerm(rbac.PermCustomersRead, portalH.MerchantListAddressChanges))
	mux.Handle("POST /api/v1/stores/{storeID}/address-change-requests/{requestID}/review", storePerm(rbac.PermCustomersWrite, portalH.MerchantReviewAddressChange))

	// Phase 5: notifications + analytics
	mux.Handle("GET /api/v1/stores/{storeID}/notifications", storePerm(rbac.PermNotificationsRead, p5.ListNotifications))
	mux.Handle("GET /api/v1/stores/{storeID}/notifications/unread-count", storePerm(rbac.PermNotificationsRead, p5.UnreadCount))
	mux.Handle("POST /api/v1/stores/{storeID}/notifications/read-all", storePerm(rbac.PermNotificationsWrite, p5.MarkAllNotificationsRead))
	mux.Handle("POST /api/v1/stores/{storeID}/notifications/{notificationID}/read", storePerm(rbac.PermNotificationsWrite, p5.MarkNotificationRead))
	mux.Handle("GET /api/v1/stores/{storeID}/notification-preferences", storePerm(rbac.PermNotificationsRead, p5.GetNotificationPreferences))
	mux.Handle("PATCH /api/v1/stores/{storeID}/notification-preferences", storePerm(rbac.PermNotificationsWrite, p5.PatchNotificationPreferences))
	mux.Handle("GET /api/v1/stores/{storeID}/analytics", storePerm(rbac.PermAnalyticsRead, p5.AnalyticsOverview))
	mux.Handle("POST /api/v1/stores/{storeID}/analytics/events", storePerm(rbac.PermAnalyticsRead, p5.RecordAnalyticsEvent))

	// Phase 4: preorder batches (kept; gated by campaign.* — still synced into campaigns)
	mux.Handle("GET /api/v1/stores/{storeID}/preorders", storePerm(rbac.PermCampaignRead, commerceH.ListBatches))
	mux.Handle("POST /api/v1/stores/{storeID}/preorders", storePerm(rbac.PermCampaignWrite, commerceH.CreateBatch))
	mux.Handle("GET /api/v1/stores/{storeID}/preorders/{batchID}", storePerm(rbac.PermCampaignRead, commerceH.GetBatch))
	mux.Handle("PATCH /api/v1/stores/{storeID}/preorders/{batchID}", storePerm(rbac.PermCampaignWrite, commerceH.PatchBatch))
	mux.Handle("GET /api/v1/stores/{storeID}/preorders/{batchID}/catalogs", storePerm(rbac.PermCampaignRead, catalogDocH.ListForPreorder))
	mux.Handle("POST /api/v1/stores/{storeID}/preorders/{batchID}/products", storePerm(rbac.PermCampaignWrite, commerceH.AssignBatchProduct))
	mux.Handle("DELETE /api/v1/stores/{storeID}/preorders/{batchID}/products/{productID}", storePerm(rbac.PermCampaignWrite, commerceH.RemoveBatchProduct))
	mux.Handle("GET /api/v1/stores/{storeID}/preorders/{batchID}/products/{productID}/variants", storePerm(rbac.PermCampaignRead, p5.ListBatchVariants))
	mux.Handle("POST /api/v1/stores/{storeID}/preorders/{batchID}/products/{productID}/variants", storePerm(rbac.PermCampaignWrite, p5.AssignBatchVariant))
	mux.Handle("DELETE /api/v1/stores/{storeID}/preorders/{batchID}/products/{productID}/variants/{variantID}", storePerm(rbac.PermCampaignWrite, p5.RemoveBatchVariant))
	mux.Handle("GET /api/v1/stores/{storeID}/preorders/{batchID}/purchasing", storePerm(rbac.PermPurchasingRead, commerceH.BatchPurchasing))
	mux.Handle("PUT /api/v1/stores/{storeID}/preorders/{batchID}/purchasing", storePerm(rbac.PermPurchasingWrite, commerceH.UpdateBatchPurchasing))
	mux.Handle("POST /api/v1/stores/{storeID}/preorders/{batchID}/allocations", storePerm(rbac.PermPurchasingWrite, commerceH.AllocateBatchItem))
	mux.Handle("GET /api/v1/stores/{storeID}/preorders/{batchID}/orders", storePerm(rbac.PermOrdersRead, commerceH.ListBatchOrders))

	// Agent 06: store-wide purchasing workspace, unavailable impact, receiving
	mux.Handle("GET /api/v1/stores/{storeID}/purchasing", storePerm(rbac.PermPurchasingRead, commerceH.StorePurchasing))
	mux.Handle("GET /api/v1/stores/{storeID}/purchasing/eligibility", storePerm(rbac.PermPurchasingRead, commerceH.PurchasingEligibility))
	mux.Handle("POST /api/v1/stores/{storeID}/purchasing/unavailable-impact", storePerm(rbac.PermPurchasingRead, commerceH.UnavailableImpact))
	mux.Handle("POST /api/v1/stores/{storeID}/orders/{orderID}/items/{itemID}/unavailable", storePerm(rbac.PermPurchasingWrite, commerceH.MarkOrderItemUnavailable))
	mux.Handle("GET /api/v1/stores/{storeID}/receiving", storePerm(rbac.PermPurchasingRead, commerceH.ListReceiving))
	mux.Handle("POST /api/v1/stores/{storeID}/receiving", storePerm(rbac.PermPurchasingWrite, commerceH.CreateReceiving))
	mux.Handle("GET /api/v1/stores/{storeID}/receiving/{receivingID}", storePerm(rbac.PermPurchasingRead, commerceH.GetReceiving))
	mux.Handle("PATCH /api/v1/stores/{storeID}/receiving/{receivingID}/lines/{lineID}", storePerm(rbac.PermPurchasingWrite, commerceH.UpdateReceivingLine))

	// Agent 09: packing + shipments
	mux.Handle("GET /api/v1/stores/{storeID}/packing", storePerm(rbac.PermPackingRead, commerceH.PackingQueue))
	mux.Handle("GET /api/v1/stores/{storeID}/packing/{orderID}", storePerm(rbac.PermPackingRead, commerceH.PackingChecklist))
	mux.Handle("PUT /api/v1/stores/{storeID}/packing/{orderID}/checklist", storePerm(rbac.PermPackingWrite, commerceH.UpdatePackingChecklist))
	mux.Handle("POST /api/v1/stores/{storeID}/packing/{orderID}/complete", storePerm(rbac.PermPackingWrite, commerceH.CompletePacking))
	mux.Handle("GET /api/v1/stores/{storeID}/shipments", storePerm(rbac.PermPackingRead, commerceH.ListShipments))
	mux.Handle("POST /api/v1/stores/{storeID}/shipments", storePerm(rbac.PermPackingWrite, commerceH.CreateShipment))
	mux.Handle("GET /api/v1/stores/{storeID}/shipments/{shipmentID}", storePerm(rbac.PermPackingRead, commerceH.GetShipment))
	mux.Handle("GET /api/v1/stores/{storeID}/orders/{orderID}/shipments", storePerm(rbac.PermPackingRead, commerceH.ListOrderShipments))

	// Agent 10: WhatsApp template registry + deep-links (preserve Brevo email_jobs)
	mux.Handle("GET /api/v1/stores/{storeID}/whatsapp/templates", storePerm(rbac.PermNotificationsRead, waH.ListTemplates))
	mux.Handle("GET /api/v1/stores/{storeID}/whatsapp/templates/{code}", storePerm(rbac.PermNotificationsRead, waH.GetTemplate))
	mux.Handle("POST /api/v1/stores/{storeID}/whatsapp/templates/{code}/preview", storePerm(rbac.PermNotificationsRead, waH.Preview))
	mux.Handle("POST /api/v1/stores/{storeID}/whatsapp/templates/{code}/open", storePerm(rbac.PermNotificationsRead, waH.Open))
	mux.Handle("GET /api/v1/stores/{storeID}/whatsapp/sends", storePerm(rbac.PermNotificationsRead, waH.ListSends))

	// Phase 4: business rules
	mux.Handle("GET /api/v1/stores/{storeID}/business-rules", storePerm(rbac.PermBusinessRulesRead, commerceH.ListRules))
	mux.Handle("POST /api/v1/stores/{storeID}/business-rules", storePerm(rbac.PermBusinessRulesWrite, commerceH.CreateRule))
	mux.Handle("PATCH /api/v1/stores/{storeID}/business-rules/{ruleID}", storePerm(rbac.PermBusinessRulesWrite, commerceH.UpdateRule))
	mux.Handle("DELETE /api/v1/stores/{storeID}/business-rules/{ruleID}", storePerm(rbac.PermBusinessRulesWrite, commerceH.DeleteRule))
	mux.Handle("POST /api/v1/stores/{storeID}/business-rules/preview", storePerm(rbac.PermBusinessRulesRead, commerceH.PreviewRules))

	// Ask Connie — MCP-shaped store-ops tools (auth + store membership required)
	mux.Handle("GET /api/v1/stores/{storeID}/connie/tools", storePerm(rbac.PermStoresRead, connieH.ListTools))
	mux.Handle("POST /api/v1/stores/{storeID}/connie/tools/call", storePerm(rbac.PermStoresRead, connieH.CallTool))
	mux.Handle("POST /api/v1/stores/{storeID}/connie/chat", storePerm(rbac.PermStoresRead, connieH.Chat))

	// Public storefront + assets (published only)
	mux.HandleFunc("GET /api/v1/public/stores/{slug}", p2.PublicStore)
	mux.HandleFunc("GET /api/v1/public/stores/{slug}/products", p2.PublicProducts)
	mux.HandleFunc("GET /api/v1/public/stores/{slug}/products/{productSlug}", p2.PublicProduct)
	mux.HandleFunc("GET /api/v1/public/stores/{slug}/categories", p2.PublicCategories)
	mux.HandleFunc("GET /api/v1/public/stores/{slug}/catalog", flipH.PublicCatalog)
	mux.HandleFunc("GET /api/v1/public/stores/{slug}/catalog/products/{productID}", flipH.PublicHotspotProduct)
	mux.HandleFunc("GET /api/v1/public/stores/{slug}/catalogs", catalogDocH.PublicCatalogs)
	mux.HandleFunc("GET /api/v1/public/stores/{slug}/catalogs/{catalogID}", flipH.PublicCatalogDoc)
	mux.HandleFunc("GET /api/v1/public/files/{storeID}/{fileID}", p2.PublicFile)
	mux.HandleFunc("GET /api/v1/public/stores/{slug}/preorders", commerceH.PublicBatches)
	mux.HandleFunc("GET /api/v1/public/stores/{slug}/preorders/{batchSlug}", commerceH.PublicBatch)
	mux.HandleFunc("GET /api/v1/public/stores/{slug}/products/{productSlug}/preorders", commerceH.PublicProductBatches)
	mux.HandleFunc("GET /api/v1/public/stores/{slug}/live", commerceH.PublicLiveCampaigns)
	mux.HandleFunc("GET /api/v1/public/stores/{slug}/live/{liveSlug}", commerceH.PublicLiveCampaign)
	mux.HandleFunc("POST /api/v1/public/stores/{slug}/live/{liveSlug}/wishlist", commerceH.PublicAddWishlist)
	mux.HandleFunc("GET /api/v1/public/stores/{slug}/live/{liveSlug}/wishlist", commerceH.PublicListWishlist)
	mux.HandleFunc("DELETE /api/v1/public/stores/{slug}/live/{liveSlug}/wishlist/{itemID}", commerceH.PublicRemoveWishlist)

	// Phase 3: public checkout + tracking
	mux.HandleFunc("POST /api/v1/public/stores/{slug}/checkout/preview", commerceH.CheckoutPreview)
	mux.HandleFunc("POST /api/v1/public/stores/{slug}/checkout/submit", commerceH.CheckoutSubmit)
	mux.HandleFunc("GET /api/v1/public/tracking/{token}", commerceH.TrackingGet)
	mux.HandleFunc("POST /api/v1/public/tracking/{token}/evidence", commerceH.TrackingEvidenceUpload)
	mux.HandleFunc("GET /api/v1/public/tracking/{token}/evidence/{evidenceID}/file", commerceH.TrackingEvidenceFile)

	// Agent 08: customer portal (separate auth from merchant; preserve guest tracking)
	// Access is via durable /{slug}/customer/{secret}-{phone}; email magic-link request is retired.
	mux.HandleFunc("POST /api/v1/portal/{storeSlug}/auth/access", portalH.AuthenticateAccessKey)
	mux.HandleFunc("POST /api/v1/portal/{storeSlug}/auth/magic-link/request", portalH.RequestMagicLink)
	mux.HandleFunc("GET /api/v1/portal/{storeSlug}/auth/magic-link/peek", portalH.PeekMagicLink)
	mux.HandleFunc("POST /api/v1/portal/{storeSlug}/auth/magic-link/verify", portalH.VerifyMagicLink)
	mux.HandleFunc("GET /api/v1/portal/{storeSlug}/auth/session", portalH.Session)
	mux.HandleFunc("POST /api/v1/portal/{storeSlug}/auth/logout", portalH.Logout)
	mux.Handle("GET /api/v1/portal/{storeSlug}/dashboard", portalH.Require(portalH.Dashboard))
	mux.Handle("GET /api/v1/portal/{storeSlug}/wishlist", portalH.Require(portalH.ListWishlist))
	mux.Handle("GET /api/v1/portal/{storeSlug}/orders", portalH.Require(portalH.ListOrders))
	mux.Handle("GET /api/v1/portal/{storeSlug}/orders/{orderID}", portalH.Require(portalH.GetOrder))
	mux.Handle("POST /api/v1/portal/{storeSlug}/orders/{orderID}/address-change", portalH.Require(portalH.RequestAddressChange))
	mux.Handle("GET /api/v1/portal/{storeSlug}/refund-preference", portalH.Require(portalH.GetRefundPreference))
	mux.Handle("PUT /api/v1/portal/{storeSlug}/refund-preference", portalH.Require(portalH.PutRefundPreference))
	mux.Handle("GET /api/v1/portal/{storeSlug}/credit", portalH.Require(portalH.GetCredit))
	mux.Handle("GET /api/v1/portal/{storeSlug}/addresses", portalH.Require(portalH.ListAddresses))
	mux.Handle("POST /api/v1/portal/{storeSlug}/addresses", portalH.Require(portalH.CreateAddress))
	mux.Handle("PATCH /api/v1/portal/{storeSlug}/addresses/{addressID}", portalH.Require(portalH.UpdateAddress))
	mux.Handle("DELETE /api/v1/portal/{storeSlug}/addresses/{addressID}", portalH.Require(portalH.DeleteAddress))
	mux.Handle("GET /api/v1/portal/{storeSlug}/address-change-requests", portalH.Require(portalH.ListAddressChanges))

	worker := &jobs.Worker{
		Queue: queue, Outbox: outbox, Email: mailSvc, BatchSync: commerceSvc,
		Logger: d.Logger, WorkerID: "api-" + uuidLike(),
		Types: []string{jobs.TypeCatalogPDFImport, jobs.TypePreorderBatchSync},
		Handlers: map[string]jobs.Handler{
			jobs.TypeCatalogPDFImport: importSvc.ProcessJob,
			jobs.TypePreorderBatchSync: func(ctx context.Context, job *jobs.Job) error {
				_, err := commerceSvc.SyncBatchStatuses(ctx, job.StoreID)
				return err
			},
		},
		Interval: 400 * time.Millisecond,
		OnFailure: func(ctx context.Context, job *jobs.Job, failErr error) {
			sev := syshealth.SeverityError
			msg := "job failed"
			if failErr != nil {
				msg = failErr.Error()
			}
			healthSvc.Record(ctx, syshealth.Event{
				Source: "worker", Severity: sev, Module: job.JobType, ErrorCode: "job_failed",
				Message: msg, StoreID: job.StoreID,
				Context: map[string]any{"job_id": job.ID.String(), "attempts": job.Attempts},
			})
		},
	}

	var handler http.Handler = mux
	handler = IPRateLimitMiddleware(d.DB, d.Cfg)(handler)
	handler = SecurityHeaders(d.Cfg)(handler)
	handler = CORSMiddleware(d.Cfg)(handler)
	handler = AccessLogMiddlewareWithHealth(d.Logger, healthSvc)(handler)
	handler = RequestIDMiddleware(handler)
	return handler, worker, RouterExtras{Health: healthSvc, Checker: healthChecker, Alerter: healthAlerter}
}

func uuidLike() string {
	return time.Now().UTC().Format("150405.000")
}
