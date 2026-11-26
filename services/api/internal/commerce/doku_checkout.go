package commerce

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const dokuCheckoutTarget = "/checkout/v1/payment"

var ErrDOKUUnavailable = errors.New("doku checkout unavailable")

type DOKUCheckoutResult struct {
	InvoiceNumber string `json:"invoice_number"`
	PaymentURL    string `json:"payment_url"`
}

func dokuDigest(body []byte) string {
	sum := sha256.Sum256(body)
	return base64.StdEncoding.EncodeToString(sum[:])
}

func dokuSignature(clientID, requestID, timestamp, target, digest, secret string) string {
	component := "Client-Id:" + clientID + "\nRequest-Id:" + requestID +
		"\nRequest-Timestamp:" + timestamp + "\nRequest-Target:" + target + "\nDigest:" + digest
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(component))
	return "HMACSHA256=" + base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// CreateDOKUCheckout creates a hosted DOKU Checkout session and stores only
// the identifiers needed for webhook reconciliation and read-only inquiry.
func (s *Service) CreateDOKUCheckout(ctx context.Context, order *Order, payment *Payment) (*DOKUCheckoutResult, error) {
	if order == nil || payment == nil {
		return nil, ErrDOKUUnavailable
	}
	credentials, err := s.LoadDOKUCredentials(ctx, order.StoreID)
	if err != nil {
		return nil, ErrDOKUUnavailable
	}
	// DOKU invoice is a provider identifier, not the merchant-facing Pikko
	// order number. UUID makes it unique across stores sharing one DOKU account.
	invoice := "DOKU-" + order.ID.String()
	requestID := uuid.NewString()
	payload := map[string]any{
		"order": map[string]any{
			"amount": payment.AmountIDR, "invoice_number": invoice,
			"currency": "IDR", "callback_url": strings.TrimRight(s.Cfg.APIBaseURL, "/") + "/api/v1/webhooks/doku",
		},
		"payment":  map[string]any{"payment_due_date": 60},
		"customer": map[string]any{"name": order.CustomerName, "email": order.CustomerEmail},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	ts := time.Now().UTC().Format(time.RFC3339)
	target := dokuCheckoutTarget
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(s.Cfg.DOKUCheckoutBaseURL, "/")+target, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Client-Id", credentials.CheckoutClientID)
	req.Header.Set("Request-Id", requestID)
	req.Header.Set("Request-Timestamp", ts)
	req.Header.Set("Digest", dokuDigest(body))
	req.Header.Set("Signature", dokuSignature(credentials.CheckoutClientID, requestID, ts, target, dokuDigest(body), credentials.CheckoutSecret))
	client := &http.Client{Timeout: 12 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: request failed", ErrDOKUUnavailable)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: status %d", ErrDOKUUnavailable, res.StatusCode)
	}
	var decoded struct {
		Response struct {
			Payment struct {
				URL string `json:"url"`
			} `json:"payment"`
			Order struct {
				InvoiceNumber string `json:"invoice_number"`
			} `json:"order"`
		} `json:"response"`
	}
	if json.Unmarshal(raw, &decoded) != nil || decoded.Response.Payment.URL == "" {
		return nil, fmt.Errorf("%w: invalid response", ErrDOKUUnavailable)
	}
	if decoded.Response.Order.InvoiceNumber != "" {
		invoice = decoded.Response.Order.InvoiceNumber
	}
	_, err = s.DB.Exec(ctx, `
		INSERT INTO payment_attempts (store_id, order_id, payment_id, provider, provider_invoice_number, amount_idr, status, request_id, checkout_url)
		VALUES ($1,$2,$3,'doku',$4,$5,'pending',$6,$7)
	`, order.StoreID, order.ID, payment.ID, invoice, payment.AmountIDR, requestID, decoded.Response.Payment.URL)
	if err != nil {
		return nil, err
	}
	return &DOKUCheckoutResult{InvoiceNumber: invoice, PaymentURL: decoded.Response.Payment.URL}, nil
}

// HandleDOKUWebhook verifies the DOKU signature and reconciles a matching
// invoice+amount idempotently. No fulfillment transition is performed.
func (s *Service) HandleDOKUWebhook(ctx context.Context, headers http.Header, body []byte) error {
	if s == nil || s.Cfg == nil {
		return ErrDOKUUnavailable
	}
	requestID, ts := headers.Get("Request-Id"), headers.Get("Request-Timestamp")
	credentials, err := s.LoadDOKUCredentialsByCheckoutClient(ctx, headers.Get("Client-Id"))
	if err != nil {
		return ErrForbidden
	}
	if requestID == "" || ts == "" {
		return ErrForbidden
	}
	stamp, err := time.Parse(time.RFC3339, ts)
	if err != nil || time.Since(stamp) > 10*time.Minute || time.Until(stamp) > 2*time.Minute {
		return ErrForbidden
	}
	target := headers.Get("Request-Target")
	if target == "" {
		target = "/api/v1/webhooks/doku"
	}
	expected := dokuSignature(headers.Get("Client-Id"), requestID, ts, target, dokuDigest(body), credentials.CheckoutSecret)
	if !hmac.Equal([]byte(expected), []byte(headers.Get("Signature"))) {
		return ErrForbidden
	}
	var event struct {
		Order struct {
			InvoiceNumber string `json:"invoice_number"`
			Amount        any    `json:"amount"`
		} `json:"order"`
		Transaction struct {
			Status string `json:"status"`
		} `json:"transaction"`
	}
	if err := json.Unmarshal(body, &event); err != nil {
		return ErrInvalidInput
	}
	amount, err := dokuAmount(event.Order.Amount)
	if err != nil || event.Order.InvoiceNumber == "" {
		return ErrInvalidInput
	}
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var attemptID, storeID, orderID uuid.UUID
	var paymentID *uuid.UUID
	var expectedAmount int64
	err = tx.QueryRow(ctx, `SELECT id, store_id, order_id, payment_id, amount_idr FROM payment_attempts WHERE provider='doku' AND provider_invoice_number=$1 FOR UPDATE`, event.Order.InvoiceNumber).Scan(&attemptID, &storeID, &orderID, &paymentID, &expectedAmount)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if storeID != credentials.StoreID || amount != expectedAmount {
		return ErrForbidden
	}
	status := strings.ToUpper(strings.TrimSpace(event.Transaction.Status))
	_, err = tx.Exec(ctx, `UPDATE payment_attempts SET status=$2, updated_at=now() WHERE id=$1`, attemptID, strings.ToLower(status))
	if err != nil {
		return err
	}
	if status == "SUCCESS" && paymentID != nil {
		_, err = tx.Exec(ctx, `UPDATE payments SET status='verified', verified_at=COALESCE(verified_at,now()), updated_at=now() WHERE id=$1 AND store_id=$2 AND amount_idr=$3`, *paymentID, storeID, amount)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE orders SET payment_status='paid', updated_at=now() WHERE id=$1 AND store_id=$2 AND total_idr=$3 AND payment_status<>'paid'`, orderID, storeID, amount)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func dokuAmount(v any) (int64, error) {
	switch n := v.(type) {
	case float64:
		return int64(n), nil
	case string:
		return strconv.ParseInt(strings.Split(n, ".")[0], 10, 64)
	default:
		return 0, ErrInvalidInput
	}
}
