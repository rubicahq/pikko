package commerce_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/google/uuid"

	"github.com/rubicahq/pikko/services/api/internal/httpapi"
	"github.com/rubicahq/pikko/services/api/internal/storage"
	"github.com/rubicahq/pikko/services/api/internal/stores"
)

// TestPackingAndShipmentFlow covers E2E-I: pack checklist → ready_to_ship → multi-parcel ship + email.
func TestPackingAndShipmentFlow(t *testing.T) {
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

	userID, token := signIn(t, pool, "packing-ops@example.com")
	st, err := storeSvc.CreateStore(context.Background(), userID, stores.CreateDirectInput{
		Name: "Pack Mart", Slug: "pack-mart", Category: "other", SellingModel: "ready_stock",
	})
	if err != nil {
		t.Fatal(err)
	}
	sid := st.ID.String()

	code, _ := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/payment-methods", token, map[string]any{
		"bank_name": "BCA", "account_holder": "Pack", "account_number": "998877",
	})
	if code != 201 {
		t.Fatalf("payment %d", code)
	}
	code, _ = doJSON(t, router, "PATCH", "/api/v1/stores/"+sid+"/fulfillment", token, map[string]any{
		"shipping_enabled": true, "shipping_fee_idr": 10000,
	})
	if code != 200 {
		t.Fatalf("fulfillment %d", code)
	}

	code, prodBody := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/products", token, map[string]any{
		"product": map[string]any{
			"name": "Ready Tee", "slug": "ready-tee", "sku": "TEE-1", "base_price_idr": 75000,
			"ready_stock_eligible": true, "preorder_eligible": false, "inventory_mode": "unlimited",
		},
	})
	if code != 201 {
		t.Fatalf("product %d %v", code, prodBody)
	}
	productID := prodBody["product"].(map[string]any)["id"].(string)
	code, _ = doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/products/"+productID+"/status", token, map[string]any{
		"status": "published",
	})
	if code != 200 {
		t.Fatalf("publish %d", code)
	}

	code, created := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/orders", token, map[string]any{
		"items":              []map[string]any{{"product_id": productID, "quantity": 3}},
		"customer_name":      "Ani Pack",
		"customer_email":     "ani.pack@example.com",
		"fulfillment_method": "shipping",
		"shipping_address": map[string]any{
			"recipient_name": "Ani Pack", "line1": "Jl. Merdeka 1", "city": "Jakarta", "country": "ID",
		},
		"idempotency_key": uuid.NewString(),
	})
	if code != 201 {
		t.Fatalf("admin order %d %v", code, created)
	}
	order := created["order"].(map[string]any)
	orderID := order["id"].(string)
	items := order["items"].([]any)
	itemID := items[0].(map[string]any)["id"].(string)

	_, err = pool.Exec(context.Background(), `
		UPDATE orders SET order_status='confirmed', payment_status='paid', fulfillment_status='unfulfilled'
		WHERE id=$1 AND store_id=$2
	`, orderID, st.ID)
	if err != nil {
		t.Fatal(err)
	}

	// PACK-01 queue
	code, queue := doJSON(t, router, "GET", "/api/v1/stores/"+sid+"/packing?group_by=order&status=eligible", token, nil)
	if code != 200 {
		t.Fatalf("queue %d %v", code, queue)
	}
	orders, _ := queue["orders"].([]any)
	if len(orders) == 0 {
		t.Fatalf("expected packing queue entry: %v", queue)
	}

	// PACK-02 checklist + save
	code, check := doJSON(t, router, "PUT", "/api/v1/stores/"+sid+"/packing/"+orderID+"/checklist", token, map[string]any{
		"items": []map[string]any{{"order_item_id": itemID, "qty_packed": 2}},
	})
	if code != 200 {
		t.Fatalf("checklist %d %v", code, check)
	}

	// Tenant isolation: other store cannot read packing
	userB, tokenB := signIn(t, pool, "other-pack@example.com")
	stB, err := storeSvc.CreateStore(context.Background(), userB, stores.CreateDirectInput{
		Name: "Other Mart", Slug: "other-pack-mart", Category: "other", SellingModel: "ready_stock",
	})
	if err != nil {
		t.Fatal(err)
	}
	code, _ = doJSON(t, router, "GET", "/api/v1/stores/"+stB.ID.String()+"/packing/"+orderID, tokenB, nil)
	if code != 404 && code != 403 {
		t.Fatalf("cross-store packing should be denied, got %d", code)
	}

	// PACK-03 complete → ready_to_ship (fills remaining packed qty)
	code, done := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/packing/"+orderID+"/complete", token, map[string]any{})
	if code != 200 {
		t.Fatalf("complete %d %v", code, done)
	}
	packing := done["packing"].(map[string]any)
	if packing["fulfillment_status"] != "ready_to_ship" {
		t.Fatalf("expected ready_to_ship got %v", packing["fulfillment_status"])
	}

	// SHIP-03 partial first parcel (2 of 3)
	code, ship1 := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/shipments", token, map[string]any{
		"order_id": orderID, "carrier": "JNE", "tracking_number": "JNE123",
		"items": []map[string]any{{"order_item_id": itemID, "quantity": 2}},
	})
	if code != 201 {
		t.Fatalf("ship1 %d %v", code, ship1)
	}
	var fulfill string
	_ = pool.QueryRow(context.Background(), `SELECT fulfillment_status FROM orders WHERE id=$1`, orderID).Scan(&fulfill)
	if fulfill != "ready_to_ship" {
		t.Fatalf("partial ship should keep ready_to_ship, got %s", fulfill)
	}

	// SHIP-01/02 remaining parcel + email job
	code, ship2 := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/shipments", token, map[string]any{
		"order_id": orderID, "carrier": "JNE", "tracking_number": "JNE456",
		"items": []map[string]any{{"order_item_id": itemID, "quantity": 1}},
	})
	if code != 201 {
		t.Fatalf("ship2 %d %v", code, ship2)
	}
	_ = pool.QueryRow(context.Background(), `SELECT fulfillment_status FROM orders WHERE id=$1`, orderID).Scan(&fulfill)
	if fulfill != "shipped" {
		t.Fatalf("expected shipped after full parcels, got %s", fulfill)
	}

	var emailCount int
	_ = pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM email_jobs WHERE template_key='order_shipped_customer' AND to_email='ani.pack@example.com'
	`).Scan(&emailCount)
	if emailCount < 2 {
		t.Fatalf("expected shipment emails, got %d", emailCount)
	}

	var outboxCount int
	_ = pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM outbox_events WHERE event_type IN ('packing.completed','shipment.created')
	`).Scan(&outboxCount)
	if outboxCount < 2 {
		t.Fatalf("expected outbox events, got %d", outboxCount)
	}

	code, list := doJSON(t, router, "GET", "/api/v1/stores/"+sid+"/orders/"+orderID+"/shipments", token, nil)
	if code != 200 {
		t.Fatalf("list shipments %d", code)
	}
	shList := list["shipments"].([]any)
	if len(shList) != 2 {
		t.Fatalf("expected 2 shipments got %d", len(shList))
	}
}

// TestPackingPickupReady ensures pickup orders become ready_for_pickup (no carrier shipment).
func TestPackingPickupReady(t *testing.T) {
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

	userID, token := signIn(t, pool, "pickup-pack@example.com")
	st, err := storeSvc.CreateStore(context.Background(), userID, stores.CreateDirectInput{
		Name: "Pickup Mart", Slug: "pickup-pack-mart", Category: "other", SellingModel: "ready_stock",
	})
	if err != nil {
		t.Fatal(err)
	}
	sid := st.ID.String()
	_, _ = doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/payment-methods", token, map[string]any{
		"bank_name": "BCA", "account_holder": "P", "account_number": "1",
	})
	_, _ = doJSON(t, router, "PATCH", "/api/v1/stores/"+sid+"/fulfillment", token, map[string]any{
		"pickup_enabled": true,
	})
	code, prodBody := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/products", token, map[string]any{
		"product": map[string]any{
			"name": "Mug", "slug": "mug", "base_price_idr": 40000,
			"ready_stock_eligible": true, "inventory_mode": "unlimited",
		},
	})
	if code != 201 {
		t.Fatalf("product %d", code)
	}
	productID := prodBody["product"].(map[string]any)["id"].(string)
	_, _ = doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/products/"+productID+"/status", token, map[string]any{"status": "published"})

	code, created := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/orders", token, map[string]any{
		"items": []map[string]any{{"product_id": productID, "quantity": 1}},
		"customer_name": "Bob", "customer_email": "bob.pickup@example.com",
		"fulfillment_method": "pickup", "idempotency_key": uuid.NewString(),
	})
	if code != 201 {
		t.Fatalf("order %d %v", code, created)
	}
	orderID := created["order"].(map[string]any)["id"].(string)
	_, _ = pool.Exec(context.Background(), `
		UPDATE orders SET order_status='confirmed', payment_status='paid', fulfillment_status='unfulfilled' WHERE id=$1
	`, orderID)

	code, done := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/packing/"+orderID+"/complete", token, map[string]any{})
	if code != 200 {
		t.Fatalf("complete %d %v", code, done)
	}
	if done["packing"].(map[string]any)["fulfillment_status"] != "ready_for_pickup" {
		t.Fatalf("expected ready_for_pickup: %v", done)
	}

	code, ship := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/shipments", token, map[string]any{
		"order_id": orderID, "carrier": "X", "tracking_number": "Y",
		"items": []map[string]any{{"order_item_id": created["order"].(map[string]any)["items"].([]any)[0].(map[string]any)["id"], "quantity": 1}},
	})
	if code != 409 {
		t.Fatalf("pickup shipment should conflict, got %d %v", code, ship)
	}
}
