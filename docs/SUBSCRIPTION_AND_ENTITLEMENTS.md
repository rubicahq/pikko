# Subscription & Entitlements

Store-based plans, usage limits, feature entitlements, and production testing activation.

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
- Prices, descriptions, limits, and entitlements live in central DB tables (`subscription_plans`, `subscription_plan_features`, `subscription_plan_limits`) and can be updated via admin API.

## Feature matrix (higher inherits lower)

Only **released** product features are entitled. Do not advertise unreleased items (Ask Connie, real subscription gateway billing, etc.).

| Feature | Basic | Standard | Premium | Ultimate |
| --- | :---: | :---: | :---: | :---: |
| Storefront, products, catalogs, flipbook | ✓ | ✓ | ✓ | ✓ |
| Orders, checkout, payments, finance, refunds | ✓ | ✓ | ✓ | ✓ |
| Packing, shipments, notifications, customer portal | ✓ | ✓ | ✓ | ✓ |
| Themes, branding, multi-store | ✓ | ✓ | ✓ | ✓ |
| Campaigns, preorders, business rules | | ✓ | ✓ | ✓ |
| AI PDF import, analytics, team | | ✓ | ✓ | ✓ |
| WhatsApp templates, purchasing, receiving, vendors | | | ✓ | ✓ |
| Live shopping ops, product attributes | | | ✓ | ✓ |

Basic = full sell + fulfill journey.

## Usage limits

| Limit | Basic | Standard | Premium | Ultimate |
| --- | ---: | ---: | ---: | ---: |
| Orders / month | 100 | 500 | 2.000 | Unlimited |
| Active products (published) | 50 | 250 | 1.000 | Unlimited |
| Catalogs | 2 | 10 | 50 | Unlimited |
| Campaigns | 0 | 5 | 25 | Unlimited |
| Team members (owner counts) | 1 | 3 | 10 | 50 |
| Storage GB | 1 | 5 | 25 | 100 |
| AI PDF pages / month | 0 | 100 | 500 | 5.000 |

- Reset: Asia/Jakarta calendar month for monthly counters.
- Backend enforces on create paths (402 `plan_limit_exceeded` / 403 `feature_not_entitled`).
- Cancelled orders do **not** restore usage.
- Plan change does **not** reset usage.
- Over-limit after downgrade: existing data kept; new creates blocked until under limit (no auto-delete).

## Entitlement stack

`global flags × plan entitlements × usage limits × store overrides × user permissions (RBAC)`

Feature keys are centralized in `services/api/internal/entitlements/keys.go`.

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

Env kill switches can further disable values stored in `platform_subscription_config`.

## Production testing mode

- Activate any plan without payment (Rp0).
- Simulated payment row: `payment_kind=internal_simulation`, `status=succeeded`, `is_test=true`, `amount=0`, `exclude_from_revenue=true`.
- Existing stores without a subscription are migrated to **Ultimate** testing on migration `000022`.
- New stores: Create → Choose Plan → Cycle → Activate → Dashboard (onboarding steps 1–4). Incomplete drafts are resumable.

## Commercial trial (prepared, disabled)

30-day trial for all plans, one trial per store — schema/flags ready; keep `FEATURE_COMMERCIAL_TRIAL=false` until approved.

## Surfaces

- Public: `/pricing`
- Merchant: Settings → **Subscription & Billing** (`/stores/{id}/settings?tab=subscription`)
- APIs:
  - `GET /api/v1/billing/plans`
  - `GET /api/v1/stores/{id}/subscription`
  - `POST /api/v1/stores/{id}/subscription/activate`
  - `POST /api/v1/stores/{id}/subscription/change`
  - `GET /api/v1/stores/{id}/subscription/usage`
  - Admin (token `SUBSCRIPTION_ADMIN_TOKEN` / `X-Admin-Token`):
    - `GET/PATCH /api/v1/admin/subscription/plans…`
    - `PUT /api/v1/admin/subscription/flags`
    - `POST /api/v1/admin/stores/{id}/subscription/overrides`

## Ops rules

- Downgrade warns when usage exceeds target limits; confirm with `force=true`.
- Expired / canceled: fulfill, reconcile, and customer portal remain available; new creates blocked.
- Subscription UI is merchant-only — never shown on storefront to customers.
- No permanent free plan.

## Migration

`database/migrations/000022_subscriptions.up.sql`
