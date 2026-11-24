package money

import "fmt"

// IDR amounts are whole rupiah integers (PRD §31). Never floats.

func ValidateIDR(amount int64, allowZero bool) error {
	if amount < 0 {
		return fmt.Errorf("amount cannot be negative")
	}
	if !allowZero && amount == 0 {
		return fmt.Errorf("amount must be greater than zero")
	}
	return nil
}

func FormatIDR(amount int64) string {
	neg := amount < 0
	if neg {
		amount = -amount
	}
	s := fmt.Sprintf("%d", amount)
	var parts []byte
	for i, c := range reverse(s) {
		if i > 0 && i%3 == 0 {
			parts = append(parts, '.')
		}
		parts = append(parts, byte(c))
	}
	out := reverse(string(parts))
	if neg {
		return "-Rp" + out
	}
	return "Rp" + out
}

func reverse(s string) string {
	b := []byte(s)
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return string(b)
}
