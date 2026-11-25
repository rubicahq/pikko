# Design System — Pikko (Friendly Commerce Workspace)

**Owner:** Design / frontend agents  
**Direction:** Buy Me a Coffee dashboard simplicity with Pikko’s orange palette — complex operations, simple experience.  
**Authority:** Project `docs/pikko-master-design-system.md` + `preferences.md` Design Authority.  
**Supersedes:** Brex-heavy, Roomlio, corporate SaaS, glassmorphism, bento, and soft-gradient-as-foundation (PR #17 ambient wash everywhere). Gradients are **optional accents only**.  
**Tokens:** `packages/design-tokens/` (`tokens.css`, `design-tokens.css`, `tailwind.preset.js`).  
**Components:** `apps/web/src/components/ui` + `apps/web/src/components/dashboard`.  
**Dev preview:** `/dev/components` (blocked in production).

This is a visual/UX redesign of the existing application — **not** a rebuild of business functionality, APIs, DB, auth, or RBAC.

---

## 1. Design Authority

This document is the definitive in-repo visual specification for Pikko. When prior docs conflict (Brex nav chrome, glass shell, soft-gradient foundation), **this document wins**.

Primary visual reference: Buy Me a Coffee merchant dashboard (simplicity, friendly personality, nav treatment, hierarchy, spacing, component styling). **Do not** adopt BMC yellow. Preserve Pikko orange.

---

## 2. Product Context

Pikko is a commerce and operations platform for independent sellers (jastipers, preorder sellers, online merchants). The product is operationally deep; the UI must feel light and approachable.

---

## 3. Visual Personality

Friendly, modern, lightweight, organized, approachable, trustworthy, slightly playful, practical.

### Avoid

- Corporate enterprise / generic admin templates
- Excessive card grids and four-KPI Overview dashboards
- Heavy borders, dense tiny type, decorative illustrations on ops pages
- Unnecessary glass / backdrop blur on shell or data surfaces
- Heavy shadows; soft-gradient-everywhere; futuristic AI aesthetics
- Purple→indigo gradients; cream `#F4F1EA` + terracotta + serif; Inter-only hierarchy
- Solid-orange sidebars; orange on every icon; pill `rounded-full` everywhere

---

## 4. Color System

| Token | CSS var | Value | Use |
| --- | --- | --- | --- |
| Brand Orange | `--brand-primary` | `#FF3D00` | Selective accent / brand button |
| Soft Orange | `--brand-soft-orange` | `#FF8050` | Soft accents, charts |
| Warm Peach | `--brand-warm-peach` / `--brand-soft` | `#FFE1D5` | Connie / soft highlight |
| Cool Mist | `--cool-mist` | `#DCE4F0` | Subtle ambient tip only |
| Cloud White | `--cloud-white` / `--workspace-bg` | `#F5F6FA` | Workspace canvas |
| Ink Black | `--ink` / `--foreground` / `--dark` | `#15191E` | Text + **primary buttons** |
| Pure White | `--background` / `--white` | `#FFFFFF` | Sidebar, cards, tables |

### Neutrals

| Role | Value |
| --- | --- |
| Text primary | `#15191E` |
| Text secondary | `#626975` |
| Text muted | `#747B86` |
| Border default | `#E5E7EB` |
| Border subtle | `#EEF0F3` |
| Surface hover | `#F3F4F6` |
| Surface active | `#F0F1F3` |
| Surface disabled | `#F3F4F6` |

### Status

| Tone | Text | Background |
| --- | --- | --- |
| Success | `#166534` | `#EAF7EE` |
| Warning | `#92400E` | `#FFF4DB` |
| Error | `#B42318` | `#FDECEC` |
| Information | `#1D4ED8` | `#EDF4FF` |
| Neutral | `#4B5563` | `#F1F2F4` |

Never use brand orange for warning/error. Prefer dark text on brand orange for small type, or white on `#C83200` when needed for contrast.

---

## 5. Color Distribution

Approximate page mix: **~45% white, 30% cloud white, 10% ink, 7% cool mist, 4% brand orange, 3% warm peach, 1% soft orange.** Orange is selective, not everywhere.

---

## 6. Gradient System

Gradients are **optional accents**, not the workspace foundation.

| Name | CSS / Tailwind | Value | Allowed use |
| --- | --- | --- | --- |
| Ambient | `--gradient-ambient` / `bg-ambient` | `160deg #F5F6FA → #F5F6FA 70% → #EAF0F7` | Subtle canvas tip only |
| Connie / warm | `--gradient-warm` / `bg-warm-feature` | `135deg #FFFFFF → #FFF7F3 → #FFE1D5` | Connie, onboarding accents, empty-state art, announcements |
| Brand | `--gradient-brand` / `bg-brand-gradient` | `135deg #FF3D00 → #FF8050` | Rare promotional washes |
| Chart soft | `--gradient-chart` / `bg-chart-soft` | Soft orange fill | Charts when present |

**Forbidden on:** tables, standard forms, sidebar, every card, badges, dropdowns.

Default workspace background: solid Cloud White (`bg-workspace` / `--workspace-bg`).

---

## 7. Typography

**Plus Jakarta Sans** for product UI and marketing display (`next/font` → `--font-jakarta`, `font-sans` / `font-display`). Inter may remain loaded (`font-inter`) for gradual migration; new UI uses Jakarta.

| Role | Size / weight | Notes |
| --- | --- | --- |
| Page title | 26 / 700 | `text-page` |
| Greeting | 24 / 700 | Overview |
| Section | 18 / 700 | `text-section` |
| Card title | 16 / 600 | `text-card` |
| Primary metric | 32 / 700 | `text-metric` |
| Secondary metric | 24 / 600 | `text-metric-secondary` |
| Body / nav / button | 14 / 400–600 | Nav 500, button 600 |
| Table | 13 / 400 | `text-table`, `tabular-nums` for money |
| Supporting | 12 / 400 | Labels / meta |
| Nav group | 10 / 600 | Uppercase sparingly |

Sentence case. Marketing display tracking ≈ `-0.02em` to `-0.03em`.

---

## 8. Spacing

4px base. Approved: 4, 8, 12, 16, 20, 24, 32, 40, 48, 64 (+ extended scale as needed). Prefer Tailwind scale — no arbitrary `mt-[13px]`.

| Context | Value |
| --- | --- |
| Card padding | 24 (compact 16) |
| Section gap | 24 |
| Desktop workspace padding | 32 |
| Mobile padding | 16 |

---

## 9. Border Radius

| Element | Radius |
| --- | --- |
| Badges | 6 (`rounded-sm`) |
| Inputs / buttons / nav items | 10 (`rounded-md`) |
| Dropdowns | 12 (`rounded-lg`) |
| Cards | 14 (`rounded-xl`) |
| Large / feature / dialogs | 16 (`rounded-2xl`) |
| Avatars | circular |

Compact dashboard actions **may** be pill; not everything is pill.

---

## 10. Elevation

- Cards: white + 1px subtle border; optional `shadow-sm` (`0 1px 3px rgba(21,25,30,0.04)`)
- Dropdown: `shadow-md`
- Modal: `shadow-overlay`
- Prefer borders and whitespace over stacked shadows. No multi-layer neon glow.

---

## 11. Desktop Shell

| Surface | Spec |
| --- | --- |
| Sidebar | **224px** (collapsed **64px** on tablet), solid white, fixed, horizontal pad 12, subtle right border `#E5E7EB` — **no glass** |
| Workspace | Cloud White `#F5F6FA`, padding 32 |
| Overview content | max width **1120px** (`max-w-overview`) |
| Ops pages | full dashboard width; tables stay dense |
| Top bar | **60px**, solid white, sticky; contextual tabs left + Messages/Notifications right — single row, no second header |

Validate: **375, 390, 768, 1024, 1440** — no horizontal overflow.

---

## 12. Hybrid Navigation

**Sidebar for primary activities. Horizontal tabs for contextual features.**

SSOT: `apps/web/src/components/dashboard/nav-config.ts` (`PRIMARY_WORKSPACES`, `SETTINGS_WORKSPACE`, match helpers).

### Sidebar (top → bottom)

Logo → store switcher → View store → divider → **7 primary items** → flex space → Settings → My Account dropdown.

No Workspace / Operations / Relationships / Manage group labels. No accordion / expandable submenus.

| Primary | Default route | Contextual tabs (implemented only) |
| --- | --- | --- |
| Overview | `/overview` | Summary, Analytics |
| Orders | `/orders` | All Orders, Preorders, Live Shopping, Payments |
| Fulfillment | `/purchasing` | Purchasing, Receiving (`?tab=receiving`), Packing, Shipments, Vendors |
| Products | `/products` | All Products, Catalogs · Attributes\* |
| Campaigns | `/campaigns` | All / Active (`?status=active`) / Completed (`?status=completed`) |
| Customers | `/customers` | All Customers · Store Credit\* |
| Storefront | `/storefront` | Appearance, Content (`?tab=content`), Preview (`?tab=preview`) |

\*Hidden until built (`implemented: false`).

**Settings** (utility): General, Team, Business Rules, Payment Settings, WhatsApp Templates (`/messages`).  
**Header utilities (right):** **Ask Connie** (compact peach chip + Soon — no AI backend), Messages, Notifications (Search omitted until implemented).

Nav items: height 40px, gap 4, radius 10, icons 20px stroke ~1.75, label 14/500.  
**Active:** `#F0F1F3` bg, `#15191E` text (600), icon `#FF3D00` — **not** a solid orange row.  
**Tabs:** 14px, active 600 + 2px `#FF3D00` bottom indicator; inactive `#626975`.

**Responsive:** desktop full sidebar; tablet collapsed icon rail + tooltips; mobile drawer; tabs scroll horizontally, single row.

---

## 13. Iconography

Lucide (or consistent rounded outline set). Stroke ~1.75. Selected nav may use orange accent. No emoji / mixed / 3D nav icons.

---

## 14. Overview Dashboard

**Vertical hierarchy benchmark** — not a four-KPI card grid.

| Block | Content |
| --- | --- |
| **A. Store summary** | One card: greeting, store identity, share/view actions, divider, **sales dominant** + supporting metrics from real data |
| **B. Needs attention** | Actionable list (setup gaps, payment reviews) |
| **C. Recent orders** | Dense rows linking to order detail |
| **D. Connie** | Compact warm-gradient entry; no fake insights; no AI backend required |

Never fabricate production metrics.

---

## 15. Page Header

Title + optional one-line description + primary/secondary actions. No banners or heroes on ops pages. Use `PageHeader`.

---

## 16. Buttons

| Variant | Treatment |
| --- | --- |
| **Primary** | Ink `#15191E` bg, white text, height 40, radius 10, weight 600 |
| **Secondary / outline** | White + border |
| **Brand** | `#FF3D00` — selective emphasis only |
| **Ghost** | Text + hover surface |
| **Destructive** | Error red |

Include hover, active, focus, disabled, loading states. Focus: subtle brand ring, no glow.

---

## 17. Forms

Input height **42**, radius 10, labels 13/600 above. Focus: subtle orange ring. Disabled uses surface-disabled. Prefer shared `Input` / `Label` / IDR helpers.

---

## 18. Data Tables

Dense and practical:

- White container, radius 14, light border
- Header ~44px; row 52px (compact 40)
- Subtle horizontal dividers; no heavy vertical grid
- Money right-aligned, `tabular-nums`
- Keep sort / filter / search / pagination / bulk
- **Do not** cardify high-volume tables on desktop

---

## 19. Status Badges

11–12px type; semantic tones from §4. Use `Badge` / `StatusBadge`. Never brand-orange for error/warning.

---

## 20. Connie

**Ask Connie** lives in the global header top-right (with Messages / Notifications). Overview may still show a warm panel. Visual only until an AI backend exists — no fabricated insights, no new AI API in this redesign. Not in the sidebar.

---

## 21. Empty States

Clear title, one supporting sentence, **one** primary CTA. Prefer `EmptyState`. Soft warm wash allowed for illustrated empties; not required on every empty.

---

## 22. Loading

Skeletons (`Skeleton` / `TableSkeleton`) matching final layout density. Respect `prefers-reduced-motion`.

---

## 23. Responsive

Mobile: drawer nav (`MobileNav` + `Sheet`), contextual tabs in sticky header (horizontal scroll), Messages/Notifications in header. Tablet: 64px collapsed sidebar. Desktop: 224px sidebar + cloud workspace.

---

## 24. Internationalization

Indonesian merchants first. Copy is practical Bahasa/English-friendly sentence case. Currency IDR with thousand separators (`formatIDR`).

---

## 25. Accessibility

Target WCAG 2.2 AA. Visible focus rings; don’t rely on color alone for status; maintain contrast on brand orange; keyboard reach for nav, dialogs, tables.

---

## 26. Implementation Principles

1. Central tokens + Tailwind preset — extend tokens, don’t invent parallel palettes.
2. Shared components in `ui/*` and `dashboard/*`.
3. Preserve API / DB / auth / RBAC / workflows.
4. Hide unimplemented nav via `implemented: false`.
5. Prefer token-driven visual changes; redesign Overview composition; keep ops tables dense.

---

## 27. Rollout Steps

| Step | Scope |
| --- | --- |
| **1** | Tokens, typography (Jakarta), shell/sidebar, core components (buttons Ink primary, inputs 10px, cards 14px) |
| **2** | Overview vertical hierarchy |
| **3** | Language across operational pages (headers, dense tables) |

---

## 28. Testing Guidance

Prefer typecheck, lint, build, and targeted unit/API tests. No full Playwright suite required for visual redesign. Reserve Playwright for critical journeys at stage gates.

---

## 29. Acceptance

- [x] Visibly BMC-simple with Pikko orange — not a mere recolor of soft-gradient shell
- [x] White 224px sidebar (64px collapsed); Cloud White workspace; no glass foundation
- [x] Hybrid nav: 7 primary workspaces + contextual header tabs
- [x] Primary buttons Ink Black; brand orange selective
- [x] Overview is vertical hierarchy (not four KPI cards)
- [x] Ask Connie is compact header utility (Coming Soon) — not in sidebar
- [x] Unimplemented nav remains hidden
- [x] No API/DB/auth/RBAC/workflow regressions
- [x] `docs/DESIGN_SYSTEM.md` matches this authority

---

## 30. Doc Ownership

Update this file when visual rules change. Project store master: `pikko-master-design-system.md`. Handoffs under Project `internal/`. Soft-gradient PR #17 notes are historical; do not re-apply ambient gradient as the default shell foundation.
