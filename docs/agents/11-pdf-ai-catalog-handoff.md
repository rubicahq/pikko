# Agent 11 → Agent 12 Handoff (PDF/AI Catalog Hardening → Live Shopping)

**Date:** 2026-11-25 (Asia/Jakarta)  
**PRD:** v3.0 (`docs/PIKKO_MASTER_PRD.md`)  
**Branch:** `cursor/agent-11-pdf-ai-catalog-84b7`  
**Prior:** Agent 10 feature merge `983c67a` (WhatsApp); main tip includes handoff PR #28  
**Feature merge SHA:** `6a51005d55196e76aebae5d0d519061647d32fef` (PR #29)  
**Merged main SHA:** `6a51005d55196e76aebae5d0d519061647d32fef`

---

## 1. What Agent 11 delivered

| Area | Outcome |
| --- | --- |
| Migration `000015_pdf_import_hardening` | Additive `confidence_pct`, `review_status`, job `warning_count` / `ai_provider` |
| PDF-01 | Retry clears draft pages/items; bbox clamp on confirm hotspots; clearer upload errors |
| Review UX | Confidence badges, reject/restore, filters, bulk select/reject/accept-high, confirm count |
| Bulk API | `POST …/catalog-imports/{jobID}/items/bulk` |
| PDF-02 CSV | Semicolon/decimal/bool/BOM/dup-header errors; row warning summary on job |
| PDF-03/04 Flipbook | NormalizeBBox; editor drag reposition; public empty/error polish; cart → checkout |
| PDF-05 AI | Mock without key/model; OpenAI-compat path; `confidence_pct` normalize; provider on job |
| E2E-B | API chain test `TestE2EB_PDFImportToPublicFlipbook` (+ CSV error test) |
| Docs | Matrix / progress / schema / contracts / catalog link / this handoff |

**Preserved:** Stage 2 ops (purchasing, portal, packing, WhatsApp), checkout, existing import/flipbook contracts.  
**No pikko.id cutover. No destructive migrations.**

---

## 2. Acceptance gate evidence

Per owner testing policy:

- `go test ./internal/ai/ ./internal/imports/ ./internal/flipbook/ ./internal/pdfutil/`
- `tsc --noEmit` on web
- No full Playwright suite required for this handover
- Merge without empty CI wait

---

## 3. How later agents use import / flipbook

1. Review items via `confidence_pct` + `review_status`; prefer bulk actions over N PATCH calls.  
2. Confirm still creates **draft** products only — publish separately before public hotspots show.  
3. Hotspot geometry must stay within page %; use `flipbook.NormalizeBBox`.  
4. Mock AI is default when `AI` key/base/model incomplete — safe for CI.  
5. Live shopping (Agent 12) should not regress catalog import or public flipbook.

---

## 4. Known limitations

| Item | Note |
| --- | --- |
| True PDF rasterization | Still uses configured renderer / placeholder in tests |
| Image crop from bbox | Page image attached as provisional product image |
| Playwright E2E-B UI | Deferred to stage consolidation smoke |
| AI quality | Depends on SumoPod/OpenAI-compat model when keyed |

---

## 5. Next — Agent 12: Live Shopping & Storefront

Start with:

- Source of truth: `docs/PIKKO_MASTER_PRD.md` LIVE-* / Stage 3  
- Prior handoff: this file  
- Preserve import/flipbook + Stage 2 surfaces  
- Campaign `live` type already exists from Agent 04
