package commerce_test

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"

	"github.com/rubicahq/pikko/services/api/internal/httpapi"
	"github.com/rubicahq/pikko/services/api/internal/storage"
	"github.com/rubicahq/pikko/services/api/internal/stores"
)

// TestPurchasingAllocationConcurrency ensures concurrent allocators cannot
// oversell purchased stock for the same exact variant.
func TestPurchasingAllocationConcurrency(t *testing.T) {
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

	userA, tokenA := signIn(t, pool, "purchasing-alloc@example.com")
	stA, err := storeSvc.CreateStore(context.Background(), userA, stores.CreateDirectInput{
		Name: "Purch Mart", Slug: "purch-mart", Category: "other", SellingModel: "preorder",
	})
	if err != nil {
		t.Fatal(err)
	}
	sid := stA.ID.String()

	code, _ := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/payment-methods", tokenA, map[string]any{
		"bank_name": "BCA", "account_holder": "Purch", "account_number": "112233",
	})
	if code != 201 {
		t.Fatalf("payment %d", code)
	}
	code, _ = doJSON(t, router, "PATCH", "/api/v1/stores/"+sid+"/fulfillment", tokenA, map[string]any{
		"pickup_enabled": true,
	})
	if code != 200 {
		t.Fatalf("fulfillment %d", code)
	}

	code, prodBody := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/products", tokenA, map[string]any{
		"product": map[string]any{
			"name": "Batch Bag", "slug": "batch-bag", "sku": "BAG-1", "base_price_idr": 100000,
			"ready_stock_eligible": false, "preorder_eligible": true, "inventory_mode": "unlimited",
		},
	})
	if code != 201 {
		t.Fatalf("product %d %v", code, prodBody)
	}
	productID := prodBody["product"].(map[string]any)["id"].(string)
	code, _ = doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/products/"+productID+"/status", tokenA, map[string]any{
		"status": "published",
	})
	if code != 200 {
		t.Fatalf("publish %d", code)
	}

	opens := "2020-01-01T00:00:00Z"
	closes := "2030-01-01T00:00:00Z"
	code, batchBody := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/preorders", tokenA, map[string]any{
		"name": "Alloc Batch", "slug": "alloc-batch",
		"opens_at": opens, "closes_at": closes,
		"publication_status": "published", "operational_status": "open",
	})
	if code != 201 {
		t.Fatalf("batch %d %v", code, batchBody)
	}
	batchID := batchBody["batch"].(map[string]any)["id"].(string)
	code, _ = doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/preorders/"+batchID+"/products", tokenA, map[string]any{
		"product_id": productID,
	})
	if code != 200 && code != 201 {
		t.Fatalf("assign product %d", code)
	}

	// Two paid customer orders × 3 qty each via admin create + verify path simplification:
	// create two admin orders then force payment_status paid via SQL for test speed.
	var orderItemIDs []string
	for i := 0; i < 2; i++ {
		idem := uuid.NewString()
		code, created := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/orders", tokenA, map[string]any{
			"items": []map[string]any{{"product_id": productID, "quantity": 3, "batch_id": batchID}},
			"customer_name": "Buyer", "customer_email": "buyer" + uuid.NewString()[:8] + "@example.com",
			"fulfillment_method": "pickup",
			"idempotency_key": idem,
		})
		if code != 201 {
			t.Fatalf("admin order %d %v", code, created)
		}
		order := created["order"].(map[string]any)
		orderID := order["id"].(string)
		items := order["items"].([]any)
		orderItemIDs = append(orderItemIDs, items[0].(map[string]any)["id"].(string))
		_, err := pool.Exec(context.Background(), `
			UPDATE orders SET order_status='confirmed', payment_status='paid', fulfillment_status='purchasing'
			WHERE id=$1 AND store_id=$2
		`, orderID, stA.ID)
		if err != nil {
			t.Fatal(err)
		}
	}

	code, _ = doJSON(t, router, "PATCH", "/api/v1/stores/"+sid+"/preorders/"+batchID, tokenA, map[string]any{
		"operational_status": "closed",
	})
	if code != 200 {
		t.Fatalf("close batch %d", code)
	}
	code, _ = doJSON(t, router, "PATCH", "/api/v1/stores/"+sid+"/preorders/"+batchID, tokenA, map[string]any{
		"operational_status": "purchasing",
	})
	if code != 200 {
		t.Fatalf("start purchasing %d", code)
	}

	// Purchase only 3 units total — concurrent allocators racing for 3+3 must leave one short.
	code, _ = doJSON(t, router, "PUT", "/api/v1/stores/"+sid+"/preorders/"+batchID+"/purchasing", tokenA, map[string]any{
		"product_id": productID, "purchased_qty": 3,
	})
	if code != 200 {
		t.Fatalf("record purchase %d", code)
	}

	code, list := doJSON(t, router, "GET", "/api/v1/stores/"+sid+"/purchasing?batch_id="+batchID, tokenA, nil)
	if code != 200 {
		t.Fatalf("store purchasing %d %v", code, list)
	}
	purch := list["purchasing"].([]any)
	if len(purch) == 0 {
		t.Fatal("expected purchasing lines")
	}
	line := purch[0].(map[string]any)
	if int(line["required_qty"].(float64)) != 6 {
		t.Fatalf("required_qty want 6 got %v", line["required_qty"])
	}
	if int(line["purchased_qty"].(float64)) != 3 {
		t.Fatalf("purchased_qty want 3 got %v", line["purchased_qty"])
	}

	var okCount, conflictCount atomic.Int64
	var wg sync.WaitGroup
	for _, itemID := range orderItemIDs {
		wg.Add(1)
		go func(itemID string) {
			defer wg.Done()
			code, body := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/preorders/"+batchID+"/allocations", tokenA, map[string]any{
				"order_item_id": itemID, "allocated_qty": 3, "unavailable_qty": 0,
			})
			if code == 200 {
				okCount.Add(1)
			} else if code == 409 || code == 400 {
				conflictCount.Add(1)
				_ = body
			} else {
				t.Errorf("unexpected alloc status %d %v", code, body)
			}
		}(itemID)
	}
	wg.Wait()

	if okCount.Load() != 1 {
		t.Fatalf("expected exactly 1 successful allocation, got ok=%d conflict=%d", okCount.Load(), conflictCount.Load())
	}
	if conflictCount.Load() != 1 {
		t.Fatalf("expected exactly 1 rejected allocation, got ok=%d conflict=%d", okCount.Load(), conflictCount.Load())
	}

	// Partial unavailable + adjustment preserves original qty via revision.
	failItem := orderItemIDs[0]
	// Find which item was NOT allocated fully — mark unavailable on the other if needed.
	code, impact := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/purchasing/unavailable-impact", tokenA, map[string]any{
		"batch_id": batchID, "product_id": productID, "unavailable_qty": 3,
	})
	if code != 200 {
		t.Fatalf("impact %d %v", code, impact)
	}
	imp := impact["impact"].(map[string]any)
	items := imp["items"].([]any)
	if len(items) < 1 {
		t.Fatal("expected impact items")
	}

	// Pick an order item and mark 1 unavailable with payable adjust.
	code, ordersList := doJSON(t, router, "GET", "/api/v1/stores/"+sid+"/preorders/"+batchID+"/orders", tokenA, nil)
	if code != 200 {
		t.Fatalf("batch orders %d", code)
	}
	_ = failItem
	firstOrder := ordersList["orders"].([]any)[0].(map[string]any)
	orderID := firstOrder["id"].(string)
	code, orderDetail := doJSON(t, router, "GET", "/api/v1/stores/"+sid+"/orders/"+orderID, tokenA, nil)
	if code != 200 {
		t.Fatalf("order detail %d", code)
	}
	oi := orderDetail["order"].(map[string]any)["items"].([]any)[0].(map[string]any)
	itemID := oi["id"].(string)
	origQty := int(oi["quantity"].(float64))
	code, marked := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/orders/"+orderID+"/items/"+itemID+"/unavailable", tokenA, map[string]any{
		"unavailable_qty": 1, "apply_adjustment": true, "reason": "vendor short",
	})
	if code != 200 {
		t.Fatalf("mark unavailable %d %v", code, marked)
	}
	ord := marked["order"].(map[string]any)
	it := ord["items"].([]any)[0].(map[string]any)
	if int(it["quantity"].(float64)) != origQty {
		t.Fatalf("quantity must be preserved, got %v want %d", it["quantity"], origQty)
	}
	if it["availability_status"] == "available" {
		t.Fatal("availability should change")
	}
	code, revs := doJSON(t, router, "GET", "/api/v1/stores/"+sid+"/orders/"+orderID+"/revisions", tokenA, nil)
	if code != 200 {
		t.Fatalf("revisions %d", code)
	}
	found := false
	for _, r := range revs["revisions"].([]any) {
		if r.(map[string]any)["change_type"] == "item_unavailable" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected item_unavailable revision")
	}

	// Receiving foundation
	code, recv := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/receiving", tokenA, map[string]any{
		"batch_id": batchID,
		"notes":    "partial arrival",
		"lines": []map[string]any{{
			"product_id": productID, "expected_qty": 3, "received_qty": 2,
			"discrepancy_notes": "1 pending",
		}},
	})
	if code != 201 {
		t.Fatalf("receiving create %d %v", code, recv)
	}
	rec := recv["receiving"].(map[string]any)
	if rec["status"] != "partial" {
		t.Fatalf("expected partial receiving, got %v", rec["status"])
	}
}
