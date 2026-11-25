package stores

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rubicahq/pikko/services/api/internal/config"
	"github.com/rubicahq/pikko/services/api/internal/validate"
)

var (
	ErrNotFound       = errors.New("store not found")
	ErrForbidden      = errors.New("forbidden")
	ErrSlugTaken      = errors.New("slug unavailable")
	ErrInvalidInput   = errors.New("invalid input")
	ErrCannotArchive  = errors.New("cannot archive store")
	ErrNoActiveDraft  = errors.New("no active onboarding draft")
)

type Store struct {
	ID                 uuid.UUID  `json:"id"`
	Name               string     `json:"name"`
	Slug               string     `json:"slug"`
	Category           string     `json:"category"`
	SellingModel       string     `json:"selling_model"`
	Currency           string     `json:"currency"`
	Timezone           string     `json:"timezone"`
	Locale             string     `json:"locale"`
	Theme              string     `json:"theme"`
	PublicationStatus  string     `json:"publication_status"`
	CheckoutEnabled    bool       `json:"checkout_enabled"`
	ArchivedAt         *time.Time `json:"archived_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	Role               string     `json:"role,omitempty"`
	Permissions        []string   `json:"permissions,omitempty"`
	StorefrontURL      string     `json:"storefront_url,omitempty"`
	StoreIconFileID    *uuid.UUID `json:"store_icon_file_id,omitempty"`
	StoreIconURL       string     `json:"store_icon_url,omitempty"`
}

type OnboardingDraft struct {
	ID           uuid.UUID `json:"id"`
	Step         int       `json:"step"`
	StoreName    *string   `json:"store_name"`
	StoreSlug    *string   `json:"store_slug"`
	Category     *string   `json:"category"`
	SellingModel *string   `json:"selling_model"`
	PlanCode     *string   `json:"plan_code"`
	BillingCycle *string   `json:"billing_cycle"`
	IsFirstStore bool      `json:"is_first_store"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Service struct {
	DB  *pgxpool.Pool
	Cfg *config.Config
}

func (s *Service) ListForUser(ctx context.Context, userID uuid.UUID, includeArchived bool) ([]Store, error) {
	q := `
		SELECT s.id, s.name, s.slug, s.category, s.selling_model, s.currency, s.timezone, s.locale,
		       s.theme, s.publication_status, s.checkout_enabled, s.archived_at, s.created_at, s.updated_at, m.role,
		       COALESCE(draft.store_icon_file_id, pub.store_icon_file_id) AS store_icon_file_id
		FROM stores s
		JOIN store_memberships m ON m.store_id = s.id
		LEFT JOIN storefront_configs draft ON draft.store_id = s.id AND draft.channel = 'draft'
		LEFT JOIN storefront_configs pub ON pub.store_id = s.id AND pub.channel = 'published'
		WHERE m.user_id=$1 AND m.status='active'
	`
	if !includeArchived {
		q += ` AND s.archived_at IS NULL`
	}
	q += ` ORDER BY s.created_at ASC`
	rows, err := s.DB.Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Store
	for rows.Next() {
		var st Store
		if err := rows.Scan(
			&st.ID, &st.Name, &st.Slug, &st.Category, &st.SellingModel, &st.Currency, &st.Timezone, &st.Locale,
			&st.Theme, &st.PublicationStatus, &st.CheckoutEnabled, &st.ArchivedAt, &st.CreatedAt, &st.UpdatedAt, &st.Role,
			&st.StoreIconFileID,
		); err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	return out, nil
}

func (s *Service) GetAuthorized(ctx context.Context, userID, storeID uuid.UUID) (*Store, error) {
	var st Store
	err := s.DB.QueryRow(ctx, `
		SELECT s.id, s.name, s.slug, s.category, s.selling_model, s.currency, s.timezone, s.locale,
		       s.theme, s.publication_status, s.checkout_enabled, s.archived_at, s.created_at, s.updated_at, m.role,
		       COALESCE(draft.store_icon_file_id, pub.store_icon_file_id) AS store_icon_file_id
		FROM stores s
		JOIN store_memberships m ON m.store_id = s.id
		LEFT JOIN storefront_configs draft ON draft.store_id = s.id AND draft.channel = 'draft'
		LEFT JOIN storefront_configs pub ON pub.store_id = s.id AND pub.channel = 'published'
		WHERE s.id=$1 AND m.user_id=$2 AND m.status='active'
	`, storeID, userID).Scan(
		&st.ID, &st.Name, &st.Slug, &st.Category, &st.SellingModel, &st.Currency, &st.Timezone, &st.Locale,
		&st.Theme, &st.PublicationStatus, &st.CheckoutEnabled, &st.ArchivedAt, &st.CreatedAt, &st.UpdatedAt, &st.Role,
		&st.StoreIconFileID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrForbidden
	}
	if err != nil {
		return nil, err
	}
	return &st, nil
}

func (s *Service) AssertMember(ctx context.Context, userID, storeID uuid.UUID) error {
	_, err := s.GetAuthorized(ctx, userID, storeID)
	return err
}

func (s *Service) CheckSlugAvailable(ctx context.Context, slug string) (bool, error) {
	slug = strings.ToLower(strings.TrimSpace(slug))
	if !validate.ValidateSlug(slug) {
		return false, ErrInvalidInput
	}
	var exists bool
	err := s.DB.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM stores WHERE slug=$1
			UNION ALL
			SELECT 1 FROM store_slug_aliases WHERE slug=$1
		)
	`, slug).Scan(&exists)
	if err != nil {
		return false, err
	}
	return !exists, nil
}

func (s *Service) GetOrCreateDraft(ctx context.Context, userID uuid.UUID) (*OnboardingDraft, error) {
	var count int
	if err := s.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM store_memberships m
		JOIN stores s ON s.id=m.store_id
		WHERE m.user_id=$1 AND m.status='active' AND s.archived_at IS NULL
	`, userID).Scan(&count); err != nil {
		return nil, err
	}
	isFirst := count == 0

	var d OnboardingDraft
	err := s.DB.QueryRow(ctx, `
		SELECT id, step, store_name, store_slug, category, selling_model, plan_code, billing_cycle, is_first_store, updated_at
		FROM onboarding_drafts
		WHERE user_id=$1 AND completed_at IS NULL
	`, userID).Scan(&d.ID, &d.Step, &d.StoreName, &d.StoreSlug, &d.Category, &d.SellingModel, &d.PlanCode, &d.BillingCycle, &d.IsFirstStore, &d.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		err = s.DB.QueryRow(ctx, `
			INSERT INTO onboarding_drafts (user_id, step, is_first_store)
			VALUES ($1, 1, $2)
			ON CONFLICT (user_id) DO UPDATE
			  SET step=1, store_name=NULL, store_slug=NULL, category=NULL, selling_model=NULL,
			      plan_code=NULL, billing_cycle=NULL,
			      is_first_store=EXCLUDED.is_first_store, completed_at=NULL, updated_at=now()
			WHERE onboarding_drafts.completed_at IS NOT NULL
			RETURNING id, step, store_name, store_slug, category, selling_model, plan_code, billing_cycle, is_first_store, updated_at
		`, userID, isFirst).Scan(&d.ID, &d.Step, &d.StoreName, &d.StoreSlug, &d.Category, &d.SellingModel, &d.PlanCode, &d.BillingCycle, &d.IsFirstStore, &d.UpdatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			// Active draft exists from race; re-read
			err = s.DB.QueryRow(ctx, `
				SELECT id, step, store_name, store_slug, category, selling_model, plan_code, billing_cycle, is_first_store, updated_at
				FROM onboarding_drafts WHERE user_id=$1 AND completed_at IS NULL
			`, userID).Scan(&d.ID, &d.Step, &d.StoreName, &d.StoreSlug, &d.Category, &d.SellingModel, &d.PlanCode, &d.BillingCycle, &d.IsFirstStore, &d.UpdatedAt)
		}
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

type IdentityInput struct {
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Category string `json:"category"`
}

func (s *Service) SaveIdentityStep(ctx context.Context, userID uuid.UUID, in IdentityInput) (*OnboardingDraft, error) {
	name := strings.TrimSpace(in.Name)
	slug := strings.ToLower(strings.TrimSpace(in.Slug))
	cat := strings.TrimSpace(in.Category)
	if !validate.ValidateStoreName(name) || !validate.ValidateSlug(slug) {
		return nil, ErrInvalidInput
	}
	if _, ok := validate.ValidCategories[cat]; !ok {
		return nil, ErrInvalidInput
	}
	ok, err := s.CheckSlugAvailable(ctx, slug)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrSlugTaken
	}
	if _, err := s.GetOrCreateDraft(ctx, userID); err != nil {
		return nil, err
	}
	var d OnboardingDraft
	err = s.DB.QueryRow(ctx, `
		UPDATE onboarding_drafts
		SET step=2, store_name=$2, store_slug=$3, category=$4, updated_at=now()
		WHERE user_id=$1 AND completed_at IS NULL
		RETURNING id, step, store_name, store_slug, category, selling_model, plan_code, billing_cycle, is_first_store, updated_at
	`, userID, name, slug, cat).Scan(&d.ID, &d.Step, &d.StoreName, &d.StoreSlug, &d.Category, &d.SellingModel, &d.PlanCode, &d.BillingCycle, &d.IsFirstStore, &d.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoActiveDraft
	}
	return &d, err
}

func (s *Service) SaveSellingModelStep(ctx context.Context, userID uuid.UUID, model string) (*OnboardingDraft, error) {
	if _, ok := validate.ValidSellingModels[model]; !ok {
		return nil, ErrInvalidInput
	}
	var d OnboardingDraft
	err := s.DB.QueryRow(ctx, `
		UPDATE onboarding_drafts
		SET selling_model=$2, step=3, updated_at=now()
		WHERE user_id=$1 AND completed_at IS NULL AND step>=2 AND store_name IS NOT NULL AND store_slug IS NOT NULL
		RETURNING id, step, store_name, store_slug, category, selling_model, plan_code, billing_cycle, is_first_store, updated_at
	`, userID, model).Scan(&d.ID, &d.Step, &d.StoreName, &d.StoreSlug, &d.Category, &d.SellingModel, &d.PlanCode, &d.BillingCycle, &d.IsFirstStore, &d.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvalidInput
	}
	return &d, err
}

func (s *Service) SavePlanStep(ctx context.Context, userID uuid.UUID, planCode, billingCycle string) (*OnboardingDraft, error) {
	planCode = strings.ToLower(strings.TrimSpace(planCode))
	billingCycle = strings.ToLower(strings.TrimSpace(billingCycle))
	if planCode == "" || (billingCycle != "monthly" && billingCycle != "annual") {
		return nil, ErrInvalidInput
	}
	var exists bool
	if err := s.DB.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM subscription_plans WHERE code=$1 AND is_active=true)
	`, planCode).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrInvalidInput
	}
	var d OnboardingDraft
	err := s.DB.QueryRow(ctx, `
		UPDATE onboarding_drafts
		SET plan_code=$2, billing_cycle=$3, step=4, updated_at=now()
		WHERE user_id=$1 AND completed_at IS NULL AND step>=3 AND selling_model IS NOT NULL
		RETURNING id, step, store_name, store_slug, category, selling_model, plan_code, billing_cycle, is_first_store, updated_at
	`, userID, planCode, billingCycle).Scan(
		&d.ID, &d.Step, &d.StoreName, &d.StoreSlug, &d.Category, &d.SellingModel, &d.PlanCode, &d.BillingCycle, &d.IsFirstStore, &d.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvalidInput
	}
	return &d, err
}

type CreateFromDraftResult struct {
	Store        *Store
	PlanCode     string
	BillingCycle string
}

func (s *Service) CreateStoreFromDraft(ctx context.Context, userID uuid.UUID) (*CreateFromDraftResult, error) {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var d OnboardingDraft
	err = tx.QueryRow(ctx, `
		SELECT id, step, store_name, store_slug, category, selling_model, plan_code, billing_cycle, is_first_store, updated_at
		FROM onboarding_drafts
		WHERE user_id=$1 AND completed_at IS NULL
		FOR UPDATE
	`, userID).Scan(&d.ID, &d.Step, &d.StoreName, &d.StoreSlug, &d.Category, &d.SellingModel, &d.PlanCode, &d.BillingCycle, &d.IsFirstStore, &d.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoActiveDraft
	}
	if err != nil {
		return nil, err
	}
	if d.StoreName == nil || d.StoreSlug == nil || d.Category == nil || d.SellingModel == nil {
		return nil, ErrInvalidInput
	}
	// Plan selection required when subscription tables exist (migration 000022+).
	if d.PlanCode == nil || d.BillingCycle == nil {
		return nil, ErrInvalidInput
	}

	var exists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM stores WHERE slug=$1 UNION ALL SELECT 1 FROM store_slug_aliases WHERE slug=$1)
	`, *d.StoreSlug).Scan(&exists); err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrSlugTaken
	}

	var st Store
	err = tx.QueryRow(ctx, `
		INSERT INTO stores (name, slug, category, selling_model, currency, timezone, locale, theme, publication_status, checkout_enabled)
		VALUES ($1,$2,$3,$4,$5,$6,$7,'pikko_signature','draft',false)
		RETURNING id, name, slug, category, selling_model, currency, timezone, locale, theme, publication_status, checkout_enabled, archived_at, created_at, updated_at
	`, *d.StoreName, *d.StoreSlug, *d.Category, *d.SellingModel, s.Cfg.DefaultCurr, s.Cfg.DefaultTZ, s.Cfg.DefaultLocale).Scan(
		&st.ID, &st.Name, &st.Slug, &st.Category, &st.SellingModel, &st.Currency, &st.Timezone, &st.Locale,
		&st.Theme, &st.PublicationStatus, &st.CheckoutEnabled, &st.ArchivedAt, &st.CreatedAt, &st.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO store_memberships (store_id, user_id, role, status)
		VALUES ($1,$2,'owner','active')
	`, st.ID, userID)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO storefront_configs (store_id, channel, theme_id, background_color, text_color, primary_color)
		VALUES ($1,'draft','pikko_signature','#FFFFFF','#15191E','#FF3D00')
		ON CONFLICT (store_id, channel) DO NOTHING
	`, st.ID)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE onboarding_drafts SET completed_at=now(), updated_at=now() WHERE id=$1
	`, d.ID)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `UPDATE users SET last_store_id=$2, updated_at=now() WHERE id=$1`, userID, st.ID)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO audit_logs (actor_user_id, store_id, action, subject_type, subject_id, metadata)
		VALUES ($1,$2,'store_created','store',$3, jsonb_build_object('is_first_store', $4::boolean))
	`, userID, st.ID, st.ID.String(), d.IsFirstStore)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	st.Role = "owner"
	return &CreateFromDraftResult{
		Store:        &st,
		PlanCode:     *d.PlanCode,
		BillingCycle: *d.BillingCycle,
	}, nil
}

type CreateDirectInput struct {
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	Category     string `json:"category"`
	SellingModel string `json:"selling_model"`
	PlanCode     string `json:"plan_code"`
	BillingCycle string `json:"billing_cycle"`
}

// CreateStore creates via the draft flow atomically when all fields provided (additional stores).
func (s *Service) CreateStore(ctx context.Context, userID uuid.UUID, in CreateDirectInput) (*Store, error) {
	name := strings.TrimSpace(in.Name)
	slug := strings.ToLower(strings.TrimSpace(in.Slug))
	if !validate.ValidateStoreName(name) || !validate.ValidateSlug(slug) {
		return nil, ErrInvalidInput
	}
	if _, ok := validate.ValidCategories[in.Category]; !ok {
		return nil, ErrInvalidInput
	}
	if _, ok := validate.ValidSellingModels[in.SellingModel]; !ok {
		return nil, ErrInvalidInput
	}
	ok, err := s.CheckSlugAvailable(ctx, slug)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrSlugTaken
	}

	// Reuse draft path for resume-safe creation
	if _, err := s.SaveIdentityStep(ctx, userID, IdentityInput{Name: name, Slug: slug, Category: in.Category}); err != nil {
		return nil, err
	}
	if _, err := s.SaveSellingModelStep(ctx, userID, in.SellingModel); err != nil {
		return nil, err
	}
	if _, err := s.SavePlanStep(ctx, userID, in.PlanCode, in.BillingCycle); err != nil {
		return nil, err
	}
	res, err := s.CreateStoreFromDraft(ctx, userID)
	if err != nil {
		return nil, err
	}
	return res.Store, nil
}

// DraftPlan returns plan selection from the completed draft just before completion, or from active draft.
func (s *Service) LastDraftPlan(ctx context.Context, userID uuid.UUID) (planCode, billingCycle string, err error) {
	err = s.DB.QueryRow(ctx, `
		SELECT COALESCE(plan_code,''), COALESCE(billing_cycle,'')
		FROM onboarding_drafts WHERE user_id=$1
		ORDER BY updated_at DESC LIMIT 1
	`, userID).Scan(&planCode, &billingCycle)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrNoActiveDraft
	}
	return planCode, billingCycle, err
}

type PatchInput struct {
	Name         *string `json:"name"`
	Category     *string `json:"category"`
	SellingModel *string `json:"selling_model"`
	DisplayName  *string `json:"-"`
}

func (s *Service) Patch(ctx context.Context, userID, storeID uuid.UUID, in PatchInput) (*Store, error) {
	stAuth, err := s.GetAuthorized(ctx, userID, storeID)
	if err != nil {
		return nil, err
	}
	_ = stAuth
	sets := []string{"updated_at=now()"}
	args := []any{storeID}
	n := 2
	if in.Name != nil {
		if !validate.ValidateStoreName(*in.Name) {
			return nil, ErrInvalidInput
		}
		sets = append(sets, fmt.Sprintf("name=$%d", n))
		args = append(args, strings.TrimSpace(*in.Name))
		n++
	}
	if in.Category != nil {
		if _, ok := validate.ValidCategories[*in.Category]; !ok {
			return nil, ErrInvalidInput
		}
		sets = append(sets, fmt.Sprintf("category=$%d", n))
		args = append(args, *in.Category)
		n++
	}
	if in.SellingModel != nil {
		if _, ok := validate.ValidSellingModels[*in.SellingModel]; !ok {
			return nil, ErrInvalidInput
		}
		sets = append(sets, fmt.Sprintf("selling_model=$%d", n))
		args = append(args, *in.SellingModel)
		n++
	}
	q := fmt.Sprintf(`UPDATE stores SET %s WHERE id=$1`, strings.Join(sets, ", "))
	if _, err := s.DB.Exec(ctx, q, args...); err != nil {
		return nil, err
	}
	return s.GetAuthorized(ctx, userID, storeID)
}

func (s *Service) Archive(ctx context.Context, userID, storeID uuid.UUID) (*Store, error) {
	st, err := s.GetAuthorized(ctx, userID, storeID)
	if err != nil {
		return nil, err
	}
	if st.Role != "owner" {
		return nil, ErrForbidden
	}
	if st.ArchivedAt != nil {
		return st, nil
	}
	// Phase 1: no unresolved commerce transactions yet — archive freely.
	_, err = s.DB.Exec(ctx, `
		UPDATE stores SET archived_at=now(), publication_status='archived', checkout_enabled=false, updated_at=now()
		WHERE id=$1
	`, storeID)
	if err != nil {
		return nil, err
	}
	_, _ = s.DB.Exec(ctx, `
		INSERT INTO audit_logs (actor_user_id, store_id, action, subject_type, subject_id)
		VALUES ($1,$2,'store_archived','store',$3)
	`, userID, storeID, storeID.String())
	_, _ = s.DB.Exec(ctx, `
		UPDATE users SET last_store_id=NULL, updated_at=now()
		WHERE id=$1 AND last_store_id=$2
	`, userID, storeID)
	return s.GetAuthorized(ctx, userID, storeID)
}

func (s *Service) TouchLastAccessed(ctx context.Context, userID, storeID uuid.UUID) error {
	if err := s.AssertMember(ctx, userID, storeID); err != nil {
		return err
	}
	_, err := s.DB.Exec(ctx, `UPDATE users SET last_store_id=$2, updated_at=now() WHERE id=$1`, userID, storeID)
	return err
}

func (s *Service) SuggestSlug(ctx context.Context, name string) (string, error) {
	base := validate.SuggestSlug(name)
	candidate := base
	for i := 0; i < 50; i++ {
		ok, err := s.CheckSlugAvailable(ctx, candidate)
		if err != nil {
			return "", err
		}
		if ok {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, i+2)
		if len(candidate) > 50 {
			candidate = candidate[:50]
			candidate = strings.Trim(candidate, "-")
		}
	}
	return "", ErrSlugTaken
}
