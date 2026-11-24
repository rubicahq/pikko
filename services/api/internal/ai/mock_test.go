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
	if res.Items[0].ConfidencePct == nil {
		t.Fatal("expected confidence")
	}
}

func TestNewExtractorWithoutKeyUsesMock(t *testing.T) {
	ex := ai.NewExtractor("", "https://example.invalid", "model")
	if _, ok := ex.(*ai.MockExtractor); !ok {
		t.Fatalf("expected MockExtractor, got %T", ex)
	}
	if ai.ProviderName(ex) != "mock" {
		t.Fatalf("provider=%s", ai.ProviderName(ex))
	}
}

func TestNewExtractorMissingModelUsesMock(t *testing.T) {
	ex := ai.NewExtractor("sk-test", "https://example.invalid", "")
	if _, ok := ex.(*ai.MockExtractor); !ok {
		t.Fatalf("expected MockExtractor, got %T", ex)
	}
}

func TestNormalizeConfidence(t *testing.T) {
	it := ai.ExtractedItem{Uncertain: true}
	it.NormalizeConfidence()
	if it.ConfidencePct == nil || *it.ConfidencePct != 55 {
		t.Fatalf("uncertain default=%v", it.ConfidencePct)
	}
	high := 150.0
	it2 := ai.ExtractedItem{ConfidencePct: &high}
	it2.NormalizeConfidence()
	if *it2.ConfidencePct != 100 {
		t.Fatalf("clamp high=%v", *it2.ConfidencePct)
	}
	low := 40.0
	it3 := ai.ExtractedItem{ConfidencePct: &low}
	it3.NormalizeConfidence()
	if !it3.Uncertain {
		t.Fatal("low confidence should mark uncertain")
	}
}
