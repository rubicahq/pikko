// Package rbac defines store-scoped roles and granular permissions (PRD §2.2 / §24).
package rbac

// Role values match store_memberships.role / team_invites.role.
const (
	RoleOwner      = "owner"
	RoleAdmin      = "admin"
	RoleCS         = "cs"
	RoleCatalog    = "catalog"
	RolePurchasing = "purchasing"
	RolePacking    = "packing"
	RoleFinance    = "finance"
)

// Permission is a granular capability checked server-side.
type Permission string

const (
	PermStoresRead     Permission = "stores.read"
	PermStoresWrite    Permission = "stores.write"
	PermStoresArchive  Permission = "stores.archive"
	PermStoresSettings Permission = "stores.settings"

	PermTeamRead   Permission = "team.read"
	PermTeamInvite Permission = "team.invite"
	PermTeamManage Permission = "team.manage"

	PermCatalogRead  Permission = "catalog.read"
	PermCatalogWrite Permission = "catalog.write"

	PermCampaignRead  Permission = "campaign.read"
	PermCampaignWrite Permission = "campaign.write"

	PermStorefrontRead  Permission = "storefront.read"
	PermStorefrontWrite Permission = "storefront.write"

	PermOrdersRead     Permission = "orders.read"
	PermOrdersWrite    Permission = "orders.write"
	PermOrdersPayments Permission = "orders.payments"

	PermCustomersRead  Permission = "customers.read"
	PermCustomersWrite Permission = "customers.write"

	PermPurchasingRead  Permission = "purchasing.read"
	PermPurchasingWrite Permission = "purchasing.write"

	PermPackingRead  Permission = "packing.read"
	PermPackingWrite Permission = "packing.write"

	PermFinanceRead  Permission = "finance.read"
	PermFinanceWrite Permission = "finance.write"

	PermBusinessRulesRead  Permission = "business_rules.read"
	PermBusinessRulesWrite Permission = "business_rules.write"

	PermAnalyticsRead      Permission = "analytics.read"
	PermNotificationsRead  Permission = "notifications.read"
	PermNotificationsWrite Permission = "notifications.write"
)

var allPermissions = []Permission{
	PermStoresRead, PermStoresWrite, PermStoresArchive, PermStoresSettings,
	PermTeamRead, PermTeamInvite, PermTeamManage,
	PermCatalogRead, PermCatalogWrite,
	PermCampaignRead, PermCampaignWrite,
	PermStorefrontRead, PermStorefrontWrite,
	PermOrdersRead, PermOrdersWrite, PermOrdersPayments,
	PermCustomersRead, PermCustomersWrite,
	PermPurchasingRead, PermPurchasingWrite,
	PermPackingRead, PermPackingWrite,
	PermFinanceRead, PermFinanceWrite,
	PermBusinessRulesRead, PermBusinessRulesWrite,
	PermAnalyticsRead, PermNotificationsRead, PermNotificationsWrite,
}

// roleMatrix maps role → granted permissions. Owner is the full set.
var roleMatrix = map[string][]Permission{
	RoleOwner: append([]Permission(nil), allPermissions...),
	RoleAdmin: {
		PermStoresRead, PermStoresWrite, PermStoresSettings,
		PermTeamRead, PermTeamInvite, PermTeamManage,
		PermCatalogRead, PermCatalogWrite,
		PermCampaignRead, PermCampaignWrite,
		PermStorefrontRead, PermStorefrontWrite,
		PermOrdersRead, PermOrdersWrite, PermOrdersPayments,
		PermCustomersRead, PermCustomersWrite,
		PermPurchasingRead, PermPurchasingWrite,
		PermPackingRead, PermPackingWrite,
		PermFinanceRead, PermFinanceWrite,
		PermBusinessRulesRead, PermBusinessRulesWrite,
		PermAnalyticsRead, PermNotificationsRead, PermNotificationsWrite,
	},
	RoleCS: {
		PermStoresRead,
		PermCatalogRead,
		PermCampaignRead,
		PermOrdersRead, PermOrdersWrite, PermOrdersPayments,
		PermCustomersRead, PermCustomersWrite,
		PermAnalyticsRead,
		PermNotificationsRead, PermNotificationsWrite,
	},
	RoleCatalog: {
		PermStoresRead,
		PermCatalogRead, PermCatalogWrite,
		PermCampaignRead, PermCampaignWrite,
		PermStorefrontRead, PermStorefrontWrite,
		PermNotificationsRead,
	},
	RolePurchasing: {
		PermStoresRead,
		PermCatalogRead,
		PermCampaignRead,
		PermOrdersRead,
		PermPurchasingRead, PermPurchasingWrite,
		PermNotificationsRead,
	},
	RolePacking: {
		PermStoresRead,
		PermOrdersRead,
		PermPackingRead, PermPackingWrite,
		PermCustomersRead,
		PermNotificationsRead,
	},
	RoleFinance: {
		PermStoresRead,
		PermOrdersRead, PermOrdersPayments,
		PermCustomersRead,
		PermFinanceRead, PermFinanceWrite,
		PermAnalyticsRead,
		PermNotificationsRead,
	},
}

// ValidMembershipRole reports whether role is allowed on store_memberships.
func ValidMembershipRole(role string) bool {
	_, ok := roleMatrix[role]
	return ok
}

// ValidInviteRole reports whether role may be assigned via invite (not owner).
func ValidInviteRole(role string) bool {
	return role != RoleOwner && ValidMembershipRole(role)
}

// HasPermission returns true when role grants perm.
func HasPermission(role string, perm Permission) bool {
	for _, p := range PermissionsFor(role) {
		if p == perm {
			return true
		}
	}
	return false
}

// PermissionsFor returns the permission list for a role (nil if unknown).
func PermissionsFor(role string) []Permission {
	perms, ok := roleMatrix[role]
	if !ok {
		return nil
	}
	out := make([]Permission, len(perms))
	copy(out, perms)
	return out
}

// AllRoles returns default PRD roles in stable order.
func AllRoles() []string {
	return []string{
		RoleOwner, RoleAdmin, RoleCS, RoleCatalog,
		RolePurchasing, RolePacking, RoleFinance,
	}
}

// InviteRoles returns roles that can be invited.
func InviteRoles() []string {
	return []string{
		RoleAdmin, RoleCS, RoleCatalog,
		RolePurchasing, RolePacking, RoleFinance,
	}
}

// NavPermission maps merchant nav href → required read permission.
// Items without a mapping stay visible to any member.
func NavPermission(href string) (Permission, bool) {
	switch href {
	case "overview":
		return PermStoresRead, true
	case "notifications":
		return PermNotificationsRead, true
	case "analytics":
		return PermAnalyticsRead, true
	case "orders":
		return PermOrdersRead, true
	case "customers":
		return PermCustomersRead, true
	case "purchasing":
		return PermPurchasingRead, true
	case "packing":
		return PermPackingRead, true
	case "shipments":
		return PermPackingRead, true
	case "finance":
		return PermFinanceRead, true
	case "catalog":
		return PermCatalogRead, true
	case "preorders":
		return PermCampaignRead, true
	case "campaigns":
		return PermCampaignRead, true
	case "storefront":
		return PermStorefrontRead, true
	case "business-rules":
		return PermBusinessRulesRead, true
	case "settings":
		return PermStoresSettings, true
	case "vendors":
		return PermPurchasingRead, true
	case "team":
		return PermTeamRead, true
	default:
		return "", false
	}
}
