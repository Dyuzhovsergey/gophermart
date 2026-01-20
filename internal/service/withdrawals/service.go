// Package withdrawals содержит бизнес-логику списаний.
package withdrawals

import (
	"context"

	"github.com/Dyuzhovsergey/gophermart/internal/service/domainerr"
	"github.com/Dyuzhovsergey/gophermart/internal/service/orders"
	"github.com/Dyuzhovsergey/gophermart/internal/service/withdrawalsrepo"
)

type Service interface {
	Withdraw(ctx context.Context, userID int64, order string, sum float64) error
	List(ctx context.Context, userID int64) ([]withdrawalsrepo.Withdrawal, error)
}

type service struct {
	repo withdrawalsrepo.Repository
}

func New(repo withdrawalsrepo.Repository) Service {
	return &service{repo: repo}
}

func (s *service) Withdraw(ctx context.Context, userID int64, order string, sum float64) error {
	// Валидация номера заказа
	if err := orders.ValidateOrderNumber(order); err != nil {
		return domainerr.ErrInvalidOrder
	}
	if sum <= 0 {
		//  bad request
		return domainerr.ErrConflict
	}
	return s.repo.Withdraw(ctx, userID, order, sum)
}

func (s *service) List(ctx context.Context, userID int64) ([]withdrawalsrepo.Withdrawal, error) {
	return s.repo.ListWithdrawals(ctx, userID)
}
