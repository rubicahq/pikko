# Agent 03 → Agent 04 Handoff (Auth & Multi-Store → Campaign & Catalog)

**Date:** 2026-11-25 (Asia/Jakarta)  
**PRD:** v3.0 (`docs/PIKKO_MASTER_PRD.md`)  
**Branch:** `cursor/agent-03-auth-multistore-f4b8`  
**Prior:** Agent 02 at `54181f59ddf2492ac5882691118c3ccdf984525f`

---

## 1. What Agent 03 delivered

| Area | Outcome |
| --- | --- |
| Migration `000007_team_rbac` | Widened `store_memberships.role` CHECK; added `team_invites` |
| `internal/rbac` | Default PRD roles + granular permissions; owner is superset |
| `internal/team` | Invite / resend / cancel / accept / revoke / role patch + Brevo email |
| HTTP | Permission-gated store routes; public invite peek/accept; team CRUD |
| UI | `/stores/{id}/team`, `/invites/accept`, role-aware nav (NAV-03), switcher role labels |
| Onboarding | Setup checklist now reflects real payment methods + fulfillment |
| Tests | `rbac` unit, `team` + stores isolation integration, `e2e/team-rbac.spec.ts` |
| Docs | Matrix / progress / schema / API contracts / this handoff |

**Preserved:** Magic-link auth, sessions, owner multi-store, Brevo delivery, tenant isolation patterns.  
**No pikko.id cutover. No destructive migrations.**

---

## 2. Acceptance gate evidence

Per owner testing policy (speed over full Playwright loops):

- `go test` on `rbac`, `team`, `stores`, `auth`, `httpapi` (invite + tenant isolation)
- `tsc --noEmit`, eslint, `next build`
- Critical journey coverage via Go integration tests (invite accept, Store A staff denied Store B, role permission 403s)
- `e2e/team-rbac.spec.ts` preserved for stage smoke later — not required to block this handover

---

## 3. How later agents use RBAC

1. Gate new routes with `storePerm(rbac.PermX, handler)` in `router.go`.  
2. Add permissions to `internal/rbac` role matrix before shipping UI.  
3. Flip `implemented: true` in `nav-config.ts`; nav auto-filters by permission.  
4. Invite roles exclude `owner`; archive remains owner-only.

---

## 4. Known limitations

| Item | Note |
| --- | --- |
| Permission overrides table | Deferred — role matrix sufficient |
| Ownership transfer | Not implemented (admin cannot become owner via invite) |
| Agent 10 WA polish | Invite email is Brevo HTML; WA templates later |
| Self-leave / last-owner guard | Revoke blocks owner + self; no leave-own-membership UX |

---

## 5. Next — Agent 04: Campaign & Catalog

Start with:

- Source of truth: `docs/PIKKO_MASTER_PRD.md` campaigns/catalog sections  
- Prior handoff: this file  
- Preserve auth, RBAC middleware, and Agent 02 design shell  
- Do not rebuild invites; do not destructive-migrate; do not cut over pikko.id

Acceptance reminder: preorder flows stay green; additive catalog gaps only.
