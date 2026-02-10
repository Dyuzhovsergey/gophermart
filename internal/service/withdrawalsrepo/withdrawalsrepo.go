// Package withdrawalsrepo описывает интерфейс репозитория списаний.
package withdrawalsrepo

import (
	"context"
	"time"
)

// Withdrawal — запись о списании (в формате, удобном для сервиса/хендлера).
type Withdrawal struct {
	Order       string
	Sum         float64
	ProcessedAt time.Time
}

type Repository interface {
	Withdraw(ctx context.Context, userID int64, order string, sum float64) error
	ListWithdrawals(ctx context.Context, userID int64) ([]Withdrawal, error)
}
