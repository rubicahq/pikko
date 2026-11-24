# API Reference (Phase 0–6)

Base path: `/api/v1`. Errors: `{ "error": { "code", "message", "request_id" } }`.

## Auth / account / stores

Unchanged from Phase 1: magic-link request/peek/verify, session, logout, account, stores CRUD/archive, onboarding.

## Meta

- `GET /meta/themes` (auth) — 24 curated themes
- `GET /meta/categories` (auth) — business categories for onboarding

## Store-scoped (auth + membership)

- `GET/PATCH /stores/{storeId}/storefront?channel=draft|published`
- `POST /stores/{storeId}/storefront/publish`
- `POST /stores/{storeId}/storefront/discard`
- `PUT /stores/{storeId}/storefront/featured`
- `GET /stores/{storeId}/setup-checklist`
- `POST /stores/{storeId}/files` multipart `purpose` + `file`
- `GET /stores/{storeId}/files/{fileId}`
- `GET/POST /stores/{storeId}/categories`
- `PATCH/DELETE /stores/{storeId}/categories/{categoryId}`
- `GET/POST /stores/{storeId}/products`
- `GET/PATCH /stores/{storeId}/products/{productId}`
- `POST /stores/{storeId}/products/{productId}/status`
- `POST /stores/{storeId}/products/{productId}/archive`
- `GET/POST /stores/{storeId}/collections`

### Phase 3 commerce (auth + membership)

| Method | Path | Notes |
| --- | --- | --- |
| GET/POST | `/stores/{storeId}/payment-methods` | Manual bank transfer accounts |
| PATCH/DELETE | `/stores/{storeId}/payment-methods/{methodId}` | Update / remove |
| GET/PATCH | `/stores/{storeId}/fulfillment` | Shipping / free shipping / pickup |
| GET | `/stores/{storeId}/orders` | Query: `q`, `order_status`, `payment_status`, `fulfillment_status`, `limit`, `offset` |
| GET/PATCH | `/stores/{storeId}/orders/{orderId}` | Detail + status transitions |
| POST | `/stores/{storeId}/orders/{orderId}/payments/{paymentId}/verify` | Verify evidence |
| POST | `/stores/{storeId}/orders/{orderId}/payments/{paymentId}/reject` | Body `{reason}` |
| GET | `/stores/{storeId}/orders/evidence/{evidenceId}/file` | Private evidence stream |
| GET | `/stores/{storeId}/customers` | Store-scoped customers |
| GET/PATCH | `/stores/{storeId}/customers/{customerId}` | Detail + notes/name/phone |

Checkout enables (`stores.checkout_enabled=true`) only when ≥1 enabled bank account and ≥1 fulfillment method exist.

### Phase 4 preorders + business rules (auth + membership)

| Method | Path | Notes |
| --- | --- | --- |
| GET/POST | `/stores/{storeId}/preorders` | List / create batches |
| GET/PATCH | `/stores/{storeId}/preorders/{batchId}` | Detail + status/window updates (`cover_file_id`, `clear_cover`) |
| POST | `/stores/{storeId}/preorders/{batchId}/products` | Assign / upsert product |
| DELETE | `/stores/{storeId}/preorders/{batchId}/products/{productId}` | Remove assignment |
| GET/POST | `/stores/{storeId}/preorders/{batchId}/products/{productId}/variants` | List / assign variants |
| DELETE | `/stores/{storeId}/preorders/{batchId}/products/{productId}/variants/{variantId}` | Remove variant |
| GET/PUT | `/stores/{storeId}/preorders/{batchId}/purchasing` | Consolidated purchasing list |
| POST | `/stores/{storeId}/preorders/{batchId}/allocations` | Allocate purchased qty to item (emails customer when unavailable) |
| GET | `/stores/{storeId}/preorders/{batchId}/orders` | Orders in batch |
| GET/POST | `/stores/{storeId}/business-rules` | List / publish rule (new version) |
| PATCH/DELETE | `/stores/{storeId}/business-rules/{ruleId}` | Update (bumps version) / delete |
| POST | `/stores/{storeId}/business-rules/preview` | Same evaluation as checkout |

Rule precedence per type: batch → selling model → store. Checkout cart lines may include `batch_id` (one group per submit).

### Phase 5 notifications + analytics

| Method | Path | Notes |
| --- | --- | --- |
| GET | `/stores/{storeId}/notifications` | List; `?unread=1` |
| GET | `/stores/{storeId}/notifications/unread-count` | Badge count |
| POST | `/stores/{storeId}/notifications/{id}/read` | Mark one read |
| POST | `/stores/{storeId}/notifications/read-all` | Mark all read |
| GET/PATCH | `/stores/{storeId}/notification-preferences` | Optional operational prefs |
| GET | `/stores/{storeId}/analytics` | `?from=&to=&tz=` (default Asia/Jakarta) |

## Catalog imports (AI PDF + CSV)

| Method | Path | Notes |
| --- | --- | --- |
| POST | `/stores/{storeId}/catalog-imports` | Multipart `file`; `source=pdf|csv` |
| GET | `/stores/{storeId}/catalog-imports/csv-template` | Downloadable CSV template |
| GET | `/stores/{storeId}/catalog-imports` | Recent jobs |
| GET | `/stores/{storeId}/catalog-imports/{jobId}` | Job status |
| GET/PATCH/DELETE | `…/items` | Review items |
| POST | `…/confirm` | Create draft products |
| POST | `…/retry` | Retry failed PDF jobs |

Uses mock AI when `AI_API_KEY` is unset. CSV is parsed synchronously into `review_required`.

## Public storefront

- `GET /public/stores/{slug}`
- `GET /public/stores/{slug}/products`
- `GET /public/stores/{slug}/products/{productSlug}` — includes `product_url` + `link_ref`
- `GET /public/stores/{slug}/categories`
- `GET /public/stores/{slug}/catalog` / `…/catalog/products/{productID}`
- `GET /public/files/{storeId}/{fileId}` — published stores only

### Phase 4 public preorders

| Method | Path | Notes |
| --- | --- | --- |
| GET | `/public/stores/{slug}/preorders` | Published batches |
| GET | `/public/stores/{slug}/preorders/{batchSlug}` | Batch + assigned products |
| GET | `/public/stores/{slug}/products/{productSlug}/preorders` | Open batches containing product |

### Phase 3–4 public checkout + tracking

| Method | Path | Notes |
| --- | --- | --- |
| POST | `/public/stores/{slug}/checkout/preview` | Body `{items[{product_id,quantity,batch_id?}], fulfillment_method?}` — rate limited |
| POST | `/public/stores/{slug}/checkout/submit` | Guest checkout; `idempotency_key` required; returns `tracking_token` once |
| GET | `/public/tracking/{token}` | Order tracking (noindex); hash lookup only |
| POST | `/public/tracking/{token}/evidence` | Multipart `file` (image/PDF ≤8 MB) |
| GET | `/public/tracking/{token}/evidence/{evidenceId}/file` | Authorized evidence download |

Submit body (essentials):

```json
{
  "items": [{ "product_id": "…", "quantity": 1 }],
  "customer_name": "Budi",
  "customer_email": "budi@example.com",
  "fulfillment_method": "shipping",
  "shipping_address": {
    "recipient_name": "Budi",
    "line1": "Jl. …",
    "city": "Jakarta",
    "country": "ID"
  },
  "payment_method_id": "…",
  "idempotency_key": "…",
  "client_expected_total_idr": 215000
}
```

All money fields are integer IDR. `price_changed` (409) when client total ≠ server total.

## Health / readiness (Phase 6)

| Path | Purpose |
| --- | --- |
| `GET /livez` | Process up (no dependency checks) |
| `GET /healthz` | DB ping |
| `GET /readyz` | DB query + storage path / config readiness |

Responses are JSON `{ ok, time, ... }`. `readyz` returns 503 when a check fails.

Coarse per-IP API rate limiting uses `API_RATE_LIMIT_IP_PER_HOUR` (default 1200) via `public_rate_limits`. Auth magic-link keeps tighter email/IP limits.
