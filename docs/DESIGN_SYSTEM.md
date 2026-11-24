# Design System — Pikko (PRD v3.0)

**Status:** Foundation live; **Agent 02 owns full expansion** against Brex reference + this stub.  
**Visual reference:** https://www.brex.com/ (character/layout/discipline only — no proprietary assets, fonts, or copy).  
**Tokens source of truth:** `packages/design-tokens/` (`design-tokens.css`, `tokens.css`, `tailwind.preset.js`).

Agent 01 (Lead Architect) audited existing tokens and UI. Do **not** invent a parallel palette. Extend the approved Brex-inspired Pikko tokens below.

---

## Personality

Premium, modern, minimal, approachable, confident. Strong typography over decoration. White/cool neutrals with **restrained** orange (~5% accent). Borders define boundaries; shadows are minimal.

### Avoid (AI-generic)

- Purple→indigo gradients; cream `#F4F1EA` + terracotta + serif
- Solid-orange sidebars; orange on every icon
- Pill `rounded-full` everywhere; heavy multi-layer shadows; glow/glass
- 3-column icon-in-circle feature grids; big-number KPI heroes on marketing
- Inter-only undifferentiated hierarchy (use scale/weight; marketing may use Plus Jakarta Sans for display)

---

## Brand & color tokens (implemented)

| Token | Value |
| --- | --- |
| `--brand-primary` | `#FF3D00` |
| `--brand-hover` | `#EB3800` |
| `--brand-pressed` | `#D63200` |
| `--brand-accessible-dark` | `#C83200` |
| `--brand-soft` | `#FFF1EB` |
| `--brand-border` | `#FFD2C2` |
| `--background` | `#FFFFFF` |
| `--background-subtle` | `#F9F9FB` |
| `--surface` | `#F3F3F7` |
| `--surface-strong` | `#E8E9EE` |
| `--foreground` | `#15191E` |
| `--foreground-secondary` | `#60646C` |
| `--foreground-muted` | `#8B8D98` |
| `--foreground-inverse` | `#FCFCFD` |
| `--dark` | `#000710` |
| `--border` | `#E1E3E8` |
| `--border-strong` | `#C7CAD2` |

Semantic: success / warning / error / info — see token CSS; **do not** use brand orange for warning/error.

**A11y:** White small text on `#FF3D00` fails WCAG. Primary buttons: dark text on brand orange, or white on `#C83200`.

Distribution guideline: white/light ~75%, charcoal ~15%, secondary neutral ~5%, orange ~5%.

---

## Typography

| Context | Font |
| --- | --- |
| Product UI | Inter via `next/font` (`--font-inter`), fallback Arial |
| Marketing display | Plus Jakarta Sans (`--font-jakarta` / `font-display`) for headlines/wordmark |

PRD scales (marketing Display XL→Label; dashboard page/section/card/body/table) — **Agent 02 must document complete type ramp in this file** and wire Tailwind utilities. Body 400, headings 500, selective 600. Tabular nums for money. Sentence case.

---

## Spacing, radius, shadow, icons

- Spacing base 4px; approved: 4, 8, 12, 16, 20, 24, 32, 40, 48, 64, 80, 96, 128
- Radii: xs 4 → 2xl 16; buttons/inputs 8; cards 10; avoid pills
- Shadows: xs/sm/md/overlay per prior PRD (subtle only)
- Icons: Lucide stroke **1.75**; sizes 16 / 20 / 24 / 32+

---

## Components (current vs Agent 02)

**Implemented** in `apps/web/src/components/ui`: Button, Input, Label, Card, Badge, IdR input.

**Dev preview:** `/dev/components` (blocked in production).

**Agent 02 must add/standardize:** tables (dense data), dialogs/drawers, tabs/filters, empty states, skeletons, toasts, status badges (Paid/Awaiting/Confirmed/Draft/Open/Closed/Cancelled/Failed), nav sidebar patterns (Workspace / Operations / Configuration IA per PRD §6), form patterns, storefront-safe primitives.

Stack: Tailwind + Radix + shadcn-where-appropriate + Lucide — customize; no default shadcn look shipped raw.

---

## Storefront themes

24 curated themes in Go `internal/themes` (Pikko Signature → Contemporary). Merchant theme colors never affect backoffice. Agent 02 validates contrast helpers remain correct.

---

## Layout & breakpoints

| Surface | Notes |
| --- | --- |
| Marketing | max ~1280; reading 680–760 |
| Dashboard | sidebar ~248 (collapsed 64); content max ~1440 |
| Auth | form max ~420 |
| Checkout | max ~1120; two-column desktop |

Validate: 375, 390, 768, 1024, 1440 — no horizontal overflow. Motion: short, subtle; respect `prefers-reduced-motion`.

---

## Agent 02 acceptance (handoff target)

1. This file expanded to full token tables, type scale, component specs, and usage rules
2. Gaps in `components/ui` closed or ticketed with owners
3. Marketing + dashboard samples match Brex discipline without copying Brex
4. Requirements `DS-*` updated in `REQUIREMENTS_MATRIX.md`
5. No destructive redesign of working storefront checkout flows

See `docs/agents/01-lead-architect-handoff.md`.
