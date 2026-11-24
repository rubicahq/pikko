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

## Catalog pages & hotspots

Migration `000003` introduces:

```
catalog_pages (
  id, store_id, image_file_id, display_order,
  status draft|published, source_import_job_id, page_number
)

catalog_page_hotspots (
  id, store_id, catalog_page_id,
  product_id REFERENCES products(id),
  import_item_id, x_pct, y_pct, width_pct, height_pct
)
```

AI PDF import renders each PDF page into a draft `catalog_pages` row and stores normalized bounding boxes on `catalog_import_items`. Confirming selected items creates draft products and optional hotspot candidates (`product_id` + bbox on the matching page).

Always scope by `store_id` and authorize membership; resolve checkout only for `status=published` products in the same store. Public flipbook surfaces must never expose draft products or draft catalog pages.

## Public flipbook

- Merchant editor: `/stores/{storeId}/catalog/flipbook`
- Public viewer: `/{storeSlug}/catalog`
- Product deep link: `/{storeSlug}/products/{productSlug}`
- Hotspot click loads `GET /api/v1/public/stores/{slug}/catalog/products/{productId}` then offers **Add to cart** via `apps/web/src/lib/cart.ts` (store-scoped cart) plus cart/checkout links. Flow: `/{storeSlug}/cart` → `/{storeSlug}/checkout` → `/track/{token}`. Prices/stock are always revalidated server-side on checkout preview/submit.
- Import review items expose `confidence_pct` and `review_status` (`pending`|`accepted`|`rejected`); confirm creates draft products and clamped hotspot candidates.
