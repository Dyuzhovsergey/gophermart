// Package worker содержит фонового воркера для опроса accrual-сервиса.
package worker

import (
	"context"
	"time"

	"github.com/Dyuzhovsergey/gophermart/internal/accrual"
	"go.uber.org/zap"
)

// OrderForWork — минимальные данные заказа для обработки воркером.
type OrderForWork struct {
	Number string
}

// OrdersRepository — минимальный интерфейс репозитория заказов для воркера.
type OrdersRepository interface {
	PickForProcessing(ctx context.Context, limit int) ([]OrderForWork, error)
	ApplyAccrualResult(ctx context.Context, number string, status string, accrual *float64) error
}

// Worker — фоновая обработка заказов через accrual-сервис.
type Worker struct {
	log      *zap.Logger
	orders   OrdersRepository
	client   *accrual.Client
	interval time.Duration
	limit    int

	nextAllowed time.Time // для 429 Retry-After
}

// New создаёт воркер.
func New(log *zap.Logger, orders OrdersRepository, client *accrual.Client, interval time.Duration, limit int) *Worker {
	if interval <= 0 {
		interval = 1 * time.Second
	}
	if limit <= 0 {
		limit = 5
	}
	return &Worker{
		log:      log,
		orders:   orders,
		client:   client,
		interval: interval,
		limit:    limit,
	}
}

// Run запускает цикл воркера до отмены ctx.
func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if w.log != nil {
				w.log.Info("worker stopped")
			}
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *Worker) tick(ctx context.Context) {
	now := time.Now()
	if now.Before(w.nextAllowed) {
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
		o, retryAfter, err := w.client.GetOrder(ctx, it.Number)
		if err != nil {
			if w.log != nil {
				w.log.Error("accrual get failed", zap.String("number", it.Number), zap.Error(err))
			}
			continue
		}

		if retryAfter != nil {
			w.nextAllowed = time.Now().Add(*retryAfter)
			if w.log != nil {
				w.log.Warn("accrual rate limit", zap.Duration("retry_after", *retryAfter))
			}
			return
		}

		// 204 — заказа ещё нет в accrual: просто пропускаем.
		if o == nil {
			continue
		}

		// Маппинг статусов accrual -> статусы Гофермарт.
		status := mapAccrualStatus(o.Status)

		if err := w.orders.ApplyAccrualResult(ctx, it.Number, status, o.Accrual); err != nil {
			if w.log != nil {
				w.log.Error("apply accrual result failed", zap.String("number", it.Number), zap.Error(err))
			}
			continue
		}
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
		// На всякий случай — считаем, что ещё в обработке.
		return "PROCESSING"
	}
}
