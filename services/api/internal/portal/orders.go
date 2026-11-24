package portal

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rubicahq/pikko/services/api/internal/commerce"
)

func (s *Service) GetDashboard(ctx context.Context, storeID, customerID uuid.UUID) (*Dashboard, error) {
	st, err := s.storeMetaByID(ctx, storeID)
	if err != nil {
		return nil, err
	}
	customer, err := s.loadCustomer(ctx, storeID, customerID)
	if err != nil {
		return nil, err
	}
	orders, total, err := s.Commerce.ListOrders(ctx, storeID, commerce.ListOrdersFilter{
		CustomerID: &customerID,
		Limit:      25,
		Offset:     0,
	})
	if err != nil {
		return nil, err
	}
	for i := range orders {
		orders[i].InternalNotes = nil
	}
	pref, _ := s.GetRefundPreference(ctx, storeID, customerID)
	creditEnabled := false
	var credit *commerce.StoreCreditBalance
	if s.Commerce != nil && s.Commerce.StoreCreditGloballyEnabled() {
		if bal, cerr := s.Commerce.GetStoreCredit(ctx, storeID, customerID); cerr == nil {
			creditEnabled = true
			credit = bal
		} else if !errors.Is(cerr, commerce.ErrStoreCreditDisabled) {
			return nil, cerr
		}
	}
	var addrCount int
	_ = s.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM customer_addresses WHERE store_id=$1 AND customer_id=$2
	`, storeID, customerID).Scan(&addrCount)

	return &Dashboard{
		Store:              *st,
		Customer:           *customer,
		Orders:             orders,
		OrdersTotal:        total,
		RefundPreference:   pref,
		StoreCreditEnabled: creditEnabled,
		StoreCredit:        credit,
		AddressesCount:     addrCount,
	}, nil
}

func (s *Service) ListOrders(ctx context.Context, storeID, customerID uuid.UUID, limit, offset int) ([]commerce.Order, int, error) {
	orders, total, err := s.Commerce.ListOrders(ctx, storeID, commerce.ListOrdersFilter{
		CustomerID: &customerID,
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		return nil, 0, err
	}
	for i := range orders {
		orders[i].InternalNotes = nil
	}
	return orders, total, nil
}

func (s *Service) GetOrderDetail(ctx context.Context, storeID, customerID, orderID uuid.UUID) (*OrderDetail, error) {
	o, err := s.Commerce.GetOrder(ctx, storeID, orderID)
	if err != nil {
		if errors.Is(err, commerce.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	// Tenant + customer isolation (E2E-M / IDOR)
	if o.CustomerID == nil || *o.CustomerID != customerID {
		return nil, ErrNotFound
	}
	o.InternalNotes = nil

	fin, _ := s.Commerce.GetOrderFinance(ctx, storeID, orderID)
	payments, _ := s.Commerce.ListOrderPayments(ctx, storeID, orderID)
	// Strip signed URLs / merchant-only paths if any; evidence ids stay for transparency.
	refunds, _ := s.Commerce.ListOrderRefunds(ctx, storeID, orderID)
	adjustments, _ := s.Commerce.ListOrderAdjustments(ctx, storeID, orderID)

	pending, _ := s.getPendingAddressChange(ctx, storeID, customerID, orderID)
	canChange := s.canRequestAddressChange(o, pending)

	return &OrderDetail{
		Order:                   *o,
		Finance:                 fin,
		Payments:                payments,
		Adjustments:             adjustments,
		Refunds:                 refunds,
		AddressChange:           pending,
		CanRequestAddressChange: canChange,
	}, nil
}

func (s *Service) storeMetaByID(ctx context.Context, storeID uuid.UUID) (*StoreMeta, error) {
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

func (s *Service) canRequestAddressChange(o *commerce.Order, pending *AddressChangeRequest) bool {
	if pending != nil {
		return false
	}
	if o.FulfillmentMethod != commerce.FulfillmentShipping && o.FulfillmentMethod != commerce.FulfillmentFreeShipping {
		return false
	}
	switch o.FulfillmentStatus {
	case commerce.FulfillmentShipped, commerce.FulfillmentDelivered, commerce.FulfillmentCollected, commerce.FulfillmentCancelled:
		return false
	}
	if o.OrderStatus == commerce.OrderCancelled {
		return false
	}
	return true
}
