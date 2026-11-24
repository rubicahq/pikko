package commerce

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type CustomerDetail struct {
	Customer
	Orders []Order `json:"orders"`
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
			phone = &p
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
