package portal_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/rubicahq/pikko/services/api/internal/httpapi"
	"github.com/rubicahq/pikko/services/api/internal/storage"
	"github.com/rubicahq/pikko/services/api/internal/stores"
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
	accessURL, _ := created["portal_access_url"].(string)
	if !strings.Contains(portalURL, "/portal/wa-shop") {
		t.Fatalf("portal_url=%q", portalURL)
	}
	if !strings.Contains(accessURL, "/portal/wa-shop/confirm?token=") {
		t.Fatalf("portal_access_url=%q", accessURL)
	}

	// Duplicate phone → conflict
	code, dup := doJSON(t, router, "POST", "/api/v1/stores/"+sid+"/customers", token, map[string]any{
		"name": "Other", "phone": "+62 812 3456 7890",
	})
	if code != 409 {
		t.Fatalf("duplicate phone want 409 got %d %v", code, dup)
	}

	// One-time access link opens portal session without email
	u, err := url.Parse(accessURL)
	if err != nil {
		t.Fatal(err)
	}
	rawToken := u.Query().Get("token")
	if rawToken == "" {
		t.Fatal("missing token in access url")
	}
	code, ver := doJSON(t, router, "POST", "/api/v1/portal/wa-shop/auth/magic-link/verify", "", map[string]any{
		"token": rawToken,
	})
	if code != 200 {
		t.Fatalf("verify access link %d %v", code, ver)
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

	// Token is single-use
	code, reuse := doJSON(t, router, "POST", "/api/v1/portal/wa-shop/auth/magic-link/verify", "", map[string]any{
		"token": rawToken,
	})
	if code != http.StatusGone {
		t.Fatalf("reuse want 410 got %d %v", code, reuse)
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
