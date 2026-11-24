package commerce

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rubicahq/pikko/services/api/internal/email"
	"github.com/rubicahq/pikko/services/api/internal/validate"
)

const (
	BatchDraft           = "draft"
	BatchScheduled       = "scheduled"
	BatchOpen            = "open"
	BatchClosed          = "closed"
	BatchPurchasing      = "purchasing"
	BatchAwaitingArrival = "awaiting_arrival"
	BatchArrived         = "arrived"
	BatchFulfilling      = "fulfilling"
	BatchCompleted       = "completed"
	BatchCancelled       = "cancelled"
)

type PreorderBatch struct {
	ID                 uuid.UUID  `json:"id"`
	StoreID            uuid.UUID  `json:"store_id"`
	Name               string     `json:"name"`
	Slug               string     `json:"slug"`
	Description        *string    `json:"description,omitempty"`
	CoverFileID        *uuid.UUID `json:"cover_file_id,omitempty"`
	OpensAt            time.Time  `json:"opens_at"`
	ClosesAt           time.Time  `json:"closes_at"`
	PurchasingDate     *time.Time `json:"purchasing_date,omitempty"`
	ETAStart           *time.Time `json:"eta_start,omitempty"`
	ETAEnd             *time.Time `json:"eta_end,omitempty"`
	MerchantNotes      *string    `json:"merchant_notes,omitempty"`
	PaymentPolicy      string     `json:"payment_policy"`
	PublicationStatus  string     `json:"publication_status"`
	OperationalStatus  string     `json:"operational_status"`
	ClosedEarlyAt      *time.Time `json:"closed_early_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	ProductCount       int        `json:"product_count,omitempty"`
	OrderCount         int        `json:"order_count,omitempty"`
	Products           []BatchProduct `json:"products,omitempty"`
}

type BatchProduct struct {
	ID               uuid.UUID  `json:"id"`
	BatchID          uuid.UUID  `json:"batch_id"`
	ProductID        uuid.UUID  `json:"product_id"`
	ProductName      string     `json:"product_name,omitempty"`
	ProductSlug      string     `json:"product_slug,omitempty"`
	BasePriceIDR     int64      `json:"base_price_idr,omitempty"`
	PriceOverrideIDR *int64     `json:"price_override_idr,omitempty"`
	EffectivePriceIDR int64     `json:"effective_price_idr"`
	QuantityLimit    *int       `json:"quantity_limit,omitempty"`
	ReservedQty      int        `json:"reserved_qty,omitempty"`
	Availability     string     `json:"availability"`
	Notes            *string    `json:"notes,omitempty"`
	DisplayOrder     int        `json:"display_order"`
	PreorderEligible bool       `json:"preorder_eligible,omitempty"`
}

type CreateBatchInput struct {
	Name              string     `json:"name"`
	Slug              string     `json:"slug"`
	Description       *string    `json:"description,omitempty"`
	CoverFileID       *uuid.UUID `json:"cover_file_id,omitempty"`
	OpensAt           time.Time  `json:"opens_at"`
	ClosesAt          time.Time  `json:"closes_at"`
	PurchasingDate    *string    `json:"purchasing_date,omitempty"` // YYYY-MM-DD
	ETAStart          *string    `json:"eta_start,omitempty"`
	ETAEnd            *string    `json:"eta_end,omitempty"`
	MerchantNotes     *string    `json:"merchant_notes,omitempty"`
	PaymentPolicy     string     `json:"payment_policy"`
	PublicationStatus string     `json:"publication_status"`
}

type PatchBatchInput struct {
	Name              *string    `json:"name,omitempty"`
	Slug              *string    `json:"slug,omitempty"`
	Description       *string    `json:"description,omitempty"`
	CoverFileID       *uuid.UUID `json:"cover_file_id,omitempty"`
	ClearCover        bool       `json:"clear_cover,omitempty"`
	OpensAt           *time.Time `json:"opens_at,omitempty"`
	ClosesAt          *time.Time `json:"closes_at,omitempty"`
	PurchasingDate    *string    `json:"purchasing_date,omitempty"`
	ETAStart          *string    `json:"eta_start,omitempty"`
	ETAEnd            *string    `json:"eta_end,omitempty"`
	MerchantNotes     *string    `json:"merchant_notes,omitempty"`
	PaymentPolicy     *string    `json:"payment_policy,omitempty"`
	PublicationStatus *string    `json:"publication_status,omitempty"`
	OperationalStatus *string    `json:"operational_status,omitempty"`
}

type AssignProductInput struct {
	ProductID        uuid.UUID `json:"product_id"`
	PriceOverrideIDR *int64    `json:"price_override_idr,omitempty"`
	QuantityLimit    *int      `json:"quantity_limit,omitempty"`
	Availability     string    `json:"availability"`
	Notes            *string   `json:"notes,omitempty"`
	DisplayOrder     int       `json:"display_order"`
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

// SyncAll opens/closes published batches based on wall-clock (job worker hook).
func (s *Service) SyncAll(ctx context.Context) error {
	_, err := s.SyncBatchStatuses(ctx, nil)
	return err
}

func (s *Service) SyncBatchStatuses(ctx context.Context, storeID *uuid.UUID) (int, error) {
	now := s.now()
	args := []any{now}
	storeFilter := ""
	if storeID != nil {
		storeFilter = " AND store_id=$2"
		args = append(args, *storeID)
	}
	// Open scheduled batches whose window started
	tag1, err := s.DB.Exec(ctx, `
		UPDATE preorder_batches SET operational_status='open', updated_at=now()
		WHERE operational_status IN ('draft','scheduled')
		  AND publication_status='published'
		  AND opens_at <= $1 AND closes_at > $1
		  AND closed_early_at IS NULL`+storeFilter, args...)
	if err != nil {
		return 0, err
	}
	// Close open batches past closes_at
	tag2, err := s.DB.Exec(ctx, `
		UPDATE preorder_batches SET operational_status='closed', updated_at=now()
		WHERE operational_status='open'
		  AND closes_at <= $1`+storeFilter, args...)
	if err != nil {
		return 0, err
	}
	// Promote draft published batches to scheduled if opens in future
	tag3, err := s.DB.Exec(ctx, `
		UPDATE preorder_batches SET operational_status='scheduled', updated_at=now()
		WHERE operational_status='draft'
		  AND publication_status='published'
		  AND opens_at > $1`+storeFilter, args...)
	if err != nil {
		return 0, err
	}
	// Mirror operational_status onto linked campaigns (CAMP-05)
	syncCampaignSQL := `
		UPDATE campaigns c SET operational_status=b.operational_status, updated_at=now()
		FROM preorder_batches b
		WHERE c.id=b.campaign_id AND c.store_id=b.store_id
		  AND c.operational_status IS DISTINCT FROM b.operational_status`
	if storeID != nil {
		_, _ = s.DB.Exec(ctx, syncCampaignSQL+` AND b.store_id=$1`, *storeID)
	} else {
		_, _ = s.DB.Exec(ctx, syncCampaignSQL)
	}
	return int(tag1.RowsAffected() + tag2.RowsAffected() + tag3.RowsAffected()), nil
}

func (s *Service) syncOneBatch(ctx context.Context, batch *PreorderBatch) error {
	now := s.now()
	newStatus := batch.OperationalStatus
	if batch.OperationalStatus == BatchCancelled ||
		batch.OperationalStatus == BatchCompleted ||
		batch.OperationalStatus == BatchPurchasing ||
		batch.OperationalStatus == BatchAwaitingArrival ||
		batch.OperationalStatus == BatchArrived ||
		batch.OperationalStatus == BatchFulfilling {
		return nil
	}
	if batch.ClosedEarlyAt != nil && batch.OperationalStatus == BatchOpen {
		newStatus = BatchClosed
	} else if batch.PublicationStatus == "published" {
		switch {
		case now.Before(batch.OpensAt):
			newStatus = BatchScheduled
		case !now.Before(batch.OpensAt) && now.Before(batch.ClosesAt) && batch.ClosedEarlyAt == nil:
			newStatus = BatchOpen
		case !now.Before(batch.ClosesAt) || batch.ClosedEarlyAt != nil:
			if batch.OperationalStatus == BatchOpen || batch.OperationalStatus == BatchScheduled {
				newStatus = BatchClosed
			}
		}
	}
	if newStatus != batch.OperationalStatus {
		_, err := s.DB.Exec(ctx, `
			UPDATE preorder_batches SET operational_status=$3, updated_at=now()
			WHERE id=$1 AND store_id=$2
		`, batch.ID, batch.StoreID, newStatus)
		if err != nil {
			return err
		}
		batch.OperationalStatus = newStatus
	}
	return nil
}

func scanBatch(row scannable) (*PreorderBatch, error) {
	var b PreorderBatch
	var purch, etaS, etaE *time.Time
	err := row.Scan(
		&b.ID, &b.StoreID, &b.Name, &b.Slug, &b.Description, &b.CoverFileID,
		&b.OpensAt, &b.ClosesAt, &purch, &etaS, &etaE, &b.MerchantNotes,
		&b.PaymentPolicy, &b.PublicationStatus, &b.OperationalStatus, &b.ClosedEarlyAt,
		&b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	b.PurchasingDate = purch
	b.ETAStart = etaS
	b.ETAEnd = etaE
	return &b, nil
}

const batchSelectCols = `
	id, store_id, name, slug, description, cover_file_id,
	opens_at, closes_at, purchasing_date, eta_start, eta_end, merchant_notes,
	payment_policy, publication_status, operational_status, closed_early_at,
	created_at, updated_at
`

func (s *Service) ListBatches(ctx context.Context, storeID uuid.UUID) ([]PreorderBatch, error) {
	_, _ = s.SyncBatchStatuses(ctx, &storeID)
	rows, err := s.DB.Query(ctx, `
		SELECT `+batchSelectCols+` FROM preorder_batches
		WHERE store_id=$1
		ORDER BY opens_at DESC, created_at DESC
	`, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PreorderBatch
	for rows.Next() {
		b, err := scanBatch(rows)
		if err != nil {
			return nil, err
		}
		_ = s.DB.QueryRow(ctx, `
			SELECT COUNT(*) FROM preorder_batch_products WHERE batch_id=$1 AND store_id=$2
		`, b.ID, storeID).Scan(&b.ProductCount)
		_ = s.DB.QueryRow(ctx, `
			SELECT COUNT(*) FROM orders WHERE store_id=$1 AND preorder_batch_id=$2
			  AND order_status NOT IN ('cancelled')
		`, storeID, b.ID).Scan(&b.OrderCount)
		out = append(out, *b)
	}
	return out, rows.Err()
}

func (s *Service) GetBatch(ctx context.Context, storeID, batchID uuid.UUID, withProducts bool) (*PreorderBatch, error) {
	row := s.DB.QueryRow(ctx, `
		SELECT `+batchSelectCols+` FROM preorder_batches WHERE id=$1 AND store_id=$2
	`, batchID, storeID)
	b, err := scanBatch(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	_ = s.syncOneBatch(ctx, b)
	if withProducts {
		products, err := s.listBatchProducts(ctx, storeID, batchID)
		if err != nil {
			return nil, err
		}
		b.Products = products
		b.ProductCount = len(products)
	}
	return b, nil
}

func (s *Service) CreateBatch(ctx context.Context, storeID uuid.UUID, in CreateBatchInput) (*PreorderBatch, error) {
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
	status := BatchDraft
	now := s.now()
	if in.PublicationStatus == "published" {
		if now.Before(in.OpensAt) {
			status = BatchScheduled
		} else if now.Before(in.ClosesAt) {
			status = BatchOpen
		} else {
			status = BatchClosed
		}
	}

	var id uuid.UUID
	err = s.DB.QueryRow(ctx, `
		INSERT INTO preorder_batches (
			store_id, name, slug, description, cover_file_id,
			opens_at, closes_at, purchasing_date, eta_start, eta_end,
			merchant_notes, payment_policy, publication_status, operational_status
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING id
	`, storeID, name, slug, in.Description, in.CoverFileID,
		in.OpensAt.UTC(), in.ClosesAt.UTC(), purch, etaS, etaE,
		in.MerchantNotes, in.PaymentPolicy, in.PublicationStatus, status,
	).Scan(&id)
	if err != nil {
		if strings.Contains(err.Error(), "preorder_batches_store_slug_unique") {
			return nil, ErrConflict
		}
		return nil, err
	}
	if s.Outbox != nil {
		_ = s.Outbox.Write(ctx, &storeID, "preorder.preorder_batch_created", map[string]any{
			"batch_id": id.String(), "slug": slug,
		}, "preorder_batch_created:"+id.String())
	}
	return s.GetBatch(ctx, storeID, id, true)
}

func (s *Service) UpdateBatch(ctx context.Context, storeID, batchID uuid.UUID, in PatchBatchInput) (*PreorderBatch, error) {
	b, err := s.GetBatch(ctx, storeID, batchID, false)
	if err != nil {
		return nil, err
	}
	name := b.Name
	if in.Name != nil {
		name = strings.TrimSpace(*in.Name)
		if len(name) < 2 || len(name) > 120 {
			return nil, ErrInvalidInput
		}
	}
	slug := b.Slug
	if in.Slug != nil {
		slug = strings.TrimSpace(strings.ToLower(*in.Slug))
		if !validate.ValidateSlug(slug) {
			return nil, ErrInvalidInput
		}
	}
	opensAt := b.OpensAt
	closesAt := b.ClosesAt
	if in.OpensAt != nil {
		opensAt = in.OpensAt.UTC()
	}
	if in.ClosesAt != nil {
		closesAt = in.ClosesAt.UTC()
	}
	if !closesAt.After(opensAt) {
		return nil, ErrInvalidInput
	}
	pub := b.PublicationStatus
	if in.PublicationStatus != nil {
		pub = *in.PublicationStatus
		if pub != "draft" && pub != "published" {
			return nil, ErrInvalidInput
		}
	}
	policy := b.PaymentPolicy
	if in.PaymentPolicy != nil {
		policy = *in.PaymentPolicy
	}
	purch := b.PurchasingDate
	if in.PurchasingDate != nil {
		purch, err = parseDatePtr(in.PurchasingDate)
		if err != nil {
			return nil, err
		}
	}
	etaS := b.ETAStart
	if in.ETAStart != nil {
		etaS, err = parseDatePtr(in.ETAStart)
		if err != nil {
			return nil, err
		}
	}
	etaE := b.ETAEnd
	if in.ETAEnd != nil {
		etaE, err = parseDatePtr(in.ETAEnd)
		if err != nil {
			return nil, err
		}
	}
	desc := b.Description
	if in.Description != nil {
		desc = in.Description
	}
	notes := b.MerchantNotes
	if in.MerchantNotes != nil {
		notes = in.MerchantNotes
	}
	cover := b.CoverFileID
	if in.ClearCover {
		cover = nil
	} else if in.CoverFileID != nil {
		cover = in.CoverFileID
	}

	opStatus := b.OperationalStatus
	closedEarly := b.ClosedEarlyAt
	if in.OperationalStatus != nil {
		to := *in.OperationalStatus
		if !canTransitionBatch(b.OperationalStatus, to) {
			return nil, ErrInvalidTransition
		}
		if to == BatchClosed && b.OperationalStatus == BatchOpen {
			now := s.now()
			closedEarly = &now
		}
		if to == BatchOpen && b.OperationalStatus == BatchClosed {
			// Reopen requires valid future closes_at
			if !closesAt.After(s.now()) {
				return nil, ErrInvalidInput
			}
			closedEarly = nil
		}
		opStatus = to
	}

	_, err = s.DB.Exec(ctx, `
		UPDATE preorder_batches SET
			name=$3, slug=$4, description=$5, cover_file_id=$6,
			opens_at=$7, closes_at=$8, purchasing_date=$9, eta_start=$10, eta_end=$11,
			merchant_notes=$12, payment_policy=$13, publication_status=$14,
			operational_status=$15, closed_early_at=$16, updated_at=now()
		WHERE id=$1 AND store_id=$2
	`, batchID, storeID, name, slug, desc, cover,
		opensAt, closesAt, purch, etaS, etaE,
		notes, policy, pub, opStatus, closedEarly,
	)
	if err != nil {
		if strings.Contains(err.Error(), "preorder_batches_store_slug_unique") {
			return nil, ErrConflict
		}
		return nil, err
	}
	if in.OperationalStatus != nil && *in.OperationalStatus == BatchClosed && b.OperationalStatus == BatchOpen {
		s.notifyBatchClosedInApp(ctx, storeID, batchID, name)
		if s.Email != nil && s.Cfg != nil {
			var ownerEmail, storeName string
			_ = s.DB.QueryRow(ctx, `
				SELECT u.email, st.name FROM stores st
				JOIN store_memberships sm ON sm.store_id=st.id AND sm.role='owner' AND sm.status='active'
				JOIN users u ON u.id=sm.user_id
				WHERE st.id=$1
				LIMIT 1
			`, storeID).Scan(&ownerEmail, &storeName)
			if ownerEmail != "" {
				dash := ""
				if s.URLs != nil {
					dash = fmt.Sprintf("%s/stores/%s/preorders/%s", strings.TrimRight(s.Cfg.DashboardURL, "/"), storeID, batchID)
				}
				subj, html, text := email.BatchClosedMerchantContent(storeName, name, dash)
				_ = s.Email.Enqueue(ctx, email.Message{
					TemplateKey: "batch_closed_merchant", To: ownerEmail,
					From: s.Cfg.EmailNotificationFrom, FromName: s.Cfg.EmailFromName, ReplyTo: s.Cfg.EmailReplyTo,
					Subject: subj, HTML: html, Text: text,
					IdempotencyKey: "email:batch_closed:" + batchID.String(),
				})
			}
		}
	}
	if s.Outbox != nil {
		_ = s.Outbox.Write(ctx, &storeID, "preorder.preorder_batch_status_updated", map[string]any{
			"batch_id": batchID.String(), "status": opStatus,
		}, fmt.Sprintf("preorder_batch_status:%s:%s:%d", batchID, opStatus, s.now().Unix()))
	}
	return s.GetBatch(ctx, storeID, batchID, true)
}

func canTransitionBatch(from, to string) bool {
	allowed := map[string][]string{
		BatchDraft:           {BatchScheduled, BatchOpen, BatchCancelled},
		BatchScheduled:       {BatchOpen, BatchClosed, BatchCancelled, BatchDraft},
		BatchOpen:            {BatchClosed, BatchCancelled},
		BatchClosed:          {BatchOpen, BatchPurchasing, BatchCancelled},
		BatchPurchasing:      {BatchAwaitingArrival, BatchClosed, BatchCancelled},
		BatchAwaitingArrival: {BatchArrived, BatchCancelled},
		BatchArrived:         {BatchFulfilling, BatchCancelled},
		BatchFulfilling:      {BatchCompleted, BatchCancelled},
		BatchCompleted:       {},
		BatchCancelled:       {},
	}
	for _, a := range allowed[from] {
		if a == to {
			return true
		}
	}
	return false
}

func (s *Service) listBatchProducts(ctx context.Context, storeID, batchID uuid.UUID) ([]BatchProduct, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT bp.id, bp.batch_id, bp.product_id, p.name, p.slug, p.base_price_idr,
		       bp.price_override_idr, bp.quantity_limit, bp.availability, bp.notes,
		       bp.display_order, p.preorder_eligible,
		       COALESCE((
		         SELECT SUM(ir.quantity) FROM inventory_reservations ir
		         WHERE ir.store_id=bp.store_id AND ir.batch_id=bp.batch_id
		           AND ir.product_id=bp.product_id AND ir.status='active'
		       ),0)
		FROM preorder_batch_products bp
		JOIN products p ON p.id=bp.product_id AND p.store_id=bp.store_id
		WHERE bp.batch_id=$1 AND bp.store_id=$2
		ORDER BY bp.display_order, p.name
	`, batchID, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BatchProduct
	for rows.Next() {
		var bp BatchProduct
		if err := rows.Scan(
			&bp.ID, &bp.BatchID, &bp.ProductID, &bp.ProductName, &bp.ProductSlug, &bp.BasePriceIDR,
			&bp.PriceOverrideIDR, &bp.QuantityLimit, &bp.Availability, &bp.Notes,
			&bp.DisplayOrder, &bp.PreorderEligible, &bp.ReservedQty,
		); err != nil {
			return nil, err
		}
		bp.EffectivePriceIDR = bp.BasePriceIDR
		if bp.PriceOverrideIDR != nil {
			bp.EffectivePriceIDR = *bp.PriceOverrideIDR
		}
		out = append(out, bp)
	}
	return out, rows.Err()
}

func (s *Service) AssignBatchProduct(ctx context.Context, storeID, batchID uuid.UUID, in AssignProductInput) (*BatchProduct, error) {
	if _, err := s.GetBatch(ctx, storeID, batchID, false); err != nil {
		return nil, err
	}
	var eligible bool
	var status string
	err := s.DB.QueryRow(ctx, `
		SELECT preorder_eligible, status FROM products WHERE id=$1 AND store_id=$2
	`, in.ProductID, storeID).Scan(&eligible, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if !eligible {
		return nil, ErrInvalidInput
	}
	if in.Availability == "" {
		in.Availability = "available"
	}
	if in.Availability != "available" && in.Availability != "unavailable" {
		return nil, ErrInvalidInput
	}
	if in.PriceOverrideIDR != nil && *in.PriceOverrideIDR < 0 {
		return nil, ErrInvalidInput
	}
	if in.QuantityLimit != nil && *in.QuantityLimit <= 0 {
		return nil, ErrInvalidInput
	}
	var id uuid.UUID
	err = s.DB.QueryRow(ctx, `
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
		RETURNING id
	`, storeID, batchID, in.ProductID, in.PriceOverrideIDR, in.QuantityLimit,
		in.Availability, in.Notes, in.DisplayOrder,
	).Scan(&id)
	if err != nil {
		return nil, err
	}
	products, err := s.listBatchProducts(ctx, storeID, batchID)
	if err != nil {
		return nil, err
	}
	for i := range products {
		if products[i].ID == id {
			return &products[i], nil
		}
	}
	return nil, ErrNotFound
}

func (s *Service) RemoveBatchProduct(ctx context.Context, storeID, batchID, productID uuid.UUID) error {
	tag, err := s.DB.Exec(ctx, `
		DELETE FROM preorder_batch_products
		WHERE store_id=$1 AND batch_id=$2 AND product_id=$3
	`, storeID, batchID, productID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// PublicBatches lists published open/scheduled/closed batches for storefront.
func (s *Service) PublicBatches(ctx context.Context, storeSlug string) ([]PreorderBatch, error) {
	st, err := s.loadPublicStore(ctx, storeSlug)
	if err != nil {
		return nil, err
	}
	_, _ = s.SyncBatchStatuses(ctx, &st.ID)
	rows, err := s.DB.Query(ctx, `
		SELECT `+batchSelectCols+` FROM preorder_batches
		WHERE store_id=$1 AND publication_status='published'
		  AND operational_status IN ('scheduled','open','closed','purchasing','awaiting_arrival','arrived','fulfilling')
		ORDER BY
		  CASE operational_status WHEN 'open' THEN 0 WHEN 'scheduled' THEN 1 ELSE 2 END,
		  closes_at ASC
	`, st.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PreorderBatch
	for rows.Next() {
		b, err := scanBatch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *b)
	}
	return out, rows.Err()
}

func (s *Service) PublicBatch(ctx context.Context, storeSlug, batchSlug string) (*PreorderBatch, error) {
	st, err := s.loadPublicStore(ctx, storeSlug)
	if err != nil {
		return nil, err
	}
	row := s.DB.QueryRow(ctx, `
		SELECT `+batchSelectCols+` FROM preorder_batches
		WHERE store_id=$1 AND slug=$2 AND publication_status='published'
	`, st.ID, batchSlug)
	b, err := scanBatch(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	_ = s.syncOneBatch(ctx, b)
	products, err := s.listBatchProducts(ctx, st.ID, b.ID)
	if err != nil {
		return nil, err
	}
	// Only available products for public
	var pub []BatchProduct
	for _, p := range products {
		if p.Availability == "available" {
			pub = append(pub, p)
		}
	}
	b.Products = pub
	return b, nil
}

// ProductOpenBatches returns open published batches containing a product (for PDP).
func (s *Service) PublicProductOpenBatches(ctx context.Context, storeSlug, productSlug string) ([]PreorderBatch, error) {
	st, err := s.loadPublicStore(ctx, storeSlug)
	if err != nil {
		return nil, err
	}
	var productID uuid.UUID
	var preorderEligible bool
	err = s.DB.QueryRow(ctx, `
		SELECT id, preorder_eligible FROM products
		WHERE store_id=$1 AND slug=$2 AND status='published'
	`, st.ID, productSlug).Scan(&productID, &preorderEligible)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if !preorderEligible {
		return []PreorderBatch{}, nil
	}
	return s.ProductOpenBatches(ctx, st.ID, productID)
}

func (s *Service) ProductOpenBatches(ctx context.Context, storeID, productID uuid.UUID) ([]PreorderBatch, error) {
	_, _ = s.SyncBatchStatuses(ctx, &storeID)
	rows, err := s.DB.Query(ctx, `
		SELECT b.id, b.store_id, b.name, b.slug, b.description, b.cover_file_id,
		       b.opens_at, b.closes_at, b.purchasing_date, b.eta_start, b.eta_end, b.merchant_notes,
		       b.payment_policy, b.publication_status, b.operational_status, b.closed_early_at,
		       b.created_at, b.updated_at
		FROM preorder_batches b
		JOIN preorder_batch_products bp ON bp.batch_id=b.id AND bp.store_id=b.store_id
		WHERE b.store_id=$1 AND bp.product_id=$2
		  AND b.publication_status='published'
		  AND b.operational_status='open'
		  AND bp.availability='available'
		ORDER BY b.closes_at ASC
	`, storeID, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PreorderBatch
	for rows.Next() {
		b, err := scanBatch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *b)
	}
	return out, rows.Err()
}

func (s *Service) assertBatchOpenForCheckout(ctx context.Context, storeID, batchID uuid.UUID) (*PreorderBatch, error) {
	b, err := s.GetBatch(ctx, storeID, batchID, false)
	if err != nil {
		return nil, err
	}
	if b.PublicationStatus != "published" {
		return nil, ErrBatchNotOpen
	}
	if b.OperationalStatus != BatchOpen {
		return nil, ErrBatchClosed
	}
	now := s.now()
	if now.Before(b.OpensAt) || !now.Before(b.ClosesAt) || b.ClosedEarlyAt != nil {
		return nil, ErrBatchClosed
	}
	return b, nil
}

func (s *Service) checkBatchQuota(ctx context.Context, storeID, batchID, productID uuid.UUID, variantID *uuid.UUID, qty int) error {
	var limit *int
	var availability string
	err := s.DB.QueryRow(ctx, `
		SELECT quantity_limit, availability FROM preorder_batch_products
		WHERE store_id=$1 AND batch_id=$2 AND product_id=$3
	`, storeID, batchID, productID).Scan(&limit, &availability)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if availability != "available" {
		return ErrInsufficientStock
	}
	if limit == nil {
		return nil
	}
	var reserved int
	err = s.DB.QueryRow(ctx, `
		SELECT COALESCE(SUM(quantity),0) FROM inventory_reservations
		WHERE store_id=$1 AND batch_id=$2 AND product_id=$3 AND status='active'
		  AND (($4::uuid IS NULL AND variant_id IS NULL) OR variant_id=$4)
	`, storeID, batchID, productID, variantID).Scan(&reserved)
	if err != nil {
		return err
	}
	if reserved+qty > *limit {
		return ErrQuotaExceeded
	}
	return nil
}
