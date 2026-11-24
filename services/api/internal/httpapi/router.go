package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rubicahq/pikko/services/api/internal/account"
	"github.com/rubicahq/pikko/services/api/internal/ai"
	"github.com/rubicahq/pikko/services/api/internal/auth"
	"github.com/rubicahq/pikko/services/api/internal/catalog"
	"github.com/rubicahq/pikko/services/api/internal/config"
	"github.com/rubicahq/pikko/services/api/internal/email"
	"github.com/rubicahq/pikko/services/api/internal/files"
	"github.com/rubicahq/pikko/services/api/internal/imports"
	"github.com/rubicahq/pikko/services/api/internal/jobs"
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
	mailSvc := &email.Service{DB: d.DB, Sender: mailSender, Logger: d.Logger}
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

	// AI catalog PDF import (PRD §18)
	mux.Handle("POST /api/v1/stores/{storeID}/catalog-imports", storeAuthed(impH.Start))
	mux.Handle("GET /api/v1/stores/{storeID}/catalog-imports", storeAuthed(impH.List))
	mux.Handle("GET /api/v1/stores/{storeID}/catalog-imports/{jobID}", storeAuthed(impH.Get))
	mux.Handle("GET /api/v1/stores/{storeID}/catalog-imports/{jobID}/items", storeAuthed(impH.ListItems))
	mux.Handle("PATCH /api/v1/stores/{storeID}/catalog-imports/{jobID}/items/{itemID}", storeAuthed(impH.PatchItem))
	mux.Handle("DELETE /api/v1/stores/{storeID}/catalog-imports/{jobID}/items/{itemID}", storeAuthed(impH.DeleteItem))
	mux.Handle("POST /api/v1/stores/{storeID}/catalog-imports/{jobID}/confirm", storeAuthed(impH.Confirm))
	mux.Handle("POST /api/v1/stores/{storeID}/catalog-imports/{jobID}/retry", storeAuthed(impH.Retry))

	// Public storefront + assets (published only)
	mux.HandleFunc("GET /api/v1/public/stores/{slug}", p2.PublicStore)
	mux.HandleFunc("GET /api/v1/public/stores/{slug}/products", p2.PublicProducts)
	mux.HandleFunc("GET /api/v1/public/stores/{slug}/products/{productSlug}", p2.PublicProduct)
	mux.HandleFunc("GET /api/v1/public/stores/{slug}/categories", p2.PublicCategories)
	mux.HandleFunc("GET /api/v1/public/files/{storeID}/{fileID}", p2.PublicFile)

	worker := &jobs.Worker{
		Queue: queue, Outbox: outbox, Logger: d.Logger, WorkerID: "api-" + uuidLike(),
		Types: []string{jobs.TypeCatalogPDFImport},
		Handlers: map[string]jobs.Handler{
			jobs.TypeCatalogPDFImport: importSvc.ProcessJob,
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
