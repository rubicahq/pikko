package imports

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rubicahq/pikko/services/api/internal/ai"
	"github.com/rubicahq/pikko/services/api/internal/catalog"
	"github.com/rubicahq/pikko/services/api/internal/files"
	"github.com/rubicahq/pikko/services/api/internal/jobs"
	"github.com/rubicahq/pikko/services/api/internal/pdfutil"
	"github.com/rubicahq/pikko/services/api/internal/storage"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrConflict     = errors.New("conflict")
	ErrFailed       = errors.New("import failed")
)

type Limits struct {
	MaxPDFBytes int64
	MaxPages    int
	MaxItems    int
}

func DefaultLimits() Limits {
	return Limits{
		MaxPDFBytes: pdfutil.DefaultMaxBytes,
		MaxPages:    pdfutil.DefaultMaxPages,
		MaxItems:    pdfutil.DefaultMaxItems,
	}
}

type Job struct {
	ID             uuid.UUID  `json:"id"`
	StoreID        uuid.UUID  `json:"store_id"`
	SourceType     string     `json:"source_type"`
	PDFFileID      *uuid.UUID `json:"pdf_file_id,omitempty"`
	CSVFileID      *uuid.UUID `json:"csv_file_id,omitempty"`
	ExtractImages  bool       `json:"extract_images"`
	Status         string     `json:"status"`
	PageCount      *int       `json:"page_count,omitempty"`
	ItemCount      int        `json:"item_count"`
	WarningCount   int        `json:"warning_count"`
	ProgressPct    int        `json:"progress_pct"`
	ErrorMessage   *string    `json:"error_message,omitempty"`
	AIProvider     *string    `json:"ai_provider,omitempty"`
	IdempotencyKey string     `json:"idempotency_key"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

const jobSelectCols = `
	id, store_id, COALESCE(source_type,'pdf'), pdf_file_id, csv_file_id, extract_images, status, page_count,
	item_count, COALESCE(warning_count,0), progress_pct, error_message, ai_provider, idempotency_key, created_at, updated_at, completed_at
`

func scanJob(row interface {
	Scan(dest ...any) error
}) (*Job, error) {
	var j Job
	err := row.Scan(
		&j.ID, &j.StoreID, &j.SourceType, &j.PDFFileID, &j.CSVFileID, &j.ExtractImages, &j.Status, &j.PageCount,
		&j.ItemCount, &j.WarningCount, &j.ProgressPct, &j.ErrorMessage, &j.AIProvider, &j.IdempotencyKey, &j.CreatedAt, &j.UpdatedAt, &j.CompletedAt,
	)
	if err != nil {
		return nil, err
	}
	if j.SourceType == "" {
		j.SourceType = "pdf"
	}
	return &j, nil
}

type Item struct {
	ID               uuid.UUID       `json:"id"`
	JobID            uuid.UUID       `json:"job_id"`
	DisplayOrder     int             `json:"display_order"`
	Selected         bool            `json:"selected"`
	ReviewStatus     string          `json:"review_status"`
	Uncertain        bool            `json:"uncertain"`
	ConfidencePct    *float64        `json:"confidence_pct,omitempty"`
	Warning          *string         `json:"warning,omitempty"`
	Name             *string         `json:"name,omitempty"`
	Description      *string         `json:"description,omitempty"`
	Brand            *string         `json:"brand,omitempty"`
	PriceIDR         *int64          `json:"price_idr,omitempty"`
	SKU              *string         `json:"sku,omitempty"`
	CategoryName     *string         `json:"category_name,omitempty"`
	VariantInfo      json.RawMessage `json:"variant_info,omitempty"`
	ImageFileID      *uuid.UUID      `json:"image_file_id,omitempty"`
	ImageURL         string          `json:"image_url,omitempty"`
	PageNumber       *int            `json:"page_number,omitempty"`
	BBoxXPct         *float64        `json:"bbox_x_pct,omitempty"`
	BBoxYPct         *float64        `json:"bbox_y_pct,omitempty"`
	BBoxWidthPct     *float64        `json:"bbox_width_pct,omitempty"`
	BBoxHeightPct    *float64        `json:"bbox_height_pct,omitempty"`
	CreatedProductID *uuid.UUID      `json:"created_product_id,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type Service struct {
	DB        *pgxpool.Pool
	Files     *files.Service
	Catalog   *catalog.Service
	Queue     *jobs.Queue
	Outbox    *jobs.Outbox
	AI        ai.CatalogExtractor
	Renderer  pdfutil.Renderer
	Limits    Limits
	Logger    *slog.Logger
	Storage   storage.ObjectStore
}

type StartInput struct {
	ExtractImages  bool   `json:"extract_images"`
	IdempotencyKey string `json:"idempotency_key"`
	Filename       string
	DeclaredType   string
	Reader         io.Reader
	Size           int64
}

func (s *Service) limits() Limits {
	if s.Limits.MaxPDFBytes == 0 {
		return DefaultLimits()
	}
	return s.Limits
}

func (s *Service) Start(ctx context.Context, storeID, userID uuid.UUID, in StartInput) (*Job, error) {
	lim := s.limits()
	if in.Size <= 0 {
		return nil, fmt.Errorf("%w: choose a PDF file to upload", ErrInvalidInput)
	}
	if in.Size > lim.MaxPDFBytes {
		return nil, fmt.Errorf("%w: PDF exceeds the %d MB size limit", ErrInvalidInput, lim.MaxPDFBytes/(1<<20))
	}
	head := make([]byte, 512)
	n, _ := io.ReadFull(in.Reader, head)
	head = head[:n]
	if !pdfutil.IsPDF(head) {
		return nil, fmt.Errorf("%w: file must be a PDF (application/pdf)", ErrInvalidInput)
	}
	body := io.MultiReader(bytes.NewReader(head), in.Reader)

	// Read full PDF into memory for page count (bounded by MaxPDFBytes)
	full, err := io.ReadAll(io.LimitReader(body, lim.MaxPDFBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(full)) > lim.MaxPDFBytes {
		return nil, fmt.Errorf("%w: PDF exceeds the %d MB size limit", ErrInvalidInput, lim.MaxPDFBytes/(1<<20))
	}
	pages, err := pdfutil.CountPages(full)
	if err != nil {
		return nil, fmt.Errorf("%w: unable to read PDF pages — file may be corrupt or password-protected", ErrInvalidInput)
	}
	if pages > lim.MaxPages {
		return nil, fmt.Errorf("%w: PDF has %d pages; maximum is %d", ErrInvalidInput, pages, lim.MaxPages)
	}

	f, err := s.Files.Upload(ctx, storeID, userID, "catalog_pdf", in.Filename, "application/pdf", bytes.NewReader(full), int64(len(full)))
	if err != nil {
		return nil, err
	}

	idem := strings.TrimSpace(in.IdempotencyKey)
	if idem == "" {
		idem = "import-" + storeID.String() + "-" + uuid.NewString()
	}

	existing, err := s.getJobByIdem(ctx, storeID, idem)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	var jobID uuid.UUID
	err = s.DB.QueryRow(ctx, `
		INSERT INTO catalog_import_jobs (
			store_id, created_by, source_type, pdf_file_id, extract_images, status, page_count, idempotency_key
		) VALUES ($1,$2,'pdf',$3,$4,'pending',$5,$6)
		RETURNING id
	`, storeID, userID, f.ID, in.ExtractImages, pages, idem).Scan(&jobID)
	if err != nil {
		return nil, err
	}

	payload := map[string]any{"import_job_id": jobID.String(), "store_id": storeID.String()}
	qJob, err := s.Queue.Enqueue(ctx, &storeID, jobs.TypeCatalogPDFImport, payload, "queue-"+idem, 5)
	if err != nil {
		return nil, err
	}
	_, _ = s.DB.Exec(ctx, `UPDATE catalog_import_jobs SET job_queue_id=$2, updated_at=now() WHERE id=$1`, jobID, qJob.ID)

	_ = s.Outbox.Write(ctx, &storeID, "catalog_import_requested", map[string]any{
		"import_job_id": jobID.String(),
	}, "outbox-requested-"+idem)

	return s.GetJob(ctx, storeID, jobID)
}

func (s *Service) getJobByIdem(ctx context.Context, storeID uuid.UUID, idem string) (*Job, error) {
	j, err := scanJob(s.DB.QueryRow(ctx, `
		SELECT `+jobSelectCols+` FROM catalog_import_jobs WHERE store_id=$1 AND idempotency_key=$2
	`, storeID, idem))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return j, err
}

func (s *Service) GetJob(ctx context.Context, storeID, jobID uuid.UUID) (*Job, error) {
	j, err := scanJob(s.DB.QueryRow(ctx, `
		SELECT `+jobSelectCols+` FROM catalog_import_jobs WHERE id=$1 AND store_id=$2
	`, jobID, storeID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return j, err
}

func (s *Service) ListJobs(ctx context.Context, storeID uuid.UUID, limit int) ([]Job, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, err := s.DB.Query(ctx, `
		SELECT `+jobSelectCols+` FROM catalog_import_jobs WHERE store_id=$1 ORDER BY created_at DESC LIMIT $2
	`, storeID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Job
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *j)
	}
	return out, rows.Err()
}

func (s *Service) ListItems(ctx context.Context, storeID, jobID uuid.UUID) ([]Item, error) {
	if _, err := s.GetJob(ctx, storeID, jobID); err != nil {
		return nil, err
	}
	rows, err := s.DB.Query(ctx, `
		SELECT id, job_id, display_order, selected, COALESCE(review_status,'pending'), uncertain, confidence_pct, warning,
		       name, description, brand, price_idr, sku, category_name, variant_info, image_file_id, page_number,
		       bbox_x_pct, bbox_y_pct, bbox_width_pct, bbox_height_pct, created_product_id, created_at, updated_at
		FROM catalog_import_items WHERE store_id=$1 AND job_id=$2 ORDER BY display_order, created_at
	`, storeID, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Item
	for rows.Next() {
		var it Item
		if err := rows.Scan(
			&it.ID, &it.JobID, &it.DisplayOrder, &it.Selected, &it.ReviewStatus, &it.Uncertain, &it.ConfidencePct, &it.Warning,
			&it.Name, &it.Description, &it.Brand, &it.PriceIDR, &it.SKU, &it.CategoryName, &it.VariantInfo, &it.ImageFileID, &it.PageNumber,
			&it.BBoxXPct, &it.BBoxYPct, &it.BBoxWidthPct, &it.BBoxHeightPct, &it.CreatedProductID, &it.CreatedAt, &it.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if it.ImageFileID != nil {
			it.ImageURL = fmt.Sprintf("/api/v1/stores/%s/files/%s", storeID, it.ImageFileID)
		}
		out = append(out, it)
	}
	return out, nil
}

type ItemPatch struct {
	Selected     *bool    `json:"selected"`
	ReviewStatus *string  `json:"review_status"`
	Uncertain    *bool    `json:"uncertain"`
	Name         *string  `json:"name"`
	Description  *string  `json:"description"`
	Brand        *string  `json:"brand"`
	PriceIDR     *int64   `json:"price_idr"`
	SKU          *string  `json:"sku"`
	CategoryName *string  `json:"category_name"`
	Warning      *string  `json:"warning"`
}

func (s *Service) PatchItem(ctx context.Context, storeID, jobID, itemID uuid.UUID, in ItemPatch) (*Item, error) {
	job, err := s.GetJob(ctx, storeID, jobID)
	if err != nil {
		return nil, err
	}
	if job.Status != "review_required" && job.Status != "completed" {
		return nil, fmt.Errorf("%w: job is not ready for review yet", ErrInvalidInput)
	}
	if job.Status == "completed" {
		return nil, fmt.Errorf("%w: import already confirmed — edit products in Catalog", ErrInvalidInput)
	}
	cur, err := s.getItem(ctx, storeID, jobID, itemID)
	if err != nil {
		return nil, err
	}
	if in.ReviewStatus != nil {
		rs := strings.ToLower(strings.TrimSpace(*in.ReviewStatus))
		switch rs {
		case "pending", "accepted", "rejected":
			cur.ReviewStatus = rs
			if rs == "rejected" {
				cur.Selected = false
			}
			if rs == "accepted" || rs == "pending" {
				cur.Selected = true
			}
		default:
			return nil, fmt.Errorf("%w: review_status must be pending, accepted, or rejected", ErrInvalidInput)
		}
	}
	if in.Selected != nil {
		cur.Selected = *in.Selected
		if !*in.Selected {
			cur.ReviewStatus = "rejected"
		} else if cur.ReviewStatus == "rejected" {
			cur.ReviewStatus = "pending"
		}
	}
	if in.Uncertain != nil {
		cur.Uncertain = *in.Uncertain
	}
	if in.Name != nil {
		cur.Name = in.Name
	}
	if in.Description != nil {
		cur.Description = in.Description
	}
	if in.Brand != nil {
		cur.Brand = in.Brand
	}
	if in.PriceIDR != nil {
		if *in.PriceIDR < 0 {
			return nil, fmt.Errorf("%w: price_idr must be a non-negative integer", ErrInvalidInput)
		}
		cur.PriceIDR = in.PriceIDR
	}
	if in.SKU != nil {
		cur.SKU = in.SKU
	}
	if in.CategoryName != nil {
		cur.CategoryName = in.CategoryName
	}
	if in.Warning != nil {
		cur.Warning = in.Warning
	}
	_, err = s.DB.Exec(ctx, `
		UPDATE catalog_import_items SET
			selected=$4, review_status=$5, uncertain=$6, name=$7, description=$8, brand=$9, price_idr=$10,
			sku=$11, category_name=$12, warning=$13, updated_at=now()
		WHERE id=$1 AND store_id=$2 AND job_id=$3
	`, itemID, storeID, jobID, cur.Selected, cur.ReviewStatus, cur.Uncertain, cur.Name, cur.Description, cur.Brand, cur.PriceIDR,
		cur.SKU, cur.CategoryName, cur.Warning)
	if err != nil {
		return nil, err
	}
	return s.getItem(ctx, storeID, jobID, itemID)
}

type BulkItemsInput struct {
	Action  string      `json:"action"` // select_all | reject_all | reject_uncertain | accept_high_confidence | set
	ItemIDs []uuid.UUID `json:"item_ids,omitempty"`
	Status  string      `json:"status,omitempty"` // for action=set: pending|accepted|rejected
}

type BulkItemsResult struct {
	Updated int    `json:"updated"`
	Action  string `json:"action"`
}

func (s *Service) BulkItems(ctx context.Context, storeID, jobID uuid.UUID, in BulkItemsInput) (*BulkItemsResult, error) {
	job, err := s.GetJob(ctx, storeID, jobID)
	if err != nil {
		return nil, err
	}
	if job.Status != "review_required" {
		return nil, fmt.Errorf("%w: job must be in review_required", ErrInvalidInput)
	}
	action := strings.ToLower(strings.TrimSpace(in.Action))
	var ct interface{ RowsAffected() int64 }
	switch action {
	case "select_all":
		ct, err = s.DB.Exec(ctx, `
			UPDATE catalog_import_items
			SET selected=true, review_status='pending', updated_at=now()
			WHERE store_id=$1 AND job_id=$2 AND review_status <> 'accepted'
		`, storeID, jobID)
	case "reject_all":
		ct, err = s.DB.Exec(ctx, `
			UPDATE catalog_import_items
			SET selected=false, review_status='rejected', updated_at=now()
			WHERE store_id=$1 AND job_id=$2 AND created_product_id IS NULL
		`, storeID, jobID)
	case "reject_uncertain":
		ct, err = s.DB.Exec(ctx, `
			UPDATE catalog_import_items
			SET selected=false, review_status='rejected', updated_at=now()
			WHERE store_id=$1 AND job_id=$2 AND uncertain=true AND created_product_id IS NULL
		`, storeID, jobID)
	case "accept_high_confidence":
		ct, err = s.DB.Exec(ctx, `
			UPDATE catalog_import_items
			SET selected=true, review_status='accepted', updated_at=now()
			WHERE store_id=$1 AND job_id=$2 AND created_product_id IS NULL
			  AND COALESCE(confidence_pct, CASE WHEN uncertain THEN 50 ELSE 90 END) >= 80
			  AND COALESCE(name,'') <> ''
		`, storeID, jobID)
	case "set":
		status := strings.ToLower(strings.TrimSpace(in.Status))
		if status != "pending" && status != "accepted" && status != "rejected" {
			return nil, fmt.Errorf("%w: status must be pending, accepted, or rejected", ErrInvalidInput)
		}
		if len(in.ItemIDs) == 0 {
			return nil, fmt.Errorf("%w: item_ids required for set action", ErrInvalidInput)
		}
		selected := status != "rejected"
		ct, err = s.DB.Exec(ctx, `
			UPDATE catalog_import_items
			SET selected=$4, review_status=$5, updated_at=now()
			WHERE store_id=$1 AND job_id=$2 AND id = ANY($3) AND created_product_id IS NULL
		`, storeID, jobID, in.ItemIDs, selected, status)
	default:
		return nil, fmt.Errorf("%w: unknown bulk action", ErrInvalidInput)
	}
	if err != nil {
		return nil, err
	}
	return &BulkItemsResult{Updated: int(ct.RowsAffected()), Action: action}, nil
}

func (s *Service) DeleteItem(ctx context.Context, storeID, jobID, itemID uuid.UUID) error {
	ct, err := s.DB.Exec(ctx, `
		DELETE FROM catalog_import_items WHERE id=$1 AND store_id=$2 AND job_id=$3
	`, itemID, storeID, jobID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	_, _ = s.DB.Exec(ctx, `
		UPDATE catalog_import_jobs SET item_count=(SELECT COUNT(*) FROM catalog_import_items WHERE job_id=$1), updated_at=now()
		WHERE id=$1 AND store_id=$2
	`, jobID, storeID)
	return nil
}

func (s *Service) getItem(ctx context.Context, storeID, jobID, itemID uuid.UUID) (*Item, error) {
	var it Item
	err := s.DB.QueryRow(ctx, `
		SELECT id, job_id, display_order, selected, COALESCE(review_status,'pending'), uncertain, confidence_pct, warning,
		       name, description, brand, price_idr, sku, category_name, variant_info, image_file_id, page_number,
		       bbox_x_pct, bbox_y_pct, bbox_width_pct, bbox_height_pct, created_product_id, created_at, updated_at
		FROM catalog_import_items WHERE id=$1 AND store_id=$2 AND job_id=$3
	`, itemID, storeID, jobID).Scan(
		&it.ID, &it.JobID, &it.DisplayOrder, &it.Selected, &it.ReviewStatus, &it.Uncertain, &it.ConfidencePct, &it.Warning,
		&it.Name, &it.Description, &it.Brand, &it.PriceIDR, &it.SKU, &it.CategoryName, &it.VariantInfo, &it.ImageFileID, &it.PageNumber,
		&it.BBoxXPct, &it.BBoxYPct, &it.BBoxWidthPct, &it.BBoxHeightPct, &it.CreatedProductID, &it.CreatedAt, &it.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if it.ImageFileID != nil {
		it.ImageURL = fmt.Sprintf("/api/v1/stores/%s/files/%s", storeID, it.ImageFileID)
	}
	return &it, err
}

type ConfirmInput struct {
	ItemIDs []uuid.UUID `json:"item_ids"`
}

type ConfirmResult struct {
	Job          *Job        `json:"job"`
	ProductIDs   []uuid.UUID `json:"product_ids"`
	CreatedCount int         `json:"created_count"`
	SkippedCount int         `json:"skipped_count"`
	HotspotCount int         `json:"hotspot_count"`
}

func clampHotspotBBox(x, y, w, h float64) (float64, float64, float64, float64, bool) {
	if w <= 0 || h <= 0 {
		return 0, 0, 0, 0, false
	}
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x > 100 {
		x = 100
	}
	if y > 100 {
		y = 100
	}
	if w > 100 {
		w = 100
	}
	if h > 100 {
		h = 100
	}
	if x+w > 100 {
		w = 100 - x
	}
	if y+h > 100 {
		h = 100 - y
	}
	if w < 0.5 || h < 0.5 {
		return 0, 0, 0, 0, false
	}
	return x, y, w, h, true
}

func (s *Service) Confirm(ctx context.Context, storeID, jobID uuid.UUID, in ConfirmInput) (*ConfirmResult, error) {
	job, err := s.GetJob(ctx, storeID, jobID)
	if err != nil {
		return nil, err
	}
	if job.Status != "review_required" {
		return nil, fmt.Errorf("%w: job must be in review_required to confirm", ErrInvalidInput)
	}
	items, err := s.ListItems(ctx, storeID, jobID)
	if err != nil {
		return nil, err
	}
	want := map[uuid.UUID]struct{}{}
	for _, id := range in.ItemIDs {
		want[id] = struct{}{}
	}
	useFilter := len(in.ItemIDs) > 0

	var created []uuid.UUID
	skipped := 0
	hotspots := 0
	for _, it := range items {
		if it.CreatedProductID != nil {
			continue
		}
		if it.ReviewStatus == "rejected" {
			continue
		}
		if useFilter {
			if _, ok := want[it.ID]; !ok {
				continue
			}
		} else if !it.Selected {
			continue
		}
		name := ""
		if it.Name != nil {
			name = strings.TrimSpace(*it.Name)
		}
		if name == "" {
			skipped++
			continue // skip nameless — do not fabricate
		}
		price := int64(0)
		if it.PriceIDR != nil {
			price = *it.PriceIDR
		}
		write := catalog.ProductWriteInput{
			Product: catalog.ProductInput{
				Name:         name,
				Description:  it.Description,
				SKU:          it.SKU,
				Brand:        it.Brand,
				BasePriceIDR: price,
			},
		}
		status := "draft"
		write.Product.Status = &status
		if it.ImageFileID != nil {
			if pid, err := s.ensureProductImage(ctx, storeID, *it.ImageFileID); err == nil && pid != uuid.Nil {
				write.ImageIDs = []uuid.UUID{pid}
			}
		}
		// Map simple variant_info Size/Color into options when present
		if len(it.VariantInfo) > 0 && string(it.VariantInfo) != "{}" && string(it.VariantInfo) != "null" {
			var vm map[string][]string
			if json.Unmarshal(it.VariantInfo, &vm) == nil {
				for optName, vals := range vm {
					if len(vals) == 0 {
						continue
					}
					write.Options = append(write.Options, catalog.OptionDef{Name: optName, Values: vals})
				}
			}
		}
		p, err := s.Catalog.CreateProduct(ctx, storeID, write)
		if err != nil {
			return nil, err
		}
		created = append(created, p.ID)
		_, _ = s.DB.Exec(ctx, `
			UPDATE catalog_import_items
			SET created_product_id=$3, selected=true, review_status='accepted', updated_at=now()
			WHERE id=$1 AND store_id=$2
		`, it.ID, storeID, p.ID)

		// Create hotspot candidate when bbox + page known
		if it.PageNumber != nil && it.BBoxXPct != nil && it.BBoxYPct != nil && it.BBoxWidthPct != nil && it.BBoxHeightPct != nil {
			x, y, w, h, ok := clampHotspotBBox(*it.BBoxXPct, *it.BBoxYPct, *it.BBoxWidthPct, *it.BBoxHeightPct)
			if !ok {
				continue
			}
			var pageID uuid.UUID
			err := s.DB.QueryRow(ctx, `
				SELECT id FROM catalog_pages
				WHERE store_id=$1 AND source_import_job_id=$2 AND page_number=$3
				LIMIT 1
			`, storeID, jobID, *it.PageNumber).Scan(&pageID)
			if err == nil {
				_, err = s.DB.Exec(ctx, `
					INSERT INTO catalog_page_hotspots (
						store_id, catalog_page_id, product_id, import_item_id, x_pct, y_pct, width_pct, height_pct
					) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
				`, storeID, pageID, p.ID, it.ID, x, y, w, h)
				if err == nil {
					hotspots++
				}
			}
		}
	}

	if len(created) == 0 {
		return nil, fmt.Errorf("%w: no eligible items to confirm — select items with a name, or fix rejected/empty lines", ErrInvalidInput)
	}

	_, err = s.DB.Exec(ctx, `
		UPDATE catalog_import_jobs SET status='completed', progress_pct=100, completed_at=now(), updated_at=now()
		WHERE id=$1 AND store_id=$2
	`, jobID, storeID)
	if err != nil {
		return nil, err
	}
	_ = s.Outbox.Write(ctx, &storeID, "catalog_import_completed", map[string]any{
		"import_job_id": jobID.String(),
		"product_count": len(created),
		"hotspot_count": hotspots,
	}, "outbox-completed-"+job.IdempotencyKey)

	job, _ = s.GetJob(ctx, storeID, jobID)
	return &ConfirmResult{
		Job: job, ProductIDs: created, CreatedCount: len(created),
		SkippedCount: skipped, HotspotCount: hotspots,
	}, nil
}

func (s *Service) Retry(ctx context.Context, storeID, jobID uuid.UUID) (*Job, error) {
	job, err := s.GetJob(ctx, storeID, jobID)
	if err != nil {
		return nil, err
	}
	if job.Status != "failed" {
		return nil, fmt.Errorf("%w: only failed jobs can retry", ErrInvalidInput)
	}
	if job.SourceType == "csv" {
		return nil, fmt.Errorf("%w: CSV imports cannot be retried — upload the file again", ErrInvalidInput)
	}
	// Clear prior draft pages/items so retry does not duplicate flipbook pages.
	_, _ = s.DB.Exec(ctx, `
		DELETE FROM catalog_page_hotspots
		WHERE store_id=$1 AND catalog_page_id IN (
			SELECT id FROM catalog_pages WHERE store_id=$1 AND source_import_job_id=$2
		)
	`, storeID, jobID)
	_, _ = s.DB.Exec(ctx, `
		DELETE FROM catalog_pages WHERE store_id=$1 AND source_import_job_id=$2 AND status='draft'
	`, storeID, jobID)
	_, _ = s.DB.Exec(ctx, `DELETE FROM catalog_import_items WHERE store_id=$1 AND job_id=$2`, storeID, jobID)

	idem := "retry-" + jobID.String() + "-" + uuid.NewString()
	payload := map[string]any{"import_job_id": jobID.String(), "store_id": storeID.String()}
	qJob, err := s.Queue.Enqueue(ctx, &storeID, jobs.TypeCatalogPDFImport, payload, idem, 5)
	if err != nil {
		return nil, err
	}
	_, err = s.DB.Exec(ctx, `
		UPDATE catalog_import_jobs
		SET status='pending', progress_pct=0, item_count=0, warning_count=0, error_message=NULL,
		    ai_provider=NULL, job_queue_id=$3, updated_at=now()
		WHERE id=$1 AND store_id=$2
	`, jobID, storeID, qJob.ID)
	if err != nil {
		return nil, err
	}
	return s.GetJob(ctx, storeID, jobID)
}

// ProcessJob runs the async import pipeline for one queue job payload.
func (s *Service) ProcessJob(ctx context.Context, queueJob *jobs.Job) error {
	var payload struct {
		ImportJobID string `json:"import_job_id"`
		StoreID     string `json:"store_id"`
	}
	if err := json.Unmarshal(queueJob.Payload, &payload); err != nil {
		return err
	}
	jobID, err := uuid.Parse(payload.ImportJobID)
	if err != nil {
		return err
	}
	storeID, err := uuid.Parse(payload.StoreID)
	if err != nil {
		return err
	}
	return s.processImport(ctx, storeID, jobID)
}

func (s *Service) processImport(ctx context.Context, storeID, jobID uuid.UUID) error {
	lim := s.limits()
	_, err := s.DB.Exec(ctx, `
		UPDATE catalog_import_jobs SET status='processing', progress_pct=5, updated_at=now() WHERE id=$1 AND store_id=$2
	`, jobID, storeID)
	if err != nil {
		return err
	}

	job, err := s.GetJob(ctx, storeID, jobID)
	if err != nil {
		return err
	}

	if job.SourceType == "csv" {
		return s.failJob(ctx, storeID, jobID, fmt.Errorf("csv jobs are processed synchronously"))
	}
	if job.PDFFileID == nil {
		return s.failJob(ctx, storeID, jobID, fmt.Errorf("missing pdf file"))
	}
	f, rc, err := s.Files.Open(ctx, storeID, *job.PDFFileID)
	if err != nil {
		return s.failJob(ctx, storeID, jobID, err)
	}
	defer rc.Close()
	pdfBytes, err := io.ReadAll(io.LimitReader(rc, lim.MaxPDFBytes+1))
	if err != nil {
		return s.failJob(ctx, storeID, jobID, err)
	}
	_ = f

	renderer := s.Renderer
	if renderer == nil {
		renderer = pdfutil.NewRenderer()
	}
	pages, err := renderer.RenderPages(pdfBytes, lim.MaxPages)
	if err != nil {
		return s.failJob(ctx, storeID, jobID, err)
	}
	if len(pages) > lim.MaxPages {
		return s.failJob(ctx, storeID, jobID, fmt.Errorf("page limit exceeded"))
	}
	_, _ = s.DB.Exec(ctx, `
		UPDATE catalog_import_jobs SET page_count=$3, progress_pct=30, updated_at=now() WHERE id=$1 AND store_id=$2
	`, jobID, storeID, len(pages))

	// Clear previous draft pages/items so retries stay idempotent.
	_, _ = s.DB.Exec(ctx, `
		DELETE FROM catalog_page_hotspots
		WHERE store_id=$1 AND catalog_page_id IN (
			SELECT id FROM catalog_pages WHERE store_id=$1 AND source_import_job_id=$2
		)
	`, storeID, jobID)
	_, _ = s.DB.Exec(ctx, `
		DELETE FROM catalog_pages WHERE store_id=$1 AND source_import_job_id=$2 AND status='draft'
	`, storeID, jobID)
	_, _ = s.DB.Exec(ctx, `DELETE FROM catalog_import_items WHERE job_id=$1 AND store_id=$2`, jobID, storeID)

	pageIDByNumber := map[int]uuid.UUID{}
	aiPages := make([]ai.PageImage, 0, len(pages))
	for _, p := range pages {
		sf, err := s.Files.Upload(ctx, storeID, uuid.Nil, "catalog_page",
			fmt.Sprintf("page-%d.png", p.PageNumber), "image/png",
			bytes.NewReader(p.PNG), int64(len(p.PNG)))
		if err != nil {
			return s.failJob(ctx, storeID, jobID, err)
		}
		var pageID uuid.UUID
		err = s.DB.QueryRow(ctx, `
			INSERT INTO catalog_pages (store_id, image_file_id, display_order, status, source_import_job_id, page_number)
			VALUES ($1,$2,$3,'draft',$4,$5)
			RETURNING id
		`, storeID, sf.ID, p.PageNumber-1, jobID, p.PageNumber).Scan(&pageID)
		if err != nil {
			return s.failJob(ctx, storeID, jobID, err)
		}
		pageIDByNumber[p.PageNumber] = pageID
		aiPages = append(aiPages, ai.PageImage{
			PageNumber: p.PageNumber,
			MIMEType:   "image/png",
			Base64Data: base64.StdEncoding.EncodeToString(p.PNG),
			Width:      p.Width,
			Height:     p.Height,
		})
	}

	_, _ = s.DB.Exec(ctx, `
		UPDATE catalog_import_jobs SET progress_pct=55, updated_at=now() WHERE id=$1 AND store_id=$2
	`, jobID, storeID)

	extractor := s.AI
	if extractor == nil {
		extractor = ai.NewMockExtractor()
	}
	provider := ai.ProviderName(extractor)
	_, _ = s.DB.Exec(ctx, `
		UPDATE catalog_import_jobs SET ai_provider=$3, updated_at=now() WHERE id=$1 AND store_id=$2
	`, jobID, storeID, provider)

	result, err := extractor.ExtractCatalog(ctx, ai.ExtractRequest{
		Pages:         aiPages,
		ExtractImages: job.ExtractImages,
		MaxItems:      lim.MaxItems,
	})
	if err != nil {
		return s.failJob(ctx, storeID, jobID, fmt.Errorf("AI extraction failed: %w", err))
	}

	count := 0
	warnCount := 0
	for i, ex := range result.Items {
		if count >= lim.MaxItems {
			break
		}
		ex.NormalizeConfidence()
		variant := ex.VariantInfo
		if len(variant) == 0 {
			variant = json.RawMessage(`{}`)
		}
		var imageID *uuid.UUID
		if job.ExtractImages && ex.PageNumber != nil {
			// Attach page image as provisional product image when crop unavailable
			if pid, ok := pageIDByNumber[*ex.PageNumber]; ok {
				var fid uuid.UUID
				err := s.DB.QueryRow(ctx, `SELECT image_file_id FROM catalog_pages WHERE id=$1 AND store_id=$2`, pid, storeID).Scan(&fid)
				if err == nil {
					imageID = &fid
				}
			}
		}
		if ex.Uncertain || (ex.Warning != nil && strings.TrimSpace(*ex.Warning) != "") {
			warnCount++
		}
		reviewStatus := "pending"
		selected := true
		name := ""
		if ex.Name != nil {
			name = strings.TrimSpace(*ex.Name)
		}
		if name == "" {
			selected = false
			reviewStatus = "rejected"
			if ex.Warning == nil {
				w := "missing product name"
				ex.Warning = &w
				ex.Uncertain = true
				warnCount++
			}
		}
		rawItem, _ := json.Marshal(ex)
		_, err = s.DB.Exec(ctx, `
			INSERT INTO catalog_import_items (
				store_id, job_id, display_order, selected, review_status, uncertain, confidence_pct, warning,
				name, description, brand, price_idr, sku, category_name, variant_info,
				image_file_id, page_number, bbox_x_pct, bbox_y_pct, bbox_width_pct, bbox_height_pct, raw_extraction
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22)
		`, storeID, jobID, i, selected, reviewStatus, ex.Uncertain, ex.ConfidencePct, ex.Warning,
			ex.Name, ex.Description, ex.Brand, ex.PriceIDR, ex.SKU, ex.Category, variant,
			imageID, ex.PageNumber, ex.BBoxXPct, ex.BBoxYPct, ex.BBoxWidthPct, ex.BBoxHeightPct, rawItem)
		if err != nil {
			return s.failJob(ctx, storeID, jobID, err)
		}
		count++
	}

	_, err = s.DB.Exec(ctx, `
		UPDATE catalog_import_jobs
		SET status='review_required', item_count=$3, warning_count=$4, progress_pct=100, updated_at=now()
		WHERE id=$1 AND store_id=$2
	`, jobID, storeID, count, warnCount)
	if err != nil {
		return err
	}
	_ = s.Outbox.Write(ctx, &storeID, "catalog_import_review_required", map[string]any{
		"import_job_id": jobID.String(),
		"item_count":    count,
		"ai_provider":   provider,
	}, "outbox-review-"+job.IdempotencyKey)
	return nil
}

// ensureProductImage returns a stored_files id with purpose=product suitable for product_images.
// Catalog page / crop images are copied into a product-scoped object when needed.
func (s *Service) ensureProductImage(ctx context.Context, storeID, fileID uuid.UUID) (uuid.UUID, error) {
	f, err := s.Files.GetInStore(ctx, storeID, fileID)
	if err != nil {
		return uuid.Nil, err
	}
	if f.Purpose == "product" {
		return f.ID, nil
	}
	rc, err := s.Storage.Get(ctx, f.StorageKey)
	if err != nil {
		return uuid.Nil, err
	}
	defer rc.Close()
	data, err := io.ReadAll(io.LimitReader(rc, 8<<20))
	if err != nil {
		return uuid.Nil, err
	}
	name := "import-product.png"
	if f.OriginalFilename != nil && *f.OriginalFilename != "" {
		name = *f.OriginalFilename
	}
	nf, err := s.Files.Upload(ctx, storeID, uuid.Nil, "product", name, f.ContentType, bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return uuid.Nil, err
	}
	return nf.ID, nil
}

func (s *Service) failJob(ctx context.Context, storeID, jobID uuid.UUID, cause error) error {
	msg := cause.Error()
	if s.Logger != nil {
		s.Logger.Error("catalog_import_failed", "store_id", storeID, "job_id", jobID, "error", msg)
	}
	_, _ = s.DB.Exec(ctx, `
		UPDATE catalog_import_jobs SET status='failed', error_message=$3, updated_at=now() WHERE id=$1 AND store_id=$2
	`, jobID, storeID, msg)
	_ = s.Outbox.Write(ctx, &storeID, "catalog_import_failed", map[string]any{
		"import_job_id": jobID.String(),
	}, "outbox-failed-"+jobID.String()+"-"+uuid.NewString())
	return cause
}
