# Catalog link contract (flipbook / hotspot ready)

Phase 2 products are the stable target for later flipbook catalog pages and checkout hotspots.

## Stable identifiers

| Field | Use |
| --- | --- |
| `products.id` (UUID) | Preferred hotspot `product_id` foreign key |
| `products.slug` (store-scoped unique) | Public URL segment |
| `products.sku` (optional) | Merchant reference; not required for linking |
| `products.status` | `draft` \| `published` \| `archived` — only `published` is public |

## Images

- Upload via `POST /api/v1/stores/{storeId}/files` with `purpose=product`
- Attach via `image_file_ids` on create/update product
- Public URL: `GET /api/v1/public/files/{storeId}/{fileId}`
- Merchant URL: `GET /api/v1/stores/{storeId}/files/{fileId}` (auth + membership)

## Merchant APIs

- `GET/POST /api/v1/stores/{storeId}/products`
- `GET/PATCH /api/v1/stores/{storeId}/products/{productId}`
- `POST /api/v1/stores/{storeId}/products/{productId}/status` body `{ "status": "published" }`
- `POST /api/v1/stores/{storeId}/products/{productId}/archive`

## Public APIs

- `GET /api/v1/public/stores/{slug}/products`
- `GET /api/v1/public/stores/{slug}/products/{productSlug}` → includes `link_ref` and `product_url`

## Suggested hotspot row (future worker)

```
catalog_page_hotspots (
  id, store_id, catalog_page_id,
  product_id REFERENCES products(id),
  x_pct, y_pct, width_pct, height_pct
)
```

Always scope by `store_id` and authorize membership; resolve checkout only for `status=published` products in the same store.
