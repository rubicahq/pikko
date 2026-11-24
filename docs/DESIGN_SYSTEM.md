# Design System — Pikko (PRD v3.0)

**Owner:** Agent 02 (Design System & Frontend)  
**Visual reference:** https://www.brex.com/ — character, layout discipline, and density only. Do **not** copy Brex assets, fonts, illustrations, or copy.  
**Tokens source of truth:** `packages/design-tokens/` (`tokens.css`, `design-tokens.css`, `tailwind.preset.js`).  
**Components:** `apps/web/src/components/ui` + `apps/web/src/components/dashboard`.  
**Dev preview:** `/dev/components` (blocked in production).

---

## 1. Personality

Premium, modern, minimal, approachable, confident. Strong typography over decoration. White / cool neutrals with **restrained** brand orange (~5% accent). Borders define boundaries; shadows are minimal.

Observable Brex traits we adopt (not copy):

- Generous whitespace and clear section rhythm
- Dark charcoal primary CTAs paired with a single warm accent
- Dense operational tables with quiet chrome
- Sidebar IA with grouped sections, not a flat icon dump
- Marketing hero = brand + one headline + one supporting line + CTA — not KPI strips

### Avoid (AI-generic)

- Purple→indigo gradients; cream `#F4F1EA` + terracotta + serif
- Solid-orange sidebars; orange on every icon
- Pill `rounded-full` everywhere; heavy multi-layer shadows; glow/glass
- 3-column icon-in-circle feature grids; big-number KPI heroes on marketing
- Undifferentiated Inter-only hierarchy (use scale/weight; marketing display = Plus Jakarta Sans)

---

## 2. Color tokens

| Token | CSS var | Value | Use |
| --- | --- | --- | --- |
| Brand primary | `--brand-primary` | `#FF3D00` | Accent, primary buttons (with dark text) |
| Brand hover | `--brand-hover` | `#EB3800` | Hover |
| Brand pressed | `--brand-pressed` | `#D63200` | Active |
| Brand accessible | `--brand-accessible-dark` | `#C83200` | White text on brand; focus rings |
| Brand soft | `--brand-soft` | `#FFF1EB` | Soft highlight / selected row |
| Brand border | `--brand-border` | `#FFD2C2` | Soft brand outlines |
| Background | `--background` | `#FFFFFF` | App canvas |
| Background subtle | `--background-subtle` | `#F9F9FB` | Alternating rows, soft panels |
| Surface | `--surface` | `#F3F3F7` | Nav active, table header |
| Surface strong | `--surface-strong` | `#E8E9EE` | Stronger chrome |
| Foreground | `--foreground` | `#15191E` | Primary text |
| Foreground secondary | `--foreground-secondary` | `#60646C` | Supporting text |
| Foreground muted | `--foreground-muted` | `#8B8D98` | Meta / placeholders |
| Foreground subtle | `--foreground-subtle` | `#B9BBC6` | Disabled hints |
| Foreground inverse | `--foreground-inverse` | `#FCFCFD` | Text on dark |
| Dark | `--dark` | `#000710` | Secondary CTA / charcoal |
| Border | `--border` | `#E1E3E8` | Default borders |
| Border strong | `--border-strong` | `#C7CAD2` | Emphasized borders |

### Semantic (never use brand orange for warning/error)

| Tone | Text | Background | Border |
| --- | --- | --- | --- |
| Success | `#166534` | `#F0FDF4` | `#BBF7D0` |
| Warning | `#92400E` | `#FFFBEB` | `#FDE68A` |
| Error | `#B91C1C` | `#FEF2F2` | `#FECACA` |
| Info | `#1D4ED8` | `#EFF6FF` | `#BFDBFE` |
| Neutral | `#4B5563` | `#F3F4F6` | `#E5E7EB` |

**Distribution:** white/light ~75%, charcoal ~15%, secondary neutral ~5%, orange ~5%.

**A11y:** White small text on `#FF3D00` fails WCAG. Primary buttons use dark text on brand orange, or white on `#C83200`.

Tailwind: `bg-brand`, `text-foreground-secondary`, `border-border`, `bg-success-bg`, etc. via `tailwind.preset.js`.

---

## 3. Typography

| Context | Font | CSS |
| --- | --- | --- |
| Product UI | Inter (`next/font` → `--font-inter`) | `font-sans` |
| Marketing display | Plus Jakarta Sans (`--font-jakarta`) | `font-display` |

Body 400, headings 500, selective 600. Tabular nums for money (`tabular-nums`). Sentence case. Marketing headlines tracking ≈ `-0.02em` to `-0.03em`.

### Product type ramp

| Role | Utility | Size / leading | Weight |
| --- | --- | --- | --- |
| Page title | `text-page` | 30 / 36 | 500 |
| Section | `text-section` | 20 / 28 | 500 |
| Card title | `text-card` | 16 / 24 | 500 |
| Body | `text-body` | 14 / 20 | 400 |
| Table cell | `text-table` | 13 / 18 | 400 |
| Label | `text-label` | 12 / 16 | 500 |
| Caption | `text-caption` | 11 / 14 | 400 |

### Marketing display ramp

| Role | Utility | Size / leading |
| --- | --- | --- |
| Display XL | `text-display-xl` | 56 / 64 |
| Display LG | `text-display-lg` | 44 / 52 |
| Display MD | `text-display-md` | 36 / 44 |
| Display SM | `text-display-sm` | 28 / 36 |

---

## 4. Spacing, radius, shadow, motion

- **Spacing base:** 4px. Approved: 4, 8, 12, 16, 20, 24, 32, 40, 48, 64, 80, 96, 128. Prefer Tailwind scale (`p-4`, `gap-3`, …). Content gutter: `--content-gutter` / `px-gutter` when needed.
- **Radii:** xs 4, sm 6, md 8 (buttons/inputs), lg 10 (cards/tables), xl 12, 2xl 16. **Avoid pills.**
- **Shadows:** `shadow-xs` / `sm` / `md` / `overlay` — use sparingly; prefer 1px borders.
- **Icons:** Lucide, stroke **1.75**; sizes 16 / 20 / 24 / 32.
- **Motion:** hover 150ms, standard 200ms, dialog 220ms. Respect `prefers-reduced-motion` (global CSS).

---

## 5. Layout & breakpoints

| Surface | Notes |
| --- | --- |
| Marketing | max ~1280; reading 680–760 |
| Dashboard | sidebar 248 (collapsed token 64); content max 1440; topbar 56 |
| Auth | form max 420 |
| Checkout | max 1120; two-column desktop |

Validate: **375, 390, 768, 1024, 1440** — no horizontal overflow.

---

## 6. Merchant navigation IA (PRD §6)

Grouped sidebar (`nav-config.ts`):

| Group | Live items | Hidden until built |
| --- | --- | --- |
| **Workspace** | Overview, Notifications, Analytics | — |
| **Operations** | Orders, Customers | Purchasing (standalone), Packing, Shipments, Payments/Finance |
| **Configuration** | Catalog, Preorders (campaigns), Storefront, Business Rules, Settings | Vendors, Team |

Rules:

- Store context always explicit via store switcher + `/stores/{storeId}/…` routes
- Unimplemented modules are **omitted** (not dead links); group labels still render when the group has ≥1 live item
- Role-aware filtering (`NAV-03`) deferred to Agent 03 / RBAC

Shell: `AppShell` + desktop `DashboardSidebar` + mobile `MobileNav` (sheet). Store switcher supports create + manage stores.

---

## 7. Component library

| Component | Path | Notes |
| --- | --- | --- |
| Button | `ui/button.tsx` | primary / secondary / outline / ghost / destructive / link; loading |
| Input / Label / IdR | `ui/input`, `label`, `idr-input` | Error ring; IDR integer formatting |
| Card | `ui/card.tsx` | Border, radius lg, padding 20 |
| Badge | `ui/badge.tsx` | Semantic tones |
| StatusBadge | `ui/status-badge.tsx` | Maps Paid / Awaiting / Confirmed / Draft / Open / Closed / Cancelled / Failed (+ order enums) |
| Table primitives | `ui/table.tsx` | Sticky header support |
| DataTable | `ui/data-table.tsx` | Client sort / filter / pagination / sticky header |
| EmptyState | `ui/empty-state.tsx` | Title, description, optional action |
| ErrorState | `ui/error-state.tsx` | Recoverable error panel |
| Skeleton | `ui/skeleton.tsx` | Loading placeholders |
| Dialog | `ui/dialog.tsx` | Radix dialog |
| Sheet | `ui/sheet.tsx` | Side drawer (mobile nav) |
| Tabs | `ui/tabs.tsx` | Radix tabs / filter chrome |
| Select | `ui/select.tsx` | Radix select |
| Separator | `ui/separator.tsx` | Quiet rules |
| Toast | `ui/toast.tsx` | Radix toast + provider |
| PageHeader | `ui/page-header.tsx` | Title + description + actions |
| FilterBar | `ui/filter-bar.tsx` | Search + trailing actions row |

Stack: Tailwind + Radix + Lucide. Customize — do not ship raw default shadcn chrome.

### Data tables (dense workflows)

- Header: `bg-surface`, uppercase `text-label`, sticky optional
- Rows: 1px `border-border`; hover `bg-background-subtle`
- Money / counts: `tabular-nums`, right-aligned
- Empty → `EmptyState`; loading → `Skeleton` rows; failure → `ErrorState`
- Client interactivity via `DataTable`; SSR list pages may use table primitives + server filter

### Status badges (required set)

| Label | Tone |
| --- | --- |
| Paid | success |
| Awaiting / Awaiting verification | warning |
| Confirmed / Open / Info | info |
| Draft / Closed / Neutral | neutral |
| Cancelled / Failed / Error | error |
| Brand highlight (Current) | brand |

---

## 8. Storefront themes

24 curated themes in Go `internal/themes` (Pikko Signature → Contemporary). Merchant theme colors **never** affect backoffice. Contrast helpers stay in themes package tests.

---

## 9. Usage rules for later agents

1. Import from `@/components/ui/*` and `@/components/dashboard/*` — do not invent parallel palettes.
2. Prefer `PageHeader` + `EmptyState` / `DataTable` / `StatusBadge` on new merchant list pages.
3. Extend `nav-config.ts` when a module ships; keep unimplemented entries out of the live list.
4. Marketing: keep hero budget tight; no AI-generic feature grids.
5. No destructive redesign of working storefront checkout flows.

---

## 10. Acceptance (Agent 02)

- [x] This file is the full token / type / component / nav spec (not a stub)
- [x] Shared components cover dense tables + empty/loading/error + dialogs/tabs/status
- [x] App shell + NAV-02 grouping wired; unimplemented modules hidden
- [x] `/dev/components` documents variants
- [x] Requirements `DS-*` / `NAV-*` updated in `REQUIREMENTS_MATRIX.md`
- [x] No storefront/checkout rebuild; no DB migrations; no pikko.id cutover

See `docs/agents/02-design-system-handoff.md`.
