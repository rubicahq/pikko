# Requirements Matrix (Phase 0–4)

Status: `done` | `partial` | `deferred`

| ID | Requirement | Status | Evidence |
| --- | --- | --- | --- |
| P0-1 | Monorepo structure | done | `apps/web`, `services/api`, `packages/*`, `database/` |
| P0-2 | Next.js + TS + Tailwind + Radix/shadcn + Lucide + Inter | done | `apps/web` |
| P0-3 | Go API | done | `services/api` |
| P0-4 | PostgreSQL + migrations | done | `database/migrations` |
| P0-5 | Env config + `.env.example` | done | root `.env.example` |
| P0-6 | Storage abstraction | done | local + store-scoped paths; S3 stub |
| P0-7 | Design tokens → Tailwind | done | `packages/design-tokens` |
| P0-8 | Reusable component library | done | `components/ui` |
| P0-9 | Dev-only component preview | done | `/dev/components` |
| P0-10 | Central URL generation | done | Go + web `urls` |
| P0-11 | Docs kept accurate | done | `docs/*` |
| P0-12 | Docker web/api + compose | done | Dockerfiles, compose |
| P1-1–P1-10 | Auth + multi-store | done | Phase 1 |
| P2-1 | Store identity + logo/favicon/store icon | done | settings + storefront draft assets |
| P2-2 | Storefront config + 24 themes, draft preview, publish | done | themes + editor + publish API |
| P2-3 | Catalog categories/products/variants/images/IDR/status | done | catalog service + UI |
| P2-4 | Public storefront SEO + favicon/logo | done | `/{slug}` pages |
| P2-5 | Store-scoped queries/cache/storage + authz | done | middleware + isolation tests |
| P2-6 | Flipbook-ready product link contract | done | `docs/CATALOG_LINK_CONTRACT.md` |
| P2-7 | AI PDF import + flipbook | done | migrations `000003`, import + flipbook UI |
| P3-1 | Real cart (store-scoped) | done | `apps/web/src/lib/cart.ts` + cart page |
| P3-2 | Guest checkout + IDR integer totals | done | checkout preview/submit APIs + UI |
| P3-3 | Order creation with snapshots + statuses | done | `orders` / `order_items` |
| P3-4 | Manual bank transfer + merchant accounts | done | `store_payment_methods` + settings UI |
| P3-5 | Payment evidence (private) + verify/reject | done | evidence upload + merchant actions |
| P3-6 | Merchant order list/detail + fulfillment | done | `/stores/{id}/orders` |
| P3-7 | Public tracking token (hash-only, ≥256-bit) | done | `/track/{token}` + `order_tracking_tokens` |
| P3-8 | Transactional emails + idempotent jobs | done | Brevo adapter + `email_jobs` drain |
| P3-9 | Stock reservations (finite) | done | `inventory_reservations` on submit |
| P3-10 | Public rate limits + store-scoped authz | done | `public_rate_limits` + cross-store tests |
| P4-1 | Preorder batches + product assignment + quotas | done | `000005` + merchant/public preorder APIs/UI |
| P4-2 | Preorder-aware checkout + cart groups | done | `batch_id` on lines; group checkout |
| P4-3 | Business rules precedence + versioning | done | `business_rules` / versions / order snapshots |
| P4-4 | Handling fee / minimum / approval | done | evaluate at preview/submit + UI |
| P4-5 | Purchasing list + allocations | done | purchasing API + batch detail tab |
| P4-6 | Extended fulfillment for preorder | done | purchasing → awaiting_arrival → ship/pickup |
| P4-7 | Emails for approval + existing drain | done | needs-confirmation template + outbox |
| — | VPS deploy / Emplicon | deferred | separate after merge |
