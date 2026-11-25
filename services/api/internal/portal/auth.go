package portal

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rubicahq/pikko/services/api/internal/auth"
	"github.com/rubicahq/pikko/services/api/internal/email"
	"github.com/rubicahq/pikko/services/api/internal/validate"
)

// RequestMagicLink sends a store-scoped customer portal magic link.
// Always returns a generic success shape when the email is valid — does not reveal
// whether a customer record exists (enumeration-safe).
func (s *Service) RequestMagicLink(ctx context.Context, storeSlug, rawEmail, ip, ua string) (*RequestMagicLinkResult, error) {
	display, normalized, ok := validate.NormalizeEmail(rawEmail)
	if !ok {
		return nil, ErrInvalidEmail
	}
	st, err := s.resolveStoreBySlug(ctx, storeSlug)
	if err != nil {
		return nil, err
	}

	if err := s.checkRateLimit(ctx, "portal:email:"+st.ID.String()+":"+normalized, s.Cfg.RateLimitEmailHour); err != nil {
		return nil, err
	}
	if ip != "" {
		if err := s.checkRateLimit(ctx, "portal:ip:"+ip, s.Cfg.RateLimitIPHour); err != nil {
			return nil, err
		}
	}

	var lastCreated time.Time
	err = s.DB.QueryRow(ctx, `
		SELECT created_at FROM customer_portal_tokens
		WHERE store_id=$1 AND email_normalized=$2
		ORDER BY created_at DESC LIMIT 1
	`, st.ID, normalized).Scan(&lastCreated)
	if err == nil && time.Since(lastCreated) < s.Cfg.ResendCooldown {
		return nil, ErrCooldown
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	raw, hash, err := auth.NewToken()
	if err != nil {
		return nil, err
	}
	expires := s.now().Add(s.Cfg.MagicLinkTTL)

	var ipPtr *net.IP
	if ip != "" {
		parsed := net.ParseIP(ip)
		if parsed != nil {
			ipPtr = &parsed
		}
	}

	_, err = s.DB.Exec(ctx, `
		INSERT INTO customer_portal_tokens (store_id, email_normalized, token_hash, expires_at, request_ip, user_agent)
		VALUES ($1,$2,$3,$4,$5,$6)
	`, st.ID, normalized, hash, expires, ipPtr, nullStr(ua))
	if err != nil {
		return nil, err
	}

	// Only email when a customer exists for this store (still return ok either way).
	if _, findErr := s.findCustomerByEmail(ctx, st.ID, normalized); findErr == nil {
		confirmURL := s.URLs.PortalConfirm(st.Slug, raw)
		subject, html, text := email.PortalMagicLinkContent(s.Cfg.EmailFromName, st.Name, confirmURL, int(s.Cfg.MagicLinkTTL.Minutes()))
		msg := email.Message{
			TemplateKey:     "portal_magic_link",
			To:              display,
			From:            s.Cfg.EmailAccountFrom,
			FromName:        s.Cfg.EmailFromName,
			ReplyTo:         s.Cfg.EmailReplyTo,
			Subject:         subject,
			HTML:            html,
			Text:            text,
			IdempotencyKey:  fmt.Sprintf("portal-magic:%s:%s:%d", st.ID, normalized, expires.Unix()),
			DisableTracking: true,
		}
		if err := s.Email.SendNow(ctx, msg); err != nil {
			s.Logger.Error("portal_magic_link_email_failed", "error", err)
			return nil, fmt.Errorf("failed to send email")
		}
	}

	res := &RequestMagicLinkResult{
		Email:           display,
		CooldownSeconds: int(s.Cfg.ResendCooldown.Seconds()),
	}
	if (s.Cfg.AppEnv == "development" || s.Cfg.AppEnv == "test") && s.Cfg.EmailMode == "log" {
		// Expose token in test/dev even when no customer — verify will still fail without customer.
		res.DevToken = raw
	}
	return res, nil
}

func (s *Service) PeekToken(ctx context.Context, storeSlug, raw string) (ok bool) {
	if raw == "" {
		return false
	}
	st, err := s.resolveStoreBySlug(ctx, storeSlug)
	if err != nil {
		return false
	}
	hash := auth.HashToken(raw)
	var storeID uuid.UUID
	var expires time.Time
	var consumed *time.Time
	err = s.DB.QueryRow(ctx, `
		SELECT store_id, expires_at, consumed_at
		FROM customer_portal_tokens WHERE token_hash=$1
	`, hash).Scan(&storeID, &expires, &consumed)
	if err != nil || consumed != nil || s.now().After(expires) || storeID != st.ID {
		return false
	}
	return true
}

func (s *Service) VerifyAndConsume(ctx context.Context, storeSlug, raw, ip, ua string) (*VerifyResult, error) {
	if raw == "" {
		return nil, ErrInvalidToken
	}
	st, err := s.resolveStoreBySlug(ctx, storeSlug)
	if err != nil {
		return nil, ErrInvalidToken
	}
	hash := auth.HashToken(raw)

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var tokenID, tokenStoreID uuid.UUID
	var normalized string
	var tokenCustomerID *uuid.UUID
	var expires time.Time
	var consumed *time.Time
	err = tx.QueryRow(ctx, `
		SELECT id, store_id, email_normalized, customer_id, expires_at, consumed_at
		FROM customer_portal_tokens WHERE token_hash=$1 FOR UPDATE
	`, hash).Scan(&tokenID, &tokenStoreID, &normalized, &tokenCustomerID, &expires, &consumed)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidToken
		}
		return nil, err
	}
	if consumed != nil || s.now().After(expires) || tokenStoreID != st.ID {
		return nil, ErrInvalidToken
	}

	tag, err := tx.Exec(ctx, `
		UPDATE customer_portal_tokens
		SET consumed_at=now()
		WHERE id=$1 AND consumed_at IS NULL AND expires_at > now()
	`, tokenID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() != 1 {
		return nil, ErrInvalidToken
	}

	var customer Customer
	if tokenCustomerID != nil {
		err = tx.QueryRow(ctx, `
			SELECT id, store_id, name, email, phone, total_orders, created_at
			FROM customers WHERE id=$1 AND store_id=$2
		`, *tokenCustomerID, st.ID).Scan(
			&customer.ID, &customer.StoreID, &customer.Name, &customer.Email, &customer.Phone,
			&customer.TotalOrders, &customer.CreatedAt,
		)
	} else {
		err = tx.QueryRow(ctx, `
			SELECT id, store_id, name, email, phone, total_orders, created_at
			FROM customers WHERE store_id=$1 AND email IS NOT NULL AND lower(email)=lower($2)
		`, st.ID, normalized).Scan(
			&customer.ID, &customer.StoreID, &customer.Name, &customer.Email, &customer.Phone,
			&customer.TotalOrders, &customer.CreatedAt,
		)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvalidToken
	}
	if err != nil {
		return nil, err
	}

	sessionRaw, sessionHash, err := auth.NewToken()
	if err != nil {
		return nil, err
	}
	sessionExpires := s.now().Add(s.Cfg.SessionTTL)
	_, err = tx.Exec(ctx, `
		INSERT INTO customer_portal_sessions (store_id, customer_id, token_hash, expires_at, user_agent)
		VALUES ($1,$2,$3,$4,$5)
	`, st.ID, customer.ID, sessionHash, sessionExpires, nullStr(ua))
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &VerifyResult{
		Customer:     customer,
		Store:        *st,
		SessionToken: sessionRaw,
		RedirectPath: fmt.Sprintf("/portal/%s/dashboard", st.Slug),
	}, nil
}

func (s *Service) Authenticate(ctx context.Context, sessionRaw string) (*Customer, *Session, *StoreMeta, error) {
	if sessionRaw == "" {
		return nil, nil, nil, ErrUnauthorized
	}
	hash := auth.HashToken(sessionRaw)
	var sess Session
	var customer Customer
	var store StoreMeta
	var primary *string
	err := s.DB.QueryRow(ctx, `
		SELECT se.id, se.store_id, se.customer_id, se.expires_at, se.created_at, se.last_seen_at,
		       c.id, c.store_id, c.name, c.email, c.phone, c.total_orders, c.created_at,
		       s.id, s.name, s.slug, COALESCE(sc.primary_color, '#FF3D00')
		FROM customer_portal_sessions se
		JOIN customers c ON c.id=se.customer_id AND c.store_id=se.store_id
		JOIN stores s ON s.id=se.store_id AND s.archived_at IS NULL
		LEFT JOIN storefront_configs sc ON sc.store_id=s.id AND sc.channel='published'
		WHERE se.token_hash=$1 AND se.revoked_at IS NULL AND se.expires_at > now()
	`, hash).Scan(
		&sess.ID, &sess.StoreID, &sess.CustomerID, &sess.ExpiresAt, &sess.CreatedAt, &sess.LastSeen,
		&customer.ID, &customer.StoreID, &customer.Name, &customer.Email, &customer.Phone,
		&customer.TotalOrders, &customer.CreatedAt,
		&store.ID, &store.Name, &store.Slug, &primary,
	)
	if err != nil {
		return nil, nil, nil, ErrUnauthorized
	}
	if primary != nil {
		store.PrimaryColor = *primary
	} else {
		store.PrimaryColor = "#FF3D00"
	}
	_, _ = s.DB.Exec(ctx, `UPDATE customer_portal_sessions SET last_seen_at=now() WHERE id=$1`, sess.ID)
	return &customer, &sess, &store, nil
}

// AuthenticateForStore ensures the portal session belongs to the store identified by slug.
func (s *Service) AuthenticateForStore(ctx context.Context, storeSlug, sessionRaw string) (*Customer, *Session, *StoreMeta, error) {
	customer, sess, store, err := s.Authenticate(ctx, sessionRaw)
	if err != nil {
		return nil, nil, nil, err
	}
	st, err := s.resolveStoreBySlug(ctx, storeSlug)
	if err != nil {
		return nil, nil, nil, ErrUnauthorized
	}
	if sess.StoreID != st.ID || store.ID != st.ID || customer.StoreID != st.ID {
		return nil, nil, nil, ErrUnauthorized
	}
	return customer, sess, st, nil
}

func (s *Service) Logout(ctx context.Context, sessionRaw string) error {
	if sessionRaw == "" {
		return nil
	}
	hash := auth.HashToken(sessionRaw)
	_, err := s.DB.Exec(ctx, `
		UPDATE customer_portal_sessions SET revoked_at=now()
		WHERE token_hash=$1 AND revoked_at IS NULL
	`, hash)
	return err
}

func (s *Service) checkRateLimit(ctx context.Context, key string, limit int) error {
	if limit <= 0 {
		return nil
	}
	window := s.now().Truncate(time.Hour)
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
