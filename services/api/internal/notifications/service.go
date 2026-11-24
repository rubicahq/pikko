package notifications

import (
	"context"
	"errors"
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

// Notification types used by commerce hooks (PRD §29).
const (
	TypeNewOrder            = "new_order"
	TypePendingConfirmation = "pending_confirmation"
	TypePaymentEvidence     = "payment_evidence"
	TypeBatchClosed         = "batch_closed"
	TypePurchasingIssue     = "purchasing_issue"
)

type Notification struct {
	ID           uuid.UUID  `json:"id"`
	StoreID      uuid.UUID  `json:"store_id"`
	Type         string     `json:"type"`
	Title        string     `json:"title"`
	Body         *string    `json:"body,omitempty"`
	ResourceType *string    `json:"resource_type,omitempty"`
	ResourceID   *uuid.UUID `json:"resource_id,omitempty"`
	ActorUserID  *uuid.UUID `json:"actor_user_id,omitempty"`
	ReadAt       *time.Time `json:"read_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

type Preferences struct {
	StoreID                 uuid.UUID `json:"store_id"`
	EmailNewOrder           bool      `json:"email_new_order"`
	EmailPaymentEvidence    bool      `json:"email_payment_evidence"`
	EmailBatchClosed        bool      `json:"email_batch_closed"`
	EmailPurchasingIssues   bool      `json:"email_purchasing_issues"`
	InAppNewOrder           bool      `json:"inapp_new_order"`
	InAppPaymentEvidence    bool      `json:"inapp_payment_evidence"`
	InAppBatchClosed        bool      `json:"inapp_batch_closed"`
	InAppPurchasingIssues   bool      `json:"inapp_purchasing_issues"`
	UpdatedAt               time.Time `json:"updated_at"`
}

type CreateInput struct {
	Type         string
	Title        string
	Body         string
	ResourceType string
	ResourceID   *uuid.UUID
	ActorUserID  *uuid.UUID
}

type Service struct {
	DB *pgxpool.Pool
}

func (s *Service) Create(ctx context.Context, storeID uuid.UUID, in CreateInput) (*Notification, error) {
	typ := strings.TrimSpace(in.Type)
	title := strings.TrimSpace(in.Title)
	if typ == "" || title == "" {
		return nil, ErrInvalidInput
	}
	var body *string
	if b := strings.TrimSpace(in.Body); b != "" {
		body = &b
	}
	var resourceType *string
	if rt := strings.TrimSpace(in.ResourceType); rt != "" {
		resourceType = &rt
	}
	var n Notification
	err := s.DB.QueryRow(ctx, `
		INSERT INTO notifications (store_id, type, title, body, resource_type, resource_id, actor_user_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, store_id, type, title, body, resource_type, resource_id, actor_user_id, read_at, created_at
	`, storeID, typ, title, body, resourceType, in.ResourceID, in.ActorUserID).Scan(
		&n.ID, &n.StoreID, &n.Type, &n.Title, &n.Body, &n.ResourceType, &n.ResourceID,
		&n.ActorUserID, &n.ReadAt, &n.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (s *Service) List(ctx context.Context, storeID uuid.UUID, unreadOnly bool, limit, offset int) ([]Notification, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	where := "store_id=$1"
	if unreadOnly {
		where += " AND read_at IS NULL"
	}
	var total int
	if err := s.DB.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE `+where, storeID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.DB.Query(ctx, `
		SELECT id, store_id, type, title, body, resource_type, resource_id, actor_user_id, read_at, created_at
		FROM notifications WHERE `+where+`
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, storeID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []Notification
	for rows.Next() {
		var n Notification
		if err := rows.Scan(
			&n.ID, &n.StoreID, &n.Type, &n.Title, &n.Body, &n.ResourceType, &n.ResourceID,
			&n.ActorUserID, &n.ReadAt, &n.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, n)
	}
	return out, total, rows.Err()
}

func (s *Service) UnreadCount(ctx context.Context, storeID uuid.UUID) (int, error) {
	var n int
	err := s.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM notifications WHERE store_id=$1 AND read_at IS NULL
	`, storeID).Scan(&n)
	return n, err
}

func (s *Service) MarkRead(ctx context.Context, storeID, id uuid.UUID) error {
	tag, err := s.DB.Exec(ctx, `
		UPDATE notifications SET read_at=COALESCE(read_at, now())
		WHERE id=$1 AND store_id=$2
	`, id, storeID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) MarkAllRead(ctx context.Context, storeID uuid.UUID) (int, error) {
	tag, err := s.DB.Exec(ctx, `
		UPDATE notifications SET read_at=now()
		WHERE store_id=$1 AND read_at IS NULL
	`, storeID)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

func (s *Service) GetPreferences(ctx context.Context, storeID uuid.UUID) (*Preferences, error) {
	p, err := s.scanPreferences(ctx, storeID)
	if err == nil {
		return p, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	// Defaults when no row yet
	return &Preferences{
		StoreID:               storeID,
		EmailNewOrder:         true,
		EmailPaymentEvidence:  true,
		EmailBatchClosed:      true,
		EmailPurchasingIssues: true,
		InAppNewOrder:         true,
		InAppPaymentEvidence:  true,
		InAppBatchClosed:      true,
		InAppPurchasingIssues: true,
		UpdatedAt:             time.Now().UTC(),
	}, nil
}

func (s *Service) scanPreferences(ctx context.Context, storeID uuid.UUID) (*Preferences, error) {
	var p Preferences
	err := s.DB.QueryRow(ctx, `
		SELECT store_id, email_new_order, email_payment_evidence, email_batch_closed, email_purchasing_issues,
		       inapp_new_order, inapp_payment_evidence, inapp_batch_closed, inapp_purchasing_issues, updated_at
		FROM notification_preferences WHERE store_id=$1
	`, storeID).Scan(
		&p.StoreID, &p.EmailNewOrder, &p.EmailPaymentEvidence, &p.EmailBatchClosed, &p.EmailPurchasingIssues,
		&p.InAppNewOrder, &p.InAppPaymentEvidence, &p.InAppBatchClosed, &p.InAppPurchasingIssues, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

type UpdatePreferencesInput struct {
	EmailNewOrder         *bool `json:"email_new_order,omitempty"`
	EmailPaymentEvidence  *bool `json:"email_payment_evidence,omitempty"`
	EmailBatchClosed      *bool `json:"email_batch_closed,omitempty"`
	EmailPurchasingIssues *bool `json:"email_purchasing_issues,omitempty"`
	InAppNewOrder         *bool `json:"inapp_new_order,omitempty"`
	InAppPaymentEvidence  *bool `json:"inapp_payment_evidence,omitempty"`
	InAppBatchClosed      *bool `json:"inapp_batch_closed,omitempty"`
	InAppPurchasingIssues *bool `json:"inapp_purchasing_issues,omitempty"`
}

func (s *Service) UpdatePreferences(ctx context.Context, storeID uuid.UUID, in UpdatePreferencesInput) (*Preferences, error) {
	cur, err := s.GetPreferences(ctx, storeID)
	if err != nil {
		return nil, err
	}
	if in.EmailNewOrder != nil {
		cur.EmailNewOrder = *in.EmailNewOrder
	}
	if in.EmailPaymentEvidence != nil {
		cur.EmailPaymentEvidence = *in.EmailPaymentEvidence
	}
	if in.EmailBatchClosed != nil {
		cur.EmailBatchClosed = *in.EmailBatchClosed
	}
	if in.EmailPurchasingIssues != nil {
		cur.EmailPurchasingIssues = *in.EmailPurchasingIssues
	}
	if in.InAppNewOrder != nil {
		cur.InAppNewOrder = *in.InAppNewOrder
	}
	if in.InAppPaymentEvidence != nil {
		cur.InAppPaymentEvidence = *in.InAppPaymentEvidence
	}
	if in.InAppBatchClosed != nil {
		cur.InAppBatchClosed = *in.InAppBatchClosed
	}
	if in.InAppPurchasingIssues != nil {
		cur.InAppPurchasingIssues = *in.InAppPurchasingIssues
	}
	_, err = s.DB.Exec(ctx, `
		INSERT INTO notification_preferences (
			store_id, email_new_order, email_payment_evidence, email_batch_closed, email_purchasing_issues,
			inapp_new_order, inapp_payment_evidence, inapp_batch_closed, inapp_purchasing_issues, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,now())
		ON CONFLICT (store_id) DO UPDATE SET
			email_new_order=EXCLUDED.email_new_order,
			email_payment_evidence=EXCLUDED.email_payment_evidence,
			email_batch_closed=EXCLUDED.email_batch_closed,
			email_purchasing_issues=EXCLUDED.email_purchasing_issues,
			inapp_new_order=EXCLUDED.inapp_new_order,
			inapp_payment_evidence=EXCLUDED.inapp_payment_evidence,
			inapp_batch_closed=EXCLUDED.inapp_batch_closed,
			inapp_purchasing_issues=EXCLUDED.inapp_purchasing_issues,
			updated_at=now()
	`, storeID, cur.EmailNewOrder, cur.EmailPaymentEvidence, cur.EmailBatchClosed, cur.EmailPurchasingIssues,
		cur.InAppNewOrder, cur.InAppPaymentEvidence, cur.InAppBatchClosed, cur.InAppPurchasingIssues)
	if err != nil {
		return nil, err
	}
	return s.GetPreferences(ctx, storeID)
}

// ShouldInApp returns whether an in-app notification of the given type should be created.
func (s *Service) ShouldInApp(ctx context.Context, storeID uuid.UUID, typ string) bool {
	p, err := s.GetPreferences(ctx, storeID)
	if err != nil {
		return true
	}
	switch typ {
	case TypeNewOrder, TypePendingConfirmation:
		return p.InAppNewOrder
	case TypePaymentEvidence:
		return p.InAppPaymentEvidence
	case TypeBatchClosed:
		return p.InAppBatchClosed
	case TypePurchasingIssue:
		return p.InAppPurchasingIssues
	default:
		return true
	}
}
