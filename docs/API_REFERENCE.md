# API Reference (Phase 0–2 + AI import)

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

Product write body:

```json
{
  "product": {
    "name": "Tote",
    "sku": "TOTE-1",
    "base_price_idr": 125000,
    "status": "draft"
  },
  "options": [],
  "variants": [],
  "image_file_ids": ["…"]
}
```

## Catalog imports (AI PDF)

Uses mock AI when `AI_API_KEY` is unset.

| Method | Path | Notes |
| --- | --- | --- |
| POST | `/stores/{storeID}/catalog-imports` | Multipart `file` (PDF) + `extract_images`. Optional `Idempotency-Key`. Limits: 25 MB, 50 pages, 1 PDF/job. |
| GET | `/stores/{storeID}/catalog-imports` | Recent jobs |
| GET | `/stores/{storeID}/catalog-imports/{jobID}` | Job status/progress |
| GET | `/stores/{storeID}/catalog-imports/{jobID}/items` | Review items |
| PATCH | `/stores/{storeID}/catalog-imports/{jobID}/items/{itemID}` | Edit/select/flag uncertain |
| DELETE | `/stores/{storeID}/catalog-imports/{jobID}/items/{itemID}` | Remove item |
| POST | `/stores/{storeID}/catalog-imports/{jobID}/confirm` | Body `{item_ids?}` → create **draft** products + hotspot candidates |
| POST | `/stores/{storeID}/catalog-imports/{jobID}/retry` | Re-queue failed job |

Statuses: `pending` → `processing` → `review_required` → `completed` (or `failed`).

## Catalog flipbook

| Method | Path | Notes |
| --- | --- | --- |
| GET | `/stores/{storeID}/catalog-pages` | Merchant pages + hotspots |
| PUT | `/stores/{storeID}/catalog-pages/order` | Body `{page_ids:[…]}` |
| POST | `/stores/{storeID}/catalog-pages/status` | Body `{status:"published"|"draft", page_ids?}` |
| POST | `/stores/{storeID}/catalog-pages/{pageID}/hotspots` | Body bbox + `product_id` |
| PATCH | `/stores/{storeID}/catalog-pages/hotspots/{hotspotID}` | Move/relink hotspot |
| DELETE | `/stores/{storeID}/catalog-pages/hotspots/{hotspotID}` | Remove hotspot |
| GET | `/public/stores/{slug}/catalog` | Published pages; hotspots only for published products |
| GET | `/public/stores/{slug}/catalog/products/{productID}` | Product card + `checkout_hook` |

## Public

- `GET /public/stores/{slug}`
- `GET /public/stores/{slug}/products`
- `GET /public/stores/{slug}/products/{productSlug}` — includes `product_url` + `link_ref`
- `GET /public/stores/{slug}/categories`
- `GET /public/files/{storeId}/{fileId}` — published stores only

All public product payloads omit `cost_price_idr`.

## Health

`GET /healthz` — DB ping.
