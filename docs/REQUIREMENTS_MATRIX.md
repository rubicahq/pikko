# Requirements Matrix (Phase 0–6)

Status: `done` | `partial` | `deferred`

| ID | Requirement | Status | Evidence |
| --- | --- | --- | --- |
| P0-1 | Monorepo structure | done | `apps/web`, `services/api`, `packages/*`, `database/` |
| P0-2 | Next.js + TS + Tailwind + Radix/shadcn + Lucide + Inter | done | `apps/web` (+ Plus Jakarta Sans display for marketing) |
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
| P2-1–P2-7 | Storefront + catalog + import/flipbook | done | Phase 2 |
| P3-1–P3-10 | Checkout + payments + tracking | done | Phase 3 |
| P4-1–P4-8 | Preorder + business rules | done | Phase 4–5 |
| P5-1–P5-6 | Notifications, analytics, CSV, customers | done | Phase 5 |
| P6-1 | Marketing landing (brand-first) | done | `apps/web/src/app/(marketing)/page.tsx` |
| P6-2 | Solution pages | done | `/solutions`, `/jastip`, `/social-sellers` |
| P6-3 | Pricing content (pilot; no invented plans) | done | `/pricing` |
| P6-4 | FAQ | done | `/faq` |
| P6-5 | Legal privacy + terms | done | `/privacy`, `/terms` |
| P6-6 | SEO sitemap + robots + OG + canonicals | done | `sitemap.ts`, `robots.ts`, `opengraph-image.tsx` |
| P6-7 | Accessibility validation (marketing) | done | landmarks, focus, FAQ keyboard, reduced motion; e2e |
| P6-8 | Security headers + hardening | done | API middleware + Next middleware/headers + nginx example |
| P6-9 | Monitoring health endpoints | done | `/livez`, `/healthz`, `/readyz` |
| P6-10 | Structured request logging | done | `AccessLogMiddleware` |
| P6-11 | Rate limiting | done | auth limits + `API_RATE_LIMIT_IP_PER_HOUR` |
| P6-12 | Scheduled DB + storage backup | done | `infrastructure/backup/*` |
| P6-13 | Backup restore validation script | done | `restore-test.sh` |
| P6-14 | Operational runbook | done | `docs/RUNBOOK.md` |
| P6-15 | Light load smoke | done | `infrastructure/ops/load-smoke.sh` |
| P6-16 | Dependency audit script | done | `infrastructure/ops/dependency-audit.sh` |
| P6-17 | E2E regression (Phase 6 + prior) | done | Playwright suite |
| — | VPS deploy / Emplicon | deferred | operator after merge |
| — | Phase 7 pikko.id GTM | deferred | needs explicit approval |
