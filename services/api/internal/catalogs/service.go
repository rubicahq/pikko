package catalogs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
)

// Catalog is a reusable PDF catalog document (pages + optional campaign/preorder links).
type Catalog struct {
	ID                 uuid.UUID  `json:"id"`
	StoreID            uuid.UUID  `json:"store_id"`
	Title              string     `json:"title"`
	Status             string     `json:"status"`
	SourceImportJobID  *uuid.UUID `json:"source_import_job_id,omitempty"`
	PageCount          int        `json:"page_count"`
	CampaignIDs        []uuid.UUID `json:"campaign_ids,omitempty"`
	PreorderBatchIDs   []uuid.UUID `json:"preorder_batch_ids,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type Service struct {
	DB *pgxpool.Pool
}

type CreateFromImportInput struct {
	Title string
}

// EnsureFromImport creates (or returns) a catalog for a completed/review import and links its pages.
func (s *Service) EnsureFromImport(ctx context.Context, storeID, jobID uuid.UUID, title string) (*Catalog, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Catalog import"
	}
	var id uuid.UUID
	existing, gerr := s.GetByImportJob(ctx, storeID, jobID)
	if gerr == nil {
		id = existing.ID
		_, _ = s.DB.Exec(ctx, `
			UPDATE catalogs SET title=$3, updated_at=now() WHERE id=$1 AND store_id=$2
		`, id, storeID, title)
	} else if !errors.Is(gerr, ErrNotFound) {
		return nil, gerr
	} else {
		err := s.DB.QueryRow(ctx, `
			INSERT INTO catalogs (store_id, title, status, source_import_job_id)
			VALUES ($1,$2,'draft',$3)
			RETURNING id
		`, storeID, title, jobID).Scan(&id)
		if err != nil {
			return nil, err
		}
	}
	if _, err := s.DB.Exec(ctx, `
		UPDATE catalog_pages SET catalog_id=$3, updated_at=now()
		WHERE store_id=$1 AND source_import_job_id=$2 AND (catalog_id IS NULL OR catalog_id <> $3)
	`, storeID, jobID, id); err != nil {
		return nil, err
	}
	return s.Get(ctx, storeID, id)
}

func (s *Service) GetByImportJob(ctx context.Context, storeID, jobID uuid.UUID) (*Catalog, error) {
	var id uuid.UUID
	err := s.DB.QueryRow(ctx, `
		SELECT id FROM catalogs WHERE store_id=$1 AND source_import_job_id=$2
	`, storeID, jobID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, storeID, id)
}

func (s *Service) List(ctx context.Context, storeID uuid.UUID) ([]Catalog, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT c.id, c.store_id, c.title, c.status, c.source_import_job_id, c.created_at, c.updated_at,
		       COALESCE((SELECT COUNT(*) FROM catalog_pages p WHERE p.catalog_id=c.id),0)
		FROM catalogs c
		WHERE c.store_id=$1 AND c.status <> 'archived'
		ORDER BY c.created_at DESC
	`, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Catalog
	for rows.Next() {
		var c Catalog
		if err := rows.Scan(&c.ID, &c.StoreID, &c.Title, &c.Status, &c.SourceImportJobID, &c.CreatedAt, &c.UpdatedAt, &c.PageCount); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	for i := range out {
		if err := s.attachLinks(ctx, &out[i]); err != nil {
			return nil, err
		}
	}
	return out, rows.Err()
}

func (s *Service) Get(ctx context.Context, storeID, catalogID uuid.UUID) (*Catalog, error) {
	var c Catalog
	err := s.DB.QueryRow(ctx, `
		SELECT c.id, c.store_id, c.title, c.status, c.source_import_job_id, c.created_at, c.updated_at,
		       COALESCE((SELECT COUNT(*) FROM catalog_pages p WHERE p.catalog_id=c.id),0)
		FROM catalogs c
		WHERE c.id=$1 AND c.store_id=$2
	`, catalogID, storeID).Scan(
		&c.ID, &c.StoreID, &c.Title, &c.Status, &c.SourceImportJobID, &c.CreatedAt, &c.UpdatedAt, &c.PageCount,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := s.attachLinks(ctx, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

type PatchInput struct {
	Title  *string `json:"title"`
	Status *string `json:"status"`
}

func (s *Service) Patch(ctx context.Context, storeID, catalogID uuid.UUID, in PatchInput) (*Catalog, error) {
	cur, err := s.Get(ctx, storeID, catalogID)
	if err != nil {
		return nil, err
	}
	title := cur.Title
	if in.Title != nil {
		t := strings.TrimSpace(*in.Title)
		if t == "" {
			return nil, fmt.Errorf("%w: title required", ErrInvalidInput)
		}
		title = t
	}
	status := cur.Status
	if in.Status != nil {
		st := strings.ToLower(strings.TrimSpace(*in.Status))
		if st != "draft" && st != "published" && st != "archived" {
			return nil, fmt.Errorf("%w: status must be draft, published, or archived", ErrInvalidInput)
		}
		status = st
	}
	_, err = s.DB.Exec(ctx, `
		UPDATE catalogs SET title=$3, status=$4, updated_at=now() WHERE id=$1 AND store_id=$2
	`, catalogID, storeID, title, status)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, storeID, catalogID)
}

func (s *Service) AttachCampaign(ctx context.Context, storeID, catalogID, campaignID uuid.UUID) error {
	if _, err := s.Get(ctx, storeID, catalogID); err != nil {
		return err
	}
	var ok bool
	if err := s.DB.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM campaigns WHERE id=$1 AND store_id=$2)
	`, campaignID, storeID).Scan(&ok); err != nil || !ok {
		return fmt.Errorf("%w: campaign not found", ErrInvalidInput)
	}
	_, err := s.DB.Exec(ctx, `
		INSERT INTO catalog_campaigns (catalog_id, campaign_id, store_id)
		VALUES ($1,$2,$3)
		ON CONFLICT DO NOTHING
	`, catalogID, campaignID, storeID)
	return err
}

func (s *Service) DetachCampaign(ctx context.Context, storeID, catalogID, campaignID uuid.UUID) error {
	ct, err := s.DB.Exec(ctx, `
		DELETE FROM catalog_campaigns WHERE catalog_id=$1 AND campaign_id=$2 AND store_id=$3
	`, catalogID, campaignID, storeID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) AttachPreorder(ctx context.Context, storeID, catalogID, batchID uuid.UUID) error {
	if _, err := s.Get(ctx, storeID, catalogID); err != nil {
		return err
	}
	var ok bool
	if err := s.DB.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM preorder_batches WHERE id=$1 AND store_id=$2)
	`, batchID, storeID).Scan(&ok); err != nil || !ok {
		return fmt.Errorf("%w: preorder batch not found", ErrInvalidInput)
	}
	_, err := s.DB.Exec(ctx, `
		INSERT INTO catalog_preorder_batches (catalog_id, batch_id, store_id)
		VALUES ($1,$2,$3)
		ON CONFLICT DO NOTHING
	`, catalogID, batchID, storeID)
	return err
}

func (s *Service) DetachPreorder(ctx context.Context, storeID, catalogID, batchID uuid.UUID) error {
	ct, err := s.DB.Exec(ctx, `
		DELETE FROM catalog_preorder_batches WHERE catalog_id=$1 AND batch_id=$2 AND store_id=$3
	`, catalogID, batchID, storeID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) ListForCampaign(ctx context.Context, storeID, campaignID uuid.UUID) ([]Catalog, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT c.id FROM catalogs c
		JOIN catalog_campaigns cc ON cc.catalog_id=c.id
		WHERE c.store_id=$1 AND cc.campaign_id=$2
		ORDER BY c.created_at DESC
	`, storeID, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.collect(ctx, storeID, rows)
}

func (s *Service) ListForPreorder(ctx context.Context, storeID, batchID uuid.UUID) ([]Catalog, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT c.id FROM catalogs c
		JOIN catalog_preorder_batches cp ON cp.catalog_id=c.id
		WHERE c.store_id=$1 AND cp.batch_id=$2
		ORDER BY c.created_at DESC
	`, storeID, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.collect(ctx, storeID, rows)
}

func (s *Service) collect(ctx context.Context, storeID uuid.UUID, rows pgx.Rows) ([]Catalog, error) {
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]Catalog, 0, len(ids))
	for _, id := range ids {
		c, err := s.Get(ctx, storeID, id)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, nil
}

func (s *Service) attachLinks(ctx context.Context, c *Catalog) error {
	rows, err := s.DB.Query(ctx, `
		SELECT campaign_id FROM catalog_campaigns WHERE catalog_id=$1 AND store_id=$2
	`, c.ID, c.StoreID)
	if err != nil {
		return err
	}
	defer rows.Close()
	c.CampaignIDs = nil
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return err
		}
		c.CampaignIDs = append(c.CampaignIDs, id)
	}
	brows, err := s.DB.Query(ctx, `
		SELECT batch_id FROM catalog_preorder_batches WHERE catalog_id=$1 AND store_id=$2
	`, c.ID, c.StoreID)
	if err != nil {
		return err
	}
	defer brows.Close()
	c.PreorderBatchIDs = nil
	for brows.Next() {
		var id uuid.UUID
		if err := brows.Scan(&id); err != nil {
			return err
		}
		c.PreorderBatchIDs = append(c.PreorderBatchIDs, id)
	}
	return brows.Err()
}
