package commerce

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rubicahq/pikko/services/api/internal/validate"
)

// ErrWishlistUnavailable is returned when the live session is not in scheduled state.
var ErrWishlistUnavailable = errors.New("wishlist unavailable")

type LiveCatalogSummary struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	PageCount int       `json:"page_count"`
	Status    string    `json:"status"`
}

type WishlistItem struct {
	ID            uuid.UUID  `json:"id"`
	StoreID       uuid.UUID  `json:"store_id"`
	CustomerID    uuid.UUID  `json:"customer_id"`
	CampaignID    uuid.UUID  `json:"campaign_id"`
	ProductID     uuid.UUID  `json:"product_id"`
	VariantID     *uuid.UUID `json:"variant_id,omitempty"`
	CatalogID     *uuid.UUID `json:"catalog_id,omitempty"`
	Note          *string    `json:"note,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	ProductName   string     `json:"product_name,omitempty"`
	ProductSlug   string     `json:"product_slug,omitempty"`
	PriceIDR      int64      `json:"price_idr,omitempty"`
	CampaignName  string     `json:"campaign_name,omitempty"`
	CampaignSlug  string     `json:"campaign_slug,omitempty"`
	CustomerName  string     `json:"customer_name,omitempty"`
	CustomerPhone *string    `json:"customer_phone,omitempty"`
}

type AddWishlistInput struct {
	Phone     string     `json:"phone"`
	Name      string     `json:"name,omitempty"`
	ProductID uuid.UUID  `json:"product_id"`
	VariantID *uuid.UUID `json:"variant_id,omitempty"`
	CatalogID *uuid.UUID `json:"catalog_id,omitempty"`
	Note      *string    `json:"note,omitempty"`
}

type AddWishlistResult struct {
	Item            WishlistItem `json:"item"`
	PortalURL       string       `json:"portal_url"`
	StoreSlug       string       `json:"store_slug"`
	CustomerCreated bool         `json:"customer_created"`
	AlreadySaved    bool         `json:"already_saved"`
	CustomerID      uuid.UUID    `json:"customer_id"`
	Phone           string       `json:"phone"`
}

// ListLiveCatalogs returns published catalogs attached to a live campaign (browse while scheduled).
func (s *Service) ListLiveCatalogs(ctx context.Context, storeID, campaignID uuid.UUID) ([]LiveCatalogSummary, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT c.id, c.title, c.status,
		       COALESCE((SELECT COUNT(*) FROM catalog_pages p WHERE p.catalog_id=c.id),0)
		FROM catalogs c
		JOIN catalog_campaigns cc ON cc.catalog_id=c.id AND cc.store_id=c.store_id
		WHERE c.store_id=$1 AND cc.campaign_id=$2 AND c.status='published'
		ORDER BY c.updated_at DESC
	`, storeID, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []LiveCatalogSummary{}
	for rows.Next() {
		var c LiveCatalogSummary
		if err := rows.Scan(&c.ID, &c.Title, &c.Status, &c.PageCount); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Service) assertWishlistScheduled(ctx context.Context, storeID, campaignID uuid.UUID) (LiveSession, error) {
	sess, err := s.assertLiveOpen(ctx, storeID, campaignID)
	if err != nil {
		return sess, err
	}
	if sess.PublicationStatus != "published" || sess.OperationalStatus != "scheduled" {
		return sess, ErrWishlistUnavailable
	}
	return sess, nil
}

func (s *Service) productOnLiveCampaign(ctx context.Context, storeID, campaignID, productID uuid.UUID) (bool, error) {
	var ok bool
	err := s.DB.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM campaign_products cp
			JOIN campaigns c ON c.id=cp.campaign_id
			WHERE cp.campaign_id=$1 AND cp.product_id=$2 AND c.store_id=$3 AND c.campaign_type='live'
		) OR EXISTS(
            SELECT 1 FROM catalog_campaigns cc
            JOIN catalogs cat ON cat.id=cc.catalog_id AND cat.status='published'
            JOIN catalog_pages pg ON pg.catalog_id=cat.id AND pg.status='published'
            JOIN catalog_page_hotspots h ON h.catalog_page_id=pg.id AND h.product_id=$2
            JOIN products p ON p.id=h.product_id AND p.store_id=$3 AND p.status='published'
            WHERE cc.campaign_id=$1 AND cc.store_id=$3
        )
	`, campaignID, productID, storeID).Scan(&ok)
	return ok, err
}

// AddPublicWishlist upserts a customer by phone and saves a wishlist item while live is scheduled.
func (s *Service) AddPublicWishlist(ctx context.Context, storeSlug, liveSlug string, in AddWishlistInput) (*AddWishlistResult, error) {
	st, err := s.loadPublicStore(ctx, storeSlug)
	if err != nil {
		return nil, err
	}
	var campaignID uuid.UUID
	err = s.DB.QueryRow(ctx, `
		SELECT id FROM campaigns
		WHERE store_id=$1 AND slug=$2 AND campaign_type='live' AND publication_status='published'
	`, st.ID, liveSlug).Scan(&campaignID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if _, err := s.assertWishlistScheduled(ctx, st.ID, campaignID); err != nil {
		return nil, err
	}

	phone, ok := validate.NormalizeIndonesianPhone(in.Phone)
	if !ok {
		return nil, ErrInvalidInput
	}
	if in.ProductID == uuid.Nil {
		return nil, ErrInvalidInput
	}
	onCampaign, err := s.productOnLiveCampaign(ctx, st.ID, campaignID, in.ProductID)
	if err != nil {
		return nil, err
	}
	if !onCampaign {
		return nil, ErrInvalidInput
	}

	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = "Pelanggan"
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var customerID uuid.UUID
	var existingName string
	var secret string
	created := false
	err = tx.QueryRow(ctx, `
		SELECT id, name, portal_access_secret FROM customers
		WHERE store_id=$1 AND phone=$2
	`, st.ID, phone).Scan(&customerID, &existingName, &secret)
	if errors.Is(err, pgx.ErrNoRows) {
		secret, err = validate.NewPortalAccessSecret()
		if err != nil {
			return nil, err
		}
		err = tx.QueryRow(ctx, `
			INSERT INTO customers (store_id, name, email, phone, total_orders, completed_order_value_idr, portal_access_secret)
			VALUES ($1,$2,NULL,$3,0,0,$4)
			RETURNING id
		`, st.ID, name, phone, secret).Scan(&customerID)
		if err != nil {
			if strings.Contains(err.Error(), "customers_store_phone_unique") {
				// Race: load existing
				err = tx.QueryRow(ctx, `
					SELECT id, name, portal_access_secret FROM customers
					WHERE store_id=$1 AND phone=$2
				`, st.ID, phone).Scan(&customerID, &existingName, &secret)
				if err != nil {
					return nil, err
				}
			} else {
				return nil, err
			}
		} else {
			created = true
		}
	} else if err != nil {
		return nil, err
	} else if name != "Pelanggan" && (existingName == "" || existingName == "Pelanggan") {
		_, _ = tx.Exec(ctx, `
			UPDATE customers SET name=$3, updated_at=now() WHERE id=$1 AND store_id=$2
		`, customerID, st.ID, name)
	}
	if !validate.ValidPortalAccessSecret(secret) {
		secret, err = validate.NewPortalAccessSecret()
		if err != nil {
			return nil, err
		}
		_, err = tx.Exec(ctx, `
			UPDATE customers SET portal_access_secret=$3, updated_at=now() WHERE id=$1 AND store_id=$2
		`, customerID, st.ID, secret)
		if err != nil {
			return nil, err
		}
	}

	var note *string
	if in.Note != nil {
		n := strings.TrimSpace(*in.Note)
		if n != "" {
			note = &n
		}
	}
	variantKey := uuid.Nil
	if in.VariantID != nil {
		variantKey = *in.VariantID
	}

	var item WishlistItem
	err = tx.QueryRow(ctx, `
		INSERT INTO wishlist_items (store_id, customer_id, campaign_id, product_id, variant_id, variant_key, catalog_id, note)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (customer_id, campaign_id, product_id, variant_key)
		DO NOTHING
		RETURNING id, store_id, customer_id, campaign_id, product_id, variant_id, catalog_id, note, created_at
	`, st.ID, customerID, campaignID, in.ProductID, in.VariantID, variantKey, in.CatalogID, note).Scan(
		&item.ID, &item.StoreID, &item.CustomerID, &item.CampaignID, &item.ProductID,
		&item.VariantID, &item.CatalogID, &item.Note, &item.CreatedAt,
	)
	already := false
	if errors.Is(err, pgx.ErrNoRows) {
		already = true
		err = tx.QueryRow(ctx, `
			SELECT id, store_id, customer_id, campaign_id, product_id, variant_id, catalog_id, note, created_at
			FROM wishlist_items
			WHERE customer_id=$1 AND campaign_id=$2 AND product_id=$3 AND variant_key=$4
		`, customerID, campaignID, in.ProductID, variantKey).Scan(
			&item.ID, &item.StoreID, &item.CustomerID, &item.CampaignID, &item.ProductID,
			&item.VariantID, &item.CatalogID, &item.Note, &item.CreatedAt,
		)
	}
	if err != nil {
		return nil, err
	}

	_ = tx.QueryRow(ctx, `
		SELECT p.name, p.slug,
		       COALESCE(cp.price_override_idr, p.base_price_idr)
		FROM products p
		LEFT JOIN campaign_products cp ON cp.product_id=p.id AND cp.campaign_id=$2
		WHERE p.id=$1 AND p.store_id=$3
	`, in.ProductID, campaignID, st.ID).Scan(&item.ProductName, &item.ProductSlug, &item.PriceIDR)

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	res := &AddWishlistResult{
		Item:            item,
		StoreSlug:       st.Slug,
		CustomerCreated: created,
		AlreadySaved:    already,
		CustomerID:      customerID,
		Phone:           phone,
	}
	if s.URLs != nil {
		res.PortalURL = s.URLs.PortalCustomer(st.Slug, secret, phone)
	}
	return res, nil
}

// ListCustomerWishlist returns wishlist items for a store customer (merchant view).
func (s *Service) ListCustomerWishlist(ctx context.Context, storeID, customerID uuid.UUID) ([]WishlistItem, error) {
	var exists bool
	err := s.DB.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM customers WHERE id=$1 AND store_id=$2)
	`, customerID, storeID).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}

	rows, err := s.DB.Query(ctx, `
		SELECT w.id, w.store_id, w.customer_id, w.campaign_id, w.product_id, w.variant_id, w.catalog_id,
		       w.note, w.created_at,
		       p.name, p.slug,
		       COALESCE(cp.price_override_idr, p.base_price_idr),
		       c.name, c.slug
		FROM wishlist_items w
		JOIN products p ON p.id=w.product_id
		JOIN campaigns c ON c.id=w.campaign_id
		LEFT JOIN campaign_products cp ON cp.campaign_id=w.campaign_id AND cp.product_id=w.product_id
		WHERE w.store_id=$1 AND w.customer_id=$2
		ORDER BY w.created_at DESC
	`, storeID, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []WishlistItem{}
	for rows.Next() {
		var item WishlistItem
		if err := rows.Scan(
			&item.ID, &item.StoreID, &item.CustomerID, &item.CampaignID, &item.ProductID,
			&item.VariantID, &item.CatalogID, &item.Note, &item.CreatedAt,
			&item.ProductName, &item.ProductSlug, &item.PriceIDR,
			&item.CampaignName, &item.CampaignSlug,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// ListPublicWishlist lists a customer's wishlist for a live session using phone identity.
func (s *Service) ListPublicWishlist(ctx context.Context, storeSlug, liveSlug, phoneRaw string) ([]WishlistItem, error) {
	phone, ok := validate.NormalizeIndonesianPhone(phoneRaw)
	if !ok {
		return nil, ErrInvalidInput
	}
	st, err := s.loadPublicStore(ctx, storeSlug)
	if err != nil {
		return nil, err
	}
	var campaignID uuid.UUID
	err = s.DB.QueryRow(ctx, `
		SELECT id FROM campaigns
		WHERE store_id=$1 AND slug=$2 AND campaign_type='live' AND publication_status='published'
	`, st.ID, liveSlug).Scan(&campaignID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var customerID uuid.UUID
	err = s.DB.QueryRow(ctx, `
		SELECT id FROM customers WHERE store_id=$1 AND phone=$2
	`, st.ID, phone).Scan(&customerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return []WishlistItem{}, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := s.DB.Query(ctx, `
		SELECT w.id, w.store_id, w.customer_id, w.campaign_id, w.product_id, w.variant_id, w.catalog_id,
		       w.note, w.created_at,
		       p.name, p.slug,
		       COALESCE(cp.price_override_idr, p.base_price_idr)
		FROM wishlist_items w
		JOIN products p ON p.id=w.product_id
		LEFT JOIN campaign_products cp ON cp.campaign_id=w.campaign_id AND cp.product_id=w.product_id
		WHERE w.store_id=$1 AND w.customer_id=$2 AND w.campaign_id=$3
		ORDER BY w.created_at DESC
	`, st.ID, customerID, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []WishlistItem{}
	for rows.Next() {
		var item WishlistItem
		if err := rows.Scan(
			&item.ID, &item.StoreID, &item.CustomerID, &item.CampaignID, &item.ProductID,
			&item.VariantID, &item.CatalogID, &item.Note, &item.CreatedAt,
			&item.ProductName, &item.ProductSlug, &item.PriceIDR,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// RemovePublicWishlist removes a wishlist item identified by phone + item id.
func (s *Service) RemovePublicWishlist(ctx context.Context, storeSlug, liveSlug, phoneRaw string, itemID uuid.UUID) error {
	phone, ok := validate.NormalizeIndonesianPhone(phoneRaw)
	if !ok {
		return ErrInvalidInput
	}
	st, err := s.loadPublicStore(ctx, storeSlug)
	if err != nil {
		return err
	}
	var campaignID uuid.UUID
	err = s.DB.QueryRow(ctx, `
		SELECT id FROM campaigns
		WHERE store_id=$1 AND slug=$2 AND campaign_type='live' AND publication_status='published'
	`, st.ID, liveSlug).Scan(&campaignID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	var customerID uuid.UUID
	err = s.DB.QueryRow(ctx, `
		SELECT id FROM customers WHERE store_id=$1 AND phone=$2
	`, st.ID, phone).Scan(&customerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	tag, err := s.DB.Exec(ctx, `
		DELETE FROM wishlist_items
		WHERE id=$1 AND store_id=$2 AND customer_id=$3 AND campaign_id=$4
	`, itemID, st.ID, customerID, campaignID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
