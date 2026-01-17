package worker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Dyuzhovsergey/gophermart/internal/accrual"
)

type applyCall struct {
	number  string
	status  string
	accrual *float64
}

type fakeOrdersRepo struct {
	mu sync.Mutex

	items   []OrderForWork
	pickErr error

	applyErr error
	applied  []applyCall

	pickCalls  int
	applyCalls int
	limitSeen  int
}

func (f *fakeOrdersRepo) PickForProcessing(ctx context.Context, limit int) ([]OrderForWork, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pickCalls++
	f.limitSeen = limit
	return f.items, f.pickErr
}

func (f *fakeOrdersRepo) ApplyAccrualResult(ctx context.Context, number string, status string, accrual *float64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.applyCalls++
	f.applied = append(f.applied, applyCall{number: number, status: status, accrual: accrual})
	return f.applyErr
}

func TestWorkerTick_MapsStatusAndAppliesResult(t *testing.T) {
	number := "79927398713"
	var httpCalls int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&httpCalls, 1)

		if r.URL.Path != "/api/orders/"+number {
			t.Fatalf("want path %q, got %q", "/api/orders/"+number, r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"order":"` + number + `","status":"PROCESSED","accrual":10.5}`))
	}))
	defer srv.Close()

	client, err := accrual.New(srv.URL)
	if err != nil {
		t.Fatalf("accrual.New returned error: %v", err)
	}

	repo := &fakeOrdersRepo{
		items: []OrderForWork{{Number: number}},
	}

	w := New(nil, repo, client, time.Second, 10)

	// Дёргаем tick напрямую, чтобы не ждать ticker.
	w.tick(context.Background())

	if atomic.LoadInt32(&httpCalls) != 1 {
		t.Fatalf("want httpCalls=1, got %d", httpCalls)
	}

	repo.mu.Lock()
	defer repo.mu.Unlock()

	if repo.pickCalls != 1 {
		t.Fatalf("want pickCalls=1, got %d", repo.pickCalls)
	}
	if repo.applyCalls != 1 {
		t.Fatalf("want applyCalls=1, got %d", repo.applyCalls)
	}
	if len(repo.applied) != 1 {
		t.Fatalf("want applied len=1, got %d", len(repo.applied))
	}

	call := repo.applied[0]
	if call.number != number {
		t.Fatalf("want number=%q, got %q", number, call.number)
	}
	if call.status != "PROCESSED" {
		t.Fatalf("want status=%q, got %q", "PROCESSED", call.status)
	}
	if call.accrual == nil {
		t.Fatal("want accrual != nil, got nil")
	}
	if *call.accrual != 10.5 {
		t.Fatalf("want accrual=10.5, got %v", *call.accrual)
	}
}

func TestWorkerTick_NoContent204_SkipsApply(t *testing.T) {
	number := "123"
	var httpCalls int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&httpCalls, 1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client, err := accrual.New(srv.URL)
	if err != nil {
		t.Fatalf("accrual.New returned error: %v", err)
	}

	repo := &fakeOrdersRepo{
		items: []OrderForWork{{Number: number}},
	}

	w := New(nil, repo, client, time.Second, 10)
	w.tick(context.Background())

	if atomic.LoadInt32(&httpCalls) != 1 {
		t.Fatalf("want httpCalls=1, got %d", httpCalls)
	}

	repo.mu.Lock()
	defer repo.mu.Unlock()

	if repo.applyCalls != 0 {
		t.Fatalf("want applyCalls=0, got %d", repo.applyCalls)
	}
}

func TestWorkerTick_TooManyRequests429_SetsNextAllowedAndStopsTick(t *testing.T) {
	n1 := "111"
	n2 := "222"
	var httpCalls int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&httpCalls, 1)
		// 429 с Retry-After=2s — воркер должен выставить nextAllowed и выйти из tick().
		w.Header().Set("Retry-After", "2")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	client, err := accrual.New(srv.URL)
	if err != nil {
		t.Fatalf("accrual.New returned error: %v", err)
	}

	repo := &fakeOrdersRepo{
		items: []OrderForWork{
			{Number: n1},
			{Number: n2}, // важно: до второго воркер дойти не должен (он return после 429)
		},
	}

	w := New(nil, repo, client, time.Second, 10)

	start := time.Now()
	w.tick(context.Background())

	// Должен был сходить только 1 раз (на первом заказе) и остановиться.
	if atomic.LoadInt32(&httpCalls) != 1 {
		t.Fatalf("want httpCalls=1, got %d", httpCalls)
	}

	// nextAllowed должен быть выставлен примерно на +2s.
	if !w.nextAllowed.After(start) {
		t.Fatalf("want nextAllowed after start, got %v (start=%v)", w.nextAllowed, start)
	}
	if w.nextAllowed.Sub(start) < 2*time.Second {
		t.Fatalf("want nextAllowed at least +2s, got %v", w.nextAllowed.Sub(start))
	}

	repo.mu.Lock()
	defer repo.mu.Unlock()

	if repo.applyCalls != 0 {
		t.Fatalf("want applyCalls=0, got %d", repo.applyCalls)
	}
}

func TestWorkerTick_SkipsWhenRateLimited(t *testing.T) {
	// Проверяем, что если now < nextAllowed — воркер даже не дергает репозиторий.
	repo := &fakeOrdersRepo{}

	// Любой клиент, он не будет использоваться.
	client, err := accrual.New("http://example.com")
	if err != nil {
		t.Fatalf("accrual.New returned error: %v", err)
	}

	w := New(nil, repo, client, time.Second, 10)
	w.nextAllowed = time.Now().Add(1 * time.Minute)

	w.tick(context.Background())

	repo.mu.Lock()
	defer repo.mu.Unlock()

	if repo.pickCalls != 0 {
		t.Fatalf("want pickCalls=0, got %d", repo.pickCalls)
	}
}

func TestMapAccrualStatus(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"INVALID", "INVALID"},
		{"PROCESSED", "PROCESSED"},
		{"PROCESSING", "PROCESSING"},
		{"REGISTERED", "PROCESSING"},
		{"UNKNOWN", "PROCESSING"},
		{"", "PROCESSING"},
	}

	for _, tt := range tests {
		if got := mapAccrualStatus(tt.in); got != tt.want {
			t.Fatalf("mapAccrualStatus(%q): want %q, got %q", tt.in, tt.want, got)
		}
	}
}
