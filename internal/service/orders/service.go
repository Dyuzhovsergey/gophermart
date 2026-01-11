// Package orders содержит бизнес-логику, связанную с заказами.
package orders

import (
	"context"

	"github.com/Dyuzhovsergey/gophermart/internal/service/ordersrepo"
)

// Service — интерфейс сервиса заказов.
type Service interface {
	// UploadOrder валидирует номер заказа и пытается добавить его в систему.
	// Возвращает nil при успешной регистрации нового заказа.
	UploadOrder(ctx context.Context, userID int64, number string) error
}

type service struct {
	repo ordersrepo.Repository
}

// New создаёт сервис заказов.
func New(repo ordersrepo.Repository) Service {
	return &service{repo: repo}
}

// UploadOrder валидирует номер заказа и добавляет его в БД.
func (s *service) UploadOrder(ctx context.Context, userID int64, number string) error {
	if err := ValidateOrderNumber(number); err != nil {
		return err
	}
	return s.repo.AddOrder(ctx, userID, number)
}
