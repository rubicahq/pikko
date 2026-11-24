package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Message struct {
	TemplateKey     string
	To              string
	From            string
	FromName        string
	ReplyTo         string
	Subject         string
	HTML            string
	Text            string
	IdempotencyKey  string
	DisableTracking bool
}

type Sender interface {
	Send(ctx context.Context, msg Message) (providerMessageID string, err error)
}

// LogSender writes email content to the logger — safe default for local/dev/test.
type LogSender struct {
	Logger *slog.Logger
}

func (s *LogSender) Send(_ context.Context, msg Message) (string, error) {
	id := "log-" + uuid.NewString()
	s.Logger.Info("email_sent_log_mode",
		"id", id,
		"to", msg.To,
		"from", msg.From,
		"subject", msg.Subject,
		"template", msg.TemplateKey,
		"text", msg.Text,
	)
	return id, nil
}

// BrevoSender calls Brevo Transactional Email API.
type BrevoSender struct {
	APIKey     string
	HTTPClient *http.Client
	Logger     *slog.Logger
}

func (s *BrevoSender) Send(ctx context.Context, msg Message) (string, error) {
	payload := map[string]any{
		"sender": map[string]string{
			"name":  msg.FromName,
			"email": msg.From,
		},
		"to": []map[string]string{
			{"email": msg.To},
		},
		"subject":     msg.Subject,
		"htmlContent": msg.HTML,
		"textContent": msg.Text,
	}
	if msg.ReplyTo != "" {
		payload["replyTo"] = map[string]string{"email": msg.ReplyTo}
	}
	if msg.DisableTracking {
		payload["headers"] = map[string]string{
			"X-Mailin-custom": "pikko-auth",
		}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.brevo.com/v3/smtp/email", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("api-key", s.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := s.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 300 {
		return "", fmt.Errorf("brevo status %d: %s", res.StatusCode, string(raw))
	}
	var out struct {
		MessageID string `json:"messageId"`
	}
	_ = json.Unmarshal(raw, &out)
	if out.MessageID == "" {
		out.MessageID = "brevo-" + uuid.NewString()
	}
	return out.MessageID, nil
}

type Service struct {
	DB              *pgxpool.Pool
	Sender          Sender
	Logger          *slog.Logger
	DefaultFromName string // used when draining email_jobs (from_name is not persisted)
}

func (s *Service) Enqueue(ctx context.Context, msg Message) error {
	if msg.IdempotencyKey == "" {
		msg.IdempotencyKey = uuid.NewString()
	}
	_, err := s.DB.Exec(ctx, `
		INSERT INTO email_jobs (template_key, to_email, from_email, subject, body_html, body_text, idempotency_key, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,'pending')
		ON CONFLICT (idempotency_key) DO NOTHING
	`, msg.TemplateKey, msg.To, msg.From, msg.Subject, msg.HTML, msg.Text, msg.IdempotencyKey)
	return err
}

// SendNow persists a job and attempts delivery immediately (used for high-priority auth).
func (s *Service) SendNow(ctx context.Context, msg Message) error {
	if msg.IdempotencyKey == "" {
		msg.IdempotencyKey = uuid.NewString()
	}
	var jobID uuid.UUID
	err := s.DB.QueryRow(ctx, `
		INSERT INTO email_jobs (template_key, to_email, from_email, subject, body_html, body_text, idempotency_key, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,'sending')
		ON CONFLICT (idempotency_key) DO UPDATE SET updated_at = now()
		RETURNING id
	`, msg.TemplateKey, msg.To, msg.From, msg.Subject, msg.HTML, msg.Text, msg.IdempotencyKey).Scan(&jobID)
	if err != nil {
		return err
	}

	providerID, sendErr := s.Sender.Send(ctx, msg)
	if sendErr != nil {
		_, _ = s.DB.Exec(ctx, `
			UPDATE email_jobs SET status='failed', attempts=attempts+1, last_error=$2, updated_at=now()
			WHERE id=$1
		`, jobID, sendErr.Error())
		return sendErr
	}
	_, err = s.DB.Exec(ctx, `
		UPDATE email_jobs SET status='sent', attempts=attempts+1, provider_message_id=$2, sent_at=now(), updated_at=now()
		WHERE id=$1
	`, jobID, providerID)
	return err
}

// DrainPending sends up to limit pending email_jobs (idempotent; PRD §28).
func (s *Service) DrainPending(ctx context.Context, limit int) (int, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.DB.Query(ctx, `
		SELECT id, template_key, to_email, from_email, subject, body_html, body_text, idempotency_key
		FROM email_jobs
		WHERE status IN ('pending', 'failed') AND attempts < 8
		ORDER BY created_at
		LIMIT $1
	`, limit)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	type job struct {
		ID             uuid.UUID
		TemplateKey    string
		To, From       string
		Subject        string
		HTML, Text     string
		IdempotencyKey string
	}
	var jobs []job
	for rows.Next() {
		var j job
		if err := rows.Scan(&j.ID, &j.TemplateKey, &j.To, &j.From, &j.Subject, &j.HTML, &j.Text, &j.IdempotencyKey); err != nil {
			return 0, err
		}
		jobs = append(jobs, j)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	sent := 0
	for _, j := range jobs {
		ct, err := s.DB.Exec(ctx, `
			UPDATE email_jobs SET status='sending', updated_at=now()
			WHERE id=$1 AND status IN ('pending','failed')
		`, j.ID)
		if err != nil || ct.RowsAffected() == 0 {
			continue
		}
		fromName := s.DefaultFromName
		if fromName == "" {
			fromName = "Pikko"
		}
		providerID, sendErr := s.Sender.Send(ctx, Message{
			TemplateKey: j.TemplateKey, To: j.To, From: j.From, FromName: fromName,
			Subject: j.Subject, HTML: j.HTML, Text: j.Text, IdempotencyKey: j.IdempotencyKey,
		})
		if sendErr != nil {
			_, _ = s.DB.Exec(ctx, `
				UPDATE email_jobs SET status='failed', attempts=attempts+1, last_error=$2, updated_at=now()
				WHERE id=$1
			`, j.ID, sendErr.Error())
			continue
		}
		_, _ = s.DB.Exec(ctx, `
			UPDATE email_jobs SET status='sent', attempts=attempts+1, provider_message_id=$2, sent_at=now(), updated_at=now()
			WHERE id=$1
		`, j.ID, providerID)
		sent++
	}
	return sent, nil
}

func NewSender(mode, apiKey string, logger *slog.Logger) Sender {
	switch strings.ToLower(mode) {
	case "brevo":
		return &BrevoSender{
			APIKey: apiKey,
			HTTPClient: &http.Client{Timeout: 15 * time.Second},
			Logger: logger,
		}
	default:
		return &LogSender{Logger: logger}
	}
}

func PortalMagicLinkContent(fromName, storeName, confirmURL string, ttlMinutes int) (subject, html, text string) {
	subject = fmt.Sprintf("Your %s order portal link", storeName)
	text = fmt.Sprintf(
		"Access your orders at %s.\n\nUse the link below to securely open your customer portal:\n%s\n\nThis link expires in %d minutes and can only be used once.\nIf you did not request this email, you can safely ignore it.\n",
		storeName, confirmURL, ttlMinutes,
	)
	html = fmt.Sprintf(`<!DOCTYPE html>
<html><body style="margin:0;padding:0;background:#FFFFFF;font-family:'Plus Jakarta Sans',Arial,sans-serif;color:#15191E;">
<table role="presentation" width="100%%" cellspacing="0" cellpadding="0"><tr><td align="center" style="padding:32px 16px;">
<table role="presentation" width="600" style="max-width:600px;width:100%%;border:1px solid #E5E7EB;border-radius:14px;">
<tr><td style="padding:32px;">
<p style="margin:0 0 8px;font-size:14px;color:#626975;">%s</p>
<h1 style="margin:0 0 16px;font-size:24px;font-weight:700;letter-spacing:-0.02em;">Your %s orders.</h1>
<p style="margin:0 0 24px;font-size:16px;line-height:1.5;color:#626975;">Use the button below to securely open your customer portal.</p>
<p style="margin:0 0 24px;"><a href="%s" style="display:inline-block;background:#15191E;color:#FFFFFF;text-decoration:none;padding:12px 20px;border-radius:10px;font-weight:600;font-size:14px;">Open portal.</a></p>
<div style="background:#F5F6FA;border:1px solid #E5E7EB;border-radius:10px;padding:16px;">
<p style="margin:0;font-size:14px;line-height:1.5;color:#626975;">This link expires in %d minutes and can only be used once.<br/>If you did not request this email, you can safely ignore it.</p>
</div>
</td></tr></table>
</td></tr></table>
</body></html>`, fromName, storeName, confirmURL, ttlMinutes)
	return subject, html, text
}

func MagicLinkContent(fromName, confirmURL string, ttlMinutes int) (subject, html, text string) {
	subject = "Your Pikko sign-in link"
	text = fmt.Sprintf(
		"Welcome to Pikko.\n\nUse the link below to securely access your account:\n%s\n\nThis link expires in %d minutes and can only be used once.\nIf you did not request this email, you can safely ignore it.\n",
		confirmURL, ttlMinutes,
	)
	html = fmt.Sprintf(`<!DOCTYPE html>
<html><body style="margin:0;padding:0;background:#FFFFFF;font-family:Inter,Arial,sans-serif;color:#15191E;">
<table role="presentation" width="100%%" cellspacing="0" cellpadding="0"><tr><td align="center" style="padding:32px 16px;">
<table role="presentation" width="600" style="max-width:600px;width:100%%;border:1px solid #E1E3E8;border-radius:12px;">
<tr><td style="padding:32px;">
<p style="margin:0 0 8px;font-size:14px;color:#60646C;">%s</p>
<h1 style="margin:0 0 16px;font-size:24px;font-weight:500;letter-spacing:-0.02em;">Welcome to Pikko.</h1>
<p style="margin:0 0 24px;font-size:16px;line-height:1.5;color:#60646C;">Use the button below to securely access your account.</p>
<p style="margin:0 0 24px;"><a href="%s" style="display:inline-block;background:#C83200;color:#FCFCFD;text-decoration:none;padding:12px 20px;border-radius:8px;font-weight:500;font-size:14px;">Continue to Pikko.</a></p>
<div style="background:#F3F3F7;border:1px solid #E1E3E8;border-radius:8px;padding:16px;">
<p style="margin:0;font-size:14px;line-height:1.5;color:#60646C;">This link expires in %d minutes and can only be used once.<br/>If you did not request this email, you can safely ignore it.</p>
</div>
</td></tr></table>
</td></tr></table>
</body></html>`, fromName, confirmURL, ttlMinutes)
	return subject, html, text
}

func TeamInviteContent(fromName, storeName, role, acceptURL string, ttlHours int) (subject, html, text string) {
	subject = fmt.Sprintf("You're invited to %s on Pikko", storeName)
	text = fmt.Sprintf(
		"You've been invited to join %s on Pikko as %s.\n\nAccept your invitation:\n%s\n\nThis link expires in about %d hours.\nIf you were not expecting this invite, you can ignore this email.\n",
		storeName, role, acceptURL, ttlHours,
	)
	html = fmt.Sprintf(`<!DOCTYPE html>
<html><body style="margin:0;padding:0;background:#FFFFFF;font-family:Inter,Arial,sans-serif;color:#15191E;">
<table role="presentation" width="100%%" cellspacing="0" cellpadding="0"><tr><td align="center" style="padding:32px 16px;">
<table role="presentation" width="600" style="max-width:600px;width:100%%;border:1px solid #E1E3E8;border-radius:12px;">
<tr><td style="padding:32px;">
<p style="margin:0 0 8px;font-size:14px;color:#60646C;">%s</p>
<h1 style="margin:0 0 16px;font-size:24px;font-weight:500;letter-spacing:-0.02em;">You're invited to %s.</h1>
<p style="margin:0 0 24px;font-size:16px;line-height:1.5;color:#60646C;">Join as <strong>%s</strong>. Use the button below to accept — you'll sign in with this email via a secure link.</p>
<p style="margin:0 0 24px;"><a href="%s" style="display:inline-block;background:#C83200;color:#FCFCFD;text-decoration:none;padding:12px 20px;border-radius:8px;font-weight:500;font-size:14px;">Accept invitation.</a></p>
<div style="background:#F3F3F7;border:1px solid #E1E3E8;border-radius:8px;padding:16px;">
<p style="margin:0;font-size:14px;line-height:1.5;color:#60646C;">This invite expires in about %d hours.<br/>If you were not expecting this, you can ignore this email.</p>
</div>
</td></tr></table>
</td></tr></table>
</body></html>`, fromName, storeName, role, acceptURL, ttlHours)
	return subject, html, text
}
