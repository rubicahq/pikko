package commerce

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type BatchVariant struct {
	ID                uuid.UUID `json:"id"`
	BatchID           uuid.UUID `json:"batch_id"`
	BatchProductID    uuid.UUID `json:"batch_product_id"`
	ProductID         uuid.UUID `json:"product_id"`
	VariantID         uuid.UUID `json:"variant_id"`
	VariantLabel      string    `json:"variant_label,omitempty"`
	SKU               *string   `json:"sku,omitempty"`
	BasePriceIDR      int64     `json:"base_price_idr"`
	PriceOverrideIDR  *int64    `json:"price_override_idr,omitempty"`
	EffectivePriceIDR int64     `json:"effective_price_idr"`
	QuantityLimit     *int      `json:"quantity_limit,omitempty"`
	Availability      string    `json:"availability"`
}

type AssignVariantInput struct {
	VariantID        uuid.UUID `json:"variant_id"`
	PriceOverrideIDR *int64    `json:"price_override_idr,omitempty"`
	QuantityLimit    *int      `json:"quantity_limit,omitempty"`
	Availability     string    `json:"availability"`
}

func (s *Service) ListBatchVariants(ctx context.Context, storeID, batchID, productID uuid.UUID) ([]BatchVariant, error) {
	if _, err := s.GetBatch(ctx, storeID, batchID, false); err != nil {
		return nil, err
	}
	rows, err := s.DB.Query(ctx, `
		SELECT bv.id, bv.batch_id, bv.batch_product_id, bv.product_id, bv.variant_id,
		       COALESCE(NULLIF(pv.option_key,''), 'Default'),
		       pv.sku,
		       COALESCE(pv.price_override_idr, p.base_price_idr),
		       bv.price_override_idr, bv.quantity_limit, bv.availability
		FROM preorder_batch_variants bv
		JOIN product_variants pv ON pv.id=bv.variant_id AND pv.store_id=bv.store_id
		JOIN products p ON p.id=bv.product_id AND p.store_id=bv.store_id
		WHERE bv.store_id=$1 AND bv.batch_id=$2 AND bv.product_id=$3
		ORDER BY pv.option_key
	`, storeID, batchID, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BatchVariant
	for rows.Next() {
		var v BatchVariant
		if err := rows.Scan(
			&v.ID, &v.BatchID, &v.BatchProductID, &v.ProductID, &v.VariantID,
			&v.VariantLabel, &v.SKU, &v.BasePriceIDR,
			&v.PriceOverrideIDR, &v.QuantityLimit, &v.Availability,
		); err != nil {
			return nil, err
		}
		v.EffectivePriceIDR = v.BasePriceIDR
		if v.PriceOverrideIDR != nil {
			v.EffectivePriceIDR = *v.PriceOverrideIDR
		}
		out = append(out, v)
	}
	if out == nil {
		out = []BatchVariant{}
	}
	return out, rows.Err()
}

func (s *Service) AssignBatchVariant(ctx context.Context, storeID, batchID, productID uuid.UUID, in AssignVariantInput) (*BatchVariant, error) {
	if _, err := s.GetBatch(ctx, storeID, batchID, false); err != nil {
		return nil, err
	}
	var batchProductID uuid.UUID
	err := s.DB.QueryRow(ctx, `
		SELECT id FROM preorder_batch_products
		WHERE store_id=$1 AND batch_id=$2 AND product_id=$3
	`, storeID, batchID, productID).Scan(&batchProductID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var variantProductID uuid.UUID
	err = s.DB.QueryRow(ctx, `
		SELECT product_id FROM product_variants WHERE id=$1 AND store_id=$2
	`, in.VariantID, storeID).Scan(&variantProductID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if variantProductID != productID {
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
	_, err = s.DB.Exec(ctx, `
		INSERT INTO preorder_batch_variants (
			store_id, batch_id, batch_product_id, product_id, variant_id,
			price_override_idr, quantity_limit, availability
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (batch_id, variant_id) DO UPDATE SET
			price_override_idr=EXCLUDED.price_override_idr,
			quantity_limit=EXCLUDED.quantity_limit,
			availability=EXCLUDED.availability
	`, storeID, batchID, batchProductID, productID, in.VariantID,
		in.PriceOverrideIDR, in.QuantityLimit, in.Availability)
	if err != nil {
		return nil, err
	}
	list, err := s.ListBatchVariants(ctx, storeID, batchID, productID)
	if err != nil {
		return nil, err
	}
	for i := range list {
		if list[i].VariantID == in.VariantID {
			return &list[i], nil
		}
	}
	return nil, ErrNotFound
}

func (s *Service) RemoveBatchVariant(ctx context.Context, storeID, batchID, productID, variantID uuid.UUID) error {
	tag, err := s.DB.Exec(ctx, `
		DELETE FROM preorder_batch_variants
		WHERE store_id=$1 AND batch_id=$2 AND product_id=$3 AND variant_id=$4
	`, storeID, batchID, productID, variantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
