package portal

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rubicahq/pikko/services/api/internal/commerce"
)

func (s *Service) ListAddresses(ctx context.Context, storeID, customerID uuid.UUID) ([]Address, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT id, store_id, customer_id, label, recipient_name, phone, line1, line2,
		       city, province, postal_code, country, is_default, created_at, updated_at
		FROM customer_addresses
		WHERE store_id=$1 AND customer_id=$2
		ORDER BY is_default DESC, updated_at DESC
	`, storeID, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Address
	for rows.Next() {
		a, err := scanAddress(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func (s *Service) CreateAddress(ctx context.Context, storeID, customerID uuid.UUID, in AddressInput) (*Address, error) {
	in, err := normalizeAddressInput(in)
	if err != nil {
		return nil, err
	}
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	isDefault := in.IsDefault != nil && *in.IsDefault
	if !isDefault {
		var count int
		if err := tx.QueryRow(ctx, `
			SELECT COUNT(*) FROM customer_addresses WHERE store_id=$1 AND customer_id=$2
		`, storeID, customerID).Scan(&count); err != nil {
			return nil, err
		}
		if count == 0 {
			isDefault = true
		}
	}
	if isDefault {
		if _, err := tx.Exec(ctx, `
			UPDATE customer_addresses SET is_default=false, updated_at=now()
			WHERE store_id=$1 AND customer_id=$2 AND is_default=true
		`, storeID, customerID); err != nil {
			return nil, err
		}
	}

	var id uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO customer_addresses (
			store_id, customer_id, label, recipient_name, phone, line1, line2,
			city, province, postal_code, country, is_default
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING id
	`, storeID, customerID, in.Label, in.RecipientName, in.Phone, in.Line1, in.Line2,
		in.City, in.Province, in.PostalCode, in.Country, isDefault).Scan(&id)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.getAddress(ctx, storeID, customerID, id)
}

func (s *Service) UpdateAddress(ctx context.Context, storeID, customerID, addressID uuid.UUID, in AddressInput) (*Address, error) {
	in, err := normalizeAddressInput(in)
	if err != nil {
		return nil, err
	}
	// Ownership check — IDOR: wrong customer/store → not found
	cur, err := s.getAddress(ctx, storeID, customerID, addressID)
	if err != nil {
		return nil, err
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	isDefault := cur.IsDefault
	if in.IsDefault != nil {
		isDefault = *in.IsDefault
	}
	if isDefault {
		if _, err := tx.Exec(ctx, `
			UPDATE customer_addresses SET is_default=false, updated_at=now()
			WHERE store_id=$1 AND customer_id=$2 AND is_default=true AND id<>$3
		`, storeID, customerID, addressID); err != nil {
			return nil, err
		}
	}

	tag, err := tx.Exec(ctx, `
		UPDATE customer_addresses SET
			label=$4, recipient_name=$5, phone=$6, line1=$7, line2=$8,
			city=$9, province=$10, postal_code=$11, country=$12, is_default=$13, updated_at=now()
		WHERE id=$1 AND store_id=$2 AND customer_id=$3
	`, addressID, storeID, customerID, in.Label, in.RecipientName, in.Phone, in.Line1, in.Line2,
		in.City, in.Province, in.PostalCode, in.Country, isDefault)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() != 1 {
		return nil, ErrNotFound
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.getAddress(ctx, storeID, customerID, addressID)
}

func (s *Service) DeleteAddress(ctx context.Context, storeID, customerID, addressID uuid.UUID) error {
	tag, err := s.DB.Exec(ctx, `
		DELETE FROM customer_addresses WHERE id=$1 AND store_id=$2 AND customer_id=$3
	`, addressID, storeID, customerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) RequestAddressChange(ctx context.Context, storeID, customerID, orderID uuid.UUID, in CreateAddressChangeInput) (*AddressChangeRequest, error) {
	detail, err := s.GetOrderDetail(ctx, storeID, customerID, orderID)
	if err != nil {
		return nil, err
	}
	if !detail.CanRequestAddressChange {
		return nil, ErrForbiddenState
	}
	in.RecipientName = strings.TrimSpace(in.RecipientName)
	in.Line1 = strings.TrimSpace(in.Line1)
	in.City = strings.TrimSpace(in.City)
	in.Country = strings.TrimSpace(in.Country)
	if in.Country == "" {
		in.Country = "ID"
	}
	if in.RecipientName == "" || in.Line1 == "" || in.City == "" {
		return nil, ErrInvalidInput
	}

	var id uuid.UUID
	err = s.DB.QueryRow(ctx, `
		INSERT INTO portal_address_change_requests (
			store_id, customer_id, order_id, status,
			recipient_name, phone, line1, line2, city, province, postal_code, country, customer_note
		) VALUES ($1,$2,$3,'pending',$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING id
	`, storeID, customerID, orderID,
		in.RecipientName, trimPtr(in.Phone), in.Line1, trimPtr(in.Line2), in.City,
		trimPtr(in.Province), trimPtr(in.PostalCode), in.Country, trimPtr(in.CustomerNote),
	).Scan(&id)
	if err != nil {
		if strings.Contains(err.Error(), "portal_address_change_requests_order_pending_unique") {
			return nil, ErrConflict
		}
		return nil, err
	}
	return s.getAddressChange(ctx, storeID, id)
}

func (s *Service) ListCustomerAddressChanges(ctx context.Context, storeID, customerID uuid.UUID) ([]AddressChangeRequest, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT r.id, r.store_id, r.customer_id, r.order_id, o.order_number, r.status,
		       r.recipient_name, r.phone, r.line1, r.line2, r.city, r.province, r.postal_code, r.country,
		       r.customer_note, r.merchant_note, r.reviewed_by, r.reviewed_at, r.created_at, r.updated_at
		FROM portal_address_change_requests r
		JOIN orders o ON o.id=r.order_id AND o.store_id=r.store_id
		WHERE r.store_id=$1 AND r.customer_id=$2
		ORDER BY r.created_at DESC
		LIMIT 50
	`, storeID, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAddressChanges(rows)
}

// ListStoreAddressChanges is the merchant-facing queue (customers.write).
func (s *Service) ListStoreAddressChanges(ctx context.Context, storeID uuid.UUID, status string, limit int) ([]AddressChangeRequest, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	args := []any{storeID}
	q := `
		SELECT r.id, r.store_id, r.customer_id, r.order_id, o.order_number, r.status,
		       r.recipient_name, r.phone, r.line1, r.line2, r.city, r.province, r.postal_code, r.country,
		       r.customer_note, r.merchant_note, r.reviewed_by, r.reviewed_at, r.created_at, r.updated_at
		FROM portal_address_change_requests r
		JOIN orders o ON o.id=r.order_id AND o.store_id=r.store_id
		WHERE r.store_id=$1`
	if status != "" {
		args = append(args, status)
		q += ` AND r.status=$2`
	}
	args = append(args, limit)
	q += fmt.Sprintf(` ORDER BY r.created_at DESC LIMIT $%d`, len(args))
	rows, err := s.DB.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAddressChanges(rows)
}

func (s *Service) ReviewAddressChange(ctx context.Context, storeID, requestID, actorID uuid.UUID, in ReviewAddressChangeInput) (*AddressChangeRequest, error) {
	status := strings.TrimSpace(in.Status)
	if status != AddressChangeApproved && status != AddressChangeRejected {
		return nil, ErrInvalidInput
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var req AddressChangeRequest
	var orderID uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT id, store_id, customer_id, order_id, status,
		       recipient_name, phone, line1, line2, city, province, postal_code, country
		FROM portal_address_change_requests
		WHERE id=$1 AND store_id=$2 FOR UPDATE
	`, requestID, storeID).Scan(
		&req.ID, &req.StoreID, &req.CustomerID, &orderID, &req.Status,
		&req.RecipientName, &req.Phone, &req.Line1, &req.Line2, &req.City, &req.Province, &req.PostalCode, &req.Country,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if req.Status != AddressChangePending {
		return nil, ErrConflict
	}
	req.OrderID = orderID

	if status == AddressChangeApproved {
		// Apply shipping snapshot on order; refuse if already shipped.
		var fulfillmentStatus, fulfillmentMethod string
		err = tx.QueryRow(ctx, `
			SELECT fulfillment_status, fulfillment_method FROM orders
			WHERE id=$1 AND store_id=$2 AND customer_id=$3 FOR UPDATE
		`, orderID, storeID, req.CustomerID).Scan(&fulfillmentStatus, &fulfillmentMethod)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		if err != nil {
			return nil, err
		}
		switch fulfillmentStatus {
		case commerce.FulfillmentShipped, commerce.FulfillmentDelivered, commerce.FulfillmentCollected, commerce.FulfillmentCancelled:
			return nil, ErrForbiddenState
		}
		_, err = tx.Exec(ctx, `
			UPDATE orders SET
				shipping_recipient=$3, shipping_phone=$4, shipping_line1=$5, shipping_line2=$6,
				shipping_city=$7, shipping_province=$8, shipping_postal_code=$9, shipping_country=$10,
				updated_at=now()
			WHERE id=$1 AND store_id=$2
		`, orderID, storeID, req.RecipientName, req.Phone, req.Line1, req.Line2,
			req.City, req.Province, req.PostalCode, req.Country)
		if err != nil {
			return nil, err
		}
	}

	_, err = tx.Exec(ctx, `
		UPDATE portal_address_change_requests SET
			status=$3, merchant_note=$4, reviewed_by=$5, reviewed_at=now(), updated_at=now()
		WHERE id=$1 AND store_id=$2 AND status='pending'
	`, requestID, storeID, status, trimPtr(in.MerchantNote), actorID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.getAddressChange(ctx, storeID, requestID)
}

func (s *Service) getPendingAddressChange(ctx context.Context, storeID, customerID, orderID uuid.UUID) (*AddressChangeRequest, error) {
	var id uuid.UUID
	err := s.DB.QueryRow(ctx, `
		SELECT id FROM portal_address_change_requests
		WHERE store_id=$1 AND customer_id=$2 AND order_id=$3 AND status='pending'
		LIMIT 1
	`, storeID, customerID, orderID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.getAddressChange(ctx, storeID, id)
}

func (s *Service) getAddressChange(ctx context.Context, storeID, id uuid.UUID) (*AddressChangeRequest, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT r.id, r.store_id, r.customer_id, r.order_id, o.order_number, r.status,
		       r.recipient_name, r.phone, r.line1, r.line2, r.city, r.province, r.postal_code, r.country,
		       r.customer_note, r.merchant_note, r.reviewed_by, r.reviewed_at, r.created_at, r.updated_at
		FROM portal_address_change_requests r
		JOIN orders o ON o.id=r.order_id AND o.store_id=r.store_id
		WHERE r.id=$1 AND r.store_id=$2
	`, id, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list, err := scanAddressChanges(rows)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, ErrNotFound
	}
	return &list[0], nil
}

func (s *Service) getAddress(ctx context.Context, storeID, customerID, id uuid.UUID) (*Address, error) {
	row := s.DB.QueryRow(ctx, `
		SELECT id, store_id, customer_id, label, recipient_name, phone, line1, line2,
		       city, province, postal_code, country, is_default, created_at, updated_at
		FROM customer_addresses
		WHERE id=$1 AND store_id=$2 AND customer_id=$3
	`, id, storeID, customerID)
	a, err := scanAddress(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return a, err
}

type addressScanner interface {
	Scan(dest ...any) error
}

func scanAddress(row addressScanner) (*Address, error) {
	var a Address
	err := row.Scan(
		&a.ID, &a.StoreID, &a.CustomerID, &a.Label, &a.RecipientName, &a.Phone, &a.Line1, &a.Line2,
		&a.City, &a.Province, &a.PostalCode, &a.Country, &a.IsDefault, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func scanAddressChanges(rows pgx.Rows) ([]AddressChangeRequest, error) {
	var out []AddressChangeRequest
	for rows.Next() {
		var r AddressChangeRequest
		if err := rows.Scan(
			&r.ID, &r.StoreID, &r.CustomerID, &r.OrderID, &r.OrderNumber, &r.Status,
			&r.RecipientName, &r.Phone, &r.Line1, &r.Line2, &r.City, &r.Province, &r.PostalCode, &r.Country,
			&r.CustomerNote, &r.MerchantNote, &r.ReviewedBy, &r.ReviewedAt, &r.CreatedAt, &r.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func normalizeAddressInput(in AddressInput) (AddressInput, error) {
	in.RecipientName = strings.TrimSpace(in.RecipientName)
	in.Line1 = strings.TrimSpace(in.Line1)
	in.City = strings.TrimSpace(in.City)
	in.Country = strings.TrimSpace(in.Country)
	if in.Country == "" {
		in.Country = "ID"
	}
	in.Label = trimPtr(in.Label)
	in.Phone = trimPtr(in.Phone)
	in.Line2 = trimPtr(in.Line2)
	in.Province = trimPtr(in.Province)
	in.PostalCode = trimPtr(in.PostalCode)
	if in.RecipientName == "" || in.Line1 == "" || in.City == "" {
		return in, ErrInvalidInput
	}
	return in, nil
}
