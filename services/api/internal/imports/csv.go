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
	if in.Size <= 0 {
		return nil, fmt.Errorf("%w: choose a CSV file to upload", ErrInvalidInput)
	}
	if in.Size > maxCSVBytes {
		return nil, fmt.Errorf("%w: CSV exceeds the 5 MB size limit", ErrInvalidInput)
	}
	raw, err := io.ReadAll(io.LimitReader(in.Reader, maxCSVBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > maxCSVBytes {
		return nil, fmt.Errorf("%w: CSV exceeds the 5 MB size limit", ErrInvalidInput)
	}
	// Strip UTF-8 BOM
	raw = bytes.TrimPrefix(raw, []byte("\xef\xbb\xbf"))
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, fmt.Errorf("%w: CSV file is empty", ErrInvalidInput)
	}

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
			store_id, created_by, source_type, csv_file_id, extract_images, status, item_count, progress_pct, idempotency_key, ai_provider
		) VALUES ($1,$2,'csv',$3,false,'processing',$4,10,$5,'csv')
		RETURNING id
	`, storeID, userID, f.ID, len(rows), idem).Scan(&jobID)
	if err != nil {
		return nil, err
	}

	warnCount := 0
	for i, row := range rows {
		warning := row.Warning
		if warning != nil {
			warnCount++
		}
		reviewStatus := "pending"
		if !row.Selected {
			reviewStatus = "rejected"
		}
		conf := row.Confidence
		_, err = s.DB.Exec(ctx, `
			INSERT INTO catalog_import_items (
				store_id, job_id, display_order, selected, review_status, uncertain, confidence_pct, warning,
				name, description, brand, price_idr, price_normal_idr, sku, category_name, variant_info, raw_extraction
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$12,$13,$14,'{}'::jsonb,$15::jsonb)
		`, storeID, jobID, i, row.Selected, reviewStatus, row.Uncertain, conf, warning,
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
	} else if warnCount > 0 {
		msg := fmt.Sprintf("%d row(s) have warnings — review before confirming", warnCount)
		errMsg = &msg
	}
	_, err = s.DB.Exec(ctx, `
		UPDATE catalog_import_jobs
		SET status=$3, progress_pct=$4, item_count=$5, warning_count=$6, error_message=$7, updated_at=now()
		WHERE id=$1 AND store_id=$2
	`, jobID, storeID, status, progress, len(rows), warnCount, errMsg)
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
	Confidence  *float64
	Warning     *string
}

func parseProductCSV(raw []byte, maxRows int) ([]csvRow, error) {
	// Detect common delimiter mistakes early (semicolon / tab only files).
	sample := string(raw)
	if len(sample) > 2048 {
		sample = sample[:2048]
	}
	if !strings.Contains(sample, ",") && (strings.Contains(sample, ";") || strings.Contains(sample, "\t")) {
		return nil, fmt.Errorf("CSV must use comma separators — download the template and save as CSV (not Excel semicolon)")
	}

	r := csv.NewReader(bytes.NewReader(raw))
	r.TrimLeadingSpace = true
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("invalid CSV format: %v — check quotes and commas", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("CSV is empty")
	}
	header := normalizeHeader(records[0])
	idx := map[string]int{}
	for i, h := range header {
		if h == "" {
			continue
		}
		if _, exists := idx[h]; exists {
			return nil, fmt.Errorf("duplicate column header: %s", h)
		}
		idx[h] = i
	}
	required := []string{"name", "price_idr"}
	var missing []string
	for _, key := range required {
		if _, ok := idx[key]; !ok {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required column(s): %s — expected headers like name,price_idr,sku,…", strings.Join(missing, ", "))
	}
	var out []csvRow
	var rowWarnings []string
	data := records[1:]
	if len(data) > maxRows {
		return nil, fmt.Errorf("CSV has %d data rows; maximum is %d", len(data), maxRows)
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
		conf := 95.0
		var price *int64
		if name == "" {
			warnings = append(warnings, "name is required")
			selected = false
			uncertain = true
			conf = 20
		}
		if priceRaw == "" {
			warnings = append(warnings, "price_idr is required")
			selected = false
			uncertain = true
			if conf > 30 {
				conf = 30
			}
		} else {
			cleaned := strings.ReplaceAll(priceRaw, ",", "")
			cleaned = strings.ReplaceAll(cleaned, " ", "")
			cleaned = strings.TrimPrefix(strings.ToUpper(cleaned), "IDR")
			cleaned = strings.TrimPrefix(cleaned, "RP")
			cleaned = strings.TrimSpace(cleaned)
			if strings.Contains(cleaned, ".") {
				warnings = append(warnings, "price_idr must be whole rupiah (no decimals)")
				selected = false
				uncertain = true
				conf = 35
			} else {
				p, err := strconv.ParseInt(cleaned, 10, 64)
				if err != nil || p < 0 {
					warnings = append(warnings, "price_idr must be a non-negative integer (IDR)")
					selected = false
					uncertain = true
					conf = 35
				} else {
					price = &p
				}
			}
		}
		sku := get("sku")
		if sku != "" {
			if prev, ok := seenSKU[strings.ToLower(sku)]; ok {
				warnings = append(warnings, fmt.Sprintf("duplicate SKU also on row %d", prev))
				uncertain = true
				if conf > 60 {
					conf = 60
				}
			} else {
				seenSKU[strings.ToLower(sku)] = rowNum
			}
		}
		readyRaw := get("ready_stock_eligible")
		preorderRaw := get("preorder_eligible")
		ready, readyOK := parseBoolStrict(readyRaw, true)
		preorder, preorderOK := parseBoolStrict(preorderRaw, false)
		if readyRaw != "" && !readyOK {
			warnings = append(warnings, "ready_stock_eligible must be true/false")
			uncertain = true
			if conf > 70 {
				conf = 70
			}
		}
		if preorderRaw != "" && !preorderOK {
			warnings = append(warnings, "preorder_eligible must be true/false")
			uncertain = true
			if conf > 70 {
				conf = 70
			}
		}
		var stock *int
		if sq := get("stock_quantity"); sq != "" {
			n, err := strconv.Atoi(sq)
			if err != nil || n < 0 {
				warnings = append(warnings, "stock_quantity must be a non-negative integer")
				uncertain = true
				if conf > 70 {
					conf = 70
				}
			} else {
				stock = &n
			}
		}
		var warnPtr *string
		if len(warnings) > 0 {
			w := fmt.Sprintf("row %d: %s", rowNum, strings.Join(warnings, "; "))
			warnPtr = &w
			rowWarnings = append(rowWarnings, w)
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
			Confidence:  &conf,
			Warning:     warnPtr,
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("CSV has no data rows — add at least one product under the header")
	}
	if len(rowWarnings) > 0 {
		summary := rowWarnings[0]
		if len(rowWarnings) > 1 {
			summary = fmt.Sprintf("%s (+%d more)", rowWarnings[0], len(rowWarnings)-1)
		}
		return out, fmt.Errorf("%s", summary)
	}
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
	b, _ := parseBoolStrict(v, def)
	return b
}

func parseBoolStrict(v string, def bool) (value bool, ok bool) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "":
		return def, true
	case "1", "true", "yes", "y":
		return true, true
	case "0", "false", "no", "n":
		return false, true
	default:
		return def, false
	}
}

func strPtrOrNil(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}
