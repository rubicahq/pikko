# Database Schema — Current + Target (PRD v3.0)

**Baseline migrations:** `000001`–`000013`  
**Policy:** **Additive migrations only.** No `DROP TABLE`, no destructive column drops on production data without explicit owner approval. Down migrations exist for local/dev rollback only.

---

## 1. Migration inventory (current)

| Version | File | Purpose |
| --- | --- | --- |
| 000001 | `phase1_auth_stores` | Users, auth tokens, sessions, stores, memberships, onboarding, email_jobs, audit_logs |
| 000002 | `phase2_storefront_catalog` | Files, storefront, products, variants, collections, featured |
| 000003 | `ai_pdf_import` | job_queue, outbox_events, catalog_import_*, catalog_pages/hotspots |
| 000004 | `phase3_orders_payments` | Customers, orders, items, payments, evidence, inventory, counters |
| 000005 | `phase4_preorder_rules` | Preorder batches, purchasing, allocations, business rules |
| 000006 | `phase5_merchant_efficiency` | Notifications, analytics_events, CSV import columns |
| 000007 | `team_rbac` | Widen membership roles; `team_invites` |
| 000008 | `campaign_catalog` | Campaigns + brands/vendors/attrs; link preorder batches |
| 000009 | `order_management` | Order source/campaign, item availability cols, revisions, drafts |
| 000010 | `purchasing_allocation` | Vendor splits on purchasing records, version, receiving foundation |
| 000011 | `payments_finance` | Deposits, payment allocations, financial events/adjustments, refunds, store credit |
| 000012 | `customer_portal` | Portal tokens/sessions, refund prefs, address change requests |
| 000013 | `packing_shipment` | packing_tasks/items + shipments/items (Agent 09) |

Next free version: **`000014_…`** (Agents 10+).

---

## 2. Current schema summary

### 2.1 Identity & tenancy

| Table | Key columns | Notes |
| --- | --- | --- |
| `users` | email (unique normalized), display_name, last_store_id | |
| `authentication_tokens` | token_hash, expires_at, consumed_at | Magic links |
| `sessions` | session_hash, user_id, expires_at, revoked_at | |
| `user_preferences` | locale, timezone | |
| `stores` | slug unique, selling_model, theme, publication_status, checkout_enabled, archived_at | |
| `store_memberships` | role CHECK (`owner`,`admin`,`cs`,`catalog`,`purchasing`,`packing`,`finance`), status active/revoked | Widened in `000007` |
| `team_invites` | store_id, email, role, token_hash, status, expires_at | Invite / accept / cancel / resend |
| `store_slug_aliases` | historical slugs | |
| `onboarding_drafts` | step JSON progress | |
| `auth_rate_limits` | email/IP windows | |
| `email_jobs` | idempotency, attempts, status | |
| `audit_logs` | actor, action, payload | |

### 2.2 Catalog & storefront

`stored_files`, `storefront_configs` (channel draft|published), `storefront_banners`, `categories`, `products`, `product_images`, `product_options`, `product_option_values`, `product_variants`, `variant_option_values`, `collections`, `collection_products`, `storefront_featured_products`, `storefront_featured_collections`.

Money on products/variants: integer IDR. Status: draft/published/archived.

### 2.3 Import & flipbook

`job_queue`, `outbox_events`, `catalog_import_jobs` (+ `source_type`, `csv_file_id`), `catalog_import_items`, `catalog_pages`, `catalog_page_hotspots`.

### 2.4 Commerce

| Table | Notes |
| --- | --- |
| `store_payment_methods` | Bank transfer accounts |
| `store_fulfillment_settings` | Shipping / free / pickup |
| `customers` / `customer_addresses` | Store-scoped |
| `orders` | Multi-dimension statuses; immutable fee totals; optional `preorder_batch_id` + snapshots |
| `order_items` | Immutable name/price snapshots; optional `batch_id` |
| `order_status_history` | dimension order\|payment\|fulfillment |
| `order_tracking_tokens` | SHA-256 only |
| `payments` / `payment_evidence` | Manual transfer + private files |
| `inventory_reservations` / `inventory_adjustments` | Finite stock + batch quotas |
| `public_rate_limits` / `store_order_counters` | |

### 2.5 Preorder & rules

`preorder_batches`, `preorder_batch_products`, `preorder_batch_variants`, `preorder_purchasing_records`, `preorder_item_allocations`, `business_rules`, `business_rule_versions`, `order_rule_applications`.

### 2.6 Merchant efficiency

`notifications`, `notification_preferences`, `analytics_events`.

### 2.7 Conventions

- PKs: UUID `gen_random_uuid()`
- Timestamps: `TIMESTAMPTZ` UTC
- Tenant: `store_id` FK on business tables
- Soft archive preferred over hard delete for stores/products

---

## 3. Target schema (additive — do not implement in Agent 01)

Mark each as **planned additive migration**. Exact DDL owned by implementing agents; this section locks intent.

### 3.1 Team RBAC (`000007` — Agent 03) ✅

```text
ALTER store_memberships: widen role CHECK to
  ('owner','admin','cs','catalog','purchasing','packing','finance')
+ team_invites (store_id, email, role, token_hash, expires_at, accepted_at, status)
Permission overrides deferred (role matrix in Go `internal/rbac` is sufficient)
```

### 3.2 Order item operations & revisions (`000009` — Agent 05) ✅

```text
orders: + source (customer|admin|csv_import), created_by_user_id, campaign_id
order_items: + availability_status, allocated_qty, packed_qty, adjusted_line_total_idr, original_quantity
order_revisions / order_item_events: append-only change log (who/when/why/before/after)
admin_order_drafts: spreadsheet entry staging + validation_errors
```

### 3.2b Campaigns & catalog foundations (`000008` — Agent 04) ✅

```text
brands (store_id, name, slug, …) + products.brand_id
vendors + product_vendors (foundation; purchasing UX Agent 06)
attribute_definitions + product_attribute_values
campaigns (type preorder|live, statuses, live flags, preorder_batch_id)
campaign_products (price/availability/selling_phase overrides)
preorder_batches.campaign_id (backfilled)
```

### 3.3 Vendors & receiving (`000010` — Agent 06) ✅

```text
preorder_purchasing_records: + vendor_id, received_qty, version (multi-vendor + concurrency)
preorder_item_allocations: allocated_qty CHECK widened to >= 0
receiving_records / receiving_lines → purchasing handoff + partial discrepancy notes
```

### 3.4 Finance (`000011` — Agent 07) ✅

```text
orders: + original_total_idr, unavailable_delta_idr, credit_applied_idr, payable_idr, settlement_status
store_deposit_policies (percent_bps | fixed)
payments: + payment_kind, idempotency_key, note, version
payment_allocations (multi-order / consolidated)
financial_events (append-only, store+idempotency unique)
financial_adjustments (unavailable/manual/credit)
refunds (+ evidence_file_id, version concurrency)
stores.store_credit_enabled + store_credits + store_credit_ledger (FEATURE_STORE_CREDIT)
```

**Never** mutate historical `payments.amount_idr` after verify; issue compensating events.

### 3.5 Customer portal (Agent 08) — shipped `000012`

```text
customer_portal_tokens (store+email magic link; SHA-256 only; ≥256-bit raw)
customer_portal_sessions (store+customer; separate from merchant sessions)
customer_refund_preferences (bank_transfer | store_credit + bank details)
customer_addresses.updated_at
portal_address_change_requests (pending unique per order; merchant review)
```

Separate customer auth (not merchant `users`) to avoid conflating roles.

### 3.6 Packing & shipments (Agent 09) — shipped `000013`

```text
packing_tasks (store+order unique; pending|in_progress|packed|cancelled; assigned_user_id?)
packing_task_items (order_item_id, qty_to_pack, qty_packed ≤ qty_to_pack)
shipments (carrier, tracking_number, status, shipped_at, notify_customer)
shipment_items (shipment_id, order_item_id, quantity > 0)
```

Reuses `order_items.packed_qty` and fulfillment statuses; no destructive changes.

### 3.7 Live shopping (Agent 12)

```text
live_sessions (store_id, title, opens_at, closes_at, status)
live_session_products
live_quick_adds / tie into orders
```

### 3.8 WhatsApp (Agent 10)

```text
# Catalog is code-defined in services/api/internal/whatsapp (WA-V01…V08, WA-C01…C18)
whatsapp_template_sends (
  store_id, template_code, audience, phone_e164?, rendered_body, deep_link,
  order_id?, customer_id?, vendor_id?, opened_by?, action open|copy|preview, created_at
)
```

Migration `000014_whatsapp_templates`. Deep-link only in MVP — no WhatsApp Business API.

---

## 4. Migration safety checklist

For every new migration:

1. **Up** is additive (`CREATE TABLE`, `ADD COLUMN` nullable or with default, widen CHECKs carefully)
2. Backfill in same migration only if online-safe
3. **No** `DROP COLUMN` / type-narrowing on live data without approval
4. Indexes concurrent where needed on large tables (orders)
5. Update this doc + `REQUIREMENTS_MATRIX.md`
6. Integration tests against migrated schema

---

## 5. ER mental model (simplified)

```
users ──< store_memberships >── stores
stores ──< products ──< variants
stores ──< customers ──< orders ──< order_items
orders ──< payments ──< payment_evidence
stores ──< preorder_batches ──< purchasing_records
                           └──< item_allocations → order_items
stores ──< business_rules ──< versions → order_rule_applications
```

Target adds vendors/receiving/finance/packing/shipments/portal around the same `stores` hub.
