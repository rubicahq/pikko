package commerce_test

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/rubicahq/pikko/services/api/internal/httpapi"
	"github.com/rubicahq/pikko/services/api/internal/storage"
	"github.com/rubicahq/pikko/services/api/internal/stores"
)

// TestLiveShoppingQuickAddFeedAndQuota covers LIVE-01–03 + concurrency-safe quota (E2E-K core).
func TestLiveShoppingQuickAddFeedAndQuota(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	reset(t, pool)

	dir := t.TempDir()
	objStore, err := storage.NewLocal(dir)
	if err != nil {
		t.Fatal(err)
	}
	c := cfg()
	router, _ := httpapi.NewRouter(httpapi.Deps{Cfg: c, DB: pool, Logger: slog.Default(), Storage: objStore})
	storeSvc := &stores.Service{DB: pool, Cfg: c}

	userID, token := signIn(t, pool, "live-ops@example.com")
	st, err := storeSvc.CreateStore(context.Background(), userID, stores.CreateDirectInput{
		Name: "Live Mart", Slug: "live-mart", Category: "other", SellingModel: "both",
	})
	if err != nil {
		t.Fatal(err)
	}
	sid := st.ID.String()

	code, _ := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/payment-methods", token, map[string]any{
		"bank_name": "BCA", "account_holder": "Live", "account_number": "112233",
	})
	if code != 201 {
		t.Fatalf("payment %d", code)
	}
	code, _ = doJSON(t, router, "PATCH", "/api/v1/stores/"+sid+"/fulfillment", token, map[string]any{
		"pickup_enabled": true,
	})
	if code != 200 {
		t.Fatalf("fulfillment %d", code)
	}
	code, _ = doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/storefront/publish", token, map[string]any{})
	if code != 200 {
		t.Fatalf("publish %d", code)
	}

	code, prodBody := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/products", token, map[string]any{
		"product": map[string]any{
			"name": "Live Tee", "slug": "live-tee", "base_price_idr": 150000,
			"status": "published", "ready_stock_eligible": true, "inventory_mode": "unlimited",
		},
	})
	if code != 201 {
		t.Fatalf("product %d %v", code, prodBody)
	}
	productID := prodBody["product"].(map[string]any)["id"].(string)

	opens := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	closes := time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339)
	code, campBody := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/campaigns", token, map[string]any{
		"campaign_type": "live", "name": "Friday Live", "slug": "friday-live",
		"opens_at": opens, "closes_at": closes, "publication_status": "published",
		"live_quick_add_enabled": true, "live_show_order_feed": true,
		"payment_policy": "full_after_checkout",
	})
	if code != 201 {
		t.Fatalf("campaign %d %v", code, campBody)
	}
	campaignID := campBody["campaign"].(map[string]any)["id"].(string)

	code, assignBody := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/campaigns/"+campaignID+"/products", token, map[string]any{
		"product_id": productID, "quantity_limit": 5, "availability": "available",
		"display_order": 1, "featured": true,
	})
	if code != 200 {
		t.Fatalf("assign %d %v", code, assignBody)
	}
	if assignBody["product"].(map[string]any)["featured"] != true {
		t.Fatalf("expected featured SKU %#v", assignBody)
	}

	code, sessBody := doJSON(t, router, "GET", "/api/v1/stores/"+sid+"/live/"+campaignID, token, nil)
	if code != 200 {
		t.Fatalf("session %d %v", code, sessBody)
	}
	sess := sessBody["session"].(map[string]any)
	if sess["is_accepting_orders"] != true {
		t.Fatalf("expected open session %#v", sess)
	}
	products := sess["products"].([]any)
	if len(products) != 1 {
		t.Fatalf("products %#v", products)
	}
	rem := products[0].(map[string]any)["remaining_qty"]
	if rem != float64(5) {
		t.Fatalf("remaining %#v", rem)
	}

	// Quick-create order for customer A
	code, qaBody := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/live/"+campaignID+"/quick-add", token, map[string]any{
		"customer_name": "Ayu Live", "customer_email": "ayu@example.com",
		"product_id": productID, "quantity": 2,
		"idempotency_key": "live-qa-1", "fulfillment_method": "pickup",
	})
	if code != 201 {
		t.Fatalf("quick-add create %d %v", code, qaBody)
	}
	orderID := qaBody["order"].(map[string]any)["id"].(string)
	if qaBody["created_order"] != true {
		t.Fatal("expected created_order")
	}

	// Idempotent replay
	code, replay := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/live/"+campaignID+"/quick-add", token, map[string]any{
		"customer_name": "Ayu Live", "customer_email": "ayu@example.com",
		"product_id": productID, "quantity": 2,
		"idempotency_key": "live-qa-1",
	})
	if code != 200 {
		t.Fatalf("replay %d %v", code, replay)
	}
	if replay["order"].(map[string]any)["id"] != orderID {
		t.Fatalf("replay order mismatch")
	}

	// Quick-add more onto same order
	code, addBody := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/live/"+campaignID+"/quick-add", token, map[string]any{
		"order_id": orderID, "product_id": productID, "quantity": 1,
		"idempotency_key": "live-qa-2",
	})
	if code != 200 {
		t.Fatalf("add to order %d %v", code, addBody)
	}
	items := addBody["order"].(map[string]any)["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("merged items %#v", items)
	}
	if items[0].(map[string]any)["quantity"] != float64(3) {
		t.Fatalf("qty after merge %#v", items[0])
	}

	code, feedBody := doJSON(t, router, "GET", "/api/v1/stores/"+sid+"/live/"+campaignID+"/feed", token, nil)
	if code != 200 {
		t.Fatalf("feed %d %v", code, feedBody)
	}
	feed := feedBody["feed"].([]any)
	if len(feed) < 1 {
		t.Fatalf("empty feed %#v", feedBody)
	}

	// Public discoverability
	code, pubList := doJSON(t, router, "GET", "/api/v1/public/stores/live-mart/live", "", nil)
	if code != 200 {
		t.Fatalf("public list %d %v", code, pubList)
	}
	lives := pubList["lives"].([]any)
	if len(lives) != 1 {
		t.Fatalf("lives %#v", lives)
	}
	code, pubOne := doJSON(t, router, "GET", "/api/v1/public/stores/live-mart/live/friday-live", "", nil)
	if code != 200 {
		t.Fatalf("public one %d %v", code, pubOne)
	}

	// Remaining quota = 2. Fire concurrent quick-adds of qty 2 — only one should succeed.
	var okCount, failCount atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			code, body := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/live/"+campaignID+"/quick-add", token, map[string]any{
				"customer_name": fmt.Sprintf("Buyer %d", i),
				"customer_email": fmt.Sprintf("buyer%d@example.com", i),
				"product_id": productID, "quantity": 2,
				"idempotency_key": fmt.Sprintf("conc-%d-%s", i, uuid.NewString()),
				"fulfillment_method": "pickup",
			})
			if code == 201 || code == 200 {
				okCount.Add(1)
			} else {
				failCount.Add(1)
				_ = body
			}
		}(i)
	}
	wg.Wait()
	if okCount.Load() != 1 {
		t.Fatalf("expected exactly 1 concurrent success, got ok=%d fail=%d", okCount.Load(), failCount.Load())
	}
	if failCount.Load() < 1 {
		t.Fatal("expected quota failures under concurrency")
	}

	code, sess2 := doJSON(t, router, "GET", "/api/v1/stores/"+sid+"/live/"+campaignID, token, nil)
	if code != 200 {
		t.Fatalf("session2 %d", code)
	}
	prods := sess2["session"].(map[string]any)["products"].([]any)
	sold := prods[0].(map[string]any)["sold_qty"]
	if sold != float64(5) {
		t.Fatalf("sold after concurrency %#v (want 5)", sold)
	}
}
