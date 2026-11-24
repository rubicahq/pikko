# Agent 10 → Stage 3 Handoff (Communication / WhatsApp → PDF/AI & Live)

**Date:** 2026-11-25 (Asia/Jakarta)  
**PRD:** v3.0 (`docs/PIKKO_MASTER_PRD.md`)  
**Branch:** `cursor/agent-10-communication-wa-0165`  
**Prior:** Agent 09 main tip includes packing PR #25 merge `a033880` (+ handoff `#26` / `a613eae`)  

---

## 1. What Agent 10 delivered

| Area | Outcome |
| --- | --- |
| Migration `000014_whatsapp_templates` | Additive `whatsapp_template_sends` log |
| WA-01 | Code registry + `{{var}}` render + `https://wa.me/<phone>?text=` deep-link + copy |
| WA-V01…V08 | Vendor templates (stok, PO, harga, bayar, follow-up, terima, kosong, ubah PO) |
| WA-C01…C18 | Customer templates (order, payment, unavailable, preorder ETA, packing, ship, pickup, portal, refund, credit) |
| APIs | `GET/POST /stores/{id}/whatsapp/templates…` + `/sends` (`notifications.read`) |
| UI | Workspace → **Messages**; quick actions on vendors; links from purchasing / packing / shipments / order detail |
| Email | Brevo `email_jobs` path untouched (including `order_shipped_customer`) |

**No pikko.id cutover. No destructive migrations. No Meta Business API.**

---

## 2. Acceptance gate evidence

Per owner testing policy:

- `go test ./internal/whatsapp/ ./internal/rbac/` (+ packing smoke via `./internal/commerce/ -run TestPackingAndShipmentFlow`)
- `tsc --noEmit` on web
- No Playwright required for this handover
- Merge without empty CI wait

---

## 3. How later agents use WhatsApp

1. Resolve copy via `POST .../whatsapp/templates/{code}/preview|open` — do not hardcode WA bodies in other modules.  
2. Prefer portal URL for customer links (`tracking_url`); guest tracking tokens remain hashed-at-rest.  
3. Keep notifying via Brevo `email_jobs` for transactional email; WA is merchant-initiated deep-link only.  
4. Gate with `notifications.read`.  
5. Stage 2 is complete; Stage 3 starts at Agent 11 (PDF/AI reliability).

---

## 4. Known limitations

| Item | Note |
| --- | --- |
| Meta WhatsApp Business API | Out of MVP; deep-link + clipboard only |
| Guest tracking URL in templates | Uses portal home (token not recoverable from hash) |
| Auto-send on status change | Manual merchant open/copy only |
| Playwright Messages smoke | Deferred to stage consolidation |

---

## 5. Next — Agent 11: PDF/AI Catalog reliability

Start with:

- Source of truth: `docs/PIKKO_MASTER_PRD.md` PDF-* / Stage 3  
- Prior handoff: this file  
- Preserve import/flipbook contracts; no schema destruction  
- Preserve WA Messages + packing/finance/portal
