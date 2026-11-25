package pdfutil_test

import (
	"os"
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

func TestUnavailableRendererFails(t *testing.T) {
	r := &pdfutil.UnavailableRenderer{}
	if _, err := r.RenderPages(pdfutil.MinimalPDF(), 1); err == nil {
		t.Fatal("expected error when poppler missing")
	}
}

func TestPopplerRendererFixturePage1(t *testing.T) {
	r := pdfutil.NewRenderer()
	if _, ok := r.(*pdfutil.PopplerRenderer); !ok {
		t.Skip("pdftoppm not installed")
	}
	pdfPath := "/cursor/stores/bc-93b8b41a-6d35-469a-ba5a-fcef7c20c2ee/media/catalog-fixtures/jastip-imbex-jakarta-nov.pdf"
	data, err := os.ReadFile(pdfPath)
	if err != nil {
		t.Skip("fixture PDF not available:", err)
	}
	pages, err := r.RenderPages(data, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 1 {
		t.Fatalf("pages=%d want 1", len(pages))
	}
	if pages[0].Width < 1000 || pages[0].Height < 1000 {
		t.Fatalf("unexpected size %dx%d — blank placeholder?", pages[0].Width, pages[0].Height)
	}
	// scale-to 2200 → longest edge ~2200
	long := pages[0].Width
	if pages[0].Height > long {
		long = pages[0].Height
	}
	if long < 2000 || long > 2400 {
		t.Fatalf("expected ~2200 long edge, got %d", long)
	}
}

func TestRejectNonPDF(t *testing.T) {
	if pdfutil.IsPDF([]byte("not a pdf")) {
		t.Fatal("should reject")
	}
}
