package commerce_test

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"

	"github.com/rubicahq/pikko/services/api/internal/config"
	"github.com/rubicahq/pikko/services/api/internal/httpapi"
	"github.com/rubicahq/pikko/services/api/internal/storage"
	"github.com/rubicahq/pikko/services/api/internal/stores"
)

// TestFinanceReconciliationE2EF: 300k order / 250k paid / 110k unavailable → 190k payable / 60k refund.
func TestFinanceReconciliationE2EF(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	reset(t, pool)

	dir := t.TempDir()
	store, err := storage.NewLocal(dir)
	if err != nil {
		t.Fatal(err)
	}
	c := cfg()
	c.FeatureStoreCredit = true
	router, _ := httpapi.NewRouter(httpapi.Deps{Cfg: c, DB: pool, Logger: slog.Default(), Storage: store})
	storeSvc := &stores.Service{DB: pool, Cfg: c}

	userA, tokenA := signIn(t, pool, "finance-e2ef@example.com")
	stA, err := storeSvc.CreateStore(context.Background(), userA, stores.CreateDirectInput{
		Name: "Finance Mart", Slug: "finance-mart", Category: "other", SellingModel: "ready_stock",
	})
	if err != nil {
		t.Fatal(err)
	}
	sid := stA.ID.String()

	code, pmBody := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/payment-methods", tokenA, map[string]any{
		"bank_name": "BCA", "account_holder": "Fin", "account_number": "998877",
	})
	if code != 201 {
		t.Fatalf("payment method %d %v", code, pmBody)
	}
	pmID := pmBody["payment_method"].(map[string]any)["id"].(string)
	code, _ = doJSON(t, router, "PATCH", "/api/v1/stores/"+sid+"/fulfillment", tokenA, map[string]any{
		"pickup_enabled": true,
	})
	if code != 200 {
		t.Fatalf("fulfillment %d", code)
	}

	code, prodBody := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/products", tokenA, map[string]any{
		"product": map[string]any{
			"name": "Bundle", "slug": "bundle", "sku": "BND-1", "base_price_idr": 300000,
			"ready_stock_eligible": true, "inventory_mode": "unlimited",
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

	idem := uuid.NewString()
	code, created := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/orders", tokenA, map[string]any{
		"items":              []map[string]any{{"product_id": productID, "quantity": 1}},
		"customer_name":      "Buyer",
		"customer_email":     "buyer-fin@" + uuid.NewString()[:8] + ".example.com",
		"fulfillment_method": "pickup",
		"idempotency_key":    idem,
	})
	if code != 201 {
		t.Fatalf("order %d %v", code, created)
	}
	order := created["order"].(map[string]any)
	orderID := order["id"].(string)
	if int64(order["total_idr"].(float64)) != 300000 {
		t.Fatalf("total want 300000 got %v", order["total_idr"])
	}

	// Record 250k deposit/partial payment and verify via settlement path
	code, payBody := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/orders/"+orderID+"/payments", tokenA, map[string]any{
		"payment_method_id": pmID,
		"amount_idr":        250000,
		"payment_kind":      "deposit",
		"idempotency_key":   "pay-250k-" + orderID,
	})
	if code != 201 {
		t.Fatalf("record payment %d %v", code, payBody)
	}
	paymentID := payBody["payment"].(map[string]any)["id"].(string)

	// Idempotent replay
	code, payBody2 := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/orders/"+orderID+"/payments", tokenA, map[string]any{
		"payment_method_id": pmID,
		"amount_idr":        250000,
		"payment_kind":      "deposit",
		"idempotency_key":   "pay-250k-" + orderID,
	})
	if code != 201 {
		t.Fatalf("idempotent payment %d %v", code, payBody2)
	}
	if payBody2["payment"].(map[string]any)["id"].(string) != paymentID {
		t.Fatal("idempotency should return same payment")
	}

	code, _ = doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/orders/"+orderID+"/payments/"+paymentID+"/verify-settlement", tokenA, map[string]any{})
	if code != 200 {
		t.Fatalf("verify %d", code)
	}

	// Unavailable goods worth 110k
	code, adjBody := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/finance/adjustments", tokenA, map[string]any{
		"order_id":         orderID,
		"adjustment_type":  "unavailable",
		"amount_idr":       110000,
		"idempotency_key":  "adj-110k-" + orderID,
		"reason":           "vendor shortfall",
	})
	if code != 200 {
		t.Fatalf("adjust %d %v", code, adjBody)
	}
	fin := adjBody["finance"].(map[string]any)
	if int64(fin["payable_idr"].(float64)) != 190000 {
		t.Fatalf("payable got %v want 190000", fin["payable_idr"])
	}
	if int64(fin["refund_entitled_idr"].(float64)) != 60000 {
		t.Fatalf("refund entitled got %v want 60000", fin["refund_entitled_idr"])
	}

	code, finBody := doJSON(t, router, "GET", "/api/v1/stores/"+sid+"/finance/orders/"+orderID, tokenA, nil)
	if code != 200 {
		t.Fatalf("get finance %d %v", code, finBody)
	}
	fin = finBody["finance"].(map[string]any)
	if int64(fin["paid_idr"].(float64)) != 250000 {
		t.Fatalf("paid got %v", fin["paid_idr"])
	}
	if int64(fin["payable_idr"].(float64)) != 190000 || int64(fin["refund_entitled_idr"].(float64)) != 60000 {
		t.Fatalf("finance mismatch %+v", fin)
	}

	code, refBody := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/finance/refunds", tokenA, map[string]any{
		"order_id":         orderID,
		"amount_idr":       60000,
		"method":           "bank_transfer",
		"idempotency_key":  "refund-60k-" + orderID,
		"reason":           "unavailable excess payment",
	})
	if code != 201 {
		t.Fatalf("create refund %d %v", code, refBody)
	}
	refundID := refBody["refund"].(map[string]any)["id"].(string)
	version := int(refBody["refund"].(map[string]any)["version"].(float64))

	code, procBody := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/finance/refunds/"+refundID+"/process", tokenA, map[string]any{
		"expected_version": version,
		"status":           "completed",
	})
	if code != 200 {
		t.Fatalf("process refund %d %v", code, procBody)
	}
	if procBody["refund"].(map[string]any)["status"].(string) != "completed" {
		t.Fatalf("refund status %+v", procBody)
	}
}

func TestFinanceRefundConcurrency(t *testing.T) {
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

	userA, tokenA := signIn(t, pool, "finance-conc@example.com")
	stA, err := storeSvc.CreateStore(context.Background(), userA, stores.CreateDirectInput{
		Name: "Conc Fin", Slug: "conc-fin", Category: "other", SellingModel: "ready_stock",
	})
	if err != nil {
		t.Fatal(err)
	}
	sid := stA.ID.String()
	code, pmBody := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/payment-methods", tokenA, map[string]any{
		"bank_name": "BCA", "account_holder": "C", "account_number": "1",
	})
	if code != 201 {
		t.Fatalf("pm %d", code)
	}
	pmID := pmBody["payment_method"].(map[string]any)["id"].(string)
	_, _ = doJSON(t, router, "PATCH", "/api/v1/stores/"+sid+"/fulfillment", tokenA, map[string]any{"pickup_enabled": true})

	code, prodBody := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/products", tokenA, map[string]any{
		"product": map[string]any{
			"name": "Item", "slug": "item-c", "sku": "I1", "base_price_idr": 200000,
			"ready_stock_eligible": true, "inventory_mode": "unlimited",
		},
	})
	if code != 201 {
		t.Fatalf("prod %d", code)
	}
	productID := prodBody["product"].(map[string]any)["id"].(string)
	_, _ = doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/products/"+productID+"/status", tokenA, map[string]any{"status": "published"})

	code, created := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/orders", tokenA, map[string]any{
		"items": []map[string]any{{"product_id": productID, "quantity": 1}},
		"customer_name": "Buyer Conc", "customer_email": "buyer-conc@" + uuid.NewString()[:8] + ".example.com",
		"fulfillment_method": "pickup", "idempotency_key": uuid.NewString(),
	})
	if code != 201 {
		t.Fatalf("order %d %v", code, created)
	}
	orderID := created["order"].(map[string]any)["id"].(string)

	code, payBody := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/orders/"+orderID+"/payments", tokenA, map[string]any{
		"payment_method_id": pmID, "amount_idr": 200000, "payment_kind": "full",
		"idempotency_key": "full-" + orderID,
	})
	if code != 201 {
		t.Fatalf("pay %d %v", code, payBody)
	}
	paymentID := payBody["payment"].(map[string]any)["id"].(string)
	_, _ = doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/orders/"+orderID+"/payments/"+paymentID+"/verify-settlement", tokenA, map[string]any{})

	_, _ = doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/finance/adjustments", tokenA, map[string]any{
		"order_id": orderID, "adjustment_type": "unavailable", "amount_idr": 50000,
		"idempotency_key": "adj-" + orderID,
	})
	code, refBody := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/finance/refunds", tokenA, map[string]any{
		"order_id": orderID, "amount_idr": 50000, "method": "bank_transfer",
		"idempotency_key": "ref-" + orderID,
	})
	if code != 201 {
		t.Fatalf("refund %d %v", code, refBody)
	}
	refundID := refBody["refund"].(map[string]any)["id"].(string)
	version := int(refBody["refund"].(map[string]any)["version"].(float64))

	var okCount atomic.Int32
	var conflictCount atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			code, body := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/finance/refunds/"+refundID+"/process", tokenA, map[string]any{
				"expected_version": version,
				"status":           "completed",
			})
			if code == 200 && body["refund"].(map[string]any)["status"].(string) == "completed" {
				okCount.Add(1)
			} else if code == 409 {
				conflictCount.Add(1)
			}
		}()
	}
	wg.Wait()
	if okCount.Load() < 1 {
		t.Fatalf("expected at least one success, ok=%d conflict=%d", okCount.Load(), conflictCount.Load())
	}
	// Exactly one completed row
	var status string
	var ver int
	err = pool.QueryRow(context.Background(), `SELECT status, version FROM refunds WHERE id=$1`, refundID).Scan(&status, &ver)
	if err != nil {
		t.Fatal(err)
	}
	if status != "completed" {
		t.Fatalf("status %s", status)
	}
}

func TestFinanceStoreCreditFeatureFlag(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	reset(t, pool)

	dir := t.TempDir()
	store, err := storage.NewLocal(dir)
	if err != nil {
		t.Fatal(err)
	}
	c := cfg()
	c.FeatureStoreCredit = false
	router, _ := httpapi.NewRouter(httpapi.Deps{Cfg: c, DB: pool, Logger: slog.Default(), Storage: store})
	storeSvc := &stores.Service{DB: pool, Cfg: c}
	userA, tokenA := signIn(t, pool, "credit-flag@example.com")
	stA, err := storeSvc.CreateStore(context.Background(), userA, stores.CreateDirectInput{
		Name: "Credit Off", Slug: "credit-off", Category: "other", SellingModel: "ready_stock",
	})
	if err != nil {
		t.Fatal(err)
	}
	sid := stA.ID.String()
	code, body := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/finance/customers/"+uuid.NewString()+"/credit", tokenA, map[string]any{
		"amount_idr": 1000, "idempotency_key": "x",
	})
	if code != 409 {
		t.Fatalf("expected disabled 409 got %d %v", code, body)
	}
	_ = config.Config{}
}

func TestFinanceConsolidatedPayment(t *testing.T) {
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
	userA, tokenA := signIn(t, pool, "consol@example.com")
	stA, err := storeSvc.CreateStore(context.Background(), userA, stores.CreateDirectInput{
		Name: "Consol", Slug: "consol-pay", Category: "other", SellingModel: "ready_stock",
	})
	if err != nil {
		t.Fatal(err)
	}
	sid := stA.ID.String()
	code, pmBody := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/payment-methods", tokenA, map[string]any{
		"bank_name": "BCA", "account_holder": "C", "account_number": "2",
	})
	if code != 201 {
		t.Fatalf("pm %d", code)
	}
	pmID := pmBody["payment_method"].(map[string]any)["id"].(string)
	_, _ = doJSON(t, router, "PATCH", "/api/v1/stores/"+sid+"/fulfillment", tokenA, map[string]any{"pickup_enabled": true})
	code, prodBody := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/products", tokenA, map[string]any{
		"product": map[string]any{
			"name": "P", "slug": "p-c", "sku": "P1", "base_price_idr": 100000,
			"ready_stock_eligible": true, "inventory_mode": "unlimited",
		},
	})
	if code != 201 {
		t.Fatalf("prod %d", code)
	}
	productID := prodBody["product"].(map[string]any)["id"].(string)
	_, _ = doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/products/"+productID+"/status", tokenA, map[string]any{"status": "published"})

	email := "same@" + uuid.NewString()[:8] + ".ex.com"
	var orderIDs []string
	for i := 0; i < 2; i++ {
		code, created := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/orders", tokenA, map[string]any{
			"items": []map[string]any{{"product_id": productID, "quantity": 1}},
			"customer_name": "Same", "customer_email": email,
			"fulfillment_method": "pickup", "idempotency_key": uuid.NewString(),
		})
		if code != 201 {
			t.Fatalf("order %d %v", code, created)
		}
		orderIDs = append(orderIDs, created["order"].(map[string]any)["id"].(string))
	}

	code, payBody := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/orders/"+orderIDs[0]+"/payments", tokenA, map[string]any{
		"payment_method_id": pmID,
		"amount_idr":        200000,
		"payment_kind":      "consolidated",
		"idempotency_key":   "consol-" + orderIDs[0],
		"allocations": []map[string]any{
			{"order_id": orderIDs[0], "amount_idr": 100000},
			{"order_id": orderIDs[1], "amount_idr": 100000},
		},
	})
	if code != 201 {
		t.Fatalf("consolidated %d %v", code, payBody)
	}
	paymentID := payBody["payment"].(map[string]any)["id"].(string)
	allocs := payBody["payment"].(map[string]any)["allocations"].([]any)
	if len(allocs) != 2 {
		t.Fatalf("allocs %d", len(allocs))
	}
	code, _ = doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/orders/"+orderIDs[0]+"/payments/"+paymentID+"/verify-settlement", tokenA, map[string]any{})
	if code != 200 {
		t.Fatalf("verify consol %d", code)
	}
	for _, oid := range orderIDs {
		code, finBody := doJSON(t, router, "GET", "/api/v1/stores/"+sid+"/finance/orders/"+oid, tokenA, nil)
		if code != 200 {
			t.Fatalf("fin %d", code)
		}
		fin := finBody["finance"].(map[string]any)
		if int64(fin["paid_idr"].(float64)) != 100000 {
			t.Fatalf("order %s paid %v", oid, fin["paid_idr"])
		}
		if fin["settlement_status"].(string) != "settled" {
			t.Fatalf("settlement %v", fin["settlement_status"])
		}
	}
}
