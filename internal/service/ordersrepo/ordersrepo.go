// Package ordersrepo содержит интерфейс репозитория заказов.
package ordersrepo

import "context"

// Repository — контракт хранилища заказов.
type Repository interface {
	// AddOrder добавляет заказ:
	// - если заказ уже у этого пользователя — ErrAlreadyUploadedByUser
	// - если заказ у другого пользователя — ErrAlreadyUploadedByAnother
	// - если заказ новый — вставляет и возвращает nil
	AddOrder(ctx context.Context, userID int64, number string) error
}
