# Database Schema (Phase 0–3)

Migrations:

- `000001_phase1_auth_stores.up.sql` — auth / stores
- `000002_phase2_storefront_catalog.up.sql` — products / storefront
- `000003_ai_pdf_import.up.sql` — import jobs, job queue, catalog pages/hotspots
- `000004_phase3_orders_payments.up.sql` — checkout, orders, payments, tracking

## Phase 1 tables

`users`, `authentication_tokens`, `sessions`, `user_preferences`, `stores`, `store_memberships`, `store_slug_aliases`, `onboarding_drafts`, `auth_rate_limits`, `email_jobs`, `audit_logs`

## Phase 2 tables

| Table | Purpose |
| --- | --- |
| `stored_files` | Store-scoped uploads (`stores/{storeId}/{purpose}/…`) |
| `storefront_configs` | Draft vs published branding/theme/colors/assets |
| `storefront_banners` | Banner rows per channel (API scaffolding) |
| `storefront_featured_products` | Up to 8 featured (draft/published) |
| `storefront_featured_collections` | Up to 4 featured |
| `categories` | Store-scoped product categories |
| `products` | Catalog masters; `status` draft/published/archived; optional `sku`; integer IDR prices |
| `product_images` | Ordered images → `stored_files` |
| `product_options` / `product_option_values` | Variant option definitions |
| `product_variants` / `variant_option_values` | Combinations + price overrides |
| `collections` / `collection_products` | Merchandising collections |

## AI import & flipbook (`000003`)

| Table | Purpose |
| --- | --- |
| `job_queue` | PostgreSQL job queue with idempotency + SKIP LOCKED claims |
| `outbox_events` | Transactional outbox |
| `catalog_import_jobs` / `catalog_import_items` | PDF import |
| `catalog_pages` / `catalog_page_hotspots` | Flipbook |

## Phase 3 commerce (`000004`)

| Table | Purpose |
| --- | --- |
| `store_payment_methods` | Per-store bank transfer accounts |
| `store_fulfillment_settings` | Shipping / free shipping / pickup |
| `customers` / `customer_addresses` | Store-scoped customers from orders |
| `orders` / `order_items` | Orders with immutable price snapshots |
| `order_status_history` | Audit of order/payment/fulfillment transitions |
| `order_tracking_tokens` | SHA-256 hashes only (≥256-bit raw tokens) |
| `payments` / `payment_evidence` | Manual transfer + private proof files |
| `inventory_reservations` / `inventory_adjustments` | Finite stock reservations |
| `public_rate_limits` | Checkout/tracking/evidence rate limits |
| `store_order_counters` | Sequential order numbers per store |

`stored_files.purpose` also allows `payment_evidence`.

Timestamps are `TIMESTAMPTZ` (UTC). Display timezone default Asia/Jakarta.

## Defaults on store create

Theme `pikko_signature`, draft `storefront_configs` row, currency IDR, timezone Asia/Jakarta, locale id-ID, publication draft, checkout false until payment + fulfillment configured.
