# Agent 12 → Agent 13 Handoff (Live Shopping → Analytics)

**Date:** 2026-11-25 (Asia/Jakarta)  
**PRD:** v3.0 (`docs/PIKKO_MASTER_PRD.md`)  
**Branch:** `cursor/agent-12-live-shopping-5fd1`  
**Prior:** Agent 11 feature merge `6a51005` (PDF hardening); main tip includes handoff PR #30  
**Feature merge SHA:** `b2a930782ca0a55ad47afe00b2282c70985fc380` (PR #31)  
**Merged main SHA:** `b2a930782ca0a55ad47afe00b2282c70985fc380`

---

## 1. What Agent 12 delivered

| Area | Outcome |
| --- | --- |
| Migration `000016_live_shopping` | `campaign_products.featured`; `orders.source=live_quick_add`; revision/event types; `live_quick_adds` audit |
| LIVE-01 | Timed live sessions on `campaign_type=live`; featured/quick-add SKUs; merchant Live console |
| LIVE-02 | `POST …/live/{id}/quick-add` — add to existing order or create; idempotent; CS/owner via `orders.write` |
| LIVE-03 | Order feed + sold/remaining quota under `FOR UPDATE`; concurrent quota test |
| Storefront | Public `/live` list + `/{slug}/live/{liveSlug}` using storefront theme colors |
| Nav / RBAC | Operations → Live (`campaign.read`); quick-add needs `orders.write` |
| Tests | `TestLiveShoppingQuickAddFeedAndQuota`; rbac nav `live` |
| Docs | Matrix / progress / schema / contracts / this handoff |

**Preserved:** Campaigns/preorders, checkout, PDF/flipbook, Stage 2 ops.  
**No video infrastructure. No pikko.id cutover. No destructive migrations.**

---

## 2. Acceptance gate evidence

Per owner testing policy:

- `go test ./internal/commerce/ -run TestLiveShopping`
- `go test ./internal/campaigns/ ./internal/rbac/`
- `tsc --noEmit` on web
- No Playwright required for this handover
- Merge without empty CI wait

---

## 3. How later agents use live shopping

1. Prefer `/stores/{id}/live/{campaignId}` console for ops; campaign detail still assigns products.  
2. Quick-add stamps `orders.campaign_id` + `source=live_quick_add`.  
3. Quota = `campaign_products.quantity_limit` vs SUM of non-cancelled order item qty for that campaign product.  
4. Public discoverability: `GET /public/stores/{slug}/live`.  
5. Analytics (Agent 13) can attribute GMV via `campaign_id` / live source.

---

## 4. Known limitations

| Item | Note |
| --- | --- |
| Server-side customer carts | Not used; local cart + orders / quick-add |
| Realtime (websocket) | Polling ~4s on console feed |
| Playwright E2E-K UI | Deferred to stage consolidation smoke |
| Closing session | Sets operational_status / closes_at via campaign PATCH |

---

## 5. Next — Agent 13: Analytics

Start with:

- Source of truth: `docs/PIKKO_MASTER_PRD.md` AN-*  
- Prior handoff: this file  
- Preserve live/campaign attribution fields for north-star metrics  
- Do not regress live quick-add or public storefront themes
