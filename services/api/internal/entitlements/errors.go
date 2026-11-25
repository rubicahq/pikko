package entitlements

import "errors"

var (
	ErrNotFound            = errors.New("subscription not found")
	ErrInvalidInput        = errors.New("invalid input")
	ErrFeatureNotEntitled  = errors.New("feature not entitled")
	ErrLimitExceeded       = errors.New("plan limit exceeded")
	ErrSubscriptionRequired = errors.New("subscription required")
	ErrPlanChangeDisabled  = errors.New("plan change disabled")
	ErrTestBypassDisabled  = errors.New("test bypass disabled")
	ErrRealBillingDisabled = errors.New("real billing disabled")
	ErrTrialDisabled       = errors.New("commercial trial disabled")
	ErrForbidden           = errors.New("forbidden")
	ErrDowngradeBlocked    = errors.New("downgrade blocked by over-limit usage")
	ErrSubscriptionsOff    = errors.New("subscriptions disabled")
)

// LimitError carries structured limit context for API responses.
type LimitError struct {
	LimitKey string
	Used     int64
	Limit    int64
	Message  string
}

func (e *LimitError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "plan limit exceeded"
}

func (e *LimitError) Unwrap() error { return ErrLimitExceeded }

// FeatureError carries feature key for API responses.
type FeatureError struct {
	FeatureKey string
	Message    string
}

func (e *FeatureError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "feature not entitled on current plan"
}

func (e *FeatureError) Unwrap() error { return ErrFeatureNotEntitled }
