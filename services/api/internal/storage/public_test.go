package storage

import "testing"

func TestIsPublicPurpose(t *testing.T) {
	if !IsPublicPurpose("product") || !IsPublicPurpose("logo") {
		t.Fatal("expected public branding/product purposes")
	}
	if IsPublicPurpose("payment_evidence") || IsPublicPurpose("catalog_pdf") || IsPublicPurpose("catalog_csv") {
		t.Fatal("private purposes must not be public")
	}
	if !IsPrivateUploadPurpose("payment_evidence") {
		t.Fatal("payment_evidence is private upload")
	}
	if IsPrivateUploadPurpose("product") {
		t.Fatal("product is allowed on catalog upload")
	}
}
