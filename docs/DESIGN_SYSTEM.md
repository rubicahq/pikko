# Design System

Source: PRD §13–14. Tokens live in `packages/design-tokens/`.

## Personality

Premium, modern, minimal, approachable. Strong typography over decoration. White/cool neutrals with restrained orange. Borders define boundaries; shadows are minimal.

## Typography

- **Product UI font:** Inter via `next/font` (`--font-inter`), fallback Arial.
- **Marketing display font:** Plus Jakarta Sans (`--font-jakarta` / `font-display`) for headlines and brand wordmark on marketing surfaces — not Inter-only.
- Body 400, headings 500. Marketing headlines use tracking about `-0.02em` to `-0.03em`.
- Avoid AI-default patterns on marketing: no big-number heroes, no 3-column icon-in-circle grids, no purple–blue gradients, no cream+terracotta serif look.

## Color tokens

| Token | Value |
| --- | --- |
| `--brand-primary` | `#FF3D00` |
| `--brand-hover` | `#EB3800` |
| `--brand-pressed` | `#D63200` |
| `--brand-accessible-dark` | `#C83200` |
| `--brand-soft` | `#FFF1EB` |
| `--background` | `#FFFFFF` |
| `--background-subtle` | `#F9F9FB` |
| `--surface` | `#F3F3F7` |
| `--foreground` | `#15191E` |
| `--foreground-secondary` | `#60646C` |
| `--border` | `#E1E3E8` |
| `--dark` | `#000710` |

Semantic success/warning/error/info colors match the PRD. Do not use brand orange for warning/error.

Accessibility: white text on `#FF3D00` fails for small text — primary buttons use dark text on brand orange, or white on `#C83200` for high-contrast CTAs (emails use accessible dark orange).

## Spacing & radius

- Base 4px. Approved: 4, 8, 12, 16, 20, 24, 32, 40, 48, 64, 80, 96, 128.
- Radii: xs 4, sm 6, md 8 (buttons/inputs), lg 10 (cards), xl 12, 2xl 16.
- Avoid pill-shaped controls.

## Components

Implemented in `apps/web/src/components/ui`: Button, Input, Label, Card, Badge.

Dev-only preview: `/dev/components` (blocked in production by page guard + middleware).

## Icons

Lucide, stroke 1.75. Sizes 16 / 20 / 24.

## Breakpoints to validate

375, 390, 768, 1024, 1440 — no horizontal overflow.
