package imports_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rubicahq/pikko/services/api/internal/ai"
	"github.com/rubicahq/pikko/services/api/internal/auth"
	"github.com/rubicahq/pikko/services/api/internal/config"
	"github.com/rubicahq/pikko/services/api/internal/email"
	"github.com/rubicahq/pikko/services/api/internal/httpapi"
	"github.com/rubicahq/pikko/services/api/internal/jobs"
	"github.com/rubicahq/pikko/services/api/internal/pdfutil"
	"github.com/rubicahq/pikko/services/api/internal/storage"
	"github.com/rubicahq/pikko/services/api/internal/stores"
	"github.com/rubicahq/pikko/services/api/internal/urls"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = "postgres://pikko:pikko@localhost:5432/pikko_test?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Skip(err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		t.Skip(err)
	}
	return pool
}

func reset(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		TRUNCATE audit_logs, email_jobs, auth_rate_limits, onboarding_drafts,
		         outbox_events, job_queue,
		         catalog_page_hotspots, catalog_pages, catalog_import_items, catalog_import_jobs,
		         storefront_featured_collections, storefront_featured_products,
		         storefront_banners, storefront_configs,
		         collection_products, collections,
		         variant_option_values, product_variants, product_option_values, product_options,
		         product_images, products, categories, stored_files,
		         store_slug_aliases, store_memberships, stores, user_preferences,
		         sessions, authentication_tokens, users CASCADE
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func cfg() *config.Config {
	return &config.Config{
		AppEnv: "test", EmailMode: "log", EmailFromName: "Pikko",
		EmailAccountFrom: "account@pikko.site", EmailReplyTo: "support@pikko.site",
		MagicLinkTTL: 15 * time.Minute, ResendCooldown: time.Second,
		SessionTTL: 24 * time.Hour, RateLimitEmailHour: 100, RateLimitIPHour: 200,
		DefaultLocale: "id-ID", DefaultTZ: "Asia/Jakarta", DefaultCurr: "IDR",
		CookieName: "pikko_session", CORSOrigins: []string{"http://localhost:3000"},
		MarketingURL: "http://localhost:3000", DashboardURL: "http://localhost:3000",
		StorefrontURL: "http://localhost:3000", APIBaseURL: "http://localhost:8080",
		ImportMaxPDFBytes: 25 << 20, ImportMaxPages: 50, ImportMaxItems: 500,
	}
}

func signIn(t *testing.T, pool *pgxpool.Pool, emailAddr string) (uuid.UUID, string) {
	t.Helper()
	c := cfg()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mail := &email.Service{DB: pool, Sender: &email.LogSender{Logger: logger}, Logger: logger}
	svc := &auth.Service{DB: pool, Cfg: c, Email: mail, URLs: urls.New(c.MarketingURL, c.DashboardURL, c.StorefrontURL, c.APIBaseURL), Logger: logger}
	res, err := svc.RequestMagicLink(context.Background(), emailAddr, "127.0.0.1", "test")
	if err != nil {
		t.Fatal(err)
	}
	v, err := svc.VerifyAndConsume(context.Background(), res.DevToken, "127.0.0.1", "test")
	if err != nil {
		t.Fatal(err)
	}
	return v.User.ID, v.SessionToken
}

func createStore(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID, name, slug string) uuid.UUID {
	t.Helper()
	svc := &stores.Service{DB: pool, Cfg: cfg()}
	st, err := svc.CreateStore(context.Background(), userID, stores.CreateDirectInput{
		Name: name, Slug: slug, Category: "other", SellingModel: "ready_stock",
	})
	if err != nil {
		t.Fatal(err)
	}
	return st.ID
}

func TestPDFImportFlowAndCrossStoreIsolation(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	reset(t, pool)

	dir := t.TempDir()
	store, err := storage.NewLocal(dir)
	if err != nil {
		t.Fatal(err)
	}
	c := cfg()
	router, worker := httpapi.NewRouter(httpapi.Deps{
		Cfg: c, DB: pool, Logger: slog.Default(), Storage: store,
		AI: ai.NewMockExtractor(), Renderer: &pdfutil.PlaceholderRenderer{Width: 80, Height: 100},
	})

	uidA, tokA := signIn(t, pool, "import-a@example.com")
	uidB, tokB := signIn(t, pool, "import-b@example.com")
	storeA := createStore(t, pool, uidA, "Store A", "store-a-import")
	storeB := createStore(t, pool, uidB, "Store B", "store-b-import")

	pdf := pdfutil.MinimalPDF()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("extract_images", "true")
	fw, err := mw.CreateFormFile("file", "catalog.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(pdf); err != nil {
		t.Fatal(err)
	}
	_ = mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/stores/"+storeA.String()+"/catalog-imports", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+tokA)
	req.Header.Set("Idempotency-Key", "test-import-1")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("start status %d body %s", rr.Code, rr.Body.String())
	}
	var startResp struct {
		Job struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"job"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &startResp); err != nil {
		t.Fatal(err)
	}
	jobID := startResp.Job.ID

	// Idempotent replay
	var buf2 bytes.Buffer
	mw2 := multipart.NewWriter(&buf2)
	_ = mw2.WriteField("extract_images", "true")
	fw2, _ := mw2.CreateFormFile("file", "catalog.pdf")
	_, _ = fw2.Write(pdf)
	_ = mw2.Close()
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/stores/"+storeA.String()+"/catalog-imports", &buf2)
	req2.Header.Set("Content-Type", mw2.FormDataContentType())
	req2.Header.Set("Authorization", "Bearer "+tokA)
	req2.Header.Set("Idempotency-Key", "test-import-1")
	rr2 := httptest.NewRecorder()
	router.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusAccepted {
		t.Fatalf("idempotent start %d", rr2.Code)
	}

	// Process queue
	for i := 0; i < 5; i++ {
		ok, err := worker.ProcessOnce(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	reqG := httptest.NewRequest(http.MethodGet, "/api/v1/stores/"+storeA.String()+"/catalog-imports/"+jobID, nil)
	reqG.Header.Set("Authorization", "Bearer "+tokA)
	rrG := httptest.NewRecorder()
	router.ServeHTTP(rrG, reqG)
	if rrG.Code != 200 {
		t.Fatalf("get job %d %s", rrG.Code, rrG.Body.String())
	}
	var getResp struct {
		Job struct {
			Status    string `json:"status"`
			ItemCount int    `json:"item_count"`
		} `json:"job"`
	}
	_ = json.Unmarshal(rrG.Body.Bytes(), &getResp)
	if getResp.Job.Status != "review_required" {
		t.Fatalf("status=%s want review_required body=%s", getResp.Job.Status, rrG.Body.String())
	}
	if getResp.Job.ItemCount < 1 {
		t.Fatal("expected items")
	}

	reqItems := httptest.NewRequest(http.MethodGet, "/api/v1/stores/"+storeA.String()+"/catalog-imports/"+jobID+"/items", nil)
	reqItems.Header.Set("Authorization", "Bearer "+tokA)
	rrItems := httptest.NewRecorder()
	router.ServeHTTP(rrItems, reqItems)
	var itemsResp struct {
		Items []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"items"`
	}
	_ = json.Unmarshal(rrItems.Body.Bytes(), &itemsResp)
	if len(itemsResp.Items) < 1 {
		t.Fatal("no items")
	}

	// Cross-store: B cannot read A's job
	reqX := httptest.NewRequest(http.MethodGet, "/api/v1/stores/"+storeA.String()+"/catalog-imports/"+jobID, nil)
	reqX.Header.Set("Authorization", "Bearer "+tokB)
	rrX := httptest.NewRecorder()
	router.ServeHTTP(rrX, reqX)
	if rrX.Code != http.StatusNotFound && rrX.Code != http.StatusForbidden {
		t.Fatalf("cross-store got %d", rrX.Code)
	}

	reqXB := httptest.NewRequest(http.MethodGet, "/api/v1/stores/"+storeB.String()+"/catalog-imports/"+jobID, nil)
	reqXB.Header.Set("Authorization", "Bearer "+tokB)
	rrXB := httptest.NewRecorder()
	router.ServeHTTP(rrXB, reqXB)
	if rrXB.Code != http.StatusNotFound {
		t.Fatalf("store B job lookup got %d", rrXB.Code)
	}

	// Confirm selected → draft products
	body, _ := json.Marshal(map[string]any{"item_ids": []string{itemsResp.Items[0].ID}})
	reqC := httptest.NewRequest(http.MethodPost, "/api/v1/stores/"+storeA.String()+"/catalog-imports/"+jobID+"/confirm", bytes.NewReader(body))
	reqC.Header.Set("Authorization", "Bearer "+tokA)
	reqC.Header.Set("Content-Type", "application/json")
	rrC := httptest.NewRecorder()
	router.ServeHTTP(rrC, reqC)
	if rrC.Code != 200 {
		t.Fatalf("confirm %d %s", rrC.Code, rrC.Body.String())
	}
	var conf struct {
		CreatedCount int `json:"created_count"`
		Job          struct {
			Status string `json:"status"`
		} `json:"job"`
	}
	_ = json.Unmarshal(rrC.Body.Bytes(), &conf)
	if conf.CreatedCount < 1 || conf.Job.Status != "completed" {
		t.Fatalf("confirm result %+v", conf)
	}

	// Products are draft
	reqP := httptest.NewRequest(http.MethodGet, "/api/v1/stores/"+storeA.String()+"/products?status=draft", nil)
	reqP.Header.Set("Authorization", "Bearer "+tokA)
	rrP := httptest.NewRecorder()
	router.ServeHTTP(rrP, reqP)
	bodyP, _ := io.ReadAll(rrP.Body)
	if !bytes.Contains(bodyP, []byte("draft")) {
		t.Fatalf("expected draft products: %s", bodyP)
	}

	_ = jobs.TypeCatalogPDFImport
}
