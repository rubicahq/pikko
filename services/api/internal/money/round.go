package money

import "math"

// RoundHalfUpIDR rounds a float amount to whole rupiah using half-up
// (deterministic; PRD §20/§31). Never use float for stored money — only for
// intermediate percentage fee calculations before rounding.
func RoundHalfUpIDR(v float64) int64 {
	if v >= 0 {
		return int64(math.Floor(v + 0.5))
	}
	return int64(math.Ceil(v - 0.5))
}

// PercentOfIDR applies percent (e.g. 2.5 for 2.5%) to an integer IDR base
// and rounds half-up to whole rupiah.
func PercentOfIDR(baseIDR int64, percent float64) int64 {
	return RoundHalfUpIDR(float64(baseIDR) * percent / 100.0)
}

// SumIDR adds amounts and panics only on programmer error; callers should
// validate non-negative inputs first.
func SumIDR(parts ...int64) int64 {
	var total int64
	for _, p := range parts {
		total += p
	}
	return total
}
