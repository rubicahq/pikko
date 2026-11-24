package urls

import (
	"fmt"
	"net/url"
	"strings"
)

// Generator centralizes all absolute URL construction (PRD §7).
// Business logic stores relative paths / identifiers only.
type Generator struct {
	Marketing     string
	Dashboard     string
	StorefrontBase string
	API           string
}

func New(marketing, dashboard, storefront, api string) *Generator {
	return &Generator{
		Marketing:      strings.TrimRight(marketing, "/"),
		Dashboard:      strings.TrimRight(dashboard, "/"),
		StorefrontBase: strings.TrimRight(storefront, "/"),
		API:            strings.TrimRight(api, "/"),
	}
}

func (g *Generator) MarketingHome() string { return g.Marketing + "/" }

func (g *Generator) AuthEnter() string     { return g.Dashboard + "/auth" }
func (g *Generator) AuthCheckInbox() string { return g.Dashboard + "/auth/check-inbox" }
func (g *Generator) AuthConfirm(token string) string {
	return g.Dashboard + "/auth/confirm?token=" + url.QueryEscape(token)
}
func (g *Generator) AuthInvalid() string { return g.Dashboard + "/auth/invalid" }

func (g *Generator) Onboarding() string { return g.Dashboard + "/onboarding" }
func (g *Generator) ManageStores() string { return g.Dashboard + "/stores" }
func (g *Generator) NewStore() string { return g.Dashboard + "/stores/new" }
func (g *Generator) AccountSettings() string { return g.Dashboard + "/account/settings" }

func (g *Generator) StoreOverview(storeID string) string {
	return fmt.Sprintf("%s/stores/%s/overview", g.Dashboard, storeID)
}

func (g *Generator) Storefront(slug string) string {
	return fmt.Sprintf("%s/%s", g.StorefrontBase, slug)
}

func (g *Generator) Product(storeSlug, productSlug string) string {
	return fmt.Sprintf("%s/%s/products/%s", g.StorefrontBase, storeSlug, productSlug)
}

func (g *Generator) Collection(storeSlug, collectionSlug string) string {
	return fmt.Sprintf("%s/%s/collections/%s", g.StorefrontBase, storeSlug, collectionSlug)
}

func (g *Generator) Preorder(storeSlug, batchSlug string) string {
	return fmt.Sprintf("%s/%s/preorders/%s", g.StorefrontBase, storeSlug, batchSlug)
}

func (g *Generator) Checkout(storeSlug string) string {
	return fmt.Sprintf("%s/%s/checkout", g.StorefrontBase, storeSlug)
}

func (g *Generator) Tracking(token string) string {
	return fmt.Sprintf("%s/track/%s", g.StorefrontBase, url.PathEscape(token))
}

func (g *Generator) PortalHome(storeSlug string) string {
	return fmt.Sprintf("%s/portal/%s", g.StorefrontBase, url.PathEscape(storeSlug))
}

func (g *Generator) PortalConfirm(storeSlug, token string) string {
	return fmt.Sprintf("%s/portal/%s/confirm?token=%s", g.StorefrontBase, url.PathEscape(storeSlug), url.QueryEscape(token))
}

func (g *Generator) PortalDashboard(storeSlug string) string {
	return fmt.Sprintf("%s/portal/%s/dashboard", g.StorefrontBase, url.PathEscape(storeSlug))
}

func (g *Generator) MagicLink(token string) string {
	return g.AuthConfirm(token)
}

func (g *Generator) TeamInvite(token string) string {
	return g.Dashboard + "/invites/accept?token=" + url.QueryEscape(token)
}

func (g *Generator) APICallback(path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return g.API + path
}

func (g *Generator) Canonical(path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return g.Marketing + path
}

// SafeRedirect returns redirect if its host is allowlisted; otherwise dashboard home.
func SafeRedirect(redirect string, allowlist []string, fallback string) string {
	if redirect == "" {
		return fallback
	}
	u, err := url.Parse(redirect)
	if err != nil || u.Host == "" {
		if strings.HasPrefix(redirect, "/") {
			return fallback // relative redirects resolved by caller
		}
		return fallback
	}
	host := strings.ToLower(u.Hostname())
	for _, allowed := range allowlist {
		allowed = strings.ToLower(strings.TrimSpace(allowed))
		if host == allowed || strings.HasSuffix(host, "."+allowed) {
			return redirect
		}
	}
	return fallback
}
