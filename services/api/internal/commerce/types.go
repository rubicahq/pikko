package commerce

import (
	"encoding/json"
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

// Additional Phase 4 errors are defined in rules.go.

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
	FulfillmentPurchasing     = "purchasing"
	FulfillmentAwaitingArrival = "awaiting_arrival"
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
	// BatchID set for preorder cart group; omit/null for ready-stock (PRD §20/§22).
	BatchID *uuid.UUID `json:"batch_id,omitempty"`
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
	BatchID       *uuid.UUID `json:"batch_id,omitempty"`
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
	StoreID            uuid.UUID              `json:"store_id"`
	StoreSlug          string                 `json:"store_slug"`
	StoreName          string                 `json:"store_name"`
	Currency           string                 `json:"currency"`
	SellingModel       string                 `json:"selling_model"`
	BatchID            *uuid.UUID             `json:"batch_id,omitempty"`
	BatchName          string                 `json:"batch_name,omitempty"`
	BatchSlug          string                 `json:"batch_slug,omitempty"`
	Lines              []PricedLine           `json:"lines"`
	SubtotalIDR        int64                  `json:"subtotal_idr"`
	ShippingFeeIDR     int64                  `json:"shipping_fee_idr"`
	HandlingFeeIDR     int64                  `json:"handling_fee_idr"`
	TotalIDR           int64                  `json:"total_idr"`
	RequiresApproval   bool                   `json:"requires_approval"`
	MinimumOK          bool                   `json:"minimum_ok"`
	MinimumMessage     string                 `json:"minimum_message,omitempty"`
	AppliedRules       []AppliedRule          `json:"applied_rules,omitempty"`
	FulfillmentMethod  string                 `json:"fulfillment_method,omitempty"`
	CheckoutEnabled    bool                   `json:"checkout_enabled"`
	PaymentMethods     []PaymentMethod        `json:"payment_methods,omitempty"`
	Fulfillment        *FulfillmentSettings   `json:"fulfillment,omitempty"`
}

type OrderItem struct {
	ID                   uuid.UUID  `json:"id"`
	ProductID            *uuid.UUID `json:"product_id,omitempty"`
	VariantID            *uuid.UUID `json:"variant_id,omitempty"`
	BatchID              *uuid.UUID `json:"batch_id,omitempty"`
	ProductName          string     `json:"product_name"`
	ProductSlug          *string    `json:"product_slug,omitempty"`
	SKU                  *string    `json:"sku,omitempty"`
	VariantLabel         *string    `json:"variant_label,omitempty"`
	UnitPriceIDR         int64      `json:"unit_price_idr"`
	Quantity             int        `json:"quantity"`
	OriginalQuantity     int        `json:"original_quantity"`
	LineTotalIDR         int64      `json:"line_total_idr"`
	AvailabilityStatus   string     `json:"availability_status"`
	AllocatedQty         int        `json:"allocated_qty"`
	PackedQty            int        `json:"packed_qty"`
	AdjustedLineTotalIDR *int64     `json:"adjusted_line_total_idr,omitempty"`
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
	PreorderBatchID     *uuid.UUID  `json:"preorder_batch_id,omitempty"`
	BatchNameSnapshot   *string     `json:"batch_name_snapshot,omitempty"`
	BatchSlugSnapshot   *string     `json:"batch_slug_snapshot,omitempty"`
	RequiresApproval    bool        `json:"requires_approval"`
	Source              string      `json:"source"`
	CreatedByUserID     *uuid.UUID  `json:"created_by_user_id,omitempty"`
	CampaignID          *uuid.UUID  `json:"campaign_id,omitempty"`
	Items               []OrderItem `json:"items,omitempty"`
	TrackingURL         string      `json:"tracking_url,omitempty"`
	TrackingToken       string      `json:"tracking_token,omitempty"` // only on create response
}

const (
	OrderSourceCustomer  = "customer"
	OrderSourceAdmin     = "admin"
	OrderSourceCSVImport = "csv_import"

	ItemAvailabilityAvailable            = "available"
	ItemAvailabilityPartiallyUnavailable = "partially_unavailable"
	ItemAvailabilityUnavailable          = "unavailable"
)

type OrderRevision struct {
	ID             uuid.UUID       `json:"id"`
	StoreID        uuid.UUID       `json:"store_id"`
	OrderID        uuid.UUID       `json:"order_id"`
	RevisionNumber int             `json:"revision_number"`
	ChangeType     string          `json:"change_type"`
	Summary        string          `json:"summary"`
	BeforeSnapshot json.RawMessage `json:"before_snapshot,omitempty"`
	AfterSnapshot  json.RawMessage `json:"after_snapshot,omitempty"`
	ActorUserID    *uuid.UUID      `json:"actor_user_id,omitempty"`
	Reason         *string         `json:"reason,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}

type OrderItemEvent struct {
	ID               uuid.UUID       `json:"id"`
	StoreID          uuid.UUID       `json:"store_id"`
	OrderID          uuid.UUID       `json:"order_id"`
	OrderItemID      uuid.UUID       `json:"order_item_id"`
	RevisionID       *uuid.UUID      `json:"revision_id,omitempty"`
	EventType        string          `json:"event_type"`
	FromQuantity     *int            `json:"from_quantity,omitempty"`
	ToQuantity       *int            `json:"to_quantity,omitempty"`
	FromAvailability *string         `json:"from_availability,omitempty"`
	ToAvailability   *string         `json:"to_availability,omitempty"`
	Meta             json.RawMessage `json:"meta,omitempty"`
	ActorUserID      *uuid.UUID      `json:"actor_user_id,omitempty"`
	Reason           *string         `json:"reason,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
}

type AdminOrderDraft struct {
	ID               uuid.UUID       `json:"id"`
	StoreID          uuid.UUID       `json:"store_id"`
	CreatedByUserID  uuid.UUID       `json:"created_by_user_id"`
	Title            string          `json:"title"`
	Payload          json.RawMessage `json:"payload"`
	LastValidatedAt  *time.Time      `json:"last_validated_at,omitempty"`
	ValidationErrors json.RawMessage `json:"validation_errors"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type Payment struct {
	ID                      uuid.UUID           `json:"id"`
	StoreID                 uuid.UUID           `json:"store_id"`
	OrderID                 uuid.UUID           `json:"order_id"`
	PaymentMethodID         *uuid.UUID          `json:"payment_method_id,omitempty"`
	MethodType              string              `json:"method_type"`
	AmountIDR               int64               `json:"amount_idr"`
	Status                  string              `json:"status"`
	BankNameSnapshot        *string             `json:"bank_name_snapshot,omitempty"`
	AccountHolderSnapshot   *string             `json:"account_holder_snapshot,omitempty"`
	AccountNumberSnapshot   *string             `json:"account_number_snapshot,omitempty"`
	InstructionsSnapshot    *string             `json:"instructions_snapshot,omitempty"`
	VerifiedAt              *time.Time          `json:"verified_at,omitempty"`
	RejectedAt              *time.Time          `json:"rejected_at,omitempty"`
	RejectReason            *string             `json:"reject_reason,omitempty"`
	CreatedAt               time.Time           `json:"created_at"`
	UpdatedAt               time.Time           `json:"updated_at"`
	Evidence                []Evidence          `json:"evidence,omitempty"`
	PaymentKind             string              `json:"payment_kind,omitempty"`
	IdempotencyKey          *string             `json:"idempotency_key,omitempty"`
	Note                    *string             `json:"note,omitempty"`
	Version                 int                 `json:"version,omitempty"`
	Allocations             []PaymentAllocation `json:"allocations,omitempty"`
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
	Shipments         []Shipment     `json:"shipments,omitempty"`
	CanSubmitEvidence bool           `json:"can_submit_evidence"`
}
