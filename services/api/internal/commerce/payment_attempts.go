package commerce

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// GetOrderByIdentifier resolves either the Pikko order UUID or exact order
// number, always within the authenticated store scope.
func (s *Service) GetOrderByIdentifier(ctx context.Context, storeID uuid.UUID, identifier string) (*Order, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return nil, ErrInvalidInput
	}
	if id, err := uuid.Parse(identifier); err == nil {
		return s.GetOrder(ctx, storeID, id)
	}
	var id uuid.UUID
	err := s.DB.QueryRow(ctx, `
		SELECT id FROM orders WHERE store_id=$1 AND lower(order_number)=lower($2)
	`, storeID, identifier).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.GetOrder(ctx, storeID, id)
}

// GetLatestPaymentAttempt reads the most recent provider attempt for an order.
// It intentionally performs no status update or reconciliation.
func (s *Service) GetLatestPaymentAttempt(ctx context.Context, storeID, orderID uuid.UUID) (*PaymentAttempt, error) {
	var a PaymentAttempt
	err := s.DB.QueryRow(ctx, `
		SELECT id, store_id, order_id, payment_id, provider,
		       provider_invoice_number, amount_idr, status, created_at, updated_at,
		       request_id, checkout_url
		FROM payment_attempts
		WHERE store_id=$1 AND order_id=$2
		ORDER BY created_at DESC LIMIT 1
	`, storeID, orderID).Scan(
		&a.ID, &a.StoreID, &a.OrderID, &a.PaymentID, &a.Provider,
		&a.ProviderInvoiceNumber, &a.AmountIDR, &a.Status, &a.CreatedAt, &a.UpdatedAt,
		&a.RequestID, &a.CheckoutURL,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}
