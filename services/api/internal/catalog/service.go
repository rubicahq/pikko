package catalog

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
	"github.com/rubicahq/pikko/services/api/internal/validate"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrConflict     = errors.New("conflict")
)

type Category struct {
	ID          uuid.UUID `json:"id"`
	StoreID     uuid.UUID `json:"store_id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description *string   `json:"description,omitempty"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ProductImage struct {
	ID           uuid.UUID `json:"id"`
	FileID       uuid.UUID `json:"file_id"`
	URL          string    `json:"url,omitempty"`
	AltText      *string   `json:"alt_text,omitempty"`
	DisplayOrder int       `json:"display_order"`
	IsPrimary    bool      `json:"is_primary"`
}

type Variant struct {
	ID                uuid.UUID         `json:"id"`
	SKU               *string           `json:"sku,omitempty"`
	PriceOverrideIDR  *int64            `json:"price_override_idr,omitempty"`
	EffectivePriceIDR int64             `json:"effective_price_idr"`
	StockQuantity     *int              `json:"stock_quantity,omitempty"`
	Availability      string            `json:"availability"`
	OptionKey         string            `json:"option_key"`
	Options           map[string]string `json:"options,omitempty"`
}

type ProductOption struct {
	ID     uuid.UUID `json:"id"`
	Name   string    `json:"name"`
	Values []string  `json:"values"`
}

type Product struct {
	ID                 uuid.UUID      `json:"id"`
	StoreID            uuid.UUID      `json:"store_id"`
	Name               string         `json:"name"`
	Slug               string         `json:"slug"`
	Description        *string        `json:"description,omitempty"`
	SKU                *string        `json:"sku,omitempty"`
	Brand              *string        `json:"brand,omitempty"`
	BrandID            *uuid.UUID     `json:"brand_id,omitempty"`
	CategoryID         *uuid.UUID     `json:"category_id,omitempty"`
	CategoryName       *string        `json:"category_name,omitempty"`
	BasePriceIDR       int64          `json:"base_price_idr"`
	CompareAtPriceIDR  *int64         `json:"compare_at_price_idr,omitempty"`
	CostPriceIDR       *int64         `json:"cost_price_idr,omitempty"` // never on public responses
	ContactForPrice    bool           `json:"contact_for_price"`
	ReadyStockEligible bool           `json:"ready_stock_eligible"`
	PreorderEligible   bool           `json:"preorder_eligible"`
	Availability       string         `json:"availability"`
	InventoryMode      string         `json:"inventory_mode"`
	StockQuantity      *int           `json:"stock_quantity,omitempty"`
	WeightGrams        *int           `json:"weight_grams,omitempty"`
	Status             string         `json:"status"`
	Images             []ProductImage `json:"images,omitempty"`
	Options            []ProductOption `json:"options,omitempty"`
	Variants           []Variant      `json:"variants,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	PrimaryImageURL    string         `json:"primary_image_url,omitempty"`
}

type Collection struct {
	ID          uuid.UUID   `json:"id"`
	StoreID     uuid.UUID   `json:"store_id"`
	Name        string      `json:"name"`
	Slug        string      `json:"slug"`
	Description *string     `json:"description,omitempty"`
	Status      string      `json:"status"`
	DisplayOrder int        `json:"display_order"`
	ProductIDs  []uuid.UUID `json:"product_ids,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type Service struct {
	DB *pgxpool.Pool
}

func productSlugify(name string) string {
	return validate.SuggestSlug(name)
}

func (s *Service) ListCategories(ctx context.Context, storeID uuid.UUID) ([]Category, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT id, store_id, name, slug, description, sort_order, created_at, updated_at
		FROM categories WHERE store_id=$1 ORDER BY sort_order, name
	`, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Category{}
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.StoreID, &c.Name, &c.Slug, &c.Description, &c.SortOrder, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

type CategoryInput struct {
	Name        string  `json:"name"`
	Slug        *string `json:"slug"`
	Description *string `json:"description"`
	SortOrder   *int    `json:"sort_order"`
}

func (s *Service) CreateCategory(ctx context.Context, storeID uuid.UUID, in CategoryInput) (*Category, error) {
	name := strings.TrimSpace(in.Name)
	if len(name) < 1 || len(name) > 80 {
		return nil, ErrInvalidInput
	}
	slug := productSlugify(name)
	if in.Slug != nil && strings.TrimSpace(*in.Slug) != "" {
		slug = strings.ToLower(strings.TrimSpace(*in.Slug))
	}
	if !validate.ValidateSlug(slug) {
		return nil, ErrInvalidInput
	}
	sortOrder := 0
	if in.SortOrder != nil {
		sortOrder = *in.SortOrder
	}
	var c Category
	err := s.DB.QueryRow(ctx, `
		INSERT INTO categories (store_id, name, slug, description, sort_order)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING id, store_id, name, slug, description, sort_order, created_at, updated_at
	`, storeID, name, slug, in.Description, sortOrder).Scan(
		&c.ID, &c.StoreID, &c.Name, &c.Slug, &c.Description, &c.SortOrder, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, err
	}
	return &c, nil
}

func (s *Service) UpdateCategory(ctx context.Context, storeID, id uuid.UUID, in CategoryInput) (*Category, error) {
	cur, err := s.getCategory(ctx, storeID, id)
	if err != nil {
		return nil, err
	}
	name := cur.Name
	if strings.TrimSpace(in.Name) != "" {
		name = strings.TrimSpace(in.Name)
	}
	slug := cur.Slug
	if in.Slug != nil && strings.TrimSpace(*in.Slug) != "" {
		slug = strings.ToLower(strings.TrimSpace(*in.Slug))
		if !validate.ValidateSlug(slug) {
			return nil, ErrInvalidInput
		}
	}
	desc := cur.Description
	if in.Description != nil {
		desc = in.Description
	}
	sortOrder := cur.SortOrder
	if in.SortOrder != nil {
		sortOrder = *in.SortOrder
	}
	var c Category
	err = s.DB.QueryRow(ctx, `
		UPDATE categories SET name=$3, slug=$4, description=$5, sort_order=$6, updated_at=now()
		WHERE id=$1 AND store_id=$2
		RETURNING id, store_id, name, slug, description, sort_order, created_at, updated_at
	`, id, storeID, name, slug, desc, sortOrder).Scan(
		&c.ID, &c.StoreID, &c.Name, &c.Slug, &c.Description, &c.SortOrder, &c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if isUniqueViolation(err) {
		return nil, ErrConflict
	}
	return &c, err
}

func (s *Service) DeleteCategory(ctx context.Context, storeID, id uuid.UUID) error {
	ct, err := s.DB.Exec(ctx, `DELETE FROM categories WHERE id=$1 AND store_id=$2`, id, storeID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) getCategory(ctx context.Context, storeID, id uuid.UUID) (*Category, error) {
	var c Category
	err := s.DB.QueryRow(ctx, `
		SELECT id, store_id, name, slug, description, sort_order, created_at, updated_at
		FROM categories WHERE id=$1 AND store_id=$2
	`, id, storeID).Scan(&c.ID, &c.StoreID, &c.Name, &c.Slug, &c.Description, &c.SortOrder, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &c, err
}

type ProductListFilter struct {
	Status     string
	CategoryID *uuid.UUID
	Query      string
	Limit      int
	Offset     int
}

func (s *Service) ListProducts(ctx context.Context, storeID uuid.UUID, f ProductListFilter, publicOnly bool) ([]Product, int, error) {
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 50
	}
	where := []string{"p.store_id=$1"}
	args := []any{storeID}
	n := 2
	if publicOnly {
		where = append(where, "p.status='published'")
	} else if f.Status != "" {
		where = append(where, fmt.Sprintf("p.status=$%d", n))
		args = append(args, f.Status)
		n++
	}
	if f.CategoryID != nil {
		where = append(where, fmt.Sprintf("p.category_id=$%d", n))
		args = append(args, *f.CategoryID)
		n++
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		where = append(where, fmt.Sprintf("(p.name ILIKE $%d OR COALESCE(p.sku,'') ILIKE $%d)", n, n))
		args = append(args, "%"+q+"%")
		n++
	}
	w := strings.Join(where, " AND ")
	var total int
	countArgs := append([]any{}, args...)
	if err := s.DB.QueryRow(ctx, `SELECT COUNT(*) FROM products p WHERE `+w, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, f.Limit, f.Offset)
	q := fmt.Sprintf(`
		SELECT p.id, p.store_id, p.name, p.slug, p.description, p.sku, p.brand, p.brand_id, p.category_id, c.name,
		       p.base_price_idr, p.compare_at_price_idr, p.cost_price_idr, COALESCE(p.contact_for_price,false),
		       p.ready_stock_eligible, p.preorder_eligible, p.availability, p.inventory_mode,
		       p.stock_quantity, p.weight_grams, p.status, p.created_at, p.updated_at
		FROM products p
		LEFT JOIN categories c ON c.id=p.category_id
		WHERE %s
		ORDER BY p.updated_at DESC
		LIMIT $%d OFFSET $%d
	`, w, n, n+1)
	rows, err := s.DB.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []Product{}
	for rows.Next() {
		var p Product
		if err := rows.Scan(
			&p.ID, &p.StoreID, &p.Name, &p.Slug, &p.Description, &p.SKU, &p.Brand, &p.BrandID, &p.CategoryID, &p.CategoryName,
			&p.BasePriceIDR, &p.CompareAtPriceIDR, &p.CostPriceIDR, &p.ContactForPrice,
			&p.ReadyStockEligible, &p.PreorderEligible, &p.Availability, &p.InventoryMode,
			&p.StockQuantity, &p.WeightGrams, &p.Status, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		_ = s.attachImages(ctx, &p)
		out = append(out, p)
	}
	return out, total, nil
}

type ProductInput struct {
	Name               string     `json:"name"`
	Slug               *string    `json:"slug"`
	Description        *string    `json:"description"`
	SKU                *string    `json:"sku"`
	Brand              *string    `json:"brand"`
	BrandID            *uuid.UUID `json:"brand_id"`
	CategoryID         *uuid.UUID `json:"category_id"`
	BasePriceIDR       int64      `json:"base_price_idr"`
	CompareAtPriceIDR  *int64     `json:"compare_at_price_idr"`
	CostPriceIDR       *int64     `json:"cost_price_idr"`
	ContactForPrice    *bool      `json:"contact_for_price"`
	ReadyStockEligible *bool      `json:"ready_stock_eligible"`
	PreorderEligible   *bool      `json:"preorder_eligible"`
	Availability       *string    `json:"availability"`
	InventoryMode      *string    `json:"inventory_mode"`
	StockQuantity      *int       `json:"stock_quantity"`
	WeightGrams        *int       `json:"weight_grams"`
	Status             *string    `json:"status"`
}

type VariantInput struct {
	SKU              *string           `json:"sku"`
	PriceOverrideIDR *int64            `json:"price_override_idr"`
	StockQuantity    *int              `json:"stock_quantity"`
	Availability     string            `json:"availability"`
	Options          map[string]string `json:"options"`
}

type OptionDef struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

type ProductWriteInput struct {
	Product  ProductInput  `json:"product"`
	Options  []OptionDef   `json:"options"`
	Variants []VariantInput `json:"variants"`
	ImageIDs []uuid.UUID   `json:"image_file_ids"`
}

func (s *Service) CreateProduct(ctx context.Context, storeID uuid.UUID, in ProductWriteInput) (*Product, error) {
	p := in.Product
	name := strings.TrimSpace(p.Name)
	if len(name) < 1 || len(name) > 120 {
		return nil, ErrInvalidInput
	}
	if err := money.ValidateIDR(p.BasePriceIDR, true); err != nil {
		return nil, ErrInvalidInput
	}
	slug := productSlugify(name)
	if p.Slug != nil && strings.TrimSpace(*p.Slug) != "" {
		slug = strings.ToLower(strings.TrimSpace(*p.Slug))
	}
	if !validate.ValidateSlug(slug) {
		return nil, ErrInvalidInput
	}
	ready := true
	if p.ReadyStockEligible != nil {
		ready = *p.ReadyStockEligible
	}
	pre := false
	if p.PreorderEligible != nil {
		pre = *p.PreorderEligible
	}
	avail := "available"
	if p.Availability != nil {
		avail = *p.Availability
	}
	inv := "unlimited"
	if p.InventoryMode != nil {
		inv = *p.InventoryMode
	}
	status := "draft"
	if p.Status != nil {
		status = *p.Status
	}
	if !validStatus(status) || !validAvail(avail) || (inv != "finite" && inv != "unlimited") {
		return nil, ErrInvalidInput
	}
	if p.CategoryID != nil {
		if _, err := s.getCategory(ctx, storeID, *p.CategoryID); err != nil {
			return nil, ErrInvalidInput
		}
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var id uuid.UUID
	contact := false
	if p.ContactForPrice != nil {
		contact = *p.ContactForPrice
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO products (
			store_id, name, slug, description, sku, brand, brand_id, category_id, base_price_idr, compare_at_price_idr, cost_price_idr,
			contact_for_price, ready_stock_eligible, preorder_eligible, availability, inventory_mode, stock_quantity, weight_grams, status
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
		RETURNING id
	`, storeID, name, slug, p.Description, p.SKU, p.Brand, p.BrandID, p.CategoryID, p.BasePriceIDR, p.CompareAtPriceIDR, p.CostPriceIDR,
		contact, ready, pre, avail, inv, p.StockQuantity, p.WeightGrams, status).Scan(&id)
	if isUniqueViolation(err) {
		return nil, ErrConflict
	}
	if err != nil {
		return nil, err
	}

	if err := s.writeOptionsVariants(ctx, tx, storeID, id, p.BasePriceIDR, in.Options, in.Variants); err != nil {
		return nil, err
	}
	if err := s.writeImages(ctx, tx, storeID, id, in.ImageIDs); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetProduct(ctx, storeID, id, false)
}

func (s *Service) UpdateProduct(ctx context.Context, storeID, productID uuid.UUID, in ProductWriteInput) (*Product, error) {
	cur, err := s.GetProduct(ctx, storeID, productID, false)
	if err != nil {
		return nil, err
	}
	p := in.Product
	name := cur.Name
	if strings.TrimSpace(p.Name) != "" {
		name = strings.TrimSpace(p.Name)
	}
	slug := cur.Slug
	if p.Slug != nil && strings.TrimSpace(*p.Slug) != "" {
		slug = strings.ToLower(strings.TrimSpace(*p.Slug))
		if !validate.ValidateSlug(slug) {
			return nil, ErrInvalidInput
		}
	}
	base := cur.BasePriceIDR
	if p.BasePriceIDR > 0 || (p.BasePriceIDR == 0 && in.Product.Name != "") {
		// allow explicit 0 only when provided in body with name context; use pointer-less: if base sent as field always use it when name present
		base = p.BasePriceIDR
	}
	if err := money.ValidateIDR(base, true); err != nil {
		return nil, ErrInvalidInput
	}
	status := cur.Status
	if p.Status != nil {
		status = *p.Status
	}
	avail := cur.Availability
	if p.Availability != nil {
		avail = *p.Availability
	}
	inv := cur.InventoryMode
	if p.InventoryMode != nil {
		inv = *p.InventoryMode
	}
	if !validStatus(status) || !validAvail(avail) {
		return nil, ErrInvalidInput
	}
	ready := cur.ReadyStockEligible
	if p.ReadyStockEligible != nil {
		ready = *p.ReadyStockEligible
	}
	pre := cur.PreorderEligible
	if p.PreorderEligible != nil {
		pre = *p.PreorderEligible
	}
	desc := cur.Description
	if p.Description != nil {
		desc = p.Description
	}
	sku := cur.SKU
	if p.SKU != nil {
		sku = p.SKU
	}
	brand := cur.Brand
	if p.Brand != nil {
		brand = p.Brand
	}
	brandID := cur.BrandID
	if p.BrandID != nil {
		brandID = p.BrandID
	}
	cat := cur.CategoryID
	if p.CategoryID != nil {
		cat = p.CategoryID
	}
	compare := cur.CompareAtPriceIDR
	if p.CompareAtPriceIDR != nil {
		compare = p.CompareAtPriceIDR
	}
	cost := cur.CostPriceIDR
	if p.CostPriceIDR != nil {
		cost = p.CostPriceIDR
	}
	stock := cur.StockQuantity
	if p.StockQuantity != nil {
		stock = p.StockQuantity
	}
	weight := cur.WeightGrams
	if p.WeightGrams != nil {
		weight = p.WeightGrams
	}
	contact := cur.ContactForPrice
	if p.ContactForPrice != nil {
		contact = *p.ContactForPrice
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	ct, err := tx.Exec(ctx, `
		UPDATE products SET
			name=$3, slug=$4, description=$5, sku=$6, brand=$7, brand_id=$8, category_id=$9,
			base_price_idr=$10, compare_at_price_idr=$11, cost_price_idr=$12,
			contact_for_price=$13, ready_stock_eligible=$14, preorder_eligible=$15, availability=$16, inventory_mode=$17,
			stock_quantity=$18, weight_grams=$19, status=$20, updated_at=now()
		WHERE id=$1 AND store_id=$2
	`, productID, storeID, name, slug, desc, sku, brand, brandID, cat, base, compare, cost, contact, ready, pre, avail, inv, stock, weight, status)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, err
	}
	if ct.RowsAffected() == 0 {
		return nil, ErrNotFound
	}

	if in.Options != nil || in.Variants != nil {
		_, _ = tx.Exec(ctx, `DELETE FROM product_variants WHERE product_id=$1 AND store_id=$2`, productID, storeID)
		_, _ = tx.Exec(ctx, `
			DELETE FROM product_option_values WHERE option_id IN (
				SELECT id FROM product_options WHERE product_id=$1
			)`, productID)
		_, _ = tx.Exec(ctx, `DELETE FROM product_options WHERE product_id=$1 AND store_id=$2`, productID, storeID)
		if err := s.writeOptionsVariants(ctx, tx, storeID, productID, base, in.Options, in.Variants); err != nil {
			return nil, err
		}
	}
	if in.ImageIDs != nil {
		_, _ = tx.Exec(ctx, `DELETE FROM product_images WHERE product_id=$1 AND store_id=$2`, productID, storeID)
		if err := s.writeImages(ctx, tx, storeID, productID, in.ImageIDs); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetProduct(ctx, storeID, productID, false)
}

func (s *Service) writeOptionsVariants(ctx context.Context, tx pgx.Tx, storeID, productID uuid.UUID, basePrice int64, options []OptionDef, variants []VariantInput) error {
	optionValueIDs := map[string]uuid.UUID{} // "Color=Red" -> id
	for i, opt := range options {
		name := strings.TrimSpace(opt.Name)
		if name == "" {
			return ErrInvalidInput
		}
		var optID uuid.UUID
		if err := tx.QueryRow(ctx, `
			INSERT INTO product_options (store_id, product_id, name, display_order)
			VALUES ($1,$2,$3,$4) RETURNING id
		`, storeID, productID, name, i).Scan(&optID); err != nil {
			return err
		}
		for j, v := range opt.Values {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			var vid uuid.UUID
			if err := tx.QueryRow(ctx, `
				INSERT INTO product_option_values (store_id, option_id, value, display_order)
				VALUES ($1,$2,$3,$4) RETURNING id
			`, storeID, optID, v, j).Scan(&vid); err != nil {
				return err
			}
			optionValueIDs[name+"="+v] = vid
		}
	}
	seenKeys := map[string]struct{}{}
	for _, v := range variants {
		key := optionKey(v.Options)
		if _, ok := seenKeys[key]; ok {
			return ErrConflict
		}
		seenKeys[key] = struct{}{}
		avail := v.Availability
		if avail == "" {
			avail = "available"
		}
		if !validAvail(avail) {
			return ErrInvalidInput
		}
		if v.PriceOverrideIDR != nil {
			if err := money.ValidateIDR(*v.PriceOverrideIDR, true); err != nil {
				return ErrInvalidInput
			}
		}
		var vid uuid.UUID
		if err := tx.QueryRow(ctx, `
			INSERT INTO product_variants (store_id, product_id, sku, price_override_idr, stock_quantity, availability, option_key)
			VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id
		`, storeID, productID, v.SKU, v.PriceOverrideIDR, v.StockQuantity, avail, key).Scan(&vid); err != nil {
			return err
		}
		for name, val := range v.Options {
			ovID, ok := optionValueIDs[name+"="+val]
			if !ok {
				return ErrInvalidInput
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO variant_option_values (variant_id, option_value_id, store_id) VALUES ($1,$2,$3)
			`, vid, ovID, storeID); err != nil {
				return err
			}
		}
		_ = basePrice
	}
	return nil
}

func optionKey(opts map[string]string) string {
	if len(opts) == 0 {
		return ""
	}
	parts := make([]string, 0, len(opts))
	// stable-ish: sort by name
	names := make([]string, 0, len(opts))
	for k := range opts {
		names = append(names, k)
	}
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			if names[j] < names[i] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}
	for _, n := range names {
		parts = append(parts, n+"="+opts[n])
	}
	return strings.Join(parts, "|")
}

func (s *Service) writeImages(ctx context.Context, tx pgx.Tx, storeID, productID uuid.UUID, fileIDs []uuid.UUID) error {
	for i, fid := range fileIDs {
		var purpose string
		err := tx.QueryRow(ctx, `SELECT purpose FROM stored_files WHERE id=$1 AND store_id=$2`, fid, storeID).Scan(&purpose)
		if err != nil || purpose != "product" {
			return ErrInvalidInput
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO product_images (store_id, product_id, file_id, display_order, is_primary)
			VALUES ($1,$2,$3,$4,$5)
		`, storeID, productID, fid, i, i == 0); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) GetProduct(ctx context.Context, storeID, productID uuid.UUID, publicOnly bool) (*Product, error) {
	var p Product
	err := s.DB.QueryRow(ctx, `
		SELECT p.id, p.store_id, p.name, p.slug, p.description, p.sku, p.brand, p.brand_id, p.category_id, c.name,
		       p.base_price_idr, p.compare_at_price_idr, p.cost_price_idr, COALESCE(p.contact_for_price,false),
		       p.ready_stock_eligible, p.preorder_eligible, p.availability, p.inventory_mode,
		       p.stock_quantity, p.weight_grams, p.status, p.created_at, p.updated_at
		FROM products p
		LEFT JOIN categories c ON c.id=p.category_id
		WHERE p.id=$1 AND p.store_id=$2
	`, productID, storeID).Scan(
		&p.ID, &p.StoreID, &p.Name, &p.Slug, &p.Description, &p.SKU, &p.Brand, &p.BrandID, &p.CategoryID, &p.CategoryName,
		&p.BasePriceIDR, &p.CompareAtPriceIDR, &p.CostPriceIDR, &p.ContactForPrice,
		&p.ReadyStockEligible, &p.PreorderEligible, &p.Availability, &p.InventoryMode,
		&p.StockQuantity, &p.WeightGrams, &p.Status, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if publicOnly && p.Status != "published" {
		return nil, ErrNotFound
	}
	if err := s.attachImages(ctx, &p); err != nil {
		return nil, err
	}
	if err := s.attachOptionsVariants(ctx, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Service) GetProductBySlug(ctx context.Context, storeID uuid.UUID, slug string, publicOnly bool) (*Product, error) {
	var id uuid.UUID
	err := s.DB.QueryRow(ctx, `SELECT id FROM products WHERE store_id=$1 AND slug=$2`, storeID, slug).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.GetProduct(ctx, storeID, id, publicOnly)
}

func (s *Service) attachImages(ctx context.Context, p *Product) error {
	rows, err := s.DB.Query(ctx, `
		SELECT id, file_id, alt_text, display_order, is_primary
		FROM product_images WHERE product_id=$1 AND store_id=$2 ORDER BY display_order
	`, p.ID, p.StoreID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var img ProductImage
		if err := rows.Scan(&img.ID, &img.FileID, &img.AltText, &img.DisplayOrder, &img.IsPrimary); err != nil {
			return err
		}
		img.URL = "/api/v1/stores/" + p.StoreID.String() + "/files/" + img.FileID.String()
		if img.IsPrimary {
			p.PrimaryImageURL = img.URL
		}
		p.Images = append(p.Images, img)
	}
	return nil
}

func (s *Service) attachOptionsVariants(ctx context.Context, p *Product) error {
	optRows, err := s.DB.Query(ctx, `
		SELECT id, name FROM product_options WHERE product_id=$1 AND store_id=$2 ORDER BY display_order
	`, p.ID, p.StoreID)
	if err != nil {
		return err
	}
	defer optRows.Close()
	for optRows.Next() {
		var o ProductOption
		var oid uuid.UUID
		if err := optRows.Scan(&oid, &o.Name); err != nil {
			return err
		}
		o.ID = oid
		vrows, err := s.DB.Query(ctx, `
			SELECT value FROM product_option_values WHERE option_id=$1 ORDER BY display_order
		`, oid)
		if err != nil {
			return err
		}
		for vrows.Next() {
			var v string
			if err := vrows.Scan(&v); err != nil {
				vrows.Close()
				return err
			}
			o.Values = append(o.Values, v)
		}
		vrows.Close()
		p.Options = append(p.Options, o)
	}

	vrows, err := s.DB.Query(ctx, `
		SELECT id, sku, price_override_idr, stock_quantity, availability, option_key
		FROM product_variants WHERE product_id=$1 AND store_id=$2 ORDER BY created_at
	`, p.ID, p.StoreID)
	if err != nil {
		return err
	}
	defer vrows.Close()
	for vrows.Next() {
		var v Variant
		if err := vrows.Scan(&v.ID, &v.SKU, &v.PriceOverrideIDR, &v.StockQuantity, &v.Availability, &v.OptionKey); err != nil {
			return err
		}
		if v.PriceOverrideIDR != nil {
			v.EffectivePriceIDR = *v.PriceOverrideIDR
		} else {
			v.EffectivePriceIDR = p.BasePriceIDR
		}
		v.Options = parseOptionKey(v.OptionKey)
		p.Variants = append(p.Variants, v)
	}
	return nil
}

func parseOptionKey(key string) map[string]string {
	out := map[string]string{}
	if key == "" {
		return out
	}
	for _, part := range strings.Split(key, "|") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			out[kv[0]] = kv[1]
		}
	}
	return out
}

func (s *Service) DeleteProduct(ctx context.Context, storeID, productID uuid.UUID) error {
	// Soft archive
	ct, err := s.DB.Exec(ctx, `
		UPDATE products SET status='archived', updated_at=now() WHERE id=$1 AND store_id=$2
	`, productID, storeID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) SetProductStatus(ctx context.Context, storeID, productID uuid.UUID, status string) (*Product, error) {
	if !validStatus(status) {
		return nil, ErrInvalidInput
	}
	ct, err := s.DB.Exec(ctx, `
		UPDATE products SET status=$3, updated_at=now() WHERE id=$1 AND store_id=$2
	`, productID, storeID, status)
	if err != nil {
		return nil, err
	}
	if ct.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return s.GetProduct(ctx, storeID, productID, false)
}

func (s *Service) CreateCollection(ctx context.Context, storeID uuid.UUID, name, slug string, description *string, productIDs []uuid.UUID) (*Collection, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidInput
	}
	if slug == "" {
		slug = productSlugify(name)
	}
	slug = strings.ToLower(strings.TrimSpace(slug))
	if !validate.ValidateSlug(slug) {
		return nil, ErrInvalidInput
	}
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var c Collection
	err = tx.QueryRow(ctx, `
		INSERT INTO collections (store_id, name, slug, description, status)
		VALUES ($1,$2,$3,$4,'draft')
		RETURNING id, store_id, name, slug, description, status, display_order, created_at, updated_at
	`, storeID, name, slug, description).Scan(
		&c.ID, &c.StoreID, &c.Name, &c.Slug, &c.Description, &c.Status, &c.DisplayOrder, &c.CreatedAt, &c.UpdatedAt,
	)
	if isUniqueViolation(err) {
		return nil, ErrConflict
	}
	if err != nil {
		return nil, err
	}
	for i, pid := range productIDs {
		var ok bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM products WHERE id=$1 AND store_id=$2)`, pid, storeID).Scan(&ok); err != nil || !ok {
			return nil, ErrInvalidInput
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO collection_products (collection_id, product_id, store_id, display_order) VALUES ($1,$2,$3,$4)
		`, c.ID, pid, storeID, i); err != nil {
			return nil, err
		}
	}
	c.ProductIDs = productIDs
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Service) ListCollections(ctx context.Context, storeID uuid.UUID, publicOnly bool) ([]Collection, error) {
	q := `SELECT id, store_id, name, slug, description, status, display_order, created_at, updated_at FROM collections WHERE store_id=$1`
	if publicOnly {
		q += ` AND status='published'`
	}
	q += ` ORDER BY display_order, name`
	rows, err := s.DB.Query(ctx, q, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Collection{}
	for rows.Next() {
		var c Collection
		if err := rows.Scan(&c.ID, &c.StoreID, &c.Name, &c.Slug, &c.Description, &c.Status, &c.DisplayOrder, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func (s *Service) SetupChecklist(ctx context.Context, storeID uuid.UUID) (map[string]bool, error) {
	var hasProduct, hasPublishedProduct, hasCustomTheme, published bool
	var hasPaymentMethods, hasFulfillment bool
	_ = s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM products WHERE store_id=$1 AND status<>'archived')`, storeID).Scan(&hasProduct)
	_ = s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM products WHERE store_id=$1 AND status='published')`, storeID).Scan(&hasPublishedProduct)
	_ = s.DB.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM storefront_configs
			WHERE store_id=$1 AND channel='draft' AND (
				theme_id<>'pikko_signature' OR logo_file_id IS NOT NULL OR description IS NOT NULL
			)
		)`, storeID).Scan(&hasCustomTheme)
	_ = s.DB.QueryRow(ctx, `SELECT publication_status='published' FROM stores WHERE id=$1`, storeID).Scan(&published)
	_ = s.DB.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM store_payment_methods WHERE store_id=$1 AND is_enabled=true)
	`, storeID).Scan(&hasPaymentMethods)
	_ = s.DB.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM store_fulfillment_settings
			WHERE store_id=$1 AND (shipping_enabled OR pickup_enabled)
		)
	`, storeID).Scan(&hasFulfillment)
	return map[string]bool{
		"create_store":         true,
		"first_product":        hasProduct,
		"customize_storefront": hasCustomTheme,
		"payment_methods":      hasPaymentMethods,
		"fulfillment":          hasFulfillment,
		"publish_and_share":    published && hasPublishedProduct,
	}, nil
}

func validStatus(s string) bool {
	return s == "draft" || s == "published" || s == "archived"
}

func validAvail(s string) bool {
	return s == "available" || s == "unavailable"
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint")
}

// StripPrivate removes merchant-only fields for public API and rewrites
// image URLs to the public file endpoint (usable by storefront + future flipbook hotspots).
func StripPrivate(p *Product) {
	p.CostPriceIDR = nil
	for i := range p.Images {
		p.Images[i].URL = PublicFileURL(p.StoreID, p.Images[i].FileID)
		if p.Images[i].IsPrimary {
			p.PrimaryImageURL = p.Images[i].URL
		}
	}
}

// PublicFileURL is the stable public asset path for a store-scoped file.
func PublicFileURL(storeID, fileID uuid.UUID) string {
	return "/api/v1/public/files/" + storeID.String() + "/" + fileID.String()
}

// LinkRef is the minimal product handle for catalog-page / hotspot linking.
type LinkRef struct {
	ID               uuid.UUID `json:"id"`
	StoreID          uuid.UUID `json:"store_id"`
	Slug             string    `json:"slug"`
	Name             string    `json:"name"`
	SKU              *string   `json:"sku,omitempty"`
	Status           string    `json:"status"`
	BasePriceIDR     int64     `json:"base_price_idr"`
	PrimaryImageURL  string    `json:"primary_image_url,omitempty"`
}

func (p *Product) LinkRef() LinkRef {
	return LinkRef{
		ID: p.ID, StoreID: p.StoreID, Slug: p.Slug, Name: p.Name, SKU: p.SKU,
		Status: p.Status, BasePriceIDR: p.BasePriceIDR, PrimaryImageURL: p.PrimaryImageURL,
	}
}
