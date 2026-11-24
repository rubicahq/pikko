# API Reference (Phase 0–2)

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

## Public

- `GET /public/stores/{slug}`
- `GET /public/stores/{slug}/products`
- `GET /public/stores/{slug}/products/{productSlug}` — includes `product_url` + `link_ref`
- `GET /public/stores/{slug}/categories`
- `GET /public/files/{storeId}/{fileId}` — published stores only

All public product payloads omit `cost_price_idr`.
