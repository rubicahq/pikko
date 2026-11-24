package storefront

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rubicahq/pikko/services/api/internal/themes"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrForbidden    = errors.New("forbidden")
)

type Config struct {
	ID               uuid.UUID       `json:"id"`
	StoreID          uuid.UUID       `json:"store_id"`
	Channel          string          `json:"channel"`
	ThemeID          string          `json:"theme_id"`
	BackgroundColor  string          `json:"background_color"`
	TextColor        string          `json:"text_color"`
	PrimaryColor     string          `json:"primary_color"`
	ButtonForeground string          `json:"button_foreground"`
	SurfaceColor     string          `json:"surface_color"`
	BorderColor      string          `json:"border_color"`
	LogoFileID       *uuid.UUID      `json:"logo_file_id,omitempty"`
	FaviconFileID    *uuid.UUID      `json:"favicon_file_id,omitempty"`
	StoreIconFileID  *uuid.UUID      `json:"store_icon_file_id,omitempty"`
	LogoURL          string          `json:"logo_url,omitempty"`
	FaviconURL       string          `json:"favicon_url,omitempty"`
	StoreIconURL     string          `json:"store_icon_url,omitempty"`
	Description      *string         `json:"description,omitempty"`
	ContactEmail     *string         `json:"contact_email,omitempty"`
	WhatsApp         *string         `json:"whatsapp,omitempty"`
	SocialLinks      json.RawMessage `json:"social_links"`
	UpdatedAt        time.Time       `json:"updated_at"`
	PublishedAt      *time.Time      `json:"published_at,omitempty"`
	ThemeName        string          `json:"theme_name,omitempty"`
}

type Banner struct {
	ID              uuid.UUID  `json:"id"`
	Headline        *string    `json:"headline,omitempty"`
	SupportingText  *string    `json:"supporting_text,omitempty"`
	CTALabel        *string    `json:"cta_label,omitempty"`
	DestinationType *string    `json:"destination_type,omitempty"`
	DestinationRef  *string    `json:"destination_ref,omitempty"`
	DisplayOrder    int        `json:"display_order"`
	IsActive        bool       `json:"is_active"`
	DesktopFileID   *uuid.UUID `json:"desktop_file_id,omitempty"`
	MobileFileID    *uuid.UUID `json:"mobile_file_id,omitempty"`
}

type Service struct {
	DB *pgxpool.Pool
}

func (s *Service) EnsureDraft(ctx context.Context, storeID uuid.UUID) (*Config, error) {
	th := themes.MustGet("pikko_signature")
	_, err := s.DB.Exec(ctx, `
		INSERT INTO storefront_configs (store_id, channel, theme_id, background_color, text_color, primary_color)
		VALUES ($1,'draft',$2,$3,$4,$5)
		ON CONFLICT (store_id, channel) DO NOTHING
	`, storeID, th.ID, th.BackgroundColor, th.TextColor, th.PrimaryColor)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, storeID, "draft")
}

func (s *Service) Get(ctx context.Context, storeID uuid.UUID, channel string) (*Config, error) {
	if channel != "draft" && channel != "published" {
		return nil, ErrInvalidInput
	}
	var c Config
	var social []byte
	err := s.DB.QueryRow(ctx, `
		SELECT id, store_id, channel, theme_id, background_color, text_color, primary_color,
		       logo_file_id, favicon_file_id, store_icon_file_id, description, contact_email, whatsapp,
		       social_links, updated_at, published_at
		FROM storefront_configs WHERE store_id=$1 AND channel=$2
	`, storeID, channel).Scan(
		&c.ID, &c.StoreID, &c.Channel, &c.ThemeID, &c.BackgroundColor, &c.TextColor, &c.PrimaryColor,
		&c.LogoFileID, &c.FaviconFileID, &c.StoreIconFileID, &c.Description, &c.ContactEmail, &c.WhatsApp,
		&social, &c.UpdatedAt, &c.PublishedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		if channel == "draft" {
			return s.EnsureDraft(ctx, storeID)
		}
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	c.SocialLinks = social
	s.decorate(&c)
	return &c, nil
}

func (s *Service) decorate(c *Config) {
	th, err := themes.ApplyThemeColors(c.ThemeID, &c.BackgroundColor, &c.TextColor, &c.PrimaryColor)
	if err != nil {
		th = themes.MustGet(c.ThemeID)
	}
	c.ButtonForeground = th.ButtonForeground
	c.SurfaceColor = th.SurfaceColor
	c.BorderColor = th.BorderColor
	c.ThemeName = th.Name
	pathFn := fileAPIPath
	if c.Channel == "published" {
		pathFn = publicFileAPIPath
	}
	if c.LogoFileID != nil {
		c.LogoURL = pathFn(c.StoreID, *c.LogoFileID)
	}
	if c.FaviconFileID != nil {
		c.FaviconURL = pathFn(c.StoreID, *c.FaviconFileID)
	}
	if c.StoreIconFileID != nil {
		c.StoreIconURL = pathFn(c.StoreID, *c.StoreIconFileID)
	}
}

func fileAPIPath(storeID, fileID uuid.UUID) string {
	return "/api/v1/stores/" + storeID.String() + "/files/" + fileID.String()
}

func publicFileAPIPath(storeID, fileID uuid.UUID) string {
	return "/api/v1/public/files/" + storeID.String() + "/" + fileID.String()
}

type PatchInput struct {
	ThemeID         *string          `json:"theme_id"`
	BackgroundColor *string          `json:"background_color"`
	TextColor       *string          `json:"text_color"`
	PrimaryColor    *string          `json:"primary_color"`
	LogoFileID      *uuid.UUID       `json:"logo_file_id"`
	ClearLogo       bool             `json:"clear_logo"`
	FaviconFileID   *uuid.UUID       `json:"favicon_file_id"`
	ClearFavicon    bool             `json:"clear_favicon"`
	StoreIconFileID *uuid.UUID       `json:"store_icon_file_id"`
	ClearStoreIcon  bool             `json:"clear_store_icon"`
	Description     *string          `json:"description"`
	ContactEmail    *string          `json:"contact_email"`
	WhatsApp        *string          `json:"whatsapp"`
	SocialLinks     *json.RawMessage `json:"social_links"`
}

func (s *Service) PatchDraft(ctx context.Context, storeID uuid.UUID, in PatchInput) (*Config, error) {
	if _, err := s.EnsureDraft(ctx, storeID); err != nil {
		return nil, err
	}
	cur, err := s.Get(ctx, storeID, "draft")
	if err != nil {
		return nil, err
	}

	themeID := cur.ThemeID
	bg, text, primary := cur.BackgroundColor, cur.TextColor, cur.PrimaryColor
	if in.ThemeID != nil {
		th, ok := themes.Get(*in.ThemeID)
		if !ok {
			return nil, ErrInvalidInput
		}
		themeID = th.ID
		// Applying a theme resets colors to curated defaults unless custom colors also provided.
		bg, text, primary = th.BackgroundColor, th.TextColor, th.PrimaryColor
	}
	if in.BackgroundColor != nil {
		bg, err = themes.NormalizeHex(*in.BackgroundColor)
		if err != nil {
			return nil, ErrInvalidInput
		}
	}
	if in.TextColor != nil {
		text, err = themes.NormalizeHex(*in.TextColor)
		if err != nil {
			return nil, ErrInvalidInput
		}
	}
	if in.PrimaryColor != nil {
		primary, err = themes.NormalizeHex(*in.PrimaryColor)
		if err != nil {
			return nil, ErrInvalidInput
		}
	}

	logo := cur.LogoFileID
	favicon := cur.FaviconFileID
	icon := cur.StoreIconFileID
	if in.ClearLogo {
		logo = nil
	} else if in.LogoFileID != nil {
		if err := s.assertFile(ctx, storeID, *in.LogoFileID, "logo"); err != nil {
			return nil, err
		}
		logo = in.LogoFileID
	}
	if in.ClearFavicon {
		favicon = nil
	} else if in.FaviconFileID != nil {
		if err := s.assertFile(ctx, storeID, *in.FaviconFileID, "favicon"); err != nil {
			return nil, err
		}
		favicon = in.FaviconFileID
	}
	if in.ClearStoreIcon {
		icon = nil
	} else if in.StoreIconFileID != nil {
		if err := s.assertFile(ctx, storeID, *in.StoreIconFileID, "store_icon"); err != nil {
			return nil, err
		}
		icon = in.StoreIconFileID
	}

	desc := cur.Description
	if in.Description != nil {
		d := strings.TrimSpace(*in.Description)
		if len(d) > 2000 {
			return nil, ErrInvalidInput
		}
		desc = &d
	}
	email := cur.ContactEmail
	if in.ContactEmail != nil {
		e := strings.TrimSpace(strings.ToLower(*in.ContactEmail))
		if e != "" && !strings.Contains(e, "@") {
			return nil, ErrInvalidInput
		}
		email = &e
	}
	wa := cur.WhatsApp
	if in.WhatsApp != nil {
		w := strings.TrimSpace(*in.WhatsApp)
		wa = &w
	}
	social := cur.SocialLinks
	if in.SocialLinks != nil {
		if !json.Valid(*in.SocialLinks) {
			return nil, ErrInvalidInput
		}
		social = *in.SocialLinks
	}

	_, err = s.DB.Exec(ctx, `
		UPDATE storefront_configs SET
			theme_id=$3, background_color=$4, text_color=$5, primary_color=$6,
			logo_file_id=$7, favicon_file_id=$8, store_icon_file_id=$9,
			description=$10, contact_email=$11, whatsapp=$12, social_links=$13,
			updated_at=now()
		WHERE store_id=$1 AND channel=$2
	`, storeID, "draft", themeID, bg, text, primary, logo, favicon, icon, desc, email, wa, social)
	if err != nil {
		return nil, err
	}
	_, _ = s.DB.Exec(ctx, `UPDATE stores SET theme=$2, updated_at=now() WHERE id=$1`, storeID, themeID)
	return s.Get(ctx, storeID, "draft")
}

func (s *Service) assertFile(ctx context.Context, storeID, fileID uuid.UUID, purpose string) error {
	var p string
	err := s.DB.QueryRow(ctx, `SELECT purpose FROM stored_files WHERE id=$1 AND store_id=$2`, fileID, storeID).Scan(&p)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInvalidInput
	}
	if err != nil {
		return err
	}
	if p != purpose {
		return ErrInvalidInput
	}
	return nil
}

func (s *Service) Publish(ctx context.Context, storeID, actor uuid.UUID) (*Config, error) {
	draft, err := s.Get(ctx, storeID, "draft")
	if err != nil {
		return nil, err
	}
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO storefront_configs (
			store_id, channel, theme_id, background_color, text_color, primary_color,
			logo_file_id, favicon_file_id, store_icon_file_id, description, contact_email, whatsapp,
			social_links, updated_at, published_at
		) VALUES (
			$1,'published',$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,now(),now()
		)
		ON CONFLICT (store_id, channel) DO UPDATE SET
			theme_id=EXCLUDED.theme_id,
			background_color=EXCLUDED.background_color,
			text_color=EXCLUDED.text_color,
			primary_color=EXCLUDED.primary_color,
			logo_file_id=EXCLUDED.logo_file_id,
			favicon_file_id=EXCLUDED.favicon_file_id,
			store_icon_file_id=EXCLUDED.store_icon_file_id,
			description=EXCLUDED.description,
			contact_email=EXCLUDED.contact_email,
			whatsapp=EXCLUDED.whatsapp,
			social_links=EXCLUDED.social_links,
			updated_at=now(),
			published_at=now()
	`, storeID, draft.ThemeID, draft.BackgroundColor, draft.TextColor, draft.PrimaryColor,
		draft.LogoFileID, draft.FaviconFileID, draft.StoreIconFileID, draft.Description,
		draft.ContactEmail, draft.WhatsApp, draft.SocialLinks)
	if err != nil {
		return nil, err
	}

	// Copy featured products
	_, err = tx.Exec(ctx, `DELETE FROM storefront_featured_products WHERE store_id=$1 AND channel='published'`, storeID)
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO storefront_featured_products (store_id, channel, product_id, display_order)
		SELECT store_id, 'published', product_id, display_order
		FROM storefront_featured_products WHERE store_id=$1 AND channel='draft'
	`, storeID)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `DELETE FROM storefront_featured_collections WHERE store_id=$1 AND channel='published'`, storeID)
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO storefront_featured_collections (store_id, channel, collection_id, display_order)
		SELECT store_id, 'published', collection_id, display_order
		FROM storefront_featured_collections WHERE store_id=$1 AND channel='draft'
	`, storeID)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `DELETE FROM storefront_banners WHERE store_id=$1 AND channel='published'`, storeID)
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO storefront_banners (
			store_id, channel, desktop_file_id, mobile_file_id, headline, supporting_text,
			cta_label, destination_type, destination_ref, display_order, is_active
		)
		SELECT store_id, 'published', desktop_file_id, mobile_file_id, headline, supporting_text,
			cta_label, destination_type, destination_ref, display_order, is_active
		FROM storefront_banners WHERE store_id=$1 AND channel='draft'
	`, storeID)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE stores SET publication_status='published', theme=$2, updated_at=now()
		WHERE id=$1 AND archived_at IS NULL
	`, storeID, draft.ThemeID)
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO audit_logs (actor_user_id, store_id, action, subject_type, subject_id)
		VALUES ($1,$2,'storefront_published','store',$3)
	`, actor, storeID, storeID.String())
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.Get(ctx, storeID, "published")
}

func (s *Service) DiscardDraft(ctx context.Context, storeID uuid.UUID) (*Config, error) {
	pub, err := s.Get(ctx, storeID, "published")
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			// Reset draft to theme defaults
			th := themes.MustGet("pikko_signature")
			_, err = s.DB.Exec(ctx, `
				UPDATE storefront_configs SET
					theme_id=$2, background_color=$3, text_color=$4, primary_color=$5,
					logo_file_id=NULL, favicon_file_id=NULL, store_icon_file_id=NULL,
					description=NULL, contact_email=NULL, whatsapp=NULL, social_links='{}'::jsonb,
					updated_at=now()
				WHERE store_id=$1 AND channel='draft'
			`, storeID, th.ID, th.BackgroundColor, th.TextColor, th.PrimaryColor)
			if err != nil {
				return nil, err
			}
			return s.Get(ctx, storeID, "draft")
		}
		return nil, err
	}
	_, err = s.DB.Exec(ctx, `
		UPDATE storefront_configs SET
			theme_id=$2, background_color=$3, text_color=$4, primary_color=$5,
			logo_file_id=$6, favicon_file_id=$7, store_icon_file_id=$8,
			description=$9, contact_email=$10, whatsapp=$11, social_links=$12,
			updated_at=now()
		WHERE store_id=$1 AND channel='draft'
	`, storeID, pub.ThemeID, pub.BackgroundColor, pub.TextColor, pub.PrimaryColor,
		pub.LogoFileID, pub.FaviconFileID, pub.StoreIconFileID, pub.Description,
		pub.ContactEmail, pub.WhatsApp, pub.SocialLinks)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, storeID, "draft")
}

type FeaturedUpdate struct {
	ProductIDs    []uuid.UUID `json:"product_ids"`
	CollectionIDs []uuid.UUID `json:"collection_ids"`
}

func (s *Service) SetFeatured(ctx context.Context, storeID uuid.UUID, in FeaturedUpdate) error {
	if len(in.ProductIDs) > 8 {
		return ErrInvalidInput
	}
	if len(in.CollectionIDs) > 4 {
		return ErrInvalidInput
	}
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM storefront_featured_products WHERE store_id=$1 AND channel='draft'`, storeID); err != nil {
		return err
	}
	for i, pid := range in.ProductIDs {
		var ok bool
		err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM products WHERE id=$1 AND store_id=$2 AND status='published')`, pid, storeID).Scan(&ok)
		if err != nil || !ok {
			return ErrInvalidInput
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO storefront_featured_products (store_id, channel, product_id, display_order)
			VALUES ($1,'draft',$2,$3)
		`, storeID, pid, i); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM storefront_featured_collections WHERE store_id=$1 AND channel='draft'`, storeID); err != nil {
		return err
	}
	for i, cid := range in.CollectionIDs {
		var ok bool
		err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM collections WHERE id=$1 AND store_id=$2 AND status='published')`, cid, storeID).Scan(&ok)
		if err != nil || !ok {
			return ErrInvalidInput
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO storefront_featured_collections (store_id, channel, collection_id, display_order)
			VALUES ($1,'draft',$2,$3)
		`, storeID, cid, i); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Service) ListFeaturedProductIDs(ctx context.Context, storeID uuid.UUID, channel string) ([]uuid.UUID, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT product_id FROM storefront_featured_products
		WHERE store_id=$1 AND channel=$2 ORDER BY display_order
	`, storeID, channel)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}

// PublicBySlug returns published store + storefront for public browsing.
type PublicStore struct {
	ID                uuid.UUID `json:"id"`
	Name              string    `json:"name"`
	Slug              string    `json:"slug"`
	PublicationStatus string    `json:"publication_status"`
	CheckoutEnabled   bool      `json:"checkout_enabled"`
	Currency          string    `json:"currency"`
	Locale            string    `json:"locale"`
	Timezone          string    `json:"timezone"`
	Config            *Config   `json:"storefront"`
}

func (s *Service) PublicBySlug(ctx context.Context, slug string) (*PublicStore, error) {
	slug = strings.ToLower(strings.TrimSpace(slug))
	var ps PublicStore
	err := s.DB.QueryRow(ctx, `
		SELECT id, name, slug, publication_status, checkout_enabled, currency, locale, timezone
		FROM stores
		WHERE slug=$1 AND archived_at IS NULL AND publication_status='published'
	`, slug).Scan(&ps.ID, &ps.Name, &ps.Slug, &ps.PublicationStatus, &ps.CheckoutEnabled, &ps.Currency, &ps.Locale, &ps.Timezone)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	cfg, err := s.Get(ctx, ps.ID, "published")
	if err != nil {
		return nil, err
	}
	ps.Config = cfg
	return &ps, nil
}

// PreviewBySlug allows merchants to preview draft via authenticated API; public preview uses draft flag with auth.
func (s *Service) DraftPreviewAuthorized(ctx context.Context, storeID uuid.UUID) (*Config, error) {
	return s.Get(ctx, storeID, "draft")
}
