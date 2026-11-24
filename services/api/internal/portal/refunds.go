package portal

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Service) GetRefundPreference(ctx context.Context, storeID, customerID uuid.UUID) (*RefundPreference, error) {
	var p RefundPreference
	err := s.DB.QueryRow(ctx, `
		SELECT store_id, customer_id, preferred_method, bank_name, account_holder, account_number, updated_at
		FROM customer_refund_preferences
		WHERE store_id=$1 AND customer_id=$2
	`, storeID, customerID).Scan(
		&p.StoreID, &p.CustomerID, &p.PreferredMethod, &p.BankName, &p.AccountHolder, &p.AccountNumber, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Service) UpsertRefundPreference(ctx context.Context, storeID, customerID uuid.UUID, in UpsertRefundPreferenceInput) (*RefundPreference, error) {
	method := strings.TrimSpace(in.PreferredMethod)
	switch method {
	case RefundMethodBankTransfer, RefundMethodStoreCredit:
	default:
		return nil, ErrInvalidInput
	}
	if method == RefundMethodStoreCredit {
		if s.Commerce == nil || !s.Commerce.StoreCreditGloballyEnabled() {
			return nil, ErrStoreCreditOff
		}
		// Per-store enablement
		var enabled bool
		err := s.DB.QueryRow(ctx, `SELECT store_credit_enabled FROM stores WHERE id=$1`, storeID).Scan(&enabled)
		if err != nil || !enabled {
			return nil, ErrStoreCreditOff
		}
	}

	bankName := trimPtr(in.BankName)
	holder := trimPtr(in.AccountHolder)
	number := trimPtr(in.AccountNumber)
	if method == RefundMethodBankTransfer {
		if bankName == nil || holder == nil || number == nil {
			return nil, ErrInvalidInput
		}
	} else {
		// Clear bank details when choosing store credit
		bankName, holder, number = nil, nil, nil
	}

	_, err := s.DB.Exec(ctx, `
		INSERT INTO customer_refund_preferences (
			store_id, customer_id, preferred_method, bank_name, account_holder, account_number, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,now())
		ON CONFLICT (store_id, customer_id) DO UPDATE SET
			preferred_method=EXCLUDED.preferred_method,
			bank_name=EXCLUDED.bank_name,
			account_holder=EXCLUDED.account_holder,
			account_number=EXCLUDED.account_number,
			updated_at=now()
	`, storeID, customerID, method, bankName, holder, number)
	if err != nil {
		return nil, err
	}
	return s.GetRefundPreference(ctx, storeID, customerID)
}
