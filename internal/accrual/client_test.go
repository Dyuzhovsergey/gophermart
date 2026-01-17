package accrual

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNew_EmptyAddr_ReturnsError(t *testing.T) {
	_, err := New("   ")
	if err == nil {
		t.Fatal("want error, got nil")
	}
}

func TestNew_AddsSchemeAndTrimsSlash(t *testing.T) {
	c, err := New("localhost:8081/")
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if c.baseURL != "http://localhost:8081" {
		t.Fatalf("want baseURL=%q, got %q", "http://localhost:8081", c.baseURL)
	}
}

func TestGetOrder_OK200_ReturnsOrder(t *testing.T) {
	number := "79927398713"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем, что клиент бьёт в правильный эндпоинт.
		if r.Method != http.MethodGet {
			t.Fatalf("want method GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/orders/"+number {
			t.Fatalf("want path %q, got %q", "/api/orders/"+number, r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"order":"` + number + `","status":"PROCESSED","accrual":729.98}`))
	}))
	defer srv.Close()

	c, err := New(srv.URL)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	got, retryAfter, err := c.GetOrder(ctx, number)
	if err != nil {
		t.Fatalf("GetOrder returned error: %v", err)
	}
	if retryAfter != nil {
		t.Fatalf("want retryAfter=nil, got %v", *retryAfter)
	}
	if got == nil {
		t.Fatal("want order != nil, got nil")
	}
	if got.Number != number {
		t.Fatalf("want number=%q, got %q", number, got.Number)
	}
	if got.Status != "PROCESSED" {
		t.Fatalf("want status=%q, got %q", "PROCESSED", got.Status)
	}
	if got.Accrual == nil {
		t.Fatal("want accrual != nil, got nil")
	}
	if *got.Accrual != 729.98 {
		t.Fatalf("want accrual=729.98, got %v", *got.Accrual)
	}
}

func TestGetOrder_NoContent204_ReturnsNilOrder(t *testing.T) {
	number := "123"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c, err := New(srv.URL)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	got, retryAfter, err := c.GetOrder(context.Background(), number)
	if err != nil {
		t.Fatalf("GetOrder returned error: %v", err)
	}
	if got != nil {
		t.Fatalf("want order=nil, got %#v", got)
	}
	if retryAfter != nil {
		t.Fatalf("want retryAfter=nil, got %v", *retryAfter)
	}
}

func TestGetOrder_TooManyRequests429_WithRetryAfterHeader(t *testing.T) {
	number := "123"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c, err := New(srv.URL)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	got, retryAfter, err := c.GetOrder(context.Background(), number)
	if err != nil {
		t.Fatalf("GetOrder returned error: %v", err)
	}
	if got != nil {
		t.Fatalf("want order=nil, got %#v", got)
	}
	if retryAfter == nil {
		t.Fatal("want retryAfter != nil, got nil")
	}
	if *retryAfter != 5*time.Second {
		t.Fatalf("want retryAfter=5s, got %v", *retryAfter)
	}
}

func TestGetOrder_TooManyRequests429_WithoutRetryAfterHeader_UsesDefault(t *testing.T) {
	number := "123"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Без Retry-After — клиент должен взять дефолт 60s.
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c, err := New(srv.URL)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	got, retryAfter, err := c.GetOrder(context.Background(), number)
	if err != nil {
		t.Fatalf("GetOrder returned error: %v", err)
	}
	if got != nil {
		t.Fatalf("want order=nil, got %#v", got)
	}
	if retryAfter == nil {
		t.Fatal("want retryAfter != nil, got nil")
	}
	if *retryAfter != 60*time.Second {
		t.Fatalf("want retryAfter=60s, got %v", *retryAfter)
	}
}

func TestGetOrder_TooManyRequests429_InvalidRetryAfterHeader_UsesDefault(t *testing.T) {
	number := "123"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "abc")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c, err := New(srv.URL)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	got, retryAfter, err := c.GetOrder(context.Background(), number)
	if err != nil {
		t.Fatalf("GetOrder returned error: %v", err)
	}
	if got != nil {
		t.Fatalf("want order=nil, got %#v", got)
	}
	if retryAfter == nil {
		t.Fatal("want retryAfter != nil, got nil")
	}
	if *retryAfter != 60*time.Second {
		t.Fatalf("want retryAfter=60s, got %v", *retryAfter)
	}
}

func TestGetOrder_UnexpectedStatus_ReturnsError(t *testing.T) {
	number := "123"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c, err := New(srv.URL)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	got, retryAfter, err := c.GetOrder(context.Background(), number)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if got != nil {
		t.Fatalf("want order=nil, got %#v", got)
	}
	if retryAfter != nil {
		t.Fatalf("want retryAfter=nil, got %v", *retryAfter)
	}
	if !strings.Contains(err.Error(), "unexpected status") {
		t.Fatalf("want error contains %q, got %q", "unexpected status", err.Error())
	}
}
