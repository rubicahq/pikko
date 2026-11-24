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

type Brand struct {
	ID          uuid.UUID  `json:"id"`
	StoreID     uuid.UUID  `json:"store_id"`
	Name        string     `json:"name"`
	Slug        string     `json:"slug"`
	Description *string    `json:"description,omitempty"`
	LogoFileID  *uuid.UUID `json:"logo_file_id,omitempty"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type BrandInput struct {
	Name        string     `json:"name"`
	Slug        *string    `json:"slug"`
	Description *string    `json:"description"`
	LogoFileID  *uuid.UUID `json:"logo_file_id"`
	Status      *string    `json:"status"`
}

func (s *Service) ListBrands(ctx context.Context, storeID uuid.UUID, includeArchived bool) ([]Brand, error) {
	q := `
		SELECT id, store_id, name, slug, description, logo_file_id, status, created_at, updated_at
		FROM brands WHERE store_id=$1`
	if !includeArchived {
		q += ` AND status='active'`
	}
	q += ` ORDER BY name`
	rows, err := s.DB.Query(ctx, q, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Brand{}
	for rows.Next() {
		var b Brand
		if err := rows.Scan(&b.ID, &b.StoreID, &b.Name, &b.Slug, &b.Description, &b.LogoFileID, &b.Status, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (s *Service) CreateBrand(ctx context.Context, storeID uuid.UUID, in BrandInput) (*Brand, error) {
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
	var b Brand
	err := s.DB.QueryRow(ctx, `
		INSERT INTO brands (store_id, name, slug, description, logo_file_id, status)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id, store_id, name, slug, description, logo_file_id, status, created_at, updated_at
	`, storeID, name, slug, in.Description, in.LogoFileID, status).Scan(
		&b.ID, &b.StoreID, &b.Name, &b.Slug, &b.Description, &b.LogoFileID, &b.Status, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "brands_store_slug_unique") {
			return nil, ErrConflict
		}
		return nil, err
	}
	return &b, nil
}

func (s *Service) UpdateBrand(ctx context.Context, storeID, brandID uuid.UUID, in BrandInput) (*Brand, error) {
	cur, err := s.GetBrand(ctx, storeID, brandID)
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
	logo := cur.LogoFileID
	if in.LogoFileID != nil {
		logo = in.LogoFileID
	}
	status := cur.Status
	if in.Status != nil && *in.Status != "" {
		status = *in.Status
		if status != "active" && status != "archived" {
			return nil, ErrInvalidInput
		}
	}
	var b Brand
	err = s.DB.QueryRow(ctx, `
		UPDATE brands SET name=$3, slug=$4, description=$5, logo_file_id=$6, status=$7, updated_at=now()
		WHERE id=$1 AND store_id=$2
		RETURNING id, store_id, name, slug, description, logo_file_id, status, created_at, updated_at
	`, brandID, storeID, name, slug, desc, logo, status).Scan(
		&b.ID, &b.StoreID, &b.Name, &b.Slug, &b.Description, &b.LogoFileID, &b.Status, &b.CreatedAt, &b.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		if strings.Contains(err.Error(), "brands_store_slug_unique") {
			return nil, ErrConflict
		}
		return nil, err
	}
	return &b, nil
}

func (s *Service) GetBrand(ctx context.Context, storeID, brandID uuid.UUID) (*Brand, error) {
	var b Brand
	err := s.DB.QueryRow(ctx, `
		SELECT id, store_id, name, slug, description, logo_file_id, status, created_at, updated_at
		FROM brands WHERE id=$1 AND store_id=$2
	`, brandID, storeID).Scan(
		&b.ID, &b.StoreID, &b.Name, &b.Slug, &b.Description, &b.LogoFileID, &b.Status, &b.CreatedAt, &b.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}
