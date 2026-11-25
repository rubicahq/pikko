package entitlements

import (
	"testing"
	"time"
)

func TestPeriodYMJakarta(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatal(err)
	}
	// 2026-03-31 23:30 UTC = 2026-04-01 06:30 Jakarta
	ts := time.Date(2026, 3, 31, 23, 30, 0, 0, time.UTC)
	got := PeriodYM(ts)
	want := ts.In(loc).Format("2006-01")
	if got != want || got != "2026-04" {
		t.Fatalf("PeriodYM=%s want %s (2026-04)", got, want)
	}
}

func TestLimitErrorUnwrap(t *testing.T) {
	err := &LimitError{LimitKey: LimitOrdersPerMonth, Used: 100, Limit: 100}
	if err.Error() == "" {
		t.Fatal("empty message")
	}
	if !isLimit(err) {
		t.Fatal("expected limit error")
	}
}

func isLimit(err error) bool {
	_, ok := err.(*LimitError)
	return ok
}

func TestDefaultFlags(t *testing.T) {
	f := DefaultFlags()
	if !f.SubscriptionsEnabled || !f.TestBypassEnabled {
		t.Fatalf("expected testing defaults on: %+v", f)
	}
	if f.CommercialTrialEnabled || f.RealBillingEnabled || f.AutoRenewEnabled {
		t.Fatalf("expected commercial/billing off: %+v", f)
	}
}
