package commerce

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/rubicahq/pikko/services/api/internal/email"
	"github.com/rubicahq/pikko/services/api/internal/money"
	"github.com/rubicahq/pikko/services/api/internal/notifications"
)

// Notifier is optional; when nil, in-app notifications are skipped.
type Notifier interface {
	Create(ctx context.Context, storeID uuid.UUID, in notifications.CreateInput) (*notifications.Notification, error)
	ShouldInApp(ctx context.Context, storeID uuid.UUID, typ string) bool
}

func (s *Service) notifyInApp(ctx context.Context, storeID uuid.UUID, typ, title, body, resourceType string, resourceID *uuid.UUID) {
	if s.Notifier == nil {
		return
	}
	if !s.Notifier.ShouldInApp(ctx, storeID, typ) {
		return
	}
	_, _ = s.Notifier.Create(ctx, storeID, notifications.CreateInput{
		Type: typ, Title: title, Body: body, ResourceType: resourceType, ResourceID: resourceID,
	})
}

func (s *Service) notifyNewOrderInApp(ctx context.Context, storeID uuid.UUID, o *Order) {
	typ := notifications.TypeNewOrder
	title := fmt.Sprintf("New order %s", o.OrderNumber)
	if o.RequiresApproval || o.OrderStatus == OrderPendingConfirmation {
		typ = notifications.TypePendingConfirmation
		title = fmt.Sprintf("Order %s needs confirmation", o.OrderNumber)
	}
	body := fmt.Sprintf("%s · %s", o.CustomerName, money.FormatIDR(o.TotalIDR))
	oid := o.ID
	s.notifyInApp(ctx, storeID, typ, title, body, "order", &oid)
}

func (s *Service) notifyPaymentEvidenceInApp(ctx context.Context, storeID uuid.UUID, o *Order) {
	oid := o.ID
	s.notifyInApp(ctx, storeID, notifications.TypePaymentEvidence,
		fmt.Sprintf("Payment evidence for %s", o.OrderNumber),
		fmt.Sprintf("%s submitted transfer proof", o.CustomerName),
		"order", &oid,
	)
}

func (s *Service) notifyBatchClosedInApp(ctx context.Context, storeID uuid.UUID, batchID uuid.UUID, batchName string) {
	s.notifyInApp(ctx, storeID, notifications.TypeBatchClosed,
		fmt.Sprintf("Batch closed: %s", batchName),
		"New orders are no longer accepted for this preorder batch.",
		"preorder_batch", &batchID,
	)
}

func (s *Service) notifyUnavailableAllocation(
	ctx context.Context, storeID uuid.UUID, orderID uuid.UUID, productName string, unavailableQty int,
) {
	if unavailableQty <= 0 {
		return
	}
	o, err := s.GetOrder(ctx, storeID, orderID)
	if err != nil || o == nil {
		return
	}
	oid := orderID
	s.notifyInApp(ctx, storeID, notifications.TypePurchasingIssue,
		fmt.Sprintf("Unavailable item on %s", o.OrderNumber),
		fmt.Sprintf("%s × %d marked unavailable", productName, unavailableQty),
		"order", &oid,
	)

	if s.Email == nil || s.Cfg == nil || strings.TrimSpace(o.CustomerEmail) == "" {
		return
	}
	trackURL := ""
	if tok, err := s.EnsureTrackingToken(ctx, storeID, orderID); err == nil && s.URLs != nil {
		trackURL = s.URLs.Tracking(tok)
	}
	storeName := ""
	_ = s.DB.QueryRow(ctx, `SELECT name FROM stores WHERE id=$1`, storeID).Scan(&storeName)
	if storeName == "" {
		storeName = "Store"
	}
	subject, html, text := email.ProductUnavailableContent(storeName, o.OrderNumber, productName, trackURL)
	_ = s.Email.Enqueue(ctx, email.Message{
		TemplateKey: "product_unavailable_customer", To: o.CustomerEmail,
		From: s.Cfg.EmailNotificationFrom, FromName: storeName, ReplyTo: s.Cfg.EmailReplyTo,
		Subject: subject, HTML: html, Text: text,
		IdempotencyKey: fmt.Sprintf("email:unavailable:%s:%s:%d", orderID.String(), productName, unavailableQty),
	})
}
