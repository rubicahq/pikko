# Requirements Matrix (Phase 0–2)

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
| P1-1–P1-10 | Auth + multi-store | done | Phase 1 PR |
| P2-1 | Store identity + logo/favicon/store icon | done | settings + storefront draft assets |
| P2-2 | Storefront config + 24 themes, draft preview, publish | done | themes package + editor + publish API |
| P2-3 | Catalog categories/products/variants/images/IDR/status | done | catalog service + UI |
| P2-4 | Public storefront SEO + favicon/logo | done | `/{slug}` pages |
| P2-5 | Store-scoped queries/cache/storage + authz | done | middleware + isolation tests |
| P2-6 | Flipbook-ready product link contract | done | `docs/CATALOG_LINK_CONTRACT.md`, `link_ref` |
| — | Flipbook UI / AI PDF import | deferred | follow-on workers |
| — | Checkout / orders / payments | deferred | Phase 3+ |
