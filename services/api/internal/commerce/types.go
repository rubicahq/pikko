package commerce

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidInput      = errors.New("invalid input")
	ErrNotFound          = errors.New("not found")
	ErrForbidden         = errors.New("forbidden")
	ErrConflict          = errors.New("conflict")
	ErrCheckoutDisabled  = errors.New("checkout disabled")
	ErrStoreUnavailable  = errors.New("store unavailable")
	ErrPriceChanged      = errors.New("price changed")
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrRateLimited       = errors.New("rate limited")
	ErrAlreadyVerified   = errors.New("already verified")
)

const (
	OrderPendingConfirmation = "pending_confirmation"
	OrderConfirmed           = "confirmed"
	OrderCompleted           = "completed"
	OrderCancelled           = "cancelled"

	PaymentUnpaid               = "unpaid"
	PaymentAwaitingVerification = "awaiting_verification"
	PaymentPaid                 = "paid"
	PaymentFailed               = "failed"
	PaymentRefunded             = "refunded"
	PaymentPartiallyRefunded    = "partially_refunded"

	FulfillmentUnfulfilled    = "unfulfilled"
	FulfillmentReadyToShip    = "ready_to_ship"
	FulfillmentReadyForPickup = "ready_for_pickup"
	FulfillmentShipped        = "shipped"
	FulfillmentDelivered      = "delivered"
	FulfillmentCollected      = "collected"
	FulfillmentCancelled      = "cancelled"

	PaymentRecordPending              = "pending"
	PaymentRecordAwaitingVerification = "awaiting_verification"
	PaymentRecordVerified             = "verified"
	PaymentRecordRejected             = "rejected"

	ReservationActive   = "active"
	ReservationReleased = "released"
	ReservationConsumed = "consumed"

	FulfillmentShipping     = "shipping"
	FulfillmentFreeShipping = "free_shipping"
	FulfillmentPickup       = "pickup"
)

type PaymentMethod struct {
	ID                   uuid.UUID `json:"id"`
	StoreID              uuid.UUID `json:"store_id"`
	MethodType           string    `json:"method_type"`
	BankName             string    `json:"bank_name"`
	AccountHolder        string    `json:"account_holder"`
	AccountNumber        string    `json:"account_number"`
	Instructions         *string   `json:"instructions,omitempty"`
	PaymentDeadlineHours int       `json:"payment_deadline_hours"`
	IsEnabled            bool      `json:"is_enabled"`
	DisplayOrder         int       `json:"display_order"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type FulfillmentSettings struct {
	StoreID             uuid.UUID `json:"store_id"`
	ShippingEnabled     bool      `json:"shipping_enabled"`
	ShippingFeeIDR      int64     `json:"shipping_fee_idr"`
	FreeShippingEnabled bool      `json:"free_shipping_enabled"`
	PickupEnabled       bool      `json:"pickup_enabled"`
	PickupInstructions  *string   `json:"pickup_instructions,omitempty"`
	ServiceAreaNotes    *string   `json:"service_area_notes,omitempty"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type CartLineInput struct {
	ProductID uuid.UUID  `json:"product_id"`
	VariantID *uuid.UUID `json:"variant_id,omitempty"`
	Quantity  int        `json:"quantity"`
}

type CheckoutAddress struct {
	RecipientName string  `json:"recipient_name"`
	Phone         *string `json:"phone,omitempty"`
	Line1         string  `json:"line1"`
	Line2         *string `json:"line2,omitempty"`
	City          string  `json:"city"`
	Province      *string `json:"province,omitempty"`
	PostalCode    *string `json:"postal_code,omitempty"`
	Country       string  `json:"country"`
}

type CheckoutSubmitInput struct {
	Items             []CartLineInput  `json:"items"`
	CustomerName      string           `json:"customer_name"`
	CustomerEmail     string           `json:"customer_email"`
	CustomerPhone     *string          `json:"customer_phone,omitempty"`
	FulfillmentMethod string           `json:"fulfillment_method"`
	ShippingAddress   *CheckoutAddress `json:"shipping_address,omitempty"`
	CustomerNotes     *string          `json:"customer_notes,omitempty"`
	PaymentMethodID   uuid.UUID        `json:"payment_method_id"`
	IdempotencyKey    string           `json:"idempotency_key"`
	// ClientExpectedTotalIDR, when set, must match server total or ErrPriceChanged.
	ClientExpectedTotalIDR *int64 `json:"client_expected_total_idr,omitempty"`
}

type PricedLine struct {
	ProductID     uuid.UUID  `json:"product_id"`
	VariantID     *uuid.UUID `json:"variant_id,omitempty"`
	ProductName   string     `json:"product_name"`
	ProductSlug   string     `json:"product_slug"`
	SKU           *string    `json:"sku,omitempty"`
	VariantLabel  *string    `json:"variant_label,omitempty"`
	UnitPriceIDR  int64      `json:"unit_price_idr"`
	Quantity      int        `json:"quantity"`
	LineTotalIDR  int64      `json:"line_total_idr"`
	AvailableQty  *int       `json:"available_qty,omitempty"`
	InventoryMode string     `json:"inventory_mode"`
}

type CheckoutPreview struct {
	StoreID           uuid.UUID    `json:"store_id"`
	StoreSlug         string       `json:"store_slug"`
	StoreName         string       `json:"store_name"`
	Currency          string       `json:"currency"`
	Lines             []PricedLine `json:"lines"`
	SubtotalIDR       int64        `json:"subtotal_idr"`
	ShippingFeeIDR    int64        `json:"shipping_fee_idr"`
	HandlingFeeIDR    int64        `json:"handling_fee_idr"`
	TotalIDR          int64        `json:"total_idr"`
	FulfillmentMethod string       `json:"fulfillment_method,omitempty"`
	CheckoutEnabled   bool         `json:"checkout_enabled"`
	PaymentMethods    []PaymentMethod      `json:"payment_methods,omitempty"`
	Fulfillment       *FulfillmentSettings `json:"fulfillment,omitempty"`
}

type OrderItem struct {
	ID           uuid.UUID  `json:"id"`
	ProductID    *uuid.UUID `json:"product_id,omitempty"`
	VariantID    *uuid.UUID `json:"variant_id,omitempty"`
	ProductName  string     `json:"product_name"`
	ProductSlug  *string    `json:"product_slug,omitempty"`
	SKU          *string    `json:"sku,omitempty"`
	VariantLabel *string    `json:"variant_label,omitempty"`
	UnitPriceIDR int64      `json:"unit_price_idr"`
	Quantity     int        `json:"quantity"`
	LineTotalIDR int64      `json:"line_total_idr"`
}

type Order struct {
	ID                  uuid.UUID   `json:"id"`
	StoreID             uuid.UUID   `json:"store_id"`
	CustomerID          *uuid.UUID  `json:"customer_id,omitempty"`
	OrderNumber         string      `json:"order_number"`
	SellingModel        string      `json:"selling_model"`
	OrderStatus         string      `json:"order_status"`
	PaymentStatus       string      `json:"payment_status"`
	FulfillmentStatus   string      `json:"fulfillment_status"`
	Currency            string      `json:"currency"`
	SubtotalIDR         int64       `json:"subtotal_idr"`
	ShippingFeeIDR      int64       `json:"shipping_fee_idr"`
	HandlingFeeIDR      int64       `json:"handling_fee_idr"`
	TotalIDR            int64       `json:"total_idr"`
	FulfillmentMethod   string      `json:"fulfillment_method"`
	CustomerName        string      `json:"customer_name"`
	CustomerEmail       string      `json:"customer_email"`
	CustomerPhone       *string     `json:"customer_phone,omitempty"`
	ShippingRecipient   *string     `json:"shipping_recipient,omitempty"`
	ShippingPhone       *string     `json:"shipping_phone,omitempty"`
	ShippingLine1       *string     `json:"shipping_line1,omitempty"`
	ShippingLine2       *string     `json:"shipping_line2,omitempty"`
	ShippingCity        *string     `json:"shipping_city,omitempty"`
	ShippingProvince    *string     `json:"shipping_province,omitempty"`
	ShippingPostalCode  *string     `json:"shipping_postal_code,omitempty"`
	ShippingCountry     *string     `json:"shipping_country,omitempty"`
	CustomerNotes       *string     `json:"customer_notes,omitempty"`
	InternalNotes       *string     `json:"internal_notes,omitempty"`
	CancelReason        *string     `json:"cancel_reason,omitempty"`
	PaymentDeadlineAt   *time.Time  `json:"payment_deadline_at,omitempty"`
	ConfirmedAt         *time.Time  `json:"confirmed_at,omitempty"`
	CompletedAt         *time.Time  `json:"completed_at,omitempty"`
	CancelledAt         *time.Time  `json:"cancelled_at,omitempty"`
	CreatedAt           time.Time   `json:"created_at"`
	UpdatedAt           time.Time   `json:"updated_at"`
	Items               []OrderItem `json:"items,omitempty"`
	TrackingURL         string      `json:"tracking_url,omitempty"`
	TrackingToken       string      `json:"tracking_token,omitempty"` // only on create response
}

type Payment struct {
	ID                      uuid.UUID  `json:"id"`
	StoreID                 uuid.UUID  `json:"store_id"`
	OrderID                 uuid.UUID  `json:"order_id"`
	PaymentMethodID         *uuid.UUID `json:"payment_method_id,omitempty"`
	MethodType              string     `json:"method_type"`
	AmountIDR               int64      `json:"amount_idr"`
	Status                  string     `json:"status"`
	BankNameSnapshot        *string    `json:"bank_name_snapshot,omitempty"`
	AccountHolderSnapshot   *string    `json:"account_holder_snapshot,omitempty"`
	AccountNumberSnapshot   *string    `json:"account_number_snapshot,omitempty"`
	InstructionsSnapshot    *string    `json:"instructions_snapshot,omitempty"`
	VerifiedAt              *time.Time `json:"verified_at,omitempty"`
	RejectedAt              *time.Time `json:"rejected_at,omitempty"`
	RejectReason            *string    `json:"reject_reason,omitempty"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
	Evidence                []Evidence `json:"evidence,omitempty"`
}

type Evidence struct {
	ID          uuid.UUID `json:"id"`
	FileID      uuid.UUID `json:"file_id"`
	SubmittedAt time.Time `json:"submitted_at"`
	Note        *string   `json:"note,omitempty"`
	ContentType string    `json:"content_type,omitempty"`
	SignedURL   string    `json:"signed_url,omitempty"`
}

type Customer struct {
	ID                      uuid.UUID  `json:"id"`
	StoreID                 uuid.UUID  `json:"store_id"`
	Name                    string     `json:"name"`
	Email                   string     `json:"email"`
	Phone                   *string    `json:"phone,omitempty"`
	FirstOrderAt            *time.Time `json:"first_order_at,omitempty"`
	LastOrderAt             *time.Time `json:"last_order_at,omitempty"`
	TotalOrders             int        `json:"total_orders"`
	CompletedOrderValueIDR  int64      `json:"completed_order_value_idr"`
	InternalNotes           *string    `json:"internal_notes,omitempty"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
}

type TrackingView struct {
	StoreName         string         `json:"store_name"`
	StoreSlug         string         `json:"store_slug"`
	PrimaryColor      string         `json:"primary_color,omitempty"`
	ContactEmail      *string        `json:"contact_email,omitempty"`
	WhatsApp          *string        `json:"whatsapp,omitempty"`
	Order             Order          `json:"order"`
	Payment           *Payment       `json:"payment,omitempty"`
	CanSubmitEvidence bool           `json:"can_submit_evidence"`
}
