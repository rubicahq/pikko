package commerce_test

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/rubicahq/pikko/services/api/internal/httpapi"
	"github.com/rubicahq/pikko/services/api/internal/storage"
	"github.com/rubicahq/pikko/services/api/internal/stores"
)

func TestScheduledLiveWishlistPhoneGateAndMerchantView(t *testing.T) {
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

	userID, token := signIn(t, pool, "wishlist-ops@example.com")
	st, err := storeSvc.CreateStore(context.Background(), userID, stores.CreateDirectInput{
		Name: "Wish Mart", Slug: "wish-mart", Category: "other", SellingModel: "both",
		PlanCode: "premium", BillingCycle: "monthly",
	})
	if err != nil {
		t.Fatal(err)
	}
	sid := st.ID.String()

	code, _ := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/payment-methods", token, map[string]any{
		"bank_name": "BCA", "account_holder": "Wish", "account_number": "998877",
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
			"name": "Wish Tee", "slug": "wish-tee", "base_price_idr": 120000,
			"status": "published", "ready_stock_eligible": true, "inventory_mode": "unlimited",
		},
	})
	if code != 201 {
		t.Fatalf("product %d %v", code, prodBody)
	}
	productID := prodBody["product"].(map[string]any)["id"].(string)

	// Scheduled window: opens in the future.
	opens := time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339)
	closes := time.Now().UTC().Add(5 * time.Hour).Format(time.RFC3339)
	code, campBody := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/campaigns", token, map[string]any{
		"campaign_type": "live", "name": "Weekend Live", "slug": "weekend-live",
		"opens_at": opens, "closes_at": closes, "publication_status": "published",
		"live_quick_add_enabled": true, "live_show_order_feed": true,
		"payment_policy": "full_after_checkout",
	})
	if code != 201 {
		t.Fatalf("campaign %d %v", code, campBody)
	}
	campaignID := campBody["campaign"].(map[string]any)["id"].(string)

	code, _ = doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/campaigns/"+campaignID+"/products", token, map[string]any{
		"product_id": productID, "availability": "available", "display_order": 1, "featured": true,
	})
	if code != 200 {
		t.Fatalf("assign %d", code)
	}

	code, liveBody := doJSON(t, router, "GET", "/api/v1/public/stores/wish-mart/live/weekend-live", "", nil)
	if code != 200 {
		t.Fatalf("public live %d %v", code, liveBody)
	}
	live := liveBody["live"].(map[string]any)
	if live["wishlist_enabled"] != true {
		t.Fatalf("expected wishlist_enabled for scheduled live, got %#v", live["wishlist_enabled"])
	}
	if live["is_accepting_orders"] == true {
		t.Fatalf("scheduled live should not accept orders")
	}

	// Missing phone → invalid
	code, bad := doJSON(t, router, "POST", "/api/v1/public/stores/wish-mart/live/weekend-live/wishlist", "", map[string]any{
		"product_id": productID,
	})
	if code != 400 {
		t.Fatalf("wishlist without phone want 400 got %d %v", code, bad)
	}

	code, addBody := doJSON(t, router, "POST", "/api/v1/public/stores/wish-mart/live/weekend-live/wishlist", "", map[string]any{
		"phone": "081234567890", "name": "Sari", "product_id": productID,
	})
	if code != 201 {
		t.Fatalf("add wishlist %d %v", code, addBody)
	}
	portalURL, _ := addBody["portal_url"].(string)
	if portalURL == "" || !strings.Contains(portalURL, "/wish-mart/customer/") || !strings.Contains(portalURL, "6281234567890") {
		t.Fatalf("portal_url=%q", portalURL)
	}
	if addBody["customer_created"] != true {
		t.Fatalf("expected customer_created")
	}
	customerID, _ := addBody["customer_id"].(string)
	if customerID == "" {
		t.Fatalf("missing customer_id")
	}

	// Idempotent re-add
	code, again := doJSON(t, router, "POST", "/api/v1/public/stores/wish-mart/live/weekend-live/wishlist", "", map[string]any{
		"phone": "081234567890", "product_id": productID,
	})
	if code != 200 {
		t.Fatalf("re-add %d %v", code, again)
	}
	if again["already_saved"] != true {
		t.Fatalf("expected already_saved %#v", again)
	}
	if again["customer_created"] == true {
		t.Fatalf("should not recreate customer")
	}

	code, listBody := doJSON(t, router, "GET", "/api/v1/public/stores/wish-mart/live/weekend-live/wishlist?phone=081234567890", "", nil)
	if code != 200 {
		t.Fatalf("list public %d %v", code, listBody)
	}
	items, _ := listBody["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("want 1 item got %d", len(items))
	}

	code, merch := doJSON(t, router, "GET", "/api/v1/stores/"+sid+"/customers/"+customerID+"/wishlist", token, nil)
	if code != 200 {
		t.Fatalf("merchant wishlist %d %v", code, merch)
	}
	mItems, _ := merch["items"].([]any)
	if len(mItems) != 1 {
		t.Fatalf("merchant want 1 got %d %#v", len(mItems), merch)
	}
	first := mItems[0].(map[string]any)
	if first["product_name"] != "Wish Tee" {
		t.Fatalf("product_name=%v", first["product_name"])
	}

	// Open live → wishlist unavailable
	pastOpen := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	futureClose := time.Now().UTC().Add(3 * time.Hour).Format(time.RFC3339)
	code, _ = doJSON(t, router, "PATCH", "/api/v1/stores/"+sid+"/campaigns/"+campaignID, token, map[string]any{
		"opens_at": pastOpen, "closes_at": futureClose,
	})
	if code != 200 {
		t.Fatalf("patch campaign %d", code)
	}
	code, closed := doJSON(t, router, "POST", "/api/v1/public/stores/wish-mart/live/weekend-live/wishlist", "", map[string]any{
		"phone": "081298765432", "product_id": productID,
	})
	if code != 409 {
		t.Fatalf("open live wishlist want 409 got %d %v", code, closed)
	}
}
