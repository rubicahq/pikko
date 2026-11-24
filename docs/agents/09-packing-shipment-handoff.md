# Agent 09 → Agent 10 Handoff (Packing/Shipment → Communication)

**Date:** 2026-11-25 (Asia/Jakarta)  
**PRD:** v3.0 (`docs/PIKKO_MASTER_PRD.md`)  
**Branch:** `cursor/agent-09-packing-shipment-c907`  
**Prior:** Agent 08 main tip `ca6e528` (portal PR #23/`bd002b7` + handoff `#24`)  
**Feature merge SHA:** _(filled after merge)_  
**Merged main SHA:** _(filled after merge)_  

---

## 1. What Agent 09 delivered

| Area | Outcome |
| --- | --- |
| Migration `000013_packing_shipment` | `packing_tasks` / `packing_task_items`; `shipments` / `shipment_items` |
| PACK-01 | Queue by order or customer consolidation (`group_by`) |
| PACK-02 | Checklist gated on available/allocated qty (preorder uses `allocated_qty`) |
| PACK-03 | Mark packed → `ready_to_ship` / `ready_for_pickup`; updates `order_items.packed_qty` |
| SHIP-01 | Create shipment with carrier + tracking number |
| SHIP-02 | `order_shipped_customer` via `email_jobs`; `shipment.created` / `packing.completed` outbox |
| SHIP-03 | Multi-parcel / partial ship; fulfillment → `shipped` when all packed qty shipped |
| RBAC / nav | `packing.read` / `packing.write`; Operations → Packing + Shipments |
| Portal / track | Additive `shipments` on portal order detail + guest tracking |
| Preserved | Portal auth, guest tracking, existing emails, finance, purchasing |

**No pikko.id cutover. No destructive migrations.**

---

## 2. Acceptance gate evidence

Per owner testing policy:

- `go test ./internal/commerce/ ./internal/portal/ ./internal/rbac/ ./internal/money/` (E2E-I packing/shipment + tenant isolation)
- `tsc --noEmit` on web
- No full Playwright suite required for this handover
- Merge without empty CI wait

---

## 3. How later agents use packing/shipments

1. Read shipments via `GET /stores/{id}/orders/{orderId}/shipments` or include from tracking/portal — do not invent parallel tracking tables.  
2. Pack only through packing APIs so `packed_qty` and packing_tasks stay consistent.  
3. Pickup orders use `ready_for_pickup` / collect — carrier shipment API returns conflict for pickup.  
4. Extend customer notifications via WhatsApp (Agent 10) without replacing Brevo `email_jobs` path.  
5. Gate with `packing.read` / `packing.write`.

---

## 4. Known limitations

| Item | Note |
| --- | --- |
| Carrier label printing | Tracking number entry only; no label PDF |
| Auto queue from receiving | Status + allocated qty gate; no push notification to packers |
| Playwright E2E-I | API-covered; UI smoke deferred to stage consolidation |
| Dashboard packing widget | DASH-04 still later |

---

## 5. Next — Agent 10: Communication / WhatsApp + team polish

Start with:

- Source of truth: `docs/PIKKO_MASTER_PRD.md` §23 WA-* / RBAC polish  
- Prior handoff: this file  
- Reuse email_jobs + outbox; add WA deep-link templates  
- Preserve packing/shipment notifications and portal auth
