package imports

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// CSVTemplate is the downloadable product import template (PRD §18).
const CSVTemplate = `name,description,brand,price_idr,sku,category,stock_quantity,ready_stock_eligible,preorder_eligible
Sample Tee,Soft cotton tee,Acme,125000,TEE-001,Apparel,10,true,false
Preorder Bag,Limited bag,,450000,BAG-PO,Accessories,,false,true
`

const maxCSVBytes = 5 << 20
const maxCSVRows = 500

type CSVStartInput struct {
	IdempotencyKey string
	Filename       string
	DeclaredType   string
	Reader         io.Reader
	Size           int64
}

func (s *Service) StartCSV(ctx context.Context, storeID, userID uuid.UUID, in CSVStartInput) (*Job, error) {
	if in.Size <= 0 || in.Size > maxCSVBytes {
		return nil, fmt.Errorf("%w: CSV exceeds size limit", ErrInvalidInput)
	}
	raw, err := io.ReadAll(io.LimitReader(in.Reader, maxCSVBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > maxCSVBytes {
		return nil, fmt.Errorf("%w: CSV exceeds size limit", ErrInvalidInput)
	}
	// Strip UTF-8 BOM
	raw = bytes.TrimPrefix(raw, []byte("\xef\xbb\xbf"))

	f, err := s.Files.Upload(ctx, storeID, userID, "catalog_csv", in.Filename, "text/csv", bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return nil, err
	}

	idem := strings.TrimSpace(in.IdempotencyKey)
	if idem == "" {
		idem = "import-csv-" + storeID.String() + "-" + uuid.NewString()
	}
	existing, err := s.getJobByIdem(ctx, storeID, idem)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	rows, parseErr := parseProductCSV(raw, maxCSVRows)
	if parseErr != nil && len(rows) == 0 {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, parseErr)
	}

	var jobID uuid.UUID
	err = s.DB.QueryRow(ctx, `
		INSERT INTO catalog_import_jobs (
			store_id, created_by, source_type, csv_file_id, extract_images, status, item_count, progress_pct, idempotency_key
		) VALUES ($1,$2,'csv',$3,false,'processing',$4,10,$5)
		RETURNING id
	`, storeID, userID, f.ID, len(rows), idem).Scan(&jobID)
	if err != nil {
		return nil, err
	}

	for i, row := range rows {
		warning := row.Warning
		_, err = s.DB.Exec(ctx, `
			INSERT INTO catalog_import_items (
				store_id, job_id, display_order, selected, uncertain, warning,
				name, description, brand, price_idr, sku, category_name, variant_info, raw_extraction
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,'{}'::jsonb,$13::jsonb)
		`, storeID, jobID, i, row.Selected, row.Uncertain, warning,
			row.Name, row.Description, row.Brand, row.PriceIDR, row.SKU, row.Category,
			fmt.Sprintf(`{"stock_quantity":%s,"ready_stock_eligible":%v,"preorder_eligible":%v,"row":%d}`,
				nullIntJSON(row.StockQty), row.ReadyStock, row.Preorder, row.RowNumber),
		)
		if err != nil {
			_ = s.failJob(ctx, storeID, jobID, err)
			return nil, err
		}
	}

	status := "review_required"
	progress := 100
	var errMsg *string
	if parseErr != nil {
		msg := parseErr.Error()
		errMsg = &msg
	}
	_, err = s.DB.Exec(ctx, `
		UPDATE catalog_import_jobs
		SET status=$3, progress_pct=$4, item_count=$5, error_message=$6, updated_at=now()
		WHERE id=$1 AND store_id=$2
	`, jobID, storeID, status, progress, len(rows), errMsg)
	if err != nil {
		return nil, err
	}

	_ = s.Outbox.Write(ctx, &storeID, "catalog_import_requested", map[string]any{
		"import_job_id": jobID.String(),
		"source_type":   "csv",
	}, "outbox-requested-"+idem)

	return s.GetJob(ctx, storeID, jobID)
}

func nullIntJSON(p *int) string {
	if p == nil {
		return "null"
	}
	return strconv.Itoa(*p)
}

type csvRow struct {
	RowNumber   int
	Name        *string
	Description *string
	Brand       *string
	PriceIDR    *int64
	SKU         *string
	Category    *string
	StockQty    *int
	ReadyStock  bool
	Preorder    bool
	Selected    bool
	Uncertain   bool
	Warning     *string
}

func parseProductCSV(raw []byte, maxRows int) ([]csvRow, error) {
	r := csv.NewReader(bytes.NewReader(raw))
	r.TrimLeadingSpace = true
	r.FieldsPerRecord = -1
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("invalid CSV: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("CSV is empty")
	}
	header := normalizeHeader(records[0])
	idx := map[string]int{}
	for i, h := range header {
		idx[h] = i
	}
	required := []string{"name", "price_idr"}
	for _, key := range required {
		if _, ok := idx[key]; !ok {
			return nil, fmt.Errorf("missing required column: %s", key)
		}
	}
	var out []csvRow
	var firstErr error
	data := records[1:]
	if len(data) > maxRows {
		return nil, fmt.Errorf("CSV exceeds %d row limit", maxRows)
	}
	seenSKU := map[string]int{}
	for i, rec := range data {
		if isBlankRow(rec) {
			continue
		}
		rowNum := i + 2
		get := func(key string) string {
			j, ok := idx[key]
			if !ok || j >= len(rec) {
				return ""
			}
			return strings.TrimSpace(rec[j])
		}
		name := get("name")
		priceRaw := get("price_idr")
		var warnings []string
		selected := true
		uncertain := false
		var price *int64
		if name == "" {
			warnings = append(warnings, "name is required")
			selected = false
			uncertain = true
		}
		if priceRaw == "" {
			warnings = append(warnings, "price_idr is required")
			selected = false
			uncertain = true
		} else {
			p, err := strconv.ParseInt(strings.ReplaceAll(priceRaw, ",", ""), 10, 64)
			if err != nil || p < 0 {
				warnings = append(warnings, "price_idr must be a non-negative integer (IDR)")
				selected = false
				uncertain = true
			} else {
				price = &p
			}
		}
		sku := get("sku")
		if sku != "" {
			if prev, ok := seenSKU[strings.ToLower(sku)]; ok {
				warnings = append(warnings, fmt.Sprintf("duplicate SKU also on row %d", prev))
				uncertain = true
			} else {
				seenSKU[strings.ToLower(sku)] = rowNum
			}
		}
		ready := parseBoolDefault(get("ready_stock_eligible"), true)
		preorder := parseBoolDefault(get("preorder_eligible"), false)
		var stock *int
		if sq := get("stock_quantity"); sq != "" {
			n, err := strconv.Atoi(sq)
			if err != nil || n < 0 {
				warnings = append(warnings, "stock_quantity must be a non-negative integer")
				uncertain = true
			} else {
				stock = &n
			}
		}
		var warnPtr *string
		if len(warnings) > 0 {
			w := strings.Join(warnings, "; ")
			warnPtr = &w
			if firstErr == nil {
				firstErr = fmt.Errorf("%s", w)
			}
		}
		out = append(out, csvRow{
			RowNumber:   rowNum,
			Name:        strPtrOrNil(name),
			Description: strPtrOrNil(get("description")),
			Brand:       strPtrOrNil(get("brand")),
			PriceIDR:    price,
			SKU:         strPtrOrNil(sku),
			Category:    strPtrOrNil(get("category")),
			StockQty:    stock,
			ReadyStock:  ready,
			Preorder:    preorder,
			Selected:    selected,
			Uncertain:   uncertain,
			Warning:     warnPtr,
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("CSV has no data rows")
	}
	_ = firstErr
	return out, nil
}

func normalizeHeader(row []string) []string {
	out := make([]string, len(row))
	for i, h := range row {
		h = strings.ToLower(strings.TrimSpace(h))
		h = strings.ReplaceAll(h, " ", "_")
		out[i] = h
	}
	return out
}

func isBlankRow(rec []string) bool {
	for _, c := range rec {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

func parseBoolDefault(v string, def bool) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "":
		return def
	case "1", "true", "yes", "y":
		return true
	case "0", "false", "no", "n":
		return false
	default:
		return def
	}
}

func strPtrOrNil(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}
