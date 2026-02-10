package worker

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Dyuzhovsergey/gophermart/internal/accrual"
	"go.uber.org/zap"
)

type applyCall struct {
	number  string
	status  string
	accrual *float64
}

type fakeOrdersRepo struct {
	mu sync.Mutex

	// что вернём на первом PickForProcessing
	items []OrderForWork
	// после первого Pick возвращаем пусто (чтобы не было повторной обработки)
	picked bool

	pickCalls int
	limitSeen int

	applyCalls int
	applied    []applyCall

	pickErr  error
	applyErr error
}

func (f *fakeOrdersRepo) PickForProcessing(ctx context.Context, limit int) ([]OrderForWork, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.pickCalls++
	f.limitSeen = limit

	if f.pickErr != nil {
		return nil, f.pickErr
	}

	if f.picked {
		return nil, nil
	}
	f.picked = true

	out := make([]OrderForWork, len(f.items))
	copy(out, f.items)
	return out, nil
}

func (f *fakeOrdersRepo) ApplyAccrualResult(ctx context.Context, number string, status string, accrual *float64) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.applyCalls++
	f.applied = append(f.applied, applyCall{
		number:  number,
		status:  status,
		accrual: accrual,
	})

	return f.applyErr
}

// AccrualClient — у тебя в воркере интерфейс. Под него делаем фейк.
type fakeAccrualClient struct {
	// поведение: для каждого number можно задать ответ
	mu sync.Mutex

	orders      map[string]*accrual.Order
	errByNumber map[string]error

	calls int32
}

func newFakeAccrualClient() *fakeAccrualClient {
	return &fakeAccrualClient{
		orders:      make(map[string]*accrual.Order),
		errByNumber: make(map[string]error),
	}
}

func (c *fakeAccrualClient) GetOrder(ctx context.Context, number string) (*accrual.Order, error) {
	atomic.AddInt32(&c.calls, 1)

	c.mu.Lock()
	defer c.mu.Unlock()

	if err, ok := c.errByNumber[number]; ok && err != nil {
		return nil, err
	}
	if o, ok := c.orders[number]; ok {
		return o, nil
	}
	// имитируем 204 (заказ не зарегистрирован в accrual)
	return nil, nil
}

func eventually(t *testing.T, timeout time.Duration, cond func() bool, msg string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timeout after %s: %s", timeout, msg)
}

func TestWorker_Run_AppliesProcessedWithAccrual(t *testing.T) {
	t.Parallel()

	number := "79927398713"
	ac := newFakeAccrualClient()
	val := 10.5
	ac.orders[number] = &accrual.Order{
		Number:  number,
		Status:  "PROCESSED",
		Accrual: &val,
	}

	repo := &fakeOrdersRepo{
		items: []OrderForWork{{Number: number}},
	}

	cfg := Config{
		Interval:    10 * time.Millisecond,
		BatchSize:   10,
		Concurrency: 2,
	}

	w := New(zap.NewNop(), repo, ac, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		w.Run(ctx)
		close(done)
	}()

	// ждём, пока воркер применит результат
	eventually(t, 500*time.Millisecond, func() bool {
		repo.mu.Lock()
		defer repo.mu.Unlock()
		return repo.applyCalls == 1
	}, "ApplyAccrualResult was not called")

	repo.mu.Lock()
	call := repo.applied[0]
	repo.mu.Unlock()

	if call.number != number {
		t.Fatalf("want number=%q, got %q", number, call.number)
	}
	if call.status != "PROCESSED" {
		t.Fatalf("want status=%q, got %q", "PROCESSED", call.status)
	}
	if call.accrual == nil || *call.accrual != 10.5 {
		if call.accrual == nil {
			t.Fatalf("want accrual=10.5, got nil")
		}
		t.Fatalf("want accrual=10.5, got %v", *call.accrual)
	}

	// останавливаем воркер
	cancel()
	eventually(t, 300*time.Millisecond, func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}, "worker did not stop")
}

func TestWorker_Run_NoContent_SkipsApply(t *testing.T) {
	t.Parallel()

	number := "123"
	ac := newFakeAccrualClient()
	// для number нет ответа => GetOrder вернёт nil,nil (как 204)

	repo := &fakeOrdersRepo{
		items: []OrderForWork{{Number: number}},
	}

	cfg := Config{
		Interval:    10 * time.Millisecond,
		BatchSize:   10,
		Concurrency: 2,
	}
	w := New(zap.NewNop(), repo, ac, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Run(ctx)

	// даём чуть времени на цикл
	time.Sleep(80 * time.Millisecond)

	repo.mu.Lock()
	defer repo.mu.Unlock()

	if repo.applyCalls != 0 {
		t.Fatalf("want applyCalls=0, got %d", repo.applyCalls)
	}
}

func TestWorker_Run_RateLimit429_MakesWorkersSleepFast(t *testing.T) {
	t.Parallel()

	// проверяем идею ревью: после 429 воркеры должны "быстро уснуть"
	// допускаем, что в гонке может улететь ещё 1 запрос.
	n1 := "111"
	n2 := "222"

	ac := newFakeAccrualClient()
	ac.errByNumber[n1] = &accrual.RateLimitError{RetryAfter: 2 * time.Second}
	ac.errByNumber[n2] = &accrual.RateLimitError{RetryAfter: 2 * time.Second}

	repo := &fakeOrdersRepo{
		items: []OrderForWork{{Number: n1}, {Number: n2}},
	}

	cfg := Config{
		Interval:    10 * time.Millisecond,
		BatchSize:   10,
		Concurrency: 2,
	}
	w := New(zap.NewNop(), repo, ac, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Run(ctx)

	// ждём, чтобы первые запросы точно успели улететь
	time.Sleep(80 * time.Millisecond)

	firstCalls := atomic.LoadInt32(&ac.calls)

	// ещё подождём: если sleep сработал, новых вызовов почти не будет
	time.Sleep(120 * time.Millisecond)
	secondCalls := atomic.LoadInt32(&ac.calls)

	// После 429 воркеры должны перестать спамить запросами.
	// Разрешаем +1..+2 вызова из-за гонок и конкуррентности.
	if secondCalls-firstCalls > 2 {
		t.Fatalf("too many calls after rate limit: before=%d after=%d", firstCalls, secondCalls)
	}

	repo.mu.Lock()
	defer repo.mu.Unlock()
	if repo.applyCalls != 0 {
		t.Fatalf("want applyCalls=0, got %d", repo.applyCalls)
	}
}

func TestWorker_Run_Concurrency_ReallyParallel(t *testing.T) {
	t.Parallel()

	// Проверим, что воркер-пул реально делает параллельные вызовы.
	// Для этого сделаем клиент, который блокирует запросы, и посчитаем maxInFlight.
	n1 := "aaa"
	n2 := "bbb"

	var inFlight int32
	var maxInFlight int32

	block := make(chan struct{})

	ac := &blockingAccrualClient{
		block:       block,
		resp:        &accrual.Order{Status: "PROCESSED"},
		inFlight:    &inFlight,
		maxInFlight: &maxInFlight,
	}

	repo := &fakeOrdersRepo{
		items: []OrderForWork{{Number: n1}, {Number: n2}},
	}

	cfg := Config{
		Interval:    10 * time.Millisecond,
		BatchSize:   10,
		Concurrency: 2,
	}
	w := New(zap.NewNop(), repo, ac, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Run(ctx)

	// даём воркерам стартануть и зайти в блокировку
	time.Sleep(80 * time.Millisecond)

	// отпускаем оба запроса
	close(block)

	// ждём пока применит оба результата
	eventually(t, 700*time.Millisecond, func() bool {
		repo.mu.Lock()
		defer repo.mu.Unlock()
		return repo.applyCalls == 2
	}, "expected 2 ApplyAccrualResult calls")

	if atomic.LoadInt32(&maxInFlight) < 2 {
		t.Fatalf("want parallel requests >= 2, got %d", atomic.LoadInt32(&maxInFlight))
	}
}

// blockingAccrualClient — вспомогательный клиент для теста параллельности.
type blockingAccrualClient struct {
	block <-chan struct{}
	resp  *accrual.Order

	inFlight    *int32
	maxInFlight *int32
}

func (c *blockingAccrualClient) GetOrder(ctx context.Context, number string) (*accrual.Order, error) {
	// +1 "в полёте"
	cur := atomic.AddInt32(c.inFlight, 1)

	// обновляем максимум
	for {
		old := atomic.LoadInt32(c.maxInFlight)
		if cur <= old || atomic.CompareAndSwapInt32(c.maxInFlight, old, cur) {
			break
		}
	}

	// блокируемся (симулируем долгий запрос)
	<-c.block

	// -1 "в полёте" только после разблокировки
	atomic.AddInt32(c.inFlight, -1)

	o := *c.resp
	o.Number = number
	return &o, nil
}
