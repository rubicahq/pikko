package account

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrInvalidInput = errors.New("invalid input")

type Account struct {
	ID          uuid.UUID  `json:"id"`
	Email       string     `json:"email"`
	DisplayName *string    `json:"display_name"`
	Locale      string     `json:"locale"`
	Language    string     `json:"language"`
	Timezone    string     `json:"timezone"`
	LastStoreID *uuid.UUID `json:"last_store_id"`
	CreatedAt   time.Time  `json:"created_at"`
}

type Service struct {
	DB *pgxpool.Pool
}

func (s *Service) Get(ctx context.Context, userID uuid.UUID) (*Account, error) {
	var a Account
	err := s.DB.QueryRow(ctx, `
		SELECT u.id, u.email, u.display_name, u.locale, u.last_store_id, u.created_at,
		       COALESCE(p.language, u.locale), COALESCE(p.timezone, 'Asia/Jakarta')
		FROM users u
		LEFT JOIN user_preferences p ON p.user_id = u.id
		WHERE u.id=$1
	`, userID).Scan(&a.ID, &a.Email, &a.DisplayName, &a.Locale, &a.LastStoreID, &a.CreatedAt, &a.Language, &a.Timezone)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvalidInput
	}
	return &a, err
}

type PatchInput struct {
	DisplayName *string `json:"display_name"`
	Language    *string `json:"language"`
	Timezone    *string `json:"timezone"`
}

func (s *Service) Patch(ctx context.Context, userID uuid.UUID, in PatchInput) (*Account, error) {
	if in.DisplayName != nil {
		name := strings.TrimSpace(*in.DisplayName)
		if name != "" {
			runes := []rune(name)
			if len(runes) > 80 {
				return nil, ErrInvalidInput
			}
		}
		var ptr *string
		if name != "" {
			ptr = &name
		}
		_, err := s.DB.Exec(ctx, `UPDATE users SET display_name=$2, updated_at=now() WHERE id=$1`, userID, ptr)
		if err != nil {
			return nil, err
		}
	}
	if in.Language != nil || in.Timezone != nil {
		lang := "id-ID"
		tz := "Asia/Jakarta"
		_ = s.DB.QueryRow(ctx, `SELECT COALESCE(language,'id-ID'), COALESCE(timezone,'Asia/Jakarta') FROM user_preferences WHERE user_id=$1`, userID).Scan(&lang, &tz)
		if in.Language != nil {
			l := strings.TrimSpace(*in.Language)
			if l != "id-ID" && l != "en-US" && l != "en" {
				return nil, ErrInvalidInput
			}
			if l == "en" {
				l = "en-US"
			}
			lang = l
			_, _ = s.DB.Exec(ctx, `UPDATE users SET locale=$2, updated_at=now() WHERE id=$1`, userID, lang)
		}
		if in.Timezone != nil {
			tz = strings.TrimSpace(*in.Timezone)
			if tz == "" {
				return nil, ErrInvalidInput
			}
		}
		_, err := s.DB.Exec(ctx, `
			INSERT INTO user_preferences (user_id, language, timezone, updated_at)
			VALUES ($1,$2,$3,now())
			ON CONFLICT (user_id) DO UPDATE SET language=$2, timezone=$3, updated_at=now()
		`, userID, lang, tz)
		if err != nil {
			return nil, err
		}
	}
	return s.Get(ctx, userID)
}
