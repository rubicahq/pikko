package commerce

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// OrderCSVTemplate is the downloadable order import template (ORD-08).
const OrderCSVTemplate = `customer_name,customer_email,customer_phone,fulfillment_method,product_sku,variant_sku,quantity,campaign_slug,notes
Budi Santoso,budi@example.com,08123456789,pickup,TEE-001,,2,,WhatsApp order
Siti Aminah,siti@example.com,,shipping,BAG-PO,BAG-PO-RED,1,preorder-batch-1,Instagram DM
`

const maxOrderCSVBytes = 2 << 20
const maxOrderCSVRows = 200

type OrderCSVColumnMap struct {
	CustomerName      string `json:"customer_name"`
	CustomerEmail     string `json:"customer_email"`
	CustomerPhone     string `json:"customer_phone"`
	FulfillmentMethod string `json:"fulfillment_method"`
	ProductSKU        string `json:"product_sku"`
	VariantSKU        string `json:"variant_sku"`
	Quantity          string `json:"quantity"`
	CampaignSlug      string `json:"campaign_slug"`
	Notes             string `json:"notes"`
	ShippingLine1     string `json:"shipping_line1"`
	ShippingCity      string `json:"shipping_city"`
}

func DefaultOrderCSVColumnMap() OrderCSVColumnMap {
	return OrderCSVColumnMap{
		CustomerName: "customer_name", CustomerEmail: "customer_email", CustomerPhone: "customer_phone",
		FulfillmentMethod: "fulfillment_method", ProductSKU: "product_sku", VariantSKU: "variant_sku",
		Quantity: "quantity", CampaignSlug: "campaign_slug", Notes: "notes",
		ShippingLine1: "shipping_line1", ShippingCity: "shipping_city",
	}
}

type OrderCSVPreviewRow struct {
	RowNumber         int        `json:"row_number"`
	CustomerName      string     `json:"customer_name"`
	CustomerEmail     string     `json:"customer_email"`
	CustomerPhone     *string    `json:"customer_phone,omitempty"`
	FulfillmentMethod string     `json:"fulfillment_method"`
	ProductSKU        string     `json:"product_sku"`
	VariantSKU        string     `json:"variant_sku,omitempty"`
	Quantity          int        `json:"quantity"`
	CampaignSlug      string     `json:"campaign_slug,omitempty"`
	Notes             *string    `json:"notes,omitempty"`
	ShippingLine1     string     `json:"shipping_line1,omitempty"`
	ShippingCity      string     `json:"shipping_city,omitempty"`
	MatchedCustomerID *uuid.UUID `json:"matched_customer_id,omitempty"`
	MatchedProductID  *uuid.UUID `json:"matched_product_id,omitempty"`
	MatchedVariantID  *uuid.UUID `json:"matched_variant_id,omitempty"`
	MatchedCampaignID *uuid.UUID `json:"matched_campaign_id,omitempty"`
	MatchedBatchID    *uuid.UUID `json:"matched_batch_id,omitempty"`
	ProductName       string     `json:"product_name,omitempty"`
	UnitPriceIDR      int64      `json:"unit_price_idr,omitempty"`
	Errors            []string   `json:"errors,omitempty"`
	Warnings          []string   `json:"warnings,omitempty"`
}

type OrderCSVPreview struct {
	Headers []string            `json:"headers"`
	Rows    []OrderCSVPreviewRow `json:"rows"`
	Valid   int                 `json:"valid_count"`
	Invalid int                 `json:"invalid_count"`
}

type OrderCSVImportResult struct {
	Created   int                 `json:"created"`
	Failed    int                 `json:"failed"`
	OrderIDs  []uuid.UUID         `json:"order_ids"`
	RowErrors []OrderCSVPreviewRow `json:"row_errors,omitempty"`
}

func (s *Service) PreviewOrderCSV(ctx context.Context, storeID uuid.UUID, r io.Reader, size int64, colMap *OrderCSVColumnMap) (*OrderCSVPreview, error) {
	raw, err := readOrderCSVBytes(r, size)
	if err != nil {
		return nil, err
	}
	headers, records, err := parseOrderCSVRecords(raw)
	if err != nil {
		return nil, err
	}
	cm := DefaultOrderCSVColumnMap()
	if colMap != nil {
		cm = *colMap
	}
	preview := &OrderCSVPreview{Headers: headers}
	for i, rec := range records {
		row := mapOrderCSVRow(i+2, headers, rec, cm)
		s.matchOrderCSVRow(ctx, storeID, &row)
		if len(row.Errors) == 0 {
			preview.Valid++
		} else {
			preview.Invalid++
		}
		preview.Rows = append(preview.Rows, row)
	}
	return preview, nil
}

func (s *Service) ImportOrderCSV(ctx context.Context, storeID, actorID uuid.UUID, r io.Reader, size int64, colMap *OrderCSVColumnMap, idemPrefix string) (*OrderCSVImportResult, error) {
	preview, err := s.PreviewOrderCSV(ctx, storeID, r, size, colMap)
	if err != nil {
		return nil, err
	}
	if idemPrefix == "" {
		idemPrefix = "csv-import-" + uuid.NewString()
	}
	out := &OrderCSVImportResult{}
	// Group rows by customer email + fulfillment + campaign into one order each group
	type groupKey struct {
		email, fulfill, campaign string
	}
	groups := map[groupKey][]OrderCSVPreviewRow{}
	var groupOrder []groupKey
	for _, row := range preview.Rows {
		if len(row.Errors) > 0 {
			out.Failed++
			out.RowErrors = append(out.RowErrors, row)
			continue
		}
		k := groupKey{email: strings.ToLower(row.CustomerEmail), fulfill: row.FulfillmentMethod, campaign: row.CampaignSlug}
		if _, ok := groups[k]; !ok {
			groupOrder = append(groupOrder, k)
		}
		groups[k] = append(groups[k], row)
	}

	for gi, k := range groupOrder {
		rows := groups[k]
		first := rows[0]
		items := make([]CartLineInput, 0, len(rows))
		for _, row := range rows {
			if row.MatchedProductID == nil {
				continue
			}
			line := CartLineInput{
				ProductID: *row.MatchedProductID,
				Quantity:  row.Quantity,
				BatchID:   row.MatchedBatchID,
			}
			if row.MatchedVariantID != nil {
				line.VariantID = row.MatchedVariantID
			}
			items = append(items, line)
		}
		if len(items) == 0 {
			out.Failed += len(rows)
			continue
		}
		in := AdminCreateOrderInput{
			Items:             items,
			CustomerName:      first.CustomerName,
			CustomerEmail:     first.CustomerEmail,
			CustomerPhone:     first.CustomerPhone,
			FulfillmentMethod: first.FulfillmentMethod,
			CustomerNotes:     first.Notes,
			CampaignID:        first.MatchedCampaignID,
			IdempotencyKey:    fmt.Sprintf("%s-%d", idemPrefix, gi),
			Source:            OrderSourceCSVImport,
			SkipEmails:        true,
		}
		if first.FulfillmentMethod == FulfillmentShipping || first.FulfillmentMethod == FulfillmentFreeShipping {
			in.ShippingAddress = &CheckoutAddress{
				RecipientName: first.CustomerName,
				Line1:         first.ShippingLine1,
				City:          first.ShippingCity,
				Country:       "ID",
				Phone:         first.CustomerPhone,
			}
			if in.ShippingAddress.Line1 == "" {
				in.ShippingAddress.Line1 = "TBD"
			}
			if in.ShippingAddress.City == "" {
				in.ShippingAddress.City = "TBD"
			}
		}
		res, err := s.CreateAdminOrder(ctx, storeID, actorID, in)
		if err != nil {
			out.Failed += len(rows)
			first.Errors = append(first.Errors, err.Error())
			out.RowErrors = append(out.RowErrors, first)
			continue
		}
		out.Created++
		out.OrderIDs = append(out.OrderIDs, res.Order.ID)
	}
	return out, nil
}

func (s *Service) matchOrderCSVRow(ctx context.Context, storeID uuid.UUID, row *OrderCSVPreviewRow) {
	if row.CustomerName == "" {
		row.Errors = append(row.Errors, "customer_name required")
	}
	if row.CustomerEmail == "" {
		row.Errors = append(row.Errors, "customer_email required")
	} else {
		var cid uuid.UUID
		err := s.DB.QueryRow(ctx, `
			SELECT id FROM customers WHERE store_id=$1 AND lower(email)=lower($2)
		`, storeID, row.CustomerEmail).Scan(&cid)
		if err == nil {
			row.MatchedCustomerID = &cid
		} else {
			row.Warnings = append(row.Warnings, "customer will be created")
		}
	}
	if row.Quantity <= 0 {
		row.Errors = append(row.Errors, "quantity must be > 0")
	}
	fm := strings.ToLower(strings.TrimSpace(row.FulfillmentMethod))
	if fm == "" {
		fm = FulfillmentPickup
		row.FulfillmentMethod = fm
	}
	if fm != FulfillmentShipping && fm != FulfillmentFreeShipping && fm != FulfillmentPickup {
		row.Errors = append(row.Errors, "invalid fulfillment_method")
	} else {
		row.FulfillmentMethod = fm
	}
	if row.ProductSKU == "" {
		row.Errors = append(row.Errors, "product_sku required")
		return
	}
	var pid uuid.UUID
	var name string
	var price int64
	err := s.DB.QueryRow(ctx, `
		SELECT id, name, base_price_idr FROM products
		WHERE store_id=$1 AND status <> 'archived' AND lower(COALESCE(sku,''))=lower($2)
		LIMIT 1
	`, storeID, row.ProductSKU).Scan(&pid, &name, &price)
	if err != nil {
		// fallback name match
		err = s.DB.QueryRow(ctx, `
			SELECT id, name, base_price_idr FROM products
			WHERE store_id=$1 AND status <> 'archived' AND lower(name)=lower($2)
			LIMIT 1
		`, storeID, row.ProductSKU).Scan(&pid, &name, &price)
	}
	if err != nil {
		row.Errors = append(row.Errors, "product not matched")
		return
	}
	row.MatchedProductID = &pid
	row.ProductName = name
	row.UnitPriceIDR = price

	if row.VariantSKU != "" {
		var vid uuid.UUID
		var vprice *int64
		err = s.DB.QueryRow(ctx, `
			SELECT id, price_override_idr FROM product_variants
			WHERE store_id=$1 AND product_id=$2 AND (
				lower(COALESCE(sku,''))=lower($3) OR lower(option_key)=lower($3)
			)
			LIMIT 1
		`, storeID, pid, row.VariantSKU).Scan(&vid, &vprice)
		if err != nil {
			row.Errors = append(row.Errors, "variant not matched")
		} else {
			row.MatchedVariantID = &vid
			if vprice != nil {
				row.UnitPriceIDR = *vprice
			}
		}
	}

	if row.CampaignSlug != "" {
		var cid, bid uuid.UUID
		err = s.DB.QueryRow(ctx, `
			SELECT c.id, COALESCE(c.preorder_batch_id, '00000000-0000-0000-0000-000000000000'::uuid)
			FROM campaigns c
			WHERE c.store_id=$1 AND lower(c.slug)=lower($2)
			LIMIT 1
		`, storeID, row.CampaignSlug).Scan(&cid, &bid)
		if err != nil {
			row.Errors = append(row.Errors, "campaign not matched")
		} else {
			row.MatchedCampaignID = &cid
			zero := uuid.Nil
			if bid != zero {
				row.MatchedBatchID = &bid
			}
		}
	}
}

func readOrderCSVBytes(r io.Reader, size int64) ([]byte, error) {
	if size <= 0 || size > maxOrderCSVBytes {
		return nil, fmt.Errorf("%w: CSV exceeds size limit", ErrInvalidInput)
	}
	raw, err := io.ReadAll(io.LimitReader(r, maxOrderCSVBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > maxOrderCSVBytes {
		return nil, fmt.Errorf("%w: CSV exceeds size limit", ErrInvalidInput)
	}
	return bytes.TrimPrefix(raw, []byte("\xef\xbb\xbf")), nil
}

func parseOrderCSVRecords(raw []byte) ([]string, [][]string, error) {
	cr := csv.NewReader(bytes.NewReader(raw))
	cr.TrimLeadingSpace = true
	cr.FieldsPerRecord = -1
	all, err := cr.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("%w: invalid CSV", ErrInvalidInput)
	}
	if len(all) < 2 {
		return nil, nil, fmt.Errorf("%w: CSV needs header and at least one row", ErrInvalidInput)
	}
	if len(all)-1 > maxOrderCSVRows {
		return nil, nil, fmt.Errorf("%w: too many rows", ErrInvalidInput)
	}
	headers := make([]string, len(all[0]))
	for i, h := range all[0] {
		headers[i] = strings.TrimSpace(strings.ToLower(h))
	}
	return headers, all[1:], nil
}

func mapOrderCSVRow(rowNum int, headers []string, rec []string, cm OrderCSVColumnMap) OrderCSVPreviewRow {
	get := func(col string) string {
		col = strings.ToLower(strings.TrimSpace(col))
		if col == "" {
			return ""
		}
		for i, h := range headers {
			if h == col && i < len(rec) {
				return strings.TrimSpace(rec[i])
			}
		}
		return ""
	}
	qty, _ := strconv.Atoi(get(cm.Quantity))
	row := OrderCSVPreviewRow{
		RowNumber:         rowNum,
		CustomerName:      get(cm.CustomerName),
		CustomerEmail:     get(cm.CustomerEmail),
		FulfillmentMethod: get(cm.FulfillmentMethod),
		ProductSKU:        get(cm.ProductSKU),
		VariantSKU:        get(cm.VariantSKU),
		Quantity:          qty,
		CampaignSlug:      get(cm.CampaignSlug),
		ShippingLine1:     get(cm.ShippingLine1),
		ShippingCity:      get(cm.ShippingCity),
	}
	if p := get(cm.CustomerPhone); p != "" {
		row.CustomerPhone = &p
	}
	if n := get(cm.Notes); n != "" {
		row.Notes = &n
	}
	return row
}
