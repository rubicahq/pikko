package flipbook

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
)

type Page struct {
	ID                uuid.UUID  `json:"id"`
	StoreID           uuid.UUID  `json:"store_id"`
	ImageFileID       uuid.UUID  `json:"image_file_id"`
	ImageURL          string     `json:"image_url,omitempty"`
	DisplayOrder      int        `json:"display_order"`
	Status            string     `json:"status"`
	SourceImportJobID *uuid.UUID `json:"source_import_job_id,omitempty"`
	CatalogID         *uuid.UUID `json:"catalog_id,omitempty"`
	PageNumber        *int       `json:"page_number,omitempty"`
	Hotspots          []Hotspot  `json:"hotspots,omitempty"`
	Items             []PageItem `json:"items,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type Hotspot struct {
	ID                uuid.UUID  `json:"id"`
	CatalogPageID     uuid.UUID  `json:"catalog_page_id"`
	ProductID         *uuid.UUID `json:"product_id,omitempty"`
	ProductSlug       string     `json:"product_slug,omitempty"`
	ProductName       string     `json:"product_name,omitempty"`
	ProductStatus     string     `json:"product_status,omitempty"`
	ImportItemID      *uuid.UUID `json:"import_item_id,omitempty"`
	XPct              float64    `json:"x_pct"`
	YPct              float64    `json:"y_pct"`
	WidthPct          float64    `json:"width_pct"`
	HeightPct         float64    `json:"height_pct"`
	ListOnly          bool       `json:"list_only"`
	BasePriceIDR      *int64     `json:"base_price_idr,omitempty"`
	CompareAtPriceIDR *int64     `json:"compare_at_price_idr,omitempty"`
	ContactForPrice   bool       `json:"contact_for_price"`
	PrimaryImageURL   string     `json:"primary_image_url,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// PageItem is the storefront right-pane row for the active flipbook page.
type PageItem struct {
	ProductID         uuid.UUID `json:"product_id"`
	ProductSlug       string    `json:"product_slug"`
	ProductName       string    `json:"product_name"`
	BasePriceIDR      int64     `json:"base_price_idr"`
	CompareAtPriceIDR *int64    `json:"compare_at_price_idr,omitempty"`
	ContactForPrice   bool      `json:"contact_for_price"`
	PrimaryImageURL   string    `json:"primary_image_url,omitempty"`
	HotspotID         uuid.UUID `json:"hotspot_id,omitempty"`
}

type Service struct {
	DB *pgxpool.Pool
}

func (s *Service) ListPages(ctx context.Context, storeID uuid.UUID, publishedOnly bool) ([]Page, error) {
	q := `
		SELECT id, store_id, image_file_id, display_order, status, source_import_job_id, catalog_id, page_number, created_at, updated_at
		FROM catalog_pages WHERE store_id=$1`
	if publishedOnly {
		q += ` AND status='published'`
	}
	q += ` ORDER BY display_order, page_number NULLS LAST, created_at`
	rows, err := s.DB.Query(ctx, q, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Page
	for rows.Next() {
		var p Page
		if err := rows.Scan(&p.ID, &p.StoreID, &p.ImageFileID, &p.DisplayOrder, &p.Status, &p.SourceImportJobID, &p.CatalogID, &p.PageNumber, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		p.ImageURL = publicFileURL(storeID, p.ImageFileID)
		out = append(out, p)
	}
	for i := range out {
		hs, err := s.listHotspots(ctx, storeID, out[i].ID, publishedOnly)
		if err != nil {
			return nil, err
		}
		out[i].Hotspots = hs
		out[i].Items = pageItemsFromHotspots(hs)
	}
	return out, nil
}

func pageItemsFromHotspots(hs []Hotspot) []PageItem {
	out := make([]PageItem, 0, len(hs))
	seen := map[uuid.UUID]struct{}{}
	for _, h := range hs {
		if h.ProductID == nil || h.ProductName == "" {
			continue
		}
		if _, ok := seen[*h.ProductID]; ok {
			continue
		}
		seen[*h.ProductID] = struct{}{}
		base := int64(0)
		if h.BasePriceIDR != nil {
			base = *h.BasePriceIDR
		}
		out = append(out, PageItem{
			ProductID:         *h.ProductID,
			ProductSlug:       h.ProductSlug,
			ProductName:       h.ProductName,
			BasePriceIDR:      base,
			CompareAtPriceIDR: h.CompareAtPriceIDR,
			ContactForPrice:   h.ContactForPrice,
			PrimaryImageURL:   h.PrimaryImageURL,
			HotspotID:         h.ID,
		})
	}
	return out
}

func publicFileURL(storeID, fileID uuid.UUID) string {
	return fmt.Sprintf("/api/v1/public/files/%s/%s", storeID, fileID)
}

func merchantFileURL(storeID, fileID uuid.UUID) string {
	return fmt.Sprintf("/api/v1/stores/%s/files/%s", storeID, fileID)
}

func (s *Service) listHotspots(ctx context.Context, storeID, pageID uuid.UUID, publishedProductsOnly bool) ([]Hotspot, error) {
	q := `
		SELECT h.id, h.catalog_page_id, h.product_id, h.import_item_id,
		       h.x_pct, h.y_pct, h.width_pct, h.height_pct, COALESCE(h.list_only,false),
		       h.created_at, h.updated_at,
		       p.slug, p.name, p.status, p.base_price_idr, p.compare_at_price_idr, COALESCE(p.contact_for_price,false),
		       (
		         SELECT pi.file_id FROM product_images pi
		         WHERE pi.product_id=p.id AND pi.store_id=p.store_id
		         ORDER BY pi.is_primary DESC, pi.display_order
		         LIMIT 1
		       )
		FROM catalog_page_hotspots h
		LEFT JOIN products p ON p.id = h.product_id AND p.store_id = h.store_id
		WHERE h.store_id=$1 AND h.catalog_page_id=$2
	`
	if publishedProductsOnly {
		q += ` AND (h.product_id IS NULL OR p.status IN ('published','unlisted'))`
	}
	q += ` ORDER BY h.created_at`
	rows, err := s.DB.Query(ctx, q, storeID, pageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Hotspot
	for rows.Next() {
		var h Hotspot
		var slug, name, status *string
		var base *int64
		var imgFile *uuid.UUID
		if err := rows.Scan(
			&h.ID, &h.CatalogPageID, &h.ProductID, &h.ImportItemID,
			&h.XPct, &h.YPct, &h.WidthPct, &h.HeightPct, &h.ListOnly, &h.CreatedAt, &h.UpdatedAt,
			&slug, &name, &status, &base, &h.CompareAtPriceIDR, &h.ContactForPrice, &imgFile,
		); err != nil {
			return nil, err
		}
		if slug != nil {
			h.ProductSlug = *slug
		}
		if name != nil {
			h.ProductName = *name
		}
		if status != nil {
			h.ProductStatus = *status
		}
		h.BasePriceIDR = base
		if imgFile != nil {
			h.PrimaryImageURL = publicFileURL(storeID, *imgFile)
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

type ReorderInput struct {
	PageIDs []uuid.UUID `json:"page_ids"`
}

func (s *Service) Reorder(ctx context.Context, storeID uuid.UUID, in ReorderInput) error {
	if len(in.PageIDs) == 0 {
		return ErrInvalidInput
	}
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for i, id := range in.PageIDs {
		ct, err := tx.Exec(ctx, `
			UPDATE catalog_pages SET display_order=$3, updated_at=now() WHERE id=$1 AND store_id=$2
		`, id, storeID, i)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return ErrNotFound
		}
	}
	return tx.Commit(ctx)
}

func (s *Service) SetPagesStatus(ctx context.Context, storeID uuid.UUID, pageIDs []uuid.UUID, status string) error {
	if status != "draft" && status != "published" {
		return ErrInvalidInput
	}
	if len(pageIDs) == 0 {
		// publish/unpublish all
		_, err := s.DB.Exec(ctx, `
			UPDATE catalog_pages SET status=$2, updated_at=now() WHERE store_id=$1
		`, storeID, status)
		return err
	}
	_, err := s.DB.Exec(ctx, `
		UPDATE catalog_pages SET status=$3, updated_at=now()
		WHERE store_id=$1 AND id = ANY($2)
	`, storeID, pageIDs, status)
	return err
}

type HotspotInput struct {
	ProductID *uuid.UUID `json:"product_id"`
	XPct      float64    `json:"x_pct"`
	YPct      float64    `json:"y_pct"`
	WidthPct  float64    `json:"width_pct"`
	HeightPct float64    `json:"height_pct"`
}

func validBBox(x, y, w, h float64) bool {
	return x >= 0 && x <= 100 && y >= 0 && y <= 100 && w > 0 && w <= 100 && h > 0 && h <= 100 &&
		x+w <= 100.0001 && y+h <= 100.0001
}

// NormalizeBBox clamps hotspot geometry into a valid page percentage box.
func NormalizeBBox(in *HotspotInput) bool {
	if in.WidthPct <= 0 {
		in.WidthPct = 10
	}
	if in.HeightPct <= 0 {
		in.HeightPct = 10
	}
	if in.XPct < 0 {
		in.XPct = 0
	}
	if in.YPct < 0 {
		in.YPct = 0
	}
	if in.WidthPct > 100 {
		in.WidthPct = 100
	}
	if in.HeightPct > 100 {
		in.HeightPct = 100
	}
	if in.XPct > 100 {
		in.XPct = 100
	}
	if in.YPct > 100 {
		in.YPct = 100
	}
	if in.XPct+in.WidthPct > 100 {
		in.XPct = 100 - in.WidthPct
		if in.XPct < 0 {
			in.WidthPct = 100
			in.XPct = 0
		}
	}
	if in.YPct+in.HeightPct > 100 {
		in.YPct = 100 - in.HeightPct
		if in.YPct < 0 {
			in.HeightPct = 100
			in.YPct = 0
		}
	}
	return validBBox(in.XPct, in.YPct, in.WidthPct, in.HeightPct)
}

func (s *Service) CreateHotspot(ctx context.Context, storeID, pageID uuid.UUID, in HotspotInput) (*Hotspot, error) {
	if !NormalizeBBox(&in) {
		return nil, ErrInvalidInput
	}
	var exists bool
	if err := s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM catalog_pages WHERE id=$1 AND store_id=$2)`, pageID, storeID).Scan(&exists); err != nil || !exists {
		return nil, ErrNotFound
	}
	if in.ProductID != nil {
		var ok bool
		if err := s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM products WHERE id=$1 AND store_id=$2 AND status <> 'archived')`, *in.ProductID, storeID).Scan(&ok); err != nil || !ok {
			return nil, ErrInvalidInput
		}
	}
	var id uuid.UUID
	err := s.DB.QueryRow(ctx, `
		INSERT INTO catalog_page_hotspots (store_id, catalog_page_id, product_id, x_pct, y_pct, width_pct, height_pct)
		VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id
	`, storeID, pageID, in.ProductID, in.XPct, in.YPct, in.WidthPct, in.HeightPct).Scan(&id)
	if err != nil {
		return nil, err
	}
	return s.getHotspot(ctx, storeID, id)
}

func (s *Service) UpdateHotspot(ctx context.Context, storeID, hotspotID uuid.UUID, in HotspotInput) (*Hotspot, error) {
	if !NormalizeBBox(&in) {
		return nil, ErrInvalidInput
	}
	if in.ProductID != nil {
		var ok bool
		if err := s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM products WHERE id=$1 AND store_id=$2 AND status <> 'archived')`, *in.ProductID, storeID).Scan(&ok); err != nil || !ok {
			return nil, ErrInvalidInput
		}
	}
	ct, err := s.DB.Exec(ctx, `
		UPDATE catalog_page_hotspots
		SET product_id=$3, x_pct=$4, y_pct=$5, width_pct=$6, height_pct=$7, updated_at=now()
		WHERE id=$1 AND store_id=$2
	`, hotspotID, storeID, in.ProductID, in.XPct, in.YPct, in.WidthPct, in.HeightPct)
	if err != nil {
		return nil, err
	}
	if ct.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return s.getHotspot(ctx, storeID, hotspotID)
}

func (s *Service) DeleteHotspot(ctx context.Context, storeID, hotspotID uuid.UUID) error {
	ct, err := s.DB.Exec(ctx, `DELETE FROM catalog_page_hotspots WHERE id=$1 AND store_id=$2`, hotspotID, storeID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) getHotspot(ctx context.Context, storeID, id uuid.UUID) (*Hotspot, error) {
	var h Hotspot
	var slug, name, status *string
	var base *int64
	var imgFile *uuid.UUID
	err := s.DB.QueryRow(ctx, `
		SELECT h.id, h.catalog_page_id, h.product_id, h.import_item_id,
		       h.x_pct, h.y_pct, h.width_pct, h.height_pct, COALESCE(h.list_only,false),
		       h.created_at, h.updated_at,
		       p.slug, p.name, p.status, p.base_price_idr, p.compare_at_price_idr, COALESCE(p.contact_for_price,false),
		       (
		         SELECT pi.file_id FROM product_images pi
		         WHERE pi.product_id=p.id AND pi.store_id=p.store_id
		         ORDER BY pi.is_primary DESC, pi.display_order
		         LIMIT 1
		       )
		FROM catalog_page_hotspots h
		LEFT JOIN products p ON p.id=h.product_id AND p.store_id=h.store_id
		WHERE h.id=$1 AND h.store_id=$2
	`, id, storeID).Scan(
		&h.ID, &h.CatalogPageID, &h.ProductID, &h.ImportItemID,
		&h.XPct, &h.YPct, &h.WidthPct, &h.HeightPct, &h.ListOnly, &h.CreatedAt, &h.UpdatedAt,
		&slug, &name, &status, &base, &h.CompareAtPriceIDR, &h.ContactForPrice, &imgFile,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if slug != nil {
		h.ProductSlug = *slug
	}
	if name != nil {
		h.ProductName = *name
	}
	if status != nil {
		h.ProductStatus = *status
	}
	h.BasePriceIDR = base
	if imgFile != nil {
		h.PrimaryImageURL = publicFileURL(storeID, *imgFile)
	}
	return &h, nil
}

// PublicStore holds public storefront fields for the catalog viewer.
type PublicStore struct {
	ID       uuid.UUID
	Name     string
	WhatsApp *string
}

// PublicByStoreSlug returns published flipbook pages for a published store.
func (s *Service) PublicByStoreSlug(ctx context.Context, slug string) (store PublicStore, pages []Page, err error) {
	store, err = s.resolvePublicStore(ctx, slug)
	if err != nil {
		return PublicStore{}, nil, err
	}
	pages, err = s.ListPages(ctx, store.ID, true)
	if err != nil {
		return PublicStore{}, nil, err
	}
	return store, filterPublicPages(store.ID, pages), nil
}

// PublicByCatalog returns published pages for one published catalog document.
func (s *Service) PublicByCatalog(ctx context.Context, slug string, catalogID uuid.UUID) (store PublicStore, pages []Page, err error) {
	store, err = s.resolvePublicStore(ctx, slug)
	if err != nil {
		return PublicStore{}, nil, err
	}
	var status string
	err = s.DB.QueryRow(ctx, `
		SELECT status FROM catalogs WHERE id=$1 AND store_id=$2
	`, catalogID, store.ID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return PublicStore{}, nil, ErrNotFound
	}
	if err != nil {
		return PublicStore{}, nil, err
	}
	if status != "published" {
		return PublicStore{}, nil, ErrNotFound
	}
	pages, err = s.ListPagesForCatalog(ctx, store.ID, catalogID, true)
	if err != nil {
		return PublicStore{}, nil, err
	}
	return store, filterPublicPages(store.ID, pages), nil
}

func (s *Service) resolvePublicStore(ctx context.Context, slug string) (PublicStore, error) {
	var store PublicStore
	err := s.DB.QueryRow(ctx, `
		SELECT s.id, s.name, sf.whatsapp
		FROM stores s
		LEFT JOIN storefront_configs sf ON sf.store_id = s.id AND sf.channel = 'published'
		WHERE s.slug=$1 AND s.archived_at IS NULL AND s.publication_status='published'
	`, slug).Scan(&store.ID, &store.Name, &store.WhatsApp)
	if errors.Is(err, pgx.ErrNoRows) {
		return PublicStore{}, ErrNotFound
	}
	if err != nil {
		return PublicStore{}, err
	}
	return store, nil
}

func filterPublicPages(storeID uuid.UUID, pages []Page) []Page {
	for i := range pages {
		pages[i].ImageURL = publicFileURL(storeID, pages[i].ImageFileID)
		filtered := pages[i].Hotspots[:0]
		for _, h := range pages[i].Hotspots {
			if h.ProductID != nil && h.ProductStatus == "published" {
				filtered = append(filtered, h)
			}
		}
		pages[i].Hotspots = filtered
		pages[i].Items = pageItemsFromHotspots(filtered)
	}
	return pages
}

// ListPagesForCatalog returns pages belonging to a catalog document.
func (s *Service) ListPagesForCatalog(ctx context.Context, storeID, catalogID uuid.UUID, publishedOnly bool) ([]Page, error) {
	q := `
		SELECT id, store_id, image_file_id, display_order, status, source_import_job_id, catalog_id, page_number, created_at, updated_at
		FROM catalog_pages WHERE store_id=$1 AND catalog_id=$2`
	if publishedOnly {
		q += ` AND status='published'`
	}
	q += ` ORDER BY display_order, page_number NULLS LAST, created_at`
	rows, err := s.DB.Query(ctx, q, storeID, catalogID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Page
	for rows.Next() {
		var p Page
		if err := rows.Scan(&p.ID, &p.StoreID, &p.ImageFileID, &p.DisplayOrder, &p.Status, &p.SourceImportJobID, &p.CatalogID, &p.PageNumber, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		p.ImageURL = publicFileURL(storeID, p.ImageFileID)
		out = append(out, p)
	}
	for i := range out {
		hs, err := s.listHotspots(ctx, storeID, out[i].ID, publishedOnly)
		if err != nil {
			return nil, err
		}
		out[i].Hotspots = hs
		out[i].Items = pageItemsFromHotspots(hs)
	}
	return out, rows.Err()
}

// MerchantList attaches merchant file URLs.
func (s *Service) MerchantList(ctx context.Context, storeID uuid.UUID) ([]Page, error) {
	pages, err := s.ListPages(ctx, storeID, false)
	if err != nil {
		return nil, err
	}
	for i := range pages {
		pages[i].ImageURL = merchantFileURL(storeID, pages[i].ImageFileID)
	}
	return pages, nil
}
