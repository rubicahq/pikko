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
	ProgressPct    int        `json:"progress_pct"`
	ErrorMessage   *string    `json:"error_message,omitempty"`
	IdempotencyKey string     `json:"idempotency_key"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

const jobSelectCols = `
	id, store_id, COALESCE(source_type,'pdf'), pdf_file_id, csv_file_id, extract_images, status, page_count,
	item_count, progress_pct, error_message, idempotency_key, created_at, updated_at, completed_at
`

func scanJob(row interface {
	Scan(dest ...any) error
}) (*Job, error) {
	var j Job
	err := row.Scan(
		&j.ID, &j.StoreID, &j.SourceType, &j.PDFFileID, &j.CSVFileID, &j.ExtractImages, &j.Status, &j.PageCount,
		&j.ItemCount, &j.ProgressPct, &j.ErrorMessage, &j.IdempotencyKey, &j.CreatedAt, &j.UpdatedAt, &j.CompletedAt,
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
	Uncertain        bool            `json:"uncertain"`
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
	if in.Size <= 0 || in.Size > lim.MaxPDFBytes {
		return nil, fmt.Errorf("%w: PDF exceeds size limit", ErrInvalidInput)
	}
	head := make([]byte, 512)
	n, _ := io.ReadFull(in.Reader, head)
	head = head[:n]
	if !pdfutil.IsPDF(head) {
		return nil, fmt.Errorf("%w: file must be a PDF", ErrInvalidInput)
	}
	body := io.MultiReader(bytes.NewReader(head), in.Reader)

	// Read full PDF into memory for page count (bounded by MaxPDFBytes)
	full, err := io.ReadAll(io.LimitReader(body, lim.MaxPDFBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(full)) > lim.MaxPDFBytes {
		return nil, fmt.Errorf("%w: PDF exceeds size limit", ErrInvalidInput)
	}
	pages, err := pdfutil.CountPages(full)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	if pages > lim.MaxPages {
		return nil, fmt.Errorf("%w: PDF exceeds %d page limit", ErrInvalidInput, lim.MaxPages)
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
		SELECT id, job_id, display_order, selected, uncertain, warning, name, description, brand,
		       price_idr, sku, category_name, variant_info, image_file_id, page_number,
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
			&it.ID, &it.JobID, &it.DisplayOrder, &it.Selected, &it.Uncertain, &it.Warning, &it.Name, &it.Description, &it.Brand,
			&it.PriceIDR, &it.SKU, &it.CategoryName, &it.VariantInfo, &it.ImageFileID, &it.PageNumber,
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
		return nil, fmt.Errorf("%w: job not in review", ErrInvalidInput)
	}
	cur, err := s.getItem(ctx, storeID, jobID, itemID)
	if err != nil {
		return nil, err
	}
	if in.Selected != nil {
		cur.Selected = *in.Selected
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
			selected=$4, uncertain=$5, name=$6, description=$7, brand=$8, price_idr=$9,
			sku=$10, category_name=$11, warning=$12, updated_at=now()
		WHERE id=$1 AND store_id=$2 AND job_id=$3
	`, itemID, storeID, jobID, cur.Selected, cur.Uncertain, cur.Name, cur.Description, cur.Brand, cur.PriceIDR,
		cur.SKU, cur.CategoryName, cur.Warning)
	if err != nil {
		return nil, err
	}
	return s.getItem(ctx, storeID, jobID, itemID)
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
		SELECT id, job_id, display_order, selected, uncertain, warning, name, description, brand,
		       price_idr, sku, category_name, variant_info, image_file_id, page_number,
		       bbox_x_pct, bbox_y_pct, bbox_width_pct, bbox_height_pct, created_product_id, created_at, updated_at
		FROM catalog_import_items WHERE id=$1 AND store_id=$2 AND job_id=$3
	`, itemID, storeID, jobID).Scan(
		&it.ID, &it.JobID, &it.DisplayOrder, &it.Selected, &it.Uncertain, &it.Warning, &it.Name, &it.Description, &it.Brand,
		&it.PriceIDR, &it.SKU, &it.CategoryName, &it.VariantInfo, &it.ImageFileID, &it.PageNumber,
		&it.BBoxXPct, &it.BBoxYPct, &it.BBoxWidthPct, &it.BBoxHeightPct, &it.CreatedProductID, &it.CreatedAt, &it.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &it, err
}

type ConfirmInput struct {
	ItemIDs []uuid.UUID `json:"item_ids"`
}

type ConfirmResult struct {
	Job           *Job            `json:"job"`
	ProductIDs    []uuid.UUID     `json:"product_ids"`
	CreatedCount  int             `json:"created_count"`
}

func (s *Service) Confirm(ctx context.Context, storeID, jobID uuid.UUID, in ConfirmInput) (*ConfirmResult, error) {
	job, err := s.GetJob(ctx, storeID, jobID)
	if err != nil {
		return nil, err
	}
	if job.Status != "review_required" {
		return nil, fmt.Errorf("%w: job must be in review_required", ErrInvalidInput)
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
	for _, it := range items {
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
			UPDATE catalog_import_items SET created_product_id=$3, selected=true, updated_at=now()
			WHERE id=$1 AND store_id=$2
		`, it.ID, storeID, p.ID)

		// Create hotspot candidate when bbox + page known
		if it.PageNumber != nil && it.BBoxXPct != nil && it.BBoxYPct != nil && it.BBoxWidthPct != nil && it.BBoxHeightPct != nil {
			var pageID uuid.UUID
			err := s.DB.QueryRow(ctx, `
				SELECT id FROM catalog_pages
				WHERE store_id=$1 AND source_import_job_id=$2 AND page_number=$3
				LIMIT 1
			`, storeID, jobID, *it.PageNumber).Scan(&pageID)
			if err == nil {
				_, _ = s.DB.Exec(ctx, `
					INSERT INTO catalog_page_hotspots (
						store_id, catalog_page_id, product_id, import_item_id, x_pct, y_pct, width_pct, height_pct
					) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
				`, storeID, pageID, p.ID, it.ID, *it.BBoxXPct, *it.BBoxYPct, *it.BBoxWidthPct, *it.BBoxHeightPct)
			}
		}
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
	}, "outbox-completed-"+job.IdempotencyKey)

	job, _ = s.GetJob(ctx, storeID, jobID)
	return &ConfirmResult{Job: job, ProductIDs: created, CreatedCount: len(created)}, nil
}

func (s *Service) Retry(ctx context.Context, storeID, jobID uuid.UUID) (*Job, error) {
	job, err := s.GetJob(ctx, storeID, jobID)
	if err != nil {
		return nil, err
	}
	if job.Status != "failed" {
		return nil, fmt.Errorf("%w: only failed jobs can retry", ErrInvalidInput)
	}
	idem := "retry-" + jobID.String() + "-" + uuid.NewString()
	payload := map[string]any{"import_job_id": jobID.String(), "store_id": storeID.String()}
	qJob, err := s.Queue.Enqueue(ctx, &storeID, jobs.TypeCatalogPDFImport, payload, idem, 5)
	if err != nil {
		return nil, err
	}
	_, err = s.DB.Exec(ctx, `
		UPDATE catalog_import_jobs SET status='pending', progress_pct=0, error_message=NULL, job_queue_id=$3, updated_at=now()
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
	result, err := extractor.ExtractCatalog(ctx, ai.ExtractRequest{
		Pages:         aiPages,
		ExtractImages: job.ExtractImages,
		MaxItems:      lim.MaxItems,
	})
	if err != nil {
		return s.failJob(ctx, storeID, jobID, err)
	}

	// Clear previous items on retry
	_, _ = s.DB.Exec(ctx, `DELETE FROM catalog_import_items WHERE job_id=$1 AND store_id=$2`, jobID, storeID)

	count := 0
	for i, ex := range result.Items {
		if count >= lim.MaxItems {
			break
		}
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
		rawItem, _ := json.Marshal(ex)
		_, err = s.DB.Exec(ctx, `
			INSERT INTO catalog_import_items (
				store_id, job_id, display_order, selected, uncertain, warning,
				name, description, brand, price_idr, sku, category_name, variant_info,
				image_file_id, page_number, bbox_x_pct, bbox_y_pct, bbox_width_pct, bbox_height_pct, raw_extraction
			) VALUES ($1,$2,$3,true,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
		`, storeID, jobID, i, ex.Uncertain, ex.Warning,
			ex.Name, ex.Description, ex.Brand, ex.PriceIDR, ex.SKU, ex.Category, variant,
			imageID, ex.PageNumber, ex.BBoxXPct, ex.BBoxYPct, ex.BBoxWidthPct, ex.BBoxHeightPct, rawItem)
		if err != nil {
			return s.failJob(ctx, storeID, jobID, err)
		}
		count++
	}

	_, err = s.DB.Exec(ctx, `
		UPDATE catalog_import_jobs
		SET status='review_required', item_count=$3, progress_pct=100, updated_at=now()
		WHERE id=$1 AND store_id=$2
	`, jobID, storeID, count)
	if err != nil {
		return err
	}
	_ = s.Outbox.Write(ctx, &storeID, "catalog_import_review_required", map[string]any{
		"import_job_id": jobID.String(),
		"item_count":    count,
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
