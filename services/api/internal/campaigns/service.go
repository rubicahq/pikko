// Package campaigns unifies selling phases (preorder + live) under CAMP-05.
package campaigns

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rubicahq/pikko/services/api/internal/validate"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrConflict     = errors.New("conflict")
)

const (
	TypePreorder = "preorder"
	TypeLive     = "live"
)

type Campaign struct {
	ID                   uuid.UUID         `json:"id"`
	StoreID              uuid.UUID         `json:"store_id"`
	CampaignType         string            `json:"campaign_type"`
	Name                 string            `json:"name"`
	Slug                 string            `json:"slug"`
	Description          *string           `json:"description,omitempty"`
	CoverFileID          *uuid.UUID        `json:"cover_file_id,omitempty"`
	OpensAt              time.Time         `json:"opens_at"`
	ClosesAt             time.Time         `json:"closes_at"`
	PurchasingDate       *time.Time        `json:"purchasing_date,omitempty"`
	ETAStart             *time.Time        `json:"eta_start,omitempty"`
	ETAEnd               *time.Time        `json:"eta_end,omitempty"`
	MerchantNotes        *string           `json:"merchant_notes,omitempty"`
	PaymentPolicy        string            `json:"payment_policy"`
	PublicationStatus    string            `json:"publication_status"`
	OperationalStatus    string            `json:"operational_status"`
	ClosedEarlyAt        *time.Time        `json:"closed_early_at,omitempty"`
	LiveQuickAddEnabled  bool              `json:"live_quick_add_enabled"`
	LiveShowOrderFeed    bool              `json:"live_show_order_feed"`
	PreorderBatchID      *uuid.UUID        `json:"preorder_batch_id,omitempty"`
	CreatedAt            time.Time         `json:"created_at"`
	UpdatedAt            time.Time         `json:"updated_at"`
	ProductCount         int               `json:"product_count,omitempty"`
	Products             []CampaignProduct `json:"products,omitempty"`
}

type CampaignProduct struct {
	ID                uuid.UUID `json:"id"`
	CampaignID        uuid.UUID `json:"campaign_id"`
	ProductID         uuid.UUID `json:"product_id"`
	ProductName       string    `json:"product_name,omitempty"`
	ProductSlug       string    `json:"product_slug,omitempty"`
	BasePriceIDR      int64     `json:"base_price_idr,omitempty"`
	PriceOverrideIDR  *int64    `json:"price_override_idr,omitempty"`
	EffectivePriceIDR int64     `json:"effective_price_idr"`
	QuantityLimit     *int      `json:"quantity_limit,omitempty"`
	Availability      string    `json:"availability"`
	SellingPhase      *string   `json:"selling_phase,omitempty"`
	Notes             *string   `json:"notes,omitempty"`
	DisplayOrder      int       `json:"display_order"`
	Featured          bool      `json:"featured"`
}

type CreateInput struct {
	CampaignType        string     `json:"campaign_type"`
	Name                string     `json:"name"`
	Slug                string     `json:"slug"`
	Description         *string    `json:"description,omitempty"`
	CoverFileID         *uuid.UUID `json:"cover_file_id,omitempty"`
	OpensAt             time.Time  `json:"opens_at"`
	ClosesAt            time.Time  `json:"closes_at"`
	PurchasingDate      *string    `json:"purchasing_date,omitempty"`
	ETAStart            *string    `json:"eta_start,omitempty"`
	ETAEnd              *string    `json:"eta_end,omitempty"`
	MerchantNotes       *string    `json:"merchant_notes,omitempty"`
	PaymentPolicy       string     `json:"payment_policy"`
	PublicationStatus   string     `json:"publication_status"`
	LiveQuickAddEnabled *bool      `json:"live_quick_add_enabled,omitempty"`
	LiveShowOrderFeed   *bool      `json:"live_show_order_feed,omitempty"`
}

type PatchInput struct {
	Name                *string    `json:"name,omitempty"`
	Slug                *string    `json:"slug,omitempty"`
	Description         *string    `json:"description,omitempty"`
	CoverFileID         *uuid.UUID `json:"cover_file_id,omitempty"`
	ClearCover          bool       `json:"clear_cover,omitempty"`
	OpensAt             *time.Time `json:"opens_at,omitempty"`
	ClosesAt            *time.Time `json:"closes_at,omitempty"`
	PurchasingDate      *string    `json:"purchasing_date,omitempty"`
	ETAStart            *string    `json:"eta_start,omitempty"`
	ETAEnd              *string    `json:"eta_end,omitempty"`
	MerchantNotes       *string    `json:"merchant_notes,omitempty"`
	PaymentPolicy       *string    `json:"payment_policy,omitempty"`
	PublicationStatus   *string    `json:"publication_status,omitempty"`
	OperationalStatus   *string    `json:"operational_status,omitempty"`
	LiveQuickAddEnabled *bool      `json:"live_quick_add_enabled,omitempty"`
	LiveShowOrderFeed   *bool      `json:"live_show_order_feed,omitempty"`
}

type AssignProductInput struct {
	ProductID        uuid.UUID `json:"product_id"`
	PriceOverrideIDR *int64    `json:"price_override_idr,omitempty"`
	QuantityLimit    *int      `json:"quantity_limit,omitempty"`
	Availability     string    `json:"availability"`
	SellingPhase     *string   `json:"selling_phase,omitempty"`
	Notes            *string   `json:"notes,omitempty"`
	DisplayOrder     int       `json:"display_order"`
	Featured         *bool     `json:"featured,omitempty"`
}

type Service struct {
	DB *pgxpool.Pool
}

func parseDatePtr(s *string) (*time.Time, error) {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", strings.TrimSpace(*s))
	if err != nil {
		return nil, ErrInvalidInput
	}
	return &t, nil
}

func deriveOperational(pub string, opens, closes time.Time, now time.Time) string {
	if pub != "published" {
		return "draft"
	}
	switch {
	case now.Before(opens):
		return "scheduled"
	case now.Before(closes):
		return "open"
	default:
		return "closed"
	}
}

const campaignSelect = `
	id, store_id, campaign_type, name, slug, description, cover_file_id,
	opens_at, closes_at, purchasing_date, eta_start, eta_end, merchant_notes,
	payment_policy, publication_status, operational_status, closed_early_at,
	live_quick_add_enabled, live_show_order_feed, preorder_batch_id,
	created_at, updated_at
`

func scanCampaign(row pgx.Row) (*Campaign, error) {
	var c Campaign
	var purch, etaS, etaE *time.Time
	err := row.Scan(
		&c.ID, &c.StoreID, &c.CampaignType, &c.Name, &c.Slug, &c.Description, &c.CoverFileID,
		&c.OpensAt, &c.ClosesAt, &purch, &etaS, &etaE, &c.MerchantNotes,
		&c.PaymentPolicy, &c.PublicationStatus, &c.OperationalStatus, &c.ClosedEarlyAt,
		&c.LiveQuickAddEnabled, &c.LiveShowOrderFeed, &c.PreorderBatchID,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	c.PurchasingDate = purch
	c.ETAStart = etaS
	c.ETAEnd = etaE
	return &c, nil
}

func (s *Service) List(ctx context.Context, storeID uuid.UUID, campaignType string) ([]Campaign, error) {
	q := `SELECT ` + campaignSelect + ` FROM campaigns WHERE store_id=$1`
	args := []any{storeID}
	if campaignType != "" {
		q += ` AND campaign_type=$2`
		args = append(args, campaignType)
	}
	q += ` ORDER BY opens_at DESC, created_at DESC`
	rows, err := s.DB.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Campaign{}
	for rows.Next() {
		c, err := scanCampaign(rows)
		if err != nil {
			return nil, err
		}
		_ = s.DB.QueryRow(ctx, `
			SELECT COUNT(*) FROM campaign_products WHERE campaign_id=$1 AND store_id=$2
		`, c.ID, storeID).Scan(&c.ProductCount)
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (s *Service) Get(ctx context.Context, storeID, campaignID uuid.UUID, withProducts bool) (*Campaign, error) {
	row := s.DB.QueryRow(ctx, `
		SELECT `+campaignSelect+` FROM campaigns WHERE id=$1 AND store_id=$2
	`, campaignID, storeID)
	c, err := scanCampaign(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if withProducts {
		products, err := s.listProducts(ctx, storeID, campaignID)
		if err != nil {
			return nil, err
		}
		c.Products = products
		c.ProductCount = len(products)
	}
	return c, nil
}

func (s *Service) Create(ctx context.Context, storeID uuid.UUID, in CreateInput) (*Campaign, error) {
	ctype := strings.TrimSpace(in.CampaignType)
	if ctype == "" {
		ctype = TypePreorder
	}
	if ctype != TypePreorder && ctype != TypeLive {
		return nil, ErrInvalidInput
	}
	name := strings.TrimSpace(in.Name)
	slug := strings.TrimSpace(strings.ToLower(in.Slug))
	if len(name) < 2 || len(name) > 120 {
		return nil, ErrInvalidInput
	}
	if !validate.ValidateSlug(slug) {
		return nil, ErrInvalidInput
	}
	if !in.ClosesAt.After(in.OpensAt) {
		return nil, ErrInvalidInput
	}
	if in.PaymentPolicy == "" {
		in.PaymentPolicy = "full_after_checkout"
	}
	if in.PaymentPolicy != "full_after_checkout" && in.PaymentPolicy != "full_after_confirmation" {
		return nil, ErrInvalidInput
	}
	if in.PublicationStatus == "" {
		in.PublicationStatus = "draft"
	}
	if in.PublicationStatus != "draft" && in.PublicationStatus != "published" {
		return nil, ErrInvalidInput
	}
	purch, err := parseDatePtr(in.PurchasingDate)
	if err != nil {
		return nil, err
	}
	etaS, err := parseDatePtr(in.ETAStart)
	if err != nil {
		return nil, err
	}
	etaE, err := parseDatePtr(in.ETAEnd)
	if err != nil {
		return nil, err
	}
	quickAdd := ctype == TypeLive
	if in.LiveQuickAddEnabled != nil {
		quickAdd = *in.LiveQuickAddEnabled
	}
	showFeed := true
	if in.LiveShowOrderFeed != nil {
		showFeed = *in.LiveShowOrderFeed
	}
	status := deriveOperational(in.PublicationStatus, in.OpensAt.UTC(), in.ClosesAt.UTC(), time.Now().UTC())

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var campaignID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO campaigns (
			store_id, campaign_type, name, slug, description, cover_file_id,
			opens_at, closes_at, purchasing_date, eta_start, eta_end, merchant_notes,
			payment_policy, publication_status, operational_status,
			live_quick_add_enabled, live_show_order_feed
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		RETURNING id
	`, storeID, ctype, name, slug, in.Description, in.CoverFileID,
		in.OpensAt.UTC(), in.ClosesAt.UTC(), purch, etaS, etaE, in.MerchantNotes,
		in.PaymentPolicy, in.PublicationStatus, status, quickAdd, showFeed,
	).Scan(&campaignID)
	if err != nil {
		if strings.Contains(err.Error(), "campaigns_store_slug_unique") {
			return nil, ErrConflict
		}
		return nil, err
	}

	if ctype == TypePreorder {
		var batchID uuid.UUID
		err = tx.QueryRow(ctx, `
			INSERT INTO preorder_batches (
				store_id, name, slug, description, cover_file_id,
				opens_at, closes_at, purchasing_date, eta_start, eta_end,
				merchant_notes, payment_policy, publication_status, operational_status,
				campaign_id
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
			RETURNING id
		`, storeID, name, slug, in.Description, in.CoverFileID,
			in.OpensAt.UTC(), in.ClosesAt.UTC(), purch, etaS, etaE,
			in.MerchantNotes, in.PaymentPolicy, in.PublicationStatus, status, campaignID,
		).Scan(&batchID)
		if err != nil {
			if strings.Contains(err.Error(), "preorder_batches_store_slug_unique") {
				return nil, ErrConflict
			}
			return nil, err
		}
		_, err = tx.Exec(ctx, `
			UPDATE campaigns SET preorder_batch_id=$2, updated_at=now() WHERE id=$1 AND store_id=$3
		`, campaignID, batchID, storeID)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.Get(ctx, storeID, campaignID, true)
}

func (s *Service) Patch(ctx context.Context, storeID, campaignID uuid.UUID, in PatchInput) (*Campaign, error) {
	cur, err := s.Get(ctx, storeID, campaignID, false)
	if err != nil {
		return nil, err
	}
	name := cur.Name
	if in.Name != nil {
		name = strings.TrimSpace(*in.Name)
		if len(name) < 2 || len(name) > 120 {
			return nil, ErrInvalidInput
		}
	}
	slug := cur.Slug
	if in.Slug != nil {
		slug = strings.TrimSpace(strings.ToLower(*in.Slug))
		if !validate.ValidateSlug(slug) {
			return nil, ErrInvalidInput
		}
	}
	opens := cur.OpensAt
	closes := cur.ClosesAt
	if in.OpensAt != nil {
		opens = in.OpensAt.UTC()
	}
	if in.ClosesAt != nil {
		closes = in.ClosesAt.UTC()
	}
	if !closes.After(opens) {
		return nil, ErrInvalidInput
	}
	pub := cur.PublicationStatus
	if in.PublicationStatus != nil {
		pub = *in.PublicationStatus
		if pub != "draft" && pub != "published" {
			return nil, ErrInvalidInput
		}
	}
	policy := cur.PaymentPolicy
	if in.PaymentPolicy != nil {
		policy = *in.PaymentPolicy
		if policy != "full_after_checkout" && policy != "full_after_confirmation" {
			return nil, ErrInvalidInput
		}
	}
	purch := cur.PurchasingDate
	if in.PurchasingDate != nil {
		purch, err = parseDatePtr(in.PurchasingDate)
		if err != nil {
			return nil, err
		}
	}
	etaS := cur.ETAStart
	if in.ETAStart != nil {
		etaS, err = parseDatePtr(in.ETAStart)
		if err != nil {
			return nil, err
		}
	}
	etaE := cur.ETAEnd
	if in.ETAEnd != nil {
		etaE, err = parseDatePtr(in.ETAEnd)
		if err != nil {
			return nil, err
		}
	}
	desc := cur.Description
	if in.Description != nil {
		desc = in.Description
	}
	notes := cur.MerchantNotes
	if in.MerchantNotes != nil {
		notes = in.MerchantNotes
	}
	cover := cur.CoverFileID
	if in.ClearCover {
		cover = nil
	} else if in.CoverFileID != nil {
		cover = in.CoverFileID
	}
	status := cur.OperationalStatus
	if in.OperationalStatus != nil {
		status = *in.OperationalStatus
	} else if in.PublicationStatus != nil || in.OpensAt != nil || in.ClosesAt != nil {
		// Re-derive only for early lifecycle states.
		if status == "draft" || status == "scheduled" || status == "open" || status == "closed" {
			status = deriveOperational(pub, opens, closes, time.Now().UTC())
		}
	}
	quickAdd := cur.LiveQuickAddEnabled
	if in.LiveQuickAddEnabled != nil {
		quickAdd = *in.LiveQuickAddEnabled
	}
	showFeed := cur.LiveShowOrderFeed
	if in.LiveShowOrderFeed != nil {
		showFeed = *in.LiveShowOrderFeed
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE campaigns SET
			name=$3, slug=$4, description=$5, cover_file_id=$6,
			opens_at=$7, closes_at=$8, purchasing_date=$9, eta_start=$10, eta_end=$11,
			merchant_notes=$12, payment_policy=$13, publication_status=$14, operational_status=$15,
			live_quick_add_enabled=$16, live_show_order_feed=$17, updated_at=now()
		WHERE id=$1 AND store_id=$2
	`, campaignID, storeID, name, slug, desc, cover,
		opens, closes, purch, etaS, etaE, notes, policy, pub, status, quickAdd, showFeed)
	if err != nil {
		if strings.Contains(err.Error(), "campaigns_store_slug_unique") {
			return nil, ErrConflict
		}
		return nil, err
	}

	if cur.PreorderBatchID != nil {
		_, err = tx.Exec(ctx, `
			UPDATE preorder_batches SET
				name=$3, slug=$4, description=$5, cover_file_id=$6,
				opens_at=$7, closes_at=$8, purchasing_date=$9, eta_start=$10, eta_end=$11,
				merchant_notes=$12, payment_policy=$13, publication_status=$14, operational_status=$15,
				updated_at=now()
			WHERE id=$1 AND store_id=$2
		`, *cur.PreorderBatchID, storeID, name, slug, desc, cover,
			opens, closes, purch, etaS, etaE, notes, policy, pub, status)
		if err != nil {
			if strings.Contains(err.Error(), "preorder_batches_store_slug_unique") {
				return nil, ErrConflict
			}
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.Get(ctx, storeID, campaignID, true)
}

func (s *Service) listProducts(ctx context.Context, storeID, campaignID uuid.UUID) ([]CampaignProduct, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT cp.id, cp.campaign_id, cp.product_id, p.name, p.slug, p.base_price_idr,
		       cp.price_override_idr, cp.quantity_limit, cp.availability, cp.selling_phase,
		       cp.notes, cp.display_order, cp.featured
		FROM campaign_products cp
		JOIN products p ON p.id = cp.product_id
		WHERE cp.campaign_id=$1 AND cp.store_id=$2
		ORDER BY cp.featured DESC, cp.display_order, p.name
	`, campaignID, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CampaignProduct{}
	for rows.Next() {
		var p CampaignProduct
		if err := rows.Scan(
			&p.ID, &p.CampaignID, &p.ProductID, &p.ProductName, &p.ProductSlug, &p.BasePriceIDR,
			&p.PriceOverrideIDR, &p.QuantityLimit, &p.Availability, &p.SellingPhase,
			&p.Notes, &p.DisplayOrder, &p.Featured,
		); err != nil {
			return nil, err
		}
		p.EffectivePriceIDR = p.BasePriceIDR
		if p.PriceOverrideIDR != nil {
			p.EffectivePriceIDR = *p.PriceOverrideIDR
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Service) AssignProduct(ctx context.Context, storeID, campaignID uuid.UUID, in AssignProductInput) (*CampaignProduct, error) {
	c, err := s.Get(ctx, storeID, campaignID, false)
	if err != nil {
		return nil, err
	}
	if in.Availability == "" {
		in.Availability = "available"
	}
	if in.Availability != "available" && in.Availability != "unavailable" {
		return nil, ErrInvalidInput
	}
	if in.SellingPhase != nil {
		sp := *in.SellingPhase
		if sp != "preorder" && sp != "live" && sp != "ready_stock" {
			return nil, ErrInvalidInput
		}
	} else {
		phase := c.CampaignType
		in.SellingPhase = &phase
	}
	if in.PriceOverrideIDR != nil && *in.PriceOverrideIDR < 0 {
		return nil, ErrInvalidInput
	}
	if in.QuantityLimit != nil && *in.QuantityLimit <= 0 {
		return nil, ErrInvalidInput
	}

	var exists bool
	err = s.DB.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM products WHERE id=$1 AND store_id=$2 AND status <> 'archived')
	`, in.ProductID, storeID).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	featured := c.CampaignType == TypeLive // new live SKUs default into quick-add panel
	setFeatured := in.Featured != nil
	if setFeatured {
		featured = *in.Featured
	}
	var cpID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO campaign_products (
			store_id, campaign_id, product_id, price_override_idr, quantity_limit,
			availability, selling_phase, notes, display_order, featured
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (campaign_id, product_id) DO UPDATE SET
			price_override_idr=EXCLUDED.price_override_idr,
			quantity_limit=EXCLUDED.quantity_limit,
			availability=EXCLUDED.availability,
			selling_phase=EXCLUDED.selling_phase,
			notes=EXCLUDED.notes,
			display_order=EXCLUDED.display_order,
			featured=CASE WHEN $11 THEN EXCLUDED.featured ELSE campaign_products.featured END,
			updated_at=now()
		RETURNING id
	`, storeID, campaignID, in.ProductID, in.PriceOverrideIDR, in.QuantityLimit,
		in.Availability, in.SellingPhase, in.Notes, in.DisplayOrder, featured, setFeatured,
	).Scan(&cpID)
	if err != nil {
		return nil, err
	}

	// Keep preorder batch products in sync for type=preorder.
	if c.CampaignType == TypePreorder && c.PreorderBatchID != nil {
		_, err = tx.Exec(ctx, `
			INSERT INTO preorder_batch_products (
				store_id, batch_id, product_id, price_override_idr, quantity_limit,
				availability, notes, display_order
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
			ON CONFLICT (batch_id, product_id) DO UPDATE SET
				price_override_idr=EXCLUDED.price_override_idr,
				quantity_limit=EXCLUDED.quantity_limit,
				availability=EXCLUDED.availability,
				notes=EXCLUDED.notes,
				display_order=EXCLUDED.display_order,
				updated_at=now()
		`, storeID, *c.PreorderBatchID, in.ProductID, in.PriceOverrideIDR, in.QuantityLimit,
			in.Availability, in.Notes, in.DisplayOrder)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	products, err := s.listProducts(ctx, storeID, campaignID)
	if err != nil {
		return nil, err
	}
	for i := range products {
		if products[i].ID == cpID {
			return &products[i], nil
		}
	}
	return nil, fmt.Errorf("assigned product missing after write")
}

func (s *Service) RemoveProduct(ctx context.Context, storeID, campaignID, productID uuid.UUID) error {
	c, err := s.Get(ctx, storeID, campaignID, false)
	if err != nil {
		return err
	}
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		DELETE FROM campaign_products WHERE store_id=$1 AND campaign_id=$2 AND product_id=$3
	`, storeID, campaignID, productID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	if c.CampaignType == TypePreorder && c.PreorderBatchID != nil {
		_, _ = tx.Exec(ctx, `
			DELETE FROM preorder_batch_products WHERE store_id=$1 AND batch_id=$2 AND product_id=$3
		`, storeID, *c.PreorderBatchID, productID)
	}
	return tx.Commit(ctx)
}

// EnsureLinkedFromBatch creates a campaign row for a newly created preorder batch
// when the legacy /preorders path is used (keeps CAMP-05 index complete).
func (s *Service) EnsureLinkedFromBatch(ctx context.Context, storeID, batchID uuid.UUID) error {
	var existing *uuid.UUID
	err := s.DB.QueryRow(ctx, `
		SELECT campaign_id FROM preorder_batches WHERE id=$1 AND store_id=$2
	`, batchID, storeID).Scan(&existing)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}

	var name, slug, pub, op, policy string
	var desc, notes *string
	var cover *uuid.UUID
	var opens, closes time.Time
	var purch, etaS, etaE *time.Time
	err = s.DB.QueryRow(ctx, `
		SELECT name, slug, description, cover_file_id, opens_at, closes_at,
		       purchasing_date, eta_start, eta_end, merchant_notes,
		       payment_policy, publication_status, operational_status
		FROM preorder_batches WHERE id=$1 AND store_id=$2
	`, batchID, storeID).Scan(
		&name, &slug, &desc, &cover, &opens, &closes, &purch, &etaS, &etaE, &notes, &policy, &pub, &op,
	)
	if err != nil {
		return err
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var campaignID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO campaigns (
			store_id, campaign_type, name, slug, description, cover_file_id,
			opens_at, closes_at, purchasing_date, eta_start, eta_end, merchant_notes,
			payment_policy, publication_status, operational_status, preorder_batch_id
		) VALUES ($1,'preorder',$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		ON CONFLICT (store_id, slug) DO UPDATE SET
			preorder_batch_id=EXCLUDED.preorder_batch_id,
			updated_at=now()
		RETURNING id
	`, storeID, name, slug, desc, cover, opens, closes, purch, etaS, etaE, notes, policy, pub, op, batchID,
	).Scan(&campaignID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		UPDATE preorder_batches SET campaign_id=$3, updated_at=now() WHERE id=$1 AND store_id=$2
	`, batchID, storeID, campaignID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// SyncFromBatch mirrors batch field changes onto the linked campaign.
func (s *Service) SyncFromBatch(ctx context.Context, storeID, batchID uuid.UUID) error {
	_, err := s.DB.Exec(ctx, `
		UPDATE campaigns c SET
			name=b.name, slug=b.slug, description=b.description, cover_file_id=b.cover_file_id,
			opens_at=b.opens_at, closes_at=b.closes_at, purchasing_date=b.purchasing_date,
			eta_start=b.eta_start, eta_end=b.eta_end, merchant_notes=b.merchant_notes,
			payment_policy=b.payment_policy, publication_status=b.publication_status,
			operational_status=b.operational_status, closed_early_at=b.closed_early_at,
			updated_at=now()
		FROM preorder_batches b
		WHERE b.id=$1 AND b.store_id=$2 AND c.id=b.campaign_id AND c.store_id=b.store_id
	`, batchID, storeID)
	return err
}

// SyncBatchProduct mirrors a preorder batch product assignment into campaign_products.
func (s *Service) SyncBatchProduct(ctx context.Context, storeID, batchID, productID uuid.UUID) error {
	_, err := s.DB.Exec(ctx, `
		INSERT INTO campaign_products (
			store_id, campaign_id, product_id, price_override_idr, quantity_limit,
			availability, selling_phase, notes, display_order
		)
		SELECT bp.store_id, b.campaign_id, bp.product_id, bp.price_override_idr, bp.quantity_limit,
		       bp.availability, 'preorder', bp.notes, bp.display_order
		FROM preorder_batch_products bp
		JOIN preorder_batches b ON b.id = bp.batch_id
		WHERE bp.store_id=$1 AND bp.batch_id=$2 AND bp.product_id=$3 AND b.campaign_id IS NOT NULL
		ON CONFLICT (campaign_id, product_id) DO UPDATE SET
			price_override_idr=EXCLUDED.price_override_idr,
			quantity_limit=EXCLUDED.quantity_limit,
			availability=EXCLUDED.availability,
			notes=EXCLUDED.notes,
			display_order=EXCLUDED.display_order,
			updated_at=now()
	`, storeID, batchID, productID)
	return err
}

func (s *Service) RemoveBatchProductMirror(ctx context.Context, storeID, batchID, productID uuid.UUID) error {
	_, err := s.DB.Exec(ctx, `
		DELETE FROM campaign_products cp
		USING preorder_batches b
		WHERE b.id=$2 AND b.store_id=$1 AND cp.campaign_id=b.campaign_id
		  AND cp.product_id=$3 AND cp.store_id=$1
	`, storeID, batchID, productID)
	return err
}
