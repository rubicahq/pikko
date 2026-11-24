# Agent 04 → Agent 05 Handoff (Campaign & Catalog → Order Management)

**Date:** 2026-11-25 (Asia/Jakarta)  
**PRD:** v3.0 (`docs/PIKKO_MASTER_PRD.md`)  
**Branch:** `cursor/agent-04-campaign-catalog-5f34`  
**Prior:** Agent 03 at `f610203b59ae803d3041dd25fcb5b165ab5ac059` (includes PR #12 + e2e hardening)  
**Merged main SHA:** `8b8f7c9bb1a924e7c3f7f8b874bb2d2234b61d3b`  
**PR:** https://github.com/rubicahq/pikko/pull/13

---

## 1. What Agent 04 delivered

| Area | Outcome |
| --- | --- |
| Migration `000008_campaign_catalog` | `brands`, `vendors`, `product_vendors`, `attribute_definitions`, `product_attribute_values`, `campaigns`, `campaign_products`; `products.brand_id`; `preorder_batches.campaign_id` + backfill |
| `internal/campaigns` | Unified CRUD for `preorder` + `live`; product assign with price/availability/phase overrides; sync helpers for legacy preorder path |
| Catalog foundations | Brands / vendors / attribute APIs on `catalog` + HTTP |
| RBAC | `campaign.read` / `campaign.write`; preorder + campaigns routes gated; nav maps updated |
| UI | `/stores/{id}/campaigns` list/create/detail; Campaigns enabled in nav; Preorders kept |
| Tests | `campaigns` integration, `rbac` unit; existing catalog/commerce/team/auth green |
| Docs | Matrix / progress / schema / architecture / API contracts / this handoff |

**Preserved:** Preorder batches, purchasing/allocations, PDF import, flipbook, collections, storefront publish.  
**No pikko.id cutover. No destructive migrations.**

---

## 2. Acceptance gate evidence

Per owner testing policy:

- `go test` on `rbac`, `campaigns`, `catalog`, `commerce`, `team`, `auth`, `httpapi`
- `tsc --noEmit`, eslint, `next build`
- No full Playwright suite required for this handover

---

## 3. How later agents use campaigns / catalog

1. Prefer `/campaigns` for new selling-window work; `/preorders` remains and stays synced.  
2. Gate with `storePerm(rbac.PermCampaignRead|Write, …)` or `PermCatalog*`.  
3. Live shopping ops (quick-add to carts, order feed concurrency) → Agent 12 on `campaign_type=live`.  
4. Vendor purchasing / receiving → Agent 06 (entities already exist).  
5. Flip `implemented: true` only when shipping new nav modules.

---

## 4. Known limitations

| Item | Note |
| --- | --- |
| Live quick-add / feed | Flags only; no checkout integration yet |
| Brand text vs `brand_id` | Both kept; UI product form not fully switched to brand picker |
| Vendor merchant nav | Still `implemented:false` until Agent 06 |
| Collections UI depth | API create/list only (unchanged from prior) |

---

## 5. Next — Agent 05: Order Management

Start with:

- Source of truth: `docs/PIKKO_MASTER_PRD.md` §11 ORD-*  
- Prior handoff: this file  
- Preserve checkout, payments, campaign/preorder links on orders  
- Additive item ops / admin entry / revisions only

Acceptance reminder: checkout + payment verify stay green; no break of campaign-linked preorder orders.
