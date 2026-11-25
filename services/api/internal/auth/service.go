package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rubicahq/pikko/services/api/internal/config"
	"github.com/rubicahq/pikko/services/api/internal/email"
	"github.com/rubicahq/pikko/services/api/internal/urls"
	"github.com/rubicahq/pikko/services/api/internal/validate"
)

var (
	ErrRateLimited     = errors.New("rate limited")
	ErrCooldown        = errors.New("resend cooldown")
	ErrInvalidToken    = errors.New("invalid or expired token")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrInvalidEmail    = errors.New("invalid email")
)

type Service struct {
	DB     *pgxpool.Pool
	Cfg    *config.Config
	Email  *email.Service
	URLs   *urls.Generator
	Logger *slog.Logger
}

type User struct {
	ID           uuid.UUID  `json:"id"`
	Email        string     `json:"email"`
	DisplayName  *string    `json:"display_name"`
	Locale       string     `json:"locale"`
	LastStoreID  *uuid.UUID `json:"last_store_id"`
	CreatedAt    time.Time  `json:"created_at"`
}

type Session struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	LastSeen  time.Time `json:"last_seen_at"`
	UserAgent *string   `json:"user_agent,omitempty"`
	Current   bool      `json:"current,omitempty"`
}

type RequestResult struct {
	Email           string `json:"email"`
	CooldownSeconds int    `json:"cooldown_seconds"`
	// DevToken is only returned when APP_ENV is development/test and EMAIL_MODE=log.
	DevToken string `json:"dev_token,omitempty"`
}

func (s *Service) RequestMagicLink(ctx context.Context, rawEmail, ip, ua string) (*RequestResult, error) {
	return s.requestMagicLink(ctx, rawEmail, ip, ua, func(raw string) string {
		return s.URLs.MagicLink(raw)
	})
}

// RequestMagicLinkHQ sends a magic link that confirms on the HQ host.
func (s *Service) RequestMagicLinkHQ(ctx context.Context, rawEmail, ip, ua string) (*RequestResult, error) {
	return s.requestMagicLink(ctx, rawEmail, ip, ua, func(raw string) string {
		return s.URLs.MagicLinkHQ(raw)
	})
}

func (s *Service) requestMagicLink(ctx context.Context, rawEmail, ip, ua string, confirm func(raw string) string) (*RequestResult, error) {
	display, normalized, ok := validate.NormalizeEmail(rawEmail)
	if !ok {
		return nil, ErrInvalidEmail
	}

	if err := s.checkRateLimit(ctx, "email:"+normalized, s.Cfg.RateLimitEmailHour); err != nil {
		return nil, err
	}
	if ip != "" {
		if err := s.checkRateLimit(ctx, "ip:"+ip, s.Cfg.RateLimitIPHour); err != nil {
			return nil, err
		}
	}

	var lastCreated time.Time
	err := s.DB.QueryRow(ctx, `
		SELECT created_at FROM authentication_tokens
		WHERE email_normalized=$1
		ORDER BY created_at DESC LIMIT 1
	`, normalized).Scan(&lastCreated)
	if err == nil && time.Since(lastCreated) < s.Cfg.ResendCooldown {
		return nil, ErrCooldown
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	raw, hash, err := NewToken()
	if err != nil {
		return nil, err
	}
	expires := time.Now().UTC().Add(s.Cfg.MagicLinkTTL)

	var ipPtr *net.IP
	if ip != "" {
		parsed := net.ParseIP(ip)
		if parsed != nil {
			ipPtr = &parsed
		}
	}

	_, err = s.DB.Exec(ctx, `
		INSERT INTO authentication_tokens (email_normalized, token_hash, expires_at, request_ip, user_agent)
		VALUES ($1,$2,$3,$4,$5)
	`, normalized, hash, expires, ipPtr, nullStr(ua))
	if err != nil {
		return nil, err
	}

	confirmURL := confirm(raw)
	subject, html, text := email.MagicLinkContent(s.Cfg.EmailFromName, confirmURL, int(s.Cfg.MagicLinkTTL.Minutes()))
	msg := email.Message{
		TemplateKey:     "magic_link",
		To:              display,
		From:            s.Cfg.EmailAccountFrom,
		FromName:        s.Cfg.EmailFromName,
		ReplyTo:         s.Cfg.EmailReplyTo,
		Subject:         subject,
		HTML:            html,
		Text:            text,
		IdempotencyKey:  "magic:" + normalized + ":" + fmt.Sprintf("%d", expires.Unix()),
		DisableTracking: true,
	}
	if err := s.Email.SendNow(ctx, msg); err != nil {
		s.Logger.Error("magic_link_email_failed", "error", err)
		return nil, fmt.Errorf("failed to send email")
	}

	res := &RequestResult{
		Email:           display,
		CooldownSeconds: int(s.Cfg.ResendCooldown.Seconds()),
	}
	if (s.Cfg.AppEnv == "development" || s.Cfg.AppEnv == "test") && s.Cfg.EmailMode == "log" {
		res.DevToken = raw
	}
	return res, nil
}

// PeekToken validates a token without consuming it (for GET confirmation page).
func (s *Service) PeekToken(ctx context.Context, raw string) (email string, ok bool) {
	if raw == "" {
		return "", false
	}
	hash := HashToken(raw)
	var normalized string
	var expires time.Time
	var consumed *time.Time
	err := s.DB.QueryRow(ctx, `
		SELECT email_normalized, expires_at, consumed_at
		FROM authentication_tokens WHERE token_hash=$1
	`, hash).Scan(&normalized, &expires, &consumed)
	if err != nil || consumed != nil || time.Now().UTC().After(expires) {
		return "", false
	}
	return normalized, true
}

type VerifyResult struct {
	User          User   `json:"user"`
	SessionToken  string `json:"-"`
	IsNewAccount  bool   `json:"is_new_account"`
	RedirectPath  string `json:"redirect_path"`
}

func (s *Service) VerifyAndConsume(ctx context.Context, raw, ip, ua string) (*VerifyResult, error) {
	if raw == "" {
		return nil, ErrInvalidToken
	}
	hash := HashToken(raw)

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var tokenID uuid.UUID
	var normalized string
	var expires time.Time
	var consumed *time.Time
	err = tx.QueryRow(ctx, `
		SELECT id, email_normalized, expires_at, consumed_at
		FROM authentication_tokens
		WHERE token_hash=$1
		FOR UPDATE
	`, hash).Scan(&tokenID, &normalized, &expires, &consumed)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidToken
		}
		return nil, err
	}
	if consumed != nil || time.Now().UTC().After(expires) {
		return nil, ErrInvalidToken
	}

	tag, err := tx.Exec(ctx, `
		UPDATE authentication_tokens
		SET consumed_at=now()
		WHERE id=$1 AND consumed_at IS NULL AND expires_at > now()
	`, tokenID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() != 1 {
		return nil, ErrInvalidToken
	}

	var user User
	isNew := false
	err = tx.QueryRow(ctx, `
		SELECT id, email, display_name, locale, last_store_id, created_at
		FROM users WHERE email_normalized=$1
	`, normalized).Scan(&user.ID, &user.Email, &user.DisplayName, &user.Locale, &user.LastStoreID, &user.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		isNew = true
		err = tx.QueryRow(ctx, `
			INSERT INTO users (email, email_normalized, locale)
			VALUES ($1,$2,$3)
			RETURNING id, email, display_name, locale, last_store_id, created_at
		`, normalized, normalized, s.Cfg.DefaultLocale).Scan(
			&user.ID, &user.Email, &user.DisplayName, &user.Locale, &user.LastStoreID, &user.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO user_preferences (user_id, language, timezone)
			VALUES ($1,$2,$3)
			ON CONFLICT (user_id) DO NOTHING
		`, user.ID, s.Cfg.DefaultLocale, s.Cfg.DefaultTZ)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	sessionRaw, sessionHash, err := NewToken()
	if err != nil {
		return nil, err
	}
	sessionExpires := time.Now().UTC().Add(s.Cfg.SessionTTL)
	var ipPtr *net.IP
	if ip != "" {
		parsed := net.ParseIP(ip)
		if parsed != nil {
			ipPtr = &parsed
		}
	}
	var sessionID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO sessions (user_id, token_hash, expires_at, created_ip, last_ip, user_agent)
		VALUES ($1,$2,$3,$4,$4,$5)
		RETURNING id
	`, user.ID, sessionHash, sessionExpires, ipPtr, nullStr(ua)).Scan(&sessionID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	redirect, err := s.resolvePostAuthRedirect(ctx, user)
	if err != nil {
		s.Logger.Warn("redirect_resolve_failed", "error", err)
		redirect = "/stores"
	}

	return &VerifyResult{
		User:         user,
		SessionToken: sessionRaw,
		IsNewAccount: isNew,
		RedirectPath: redirect,
	}, nil
}

func (s *Service) resolvePostAuthRedirect(ctx context.Context, user User) (string, error) {
	// Resume unfinished onboarding
	var draftID uuid.UUID
	err := s.DB.QueryRow(ctx, `
		SELECT id FROM onboarding_drafts
		WHERE user_id=$1 AND completed_at IS NULL
		LIMIT 1
	`, user.ID).Scan(&draftID)
	if err == nil {
		return "/onboarding", nil
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}

	type storeRow struct {
		ID uuid.UUID
	}
	rows, err := s.DB.Query(ctx, `
		SELECT s.id
		FROM stores s
		JOIN store_memberships m ON m.store_id = s.id
		WHERE m.user_id=$1 AND m.status='active' AND s.archived_at IS NULL
		ORDER BY s.created_at ASC
	`, user.ID)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var stores []storeRow
	for rows.Next() {
		var r storeRow
		if err := rows.Scan(&r.ID); err != nil {
			return "", err
		}
		stores = append(stores, r)
	}
	if len(stores) == 0 {
		return "/onboarding", nil
	}
	if len(stores) == 1 {
		return fmt.Sprintf("/stores/%s/overview", stores[0].ID), nil
	}
	if user.LastStoreID != nil {
		for _, st := range stores {
			if st.ID == *user.LastStoreID {
				return fmt.Sprintf("/stores/%s/overview", st.ID), nil
			}
		}
		return "/stores", nil
	}
	return fmt.Sprintf("/stores/%s/overview", stores[0].ID), nil
}

func (s *Service) Authenticate(ctx context.Context, sessionRaw string) (*User, *Session, error) {
	if sessionRaw == "" {
		return nil, nil, ErrUnauthorized
	}
	hash := HashToken(sessionRaw)
	var sess Session
	var user User
	err := s.DB.QueryRow(ctx, `
		SELECT se.id, se.user_id, se.expires_at, se.created_at, se.last_seen_at, se.user_agent,
		       u.id, u.email, u.display_name, u.locale, u.last_store_id, u.created_at
		FROM sessions se
		JOIN users u ON u.id = se.user_id
		WHERE se.token_hash=$1 AND se.revoked_at IS NULL AND se.expires_at > now()
	`, hash).Scan(
		&sess.ID, &sess.UserID, &sess.ExpiresAt, &sess.CreatedAt, &sess.LastSeen, &sess.UserAgent,
		&user.ID, &user.Email, &user.DisplayName, &user.Locale, &user.LastStoreID, &user.CreatedAt,
	)
	if err != nil {
		return nil, nil, ErrUnauthorized
	}
	_, _ = s.DB.Exec(ctx, `UPDATE sessions SET last_seen_at=now() WHERE id=$1`, sess.ID)
	sess.Current = true
	return &user, &sess, nil
}

func (s *Service) Logout(ctx context.Context, sessionRaw string) error {
	if sessionRaw == "" {
		return nil
	}
	hash := HashToken(sessionRaw)
	_, err := s.DB.Exec(ctx, `
		UPDATE sessions SET revoked_at=now() WHERE token_hash=$1 AND revoked_at IS NULL
	`, hash)
	return err
}

func (s *Service) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	_, err := s.DB.Exec(ctx, `
		UPDATE sessions SET revoked_at=now() WHERE user_id=$1 AND revoked_at IS NULL
	`, userID)
	return err
}

func (s *Service) ListSessions(ctx context.Context, userID uuid.UUID, currentRaw string) ([]Session, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT id, user_id, expires_at, created_at, last_seen_at, user_agent, token_hash
		FROM sessions
		WHERE user_id=$1 AND revoked_at IS NULL AND expires_at > now()
		ORDER BY last_seen_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	currentHash := HashToken(currentRaw)
	var out []Session
	for rows.Next() {
		var se Session
		var hash []byte
		if err := rows.Scan(&se.ID, &se.UserID, &se.ExpiresAt, &se.CreatedAt, &se.LastSeen, &se.UserAgent, &hash); err != nil {
			return nil, err
		}
		se.Current = string(hash) == string(currentHash) || bytesEqual(hash, currentHash)
		out = append(out, se)
	}
	return out, nil
}

func (s *Service) RevokeSession(ctx context.Context, userID, sessionID uuid.UUID) error {
	tag, err := s.DB.Exec(ctx, `
		UPDATE sessions SET revoked_at=now()
		WHERE id=$1 AND user_id=$2 AND revoked_at IS NULL
	`, sessionID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUnauthorized
	}
	return nil
}

func (s *Service) checkRateLimit(ctx context.Context, key string, limit int) error {
	window := time.Now().UTC().Truncate(time.Hour)
	var count int
	err := s.DB.QueryRow(ctx, `
		INSERT INTO auth_rate_limits (bucket_key, window_start, count)
		VALUES ($1,$2,1)
		ON CONFLICT (bucket_key, window_start)
		DO UPDATE SET count = auth_rate_limits.count + 1
		RETURNING count
	`, key, window).Scan(&count)
	if err != nil {
		return err
	}
	if count > limit {
		return ErrRateLimited
	}
	return nil
}

func nullStr(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
