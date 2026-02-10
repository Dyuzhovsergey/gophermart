// Package accounts содержит сервис для работы со счётом пользователя.
package accounts

import (
	"context"
	"fmt"

	"github.com/Dyuzhovsergey/gophermart/internal/models"
	"github.com/Dyuzhovsergey/gophermart/internal/service/accountsrepo"
)

// Service — сервис работы со счётом пользователя.
type Service interface {
	ProvideAccount(ctx context.Context, userID int64) error
	GetBalance(ctx context.Context, userID int64) (models.Balance, error)
}

type service struct {
	repo accountsrepo.Repository
}

// New создаёт сервис счетов.
func New(repo accountsrepo.Repository) Service {
	return &service{repo: repo}
}

// ProvideAccount гарантирует наличие записи счёта.
func (s *service) ProvideAccount(ctx context.Context, userID int64) error {
	return s.repo.ProvideAccount(ctx, userID)
}

// GetBalance возвращает баланс пользователя.
func (s *service) GetBalance(ctx context.Context, userID int64) (models.Balance, error) {
	cur, w, err := s.repo.GetBalance(ctx, userID)
	if err != nil {
		return models.Balance{}, fmt.Errorf("get balance: %w", err)
	}
	return models.Balance{
		Current:   cur,
		Withdrawn: w,
	}, nil
}
