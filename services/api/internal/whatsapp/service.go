package whatsapp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rubicahq/pikko/services/api/internal/money"
	"github.com/rubicahq/pikko/services/api/internal/urls"
)

var (
	ErrNotFound = errors.New("whatsapp template not found")
	ErrInvalid  = errors.New("invalid whatsapp input")
)

type Service struct {
	DB   *pgxpool.Pool
	URLs *urls.Generator
}

type ContextInput struct {
	OrderID    *uuid.UUID
	CustomerID *uuid.UUID
	VendorID   *uuid.UUID
	Overrides  map[string]string
}

type ResolveResult struct {
	Template   Template          `json:"template"`
	Variables  map[string]string `json:"variables"`
	Missing    []string          `json:"missing"`
	Body       string            `json:"body"`
	Phone      string            `json:"phone,omitempty"`
	PhoneE164  string            `json:"phone_e164,omitempty"`
	DeepLink   string            `json:"deep_link"`
	OrderID    *uuid.UUID        `json:"order_id,omitempty"`
	CustomerID *uuid.UUID        `json:"customer_id,omitempty"`
	VendorID   *uuid.UUID        `json:"vendor_id,omitempty"`
}

type LogInput struct {
	TemplateCode string
	Action       string // open | copy | preview
	Phone        string
	Body         string
	DeepLink     string
	OrderID      *uuid.UUID
	CustomerID   *uuid.UUID
	VendorID     *uuid.UUID
	OpenedBy     uuid.UUID
}

type SendLog struct {
	ID           uuid.UUID  `json:"id"`
	StoreID      uuid.UUID  `json:"store_id"`
	TemplateCode string     `json:"template_code"`
	Audience     string     `json:"audience"`
	PhoneE164    *string    `json:"phone_e164,omitempty"`
	RenderedBody string     `json:"rendered_body"`
	DeepLink     string     `json:"deep_link"`
	OrderID      *uuid.UUID `json:"order_id,omitempty"`
	CustomerID   *uuid.UUID `json:"customer_id,omitempty"`
	VendorID     *uuid.UUID `json:"vendor_id,omitempty"`
	OpenedBy     *uuid.UUID `json:"opened_by,omitempty"`
	Action       string     `json:"action"`
	CreatedAt    time.Time  `json:"created_at"`
}

func (s *Service) ListTemplates(audience string) []Template {
	return List(audience)
}

func (s *Service) GetTemplate(code string) (Template, error) {
	t, ok := ByCode(code)
	if !ok {
		return Template{}, ErrNotFound
	}
	return t, nil
}

func (s *Service) Resolve(ctx context.Context, storeID uuid.UUID, code string, in ContextInput) (*ResolveResult, error) {
	t, ok := ByCode(code)
	if !ok {
		return nil, ErrNotFound
	}
	vars, phone, orderID, customerID, vendorID, err := s.buildVars(ctx, storeID, t, in)
	if err != nil {
		return nil, err
	}
	if in.Overrides != nil {
		for k, v := range in.Overrides {
			vars[k] = v
		}
		if p, ok := in.Overrides["phone"]; ok && strings.TrimSpace(p) != "" {
			phone = p
		}
	}
	body := Render(t.Body, vars)
	digits := NormalizePhoneDigits(phone)
	return &ResolveResult{
		Template:   t,
		Variables:  vars,
		Missing:    MissingVariables(t, vars),
		Body:       body,
		Phone:      phone,
		PhoneE164:  digits,
		DeepLink:   DeepLink(digits, body),
		OrderID:    orderID,
		CustomerID: customerID,
		VendorID:   vendorID,
	}, nil
}

func (s *Service) LogSend(ctx context.Context, storeID uuid.UUID, in LogInput) (*SendLog, error) {
	t, ok := ByCode(in.TemplateCode)
	if !ok {
		return nil, ErrNotFound
	}
	action := in.Action
	if action == "" {
		action = "open"
	}
	switch action {
	case "open", "copy", "preview":
	default:
		return nil, fmt.Errorf("%w: invalid action", ErrInvalid)
	}
	body := in.Body
	if body == "" {
		body = t.Body
	}
	link := in.DeepLink
	if link == "" {
		link = DeepLink(in.Phone, body)
	}
	digits := NormalizePhoneDigits(in.Phone)
	var phonePtr *string
	if digits != "" {
		phonePtr = &digits
	}
	var openedBy *uuid.UUID
	if in.OpenedBy != uuid.Nil {
		openedBy = &in.OpenedBy
	}
	var log SendLog
	err := s.DB.QueryRow(ctx, `
		INSERT INTO whatsapp_template_sends (
			store_id, template_code, audience, phone_e164, rendered_body, deep_link,
			order_id, customer_id, vendor_id, opened_by, action
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id, store_id, template_code, audience, phone_e164, rendered_body, deep_link,
		          order_id, customer_id, vendor_id, opened_by, action, created_at
	`, storeID, t.Code, string(t.Audience), phonePtr, body, link,
		in.OrderID, in.CustomerID, in.VendorID, openedBy, action,
	).Scan(
		&log.ID, &log.StoreID, &log.TemplateCode, &log.Audience, &log.PhoneE164, &log.RenderedBody, &log.DeepLink,
		&log.OrderID, &log.CustomerID, &log.VendorID, &log.OpenedBy, &log.Action, &log.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &log, nil
}

func (s *Service) ListSends(ctx context.Context, storeID uuid.UUID, limit int) ([]SendLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	rows, err := s.DB.Query(ctx, `
		SELECT id, store_id, template_code, audience, phone_e164, rendered_body, deep_link,
		       order_id, customer_id, vendor_id, opened_by, action, created_at
		FROM whatsapp_template_sends
		WHERE store_id=$1
		ORDER BY created_at DESC
		LIMIT $2
	`, storeID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SendLog{}
	for rows.Next() {
		var log SendLog
		if err := rows.Scan(
			&log.ID, &log.StoreID, &log.TemplateCode, &log.Audience, &log.PhoneE164, &log.RenderedBody, &log.DeepLink,
			&log.OrderID, &log.CustomerID, &log.VendorID, &log.OpenedBy, &log.Action, &log.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, log)
	}
	return out, rows.Err()
}

func (s *Service) buildVars(ctx context.Context, storeID uuid.UUID, t Template, in ContextInput) (map[string]string, string, *uuid.UUID, *uuid.UUID, *uuid.UUID, error) {
	vars := map[string]string{
		"store_name":          "",
		"customer_name":       "",
		"customer_email":      "",
		"order_number":        "",
		"amount_formatted":    "",
		"tracking_url":        "",
		"portal_url":          "",
		"item_summary":        "",
		"batch_name":          "",
		"eta_text":            "",
		"fulfillment_method":  "",
		"carrier":             "",
		"tracking_number":     "",
		"pickup_notes":        "",
		"shipping_address":    "",
		"reason":              "",
		"vendor_contact_name": "",
		"vendor_name":         "",
	}
	var phone string
	var orderID, customerID, vendorID *uuid.UUID

	var storeName, storeSlug string
	err := s.DB.QueryRow(ctx, `SELECT name, slug FROM stores WHERE id=$1`, storeID).Scan(&storeName, &storeSlug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", nil, nil, nil, ErrNotFound
		}
		return nil, "", nil, nil, nil, err
	}
	vars["store_name"] = storeName
	if s.URLs != nil {
		vars["portal_url"] = s.URLs.PortalHome(storeSlug)
	}

	if in.VendorID != nil {
		vendorID = in.VendorID
		var name string
		var contactName, contactPhone *string
		err := s.DB.QueryRow(ctx, `
			SELECT name, contact_name, contact_phone FROM vendors WHERE id=$1 AND store_id=$2
		`, *in.VendorID, storeID).Scan(&name, &contactName, &contactPhone)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, "", nil, nil, nil, fmt.Errorf("%w: vendor", ErrNotFound)
			}
			return nil, "", nil, nil, nil, err
		}
		vars["vendor_name"] = name
		if contactName != nil && *contactName != "" {
			vars["vendor_contact_name"] = *contactName
		} else {
			vars["vendor_contact_name"] = name
		}
		if contactPhone != nil {
			phone = *contactPhone
		}
	}

	if in.CustomerID != nil && in.OrderID == nil {
		customerID = in.CustomerID
		var name, email string
		var custPhone *string
		err := s.DB.QueryRow(ctx, `
			SELECT name, email, phone FROM customers WHERE id=$1 AND store_id=$2
		`, *in.CustomerID, storeID).Scan(&name, &email, &custPhone)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, "", nil, nil, nil, fmt.Errorf("%w: customer", ErrNotFound)
			}
			return nil, "", nil, nil, nil, err
		}
		vars["customer_name"] = name
		vars["customer_email"] = email
		if custPhone != nil {
			phone = *custPhone
		}
	}

	if in.OrderID != nil {
		orderID = in.OrderID
		var (
			orderNumber, custName, custEmail, fulfillmentMethod string
			grandTotal                                          int64
			custPhone, shippingRecipient, shippingPhone         *string
			shippingLine1, shippingLine2, shippingCity          *string
			shippingProvince, shippingPostal                    *string
			custID                                              *uuid.UUID
		)
		err := s.DB.QueryRow(ctx, `
			SELECT o.order_number, o.customer_name, o.customer_email, o.customer_phone,
			       o.fulfillment_method, o.total_idr, o.customer_id,
			       o.shipping_recipient, o.shipping_phone, o.shipping_line1, o.shipping_line2,
			       o.shipping_city, o.shipping_province, o.shipping_postal_code
			FROM orders o
			WHERE o.id=$1 AND o.store_id=$2
		`, *in.OrderID, storeID).Scan(
			&orderNumber, &custName, &custEmail, &custPhone,
			&fulfillmentMethod, &grandTotal, &custID,
			&shippingRecipient, &shippingPhone, &shippingLine1, &shippingLine2,
			&shippingCity, &shippingProvince, &shippingPostal,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, "", nil, nil, nil, fmt.Errorf("%w: order", ErrNotFound)
			}
			return nil, "", nil, nil, nil, err
		}
		customerID = custID
		vars["order_number"] = orderNumber
		vars["customer_name"] = custName
		vars["customer_email"] = custEmail
		vars["fulfillment_method"] = fulfillmentMethod
		vars["amount_formatted"] = money.FormatIDR(grandTotal)
		if custPhone != nil && *custPhone != "" {
			phone = *custPhone
		} else if shippingPhone != nil {
			phone = *shippingPhone
		}
		// Guest tracking tokens are hashed-at-rest; portal is the recoverable customer link.
		if vars["portal_url"] != "" {
			vars["tracking_url"] = vars["portal_url"]
		}
		vars["shipping_address"] = formatAddress(shippingRecipient, shippingLine1, shippingLine2, shippingCity, shippingProvince, shippingPostal)

		items, _ := s.orderItemSummary(ctx, storeID, *in.OrderID)
		vars["item_summary"] = items

		var carrier, trackingNumber *string
		_ = s.DB.QueryRow(ctx, `
			SELECT carrier, tracking_number FROM shipments
			WHERE store_id=$1 AND order_id=$2 AND status <> 'cancelled'
			ORDER BY created_at DESC LIMIT 1
		`, storeID, *in.OrderID).Scan(&carrier, &trackingNumber)
		if carrier != nil {
			vars["carrier"] = *carrier
		}
		if trackingNumber != nil {
			vars["tracking_number"] = *trackingNumber
		}

		var batchName string
		var etaStart, etaEnd *time.Time
		errBatch := s.DB.QueryRow(ctx, `
			SELECT b.name, b.eta_start, b.eta_end
			FROM order_items oi
			JOIN preorder_batches b ON b.id = oi.preorder_batch_id
			WHERE oi.order_id=$1 AND oi.store_id=$2 AND oi.preorder_batch_id IS NOT NULL
			LIMIT 1
		`, *in.OrderID, storeID).Scan(&batchName, &etaStart, &etaEnd)
		if errBatch == nil {
			vars["batch_name"] = batchName
			vars["eta_text"] = formatETA(etaStart, etaEnd)
		}

		var pickupNotes *string
		_ = s.DB.QueryRow(ctx, `
			SELECT pickup_instructions FROM store_fulfillment_settings WHERE store_id=$1
		`, storeID).Scan(&pickupNotes)
		if pickupNotes != nil {
			vars["pickup_notes"] = *pickupNotes
		}
	}

	// Default placeholders so preview is readable without context
	if t.Audience == AudienceVendor && vars["vendor_contact_name"] == "" {
		vars["vendor_contact_name"] = "Kak"
	}
	if t.Audience == AudienceCustomer && vars["customer_name"] == "" {
		vars["customer_name"] = "Kak"
	}
	if vars["item_summary"] == "" {
		vars["item_summary"] = "—"
	}
	if vars["amount_formatted"] == "" {
		vars["amount_formatted"] = money.FormatIDR(0)
	}
	if vars["reason"] == "" {
		vars["reason"] = "bukti kurang jelas / tidak sesuai"
	}
	if vars["batch_name"] == "" {
		vars["batch_name"] = "batch preorder"
	}
	if vars["eta_text"] == "" {
		vars["eta_text"] = "menyusul"
	}
	if vars["fulfillment_method"] == "" {
		vars["fulfillment_method"] = "pengiriman"
	}
	if vars["carrier"] == "" {
		vars["carrier"] = "kurir"
	}
	if vars["tracking_number"] == "" {
		vars["tracking_number"] = "—"
	}
	if vars["pickup_notes"] == "" {
		vars["pickup_notes"] = "hubungi kami untuk jadwal pengambilan"
	}
	if vars["shipping_address"] == "" {
		vars["shipping_address"] = "—"
	}
	if vars["order_number"] == "" {
		vars["order_number"] = "—"
	}
	if vars["customer_email"] == "" {
		vars["customer_email"] = "email terdaftar"
	}

	_ = vendorID
	return vars, phone, orderID, customerID, vendorID, nil
}

func (s *Service) orderItemSummary(ctx context.Context, storeID, orderID uuid.UUID) (string, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT product_name, COALESCE(variant_label, ''), quantity
		FROM order_items
		WHERE store_id=$1 AND order_id=$2
		ORDER BY created_at
		LIMIT 20
	`, storeID, orderID)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var lines []string
	for rows.Next() {
		var name, variant string
		var qty int
		if err := rows.Scan(&name, &variant, &qty); err != nil {
			return "", err
		}
		line := fmt.Sprintf("- %s x%d", name, qty)
		if variant != "" {
			line = fmt.Sprintf("- %s (%s) x%d", name, variant, qty)
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return "—", nil
	}
	return strings.Join(lines, "\n"), rows.Err()
}

func formatETA(start, end *time.Time) string {
	const layout = "2 Jan 2006"
	switch {
	case start != nil && end != nil:
		if start.Equal(*end) {
			return start.Format(layout)
		}
		return start.Format(layout) + " – " + end.Format(layout)
	case start != nil:
		return "mulai " + start.Format(layout)
	case end != nil:
		return "sampai " + end.Format(layout)
	default:
		return ""
	}
}

func formatAddress(recipient, line1, line2, city, province, postal *string) string {
	parts := make([]string, 0, 6)
	add := func(p *string) {
		if p != nil && strings.TrimSpace(*p) != "" {
			parts = append(parts, strings.TrimSpace(*p))
		}
	}
	add(recipient)
	add(line1)
	add(line2)
	add(city)
	add(province)
	add(postal)
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ", ")
}
