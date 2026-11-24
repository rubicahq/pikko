# Requirements Matrix (Phase 0 + Phase 1)

Status: `done` | `partial` | `deferred` (Phase 2+)

| ID | Requirement | Status | Evidence |
| --- | --- | --- | --- |
| P0-1 | Monorepo structure | done | `apps/web`, `services/api`, `packages/*`, `database/` |
| P0-2 | Next.js + TS + Tailwind + Radix/shadcn + Lucide + Inter | done | `apps/web` |
| P0-3 | Go API | done | `services/api` |
| P0-4 | PostgreSQL + migrations | done | `database/migrations`, migrate cmd |
| P0-5 | Env config + `.env.example` | done | root `.env.example` |
| P0-6 | Storage abstraction | done | `internal/storage` local + S3 stub |
| P0-7 | Design tokens → Tailwind | done | `packages/design-tokens` |
| P0-8 | Reusable component library | done | `components/ui` + preview |
| P0-9 | Dev-only component preview | done | `/dev/components` gated |
| P0-10 | Central URL generation | done | Go + web `urls` |
| P0-11 | Nine docs kept accurate | done | `docs/*` |
| P0-12 | Docker web/api + compose | done | Dockerfiles, `docker-compose.yml` |
| P1-1 | Brevo adapter + safe mail mode | done | `EMAIL_MODE=log\|brevo` |
| P1-2 | Magic link auth per §9 | done | peek GET / verify POST / hashed / rate limits / sessions |
| P1-3 | Account creation after verify | done | idempotent user insert |
| P1-4 | Two-step first-store onboarding + resume | done | drafts + UI wizard |
| P1-5 | Store switcher | done | sidebar dropdown |
| P1-6 | Manage stores + additional + archive | done | `/stores` |
| P1-7 | Account settings | done | `/account/settings` |
| P1-8 | Store auth middleware + explicit routes | done | `/stores/{id}` API + web |
| P1-9 | Cross-store access tests | done | Go integration tests |
| P1-10 | Persist in Postgres, no placeholders | done | real screens wired to API |
| — | Phase 2+ commerce modules | deferred | nav stubs only |
