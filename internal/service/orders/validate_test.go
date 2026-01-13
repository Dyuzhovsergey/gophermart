package orders

import (
	"testing"

	"github.com/Dyuzhovsergey/gophermart/internal/service/domainerr"
)

func TestValidateOrderNumber_Valid(t *testing.T) {
	t.Parallel()

	// Набор известных валидных по Луну номеров
	valid := []string{
		"79927398713", // классический пример Луна
		"4012888888881881",
		"4532015112830366",
	}

	for _, s := range valid {
		s := s
		t.Run(s, func(t *testing.T) {
			t.Parallel()
			if err := ValidateOrderNumber(s); err != nil {
				t.Fatalf("want nil, got %v", err)
			}
		})
	}
}

func TestValidateOrderNumber_Invalid(t *testing.T) {
	t.Parallel()

	invalid := []string{
		"",            // пустая строка
		"   ",         // пробелы
		"123456789",   // не проходит Лун
		"79927398710", // неправильная контрольная цифра
		"abcdef",      // не цифры
		"1234abcd",    // смешанный ввод
		"12-34",       // символы
	}

	for _, s := range invalid {
		s := s
		t.Run(s, func(t *testing.T) {
			t.Parallel()
			if err := ValidateOrderNumber(s); err != domainerr.ErrInvalidOrder {
				t.Fatalf("want %v, got %v", domainerr.ErrInvalidOrder, err)
			}
		})
	}
}
