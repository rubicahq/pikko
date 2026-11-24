package portal

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rubicahq/pikko/services/api/internal/commerce"
	"github.com/rubicahq/pikko/services/api/internal/config"
	"github.com/rubicahq/pikko/services/api/internal/email"
	"github.com/rubicahq/pikko/services/api/internal/urls"
)

var (
	ErrInvalidEmail    = errors.New("invalid email")
	ErrRateLimited     = errors.New("rate limited")
	ErrCooldown        = errors.New("resend cooldown")
	ErrInvalidToken    = errors.New("invalid or expired token")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrNotFound        = errors.New("not found")
	ErrInvalidInput    = errors.New("invalid input")
	ErrConflict        = errors.New("conflict")
	ErrForbiddenState  = errors.New("forbidden state")
	ErrStoreCreditOff  = errors.New("store credit disabled")
)

type Service struct {
	DB       *pgxpool.Pool
	Cfg      *config.Config
	Email    *email.Service
	URLs     *urls.Generator
	Logger   *slog.Logger
	Commerce *commerce.Service
}

func (s *Service) now() time.Time {
	return time.Now().UTC()
}

func (s *Service) resolveStoreBySlug(ctx context.Context, slug string) (*StoreMeta, error) {
	slug = strings.TrimSpace(strings.ToLower(slug))
	if slug == "" {
		return nil, ErrNotFound
	}
	var st StoreMeta
	var primary *string
	err := s.DB.QueryRow(ctx, `
		SELECT s.id, s.name, s.slug, COALESCE(sc.primary_color, '#FF3D00')
		FROM stores s
		LEFT JOIN storefront_configs sc ON sc.store_id=s.id AND sc.channel='published'
		WHERE s.slug=$1 AND s.archived_at IS NULL
	`, slug).Scan(&st.ID, &st.Name, &st.Slug, &primary)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if primary != nil {
		st.PrimaryColor = *primary
	} else {
		st.PrimaryColor = "#FF3D00"
	}
	return &st, nil
}

func (s *Service) loadCustomer(ctx context.Context, storeID, customerID uuid.UUID) (*Customer, error) {
	var c Customer
	err := s.DB.QueryRow(ctx, `
		SELECT id, store_id, name, email, phone, total_orders, created_at
		FROM customers WHERE id=$1 AND store_id=$2
	`, customerID, storeID).Scan(
		&c.ID, &c.StoreID, &c.Name, &c.Email, &c.Phone, &c.TotalOrders, &c.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Service) findCustomerByEmail(ctx context.Context, storeID uuid.UUID, emailNormalized string) (*Customer, error) {
	var c Customer
	err := s.DB.QueryRow(ctx, `
		SELECT id, store_id, name, email, phone, total_orders, created_at
		FROM customers WHERE store_id=$1 AND lower(email)=lower($2)
	`, storeID, emailNormalized).Scan(
		&c.ID, &c.StoreID, &c.Name, &c.Email, &c.Phone, &c.TotalOrders, &c.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func nullStr(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}

func trimPtr(p *string) *string {
	if p == nil {
		return nil
	}
	v := strings.TrimSpace(*p)
	if v == "" {
		return nil
	}
	return &v
}
