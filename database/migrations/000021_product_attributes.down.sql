ALTER TABLE product_attribute_values
    DROP CONSTRAINT IF EXISTS product_attribute_values_has_value;

ALTER TABLE product_attribute_values
    DROP COLUMN IF EXISTS value_option_ids;

ALTER TABLE product_attribute_values
    ADD CONSTRAINT product_attribute_values_has_value CHECK (
        value_text IS NOT NULL OR value_number IS NOT NULL OR value_boolean IS NOT NULL
    );

DROP TABLE IF EXISTS attribute_options;

UPDATE attribute_definitions SET data_type = 'text' WHERE data_type = 'short_text';
UPDATE attribute_definitions SET data_type = 'select' WHERE data_type IN ('single_option', 'multiple_option');

ALTER TABLE attribute_definitions
    DROP CONSTRAINT IF EXISTS attribute_definitions_data_type_check;

ALTER TABLE attribute_definitions
    ADD CONSTRAINT attribute_definitions_data_type_check
    CHECK (data_type IN ('text', 'number', 'boolean', 'select'));
