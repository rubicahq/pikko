package commerce

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rubicahq/pikko/services/api/internal/auth"
	"github.com/rubicahq/pikko/services/api/internal/validate"
)

// AdminCreateOrderInput creates an order in the same operational system as checkout.
type AdminCreateOrderInput struct {
	Items             []CartLineInput  `json:"items"`
	CustomerID        *uuid.UUID       `json:"customer_id,omitempty"`
	CustomerName      string           `json:"customer_name"`
	CustomerEmail     string           `json:"customer_email"`
	CustomerPhone     *string          `json:"customer_phone,omitempty"`
	FulfillmentMethod string           `json:"fulfillment_method"`
	ShippingAddress   *CheckoutAddress `json:"shipping_address,omitempty"`
	CustomerNotes     *string          `json:"customer_notes,omitempty"`
	InternalNotes     *string          `json:"internal_notes,omitempty"`
	PaymentMethodID   *uuid.UUID       `json:"payment_method_id,omitempty"`
	// PaymentOption: "bank_transfer" (default) or "paid_by_admin".
	// paid_by_admin marks the order paid immediately so it appears in Fulfilment.
	PaymentOption   string     `json:"payment_option,omitempty"`
	CampaignID      *uuid.UUID `json:"campaign_id,omitempty"`
	PreorderBatchID *uuid.UUID `json:"preorder_batch_id,omitempty"`
	IdempotencyKey  string     `json:"idempotency_key"`
	Source          string     `json:"source,omitempty"` // admin | csv_import
	SkipEmails      bool       `json:"-"`
}

// adminOrderRelations is the store-scoped campaign / preorder link for an order.
type adminOrderRelations struct {
	CampaignID *uuid.UUID
	BatchID    *uuid.UUID
	BatchName  *string
	BatchSlug  *string
	ETAStart   *time.Time
	ETAEnd     *time.Time
}

// resolveAdminOrderRelations validates optional campaign + preorder links for the same store.
// Empty relations are allowed. Cart-derived batch (from line items) must not conflict with an
// explicit preorder_batch_id. A preorder campaign may auto-fill its linked batch; a batch may
// auto-fill its linked campaign when the other side is omitted.
func (s *Service) resolveAdminOrderRelations(
	ctx context.Context,
	storeID uuid.UUID,
	inCampaign, inBatch, cartBatch *uuid.UUID,
) (*adminOrderRelations, error) {
	out := &adminOrderRelations{}
	if cartBatch != nil {
		out.BatchID = cartBatch
	}

	if inBatch != nil {
		batch, err := s.GetBatch(ctx, storeID, *inBatch, false)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return nil, fmt.Errorf("%w: preorder batch not found in this store", ErrInvalidInput)
			}
			return nil, err
		}
		if cartBatch != nil && *cartBatch != *inBatch {
			return nil, fmt.Errorf("%w: preorder batch does not match the batch on order items", ErrInvalidInput)
		}
		out.BatchID = inBatch
		name, slug := batch.Name, batch.Slug
		out.BatchName = &name
		out.BatchSlug = &slug
		out.ETAStart = batch.ETAStart
		out.ETAEnd = batch.ETAEnd
	}

	if inCampaign != nil {
		var linkedBatch *uuid.UUID
		err := s.DB.QueryRow(ctx, `
			SELECT preorder_batch_id
			FROM campaigns WHERE id=$1 AND store_id=$2
		`, *inCampaign, storeID).Scan(&linkedBatch)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: campaign not found in this store", ErrInvalidInput)
		}
		if err != nil {
			return nil, err
		}
		out.CampaignID = inCampaign
		if linkedBatch != nil {
			if out.BatchID != nil && *out.BatchID != *linkedBatch {
				return nil, fmt.Errorf("%w: campaign is linked to a different preorder batch", ErrInvalidInput)
			}
			if out.BatchID == nil {
				out.BatchID = linkedBatch
			}
		}
	}

	if out.BatchID != nil {
		if out.BatchName == nil {
			batch, err := s.GetBatch(ctx, storeID, *out.BatchID, false)
			if err != nil {
				if errors.Is(err, ErrNotFound) {
					return nil, fmt.Errorf("%w: preorder batch not found in this store", ErrInvalidInput)
				}
				return nil, err
			}
			name, slug := batch.Name, batch.Slug
			out.BatchName = &name
			out.BatchSlug = &slug
			out.ETAStart = batch.ETAStart
			out.ETAEnd = batch.ETAEnd
		}
		if out.CampaignID == nil {
			var cid uuid.UUID
			cerr := s.DB.QueryRow(ctx, `
				SELECT campaign_id FROM preorder_batches
				WHERE id=$1 AND store_id=$2 AND campaign_id IS NOT NULL
			`, *out.BatchID, storeID).Scan(&cid)
			if cerr == nil {
				out.CampaignID = &cid
			} else if !errors.Is(cerr, pgx.ErrNoRows) {
				return nil, cerr
			}
		}
	}

	return out, nil
}

type AdminCreateResult struct {
	Order         *Order   `json:"order"`
	Payment       *Payment `json:"payment,omitempty"`
	TrackingToken string   `json:"tracking_token,omitempty"`
	TrackingURL   string   `json:"tracking_url,omitempty"`
}

type OrderLineValidateError struct {
	Index   int    `json:"index"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

type BulkValidateResult struct {
	Valid  bool                    `json:"valid"`
	Errors []OrderLineValidateError `json:"errors"`
	Lines  []PricedLine            `json:"lines,omitempty"`
	TotalIDR int64                 `json:"total_idr,omitempty"`
}

type CatalogSuggestItem struct {
	ProductID    uuid.UUID  `json:"product_id"`
	VariantID    *uuid.UUID `json:"variant_id,omitempty"`
	Name         string     `json:"name"`
	SKU          *string    `json:"sku,omitempty"`
	VariantLabel *string    `json:"variant_label,omitempty"`
	UnitPriceIDR int64      `json:"unit_price_idr"`
	Status       string     `json:"status"`
}

func (s *Service) SuggestCatalog(ctx context.Context, storeID uuid.UUID, q string, limit int) ([]CatalogSuggestItem, error) {
	if limit <= 0 || limit > 25 {
		limit = 12
	}
	q = strings.TrimSpace(q)
	if q == "" {
		return []CatalogSuggestItem{}, nil
	}
	like := "%" + q + "%"
	rows, err := s.DB.Query(ctx, `
		SELECT p.id, v.id, p.name, COALESCE(NULLIF(v.sku,''), p.sku),
		       NULLIF(v.option_key,''),
		       COALESCE(v.price_override_idr, p.base_price_idr), p.status
		FROM products p
		LEFT JOIN product_variants v ON v.product_id=p.id AND v.store_id=p.store_id
		WHERE p.store_id=$1 AND p.status <> 'archived'
		  AND (
		    p.name ILIKE $2 OR COALESCE(p.sku,'') ILIKE $2
		    OR COALESCE(v.sku,'') ILIKE $2 OR COALESCE(v.option_key,'') ILIKE $2
		  )
		ORDER BY p.name ASC, v.option_key ASC NULLS FIRST
		LIMIT $3
	`, storeID, like, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CatalogSuggestItem
	for rows.Next() {
		var it CatalogSuggestItem
		var vid *uuid.UUID
		var sku, label *string
		if err := rows.Scan(&it.ProductID, &vid, &it.Name, &sku, &label, &it.UnitPriceIDR, &it.Status); err != nil {
			return nil, err
		}
		it.VariantID = vid
		it.SKU = sku
		it.VariantLabel = label
		out = append(out, it)
	}
	return out, rows.Err()
}

func (s *Service) BulkValidateLines(ctx context.Context, storeID uuid.UUID, items []CartLineInput) (*BulkValidateResult, error) {
	res := &BulkValidateResult{Valid: true}
	if len(items) == 0 {
		res.Valid = false
		res.Errors = append(res.Errors, OrderLineValidateError{Index: -1, Field: "items", Message: "At least one line is required."})
		return res, nil
	}
	for i, it := range items {
		if it.ProductID == uuid.Nil {
			res.Valid = false
			res.Errors = append(res.Errors, OrderLineValidateError{Index: i, Field: "product_id", Message: "Product is required."})
		}
		if it.Quantity <= 0 {
			res.Valid = false
			res.Errors = append(res.Errors, OrderLineValidateError{Index: i, Field: "quantity", Message: "Quantity must be > 0."})
		}
		if it.ProductID != uuid.Nil && it.VariantID == nil {
			var variantCount int
			err := s.DB.QueryRow(ctx, `
				SELECT COUNT(*) FROM product_variants WHERE product_id=$1 AND store_id=$2
			`, it.ProductID, storeID).Scan(&variantCount)
			if err != nil {
				return nil, err
			}
			if variantCount > 0 {
				res.Valid = false
				res.Errors = append(res.Errors, OrderLineValidateError{
					Index: i, Field: "variant_id", Message: "Select a product option before continuing.",
				})
			}
		}
	}
	if !res.Valid {
		return res, nil
	}
	if _, _, err := resolveCartGroup(items); err != nil {
		res.Valid = false
		res.Errors = append(res.Errors, OrderLineValidateError{Index: -1, Field: "items", Message: "Mix of ready-stock and preorder lines is not allowed."})
		return res, nil
	}
	lines, subtotal, err := s.priceLines(ctx, storeID, items)
	if err != nil {
		res.Valid = false
		msg := "Unable to price lines."
		switch {
		case errors.Is(err, ErrNotFound):
			msg = "Product or variant not found."
		case errors.Is(err, ErrInsufficientStock):
			msg = "Insufficient stock."
		case errors.Is(err, ErrBatchClosed), errors.Is(err, ErrQuotaExceeded):
			msg = err.Error()
		}
		res.Errors = append(res.Errors, OrderLineValidateError{Index: -1, Field: "items", Message: msg})
		return res, nil
	}
	res.Lines = lines
	res.TotalIDR = subtotal
	return res, nil
}

func (s *Service) CreateAdminOrder(ctx context.Context, storeID, actorID uuid.UUID, in AdminCreateOrderInput) (*AdminCreateResult, error) {
	source := strings.TrimSpace(in.Source)
	if source == "" {
		source = OrderSourceAdmin
	}
	if source != OrderSourceAdmin && source != OrderSourceCSVImport {
		return nil, ErrInvalidInput
	}
	if strings.TrimSpace(in.IdempotencyKey) == "" {
		return nil, ErrInvalidInput
	}

	var existingID uuid.UUID
	err := s.DB.QueryRow(ctx, `SELECT id FROM orders WHERE idempotency_key=$1`, in.IdempotencyKey).Scan(&existingID)
	if err == nil {
		o, err := s.GetOrder(ctx, storeID, existingID)
		if err != nil {
			return nil, err
		}
		pay, _ := s.GetLatestPayment(ctx, storeID, existingID)
		return &AdminCreateResult{Order: o, Payment: pay}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	name := strings.TrimSpace(in.CustomerName)
	phone := in.CustomerPhone
	var emailAddr string
	var linkedCustomerID *uuid.UUID

	if in.CustomerID != nil {
		var cName string
		var cEmail, cPhone *string
		err = s.DB.QueryRow(ctx, `
			SELECT name, email, phone FROM customers WHERE id=$1 AND store_id=$2
		`, *in.CustomerID, storeID).Scan(&cName, &cEmail, &cPhone)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: customer not found in this store", ErrInvalidInput)
		}
		if err != nil {
			return nil, err
		}
		linkedCustomerID = in.CustomerID
		if name == "" {
			name = cName
		}
		if len(strings.TrimSpace(name)) < 2 {
			return nil, ErrInvalidInput
		}
		if phone == nil || strings.TrimSpace(*phone) == "" {
			phone = cPhone
		}
		if cEmail != nil && strings.TrimSpace(*cEmail) != "" {
			_, emailAddr, _ = validate.NormalizeEmail(*cEmail)
		}
		if emailAddr == "" {
			if _, normalized, ok := validate.NormalizeEmail(in.CustomerEmail); ok {
				emailAddr = normalized
			}
		}
		if emailAddr == "" {
			// orders.customer_email is NOT NULL; phone-only CRM customers use a stable snapshot.
			emailAddr = fmt.Sprintf("customer+%s@customers.local", in.CustomerID.String())
		}
	} else {
		_, normalized, emailOK := validate.NormalizeEmail(in.CustomerEmail)
		if len(name) < 2 || !emailOK {
			return nil, ErrInvalidInput
		}
		emailAddr = normalized
	}
	in.CustomerPhone = phone

	if in.FulfillmentMethod == "" {
		in.FulfillmentMethod = FulfillmentPickup
	}
	if in.FulfillmentMethod != FulfillmentShipping && in.FulfillmentMethod != FulfillmentFreeShipping && in.FulfillmentMethod != FulfillmentPickup {
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

	sellingModel, batchID, err := resolveCartGroup(in.Items)
	if err != nil {
		return nil, err
	}
	lines, subtotal, err := s.priceLines(ctx, storeID, in.Items)
	if err != nil {
		return nil, err
	}

	var currency string
	err = s.DB.QueryRow(ctx, `SELECT currency FROM stores WHERE id=$1`, storeID).Scan(&currency)
	if err != nil {
		return nil, err
	}

	shipFee, err := s.resolveShipping(ctx, storeID, in.FulfillmentMethod)
	if err != nil {
		// Admin may create before fulfillment is fully configured; default 0 for pickup.
		if in.FulfillmentMethod == FulfillmentPickup {
			shipFee = 0
		} else {
			return nil, err
		}
	}
	itemQty := 0
	for _, l := range lines {
		itemQty += l.Quantity
	}
	eval, evalErr := s.EvaluateCheckoutRules(ctx, storeID, sellingModel, batchID, subtotal, itemQty, shipFee)
	if evalErr != nil {
		return nil, evalErr
	}
	total := eval.TotalIDR

	rels, err := s.resolveAdminOrderRelations(ctx, storeID, in.CampaignID, in.PreorderBatchID, batchID)
	if err != nil {
		return nil, err
	}
	campaignID := rels.CampaignID
	if rels.BatchID != nil {
		batchID = rels.BatchID
	}
	batchName, batchSlug := rels.BatchName, rels.BatchSlug
	etaStart, etaEnd := rels.ETAStart, rels.ETAEnd
	// Order-level preorder link (without batch_id on lines) still runs as preorder ops.
	if sellingModel == "ready_stock" && batchID != nil {
		sellingModel = "preorder"
	}

	paymentOption := strings.TrimSpace(strings.ToLower(in.PaymentOption))
	if paymentOption == "" {
		paymentOption = PaymentOptionBankTransfer
	}
	if paymentOption != PaymentOptionBankTransfer && paymentOption != PaymentOptionPaidByAdmin {
		return nil, ErrInvalidInput
	}
	paidByAdmin := paymentOption == PaymentOptionPaidByAdmin

	var pm *PaymentMethod
	if !paidByAdmin {
		if in.PaymentMethodID != nil {
			pm, err = s.getPaymentMethod(ctx, storeID, *in.PaymentMethodID)
			if err != nil || !pm.IsEnabled {
				return nil, ErrInvalidInput
			}
		} else {
			// Prefer first enabled method so bank-transfer evidence path stays available.
			methods, _ := s.ListPaymentMethods(ctx, storeID)
			for i := range methods {
				if methods[i].IsEnabled {
					pm = &methods[i]
					break
				}
			}
		}
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var customerID uuid.UUID
	if linkedCustomerID != nil {
		customerID = *linkedCustomerID
		_, err = tx.Exec(ctx, `
			UPDATE customers SET
				name=$3,
				phone=COALESCE($4, phone),
				first_order_at=COALESCE(first_order_at, now()),
				last_order_at=now(),
				total_orders=total_orders+1,
				updated_at=now()
			WHERE id=$1 AND store_id=$2
		`, customerID, storeID, name, in.CustomerPhone)
		if err != nil {
			return nil, err
		}
	} else {
		customerID, err = upsertCustomerTx(ctx, tx, storeID, name, emailAddr, in.CustomerPhone)
		if err != nil {
			return nil, err
		}
	}

	orderNumber, err := nextOrderNumberTx(ctx, tx, storeID)
	if err != nil {
		return nil, err
	}

	now := s.now()
	orderStatus := OrderConfirmed
	var confirmedAt *time.Time
	confirmedAt = &now
	if eval.RequiresApproval {
		orderStatus = OrderPendingConfirmation
		confirmedAt = nil
	}
	initialFulfillment := FulfillmentUnfulfilled
	if sellingModel == "preorder" {
		initialFulfillment = FulfillmentPurchasing
	}
	initialPayment := PaymentUnpaid
	if paidByAdmin {
		initialPayment = PaymentPaid
	}

	var deadline *time.Time
	if pm != nil && !paidByAdmin {
		d := now.Add(time.Duration(pm.PaymentDeadlineHours) * time.Hour)
		deadline = &d
	}

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
			original_total_idr, payable_idr,
			fulfillment_method, customer_name, customer_email, customer_phone,
			shipping_recipient, shipping_phone, shipping_line1, shipping_line2,
			shipping_city, shipping_province, shipping_postal_code, shipping_country,
			customer_notes, internal_notes, payment_deadline_at, confirmed_at,
			preorder_batch_id, batch_name_snapshot, batch_slug_snapshot,
			batch_eta_start_snapshot, batch_eta_end_snapshot,
			requires_approval, source, created_by_user_id, campaign_id
		) VALUES (
			$1,$2,$3,$4,$5,
			$6,$7,$8,$9,
			$10,$11,$12,$13,
			$13,$13,
			$14,$15,$16,$17,
			$18,$19,$20,$21,
			$22,$23,$24,$25,
			$26,$27,$28,$29,
			$30,$31,$32,
			$33,$34,
			$35,$36,$37,$38
		)
		RETURNING id
	`, storeID, customerID, orderNumber, in.IdempotencyKey, sellingModel,
		orderStatus, initialPayment, initialFulfillment, currency,
		subtotal, shipFee, eval.HandlingFeeIDR, total,
		in.FulfillmentMethod, name, emailAddr, in.CustomerPhone,
		shipRecipient, shipPhone, shipLine1, shipLine2,
		shipCity, shipProvince, shipPostal, shipCountry,
		in.CustomerNotes, in.InternalNotes, deadline, confirmedAt,
		batchID, batchName, batchSlug,
		etaStart, etaEnd,
		eval.RequiresApproval, source, actorID, campaignID,
	).Scan(&orderID)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO order_status_history (store_id, order_id, dimension, from_status, to_status, actor_user_id, reason)
		VALUES ($1,$2,'order',NULL,$3,$4,$5),
		       ($1,$2,'payment',NULL,$6,$4,$5),
		       ($1,$2,'fulfillment',NULL,$7,$4,$5)
	`, storeID, orderID, orderStatus, actorID, source, initialPayment, initialFulfillment)
	if err != nil {
		return nil, err
	}

	for _, line := range lines {
		var itemID uuid.UUID
		err = tx.QueryRow(ctx, `
			INSERT INTO order_items (
				store_id, order_id, product_id, variant_id, batch_id, product_name, product_slug,
				sku, variant_label, unit_price_idr, quantity, original_quantity, line_total_idr, batch_price_idr,
				availability_status, allocated_qty, packed_qty
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$11,$12,$13,'available',0,0)
			RETURNING id
		`, storeID, orderID, line.ProductID, line.VariantID, line.BatchID, line.ProductName, line.ProductSlug,
			line.SKU, line.VariantLabel, line.UnitPriceIDR, line.Quantity, line.LineTotalIDR, line.UnitPriceIDR,
		).Scan(&itemID)
		if err != nil {
			return nil, err
		}
		_, _ = tx.Exec(ctx, `
			INSERT INTO order_item_events (
				store_id, order_id, order_item_id, event_type, from_quantity, to_quantity, actor_user_id, reason
			) VALUES ($1,$2,$3,'created',NULL,$4,$5,$6)
		`, storeID, orderID, itemID, line.Quantity, actorID, source)

		if line.InventoryMode == "finite" {
			var stock int
			err = tx.QueryRow(ctx, `
				SELECT COALESCE(stock_quantity,0) FROM products WHERE id=$1 AND store_id=$2 FOR UPDATE
			`, line.ProductID, storeID).Scan(&stock)
			if err != nil {
				return nil, err
			}
			if line.VariantID != nil {
				_ = tx.QueryRow(ctx, `
					SELECT COALESCE(stock_quantity, $2) FROM product_variants WHERE id=$1 FOR UPDATE
				`, *line.VariantID, stock).Scan(&stock)
			}
			var reserved int
			err = tx.QueryRow(ctx, `
				SELECT COALESCE(SUM(quantity),0) FROM inventory_reservations
				WHERE store_id=$1 AND product_id=$2 AND status='active' AND batch_id IS NULL
				  AND (($3::uuid IS NULL AND variant_id IS NULL) OR variant_id=$3)
			`, storeID, line.ProductID, line.VariantID).Scan(&reserved)
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
			`, storeID, orderID, itemID, line.ProductID, line.VariantID, line.Quantity)
			if err != nil {
				return nil, err
			}
		} else if line.InventoryMode == "quota" && line.BatchID != nil {
			var limit *int
			err = tx.QueryRow(ctx, `
				SELECT quantity_limit FROM preorder_batch_products
				WHERE store_id=$1 AND batch_id=$2 AND product_id=$3 FOR UPDATE
			`, storeID, *line.BatchID, line.ProductID).Scan(&limit)
			if err != nil {
				return nil, err
			}
			if limit != nil {
				var reserved int
				err = tx.QueryRow(ctx, `
					SELECT COALESCE(SUM(quantity),0) FROM inventory_reservations
					WHERE store_id=$1 AND batch_id=$2 AND product_id=$3 AND status='active'
					  AND (($4::uuid IS NULL AND variant_id IS NULL) OR variant_id=$4)
				`, storeID, *line.BatchID, line.ProductID, line.VariantID).Scan(&reserved)
				if err != nil {
					return nil, err
				}
				if reserved+line.Quantity > *limit {
					return nil, ErrQuotaExceeded
				}
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO inventory_reservations (
					store_id, order_id, order_item_id, product_id, variant_id, batch_id, quantity, status
				) VALUES ($1,$2,$3,$4,$5,$6,$7,'active')
			`, storeID, orderID, itemID, line.ProductID, line.VariantID, line.BatchID, line.Quantity)
			if err != nil {
				return nil, err
			}
		}
	}

	var paymentID *uuid.UUID
	if paidByAdmin {
		note := "Paid by admin"
		var pid uuid.UUID
		err = tx.QueryRow(ctx, `
			INSERT INTO payments (
				store_id, order_id, payment_method_id, method_type, amount_idr, status,
				instructions_snapshot, verified_at, verified_by, payment_kind, note
			) VALUES ($1,$2,NULL,$3,$4,'verified',$5,now(),$6,$7,$5)
			RETURNING id
		`, storeID, orderID, MethodTypePaidByAdmin, total, note, actorID, PaymentKindFull,
		).Scan(&pid)
		if err != nil {
			return nil, err
		}
		paymentID = &pid
		if _, err := s.recomputeSettlementTx(ctx, tx, storeID, orderID); err != nil {
			return nil, err
		}
	} else if pm != nil {
		var pid uuid.UUID
		err = tx.QueryRow(ctx, `
			INSERT INTO payments (
				store_id, order_id, payment_method_id, method_type, amount_idr, status,
				bank_name_snapshot, account_holder_snapshot, account_number_snapshot, instructions_snapshot
			) VALUES ($1,$2,$3,$4,$5,'pending',$6,$7,$8,$9)
			RETURNING id
		`, storeID, orderID, pm.ID, MethodTypeBankTransfer, total, pm.BankName, pm.AccountHolder, pm.AccountNumber, pm.Instructions,
		).Scan(&pid)
		if err != nil {
			return nil, err
		}
		paymentID = &pid
	}

	rawToken, tokenHash, err := auth.NewToken()
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO order_tracking_tokens (store_id, order_id, token_hash)
		VALUES ($1,$2,$3)
	`, storeID, orderID, tokenHash)
	if err != nil {
		return nil, err
	}

	afterSnap, _ := json.Marshal(map[string]any{
		"order_number": orderNumber, "total_idr": total, "source": source, "item_count": len(lines),
	})
	_, err = tx.Exec(ctx, `
		INSERT INTO order_revisions (
			store_id, order_id, revision_number, change_type, summary, after_snapshot, actor_user_id, reason
		) VALUES ($1,$2,1,'created',$3,$4,$5,$6)
	`, storeID, orderID, fmt.Sprintf("Order created via %s", source), afterSnap, actorID, source)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	_ = paymentID
	if s.Outbox != nil {
		_ = s.Outbox.Write(ctx, &storeID, "order.order_created", map[string]any{
			"order_id": orderID.String(), "order_number": orderNumber, "total_idr": total, "source": source,
		}, "order_created:"+orderID.String())
	}

	order, err := s.GetOrder(ctx, storeID, orderID)
	if err != nil {
		return nil, err
	}
	order.TrackingToken = rawToken
	if s.URLs != nil {
		order.TrackingURL = s.URLs.Tracking(rawToken)
	}
	pay, _ := s.GetLatestPayment(ctx, storeID, orderID)

	if !in.SkipEmails {
		st := &publicStoreMeta{ID: storeID, Currency: currency}
		_ = s.DB.QueryRow(ctx, `
			SELECT s.name, s.slug, COALESCE(u.email,''), u.display_name
			FROM stores s
			LEFT JOIN store_memberships sm ON sm.store_id=s.id AND sm.role='owner' AND sm.status='active'
			LEFT JOIN users u ON u.id=sm.user_id
			WHERE s.id=$1
		`, storeID).Scan(&st.Name, &st.Slug, &st.OwnerEmail, &st.OwnerName)
		s.enqueueOrderEmails(ctx, st, order, pay, rawToken)
	}

	return &AdminCreateResult{
		Order: order, Payment: pay, TrackingToken: rawToken, TrackingURL: order.TrackingURL,
	}, nil
}

// --- Drafts (spreadsheet autosave) ---

type SaveDraftInput struct {
	Title   string          `json:"title"`
	Payload json.RawMessage `json:"payload"`
}

func (s *Service) ListOrderDrafts(ctx context.Context, storeID, userID uuid.UUID) ([]AdminOrderDraft, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT id, store_id, created_by_user_id, title, payload, last_validated_at, validation_errors, created_at, updated_at
		FROM admin_order_drafts
		WHERE store_id=$1 AND created_by_user_id=$2
		ORDER BY updated_at DESC
		LIMIT 50
	`, storeID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AdminOrderDraft
	for rows.Next() {
		var d AdminOrderDraft
		if err := rows.Scan(
			&d.ID, &d.StoreID, &d.CreatedByUserID, &d.Title, &d.Payload,
			&d.LastValidatedAt, &d.ValidationErrors, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Service) GetOrderDraft(ctx context.Context, storeID, userID, draftID uuid.UUID) (*AdminOrderDraft, error) {
	var d AdminOrderDraft
	err := s.DB.QueryRow(ctx, `
		SELECT id, store_id, created_by_user_id, title, payload, last_validated_at, validation_errors, created_at, updated_at
		FROM admin_order_drafts WHERE id=$1 AND store_id=$2 AND created_by_user_id=$3
	`, draftID, storeID, userID).Scan(
		&d.ID, &d.StoreID, &d.CreatedByUserID, &d.Title, &d.Payload,
		&d.LastValidatedAt, &d.ValidationErrors, &d.CreatedAt, &d.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *Service) SaveOrderDraft(ctx context.Context, storeID, userID uuid.UUID, draftID *uuid.UUID, in SaveDraftInput) (*AdminOrderDraft, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = "Untitled draft"
	}
	payload := in.Payload
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	if draftID != nil {
		tag, err := s.DB.Exec(ctx, `
			UPDATE admin_order_drafts
			SET title=$4, payload=$5, updated_at=now()
			WHERE id=$1 AND store_id=$2 AND created_by_user_id=$3
		`, *draftID, storeID, userID, title, payload)
		if err != nil {
			return nil, err
		}
		if tag.RowsAffected() == 0 {
			return nil, ErrNotFound
		}
		return s.GetOrderDraft(ctx, storeID, userID, *draftID)
	}
	var id uuid.UUID
	err := s.DB.QueryRow(ctx, `
		INSERT INTO admin_order_drafts (store_id, created_by_user_id, title, payload)
		VALUES ($1,$2,$3,$4)
		RETURNING id
	`, storeID, userID, title, payload).Scan(&id)
	if err != nil {
		return nil, err
	}
	return s.GetOrderDraft(ctx, storeID, userID, id)
}

func (s *Service) ValidateOrderDraft(ctx context.Context, storeID, userID, draftID uuid.UUID) (*AdminOrderDraft, *BulkValidateResult, error) {
	d, err := s.GetOrderDraft(ctx, storeID, userID, draftID)
	if err != nil {
		return nil, nil, err
	}
	var payload struct {
		Items []CartLineInput `json:"items"`
	}
	_ = json.Unmarshal(d.Payload, &payload)
	vres, err := s.BulkValidateLines(ctx, storeID, payload.Items)
	if err != nil {
		return nil, nil, err
	}
	errsJSON, _ := json.Marshal(vres.Errors)
	now := s.now()
	_, err = s.DB.Exec(ctx, `
		UPDATE admin_order_drafts
		SET last_validated_at=$3, validation_errors=$4, updated_at=now()
		WHERE id=$1 AND store_id=$2
	`, draftID, storeID, now, errsJSON)
	if err != nil {
		return nil, nil, err
	}
	d, err = s.GetOrderDraft(ctx, storeID, userID, draftID)
	return d, vres, err
}

func (s *Service) DeleteOrderDraft(ctx context.Context, storeID, userID, draftID uuid.UUID) error {
	tag, err := s.DB.Exec(ctx, `
		DELETE FROM admin_order_drafts WHERE id=$1 AND store_id=$2 AND created_by_user_id=$3
	`, draftID, storeID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
