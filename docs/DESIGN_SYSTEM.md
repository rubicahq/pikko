# Design System — Pikko (soft-gradient SaaS)

**Owner:** Design / frontend agents  
**Direction:** Soft-gradient SaaS built on Brex-inspired nav, hierarchy, and operational density. Warmer, more approachable identity — not a flat corporate dashboard and not a neon/generic AI look.  
**Tokens source of truth:** `packages/design-tokens/` (`tokens.css`, `design-tokens.css`, `tailwind.preset.js`). Project preference: soft-gradient palette + Plus Jakarta Sans.  
**Components:** `apps/web/src/components/ui` + `apps/web/src/components/dashboard`.  
**Dev preview:** `/dev/components` (blocked in production).

---

## 1. Personality

Premium, modern, soft, approachable, operationally sharp. Ambient cool mist → cloud wash behind solid content surfaces. Brand orange as accent (~5%) plus warm peach / soft-orange feature washes where intentional. Borders stay lightweight; card shadows are subtle and realistic. Glass is restrained (shell chrome only) — never heavy blur on tables, lists, or repeated rows.

Keep from the Brex-inspired baseline:

- Clear sidebar IA and dense, readable tables
- Strong typography hierarchy over decoration
- Marketing hero budget: brand + one headline + one line + CTA

### Avoid

- Purple→indigo gradients; cream `#F4F1EA` + terracotta + serif
- Solid-orange sidebars; orange on every icon
- Pill `rounded-full` everywhere; heavy multi-layer shadows; neon glow
- Heavy glass / blur on scroll surfaces, data tables, or list rows
- 3-column icon-in-circle feature grids; big-number KPI heroes on marketing
- Inter-only undifferentiated hierarchy (product face is Plus Jakarta Sans)

---

## 2. Color tokens

| Token | CSS var | Value | Use |
| --- | --- | --- | --- |
| Brand primary | `--brand-primary` | `#FF3D00` | Accent, primary buttons |
| Soft orange | `--brand-soft-orange` | `#FF8050` | Charts, soft accents (`brand-orange`) |
| Warm peach | `--brand-warm-peach` / `--brand-soft` | `#FFE1D5` | Soft highlight / selected |
| Brand hover | `--brand-hover` | `#EB3800` | Hover |
| Brand pressed | `--brand-pressed` | `#D63200` | Active |
| Brand accessible | `--brand-accessible-dark` | `#C83200` | White text on brand; focus rings |
| Brand border | `--brand-border` | `#FFD1BF` | Soft brand outlines |
| Cool mist | `--cool-mist` | `#DCE4F0` | Ambient wash, stronger chrome (`mist`) |
| Cloud white | `--cloud-white` / `--background-subtle` | `#F5F6FA` | Soft panels (`cloud`) |
| White | `--background` / `--white` | `#FFFFFF` | Solid content surfaces |
| Ink | `--foreground` / `--ink` | `#15191E` | Primary text |
| Secondary text | `--foreground-secondary` | `#606975` | Supporting text |
| Foreground muted | `--foreground-muted` | `#8B8D98` | Meta / placeholders |
| Foreground subtle | `--foreground-subtle` | `#B9BBC6` | Disabled hints |
| Foreground inverse | `--foreground-inverse` | `#FCFCFD` | Text on dark |
| Dark | `--dark` | `#000710` | Secondary CTA / charcoal |
| Border | `--border` | `#E4E8EF` | Lightweight default borders |
| Border strong | `--border-strong` | `#C7CAD2` | Emphasized borders |
| Surface | `--surface` | `#F0F3F8` | Nav active, table header |
| Surface strong | `--surface-strong` | `#DCE4F0` | Stronger chrome |

### Gradients

| Name | CSS var / Tailwind | Value | Use |
| --- | --- | --- | --- |
| Ambient | `--gradient-ambient` / `bg-ambient` | `155deg #DCE4F0 → #F0F3F8 → #FAFAFC` | App canvas / shell |
| Warm feature | `--gradient-warm` / `bg-warm-feature` | `140deg #FFFFFF → #FFF6F1 → #FFD1BF` | Feature panels, Connie |
| Brand | `--gradient-brand` / `bg-brand-gradient` | `135deg #FF3D00 → #FF8050` | Primary brand washes / CTAs |
| Chart soft | `--gradient-chart` / `bg-chart-soft` | `180deg #FF8050 → #FFD1BF` | Chart fills |

### Semantic (never use brand orange for warning/error)

| Tone | Text | Background | Border |
| --- | --- | --- | --- |
| Success | `#166534` | `#F0FDF4` | `#BBF7D0` |
| Warning | `#92400E` | `#FFFBEB` | `#FDE68A` |
| Error | `#B91C1C` | `#FEF2F2` | `#FECACA` |
| Info | `#1D4ED8` | `#EFF6FF` | `#BFDBFE` |
| Neutral | `#4B5563` | `#F3F4F6` | `#E5E7EB` |

**A11y:** White small text on `#FF3D00` fails WCAG. Prefer dark text on brand orange, or white on `#C83200`.

Tailwind: `bg-brand`, `bg-brand-orange`, `bg-mist`, `bg-cloud`, `text-ink`, `bg-ambient`, `bg-warm-feature`, etc. via `tailwind.preset.js`.

---

## 3. Typography

| Context | Font | CSS |
| --- | --- | --- |
| Product UI (default) | Plus Jakarta Sans (`next/font` → `--font-jakarta`) | `font-sans` |
| Marketing display | Plus Jakarta Sans | `font-display` |
| Legacy / opt-in Inter | Inter (`--font-inter`) | `font-inter` |

Inter remains loaded so abrupt consumer breakage is avoided; new UI should use the Jakarta stack. Body 400, headings 500, selective 600. Tabular nums for money (`tabular-nums`). Sentence case. Marketing headlines tracking ≈ `-0.02em` to `-0.03em`.

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

- **Spacing base:** 4px. Approved: 4, 8, 12, 16, 20, 24, 32, 40, 48, 64, 80, 96, 128. Prefer Tailwind scale. Content gutter: `--content-gutter`.
- **Radii:** xs 4, sm 6, **md 10** (buttons/inputs), **lg 12**, **xl 16** (cards), **2xl 20**. Controls stay 10–12px; cards 16–20px. **Avoid pills.**
- **Shadows:** `shadow-xs` / `sm` / `md` / `overlay` — subtle realistic elevation on cards; prefer 1px borders for tables/lists.
- **Icons:** Lucide, stroke **1.75**; sizes 16 / 20 / 24 / 32.
- **Motion:** hover 150ms, standard 200ms, dialog 220ms. Respect `prefers-reduced-motion`.

---

## 5. Layout & shell

| Surface | Notes |
| --- | --- |
| App canvas | `bg-ambient` (cool mist wash); content cards stay solid white |
| Sidebar | Soft white / light translucent (`bg-white/80`, light blur); not solid orange |
| Topbar | Matching translucent white; restrained glass only |
| Tables / lists | Solid `bg-background`, high readability — **no** backdrop blur on rows |
| Marketing | max ~1280; reading 680–760 |
| Dashboard | sidebar 248; content max 1440; topbar 56 |
| Auth | form max 420 |
| Checkout | max 1120; two-column desktop |

Validate: **375, 390, 768, 1024, 1440** — no horizontal overflow.

---

## 6. Merchant navigation IA

Grouped sidebar (`nav-config.ts`) — structure unchanged by this visual update:

| Group | Live items | Hidden until built |
| --- | --- | --- |
| **Workspace** | Overview, Notifications, Analytics | — |
| **Operations** | Orders, Customers | Purchasing, Packing, Shipments, Payments/Finance |
| **Configuration** | Catalog, Preorders, Storefront, Business Rules, Settings, Team (when live) | Vendors |

Shell: `AppShell` + `DashboardSidebar` + `MobileNav`. Store context via switcher + `/stores/{storeId}/…`.

---

## 7. Component library

| Component | Path | Notes |
| --- | --- | --- |
| Button | `ui/button.tsx` | `rounded-md` (10px); primary / secondary / outline / ghost / destructive / link |
| Input / Label / IdR | `ui/input`, `label`, `idr-input` | Control radius 10px |
| Card | `ui/card.tsx` | `rounded-xl` (16px), light `shadow-sm`, solid/near-opaque white |
| Badge / StatusBadge | `ui/badge`, `status-badge` | Semantic tones |
| Table / DataTable | `ui/table`, `data-table` | Solid white container; sticky header; no glass rows |
| Empty / Error / Skeleton | `ui/empty-state`, `error-state`, `skeleton` | |
| Dialog / Sheet / Tabs / Select / Toast | corresponding `ui/*` | Sheet uses soft white panel |
| PageHeader / FilterBar | `ui/page-header`, `filter-bar` | |

### Charts

When charts ship: use soft orange fills (`bg-chart-soft` / `--gradient-chart` / `--brand-soft-orange`). Avoid neon series colors and heavy glow.

### Connie (if / when present)

Apply warm-feature treatment (`bg-warm-feature` / `--gradient-warm`) — soft peach wash, not flat charcoal and not neon glass. No Connie surface in the repo at time of this update; follow this when introducing one.

### Data tables

- Header: `bg-surface`, uppercase `text-label`, sticky optional
- Rows: 1px `border-border`; hover `bg-background-subtle`; solid backgrounds only
- Money / counts: `tabular-nums`, right-aligned
- Empty → `EmptyState`; loading → `Skeleton`; failure → `ErrorState`

---

## 8. Storefront themes

24 curated themes in Go `internal/themes`. Merchant theme colors **never** affect backoffice.

---

## 9. Usage rules

1. Import from `@/components/ui/*` and `@/components/dashboard/*` — extend tokens, do not invent parallel palettes.
2. Prefer token-driven visual changes; do **not** rebuild pages or change API/DB/business logic for design direction.
3. Preserve nav hierarchy and operational density.
4. Marketing: keep hero budget tight; no AI-generic feature grids.
5. Glass only on shell chrome (sidebar/topbar); never on tables/lists.

---

## 10. Acceptance (soft-gradient update)

- [x] Tokens + Tailwind preset encode palette, gradients, radii, fonts
- [x] Global CSS + root layout load Plus Jakarta Sans; Inter remains available
- [x] Shared shell/card/table updated; pages not rebuilt
- [x] This doc reflects soft-gradient direction
- [x] No API/DB/business-logic changes

See also Project `preferences.md` (design direction) and agent handoffs under the Project store `internal/`.
