package storage

// PublicPurposes are the only stored_files purposes that may be served via
// unauthenticated GET /api/v1/public/files/{storeID}/{fileID} (SEC-04).
// Private purposes (payment_evidence, catalog_pdf, catalog_csv, …) must use
// authz-gated merchant or tracking-scoped routes.
var PublicPurposes = map[string]struct{}{
	"logo":           {},
	"favicon":        {},
	"store_icon":     {},
	"product":        {},
	"banner":         {},
	"collection":     {},
	"other":          {},
	"catalog_page":   {},
	"preorder_cover": {},
	"import_crop":    {},
}

// IsPublicPurpose reports whether purpose is allowed on the public file route.
func IsPublicPurpose(purpose string) bool {
	_, ok := PublicPurposes[purpose]
	return ok
}

// IsPrivateUploadPurpose is rejected on the generic catalog file upload route
// (must go through dedicated evidence / import paths with tighter RBAC).
func IsPrivateUploadPurpose(purpose string) bool {
	switch purpose {
	case "payment_evidence", "refund_evidence":
		return true
	default:
		return false
	}
}
