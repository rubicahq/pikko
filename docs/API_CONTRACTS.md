# API Contracts — Conventions + Inventory (PRD v3.0)

**Base path:** `/api/v1`  
**Error shape:** `{ "error": { "code": string, "message": string, "request_id": string } }`  
**Prior doc:** `API_REFERENCE.md` remains as historical phase notes; this file is the v3 contract source.

---

## 1. Conventions

| Topic | Rule |
| --- | --- |
| Auth | Session cookie (`AUTH_COOKIE_NAME`, default `pikko_session`) or future Bearer if added; cookie primary |
| Tenancy | Store-scoped routes include `{storeID}`; middleware checks membership |
| IDs | UUID strings |
| Money | Integer IDR fields `*_idr` (int64) |
| Time | RFC3339 / timestamptz UTC in JSON; clients display Asia/Jakarta |
| Idempotency | Required on checkout submit + import start (`Idempotency-Key` header or body field) |
| Pagination | `limit` + `offset` (existing); prefer stable defaults ≤100 |
| Public vs private | `/public/…` unauthenticated; evidence/files remain authz-gated |
| Versioning | Single `/v1`; breaking changes require new version or additive fields only |

### 1.1 Status codes (common)

| Code | Meaning |
| --- | --- |
| 200/201 | Success |
| 400 | Validation |
| 401 | Unauthenticated |
| 403 | Forbidden (no membership / wrong tenant) |
| 404 | Not found (also used to avoid enumeration where appropriate) |
| 409 | Conflict (price_changed, duplicate, invalid transition) |
| 429 | Rate limited |
| 503 | Not ready (readyz) |

### 1.2 Target module prefixes (future)

| Prefix | Agent |
| --- | --- |
| `/stores/{id}/team` | 03/10 |
| `/stores/{id}/campaigns` | 04 |
| `/stores/{id}/brands` / `attributes` | 04 |
| `/stores/{id}/vendors` | 04 foundation / 06 |
| `/stores/{id}/receiving` | 06 |
| `/stores/{id}/finance` | 07 |
| `/stores/{id}/packing` | 09 |
| `/stores/{id}/shipments` | 09 |
| `/stores/{id}/live` | 12 |
| `/portal/…` | 08 |

---

## 2. Existing endpoint inventory

Source: `services/api/internal/httpapi/router.go`.

### 2.1 Health

| Method | Path | Auth |
| --- | --- | --- |
| GET | `/livez` | none |
| GET | `/healthz` | none |
| GET | `/readyz` | none |

### 2.2 Auth

| Method | Path | Auth |
| --- | --- | --- |
| POST | `/auth/magic-link/request` | public |
| GET | `/auth/magic-link/peek` | public (no consume) |
| POST | `/auth/magic-link/verify` | public (consume) |
| GET | `/auth/session` | cookie optional |
| POST | `/auth/logout` | cookie |
| POST | `/auth/logout-all` | required |
| GET | `/auth/sessions` | required |
| DELETE | `/auth/sessions/{sessionID}` | required |

### 2.3 Account / meta / stores / onboarding

| Method | Path | Auth |
| --- | --- | --- |
| GET/PATCH | `/account` | required |
| GET | `/meta/categories` | required |
| GET | `/meta/themes` | required |
| GET/POST | `/stores` | required |
| GET | `/stores/slug-available` | required |
| GET | `/stores/suggest-slug` | required |
| GET | `/onboarding/draft` | required |
| POST | `/onboarding/identity` | required |
| POST | `/onboarding/selling-model` | required |
| POST | `/onboarding/complete` | required |
| GET/PATCH | `/stores/{storeID}` | store member (+ permissions on mutate) |
| POST | `/stores/{storeID}/archive` | owner (`stores.archive`) |

### 2.4 Team RBAC (Agent 03)

| Method | Path | Auth / permission |
| --- | --- | --- |
| GET | `/team/invites/peek?token=` | public |
| POST | `/team/invites/accept` | public (sets session cookie) |
| GET | `/stores/{storeID}/team/me` | `stores.read` |
| GET | `/stores/{storeID}/team/members` | `team.read` |
| GET | `/stores/{storeID}/team/invites` | `team.read` |
| POST | `/stores/{storeID}/team/invites` | `team.invite` |
| POST | `/stores/{storeID}/team/invites/{inviteID}/resend` | `team.invite` |
| POST | `/stores/{storeID}/team/invites/{inviteID}/cancel` | `team.manage` |
| POST | `/stores/{storeID}/team/members/{membershipID}/revoke` | `team.manage` |
| PATCH | `/stores/{storeID}/team/members/{membershipID}` | `team.manage` |

Store-scoped routes also enforce granular permissions via `RequirePermission` (catalog, orders, purchasing, etc.).

### 2.5 Storefront / files / catalog

| Method | Path |
| --- | --- |
| GET/PATCH | `/stores/{storeID}/storefront` |
| POST | `/stores/{storeID}/storefront/publish` |
| POST | `/stores/{storeID}/storefront/discard` |
| PUT | `/stores/{storeID}/storefront/featured` |
| GET | `/stores/{storeID}/setup-checklist` |
| POST | `/stores/{storeID}/files` |
| GET | `/stores/{storeID}/files/{fileID}` |
| GET/POST | `/stores/{storeID}/categories` |
| PATCH/DELETE | `/stores/{storeID}/categories/{categoryID}` |
| GET/POST | `/stores/{storeID}/products` |
| GET/PATCH | `/stores/{storeID}/products/{productID}` |
| POST | `/stores/{storeID}/products/{productID}/status` |
| POST | `/stores/{storeID}/products/{productID}/archive` |
| GET/POST | `/stores/{storeID}/collections` |
| GET/POST | `/stores/{storeID}/brands` |
| PATCH | `/stores/{storeID}/brands/{brandID}` |
| GET/POST | `/stores/{storeID}/vendors` |
| PATCH | `/stores/{storeID}/vendors/{vendorID}` |
| GET/POST/DELETE | `/stores/{storeID}/products/{productID}/vendors[/{vendorID}]` |
| GET/POST | `/stores/{storeID}/attributes` |
| PATCH | `/stores/{storeID}/attributes/{attributeID}` |
| GET/PUT/DELETE | `/stores/{storeID}/products/{productID}/attributes[/{attributeID}]` |

### 2.5b Campaigns (Agent 04)

| Method | Path | Auth / permission |
| --- | --- | --- |
| GET/POST | `/stores/{storeID}/campaigns` | `campaign.read` / `campaign.write` |
| GET/PATCH | `/stores/{storeID}/campaigns/{campaignID}` | `campaign.read` / `campaign.write` |
| POST/DELETE | `/stores/{storeID}/campaigns/{campaignID}/products[/{productID}]` | `campaign.write` |

Preorder routes remain; gated by `campaign.*` and sync into `campaigns`.

### 2.6 Catalog import + flipbook

| Method | Path |
| --- | --- |
| POST | `/stores/{storeID}/catalog-imports` |
| GET | `/stores/{storeID}/catalog-imports` |
| GET | `/stores/{storeID}/catalog-imports/csv-template` |
| GET | `/stores/{storeID}/catalog-imports/{jobID}` |
| GET | `/stores/{storeID}/catalog-imports/{jobID}/items` |
| PATCH/DELETE | `…/items/{itemID}` |
| POST | `…/confirm` / `…/retry` |
| GET | `/stores/{storeID}/catalog-pages` |
| PUT | `/stores/{storeID}/catalog-pages/order` |
| POST | `/stores/{storeID}/catalog-pages/status` |
| POST | `/stores/{storeID}/catalog-pages/{pageID}/hotspots` |
| PATCH/DELETE | `/stores/{storeID}/catalog-pages/hotspots/{hotspotID}` |

### 2.7 Commerce settings, orders, customers

| Method | Path |
| --- | --- |
| GET/POST | `/stores/{storeID}/payment-methods` |
| PATCH/DELETE | `/stores/{storeID}/payment-methods/{methodID}` |
| GET/PATCH | `/stores/{storeID}/fulfillment` |
| GET | `/stores/{storeID}/orders` |
| POST | `/stores/{storeID}/orders` (admin create; `orders.write`) |
| GET/PATCH | `/stores/{storeID}/orders/{orderID}` |
| GET | `/stores/{storeID}/orders/{orderID}/revisions` |
| GET | `/stores/{storeID}/orders/{orderID}/item-events` |
| PATCH | `/stores/{storeID}/orders/{orderID}/items/{itemID}` (qty adjust → revision) |
| POST | `/stores/{storeID}/orders/{orderID}/payments/{paymentID}/verify` |
| POST | `/stores/{storeID}/orders/{orderID}/payments/{paymentID}/reject` |
| GET | `/stores/{storeID}/orders/evidence/{evidenceID}/file` |
| POST | `/stores/{storeID}/order-entry/validate` |
| GET | `/stores/{storeID}/order-entry/catalog-suggest` |
| GET/POST | `/stores/{storeID}/order-entry/drafts` |
| GET/PUT/DELETE | `/stores/{storeID}/order-entry/drafts/{draftID}` |
| POST | `/stores/{storeID}/order-entry/drafts/{draftID}/validate` |
| GET | `/stores/{storeID}/order-entry/csv-template` |
| POST | `/stores/{storeID}/order-entry/csv-preview` |
| POST | `/stores/{storeID}/order-entry/csv-import` |
| GET | `/stores/{storeID}/customers` |
| GET/PATCH | `/stores/{storeID}/customers/{customerID}` |

### 2.8 Notifications + analytics

| Method | Path |
| --- | --- |
| GET | `/stores/{storeID}/notifications` |
| GET | `/stores/{storeID}/notifications/unread-count` |
| POST | `/stores/{storeID}/notifications/read-all` |
| POST | `/stores/{storeID}/notifications/{notificationID}/read` |
| GET/PATCH | `/stores/{storeID}/notification-preferences` |
| GET | `/stores/{storeID}/analytics` |

### 2.9 Preorders + purchasing + rules

| Method | Path |
| --- | --- |
| GET/POST | `/stores/{storeID}/preorders` |
| GET/PATCH | `/stores/{storeID}/preorders/{batchID}` |
| POST | `/stores/{storeID}/preorders/{batchID}/products` |
| DELETE | `/stores/{storeID}/preorders/{batchID}/products/{productID}` |
| GET/POST | `/stores/{storeID}/preorders/{batchID}/products/{productID}/variants` |
| DELETE | `…/variants/{variantID}` |
| GET/PUT | `/stores/{storeID}/preorders/{batchID}/purchasing` |
| POST | `/stores/{storeID}/preorders/{batchID}/allocations` |
| GET | `/stores/{storeID}/preorders/{batchID}/orders` |
| GET | `/stores/{storeID}/purchasing` |
| GET | `/stores/{storeID}/purchasing/eligibility` |
| POST | `/stores/{storeID}/purchasing/unavailable-impact` |
| POST | `/stores/{storeID}/orders/{orderID}/items/{itemID}/unavailable` |
| GET/POST | `/stores/{storeID}/receiving` |
| GET | `/stores/{storeID}/receiving/{receivingID}` |
| PATCH | `/stores/{storeID}/receiving/{receivingID}/lines/{lineID}` |
| GET/POST | `/stores/{storeID}/business-rules` |
| PATCH/DELETE | `/stores/{storeID}/business-rules/{ruleID}` |
| POST | `/stores/{storeID}/business-rules/preview` |

### 2.10 Public storefront / checkout / tracking

| Method | Path |
| --- | --- |
| GET | `/public/stores/{slug}` |
| GET | `/public/stores/{slug}/products` |
| GET | `/public/stores/{slug}/products/{productSlug}` |
| GET | `/public/stores/{slug}/categories` |
| GET | `/public/stores/{slug}/catalog` |
| GET | `/public/stores/{slug}/catalog/products/{productID}` |
| GET | `/public/files/{storeID}/{fileID}` |
| GET | `/public/stores/{slug}/preorders` |
| GET | `/public/stores/{slug}/preorders/{batchSlug}` |
| GET | `/public/stores/{slug}/products/{productSlug}/preorders` |
| POST | `/public/stores/{slug}/checkout/preview` |
| POST | `/public/stores/{slug}/checkout/submit` |
| GET | `/public/tracking/{token}` |
| POST | `/public/tracking/{token}/evidence` |
| GET | `/public/tracking/{token}/evidence/{evidenceID}/file` |

---

## 3. Gaps vs v3 (contract-level)

| Missing capability | Suggested contract owner |
| --- | --- |
| Admin create/edit order | **Shipped Agent 05** — `POST /orders` + `/order-entry/*` |
| Order item patch / revision list | **Shipped Agent 05** — revisions + item-events + qty PATCH |
| Refunds + store credit | Agent 07 `/finance/*` |
| Customer portal auth + orders | Agent 08 `/portal/*` |
| Packing + shipments | Agent 09 |
| Team invite/RBAC | **Shipped Agent 03** — see §2.4 Team |
| Vendors / receiving | **Shipped Agent 06** — vendors RBAC + `/purchasing/*` + `/receiving/*` |
| Live sessions | Agent 12 |
| WhatsApp template resolve | Agent 10 |

Additive fields preferred over breaking renames. When extending order JSON, keep existing keys stable.
