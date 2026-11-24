package commerce

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rubicahq/pikko/services/api/internal/auth"
)

func (s *Service) resolveTracking(ctx context.Context, rawToken string) (*TrackingView, uuid.UUID, uuid.UUID, error) {
	if rawToken == "" {
		return nil, uuid.Nil, uuid.Nil, ErrNotFound
	}
	hash := auth.HashToken(rawToken)
	var storeID, orderID uuid.UUID
	var revoked *time.Time
	var expires *time.Time
	err := s.DB.QueryRow(ctx, `
		SELECT store_id, order_id, revoked_at, expires_at
		FROM order_tracking_tokens WHERE token_hash=$1
	`, hash).Scan(&storeID, &orderID, &revoked, &expires)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, uuid.Nil, uuid.Nil, ErrNotFound
	}
	if err != nil {
		return nil, uuid.Nil, uuid.Nil, err
	}
	if revoked != nil {
		return nil, uuid.Nil, uuid.Nil, ErrNotFound
	}
	if expires != nil && expires.Before(s.now()) {
		return nil, uuid.Nil, uuid.Nil, ErrNotFound
	}
	view, err := s.buildTrackingView(ctx, storeID, orderID)
	if err != nil {
		return nil, uuid.Nil, uuid.Nil, err
	}
	return view, orderID, storeID, nil
}

func (s *Service) GetTracking(ctx context.Context, rawToken string) (*TrackingView, error) {
	view, _, _, err := s.resolveTracking(ctx, rawToken)
	return view, err
}

func (s *Service) buildTrackingView(ctx context.Context, storeID, orderID uuid.UUID) (*TrackingView, error) {
	var storeName, storeSlug string
	var primary *string
	var contactEmail, whatsapp *string
	err := s.DB.QueryRow(ctx, `
		SELECT s.name, s.slug,
		       COALESCE(sc.primary_color, '#FF3D00'),
		       sc.contact_email, sc.whatsapp
		FROM stores s
		LEFT JOIN storefront_configs sc ON sc.store_id=s.id AND sc.channel='published'
		WHERE s.id=$1
	`, storeID).Scan(&storeName, &storeSlug, &primary, &contactEmail, &whatsapp)
	if err != nil {
		return nil, err
	}
	o, err := s.GetOrder(ctx, storeID, orderID)
	if err != nil {
		return nil, err
	}
	// Never expose internal notes on tracking
	o.InternalNotes = nil
	pay, _ := s.GetLatestPayment(ctx, storeID, orderID)
	shipments, _ := s.ListOrderShipments(ctx, storeID, orderID)
	canSubmit := o.OrderStatus != OrderCancelled && o.PaymentStatus != PaymentPaid
	primaryColor := "#FF3D00"
	if primary != nil {
		primaryColor = *primary
	}
	return &TrackingView{
		StoreName: storeName, StoreSlug: storeSlug, PrimaryColor: primaryColor,
		ContactEmail: contactEmail, WhatsApp: whatsapp,
		Order: *o, Payment: pay, Shipments: shipments, CanSubmitEvidence: canSubmit,
	}, nil
}

// EnsureTrackingToken mints a new active tracking token (previous tokens remain valid unless revoked).
func (s *Service) EnsureTrackingToken(ctx context.Context, storeID, orderID uuid.UUID) (string, error) {
	raw, hash, err := auth.NewToken()
	if err != nil {
		return "", err
	}
	_, err = s.DB.Exec(ctx, `
		INSERT INTO order_tracking_tokens (store_id, order_id, token_hash)
		VALUES ($1,$2,$3)
	`, storeID, orderID, hash)
	if err != nil {
		return "", err
	}
	return raw, nil
}

func (s *Service) OpenEvidenceStream(ctx context.Context, storeID, evidenceID uuid.UUID) (contentType string, key string, err error) {
	err = s.DB.QueryRow(ctx, `
		SELECT f.content_type, f.storage_key
		FROM payment_evidence e
		JOIN stored_files f ON f.id=e.file_id AND f.store_id=e.store_id
		WHERE e.id=$1 AND e.store_id=$2 AND f.purpose='payment_evidence'
	`, evidenceID, storeID).Scan(&contentType, &key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrNotFound
	}
	return contentType, key, err
}

func (s *Service) OpenEvidenceStreamForTracking(ctx context.Context, rawToken string, evidenceID uuid.UUID) (contentType, key string, err error) {
	_, orderID, storeID, err := s.resolveTracking(ctx, rawToken)
	if err != nil {
		return "", "", err
	}
	err = s.DB.QueryRow(ctx, `
		SELECT f.content_type, f.storage_key
		FROM payment_evidence e
		JOIN stored_files f ON f.id=e.file_id AND f.store_id=e.store_id
		WHERE e.id=$1 AND e.store_id=$2 AND e.order_id=$3 AND f.purpose='payment_evidence'
	`, evidenceID, storeID, orderID).Scan(&contentType, &key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrNotFound
	}
	return contentType, key, err
}
