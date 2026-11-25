-- Product Attributes: typed custom fields replace first-class product Category UX.
-- Legacy data_type values are normalized; option rows move to attribute_options.

-- Normalize legacy attribute types
UPDATE attribute_definitions SET data_type = 'short_text' WHERE data_type IN ('text', 'boolean');
UPDATE attribute_definitions SET data_type = 'single_option' WHERE data_type = 'select';

ALTER TABLE attribute_definitions
    DROP CONSTRAINT IF EXISTS attribute_definitions_data_type_check;

ALTER TABLE attribute_definitions
    ADD CONSTRAINT attribute_definitions_data_type_check
    CHECK (data_type IN ('single_option', 'multiple_option', 'short_text', 'number'));

CREATE TABLE IF NOT EXISTS attribute_options (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id        UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    attribute_id    UUID NOT NULL REFERENCES attribute_definitions(id) ON DELETE CASCADE,
    label           TEXT NOT NULL,
    color           TEXT NOT NULL DEFAULT 'slate'
        CHECK (color IN (
            'coral', 'peach', 'amber', 'lime', 'teal',
            'sky', 'indigo', 'violet', 'rose', 'slate'
        )),
    sort_order      INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS attribute_options_attr_idx
    ON attribute_options (attribute_id, sort_order, label);

CREATE INDEX IF NOT EXISTS attribute_options_store_idx
    ON attribute_options (store_id);

-- Expand product values to reference option ids (single or multi)
ALTER TABLE product_attribute_values
    ADD COLUMN IF NOT EXISTS value_option_ids UUID[] NOT NULL DEFAULT '{}';

ALTER TABLE product_attribute_values
    DROP CONSTRAINT IF EXISTS product_attribute_values_has_value;

ALTER TABLE product_attribute_values
    ADD CONSTRAINT product_attribute_values_has_value CHECK (
        value_text IS NOT NULL
        OR value_number IS NOT NULL
        OR value_boolean IS NOT NULL
        OR cardinality(value_option_ids) > 0
    );

-- Clear product category linkage (Category is no longer first-class on products).
-- Category rows remain for historical CSV/import metadata only.
UPDATE products SET category_id = NULL WHERE category_id IS NOT NULL;
