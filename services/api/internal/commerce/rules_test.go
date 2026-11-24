package commerce

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestScopePrecedence(t *testing.T) {
	if scopePrecedence(ScopeBatch) <= scopePrecedence(ScopeSellingModel) {
		t.Fatal("batch should beat selling model")
	}
	if scopePrecedence(ScopeSellingModel) <= scopePrecedence(ScopeStore) {
		t.Fatal("selling model should beat store")
	}
}

func TestEvaluateRulesBatchOverridesStore(t *testing.T) {
	batchID := uuid.New()
	storeFee := int64(5000)
	batchFee := int64(15000)
	pct := 10.0
	vid1, vid2, vid3 := uuid.New(), uuid.New(), uuid.New()
	rules := []BusinessRule{
		{
			ID: uuid.New(), RuleType: RuleHandlingFee, ScopeType: ScopeStore, VersionID: &vid1,
			Config: RuleConfig{FeeMode: FeeModeFixed, FeeFixedIDR: &storeFee},
		},
		{
			ID: uuid.New(), RuleType: RuleHandlingFee, ScopeType: ScopeBatch, ScopeBatchID: &batchID, VersionID: &vid2,
			Config: RuleConfig{FeeMode: FeeModeFixed, FeeFixedIDR: &batchFee},
		},
		{
			ID: uuid.New(), RuleType: RuleHandlingFee, ScopeType: ScopeSellingModel,
			ScopeSellingModel: strPtr("preorder"), VersionID: &vid3,
			Config: RuleConfig{FeeMode: FeeModePercent, FeePercent: &pct},
		},
	}
	eval, err := EvaluateRules(rules, "preorder", &batchID, 100_000, 2, 10_000)
	if err != nil {
		t.Fatal(err)
	}
	if eval.HandlingFeeIDR != 15000 {
		t.Fatalf("expected batch fee 15000, got %d", eval.HandlingFeeIDR)
	}
	if eval.TotalIDR != 125_000 {
		t.Fatalf("expected total 125000, got %d", eval.TotalIDR)
	}
}

func TestEvaluateRulesMinimumAndApproval(t *testing.T) {
	minSub := int64(50_000)
	minQty := 3
	vid1, vid2 := uuid.New(), uuid.New()
	rules := []BusinessRule{
		{
			ID: uuid.New(), RuleType: RuleMinimumOrder, ScopeType: ScopeStore, VersionID: &vid1,
			Config: RuleConfig{MinSubtotalIDR: &minSub, MinQuantity: &minQty},
		},
		{
			ID: uuid.New(), RuleType: RuleOrderApproval, ScopeType: ScopeStore, VersionID: &vid2,
			Config: RuleConfig{},
		},
	}
	_, err := EvaluateRules(rules, "ready_stock", nil, 40_000, 5, 0)
	if !errors.Is(err, ErrMinimumNotMet) {
		t.Fatalf("expected minimum not met, got %v", err)
	}
	eval, err := EvaluateRules(rules, "ready_stock", nil, 60_000, 3, 5000)
	if err != nil {
		t.Fatal(err)
	}
	if !eval.RequiresApproval {
		t.Fatal("expected approval")
	}
	if eval.TotalIDR != 65_000 {
		t.Fatalf("total %d", eval.TotalIDR)
	}
	if len(eval.Applied) != 2 {
		t.Fatalf("applied %d", len(eval.Applied))
	}
	var result map[string]any
	_ = json.Unmarshal(eval.Applied[0].Result, &result)
}

func TestEvaluateRulesPercentRounding(t *testing.T) {
	pct := 2.5
	vid := uuid.New()
	rules := []BusinessRule{{
		ID: uuid.New(), RuleType: RuleHandlingFee, ScopeType: ScopeStore, VersionID: &vid,
		Config: RuleConfig{FeeMode: FeeModePercent, FeePercent: &pct},
	}}
	// 2.5% of 10001 = 250.025 → 250 half-up
	eval, err := EvaluateRules(rules, "ready_stock", nil, 10001, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if eval.HandlingFeeIDR != 250 {
		t.Fatalf("got %d", eval.HandlingFeeIDR)
	}
}

func TestResolveCartGroup(t *testing.T) {
	b1, b2 := uuid.New(), uuid.New()
	_, _, err := resolveCartGroup([]CartLineInput{
		{ProductID: uuid.New(), Quantity: 1},
		{ProductID: uuid.New(), Quantity: 1, BatchID: &b1},
	})
	if !errors.Is(err, ErrIncompatibleCart) {
		t.Fatal(err)
	}
	_, _, err = resolveCartGroup([]CartLineInput{
		{ProductID: uuid.New(), Quantity: 1, BatchID: &b1},
		{ProductID: uuid.New(), Quantity: 1, BatchID: &b2},
	})
	if !errors.Is(err, ErrIncompatibleCart) {
		t.Fatal(err)
	}
	sm, batch, err := resolveCartGroup([]CartLineInput{
		{ProductID: uuid.New(), Quantity: 1, BatchID: &b1},
		{ProductID: uuid.New(), Quantity: 2, BatchID: &b1},
	})
	if err != nil || sm != "preorder" || batch == nil || *batch != b1 {
		t.Fatalf("%s %v %v", sm, batch, err)
	}
}

func strPtr(s string) *string { return &s }
