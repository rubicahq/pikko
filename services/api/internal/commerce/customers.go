package commerce

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rubicahq/pikko/services/api/internal/validate"
)

type CustomerDetail struct {
	Customer
	Orders []Order `json:"orders"`
}

type CreateCustomerInput struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

type CreateCustomerResult struct {
	Customer  Customer `json:"customer"`
	PortalURL string   `json:"portal_url"`
	StoreSlug string   `json:"store_slug"`
}

func (s *Service) GetCustomer(ctx context.Context, storeID, customerID uuid.UUID) (*CustomerDetail, error) {
	var c Customer
	err := s.DB.QueryRow(ctx, `
		SELECT id, store_id, name, email, phone, first_order_at, last_order_at,
		       total_orders, completed_order_value_idr, internal_notes, created_at, updated_at
		FROM customers WHERE id=$1 AND store_id=$2
	`, customerID, storeID).Scan(
		&c.ID, &c.StoreID, &c.Name, &c.Email, &c.Phone, &c.FirstOrderAt, &c.LastOrderAt,
		&c.TotalOrders, &c.CompletedOrderValueIDR, &c.InternalNotes, &c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	orders, _, err := s.ListOrders(ctx, storeID, ListOrdersFilter{
		CustomerID: &customerID,
		Limit:      50,
		Offset:     0,
	})
	if err != nil {
		return nil, err
	}
	return &CustomerDetail{Customer: c, Orders: orders}, nil
}

// CreateCustomer inserts a store-scoped customer with name + phone/WA only (no order).
func (s *Service) CreateCustomer(ctx context.Context, storeID uuid.UUID, in CreateCustomerInput) (*CreateCustomerResult, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, ErrInvalidInput
	}
	phone, ok := validate.NormalizeIndonesianPhone(in.Phone)
	if !ok {
		return nil, ErrInvalidInput
	}

	var slug string
	err := s.DB.QueryRow(ctx, `
		SELECT slug FROM stores WHERE id=$1 AND archived_at IS NULL
	`, storeID).Scan(&slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	secret, err := validate.NewPortalAccessSecret()
	if err != nil {
		return nil, err
	}

	var c Customer
	err = s.DB.QueryRow(ctx, `
		INSERT INTO customers (store_id, name, email, phone, total_orders, completed_order_value_idr, portal_access_secret)
		VALUES ($1, $2, NULL, $3, 0, 0, $4)
		RETURNING id, store_id, name, email, phone, first_order_at, last_order_at,
		          total_orders, completed_order_value_idr, internal_notes, created_at, updated_at
	`, storeID, name, phone, secret).Scan(
		&c.ID, &c.StoreID, &c.Name, &c.Email, &c.Phone, &c.FirstOrderAt, &c.LastOrderAt,
		&c.TotalOrders, &c.CompletedOrderValueIDR, &c.InternalNotes, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "customers_store_phone_unique") {
			return nil, ErrConflict
		}
		return nil, err
	}

	res := &CreateCustomerResult{Customer: c, StoreSlug: slug}
	if s.URLs != nil {
		res.PortalURL = s.URLs.PortalCustomer(slug, secret, phone)
	}
	return res, nil
}

type UpdateCustomerInput struct {
	InternalNotes *string `json:"internal_notes,omitempty"`
	Name          *string `json:"name,omitempty"`
	Phone         *string `json:"phone,omitempty"`
}

func (s *Service) UpdateCustomer(ctx context.Context, storeID, customerID uuid.UUID, in UpdateCustomerInput) (*Customer, error) {
	cur, err := s.GetCustomer(ctx, storeID, customerID)
	if err != nil {
		return nil, err
	}
	name := cur.Name
	if in.Name != nil {
		n := strings.TrimSpace(*in.Name)
		if n == "" {
			return nil, ErrInvalidInput
		}
		name = n
	}
	phone := cur.Phone
	if in.Phone != nil {
		p := strings.TrimSpace(*in.Phone)
		if p == "" {
			phone = nil
		} else {
			normalized, ok := validate.NormalizeIndonesianPhone(p)
			if !ok {
				return nil, ErrInvalidInput
			}
			phone = &normalized
		}
	}
	notes := cur.InternalNotes
	if in.InternalNotes != nil {
		n := strings.TrimSpace(*in.InternalNotes)
		if n == "" {
			notes = nil
		} else {
			notes = &n
		}
	}
	_, err = s.DB.Exec(ctx, `
		UPDATE customers SET name=$3, phone=$4, internal_notes=$5, updated_at=now()
		WHERE id=$1 AND store_id=$2
	`, customerID, storeID, name, phone, notes)
	if err != nil {
		return nil, err
	}
	detail, err := s.GetCustomer(ctx, storeID, customerID)
	if err != nil {
		return nil, err
	}
	return &detail.Customer, nil
}

// CustomerPortalURL returns the durable shareable portal URL for a store customer.
// Mints a portal_access_secret if missing (pre-migration rows should already be backfilled).
func (s *Service) CustomerPortalURL(ctx context.Context, storeID, customerID uuid.UUID) (portalURL, storeSlug string, err error) {
	var slug string
	var phone *string
	var secret string
	err = s.DB.QueryRow(ctx, `
		SELECT s.slug, c.phone, c.portal_access_secret
		FROM customers c
		JOIN stores s ON s.id=c.store_id AND s.archived_at IS NULL
		WHERE c.id=$1 AND c.store_id=$2
	`, customerID, storeID).Scan(&slug, &phone, &secret)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrNotFound
	}
	if err != nil {
		return "", "", err
	}
	if !validate.ValidPortalAccessSecret(secret) {
		secret, err = validate.NewPortalAccessSecret()
		if err != nil {
			return "", "", err
		}
		_, err = s.DB.Exec(ctx, `
			UPDATE customers SET portal_access_secret=$3, updated_at=now()
			WHERE id=$1 AND store_id=$2
		`, customerID, storeID, secret)
		if err != nil {
			return "", "", err
		}
	}
	if phone == nil || strings.TrimSpace(*phone) == "" || s.URLs == nil {
		return "", slug, nil
	}
	return s.URLs.PortalCustomer(slug, secret, strings.TrimSpace(*phone)), slug, nil
}

// RotatePortalAccessSecret invalidates the prior portal URL and mints a new secret.
// Also revokes active portal sessions for that customer.
func (s *Service) RotatePortalAccessSecret(ctx context.Context, storeID, customerID uuid.UUID) (portalURL, storeSlug string, err error) {
	secret, err := validate.NewPortalAccessSecret()
	if err != nil {
		return "", "", err
	}
	tag, err := s.DB.Exec(ctx, `
		UPDATE customers SET portal_access_secret=$3, updated_at=now()
		WHERE id=$1 AND store_id=$2
	`, customerID, storeID, secret)
	if err != nil {
		return "", "", err
	}
	if tag.RowsAffected() != 1 {
		return "", "", ErrNotFound
	}
	_, _ = s.DB.Exec(ctx, `
		UPDATE customer_portal_sessions SET revoked_at=now()
		WHERE store_id=$1 AND customer_id=$2 AND revoked_at IS NULL
	`, storeID, customerID)
	return s.CustomerPortalURL(ctx, storeID, customerID)
}
