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
	"github.com/rubicahq/pikko/services/api/internal/catalog"
	"github.com/rubicahq/pikko/services/api/internal/commerce"
	"github.com/rubicahq/pikko/services/api/internal/config"
	"github.com/rubicahq/pikko/services/api/internal/email"
	"github.com/rubicahq/pikko/services/api/internal/files"
	"github.com/rubicahq/pikko/services/api/internal/flipbook"
	"github.com/rubicahq/pikko/services/api/internal/imports"
	"github.com/rubicahq/pikko/services/api/internal/jobs"
	"github.com/rubicahq/pikko/services/api/internal/notifications"
	"github.com/rubicahq/pikko/services/api/internal/pdfutil"
	"github.com/rubicahq/pikko/services/api/internal/rbac"
	"github.com/rubicahq/pikko/services/api/internal/storage"
	"github.com/rubicahq/pikko/services/api/internal/storefront"
	"github.com/rubicahq/pikko/services/api/internal/stores"
	"github.com/rubicahq/pikko/services/api/internal/team"
	"github.com/rubicahq/pikko/services/api/internal/urls"
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

// NewRouter builds the HTTP handler and an optional background job worker.
func NewRouter(d Deps) (http.Handler, *jobs.Worker) {
	urlGen := urls.New(d.Cfg.MarketingURL, d.Cfg.DashboardURL, d.Cfg.StorefrontURL, d.Cfg.APIBaseURL)
	mailSender := email.NewSender(d.Cfg.EmailMode, d.Cfg.BrevoAPIKey, d.Logger)
	mailSvc := &email.Service{DB: d.DB, Sender: mailSender, Logger: d.Logger, DefaultFromName: d.Cfg.EmailFromName}
	authSvc := &auth.Service{DB: d.DB, Cfg: d.Cfg, Email: mailSvc, URLs: urlGen, Logger: d.Logger}
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
	importSvc := &imports.Service{
		DB: d.DB, Files: filesSvc, Catalog: catalogSvc, Queue: queue, Outbox: outbox,
		AI: extractor, Renderer: renderer, Storage: d.Storage, Logger: d.Logger,
		Limits: imports.Limits{
			MaxPDFBytes: d.Cfg.ImportMaxPDFBytes,
			MaxPages:    d.Cfg.ImportMaxPages,
			MaxItems:    d.Cfg.ImportMaxItems,
		},
	}
	if importSvc.Limits.MaxPDFBytes == 0 {
		importSvc.Limits = imports.DefaultLimits()
	}

	authH := &AuthHandlers{Auth: authSvc, Cfg: d.Cfg}
	accountH := &AccountHandlers{Account: accountSvc}
	storeH := &StoreHandlers{Stores: storeSvc, URLs: urlGen, Cfg: d.Cfg}
	teamSvc := &team.Service{DB: d.DB, Cfg: d.Cfg, Email: mailSvc, URLs: urlGen, Logger: d.Logger}
	teamH := &TeamHandlers{Team: teamSvc, Cfg: d.Cfg}
	p2 := &Phase2Handlers{Storefront: storefrontSvc, Catalog: catalogSvc, Files: filesSvc, URLs: urlGen}
	impH := &ImportHandlers{Imports: importSvc}
	flipSvc := &flipbook.Service{DB: d.DB}
	flipH := &FlipbookHandlers{Flipbook: flipSvc, Catalog: catalogSvc, URLs: urlGen}
	notifSvc := &notifications.Service{DB: d.DB}
	analyticsSvc := &analytics.Service{DB: d.DB}
	commerceSvc := &commerce.Service{
		DB: d.DB, Cfg: d.Cfg, Files: filesSvc, Storage: d.Storage,
		Email: mailSvc, Outbox: outbox, URLs: urlGen, Logger: d.Logger,
		Notifier: notifSvc,
	}
	commerceH := &CommerceHandlers{Commerce: commerceSvc, URLs: urlGen, Cfg: d.Cfg}
	p5 := &Phase5Handlers{Notifications: notifSvc, Analytics: analyticsSvc, Commerce: commerceSvc}

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
	mux.Handle("POST /api/v1/onboarding/complete", authMW.Require(http.HandlerFunc(storeH.CompleteOnboarding)))

	// Explicit store-scoped routes with membership + optional permission checks
	storeAuthed := func(h http.HandlerFunc) http.Handler {
		return authMW.Require(storeMW.Require(h))
	}
	storePerm := func(perm rbac.Permission, h http.HandlerFunc) http.Handler {
		return authMW.Require(storeMW.RequirePermission(perm)(h))
	}

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

	// Catalog import PDF + CSV (PRD §18)
	mux.Handle("POST /api/v1/stores/{storeID}/catalog-imports", storePerm(rbac.PermCatalogWrite, impH.Start))
	mux.Handle("GET /api/v1/stores/{storeID}/catalog-imports", storePerm(rbac.PermCatalogRead, impH.List))
	mux.Handle("GET /api/v1/stores/{storeID}/catalog-imports/csv-template", storePerm(rbac.PermCatalogRead, impH.CSVTemplate))
	mux.Handle("GET /api/v1/stores/{storeID}/catalog-imports/{jobID}", storePerm(rbac.PermCatalogRead, impH.Get))
	mux.Handle("GET /api/v1/stores/{storeID}/catalog-imports/{jobID}/items", storePerm(rbac.PermCatalogRead, impH.ListItems))
	mux.Handle("PATCH /api/v1/stores/{storeID}/catalog-imports/{jobID}/items/{itemID}", storePerm(rbac.PermCatalogWrite, impH.PatchItem))
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

	// Phase 3: merchant commerce settings + orders
	mux.Handle("GET /api/v1/stores/{storeID}/payment-methods", storePerm(rbac.PermStoresSettings, commerceH.ListPaymentMethods))
	mux.Handle("POST /api/v1/stores/{storeID}/payment-methods", storePerm(rbac.PermStoresSettings, commerceH.CreatePaymentMethod))
	mux.Handle("PATCH /api/v1/stores/{storeID}/payment-methods/{methodID}", storePerm(rbac.PermStoresSettings, commerceH.UpdatePaymentMethod))
	mux.Handle("DELETE /api/v1/stores/{storeID}/payment-methods/{methodID}", storePerm(rbac.PermStoresSettings, commerceH.DeletePaymentMethod))
	mux.Handle("GET /api/v1/stores/{storeID}/fulfillment", storePerm(rbac.PermStoresSettings, commerceH.GetFulfillment))
	mux.Handle("PATCH /api/v1/stores/{storeID}/fulfillment", storePerm(rbac.PermStoresSettings, commerceH.PatchFulfillment))
	mux.Handle("GET /api/v1/stores/{storeID}/orders", storePerm(rbac.PermOrdersRead, commerceH.ListOrders))
	mux.Handle("GET /api/v1/stores/{storeID}/orders/{orderID}", storePerm(rbac.PermOrdersRead, commerceH.GetOrder))
	mux.Handle("PATCH /api/v1/stores/{storeID}/orders/{orderID}", storePerm(rbac.PermOrdersWrite, commerceH.PatchOrder))
	mux.Handle("POST /api/v1/stores/{storeID}/orders/{orderID}/payments/{paymentID}/verify", storePerm(rbac.PermOrdersPayments, commerceH.VerifyPayment))
	mux.Handle("POST /api/v1/stores/{storeID}/orders/{orderID}/payments/{paymentID}/reject", storePerm(rbac.PermOrdersPayments, commerceH.RejectPayment))
	mux.Handle("GET /api/v1/stores/{storeID}/orders/evidence/{evidenceID}/file", storePerm(rbac.PermOrdersRead, commerceH.MerchantEvidenceFile))
	mux.Handle("GET /api/v1/stores/{storeID}/customers", storePerm(rbac.PermCustomersRead, commerceH.ListCustomers))
	mux.Handle("GET /api/v1/stores/{storeID}/customers/{customerID}", storePerm(rbac.PermCustomersRead, p5.GetCustomer))
	mux.Handle("PATCH /api/v1/stores/{storeID}/customers/{customerID}", storePerm(rbac.PermCustomersWrite, p5.PatchCustomer))

	// Phase 5: notifications + analytics
	mux.Handle("GET /api/v1/stores/{storeID}/notifications", storePerm(rbac.PermNotificationsRead, p5.ListNotifications))
	mux.Handle("GET /api/v1/stores/{storeID}/notifications/unread-count", storePerm(rbac.PermNotificationsRead, p5.UnreadCount))
	mux.Handle("POST /api/v1/stores/{storeID}/notifications/read-all", storePerm(rbac.PermNotificationsWrite, p5.MarkAllNotificationsRead))
	mux.Handle("POST /api/v1/stores/{storeID}/notifications/{notificationID}/read", storePerm(rbac.PermNotificationsWrite, p5.MarkNotificationRead))
	mux.Handle("GET /api/v1/stores/{storeID}/notification-preferences", storePerm(rbac.PermNotificationsRead, p5.GetNotificationPreferences))
	mux.Handle("PATCH /api/v1/stores/{storeID}/notification-preferences", storePerm(rbac.PermNotificationsWrite, p5.PatchNotificationPreferences))
	mux.Handle("GET /api/v1/stores/{storeID}/analytics", storePerm(rbac.PermAnalyticsRead, p5.AnalyticsOverview))

	// Phase 4: preorder batches
	mux.Handle("GET /api/v1/stores/{storeID}/preorders", storePerm(rbac.PermCatalogRead, commerceH.ListBatches))
	mux.Handle("POST /api/v1/stores/{storeID}/preorders", storePerm(rbac.PermCatalogWrite, commerceH.CreateBatch))
	mux.Handle("GET /api/v1/stores/{storeID}/preorders/{batchID}", storePerm(rbac.PermCatalogRead, commerceH.GetBatch))
	mux.Handle("PATCH /api/v1/stores/{storeID}/preorders/{batchID}", storePerm(rbac.PermCatalogWrite, commerceH.PatchBatch))
	mux.Handle("POST /api/v1/stores/{storeID}/preorders/{batchID}/products", storePerm(rbac.PermCatalogWrite, commerceH.AssignBatchProduct))
	mux.Handle("DELETE /api/v1/stores/{storeID}/preorders/{batchID}/products/{productID}", storePerm(rbac.PermCatalogWrite, commerceH.RemoveBatchProduct))
	mux.Handle("GET /api/v1/stores/{storeID}/preorders/{batchID}/products/{productID}/variants", storePerm(rbac.PermCatalogRead, p5.ListBatchVariants))
	mux.Handle("POST /api/v1/stores/{storeID}/preorders/{batchID}/products/{productID}/variants", storePerm(rbac.PermCatalogWrite, p5.AssignBatchVariant))
	mux.Handle("DELETE /api/v1/stores/{storeID}/preorders/{batchID}/products/{productID}/variants/{variantID}", storePerm(rbac.PermCatalogWrite, p5.RemoveBatchVariant))
	mux.Handle("GET /api/v1/stores/{storeID}/preorders/{batchID}/purchasing", storePerm(rbac.PermPurchasingRead, commerceH.BatchPurchasing))
	mux.Handle("PUT /api/v1/stores/{storeID}/preorders/{batchID}/purchasing", storePerm(rbac.PermPurchasingWrite, commerceH.UpdateBatchPurchasing))
	mux.Handle("POST /api/v1/stores/{storeID}/preorders/{batchID}/allocations", storePerm(rbac.PermPurchasingWrite, commerceH.AllocateBatchItem))
	mux.Handle("GET /api/v1/stores/{storeID}/preorders/{batchID}/orders", storePerm(rbac.PermOrdersRead, commerceH.ListBatchOrders))

	// Phase 4: business rules
	mux.Handle("GET /api/v1/stores/{storeID}/business-rules", storePerm(rbac.PermBusinessRulesRead, commerceH.ListRules))
	mux.Handle("POST /api/v1/stores/{storeID}/business-rules", storePerm(rbac.PermBusinessRulesWrite, commerceH.CreateRule))
	mux.Handle("PATCH /api/v1/stores/{storeID}/business-rules/{ruleID}", storePerm(rbac.PermBusinessRulesWrite, commerceH.UpdateRule))
	mux.Handle("DELETE /api/v1/stores/{storeID}/business-rules/{ruleID}", storePerm(rbac.PermBusinessRulesWrite, commerceH.DeleteRule))
	mux.Handle("POST /api/v1/stores/{storeID}/business-rules/preview", storePerm(rbac.PermBusinessRulesRead, commerceH.PreviewRules))

	// Public storefront + assets (published only)
	mux.HandleFunc("GET /api/v1/public/stores/{slug}", p2.PublicStore)
	mux.HandleFunc("GET /api/v1/public/stores/{slug}/products", p2.PublicProducts)
	mux.HandleFunc("GET /api/v1/public/stores/{slug}/products/{productSlug}", p2.PublicProduct)
	mux.HandleFunc("GET /api/v1/public/stores/{slug}/categories", p2.PublicCategories)
	mux.HandleFunc("GET /api/v1/public/stores/{slug}/catalog", flipH.PublicCatalog)
	mux.HandleFunc("GET /api/v1/public/stores/{slug}/catalog/products/{productID}", flipH.PublicHotspotProduct)
	mux.HandleFunc("GET /api/v1/public/files/{storeID}/{fileID}", p2.PublicFile)
	mux.HandleFunc("GET /api/v1/public/stores/{slug}/preorders", commerceH.PublicBatches)
	mux.HandleFunc("GET /api/v1/public/stores/{slug}/preorders/{batchSlug}", commerceH.PublicBatch)
	mux.HandleFunc("GET /api/v1/public/stores/{slug}/products/{productSlug}/preorders", commerceH.PublicProductBatches)

	// Phase 3: public checkout + tracking
	mux.HandleFunc("POST /api/v1/public/stores/{slug}/checkout/preview", commerceH.CheckoutPreview)
	mux.HandleFunc("POST /api/v1/public/stores/{slug}/checkout/submit", commerceH.CheckoutSubmit)
	mux.HandleFunc("GET /api/v1/public/tracking/{token}", commerceH.TrackingGet)
	mux.HandleFunc("POST /api/v1/public/tracking/{token}/evidence", commerceH.TrackingEvidenceUpload)
	mux.HandleFunc("GET /api/v1/public/tracking/{token}/evidence/{evidenceID}/file", commerceH.TrackingEvidenceFile)

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
	}

	var handler http.Handler = mux
	handler = IPRateLimitMiddleware(d.DB, d.Cfg)(handler)
	handler = SecurityHeaders(d.Cfg)(handler)
	handler = CORSMiddleware(d.Cfg)(handler)
	handler = AccessLogMiddleware(d.Logger)(handler)
	handler = RequestIDMiddleware(handler)
	return handler, worker
}

func uuidLike() string {
	return time.Now().UTC().Format("150405.000")
}
