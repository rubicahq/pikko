package email_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rubicahq/pikko/services/api/internal/email"
)

type captureSender struct {
	last email.Message
	n    int
}

func (c *captureSender) Send(_ context.Context, msg email.Message) (string, error) {
	c.last = msg
	c.n++
	return "cap-" + uuid.NewString(), nil
}

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = "postgres://pikko:pikko@localhost:5432/pikko_test?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Skip(err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		t.Skip(err)
	}
	return pool
}

// Regression: email_jobs has no from_name column, so DrainPending must apply
// DefaultFromName (or "Pikko") or Brevo rejects with 400 empty sender.name.
func TestDrainPendingAppliesDefaultFromName(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()

	_, err := pool.Exec(context.Background(), `
		DELETE FROM email_jobs WHERE idempotency_key LIKE 'drain-fromname-%'
	`)
	if err != nil {
		t.Fatal(err)
	}

	cap := &captureSender{}
	svc := &email.Service{
		DB:              pool,
		Sender:          cap,
		Logger:          slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})),
		DefaultFromName: "Pikko Storefront",
	}

	key := "drain-fromname-" + uuid.NewString()
	err = svc.Enqueue(context.Background(), email.Message{
		TemplateKey:    "order_confirmation_customer",
		To:             "customer@example.com",
		From:           "notifications@pikko.site",
		FromName:       "ShouldNotPersist", // not stored in email_jobs
		Subject:        "Order received",
		HTML:           "<p>hi</p>",
		Text:           "hi",
		IdempotencyKey: key,
	})
	if err != nil {
		t.Fatal(err)
	}

	n, err := svc.DrainPending(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if n < 1 {
		t.Fatalf("expected at least 1 sent, got %d", n)
	}
	if cap.last.FromName != "Pikko Storefront" {
		t.Fatalf("FromName=%q want configured DefaultFromName", cap.last.FromName)
	}
	if cap.last.From != "notifications@pikko.site" {
		t.Fatalf("From=%q", cap.last.From)
	}
	if cap.last.To != "customer@example.com" {
		t.Fatalf("To=%q", cap.last.To)
	}
}

func TestDrainPendingFallsBackToPikkoWhenDefaultEmpty(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()

	_, _ = pool.Exec(context.Background(), `
		DELETE FROM email_jobs WHERE idempotency_key LIKE 'drain-fallback-%'
	`)

	cap := &captureSender{}
	svc := &email.Service{
		DB:     pool,
		Sender: cap,
		Logger: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})),
		// DefaultFromName intentionally empty
	}

	key := "drain-fallback-" + uuid.NewString()
	if err := svc.Enqueue(context.Background(), email.Message{
		TemplateKey: "payment_verified_customer", To: "a@example.com",
		From: "notifications@pikko.site", Subject: "Paid", Text: "ok",
		IdempotencyKey: key,
	}); err != nil {
		t.Fatal(err)
	}

	n, err := svc.DrainPending(context.Background(), 5)
	if err != nil {
		t.Fatal(err)
	}
	if n < 1 {
		t.Fatalf("sent=%d", n)
	}
	if cap.last.FromName != "Pikko" {
		t.Fatalf("FromName=%q want Pikko fallback", cap.last.FromName)
	}
}
