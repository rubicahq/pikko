package commerce

import (
	"time"

	"github.com/google/uuid"

	"github.com/rubicahq/pikko/services/api/internal/money"
)

const (
	PaymentKindFull          = "full"
	PaymentKindDeposit       = "deposit"
	PaymentKindFinal         = "final"
	PaymentKindPartial       = "partial"
	PaymentKindConsolidated  = "consolidated"

	RefundMethodBankTransfer = "bank_transfer"
	RefundMethodStoreCredit  = "store_credit"

	RefundPending    = "pending"
	RefundProcessing = "processing"
	RefundCompleted  = "completed"
	RefundFailed     = "failed"
	RefundCancelled  = "cancelled"

	AdjustUnavailable = "unavailable"
	AdjustManual      = "manual"
	AdjustFee         = "fee"
	AdjustCredit      = "credit_applied"
)

// OrderFinanceSummary is the reconciliation workspace row for one order (FIN-05).
type OrderFinanceSummary struct {
	OrderID             uuid.UUID                  `json:"order_id"`
	OrderNumber         string                     `json:"order_number"`
	CustomerID          *uuid.UUID                 `json:"customer_id,omitempty"`
	CustomerName        string                     `json:"customer_name"`
	CustomerEmail       string                     `json:"customer_email"`
	OriginalTotalIDR    int64                      `json:"original_total_idr"`
	UnavailableDeltaIDR int64                      `json:"unavailable_delta_idr"`
	CreditAppliedIDR    int64                      `json:"credit_applied_idr"`
	PayableIDR          int64                      `json:"payable_idr"`
	PaidIDR             int64                      `json:"paid_idr"`
	RefundEntitledIDR   int64                      `json:"refund_entitled_idr"`
	RemainingDueIDR     int64                      `json:"remaining_due_idr"`
	RefundedIDR         int64                      `json:"refunded_idr"`
	PaymentStatus       string                     `json:"payment_status"`
	SettlementStatus    string                     `json:"settlement_status"`
	Money               money.ReconciliationResult `json:"money"`
}

type DepositPolicy struct {
	ID             uuid.UUID `json:"id"`
	StoreID        uuid.UUID `json:"store_id"`
	Name           string    `json:"name"`
	AppliesTo      string    `json:"applies_to"`
	DepositType    string    `json:"deposit_type"`
	PercentBPS     *int      `json:"percent_bps,omitempty"`
	FixedAmountIDR *int64    `json:"fixed_amount_idr,omitempty"`
	IsEnabled      bool      `json:"is_enabled"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type UpsertDepositPolicyInput struct {
	Name           string `json:"name"`
	AppliesTo      string `json:"applies_to"`
	DepositType    string `json:"deposit_type"`
	PercentBPS     *int   `json:"percent_bps,omitempty"`
	FixedAmountIDR *int64 `json:"fixed_amount_idr,omitempty"`
	IsEnabled      *bool  `json:"is_enabled,omitempty"`
}

type RecordPaymentInput struct {
	OrderID          uuid.UUID   `json:"order_id"`
	PaymentMethodID  uuid.UUID   `json:"payment_method_id"`
	AmountIDR        int64       `json:"amount_idr"`
	PaymentKind      string      `json:"payment_kind"`
	IdempotencyKey   string      `json:"idempotency_key"`
	Note             *string     `json:"note,omitempty"`
	// Allocations for consolidated payments; when empty, full amount goes to OrderID.
	Allocations      []PaymentAllocationInput `json:"allocations,omitempty"`
}

type PaymentAllocationInput struct {
	OrderID   uuid.UUID `json:"order_id"`
	AmountIDR int64     `json:"amount_idr"`
}

type PaymentAllocation struct {
	ID        uuid.UUID `json:"id"`
	StoreID   uuid.UUID `json:"store_id"`
	PaymentID uuid.UUID `json:"payment_id"`
	OrderID   uuid.UUID `json:"order_id"`
	AmountIDR int64     `json:"amount_idr"`
	CreatedAt time.Time `json:"created_at"`
}

type FinancialAdjustmentInput struct {
	OrderID         uuid.UUID  `json:"order_id"`
	OrderItemID     *uuid.UUID `json:"order_item_id,omitempty"`
	AdjustmentType  string     `json:"adjustment_type"`
	AmountIDR       int64      `json:"amount_idr"`
	Reason          *string    `json:"reason,omitempty"`
	IdempotencyKey  string     `json:"idempotency_key"`
	RevisionID      *uuid.UUID `json:"revision_id,omitempty"`
}

type FinancialAdjustment struct {
	ID               uuid.UUID  `json:"id"`
	StoreID          uuid.UUID  `json:"store_id"`
	OrderID          uuid.UUID  `json:"order_id"`
	OrderItemID      *uuid.UUID `json:"order_item_id,omitempty"`
	RevisionID       *uuid.UUID `json:"revision_id,omitempty"`
	FinancialEventID *uuid.UUID `json:"financial_event_id,omitempty"`
	AdjustmentType   string     `json:"adjustment_type"`
	AmountIDR        int64      `json:"amount_idr"`
	Reason           *string    `json:"reason,omitempty"`
	ActorUserID      *uuid.UUID `json:"actor_user_id,omitempty"`
	IdempotencyKey   string     `json:"idempotency_key"`
	CreatedAt        time.Time  `json:"created_at"`
}

type CreateRefundInput struct {
	OrderID        uuid.UUID  `json:"order_id"`
	AmountIDR      int64      `json:"amount_idr"`
	Method         string     `json:"method"`
	Reason         *string    `json:"reason,omitempty"`
	IdempotencyKey string     `json:"idempotency_key"`
	EvidenceFileID *uuid.UUID `json:"evidence_file_id,omitempty"`
}

type ProcessRefundInput struct {
	ExpectedVersion int     `json:"expected_version"`
	Status          string  `json:"status"` // completed | failed | cancelled
	Reason          *string `json:"reason,omitempty"`
}

type Refund struct {
	ID               uuid.UUID  `json:"id"`
	StoreID          uuid.UUID  `json:"store_id"`
	OrderID          uuid.UUID  `json:"order_id"`
	CustomerID       *uuid.UUID `json:"customer_id,omitempty"`
	AmountIDR        int64      `json:"amount_idr"`
	Method           string     `json:"method"`
	Status           string     `json:"status"`
	Reason           *string    `json:"reason,omitempty"`
	EvidenceFileID   *uuid.UUID `json:"evidence_file_id,omitempty"`
	FinancialEventID *uuid.UUID `json:"financial_event_id,omitempty"`
	IdempotencyKey   string     `json:"idempotency_key"`
	Version          int        `json:"version"`
	ProcessedAt      *time.Time `json:"processed_at,omitempty"`
	ProcessedBy      *uuid.UUID `json:"processed_by,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type StoreCreditBalance struct {
	StoreID    uuid.UUID `json:"store_id"`
	CustomerID uuid.UUID `json:"customer_id"`
	BalanceIDR int64     `json:"balance_idr"`
	Version    int       `json:"version"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type StoreCreditLedgerEntry struct {
	ID               uuid.UUID  `json:"id"`
	StoreID          uuid.UUID  `json:"store_id"`
	CustomerID       uuid.UUID  `json:"customer_id"`
	EntryType        string     `json:"entry_type"`
	AmountIDR        int64      `json:"amount_idr"`
	BalanceAfterIDR  int64      `json:"balance_after_idr"`
	OrderID          *uuid.UUID `json:"order_id,omitempty"`
	RefundID         *uuid.UUID `json:"refund_id,omitempty"`
	FinancialEventID *uuid.UUID `json:"financial_event_id,omitempty"`
	IdempotencyKey   string     `json:"idempotency_key"`
	Note             *string    `json:"note,omitempty"`
	ActorUserID      *uuid.UUID `json:"actor_user_id,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

type ApplyStoreCreditInput struct {
	OrderID        uuid.UUID `json:"order_id"`
	AmountIDR      int64     `json:"amount_idr"`
	IdempotencyKey string    `json:"idempotency_key"`
	Note           *string   `json:"note,omitempty"`
}

type GrantStoreCreditInput struct {
	CustomerID     uuid.UUID  `json:"customer_id"`
	AmountIDR      int64      `json:"amount_idr"`
	IdempotencyKey string     `json:"idempotency_key"`
	OrderID        *uuid.UUID `json:"order_id,omitempty"`
	RefundID       *uuid.UUID `json:"refund_id,omitempty"`
	Note           *string    `json:"note,omitempty"`
}

type FinancialEvent struct {
	ID             uuid.UUID       `json:"id"`
	StoreID        uuid.UUID       `json:"store_id"`
	CustomerID     *uuid.UUID      `json:"customer_id,omitempty"`
	OrderID        *uuid.UUID       `json:"order_id,omitempty"`
	PaymentID      *uuid.UUID       `json:"payment_id,omitempty"`
	RefundID       *uuid.UUID       `json:"refund_id,omitempty"`
	EventType      string          `json:"event_type"`
	Direction      string          `json:"direction"`
	AmountIDR      int64           `json:"amount_idr"`
	IdempotencyKey string          `json:"idempotency_key"`
	Meta           map[string]any  `json:"meta,omitempty"`
	ActorUserID    *uuid.UUID      `json:"actor_user_id,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}
