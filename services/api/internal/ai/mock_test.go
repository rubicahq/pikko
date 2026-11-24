package ai_test

import (
	"context"
	"testing"

	"github.com/rubicahq/pikko/services/api/internal/ai"
)

func TestMockExtractor(t *testing.T) {
	m := ai.NewMockExtractor()
	res, err := m.ExtractCatalog(context.Background(), ai.ExtractRequest{
		Pages:    []ai.PageImage{{PageNumber: 1, MIMEType: "image/png", Base64Data: "xx"}},
		MaxItems: 500,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Items) < 1 {
		t.Fatal("expected items")
	}
	if res.Items[0].Name == nil || *res.Items[0].Name == "" {
		t.Fatal("expected name")
	}
}

func TestNewExtractorWithoutKeyUsesMock(t *testing.T) {
	ex := ai.NewExtractor("", "https://example.invalid", "model")
	if _, ok := ex.(*ai.MockExtractor); !ok {
		t.Fatalf("expected MockExtractor, got %T", ex)
	}
}
