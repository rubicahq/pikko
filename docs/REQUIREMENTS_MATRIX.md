# Requirements Matrix (Phase 0–7)

Status: `done` | `partial` | `deferred` | `owner`

| ID | Requirement | Status | Evidence |
| --- | --- | --- | --- |
| P0-1 | Monorepo structure | done | `apps/web`, `services/api`, `packages/*`, `database/` |
| P0-2 | Next.js + TS + Tailwind + Radix/shadcn + Lucide + Inter | done | `apps/web` (+ Plus Jakarta Sans display for marketing) |
| P0-3 | Go API | done | `services/api` |
| P0-4 | PostgreSQL + migrations | done | `database/migrations` |
| P0-5 | Env config + `.env.example` | done | root `.env.example`, `.env.production.pikko.id.example` |
| P0-6 | Storage abstraction | done | local + store-scoped paths; S3 stub |
| P0-7 | Design tokens → Tailwind | done | `packages/design-tokens` |
| P0-8 | Reusable component library | done | `components/ui` |
| P0-9 | Dev-only component preview | done | `/dev/components` |
| P0-10 | Central URL generation | done | Go `urls` + web `urls` / `site.ts` |
| P0-11 | Docs kept accurate | done | `docs/*` |
| P0-12 | Docker web/api + compose | done | Dockerfiles, compose |
| P1-1–P1-10 | Auth + multi-store | done | Phase 1 |
| P2-1–P2-7 | Storefront + catalog + import/flipbook | done | Phase 2 |
| P3-1–P3-10 | Checkout + payments + tracking | done | Phase 3 |
| P4-1–P4-8 | Preorder + business rules | done | Phase 4–5 |
| P5-1–P5-6 | Notifications, analytics, CSV, customers | done | Phase 5 |
| P6-1–P6-17 | Marketing + production readiness | done | Phase 6 |
| P7-1 | Domain-agnostic APP_URL / surface URLs | done | config `APP_URL`, compose, `site.ts` |
| P7-2 | Cookie domain configurable | done | `AUTH_COOKIE_DOMAIN` |
| P7-3 | CORS / redirect allowlist for new domain | done | config defaults + legacy |
| P7-4 | Email links / From from base domain | done | email defaults + URL generator |
| P7-5 | Canonical + sitemap from env | done | `marketing-seo.ts`, `sitemap.ts` |
| P7-6 | Redirect pikko.site → pikko.id templates | done | nginx redirect example + middleware |
| P7-7 | Nginx/compose templates for pikko.id | done | `emplicon-nginx.d-pikko.id.conf.example`, env example |
| P7-8 | Launch runbook | done | `docs/LAUNCH.md` |
| P7-9 | Launch QA checklist (code gates) | done | handoff + Playwright/Go gates |
| P7-10 | Execute DNS/TLS/Brevo cutover | owner | not by agent — needs approval |
| P7-11 | Commercial pricing | deferred | open business decision |
| P7-12 | Counsel-final legal / support process | owner / deferred | working notices shipped |
| — | VPS backup without docker group | done | `backup.sh` / `restore-test.sh` sudo fallback |
| — | VPS deploy / Emplicon cutover | owner | `docs/LAUNCH.md` |
