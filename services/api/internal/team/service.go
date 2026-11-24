package team

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

	"github.com/rubicahq/pikko/services/api/internal/auth"
	"github.com/rubicahq/pikko/services/api/internal/config"
	"github.com/rubicahq/pikko/services/api/internal/email"
	"github.com/rubicahq/pikko/services/api/internal/rbac"
	"github.com/rubicahq/pikko/services/api/internal/urls"
	"github.com/rubicahq/pikko/services/api/internal/validate"
)

var (
	ErrNotFound       = errors.New("not found")
	ErrForbidden      = errors.New("forbidden")
	ErrInvalidInput   = errors.New("invalid input")
	ErrAlreadyMember  = errors.New("already a member")
	ErrInvitePending  = errors.New("invite already pending")
	ErrInviteInvalid  = errors.New("invite invalid or expired")
	ErrCannotRevoke   = errors.New("cannot revoke")
	ErrCooldown = errors.New("resend cooldown")
)

const (
	InviteTTL      = 7 * 24 * time.Hour
	ResendCooldown = 60 * time.Second
)

type Service struct {
	DB     *pgxpool.Pool
	Cfg    *config.Config
	Email  *email.Service
	URLs   *urls.Generator
	Logger *slog.Logger
}

type Member struct {
	ID          uuid.UUID `json:"id"`
	StoreID     uuid.UUID `json:"store_id"`
	UserID      uuid.UUID `json:"user_id"`
	Email       string    `json:"email"`
	DisplayName *string   `json:"display_name"`
	Role        string    `json:"role"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type Invite struct {
	ID               uuid.UUID  `json:"id"`
	StoreID          uuid.UUID  `json:"store_id"`
	Email            string     `json:"email"`
	Role             string     `json:"role"`
	Status           string     `json:"status"`
	InvitedByUserID  uuid.UUID  `json:"invited_by_user_id"`
	ExpiresAt        time.Time  `json:"expires_at"`
	AcceptedAt       *time.Time `json:"accepted_at,omitempty"`
	CancelledAt      *time.Time `json:"cancelled_at,omitempty"`
	LastSentAt       time.Time  `json:"last_sent_at"`
	CreatedAt        time.Time  `json:"created_at"`
	DevToken         string     `json:"dev_token,omitempty"`
}

type InvitePeek struct {
	Valid     bool   `json:"valid"`
	StoreID   string `json:"store_id,omitempty"`
	StoreName string `json:"store_name,omitempty"`
	Role      string `json:"role,omitempty"`
	Email     string `json:"email,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

type AcceptResult struct {
	User         auth.User `json:"user"`
	SessionToken string    `json:"-"`
	StoreID      uuid.UUID `json:"store_id"`
	Role         string    `json:"role"`
	IsNewAccount bool      `json:"is_new_account"`
	RedirectPath string    `json:"redirect_path"`
}

func (s *Service) ListMembers(ctx context.Context, storeID uuid.UUID) ([]Member, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT m.id, m.store_id, m.user_id, u.email, u.display_name, m.role, m.status, m.created_at
		FROM store_memberships m
		JOIN users u ON u.id = m.user_id
		WHERE m.store_id=$1 AND m.status='active'
		ORDER BY
		  CASE m.role
		    WHEN 'owner' THEN 0 WHEN 'admin' THEN 1 ELSE 2
		  END,
		  m.created_at ASC
	`, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Member
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.ID, &m.StoreID, &m.UserID, &m.Email, &m.DisplayName, &m.Role, &m.Status, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

func (s *Service) ListInvites(ctx context.Context, storeID uuid.UUID) ([]Invite, error) {
	_, _ = s.DB.Exec(ctx, `
		UPDATE team_invites
		SET status='expired', updated_at=now()
		WHERE store_id=$1 AND status='pending' AND expires_at <= now()
	`, storeID)

	rows, err := s.DB.Query(ctx, `
		SELECT id, store_id, email, role, status, invited_by_user_id, expires_at,
		       accepted_at, cancelled_at, last_sent_at, created_at
		FROM team_invites
		WHERE store_id=$1 AND status IN ('pending','accepted','cancelled','expired')
		ORDER BY created_at DESC
		LIMIT 100
	`, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Invite
	for rows.Next() {
		var inv Invite
		if err := rows.Scan(
			&inv.ID, &inv.StoreID, &inv.Email, &inv.Role, &inv.Status, &inv.InvitedByUserID,
			&inv.ExpiresAt, &inv.AcceptedAt, &inv.CancelledAt, &inv.LastSentAt, &inv.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, inv)
	}
	return out, nil
}

type CreateInviteInput struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

func (s *Service) CreateInvite(ctx context.Context, storeID, actorID uuid.UUID, in CreateInviteInput) (*Invite, error) {
	display, normalized, ok := validate.NormalizeEmail(in.Email)
	if !ok {
		return nil, ErrInvalidInput
	}
	role := strings.TrimSpace(strings.ToLower(in.Role))
	if !rbac.ValidInviteRole(role) {
		return nil, ErrInvalidInput
	}

	var existingRole string
	err := s.DB.QueryRow(ctx, `
		SELECT role FROM store_memberships
		WHERE store_id=$1 AND user_id=(SELECT id FROM users WHERE email_normalized=$2)
		  AND status='active'
	`, storeID, normalized).Scan(&existingRole)
	if err == nil {
		return nil, ErrAlreadyMember
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	var pendingID uuid.UUID
	err = s.DB.QueryRow(ctx, `
		SELECT id FROM team_invites
		WHERE store_id=$1 AND email_normalized=$2 AND status='pending' AND expires_at > now()
	`, storeID, normalized).Scan(&pendingID)
	if err == nil {
		return nil, ErrInvitePending
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	raw, hash, err := auth.NewToken()
	if err != nil {
		return nil, err
	}
	expires := time.Now().UTC().Add(InviteTTL)

	var inv Invite
	err = s.DB.QueryRow(ctx, `
		INSERT INTO team_invites (
			store_id, email, email_normalized, role, invited_by_user_id,
			token_hash, status, expires_at, last_sent_at
		) VALUES ($1,$2,$3,$4,$5,$6,'pending',$7,now())
		RETURNING id, store_id, email, role, status, invited_by_user_id, expires_at,
		          accepted_at, cancelled_at, last_sent_at, created_at
	`, storeID, display, normalized, role, actorID, hash, expires).Scan(
		&inv.ID, &inv.StoreID, &inv.Email, &inv.Role, &inv.Status, &inv.InvitedByUserID,
		&inv.ExpiresAt, &inv.AcceptedAt, &inv.CancelledAt, &inv.LastSentAt, &inv.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrInvitePending
		}
		return nil, err
	}

	if err := s.sendInviteEmail(ctx, storeID, display, role, raw, expires); err != nil {
		s.Logger.Error("team_invite_email_failed", "error", err, "invite_id", inv.ID)
		return nil, fmt.Errorf("failed to send invite email")
	}

	_, _ = s.DB.Exec(ctx, `
		INSERT INTO audit_logs (actor_user_id, store_id, action, subject_type, subject_id, metadata)
		VALUES ($1,$2,'team_invite_created','team_invite',$3, jsonb_build_object('email',$4::text,'role',$5::text))
	`, actorID, storeID, inv.ID.String(), display, role)

	if (s.Cfg.AppEnv == "development" || s.Cfg.AppEnv == "test") && s.Cfg.EmailMode == "log" {
		inv.DevToken = raw
	}
	return &inv, nil
}

func (s *Service) ResendInvite(ctx context.Context, storeID, actorID, inviteID uuid.UUID) (*Invite, error) {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var inv Invite
	var normalized string
	err = tx.QueryRow(ctx, `
		SELECT id, store_id, email, email_normalized, role, status, invited_by_user_id, expires_at,
		       accepted_at, cancelled_at, last_sent_at, created_at
		FROM team_invites
		WHERE id=$1 AND store_id=$2
		FOR UPDATE
	`, inviteID, storeID).Scan(
		&inv.ID, &inv.StoreID, &inv.Email, &normalized, &inv.Role, &inv.Status, &inv.InvitedByUserID,
		&inv.ExpiresAt, &inv.AcceptedAt, &inv.CancelledAt, &inv.LastSentAt, &inv.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	_ = normalized
	if inv.Status != "pending" || time.Now().UTC().After(inv.ExpiresAt) {
		return nil, ErrInviteInvalid
	}
	if time.Since(inv.LastSentAt) < ResendCooldown {
		return nil, ErrCooldown
	}

	raw, newHash, err := auth.NewToken()
	if err != nil {
		return nil, err
	}
	expires := time.Now().UTC().Add(InviteTTL)
	err = tx.QueryRow(ctx, `
		UPDATE team_invites
		SET token_hash=$3, expires_at=$4, last_sent_at=now(), updated_at=now()
		WHERE id=$1 AND store_id=$2 AND status='pending'
		RETURNING id, store_id, email, role, status, invited_by_user_id, expires_at,
		          accepted_at, cancelled_at, last_sent_at, created_at
	`, inviteID, storeID, newHash, expires).Scan(
		&inv.ID, &inv.StoreID, &inv.Email, &inv.Role, &inv.Status, &inv.InvitedByUserID,
		&inv.ExpiresAt, &inv.AcceptedAt, &inv.CancelledAt, &inv.LastSentAt, &inv.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	if err := s.sendInviteEmail(ctx, storeID, inv.Email, inv.Role, raw, expires); err != nil {
		return nil, fmt.Errorf("failed to send invite email")
	}
	_, _ = s.DB.Exec(ctx, `
		INSERT INTO audit_logs (actor_user_id, store_id, action, subject_type, subject_id)
		VALUES ($1,$2,'team_invite_resent','team_invite',$3)
	`, actorID, storeID, inviteID.String())

	if (s.Cfg.AppEnv == "development" || s.Cfg.AppEnv == "test") && s.Cfg.EmailMode == "log" {
		inv.DevToken = raw
	}
	return &inv, nil
}

func (s *Service) CancelInvite(ctx context.Context, storeID, actorID, inviteID uuid.UUID) (*Invite, error) {
	var inv Invite
	err := s.DB.QueryRow(ctx, `
		UPDATE team_invites
		SET status='cancelled', cancelled_at=now(), updated_at=now()
		WHERE id=$1 AND store_id=$2 AND status='pending'
		RETURNING id, store_id, email, role, status, invited_by_user_id, expires_at,
		          accepted_at, cancelled_at, last_sent_at, created_at
	`, inviteID, storeID).Scan(
		&inv.ID, &inv.StoreID, &inv.Email, &inv.Role, &inv.Status, &inv.InvitedByUserID,
		&inv.ExpiresAt, &inv.AcceptedAt, &inv.CancelledAt, &inv.LastSentAt, &inv.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	_, _ = s.DB.Exec(ctx, `
		INSERT INTO audit_logs (actor_user_id, store_id, action, subject_type, subject_id)
		VALUES ($1,$2,'team_invite_cancelled','team_invite',$3)
	`, actorID, storeID, inviteID.String())
	return &inv, nil
}

func (s *Service) RevokeMember(ctx context.Context, storeID, actorID, membershipID uuid.UUID) error {
	var role string
	var userID uuid.UUID
	err := s.DB.QueryRow(ctx, `
		SELECT role, user_id FROM store_memberships
		WHERE id=$1 AND store_id=$2 AND status='active'
	`, membershipID, storeID).Scan(&role, &userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if role == rbac.RoleOwner {
		return ErrCannotRevoke
	}
	if userID == actorID {
		return ErrCannotRevoke
	}

	tag, err := s.DB.Exec(ctx, `
		UPDATE store_memberships SET status='revoked'
		WHERE id=$1 AND store_id=$2 AND status='active' AND role<>'owner'
	`, membershipID, storeID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrNotFound
	}
	_, _ = s.DB.Exec(ctx, `
		UPDATE users SET last_store_id=NULL, updated_at=now()
		WHERE id=$1 AND last_store_id=$2
	`, userID, storeID)
	_, _ = s.DB.Exec(ctx, `
		INSERT INTO audit_logs (actor_user_id, store_id, action, subject_type, subject_id, metadata)
		VALUES ($1,$2,'team_member_revoked','store_membership',$3, jsonb_build_object('user_id',$4::text,'role',$5::text))
	`, actorID, storeID, membershipID.String(), userID.String(), role)
	return nil
}

type PatchMemberInput struct {
	Role string `json:"role"`
}

func (s *Service) PatchMember(ctx context.Context, storeID, actorID, membershipID uuid.UUID, in PatchMemberInput) (*Member, error) {
	role := strings.TrimSpace(strings.ToLower(in.Role))
	if !rbac.ValidInviteRole(role) {
		return nil, ErrInvalidInput
	}
	var currentRole string
	var userID uuid.UUID
	err := s.DB.QueryRow(ctx, `
		SELECT role, user_id FROM store_memberships
		WHERE id=$1 AND store_id=$2 AND status='active'
	`, membershipID, storeID).Scan(&currentRole, &userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if currentRole == rbac.RoleOwner {
		return nil, ErrCannotRevoke
	}

	_, err = s.DB.Exec(ctx, `
		UPDATE store_memberships SET role=$3 WHERE id=$1 AND store_id=$2 AND status='active' AND role<>'owner'
	`, membershipID, storeID, role)
	if err != nil {
		return nil, err
	}
	_, _ = s.DB.Exec(ctx, `
		INSERT INTO audit_logs (actor_user_id, store_id, action, subject_type, subject_id, metadata)
		VALUES ($1,$2,'team_member_role_changed','store_membership',$3,
		        jsonb_build_object('from',$4::text,'to',$5::text,'user_id',$6::text))
	`, actorID, storeID, membershipID.String(), currentRole, role, userID.String())

	var m Member
	err = s.DB.QueryRow(ctx, `
		SELECT m.id, m.store_id, m.user_id, u.email, u.display_name, m.role, m.status, m.created_at
		FROM store_memberships m
		JOIN users u ON u.id = m.user_id
		WHERE m.id=$1
	`, membershipID).Scan(&m.ID, &m.StoreID, &m.UserID, &m.Email, &m.DisplayName, &m.Role, &m.Status, &m.CreatedAt)
	return &m, err
}

func (s *Service) PeekInvite(ctx context.Context, raw string) (*InvitePeek, error) {
	if raw == "" {
		return &InvitePeek{Valid: false}, nil
	}
	hash := auth.HashToken(raw)
	var storeID uuid.UUID
	var storeName, role, email string
	var expires time.Time
	var status string
	err := s.DB.QueryRow(ctx, `
		SELECT i.store_id, s.name, i.role, i.email, i.expires_at, i.status
		FROM team_invites i
		JOIN stores s ON s.id = i.store_id
		WHERE i.token_hash=$1
	`, hash).Scan(&storeID, &storeName, &role, &email, &expires, &status)
	if err != nil || status != "pending" || time.Now().UTC().After(expires) {
		return &InvitePeek{Valid: false}, nil
	}
	return &InvitePeek{
		Valid:     true,
		StoreID:   storeID.String(),
		StoreName: storeName,
		Role:      role,
		Email:     email,
		ExpiresAt: expires.UTC().Format(time.RFC3339),
	}, nil
}

func (s *Service) AcceptInvite(ctx context.Context, raw, ip, ua string) (*AcceptResult, error) {
	if raw == "" {
		return nil, ErrInviteInvalid
	}
	hash := auth.HashToken(raw)

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var inviteID, storeID, invitedBy uuid.UUID
	var normalized, displayEmail, role, status string
	var expires time.Time
	err = tx.QueryRow(ctx, `
		SELECT id, store_id, email, email_normalized, role, status, expires_at, invited_by_user_id
		FROM team_invites
		WHERE token_hash=$1
		FOR UPDATE
	`, hash).Scan(&inviteID, &storeID, &displayEmail, &normalized, &role, &status, &expires, &invitedBy)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInviteInvalid
		}
		return nil, err
	}
	if status != "pending" || time.Now().UTC().After(expires) {
		if status == "pending" {
			_, _ = tx.Exec(ctx, `UPDATE team_invites SET status='expired', updated_at=now() WHERE id=$1`, inviteID)
		}
		return nil, ErrInviteInvalid
	}

	var user auth.User
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
		`, displayEmail, normalized, s.Cfg.DefaultLocale).Scan(
			&user.ID, &user.Email, &user.DisplayName, &user.Locale, &user.LastStoreID, &user.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO user_preferences (user_id, language, timezone)
			VALUES ($1,$2,$3) ON CONFLICT (user_id) DO NOTHING
		`, user.ID, s.Cfg.DefaultLocale, s.Cfg.DefaultTZ)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	var existingStatus string
	err = tx.QueryRow(ctx, `
		SELECT status FROM store_memberships WHERE store_id=$1 AND user_id=$2
	`, storeID, user.ID).Scan(&existingStatus)
	if err == nil {
		if existingStatus == "active" {
			_, _ = tx.Exec(ctx, `
				UPDATE team_invites
				SET status='accepted', accepted_at=now(), accepted_user_id=$2, updated_at=now()
				WHERE id=$1
			`, inviteID, user.ID)
			if err := tx.Commit(ctx); err != nil {
				return nil, err
			}
			sessionRaw, err := s.createSession(ctx, user.ID, ip, ua)
			if err != nil {
				return nil, err
			}
			return &AcceptResult{
				User: user, SessionToken: sessionRaw, StoreID: storeID, Role: role,
				IsNewAccount: false, RedirectPath: fmt.Sprintf("/stores/%s/overview", storeID.String()),
			}, nil
		}
		_, err = tx.Exec(ctx, `
			UPDATE store_memberships SET role=$3, status='active' WHERE store_id=$1 AND user_id=$2
		`, storeID, user.ID, role)
		if err != nil {
			return nil, err
		}
	} else if errors.Is(err, pgx.ErrNoRows) {
		_, err = tx.Exec(ctx, `
			INSERT INTO store_memberships (store_id, user_id, role, status)
			VALUES ($1,$2,$3,'active')
		`, storeID, user.ID, role)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE team_invites
		SET status='accepted', accepted_at=now(), accepted_user_id=$2, updated_at=now()
		WHERE id=$1 AND status='pending'
	`, inviteID, user.ID)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `UPDATE users SET last_store_id=$2, updated_at=now() WHERE id=$1`, user.ID, storeID)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO audit_logs (actor_user_id, store_id, action, subject_type, subject_id, metadata)
		VALUES ($1,$2,'team_invite_accepted','team_invite',$3, jsonb_build_object('role',$4::text))
	`, user.ID, storeID, inviteID.String(), role)
	if err != nil {
		return nil, err
	}

	sessionRaw, sessionHash, err := auth.NewToken()
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
	_, err = tx.Exec(ctx, `
		INSERT INTO sessions (user_id, token_hash, expires_at, created_ip, last_ip, user_agent)
		VALUES ($1,$2,$3,$4,$4,$5)
	`, user.ID, sessionHash, sessionExpires, ipPtr, nullStr(ua))
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &AcceptResult{
		User:         user,
		SessionToken: sessionRaw,
		StoreID:      storeID,
		Role:         role,
		IsNewAccount: isNew,
		RedirectPath: fmt.Sprintf("/stores/%s/overview", storeID.String()),
	}, nil
}

func (s *Service) createSession(ctx context.Context, userID uuid.UUID, ip, ua string) (string, error) {
	sessionRaw, sessionHash, err := auth.NewToken()
	if err != nil {
		return "", err
	}
	sessionExpires := time.Now().UTC().Add(s.Cfg.SessionTTL)
	var ipPtr *net.IP
	if ip != "" {
		parsed := net.ParseIP(ip)
		if parsed != nil {
			ipPtr = &parsed
		}
	}
	_, err = s.DB.Exec(ctx, `
		INSERT INTO sessions (user_id, token_hash, expires_at, created_ip, last_ip, user_agent)
		VALUES ($1,$2,$3,$4,$4,$5)
	`, userID, sessionHash, sessionExpires, ipPtr, nullStr(ua))
	return sessionRaw, err
}

func (s *Service) MembershipRole(ctx context.Context, userID, storeID uuid.UUID) (string, error) {
	var role string
	err := s.DB.QueryRow(ctx, `
		SELECT role FROM store_memberships
		WHERE user_id=$1 AND store_id=$2 AND status='active'
	`, userID, storeID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrForbidden
	}
	return role, err
}

func (s *Service) sendInviteEmail(ctx context.Context, storeID uuid.UUID, toEmail, role, rawToken string, expires time.Time) error {
	var storeName string
	if err := s.DB.QueryRow(ctx, `SELECT name FROM stores WHERE id=$1`, storeID).Scan(&storeName); err != nil {
		return err
	}
	acceptURL := s.URLs.TeamInvite(rawToken)
	ttlHours := int(time.Until(expires).Hours())
	if ttlHours < 1 {
		ttlHours = 1
	}
	subject, html, text := email.TeamInviteContent(s.Cfg.EmailFromName, storeName, role, acceptURL, ttlHours)
	msg := email.Message{
		TemplateKey:     "team_invite",
		To:              toEmail,
		From:            s.Cfg.EmailAccountFrom,
		FromName:        s.Cfg.EmailFromName,
		ReplyTo:         s.Cfg.EmailReplyTo,
		Subject:         subject,
		HTML:            html,
		Text:            text,
		IdempotencyKey:  fmt.Sprintf("team_invite:%s:%s:%d", storeID.String(), strings.ToLower(toEmail), expires.Unix()),
		DisableTracking: true,
	}
	return s.Email.SendNow(ctx, msg)
}

func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "duplicate key") || strings.Contains(msg, "unique constraint")
}
