package commerce_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/rubicahq/pikko/services/api/internal/httpapi"
	"github.com/rubicahq/pikko/services/api/internal/storage"
	"github.com/rubicahq/pikko/services/api/internal/stores"
)

func tinyPNG() []byte {
	return []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xde, 0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41, 0x54, 0x08, 0xd7, 0x63, 0xf8, 0xff, 0xff, 0x3f,
		0x00, 0x05, 0xfe, 0x02, 0xfe, 0xa7, 0x35, 0x81, 0x84, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e,
		0x44, 0xae, 0x42, 0x60, 0x82,
	}
}

// TestPublicFileRejectsPaymentEvidence covers SEC-04: evidence file_ids must not
// be downloadable via the unauthenticated public files route.
func TestPublicFileRejectsPaymentEvidence(t *testing.T) {
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

	userA, tokenA := signIn(t, pool, "sec-evidence@example.com")
	stA, err := storeSvc.CreateStore(context.Background(), userA, stores.CreateDirectInput{
		Name: "Sec Evidence", Slug: "sec-evidence", Category: "other", SellingModel: "ready_stock",
	})
	if err != nil {
		t.Fatal(err)
	}
	sid := stA.ID.String()

	code, _ := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/payment-methods", tokenA, map[string]any{
		"bank_name": "BCA", "account_holder": "Sec", "account_number": "111",
	})
	if code != 201 {
		t.Fatalf("pm %d", code)
	}
	_, _ = doJSON(t, router, "PATCH", "/api/v1/stores/"+sid+"/fulfillment", tokenA, map[string]any{"pickup_enabled": true})
	_, _ = doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/storefront/publish", tokenA, map[string]any{})

	code, prodBody := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/products", tokenA, map[string]any{
		"product": map[string]any{
			"name": "Sec Tee", "slug": "sec-tee", "base_price_idr": 50000,
			"ready_stock_eligible": true, "inventory_mode": "unlimited",
		},
	})
	if code != 201 {
		t.Fatalf("product %d %v", code, prodBody)
	}
	productID := prodBody["product"].(map[string]any)["id"].(string)
	_, _ = doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/products/"+productID+"/status", tokenA, map[string]any{"status": "published"})

	code, methods := doJSON(t, router, "GET", "/api/v1/stores/"+sid+"/payment-methods", tokenA, nil)
	if code != 200 {
		t.Fatal(methods)
	}
	pmID := methods["payment_methods"].([]any)[0].(map[string]any)["id"].(string)

	code, submit := doJSON(t, router, "POST", "/api/v1/public/stores/sec-evidence/checkout/submit", "", map[string]any{
		"items": []map[string]any{{"product_id": productID, "quantity": 1}},
		"customer_name": "Buyer", "customer_email": "buyer-sec@" + uuid.NewString()[:8] + ".example.com",
		"fulfillment_method": "pickup", "payment_method_id": pmID,
		"idempotency_key": uuid.NewString(), "client_expected_total_idr": 50000,
	})
	if code != 201 {
		t.Fatalf("checkout %d %v", code, submit)
	}
	trackingToken := submit["tracking_token"].(string)

	body := &bytes.Buffer{}
	ct := multipartWriter(t, body, tinyPNG())
	req := httptest.NewRequest("POST", "/api/v1/public/tracking/"+trackingToken+"/evidence", body)
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("evidence upload %d %s", rec.Code, rec.Body.String())
	}

	var fileID string
	err = pool.QueryRow(context.Background(), `
		SELECT file_id::text FROM payment_evidence WHERE store_id=$1 LIMIT 1
	`, stA.ID).Scan(&fileID)
	if err != nil || fileID == "" {
		t.Fatalf("evidence file_id: %v", err)
	}

	// Public route must 404 private purpose even when store is published.
	req = httptest.NewRequest("GET", "/api/v1/public/files/"+sid+"/"+fileID, nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("public evidence download want 404 got %d body=%s", rec.Code, rec.Body.String())
	}

	// Catalog upload must reject planting payment_evidence.
	body = &bytes.Buffer{}
	ct = multipartWriter(t, body, tinyPNG())
	// multipartWriter only sets file; add purpose via custom form
	body2 := &bytes.Buffer{}
	mw := multipartWriterWithPurpose(t, body2, tinyPNG(), "payment_evidence")
	req = httptest.NewRequest("POST", "/api/v1/stores/"+sid+"/files", body2)
	req.Header.Set("Content-Type", mw)
	req.Header.Set("Authorization", "Bearer "+tokenA)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("catalog upload payment_evidence want 403 got %d %s", rec.Code, rec.Body.String())
	}
	_ = ct
}

// TestRefundPendingReservesEntitlement ensures two pending refunds cannot exceed entitled (money).
func TestRefundPendingReservesEntitlement(t *testing.T) {
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

	userA, tokenA := signIn(t, pool, "sec-refund@example.com")
	stA, err := storeSvc.CreateStore(context.Background(), userA, stores.CreateDirectInput{
		Name: "Sec Refund", Slug: "sec-refund", Category: "other", SellingModel: "ready_stock",
	})
	if err != nil {
		t.Fatal(err)
	}
	sid := stA.ID.String()
	code, pmBody := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/payment-methods", tokenA, map[string]any{
		"bank_name": "BCA", "account_holder": "R", "account_number": "2",
	})
	if code != 201 {
		t.Fatalf("pm %d", code)
	}
	pmID := pmBody["payment_method"].(map[string]any)["id"].(string)
	_, _ = doJSON(t, router, "PATCH", "/api/v1/stores/"+sid+"/fulfillment", tokenA, map[string]any{"pickup_enabled": true})

	code, prodBody := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/products", tokenA, map[string]any{
		"product": map[string]any{
			"name": "Item", "slug": "item-r", "sku": "R1", "base_price_idr": 200000,
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
		"customer_name": "Buyer", "customer_email": "buyer-r@" + uuid.NewString()[:8] + ".example.com",
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

	code, ref1 := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/finance/refunds", tokenA, map[string]any{
		"order_id": orderID, "amount_idr": 50000, "method": "bank_transfer",
		"idempotency_key": "ref-a-" + orderID,
	})
	if code != 201 {
		t.Fatalf("refund1 %d %v", code, ref1)
	}

	// Second pending for the same entitled amount must fail (pending reserves entitlement).
	code, ref2 := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/finance/refunds", tokenA, map[string]any{
		"order_id": orderID, "amount_idr": 50000, "method": "bank_transfer",
		"idempotency_key": "ref-b-" + orderID,
	})
	if code != 400 {
		t.Fatalf("second refund want 400 got %d %v", code, ref2)
	}
}

// TestStoreCreditCannotExceedRemainingDue covers REF-03 money cap (E2E-G hardening).
func TestStoreCreditCannotExceedRemainingDue(t *testing.T) {
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

	userA, tokenA := signIn(t, pool, "sec-credit@example.com")
	stA, err := storeSvc.CreateStore(context.Background(), userA, stores.CreateDirectInput{
		Name: "Sec Credit", Slug: "sec-credit", Category: "other", SellingModel: "ready_stock",
	})
	if err != nil {
		t.Fatal(err)
	}
	sid := stA.ID.String()
	_, _ = doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/payment-methods", tokenA, map[string]any{
		"bank_name": "BCA", "account_holder": "C", "account_number": "3",
	})
	_, _ = doJSON(t, router, "PATCH", "/api/v1/stores/"+sid+"/fulfillment", tokenA, map[string]any{"pickup_enabled": true})
	code, _ := doJSON(t, router, "PATCH", "/api/v1/stores/"+sid+"/finance/store-credit-settings", tokenA, map[string]any{
		"enabled": true,
	})
	if code != 200 {
		t.Fatalf("enable credit %d", code)
	}

	code, prodBody := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/products", tokenA, map[string]any{
		"product": map[string]any{
			"name": "Cred Item", "slug": "cred-item", "sku": "C1", "base_price_idr": 100000,
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
		"customer_name": "Credit Buyer", "customer_email": "buyer-c@" + uuid.NewString()[:8] + ".example.com",
		"fulfillment_method": "pickup", "idempotency_key": uuid.NewString(),
	})
	if code != 201 {
		t.Fatalf("order %d %v", code, created)
	}
	order := created["order"].(map[string]any)
	orderID := order["id"].(string)
	customerID, _ := order["customer_id"].(string)
	if customerID == "" {
		t.Fatalf("missing customer_id %#v", order)
	}

	code, grant := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/finance/customers/"+customerID+"/credit", tokenA, map[string]any{
		"amount_idr": 500000, "idempotency_key": "grant-" + orderID,
	})
	if code != 200 && code != 201 {
		t.Fatalf("grant %d %v", code, grant)
	}

	// Apply within remaining due (100k) — ok
	code, apply := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/finance/credit/apply", tokenA, map[string]any{
		"order_id": orderID, "amount_idr": 40000, "idempotency_key": "apply-ok-" + orderID,
	})
	if code != 200 {
		t.Fatalf("apply ok %d %v", code, apply)
	}

	// Remaining due is 60k — applying 100k must fail (would create artificial refund entitlement).
	code, over := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/finance/credit/apply", tokenA, map[string]any{
		"order_id": orderID, "amount_idr": 100000, "idempotency_key": "apply-over-" + orderID,
	})
	if code != 400 {
		t.Fatalf("over-apply want 400 got %d %v", code, over)
	}
}

// TestRejectPaymentAfterVerifyDoesNotUnpay ensures reject loses the race cleanly (FIN-07).
func TestRejectPaymentAfterVerifyDoesNotUnpay(t *testing.T) {
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

	userA, tokenA := signIn(t, pool, "sec-reject@example.com")
	stA, err := storeSvc.CreateStore(context.Background(), userA, stores.CreateDirectInput{
		Name: "Sec Reject", Slug: "sec-reject", Category: "other", SellingModel: "ready_stock",
	})
	if err != nil {
		t.Fatal(err)
	}
	sid := stA.ID.String()
	_, _ = doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/payment-methods", tokenA, map[string]any{
		"bank_name": "BCA", "account_holder": "R", "account_number": "9",
	})
	_, _ = doJSON(t, router, "PATCH", "/api/v1/stores/"+sid+"/fulfillment", tokenA, map[string]any{"pickup_enabled": true})
	_, _ = doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/storefront/publish", tokenA, map[string]any{})

	code, prodBody := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/products", tokenA, map[string]any{
		"product": map[string]any{
			"name": "Reject Tee", "slug": "reject-tee", "base_price_idr": 75000,
			"ready_stock_eligible": true, "inventory_mode": "unlimited",
		},
	})
	if code != 201 {
		t.Fatalf("product %d", code)
	}
	productID := prodBody["product"].(map[string]any)["id"].(string)
	_, _ = doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/products/"+productID+"/status", tokenA, map[string]any{"status": "published"})
	code, methods := doJSON(t, router, "GET", "/api/v1/stores/"+sid+"/payment-methods", tokenA, nil)
	pmID := methods["payment_methods"].([]any)[0].(map[string]any)["id"].(string)

	code, submit := doJSON(t, router, "POST", "/api/v1/public/stores/sec-reject/checkout/submit", "", map[string]any{
		"items": []map[string]any{{"product_id": productID, "quantity": 1}},
		"customer_name": "Buyer", "customer_email": "buyer-rj@" + uuid.NewString()[:8] + ".example.com",
		"fulfillment_method": "pickup", "payment_method_id": pmID,
		"idempotency_key": uuid.NewString(), "client_expected_total_idr": 75000,
	})
	if code != 201 {
		t.Fatalf("checkout %d %v", code, submit)
	}
	orderID := submit["order"].(map[string]any)["id"].(string)
	paymentID := submit["payment"].(map[string]any)["id"].(string)
	trackingToken := submit["tracking_token"].(string)

	body := &bytes.Buffer{}
	ct := multipartWriter(t, body, tinyPNG())
	req := httptest.NewRequest("POST", "/api/v1/public/tracking/"+trackingToken+"/evidence", body)
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("evidence %d %s", rec.Code, rec.Body.String())
	}

	code, _ = doJSON(t, router, "POST",
		"/api/v1/stores/"+sid+"/orders/"+orderID+"/payments/"+paymentID+"/verify",
		tokenA, map[string]any{})
	if code != 200 {
		t.Fatalf("verify %d", code)
	}

	code, rej := doJSON(t, router, "POST",
		"/api/v1/stores/"+sid+"/orders/"+orderID+"/payments/"+paymentID+"/reject",
		tokenA, map[string]any{"reason": "stale client"})
	if code != 409 {
		t.Fatalf("reject after verify want 409 got %d %v", code, rej)
	}

	code, ord := doJSON(t, router, "GET", "/api/v1/stores/"+sid+"/orders/"+orderID, tokenA, nil)
	if code != 200 {
		t.Fatalf("get order %d", code)
	}
	if ord["order"].(map[string]any)["payment_status"] != "paid" {
		t.Fatalf("order must stay paid, got %#v", ord["order"].(map[string]any)["payment_status"])
	}
}
