package catalog

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Fixed palette — exactly 10 colors for option chips.
var AttributeOptionColors = []string{
	"coral", "peach", "amber", "lime", "teal",
	"sky", "indigo", "violet", "rose", "slate",
}

var attributeOptionColorSet = func() map[string]struct{} {
	m := make(map[string]struct{}, len(AttributeOptionColors))
	for _, c := range AttributeOptionColors {
		m[c] = struct{}{}
	}
	return m
}()

const (
	AttrTypeSingleOption   = "single_option"
	AttrTypeMultipleOption = "multiple_option"
	AttrTypeShortText      = "short_text"
	AttrTypeNumber         = "number"
)

type AttributeOption struct {
	ID           uuid.UUID `json:"id"`
	AttributeID  uuid.UUID `json:"attribute_id"`
	Label        string    `json:"label"`
	Color        string    `json:"color"`
	SortOrder    int       `json:"sort_order"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type AttributeOptionInput struct {
	ID        *uuid.UUID `json:"id,omitempty"`
	Label     string     `json:"label"`
	Color     string     `json:"color"`
	SortOrder *int       `json:"sort_order,omitempty"`
}

type AttributeDefinition struct {
	ID           uuid.UUID         `json:"id"`
	StoreID      uuid.UUID         `json:"store_id"`
	Key          string            `json:"key"`
	Name         string            `json:"name"`
	Label        string            `json:"label"` // alias of name (compat)
	Type         string            `json:"type"`
	DataType     string            `json:"data_type"` // alias of type (compat)
	Options      []AttributeOption `json:"options"`
	AppliesTo    string            `json:"applies_to"`
	IsRequired   bool              `json:"is_required"`
	DisplayOrder int               `json:"display_order"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

type AttributeDefinitionInput struct {
	Key          string                 `json:"key"`
	Name         string                 `json:"name"`
	Label        string                 `json:"label"` // alias of name
	Type         string                 `json:"type"`
	DataType     string                 `json:"data_type"` // alias of type
	Options      []AttributeOptionInput `json:"options"`
	AppliesTo    string                 `json:"applies_to"`
	IsRequired   *bool                  `json:"is_required"`
	DisplayOrder *int                   `json:"display_order"`
}

type ProductAttributeValue struct {
	ID             uuid.UUID   `json:"id"`
	ProductID      uuid.UUID   `json:"product_id"`
	AttributeID    uuid.UUID   `json:"attribute_id"`
	AttributeKey   string      `json:"attribute_key,omitempty"`
	AttributeName  string      `json:"attribute_name,omitempty"`
	AttributeLabel string      `json:"attribute_label,omitempty"`
	Type           string      `json:"type,omitempty"`
	DataType       string      `json:"data_type,omitempty"`
	ValueText      *string     `json:"value_text,omitempty"`
	ValueNumber    *float64    `json:"value_number,omitempty"`
	OptionIDs      []uuid.UUID `json:"option_ids,omitempty"`
}

type SetAttributeValueInput struct {
	AttributeID uuid.UUID   `json:"attribute_id"`
	ValueText   *string     `json:"value_text"`
	ValueNumber *float64    `json:"value_number"`
	OptionIDs   []uuid.UUID `json:"option_ids"`
}

func normalizeAttributeType(raw string) (string, bool) {
	t := strings.TrimSpace(strings.ToLower(raw))
	switch t {
	case "text", "short_text":
		return AttrTypeShortText, true
	case "number":
		return AttrTypeNumber, true
	case "select", "single_option":
		return AttrTypeSingleOption, true
	case "multiple_option", "multi_option", "multiselect":
		return AttrTypeMultipleOption, true
	case "boolean":
		// legacy → short_text (no boolean in product attributes UX)
		return AttrTypeShortText, true
	default:
		return "", false
	}
}

func isOptionType(t string) bool {
	return t == AttrTypeSingleOption || t == AttrTypeMultipleOption
}

func normalizeOptionColor(raw string) string {
	c := strings.TrimSpace(strings.ToLower(raw))
	if _, ok := attributeOptionColorSet[c]; ok {
		return c
	}
	return "slate"
}

func attributeKeyFromName(name, explicit string) string {
	if k := strings.TrimSpace(strings.ToLower(explicit)); k != "" {
		return sanitizeAttributeKey(k)
	}
	return sanitizeAttributeKey(name)
}

func sanitizeAttributeKey(raw string) string {
	var b strings.Builder
	prevUnderscore := false
	for _, r := range strings.ToLower(strings.TrimSpace(raw)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevUnderscore = false
		case r == ' ' || r == '-' || r == '_' || r == '.':
			if b.Len() > 0 && !prevUnderscore {
				b.WriteByte('_')
				prevUnderscore = true
			}
		}
	}
	s := strings.Trim(b.String(), "_")
	if len(s) > 64 {
		s = strings.Trim(s[:64], "_")
	}
	if s == "" || s[0] < 'a' || s[0] > 'z' {
		s = "attr_" + s
		if len(s) > 64 {
			s = s[:64]
		}
	}
	if !validAttributeKey(s) {
		return "attr_field"
	}
	return s
}

func (s *Service) ListAttributeDefinitions(ctx context.Context, storeID uuid.UUID) ([]AttributeDefinition, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT id, store_id, key, label, data_type, applies_to, is_required, display_order, created_at, updated_at
		FROM attribute_definitions WHERE store_id=$1
		ORDER BY display_order, label
	`, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AttributeDefinition{}
	ids := []uuid.UUID{}
	for rows.Next() {
		var a AttributeDefinition
		if err := rows.Scan(
			&a.ID, &a.StoreID, &a.Key, &a.Label, &a.DataType,
			&a.AppliesTo, &a.IsRequired, &a.DisplayOrder, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, err
		}
		a.Name = a.Label
		a.Type = a.DataType
		a.Options = []AttributeOption{}
		out = append(out, a)
		ids = append(ids, a.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return out, nil
	}
	optsByAttr, err := s.listOptionsForAttributes(ctx, storeID, ids)
	if err != nil {
		return nil, err
	}
	for i := range out {
		if opts, ok := optsByAttr[out[i].ID]; ok {
			out[i].Options = opts
		}
	}
	return out, nil
}

func (s *Service) listOptionsForAttributes(ctx context.Context, storeID uuid.UUID, attrIDs []uuid.UUID) (map[uuid.UUID][]AttributeOption, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT id, attribute_id, label, color, sort_order, created_at, updated_at
		FROM attribute_options
		WHERE store_id=$1 AND attribute_id = ANY($2)
		ORDER BY sort_order, label
	`, storeID, attrIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[uuid.UUID][]AttributeOption{}
	for rows.Next() {
		var o AttributeOption
		if err := rows.Scan(&o.ID, &o.AttributeID, &o.Label, &o.Color, &o.SortOrder, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		out[o.AttributeID] = append(out[o.AttributeID], o)
	}
	return out, rows.Err()
}

func (s *Service) CreateAttributeDefinition(ctx context.Context, storeID uuid.UUID, in AttributeDefinitionInput) (*AttributeDefinition, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = strings.TrimSpace(in.Label)
	}
	if name == "" {
		return nil, ErrInvalidInput
	}
	key := attributeKeyFromName(name, in.Key)
	if !validAttributeKey(key) {
		return nil, ErrInvalidInput
	}
	rawType := in.Type
	if rawType == "" {
		rawType = in.DataType
	}
	dt, ok := normalizeAttributeType(rawType)
	if !ok {
		return nil, ErrInvalidInput
	}
	if isOptionType(dt) && len(in.Options) == 0 {
		return nil, ErrInvalidInput
	}
	applies := in.AppliesTo
	if applies == "" {
		applies = "product"
	}
	if applies != "product" && applies != "variant" {
		return nil, ErrInvalidInput
	}
	req := false
	if in.IsRequired != nil {
		req = *in.IsRequired
	}
	order := 0
	if in.DisplayOrder != nil {
		order = *in.DisplayOrder
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var a AttributeDefinition
	err = tx.QueryRow(ctx, `
		INSERT INTO attribute_definitions (store_id, key, label, data_type, options, applies_to, is_required, display_order)
		VALUES ($1,$2,$3,$4,'[]'::jsonb,$5,$6,$7)
		RETURNING id, store_id, key, label, data_type, applies_to, is_required, display_order, created_at, updated_at
	`, storeID, key, name, dt, applies, req, order).Scan(
		&a.ID, &a.StoreID, &a.Key, &a.Label, &a.DataType, &a.AppliesTo, &a.IsRequired, &a.DisplayOrder, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "attribute_definitions_store_key_unique") ||
			strings.Contains(err.Error(), "attribute_definitions_key_format") {
			return nil, ErrInvalidInput
		}
		return nil, err
	}
	a.Name = a.Label
	a.Type = a.DataType
	a.Options = []AttributeOption{}

	if isOptionType(dt) {
		opts, err := insertAttributeOptions(ctx, tx, storeID, a.ID, in.Options)
		if err != nil {
			return nil, err
		}
		a.Options = opts
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &a, nil
}

func validAttributeKey(key string) bool {
	if len(key) < 1 || len(key) > 64 {
		return false
	}
	if key[0] < 'a' || key[0] > 'z' {
		return false
	}
	for i := 1; i < len(key); i++ {
		c := key[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' {
			continue
		}
		return false
	}
	return true
}

func insertAttributeOptions(ctx context.Context, tx pgx.Tx, storeID, attrID uuid.UUID, in []AttributeOptionInput) ([]AttributeOption, error) {
	return syncAttributeOptions(ctx, tx, storeID, attrID, in)
}

func syncAttributeOptions(ctx context.Context, tx pgx.Tx, storeID, attrID uuid.UUID, in []AttributeOptionInput) ([]AttributeOption, error) {
	keep := make([]uuid.UUID, 0, len(in))
	out := make([]AttributeOption, 0, len(in))
	for i, opt := range in {
		label := strings.TrimSpace(opt.Label)
		if label == "" {
			return nil, ErrInvalidInput
		}
		color := normalizeOptionColor(opt.Color)
		order := i
		if opt.SortOrder != nil {
			order = *opt.SortOrder
		}
		var o AttributeOption
		if opt.ID != nil {
			err := tx.QueryRow(ctx, `
				UPDATE attribute_options
				SET label=$4, color=$5, sort_order=$6, updated_at=now()
				WHERE id=$1 AND attribute_id=$2 AND store_id=$3
				RETURNING id, attribute_id, label, color, sort_order, created_at, updated_at
			`, *opt.ID, attrID, storeID, label, color, order).Scan(
				&o.ID, &o.AttributeID, &o.Label, &o.Color, &o.SortOrder, &o.CreatedAt, &o.UpdatedAt,
			)
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrInvalidInput
			}
			if err != nil {
				return nil, err
			}
		} else {
			err := tx.QueryRow(ctx, `
				INSERT INTO attribute_options (store_id, attribute_id, label, color, sort_order)
				VALUES ($1,$2,$3,$4,$5)
				RETURNING id, attribute_id, label, color, sort_order, created_at, updated_at
			`, storeID, attrID, label, color, order).Scan(
				&o.ID, &o.AttributeID, &o.Label, &o.Color, &o.SortOrder, &o.CreatedAt, &o.UpdatedAt,
			)
			if err != nil {
				return nil, err
			}
		}
		keep = append(keep, o.ID)
		out = append(out, o)
	}
	if len(keep) == 0 {
		return nil, ErrInvalidInput
	}
	_, err := tx.Exec(ctx, `
		DELETE FROM attribute_options
		WHERE attribute_id=$1 AND store_id=$2 AND NOT (id = ANY($3))
	`, attrID, storeID, keep)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Service) UpdateAttributeDefinition(ctx context.Context, storeID, attrID uuid.UUID, in AttributeDefinitionInput) (*AttributeDefinition, error) {
	cur, err := s.getAttributeDefinition(ctx, storeID, attrID)
	if err != nil {
		return nil, err
	}
	name := cur.Label
	if strings.TrimSpace(in.Name) != "" {
		name = strings.TrimSpace(in.Name)
	} else if strings.TrimSpace(in.Label) != "" {
		name = strings.TrimSpace(in.Label)
	}
	req := cur.IsRequired
	if in.IsRequired != nil {
		req = *in.IsRequired
	}
	order := cur.DisplayOrder
	if in.DisplayOrder != nil {
		order = *in.DisplayOrder
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var a AttributeDefinition
	err = tx.QueryRow(ctx, `
		UPDATE attribute_definitions SET label=$3, is_required=$4, display_order=$5, updated_at=now()
		WHERE id=$1 AND store_id=$2
		RETURNING id, store_id, key, label, data_type, applies_to, is_required, display_order, created_at, updated_at
	`, attrID, storeID, name, req, order).Scan(
		&a.ID, &a.StoreID, &a.Key, &a.Label, &a.DataType, &a.AppliesTo, &a.IsRequired, &a.DisplayOrder, &a.CreatedAt, &a.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	a.Name = a.Label
	a.Type = a.DataType

	if isOptionType(a.DataType) && in.Options != nil {
		if len(in.Options) == 0 {
			return nil, ErrInvalidInput
		}
		opts, err := syncAttributeOptions(ctx, tx, storeID, attrID, in.Options)
		if err != nil {
			return nil, err
		}
		a.Options = opts
	} else {
		optsByAttr, err := s.listOptionsForAttributes(ctx, storeID, []uuid.UUID{attrID})
		if err != nil {
			return nil, err
		}
		a.Options = optsByAttr[attrID]
		if a.Options == nil {
			a.Options = []AttributeOption{}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *Service) DeleteAttributeDefinition(ctx context.Context, storeID, attrID uuid.UUID) error {
	tag, err := s.DB.Exec(ctx, `DELETE FROM attribute_definitions WHERE id=$1 AND store_id=$2`, attrID, storeID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) getAttributeDefinition(ctx context.Context, storeID, attrID uuid.UUID) (*AttributeDefinition, error) {
	var a AttributeDefinition
	err := s.DB.QueryRow(ctx, `
		SELECT id, store_id, key, label, data_type, applies_to, is_required, display_order, created_at, updated_at
		FROM attribute_definitions WHERE id=$1 AND store_id=$2
	`, attrID, storeID).Scan(
		&a.ID, &a.StoreID, &a.Key, &a.Label, &a.DataType, &a.AppliesTo, &a.IsRequired, &a.DisplayOrder, &a.CreatedAt, &a.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	a.Name = a.Label
	a.Type = a.DataType
	optsByAttr, err := s.listOptionsForAttributes(ctx, storeID, []uuid.UUID{attrID})
	if err != nil {
		return nil, err
	}
	a.Options = optsByAttr[attrID]
	if a.Options == nil {
		a.Options = []AttributeOption{}
	}
	return &a, nil
}

func (s *Service) ListProductAttributes(ctx context.Context, storeID, productID uuid.UUID) ([]ProductAttributeValue, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT pav.id, pav.product_id, pav.attribute_id, ad.key, ad.label, ad.data_type,
		       pav.value_text, pav.value_number, COALESCE(pav.value_option_ids, '{}')
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
		if err := rows.Scan(
			&v.ID, &v.ProductID, &v.AttributeID, &v.AttributeKey, &v.AttributeLabel, &v.DataType,
			&v.ValueText, &v.ValueNumber, &v.OptionIDs,
		); err != nil {
			return nil, err
		}
		v.AttributeName = v.AttributeLabel
		v.Type = v.DataType
		if v.OptionIDs == nil {
			v.OptionIDs = []uuid.UUID{}
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
	optionIDs := []uuid.UUID{}

	switch def.DataType {
	case AttrTypeShortText:
		if in.ValueText == nil || strings.TrimSpace(*in.ValueText) == "" {
			return nil, ErrInvalidInput
		}
		t := strings.TrimSpace(*in.ValueText)
		text = &t
	case AttrTypeNumber:
		if in.ValueNumber == nil {
			return nil, ErrInvalidInput
		}
		num = in.ValueNumber
	case AttrTypeSingleOption:
		if len(in.OptionIDs) != 1 {
			return nil, ErrInvalidInput
		}
		if err := s.validateOptionIDs(ctx, storeID, def.ID, in.OptionIDs); err != nil {
			return nil, err
		}
		optionIDs = in.OptionIDs
	case AttrTypeMultipleOption:
		if len(in.OptionIDs) == 0 {
			return nil, ErrInvalidInput
		}
		if err := s.validateOptionIDs(ctx, storeID, def.ID, in.OptionIDs); err != nil {
			return nil, err
		}
		optionIDs = in.OptionIDs
	default:
		return nil, ErrInvalidInput
	}

	_, err = s.DB.Exec(ctx, `
		INSERT INTO product_attribute_values (store_id, product_id, attribute_id, value_text, value_number, value_boolean, value_option_ids)
		VALUES ($1,$2,$3,$4,$5,NULL,$6)
		ON CONFLICT (product_id, attribute_id) DO UPDATE SET
			value_text=EXCLUDED.value_text,
			value_number=EXCLUDED.value_number,
			value_boolean=NULL,
			value_option_ids=EXCLUDED.value_option_ids,
			updated_at=now()
	`, storeID, productID, in.AttributeID, text, num, optionIDs)
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

func (s *Service) validateOptionIDs(ctx context.Context, storeID, attrID uuid.UUID, ids []uuid.UUID) error {
	seen := map[uuid.UUID]struct{}{}
	for _, id := range ids {
		if _, dup := seen[id]; dup {
			return ErrInvalidInput
		}
		seen[id] = struct{}{}
		var ok bool
		err := s.DB.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM attribute_options
				WHERE id=$1 AND attribute_id=$2 AND store_id=$3
			)
		`, id, attrID, storeID).Scan(&ok)
		if err != nil {
			return err
		}
		if !ok {
			return ErrInvalidInput
		}
	}
	return nil
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

// SetProductAttributesBulk replaces/sets multiple attribute values on a product.
// Empty values for an attribute delete any existing value.
func (s *Service) SetProductAttributesBulk(ctx context.Context, storeID, productID uuid.UUID, values []SetAttributeValueInput) error {
	var ok bool
	err := s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM products WHERE id=$1 AND store_id=$2)`, productID, storeID).Scan(&ok)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	for _, in := range values {
		empty := (in.ValueText == nil || strings.TrimSpace(*in.ValueText) == "") &&
			in.ValueNumber == nil &&
			len(in.OptionIDs) == 0
		if empty {
			_ = s.DeleteProductAttribute(ctx, storeID, productID, in.AttributeID)
			continue
		}
		if _, err := s.SetProductAttribute(ctx, storeID, productID, in); err != nil {
			return err
		}
	}
	return nil
}
