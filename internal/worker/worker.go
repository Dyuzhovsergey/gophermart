// Package worker содержит воркер для опроса accrual-сервиса.
package worker

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Dyuzhovsergey/gophermart/internal/accrual"
	"go.uber.org/zap"
)

// OrderForWork — данные заказа для обработки воркером.
type OrderForWork struct {
	Number string
}

// OrdersRepository — интерфейс репозитория заказов для воркера.
type OrdersRepository interface {
	PickForProcessing(ctx context.Context, limit int) ([]OrderForWork, error)
	ApplyAccrualResult(ctx context.Context, number string, status string, accrual *float64) error
}

// AccrualClient —  интерфейс клиента accrual
type AccrualClient interface {
	GetOrder(ctx context.Context, number string) (*accrual.Order, error)
}

// Config — настройки воркера.
type Config struct {
	Interval    time.Duration // как часто диспетчер выбирает пачку заказов
	BatchSize   int           // сколько заказов за раз выбираем из БД
	Concurrency int           // сколько параллельных воркеров (worker pool)
	JobsBuffer  int           // размер буфера канала jobs
}

// Worker — фоновая обработка заказов через accrual-сервис.
type Worker struct {
	log    *zap.Logger
	orders OrdersRepository
	client AccrualClient

	interval     time.Duration
	limit        int
	workersCount int

	jobs chan OrderForWork
	wg   sync.WaitGroup

	// inWork защищает от дублей
	inWorkMu sync.Mutex
	inWork   map[string]struct{}

	// nextAllowed — общий сон для всех воркеров при 429 Retry-After.
	rateMu      sync.Mutex
	nextAllowed time.Time
}

// New создаёт worker pool воркер
func New(log *zap.Logger, orders OrdersRepository, client AccrualClient, cfg Config) *Worker {
	interval := cfg.Interval
	if interval <= 0 {
		interval = 1 * time.Second
	}

	limit := cfg.BatchSize
	if limit <= 0 {
		limit = 5
	}

	concurrency := cfg.Concurrency
	if concurrency <= 0 {
		concurrency = 1
	}

	jobsBuf := cfg.JobsBuffer
	if jobsBuf <= 0 {
		jobsBuf = limit * 2
	}

	return &Worker{
		log:          log,
		orders:       orders,
		client:       client,
		interval:     interval,
		limit:        limit,
		workersCount: concurrency,
		jobs:         make(chan OrderForWork, jobsBuf),
		inWork:       make(map[string]struct{}),
	}
}

// Run запускает:
// 1) pool воркеров,
// 2) диспетчер, который по тикеру выбирает заказы из БД и кладёт их в jobs.
func (w *Worker) Run(ctx context.Context) {
	// Запускаем воркеры.
	for i := 0; i < w.workersCount; i++ {
		w.wg.Add(1)
		go func(workerID int) {
			defer w.wg.Done()
			w.workerLoop(ctx, workerID)
		}(i + 1)
	}

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			close(w.jobs)
			w.wg.Wait()
			if w.log != nil {
				w.log.Info("worker stopped")
			}
			return

		case <-ticker.C:
			w.dispatch(ctx)
		}
	}
}

// dispatch выбирает из БД пачку заказов и кладёт в канал jobs.
func (w *Worker) dispatch(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	if !w.isAllowedNow() {
		return
	}

	items, err := w.orders.PickForProcessing(ctx, w.limit)
	if err != nil {
		if w.log != nil {
			w.log.Error("pick orders failed", zap.Error(err))
		}
		return
	}
	if len(items) == 0 {
		return
	}

	for _, it := range items {
		if ctx.Err() != nil {
			return
		}

		ok := w.markInWork(it.Number)
		if !ok {
			continue
		}

		select {
		case w.jobs <- it: // ok

		case <-ctx.Done():
			w.unmarkInWork(it.Number)
			return
		}
	}
}

// workerLoop — цикл одного воркера: берёт задания из jobs и обрабатывает.
func (w *Worker) workerLoop(ctx context.Context, workerID int) {
	for {
		select {
		case <-ctx.Done():
			return

		case it, ok := <-w.jobs:
			if !ok {
				return
			}
			w.processOne(ctx, workerID, it)
		}
	}
}

func (w *Worker) processOne(ctx context.Context, workerID int, it OrderForWork) {
	defer w.unmarkInWork(it.Number)

	if ctx.Err() != nil {
		return
	}

	// 429 и nextAllowed — в сон.
	ok := w.sleepIfRateLimited(ctx)
	if !ok {
		return
	}

	o, err := w.client.GetOrder(ctx, it.Number)
	if err != nil {
		// 429
		var rl *accrual.RateLimitError
		if errors.As(err, &rl) {
			w.setNextAllowed(rl.RetryAfter)

			if w.log != nil {
				w.log.Warn("accrual rate limit",
					zap.Int("worker_id", workerID),
					zap.String("number", it.Number),
					zap.Duration("retry_after", rl.RetryAfter),
				)
			}
			return
		}

		if w.log != nil {
			w.log.Error("accrual get failed",
				zap.Int("worker_id", workerID),
				zap.String("number", it.Number),
				zap.Error(err),
			)
		}
		return
	}

	// 204
	if o == nil {
		return
	}

	// Маппинг статусов accrual -> статусы Gophermart.
	status := mapAccrualStatus(o.Status)

	if ctx.Err() != nil {
		return
	}

	if err := w.orders.ApplyAccrualResult(ctx, it.Number, status, o.Accrual); err != nil {
		if w.log != nil {
			w.log.Error("apply accrual result failed",
				zap.Int("worker_id", workerID),
				zap.String("number", it.Number),
				zap.Error(err),
			)
		}
		return
	}
}

func mapAccrualStatus(s string) string {
	switch s {
	case "INVALID":
		return "INVALID"
	case "PROCESSED":
		return "PROCESSED"
	case "PROCESSING", "REGISTERED":
		return "PROCESSING"
	default:
		return "PROCESSING"
	}
}

// markInWork ставит флаг "заказ уже отдан воркеру".
func (w *Worker) markInWork(number string) bool {
	w.inWorkMu.Lock()
	defer w.inWorkMu.Unlock()

	if _, exists := w.inWork[number]; exists {
		return false
	}
	w.inWork[number] = struct{}{}
	return true
}

// unmarkInWork снимает флаг inWork.
func (w *Worker) unmarkInWork(number string) {
	w.inWorkMu.Lock()
	delete(w.inWork, number)
	w.inWorkMu.Unlock()
}

// isAllowedNow проверяет сон:
func (w *Worker) isAllowedNow() bool {
	w.rateMu.Lock()
	na := w.nextAllowed
	w.rateMu.Unlock()

	return !time.Now().Before(na)
}

// setNextAllowed устанавливает общий "сон" по Retry-After.
func (w *Worker) setNextAllowed(retryAfter time.Duration) {
	if retryAfter <= 0 {
		return
	}
	target := time.Now().Add(retryAfter)

	w.rateMu.Lock()
	if target.After(w.nextAllowed) {
		w.nextAllowed = target
	}
	w.rateMu.Unlock()
}

// sleepIfRateLimited усыпляет воркера до nextAllowed
func (w *Worker) sleepIfRateLimited(ctx context.Context) bool {
	w.rateMu.Lock()
	na := w.nextAllowed
	w.rateMu.Unlock()

	now := time.Now()
	if !now.Before(na) {
		return true
	}

	d := time.Until(na)
	t := time.NewTimer(d)
	defer t.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
