package rbac_test

import (
	"testing"

	"github.com/rubicahq/pikko/services/api/internal/rbac"
)

func TestOwnerIsSuperset(t *testing.T) {
	owner := rbac.PermissionsFor(rbac.RoleOwner)
	if len(owner) == 0 {
		t.Fatal("owner has no permissions")
	}
	for _, role := range rbac.AllRoles() {
		if role == rbac.RoleOwner {
			continue
		}
		for _, p := range rbac.PermissionsFor(role) {
			if !rbac.HasPermission(rbac.RoleOwner, p) {
				t.Fatalf("owner missing %s from role %s", p, role)
			}
		}
	}
}

func TestAdminCannotArchive(t *testing.T) {
	if rbac.HasPermission(rbac.RoleAdmin, rbac.PermStoresArchive) {
		t.Fatal("admin must not archive (ownership boundary)")
	}
	if !rbac.HasPermission(rbac.RoleOwner, rbac.PermStoresArchive) {
		t.Fatal("owner must archive")
	}
}

func TestCatalogCannotManageTeam(t *testing.T) {
	if rbac.HasPermission(rbac.RoleCatalog, rbac.PermTeamInvite) {
		t.Fatal("catalog must not invite")
	}
	if !rbac.HasPermission(rbac.RoleCatalog, rbac.PermCatalogWrite) {
		t.Fatal("catalog must write catalog")
	}
}

func TestInviteRolesExcludeOwner(t *testing.T) {
	for _, r := range rbac.InviteRoles() {
		if r == rbac.RoleOwner {
			t.Fatal("owner not inviteable")
		}
		if !rbac.ValidInviteRole(r) {
			t.Fatalf("expected inviteable %s", r)
		}
	}
	if rbac.ValidInviteRole(rbac.RoleOwner) {
		t.Fatal("owner must not be inviteable")
	}
}

func TestCSCanVerifyPayments(t *testing.T) {
	if !rbac.HasPermission(rbac.RoleCS, rbac.PermOrdersPayments) {
		t.Fatal("cs needs payment verification")
	}
}

func TestCampaignPermissions(t *testing.T) {
	if !rbac.HasPermission(rbac.RoleCatalog, rbac.PermCampaignWrite) {
		t.Fatal("catalog role needs campaign.write")
	}
	if !rbac.HasPermission(rbac.RoleCS, rbac.PermCampaignRead) {
		t.Fatal("cs needs campaign.read")
	}
	if rbac.HasPermission(rbac.RoleCS, rbac.PermCampaignWrite) {
		t.Fatal("cs must not write campaigns")
	}
	perm, ok := rbac.NavPermission("campaigns")
	if !ok || perm != rbac.PermCampaignRead {
		t.Fatalf("nav campaigns -> %s ok=%v", perm, ok)
	}
	perm, ok = rbac.NavPermission("preorders")
	if !ok || perm != rbac.PermCampaignRead {
		t.Fatalf("nav preorders -> %s ok=%v", perm, ok)
	}
}
