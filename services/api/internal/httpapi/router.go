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
	"github.com/rubicahq/pikko/services/api/internal/storage"
	"github.com/rubicahq/pikko/services/api/internal/storefront"
	"github.com/rubicahq/pikko/services/api/internal/stores"
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

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if err := d.DB.Ping(ctx); err != nil {
			writeError(w, r, http.StatusServiceUnavailable, "unhealthy", "Database unavailable.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "time": time.Now().UTC()})
	})

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

	// Explicit store-scoped routes with authorization middleware
	storeAuthed := func(h http.HandlerFunc) http.Handler {
		return authMW.Require(storeMW.Require(h))
	}
	mux.Handle("GET /api/v1/stores/{storeID}", storeAuthed(storeH.Get))
	mux.Handle("PATCH /api/v1/stores/{storeID}", storeAuthed(storeH.Patch))
	mux.Handle("POST /api/v1/stores/{storeID}/archive", storeAuthed(storeH.Archive))

	// Phase 2: storefront, branding files, catalog
	mux.Handle("GET /api/v1/meta/themes", authMW.Require(http.HandlerFunc(p2.ListThemes)))
	mux.Handle("GET /api/v1/stores/{storeID}/storefront", storeAuthed(p2.GetStorefront))
	mux.Handle("PATCH /api/v1/stores/{storeID}/storefront", storeAuthed(p2.PatchStorefront))
	mux.Handle("POST /api/v1/stores/{storeID}/storefront/publish", storeAuthed(p2.PublishStorefront))
	mux.Handle("POST /api/v1/stores/{storeID}/storefront/discard", storeAuthed(p2.DiscardStorefront))
	mux.Handle("PUT /api/v1/stores/{storeID}/storefront/featured", storeAuthed(p2.SetFeatured))
	mux.Handle("GET /api/v1/stores/{storeID}/setup-checklist", storeAuthed(p2.SetupChecklist))

	mux.Handle("POST /api/v1/stores/{storeID}/files", storeAuthed(p2.UploadFile))
	mux.Handle("GET /api/v1/stores/{storeID}/files/{fileID}", storeAuthed(p2.GetFile))

	mux.Handle("GET /api/v1/stores/{storeID}/categories", storeAuthed(p2.ListCategories))
	mux.Handle("POST /api/v1/stores/{storeID}/categories", storeAuthed(p2.CreateCategory))
	mux.Handle("PATCH /api/v1/stores/{storeID}/categories/{categoryID}", storeAuthed(p2.UpdateCategory))
	mux.Handle("DELETE /api/v1/stores/{storeID}/categories/{categoryID}", storeAuthed(p2.DeleteCategory))

	mux.Handle("GET /api/v1/stores/{storeID}/products", storeAuthed(p2.ListProducts))
	mux.Handle("POST /api/v1/stores/{storeID}/products", storeAuthed(p2.CreateProduct))
	mux.Handle("GET /api/v1/stores/{storeID}/products/{productID}", storeAuthed(p2.GetProduct))
	mux.Handle("PATCH /api/v1/stores/{storeID}/products/{productID}", storeAuthed(p2.UpdateProduct))
	mux.Handle("POST /api/v1/stores/{storeID}/products/{productID}/status", storeAuthed(p2.PatchProductStatus))
	mux.Handle("POST /api/v1/stores/{storeID}/products/{productID}/archive", storeAuthed(p2.ArchiveProduct))

	mux.Handle("GET /api/v1/stores/{storeID}/collections", storeAuthed(p2.ListCollections))
	mux.Handle("POST /api/v1/stores/{storeID}/collections", storeAuthed(p2.CreateCollection))

	// Catalog import PDF + CSV (PRD §18)
	mux.Handle("POST /api/v1/stores/{storeID}/catalog-imports", storeAuthed(impH.Start))
	mux.Handle("GET /api/v1/stores/{storeID}/catalog-imports", storeAuthed(impH.List))
	mux.Handle("GET /api/v1/stores/{storeID}/catalog-imports/csv-template", storeAuthed(impH.CSVTemplate))
	mux.Handle("GET /api/v1/stores/{storeID}/catalog-imports/{jobID}", storeAuthed(impH.Get))
	mux.Handle("GET /api/v1/stores/{storeID}/catalog-imports/{jobID}/items", storeAuthed(impH.ListItems))
	mux.Handle("PATCH /api/v1/stores/{storeID}/catalog-imports/{jobID}/items/{itemID}", storeAuthed(impH.PatchItem))
	mux.Handle("DELETE /api/v1/stores/{storeID}/catalog-imports/{jobID}/items/{itemID}", storeAuthed(impH.DeleteItem))
	mux.Handle("POST /api/v1/stores/{storeID}/catalog-imports/{jobID}/confirm", storeAuthed(impH.Confirm))
	mux.Handle("POST /api/v1/stores/{storeID}/catalog-imports/{jobID}/retry", storeAuthed(impH.Retry))

	// Flipbook catalog pages / hotspots
	mux.Handle("GET /api/v1/stores/{storeID}/catalog-pages", storeAuthed(flipH.ListPages))
	mux.Handle("PUT /api/v1/stores/{storeID}/catalog-pages/order", storeAuthed(flipH.Reorder))
	mux.Handle("POST /api/v1/stores/{storeID}/catalog-pages/status", storeAuthed(flipH.Publish))
	mux.Handle("POST /api/v1/stores/{storeID}/catalog-pages/{pageID}/hotspots", storeAuthed(flipH.CreateHotspot))
	mux.Handle("PATCH /api/v1/stores/{storeID}/catalog-pages/hotspots/{hotspotID}", storeAuthed(flipH.UpdateHotspot))
	mux.Handle("DELETE /api/v1/stores/{storeID}/catalog-pages/hotspots/{hotspotID}", storeAuthed(flipH.DeleteHotspot))

	// Phase 3: merchant commerce settings + orders
	mux.Handle("GET /api/v1/stores/{storeID}/payment-methods", storeAuthed(commerceH.ListPaymentMethods))
	mux.Handle("POST /api/v1/stores/{storeID}/payment-methods", storeAuthed(commerceH.CreatePaymentMethod))
	mux.Handle("PATCH /api/v1/stores/{storeID}/payment-methods/{methodID}", storeAuthed(commerceH.UpdatePaymentMethod))
	mux.Handle("DELETE /api/v1/stores/{storeID}/payment-methods/{methodID}", storeAuthed(commerceH.DeletePaymentMethod))
	mux.Handle("GET /api/v1/stores/{storeID}/fulfillment", storeAuthed(commerceH.GetFulfillment))
	mux.Handle("PATCH /api/v1/stores/{storeID}/fulfillment", storeAuthed(commerceH.PatchFulfillment))
	mux.Handle("GET /api/v1/stores/{storeID}/orders", storeAuthed(commerceH.ListOrders))
	mux.Handle("GET /api/v1/stores/{storeID}/orders/{orderID}", storeAuthed(commerceH.GetOrder))
	mux.Handle("PATCH /api/v1/stores/{storeID}/orders/{orderID}", storeAuthed(commerceH.PatchOrder))
	mux.Handle("POST /api/v1/stores/{storeID}/orders/{orderID}/payments/{paymentID}/verify", storeAuthed(commerceH.VerifyPayment))
	mux.Handle("POST /api/v1/stores/{storeID}/orders/{orderID}/payments/{paymentID}/reject", storeAuthed(commerceH.RejectPayment))
	mux.Handle("GET /api/v1/stores/{storeID}/orders/evidence/{evidenceID}/file", storeAuthed(commerceH.MerchantEvidenceFile))
	mux.Handle("GET /api/v1/stores/{storeID}/customers", storeAuthed(commerceH.ListCustomers))
	mux.Handle("GET /api/v1/stores/{storeID}/customers/{customerID}", storeAuthed(p5.GetCustomer))
	mux.Handle("PATCH /api/v1/stores/{storeID}/customers/{customerID}", storeAuthed(p5.PatchCustomer))

	// Phase 5: notifications + analytics
	mux.Handle("GET /api/v1/stores/{storeID}/notifications", storeAuthed(p5.ListNotifications))
	mux.Handle("GET /api/v1/stores/{storeID}/notifications/unread-count", storeAuthed(p5.UnreadCount))
	mux.Handle("POST /api/v1/stores/{storeID}/notifications/read-all", storeAuthed(p5.MarkAllNotificationsRead))
	mux.Handle("POST /api/v1/stores/{storeID}/notifications/{notificationID}/read", storeAuthed(p5.MarkNotificationRead))
	mux.Handle("GET /api/v1/stores/{storeID}/notification-preferences", storeAuthed(p5.GetNotificationPreferences))
	mux.Handle("PATCH /api/v1/stores/{storeID}/notification-preferences", storeAuthed(p5.PatchNotificationPreferences))
	mux.Handle("GET /api/v1/stores/{storeID}/analytics", storeAuthed(p5.AnalyticsOverview))

	// Phase 4: preorder batches
	mux.Handle("GET /api/v1/stores/{storeID}/preorders", storeAuthed(commerceH.ListBatches))
	mux.Handle("POST /api/v1/stores/{storeID}/preorders", storeAuthed(commerceH.CreateBatch))
	mux.Handle("GET /api/v1/stores/{storeID}/preorders/{batchID}", storeAuthed(commerceH.GetBatch))
	mux.Handle("PATCH /api/v1/stores/{storeID}/preorders/{batchID}", storeAuthed(commerceH.PatchBatch))
	mux.Handle("POST /api/v1/stores/{storeID}/preorders/{batchID}/products", storeAuthed(commerceH.AssignBatchProduct))
	mux.Handle("DELETE /api/v1/stores/{storeID}/preorders/{batchID}/products/{productID}", storeAuthed(commerceH.RemoveBatchProduct))
	mux.Handle("GET /api/v1/stores/{storeID}/preorders/{batchID}/products/{productID}/variants", storeAuthed(p5.ListBatchVariants))
	mux.Handle("POST /api/v1/stores/{storeID}/preorders/{batchID}/products/{productID}/variants", storeAuthed(p5.AssignBatchVariant))
	mux.Handle("DELETE /api/v1/stores/{storeID}/preorders/{batchID}/products/{productID}/variants/{variantID}", storeAuthed(p5.RemoveBatchVariant))
	mux.Handle("GET /api/v1/stores/{storeID}/preorders/{batchID}/purchasing", storeAuthed(commerceH.BatchPurchasing))
	mux.Handle("PUT /api/v1/stores/{storeID}/preorders/{batchID}/purchasing", storeAuthed(commerceH.UpdateBatchPurchasing))
	mux.Handle("POST /api/v1/stores/{storeID}/preorders/{batchID}/allocations", storeAuthed(commerceH.AllocateBatchItem))
	mux.Handle("GET /api/v1/stores/{storeID}/preorders/{batchID}/orders", storeAuthed(commerceH.ListBatchOrders))

	// Phase 4: business rules
	mux.Handle("GET /api/v1/stores/{storeID}/business-rules", storeAuthed(commerceH.ListRules))
	mux.Handle("POST /api/v1/stores/{storeID}/business-rules", storeAuthed(commerceH.CreateRule))
	mux.Handle("PATCH /api/v1/stores/{storeID}/business-rules/{ruleID}", storeAuthed(commerceH.UpdateRule))
	mux.Handle("DELETE /api/v1/stores/{storeID}/business-rules/{ruleID}", storeAuthed(commerceH.DeleteRule))
	mux.Handle("POST /api/v1/stores/{storeID}/business-rules/preview", storeAuthed(commerceH.PreviewRules))

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
	handler = SecurityHeaders(handler)
	handler = CORSMiddleware(d.Cfg)(handler)
	handler = RequestIDMiddleware(handler)
	return handler, worker
}

func uuidLike() string {
	return time.Now().UTC().Format("150405.000")
}
