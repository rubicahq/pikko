package commerce

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rubicahq/pikko/services/api/internal/auth"
	"github.com/rubicahq/pikko/services/api/internal/email"
	"github.com/rubicahq/pikko/services/api/internal/money"
	"github.com/rubicahq/pikko/services/api/internal/validate"
)

type publicStoreMeta struct {
	ID              uuid.UUID
	Name            string
	Slug            string
	Currency        string
	CheckoutEnabled bool
	Published       bool
	Archived        bool
	OwnerEmail      string
	OwnerName       *string
}

func (s *Service) loadPublicStore(ctx context.Context, slug string) (*publicStoreMeta, error) {
	var m publicStoreMeta
	err := s.DB.QueryRow(ctx, `
		SELECT s.id, s.name, s.slug, s.currency, s.checkout_enabled,
		       s.publication_status='published', s.archived_at IS NOT NULL,
		       COALESCE(u.email, ''), u.display_name
		FROM stores s
		LEFT JOIN store_memberships sm ON sm.store_id=s.id AND sm.role='owner' AND sm.status='active'
		LEFT JOIN users u ON u.id=sm.user_id
		WHERE s.slug=$1
	`, slug).Scan(
		&m.ID, &m.Name, &m.Slug, &m.Currency, &m.CheckoutEnabled,
		&m.Published, &m.Archived, &m.OwnerEmail, &m.OwnerName,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if m.Archived || !m.Published {
		return nil, ErrStoreUnavailable
	}
	return &m, nil
}

func (s *Service) priceLines(ctx context.Context, storeID uuid.UUID, items []CartLineInput) ([]PricedLine, int64, error) {
	if len(items) == 0 {
		return nil, 0, ErrInvalidInput
	}
	seen := map[string]struct{}{}
	var lines []PricedLine
	var subtotal int64
	for _, it := range items {
		if it.Quantity <= 0 || it.Quantity > 999 {
			return nil, 0, ErrInvalidInput
		}
		key := it.ProductID.String()
		if it.VariantID != nil {
			key += ":" + it.VariantID.String()
		}
		if _, ok := seen[key]; ok {
			return nil, 0, ErrInvalidInput
		}
		seen[key] = struct{}{}

		var name, slug, status, availability, invMode string
		var basePrice int64
		var sku *string
		var stock *int
		err := s.DB.QueryRow(ctx, `
			SELECT name, slug, sku, base_price_idr, status, availability, inventory_mode, stock_quantity
			FROM products WHERE id=$1 AND store_id=$2
		`, it.ProductID, storeID).Scan(&name, &slug, &sku, &basePrice, &status, &availability, &invMode, &stock)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, 0, ErrNotFound
		}
		if err != nil {
			return nil, 0, err
		}
		if status != "published" || availability != "available" {
			return nil, 0, ErrInsufficientStock
		}
		unit := basePrice
		var variantLabel *string
		var variantID *uuid.UUID
		if it.VariantID != nil {
			var vAvail string
			var override *int64
			var vStock *int
			var optionKey string
			var vSKU *string
			err = s.DB.QueryRow(ctx, `
				SELECT availability, price_override_idr, stock_quantity, option_key, sku
				FROM product_variants WHERE id=$1 AND product_id=$2 AND store_id=$3
			`, *it.VariantID, it.ProductID, storeID).Scan(&vAvail, &override, &vStock, &optionKey, &vSKU)
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, 0, ErrNotFound
			}
			if err != nil {
				return nil, 0, err
			}
			if vAvail != "available" {
				return nil, 0, ErrInsufficientStock
			}
			if override != nil {
				unit = *override
			}
			if optionKey != "" {
				variantLabel = &optionKey
			}
			if vSKU != nil {
				sku = vSKU
			}
			if vStock != nil {
				stock = vStock
			}
			variantID = it.VariantID
		}
		if invMode == "finite" {
			reserved := 0
			_ = s.DB.QueryRow(ctx, `
				SELECT COALESCE(SUM(quantity),0) FROM inventory_reservations
				WHERE store_id=$1 AND product_id=$2 AND status='active'
				  AND (($3::uuid IS NULL AND variant_id IS NULL) OR variant_id=$3)
			`, storeID, it.ProductID, variantID).Scan(&reserved)
			avail := 0
			if stock != nil {
				avail = *stock - reserved
			}
			if avail < it.Quantity {
				return nil, 0, ErrInsufficientStock
			}
			stock = &avail
		}
		lineTotal := unit * int64(it.Quantity)
		lines = append(lines, PricedLine{
			ProductID: it.ProductID, VariantID: variantID, ProductName: name, ProductSlug: slug,
			SKU: sku, VariantLabel: variantLabel, UnitPriceIDR: unit, Quantity: it.Quantity,
			LineTotalIDR: lineTotal, AvailableQty: stock, InventoryMode: invMode,
		})
		subtotal = money.SumIDR(subtotal, lineTotal)
	}
	return lines, subtotal, nil
}

func (s *Service) resolveShipping(ctx context.Context, storeID uuid.UUID, method string) (int64, error) {
	f, err := s.GetFulfillmentSettings(ctx, storeID)
	if err != nil {
		return 0, err
	}
	switch method {
	case FulfillmentShipping:
		if !f.ShippingEnabled {
			return 0, ErrInvalidInput
		}
		return f.ShippingFeeIDR, nil
	case FulfillmentFreeShipping:
		if !f.FreeShippingEnabled {
			return 0, ErrInvalidInput
		}
		return 0, nil
	case FulfillmentPickup:
		if !f.PickupEnabled {
			return 0, ErrInvalidInput
		}
		return 0, nil
	default:
		return 0, ErrInvalidInput
	}
}

func (s *Service) PreviewCheckout(ctx context.Context, slug string, items []CartLineInput, fulfillmentMethod string) (*CheckoutPreview, error) {
	st, err := s.loadPublicStore(ctx, slug)
	if err != nil {
		return nil, err
	}
	lines, subtotal, err := s.priceLines(ctx, st.ID, items)
	if err != nil {
		return nil, err
	}
	var shipFee int64
	if fulfillmentMethod != "" {
		shipFee, err = s.resolveShipping(ctx, st.ID, fulfillmentMethod)
		if err != nil {
			return nil, err
		}
	}
	methods, _ := s.ListEnabledPaymentMethods(ctx, st.ID)
	fulfillment, _ := s.GetFulfillmentSettings(ctx, st.ID)
	return &CheckoutPreview{
		StoreID: st.ID, StoreSlug: st.Slug, StoreName: st.Name, Currency: st.Currency,
		Lines: lines, SubtotalIDR: subtotal, ShippingFeeIDR: shipFee, HandlingFeeIDR: 0,
		TotalIDR: money.SumIDR(subtotal, shipFee), FulfillmentMethod: fulfillmentMethod,
		CheckoutEnabled: st.CheckoutEnabled, PaymentMethods: methods, Fulfillment: fulfillment,
	}, nil
}

type SubmitResult struct {
	Order         *Order   `json:"order"`
	Payment       *Payment `json:"payment"`
	TrackingToken string   `json:"tracking_token"`
	TrackingURL   string   `json:"tracking_url"`
}

func (s *Service) SubmitCheckout(ctx context.Context, slug string, in CheckoutSubmitInput) (*SubmitResult, error) {
	st, err := s.loadPublicStore(ctx, slug)
	if err != nil {
		return nil, err
	}
	if !st.CheckoutEnabled {
		return nil, ErrCheckoutDisabled
	}
	if strings.TrimSpace(in.IdempotencyKey) == "" {
		return nil, ErrInvalidInput
	}
	// Idempotent replay
	var existingID uuid.UUID
	err = s.DB.QueryRow(ctx, `SELECT id FROM orders WHERE idempotency_key=$1`, in.IdempotencyKey).Scan(&existingID)
	if err == nil {
		o, err := s.GetOrder(ctx, st.ID, existingID)
		if err != nil {
			return nil, err
		}
		pay, _ := s.GetLatestPayment(ctx, st.ID, existingID)
		return &SubmitResult{Order: o, Payment: pay}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	name := strings.TrimSpace(in.CustomerName)
	_, emailAddr, emailOK := validate.NormalizeEmail(in.CustomerEmail)
	if len(name) < 2 || !emailOK {
		return nil, ErrInvalidInput
	}
	if in.FulfillmentMethod == FulfillmentShipping || in.FulfillmentMethod == FulfillmentFreeShipping {
		if in.ShippingAddress == nil || strings.TrimSpace(in.ShippingAddress.Line1) == "" ||
			strings.TrimSpace(in.ShippingAddress.City) == "" || strings.TrimSpace(in.ShippingAddress.RecipientName) == "" {
			return nil, ErrInvalidInput
		}
		if in.ShippingAddress.Country == "" {
			in.ShippingAddress.Country = "ID"
		}
	}

	pm, err := s.getPaymentMethod(ctx, st.ID, in.PaymentMethodID)
	if err != nil || !pm.IsEnabled {
		return nil, ErrInvalidInput
	}

	lines, subtotal, err := s.priceLines(ctx, st.ID, in.Items)
	if err != nil {
		return nil, err
	}
	shipFee, err := s.resolveShipping(ctx, st.ID, in.FulfillmentMethod)
	if err != nil {
		return nil, err
	}
	total := money.SumIDR(subtotal, shipFee)
	if in.ClientExpectedTotalIDR != nil && *in.ClientExpectedTotalIDR != total {
		return nil, ErrPriceChanged
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	customerID, err := upsertCustomerTx(ctx, tx, st.ID, name, emailAddr, in.CustomerPhone)
	if err != nil {
		return nil, err
	}

	orderNumber, err := nextOrderNumberTx(ctx, tx, st.ID)
	if err != nil {
		return nil, err
	}

	deadline := s.now().Add(time.Duration(pm.PaymentDeadlineHours) * time.Hour)
	confirmedAt := s.now()

	var shipRecipient, shipPhone, shipLine1, shipLine2, shipCity, shipProvince, shipPostal, shipCountry *string
	if in.ShippingAddress != nil {
		shipRecipient = ptrStr(in.ShippingAddress.RecipientName)
		shipPhone = in.ShippingAddress.Phone
		shipLine1 = ptrStr(in.ShippingAddress.Line1)
		shipLine2 = in.ShippingAddress.Line2
		shipCity = ptrStr(in.ShippingAddress.City)
		shipProvince = in.ShippingAddress.Province
		shipPostal = in.ShippingAddress.PostalCode
		shipCountry = ptrStr(in.ShippingAddress.Country)
	}

	var orderID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO orders (
			store_id, customer_id, order_number, idempotency_key, selling_model,
			order_status, payment_status, fulfillment_status, currency,
			subtotal_idr, shipping_fee_idr, handling_fee_idr, total_idr,
			fulfillment_method, customer_name, customer_email, customer_phone,
			shipping_recipient, shipping_phone, shipping_line1, shipping_line2,
			shipping_city, shipping_province, shipping_postal_code, shipping_country,
			customer_notes, payment_deadline_at, confirmed_at
		) VALUES (
			$1,$2,$3,$4,'ready_stock',
			'confirmed','unpaid','unfulfilled',$5,
			$6,$7,0,$8,
			$9,$10,$11,$12,
			$13,$14,$15,$16,
			$17,$18,$19,$20,
			$21,$22,$23
		)
		RETURNING id
	`, st.ID, customerID, orderNumber, in.IdempotencyKey, st.Currency,
		subtotal, shipFee, total,
		in.FulfillmentMethod, name, emailAddr, in.CustomerPhone,
		shipRecipient, shipPhone, shipLine1, shipLine2,
		shipCity, shipProvince, shipPostal, shipCountry,
		in.CustomerNotes, deadline, confirmedAt,
	).Scan(&orderID)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO order_status_history (store_id, order_id, dimension, from_status, to_status, reason)
		VALUES ($1,$2,'order',NULL,'confirmed','checkout'),
		       ($1,$2,'payment',NULL,'unpaid','checkout'),
		       ($1,$2,'fulfillment',NULL,'unfulfilled','checkout')
	`, st.ID, orderID)
	if err != nil {
		return nil, err
	}

	for _, line := range lines {
		var itemID uuid.UUID
		err = tx.QueryRow(ctx, `
			INSERT INTO order_items (
				store_id, order_id, product_id, variant_id, product_name, product_slug,
				sku, variant_label, unit_price_idr, quantity, line_total_idr
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
			RETURNING id
		`, st.ID, orderID, line.ProductID, line.VariantID, line.ProductName, line.ProductSlug,
			line.SKU, line.VariantLabel, line.UnitPriceIDR, line.Quantity, line.LineTotalIDR,
		).Scan(&itemID)
		if err != nil {
			return nil, err
		}
		if line.InventoryMode == "finite" {
			// Lock product row and re-check stock under transaction
			var stock int
			var invMode, availability string
			err = tx.QueryRow(ctx, `
				SELECT inventory_mode, availability, COALESCE(stock_quantity,0)
				FROM products WHERE id=$1 AND store_id=$2 FOR UPDATE
			`, line.ProductID, st.ID).Scan(&invMode, &availability, &stock)
			if err != nil {
				return nil, err
			}
			if line.VariantID != nil {
				_ = tx.QueryRow(ctx, `
					SELECT COALESCE(stock_quantity, $2) FROM product_variants
					WHERE id=$1 FOR UPDATE
				`, *line.VariantID, stock).Scan(&stock)
			}
			var reserved int
			err = tx.QueryRow(ctx, `
				SELECT COALESCE(SUM(quantity),0) FROM inventory_reservations
				WHERE store_id=$1 AND product_id=$2 AND status='active'
				  AND (($3::uuid IS NULL AND variant_id IS NULL) OR variant_id=$3)
			`, st.ID, line.ProductID, line.VariantID).Scan(&reserved)
			if err != nil {
				return nil, err
			}
			if stock-reserved < line.Quantity {
				return nil, ErrInsufficientStock
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO inventory_reservations (
					store_id, order_id, order_item_id, product_id, variant_id, quantity, status
				) VALUES ($1,$2,$3,$4,$5,$6,'active')
			`, st.ID, orderID, itemID, line.ProductID, line.VariantID, line.Quantity)
			if err != nil {
				return nil, err
			}
		}
	}

	var paymentID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO payments (
			store_id, order_id, payment_method_id, method_type, amount_idr, status,
			bank_name_snapshot, account_holder_snapshot, account_number_snapshot, instructions_snapshot
		) VALUES ($1,$2,$3,'bank_transfer',$4,'pending',$5,$6,$7,$8)
		RETURNING id
	`, st.ID, orderID, pm.ID, total, pm.BankName, pm.AccountHolder, pm.AccountNumber, pm.Instructions,
	).Scan(&paymentID)
	if err != nil {
		return nil, err
	}

	rawToken, tokenHash, err := auth.NewToken()
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO order_tracking_tokens (store_id, order_id, token_hash)
		VALUES ($1,$2,$3)
	`, st.ID, orderID, tokenHash)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	if s.Outbox != nil {
		_ = s.Outbox.Write(ctx, &st.ID, "order.order_created", map[string]any{
			"order_id": orderID.String(), "order_number": orderNumber, "total_idr": total,
		}, "order_created:"+orderID.String())
	}

	order, err := s.GetOrder(ctx, st.ID, orderID)
	if err != nil {
		return nil, err
	}
	order.TrackingToken = rawToken
	if s.URLs != nil {
		order.TrackingURL = s.URLs.Tracking(rawToken)
	}
	pay, _ := s.GetLatestPayment(ctx, st.ID, orderID)

	s.enqueueOrderEmails(ctx, st, order, pay, rawToken)

	return &SubmitResult{
		Order: order, Payment: pay, TrackingToken: rawToken, TrackingURL: order.TrackingURL,
	}, nil
}

func upsertCustomerTx(ctx context.Context, tx pgx.Tx, storeID uuid.UUID, name, emailAddr string, phone *string) (uuid.UUID, error) {
	var id uuid.UUID
	err := tx.QueryRow(ctx, `
		INSERT INTO customers (store_id, name, email, phone, first_order_at, last_order_at, total_orders)
		VALUES ($1,$2,$3,$4,now(),now(),1)
		ON CONFLICT (store_id, email) DO UPDATE SET
			name=EXCLUDED.name,
			phone=COALESCE(EXCLUDED.phone, customers.phone),
			last_order_at=now(),
			total_orders=customers.total_orders+1,
			updated_at=now()
		RETURNING id
	`, storeID, name, emailAddr, phone).Scan(&id)
	return id, err
}

func nextOrderNumberTx(ctx context.Context, tx pgx.Tx, storeID uuid.UUID) (string, error) {
	var n int64
	err := tx.QueryRow(ctx, `
		INSERT INTO store_order_counters (store_id, last_number) VALUES ($1,1)
		ON CONFLICT (store_id) DO UPDATE SET last_number=store_order_counters.last_number+1
		RETURNING last_number
	`, storeID).Scan(&n)
	if err != nil {
		return "", err
	}
	day := time.Now().UTC().Format("20060102")
	return fmt.Sprintf("PK-%s-%04d", day, n), nil
}

func (s *Service) enqueueOrderEmails(ctx context.Context, st *publicStoreMeta, order *Order, pay *Payment, trackingToken string) {
	if s.Email == nil || s.Cfg == nil {
		return
	}
	trackURL := ""
	if s.URLs != nil {
		trackURL = s.URLs.Tracking(trackingToken)
	}
	from := s.Cfg.EmailNotificationFrom
	fromName := st.Name
	if fromName == "" {
		fromName = s.Cfg.EmailFromName
	}

	custSubject, custHTML, custText := email.OrderConfirmationContent(
		st.Name, order.OrderNumber, money.FormatIDR(order.TotalIDR), trackURL,
	)
	_ = s.Email.Enqueue(ctx, email.Message{
		TemplateKey: "order_confirmation_customer", To: order.CustomerEmail,
		From: from, FromName: fromName, ReplyTo: s.Cfg.EmailReplyTo,
		Subject: custSubject, HTML: custHTML, Text: custText,
		IdempotencyKey: "email:order_confirm:" + order.ID.String(),
	})

	if st.OwnerEmail != "" {
		dashURL := ""
		if s.URLs != nil {
			dashURL = fmt.Sprintf("%s/stores/%s/orders/%s", strings.TrimRight(s.Cfg.DashboardURL, "/"), st.ID, order.ID)
		}
		mSubject, mHTML, mText := email.NewOrderMerchantContent(
			st.Name, order.OrderNumber, order.CustomerName, money.FormatIDR(order.TotalIDR), dashURL,
		)
		_ = s.Email.Enqueue(ctx, email.Message{
			TemplateKey: "order_new_merchant", To: st.OwnerEmail,
			From: from, FromName: s.Cfg.EmailFromName, ReplyTo: s.Cfg.EmailReplyTo,
			Subject: mSubject, HTML: mHTML, Text: mText,
			IdempotencyKey: "email:order_merchant:" + order.ID.String(),
		})
	}
	_ = pay
}
