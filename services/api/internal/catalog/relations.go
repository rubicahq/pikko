package catalog

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ProductRelation is a soft cross-catalog identity link (same physical item / merge target).
type ProductRelation struct {
	ID               uuid.UUID  `json:"id"`
	StoreID          uuid.UUID  `json:"store_id"`
	ProductID        uuid.UUID  `json:"product_id"`
	RelatedProductID uuid.UUID  `json:"related_product_id"`
	RelationType     string     `json:"relation_type"`
	Notes            *string    `json:"notes,omitempty"`
	CreatedBy        *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	RelatedName      string     `json:"related_name,omitempty"`
	RelatedSlug      string     `json:"related_slug,omitempty"`
	ProductName      string     `json:"product_name,omitempty"`
}

type RelateInput struct {
	RelatedProductID uuid.UUID `json:"related_product_id"`
	RelationType     string    `json:"relation_type"` // same_item | merged_into
	Notes            *string   `json:"notes"`
}

func (s *Service) ListProductRelations(ctx context.Context, storeID, productID uuid.UUID) ([]ProductRelation, error) {
	if _, err := s.GetProduct(ctx, storeID, productID, false); err != nil {
		return nil, err
	}
	rows, err := s.DB.Query(ctx, `
		SELECT r.id, r.store_id, r.product_id, r.related_product_id, r.relation_type, r.notes, r.created_by, r.created_at,
		       p.name, p.slug, src.name
		FROM product_relations r
		JOIN products p ON p.id = r.related_product_id AND p.store_id = r.store_id
		JOIN products src ON src.id = r.product_id AND src.store_id = r.store_id
		WHERE r.store_id=$1 AND (r.product_id=$2 OR r.related_product_id=$2)
		ORDER BY r.created_at DESC
	`, storeID, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProductRelation
	for rows.Next() {
		var r ProductRelation
		if err := rows.Scan(
			&r.ID, &r.StoreID, &r.ProductID, &r.RelatedProductID, &r.RelationType, &r.Notes, &r.CreatedBy, &r.CreatedAt,
			&r.RelatedName, &r.RelatedSlug, &r.ProductName,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Service) RelateProduct(ctx context.Context, storeID, productID uuid.UUID, userID *uuid.UUID, in RelateInput) (*ProductRelation, error) {
	if productID == in.RelatedProductID {
		return nil, fmt.Errorf("%w: cannot relate a product to itself", ErrInvalidInput)
	}
	relType := strings.ToLower(strings.TrimSpace(in.RelationType))
	if relType != "same_item" && relType != "merged_into" {
		return nil, fmt.Errorf("%w: relation_type must be same_item or merged_into", ErrInvalidInput)
	}
	if _, err := s.GetProduct(ctx, storeID, productID, false); err != nil {
		return nil, err
	}
	if _, err := s.GetProduct(ctx, storeID, in.RelatedProductID, false); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, fmt.Errorf("%w: related product not found in this store", ErrInvalidInput)
		}
		return nil, err
	}
	var id uuid.UUID
	err := s.DB.QueryRow(ctx, `
		INSERT INTO product_relations (store_id, product_id, related_product_id, relation_type, notes, created_by)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (store_id, product_id, related_product_id, relation_type)
		DO UPDATE SET notes = EXCLUDED.notes
		RETURNING id
	`, storeID, productID, in.RelatedProductID, relType, in.Notes, userID).Scan(&id)
	if err != nil {
		return nil, err
	}
	return s.getRelation(ctx, storeID, id)
}

func (s *Service) DeleteProductRelation(ctx context.Context, storeID, relationID uuid.UUID) error {
	ct, err := s.DB.Exec(ctx, `
		DELETE FROM product_relations WHERE id=$1 AND store_id=$2
	`, relationID, storeID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) getRelation(ctx context.Context, storeID, id uuid.UUID) (*ProductRelation, error) {
	var r ProductRelation
	err := s.DB.QueryRow(ctx, `
		SELECT r.id, r.store_id, r.product_id, r.related_product_id, r.relation_type, r.notes, r.created_by, r.created_at,
		       p.name, p.slug, src.name
		FROM product_relations r
		JOIN products p ON p.id = r.related_product_id AND p.store_id = r.store_id
		JOIN products src ON src.id = r.product_id AND src.store_id = r.store_id
		WHERE r.id=$1 AND r.store_id=$2
	`, id, storeID).Scan(
		&r.ID, &r.StoreID, &r.ProductID, &r.RelatedProductID, &r.RelationType, &r.Notes, &r.CreatedBy, &r.CreatedAt,
		&r.RelatedName, &r.RelatedSlug, &r.ProductName,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &r, err
}
