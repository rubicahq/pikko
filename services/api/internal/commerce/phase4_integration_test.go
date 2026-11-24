package commerce_test

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/rubicahq/pikko/services/api/internal/httpapi"
	"github.com/rubicahq/pikko/services/api/internal/storage"
	"github.com/rubicahq/pikko/services/api/internal/stores"
)

func TestPhase4PreorderRulesQuotaAndIsolation(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	reset(t, pool)

	dir := t.TempDir()
	store, err := storage.NewLocal(dir)
	if err != nil {
		t.Fatal(err)
	}
	c := cfg()
	router, _ := httpapi.NewRouter(httpapi.Deps{Cfg: c, DB: pool, Logger: slog.Default(), Storage: store})
	storeSvc := &stores.Service{DB: pool, Cfg: c}

	userA, tokenA := signIn(t, pool, "p4-a@example.com")
	userB, tokenB := signIn(t, pool, "p4-b@example.com")
	stA, err := storeSvc.CreateStore(context.Background(), userA, stores.CreateDirectInput{
		Name: "Jastip A", Slug: "jastip-a", Category: "personal_shopper_jastip", SellingModel: "both",
	})
	if err != nil {
		t.Fatal(err)
	}
	stB, err := storeSvc.CreateStore(context.Background(), userB, stores.CreateDirectInput{
		Name: "Jastip B", Slug: "jastip-b", Category: "other", SellingModel: "preorder",
	})
	if err != nil {
		t.Fatal(err)
	}

	code, _ := doJSON(t, router, "POST", "/api/v1/stores/"+stA.ID.String()+"/payment-methods", tokenA, map[string]any{
		"bank_name": "BCA", "account_holder": "Jastip A", "account_number": "111",
	})
	if code != 201 {
		t.Fatalf("pm %d", code)
	}
	code, _ = doJSON(t, router, "PATCH", "/api/v1/stores/"+stA.ID.String()+"/fulfillment", tokenA, map[string]any{
		"pickup_enabled": true,
	})
	if code != 200 {
		t.Fatalf("fulfillment %d", code)
	}
	code, _ = doJSON(t, router, "POST", "/api/v1/stores/"+stA.ID.String()+"/storefront/publish", tokenA, nil)
	if code != 200 {
		t.Fatalf("publish %d", code)
	}

	code, prodBody := doJSON(t, router, "POST", "/api/v1/stores/"+stA.ID.String()+"/products", tokenA, map[string]any{
		"product": map[string]any{
			"name": "Japan Bag", "slug": "japan-bag", "base_price_idr": 200000,
			"ready_stock_eligible": false, "preorder_eligible": true, "inventory_mode": "unlimited",
		},
	})
	if code != 201 {
		t.Fatalf("product %d %v", code, prodBody)
	}
	productID := prodBody["product"].(map[string]any)["id"].(string)
	code, _ = doJSON(t, router, "POST", "/api/v1/stores/"+stA.ID.String()+"/products/"+productID+"/status", tokenA, map[string]any{
		"status": "published",
	})
	if code != 200 {
		t.Fatalf("publish product %d", code)
	}

	opens := time.Now().UTC().Add(-time.Hour)
	closes := time.Now().UTC().Add(48 * time.Hour)
	code, batchBody := doJSON(t, router, "POST", "/api/v1/stores/"+stA.ID.String()+"/preorders", tokenA, map[string]any{
		"name": "Japan Dec", "slug": "japan-dec",
		"opens_at": opens.Format(time.RFC3339), "closes_at": closes.Format(time.RFC3339),
		"publication_status": "published", "payment_policy": "full_after_checkout",
		"eta_start": time.Now().UTC().Add(20 * 24 * time.Hour).Format("2006-01-02"),
	})
	if code != 201 {
		t.Fatalf("batch %d %v", code, batchBody)
	}
	batchID := batchBody["batch"].(map[string]any)["id"].(string)

	code, _ = doJSON(t, router, "POST", "/api/v1/stores/"+stA.ID.String()+"/preorders/"+batchID+"/products", tokenA, map[string]any{
		"product_id": productID, "price_override_idr": 180000, "quantity_limit": 2, "availability": "available",
	})
	if code != 200 {
		t.Fatalf("assign %d", code)
	}

	// Handling fee + approval rules (batch overrides store)
	code, _ = doJSON(t, router, "POST", "/api/v1/stores/"+stA.ID.String()+"/business-rules", tokenA, map[string]any{
		"rule_type": "handling_fee", "name": "Store fee", "scope_type": "store",
		"config": map[string]any{"fee_mode": "fixed", "fee_fixed_idr": 5000},
	})
	if code != 201 {
		t.Fatalf("store fee %d", code)
	}
	code, _ = doJSON(t, router, "POST", "/api/v1/stores/"+stA.ID.String()+"/business-rules", tokenA, map[string]any{
		"rule_type": "handling_fee", "name": "Batch fee", "scope_type": "batch", "scope_batch_id": batchID,
		"config": map[string]any{"fee_mode": "fixed", "fee_fixed_idr": 20000},
	})
	if code != 201 {
		t.Fatalf("batch fee %d", code)
	}
	code, _ = doJSON(t, router, "POST", "/api/v1/stores/"+stA.ID.String()+"/business-rules", tokenA, map[string]any{
		"rule_type": "order_approval", "name": "Approve preorders", "scope_type": "selling_model",
		"scope_selling_model": "preorder",
		"config":              map[string]any{"require_approval": true},
	})
	if code != 201 {
		t.Fatalf("approval %d", code)
	}

	code, preview := doJSON(t, router, "POST", "/api/v1/public/stores/jastip-a/checkout/preview", "", map[string]any{
		"items": []map[string]any{
			{"product_id": productID, "quantity": 1, "batch_id": batchID},
		},
		"fulfillment_method": "pickup",
	})
	if code != 200 {
		t.Fatalf("preview %d %v", code, preview)
	}
	p := preview["preview"].(map[string]any)
	if int64(p["handling_fee_idr"].(float64)) != 20000 {
		t.Fatalf("expected batch fee 20000, got %v", p["handling_fee_idr"])
	}
	if int64(p["subtotal_idr"].(float64)) != 180000 {
		t.Fatalf("batch price %v", p["subtotal_idr"])
	}
	if p["requires_approval"] != true {
		t.Fatalf("approval %v", p["requires_approval"])
	}

	code, methods := doJSON(t, router, "GET", "/api/v1/stores/"+stA.ID.String()+"/payment-methods", tokenA, nil)
	pmID := methods["payment_methods"].([]any)[0].(map[string]any)["id"].(string)

	code, submit := doJSON(t, router, "POST", "/api/v1/public/stores/jastip-a/checkout/submit", "", map[string]any{
		"items": []map[string]any{
			{"product_id": productID, "quantity": 1, "batch_id": batchID},
		},
		"customer_name": "Ani", "customer_email": "ani@example.com",
		"fulfillment_method": "pickup", "payment_method_id": pmID,
		"idempotency_key": uuid.NewString(), "client_expected_total_idr": 200000,
	})
	if code != 201 {
		t.Fatalf("submit %d %v", code, submit)
	}
	order := submit["order"].(map[string]any)
	if order["order_status"] != "pending_confirmation" {
		t.Fatalf("status %v", order["order_status"])
	}
	if order["selling_model"] != "preorder" {
		t.Fatalf("model %v", order["selling_model"])
	}
	orderID := order["id"].(string)

	// Snapshot rule applications
	var ruleApps int
	_ = pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM order_rule_applications WHERE order_id=$1`, orderID).Scan(&ruleApps)
	if ruleApps < 2 {
		t.Fatalf("expected rule snapshots, got %d", ruleApps)
	}

	// Cross-store cannot read batch
	code, _ = doJSON(t, router, "GET", "/api/v1/stores/"+stB.ID.String()+"/preorders/"+batchID, tokenB, nil)
	if code != 404 {
		t.Fatalf("cross-store batch %d", code)
	}

	// Close batch → reject new orders
	code, _ = doJSON(t, router, "PATCH", "/api/v1/stores/"+stA.ID.String()+"/preorders/"+batchID, tokenA, map[string]any{
		"operational_status": "closed",
	})
	if code != 200 {
		t.Fatalf("close %d", code)
	}
	code, closed := doJSON(t, router, "POST", "/api/v1/public/stores/jastip-a/checkout/submit", "", map[string]any{
		"items": []map[string]any{
			{"product_id": productID, "quantity": 1, "batch_id": batchID},
		},
		"customer_name": "Cici", "customer_email": "cici@example.com",
		"fulfillment_method": "pickup", "payment_method_id": pmID,
		"idempotency_key": uuid.NewString(),
	})
	if code != 409 {
		t.Fatalf("expected closed batch conflict, got %d %v", code, closed)
	}

	// Reopen and fill remaining quota (limit 2, already 1 reserved)
	code, reopened := doJSON(t, router, "PATCH", "/api/v1/stores/"+stA.ID.String()+"/preorders/"+batchID, tokenA, map[string]any{
		"operational_status": "open",
		"closes_at":          time.Now().UTC().Add(72 * time.Hour).Format(time.RFC3339),
	})
	if code != 200 {
		t.Fatalf("reopen %d %v", code, reopened)
	}
	if reopened["batch"].(map[string]any)["operational_status"] != "open" {
		t.Fatalf("batch not open after reopen: %v", reopened)
	}

	code, submit2 := doJSON(t, router, "POST", "/api/v1/public/stores/jastip-a/checkout/submit", "", map[string]any{
		"items": []map[string]any{
			{"product_id": productID, "quantity": 1, "batch_id": batchID},
		},
		"customer_name": "Q2", "customer_email": "q2@example.com",
		"fulfillment_method": "pickup", "payment_method_id": pmID,
		"idempotency_key": uuid.NewString(), "client_expected_total_idr": 200000,
	})
	if code != 201 {
		t.Fatalf("second order within quota %d %v", code, submit2)
	}

	// Concurrent oversell attempts should both fail (quota full)
	var wg sync.WaitGroup
	results := make(chan int, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			code, _ := doJSON(t, router, "POST", "/api/v1/public/stores/jastip-a/checkout/submit", "", map[string]any{
				"items": []map[string]any{
					{"product_id": productID, "quantity": 1, "batch_id": batchID},
				},
				"customer_name": "Q", "customer_email": "q" + uuid.NewString()[:8] + "@example.com",
				"fulfillment_method": "pickup", "payment_method_id": pmID,
				"idempotency_key": uuid.NewString(), "client_expected_total_idr": 200000,
			})
			results <- code
		}()
	}
	wg.Wait()
	close(results)
	for code := range results {
		if code == 201 {
			t.Fatal("quota oversold under concurrency")
		}
	}

	// Confirm pending order
	code, conf := doJSON(t, router, "PATCH", "/api/v1/stores/"+stA.ID.String()+"/orders/"+orderID, tokenA, map[string]any{
		"order_status": "confirmed",
	})
	if code != 200 {
		t.Fatalf("confirm %d %v", code, conf)
	}
	if conf["order"].(map[string]any)["order_status"] != "confirmed" {
		t.Fatalf("confirm status %v", conf)
	}

	// Rule preview uses same evaluation
	code, rp := doJSON(t, router, "POST", "/api/v1/stores/"+stA.ID.String()+"/business-rules/preview", tokenA, map[string]any{
		"selling_model": "preorder", "batch_id": batchID, "subtotal_idr": 180000,
		"item_quantity": 1, "fulfillment_method": "pickup",
	})
	if code != 200 {
		t.Fatalf("rule preview %d", code)
	}
	if int64(rp["preview"].(map[string]any)["handling_fee_idr"].(float64)) != 20000 {
		t.Fatalf("preview fee %v", rp)
	}
}
