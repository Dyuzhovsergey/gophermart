package accrual

import (
	"net/http"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestParseRetryAfter_NoHeader_ReturnsDefault(t *testing.T) {
	t.Parallel()

	h := make(http.Header)
	def := 60 * time.Second

	got := parseRetryAfter(h, def, zap.NewNop())
	if got != def {
		t.Fatalf("want %s, got %s", def, got)
	}
}

func TestParseRetryAfter_EmptyHeader_ReturnsDefault(t *testing.T) {
	t.Parallel()

	h := make(http.Header)
	h.Set("Retry-After", "   ")
	def := 60 * time.Second

	got := parseRetryAfter(h, def, zap.NewNop())
	if got != def {
		t.Fatalf("want %s, got %s", def, got)
	}
}

func TestParseRetryAfter_InvalidHeader_ReturnsDefault(t *testing.T) {
	t.Parallel()

	h := make(http.Header)
	h.Set("Retry-After", "abc")
	def := 60 * time.Second

	got := parseRetryAfter(h, def, zap.NewNop())
	if got != def {
		t.Fatalf("want %s, got %s", def, got)
	}
}

func TestParseRetryAfter_ZeroOrNegative_ReturnsDefault(t *testing.T) {
	t.Parallel()

	tests := []string{"0", "-1"}
	def := 60 * time.Second

	for _, v := range tests {
		h := make(http.Header)
		h.Set("Retry-After", v)

		got := parseRetryAfter(h, def, zap.NewNop())
		if got != def {
			t.Fatalf("Retry-After=%q want %s, got %s", v, def, got)
		}
	}
}

func TestParseRetryAfter_ValidSeconds_ReturnsDuration(t *testing.T) {
	t.Parallel()

	h := make(http.Header)
	h.Set("Retry-After", "5")
	def := 60 * time.Second

	got := parseRetryAfter(h, def, zap.NewNop())
	if got != 5*time.Second {
		t.Fatalf("want %s, got %s", 5*time.Second, got)
	}
}
