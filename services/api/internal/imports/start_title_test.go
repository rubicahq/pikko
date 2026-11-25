package imports

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestStartRequiresCatalogTitle(t *testing.T) {
	s := &Service{}
	_, err := s.Start(context.Background(), uuid.New(), uuid.New(), StartInput{
		CatalogTitle: "   ",
		Size:         10,
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
	if !strings.Contains(err.Error(), "catalog name is required") {
		t.Fatalf("message=%v", err)
	}
}

func TestStartRejectsOverlongCatalogTitle(t *testing.T) {
	s := &Service{}
	_, err := s.Start(context.Background(), uuid.New(), uuid.New(), StartInput{
		CatalogTitle: strings.Repeat("a", 201),
		Size:         10,
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
	if !strings.Contains(err.Error(), "200 characters") {
		t.Fatalf("message=%v", err)
	}
}
