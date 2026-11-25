package portal

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rubicahq/pikko/services/api/internal/auth"
	"github.com/rubicahq/pikko/services/api/internal/commerce"
	"github.com/rubicahq/pikko/services/api/internal/config"
	"github.com/rubicahq/pikko/services/api/internal/email"
	"github.com/rubicahq/pikko/services/api/internal/urls"
	"github.com/rubicahq/pikko/services/api/internal/validate"
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

func (s *Service) resolveStoreByID(ctx context.Context, storeID uuid.UUID) (*StoreMeta, error) {
	var st StoreMeta
	var primary *string
	err := s.DB.QueryRow(ctx, `
		SELECT s.id, s.name, s.slug, COALESCE(sc.primary_color, '#FF3D00')
		FROM stores s
		LEFT JOIN storefront_configs sc ON sc.store_id=s.id AND sc.channel='published'
		WHERE s.id=$1 AND s.archived_at IS NULL
	`, storeID).Scan(&st.ID, &st.Name, &st.Slug, &primary)
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
		FROM customers WHERE store_id=$1 AND email IS NOT NULL AND lower(email)=lower($2)
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

// IssueAccessLink mints a one-time portal confirm URL for a store customer (merchant-issued).
// Works for phone-only customers (no email) by binding the token to customer_id.
func (s *Service) IssueAccessLink(ctx context.Context, storeID, customerID uuid.UUID) (portalURL, accessURL string, err error) {
	st, err := s.resolveStoreByID(ctx, storeID)
	if err != nil {
		return "", "", err
	}
	customer, err := s.loadCustomer(ctx, storeID, customerID)
	if err != nil {
		return "", "", err
	}

	raw, hash, err := auth.NewToken()
	if err != nil {
		return "", "", err
	}
	expires := s.now().Add(s.Cfg.MagicLinkTTL)
	emailKey := "phone:" + derefStr(customer.Phone)
	if customer.Email != nil && strings.TrimSpace(*customer.Email) != "" {
		_, normalized, ok := validate.NormalizeEmail(*customer.Email)
		if ok {
			emailKey = normalized
		}
	}
	_, err = s.DB.Exec(ctx, `
		INSERT INTO customer_portal_tokens (store_id, email_normalized, customer_id, token_hash, expires_at)
		VALUES ($1,$2,$3,$4,$5)
	`, storeID, emailKey, customerID, hash, expires)
	if err != nil {
		return "", "", err
	}
	if s.URLs == nil {
		return "", "", fmt.Errorf("urls not configured")
	}
	return s.URLs.PortalHome(st.Slug), s.URLs.PortalConfirm(st.Slug, raw), nil
}

func nullStr(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
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
