# Agent 15 Handoff — Integration & Deploy Hardening (PRD v3 final)

**Date:** 2026-11-25 (Asia/Jakarta)  
**PRD:** v3.0 (`docs/PIKKO_MASTER_PRD.md` §§29, 32)  
**Branch:** `cursor/agent-15-deploy-hardening-c5a2`  
**Prior:** Agent 14 Security feature merge `db6c97f` (PR #35); handoff SHA tip `58f0927` (PR #36)  
**Feature merge SHA:** _(filled after merge)_  
**Merged main SHA:** _(filled after merge)_

---

## 1. What Agent 15 delivered

| Area | Outcome |
| --- | --- |
| `scripts/deploy.sh` | BuildKit; selective parallel rebuild; postgres healthy wait; backup-on-migrate (+ `--backup`) under `~/backups/pikko/deploy/` with sha256; migrate image rebuild with api; `pre-<stamp>` rollback; port + `healthz` gates; env-driven public smoke; never touches Emplicon project |
| `scripts/smoke.sh` | Loopback + `readyz`; public URLs **env-only** (E2E-N); defaults = `pikko.site`; no `pikko.id` hardcode/cutover |
| Compose | Web container healthcheck (node fetch); api health remains host-side (distroless); `pikko-postgres` hostname preserved |
| Runbook | `docs/DEPLOYMENT_RUNBOOK.md` owner-ready: SSH `-p 2021`, migration table through **`000016`**, backup/rollback, incident notes |
| Related docs | `LAUNCH.md` / `DEPLOYMENT.md` / `RUNBOOK.md` cross-links; `.env.example` smoke knobs |
| Progress / matrix | Agents 13–15 marked done; Stage 3 gaps closed; **no auto pikko.id** |
| Migration | **None** (docs only; head remains `000016`) |
| Deploy to VPS | **Not required** from this agent — scripts/docs only; operator may `--dry-run` then deploy |

**Preserved:** Agent 14 security fixes, analytics, Emplicon coexistence, domain-agnostic config.  
**No destructive migrations. No pikko.id DNS/env cutover.**

---

## 2. Acceptance gate evidence

- `bash -n scripts/deploy.sh scripts/smoke.sh`
- `./scripts/deploy.sh --dry-run --no-pull` (plan only)
- Docs consistency: migration head `000016` in runbook matches `database/migrations/`
- Merge without empty CI wait (repo has no blocking Actions)

---

## 3. How the owner should use this

1. Day-2 entrypoint: **`docs/DEPLOYMENT_RUNBOOK.md`**.  
2. Deploy: `ssh -p 2021 …` → `cd ~/pikko` → `./scripts/deploy.sh` (add `--public-smoke` when validating edge).  
3. Cutover remains **`docs/LAUNCH.md`** only after written approval.  
4. Keep nginx `X-Real-IP`; do not strip Agent 14 rate-limit trust.

---

## 4. Residual / out of scope

| Item | Note |
| --- | --- |
| VPS live deploy from this PR | Optional operator action — not gated on Agent 15 |
| pikko.id cutover | Owner only |
| SEC-06 PDPA | Owner/counsel (from Agent 14) |
| Down migrations on prod | Forbidden |

---

## 5. PRD v3 serial plan

**Agents 01–15 complete.** Production continues on `pikko.site` until the owner runs LAUNCH.md.
