-- Agent 10: WhatsApp template send log (deep-link MVP; no Meta Business API).
-- Template catalog is code-defined (WA-V01..V08, WA-C01..C18). Additive only.

CREATE TABLE IF NOT EXISTS whatsapp_template_sends (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id        UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    template_code   TEXT NOT NULL,
    audience        TEXT NOT NULL CHECK (audience IN ('vendor', 'customer')),
    phone_e164      TEXT,
    rendered_body   TEXT NOT NULL,
    deep_link       TEXT NOT NULL,
    order_id        UUID REFERENCES orders(id) ON DELETE SET NULL,
    customer_id     UUID REFERENCES customers(id) ON DELETE SET NULL,
    vendor_id       UUID REFERENCES vendors(id) ON DELETE SET NULL,
    opened_by       UUID REFERENCES users(id) ON DELETE SET NULL,
    action          TEXT NOT NULL DEFAULT 'open'
        CHECK (action IN ('open', 'copy', 'preview')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS whatsapp_template_sends_store_created_idx
    ON whatsapp_template_sends (store_id, created_at DESC);
CREATE INDEX IF NOT EXISTS whatsapp_template_sends_store_code_idx
    ON whatsapp_template_sends (store_id, template_code, created_at DESC);
CREATE INDEX IF NOT EXISTS whatsapp_template_sends_order_idx
    ON whatsapp_template_sends (order_id)
    WHERE order_id IS NOT NULL;
