package catalog

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rubicahq/pikko/services/api/internal/validate"
)

type Vendor struct {
	ID           uuid.UUID `json:"id"`
	StoreID      uuid.UUID `json:"store_id"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	ContactName  *string   `json:"contact_name,omitempty"`
	ContactEmail *string   `json:"contact_email,omitempty"`
	ContactPhone *string   `json:"contact_phone,omitempty"`
	Notes        *string   `json:"notes,omitempty"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type VendorInput struct {
	Name         string  `json:"name"`
	Slug         *string `json:"slug"`
	ContactName  *string `json:"contact_name"`
	ContactEmail *string `json:"contact_email"`
	ContactPhone *string `json:"contact_phone"`
	Notes        *string `json:"notes"`
	Status       *string `json:"status"`
}

type ProductVendorLink struct {
	ProductID   uuid.UUID `json:"product_id"`
	VendorID    uuid.UUID `json:"vendor_id"`
	VendorName  string    `json:"vendor_name,omitempty"`
	IsPreferred bool      `json:"is_preferred"`
	Notes       *string   `json:"notes,omitempty"`
}

type LinkVendorInput struct {
	VendorID    uuid.UUID `json:"vendor_id"`
	IsPreferred bool      `json:"is_preferred"`
	Notes       *string   `json:"notes"`
}

func (s *Service) ListVendors(ctx context.Context, storeID uuid.UUID, includeArchived bool) ([]Vendor, error) {
	q := `
		SELECT id, store_id, name, slug, contact_name, contact_email, contact_phone, notes, status, created_at, updated_at
		FROM vendors WHERE store_id=$1`
	if !includeArchived {
		q += ` AND status='active'`
	}
	q += ` ORDER BY name`
	rows, err := s.DB.Query(ctx, q, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Vendor{}
	for rows.Next() {
		var v Vendor
		if err := rows.Scan(&v.ID, &v.StoreID, &v.Name, &v.Slug, &v.ContactName, &v.ContactEmail, &v.ContactPhone, &v.Notes, &v.Status, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Service) CreateVendor(ctx context.Context, storeID uuid.UUID, in VendorInput) (*Vendor, error) {
	name := strings.TrimSpace(in.Name)
	if len(name) < 1 || len(name) > 120 {
		return nil, ErrInvalidInput
	}
	slug := productSlugify(name)
	if in.Slug != nil && strings.TrimSpace(*in.Slug) != "" {
		slug = strings.ToLower(strings.TrimSpace(*in.Slug))
	}
	if !validate.ValidateSlug(slug) {
		return nil, ErrInvalidInput
	}
	status := "active"
	if in.Status != nil && *in.Status != "" {
		status = *in.Status
	}
	if status != "active" && status != "archived" {
		return nil, ErrInvalidInput
	}
	var v Vendor
	err := s.DB.QueryRow(ctx, `
		INSERT INTO vendors (store_id, name, slug, contact_name, contact_email, contact_phone, notes, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id, store_id, name, slug, contact_name, contact_email, contact_phone, notes, status, created_at, updated_at
	`, storeID, name, slug, in.ContactName, in.ContactEmail, in.ContactPhone, in.Notes, status).Scan(
		&v.ID, &v.StoreID, &v.Name, &v.Slug, &v.ContactName, &v.ContactEmail, &v.ContactPhone, &v.Notes, &v.Status, &v.CreatedAt, &v.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "vendors_store_slug_unique") {
			return nil, ErrConflict
		}
		return nil, err
	}
	return &v, nil
}

func (s *Service) UpdateVendor(ctx context.Context, storeID, vendorID uuid.UUID, in VendorInput) (*Vendor, error) {
	cur, err := s.getVendor(ctx, storeID, vendorID)
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
	contactName := cur.ContactName
	if in.ContactName != nil {
		contactName = in.ContactName
	}
	contactEmail := cur.ContactEmail
	if in.ContactEmail != nil {
		contactEmail = in.ContactEmail
	}
	contactPhone := cur.ContactPhone
	if in.ContactPhone != nil {
		contactPhone = in.ContactPhone
	}
	notes := cur.Notes
	if in.Notes != nil {
		notes = in.Notes
	}
	status := cur.Status
	if in.Status != nil && *in.Status != "" {
		status = *in.Status
		if status != "active" && status != "archived" {
			return nil, ErrInvalidInput
		}
	}
	var v Vendor
	err = s.DB.QueryRow(ctx, `
		UPDATE vendors SET name=$3, slug=$4, contact_name=$5, contact_email=$6, contact_phone=$7, notes=$8, status=$9, updated_at=now()
		WHERE id=$1 AND store_id=$2
		RETURNING id, store_id, name, slug, contact_name, contact_email, contact_phone, notes, status, created_at, updated_at
	`, vendorID, storeID, name, slug, contactName, contactEmail, contactPhone, notes, status).Scan(
		&v.ID, &v.StoreID, &v.Name, &v.Slug, &v.ContactName, &v.ContactEmail, &v.ContactPhone, &v.Notes, &v.Status, &v.CreatedAt, &v.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		if strings.Contains(err.Error(), "vendors_store_slug_unique") {
			return nil, ErrConflict
		}
		return nil, err
	}
	return &v, nil
}

func (s *Service) getVendor(ctx context.Context, storeID, vendorID uuid.UUID) (*Vendor, error) {
	var v Vendor
	err := s.DB.QueryRow(ctx, `
		SELECT id, store_id, name, slug, contact_name, contact_email, contact_phone, notes, status, created_at, updated_at
		FROM vendors WHERE id=$1 AND store_id=$2
	`, vendorID, storeID).Scan(
		&v.ID, &v.StoreID, &v.Name, &v.Slug, &v.ContactName, &v.ContactEmail, &v.ContactPhone, &v.Notes, &v.Status, &v.CreatedAt, &v.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (s *Service) ListProductVendors(ctx context.Context, storeID, productID uuid.UUID) ([]ProductVendorLink, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT pv.product_id, pv.vendor_id, v.name, pv.is_preferred, pv.notes
		FROM product_vendors pv
		JOIN vendors v ON v.id = pv.vendor_id
		WHERE pv.store_id=$1 AND pv.product_id=$2
		ORDER BY pv.is_preferred DESC, v.name
	`, storeID, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ProductVendorLink{}
	for rows.Next() {
		var l ProductVendorLink
		if err := rows.Scan(&l.ProductID, &l.VendorID, &l.VendorName, &l.IsPreferred, &l.Notes); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (s *Service) LinkProductVendor(ctx context.Context, storeID, productID uuid.UUID, in LinkVendorInput) (*ProductVendorLink, error) {
	var ok bool
	err := s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM products WHERE id=$1 AND store_id=$2)`, productID, storeID).Scan(&ok)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	err = s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM vendors WHERE id=$1 AND store_id=$2)`, in.VendorID, storeID).Scan(&ok)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	_, err = s.DB.Exec(ctx, `
		INSERT INTO product_vendors (product_id, vendor_id, store_id, is_preferred, notes)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (product_id, vendor_id) DO UPDATE SET
			is_preferred=EXCLUDED.is_preferred,
			notes=EXCLUDED.notes
	`, productID, in.VendorID, storeID, in.IsPreferred, in.Notes)
	if err != nil {
		return nil, err
	}
	list, err := s.ListProductVendors(ctx, storeID, productID)
	if err != nil {
		return nil, err
	}
	for i := range list {
		if list[i].VendorID == in.VendorID {
			return &list[i], nil
		}
	}
	return nil, ErrNotFound
}

func (s *Service) UnlinkProductVendor(ctx context.Context, storeID, productID, vendorID uuid.UUID) error {
	tag, err := s.DB.Exec(ctx, `
		DELETE FROM product_vendors WHERE store_id=$1 AND product_id=$2 AND vendor_id=$3
	`, storeID, productID, vendorID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
