// Package ordersrepo содержит интерфейс репозитория заказов.
package ordersrepo

import (
	"context"
	"time"
)

// Order — запись заказа для выдачи пользователю.
type Order struct {
	Number     string
	Status     string
	Accrual    *float64
	UploadedAt time.Time
}

// Repository — контракт хранилища заказов.
type Repository interface {
	// AddOrder добавляет заказ:
	// - если заказ уже у этого пользователя — ErrAlreadyUploadedByUser
	// - если заказ у другого пользователя — ErrAlreadyUploadedByAnother
	// - если заказ новый — вставляет и возвращает nil
	AddOrder(ctx context.Context, userID int64, number string) error

	// ListOrdersByUser возвращает список заказов пользователя,
	// отсортированных по времени загрузки от новых к старым.
	ListOrdersByUser(ctx context.Context, userID int64) ([]Order, error)
}
