package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/rubicahq/pikko/services/api/internal/imports"
)

type ImportHandlers struct {
	Imports *imports.Service
}

func (h *ImportHandlers) Start(w http.ResponseWriter, r *http.Request) {
	const maxMem = 32 << 20
	if err := r.ParseMultipartForm(maxMem); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid multipart upload.")
		return
	}
	file, hdr, err := r.FormFile("file")
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Import file is required.")
		return
	}
	defer file.Close()

	source := strings.ToLower(strings.TrimSpace(r.FormValue("source")))
	nameLower := strings.ToLower(hdr.Filename)
	ct := strings.ToLower(hdr.Header.Get("Content-Type"))
	isCSV := source == "csv" || strings.HasSuffix(nameLower, ".csv") || strings.Contains(ct, "csv")

	var job *imports.Job
	if isCSV {
		job, err = h.Imports.StartCSV(r.Context(), StoreID(r), UserID(r), imports.CSVStartInput{
			IdempotencyKey: r.Header.Get("Idempotency-Key"),
			Filename:       hdr.Filename,
			DeclaredType:   hdr.Header.Get("Content-Type"),
			Reader:         file,
			Size:           hdr.Size,
		})
	} else {
		job, err = h.Imports.Start(r.Context(), StoreID(r), UserID(r), imports.StartInput{
			IdempotencyKey: r.Header.Get("Idempotency-Key"),
			Filename:       hdr.Filename,
			DeclaredType:   hdr.Header.Get("Content-Type"),
			Reader:         file,
			Size:           hdr.Size,
		})
	}
	if err != nil {
		if errors.Is(err, imports.ErrInvalidInput) || strings.Contains(err.Error(), "invalid input") {
			writeError(w, r, http.StatusBadRequest, "invalid_input", publicImportErr(err))
			return
		}
		writeError(w, r, http.StatusInternalServerError, "import_failed", "Unable to start catalog import.")
		return
	}
	status := http.StatusAccepted
	if job != nil && job.SourceType == "csv" {
		status = http.StatusOK
	}
	writeJSON(w, status, map[string]any{"job": job})
}

func (h *ImportHandlers) CSVTemplate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="pikko-product-import-template.csv"`)
	_, _ = w.Write([]byte(imports.CSVTemplate))
}

func (h *ImportHandlers) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.Imports.ListJobs(r.Context(), StoreID(r), 20)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "list_failed", "Unable to list import jobs.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"jobs": list})
}

func (h *ImportHandlers) Get(w http.ResponseWriter, r *http.Request) {
	jobID, err := uuid.Parse(r.PathValue("jobID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Import job not found.")
		return
	}
	job, err := h.Imports.GetJob(r.Context(), StoreID(r), jobID)
	if err != nil {
		if errors.Is(err, imports.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "Import job not found.")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "get_failed", "Unable to load import job.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"job": job})
}

func (h *ImportHandlers) ListItems(w http.ResponseWriter, r *http.Request) {
	jobID, err := uuid.Parse(r.PathValue("jobID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Import job not found.")
		return
	}
	items, err := h.Imports.ListItems(r.Context(), StoreID(r), jobID)
	if err != nil {
		if errors.Is(err, imports.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "Import job not found.")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "list_failed", "Unable to list import items.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *ImportHandlers) PatchItem(w http.ResponseWriter, r *http.Request) {
	jobID, err := uuid.Parse(r.PathValue("jobID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Import job not found.")
		return
	}
	itemID, err := uuid.Parse(r.PathValue("itemID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Import item not found.")
		return
	}
	var in imports.ItemPatch
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	item, err := h.Imports.PatchItem(r.Context(), StoreID(r), jobID, itemID, in)
	if err != nil {
		if errors.Is(err, imports.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "Import item not found.")
			return
		}
		if errors.Is(err, imports.ErrInvalidInput) {
			writeError(w, r, http.StatusBadRequest, "invalid_input", publicImportErr(err))
			return
		}
		writeError(w, r, http.StatusInternalServerError, "patch_failed", "Unable to update import item.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (h *ImportHandlers) BulkItems(w http.ResponseWriter, r *http.Request) {
	jobID, err := uuid.Parse(r.PathValue("jobID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Import job not found.")
		return
	}
	var in imports.BulkItemsInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	res, err := h.Imports.BulkItems(r.Context(), StoreID(r), jobID, in)
	if err != nil {
		if errors.Is(err, imports.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "Import job not found.")
			return
		}
		if errors.Is(err, imports.ErrInvalidInput) {
			writeError(w, r, http.StatusBadRequest, "invalid_input", publicImportErr(err))
			return
		}
		writeError(w, r, http.StatusInternalServerError, "bulk_failed", "Unable to update import items.")
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *ImportHandlers) DeleteItem(w http.ResponseWriter, r *http.Request) {
	jobID, err := uuid.Parse(r.PathValue("jobID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Import job not found.")
		return
	}
	itemID, err := uuid.Parse(r.PathValue("itemID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Import item not found.")
		return
	}
	if err := h.Imports.DeleteItem(r.Context(), StoreID(r), jobID, itemID); err != nil {
		if errors.Is(err, imports.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "Import item not found.")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "delete_failed", "Unable to delete import item.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *ImportHandlers) Confirm(w http.ResponseWriter, r *http.Request) {
	jobID, err := uuid.Parse(r.PathValue("jobID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Import job not found.")
		return
	}
	var in imports.ConfirmInput
	_ = json.NewDecoder(r.Body).Decode(&in)
	res, err := h.Imports.Confirm(r.Context(), StoreID(r), jobID, in)
	if err != nil {
		if errors.Is(err, imports.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "Import job not found.")
			return
		}
		if errors.Is(err, imports.ErrInvalidInput) {
			writeError(w, r, http.StatusBadRequest, "invalid_input", publicImportErr(err))
			return
		}
		writeError(w, r, http.StatusInternalServerError, "confirm_failed", "Unable to confirm import.")
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *ImportHandlers) Retry(w http.ResponseWriter, r *http.Request) {
	jobID, err := uuid.Parse(r.PathValue("jobID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Import job not found.")
		return
	}
	job, err := h.Imports.Retry(r.Context(), StoreID(r), jobID)
	if err != nil {
		if errors.Is(err, imports.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "Import job not found.")
			return
		}
		if errors.Is(err, imports.ErrInvalidInput) {
			writeError(w, r, http.StatusBadRequest, "invalid_input", publicImportErr(err))
			return
		}
		writeError(w, r, http.StatusInternalServerError, "retry_failed", "Unable to retry import.")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"job": job})
}

func publicImportErr(err error) string {
	msg := err.Error()
	msg = strings.TrimPrefix(msg, "invalid input: ")
	msg = strings.TrimPrefix(msg, "invalid input")
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return "Invalid import request."
	}
	// Capitalize first rune for UI copy.
	r := []rune(msg)
	if r[0] >= 'a' && r[0] <= 'z' {
		r[0] = r[0] - ('a' - 'A')
	}
	return string(r)
}
