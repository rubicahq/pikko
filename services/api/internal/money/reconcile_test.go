package money

import "testing"

// E2E-F: Rp300k order / 250k paid / 110k unavailable → 190k payable, 60k refund.
func TestReconcileE2EF(t *testing.T) {
	got := Reconcile(ReconciliationInput{
		OriginalTotalIDR:    300_000,
		UnavailableDeltaIDR: 110_000,
		PaidIDR:             250_000,
	})
	if got.PayableIDR != 190_000 {
		t.Fatalf("payable: got %d want 190000", got.PayableIDR)
	}
	if got.RefundEntitledIDR != 60_000 {
		t.Fatalf("refund: got %d want 60000", got.RefundEntitledIDR)
	}
	if got.RemainingDueIDR != 0 {
		t.Fatalf("due: got %d want 0", got.RemainingDueIDR)
	}
	if got.SettlementStatus != SettlementRefundDue {
		t.Fatalf("status: got %s want %s", got.SettlementStatus, SettlementRefundDue)
	}
}

func TestReconcileFullyPaid(t *testing.T) {
	got := Reconcile(ReconciliationInput{
		OriginalTotalIDR: 100_000,
		PaidIDR:          100_000,
	})
	if got.PayableIDR != 100_000 || got.RefundEntitledIDR != 0 || got.RemainingDueIDR != 0 {
		t.Fatalf("unexpected %+v", got)
	}
	if got.SettlementStatus != SettlementSettled {
		t.Fatalf("status %s", got.SettlementStatus)
	}
}

func TestReconcilePartialDeposit(t *testing.T) {
	got := Reconcile(ReconciliationInput{
		OriginalTotalIDR: 500_000,
		PaidIDR:          100_000,
	})
	if got.RemainingDueIDR != 400_000 || got.SettlementStatus != SettlementPartiallySettled {
		t.Fatalf("unexpected %+v", got)
	}
}

func TestReconcileWithStoreCredit(t *testing.T) {
	got := Reconcile(ReconciliationInput{
		OriginalTotalIDR:    200_000,
		UnavailableDeltaIDR: 50_000,
		CreditAppliedIDR:    20_000,
		PaidIDR:             100_000,
	})
	// payable = 200k - 50k - 20k = 130k; paid 100k → due 30k
	if got.PayableIDR != 130_000 || got.RemainingDueIDR != 30_000 || got.RefundEntitledIDR != 0 {
		t.Fatalf("unexpected %+v", got)
	}
}

func TestReconcileNeverNegative(t *testing.T) {
	got := Reconcile(ReconciliationInput{
		OriginalTotalIDR:    50_000,
		UnavailableDeltaIDR: 80_000,
		CreditAppliedIDR:    10_000,
		PaidIDR:             0,
	})
	if got.PayableIDR != 0 {
		t.Fatalf("payable %d", got.PayableIDR)
	}
}

func TestDepositAmountIDR(t *testing.T) {
	if got := DepositAmountIDR(300_000, "percent", 5000, 0); got != 150_000 {
		t.Fatalf("50%% of 300k: got %d", got)
	}
	if got := DepositAmountIDR(300_000, "fixed", 0, 100_000); got != 100_000 {
		t.Fatalf("fixed: got %d", got)
	}
	if got := DepositAmountIDR(50_000, "fixed", 0, 100_000); got != 50_000 {
		t.Fatalf("fixed capped: got %d", got)
	}
	// half-up integer: (100001*3333 + 5000) / 10000 = 33330
	if got := DepositAmountIDR(100_001, "percent", 3333, 0); got != 33330 {
		t.Fatalf("bps round: got %d want 33330", got)
	}
	if got := DepositAmountIDR(100_000, "percent", 1, 0); got != 10 {
		t.Fatalf("1 bps of 100k: got %d want 10", got)
	}
}
