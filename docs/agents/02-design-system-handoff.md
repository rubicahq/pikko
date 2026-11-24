# Agent 02 → Agent 03 Handoff (Design System → Auth & Multi-Store)

**Date:** 2026-11-24 (Asia/Jakarta)  
**PRD:** v3.0 (`docs/PIKKO_MASTER_PRD.md`)  
**Branch:** `cursor/agent-02-design-system-5ff3`  
**Prior:** Agent 01 at `f029fc489d69f09b6937d2ed1a6ba1a8bd720d1f`

---

## 1. What Agent 02 delivered

| Area | Outcome |
| --- | --- |
| `docs/DESIGN_SYSTEM.md` | Full token tables, type ramp, component specs, nav IA, usage rules |
| Tokens | Type ramp + layout vars in `packages/design-tokens/tokens.css`; Tailwind `fontSize` utilities in preset |
| Shared UI | Table, DataTable (sort/filter/pagination/sticky), Empty/Error/Skeleton, Dialog, Sheet, Tabs, Select, Separator, StatusBadge, Toast, PageHeader, FilterBar |
| App shell | `AppShell` + desktop sidebar + mobile sheet; store switcher preserved |
| NAV-02 | Workspace / Operations / Configuration via `nav-config.ts`; unimplemented modules omitted |
| List pages | Orders, Customers, Catalog use shared primitives |
| Preview / tests | `/dev/components` expanded; `e2e/design-system.spec.ts` |
| Matrix | `DS-*`, `NAV-01/02` updated |

**No Go/API changes. No migrations. No pikko.id cutover. Storefront checkout flows preserved.**

---

## 2. Visual reference notes (Brex — character only)

Adopted: cool neutrals, restrained accent, dense tables, grouped sidebar IA, marketing hero discipline.  
Not copied: Brex assets, fonts, illustrations, marketing copy, proprietary layouts.

Brand remains `#FF3D00` with Inter (product) + Plus Jakarta Sans (marketing display).

---

## 3. How later agents extend the shell

1. Flip `implemented: true` in `apps/web/src/components/dashboard/nav-config.ts` when a route ships.
2. Prefer `@/components/ui/*` — do not invent parallel colors.
3. New list pages: `PageHeader` + `EmptyState` / `Table` or `DataTable` + `StatusBadge`.
4. Wrap interactive trees that need toasts with `<Toaster>`.

---

## 4. Known limitations

| Item | Note |
| --- | --- |
| NAV-03 role-aware nav | Needs RBAC (Agent 03+) |
| Desktop collapsed icon rail | Token `--sidebar-collapsed` reserved; not wired |
| Global toast host | Opt-in `Toaster` per subtree (preview demo); Agent 03+ may mount at shell |
| Purchasing as top-level nav | Still lives under preorder batch detail until Agent 06 |
| Marketing homepage redesign | Intentionally untouched beyond discipline rules |

---

## 5. Validation run (this agent)

- `npx tsc --noEmit` (apps/web)
- `npm run lint`
- `npm run build`
- Playwright: design-system + existing suites (as CI allows)

---

## 6. Next — Agent 03: Auth & Multi-Store

Start with:

- Source of truth: `docs/PIKKO_MASTER_PRD.md` §5 / RBAC
- Prior handoff: this file
- Focus: session hardening; additive RBAC (`STORE-06`); keep design shell; do not break magic-link auth
- Preserve Agent 02 components and NAV grouping

Acceptance gate reminder: auth regression green; role CHECK widened safely; no destructive data loss.
