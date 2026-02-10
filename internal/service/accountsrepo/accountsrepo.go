// Package accountsrepo содержит интерфейс репозитория счетов пользователей.
package accountsrepo

import "context"

// Repository — контракт хранилища счетов пользователей.
type Repository interface {
	// EnsureAccount гарантирует наличие записи счёта для пользователя.
	// Если запись уже есть — ничего не делает.
	ProvideAccount(ctx context.Context, userID int64) error

	// GetBalance возвращает текущий баланс и сумму списаний.
	// Если счёта ещё нет — возвращает (0, 0, nil).
	GetBalance(ctx context.Context, userID int64) (current float64, withdrawn float64, err error)
}
