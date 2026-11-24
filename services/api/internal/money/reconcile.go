package money

// ReconciliationInput is the FIN-04 / E2E-F money model in whole IDR.
// payable = goods + fees − unavailable − credits (never negative).
// refund_entitled = max(0, paid − payable); remaining_due = max(0, payable − paid).
type ReconciliationInput struct {
	OriginalTotalIDR   int64 // goods + fees before unavailable/credits
	UnavailableDeltaIDR int64
	CreditAppliedIDR   int64
	PaidIDR            int64 // verified received (never rewrite historical payments)
}

// ReconciliationResult is the exact integer settlement snapshot.
type ReconciliationResult struct {
	PayableIDR       int64 `json:"payable_idr"`
	PaidIDR          int64 `json:"paid_idr"`
	RefundEntitledIDR int64 `json:"refund_entitled_idr"`
	RemainingDueIDR  int64 `json:"remaining_due_idr"`
	SettlementStatus string `json:"settlement_status"`
}

const (
	SettlementOpen             = "open"
	SettlementPartiallySettled = "partially_settled"
	SettlementSettled          = "settled"
	SettlementRefundDue        = "refund_due"
	SettlementRefunded         = "refunded"
)

// Reconcile computes payable / refund / due with integer-only math (PRD §14 FIN-06).
func Reconcile(in ReconciliationInput) ReconciliationResult {
	if in.OriginalTotalIDR < 0 {
		in.OriginalTotalIDR = 0
	}
	if in.UnavailableDeltaIDR < 0 {
		in.UnavailableDeltaIDR = 0
	}
	if in.CreditAppliedIDR < 0 {
		in.CreditAppliedIDR = 0
	}
	if in.PaidIDR < 0 {
		in.PaidIDR = 0
	}

	payable := in.OriginalTotalIDR - in.UnavailableDeltaIDR - in.CreditAppliedIDR
	if payable < 0 {
		payable = 0
	}

	refund := in.PaidIDR - payable
	if refund < 0 {
		refund = 0
	}
	due := payable - in.PaidIDR
	if due < 0 {
		due = 0
	}

	status := SettlementOpen
	switch {
	case refund > 0 && due == 0:
		status = SettlementRefundDue
	case due == 0 && in.PaidIDR > 0:
		status = SettlementSettled
	case in.PaidIDR > 0 && due > 0:
		status = SettlementPartiallySettled
	case in.PaidIDR == 0 && payable == 0:
		status = SettlementSettled
	default:
		status = SettlementOpen
	}

	return ReconciliationResult{
		PayableIDR:        payable,
		PaidIDR:           in.PaidIDR,
		RefundEntitledIDR: refund,
		RemainingDueIDR:   due,
		SettlementStatus:  status,
	}
}

// DepositAmountIDR computes deposit from a policy (percent in basis points or fixed).
func DepositAmountIDR(orderTotalIDR int64, depositType string, percentBPS int, fixedIDR int64) int64 {
	if orderTotalIDR < 0 {
		orderTotalIDR = 0
	}
	switch depositType {
	case "fixed":
		if fixedIDR < 0 {
			return 0
		}
		if fixedIDR > orderTotalIDR {
			return orderTotalIDR
		}
		return fixedIDR
	case "percent":
		if percentBPS <= 0 {
			return 0
		}
		if percentBPS > 10000 {
			percentBPS = 10000
		}
		// Round half-up on whole rupiah: (total * bps + 5000) / 10000
		return (orderTotalIDR*int64(percentBPS) + 5000) / 10000
	default:
		return 0
	}
}
