# Database Schema — Current + Target (PRD v3.0)

**Baseline migrations:** `000001`–`000006`  
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

Next free version: **`000007_…`** (Agents 03+).

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
| `store_memberships` | role **CHECK (`owner` only)**, status active/revoked | **Target:** widen roles |
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

### 3.1 Team RBAC (`000007` candidate — Agent 03)

```text
ALTER store_memberships: widen role CHECK to
  ('owner','admin','cs','catalog','purchasing','packing','finance')
+ team_invites (store_id, email, role, token_hash, expires_at, accepted_at)
+ optional permission_overrides (defer if role matrix sufficient)
```

### 3.2 Order item operations & revisions (Agent 05)

```text
order_items: + availability_status, allocated_qty, packed_qty, adjusted_line_total_idr
order_item_events / order_revisions: append-only change log (who/when/why/before/after)
admin_order_drafts (optional): spreadsheet entry staging
```

### 3.3 Vendors & receiving (Agent 06)

```text
vendors (store_id, name, contacts, notes)
product_vendors (product_id, vendor_id, preference)
purchase_orders / purchase_order_lines (optional MVP-lite: purchasing_records + vendor_id)
receiving_records / receiving_lines → link purchasing + inventory adjustments
```

### 3.4 Finance (Agent 07)

```text
financial_events (store_id, customer_id?, order_id?, type, amount_idr, direction, idempotency_key)
refunds (order_id, amount_idr, reason, status, evidence_file_id?)
store_credits (customer_id, balance_idr) + store_credit_ledger (append-only)
reconciliation_snapshots (optional materialized view / table)
```

**Never** mutate historical `payments.amount_idr` after verify; issue compensating events.

### 3.5 Customer portal (Agent 08)

```text
customer_identities / customer_sessions (email magic link scoped to customer+store)
OR reuse users with customer_store_links — choose one model; prefer separate customer auth to avoid conflating merchant users
portal_address_change_requests
```

### 3.6 Packing & shipments (Agent 09)

```text
packing_tasks (order_id, status, assigned_user_id?)
packing_task_items (order_item_id, qty_packed)
shipments (carrier, tracking_number, status, shipped_at)
shipment_items (shipment_id, order_item_id, qty)
```

### 3.7 Live shopping (Agent 12)

```text
live_sessions (store_id, title, opens_at, closes_at, status)
live_session_products
live_quick_adds / tie into orders
```

### 3.8 WhatsApp (Agent 10)

```text
whatsapp_templates (code V01–V08, C01–C18, body, variables)
whatsapp_sends (optional log; deep-link only in MVP — no WhatsApp Business API required)
```

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
