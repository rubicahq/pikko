package portal

import (
	"time"

	"github.com/google/uuid"

	"github.com/rubicahq/pikko/services/api/internal/commerce"
)

const (
	RefundMethodBankTransfer = "bank_transfer"
	RefundMethodStoreCredit  = "store_credit"

	AddressChangePending   = "pending"
	AddressChangeApproved  = "approved"
	AddressChangeRejected  = "rejected"
	AddressChangeCancelled = "cancelled"
)

type Customer struct {
	ID           uuid.UUID `json:"id"`
	StoreID      uuid.UUID `json:"store_id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Phone        *string   `json:"phone,omitempty"`
	TotalOrders  int       `json:"total_orders"`
	CreatedAt    time.Time `json:"created_at"`
}

type Session struct {
	ID         uuid.UUID `json:"id"`
	StoreID    uuid.UUID `json:"store_id"`
	CustomerID uuid.UUID `json:"customer_id"`
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
	LastSeen   time.Time `json:"last_seen_at"`
}

type StoreMeta struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	PrimaryColor string    `json:"primary_color,omitempty"`
}

type RequestMagicLinkResult struct {
	Email           string `json:"email"`
	CooldownSeconds int    `json:"cooldown_seconds"`
	DevToken        string `json:"dev_token,omitempty"`
}

type VerifyResult struct {
	Customer     Customer  `json:"customer"`
	Store        StoreMeta `json:"store"`
	SessionToken string    `json:"-"`
	RedirectPath string    `json:"redirect_path"`
}

type Dashboard struct {
	Store              StoreMeta                     `json:"store"`
	Customer           Customer                      `json:"customer"`
	Orders             []commerce.Order              `json:"orders"`
	OrdersTotal        int                           `json:"orders_total"`
	RefundPreference   *RefundPreference             `json:"refund_preference,omitempty"`
	StoreCreditEnabled bool                          `json:"store_credit_enabled"`
	StoreCredit        *commerce.StoreCreditBalance  `json:"store_credit,omitempty"`
	AddressesCount     int                           `json:"addresses_count"`
}

type OrderDetail struct {
	Order        commerce.Order                   `json:"order"`
	Finance      *commerce.OrderFinanceSummary    `json:"finance,omitempty"`
	Payments     []commerce.Payment               `json:"payments"`
	Adjustments  []commerce.FinancialAdjustment   `json:"adjustments"`
	Refunds      []commerce.Refund                `json:"refunds"`
	AddressChange *AddressChangeRequest           `json:"pending_address_change,omitempty"`
	CanRequestAddressChange bool                  `json:"can_request_address_change"`
}

type RefundPreference struct {
	StoreID          uuid.UUID `json:"store_id"`
	CustomerID       uuid.UUID `json:"customer_id"`
	PreferredMethod  string    `json:"preferred_method"`
	BankName         *string   `json:"bank_name,omitempty"`
	AccountHolder    *string   `json:"account_holder,omitempty"`
	AccountNumber    *string   `json:"account_number,omitempty"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type UpsertRefundPreferenceInput struct {
	PreferredMethod string  `json:"preferred_method"`
	BankName        *string `json:"bank_name,omitempty"`
	AccountHolder   *string `json:"account_holder,omitempty"`
	AccountNumber   *string `json:"account_number,omitempty"`
}

type Address struct {
	ID            uuid.UUID `json:"id"`
	StoreID       uuid.UUID `json:"store_id"`
	CustomerID    uuid.UUID `json:"customer_id"`
	Label         *string   `json:"label,omitempty"`
	RecipientName string    `json:"recipient_name"`
	Phone         *string   `json:"phone,omitempty"`
	Line1         string    `json:"line1"`
	Line2         *string   `json:"line2,omitempty"`
	City          string    `json:"city"`
	Province      *string   `json:"province,omitempty"`
	PostalCode    *string   `json:"postal_code,omitempty"`
	Country       string    `json:"country"`
	IsDefault     bool      `json:"is_default"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type AddressInput struct {
	Label         *string `json:"label,omitempty"`
	RecipientName string  `json:"recipient_name"`
	Phone         *string `json:"phone,omitempty"`
	Line1         string  `json:"line1"`
	Line2         *string `json:"line2,omitempty"`
	City          string  `json:"city"`
	Province      *string `json:"province,omitempty"`
	PostalCode    *string `json:"postal_code,omitempty"`
	Country       string  `json:"country"`
	IsDefault     *bool   `json:"is_default,omitempty"`
}

type AddressChangeRequest struct {
	ID            uuid.UUID  `json:"id"`
	StoreID       uuid.UUID  `json:"store_id"`
	CustomerID    uuid.UUID  `json:"customer_id"`
	OrderID       uuid.UUID  `json:"order_id"`
	OrderNumber   string     `json:"order_number,omitempty"`
	Status        string     `json:"status"`
	RecipientName string     `json:"recipient_name"`
	Phone         *string    `json:"phone,omitempty"`
	Line1         string     `json:"line1"`
	Line2         *string    `json:"line2,omitempty"`
	City          string     `json:"city"`
	Province      *string    `json:"province,omitempty"`
	PostalCode    *string    `json:"postal_code,omitempty"`
	Country       string     `json:"country"`
	CustomerNote  *string    `json:"customer_note,omitempty"`
	MerchantNote  *string    `json:"merchant_note,omitempty"`
	ReviewedBy    *uuid.UUID `json:"reviewed_by,omitempty"`
	ReviewedAt    *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type CreateAddressChangeInput struct {
	RecipientName string  `json:"recipient_name"`
	Phone         *string `json:"phone,omitempty"`
	Line1         string  `json:"line1"`
	Line2         *string `json:"line2,omitempty"`
	City          string  `json:"city"`
	Province      *string `json:"province,omitempty"`
	PostalCode    *string `json:"postal_code,omitempty"`
	Country       string  `json:"country"`
	CustomerNote  *string `json:"customer_note,omitempty"`
}

type ReviewAddressChangeInput struct {
	Status       string  `json:"status"` // approved | rejected
	MerchantNote *string `json:"merchant_note,omitempty"`
}
