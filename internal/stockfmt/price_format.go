package stockfmt

import "fmt"

// FormatPriceRange formats low~high for reports.
func FormatPriceRange(low, high float64) string {
	if low == high {
		return fmt.Sprintf("%.2f", low)
	}
	return fmt.Sprintf("%.2f~%.2f", low, high)
}
