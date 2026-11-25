# Subscription & Entitlements

Store-based plans, usage limits, feature entitlements, and production testing activation.

**Locked matrix:** 2026-11-25 (user lock). Seeds corrected by migration `000023` (upsert after `000022`).

## Plans (IDR)

| Plan | Monthly | Annually |
| --- | ---: | ---: |
| Basic | 49.000 | 490.000 |
| Standard | 99.000 | 990.000 |
| Premium (recommended) | 199.000 | 1.990.000 |
| Ultimate | 399.000 | 3.990.000 |

- Annual = 2 months free.
- Ownership is per **store** (not account).
- No platform transaction fee.
- Prices, descriptions, limits, and entitlements live in central DB tables and can be updated via admin API.

## Feature matrix (higher inherits lower)

Basic = **full sell + fulfill**. Only **released** product features are gated or advertised on `/pricing`. Reserved keys may exist in DB for higher plans without UI.

### Basic (all included)

Storefront, themes/branding, products + variants/attributes, interactive PDF catalogs + AI extraction, campaigns, preorders, live shopping, orders (incl. manual), purchasing + allocation, receiving, packing **by order**, shipments, vendors, customer portal, manual payment verification, deposit/refund, basic reconciliation, basic WhatsApp templates, basic business rules, basic analytics, multi-store, notifications.

### Standard+

| Key | Product status |
| --- | --- |
| `remove_branding` | Released — hides “Powered by Pikko” on storefront |
| `quick_order` | Released — `/order-entry/*` spreadsheet helpers |
| `bulk_import` | Released — product CSV import |
| `packing_by_customer` | Released — packing `group_by=customer` |
| `consolidated_payments` | Released — payment kind `consolidated` |
| `team` | Released — team invites / RBAC |
| `whatsapp_templates_custom` | **Gap** — seeded Standard+; no custom-template editor UI |

### Premium+

| Key | Product status |
| --- | --- |
| `bulk_ops` | Released — order CSV import |
| `store_credit` | Partial — API + `FEATURE_STORE_CREDIT`; nav still hidden |
| `granular_permissions` | Seeded Premium+; current RBAC ships with `team` (Standard) |
| `custom_domain` | **Gap** — post-MVP |
| `advanced_reconciliation` | **Gap** — no separate advanced recon UX |
| `advanced_rules` | **Gap** — same business-rules product as Basic |
| `advanced_analytics` | **Gap** — same analytics product as Basic |

### Ultimate+

| Key | Product status |
| --- | --- |
| `merchant_audit_log` | **Gap** — `audit_logs` backend only; no merchant UI |
| `priority_support` | **Gap** — ops/support flag only |

Do **not** invent UI for gaps. Do **not** advertise gaps on `/pricing`.

## Usage limits

| Limit | Basic | Standard | Premium | Ultimate |
| --- | ---: | ---: | ---: | ---: |
| Orders / month | 100 | 500 | 2.000 | 10.000 |
| Active products | 200 | 1.000 | 5.000 | 20.000 |
| Active catalogs | 3 | 15 | 50 | 200 |
| Active campaigns | 2 | 10 | 30 | 100 |
| Team members (owner counts) | 1 | 3 | 5 | 15 |
| Storage GB | 1 | 3 | 10 | 30 |
| AI PDF pages / month | 10 | 50 | 200 | 500 |

DB keys: `orders_per_month`, `active_products`, `catalogs`, `campaigns`, `team_members`, `storage_gb`, `ai_pdf_pages_per_month`.

- Reset: Asia/Jakarta calendar month for monthly counters.
- Backend enforces on create paths (`402 plan_limit_exceeded` / `403 feature_not_entitled`).
- Cancelled orders do **not** restore usage.
- Plan change does **not** reset usage.
- Over-limit after downgrade: existing data kept; new creates blocked (no auto-delete).

## Entitlement stack

`global flags × plan entitlements × usage limits × store overrides × user permissions (RBAC)`

Feature keys: `services/api/internal/entitlements/keys.go`.

## Flags (defaults for production testing)

| Flag | Default |
| --- | --- |
| `FEATURE_SUBSCRIPTIONS` | ON |
| `FEATURE_SUBSCRIPTION_PLAN_SELECT` | ON |
| `FEATURE_SUBSCRIPTION_TEST_BYPASS` | ON |
| `FEATURE_SUBSCRIPTION_USAGE_ENFORCE` | ON |
| `FEATURE_SUBSCRIPTION_PLAN_CHANGE` | ON |
| `FEATURE_COMMERCIAL_TRIAL` | **OFF** |
| `FEATURE_REAL_BILLING` | **OFF** |
| `FEATURE_SUBSCRIPTION_AUTO_RENEW` | **OFF** |

## Production testing mode

- Activate any plan without payment (Rp0).
- Simulated payment: `internal_simulation` / `succeeded` / `is_test` / amount 0 / `exclude_from_revenue`.
- Existing stores without a subscription → **Ultimate** testing on `000022`.
- New stores: Create → Choose Plan → Cycle → Activate → Dashboard (steps 1–4).

## Surfaces

- Public: `/pricing` (released features + limits only)
- Merchant: Settings → **Subscription & Billing**
- APIs: `GET /api/v1/billing/plans`, store subscription activate/change/usage, admin plan/flag/override routes

## Migrations

1. `database/migrations/000022_subscriptions.up.sql` — schema + initial seeds
2. `database/migrations/000023_subscription_matrix_lock.up.sql` — **upsert** limits/features to locked matrix (safe if 000022 already applied)
