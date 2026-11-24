package pdfutil_test

import (
	"testing"

	"github.com/rubicahq/pikko/services/api/internal/pdfutil"
)

func TestIsPDFAndCountPages(t *testing.T) {
	pdf := pdfutil.MinimalPDF()
	if !pdfutil.IsPDF(pdf) {
		t.Fatal("expected PDF magic")
	}
	n, err := pdfutil.CountPages(pdf)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("pages=%d want 1", n)
	}
}

func TestPlaceholderRenderer(t *testing.T) {
	r := &pdfutil.PlaceholderRenderer{Width: 100, Height: 120}
	pages, err := r.RenderPages(pdfutil.MinimalPDF(), 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 1 {
		t.Fatalf("got %d pages", len(pages))
	}
	if len(pages[0].PNG) < 50 {
		t.Fatal("expected PNG bytes")
	}
}

func TestRejectNonPDF(t *testing.T) {
	if pdfutil.IsPDF([]byte("not a pdf")) {
		t.Fatal("should reject")
	}
}
