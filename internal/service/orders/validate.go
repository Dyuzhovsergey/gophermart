// Package orders содержит бизнес-логику, связанную с заказами.
package orders

import "github.com/Dyuzhovsergey/gophermart/internal/service/domainerr"

// ValidateOrderNumber проверяет, что номер заказа состоит только из цифр
// и проходит проверку алгоритмом Луна.
func ValidateOrderNumber(s string) error {
	if s == "" {
		return domainerr.ErrInvalidOrder
	}

	// Только цифры
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return domainerr.ErrInvalidOrder
		}
	}

	// Алгоритм Луна
	var sum int
	alt := false
	for i := len(s) - 1; i >= 0; i-- {
		d := int(s[i] - '0')
		if alt {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		alt = !alt
	}

	if sum%10 != 0 {
		return domainerr.ErrInvalidOrder
	}

	return nil
}
