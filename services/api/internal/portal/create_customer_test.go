package portal_test

import (
	"context"
	"log/slog"
	"net/url"
	"strings"
	"testing"

	"github.com/rubicahq/pikko/services/api/internal/httpapi"
	"github.com/rubicahq/pikko/services/api/internal/storage"
	"github.com/rubicahq/pikko/services/api/internal/stores"
	"github.com/rubicahq/pikko/services/api/internal/validate"
)

func TestCreateCustomerWithoutOrderPortalAccess(t *testing.T) {
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

	_, token := signInMerchant(t, pool, "owner-create-cust@example.com")
	st, err := storeSvc.CreateStore(context.Background(), mustUser(t, pool, "owner-create-cust@example.com"), stores.CreateDirectInput{
		Name: "WA Shop", Slug: "wa-shop", Category: "other", SellingModel: "ready_stock",
	})
	if err != nil {
		t.Fatal(err)
	}
	sid := st.ID.String()

	// Reject invalid phone
	code, body := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/customers", token, map[string]any{
		"name": "Budi", "phone": "0211234567",
	})
	if code != 400 {
		t.Fatalf("invalid phone want 400 got %d %v", code, body)
	}

	// Create with name + WA only
	code, created := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/customers", token, map[string]any{
		"name": "Budi Santoso", "phone": "0812-3456-7890",
	})
	if code != 201 {
		t.Fatalf("create customer %d %v", code, created)
	}
	cust, _ := created["customer"].(map[string]any)
	if cust == nil {
		t.Fatalf("missing customer %v", created)
	}
	if cust["name"] != "Budi Santoso" {
		t.Fatalf("name=%v", cust["name"])
	}
	if cust["phone"] != "6281234567890" {
		t.Fatalf("phone=%v want normalized 6281234567890", cust["phone"])
	}
	if cust["email"] != nil {
		t.Fatalf("email should be omitted/nil got %v", cust["email"])
	}
	if int(cust["total_orders"].(float64)) != 0 {
		t.Fatalf("total_orders=%v", cust["total_orders"])
	}
	portalURL, _ := created["portal_url"].(string)
	if !strings.Contains(portalURL, "/wa-shop/customer/") {
		t.Fatalf("portal_url=%q want /wa-shop/customer/{secret}-{phone}", portalURL)
	}
	u, err := url.Parse(portalURL)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if len(parts) != 3 || parts[0] != "wa-shop" || parts[1] != "customer" {
		t.Fatalf("path parts=%v", parts)
	}
	secret, phone, ok := validate.ParsePortalAccessKey(parts[2])
	if !ok || phone != "6281234567890" || !validate.ValidPortalAccessSecret(secret) {
		t.Fatalf("access key=%q secret=%q phone=%q", parts[2], secret, phone)
	}
	if _, hasAccess := created["portal_access_url"]; hasAccess {
		t.Fatalf("portal_access_url should be omitted, got %v", created["portal_access_url"])
	}

	// Duplicate phone → conflict
	code, dup := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/customers", token, map[string]any{
		"name": "Other", "phone": "+62 812 3456 7890",
	})
	if code != 409 {
		t.Fatalf("duplicate phone want 409 got %d %v", code, dup)
	}

	// Email magic-link request is retired
	code, retired := doJSON(t, router, "POST", "/api/v1/portal/wa-shop/auth/magic-link/request", "", map[string]any{
		"email": "anyone@example.com",
	})
	if code != 410 {
		t.Fatalf("magic-link request want 410 got %d %v", code, retired)
	}

	// Durable access key opens portal session
	code, ver := doJSON(t, router, "POST", "/api/v1/portal/wa-shop/auth/access", "", map[string]any{
		"access_key": parts[2],
	})
	if code != 200 {
		t.Fatalf("access auth %d %v", code, ver)
	}
	sessionTok, _ := ver["session_token"].(string)
	if sessionTok == "" {
		t.Fatal("missing session_token")
	}
	code, sess := doJSON(t, router, "GET", "/api/v1/portal/wa-shop/auth/session", sessionTok, nil)
	if code != 200 {
		t.Fatalf("portal session %d %v", code, sess)
	}
	pcust := sess["customer"].(map[string]any)
	if pcust["id"] != cust["id"] {
		t.Fatalf("session customer %v want %v", pcust["id"], cust["id"])
	}

	// Wrong secret → gone
	code, bad := doJSON(t, router, "POST", "/api/v1/portal/wa-shop/auth/access", "", map[string]any{
		"access_key": "zzzz-" + phone,
	})
	if code != 410 {
		t.Fatalf("wrong secret want 410 got %d %v", code, bad)
	}

	// Rotate invalidates prior URL
	code, rotated := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/customers/"+cust["id"].(string)+"/portal-access/rotate", token, nil)
	if code != 200 {
		t.Fatalf("rotate %d %v", code, rotated)
	}
	newURL, _ := rotated["portal_url"].(string)
	if newURL == "" || newURL == portalURL {
		t.Fatalf("expected new portal_url, got %q (old %q)", newURL, portalURL)
	}
	code, stale := doJSON(t, router, "POST", "/api/v1/portal/wa-shop/auth/access", "", map[string]any{
		"access_key": parts[2],
	})
	if code != 410 {
		t.Fatalf("stale key after rotate want 410 got %d %v", code, stale)
	}
	nu, _ := url.Parse(newURL)
	nparts := strings.Split(strings.TrimPrefix(nu.Path, "/"), "/")
	code, ver2 := doJSON(t, router, "POST", "/api/v1/portal/wa-shop/auth/access", "", map[string]any{
		"access_key": nparts[2],
	})
	if code != 200 {
		t.Fatalf("new key auth %d %v", code, ver2)
	}

	// List includes phone-only customer
	code, list := doJSON(t, router, "GET", "/api/v1/stores/"+sid+"/customers", token, nil)
	if code != 200 {
		t.Fatalf("list %d %v", code, list)
	}
	rows, _ := list["customers"].([]any)
	if len(rows) != 1 {
		t.Fatalf("list want 1 got %d %v", len(rows), list)
	}
}
