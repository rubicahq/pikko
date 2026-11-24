package commerce

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rubicahq/pikko/services/api/internal/config"
	"github.com/rubicahq/pikko/services/api/internal/email"
	"github.com/rubicahq/pikko/services/api/internal/files"
	"github.com/rubicahq/pikko/services/api/internal/jobs"
	"github.com/rubicahq/pikko/services/api/internal/storage"
	"github.com/rubicahq/pikko/services/api/internal/urls"
)

type Service struct {
	DB       *pgxpool.Pool
	Cfg      *config.Config
	Files    *files.Service
	Storage  storage.ObjectStore
	Email    *email.Service
	Outbox   *jobs.Outbox
	URLs     *urls.Generator
	Logger   *slog.Logger
	Notifier Notifier
}

func (s *Service) refreshCheckoutEnabled(ctx context.Context, storeID uuid.UUID) error {
	var hasPayment bool
	err := s.DB.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM store_payment_methods WHERE store_id=$1 AND is_enabled=true
		)
	`, storeID).Scan(&hasPayment)
	if err != nil {
		return err
	}
	var ship, freeShip, pickup bool
	err = s.DB.QueryRow(ctx, `
		SELECT COALESCE(shipping_enabled,false), COALESCE(free_shipping_enabled,false), COALESCE(pickup_enabled,false)
		FROM store_fulfillment_settings WHERE store_id=$1
	`, storeID).Scan(&ship, &freeShip, &pickup)
	if err != nil {
		// no row → fulfillment not configured
		ship, freeShip, pickup = false, false, false
	}
	hasFulfillment := ship || freeShip || pickup
	enabled := hasPayment && hasFulfillment
	_, err = s.DB.Exec(ctx, `
		UPDATE stores SET checkout_enabled=$2, updated_at=now() WHERE id=$1
	`, storeID, enabled)
	return err
}

func normalizeEmail(emailAddr string) string {
	return strings.ToLower(strings.TrimSpace(emailAddr))
}

func ptrStr(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
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

func (s *Service) now() time.Time {
	return time.Now().UTC()
}
