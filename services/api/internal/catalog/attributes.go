package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type AttributeDefinition struct {
	ID           uuid.UUID       `json:"id"`
	StoreID      uuid.UUID       `json:"store_id"`
	Key          string          `json:"key"`
	Label        string          `json:"label"`
	DataType     string          `json:"data_type"`
	Options      json.RawMessage `json:"options"`
	AppliesTo    string          `json:"applies_to"`
	IsRequired   bool            `json:"is_required"`
	DisplayOrder int             `json:"display_order"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type AttributeDefinitionInput struct {
	Key          string          `json:"key"`
	Label        string          `json:"label"`
	DataType     string          `json:"data_type"`
	Options      json.RawMessage `json:"options"`
	AppliesTo    string          `json:"applies_to"`
	IsRequired   *bool           `json:"is_required"`
	DisplayOrder *int            `json:"display_order"`
}

type ProductAttributeValue struct {
	ID            uuid.UUID `json:"id"`
	ProductID     uuid.UUID `json:"product_id"`
	AttributeID   uuid.UUID `json:"attribute_id"`
	AttributeKey  string    `json:"attribute_key,omitempty"`
	AttributeLabel string   `json:"attribute_label,omitempty"`
	DataType      string    `json:"data_type,omitempty"`
	ValueText     *string   `json:"value_text,omitempty"`
	ValueNumber   *float64  `json:"value_number,omitempty"`
	ValueBoolean  *bool     `json:"value_boolean,omitempty"`
}

type SetAttributeValueInput struct {
	AttributeID  uuid.UUID `json:"attribute_id"`
	ValueText    *string   `json:"value_text"`
	ValueNumber  *float64  `json:"value_number"`
	ValueBoolean *bool     `json:"value_boolean"`
}

func (s *Service) ListAttributeDefinitions(ctx context.Context, storeID uuid.UUID) ([]AttributeDefinition, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT id, store_id, key, label, data_type, options, applies_to, is_required, display_order, created_at, updated_at
		FROM attribute_definitions WHERE store_id=$1
		ORDER BY display_order, label
	`, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AttributeDefinition{}
	for rows.Next() {
		var a AttributeDefinition
		if err := rows.Scan(&a.ID, &a.StoreID, &a.Key, &a.Label, &a.DataType, &a.Options, &a.AppliesTo, &a.IsRequired, &a.DisplayOrder, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		if len(a.Options) == 0 {
			a.Options = json.RawMessage("[]")
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Service) CreateAttributeDefinition(ctx context.Context, storeID uuid.UUID, in AttributeDefinitionInput) (*AttributeDefinition, error) {
	key := strings.TrimSpace(strings.ToLower(in.Key))
	label := strings.TrimSpace(in.Label)
	if key == "" || label == "" {
		return nil, ErrInvalidInput
	}
	dt := strings.TrimSpace(in.DataType)
	if dt != "text" && dt != "number" && dt != "boolean" && dt != "select" {
		return nil, ErrInvalidInput
	}
	applies := in.AppliesTo
	if applies == "" {
		applies = "product"
	}
	if applies != "product" && applies != "variant" {
		return nil, ErrInvalidInput
	}
	opts := in.Options
	if len(opts) == 0 {
		opts = json.RawMessage("[]")
	}
	req := false
	if in.IsRequired != nil {
		req = *in.IsRequired
	}
	order := 0
	if in.DisplayOrder != nil {
		order = *in.DisplayOrder
	}
	var a AttributeDefinition
	err := s.DB.QueryRow(ctx, `
		INSERT INTO attribute_definitions (store_id, key, label, data_type, options, applies_to, is_required, display_order)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id, store_id, key, label, data_type, options, applies_to, is_required, display_order, created_at, updated_at
	`, storeID, key, label, dt, opts, applies, req, order).Scan(
		&a.ID, &a.StoreID, &a.Key, &a.Label, &a.DataType, &a.Options, &a.AppliesTo, &a.IsRequired, &a.DisplayOrder, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "attribute_definitions_store_key_unique") ||
			strings.Contains(err.Error(), "attribute_definitions_key_format") {
			return nil, ErrInvalidInput
		}
		return nil, err
	}
	return &a, nil
}

func (s *Service) UpdateAttributeDefinition(ctx context.Context, storeID, attrID uuid.UUID, in AttributeDefinitionInput) (*AttributeDefinition, error) {
	cur, err := s.getAttributeDefinition(ctx, storeID, attrID)
	if err != nil {
		return nil, err
	}
	label := cur.Label
	if strings.TrimSpace(in.Label) != "" {
		label = strings.TrimSpace(in.Label)
	}
	opts := cur.Options
	if len(in.Options) > 0 {
		opts = in.Options
	}
	req := cur.IsRequired
	if in.IsRequired != nil {
		req = *in.IsRequired
	}
	order := cur.DisplayOrder
	if in.DisplayOrder != nil {
		order = *in.DisplayOrder
	}
	var a AttributeDefinition
	err = s.DB.QueryRow(ctx, `
		UPDATE attribute_definitions SET label=$3, options=$4, is_required=$5, display_order=$6, updated_at=now()
		WHERE id=$1 AND store_id=$2
		RETURNING id, store_id, key, label, data_type, options, applies_to, is_required, display_order, created_at, updated_at
	`, attrID, storeID, label, opts, req, order).Scan(
		&a.ID, &a.StoreID, &a.Key, &a.Label, &a.DataType, &a.Options, &a.AppliesTo, &a.IsRequired, &a.DisplayOrder, &a.CreatedAt, &a.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *Service) getAttributeDefinition(ctx context.Context, storeID, attrID uuid.UUID) (*AttributeDefinition, error) {
	var a AttributeDefinition
	err := s.DB.QueryRow(ctx, `
		SELECT id, store_id, key, label, data_type, options, applies_to, is_required, display_order, created_at, updated_at
		FROM attribute_definitions WHERE id=$1 AND store_id=$2
	`, attrID, storeID).Scan(
		&a.ID, &a.StoreID, &a.Key, &a.Label, &a.DataType, &a.Options, &a.AppliesTo, &a.IsRequired, &a.DisplayOrder, &a.CreatedAt, &a.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *Service) ListProductAttributes(ctx context.Context, storeID, productID uuid.UUID) ([]ProductAttributeValue, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT pav.id, pav.product_id, pav.attribute_id, ad.key, ad.label, ad.data_type,
		       pav.value_text, pav.value_number, pav.value_boolean
		FROM product_attribute_values pav
		JOIN attribute_definitions ad ON ad.id = pav.attribute_id
		WHERE pav.store_id=$1 AND pav.product_id=$2
		ORDER BY ad.display_order, ad.label
	`, storeID, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ProductAttributeValue{}
	for rows.Next() {
		var v ProductAttributeValue
		if err := rows.Scan(&v.ID, &v.ProductID, &v.AttributeID, &v.AttributeKey, &v.AttributeLabel, &v.DataType, &v.ValueText, &v.ValueNumber, &v.ValueBoolean); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Service) SetProductAttribute(ctx context.Context, storeID, productID uuid.UUID, in SetAttributeValueInput) (*ProductAttributeValue, error) {
	def, err := s.getAttributeDefinition(ctx, storeID, in.AttributeID)
	if err != nil {
		return nil, err
	}
	var ok bool
	err = s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM products WHERE id=$1 AND store_id=$2)`, productID, storeID).Scan(&ok)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	var text *string
	var num *float64
	var boolean *bool
	switch def.DataType {
	case "text", "select":
		if in.ValueText == nil || strings.TrimSpace(*in.ValueText) == "" {
			return nil, ErrInvalidInput
		}
		t := strings.TrimSpace(*in.ValueText)
		text = &t
	case "number":
		if in.ValueNumber == nil {
			return nil, ErrInvalidInput
		}
		num = in.ValueNumber
	case "boolean":
		if in.ValueBoolean == nil {
			return nil, ErrInvalidInput
		}
		boolean = in.ValueBoolean
	default:
		return nil, ErrInvalidInput
	}
	_, err = s.DB.Exec(ctx, `
		INSERT INTO product_attribute_values (store_id, product_id, attribute_id, value_text, value_number, value_boolean)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (product_id, attribute_id) DO UPDATE SET
			value_text=EXCLUDED.value_text,
			value_number=EXCLUDED.value_number,
			value_boolean=EXCLUDED.value_boolean,
			updated_at=now()
	`, storeID, productID, in.AttributeID, text, num, boolean)
	if err != nil {
		return nil, err
	}
	list, err := s.ListProductAttributes(ctx, storeID, productID)
	if err != nil {
		return nil, err
	}
	for i := range list {
		if list[i].AttributeID == in.AttributeID {
			return &list[i], nil
		}
	}
	return nil, ErrNotFound
}

func (s *Service) DeleteProductAttribute(ctx context.Context, storeID, productID, attributeID uuid.UUID) error {
	tag, err := s.DB.Exec(ctx, `
		DELETE FROM product_attribute_values WHERE store_id=$1 AND product_id=$2 AND attribute_id=$3
	`, storeID, productID, attributeID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
