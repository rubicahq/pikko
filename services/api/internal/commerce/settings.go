package commerce

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type UpsertPaymentMethodInput struct {
	BankName             string  `json:"bank_name"`
	AccountHolder        string  `json:"account_holder"`
	AccountNumber        string  `json:"account_number"`
	Instructions         *string `json:"instructions,omitempty"`
	PaymentDeadlineHours *int    `json:"payment_deadline_hours,omitempty"`
	IsEnabled            *bool   `json:"is_enabled,omitempty"`
	DisplayOrder         *int    `json:"display_order,omitempty"`
}

type PatchFulfillmentInput struct {
	ShippingEnabled     *bool   `json:"shipping_enabled,omitempty"`
	ShippingFeeIDR      *int64  `json:"shipping_fee_idr,omitempty"`
	FreeShippingEnabled *bool   `json:"free_shipping_enabled,omitempty"`
	PickupEnabled       *bool   `json:"pickup_enabled,omitempty"`
	PickupInstructions  *string `json:"pickup_instructions,omitempty"`
	ServiceAreaNotes    *string `json:"service_area_notes,omitempty"`
}

func (s *Service) ListPaymentMethods(ctx context.Context, storeID uuid.UUID) ([]PaymentMethod, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT id, store_id, method_type, bank_name, account_holder, account_number, instructions,
		       payment_deadline_hours, is_enabled, display_order, created_at, updated_at
		FROM store_payment_methods
		WHERE store_id=$1
		ORDER BY display_order, created_at
	`, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PaymentMethod
	for rows.Next() {
		var m PaymentMethod
		if err := rows.Scan(
			&m.ID, &m.StoreID, &m.MethodType, &m.BankName, &m.AccountHolder, &m.AccountNumber, &m.Instructions,
			&m.PaymentDeadlineHours, &m.IsEnabled, &m.DisplayOrder, &m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Service) ListEnabledPaymentMethods(ctx context.Context, storeID uuid.UUID) ([]PaymentMethod, error) {
	all, err := s.ListPaymentMethods(ctx, storeID)
	if err != nil {
		return nil, err
	}
	var out []PaymentMethod
	for _, m := range all {
		if m.IsEnabled {
			out = append(out, m)
		}
	}
	return out, nil
}

func (s *Service) CreatePaymentMethod(ctx context.Context, storeID uuid.UUID, in UpsertPaymentMethodInput) (*PaymentMethod, error) {
	bank := strings.TrimSpace(in.BankName)
	holder := strings.TrimSpace(in.AccountHolder)
	number := strings.TrimSpace(in.AccountNumber)
	if bank == "" || holder == "" || number == "" {
		return nil, ErrInvalidInput
	}
	hours := 48
	if in.PaymentDeadlineHours != nil {
		if *in.PaymentDeadlineHours <= 0 || *in.PaymentDeadlineHours > 24*30 {
			return nil, ErrInvalidInput
		}
		hours = *in.PaymentDeadlineHours
	}
	enabled := true
	if in.IsEnabled != nil {
		enabled = *in.IsEnabled
	}
	order := 0
	if in.DisplayOrder != nil {
		order = *in.DisplayOrder
	}
	var m PaymentMethod
	err := s.DB.QueryRow(ctx, `
		INSERT INTO store_payment_methods (
			store_id, bank_name, account_holder, account_number, instructions,
			payment_deadline_hours, is_enabled, display_order
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id, store_id, method_type, bank_name, account_holder, account_number, instructions,
		          payment_deadline_hours, is_enabled, display_order, created_at, updated_at
	`, storeID, bank, holder, number, in.Instructions, hours, enabled, order).Scan(
		&m.ID, &m.StoreID, &m.MethodType, &m.BankName, &m.AccountHolder, &m.AccountNumber, &m.Instructions,
		&m.PaymentDeadlineHours, &m.IsEnabled, &m.DisplayOrder, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	_ = s.refreshCheckoutEnabled(ctx, storeID)
	return &m, nil
}

func (s *Service) UpdatePaymentMethod(ctx context.Context, storeID, methodID uuid.UUID, in UpsertPaymentMethodInput) (*PaymentMethod, error) {
	cur, err := s.getPaymentMethod(ctx, storeID, methodID)
	if err != nil {
		return nil, err
	}
	bank := cur.BankName
	holder := cur.AccountHolder
	number := cur.AccountNumber
	instr := cur.Instructions
	hours := cur.PaymentDeadlineHours
	enabled := cur.IsEnabled
	order := cur.DisplayOrder
	if strings.TrimSpace(in.BankName) != "" {
		bank = strings.TrimSpace(in.BankName)
	}
	if strings.TrimSpace(in.AccountHolder) != "" {
		holder = strings.TrimSpace(in.AccountHolder)
	}
	if strings.TrimSpace(in.AccountNumber) != "" {
		number = strings.TrimSpace(in.AccountNumber)
	}
	if in.Instructions != nil {
		instr = in.Instructions
	}
	if in.PaymentDeadlineHours != nil {
		if *in.PaymentDeadlineHours <= 0 {
			return nil, ErrInvalidInput
		}
		hours = *in.PaymentDeadlineHours
	}
	if in.IsEnabled != nil {
		enabled = *in.IsEnabled
	}
	if in.DisplayOrder != nil {
		order = *in.DisplayOrder
	}
	var m PaymentMethod
	err = s.DB.QueryRow(ctx, `
		UPDATE store_payment_methods SET
			bank_name=$3, account_holder=$4, account_number=$5, instructions=$6,
			payment_deadline_hours=$7, is_enabled=$8, display_order=$9, updated_at=now()
		WHERE id=$1 AND store_id=$2
		RETURNING id, store_id, method_type, bank_name, account_holder, account_number, instructions,
		          payment_deadline_hours, is_enabled, display_order, created_at, updated_at
	`, methodID, storeID, bank, holder, number, instr, hours, enabled, order).Scan(
		&m.ID, &m.StoreID, &m.MethodType, &m.BankName, &m.AccountHolder, &m.AccountNumber, &m.Instructions,
		&m.PaymentDeadlineHours, &m.IsEnabled, &m.DisplayOrder, &m.CreatedAt, &m.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	_ = s.refreshCheckoutEnabled(ctx, storeID)
	return &m, nil
}

func (s *Service) DeletePaymentMethod(ctx context.Context, storeID, methodID uuid.UUID) error {
	ct, err := s.DB.Exec(ctx, `DELETE FROM store_payment_methods WHERE id=$1 AND store_id=$2`, methodID, storeID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return s.refreshCheckoutEnabled(ctx, storeID)
}

func (s *Service) getPaymentMethod(ctx context.Context, storeID, methodID uuid.UUID) (*PaymentMethod, error) {
	var m PaymentMethod
	err := s.DB.QueryRow(ctx, `
		SELECT id, store_id, method_type, bank_name, account_holder, account_number, instructions,
		       payment_deadline_hours, is_enabled, display_order, created_at, updated_at
		FROM store_payment_methods WHERE id=$1 AND store_id=$2
	`, methodID, storeID).Scan(
		&m.ID, &m.StoreID, &m.MethodType, &m.BankName, &m.AccountHolder, &m.AccountNumber, &m.Instructions,
		&m.PaymentDeadlineHours, &m.IsEnabled, &m.DisplayOrder, &m.CreatedAt, &m.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *Service) GetFulfillmentSettings(ctx context.Context, storeID uuid.UUID) (*FulfillmentSettings, error) {
	var f FulfillmentSettings
	err := s.DB.QueryRow(ctx, `
		SELECT store_id, shipping_enabled, shipping_fee_idr, free_shipping_enabled,
		       pickup_enabled, pickup_instructions, service_area_notes, updated_at
		FROM store_fulfillment_settings WHERE store_id=$1
	`, storeID).Scan(
		&f.StoreID, &f.ShippingEnabled, &f.ShippingFeeIDR, &f.FreeShippingEnabled,
		&f.PickupEnabled, &f.PickupInstructions, &f.ServiceAreaNotes, &f.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return &FulfillmentSettings{StoreID: storeID}, nil
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (s *Service) UpsertFulfillmentSettings(ctx context.Context, storeID uuid.UUID, in PatchFulfillmentInput) (*FulfillmentSettings, error) {
	cur, err := s.GetFulfillmentSettings(ctx, storeID)
	if err != nil {
		return nil, err
	}
	if in.ShippingEnabled != nil {
		cur.ShippingEnabled = *in.ShippingEnabled
	}
	if in.ShippingFeeIDR != nil {
		if *in.ShippingFeeIDR < 0 {
			return nil, ErrInvalidInput
		}
		cur.ShippingFeeIDR = *in.ShippingFeeIDR
	}
	if in.FreeShippingEnabled != nil {
		cur.FreeShippingEnabled = *in.FreeShippingEnabled
	}
	if in.PickupEnabled != nil {
		cur.PickupEnabled = *in.PickupEnabled
	}
	if in.PickupInstructions != nil {
		cur.PickupInstructions = in.PickupInstructions
	}
	if in.ServiceAreaNotes != nil {
		cur.ServiceAreaNotes = in.ServiceAreaNotes
	}
	// Flat-rate shipping requires a known fee (may be 0 only if free shipping is the method).
	if cur.ShippingEnabled && cur.ShippingFeeIDR < 0 {
		return nil, ErrInvalidInput
	}
	err = s.DB.QueryRow(ctx, `
		INSERT INTO store_fulfillment_settings (
			store_id, shipping_enabled, shipping_fee_idr, free_shipping_enabled,
			pickup_enabled, pickup_instructions, service_area_notes, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,now())
		ON CONFLICT (store_id) DO UPDATE SET
			shipping_enabled=EXCLUDED.shipping_enabled,
			shipping_fee_idr=EXCLUDED.shipping_fee_idr,
			free_shipping_enabled=EXCLUDED.free_shipping_enabled,
			pickup_enabled=EXCLUDED.pickup_enabled,
			pickup_instructions=EXCLUDED.pickup_instructions,
			service_area_notes=EXCLUDED.service_area_notes,
			updated_at=now()
		RETURNING store_id, shipping_enabled, shipping_fee_idr, free_shipping_enabled,
		          pickup_enabled, pickup_instructions, service_area_notes, updated_at
	`, storeID, cur.ShippingEnabled, cur.ShippingFeeIDR, cur.FreeShippingEnabled,
		cur.PickupEnabled, cur.PickupInstructions, cur.ServiceAreaNotes,
	).Scan(
		&cur.StoreID, &cur.ShippingEnabled, &cur.ShippingFeeIDR, &cur.FreeShippingEnabled,
		&cur.PickupEnabled, &cur.PickupInstructions, &cur.ServiceAreaNotes, &cur.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	_ = s.refreshCheckoutEnabled(ctx, storeID)
	return cur, nil
}
