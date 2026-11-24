# Database Schema (Phase 0–2)

Migrations:

- `000001_phase1_auth_stores.up.sql`
- `000002_phase2_storefront_catalog.up.sql`

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

## Flipbook linkage

Hotspots should reference `products.id` (UUID) within the same `store_id`. Only `status='published'` products are exposed on public APIs. See `docs/CATALOG_LINK_CONTRACT.md`.

## Defaults on store create

Theme `pikko_signature`, draft `storefront_configs` row, currency IDR, timezone Asia/Jakarta, locale id-ID, publication draft, checkout false.
